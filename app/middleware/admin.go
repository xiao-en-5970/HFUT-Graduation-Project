package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

// AdminAuth 管理员权限中间件，需在 JWTAuth 之后使用，role 至少为管理员
func AdminAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := GetUserID(ctx)
		if userID == 0 {
			ctx.JSON(http.StatusUnauthorized, response.Response{
				Code:    401,
				Message: "未认证",
			})
			ctx.Abort()
			return
		}
		user, err := dao.User().GetByID(ctx.Request.Context(), userID)
		if err != nil || user == nil {
			ctx.JSON(http.StatusForbidden, response.Response{
				Code:    403,
				Message: "用户不存在",
			})
			ctx.Abort()
			return
		}
		if user.Status == constant.StatusInvalid {
			ctx.JSON(http.StatusForbidden, response.Response{
				Code:    403,
				Message: "账户已禁用",
			})
			ctx.Abort()
			return
		}
		if user.Role < constant.RoleAdmin {
			ctx.JSON(http.StatusForbidden, response.Response{
				Code:    403,
				Message: "无管理员权限",
			})
			ctx.Abort()
			return
		}
		ctx.Set("admin_role", user.Role)
		ctx.Next()
	}
}
