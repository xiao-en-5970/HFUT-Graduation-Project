package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"gorm.io/gorm"
)

type CollectItemStore struct{}

func (s *CollectItemStore) Create(ctx context.Context, item *model.CollectItem) (uint, error) {
	err := pgsql.DB.WithContext(ctx).Create(item).Error
	if err != nil {
		return 0, err
	}
	return item.ID, nil
}

// CreateWithDB 事务内创建收藏项
func (s *CollectItemStore) CreateWithDB(db *gorm.DB, item *model.CollectItem) error {
	return db.Create(item).Error
}

// GetByCollectExt 按收藏夹+关联获取记录（含 status=2 的惰性删除记录）
func (s *CollectItemStore) GetByCollectExt(ctx context.Context, collectID uint, extID int, extType int) (*model.CollectItem, error) {
	item := &model.CollectItem{}
	err := pgsql.DB.WithContext(ctx).
		Where("collect_id = ? AND ext_id = ? AND ext_type = ?", collectID, extID, extType).
		First(item).Error
	return item, err
}

// SoftDeleteWithDB 事务内惰性删除（status 设为 2）
func (s *CollectItemStore) SoftDeleteWithDB(db *gorm.DB, collectID uint, extID int, extType int) error {
	return db.Model(&model.CollectItem{}).
		Where("collect_id = ? AND ext_id = ? AND ext_type = ?", collectID, extID, extType).
		UpdateColumn("status", constant.StatusInvalid).Error
}

// RestoreWithDB 事务内恢复（status 设为 1，用于取消收藏后再次收藏）
func (s *CollectItemStore) RestoreWithDB(db *gorm.DB, collectID uint, extID int, extType int) error {
	return db.Model(&model.CollectItem{}).
		Where("collect_id = ? AND ext_id = ? AND ext_type = ?", collectID, extID, extType).
		UpdateColumn("status", constant.StatusValid).Error
}

// Exists 是否已收藏（同收藏夹+ext_id+ext_type）
func (s *CollectItemStore) Exists(ctx context.Context, collectID uint, extID int, extType int) (bool, error) {
	var count int64
	err := pgsql.DB.WithContext(ctx).Model(&model.CollectItem{}).
		Where("collect_id = ? AND ext_id = ? AND ext_type = ? AND status = ?",
			collectID, extID, extType, constant.StatusValid).
		Count(&count).Error
	return count > 0, err
}

// ExistsByUserExt 当前用户在任意有效收藏夹中是否已收藏该资源
func (s *CollectItemStore) ExistsByUserExt(ctx context.Context, userID uint, extID int, extType int) (bool, error) {
	var count int64
	uid := int(userID)
	err := pgsql.DB.WithContext(ctx).Model(&model.CollectItem{}).
		Joins("JOIN collect ON collect.id = collect_item.collect_id AND collect.status = ?", constant.StatusValid).
		Where("collect.user_id = ? AND collect_item.ext_id = ? AND collect_item.ext_type = ? AND collect_item.status = ?",
			uid, extID, extType, constant.StatusValid).
		Count(&count).Error
	return count > 0, err
}

// Delete 取消收藏（惰性删除）
func (s *CollectItemStore) Delete(ctx context.Context, collectID uint, extID int, extType int) error {
	return pgsql.DB.WithContext(ctx).Model(&model.CollectItem{}).
		Where("collect_id = ? AND ext_id = ? AND ext_type = ?", collectID, extID, extType).
		UpdateColumn("status", constant.StatusInvalid).Error
}

// ListByCollect 按收藏夹分页列出收藏项，extType=0 全部，>0 按类型筛选
func (s *CollectItemStore) ListByCollect(ctx context.Context, collectID uint, extType int, page, pageSize int) ([]*model.CollectItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := pgsql.DB.WithContext(ctx).Model(&model.CollectItem{}).
		Where("collect_id = ? AND status = ?", collectID, constant.StatusValid)
	if extType > 0 {
		q = q.Where("ext_type = ?", extType)
	}
	var total int64
	q.Count(&total)
	var list []*model.CollectItem
	err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}
