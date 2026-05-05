package service

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/geo"
	"gorm.io/gorm"
)

type orderService struct{}

// goodAddrForOrder 商品上的统一地址：用于自提默认收货、送货上门默认发货地（与商品 goods_addr / pickup_addr 一致）
func goodAddrForOrder(g *model.Good) string {
	if g == nil {
		return ""
	}
	s := strings.TrimSpace(g.GoodsAddr)
	if s != "" {
		return s
	}
	return strings.TrimSpace(g.PickupAddr)
}

// CreateOrderReq 创建订单
// user_location_id 为 0 或省略：创建不完整订单（order_status=待买方完善），仅含 goods_id，用于商品页「我想要」。
// user_location_id > 0：与旧版一致，直接进入「待卖方确认收款」。
// 发货：未传 sender_* 时由商品 goods_addr / goods_lat|lng 写入；可显式传 sender_* 覆盖。
type CreateOrderReq struct {
	GoodsID        uint     `json:"goods_id" binding:"required"`
	UserLocationID uint     `json:"user_location_id"`
	SenderAddr     string   `json:"sender_addr"`
	SenderLat      *float64 `json:"sender_lat"`
	SenderLng      *float64 `json:"sender_lng"`
}

// UpdateSellerAddrReq 卖方更新发货文字与地图坐标（坐标须成对传才写入）
type UpdateSellerAddrReq struct {
	SenderAddr string   `json:"sender_addr"`
	SenderLat  *float64 `json:"sender_lat"`
	SenderLng  *float64 `json:"sender_lng"`
}

func (s *orderService) Create(ctx *gin.Context, buyerID uint, schoolID uint, req CreateOrderReq) (uint, error) {
	if req.UserLocationID == 0 {
		// 有偿求助不走"完善地址"流程：接单者点「我来接」直接建单进行中
		good, gerr := dao.Good().GetByIDWithSchool(ctx.Request.Context(), req.GoodsID, schoolID)
		if gerr == nil && good != nil && good.GoodsCategory == constant.GoodsCategoryHelp {
			return s.createHelpOrder(ctx, buyerID, good)
		}
		return s.createDraft(ctx, buyerID, schoolID, req.GoodsID)
	}
	loc, err := dao.UserLocation().GetByIDAndUserID(ctx.Request.Context(), req.UserLocationID, buyerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errno.ErrUserLocationNotFound // 地址不存在、非本人或已删除
		}
		return 0, err
	}
	good, err := dao.Good().GetByIDWithSchool(ctx.Request.Context(), req.GoodsID, schoolID)
	if err != nil || good == nil {
		return 0, errno.ErrOrderGoodNotFound
	}
	if good.GoodStatus != dao.GoodStatusOnSale {
		return 0, errno.ErrOrderGoodNotOnSale
	}
	if good.Stock < 1 {
		return 0, errno.ErrOrderInsufficientStock
	}
	if good.UserID != nil && uint(*good.UserID) == buyerID {
		return 0, errors.New("不能购买自己发布的商品")
	}
	uid := int(buyerID)
	gid := int(req.GoodsID)
	receiver := strings.TrimSpace(loc.Addr)
	sender := strings.TrimSpace(req.SenderAddr)
	if sender == "" {
		sender = goodAddrForOrder(good)
	}
	now := time.Now()
	lid := loc.ID
	o := &model.Order{
		UserID:                 &uid,
		GoodsID:                &gid,
		OrderStatus:            constant.OrderStatusAwaitSellerPaymentConfirm,
		ReceiverUserLocationID: &lid,
		ReceiverAddr:           receiver,
		SenderAddr:             sender,
		BuyerAgreedAt:          &now,
	}
	if loc.Lat != nil && loc.Lng != nil {
		o.ReceiverLat = copyFloatPtr(*loc.Lat)
		o.ReceiverLng = copyFloatPtr(*loc.Lng)
	}
	if req.SenderLat != nil && req.SenderLng != nil {
		o.SenderLat = copyFloatPtr(*req.SenderLat)
		o.SenderLng = copyFloatPtr(*req.SenderLng)
	} else if good.GoodsLat != nil && good.GoodsLng != nil {
		// 与商品表解耦，避免与 GORM 扫描缓冲共用指针导致落库异常
		o.SenderLat = copyFloatPtr(*good.GoodsLat)
		o.SenderLng = copyFloatPtr(*good.GoodsLng)
	}
	// 送货上门 / 自提：两端均有成对经纬度时 Haversine 球面直线距离（米）。自提为「自提点→买方位置」。
	if good.GoodsType == constant.GoodsTypeDelivery || good.GoodsType == constant.GoodsTypePickup {
		if d := computeOrderDistanceMeters(sender, receiver, o.SenderLat, o.SenderLng, o.ReceiverLat, o.ReceiverLng); d != nil {
			o.DistanceMeters = d
		}
	}
	return dao.Order().Create(ctx.Request.Context(), o)
}

