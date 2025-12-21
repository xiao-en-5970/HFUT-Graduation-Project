package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

type OrderController struct {
	orderService *service.OrderService
}

// 确保 OrderController 实现了 OrderControllerInterface 接口
var _ OrderControllerInterface = (*OrderController)(nil)

// NewOrderController 创建订单控制器
func NewOrderController() *OrderController {
	return &OrderController{
		orderService: service.NewOrderService(),
	}
}

// Create 创建订单
func (c *OrderController) Create(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)

	var req request.OrderCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	order, err := c.orderService.Create(userID, &req)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrOrderCreateFailed, err.Error())
		return
	}

	reply.ReplyOKWithMessageAndData(ctx, "创建订单成功", order)
}

// GetByID 根据 ID 获取订单
func (c *OrderController) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	order, err := c.orderService.GetByID(uint(id))
	if err != nil {
		reply.ReplyNotFound(ctx, errcode.ErrOrderNotFound)
		return
	}

	reply.ReplyOKWithData(ctx, order)
}

// Update 更新订单
func (c *OrderController) Update(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	userID := middleware.GetUserID(ctx)

	var req request.OrderUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	order, err := c.orderService.Update(userID, uint(id), &req)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrOrderUpdateFailed, err.Error())
		return
	}

	reply.ReplyOKWithMessageAndData(ctx, "更新成功", order)
}

// Delete 删除订单
func (c *OrderController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	userID := middleware.GetUserID(ctx)

	if err := c.orderService.Delete(userID, uint(id)); err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrOrderNoPermission, err.Error())
		return
	}

	reply.ReplyOKWithMessage(ctx, "删除成功")
}

// List 获取订单列表
func (c *OrderController) List(ctx *gin.Context) {
	var req request.OrderListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	result, err := c.orderService.List(&req)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}

	reply.ReplyOKWithData(ctx, result)
}

