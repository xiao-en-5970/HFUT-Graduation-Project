package model

import (
	"time"

	"github.com/lib/pq"
)

// 文章 / 提问 / 回答的 Status 取值常量。
//
// Status 是统一的"行级生命周期"字段：1:正常 2:禁用 3:草稿 4:已关闭。
// 4=已关闭仅对**提问**有意义（用户主动关闭提问，停止接受新回答）；其它类型应保持 1/2/3。
const (
	ArticleStatusNormal int16 = 1
	ArticleStatusBanned int16 = 2
	ArticleStatusDraft  int16 = 3
	ArticleStatusClosed int16 = 4
)

// Article 文章表
type Article struct {
	ID            uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        *int           `gorm:"column:user_id;index" json:"user_id"`                                          // 用户ID
	SchoolID      *int           `gorm:"column:school_id;index" json:"school_id"`                                      // 学校ID
	ParentID      *int           `gorm:"column:parent_id;index" json:"parent_id"`                                      // 父文章ID，仅回答指向提问
	Title         string         `gorm:"type:varchar(255);not null" json:"title"`                                      // 文章标题
	Content       string         `gorm:"type:text;not null" json:"content"`                                            // 文章内容
	Status        int16          `gorm:"type:smallint;not null;default:1" json:"status"`                               // 1:正常 2:禁用 3:草稿 4:已关闭(提问主动关闭，停止接受新回答)
	PublishStatus int16          `gorm:"column:publish_status;type:smallint;not null;default:1" json:"publish_status"` // 1:私密 2:公开
	Type          int            `gorm:"type:int;not null;default:1" json:"type"`                                      // 1:普通文章 2:提问 3:回答
	ViewCount     int            `gorm:"column:view_count;not null;default:0" json:"view_count"`                       // 浏览次数
	LikeCount     int            `gorm:"column:like_count;not null;default:0" json:"like_count"`                       // 点赞/同问次数
	CollectCount  int            `gorm:"column:collect_count;not null;default:0" json:"collect_count"`                 // 收藏次数
	Images        pq.StringArray `gorm:"type:varchar(255)[]" json:"images"`                                            // 图片URL数组
	ImageCount    int            `gorm:"column:image_count;not null;default:0" json:"image_count"`                     // 图片数量
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Article) TableName() string {
	return "articles"
}
