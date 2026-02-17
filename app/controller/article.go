package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
	"gorm.io/gorm"
)

// ArticleCreate 创建帖子 POST /article
func ArticleCreate(ctx *gin.Context) {
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
	id, err := service.Article().Create(ctx, userID, schoolID, req)
	if err != nil {
		if errors.Is(err, service.ErrSchoolNotBound) {
			reply.ReplyErrWithMessage(ctx, "请先绑定学校")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"id": id})
}

// ArticleGet 获取帖子详情 GET /article/:id（需登录，学校隔离）
func ArticleGet(ctx *gin.Context) {
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
	art, err := service.Article().Get(ctx, uint(id), userID, schoolID)
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
	reply.ReplyOKWithData(ctx, art)
}

// ArticleUpdate 更新帖子 PUT /article/:id
func ArticleUpdate(ctx *gin.Context) {
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
	if err := service.Article().Update(ctx, uint(id), userID, schoolID, req); err != nil {
		if errors.Is(err, service.ErrArticleNotFoundOrNoPermission) {
			reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// ArticleUploadImages 批量上传帖子图片 POST /article/:id/images（multipart form 字段 files）
func ArticleUploadImages(ctx *gin.Context) {
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
	urls, err := service.Article().UploadImages(ctx, uint(id), userID, schoolID, files)
	if err != nil {
		if errors.Is(err, service.ErrArticleNotFoundOrNoPermission) {
			reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"urls": urls})
}

// ArticleUpdateImages 更新帖子图片列表（仅 URL 元数据）PUT /article/:id/images
func ArticleUpdateImages(ctx *gin.Context) {
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
	var body struct {
		Images []string `json:"images"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.Article().UpdateImages(ctx, uint(id), userID, schoolID, body.Images); err != nil {
		if errors.Is(err, service.ErrArticleNotFoundOrNoPermission) {
			reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// ArticleDelete 删除帖子（惰性）DELETE /article/:id
func ArticleDelete(ctx *gin.Context) {
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
	if err := service.Article().Delete(ctx, uint(id), userID, schoolID); err != nil {
		if errors.Is(err, service.ErrArticleNotFoundOrNoPermission) {
			reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// ArticleList 帖子列表 GET /article?page=1&pageSize=20
func ArticleList(ctx *gin.Context) {
	schoolID := middleware.GetSchoolID(ctx)
	if schoolID == 0 {
		reply.ReplyErrWithMessage(ctx, "请先绑定学校")
		return
	}
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	list, total, err := service.Article().List(ctx, schoolID, page, pageSize)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ArticleSearch 帖子搜索 GET /article/search?q=xxx&page=1&pageSize=20
func ArticleSearch(ctx *gin.Context) {
	schoolID := middleware.GetSchoolID(ctx)
	if schoolID == 0 {
		reply.ReplyErrWithMessage(ctx, "请先绑定学校")
		return
	}
	keyword := ctx.Query("q")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	list, total, err := service.Article().Search(ctx, schoolID, keyword, page, pageSize)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
