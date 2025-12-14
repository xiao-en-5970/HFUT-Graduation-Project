package model

import (
	"time"

	"github.com/lib/pq"
)

// Comment 评论模型
type Comment struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	User      *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ExtType   int            `gorm:"not null;default:1" json:"ext_type"` // 1:articles 2:goods
	ExtID     int            `gorm:"not null" json:"ext_id"`
	ParentID  *uint          `gorm:"index" json:"parent_id,omitempty"`
	Parent    *Comment       `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	ReplyID   *uint          `gorm:"index" json:"reply_id,omitempty"`
	Reply     *Comment       `gorm:"foreignKey:ReplyID" json:"reply,omitempty"`
	Images    pq.StringArray `gorm:"type:varchar(255)[]" json:"images,omitempty"`
	Type      int            `gorm:"not null;default:1" json:"type"` // 1:顶层评论 2:评论回复
	Content   string         `gorm:"type:text;not null" json:"content"`
	Status    int8      `gorm:"not null;default:1" json:"status"` // 1:正常 2:禁用
	LikeCount int       `gorm:"not null;default:0" json:"like_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Comment) TableName() string {
	return "comments"
}
