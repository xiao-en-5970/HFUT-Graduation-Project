package model

import "time"

// UserLocation 用户收货地址（多地址、默认、软删除）
type UserLocation struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"column:user_id;index;not null" json:"user_id"`
	Label     string    `gorm:"column:label;type:varchar(64);not null;default:''" json:"label"`
	Addr      string    `gorm:"column:addr;type:varchar(512);not null" json:"addr"`
	Lat       *float64  `gorm:"column:lat" json:"lat,omitempty"`
	Lng       *float64  `gorm:"column:lng" json:"lng,omitempty"`
	IsDefault bool      `gorm:"column:is_default;not null;default:false" json:"is_default"`
	Status    int16     `gorm:"column:status;type:smallint;not null;default:1;index" json:"status"` // 1 正常 2 已删除
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserLocation) TableName() string {
	return "user_locations"
}
