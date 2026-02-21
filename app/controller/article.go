package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
	"gorm.io/gorm"
)

// ArticleHandlers 返回指定类型的文章 CRUD 处理器（帖子1/提问2/回答3），学校+类型隔离
func ArticleHandlers(articleType int) struct {
	ListDrafts, List, Search, Create, Get, Update, UploadImages, Publish, Delete gin.HandlerFunc
} {
	return struct {
		ListDrafts, List, Search, Create, Get, Update, UploadImages, Publish, Delete gin.HandlerFunc
	}{
		ListDrafts: func(ctx *gin.Context) {
			userID := middleware.GetUserID(ctx)
			if userID == 0 {
				reply.ReplyUnauthorized(ctx)
				return
			}
			schoolID := middleware.GetSchoolID(ctx)
			page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
			pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
			list, total, err := service.Article().ListDrafts(ctx, userID, schoolID, articleType, page, pageSize)
			if err != nil {
				reply.ReplyInternalError(ctx, err)
				return
			}
			for _, a := range list {
				a.Images = oss.TransformImageURLs(a.Images)
			}
			reply.ReplyOKWithData(ctx, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
		},
		List: func(ctx *gin.Context) {
			schoolID := middleware.GetSchoolID(ctx)
			page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
			pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
			list, total, err := service.Article().List(ctx, schoolID, articleType, page, pageSize)
			if err != nil {
				reply.ReplyInternalError(ctx, err)
				return
			}
			for _, a := range list {
				a.Images = oss.TransformImageURLs(a.Images)
			}
			reply.ReplyOKWithData(ctx, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
		},
		Search: func(ctx *gin.Context) {
			schoolID := middleware.GetSchoolID(ctx)
			keyword := ctx.Query("q")
			page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
			pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
			list, total, err := service.Article().Search(ctx, schoolID, articleType, keyword, page, pageSize)
			if err != nil {
				reply.ReplyInternalError(ctx, err)
				return
			}
			for _, a := range list {
				a.Images = oss.TransformImageURLs(a.Images)
			}
			reply.ReplyOKWithData(ctx, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
		},
		Create: func(ctx *gin.Context) {
			userID := middleware.GetUserID(ctx)
			schoolID := middleware.GetSchoolID(ctx)
			if userID == 0 {
				reply.ReplyUnauthorized(ctx)
				return
			}
			var req service.CreateArticleReq
			if err := ctx.BindJSON(&req); err != nil {
				reply.ReplyInvalidParams(ctx, err)
				return
			}
			id, err := service.Article().Create(ctx, userID, schoolID, articleType, req)
			if err != nil {
				if errors.Is(err, service.ErrSchoolNotBound) {
					reply.ReplyErrWithMessage(ctx, "请先绑定学校")
					return
				}
				if errors.Is(err, service.ErrParentQuestionRequired) {
					reply.ReplyErrWithMessage(ctx, "回答必须指定 parent_id 指向提问")
					return
				}
				if errors.Is(err, service.ErrParentQuestionNotFound) {
					reply.ReplyErrWithMessage(ctx, "父提问不存在或非本校")
					return
				}
				reply.ReplyInternalError(ctx, err)
				return
			}
			reply.ReplyOKWithData(ctx, gin.H{"id": id})
		},
		Get: func(ctx *gin.Context) {
			userID := middleware.GetUserID(ctx)
			schoolID := middleware.GetSchoolID(ctx)
			if userID == 0 {
				reply.ReplyUnauthorized(ctx)
				return
			}
			idStr := ctx.Param("id")
			id, err := strconv.ParseUint(idStr, 10, 32)
			if err != nil {
				reply.ReplyInvalidParams(ctx, err)
				return
			}
			art, err := service.Article().Get(ctx, uint(id), userID, schoolID, articleType)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, service.ErrArticleNotFoundOrNoPermission) {
					reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
					return
				}
				if errors.Is(err, service.ErrSchoolNotBound) {
					reply.ReplyErrWithMessage(ctx, "请先绑定学校")
					return
				}
				reply.ReplyInternalError(ctx, err)
				return
			}
			art.Images = oss.TransformImageURLs(art.Images)
			reply.ReplyOKWithData(ctx, art)
		},
		Update: func(ctx *gin.Context) {
			userID := middleware.GetUserID(ctx)
			schoolID := middleware.GetSchoolID(ctx)
			if userID == 0 {
				reply.ReplyUnauthorized(ctx)
				return
			}
			idStr := ctx.Param("id")
			id, err := strconv.ParseUint(idStr, 10, 32)
			if err != nil {
				reply.ReplyInvalidParams(ctx, err)
				return
			}
			var req service.UpdateArticleReq
			if err := ctx.BindJSON(&req); err != nil {
				reply.ReplyInvalidParams(ctx, err)
				return
			}
			if err := service.Article().Update(ctx, uint(id), userID, schoolID, articleType, req); err != nil {
				if errors.Is(err, service.ErrArticleNotFoundOrNoPermission) {
					reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
					return
				}
				reply.ReplyInternalError(ctx, err)
				return
			}
			reply.ReplyOK(ctx)
		},
		UploadImages: func(ctx *gin.Context) {
			userID := middleware.GetUserID(ctx)
			schoolID := middleware.GetSchoolID(ctx)
			if userID == 0 {
				reply.ReplyUnauthorized(ctx)
				return
			}
			idStr := ctx.Param("id")
			id, err := strconv.ParseUint(idStr, 10, 32)
			if err != nil {
				reply.ReplyInvalidParams(ctx, err)
				return
			}
			form, err := ctx.MultipartForm()
			if err != nil {
				reply.ReplyInvalidParams(ctx, err)
				return
			}
			files := form.File["files"]
			if len(files) == 0 {
				reply.ReplyErrWithMessage(ctx, "至少需要上传一张图片，使用 form 字段 files")
				return
			}
			urls, err := service.Article().UploadImages(ctx, uint(id), userID, schoolID, articleType, files)
			if err != nil {
				if errors.Is(err, service.ErrArticleNotFoundOrNoPermission) {
					reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
					return
				}
				reply.ReplyInternalError(ctx, err)
				return
			}
			reply.ReplyOKWithData(ctx, gin.H{"urls": urls})
		},
		Publish: func(ctx *gin.Context) {
			userID := middleware.GetUserID(ctx)
			if userID == 0 {
				reply.ReplyUnauthorized(ctx)
				return
			}
			idStr := ctx.Param("id")
			id, err := strconv.ParseUint(idStr, 10, 32)
			if err != nil {
				reply.ReplyInvalidParams(ctx, err)
				return
			}
			if err := service.Article().PublishDraft(ctx, uint(id), userID); err != nil {
				if errors.Is(err, service.ErrDraftNotFoundOrNoPermission) {
					reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
					return
				}
				reply.ReplyInternalError(ctx, err)
				return
			}
			reply.ReplyOK(ctx)
		},
		Delete: func(ctx *gin.Context) {
			userID := middleware.GetUserID(ctx)
			schoolID := middleware.GetSchoolID(ctx)
			if userID == 0 {
				reply.ReplyUnauthorized(ctx)
				return
			}
			idStr := ctx.Param("id")
			id, err := strconv.ParseUint(idStr, 10, 32)
			if err != nil {
				reply.ReplyInvalidParams(ctx, err)
				return
			}
			if err := service.Article().Delete(ctx, uint(id), userID, schoolID, articleType); err != nil {
				if errors.Is(err, service.ErrArticleNotFoundOrNoPermission) {
					reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
					return
				}
				reply.ReplyInternalError(ctx, err)
				return
			}
			reply.ReplyOK(ctx)
		},
	}
}

// 预定义三类处理器
var (
	PostHandlers     = ArticleHandlers(constant.ArticleTypeNormal)
	QuestionHandlers = ArticleHandlers(constant.ArticleTypeQuestion)
	AnswerHandlers   = ArticleHandlers(constant.ArticleTypeAnswer)
)

// QuestionListAnswers 列出某提问下的回答 GET /question/:id/answers
func QuestionListAnswers(ctx *gin.Context) {
	schoolID := middleware.GetSchoolID(ctx)
	idStr := ctx.Param("id")
	questionID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	list, total, err := service.Article().ListAnswersByQuestionID(ctx, uint(questionID), schoolID, page, pageSize)
	if err != nil {
		if errors.Is(err, service.ErrParentQuestionNotFound) {
			reply.ReplyErrWithMessage(ctx, "提问不存在或非本校")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	for _, a := range list {
		a.Images = oss.TransformImageURLs(a.Images)
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
}
