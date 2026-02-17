package model

import (
	"time"

	"github.com/lib/pq"
)

// Comment 评论表
type Comment struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    *int           `gorm:"column:user_id;index" json:"user_id"`                             // 用户ID
	ExtType   int            `gorm:"column:ext_type;type:integer;not null;default:1" json:"ext_type"` // 关联类型 1:articles 2:goods
	ExtID     int            `gorm:"column:ext_id;type:integer;not null" json:"ext_id"`               // 关联ID
	ParentID  *int           `gorm:"column:parent_id;index" json:"parent_id"`                         // 父评论ID
	ReplyID   *int           `gorm:"column:reply_id;index" json:"reply_id"`                           // 回复评论ID
	Images    pq.StringArray `gorm:"type:varchar(255)[]" json:"images"`                               // 图片数组
	Type      int            `gorm:"type:integer;not null;default:1" json:"type"`                     // 1:顶层评论 2:评论回复
	Content   string         `gorm:"type:text;not null" json:"content"`                               // 评论内容
	Status    int16          `gorm:"type:smallint;not null;default:1" json:"status"`                  // 1:正常 2:禁用
	LikeCount int            `gorm:"column:like_count;not null;default:0" json:"like_count"`          // 点赞次数
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Comment) TableName() string {
	return "comments"
}
