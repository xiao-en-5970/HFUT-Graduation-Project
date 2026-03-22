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

// CreateOrderReq 创建订单：默认「待买方付款下单」可聊天；可选 buyer_claim_paid 直接进入待卖方确认收款
type CreateOrderReq struct {
	GoodsID        uint   `json:"goods_id" binding:"required"`
	ReceiverAddr   string `json:"receiver_addr"`    // 可选
	SenderAddr     string `json:"sender_addr"`      // 可选
	BuyerClaimPaid bool   `json:"buyer_claim_paid"` // 为 true 时表示创建同时已线下付款，进入待卖方确认收款
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
	o := &model.Order{
		UserID:       &uid,
		GoodsID:      &gid,
		OrderStatus:  constant.OrderStatusPendingBuyerPayment,
		ReceiverAddr: receiver,
		SenderAddr:   sender,
	}
	if req.BuyerClaimPaid {
		now := time.Now()
		o.OrderStatus = constant.OrderStatusAwaitSellerPaymentConfirm
		o.BuyerAgreedAt = &now
	}
	// 仅「送货上门」在买卖地址齐全时计算步行距离；自提/在线不计算
	if good.GoodsType == constant.GoodsTypeDelivery {
		if sender != "" && receiver != "" {
			if d := computeOrderDistanceMeters(ctx, sender, receiver); d != nil {
				o.DistanceMeters = d
			}
		}
	}
	return dao.Order().Create(ctx.Request.Context(), o)
}

// computeOrderDistanceMeters 发货地→收货地步行规划距离（米）。需配置 AMAP_KEY；地址不全或 API 失败时返回 nil，不阻断下单。
func computeOrderDistanceMeters(ctx *gin.Context, senderAddr, receiverAddr string) *int {
	if config.AmapKey == "" || senderAddr == "" || receiverAddr == "" {
		return nil
	}
	m, err := amap.DistanceBetweenAddresses(ctx.Request.Context(), config.AmapKey, senderAddr, receiverAddr)
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

// UpdateSellerInfo 卖家仅可更新发货地址；订单为待下单/正在派送时可改。不再允许任意改 order_status。
func (s *orderService) UpdateSellerInfo(ctx *gin.Context, id uint, sellerID uint, senderAddr string) error {
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
	if o.OrderStatus != constant.OrderStatusPendingBuyerPayment &&
		o.OrderStatus != constant.OrderStatusAwaitSellerPaymentConfirm &&
		o.OrderStatus != constant.OrderStatusFulfillment {
		return errno.ErrOrderInvalidState
	}
	updates := make(map[string]interface{})
	senderAddr = strings.TrimSpace(senderAddr)
	if senderAddr != "" {
		updates["sender_addr"] = senderAddr
		if g.GoodsType == constant.GoodsTypeDelivery {
			receiver := strings.TrimSpace(o.ReceiverAddr)
			if d := computeOrderDistanceMeters(ctx, senderAddr, receiver); d != nil {
				updates["distance_meters"] = *d
			}
		}
	}
	if len(updates) == 0 {
		return nil
	}
	return dao.Order().UpdateColumns(ctx.Request.Context(), id, updates)
}
