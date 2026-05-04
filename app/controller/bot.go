package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// =============================================================================
// 这一组 handler 全部挂在 /api/v1/bot/* 路由下，由 middleware.BotServiceAuth 鉴权。
// 不依赖用户 JWT；调用方是 QQ-bot 这类 service-account。
// 设计文档：QQ-bot 仓库 skill/bot/SKILL.md
// =============================================================================

// BotQQChildUpsert: POST /api/v1/bot/users/qq-child
//
// idempotent：找到则返回 (user_id, created=false)；找不到按 group_id 反查 school 后创建并返回 created=true。
// 群没配学校（schools.qq_groups 都不含此 group_id）→ 返回带特定错误码，bot 看到这个错应该静默忽略。
func BotQQChildUpsert(ctx *gin.Context) {
	var body service.BotUpsertQQChildReq
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	resp, err := service.BotUpsertQQChild(ctx.Request.Context(), body)
	if err != nil {
		if errors.Is(err, service.ErrBotGroupNoSchool) {
			// 用 404 做特殊语义：bot 客户端按 status code 区分"群没配学校"vs 一般错
			reply.ReplyErrWithCodeAndMessage(ctx, 404, 404, err.Error())
			return
		}
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithData(ctx, resp)
}

// BotPublishGood: POST /api/v1/bot/goods
func BotPublishGood(ctx *gin.Context) {
	var body service.BotPublishGoodReq
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	resp, err := service.BotPublishGood(ctx.Request.Context(), body)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithData(ctx, resp)
}

// BotOffShelfGood: POST /api/v1/bot/goods/:id/off-shelf
//
// Body: { "user_id": <调用方 user_id> }——必须是 owner 或 owner 的主账号。
func BotOffShelfGood(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	var body struct {
		UserID uint `json:"user_id"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.BotOffShelfGood(ctx.Request.Context(), uint(id), body.UserID); err != nil {
		switch {
		case errors.Is(err, service.ErrBotGoodNotFound):
			reply.ReplyErrWithCodeAndMessage(ctx, 404, 404, err.Error())
		case errors.Is(err, service.ErrBotGoodNotOwner):
			reply.ReplyErrWithCodeAndMessage(ctx, 403, 403, err.Error())
		default:
			reply.ReplyErrWithMessage(ctx, err.Error())
		}
		return
	}
	reply.ReplyOK(ctx)
}

// BotListActiveGoods: GET /api/v1/bot/users/:user_id/goods/active?limit=20
func BotListActiveGoods(ctx *gin.Context) {
	userID, err := strconv.ParseUint(ctx.Param("user_id"), 10, 64)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	limit, _ := strconv.Atoi(ctx.Query("limit"))
	goods, err := service.BotListActiveGoodsOfUser(ctx.Request.Context(), uint(userID), limit)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": goods, "total": len(goods)})
}

// BotPublishArticle: POST /api/v1/bot/articles
func BotPublishArticle(ctx *gin.Context) {
	var body service.BotPublishArticleReq
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	resp, err := service.BotPublishArticle(ctx.Request.Context(), body)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithData(ctx, resp)
}

// BotCloseArticle: POST /api/v1/bot/articles/:id/close
//
// Body: { "user_id": <调用方 user_id> }——必须是文章作者或其主账号。
func BotCloseArticle(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	var body struct {
		UserID uint `json:"user_id"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.BotCloseArticle(ctx.Request.Context(), uint(id), body.UserID); err != nil {
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOK(ctx)
}

// BotListOpenQuestions: GET /api/v1/bot/groups/:group_id/articles/open?limit=20
//
// 列出该群对应学校下的开放提问；bot 想给某条提问写回答时按 hint 文本在标题/内容里找 parent_id。
func BotListOpenQuestions(ctx *gin.Context) {
	groupID, err := strconv.ParseInt(ctx.Param("group_id"), 10, 64)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	limit, _ := strconv.Atoi(ctx.Query("limit"))
	list, err := service.BotListOpenQuestionsByGroup(ctx.Request.Context(), groupID, limit)
	if err != nil {
		if errors.Is(err, service.ErrBotGroupNoSchool) {
			reply.ReplyErrWithCodeAndMessage(ctx, 404, 404, err.Error())
			return
		}
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": list, "total": len(list)})
}
