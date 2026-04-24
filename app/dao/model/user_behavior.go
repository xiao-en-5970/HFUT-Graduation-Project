package model

import "time"

// UserBehavior 用户行为流水
type UserBehavior struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int       `gorm:"column:user_id;not null;index" json:"user_id"`
	ExtType   int16     `gorm:"column:ext_type;type:smallint;not null" json:"ext_type"` // 1帖 2问 3答 4商品
	ExtID     int       `gorm:"column:ext_id;not null;default:0" json:"ext_id"`
	Action    int16     `gorm:"column:action;type:smallint;not null" json:"action"` // 1view 2like 3unlike 4collect 5uncollect 6comment 7search
	Weight    float32   `gorm:"column:weight;not null;default:1" json:"weight"`
	Keyword   string    `gorm:"column:keyword;type:varchar(128)" json:"keyword,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (UserBehavior) TableName() string {
	return "user_behaviors"
}
