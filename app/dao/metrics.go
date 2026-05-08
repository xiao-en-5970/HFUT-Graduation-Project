// dao/metrics.go —— 运维指标持久化 DAO。
//
// 两张目标表：
//
//   - metric_minute       分钟桶聚合（source/metric 维度长表）
//   - bot_dispatch_event  bot 自动识别 + 派发事件流（含 AI 判断的 reason）
//
// schema 定义同 package/sql/migrate_metrics_persistence.sql；本 dao 提供启动时
// EnsureSchema（IF NOT EXISTS DDL，幂等安全）+ flush UPSERT + 时间窗口查询 +
// 旧数据清理。所有操作要求 ctx 已带超时；persister 调用方负责。
package dao

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"gorm.io/gorm/clause"
)

// MetricRow 单条 metric_minute 行。
type MetricRow struct {
	MinuteTS int64  `gorm:"column:minute_ts" json:"minute_ts"`
	Source   string `gorm:"column:source"    json:"source"`
	Metric   string `gorm:"column:metric"    json:"metric"`
	Value    int64  `gorm:"column:value"     json:"value"`
}

// TableName metric_minute 实际表名（GORM 不要复数化）。
func (MetricRow) TableName() string { return "metric_minute" }

// BotEventRow 单条 bot_dispatch_event 行。
type BotEventRow struct {
	ID          int64     `gorm:"column:id;primaryKey"   json:"id"`
	OccurredAt  time.Time `gorm:"column:occurred_at"     json:"occurred_at"`
	GroupID     *int64    `gorm:"column:group_id"        json:"group_id,omitempty"`
	UserID      *int64    `gorm:"column:user_id"         json:"user_id,omitempty"`
	Action      string    `gorm:"column:action"          json:"action"`
	Outcome     string    `gorm:"column:outcome"         json:"outcome"`
	Title       *string   `gorm:"column:title"           json:"title,omitempty"`
	Category    *int16    `gorm:"column:category"        json:"category,omitempty"`
	PriceCents  *int      `gorm:"column:price_cents"     json:"price_cents,omitempty"`
	Confidence  *float32  `gorm:"column:confidence"      json:"confidence,omitempty"`
	Reason      *string   `gorm:"column:reason"          json:"reason,omitempty"`
	ErrMessage  *string   `gorm:"column:err_message"     json:"err_message,omitempty"`
	Fingerprint string    `gorm:"column:fingerprint"     json:"-"`
}

func (BotEventRow) TableName() string { return "bot_dispatch_event" }

