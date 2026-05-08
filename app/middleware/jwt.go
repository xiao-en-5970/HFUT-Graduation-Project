package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/util"
)

// AccessTokenExpiredCode 业务码：access token 过期，前端应以 refresh token 换发后重试。
//
// 中间件用 HTTP 401 + 这个 code 区分"token 过期"与"无 token / token 格式错"，让客户端
// 拦截器只对 4011 触发自动刷新，避免把"未登录"也错刷一遍。
const AccessTokenExpiredCode = 4011

// JWTAuth JWT 认证中间件——只接受 access token。
//
// 错误分流（统一 HTTP 401）：
//
//   - code=401  缺 token / 格式错 / 签名不对 / 用 refresh token 来打业务接口
//   - code=4011 access token 已过期 —— 前端应自动调 /user/refresh 拿新 token 后 retry
//
// refresh token 永远不能拿来打业务接口（util.ParseAccessToken 会因 typ=refresh 拒绝）。
func JWTAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.JSON(http.StatusUnauthorized, response.Response{
				Code:    401,
				Message: "未提供认证 token",
			})
			ctx.Abort()
			return
		}
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

		claims, err := util.ParseAccessToken(token)
		if err != nil {
			if errors.Is(err, util.ErrTokenExpired) {
				ctx.JSON(http.StatusUnauthorized, response.Response{
					Code:    AccessTokenExpiredCode,
					Message: "access token 已过期",
				})
				ctx.Abort()
				return
			}
			ctx.JSON(http.StatusUnauthorized, response.Response{
				Code:    401,
				Message: "无效的 token: " + err.Error(),
			})
			ctx.Abort()
			return
		}

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

// LoadUserSchool 加载当前用户的学校ID到上下文，需在 JWTAuth 之后使用
func LoadUserSchool() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := GetUserID(ctx)
		if userID == 0 {
			ctx.Next()
			return
		}
		user, err := dao.User().GetByID(ctx.Request.Context(), userID)
		if err != nil {
			ctx.Next()
			return
		}
		ctx.Set("school_id", user.SchoolID)
		ctx.Next()
	}
}

// GetSchoolID 从上下文获取学校ID（需在 LoadUserSchool 之后调用，否则返回 0）
func GetSchoolID(ctx *gin.Context) uint {
	if schoolID, exists := ctx.Get("school_id"); exists {
		if id, ok := schoolID.(uint); ok {
			return id
		}
	}
	return 0
}
