package service

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"gorm.io/gorm"
)

// resolveOrderParticipant 校验 user 为买方或卖方，返回订单与商品
func (s *orderService) resolveOrderParticipant(ctx *gin.Context, orderID uint, userID uint) (*model.Order, *model.Good, bool, bool, error) {
	o, err := dao.Order().GetByID(ctx.Request.Context(), orderID)
	if err != nil || o == nil {
		return nil, nil, false, false, errno.ErrOrderNotFound
	}
	if o.GoodsID == nil {
		return nil, nil, false, false, errno.ErrOrderNotFound
	}
	g, err := dao.Good().GetByID(ctx.Request.Context(), uint(*o.GoodsID))
	if err != nil || g == nil {
		return nil, nil, false, false, errno.ErrOrderNotFound
	}
	isBuyer := o.UserID != nil && uint(*o.UserID) == userID
	isSeller := g.UserID != nil && uint(*g.UserID) == userID
	if !isBuyer && !isSeller {
		return nil, nil, false, false, errno.ErrOrderNotParticipant
	}
	return o, g, isBuyer, isSeller, nil
}

// ListOrderMessages 订单聊天记录（买卖双方）
func (s *orderService) ListOrderMessages(ctx *gin.Context, orderID uint, userID uint, page, pageSize int) ([]*model.OrderMessage, int64, error) {
	_, _, _, _, err := s.resolveOrderParticipant(ctx, orderID, userID)
	if err != nil {
		return nil, 0, err
	}
	return s.listOrderMessagesRaw(ctx, orderID, page, pageSize)
}

// ListOrderMessagesAdmin 管理端查看聊天记录（不校验买卖双方）
func (s *orderService) ListOrderMessagesAdmin(ctx *gin.Context, orderID uint, page, pageSize int) ([]*model.OrderMessage, int64, error) {
	o, err := dao.Order().GetByID(ctx.Request.Context(), orderID)
	if err != nil || o == nil {
		return nil, 0, errno.ErrOrderNotFound
	}
	_ = o
	return s.listOrderMessagesRaw(ctx, orderID, page, pageSize)
}

