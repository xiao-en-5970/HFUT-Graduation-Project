package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// getUserBrief 拿"作者简要"——非孤儿 QQ 旗下号会带上 from_user_id + from_username 让前端
// 拼成"username（来自用户 xxx）"展示（详见 vo/response.AuthorProfile 注释）。
func getUserBrief(ctx *gin.Context, userID uint) (map[string]interface{}, error) {
	u, err := dao.User().GetByIDIfValid(ctx.Request.Context(), userID)
	if err != nil || u == nil {
		return nil, err
	}
	out := map[string]interface{}{
		"id": u.ID, "username": u.Username, "avatar": oss.ToFullURL(u.Avatar),
	}
	if u.IsQQChild() && u.ParentUserID != nil && *u.ParentUserID > 0 {
		if parent, perr := dao.User().GetByIDIfValid(ctx.Request.Context(), uint(*u.ParentUserID)); perr == nil && parent != nil {
			out["from_user_id"] = parent.ID
			out["from_username"] = parent.Username
		}
	}
	return out, nil
}

// effectiveGoodAddr 商品展示用统一地址：优先 goods_addr，兼容仅填过 pickup_addr 的旧数据
func effectiveGoodAddr(g *model.Good) string {
	if g == nil {
		return ""
	}
	s := strings.TrimSpace(g.GoodsAddr)
	if s != "" {
		return s
	}
	return strings.TrimSpace(g.PickupAddr)
}

func enrichGoodsWithAuthor(ctx *gin.Context, list []*model.Good) []map[string]interface{} {
	out := make([]map[string]interface{}, len(list))
	for i, g := range list {
		out[i] = enrichGoodWithAuthor(ctx, g)
	}
	return out
}

func enrichGoodWithAuthor(ctx *gin.Context, g *model.Good) map[string]interface{} {
	addr := effectiveGoodAddr(g)
	m := map[string]interface{}{
		"id": g.ID, "user_id": g.UserID, "school_id": g.SchoolID, "title": g.Title, "content": g.Content,
		"images": oss.TransformImageURLs(g.Images), "image_count": g.ImageCount,
		"goods_type": g.GoodsType, "goods_type_label": constant.GoodsTypeLabel(g.GoodsType),
		"goods_category":       g.GoodsCategory,
		"goods_category_label": constant.GoodsCategoryLabel(g.GoodsCategory),
		"goods_addr":           addr,
		"pickup_addr":          addr,
		"price":                g.Price, "marked_price": g.MarkedPrice, "stock": g.Stock,
		"good_status": g.GoodStatus, "status": g.Status,
		"like_count": g.LikeCount, "collect_count": g.CollectCount,
		"payment_qr_url": oss.ToFullURL(g.PaymentQRURL),
		"has_deadline":   g.HasDeadline,
		"created_at":     g.CreatedAt, "updated_at": g.UpdatedAt,
	}
	if g.HasDeadline && g.Deadline != nil {
		m["deadline"] = g.Deadline.Format(time.RFC3339)
		// 剩余秒数：给前端直接展示「还有 X 天」，负数表示已过期
		m["deadline_remaining_seconds"] = int64(time.Until(*g.Deadline).Seconds())
	} else {
		m["deadline"] = nil
		m["deadline_remaining_seconds"] = nil
	}
	if g.GoodsLat != nil {
		m["goods_lat"] = *g.GoodsLat
	} else {
		m["goods_lat"] = nil
	}
	if g.GoodsLng != nil {
		m["goods_lng"] = *g.GoodsLng
	} else {
		m["goods_lng"] = nil
	}
	if g.UserID != nil && *g.UserID > 0 {
		if u, err := getUserBrief(ctx, uint(*g.UserID)); err == nil {
			m["author"] = u
		}
	}
	uid := middleware.GetUserID(ctx)
	if uid > 0 {
		gid := int(g.ID)
		if ok, err := dao.Like().Exists(ctx.Request.Context(), uid, gid, constant.ExtTypeGoods); err == nil {
			m["is_liked"] = ok
		} else {
			m["is_liked"] = false
		}
		if ok, err := dao.CollectItem().ExistsByUserExt(ctx.Request.Context(), uid, gid, constant.ExtTypeGoods); err == nil {
			m["is_collected"] = ok
		} else {
			m["is_collected"] = false
		}
	} else {
		m["is_liked"] = false
		m["is_collected"] = false
	}
	return m
}

// GoodList 商品列表 GET /goods
// Query: page, pageSize, q（标题模糊）, sort（空/newest=上架时间降序；updated_at=最近更新降序；recommend=个性化推荐）
// category: 0/缺省=不过滤；1=二手买卖；2=有偿求助
func GoodList(ctx *gin.Context) {
	schoolID := middleware.GetSchoolID(ctx)
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	keyword := strings.TrimSpace(ctx.Query("q"))
	sort := strings.TrimSpace(ctx.Query("sort"))
	cat, _ := strconv.Atoi(ctx.Query("category"))
	category := int16(cat)
	if sort == "newest" {
		sort = ""
	}
	// sort=recommend 走推荐链路；关键字搜索仍走原流程
	if sort == dao.SortRecommend && keyword == "" {
		userID := middleware.GetUserID(ctx)
		token := service.Recommend().EnsureRefreshToken(ctx.Query("refresh_token"))
		list, total, err := service.Recommend().RecallGoods(ctx.Request.Context(), userID, schoolID, page, pageSize, token, category)
		if err != nil {
			reply.ReplyInternalError(ctx, err)
			return
		}
		for _, g := range list {
			g.Images = oss.TransformImageURLs(g.Images)
		}
		rows := enrichGoodsWithAuthor(ctx, list)
		stampGoodsViewedBatch(ctx.Request.Context(), userID, rows)
		reply.ReplyOKWithData(ctx, gin.H{
			"list":          rows,
			"total":         total,
			"page":          page,
			"page_size":     pageSize,
			"refresh_token": token,
			"sort":          dao.SortRecommend,
		})
		return
	}
	list, total, err := service.Good().List(ctx, schoolID, page, pageSize, keyword, sort, category)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	for _, g := range list {
		g.Images = oss.TransformImageURLs(g.Images)
	}
	rows := enrichGoodsWithAuthor(ctx, list)
	stampGoodsViewedBatch(ctx.Request.Context(), middleware.GetUserID(ctx), rows)
	reply.ReplyOKWithData(ctx, gin.H{"list": rows, "total": total, "page": page, "page_size": pageSize})
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
	if g.Status == constant.StatusValid && userID > 0 {
		service.Recommend().RecordBehavior(ctx.Request.Context(), userID, constant.ExtTypeGoods, int(g.ID), constant.BehaviorView, "")
	}
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
	ownList := viewerID != 0 && uint(targetID) == viewerID           // 本人列表不按学校过滤，避免空列表
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	list, total, err := service.Good().ListByUserID(ctx, uint(targetID), schoolID, includeOffShelf, ownList, page, pageSize)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	for _, g := range list {
		g.Images = oss.TransformImageURLs(g.Images)
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": enrichGoodsWithAuthor(ctx, list), "total": total, "page": page, "page_size": pageSize})
}
