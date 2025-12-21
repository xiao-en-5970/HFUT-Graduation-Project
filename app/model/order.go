package model

import (
	"time"
)

// Order 订单模型
type Order struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	User        *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	GoodsID     uint      `gorm:"not null;index" json:"goods_id"`
	Good        *Good     `gorm:"foreignKey:GoodsID" json:"good,omitempty"`
	Status      int8      `gorm:"not null;default:1" json:"status"`      // 1:正常 2:禁用
	OrderStatus int8      `gorm:"not null;default:1" json:"order_status"` // 1:待支付 2:已支付 3:已发货 4:已收货 5:已取消
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Order) TableName() string {
	return "orders"
}

