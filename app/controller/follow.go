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

type FollowController struct {
	followService *service.FollowService
}

// 确保 FollowController 实现了 FollowControllerInterface 接口
var _ FollowControllerInterface = (*FollowController)(nil)

// NewFollowController 创建关注控制器
func NewFollowController() *FollowController {
	return &FollowController{
		followService: service.NewFollowService(),
	}
}

// Follow 关注用户
func (c *FollowController) Follow(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)

	var req request.FollowRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	follow, err := c.followService.Follow(userID, req.FollowID)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrFollowAlreadyExists, err.Error())
		return
	}

	reply.ReplyOKWithMessageAndData(ctx, "关注成功", follow)
}

// Unfollow 取消关注
func (c *FollowController) Unfollow(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)

	var req request.FollowRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	if err := c.followService.Unfollow(userID, req.FollowID); err != nil {
		reply.ReplyErrWithMessage(ctx, errcode.ErrFollowNoPermission, err.Error())
		return
	}

	reply.ReplyOKWithMessage(ctx, "取消关注成功")
}

// GetFollowingList 获取关注列表（我关注的人）
func (c *FollowController) GetFollowingList(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)

	var req request.FollowListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	// 如果未指定 user_id，使用当前登录用户
	if req.UserID == 0 {
		req.UserID = userID
	}

	result, err := c.followService.GetFollowingList(&req)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}

	reply.ReplyOKWithData(ctx, result)
}

// GetFollowersList 获取粉丝列表（关注我的人）
func (c *FollowController) GetFollowersList(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)

	var req request.FollowListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	// 如果未指定 user_id，使用当前登录用户
	if req.UserID == 0 {
		req.UserID = userID
	}

	result, err := c.followService.GetFollowersList(&req)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}

	reply.ReplyOKWithData(ctx, result)
}

// GetFollowCount 获取关注和粉丝数量
func (c *FollowController) GetFollowCount(ctx *gin.Context) {
	userIDStr := ctx.Param("user_id")
	var userID uint

	if userIDStr == "" {
		// 如果未指定 user_id，使用当前登录用户
		userID = middleware.GetUserID(ctx)
	} else {
		id, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			reply.ReplyInvalidParams(ctx, err)
			return
		}
		userID = uint(id)
	}

	result, err := c.followService.GetFollowCount(userID)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}

	reply.ReplyOKWithData(ctx, result)
}

// IsFollowing 检查是否已关注
func (c *FollowController) IsFollowing(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)

	followIDStr := ctx.Query("follow_id")
	if followIDStr == "" {
		reply.ReplyInvalidParams(ctx, nil)
		return
	}

	followID, err := strconv.ParseUint(followIDStr, 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}

	isFollowing, err := c.followService.IsFollowing(userID, uint(followID))
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}

	reply.ReplyOKWithData(ctx, gin.H{
		"is_following": isFollowing,
	})
}

