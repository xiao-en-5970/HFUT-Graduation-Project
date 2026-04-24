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
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
	"gorm.io/gorm"
)

// AnswerListWithParent GET /answer — 列表附带 parent_question，供社区流展示
func AnswerListWithParent(ctx *gin.Context) {
	schoolID := middleware.GetSchoolID(ctx)
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	// sort=recommend 走推荐链路
	if ctx.Query("sort") == dao.SortRecommend {
		userID := middleware.GetUserID(ctx)
		token := service.Recommend().EnsureRefreshToken(ctx.Query("refresh_token"))
		list, total, err := service.Recommend().RecallArticles(ctx.Request.Context(), userID, schoolID, constant.ArticleTypeAnswer, page, pageSize, token)
		if err != nil {
			reply.ReplyInternalError(ctx, err)
			return
		}
		for _, a := range list {
			a.Images = oss.TransformImageURLs(a.Images)
		}
		enriched := enrichAnswersWithParentQuestion(ctx, schoolID, list)
		reply.ReplyOKWithData(ctx, gin.H{
			"list":          enriched,
			"total":         total,
			"page":          page,
			"page_size":     pageSize,
			"refresh_token": token,
			"sort":          dao.SortRecommend,
		})
		return
	}
	list, total, err := service.Article().List(ctx, schoolID, constant.ArticleTypeAnswer, page, pageSize, articleListSort(ctx))
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	for _, a := range list {
		a.Images = oss.TransformImageURLs(a.Images)
	}
	enriched := enrichAnswersWithParentQuestion(ctx, schoolID, list)
	reply.ReplyOKWithData(ctx, gin.H{"list": enriched, "total": total, "page": page, "page_size": pageSize})
}

// AnswerGetWithParent GET /answer/:id — 详情附带 parent_question
func AnswerGetWithParent(ctx *gin.Context) {
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
	art, err := service.Article().Get(ctx, uint(id), userID, schoolID, constant.ArticleTypeAnswer)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, errno.ErrArticleNotFoundOrNoPermission) {
			reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)
			return
		}
		if errors.Is(err, errno.ErrSchoolNotBound) {
			reply.ReplyErrWithMessage(ctx, "请先绑定学校")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	art.Images = oss.TransformImageURLs(art.Images)
	enriched := enrichAnswerWithParent(ctx, schoolID, art)
	if art.Status == constant.StatusValid && art.PublishStatus == 2 {
		service.Recommend().RecordBehavior(ctx.Request.Context(), userID, constant.ArticleTypeAnswer, int(art.ID), constant.BehaviorView, "")
	}
	reply.ReplyOKWithData(ctx, enriched)
}
