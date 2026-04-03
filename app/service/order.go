package service

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/amap"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
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

// CreateOrderReq 创建订单：直接进入「待卖方确认收款」；buyer_agreed_at 记下单时间（买方契约）
// 收货/发货各含：文字地址 + 地图选点（GCJ-02 经纬度，成对传）；距离优先用两端坐标计算
type CreateOrderReq struct {
	GoodsID      uint     `json:"goods_id" binding:"required"`
	ReceiverAddr string   `json:"receiver_addr"`
	SenderAddr   string   `json:"sender_addr"`
	ReceiverLat  *float64 `json:"receiver_lat"`
	ReceiverLng  *float64 `json:"receiver_lng"`
	SenderLat    *float64 `json:"sender_lat"`
	SenderLng    *float64 `json:"sender_lng"`
}

// UpdateSellerAddrReq 卖方更新发货文字与地图坐标（坐标须成对传才写入）
type UpdateSellerAddrReq struct {
	SenderAddr string   `json:"sender_addr"`
	SenderLat  *float64 `json:"sender_lat"`
	SenderLng  *float64 `json:"sender_lng"`
}

func (s *orderService) Create(ctx *gin.Context, buyerID uint, schoolID uint, req CreateOrderReq) (uint, error) {
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
	receiver := strings.TrimSpace(req.ReceiverAddr)
	if receiver == "" && good.GoodsType == constant.GoodsTypePickup {
		receiver = goodAddrForOrder(good)
	}
	sender := strings.TrimSpace(req.SenderAddr)
	if sender == "" {
		sender = goodAddrForOrder(good)
	}
	now := time.Now()
	o := &model.Order{
		UserID:        &uid,
		GoodsID:       &gid,
		OrderStatus:   constant.OrderStatusAwaitSellerPaymentConfirm,
		ReceiverAddr:  receiver,
		SenderAddr:    sender,
		BuyerAgreedAt: &now,
	}
	if req.ReceiverLat != nil && req.ReceiverLng != nil {
		o.ReceiverLat = req.ReceiverLat
		o.ReceiverLng = req.ReceiverLng
	}
	if req.SenderLat != nil && req.SenderLng != nil {
		o.SenderLat = req.SenderLat
		o.SenderLng = req.SenderLng
	}
	// 仅「送货上门」计算步行距离：优先两端地图坐标，否则两段文字地址地理编码
	if good.GoodsType == constant.GoodsTypeDelivery {
		if d := computeOrderDistanceMeters(ctx, sender, receiver, o.SenderLat, o.SenderLng, o.ReceiverLat, o.ReceiverLng); d != nil {
			o.DistanceMeters = d
		}
	}
	return dao.Order().Create(ctx.Request.Context(), o)
}

// computeOrderDistanceMeters 发货地→收货地步行规划距离（米）。需配置 AMAP_KEY；失败返回 nil，不阻断下单。
// 若收发两端各有成对经纬度，直接测距；否则在两段文字地址均非空时走地理编码再测距。
func computeOrderDistanceMeters(ctx *gin.Context, senderAddr, receiverAddr string, senderLat, senderLng, receiverLat, receiverLng *float64) *int {
	if config.AmapKey == "" {
		return nil
	}
	if senderLat != nil && senderLng != nil && receiverLat != nil && receiverLng != nil {
		from := &amap.GeocodeResult{Lng: *senderLng, Lat: *senderLat}
		to := &amap.GeocodeResult{Lng: *receiverLng, Lat: *receiverLat}
		m, err := amap.WalkingDistanceMeters(ctx.Request.Context(), config.AmapKey, from, to)
		if err != nil {
			return nil
		}
		v := m
		return &v
	}
	sa := strings.TrimSpace(senderAddr)
	ra := strings.TrimSpace(receiverAddr)
	if sa == "" || ra == "" {
		return nil
	}
	m, err := amap.DistanceBetweenAddresses(ctx.Request.Context(), config.AmapKey, sa, ra)
	if err != nil {
		return nil
	}
	v := m
	return &v
}

func (s *orderService) ListByBuyer(ctx *gin.Context, userID uint, page, pageSize int) ([]*model.Order, int64, error) {
	return dao.Order().ListByUserID(ctx.Request.Context(), userID, page, pageSize)
}

func (s *orderService) ListBySeller(ctx *gin.Context, sellerID uint, page, pageSize int) ([]*model.Order, int64, error) {
	return dao.Order().ListBySellerID(ctx.Request.Context(), sellerID, page, pageSize)
}

func (s *orderService) GetByID(ctx *gin.Context, id uint) (*model.Order, error) {
	return dao.Order().GetByID(ctx.Request.Context(), id)
}

func (s *orderService) UpdateStatus(ctx *gin.Context, id uint, orderStatus int16) error {
	return dao.Order().UpdateOrderStatus(ctx.Request.Context(), id, orderStatus)
}

// UpdateSellerInfo 卖家更新发货文字地址与/或地图坐标；订单为待卖方确认收款或履约中时可改。
func (s *orderService) UpdateSellerInfo(ctx *gin.Context, id uint, sellerID uint, req UpdateSellerAddrReq) error {
	o, err := dao.Order().GetByID(ctx.Request.Context(), id)
	if err != nil || o == nil {
		return errno.ErrOrderNotFound
	}
	if o.GoodsID == nil {
		return errno.ErrOrderNotFound
	}
	g, err := dao.Good().GetByID(ctx.Request.Context(), uint(*o.GoodsID))
	if err != nil || g == nil || g.UserID == nil || uint(*g.UserID) != sellerID {
		return errors.New("订单不存在或无权操作")
	}
	if o.OrderStatus != constant.OrderStatusAwaitSellerPaymentConfirm &&
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
	if g.GoodsType == constant.GoodsTypeDelivery {
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
		if d := computeOrderDistanceMeters(ctx, senderStr, receiverStr, sLat, sLng, rLat, rLng); d != nil {
			updates["distance_meters"] = *d
		}
	}
	return dao.Order().UpdateColumns(ctx.Request.Context(), id, updates)
}
