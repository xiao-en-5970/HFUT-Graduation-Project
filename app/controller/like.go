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

type LikeController struct {
	likeService *service.LikeService
}

// 确保 LikeController 实现了 LikeControllerInterface 接口
var _ LikeControllerInterface = (*LikeController)(nil)

// NewLikeController 创建点赞控制器
func NewLikeController() *LikeController {
	return &LikeController{
		likeService: service.NewLikeService(),
	}
}

// ToggleLike 切换点赞状态
func (c *LikeController) ToggleLike(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)

	var req request.LikeCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	isLiked, err := c.likeService.ToggleLike(userID, &req)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrLikeAlreadyExists, err.Error())
		return
	}

	action := "取消点赞"
	if isLiked {
		action = "点赞成功"
	}

	reply.ReplyOKWithMessageAndData(ctx, action, gin.H{
		"is_liked": isLiked,
	})
}

// IsLiked 检查是否已点赞
func (c *LikeController) IsLiked(ctx *gin.Context) {
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

	isLiked, err := c.likeService.IsLiked(userID, extType, extID)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}

	reply.ReplyOKWithData(ctx, gin.H{
		"is_liked": isLiked,
	})
}

