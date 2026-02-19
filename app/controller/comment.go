package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// ArticleCommentHandlers 返回文章评论区处理器，按类型特化（帖子/提问/回答）
func ArticleCommentHandlers(articleType int) struct {
	Create, ListComments, ListReplies gin.HandlerFunc
} {
	return struct {
		Create, ListComments, ListReplies gin.HandlerFunc
	}{
		Create: func(ctx *gin.Context) {
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
			// 校验文章存在且类型匹配
			_, err = dao.Article().GetByIDWithSchoolAndType(ctx.Request.Context(), uint(articleID), schoolID, articleType)
			if err != nil {
				reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
				return
			}
			var req service.CreateCommentReq
			if err := ctx.BindJSON(&req); err != nil {
				reply.ReplyInvalidParams(ctx, err)
				return
			}
			commentID, err := service.Comment().Create(ctx, userID, schoolID, uint(articleID), articleType, req)
			if err != nil {
				if errors.Is(err, service.ErrCommentArticleNotFound) {
					reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
					return
				}
				if errors.Is(err, service.ErrCommentParentNotFound) {
					reply.ReplyErrWithMessage(ctx, "父评论不存在")
					return
				}
				reply.ReplyInternalError(ctx, err)
				return
			}
			reply.ReplyOKWithData(ctx, gin.H{"id": commentID})
		},
		ListComments: func(ctx *gin.Context) {
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
			_, err = dao.Article().GetByIDWithSchoolAndType(ctx.Request.Context(), uint(articleID), schoolID, articleType)
			if err != nil {
				reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
				return
			}
			page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
			pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
			list, total, err := service.Comment().ListComments(ctx, userID, schoolID, uint(articleID), articleType, page, pageSize)
			if err != nil {
				if errors.Is(err, service.ErrCommentArticleNotFound) {
					reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
					return
				}
				reply.ReplyInternalError(ctx, err)
				return
			}
			reply.ReplyOKWithData(ctx, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
		},
		ListReplies: func(ctx *gin.Context) {
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
			commentIDStr := ctx.Param("commentId")
			commentID, err := strconv.ParseUint(commentIDStr, 10, 32)
			if err != nil {
				reply.ReplyInvalidParams(ctx, err)
				return
			}
			_, err = dao.Article().GetByIDWithSchoolAndType(ctx.Request.Context(), uint(articleID), schoolID, articleType)
			if err != nil {
				reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
				return
			}
			page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
			pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
			list, total, err := service.Comment().ListReplies(ctx, userID, schoolID, uint(articleID), uint(commentID), articleType, page, pageSize)
			if err != nil {
				if errors.Is(err, service.ErrCommentArticleNotFound) {
					reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
					return
				}
				if errors.Is(err, service.ErrCommentParentNotFound) {
					reply.ReplyErrWithMessage(ctx, "评论不存在")
					return
				}
				reply.ReplyInternalError(ctx, err)
				return
			}
			reply.ReplyOKWithData(ctx, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
		},
	}
}

var (
	PostCommentHandlers     = ArticleCommentHandlers(constant.ArticleTypeNormal)
	QuestionCommentHandlers = ArticleCommentHandlers(constant.ArticleTypeQuestion)
	AnswerCommentHandlers   = ArticleCommentHandlers(constant.ArticleTypeAnswer)
)
