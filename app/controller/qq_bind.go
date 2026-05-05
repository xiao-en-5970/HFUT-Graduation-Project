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

// UserQQUnbind: POST /api/v1/user/qq-unbind
//
// 无需 body；解绑当前主账号下的旗下账号（parent_user_id 设回 NULL，数据保留）。
//
// 错误状态码：
//
//	400 = 当前账号没绑 QQ
//	500 = DB 错
func UserQQUnbind(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	if err := service.QQUnbind(ctx.Request.Context(), userID); err != nil {
		switch {
		case errors.Is(err, service.ErrUserHasNoQQChild),
			errors.Is(err, service.ErrUserNotFound):
			reply.ReplyErrWithMessage(ctx, err.Error())
		default:
			reply.ReplyInternalError(ctx, err)
		}
		return
	}
	reply.ReplyOK(ctx)
}