// createHelpOrder 有偿求助「我来接」：无需收货地址，订单直接进入进行中
//
// 复用 schema 状态值：status=AwaitSellerPaymentConfirm(1) 表示"进行中（待发布者付酬）"。
// 发布者在聊天里上传付酬截图后转到 PendingBuyerConfirm(3)；接单者确认收到后 → Completed(4)。
func (s *orderService) createHelpOrder(ctx *gin.Context, takerID uint, good *model.Good) (uint, error) {
	if good.GoodStatus != dao.GoodStatusOnSale {
		return 0, errno.ErrOrderGoodNotOnSale
	}
	if good.Stock < 1 {
		return 0, errno.ErrOrderInsufficientStock
	}
	if good.UserID != nil && uint(*good.UserID) == takerID {
		return 0, errors.New("不能接自己发布的求助")
	}
	if existing, err := dao.Order().FindActiveBuyerOrderForGoods(ctx.Request.Context(), takerID, uint(good.ID)); err == nil && existing != nil {
		return existing.ID, nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	uid := int(takerID)
	gid := int(good.ID)
	now := time.Now()
	o := &model.Order{
		UserID:        &uid,
		GoodsID:       &gid,
		OrderStatus:   constant.OrderStatusAwaitSellerPaymentConfirm,
		BuyerAgreedAt: &now,
	}
	return dao.Order().Create(ctx.Request.Context(), o)
}

// createDraft 商品页「我想要」：无收货地址，order_status=待买方完善
func (s *orderService) createDraft(ctx *gin.Context, buyerID uint, schoolID uint, goodsID uint) (uint, error) {
	good, err := dao.Good().GetByIDWithSchool(ctx.Request.Context(), goodsID, schoolID)
	if err != nil || good == nil {
		return 0, errno.ErrOrderGoodNotFound
	}
	if good.GoodStatus != dao.GoodStatusOnSale {
		return 0, errno.ErrOrderGoodNotOnSale
	}
	if good.Stock < 1 {
		return 0, errno.ErrOrderInsufficientStock
	}
	if good.UserID != nil && uint(*good.UserID) == buyerID {
		return 0, errors.New("不能购买自己发布的商品")
	}
	// 已存在与卖家的未完成订单则复用，避免重复会话
	if existing, err := dao.Order().FindActiveBuyerOrderForGoods(ctx.Request.Context(), buyerID, goodsID); err == nil && existing != nil {
		return existing.ID, nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	uid := int(buyerID)
	gid := int(goodsID)
	sender := goodAddrForOrder(good)
	o := &model.Order{
		UserID:      &uid,
		GoodsID:     &gid,
		OrderStatus: constant.OrderStatusAwaitBuyerLocation,
		SenderAddr:  sender,
	}
	if good.GoodsLat != nil && good.GoodsLng != nil {
		o.SenderLat = copyFloatPtr(*good.GoodsLat)
		o.SenderLng = copyFloatPtr(*good.GoodsLng)
	}
	return dao.Order().Create(ctx.Request.Context(), o)
}

// OrderLocationUpdateReq 统一更新收货/发货（POST /orders/:id/location）
type OrderLocationUpdateReq struct {
	Type string `json:"type" binding:"required"` // buyer | seller

	UserLocationID uint `json:"user_location_id"`
	ProposalOnly   bool `json:"proposal_only"` // 买方：仅提交待卖方确认的地址修改

	SenderAddr string   `json:"sender_addr"`
	SenderLat  *float64 `json:"sender_lat"`
	SenderLng  *float64 `json:"sender_lng"`

	ConfirmBuyerLocation bool `json:"confirm_buyer_location"`
	RejectBuyerLocation  bool `json:"reject_buyer_location"`
}

// OrderLocationUpdate 买方完善草稿/申请改址；卖方确认买方改址或更新发货地
func (s *orderService) OrderLocationUpdate(ctx *gin.Context, orderID uint, userID uint, req OrderLocationUpdateReq) error {
	t := strings.TrimSpace(strings.ToLower(req.Type))
	switch t {
	case "buyer":
		return s.orderLocationBuyer(ctx, orderID, userID, req)
	case "seller":
		return s.orderLocationSeller(ctx, orderID, userID, req)
	default:
		return errors.New("type 须为 buyer 或 seller")
	}
}

func (s *orderService) orderLocationBuyer(ctx *gin.Context, orderID uint, userID uint, req OrderLocationUpdateReq) error {
	o, g, isBuyer, _, err := s.resolveOrderParticipant(ctx, orderID, userID)
	if err != nil {
		return err
	}
	if !isBuyer {
		return errno.ErrOrderNotParticipant
	}
	if req.UserLocationID == 0 {
		return errors.New("请选择收货地址 user_location_id")
	}
	loc, err := dao.UserLocation().GetByIDAndUserID(ctx.Request.Context(), req.UserLocationID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errno.ErrUserLocationNotFound
		}
		return err
	}
	receiver := strings.TrimSpace(loc.Addr)
	lid := loc.ID

	switch o.OrderStatus {
	case constant.OrderStatusAwaitBuyerLocation:
		if req.ProposalOnly {
			return errors.New("不完整订单请直接选择收货地址，无需申请卖方确认")
		}
		now := time.Now()
		updates := map[string]interface{}{
			"receiver_user_location_id":         lid,
			"receiver_addr":                     receiver,
			"order_status":                      constant.OrderStatusAwaitSellerPaymentConfirm,
			"buyer_agreed_at":                   &now,
			"pending_receiver_user_location_id": gorm.Expr("NULL"),
			"pending_receiver_addr":             "",
			"pending_receiver_lat":              gorm.Expr("NULL"),
			"pending_receiver_lng":              gorm.Expr("NULL"),
		}
		if loc.Lat != nil && loc.Lng != nil {
			updates["receiver_lat"] = *loc.Lat
			updates["receiver_lng"] = *loc.Lng
		} else {
			updates["receiver_lat"] = gorm.Expr("NULL")
			updates["receiver_lng"] = gorm.Expr("NULL")
		}
		s.applyDistanceToUpdates(g, o, updates, receiver)
		return dao.Order().UpdateColumns(ctx.Request.Context(), orderID, updates)

	case constant.OrderStatusAwaitSellerPaymentConfirm, constant.OrderStatusFulfillment:
		if !req.ProposalOnly {
			return errors.New("修改收货地址请传 proposal_only: true，由卖方确认后生效")
		}
		pu := uint(lid)
		updates := map[string]interface{}{
			"pending_receiver_user_location_id": pu,
			"pending_receiver_addr":             receiver,
		}
		if loc.Lat != nil && loc.Lng != nil {
			updates["pending_receiver_lat"] = *loc.Lat
			updates["pending_receiver_lng"] = *loc.Lng
		} else {
			updates["pending_receiver_lat"] = gorm.Expr("NULL")
			updates["pending_receiver_lng"] = gorm.Expr("NULL")
		}
		return dao.Order().UpdateColumns(ctx.Request.Context(), orderID, updates)

	default:
		return errno.ErrOrderInvalidState
	}
}

func (s *orderService) orderLocationSeller(ctx *gin.Context, orderID uint, userID uint, req OrderLocationUpdateReq) error {
	o, g, _, isSeller, err := s.resolveOrderParticipant(ctx, orderID, userID)
	if err != nil {
		return err
	}
	if !isSeller {
		return errno.ErrOrderNotParticipant
	}
	if req.ConfirmBuyerLocation && req.RejectBuyerLocation {
		return errors.New("不能同时确认与拒绝")
	}
	if req.ConfirmBuyerLocation {
		if o.PendingReceiverUserLocationID == nil || strings.TrimSpace(o.PendingReceiverAddr) == "" {
			return errors.New("暂无待确认的买方地址修改")
		}
		lid := *o.PendingReceiverUserLocationID
		updates := map[string]interface{}{
			"receiver_user_location_id":         lid,
			"receiver_addr":                     strings.TrimSpace(o.PendingReceiverAddr),
			"pending_receiver_user_location_id": gorm.Expr("NULL"),
			"pending_receiver_addr":             "",
			"pending_receiver_lat":              gorm.Expr("NULL"),
			"pending_receiver_lng":              gorm.Expr("NULL"),
		}
		if o.PendingReceiverLat != nil && o.PendingReceiverLng != nil {
			updates["receiver_lat"] = *o.PendingReceiverLat
			updates["receiver_lng"] = *o.PendingReceiverLng
		} else {
			updates["receiver_lat"] = gorm.Expr("NULL")
			updates["receiver_lng"] = gorm.Expr("NULL")
		}
		receiver := strings.TrimSpace(o.PendingReceiverAddr)
		s.applyDistanceToUpdatesFromOrder(g, o, updates, receiver)
		return dao.Order().UpdateColumns(ctx.Request.Context(), orderID, updates)
	}
	if req.RejectBuyerLocation {
		return dao.Order().UpdateColumns(ctx.Request.Context(), orderID, map[string]interface{}{
			"pending_receiver_user_location_id": gorm.Expr("NULL"),
			"pending_receiver_addr":             "",
			"pending_receiver_lat":              gorm.Expr("NULL"),
			"pending_receiver_lng":              gorm.Expr("NULL"),
		})
	}
	senderAddr := strings.TrimSpace(req.SenderAddr)
	if senderAddr == "" && req.SenderLat == nil && req.SenderLng == nil {
		return errors.New("请填写发货地址或坐标，或使用 confirm_buyer_location / reject_buyer_location")
	}
	// 卖方更新发货地（与 PUT /orders/:id 一致）
	return s.UpdateSellerInfo(ctx, orderID, userID, UpdateSellerAddrReq{
		SenderAddr: req.SenderAddr,
		SenderLat:  req.SenderLat,
		SenderLng:  req.SenderLng,
	})
}

func (s *orderService) applyDistanceToUpdates(g *model.Good, o *model.Order, updates map[string]interface{}, receiverAddr string) {
	if g.GoodsType != constant.GoodsTypeDelivery && g.GoodsType != constant.GoodsTypePickup {
		return
	}
	senderStr := strings.TrimSpace(o.SenderAddr)
	sLat, sLng := o.SenderLat, o.SenderLng
	var rLat, rLng *float64
	if v, ok := updates["receiver_lat"].(float64); ok {
		rLat = &v
	}
	if v, ok := updates["receiver_lng"].(float64); ok {
		rLng = &v
	}
	if d := computeOrderDistanceMeters(senderStr, receiverAddr, sLat, sLng, rLat, rLng); d != nil {
		updates["distance_meters"] = *d
	} else {
		updates["distance_meters"] = gorm.Expr("NULL")
	}
}

func (s *orderService) applyDistanceToUpdatesFromOrder(g *model.Good, o *model.Order, updates map[string]interface{}, receiverAddr string) {
	if g.GoodsType != constant.GoodsTypeDelivery && g.GoodsType != constant.GoodsTypePickup {
		return
	}
	senderStr := strings.TrimSpace(o.SenderAddr)
	sLat, sLng := o.SenderLat, o.SenderLng
	var rLat, rLng *float64
	if v, ok := updates["receiver_lat"].(float64); ok {
		rLat = &v
	}
	if v, ok := updates["receiver_lng"].(float64); ok {
		rLng = &v
	}
	if d := computeOrderDistanceMeters(senderStr, receiverAddr, sLat, sLng, rLat, rLng); d != nil {
		updates["distance_meters"] = *d
	} else {
		updates["distance_meters"] = gorm.Expr("NULL")
	}
}

func copyFloatPtr(f float64) *float64 {
	v := f
	return &v
}

// computeOrderDistanceMeters 发货/自提点→收货（买方）球面直线距离（米，Haversine / WGS84）。两端须均有成对经纬度。
func computeOrderDistanceMeters(senderAddr, receiverAddr string, senderLat, senderLng, receiverLat, receiverLng *float64) *int {
	_ = senderAddr
	_ = receiverAddr
	if senderLat == nil || senderLng == nil || receiverLat == nil || receiverLng == nil {
		return nil
	}
	v := geo.HaversineMeters(*senderLat, *senderLng, *receiverLat, *receiverLng)
	return &v
}

// ListByBuyer "我的买家订单"——把 caller 主账号 + 旗下号下过的单合并展示。
func (s *orderService) ListByBuyer(ctx *gin.Context, userID uint, page, pageSize int) ([]*model.Order, int64, error) {
	ids, err := GetAccountIDsForOps(ctx.Request.Context(), userID)
	if err == nil && ids.IsAggregated() {
		return dao.Order().ListByUserIDs(ctx.Request.Context(), ids.AllIDs, page, pageSize)
	}
	return dao.Order().ListByUserID(ctx.Request.Context(), userID, page, pageSize)
}

// ListBySeller "我的卖家订单"——把 caller 主账号 + 旗下号挂的商品对应的订单合并展示。
func (s *orderService) ListBySeller(ctx *gin.Context, sellerID uint, page, pageSize int) ([]*model.Order, int64, error) {
	ids, err := GetAccountIDsForOps(ctx.Request.Context(), sellerID)
	if err == nil && ids.IsAggregated() {
		return dao.Order().ListBySellerIDs(ctx.Request.Context(), ids.AllIDs, page, pageSize)
	}
	return dao.Order().ListBySellerID(ctx.Request.Context(), sellerID, page, pageSize)
}

func (s *orderService) GetByID(ctx *gin.Context, id uint) (*model.Order, error) {
	return dao.Order().GetByID(ctx.Request.Context(), id)
}

func (s *orderService) UpdateStatus(ctx *gin.Context, id uint, orderStatus int16) error {
	return dao.Order().UpdateOrderStatus(ctx.Request.Context(), id, orderStatus)
}

// UpdateSellerInfo 卖家更新发货文字地址与/或地图坐标；订单为待卖方确认收款或履约中时可改。
//
// "账号集"权限：caller (sellerID) 是商品 owner 本人 **或** 是 owner 的主账号 / 旗下号 都允许。
func (s *orderService) UpdateSellerInfo(ctx *gin.Context, id uint, sellerID uint, req UpdateSellerAddrReq) error {
	o, err := dao.Order().GetByID(ctx.Request.Context(), id)
	if err != nil || o == nil {
		return errno.ErrOrderNotFound
	}
	if o.GoodsID == nil {
		return errno.ErrOrderNotFound
	}
	g, err := dao.Good().GetByID(ctx.Request.Context(), uint(*o.GoodsID))
	if err != nil || g == nil || g.UserID == nil {
		return errors.New("订单不存在或无权操作")
	}
	ids, ierr := GetAccountIDsForOps(ctx.Request.Context(), sellerID)
	if ierr != nil || !ids.IsOwnedByOneOf(uint(*g.UserID)) {
		return errors.New("订单不存在或无权操作")
	}
	if o.OrderStatus != constant.OrderStatusAwaitBuyerLocation &&
		o.OrderStatus != constant.OrderStatusAwaitSellerPaymentConfirm &&
		o.OrderStatus != constant.OrderStatusFulfillment {
		return errno.ErrOrderInvalidState
	}
	updates := make(map[string]interface{})
	senderAddr := strings.TrimSpace(req.SenderAddr)
	if senderAddr != "" {
		updates["sender_addr"] = senderAddr
	}
	if req.SenderLat != nil && req.SenderLng != nil {
		updates["sender_lat"] = *req.SenderLat
		updates["sender_lng"] = *req.SenderLng
	}
	if len(updates) == 0 {
		return nil
	}
	if g.GoodsType == constant.GoodsTypeDelivery || g.GoodsType == constant.GoodsTypePickup {
		senderStr := strings.TrimSpace(o.SenderAddr)
		if senderAddr != "" {
			senderStr = senderAddr
		}
		var sLat, sLng *float64
		if req.SenderLat != nil && req.SenderLng != nil {
			sLat, sLng = req.SenderLat, req.SenderLng
		} else {
			sLat, sLng = o.SenderLat, o.SenderLng
		}
		receiverStr := strings.TrimSpace(o.ReceiverAddr)
		rLat, rLng := o.ReceiverLat, o.ReceiverLng
		if d := computeOrderDistanceMeters(senderStr, receiverStr, sLat, sLng, rLat, rLng); d != nil {
			updates["distance_meters"] = *d
		}
	}
	return dao.Order().UpdateColumns(ctx.Request.Context(), id, updates)
}