func (s *orderService) listOrderMessagesRaw(ctx *gin.Context, orderID uint, page, pageSize int) ([]*model.OrderMessage, int64, error) {
	list, total, err := dao.OrderMessage().ListByOrderID(ctx.Request.Context(), orderID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	for _, m := range list {
		if m.ImageURL != "" {
			m.ImageURL = oss.ToFullURL(m.ImageURL)
		}
	}
	return list, total, err
}

// CreateOrderMessageReq 发送聊天
type CreateOrderMessageReq struct {
	MsgType  int16  `json:"msg_type"` // 1文字 2图片
	Content  string `json:"content"`
	ImageURL string `json:"image_url"` // 图片消息时填 OSS 路径或完整 URL
}

func (s *orderService) CreateOrderMessage(ctx *gin.Context, orderID uint, userID uint, req CreateOrderMessageReq) error {
	_, _, _, _, err := s.resolveOrderParticipant(ctx, orderID, userID)
	if err != nil {
		return err
	}
	// 已完成/已取消仍允许发消息，便于售后沟通
	if req.MsgType == constant.OrderMsgTypeOfficial {
		return errors.New("不能使用官方消息类型")
	}
	if req.MsgType != constant.OrderMsgTypeText && req.MsgType != constant.OrderMsgTypeImage {
		req.MsgType = constant.OrderMsgTypeText
	}
	if req.MsgType == constant.OrderMsgTypeImage {
		req.Content = strings.TrimSpace(req.Content)
		req.ImageURL = strings.TrimSpace(req.ImageURL)
		if req.ImageURL == "" {
			return errors.New("图片消息需要 image_url")
		}
		req.ImageURL = oss.PathForStorage(req.ImageURL)
	} else {
		req.Content = strings.TrimSpace(req.Content)
		if req.Content == "" {
			return errors.New("消息内容不能为空")
		}
	}
	sid := int(userID)
	m := &model.OrderMessage{
		OrderID:  orderID,
		SenderID: sid,
		MsgType:  req.MsgType,
		Content:  req.Content,
		ImageURL: req.ImageURL,
	}
	return dao.OrderMessage().Create(ctx.Request.Context(), m)
}

// SellerConfirmPayment 卖方确认已收款 → 送货上门/自提进入履约中；在线商品直接进入待买方确认收货
func (s *orderService) SellerConfirmPayment(ctx *gin.Context, orderID uint, userID uint) error {
	o, g, _, isSeller, err := s.resolveOrderParticipant(ctx, orderID, userID)
	if err != nil {
		return err
	}
	if !isSeller {
		return errno.ErrOrderNotParticipant
	}
	if o.OrderStatus != constant.OrderStatusAwaitSellerPaymentConfirm {
		return errno.ErrOrderInvalidState
	}
	now := time.Now()
	next := constant.OrderStatusFulfillment
	if g.GoodsType == constant.GoodsTypeOnline {
		next = constant.OrderStatusPendingBuyerConfirm
	}
	updates := map[string]interface{}{
		"order_status":     next,
		"seller_agreed_at": &now,
	}
	return dao.Order().UpdateColumns(ctx.Request.Context(), orderID, updates)
}

// ConfirmDeliveryReq 卖家确认送达（须至少一张送达凭证图）
type ConfirmDeliveryReq struct {
	DeliveryImages []string `json:"delivery_images"` // 必填，OSS URL 或完整 URL，至少 1 张
}

func (s *orderService) ConfirmDelivery(ctx *gin.Context, orderID uint, sellerID uint, req ConfirmDeliveryReq) error {
	o, g, _, isSeller, err := s.resolveOrderParticipant(ctx, orderID, sellerID)
	if err != nil {
		return err
	}
	if !isSeller {
		return errno.ErrOrderNotParticipant
	}
	// 在线商品双方同意后直接进入待确认收货，不再经过本步骤
	if g.GoodsType == constant.GoodsTypeOnline {
		return errno.ErrOrderInvalidState
	}
	if o.OrderStatus != constant.OrderStatusFulfillment {
		return errno.ErrOrderInvalidState
	}
	paths := make([]string, 0, len(req.DeliveryImages))
	for _, u := range req.DeliveryImages {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		paths = append(paths, oss.PathForStorage(u))
	}
	if len(paths) == 0 {
		return errors.New("请至少上传一张送达凭证图片")
	}
	updates := map[string]interface{}{
		"order_status":    constant.OrderStatusPendingBuyerConfirm,
		"delivery_images": pq.StringArray(paths),
	}
	return dao.Order().UpdateColumns(ctx.Request.Context(), orderID, updates)
}

// HelpPublisherPayReq 有偿求助发布者上传付酬截图
type HelpPublisherPayReq struct {
	PaymentImage string `json:"payment_image"` // 必填，OSS URL 或完整 URL
	Note         string `json:"note"`          // 可选附言，随截图一并发至会话
}

// HelpPublisherPay 有偿求助：发布者（schema seller）上传付酬截图 → 订单从"进行中"进入"待接单者确认"
//
// 同时把付酬说明文字 + 截图写入订单会话（两条 order_message），方便接单者核对后再确认。
func (s *orderService) HelpPublisherPay(ctx *gin.Context, orderID uint, userID uint, req HelpPublisherPayReq) error {
	o, g, _, isSeller, err := s.resolveOrderParticipant(ctx, orderID, userID)
	if err != nil {
		return err
	}
	if !isSeller {
		return errno.ErrOrderNotParticipant
	}
	if g.GoodsCategory != constant.GoodsCategoryHelp {
		return errno.ErrOrderInvalidState
	}
	if o.OrderStatus != constant.OrderStatusAwaitSellerPaymentConfirm {
		return errno.ErrOrderInvalidState
	}
	imgPath := strings.TrimSpace(req.PaymentImage)
	if imgPath == "" {
		return errors.New("请上传付酬截图")
	}
	imgPath = oss.PathForStorage(imgPath)
	sid := int(userID)
	note := strings.TrimSpace(req.Note)
	if note == "" {
		note = "我已支付酬劳，请核对下图付款凭证并确认收到。"
	}
	return pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := dao.Order().UpdateColumnsTx(ctx.Request.Context(), tx, orderID, map[string]interface{}{
			"order_status": constant.OrderStatusPendingBuyerConfirm,
		}); err != nil {
			return err
		}
		text := &model.OrderMessage{
			OrderID:  orderID,
			SenderID: sid,
			MsgType:  constant.OrderMsgTypeText,
			Content:  note,
		}
		if err := dao.OrderMessage().Create(ctx.Request.Context(), text); err != nil {
			return err
		}
		img := &model.OrderMessage{
			OrderID:  orderID,
			SenderID: sid,
			MsgType:  constant.OrderMsgTypeImage,
			ImageURL: imgPath,
		}
		return dao.OrderMessage().Create(ctx.Request.Context(), img)
	})
}

