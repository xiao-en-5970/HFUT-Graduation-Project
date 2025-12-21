package model

import (
	"time"
)

// Collect 收藏模型
type Collect struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ExtID     int       `gorm:"not null;index" json:"ext_id"`
	ExtType   int       `gorm:"not null;default:1;index" json:"ext_type"` // 1:articles 2:goods
	Status    int8      `gorm:"not null;default:1" json:"status"`         // 1:正常 2:禁用
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Collect) TableName() string {
	return "collect"
}

