package model

import "time"

// CollectItem 收藏表，收藏夹中的具体收藏项
type CollectItem struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CollectID uint      `gorm:"column:collect_id;index;not null" json:"collect_id"`
	ExtID     int       `gorm:"column:ext_id;not null" json:"ext_id"`
	ExtType   int       `gorm:"column:ext_type;not null;default:1" json:"ext_type"` // 1:帖子 2:提问 3:回答 4:商品
	Status    int16     `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (CollectItem) TableName() string {
	return "collect_item"
}
