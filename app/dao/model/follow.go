package model

import "time"

// Follow 关注表
type Follow struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    *int      `gorm:"column:user_id;index" json:"user_id"`            // 用户ID
	FollowID  *int      `gorm:"column:follow_id;index" json:"follow_id"`        // 关注用户ID
	Status    int16     `gorm:"type:smallint;not null;default:1" json:"status"` // 1:正常 2:禁用
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Follow) TableName() string {
	return "follow"
}
