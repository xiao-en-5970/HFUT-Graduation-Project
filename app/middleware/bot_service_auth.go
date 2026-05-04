package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/util"
)

// botServiceTokenHeader bot 调 hfut 时携带 service token 的 header 名。
//
// 用 X- 前缀表明这是非标准（自定义）头；跟 user JWT 的 Authorization: Bearer 区分开。
const botServiceTokenHeader = "X-Bot-Service-Token"

// ctx key
const (
	botServiceCtxKeyService = "bot_service_service_name"
	botServiceCtxKeyJTI     = "bot_service_jti"
)

// BotServiceAuth 校验 X-Bot-Service-Token header 里的 service-to-service JWT。
//
// 流程（**纯本地验签，不查任何 DB**）：
//  1. JWT 验签（HS256 + 共享 secret BotServiceJWTSecret）
//  2. 校验 iss = "HFUT-Graduation-Project-bot"，防止 user 登录 JWT 误用
//  3. 校验 exp 未过期（jwt 库自动）
//  4. 通过后把 service / jti 写进 ctx 给 handler 审计 / log
//
// 不再有"DB token 表 + revoke 机制"——bot 端每次签发短期 token（60s）已经是默认快速失效；
// 真泄漏想立刻让所有现存 token 失效，rotate BotServiceJWTSecret 即可。
func BotServiceAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		plain := ctx.GetHeader(botServiceTokenHeader)
		if plain == "" {
			ctx.JSON(http.StatusUnauthorized, response.Response{
				Code:    401,
				Message: "缺少 " + botServiceTokenHeader + " 头",
			})
			ctx.Abort()
			return
		}
		claims, err := util.ParseBotServiceToken(plain)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, response.Response{
				Code:    401,
				Message: "service token 无效: " + err.Error(),
			})
			ctx.Abort()
			return
		}
		ctx.Set(botServiceCtxKeyService, claims.Service)
		ctx.Set(botServiceCtxKeyJTI, claims.RegisteredClaims.ID)
		ctx.Next()
	}
}

// GetBotServiceServiceName 从 ctx 取出 token 里的服务名（"qq-bot" 等）。
//
// 业务 handler 想 log "这次操作来自哪个 service" 时用。
func GetBotServiceServiceName(ctx *gin.Context) string {
	v, ok := ctx.Get(botServiceCtxKeyService)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// GetBotServiceJTI 从 ctx 取出当前请求 JWT 的 jti。每次 bot 签发的 jti 都不一样，
// 适合做 trace id（一次端到端调用串起来）。
func GetBotServiceJTI(ctx *gin.Context) string {
	v, ok := ctx.Get(botServiceCtxKeyJTI)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}
