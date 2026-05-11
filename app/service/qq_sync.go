// Package service 的 qq_sync.go：旗下号 nickname / 头像同步辅助。
//
// 不真正调 NapCat 拉数据——那是 bot 的活；这里只提供：
//   - BotListQQChildren：分页吐出 hfut 端所有旗下号，给 bot 同步任务遍历用
//   - QQSyncGetForceAt / QQSyncSetForceNow：用 Redis 做"admin 触发立即同步"信号位
//
// 设计文档：controller/qq_sync.go 头注释。
package service

import (
	"context"
	"strconv"
	"time"

	commonredis "github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/redis"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

// BotQQChildEntry 同步任务需要的字段集；不暴露 password / email / 学校信息等敏感字段。
type BotQQChildEntry struct {
	UserID           uint   `json:"user_id"`
	QQNumber         string `json:"qq_number"`
	Nickname         string `json:"nickname,omitempty"`
	QQAvatarURL      string `json:"qq_avatar_url,omitempty"`
	CreatedInGroupID int64  `json:"created_in_group_id,omitempty"`
}

// BotListQQChildren 按 user_id ASC 分页拉所有 status=valid 的旗下号。
//
// 返回 (entries, next_cursor)：next_cursor == 0 表示已到末尾。bot 同步任务循环
//
//	cursor = 0
//	for {
//	  entries, next = BotListQQChildren(cursor, 200)
//	  for each entry: 拉 NapCat → UpsertQQChild
//	  if next == 0 break
//	  cursor = next
//	}
func BotListQQChildren(ctx context.Context, cursor uint, limit int) ([]BotQQChildEntry, uint, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var rows []model.User
	q := pgsql.DB.WithContext(ctx).
		Where("account_type = ? AND status = ?", model.AccountTypeQQChild, constant.StatusValid)
	if cursor > 0 {
		q = q.Where("id > ?", cursor)
	}
	if err := q.Order("id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]BotQQChildEntry, 0, len(rows))
	for i := range rows {
		u := &rows[i]
		entry := BotQQChildEntry{
			UserID:      u.ID,
			QQAvatarURL: u.QQAvatarURL,
		}
		if u.QQNumber != nil {
			entry.QQNumber = *u.QQNumber
		}
		if u.Nickname != nil {
			entry.Nickname = *u.Nickname
		}
		if u.CreatedInGroupID != nil {
			entry.CreatedInGroupID = *u.CreatedInGroupID
		}
		out = append(out, entry)
	}
	next := uint(0)
	if len(rows) >= limit && len(rows) > 0 {
		next = rows[len(rows)-1].ID
	}
	return out, next, nil
}

// QQSyncForceKey Redis key——存 admin 最后一次按"立即同步"按钮的 unix-ms 时间戳。
const QQSyncForceKey = "bot:qq_sync:force_at"

// QQSyncGetForceAt 读 Redis 里的 force_at；不存在 / Redis 挂了 → 返 0。
func QQSyncGetForceAt(ctx context.Context) (int64, error) {
	if commonredis.Client == nil {
		return 0, nil
	}
	v, err := commonredis.Client.Get(ctx, QQSyncForceKey).Result()
	if err != nil {
		// redis.Nil = 没设过；其它错误也保守返 0 让 bot 不至于盲跑
		return 0, nil
	}
	at, _ := strconv.ParseInt(v, 10, 64)
	return at, nil
}

// QQSyncSetForceNow 把 force_at 设为 now()，返回这个时间戳。
//
// Redis 挂了直接报错——admin 按按钮失败应该让管理员看到，而不是默默吞掉。
func QQSyncSetForceNow(ctx context.Context) (int64, error) {
	at := time.Now().UnixMilli()
	if commonredis.Client == nil {
		return at, nil // Redis 没装也认为成功——下次 bot 周期任务会跑到（兜底）
	}
	// 24h 自动过期——bot 一定会在这之前 poll 到，避免 key 堆积
	_, err := commonredis.Client.Set(ctx, QQSyncForceKey, strconv.FormatInt(at, 10), 24*time.Hour).Result()
	if err != nil {
		return 0, err
	}
	return at, nil
}
