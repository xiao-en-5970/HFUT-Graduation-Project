package model

import (
	"time"

	"github.com/lib/pq"
)

// Good 商品表
type Good struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      *int           `gorm:"column:user_id;index" json:"user_id"`                                     // 用户ID
	SchoolID    *int           `gorm:"column:school_id;index" json:"school_id"`                                 // 学校ID
	Title       string         `gorm:"type:varchar(255);not null" json:"title"`                                 // 商品名称
	Images      pq.StringArray `gorm:"type:varchar(255)[]" json:"images"`                                       // 图片数组
	Content     string         `gorm:"type:text;not null" json:"content"`                                       // 商品内容
	Status      int16          `gorm:"type:smallint;not null;default:1" json:"status"`                          // 1:正常 2:禁用
	GoodStatus  int            `gorm:"column:good_status;type:int;not null;default:1" json:"good_status"`       // 1:在售 2:下架 3:已售出
	Price       int            `gorm:"type:integer;not null;default:0" json:"price"`                            // 商品价格，单位分
	MarkedPrice int            `gorm:"column:marked_price;type:integer;not null;default:0" json:"marked_price"` // 标价，单位分
	Stock       int            `gorm:"type:integer;not null;default:0" json:"stock"`                            // 库存数量
	StartTime   int            `gorm:"column:start_time;type:integer;not null;default:0" json:"start_time"`     // 开始时间
	EndTime     int            `gorm:"column:end_time;type:integer;not null;default:0" json:"end_time"`         // 结束时间
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Good) TableName() string {
	return "goods"
}
