package model

import "time"

// BotServiceToken 服务间调用的 token 记录。
//
// 鉴权流程：
//  1. admin 调 POST /admin/bot/service-tokens 创建一条新记录，明文 token 一次性返回管理员
//  2. 服务端只存 sha256 hex (TokenHash 字段)，不存明文
//  3. bot 调 hfut 时 header 带 X-Bot-Service-Token: <明文>
//  4. middleware sha256 后查表，未 revoked + 未 expired 则放行，并刷新 LastUsedAt
type BotServiceToken struct {
	ID          uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string     `gorm:"type:varchar(64);not null" json:"name"`
	Description string     `gorm:"type:varchar(255)" json:"description,omitempty"`
	TokenHash   string     `gorm:"column:token_hash;type:varchar(255);not null;uniqueIndex" json:"-"` // 不暴露给前端
	CreatedBy   *int       `gorm:"column:created_by" json:"created_by,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	ExpiresAt   *time.Time `gorm:"column:expires_at" json:"expires_at,omitempty"`
	RevokedAt   *time.Time `gorm:"column:revoked_at" json:"revoked_at,omitempty"`
	LastUsedAt  *time.Time `gorm:"column:last_used_at" json:"last_used_at,omitempty"`
}

func (BotServiceToken) TableName() string { return "bot_service_tokens" }

// IsActive token 当前是否有效（未作废 + 未过期）。
func (t *BotServiceToken) IsActive(now time.Time) bool {
	if t == nil {
		return false
	}
	if t.RevokedAt != nil {
		return false
	}
	if t.ExpiresAt != nil && !t.ExpiresAt.After(now) {
		return false
	}
	return true
}
