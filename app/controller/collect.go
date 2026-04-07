package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// validCollectExtTypes 收藏支持的 extType：1帖子 2提问 3回答 4商品
var validCollectExtTypes = map[int]bool{
	constant.ExtTypePost: true, constant.ExtTypeQuestion: true,
	constant.ExtTypeAnswer: true, constant.ExtTypeGoods: true,
}

// CollectAdd 统一收藏接口：POST /collect/:extType/:id
func CollectAdd(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	schoolID := middleware.GetSchoolID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	// schoolID=0 时仅可收藏公开文章
	extType, extID, ok := parseCollectExtTypeAndID(ctx)
	if !ok {
		return
	}
	if !validCollectExtTypes[extType] {
		reply.ReplyErrWithMessage(ctx, "extType 无效，收藏支持 1帖子 2提问 3回答 4商品")
		return
	}
	var body struct {
		CollectID uint `json:"collect_id"` // 0 表示默认收藏夹
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.Collect().AddArticle(ctx, userID, schoolID, body.CollectID, extID, extType); err != nil {
		if errors.Is(err, errno.ErrCollectArticleNotFound) {
			reply.ReplyErrWithMessage(ctx, "内容不存在")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// CollectRemove 统一取消收藏：DELETE /collect/:extType/:id
func CollectRemove(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	extType, extID, ok := parseCollectExtTypeAndID(ctx)
	if !ok {
		return
	}
	if !validCollectExtTypes[extType] {
		reply.ReplyErrWithMessage(ctx, "extType 无效，收藏支持 1帖子 2提问 3回答 4商品")
		return
	}
	collectIDStr := ctx.DefaultQuery("collect_id", "0")
	collectID, _ := strconv.ParseUint(collectIDStr, 10, 32)
	if err := service.Collect().RemoveArticle(ctx, userID, uint(collectID), extID, extType); err != nil {
		if errors.Is(err, errno.ErrCollectFolderNotFound) {
			reply.ReplyErrWithMessage(ctx, "收藏夹不存在")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

func parseCollectExtTypeAndID(ctx *gin.Context) (extType int, id uint, ok bool) {
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

// CreateCollectFolder 创建收藏夹
func CreateCollectFolder(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	id, err := service.Collect().CreateFolder(ctx, userID, body.Name)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"id": id})
}

// ListCollectFolders 列出收藏夹
func ListCollectFolders(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	list, err := service.Collect().ListFolders(ctx, userID)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": list})
}

// ListCollectItems 收藏夹内容（统一接口）
// GET /folders/:id/items?ext_type=0 全部混合，ext_type=1|2|3|4 按类型筛选
func ListCollectItems(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	idStr := ctx.Param("id")
	folderID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	extTypeStr := ctx.DefaultQuery("ext_type", "0")
	extType, _ := strconv.Atoi(extTypeStr)
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	list, total, err := service.Collect().ListItems(ctx, userID, uint(folderID), extType, page, pageSize)
	if err != nil {
		if errors.Is(err, errno.ErrCollectFolderNotFound) {
			reply.ReplyErrWithMessage(ctx, "收藏夹不存在")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
}

// UserListCollects GET /user/collects?ext_type=1|2|3|4&page=&page_size=
// 当前用户默认收藏夹下的条目（须注册在 GET /user/:id 之前，否则 "collects" 会误匹配为用户 id）
func UserListCollects(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	schoolID := middleware.GetSchoolID(ctx)
	extTypeStr := ctx.DefaultQuery("ext_type", "0")
	extType, _ := strconv.Atoi(extTypeStr)
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "0"))
	if pageSize < 1 {
		pageSize, _ = strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	if extType < 1 || extType > 4 {
		reply.ReplyErrWithMessage(ctx, "ext_type 无效，应为 1帖子 2提问 3回答 4商品")
		return
	}

	items, total, err := service.Collect().ListItems(ctx, userID, 0, extType, page, pageSize)
	if err != nil {
		if errors.Is(err, errno.ErrCollectFolderNotFound) {
			reply.ReplyErrWithMessage(ctx, "收藏夹不存在")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}

	out := make([]interface{}, 0, len(items))
	for _, it := range items {
		switch it.ExtType {
		case constant.ExtTypePost, constant.ExtTypeQuestion:
			a, err := dao.Article().GetByIDWithSchoolOrPublicAndType(ctx.Request.Context(), uint(it.ExtID), schoolID, it.ExtType)
			if err != nil || a == nil {
				continue
			}
			a.Images = oss.TransformImageURLs(a.Images)
			aw := enrichArticleWithAuthorForViewer(ctx.Request.Context(), userID, it.ExtType, a)
			out = append(out, aw)
		case constant.ExtTypeAnswer:
			a, err := dao.Article().GetByIDWithSchoolOrPublicAndType(ctx.Request.Context(), uint(it.ExtID), schoolID, constant.ArticleTypeAnswer)
			if err != nil || a == nil {
				continue
			}
			a.Images = oss.TransformImageURLs(a.Images)
			out = append(out, enrichAnswerWithParent(ctx, schoolID, a))
		case constant.ExtTypeGoods:
			// 我的收藏：仅按 id+有效状态加载，不按学校过滤。
			// 否则未绑定学校(school_id=0)时本校商品会被 applyGoodSchoolVisibility 全部筛掉，列表为空。
			g, err := dao.Good().GetByID(ctx.Request.Context(), uint(it.ExtID))
			if err != nil || g == nil {
				continue
			}
			g.Images = oss.TransformImageURLs(g.Images)
			out = append(out, enrichGoodWithAuthor(ctx, g))
		default:
			continue
		}
	}

	reply.ReplyOKWithData(ctx, gin.H{
		"list":      out,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
