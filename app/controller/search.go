package controller

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// SearchArticles 聚合搜索：帖子+提问+回答，需 JWT
// GET /api/v1/search/articles
// Query: q, type, visibility, time_range, created_after, created_before, sort, page, page_size
// sort: combined(推荐 相关度+热度) | latest(最新发布时间) | relevance | popularity | updated_at
func SearchArticles(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	schoolID := middleware.GetSchoolID(ctx)

	params := dao.AggregateSearchParams{
		Keyword:      ctx.Query("q"),
		Type:         0,
		Visibility:   dao.VisibilityAll,
		ViewerSchool: schoolID,
		TimeRange:    "all",
		Sort:         dao.SortRelevance,
		Page:         1,
		PageSize:     20,
		// 排序权重从环境变量读取
		WeightCollect:        config.SearchWeightCollect,
		WeightLike:           config.SearchWeightLike,
		WeightView:           config.SearchWeightView,
		InteractionDecayDays: config.SearchInteractionDecayDays,
		CombinedRelevance:    config.SearchCombinedRelevance,
		CombinedPopularity:   config.SearchCombinedPopularity,
	}

	if v := ctx.Query("type"); v != "" {
		if t, err := strconv.Atoi(v); err == nil && t >= 0 && t <= 3 {
			params.Type = t
		}
	}
	if v := ctx.Query("visibility"); v != "" {
		switch v {
		case dao.VisibilityPublic, dao.VisibilityMySchool, dao.VisibilityAll:
			params.Visibility = v
		}
	}
	if v := ctx.Query("time_range"); v != "" {
		switch v {
		case "7d", "30d", "90d", "all":
			params.TimeRange = v
		}
	}
	if v := ctx.Query("created_after"); v != "" {
		if t, err := time.ParseInLocation("2006-01-02", v, time.Local); err == nil {
			params.CreatedAfter = &t
		}
	}
	if v := ctx.Query("created_before"); v != "" {
		if t, err := time.ParseInLocation("2006-01-02", v, time.Local); err == nil {
			// 包含当天：created_at <= 23:59:59
			end := t.Add(24*time.Hour - time.Nanosecond)
			params.CreatedBefore = &end
		}
	}
	if v := ctx.Query("sort"); v != "" {
		switch v {
		case dao.SortRelevance, dao.SortPopularity, dao.SortCombined, dao.SortUpdatedAt, dao.SortLatest:
			params.Sort = v
		}
	}
	if v := ctx.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p >= 1 {
			params.Page = p
		}
	}
	if v := ctx.Query("page_size"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil && ps >= 1 && ps <= 100 {
			params.PageSize = ps
		}
	}

	list, total, err := service.Article().AggregateSearch(ctx, params)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	for _, a := range list {
		a.Images = oss.TransformImageURLs(a.Images)
	}
	reply.ReplyOKWithData(ctx, gin.H{
		"list":      enrichArticlesWithAuthor(ctx, list),
		"total":     total,
		"page":      params.Page,
		"page_size": params.PageSize,
	})
}
