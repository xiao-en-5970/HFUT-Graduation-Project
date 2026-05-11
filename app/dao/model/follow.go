package model

import "time"

// Follow 关注关系表——单向边：UserID 关注了 FollowID。
//
// 语义：
//   - UserID = 关注者（"follower"，他主动点了关注）
//   - FollowID = 被关注者（"followee"，他的粉丝里有 UserID）
//
// 一次关注事件 = 一条 row。互关 = 两条对称 row。取消关注由 dao 层硬删除（避免软删扰乱计数）。
//
// 唯一约束：partial unique index `(user_id, follow_id) WHERE user_id IS NOT NULL AND follow_id IS NOT NULL`
// 保证不会重复关注（详见 migrate_profile_social.sql）。同时维护 users.follow_count / users.fans_count
// 的"反范式计数"——所有 follow/unfollow 都通过事务一起更新这两个字段，详见 dao/follow.go。
//
// Status：1=正常，2=禁用（保留语义，目前未使用；如未来需要被举报封禁，可以软禁不删 row）。
type Follow struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    *int      `gorm:"column:user_id;index" json:"user_id"`
	FollowID  *int      `gorm:"column:follow_id;index" json:"follow_id"`
	Status    int16     `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Follow) TableName() string {
	return "follow"
}
