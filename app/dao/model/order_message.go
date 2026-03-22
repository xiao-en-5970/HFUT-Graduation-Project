package model

import "time"

// OrderMessage 订单内买卖双方聊天（与订单绑定，不经过平台资金）
type OrderMessage struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID   uint      `gorm:"column:order_id;index;not null" json:"order_id"`
	SenderID  int       `gorm:"column:sender_id;not null" json:"sender_id"`
	MsgType   int16     `gorm:"column:msg_type;type:smallint;not null;default:1" json:"msg_type"` // 1文字 2图片
	Content   string    `gorm:"type:text" json:"content"`
	ImageURL  string    `gorm:"column:image_url;type:varchar(1024)" json:"image_url,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (OrderMessage) TableName() string {
	return "order_messages"
}
