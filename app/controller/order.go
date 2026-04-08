package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// OrderCreate 创建订单 POST /orders
func OrderCreate(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	schoolID := middleware.GetSchoolID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	var req service.CreateOrderReq
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	id, err := service.Order().Create(ctx, userID, schoolID, req)
	if err != nil {
		if errors.Is(err, errno.ErrOrderGoodNotFound) {
			reply.ReplyErrWithMessage(ctx, "商品不存在或已下架")
			return
		}
		if errors.Is(err, errno.ErrOrderGoodNotOnSale) {
			reply.ReplyErrWithMessage(ctx, "商品未上架")
			return
		}
		if errors.Is(err, errno.ErrOrderInsufficientStock) {
			reply.ReplyErrWithMessage(ctx, "库存不足")
			return
		}
		if errors.Is(err, errno.ErrUserLocationNotFound) {
			reply.ReplyErrWithMessage(ctx, "收货地址不存在或已删除")
			return
		}
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"id": id})
}

// OrderLocationUpdate POST /orders/:id/location 统一更新买方收货地 / 卖方发货地 / 卖方确认或拒绝买方改址
func OrderLocationUpdate(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	var req service.OrderLocationUpdateReq
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.Order().OrderLocationUpdate(ctx, uint(id), userID, req); err != nil {
		if errors.Is(err, errno.ErrOrderNotFound) || errors.Is(err, errno.ErrOrderNotParticipant) {
			reply.ReplyErrWithMessage(ctx, "订单不存在或无权操作")
			return
		}
		if errors.Is(err, errno.ErrOrderInvalidState) {
			reply.ReplyErrWithMessage(ctx, "当前状态不可修改地址")
			return
		}
		if errors.Is(err, errno.ErrUserLocationNotFound) {
			reply.ReplyErrWithMessage(ctx, "收货地址不存在或已删除")
			return
		}
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOK(ctx)
}

// OrderList 我的订单（买家视角）GET /orders
func OrderList(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	list, total, err := service.Order().ListByBuyer(ctx, userID, page, pageSize)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	out := make([]map[string]interface{}, len(list))
	for i, o := range list {
		out[i] = orderToMap(ctx, o)
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": out, "total": total, "page": page, "page_size": pageSize})
}

func orderStatusLabel(st int16, goodsType int16) string {
	switch st {
	case constant.OrderStatusAwaitBuyerLocation:
		return "待买方完善地址与付款"
	case constant.OrderStatusAwaitSellerPaymentConfirm:
		return "待卖方确认收款"
	case constant.OrderStatusFulfillment:
		if goodsType == constant.GoodsTypePickup {
			return "待买方自提"
		}
		return "正在派送"
	case constant.OrderStatusPendingBuyerConfirm:
		return "待买方确认收货"
	case constant.OrderStatusCompleted:
		return "已完成"
	case constant.OrderStatusCancelled:
		return "已取消"
	default:
		return ""
	}
}

func orderToMap(ctx *gin.Context, o *model.Order) map[string]interface{} {
	var gt int16
	m := map[string]interface{}{
		"id": o.ID, "user_id": o.UserID, "goods_id": o.GoodsID,
		"status":        o.Status,
		"receiver_addr": o.ReceiverAddr, "sender_addr": o.SenderAddr,
		"buyer_agreed_at": o.BuyerAgreedAt, "seller_agreed_at": o.SellerAgreedAt,
		"delivery_images":      oss.TransformImageURLs([]string(o.DeliveryImages)),
		"buyer_confirm_images": oss.TransformImageURLs([]string(o.BuyerConfirmImages)),
		"completed_at":         o.CompletedAt,
		"created_at":           o.CreatedAt,
	}
	if o.ReceiverUserLocationID != nil {
		m["receiver_user_location_id"] = *o.ReceiverUserLocationID
	} else {
		m["receiver_user_location_id"] = nil
	}
	if o.DistanceMeters != nil {
		m["distance_meters"] = *o.DistanceMeters
	} else {
		m["distance_meters"] = nil
	}
	if o.ReceiverLat != nil {
		m["receiver_lat"] = *o.ReceiverLat
	} else {
		m["receiver_lat"] = nil
	}
	if o.ReceiverLng != nil {
		m["receiver_lng"] = *o.ReceiverLng
	} else {
		m["receiver_lng"] = nil
	}
	if o.PendingReceiverUserLocationID != nil {
		m["pending_receiver_user_location_id"] = *o.PendingReceiverUserLocationID
	} else {
		m["pending_receiver_user_location_id"] = nil
	}
	m["pending_receiver_addr"] = o.PendingReceiverAddr
	if o.PendingReceiverLat != nil {
		m["pending_receiver_lat"] = *o.PendingReceiverLat
	} else {
		m["pending_receiver_lat"] = nil
	}
	if o.PendingReceiverLng != nil {
		m["pending_receiver_lng"] = *o.PendingReceiverLng
	} else {
		m["pending_receiver_lng"] = nil
	}
	if o.SenderLat != nil {
		m["sender_lat"] = *o.SenderLat
	} else {
		m["sender_lat"] = nil
	}
	if o.SenderLng != nil {
		m["sender_lng"] = *o.SenderLng
	} else {
		m["sender_lng"] = nil
	}
	if o.GoodsID != nil && *o.GoodsID > 0 {
		g, err := dao.Good().GetByID(ctx.Request.Context(), uint(*o.GoodsID))
		if err == nil && g != nil {
			gt = g.GoodsType
			g.Images = oss.TransformImageURLs(g.Images)
			ga := effectiveGoodAddr(g)
			gm := map[string]interface{}{
				"id": g.ID, "title": g.Title, "images": g.Images, "price": g.Price,
				"user_id":    g.UserID,
				"goods_type": g.GoodsType, "goods_type_label": constant.GoodsTypeLabel(g.GoodsType),
				"goods_addr":  ga,
				"pickup_addr": ga,
			}
			if g.GoodsLat != nil {
				gm["goods_lat"] = *g.GoodsLat
			} else {
				gm["goods_lat"] = nil
			}
			if g.GoodsLng != nil {
				gm["goods_lng"] = *g.GoodsLng
			} else {
				gm["goods_lng"] = nil
			}
			m["good"] = gm
		}
	}
	m["order_status"] = o.OrderStatus
	m["order_status_label"] = orderStatusLabel(o.OrderStatus, gt)
	return m
}

// OrderListSold 我卖出的（卖家视角）GET /orders/sold
func OrderListSold(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	list, total, err := service.Order().ListBySeller(ctx, userID, page, pageSize)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	out := make([]map[string]interface{}, len(list))
	for i, o := range list {
		out[i] = orderToMap(ctx, o)
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": out, "total": total, "page": page, "page_size": pageSize})
}

// OrderGet 订单详情 GET /orders/:id
func OrderGet(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	o, err := service.Order().GetByID(ctx, uint(id))
	if err != nil || o == nil {
		reply.ReplyErrWithMessage(ctx, "订单不存在")
		return
	}
	if o.UserID != nil && uint(*o.UserID) != userID {
		g, _ := dao.Good().GetByID(ctx.Request.Context(), uint(*o.GoodsID))
		if g == nil || g.UserID == nil || uint(*g.UserID) != userID {
			reply.ReplyErrWithMessage(ctx, "订单不存在")
			return
		}
	}
	reply.ReplyOKWithData(ctx, orderToMap(ctx, o))
}

// OrderUpdate 卖家更新订单 PUT /orders/:id
func OrderUpdate(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	var body service.UpdateSellerAddrReq
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.Order().UpdateSellerInfo(ctx, uint(id), userID, body); err != nil {
		if errors.Is(err, errno.ErrOrderInvalidState) {
			reply.ReplyErrWithMessage(ctx, "当前订单状态不可修改发货地址")
			return
		}
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOK(ctx)
}
