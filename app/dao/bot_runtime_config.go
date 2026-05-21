// dao/bot_runtime_config.go —— QQ-bot 运行时配置 KV DAO。
//
// 目标表 bot_runtime_config，schema 同 package/sql/migrate_bot_runtime_config.sql。
//
// 设计：
//   - 单表 KV (key text PK, value jsonb)；将来扩字段不改 schema
//   - 启动 EnsureSchema 幂等建表 + 种子默认 keys
//   - bot 周期调 GET /api/v1/bot/runtime-config 一次拉全集，本地热替换
//   - admin 改完单 key 直接 UPSERT；不强求 bot 立即生效（等下一次 poll 周期）
package dao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"gorm.io/gorm"
)

// 支持的 key 名 + 默认值。改名 / 加 key 都从这里开始——bot 端要同步加解析。
const (
	BotCfgAutoReplyWhitelist = "auto_reply_whitelist"
	BotCfgOpsGroupIDs        = "ops_group_ids"
	BotCfgSilentMode         = "silent_mode"
)

// BotRuntimeConfigRow 单行表示。Value 持 JSON 原始 bytes，由调用方按 key 解读。
type BotRuntimeConfigRow struct {
	Key       string          `gorm:"column:key;primaryKey"   json:"key"`
	Value     json.RawMessage `gorm:"column:value;type:jsonb" json:"value"`
	UpdatedAt int64           `gorm:"column:updated_at;autoUpdateTime:false" json:"updated_at"`
	UpdatedBy *int64          `gorm:"column:updated_by"       json:"updated_by,omitempty"`
	Comment   *string         `gorm:"column:comment"          json:"comment,omitempty"`
}

func (BotRuntimeConfigRow) TableName() string { return "bot_runtime_config" }

// BotRuntimeConfigStore DAO 入口。
type BotRuntimeConfigStore struct{}

// BotRuntimeConfig 返回 store 实例（跟 dao.User() / dao.School() 的风格保持一致）。
func BotRuntimeConfig() *BotRuntimeConfigStore { return &BotRuntimeConfigStore{} }

// EnsureBotRuntimeConfigSchema 启动时调一次：CREATE TABLE IF NOT EXISTS + 写入默认 keys。
//
// 与 EnsureMetricsSchema 同模式：失败返回 error 让上游决定是否致命。
func EnsureBotRuntimeConfigSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS bot_runtime_config (
			key        VARCHAR(64) PRIMARY KEY,
			value      JSONB NOT NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_by INTEGER REFERENCES users(id),
			comment    TEXT
		)`,
	}
	for _, s := range stmts {
		if err := pgsql.DB.WithContext(ctx).Exec(s).Error; err != nil {
			return fmt.Errorf("ensure bot_runtime_config schema: %w", err)
		}
	}
	// 种子默认值——已存在的 key 不覆盖
	seeds := []struct {
		key, val, comment string
	}{
		{
			BotCfgAutoReplyWhitelist, `[]`,
			"启用自动监听的 QQ 群号列表。空列表 = bot 不在任何群里自动识别，仅响应 @bot 命令",
		},
		{
			BotCfgOpsGroupIDs, `[1084352497]`,
			"运维通知 / 运维查询群。bot 上架成功 / 群接入申请等通知会广播；群内 @bot 会进运维 SQL 查询路径",
		},
		{
			BotCfgSilentMode, `false`,
			"灰度静默模式。开启后非运维群的群消息、所有私聊、群文件上传都被静默；运维群 + hfut 落库照常",
		},
	}
	for _, s := range seeds {
		if err := pgsql.DB.WithContext(ctx).Exec(
			`INSERT INTO bot_runtime_config (key, value, comment) VALUES (?, ?::jsonb, ?)
			 ON CONFLICT (key) DO NOTHING`,
			s.key, s.val, s.comment,
		).Error; err != nil {
			return fmt.Errorf("seed bot_runtime_config %s: %w", s.key, err)
		}
	}
	return nil
}

// List 返回全部行（按 key 升序）。bot 端拉取 / admin 列表都走这个。
func (s *BotRuntimeConfigStore) List(ctx context.Context) ([]BotRuntimeConfigRow, error) {
	var rows []BotRuntimeConfigRow
	if err := pgsql.DB.WithContext(ctx).
		Order("key ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// Upsert 写入或覆盖单个 key。
//
// updatedBy 可空（0 = NULL，表示后台脚本 / 内部维护）；admin API 调用时传 admin user id。
func (s *BotRuntimeConfigStore) Upsert(ctx context.Context,
	key string, value json.RawMessage, updatedBy int64,
) error {
	if key == "" {
		return errors.New("key is empty")
	}
	if len(value) == 0 {
		return errors.New("value is empty")
	}
	// 用原生 SQL 而非 GORM 的 OnConflict——避免 GORM 把 value 的 jsonb 当字符串字面量处理。
	sql := `
		INSERT INTO bot_runtime_config (key, value, updated_at, updated_by)
		VALUES (?, ?::jsonb, CURRENT_TIMESTAMP, NULLIF(?, 0))
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value,
		    updated_at = CURRENT_TIMESTAMP,
		    updated_by = EXCLUDED.updated_by`
	if err := pgsql.DB.WithContext(ctx).Exec(sql, key, string(value), updatedBy).Error; err != nil {
		return fmt.Errorf("upsert bot_runtime_config %s: %w", key, err)
	}
	return nil
}

// Get 单 key 读取——失败时不区分"不存在"与"DB 错误"，返回 (nil, gorm.ErrRecordNotFound)。
func (s *BotRuntimeConfigStore) Get(ctx context.Context, key string) (*BotRuntimeConfigRow, error) {
	var row BotRuntimeConfigRow
	err := pgsql.DB.WithContext(ctx).Where("key = ?", key).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}
