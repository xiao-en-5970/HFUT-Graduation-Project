package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/util"
)

// JWTAuth JWT 认证中间件
func JWTAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 从请求头获取 token
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.JSON(http.StatusUnauthorized, response.Response{
				Code:    401,
				Message: "未提供认证 token",
			})
			ctx.Abort()
			return
		}

		// 检查 Bearer 前缀
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			ctx.JSON(http.StatusUnauthorized, response.Response{
				Code:    401,
				Message: "认证 token 格式错误，应为: Bearer <token>",
			})
			ctx.Abort()
			return
		}

		token := parts[1]

		// 解析 token
		claims, err := util.ParseToken(token)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, response.Response{
				Code:    401,
				Message: "无效的 token: " + err.Error(),
			})
			ctx.Abort()
			return
		}

		// 将用户信息存储到上下文中
		ctx.Set("user_id", claims.UserID)
		ctx.Set("username", claims.Username)

		ctx.Next()
	}
}

// GetUserID 从上下文获取用户 ID
func GetUserID(ctx *gin.Context) uint {
	if userID, exists := ctx.Get("user_id"); exists {
		if id, ok := userID.(uint); ok {
			return id
		}
	}
	return 0
}

// GetUsername 从上下文获取用户名
func GetUsername(ctx *gin.Context) string {
	if username, exists := ctx.Get("username"); exists {
		if name, ok := username.(string); ok {
			return name
		}
	}
	return ""
}
