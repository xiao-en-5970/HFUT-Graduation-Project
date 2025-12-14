package model

import (
	"time"
)

// Article 文章模型
type Article struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	UserID        uint           `gorm:"not null;index" json:"user_id"`
	User          *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Title         string         `gorm:"type:varchar(255);not null" json:"title"`
	Content       string         `gorm:"type:text;not null" json:"content"`
	Status        int8           `gorm:"not null;default:1" json:"status"`         // 1:正常 2:禁用
	PublishStatus int8           `gorm:"not null;default:1" json:"publish_status"` // 1:私密 2:公开
	Type          int            `gorm:"not null;default:1" json:"type"`           // 1:普通文章 2:提问 3:回答
	ViewCount    int       `gorm:"not null;default:0" json:"view_count"`
	LikeCount    int       `gorm:"not null;default:0" json:"like_count"`
	CollectCount int       `gorm:"not null;default:0" json:"collect_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Article) TableName() string {
	return "articles"
}
