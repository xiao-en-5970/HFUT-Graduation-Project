package model

import (
	"time"
)

// Tag 标签模型
type Tag struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"type:varchar(255);not null" json:"name"`
	ExtID     int            `gorm:"not null;index" json:"ext_id"`
	ExtType   int       `gorm:"not null;default:1;index" json:"ext_type"` // 1:articles 2:goods
	Status    int8      `gorm:"not null;default:1" json:"status"`         // 1:正常 2:禁用
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Tag) TableName() string {
	return "tags"
}
