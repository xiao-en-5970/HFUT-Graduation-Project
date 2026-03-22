package service

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/amap"
)

type orderService struct{}

type CreateOrderReq struct {
	GoodsID      uint   `json:"goods_id" binding:"required"`
	ReceiverAddr string `json:"receiver_addr" binding:"required"` // 收货地址
	SenderAddr   string `json:"sender_addr"`                      // 发货地址（可选，卖家可后续填写）
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
	// 不能买自己的商品
	if good.UserID != nil && uint(*good.UserID) == buyerID {
		return 0, errors.New("不能购买自己发布的商品")
	}
	uid := int(buyerID)
	gid := int(req.GoodsID)
	o := &model.Order{
		UserID:       &uid,
		GoodsID:      &gid,
		OrderStatus:  1, // 待支付
		ReceiverAddr: req.ReceiverAddr,
		SenderAddr:   req.SenderAddr,
	}
	if d := computeOrderDistanceMeters(ctx, strings.TrimSpace(req.SenderAddr), strings.TrimSpace(req.ReceiverAddr)); d != nil {
		o.DistanceMeters = d
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

func (s *orderService) UpdateSellerInfo(ctx *gin.Context, id uint, sellerID uint, senderAddr string, orderStatus *int16) error {
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
	updates := make(map[string]interface{})
	if senderAddr != "" {
		updates["sender_addr"] = senderAddr
	}
	if orderStatus != nil && *orderStatus >= 1 && *orderStatus <= 5 {
		updates["order_status"] = *orderStatus
	}
	// 卖家填写或修改发货地址后，用高德计算与收货地的步行距离
	if senderAddr != "" {
		receiver := strings.TrimSpace(o.ReceiverAddr)
		sender := strings.TrimSpace(senderAddr)
		if d := computeOrderDistanceMeters(ctx, sender, receiver); d != nil {
			updates["distance_meters"] = *d
		}
	}
	if len(updates) == 0 {
		return nil
	}
	return dao.Order().UpdateColumns(ctx.Request.Context(), id, updates)
}
