package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

type CollectItemStore struct{}

func (s *CollectItemStore) Create(ctx context.Context, item *model.CollectItem) (uint, error) {
	err := pgsql.DB.WithContext(ctx).Create(item).Error
	if err != nil {
		return 0, err
	}
	return item.ID, nil
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

// Delete 取消收藏（物理删除）
func (s *CollectItemStore) Delete(ctx context.Context, collectID uint, extID int, extType int) error {
	return pgsql.DB.WithContext(ctx).
		Where("collect_id = ? AND ext_id = ? AND ext_type = ?", collectID, extID, extType).
		Delete(&model.CollectItem{}).Error
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
