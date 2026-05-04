package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
)

// botServiceTokenHeader bot 调 hfut 时携带 service token 的 header 名。
//
// 用 X- 前缀表明这是非标准（自定义）头；跟 JWT 的 Authorization: Bearer 区分开，
// 避免反代或日志中间件把它当登录凭证打日志（日志已有屏蔽规则的话会少漏）。
const botServiceTokenHeader = "X-Bot-Service-Token"

// botServiceTokenIDCtxKey ctx 里存"当前用的 token id"用的 key。
// 业务 handler 可以拿这个值做审计 / 限流（"这次操作是哪个 token 发起的"）。
const botServiceTokenIDCtxKey = "bot_service_token_id"

// BotServiceAuth 校验 X-Bot-Service-Token header；token 有效（未作废 + 未过期）才放行。
//
// 跟 JWTAuth 不一样的是：
//   - 不依赖用户身份；这是服务间互信
//   - 任何调用都必须显式带 token；缺失 = 401
//   - 校验方式：sha256(明文) → 查 bot_service_tokens 表
//
// 校验通过后把 token id 写进 ctx；handler 可用 GetBotServiceTokenID(ctx) 取。
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
		t, err := service.VerifyBotServiceToken(ctx.Request.Context(), plain)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, response.Response{
				Code:    500,
				Message: "service token 校验失败: " + err.Error(),
			})
			ctx.Abort()
			return
		}
		if t == nil {
			ctx.JSON(http.StatusUnauthorized, response.Response{
				Code:    401,
				Message: "service token 无效（已作废 / 已过期 / 不存在）",
			})
			ctx.Abort()
			return
		}
		ctx.Set(botServiceTokenIDCtxKey, t.ID)
		ctx.Next()
	}
}

// GetBotServiceTokenID 从 ctx 取出当前请求用的 token id；非 bot 路由组返回 0。
func GetBotServiceTokenID(ctx *gin.Context) uint {
	v, ok := ctx.Get(botServiceTokenIDCtxKey)
	if !ok {
		return 0
	}
	id, _ := v.(uint)
	return id
}
