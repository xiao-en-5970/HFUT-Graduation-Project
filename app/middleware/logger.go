package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"go.uber.org/zap"
)

// ZapLogger zap 日志中间件
func ZapLogger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		path := ctx.Request.URL.Path
		raw := ctx.Request.URL.RawQuery

		// 处理请求
		ctx.Next()

		// 计算请求耗时
		latency := time.Since(start)

		// 构建日志字段
		fields := []zap.Field{
			zap.Int("status", ctx.Writer.Status()),
			zap.String("method", ctx.Request.Method),
			zap.String("path", path),
			zap.String("ip", ctx.ClientIP()),
			zap.Duration("latency", latency),
			zap.String("user_agent", ctx.Request.UserAgent()),
		}

		// 如果有查询参数，添加到日志
		if raw != "" {
			fields = append(fields, zap.String("query", raw))
		}

		// 如果有用户ID，添加到日志
		if userID := GetUserID(ctx); userID > 0 {
			fields = append(fields, zap.Uint("user_id", userID))
		}

		// 如果有错误，记录错误信息
		if len(ctx.Errors) > 0 {
			fields = append(fields, zap.Strings("errors", ctx.Errors.Errors()))
		}

		// 根据状态码选择日志级别
		status := ctx.Writer.Status()
		switch {
		case status >= 500:
			logger.Logger.Error("HTTP Request", fields...)
		case status >= 400:
			logger.Logger.Warn("HTTP Request", fields...)
		default:
			logger.Logger.Info("HTTP Request", fields...)
		}
	}
}

