package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// AdminOrderList GET /admin/orders
func AdminOrderList(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	schoolID, _ := strconv.ParseUint(ctx.DefaultQuery("school_id", "0"), 10, 32)
	includeInvalid := ctx.Query("include_invalid") != "0"
	list, total, err := dao.Order().ListAllForAdmin(ctx.Request.Context(), page, pageSize, uint(schoolID), includeInvalid)
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

// AdminOrderGet GET /admin/orders/:id
func AdminOrderGet(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	o, err := dao.Order().GetByID(ctx.Request.Context(), id)
	if err != nil || o == nil {
		reply.ReplyErrWithMessage(ctx, "订单不存在")
		return
	}
	reply.ReplyOKWithData(ctx, orderToMap(ctx, o))
}

// AdminOrderMessages GET /admin/orders/:id/messages
func AdminOrderMessages(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "100"))
	list, total, err := service.Order().ListOrderMessagesAdmin(ctx, id, page, pageSize)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
}