// EnsureMetricsSchema 启动时调一次：CREATE TABLE IF NOT EXISTS。
//
// 跑一遍即使表已存在也无副作用——避免运维忘了应用 migration 时面板默默不工作。
// 如果 DDL 失败（DB 不连通 / 权限不足）返回错误；调用方决定是否致命。
func EnsureMetricsSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS metric_minute (
			minute_ts bigint      NOT NULL,
			source    varchar(16) NOT NULL,
			metric    varchar(64) NOT NULL,
			value     bigint      NOT NULL DEFAULT 0,
			PRIMARY KEY (minute_ts, source, metric)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_metric_minute_source_metric_ts
			ON metric_minute (source, metric, minute_ts)`,
		`CREATE INDEX IF NOT EXISTS idx_metric_minute_ts
			ON metric_minute (minute_ts)`,
		`CREATE TABLE IF NOT EXISTS bot_dispatch_event (
			id           bigserial PRIMARY KEY,
			occurred_at  timestamptz NOT NULL,
			group_id     bigint,
			user_id      bigint,
			action       varchar(32) NOT NULL,
			outcome      varchar(16) NOT NULL,
			title        varchar(256),
			category     smallint,
			price_cents  integer,
			confidence   real,
			reason       text,
			err_message  text,
			fingerprint  varchar(64) NOT NULL UNIQUE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bot_event_occurred_desc
			ON bot_dispatch_event (occurred_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_bot_event_action_occurred
			ON bot_dispatch_event (action, occurred_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_bot_event_outcome_occurred
			ON bot_dispatch_event (outcome, occurred_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_bot_event_user_occurred
			ON bot_dispatch_event (user_id, occurred_at DESC)`,
	}
	for _, s := range stmts {
		if err := pgsql.DB.WithContext(ctx).Exec(s).Error; err != nil {
			return fmt.Errorf("ensure metrics schema: %w", err)
		}
	}
	return nil
}

// UpsertMinutes 批量 UPSERT 多条 metric_minute 行。
//
// PK = (minute_ts, source, metric)；冲突时**直接覆盖** value——这是因为内存里的
// 是同分钟的累计值，每分钟我们都用最新累计覆盖 DB，比较"increment by delta"模式
// 简单且无需在 hfut 端维护"上次写到哪了"的状态。
func UpsertMinutes(ctx context.Context, rows []MetricRow) error {
	if len(rows) == 0 {
		return nil
	}
	return pgsql.DB.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "minute_ts"}, {Name: "source"}, {Name: "metric"}},
			DoUpdates: clause.AssignmentColumns([]string{"value"}),
		}).
		CreateInBatches(rows, 200).Error
}

// QueryTimeSeries 按 [start, end] 拉 metric_minute；返回原始分钟粒度数据。
//
// 上层（API/前端）按 step 自行下采样：本函数不做下采样，避免读后再聚合时丢精度。
// 时间窗最多 7 天 + 60 metric × 3 source ≈ 几万行，PG 单查不到 100ms。
func QueryTimeSeries(ctx context.Context, startTS, endTS int64, source string, metrics []string) ([]MetricRow, error) {
	q := pgsql.DB.WithContext(ctx).
		Table("metric_minute").
		Where("minute_ts >= ? AND minute_ts <= ?", startTS, endTS)
	if source != "" && source != "all" {
		q = q.Where("source = ?", source)
	}
	if len(metrics) > 0 {
		q = q.Where("metric IN ?", metrics)
	}
	var rows []MetricRow
	err := q.Order("minute_ts ASC").Find(&rows).Error
	return rows, err
}

// PurgeMetricMinutes 删除 cutoffTS 之前的 metric_minute 行；返回删除条数。
func PurgeMetricMinutes(ctx context.Context, cutoffTS int64) (int64, error) {
	res := pgsql.DB.WithContext(ctx).
		Where("minute_ts < ?", cutoffTS).
		Delete(&MetricRow{})
	return res.RowsAffected, res.Error
}

// InsertBotEventsIgnoreDup 批量插入；fingerprint 冲突的行**静默跳过**（DO NOTHING）。
//
// 这是 pull 模式幂等的核心：hfut 每分钟拉一次 bot 的 ring buffer (50 条)，新事件
// 进表，老事件 skip。两次 pull 之间 bot 收到 < 50 条事件就保证不丢。
func InsertBotEventsIgnoreDup(ctx context.Context, rows []BotEventRow) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	res := pgsql.DB.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "fingerprint"}},
			DoNothing: true,
		}).
		CreateInBatches(rows, 100)
	return res.RowsAffected, res.Error
}

// QueryBotEvents 时间窗 + 可选筛选条件。
//
// 用于面板事件表：默认按 occurred_at 倒序，limit/offset 分页；按 action/outcome 过滤。
func QueryBotEvents(ctx context.Context, startAt, endAt time.Time, action, outcome string, limit, offset int) ([]BotEventRow, int64, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	q := pgsql.DB.WithContext(ctx).Model(&BotEventRow{}).
		Where("occurred_at >= ? AND occurred_at <= ?", startAt, endAt)
	if action != "" {
		q = q.Where("action = ?", action)
	}
	if outcome != "" {
		q = q.Where("outcome = ?", outcome)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []BotEventRow
	err := q.Order("occurred_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

// PurgeBotEvents 删除 cutoff 之前的 bot 事件。
func PurgeBotEvents(ctx context.Context, cutoff time.Time) (int64, error) {
	res := pgsql.DB.WithContext(ctx).
		Where("occurred_at < ?", cutoff).
		Delete(&BotEventRow{})
	return res.RowsAffected, res.Error
}

// BotEventFingerprint 给一条 bot 事件算 fingerprint：sha1(group|user|action|ts_unix)。
//
// bot snapshot ring buffer 给的 TS 字段是 RFC3339 秒级，所以同一秒内同一群同一用户做
// 同一 action 会被去重——业务上正常情况不会发生（ring buffer 也不会保留两条），所以
// 这个粒度足够安全。
func BotEventFingerprint(groupID, userID int64, action string, ts time.Time) string {
	h := sha1.New()
	fmt.Fprintf(h, "%d|%d|%s|%d", groupID, userID, strings.ToLower(action), ts.Unix())
	return hex.EncodeToString(h.Sum(nil))
}
