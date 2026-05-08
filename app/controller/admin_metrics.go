package controller

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/botinternal"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// AdminMetrics GET /api/v1/admin/metrics
//
// 拼接两份指标：
//
//   - 进程级 service.Metrics() 快照（HTTP 路由总量 / 错误率 / 平均延迟）
//   - QQ-bot 内部 /internal/metrics 拉到的最新一份（识别 / 上架 / 下架 / 限流命中等）
//
// 拉取 bot 失败不阻塞主响应：把 bot.error 写到响应里，前端自行展示。
func AdminMetrics(c *gin.Context) {
	if botinternal.Default != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if data, err := botinternal.Default.FetchMetrics(ctx); err == nil {
			service.Metrics().PushBotMetrics(data)
		} else {
			service.Metrics().PushBotMetrics(map[string]any{"error": err.Error()})
		}
	} else {
		service.Metrics().PushBotMetrics(map[string]any{"error": "BOT_INTERNAL_API_URL 未配置"})
	}
	reply.ReplyOKWithData(c, service.Metrics().Snapshot())
}
