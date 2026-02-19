package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

type LikeStore struct{}

func Like() *LikeStore {
	return &LikeStore{}
}

func (s *LikeStore) Create(ctx context.Context, l *model.Like) (uint, error) {
	err := pgsql.DB.WithContext(ctx).Create(l).Error
	if err != nil {
		return 0, err
	}
	return l.ID, nil
}

// Exists 是否已点赞
func (s *LikeStore) Exists(ctx context.Context, userID uint, extID int, extType int) (bool, error) {
	var count int64
	uid := int(userID)
	err := pgsql.DB.WithContext(ctx).Model(&model.Like{}).
		Where("user_id = ? AND ext_id = ? AND ext_type = ? AND status = ?", uid, extID, extType, constant.StatusValid).
		Count(&count).Error
	return count > 0, err
}

// Delete 取消点赞（物理删除）
func (s *LikeStore) Delete(ctx context.Context, userID uint, extID int, extType int) error {
	uid := int(userID)
	return pgsql.DB.WithContext(ctx).
		Where("user_id = ? AND ext_id = ? AND ext_type = ?", uid, extID, extType).
		Delete(&model.Like{}).Error
}
