package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// AdminBotServiceTokenCreate 管理员创建一条新的 service token；明文 token 一次性返回。
//
// 调用方（admin 前端）应当：
//   - 把响应里的 "token" 字段妥善保存到环境变量 / KMS / 等安全位置
//   - 任何后续接口都查不到原文；只能通过本接口"重新创建"
//
// POST /api/v1/admin/bot/service-tokens
// Body: { "name": "qq-bot-prod", "description": "...", "expires_in_days": 0 }
func AdminBotServiceTokenCreate(ctx *gin.Context) {
	var body service.CreateBotServiceTokenReq
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	creatorID := middleware.GetUserID(ctx)
	var creatorPtr *int
	if creatorID > 0 {
		v := int(creatorID)
		creatorPtr = &v
	}
	resp, err := service.CreateBotServiceToken(ctx.Request.Context(), creatorPtr, body)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithMessageAndData(ctx, "创建成功，请妥善保存 token，刷新后将不可见", resp)
}

// AdminBotServiceTokenList 列出全部 service token（不含明文 / hash）。
//
// GET /api/v1/admin/bot/service-tokens
func AdminBotServiceTokenList(ctx *gin.Context) {
	tokens, err := service.ListBotServiceTokens(ctx.Request.Context())
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": tokens, "total": len(tokens)})
}

// AdminBotServiceTokenRevoke 把某条 service token 立刻置为无效。幂等。
//
// POST /api/v1/admin/bot/service-tokens/:id/revoke
func AdminBotServiceTokenRevoke(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.RevokeBotServiceToken(ctx.Request.Context(), uint(id)); err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}
