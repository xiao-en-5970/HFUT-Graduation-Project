package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
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
		if errors.Is(err, service.ErrOrderGoodNotFound) {
			reply.ReplyErrWithMessage(ctx, "商品不存在或已下架")
			return
		}
		if errors.Is(err, service.ErrOrderGoodNotOnSale) {
			reply.ReplyErrWithMessage(ctx, "商品未上架")
			return
		}
		if errors.Is(err, service.ErrOrderInsufficientStock) {
			reply.ReplyErrWithMessage(ctx, "库存不足")
			return
		}
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"id": id})
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

func orderToMap(ctx *gin.Context, o *model.Order) map[string]interface{} {
	m := map[string]interface{}{
		"id": o.ID, "user_id": o.UserID, "goods_id": o.GoodsID,
		"order_status": o.OrderStatus, "receiver_addr": o.ReceiverAddr, "sender_addr": o.SenderAddr,
		"created_at": o.CreatedAt,
	}
	if o.DistanceMeters != nil {
		m["distance_meters"] = *o.DistanceMeters
	} else {
		m["distance_meters"] = nil
	}
	if o.GoodsID != nil && *o.GoodsID > 0 {
		g, err := dao.Good().GetByID(ctx.Request.Context(), uint(*o.GoodsID))
		if err == nil && g != nil {
			g.Images = oss.TransformImageURLs(g.Images)
			m["good"] = map[string]interface{}{
				"id": g.ID, "title": g.Title, "images": g.Images, "price": g.Price,
			}
		}
	}
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
	var body struct {
		SenderAddr  string `json:"sender_addr"`
		OrderStatus *int16 `json:"order_status"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.Order().UpdateSellerInfo(ctx, uint(id), userID, body.SenderAddr, body.OrderStatus); err != nil {
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOK(ctx)
}
