package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// validLikeExtTypes 点赞支持的 extType：1帖子 2提问 3回答 4商品 5评论
var validLikeExtTypes = map[int]bool{
	constant.ExtTypePost: true, constant.ExtTypeQuestion: true, constant.ExtTypeAnswer: true, constant.ExtTypeGoods: true, constant.ExtTypeComment: true,
}

// LikeAdd 统一点赞接口：POST /like/:extType/:id
func LikeAdd(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	schoolID := middleware.GetSchoolID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	// schoolID=0 时仅可对公开文章点赞
	extType, extID, ok := parseLikeExtTypeAndID(ctx)
	if !ok {
		return
	}
	if !validLikeExtTypes[extType] {
		reply.ReplyErrWithMessage(ctx, "extType 无效，点赞仅支持 1帖子 2提问 3回答 4商品 5评论")
		return
	}
	if extType == constant.ExtTypeComment {
		if err := service.Like().AddComment(ctx, userID, extID); err != nil {
			if errors.Is(err, errno.ErrLikeArticleNotFound) {
				reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
				return
			}
			reply.ReplyInternalError(ctx, err)
			return
		}
		reply.ReplyOK(ctx)
		return
	}
	if err := service.Like().AddArticle(ctx, userID, schoolID, extID, extType); err != nil {
		if errors.Is(err, errno.ErrLikeArticleNotFound) {
			reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// LikeRemove 统一取消点赞：DELETE /like/:extType/:id
func LikeRemove(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	schoolID := middleware.GetSchoolID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	extType, extID, ok := parseLikeExtTypeAndID(ctx)
	if !ok {
		return
	}
	if !validLikeExtTypes[extType] {
		reply.ReplyErrWithMessage(ctx, "extType 无效，点赞仅支持 1帖子 2提问 3回答 4商品 5评论")
		return
	}
	if extType == constant.ExtTypeComment {
		if err := service.Like().RemoveComment(ctx, userID, extID); err != nil {
			if errors.Is(err, errno.ErrLikeArticleNotFound) {
				reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
				return
			}
			reply.ReplyInternalError(ctx, err)
			return
		}
		reply.ReplyOK(ctx)
		return
	}
	if err := service.Like().RemoveArticle(ctx, userID, schoolID, extID, extType); err != nil {
		if errors.Is(err, errno.ErrLikeArticleNotFound) {
			reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

func parseLikeExtTypeAndID(ctx *gin.Context) (extType int, id uint, ok bool) {
	extTypeStr := ctx.Param("extType")
	extTypeNum, err := strconv.Atoi(extTypeStr)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, "extType 必须为整数")
		return 0, 0, false
	}
	idStr := ctx.Param("id")
	extID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return 0, 0, false
	}
	return extTypeNum, uint(extID), true
}
