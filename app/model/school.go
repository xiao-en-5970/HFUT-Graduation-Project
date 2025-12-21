package model

import (
	"time"
)

// School 学校模型
type School struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(50)" json:"name"`
	LoginURL  string    `gorm:"type:varchar(255)" json:"login_url,omitempty"`
	UserCount int       `gorm:"default:0" json:"user_count"`
	Status    int8      `gorm:"not null;default:1" json:"status"` // 1:正常 2:禁用
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (School) TableName() string {
	return "schools"
}
