package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"gorm.io/gorm"
)

type UserLocationStore struct{}

func (s *UserLocationStore) Create(ctx context.Context, o *model.UserLocation) error {
	return pgsql.DB.WithContext(ctx).Create(o).Error
}

func (s *UserLocationStore) CreateTx(ctx context.Context, tx *gorm.DB, o *model.UserLocation) error {
	return tx.WithContext(ctx).Create(o).Error
}

// ClearDefaultTx 将该用户下所有「正常」地址的 is_default 置为 false（事务内）
func (s *UserLocationStore) ClearDefaultTx(ctx context.Context, tx *gorm.DB, userID uint) error {
	return tx.WithContext(ctx).Model(&model.UserLocation{}).
		Where("user_id = ? AND status = ?", userID, constant.StatusValid).
		Update("is_default", false).Error
}

func (s *UserLocationStore) ListByUserID(ctx context.Context, userID uint) ([]*model.UserLocation, error) {
	var list []*model.UserLocation
	err := pgsql.DB.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, constant.StatusValid).
		Order("is_default DESC, updated_at DESC").
		Find(&list).Error
	return list, err
}

// GetByID 按主键查询（任意 status，管理端用）
func (s *UserLocationStore) GetByID(ctx context.Context, id uint) (*model.UserLocation, error) {
	var row model.UserLocation
	err := pgsql.DB.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ListForAdmin 管理端分页；filterUserID=0 表示不限用户；allStatus=false 仅 status=1
func (s *UserLocationStore) ListForAdmin(ctx context.Context, page, pageSize int, filterUserID uint, allStatus bool) ([]*model.UserLocation, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := pgsql.DB.WithContext(ctx).Model(&model.UserLocation{})
	if filterUserID > 0 {
		q = q.Where("user_id = ?", filterUserID)
	}
	if !allStatus {
		q = q.Where("status = ?", constant.StatusValid)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.UserLocation
	err := q.Order("id DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

func (s *UserLocationStore) GetByIDAndUserID(ctx context.Context, id, userID uint) (*model.UserLocation, error) {
	var row model.UserLocation
	err := pgsql.DB.WithContext(ctx).
		Where("id = ? AND user_id = ? AND status = ?", id, userID, constant.StatusValid).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *UserLocationStore) CountActiveByUser(ctx context.Context, userID uint) (int64, error) {
	var n int64
	err := pgsql.DB.WithContext(ctx).Model(&model.UserLocation{}).
		Where("user_id = ? AND status = ?", userID, constant.StatusValid).
		Count(&n).Error
	return n, err
}

// FirstActiveOther 取除 excludeID 外一条正常地址（用于删默认后递补），按 updated_at 降序
func (s *UserLocationStore) FirstActiveOther(ctx context.Context, userID, excludeID uint) (*model.UserLocation, error) {
	var row model.UserLocation
	err := pgsql.DB.WithContext(ctx).
		Where("user_id = ? AND status = ? AND id <> ?", userID, constant.StatusValid, excludeID).
		Order("updated_at DESC").
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *UserLocationStore) UpdateColumns(ctx context.Context, id uint, updates map[string]interface{}) error {
	return pgsql.DB.WithContext(ctx).Model(&model.UserLocation{}).Where("id = ?", id).Updates(updates).Error
}

func (s *UserLocationStore) UpdateColumnsTx(ctx context.Context, tx *gorm.DB, id uint, updates map[string]interface{}) error {
	return tx.WithContext(ctx).Model(&model.UserLocation{}).Where("id = ?", id).Updates(updates).Error
}

func (s *UserLocationStore) SoftDelete(ctx context.Context, id uint) error {
	return pgsql.DB.WithContext(ctx).Model(&model.UserLocation{}).Where("id = ?", id).
		Updates(map[string]interface{}{"status": constant.StatusInvalid, "is_default": false}).Error
}
