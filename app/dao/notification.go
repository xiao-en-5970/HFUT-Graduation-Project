package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

type NotificationStore struct{}

// Create 写入一条通知；失败不会上报 panic，由调用方决定是否忽略。
func (s *NotificationStore) Create(ctx context.Context, n *model.Notification) error {
	return pgsql.DB.WithContext(ctx).Create(n).Error
}

// List 查询通知流水
//
//	types: 若为空则不限类型；否则只返回指定 type
//	onlyUnread: 仅未读
func (s *NotificationStore) List(ctx context.Context, userID uint, types []int, onlyUnread bool, page, pageSize int) ([]*model.Notification, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	q := pgsql.DB.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND status = ?", int(userID), constant.StatusValid)
	if len(types) > 0 {
		q = q.Where("type IN ?", types)
	}
	if onlyUnread {
		q = q.Where("is_read = FALSE")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []*model.Notification
	err := q.Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error
	return list, total, err
}

// UnreadCountByType 返回当前用户各 type 的未读数。
// 返回 map[type]count；另单独返回 total 便于前端直接使用。
func (s *NotificationStore) UnreadCountByType(ctx context.Context, userID uint) (map[int]int64, int64, error) {
	type row struct {
		Type  int
		Count int64
	}
	var rows []row
	err := pgsql.DB.WithContext(ctx).Model(&model.Notification{}).
		Select("type, COUNT(1) AS count").
		Where("user_id = ? AND status = ? AND is_read = FALSE", int(userID), constant.StatusValid).
		Group("type").
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make(map[int]int64, len(rows))
	var total int64
	for _, r := range rows {
		out[r.Type] = r.Count
		total += r.Count
	}
	return out, total, nil
}

// MarkReadByIDs 按 ID 批量标记已读（仅本人数据）
func (s *NotificationStore) MarkReadByIDs(ctx context.Context, userID uint, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return pgsql.DB.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND id IN ?", int(userID), ids).
		UpdateColumn("is_read", true).Error
}

// MarkAllRead 全部已读（可按 type 过滤；type=0 表示全部）
func (s *NotificationStore) MarkAllRead(ctx context.Context, userID uint, typ int) error {
	q := pgsql.DB.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND is_read = FALSE AND status = ?", int(userID), constant.StatusValid)
	if typ > 0 {
		q = q.Where("type = ?", typ)
	}
	return q.UpdateColumn("is_read", true).Error
}
