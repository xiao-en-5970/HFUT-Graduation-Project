package model

import "time"

// OrderMessageRead 用户对某订单会话的已读游标（仅统计对方发来的消息为未读）
type OrderMessageRead struct {
	ID                uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID            uint      `gorm:"column:user_id;not null;uniqueIndex:uniq_user_order" json:"user_id"`
	OrderID           uint      `gorm:"column:order_id;not null;uniqueIndex:uniq_user_order" json:"order_id"`
	LastReadMessageID uint      `gorm:"column:last_read_message_id;not null;default:0" json:"last_read_message_id"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (OrderMessageRead) TableName() string {
	return "order_message_reads"
}
