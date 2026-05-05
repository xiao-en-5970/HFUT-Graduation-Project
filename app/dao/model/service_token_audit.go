package model

import "time"

// ServiceTokenAudit bot service-to-service 调用审计日志（一次请求一条）。
//
// 写入路径：middleware.BotServiceAuth 在 Next() 后异步写一条；详见
// QQ-bot/skill/bot/SKILL.md "P3.4 限流/审计"段。
type ServiceTokenAudit struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Service    string    `gorm:"column:service;type:varchar(64);not null;index:idx_service_token_audit_service_created,priority:1" json:"service"`
	JTI        string    `gorm:"column:jti;type:varchar(64);not null;index:idx_service_token_audit_jti" json:"jti"`
	Method     string    `gorm:"column:method;type:varchar(8);not null" json:"method"`
	Path       string    `gorm:"column:path;type:varchar(255);not null" json:"path"`
	StatusCode int       `gorm:"column:status_code;not null" json:"status_code"`
	RemoteIP   string    `gorm:"column:remote_ip;type:varchar(64)" json:"remote_ip,omitempty"`
	DurationMS int       `gorm:"column:duration_ms;not null;default:0" json:"duration_ms"`
	CreatedAt  time.Time `gorm:"autoCreateTime;index:idx_service_token_audit_service_created,priority:2" json:"created_at"`
}

func (ServiceTokenAudit) TableName() string {
	return "service_token_audit"
}
