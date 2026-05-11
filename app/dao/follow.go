// Package dao 的 follow.go 实现关注关系的存取——配合 users.follow_count / fans_count 计数。
//
// 设计：
//   - 单向边表 follow(user_id 关注者 → follow_id 被关注者)，partial unique index 防重
//   - Follow / Unfollow 都在事务里同步更新 users.follow_count / fans_count，避免数据漂移
//   - 计数走"反范式存"——用户列表、个人页要展示总量；每次都 COUNT 太慢
//   - 反向回填脚本在 migrate_profile_social.sql 末尾，老数据修一次即可对齐
//
// 一致性保证：
//   - 同一时刻多次"我要关注 X"并发 → 第二次因为 unique index 撞冲突，被吃掉返回 created=false
//   - Unfollow 用 affected_rows 判断是否真的删除了行，0 = 之前就没关注过，不递减计数
package dao

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

type FollowStore struct{}

// IsFollowing userID 是否关注了 targetID。任一为 0 / 二者相等 / status != 1 → false。
func (s *FollowStore) IsFollowing(ctx context.Context, userID, targetID uint) (bool, error) {
	if userID == 0 || targetID == 0 || userID == targetID {
		return false, nil
	}
	var cnt int64
	uid, tid := int(userID), int(targetID)
	err := pgsql.DB.WithContext(ctx).Model(&model.Follow{}).
		Where("user_id = ? AND follow_id = ? AND status = ?", uid, tid, constant.StatusValid).
		Count(&cnt).Error
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// BulkIsFollowing 给定一组 targetIDs，返回 userID 是否关注它们的 bool 映射。
//
// 用途：评论 / 列表里给每位作者打"已关注"标签，单条 SQL 解决，避免 N+1。
// userID == 0 或 targetIDs 为空时返回空 map。
func (s *FollowStore) BulkIsFollowing(ctx context.Context, userID uint, targetIDs []uint) (map[uint]bool, error) {
	out := make(map[uint]bool)
	if userID == 0 || len(targetIDs) == 0 {
		return out, nil
	}
	uid := int(userID)
	tids := make([]int, 0, len(targetIDs))
	for _, t := range targetIDs {
		if t != 0 && t != userID {
			tids = append(tids, int(t))
		}
	}
	if len(tids) == 0 {
		return out, nil
	}
	var rows []model.Follow
	if err := pgsql.DB.WithContext(ctx).
		Select("follow_id").
		Where("user_id = ? AND follow_id IN ? AND status = ?", uid, tids, constant.StatusValid).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.FollowID != nil {
			out[uint(*r.FollowID)] = true
		}
	}
	return out, nil
}

// Follow userID 关注 targetID。
//
// 返回 created：true=本次新建；false=之前已关注（幂等，不报错）。
// 自己关注自己 / 任一为 0 → ErrInvalidFollowParam。
//
// 内部事务：
//  1. INSERT follow（ON CONFLICT DO NOTHING——partial unique index 兜底）
//  2. 仅在真插入了 row 时才递增 users.follow_count(userID) + users.fans_count(targetID)
//
// 性能：3 条 SQL（INSERT + 2 UPDATE），都走主键 / unique 索引。
func (s *FollowStore) Follow(ctx context.Context, userID, targetID uint) (created bool, err error) {
	if userID == 0 || targetID == 0 {
		return false, ErrInvalidFollowParam
	}
	if userID == targetID {
		return false, ErrSelfFollow
	}
	uid, tid := int(userID), int(targetID)
	row := &model.Follow{UserID: &uid, FollowID: &tid, Status: constant.StatusValid}
	err = pgsql.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(row)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			created = false
			return nil
		}
		created = true
		if err := tx.Model(&model.User{}).Where("id = ?", userID).
			UpdateColumn("follow_count", gorm.Expr("follow_count + 1")).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("id = ?", targetID).
			UpdateColumn("fans_count", gorm.Expr("fans_count + 1")).Error; err != nil {
			return err
		}
		return nil
	})
	return created, err
}

// Unfollow userID 取消关注 targetID。
//
// 返回 deleted：true=真的删除了一条 row；false=之前就没关注过（幂等，不报错）。
// 自己 / 任一为 0 → ErrInvalidFollowParam。
//
// 内部事务：硬删除 row，仅在真删了行时才递减计数（用 GREATEST(_, 0) 避免负数）。
func (s *FollowStore) Unfollow(ctx context.Context, userID, targetID uint) (deleted bool, err error) {
	if userID == 0 || targetID == 0 {
		return false, ErrInvalidFollowParam
	}
	if userID == targetID {
		return false, ErrSelfFollow
	}
	uid, tid := int(userID), int(targetID)
	err = pgsql.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("user_id = ? AND follow_id = ?", uid, tid).Delete(&model.Follow{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			deleted = false
			return nil
		}
		deleted = true
		if err := tx.Model(&model.User{}).Where("id = ?", userID).
			UpdateColumn("follow_count", gorm.Expr("GREATEST(follow_count - 1, 0)")).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("id = ?", targetID).
			UpdateColumn("fans_count", gorm.Expr("GREATEST(fans_count - 1, 0)")).Error; err != nil {
			return err
		}
		return nil
	})
	return deleted, err
}

// ListFollowing userID 关注的人——返回他们的 user_id 列表 + 总数（用于分页）。
//
// 按 follow.id 倒序（最新关注的在前）；超出范围返回空列表。
func (s *FollowStore) ListFollowing(ctx context.Context, userID uint, page, pageSize int) (ids []uint, total int64, err error) {
	if userID == 0 {
		return nil, 0, ErrInvalidFollowParam
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	uid := int(userID)
	q := pgsql.DB.WithContext(ctx).Model(&model.Follow{}).
		Where("user_id = ? AND status = ?", uid, constant.StatusValid)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Follow
	if err := q.Order("id DESC").Limit(pageSize).Offset((page - 1) * pageSize).
		Select("follow_id, id").
		Find(&rows).Error; err != nil {
		return nil, total, err
	}
	for _, r := range rows {
		if r.FollowID != nil {
			ids = append(ids, uint(*r.FollowID))
		}
	}
	return ids, total, nil
}

// ListFollowers targetID 的粉丝列表——返回粉丝的 user_id + 总数。
func (s *FollowStore) ListFollowers(ctx context.Context, targetID uint, page, pageSize int) (ids []uint, total int64, err error) {
	if targetID == 0 {
		return nil, 0, ErrInvalidFollowParam
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	tid := int(targetID)
	q := pgsql.DB.WithContext(ctx).Model(&model.Follow{}).
		Where("follow_id = ? AND status = ?", tid, constant.StatusValid)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Follow
	if err := q.Order("id DESC").Limit(pageSize).Offset((page - 1) * pageSize).
		Select("user_id, id").
		Find(&rows).Error; err != nil {
		return nil, total, err
	}
	for _, r := range rows {
		if r.UserID != nil {
			ids = append(ids, uint(*r.UserID))
		}
	}
	return ids, total, nil
}

// ErrInvalidFollowParam userID / targetID 为 0。
var ErrInvalidFollowParam = errors.New("user_id 或 target_id 不能为空")

// ErrSelfFollow 自己关注自己——业务层拒绝。
var ErrSelfFollow = errors.New("不能关注自己")