// ConfirmReceiptReq 买家确认收货
type ConfirmReceiptReq struct {
	Images []string `json:"images"` // 可选
}

func (s *orderService) ConfirmReceipt(ctx *gin.Context, orderID uint, buyerID uint, req ConfirmReceiptReq) error {
	o, g, isBuyer, _, err := s.resolveOrderParticipant(ctx, orderID, buyerID)
	if err != nil {
		return err
	}
	if !isBuyer {
		return errno.ErrOrderNotParticipant
	}
	canConfirm := o.OrderStatus == constant.OrderStatusPendingBuyerConfirm
	if !canConfirm && o.OrderStatus == constant.OrderStatusFulfillment && g.GoodsType == constant.GoodsTypePickup {
		canConfirm = true
	}
	if !canConfirm {
		return errno.ErrOrderInvalidState
	}
	if o.GoodsID == nil {
		return errno.ErrOrderNotFound
	}
	gid := uint(*o.GoodsID)
	paths := make([]string, 0, len(req.Images))
	for _, u := range req.Images {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		paths = append(paths, oss.PathForStorage(u))
	}
	now := time.Now()
	return pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"order_status": constant.OrderStatusCompleted,
			"completed_at": &now,
		}
		if len(paths) > 0 {
			updates["buyer_confirm_images"] = pq.StringArray(paths)
		}
		if err := dao.Order().UpdateColumnsTx(ctx.Request.Context(), tx, orderID, updates); err != nil {
			return err
		}
		if err := dao.Good().DecrementStockAfterSale(ctx.Request.Context(), tx, gid); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errno.ErrOrderInsufficientStock
			}
			return err
		}
		return nil
	})
}

// CancelOrder 取消订单（买卖双方，未完成前可取消的阶段）
func (s *orderService) CancelOrder(ctx *gin.Context, orderID uint, userID uint) error {
	o, _, _, _, err := s.resolveOrderParticipant(ctx, orderID, userID)
	if err != nil {
		return err
	}
	switch o.OrderStatus {
	case constant.OrderStatusAwaitBuyerLocation, constant.OrderStatusAwaitSellerPaymentConfirm, constant.OrderStatusFulfillment, constant.OrderStatusPendingBuyerConfirm:
		return dao.Order().UpdateColumns(ctx.Request.Context(), orderID, map[string]interface{}{
			"order_status": constant.OrderStatusCancelled,
		})
	default:
		return errno.ErrOrderInvalidState
	}
}

// ChatUnreadSummary 当前用户所有订单会话的未读条数（对方发送且 id > 已读游标）
func (s *orderService) ChatUnreadSummary(ctx *gin.Context, userID uint) (uint, map[uint]uint, error) {
	return dao.OrderMessageRead().UnreadCountsByUser(ctx.Request.Context(), userID)
}

// MarkOrderMessagesRead 将已读游标设为 max(已有, upTo)；upTo=0 时取当前会话最大消息 id（视为全部已读）
func (s *orderService) MarkOrderMessagesRead(ctx *gin.Context, orderID uint, userID uint, upTo uint) error {
	if _, _, _, _, err := s.resolveOrderParticipant(ctx, orderID, userID); err != nil {
		return err
	}
	if upTo == 0 {
		var err2 error
		upTo, err2 = dao.OrderMessage().MaxMessageIDByOrder(ctx.Request.Context(), orderID)
		if err2 != nil {
			return err2
		}
	}
	return dao.OrderMessageRead().UpsertLastReadGreatest(ctx.Request.Context(), userID, orderID, upTo)
}
