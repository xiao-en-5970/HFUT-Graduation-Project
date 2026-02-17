package reply

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
)

// ReplyOK 成功响应（无数据）
func ReplyOK(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, response.Response{
		Code:    errcode.Success,
		Message: errcode.GetMsg(errcode.Success),
	})
}

// ReplyOKWithData 成功响应（带数据）
func ReplyOKWithData(ctx *gin.Context, data interface{}) {
	ctx.JSON(http.StatusOK, response.Response{
		Code:    errcode.Success,
		Message: errcode.GetMsg(errcode.Success),
		Data:    data,
	})
}

// ReplyOKWithMessage 成功响应（自定义消息）
func ReplyOKWithMessage(ctx *gin.Context, message string) {
	ctx.JSON(http.StatusOK, response.Response{
		Code:    errcode.Success,
		Message: message,
	})
}

// ReplyOKWithMessageAndData 成功响应（自定义消息和数据）
func ReplyOKWithMessageAndData(ctx *gin.Context, message string, data interface{}) {
	ctx.JSON(http.StatusOK, response.Response{
		Code:    errcode.Success,
		Message: message,
		Data:    data,
	})
}

// ReplyErr 错误响应（使用错误码）
func ReplyErr(ctx *gin.Context, err error) {
	httpStatus := getHTTPStatus(500)
	ctx.JSON(httpStatus, response.Response{
		Code:    500,
		Message: err.Error(),
	})
}

// ReplyErrWithMessage 错误响应（使用错误码和自定义消息）
func ReplyErrWithMessage(ctx *gin.Context, message string) {
	httpStatus := getHTTPStatus(500)
	ctx.JSON(httpStatus, response.Response{
		Code:    500,
		Message: message,
	})
}

// ReplyErrWithCode 错误响应（使用错误码，可自定义HTTP状态码）
func ReplyErrWithCode(ctx *gin.Context, httpStatus int, code int) {
	ctx.JSON(httpStatus, response.Response{
		Code:    code,
		Message: errcode.GetMsg(code),
	})
}

// ReplyErrWithCodeAndMessage 错误响应（使用错误码和自定义消息，可自定义HTTP状态码）
func ReplyErrWithCodeAndMessage(ctx *gin.Context, httpStatus int, code int, message string) {
	ctx.JSON(httpStatus, response.Response{
		Code:    code,
		Message: message,
	})
}

// ReplyInvalidParams 参数错误响应
func ReplyInvalidParams(ctx *gin.Context, err error) {
	message := errcode.GetMsg(errcode.InvalidParams)
	if err != nil {
		message = message + ": " + err.Error()
	}
	ctx.JSON(http.StatusBadRequest, response.Response{
		Code:    errcode.InvalidParams,
		Message: message,
	})
}

// ReplyUnauthorized 未认证响应
func ReplyUnauthorized(ctx *gin.Context) {
	ctx.JSON(http.StatusUnauthorized, response.Response{
		Code:    errcode.Unauthorized,
		Message: errcode.GetMsg(errcode.Unauthorized),
	})
}

// ReplyForbidden 无权限响应
func ReplyForbidden(ctx *gin.Context) {
	ctx.JSON(http.StatusForbidden, response.Response{
		Code:    errcode.Forbidden,
		Message: errcode.GetMsg(errcode.Forbidden),
	})
}

// ReplyNotFound 资源不存在响应
func ReplyNotFound(ctx *gin.Context, code int) {
	ctx.JSON(http.StatusNotFound, response.Response{
		Code:    code,
		Message: errcode.GetMsg(code),
	})
}

// ReplyInternalError 服务器内部错误响应
func ReplyInternalError(ctx *gin.Context, err error) {
	message := errcode.GetMsg(errcode.InternalServerError)
	if err != nil {
		message = message + ": " + err.Error()
	}
	ctx.JSON(http.StatusInternalServerError, response.Response{
		Code:    errcode.InternalServerError,
		Message: message,
	})
}

// getHTTPStatus 根据错误码获取HTTP状态码
func getHTTPStatus(code int) int {
	switch code {
	case errcode.InvalidParams:
		return http.StatusBadRequest
	case errcode.Unauthorized:
		return http.StatusUnauthorized
	case errcode.Forbidden:
		return http.StatusForbidden
	case errcode.NotFound:
		return http.StatusNotFound
	case errcode.InternalServerError:
		return http.StatusInternalServerError
	default:
		// 根据错误码范围判断
		if code >= 2000 && code < 3000 {
			// 业务错误，通常返回400
			return http.StatusBadRequest
		}
		return http.StatusInternalServerError
	}
}
