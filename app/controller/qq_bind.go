package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// =============================================================================
// QQ 绑定 / 解绑：用户主账号侧操作（走 JWTAuth），需要主账号已登录。
// 设计文档：QQ-bot 仓库 skill/bot/SKILL.md "绑定 QQ 流程" 段
// =============================================================================

// UserQQBindRequestCode: POST /api/v1/user/qq-bind/request-code
//
// Body: { "qq_number": "12345678" }
// Resp: { "ttl_seconds": 300 } 给前端展示倒计时
//
// 错误状态码：
//
//	400 = QQ 格式错 / 未绑学校 / 已绑过 QQ
//	404 = 目标 QQ 不是 bot 好友（让用户先去加好友）
//	429 = 限流（5min 内重复请求）
//	502 = bot 服务不可达
func UserQQBindRequestCode(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	var body struct {
		QQNumber string `json:"qq_number"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	ttl, err := service.QQBindRequestCode(ctx.Request.Context(), userID, body.QQNumber)
	if err != nil {
		// 限流先单独处理——要把剩余秒数 retry_after_seconds 放到 data，让前端做倒计时
		var throttled *service.ThrottledError
		if errors.As(err, &throttled) {
			ctx.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": throttled.Error(),
				"data":    gin.H{"retry_after_seconds": throttled.RetryAfterSeconds},
			})
			return
		}
		switch {
		case errors.Is(err, service.ErrQQNumberInvalid),
			errors.Is(err, service.ErrUserNotBoundSchool),
			errors.Is(err, service.ErrUserAlreadyBoundQQ),
			errors.Is(err, service.ErrUserNotFound):
			reply.ReplyErrWithMessage(ctx, err.Error())
		case errors.Is(err, service.ErrBotNotFriend):
			reply.ReplyErrWithCodeAndMessage(ctx, 404, 404, err.Error())
		case errors.Is(err, service.ErrBotUnavailable):
			reply.ReplyErrWithCodeAndMessage(ctx, 502, 502, err.Error())
		default:
			reply.ReplyInternalError(ctx, err)
		}
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"ttl_seconds": ttl})
}

// UserQQBindConfirm: POST /api/v1/user/qq-bind/confirm
//
// Body: { "qq_number": "12345678", "code": "123456" }
//
// 错误状态码：
//
//	400 = 验证码错 / 过期 / 格式错 / 未绑学校 / 已绑过 QQ
//	500 = DB 事务错
func UserQQBindConfirm(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	var body struct {
		QQNumber string `json:"qq_number"`
		Code     string `json:"code"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.QQBindConfirm(ctx.Request.Context(), userID, body.QQNumber, body.Code); err != nil {
		switch {
		case errors.Is(err, service.ErrCodeInvalid),
			errors.Is(err, service.ErrCodeExpired),
			errors.Is(err, service.ErrQQNumberInvalid),
			errors.Is(err, service.ErrUserNotBoundSchool),
			errors.Is(err, service.ErrUserAlreadyBoundQQ),
			errors.Is(err, service.ErrUserNotFound):
			reply.ReplyErrWithMessage(ctx, err.Error())
		default:
			reply.ReplyInternalError(ctx, err)
		}
		return
	}
	reply.ReplyOK(ctx)
}

// UserQQUnbindRequestCode: POST /api/v1/user/qq-unbind/request-code
//
// 无需 body；给当前绑定的 QQ 发"解绑确认验证码"私聊。
//
// 安全设计：跟绑定流程对称——要 QQ 端能收到验证码才能解绑，防主账号 token 被盗后
// 攻击者解绑 + 自己重新绑 + 盗取旗下账号数据。
//
// Resp: { ttl_seconds: 300 } 给前端展示倒计时
//
// 错误状态码：
//
//	400 = 当前账号没绑 QQ
//	429 = 限流（60s 内重复请求）；data 里 retry_after_seconds 给前端做按钮倒计时
//	502 = bot 服务不可达 / 用户已把 bot 删了好友
func UserQQUnbindRequestCode(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	ttl, err := service.QQUnbindRequestCode(ctx.Request.Context(), userID)
	if err != nil {
		// 限流：跟绑定 RequestCode 一样把 retry_after_seconds 放进 data
		var throttled *service.ThrottledError
		if errors.As(err, &throttled) {
			ctx.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": throttled.Error(),
				"data":    gin.H{"retry_after_seconds": throttled.RetryAfterSeconds},
			})
			return
		}
		switch {
		case errors.Is(err, service.ErrUserHasNoQQChild),
			errors.Is(err, service.ErrUserNotFound):
			reply.ReplyErrWithMessage(ctx, err.Error())
		case errors.Is(err, service.ErrBotUnavailable):
			reply.ReplyErrWithCodeAndMessage(ctx, 502, 502, err.Error())
		default:
			reply.ReplyInternalError(ctx, err)
		}
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"ttl_seconds": ttl})
}

// UserQQUnbindConfirm: POST /api/v1/user/qq-unbind/confirm
//
// Body: { "code": "123456" }
//
// 校验 code 命中 redis 里的解绑验证码 → 真把 parent_user_id 设回 NULL。
// 旗下账号的所有数据（商品 / 提问 / 订单）保留，变孤儿等以后再被绑回来。
//
// 错误状态码：
//
//	400 = 验证码错 / 过期 / 当前账号没绑 QQ
//	500 = DB 错
func UserQQUnbindConfirm(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := ctx.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.QQUnbindConfirm(ctx.Request.Context(), userID, body.Code); err != nil {
		switch {
		case errors.Is(err, service.ErrCodeInvalid),
			errors.Is(err, service.ErrCodeExpired),
			errors.Is(err, service.ErrUserHasNoQQChild),
			errors.Is(err, service.ErrUserNotFound):
			reply.ReplyErrWithMessage(ctx, err.Error())
		default:
			reply.ReplyInternalError(ctx, err)
		}
		return
	}
	reply.ReplyOK(ctx)
}
