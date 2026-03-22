package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

func getUserBrief(ctx *gin.Context, userID uint) (map[string]interface{}, error) {
	u, err := dao.User().GetByIDIfValid(ctx.Request.Context(), userID)
	if err != nil || u == nil {
		return nil, err
	}
	return map[string]interface{}{
		"id": u.ID, "username": u.Username, "avatar": oss.ToFullURL(u.Avatar),
	}, nil
}

func enrichGoodsWithAuthor(ctx *gin.Context, list []*model.Good) []map[string]interface{} {
	out := make([]map[string]interface{}, len(list))
	for i, g := range list {
		out[i] = enrichGoodWithAuthor(ctx, g)
	}
	return out
}

func enrichGoodWithAuthor(ctx *gin.Context, g *model.Good) map[string]interface{} {
	m := map[string]interface{}{
		"id": g.ID, "user_id": g.UserID, "school_id": g.SchoolID, "title": g.Title, "content": g.Content,
		"images": oss.TransformImageURLs(g.Images), "image_count": g.ImageCount,
		"price": g.Price, "marked_price": g.MarkedPrice, "stock": g.Stock,
		"good_status": g.GoodStatus, "status": g.Status,
		"like_count": g.LikeCount, "collect_count": g.CollectCount,
		"created_at": g.CreatedAt, "updated_at": g.UpdatedAt,
	}
	if g.UserID != nil && *g.UserID > 0 {
		if u, err := getUserBrief(ctx, uint(*g.UserID)); err == nil {
			m["author"] = u
		}
	}
	return m
}

// GoodList 商品列表 GET /goods
func GoodList(ctx *gin.Context) {
	schoolID := middleware.GetSchoolID(ctx)
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	list, total, err := service.Good().List(ctx, schoolID, page, pageSize)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	for _, g := range list {
		g.Images = oss.TransformImageURLs(g.Images)
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": enrichGoodsWithAuthor(ctx, list), "total": total, "page": page, "page_size": pageSize})
}

// GoodGet 商品详情 GET /goods/:id
func GoodGet(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	schoolID := middleware.GetSchoolID(ctx)
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	g, err := service.Good().Get(ctx, uint(id), userID, schoolID)
	if err != nil {
		if errors.Is(err, errno.ErrGoodNotFoundOrNoPermission) {
			reply.ReplyNotFound(ctx, errcode.ErrGoodNotFound)
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	g.Images = oss.TransformImageURLs(g.Images)
	reply.ReplyOKWithData(ctx, enrichGoodWithAuthor(ctx, g))
}

// GoodCreate 新建商品（下架状态）POST /goods
func GoodCreate(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	schoolID := middleware.GetSchoolID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	var req service.CreateGoodReq
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	id, err := service.Good().Create(ctx, userID, schoolID, req)
	if err != nil {
		if errors.Is(err, errno.ErrSchoolNotBound) {
			reply.ReplyErrWithMessage(ctx, "请先绑定学校")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"id": id})
}

// GoodUpdate 更新商品 PUT /goods/:id
func GoodUpdate(ctx *gin.Context) {
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
	var req service.UpdateGoodReq
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.Good().Update(ctx, uint(id), userID, schoolID, req); err != nil {
		if errors.Is(err, errno.ErrGoodNotFoundOrNoPermission) {
			reply.ReplyErrWithMessage(ctx, "商品不存在或无权限")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// GoodPublish 上架商品 POST /goods/:id/publish
func GoodPublish(ctx *gin.Context) {
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
	if err := service.Good().Publish(ctx, uint(id), userID); err != nil {
		if errors.Is(err, errno.ErrGoodNotFoundOrNoPermission) {
			reply.ReplyErrWithMessage(ctx, "商品不存在或无权限")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// GoodOffShelf 下架商品 POST /goods/:id/off-shelf
func GoodOffShelf(ctx *gin.Context) {
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
	if err := service.Good().OffShelf(ctx, uint(id), userID); err != nil {
		if errors.Is(err, errno.ErrGoodNotFoundOrNoPermission) {
			reply.ReplyErrWithMessage(ctx, "商品不存在或无权限")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// GoodUploadImages 上传商品图片 POST /goods/:id/images
func GoodUploadImages(ctx *gin.Context) {
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
	urls, err := service.Good().UploadImages(ctx, uint(id), userID, files)
	if err != nil {
		if errors.Is(err, errno.ErrGoodNotFoundOrNoPermission) {
			reply.ReplyErrWithMessage(ctx, "商品不存在或无权限")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"urls": urls})
}

// GoodListByUser 用户发布的商品 GET /user/:id/goods
func GoodListByUser(ctx *gin.Context) {
	viewerID := middleware.GetUserID(ctx)
	schoolID := middleware.GetSchoolID(ctx)
	targetStr := ctx.Param("id")
	targetID, err := strconv.ParseUint(targetStr, 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	includeOffShelf := (viewerID != 0 && uint(targetID) == viewerID) // 本人可看下架
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	list, total, err := service.Good().ListByUserID(ctx, uint(targetID), schoolID, includeOffShelf, page, pageSize)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	for _, g := range list {
		g.Images = oss.TransformImageURLs(g.Images)
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": enrichGoodsWithAuthor(ctx, list), "total": total, "page": page, "page_size": pageSize})
}
