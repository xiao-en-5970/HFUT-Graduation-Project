// Package controller 的 follow.go 是关注 / 粉丝相关 HTTP handler。
//
// 路由：
//
//	POST   /user/:id/follow       关注 user :id
//	DELETE /user/:id/follow       取消关注 user :id
//	GET    /user/:id/following    user :id 关注的人（分页）
//	GET    /user/:id/followers    user :id 的粉丝（分页）
//
// 鉴权：全部需 JWT。viewerID 来自 token，target id 来自 URL 路径。
package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// UserFollow POST /user/:id/follow——viewer 关注 :id。
func UserFollow(ctx *gin.Context) {
	viewerID := middleware.GetUserID(ctx)
	if viewerID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	targetID, err := parseUserID(ctx)
	if err != nil {
		return
	}
	res, err := service.Follow(ctx.Request.Context(), viewerID, targetID)
	if err != nil {
		handleFollowError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, res)
}

// UserUnfollow DELETE /user/:id/follow——viewer 取消关注 :id。
func UserUnfollow(ctx *gin.Context) {
	viewerID := middleware.GetUserID(ctx)
	if viewerID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	targetID, err := parseUserID(ctx)
	if err != nil {
		return
	}
	res, err := service.Unfollow(ctx.Request.Context(), viewerID, targetID)
	if err != nil {
		handleFollowError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, res)
}

// UserListFollowing GET /user/:id/following?page=1&pageSize=20
func UserListFollowing(ctx *gin.Context) {
	viewerID := middleware.GetUserID(ctx)
	targetID, err := parseUserID(ctx)
	if err != nil {
		return
	}
	page, pageSize := parsePagination(ctx)
	res, err := service.ListFollowing(ctx.Request.Context(), targetID, viewerID, page, pageSize)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, res)
}

// UserListFollowers GET /user/:id/followers?page=1&pageSize=20
func UserListFollowers(ctx *gin.Context) {
	viewerID := middleware.GetUserID(ctx)
	targetID, err := parseUserID(ctx)
	if err != nil {
		return
	}
	page, pageSize := parsePagination(ctx)
	res, err := service.ListFollowers(ctx.Request.Context(), targetID, viewerID, page, pageSize)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, res)
}

func parseUserID(ctx *gin.Context) (uint, error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		reply.ReplyErrWithMessage(ctx, "用户ID无效")
		return 0, errors.New("invalid user id")
	}
	return uint(id), nil
}

func parsePagination(ctx *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return
}

func handleFollowError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, dao.ErrSelfFollow):
		reply.ReplyErrWithMessage(ctx, "不能关注自己")
	case errors.Is(err, service.ErrUserNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		reply.ReplyNotFound(ctx, errcode.ErrUserNotFound)
	default:
		reply.ReplyInternalError(ctx, err)
	}
}
