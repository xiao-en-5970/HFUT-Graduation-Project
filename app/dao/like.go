package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"gorm.io/gorm"
)

type LikeStore struct{}

func (s *LikeStore) Create(ctx context.Context, l *model.Like) (uint, error) {
	err := pgsql.DB.WithContext(ctx).Create(l).Error
	if err != nil {
		return 0, err
	}
	return l.ID, nil
}

// CreateWithDB 事务内创建点赞
func (s *LikeStore) CreateWithDB(db *gorm.DB, l *model.Like) error {
	return db.Create(l).Error
}

// GetByUserExt 按用户+关联获取记录（含 status=2 的惰性删除记录）
func (s *LikeStore) GetByUserExt(ctx context.Context, userID uint, extID int, extType int) (*model.Like, error) {
	l := &model.Like{}
	uid := int(userID)
	err := pgsql.DB.WithContext(ctx).
		Where("user_id = ? AND ext_id = ? AND ext_type = ?", uid, extID, extType).
		First(l).Error
	return l, err
}

// SoftDeleteWithDB 事务内惰性删除（status 设为 2）
func (s *LikeStore) SoftDeleteWithDB(db *gorm.DB, userID uint, extID int, extType int) error {
	uid := int(userID)
	return db.Model(&model.Like{}).
		Where("user_id = ? AND ext_id = ? AND ext_type = ?", uid, extID, extType).
		UpdateColumn("status", constant.StatusInvalid).Error
}

// RestoreWithDB 事务内恢复（status 设为 1，用于取消点赞后再次点赞）
func (s *LikeStore) RestoreWithDB(db *gorm.DB, userID uint, extID int, extType int) error {
	uid := int(userID)
	return db.Model(&model.Like{}).
		Where("user_id = ? AND ext_id = ? AND ext_type = ?", uid, extID, extType).
		UpdateColumn("status", constant.StatusValid).Error
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

// Delete 取消点赞（惰性删除）
func (s *LikeStore) Delete(ctx context.Context, userID uint, extID int, extType int) error {
	uid := int(userID)
	return pgsql.DB.WithContext(ctx).Model(&model.Like{}).
		Where("user_id = ? AND ext_id = ? AND ext_type = ?", uid, extID, extType).
		UpdateColumn("status", constant.StatusInvalid).Error
}
