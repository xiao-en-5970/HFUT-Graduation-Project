// scheduler/metrics_persist.go —— 指标持久化定时任务。
//
// 每分钟做一次：
//
//   1. 把 hfut backend 内存里的 minute_series UPSERT 到 metric_minute（source='http'）
//   2. 拉一次 QQ-bot 的 /internal/metrics（如果 BOT_INTERNAL_API_URL 配置了），把
//      bot 的 series UPSERT 进 metric_minute（source='bot'），把 recent_events 增量
//      INSERT 到 bot_dispatch_event（fingerprint 去重）
//   3. 每天清理一次 metric_minute（90 天前）和 bot_dispatch_event（30 天前）
//
// 设计权衡：
//
//   - pull > push：bot 端不修改一行代码就能接入持久化。代价是 bot 高峰期超过 50
//     条/分钟事件会丢；当前业务体量远低于此阈值，可接受。
//   - UPSERT 用最新累计覆盖：内存桶里始终是"该分钟从 0 累计到现在"的快照，
//     反复写不会丢精度。
//   - flush 间隔 = 1 分钟，跟桶粒度对齐；不要更长（最近 60min 桶里可能有还没落库
//     就被 GC 的）。

package scheduler

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/botinternal"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"go.uber.org/zap"
)

const (
	metricsFlushInterval  = time.Minute
	metricsRetentionDays  = 90
	botEventRetentionDays = 30
	botPullTimeout        = 5 * time.Second
	metricsFlushDBTimeout = 15 * time.Second
)

// StartMetricsPersister 启动后台 metrics 持久化任务。
//
// 进程退出（ctx 取消）时退出。函数立刻返回；ensure schema 在第一次 tick 之前完成，
// 失败仅 warn——schema 缺失最多让面板查不到历史数据，不影响业务。
func StartMetricsPersister(ctx context.Context) {
	go func() {
		// 启动期：确保 schema 存在；失败不阻断（当前进程内存里仍能展示实时数据）
		ensureCtx, cancel := context.WithTimeout(ctx, metricsFlushDBTimeout)
		if err := dao.EnsureMetricsSchema(ensureCtx); err != nil {
			logger.Warn(ctx, "metrics persister: ensure schema failed", zap.Error(err))
		}
		cancel()

		logger.Info(ctx, "scheduler: metrics persister started",
			zap.Duration("interval", metricsFlushInterval),
		)

		// 启动后立即 flush 一次：让重启后的"前一分钟桶"立刻进 DB，避免空缺
		runMetricsFlushOnce(ctx)

		t := time.NewTicker(metricsFlushInterval)
		defer t.Stop()

		// 每天对齐时刻（UTC 03:00，约北京 11:00）做一次清理；用 ticker 模拟即可
		var lastPurge time.Time
		for {
			select {
			case <-ctx.Done():
				logger.Info(ctx, "scheduler: metrics persister stopping", zap.Error(ctx.Err()))
				return
			case <-t.C:
				runMetricsFlushOnce(ctx)
				if time.Since(lastPurge) > 24*time.Hour {
					runMetricsPurgeOnce(ctx)
					lastPurge = time.Now()
				}
			}
		}
	}()
}

// runMetricsFlushOnce 单次 flush。所有错误仅记日志，绝不 panic。
func runMetricsFlushOnce(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, metricsFlushDBTimeout)
	defer cancel()

	rows := buildHTTPMetricRows()

	// pull 一次 bot snapshot；失败不阻塞 http 部分入库
	botRows, botEvents := pullBotMetricsLocked(parent)
	rows = append(rows, botRows...)

	if err := dao.UpsertMinutes(ctx, rows); err != nil {
		logger.Warn(parent, "metrics persister: upsert minutes failed",
			zap.Int("rows", len(rows)),
			zap.Error(err))
		return
	}

	if len(botEvents) > 0 {
		if n, err := dao.InsertBotEventsIgnoreDup(ctx, botEvents); err != nil {
			logger.Warn(parent, "metrics persister: insert bot events failed",
				zap.Int("input", len(botEvents)),
				zap.Error(err))
		} else if n > 0 {
			logger.Info(parent, "metrics persister: bot events ingested",
				zap.Int64("inserted", n),
				zap.Int("input", len(botEvents)))
		}
	}
}

// runMetricsPurgeOnce 清理过期数据，不阻塞 flush。
func runMetricsPurgeOnce(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()

	cutoffMin := time.Now().Add(-time.Duration(metricsRetentionDays) * 24 * time.Hour).Unix()
	if n, err := dao.PurgeMetricMinutes(ctx, cutoffMin); err != nil {
		logger.Warn(parent, "metrics persister: purge metric_minute failed", zap.Error(err))
	} else if n > 0 {
		logger.Info(parent, "metrics persister: purged metric_minute",
			zap.Int64("affected", n),
			zap.Int("retention_days", metricsRetentionDays))
	}

	cutoffEvent := time.Now().Add(-time.Duration(botEventRetentionDays) * 24 * time.Hour)
	if n, err := dao.PurgeBotEvents(ctx, cutoffEvent); err != nil {
		logger.Warn(parent, "metrics persister: purge bot_dispatch_event failed", zap.Error(err))
	} else if n > 0 {
		logger.Info(parent, "metrics persister: purged bot_dispatch_event",
			zap.Int64("affected", n),
			zap.Int("retention_days", botEventRetentionDays))
	}
}

