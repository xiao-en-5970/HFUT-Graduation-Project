package controller

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
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

// AdminMetricsTimeSeries GET /api/v1/admin/metrics/timeseries
//
// Query：
//
//	start / end       —— epoch 秒。end 缺省 = 现在；start 缺省 = end - 1h。
//	                     最大窗口 90d（与 metric_minute 保留期一致），超出会截断到 90d。
//	step              —— 下采样粒度（秒）；默认按窗口自动选：
//	                       ≤  1h  → 60s（不下采样）
//	                       ≤  6h  → 300s
//	                       ≤ 24h  → 900s
//	                       ≤  7d  → 3600s
//	                       其它   → 86400s
//	source            —— 'http' / 'bot' / 'all'（默认 all）
//	metrics           —— 逗号分隔白名单；空 = 全部
//
// 返回结构：
//
//	{
//	  "start": 1700000000, "end": 1700003600, "step": 60,
//	  "series": [
//	    { "source": "http", "metric": "requests",
//	      "points": [ {"t": 1700000000, "v": 12 }, ... ] },
//	    ...
//	  ]
//	}
//
// 下采样规则：每个 (source, metric) 在每个 step 桶里 SUM 起来——counter 类指标
// SUM 是正确语义，平均延迟由前端用 latency_sum/latency_count 自行算。
func AdminMetricsTimeSeries(c *gin.Context) {
	now := time.Now().Unix()
	end := parseInt64Default(c.Query("end"), now)
	start := parseInt64Default(c.Query("start"), end-3600)
	if end > now {
		end = now
	}
	if start >= end {
		start = end - 3600
	}
	// 上限 90d
	const maxWindow = int64(90 * 24 * 3600)
	if end-start > maxWindow {
		start = end - maxWindow
	}
	// 对齐到分钟（DB 是分钟桶）
	start = (start / 60) * 60
	end = ((end + 59) / 60) * 60

	step := parseInt64Default(c.Query("step"), 0)
	if step < 60 {
		step = autoStep(end - start)
	}
	// step 必须是 60 的倍数
	if step%60 != 0 {
		step = (step / 60) * 60
	}
	if step < 60 {
		step = 60
	}

	source := strings.TrimSpace(c.Query("source"))
	metrics := splitNonEmpty(c.Query("metrics"))

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	rows, err := dao.QueryTimeSeries(ctx, start, end, source, metrics)
	if err != nil {
		reply.ReplyInternalError(c, err)
		return
	}

	// 下采样：(source, metric) -> (bucket_start_ts) -> sum
	type seriesKey struct{ source, metric string }
	agg := map[seriesKey]map[int64]int64{}
	for _, r := range rows {
		bucket := (r.MinuteTS / step) * step
		k := seriesKey{r.Source, r.Metric}
		if agg[k] == nil {
			agg[k] = map[int64]int64{}
		}
		agg[k][bucket] += r.Value
	}

	type point struct {
		T int64 `json:"t"`
		V int64 `json:"v"`
	}
	type series struct {
		Source string  `json:"source"`
		Metric string  `json:"metric"`
		Points []point `json:"points"`
	}
	out := make([]series, 0, len(agg))
	for k, buckets := range agg {
		// 用 bucket 时间戳作为 X 轴：从 start 到 end 按 step 步长生成完整序列，
		// 缺失的时刻填 0——前端折线图就不会因为"中间没数据"而画错位
		points := make([]point, 0, (end-start)/step+1)
		for t := (start / step) * step; t <= end; t += step {
			points = append(points, point{T: t, V: buckets[t]})
		}
		out = append(out, series{Source: k.source, Metric: k.metric, Points: points})
	}

	reply.ReplyOKWithData(c, gin.H{
		"start":  start,
		"end":    end,
		"step":   step,
		"series": out,
	})
}

// AdminMetricsEvents GET /api/v1/admin/metrics/events
//
// Query：start / end（epoch 秒，缺省同 timeseries）, action, outcome,
//
//	limit (≤500), offset。返回 { total, list, limit, offset }。
func AdminMetricsEvents(c *gin.Context) {
	now := time.Now().Unix()
	end := parseInt64Default(c.Query("end"), now)
	start := parseInt64Default(c.Query("start"), end-3600)
	limit := int(parseInt64Default(c.Query("limit"), 100))
	offset := int(parseInt64Default(c.Query("offset"), 0))

	startAt := time.Unix(start, 0)
	endAt := time.Unix(end, 0)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	rows, total, err := dao.QueryBotEvents(ctx, startAt, endAt,
		strings.TrimSpace(c.Query("action")),
		strings.TrimSpace(c.Query("outcome")),
		limit, offset)
	if err != nil {
		reply.ReplyInternalError(c, err)
		return
	}
	reply.ReplyOKWithData(c, gin.H{
		"total":  total,
		"limit":  limit,
		"offset": offset,
		"list":   rows,
	})
}

// autoStep 根据时间窗自动选下采样粒度。和前端 chip 1h/6h/24h/7d 对齐。
func autoStep(windowSeconds int64) int64 {
	switch {
	case windowSeconds <= 3600:
		return 60
	case windowSeconds <= 6*3600:
		return 300
	case windowSeconds <= 24*3600:
		return 900
	case windowSeconds <= 7*24*3600:
		return 3600
	default:
		return 86400
	}
}

func parseInt64Default(s string, def int64) int64 {
	if s == "" {
		return def
	}
	if n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
		return n
	}
	return def
}

func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
