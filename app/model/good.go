package model

import (
	"time"

	"github.com/lib/pq"
)

// Good 商品模型
type Good struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	UserID     uint           `gorm:"not null;index" json:"user_id"`
	User       *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Title      string         `gorm:"type:varchar(255);not null" json:"title"`
	Images     pq.StringArray `gorm:"type:varchar(255)[]" json:"images,omitempty"`
	Content    string         `gorm:"type:text;not null" json:"content"`
	Status     int8           `gorm:"not null;default:1" json:"status"`      // 1:正常 2:禁用
	GoodStatus int       `gorm:"not null;default:1" json:"good_status"` // 1:在售 2:下架
	Price      int       `gorm:"not null;default:0" json:"price"`       // 价格，单位分
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Good) TableName() string {
	return "goods"
}
