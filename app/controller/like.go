package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// ArticleLikeHandlers 文章点赞处理器，按类型特化（帖子/提问/回答）
func ArticleLikeHandlers(extType int) struct {
	Add, Remove gin.HandlerFunc
} {
	return struct {
		Add, Remove gin.HandlerFunc
	}{
		Add: func(ctx *gin.Context) {
			userID := middleware.GetUserID(ctx)
			schoolID := middleware.GetSchoolID(ctx)
			if userID == 0 {
				reply.ReplyUnauthorized(ctx)
				return
			}
			if schoolID == 0 {
				reply.ReplyErrWithMessage(ctx, "请先绑定学校")
				return
			}
			idStr := ctx.Param("id")
			articleID, err := strconv.ParseUint(idStr, 10, 32)
			if err != nil {
				reply.ReplyInvalidParams(ctx, err)
				return
			}
			if err := service.Like().AddArticle(ctx, userID, schoolID, uint(articleID), extType); err != nil {
				if errors.Is(err, service.ErrLikeArticleNotFound) {
					reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
					return
				}
				if errors.Is(err, service.ErrLikeAlreadyLiked) {
					reply.ReplyErrWithMessage(ctx, "已点赞")
					return
				}
				reply.ReplyInternalError(ctx, err)
				return
			}
			reply.ReplyOK(ctx)
		},
		Remove: func(ctx *gin.Context) {
			userID := middleware.GetUserID(ctx)
			schoolID := middleware.GetSchoolID(ctx)
			if userID == 0 {
				reply.ReplyUnauthorized(ctx)
				return
			}
			if schoolID == 0 {
				reply.ReplyErrWithMessage(ctx, "请先绑定学校")
				return
			}
			idStr := ctx.Param("id")
			articleID, err := strconv.ParseUint(idStr, 10, 32)
			if err != nil {
				reply.ReplyInvalidParams(ctx, err)
				return
			}
			if err := service.Like().RemoveArticle(ctx, userID, schoolID, uint(articleID), extType); err != nil {
				if errors.Is(err, service.ErrLikeArticleNotFound) {
					reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
					return
				}
				if errors.Is(err, service.ErrLikeNotLiked) {
					reply.ReplyErrWithMessage(ctx, "未点赞")
					return
				}
				reply.ReplyInternalError(ctx, err)
				return
			}
			reply.ReplyOK(ctx)
		},
	}
}

var (
	PostLikeHandlers     = ArticleLikeHandlers(constant.ExtTypePost)
	QuestionLikeHandlers = ArticleLikeHandlers(constant.ExtTypeQuestion)
	AnswerLikeHandlers   = ArticleLikeHandlers(constant.ExtTypeAnswer)
)
