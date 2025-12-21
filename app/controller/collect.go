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

type CollectController struct {
	collectService *service.CollectService
}

// 确保 CollectController 实现了 CollectControllerInterface 接口
var _ CollectControllerInterface = (*CollectController)(nil)

// NewCollectController 创建收藏控制器
func NewCollectController() *CollectController {
	return &CollectController{
		collectService: service.NewCollectService(),
	}
}

// Create 创建收藏
func (c *CollectController) Create(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)

	var req request.CollectCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	collect, err := c.collectService.Create(userID, &req)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrCollectAlreadyExists, err.Error())
		return
	}

	reply.ReplyOKWithMessageAndData(ctx, "收藏成功", collect)
}

// GetByID 根据 ID 获取收藏
func (c *CollectController) GetByID(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	collect, err := c.collectService.GetByID(uint(id))
	if err != nil {
		reply.ReplyNotFound(ctx, errcode.ErrCollectNotFound)
		return
	}

	reply.ReplyOKWithData(ctx, collect)
}

// Delete 删除收藏（通过收藏ID）
func (c *CollectController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	userID := middleware.GetUserID(ctx)

	if err := c.collectService.Delete(userID, uint(id)); err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrCollectNoPermission, err.Error())
		return
	}

	reply.ReplyOKWithMessage(ctx, "取消收藏成功")
}

// DeleteByExt 根据关联对象删除收藏
func (c *CollectController) DeleteByExt(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)

	var req request.CollectDeleteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	if err := c.collectService.DeleteByExt(userID, &req); err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrCollectNoPermission, err.Error())
		return
	}

	reply.ReplyOKWithMessage(ctx, "取消收藏成功")
}

// List 获取收藏列表
func (c *CollectController) List(ctx *gin.Context) {
	var req request.CollectListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	result, err := c.collectService.List(&req)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}

	reply.ReplyOKWithData(ctx, result)
}

// IsCollected 检查是否已收藏
func (c *CollectController) IsCollected(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)

	extTypeStr := ctx.Query("ext_type")
	extIDStr := ctx.Query("ext_id")

	if extTypeStr == "" || extIDStr == "" {
		reply.ReplyInvalidParams(ctx, nil)
		return
	}

	extType, err := strconv.Atoi(extTypeStr)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	extID, err := strconv.Atoi(extIDStr)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	isCollected, err := c.collectService.IsCollected(userID, extType, extID)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}

	reply.ReplyOKWithData(ctx, gin.H{
		"is_collected": isCollected,
	})
}

