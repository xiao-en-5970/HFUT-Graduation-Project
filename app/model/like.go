package model

import (
	"time"

	"github.com/lib/pq"
)

// Like 点赞模型
type Like struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	User      *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ExtID     int            `gorm:"not null;index" json:"ext_id"`
	ExtType   int            `gorm:"not null;default:1;index" json:"ext_type"` // 1:articles 2:comments 3:goods
	Images    pq.StringArray `gorm:"type:varchar(255)[]" json:"images,omitempty"`
	Status    int8           `gorm:"not null;default:1" json:"status"` // 1:正常 2:禁用
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// TableName 指定表名
func (Like) TableName() string {
	return "likes"
}
