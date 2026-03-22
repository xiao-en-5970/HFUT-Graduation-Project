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
	o, _, _, _, err := s.resolveOrderParticipant(ctx, orderID, userID)
	if err != nil {
		return err
	}
	if o.OrderStatus == constant.OrderStatusCompleted || o.OrderStatus == constant.OrderStatusCancelled {
		return errno.ErrOrderInvalidState
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

// AgreeToDeliver 买卖双方各自点击同意，双方均同意后进入「正在派送」
func (s *orderService) AgreeToDeliver(ctx *gin.Context, orderID uint, userID uint) error {
	o, _, isBuyer, isSeller, err := s.resolveOrderParticipant(ctx, orderID, userID)
	if err != nil {
		return err
	}
	if o.OrderStatus != constant.OrderStatusPendingIntent {
		return errno.ErrOrderInvalidState
	}
	now := time.Now()
	updates := make(map[string]interface{})
	if isBuyer {
		if o.BuyerAgreedAt != nil {
			return nil // 幂等
		}
		updates["buyer_agreed_at"] = &now
	}
	if isSeller {
		if o.SellerAgreedAt != nil {
			return nil
		}
		updates["seller_agreed_at"] = &now
	}
	if len(updates) == 0 {
		return nil
	}
	// 先写入本方同意时间
	if err := dao.Order().UpdateColumns(ctx.Request.Context(), orderID, updates); err != nil {
		return err
	}
	o2, err := dao.Order().GetByID(ctx.Request.Context(), orderID)
	if err != nil || o2 == nil {
		return err
	}
	if o2.BuyerAgreedAt != nil && o2.SellerAgreedAt != nil {
		if o2.GoodsID == nil {
			return errno.ErrOrderNotFound
		}
		g2, err := dao.Good().GetByID(ctx.Request.Context(), uint(*o2.GoodsID))
		if err != nil || g2 == nil {
			return err
		}
		next := constant.OrderStatusDelivering
		// 在线商品：无实体派送，双方同意后即视为卖方已发货，进入待买方确认收货
		if g2.GoodsType == constant.GoodsTypeOnline {
			next = constant.OrderStatusPendingBuyerConfirm
		}
		return dao.Order().UpdateColumns(ctx.Request.Context(), orderID, map[string]interface{}{
			"order_status": next,
		})
	}
	return nil
}

// ConfirmDeliveryReq 卖家确认送达
type ConfirmDeliveryReq struct {
	DeliveryImages []string `json:"delivery_images"` // 可选，OSS URL
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
	if o.OrderStatus != constant.OrderStatusDelivering {
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
	updates := map[string]interface{}{
		"order_status": constant.OrderStatusPendingBuyerConfirm,
	}
	if len(paths) > 0 {
		updates["delivery_images"] = pq.StringArray(paths)
	}
	return dao.Order().UpdateColumns(ctx.Request.Context(), orderID, updates)
}

// ConfirmReceiptReq 买家确认收货
type ConfirmReceiptReq struct {
	Images []string `json:"images"` // 可选
}

func (s *orderService) ConfirmReceipt(ctx *gin.Context, orderID uint, buyerID uint, req ConfirmReceiptReq) error {
	o, _, isBuyer, _, err := s.resolveOrderParticipant(ctx, orderID, buyerID)
	if err != nil {
		return err
	}
	if !isBuyer {
		return errno.ErrOrderNotParticipant
	}
	if o.OrderStatus != constant.OrderStatusPendingBuyerConfirm {
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

// CancelOrder 取消订单（买卖双方，仅待下单或正在派送阶段）
func (s *orderService) CancelOrder(ctx *gin.Context, orderID uint, userID uint) error {
	o, _, _, _, err := s.resolveOrderParticipant(ctx, orderID, userID)
	if err != nil {
		return err
	}
	switch o.OrderStatus {
	case constant.OrderStatusPendingIntent, constant.OrderStatusDelivering:
		return dao.Order().UpdateColumns(ctx.Request.Context(), orderID, map[string]interface{}{
			"order_status": constant.OrderStatusCancelled,
		})
	default:
		return errno.ErrOrderInvalidState
	}
}
