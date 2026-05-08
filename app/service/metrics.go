// Package service 的 metrics.go 维护进程级运行时计数器，给运维面板展示。
//
// 设计原则：
//
//   - **不依赖外部 TSDB**：只用 sync.Map + sync/atomic，进程内常驻，重启清零；
//     这避免引入 Prometheus 等额外组件，对毕设系统足够。
//   - **不阻塞业务路径**：Inc/Observe 都是无锁路径；Snapshot 加 RLock 一次，
//     仅在面板查询时调用，频率极低。
//   - **聚合粒度按 route**：HTTP 路径模板化（Gin 已经解析过 `:id`、`:extType`），
//     直接用 c.FullPath() 做 key，避免按真实 URL 维度爆炸。
//
// 暴露给外部的接口：
//
//   - service.Metrics().IncRequest(method, route, status, biz, latencyMs)
//   - service.Metrics().Snapshot()
//   - service.Metrics().PushBotMetrics(payload)  // 接收 QQ-bot 推送的内部指标
package service

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsService 进程级单例。
type MetricsService struct {
	mu        sync.RWMutex
	startedAt time.Time

	totalRequests  atomic.Int64
	totalErrors4xx atomic.Int64
	totalErrors5xx atomic.Int64
	totalBizErrors atomic.Int64
	totalLatencyMs atomic.Int64 // 累加，再除以 totalRequests 得平均

	// per-route 计数：method+" "+route -> *routeMetric
	routes map[string]*routeMetric

	// QQ-bot 推送过来的最新一份指标快照
	botSnapshot map[string]any
	botUpdated  time.Time
}

type routeMetric struct {
	count      atomic.Int64
	error4xx   atomic.Int64
	error5xx   atomic.Int64
	bizErr     atomic.Int64
	latencySum atomic.Int64
}

var (
	metricsOnce     sync.Once
	metricsInstance *MetricsService
)

// Metrics 返回进程级 MetricsService 单例。
func Metrics() *MetricsService {
	metricsOnce.Do(func() {
		metricsInstance = &MetricsService{
			startedAt: time.Now(),
			routes:    make(map[string]*routeMetric),
		}
	})
	return metricsInstance
}

// IncRequest 中间件每次请求都会调用一次。
//
// route 已经是 Gin 的 FullPath（如 /api/v1/goods/:id）；空字符串表示未命中任何注册路由
// （404），统一记到 "<unmatched>" 桶里以免 routes map 暴增。
func (m *MetricsService) IncRequest(method, route string, status int, bizCode int, latencyMs int64) {
	if route == "" {
		route = "<unmatched>"
	}
	key := method + " " + route

	m.totalRequests.Add(1)
	m.totalLatencyMs.Add(latencyMs)
	switch {
	case status >= 500:
		m.totalErrors5xx.Add(1)
	case status >= 400:
		m.totalErrors4xx.Add(1)
	}
	// 非 0 / 非 200 的业务 code 单独计；用 200 做正常码兼容现有 reply.ReplyOK
	if bizCode != 0 && bizCode != 200 {
		m.totalBizErrors.Add(1)
	}

	// fast-path：先 RLock 试探
	m.mu.RLock()
	rm, ok := m.routes[key]
	m.mu.RUnlock()
	if !ok {
		m.mu.Lock()
		rm, ok = m.routes[key]
		if !ok {
			rm = &routeMetric{}
			m.routes[key] = rm
		}
		m.mu.Unlock()
	}
	rm.count.Add(1)
	rm.latencySum.Add(latencyMs)
	switch {
	case status >= 500:
		rm.error5xx.Add(1)
	case status >= 400:
		rm.error4xx.Add(1)
	}
	if bizCode != 0 && bizCode != 200 {
		rm.bizErr.Add(1)
	}
}

// Snapshot 拿一份当前指标快照，给运维面板展示。
func (m *MetricsService) Snapshot() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	type routeRow struct {
		Method     string  `json:"method"`
		Route      string  `json:"route"`
		Count      int64   `json:"count"`
		Errors4xx  int64   `json:"errors_4xx"`
		Errors5xx  int64   `json:"errors_5xx"`
		BizErrors  int64   `json:"biz_errors"`
		AvgLatency float64 `json:"avg_latency_ms"`
	}
	rows := make([]routeRow, 0, len(m.routes))
	for key, rm := range m.routes {
		method, route := splitMethodRoute(key)
		c := rm.count.Load()
		avg := 0.0
		if c > 0 {
			avg = float64(rm.latencySum.Load()) / float64(c)
		}
		rows = append(rows, routeRow{
			Method: method, Route: route, Count: c,
			Errors4xx: rm.error4xx.Load(), Errors5xx: rm.error5xx.Load(),
			BizErrors: rm.bizErr.Load(), AvgLatency: avg,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Count > rows[j].Count })

	total := m.totalRequests.Load()
	avgAll := 0.0
	if total > 0 {
		avgAll = float64(m.totalLatencyMs.Load()) / float64(total)
	}

	out := map[string]any{
		"started_at":       m.startedAt.Format(time.RFC3339),
		"uptime_seconds":   int64(time.Since(m.startedAt).Seconds()),
		"total_requests":   total,
		"total_errors_4xx": m.totalErrors4xx.Load(),
		"total_errors_5xx": m.totalErrors5xx.Load(),
		"total_biz_errors": m.totalBizErrors.Load(),
		"avg_latency_ms":   avgAll,
		"routes":           rows,
		"bot": map[string]any{
			"updated_at": m.botUpdated.Format(time.RFC3339),
			"data":       m.botSnapshot,
		},
	}
	return out
}

// PushBotMetrics 由 admin metrics handler 主动 pull QQ-bot 后写入；
// 也可允许 QQ-bot 主动 push（当前实现走 pull）。
func (m *MetricsService) PushBotMetrics(payload map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.botSnapshot = payload
	m.botUpdated = time.Now()
}

func splitMethodRoute(key string) (method, route string) {
	for i := 0; i < len(key); i++ {
		if key[i] == ' ' {
			return key[:i], key[i+1:]
		}
	}
	return "", key
}
