package model

import "time"

// Collect 收藏夹表
type Collect struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    *int      `gorm:"column:user_id;index" json:"user_id"`
	Name      string    `gorm:"column:name;type:varchar(100);not null;default:'默认'" json:"name"`
	IsDefault bool      `gorm:"column:is_default;not null;default:false" json:"is_default"`
	Status    int16     `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Collect) TableName() string {
	return "collect"
}
