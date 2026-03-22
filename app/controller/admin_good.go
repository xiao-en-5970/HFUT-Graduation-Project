package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
	"gorm.io/gorm"
)

// AdminGoodList GET /admin/goods 全站商品列表
func AdminGoodList(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	schoolID, _ := strconv.ParseUint(ctx.DefaultQuery("school_id", "0"), 10, 32)
	includeInvalid := ctx.Query("include_invalid") != "0"
	list, total, err := dao.Good().ListAllForAdmin(ctx.Request.Context(), page, pageSize, uint(schoolID), includeInvalid)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	for _, g := range list {
		g.Images = oss.TransformImageURLs(g.Images)
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": enrichGoodsWithAuthor(ctx, list), "total": total, "page": page, "page_size": pageSize})
}

// AdminGoodGet GET /admin/goods/:id
func AdminGoodGet(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	g, err := dao.Good().GetByIDAdmin(ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			reply.ReplyNotFound(ctx, errcode.ErrGoodNotFound)
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	g.Images = oss.TransformImageURLs(g.Images)
	reply.ReplyOKWithData(ctx, enrichGoodWithAuthor(ctx, g))
}

// AdminGoodCreate POST /admin/goods
func AdminGoodCreate(ctx *gin.Context) {
	var req service.AdminCreateGoodReq
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	id, err := service.Good().AdminCreate(ctx, req)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"id": id})
}

// AdminGoodUpdate PUT /admin/goods/:id
func AdminGoodUpdate(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	var req service.AdminUpdateGoodReq
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.Good().AdminUpdate(ctx, id, req); err != nil {
		if errors.Is(err, service.ErrGoodNotFoundOrNoPermission) {
			reply.ReplyErrWithMessage(ctx, "商品不存在")
			return
		}
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOK(ctx)
}

// AdminGoodPublish POST /admin/goods/:id/publish
func AdminGoodPublish(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	if err := service.Good().AdminPublish(ctx, id); err != nil {
		if errors.Is(err, service.ErrGoodNotFoundOrNoPermission) {
			reply.ReplyErrWithMessage(ctx, "商品不存在")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// AdminGoodOffShelf POST /admin/goods/:id/off-shelf
func AdminGoodOffShelf(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	if err := service.Good().AdminOffShelf(ctx, id); err != nil {
		if errors.Is(err, service.ErrGoodNotFoundOrNoPermission) {
			reply.ReplyErrWithMessage(ctx, "商品不存在")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// AdminGoodUploadImages POST /admin/goods/:id/images
func AdminGoodUploadImages(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
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
	urls, err := service.Good().AdminUploadImages(ctx, id, files)
	if err != nil {
		if errors.Is(err, service.ErrGoodNotFoundOrNoPermission) {
			reply.ReplyErrWithMessage(ctx, "商品不存在")
			return
		}
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"urls": urls})
}

// AdminGoodDisable DELETE /admin/goods/:id 软删除（禁用）
func AdminGoodDisable(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	st := constant.StatusInvalid
	if err := service.Good().AdminUpdate(ctx, id, service.AdminUpdateGoodReq{Status: &st}); err != nil {
		if errors.Is(err, service.ErrGoodNotFoundOrNoPermission) {
			reply.ReplyErrWithMessage(ctx, "商品不存在")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// AdminGoodRestore POST /admin/goods/:id/restore
func AdminGoodRestore(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	st := constant.StatusValid
	if err := service.Good().AdminUpdate(ctx, id, service.AdminUpdateGoodReq{Status: &st}); err != nil {
		if errors.Is(err, service.ErrGoodNotFoundOrNoPermission) {
			reply.ReplyErrWithMessage(ctx, "商品不存在")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}