// buildHTTPMetricRows 拉一次 service.Metrics() 的内存快照拼成 dao 行。
func buildHTTPMetricRows() []dao.MetricRow {
	snap := service.Metrics().MinuteSeriesSnapshot()
	rows := make([]dao.MetricRow, 0, len(snap)*6)
	for ts, m := range snap {
		for metric, val := range m {
			rows = append(rows, dao.MetricRow{
				MinuteTS: ts,
				Source:   "http",
				Metric:   metric,
				Value:    val,
			})
		}
	}
	return rows
}

// pullBotMetricsLocked pull bot snapshot，转 dao 行。失败返回空。
func pullBotMetricsLocked(parent context.Context) ([]dao.MetricRow, []dao.BotEventRow) {
	if botinternal.Default == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(parent, botPullTimeout)
	defer cancel()
	data, err := botinternal.Default.FetchMetrics(ctx)
	if err != nil {
		logger.Warn(parent, "metrics persister: pull bot snapshot failed", zap.Error(err))
		return nil, nil
	}
	// 把 snapshot 也写入实时面板（顺手）：admin metrics 端点也在 pull，但这里
	// 多一份能让"实时面板没人在看时"也有最近一份数据
	service.Metrics().PushBotMetrics(data)

	rows := botSeriesToRows(data)
	events := botEventsToRows(data)
	return rows, events
}

// botSeriesToRows 把 bot snapshot["series"] 转成 metric_minute 行。
//
// bot 那边 series item 字段（参 utils/metrics/metrics.go.seriesPoint）：
//
//	minute, ws_msgs, recognize_called, recognize_success, dispatch_success, dispatch_fail
//
// 我们映射到 metric_minute（source='bot'），metric 名跟字段同名以保持一致。
func botSeriesToRows(snap map[string]any) []dao.MetricRow {
	if snap == nil {
		return nil
	}
	raw, ok := snap["series"].([]any)
	if !ok {
		return nil
	}
	out := make([]dao.MetricRow, 0, len(raw)*5)
	for _, it := range raw {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		ts := toInt64(m["minute"])
		if ts <= 0 {
			continue
		}
		for _, metric := range []string{
			"ws_msgs", "recognize_called", "recognize_success",
			"dispatch_success", "dispatch_fail",
		} {
			v := toInt64(m[metric])
			out = append(out, dao.MetricRow{
				MinuteTS: ts,
				Source:   "bot",
				Metric:   metric,
				Value:    v,
			})
		}
	}
	return out
}

// botEventsToRows 把 bot snapshot["recent_events"] 转 dao。fingerprint 由本端按
// (group|user|action|ts_unix) 算，bot 不需要修改任何代码。
func botEventsToRows(snap map[string]any) []dao.BotEventRow {
	if snap == nil {
		return nil
	}
	raw, ok := snap["recent_events"].([]any)
	if !ok {
		return nil
	}
	out := make([]dao.BotEventRow, 0, len(raw))
	for _, it := range raw {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		tsStr, _ := m["ts"].(string)
		ts, err := time.Parse(time.RFC3339, tsStr)
		if err != nil || ts.IsZero() {
			continue
		}
		action, _ := m["action_type"].(string)
		if action == "" {
			continue
		}
		gid := toInt64(m["group_id"])
		uid := toInt64(m["user_id"])
		row := dao.BotEventRow{
			OccurredAt:  ts.UTC(),
			Action:      action,
			Outcome:     toString(m["outcome"]),
			Fingerprint: dao.BotEventFingerprint(gid, uid, action, ts),
		}
		if gid != 0 {
			row.GroupID = ptrInt64(gid)
		}
		if uid != 0 {
			row.UserID = ptrInt64(uid)
		}
		if t := toString(m["title"]); t != "" {
			row.Title = ptrString(t)
		}
		if c := toInt(m["category"]); c != 0 {
			row.Category = ptrInt16(int16(c))
		}
		// price 来自 bot snapshot 是元（float64）；DB 存"分"避免精度问题
		if f, ok := m["price"].(float64); ok && f > 0 {
			row.PriceCents = ptrInt(int(f * 100))
		}
		if conf, ok := m["confidence"].(float64); ok && conf > 0 {
			cf := float32(conf)
			row.Confidence = &cf
		}
		if r := toString(m["reason"]); r != "" {
			row.Reason = ptrString(r)
		}
		// outcome=fail/err 时，bot 没单独提供 err_message 字段——走 ack_text 兜底
		if row.Outcome == "fail" || row.Outcome == "err" {
			if ack := toString(m["ack_text"]); ack != "" {
				row.ErrMessage = ptrString(ack)
			}
		}
		out = append(out, row)
	}
	return out
}

func toInt64(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int:
		return int64(x)
	case int64:
		return x
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	}
	return 0
}

func toInt(v any) int { return int(toInt64(v)) }

func toString(v any) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func ptrInt64(v int64) *int64    { return &v }
func ptrString(v string) *string { return &v }
func ptrInt(v int) *int          { return &v }
func ptrInt16(v int16) *int16    { return &v }
