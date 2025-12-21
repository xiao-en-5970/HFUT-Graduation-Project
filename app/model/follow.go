package model

import (
	"time"
)

// Follow 关注模型
type Follow struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`     // 关注者ID
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	FollowID  uint      `gorm:"not null;index" json:"follow_id"`  // 被关注者ID
	Followed  *User     `gorm:"foreignKey:FollowID" json:"followed,omitempty"`
	Status    int8      `gorm:"not null;default:1" json:"status"` // 1:正常 2:禁用
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Follow) TableName() string {
	return "follow"
}

