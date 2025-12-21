package dao

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
)

type CollectDAO struct{}

// 确保 CollectDAO 实现了 CollectDAOInterface 接口
var _ CollectDAOInterface = (*CollectDAO)(nil)

// NewCollectDAO 创建收藏 DAO
func NewCollectDAO() *CollectDAO {
	return &CollectDAO{}
}

// Create 创建收藏
func (d *CollectDAO) Create(collect *model.Collect) error {
	return pgsql.DB.Create(collect).Error
}

// GetByID 根据 ID 获取收藏
func (d *CollectDAO) GetByID(id uint) (*model.Collect, error) {
	var collect model.Collect
	err := pgsql.DB.Where("status = ?", 1).Preload("User").Preload("User.School").First(&collect, id).Error
	if err != nil {
		return nil, err
	}
	return &collect, nil
}

// GetByUserAndExt 根据用户和关联对象获取收藏
func (d *CollectDAO) GetByUserAndExt(userID uint, extType int, extID int) (*model.Collect, error) {
	var collect model.Collect
	err := pgsql.DB.Where("user_id = ? AND ext_type = ? AND ext_id = ? AND status = ?", userID, extType, extID, 1).First(&collect).Error
	if err != nil {
		return nil, err
	}
	return &collect, nil
}

// Delete 删除收藏（软删除）
func (d *CollectDAO) Delete(id uint) error {
	return pgsql.DB.Model(&model.Collect{}).Where("id = ?", id).Update("status", 2).Error
}

// DeleteByUserAndExt 根据用户和关联对象删除收藏（软删除）
func (d *CollectDAO) DeleteByUserAndExt(userID uint, extType int, extID int) error {
	return pgsql.DB.Model(&model.Collect{}).Where("user_id = ? AND ext_type = ? AND ext_id = ?", userID, extType, extID).Update("status", 2).Error
}

// List 获取收藏列表
func (d *CollectDAO) List(page, pageSize int, userID *uint, extType *int, extID *int) ([]model.Collect, int64, error) {
	var collects []model.Collect
	var total int64

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	query := pgsql.DB.Model(&model.Collect{}).Where("status = ?", 1).Preload("User").Preload("User.School")

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if extType != nil {
		query = query.Where("ext_type = ?", *extType)
	}
	if extID != nil {
		query = query.Where("ext_id = ?", *extID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&collects).Error
	return collects, total, err
}

// CountByExt 统计关联对象的收藏数
func (d *CollectDAO) CountByExt(extType int, extID int) (int64, error) {
	var count int64
	err := pgsql.DB.Model(&model.Collect{}).
		Where("ext_type = ? AND ext_id = ? AND status = ?", extType, extID, 1).
		Count(&count).Error
	return count, err
}
