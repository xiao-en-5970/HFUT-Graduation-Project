package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// OrderMessagesList GET /orders/:id/messages
func OrderMessagesList(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "50"))
	list, total, err := service.Order().ListOrderMessages(ctx, uint(id), userID, page, pageSize)
	if err != nil {
		if errors.Is(err, errno.ErrOrderNotFound) || errors.Is(err, errno.ErrOrderNotParticipant) {
			reply.ReplyErrWithMessage(ctx, "订单不存在或无权查看")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
}

// OrderMessagesMarkRead POST /orders/:id/messages/read 标记已读（默认可不传 body，视为读到当前最后一条）
func OrderMessagesMarkRead(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	var req struct {
		LastReadMessageID uint `json:"last_read_message_id"`
	}
	_ = ctx.BindJSON(&req)
	if err := service.Order().MarkOrderMessagesRead(ctx, uint(id), userID, req.LastReadMessageID); err != nil {
		if errors.Is(err, errno.ErrOrderNotFound) || errors.Is(err, errno.ErrOrderNotParticipant) {
			reply.ReplyErrWithMessage(ctx, "订单不存在或无权操作")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// OrderMessageCreate POST /orders/:id/messages
func OrderMessageCreate(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	var req service.CreateOrderMessageReq
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.Order().CreateOrderMessage(ctx, uint(id), userID, req); err != nil {
		if errors.Is(err, errno.ErrOrderNotFound) || errors.Is(err, errno.ErrOrderNotParticipant) {
			reply.ReplyErrWithMessage(ctx, "订单不存在或无权操作")
			return
		}
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOK(ctx)
}

// OrderSellerConfirmPayment POST /orders/:id/seller-confirm-payment 卖方确认收款
func OrderSellerConfirmPayment(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.Order().SellerConfirmPayment(ctx, uint(id), userID); err != nil {
		if errors.Is(err, errno.ErrOrderNotFound) || errors.Is(err, errno.ErrOrderNotParticipant) {
			reply.ReplyErrWithMessage(ctx, "订单不存在或仅卖方可确认收款")
			return
		}
		if errors.Is(err, errno.ErrOrderInvalidState) {
			reply.ReplyErrWithMessage(ctx, "当前状态不可确认收款")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// OrderConfirmDelivery POST /orders/:id/confirm-delivery
func OrderConfirmDelivery(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	var req service.ConfirmDeliveryReq
	_ = ctx.BindJSON(&req)
	if err := service.Order().ConfirmDelivery(ctx, uint(id), userID, req); err != nil {
		if errors.Is(err, errno.ErrOrderNotFound) || errors.Is(err, errno.ErrOrderNotParticipant) {
			reply.ReplyErrWithMessage(ctx, "订单不存在或仅卖方可确认送达")
			return
		}
		if errors.Is(err, errno.ErrOrderInvalidState) {
			reply.ReplyErrWithMessage(ctx, "当前状态不可确认送达")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// OrderConfirmReceipt POST /orders/:id/confirm-receipt
func OrderConfirmReceipt(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	var req service.ConfirmReceiptReq
	_ = ctx.BindJSON(&req)
	if err := service.Order().ConfirmReceipt(ctx, uint(id), userID, req); err != nil {
		if errors.Is(err, errno.ErrOrderNotFound) || errors.Is(err, errno.ErrOrderNotParticipant) {
			reply.ReplyErrWithMessage(ctx, "订单不存在或仅买方可确认收货")
			return
		}
		if errors.Is(err, errno.ErrOrderInvalidState) {
			reply.ReplyErrWithMessage(ctx, "当前状态不可确认收货")
			return
		}
		if errors.Is(err, errno.ErrOrderInsufficientStock) {
			reply.ReplyErrWithMessage(ctx, "库存不足，无法完成订单")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// OrderCancel POST /orders/:id/cancel
func OrderCancel(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.Order().CancelOrder(ctx, uint(id), userID); err != nil {
		if errors.Is(err, errno.ErrOrderNotFound) || errors.Is(err, errno.ErrOrderNotParticipant) {
			reply.ReplyErrWithMessage(ctx, "订单不存在或无权操作")
			return
		}
		if errors.Is(err, errno.ErrOrderInvalidState) {
			reply.ReplyErrWithMessage(ctx, "当前状态不可取消")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}
