package model

import "time"

// Order 订单表
type Order struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      *int      `gorm:"column:user_id;index" json:"user_id"`                                      // 用户ID
	GoodsID     *int      `gorm:"column:goods_id;index" json:"goods_id"`                                    // 商品ID
	Status      int16     `gorm:"type:smallint;not null;default:1" json:"status"`                           // 1:正常 2:禁用
	OrderStatus int16     `gorm:"column:order_status;type:smallint;not null;default:1" json:"order_status"` // 1:待支付 2:已支付 3:已发货 4:已收货 5:已取消
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Order) TableName() string {
	return "orders"
}
