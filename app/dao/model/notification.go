package model

import "time"

// Notification 站内通知（点赞/评论/回复/官方通知）
//
// Type 取值：
//
//	1 = 点赞了你的帖子/提问/回答/商品
//	2 = 点赞了你的评论
//	3 = 评论了你的帖子/提问/回答/商品（顶层评论）
//	4 = 回复了你的评论
//	5 = 官方通知（FromUserID = 0）
//
// TargetType / RefExtType 使用与 likes/comments 一致的 ext_type：
//
//	1帖子 2提问 3回答 4商品 5评论
type Notification struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int       `gorm:"column:user_id;index;not null" json:"user_id"`
	FromUserID int       `gorm:"column:from_user_id;not null;default:0" json:"from_user_id"`
	Type       int16     `gorm:"type:smallint;not null" json:"type"`
	TargetType int16     `gorm:"column:target_type;type:smallint;not null;default:0" json:"target_type"`
	TargetID   int       `gorm:"column:target_id;not null;default:0" json:"target_id"`
	RefExtType int16     `gorm:"column:ref_ext_type;type:smallint;not null;default:0" json:"ref_ext_type"`
	RefID      int       `gorm:"column:ref_id;not null;default:0" json:"ref_id"`
	Title      string    `gorm:"type:varchar(255);default:''" json:"title"`
	Summary    string    `gorm:"type:varchar(512);default:''" json:"summary"`
	Image      string    `gorm:"type:varchar(512);default:''" json:"image"`
	IsRead     bool      `gorm:"column:is_read;not null;default:false" json:"is_read"`
	Status     int16     `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (Notification) TableName() string {
	return "notifications"
}

// 通知 Type 常量（与 SQL 注释一致）
const (
	NotifyTypeLikeArticle = 1 // 赞了帖子/提问/回答/商品
	NotifyTypeLikeComment = 2 // 赞了评论
	NotifyTypeComment     = 3 // 评论了文章/商品（顶层评论）
	NotifyTypeReply       = 4 // 回复了评论
	NotifyTypeOfficial    = 5 // 官方通知
)

// 官方账号预留 ID
const OfficialUserID = 0
