package model

import "time"

// Tag 标签表
type Tag struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`                          // 标签名称
	ExtID     int       `gorm:"column:ext_id;type:integer;not null" json:"ext_id"`               // 关联ID
	ExtType   int       `gorm:"column:ext_type;type:integer;not null;default:1" json:"ext_type"` // 关联类型 1:articles 2:goods
	Status    int16     `gorm:"type:smallint;not null;default:1" json:"status"`                  // 1:正常 2:禁用
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Tag) TableName() string {
	return "tags"
}
