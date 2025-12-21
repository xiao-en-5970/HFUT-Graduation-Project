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

type GoodController struct {
	goodService *service.GoodService
}

// 确保 GoodController 实现了 GoodControllerInterface 接口
var _ GoodControllerInterface = (*GoodController)(nil)

// NewGoodController 创建商品控制器
func NewGoodController() *GoodController {
	return &GoodController{
		goodService: service.NewGoodService(),
	}
}

// Create 创建商品
func (c *GoodController) Create(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)

	var req request.GoodCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	good, err := c.goodService.Create(userID, &req)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrGoodCreateFailed, err.Error())
		return
	}

	reply.ReplyOKWithMessageAndData(ctx, "创建成功", good)
}

// GetByID 根据 ID 获取商品
func (c *GoodController) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	good, err := c.goodService.GetByID(uint(id))
	if err != nil {
		reply.ReplyNotFound(ctx, errcode.ErrGoodNotFound)
		return
	}

	reply.ReplyOKWithData(ctx, good)
}

// Update 更新商品
func (c *GoodController) Update(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	userID := middleware.GetUserID(ctx)

	var req request.GoodUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	good, err := c.goodService.Update(userID, uint(id), &req)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrGoodUpdateFailed, err.Error())
		return
	}

	reply.ReplyOKWithMessageAndData(ctx, "更新成功", good)
}

// Delete 删除商品
func (c *GoodController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	userID := middleware.GetUserID(ctx)

	if err := c.goodService.Delete(userID, uint(id)); err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrGoodDeleteFailed, err.Error())
		return
	}

	reply.ReplyOKWithMessage(ctx, "删除成功")
}

// List 获取商品列表
func (c *GoodController) List(ctx *gin.Context) {
	var req request.GoodListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	result, err := c.goodService.List(&req)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}

	reply.ReplyOKWithData(ctx, result)
}
