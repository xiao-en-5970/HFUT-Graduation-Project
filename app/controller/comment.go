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
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// validCommentExtTypes 评论支持的 extType：1帖子 2提问 3回答 4商品
var validCommentExtTypes = map[int]bool{
	constant.ExtTypePost: true, constant.ExtTypeQuestion: true, constant.ExtTypeAnswer: true, constant.ExtTypeGoods: true,
}

// CommentCreate 统一评论接口：POST /comments/:extType/:id
func CommentCreate(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	schoolID := middleware.GetSchoolID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	// schoolID=0 时仅可对公开文章评论
	extType, articleID, ok := parseExtTypeAndID(ctx)
	if !ok {
		return
	}
	if !validCommentExtTypes[extType] {
		reply.ReplyErrWithMessage(ctx, "extType 无效，评论仅支持 1帖子 2提问 3回答")
		return
	}
	var req service.CreateCommentReq
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	commentID, err := service.Comment().Create(ctx, userID, schoolID, articleID, extType, req)
	if err != nil {
		if errors.Is(err, errno.ErrCommentArticleNotFound) {
			reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
			return
		}
		if errors.Is(err, errno.ErrCommentParentNotFound) {
			reply.ReplyErrWithMessage(ctx, "父评论不存在")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	service.Recommend().RecordBehavior(ctx.Request.Context(), userID, extType, int(articleID), constant.BehaviorComment, "")
	reply.ReplyOKWithData(ctx, gin.H{"id": commentID})
}

// CommentList 统一评论列表：GET /comments/:extType/:id
func CommentList(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	schoolID := middleware.GetSchoolID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	extType, articleID, ok := parseExtTypeAndID(ctx)
	if !ok {
		return
	}
	if !validCommentExtTypes[extType] {
		reply.ReplyErrWithMessage(ctx, "extType 无效，评论仅支持 1帖子 2提问 3回答")
		return
	}
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	list, total, err := service.Comment().ListComments(ctx, userID, schoolID, articleID, extType, page, pageSize)
	if err != nil {
		if errors.Is(err, errno.ErrCommentArticleNotFound) {
			reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	for _, c := range list {
		c.Images = oss.TransformImageURLs(c.Images)
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": enrichCommentsWithAuthor(ctx, list), "total": total, "page": page, "page_size": pageSize})
}

// CommentListReplies 统一回复列表：GET /comments/:extType/:id/:commentId/replies
func CommentListReplies(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	schoolID := middleware.GetSchoolID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	extType, articleID, ok := parseExtTypeAndID(ctx)
	if !ok {
		return
	}
	if !validCommentExtTypes[extType] {
		reply.ReplyErrWithMessage(ctx, "extType 无效，评论仅支持 1帖子 2提问 3回答")
		return
	}
	commentIDStr := ctx.Param("commentId")
	commentID, err := strconv.ParseUint(commentIDStr, 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	list, total, err := service.Comment().ListReplies(ctx, userID, schoolID, articleID, uint(commentID), extType, page, pageSize)
	if err != nil {
		if errors.Is(err, errno.ErrCommentArticleNotFound) {
			reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
			return
		}
		if errors.Is(err, errno.ErrCommentParentNotFound) {
			reply.ReplyErrWithMessage(ctx, "评论不存在")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	for _, c := range list {
		c.Images = oss.TransformImageURLs(c.Images)
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": enrichCommentsWithAuthor(ctx, list), "total": total, "page": page, "page_size": pageSize})
}

func parseExtTypeAndID(ctx *gin.Context) (extType int, id uint, ok bool) {
	extTypeStr := ctx.Param("extType")
	extTypeNum, err := strconv.Atoi(extTypeStr)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, "extType 必须为整数")
		return 0, 0, false
	}
	idStr := ctx.Param("id")
	articleID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return 0, 0, false
	}
	return extTypeNum, uint(articleID), true
}
