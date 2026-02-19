package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// ArticleCollectHandlers 文章收藏处理器，按类型特化（帖子/提问/回答）
func ArticleCollectHandlers(extType int) struct {
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
			var body struct {
				CollectID uint `json:"collect_id"` // 0 表示默认收藏夹
			}
			_ = ctx.BindJSON(&body)
			if err := service.Collect().AddArticle(ctx, userID, schoolID, body.CollectID, uint(articleID), extType); err != nil {
				if errors.Is(err, service.ErrCollectArticleNotFound) {
					reply.ReplyErrWithMessage(ctx, "文章不存在")
					return
				}
				if errors.Is(err, service.ErrCollectAlreadyCollected) {
					reply.ReplyErrWithMessage(ctx, "已收藏")
					return
				}
				reply.ReplyInternalError(ctx, err)
				return
			}
			reply.ReplyOK(ctx)
		},
		Remove: func(ctx *gin.Context) {
			userID := middleware.GetUserID(ctx)
			if userID == 0 {
				reply.ReplyUnauthorized(ctx)
				return
			}
			idStr := ctx.Param("id")
			articleID, err := strconv.ParseUint(idStr, 10, 32)
			if err != nil {
				reply.ReplyInvalidParams(ctx, err)
				return
			}
			collectIDStr := ctx.DefaultQuery("collect_id", "0")
			collectID, _ := strconv.ParseUint(collectIDStr, 10, 32)
			if err := service.Collect().RemoveArticle(ctx, userID, uint(collectID), uint(articleID), extType); err != nil {
				reply.ReplyInternalError(ctx, err)
				return
			}
			reply.ReplyOK(ctx)
		},
	}
}

var (
	PostCollectHandlers     = ArticleCollectHandlers(constant.ExtTypePost)
	QuestionCollectHandlers = ArticleCollectHandlers(constant.ExtTypeQuestion)
	AnswerCollectHandlers   = ArticleCollectHandlers(constant.ExtTypeAnswer)
)

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

// ListCollectItems 列出收藏夹内容
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
		if errors.Is(err, service.ErrCollectFolderNotFound) {
			reply.ReplyErrWithMessage(ctx, "收藏夹不存在")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
}
