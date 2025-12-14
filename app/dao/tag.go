package dao

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
)

type TagDAO struct{}

// 确保 TagDAO 实现了 TagDAOInterface 接口
var _ TagDAOInterface = (*TagDAO)(nil)

// NewTagDAO 创建标签 DAO
func NewTagDAO() *TagDAO {
	return &TagDAO{}
}

// Create 创建标签
func (d *TagDAO) Create(tag *model.Tag) error {
	return pgsql.DB.Create(tag).Error
}

// GetByID 根据 ID 获取标签
func (d *TagDAO) GetByID(id uint) (*model.Tag, error) {
	var tag model.Tag
	err := pgsql.DB.First(&tag, id).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// GetByExt 根据关联对象获取标签列表
func (d *TagDAO) GetByExt(extType int, extID int) ([]model.Tag, error) {
	var tags []model.Tag
	err := pgsql.DB.Where("ext_type = ? AND ext_id = ? AND status = ?", extType, extID, 1).Find(&tags).Error
	return tags, err
}

// Delete 删除标签（软删除）
func (d *TagDAO) Delete(id uint) error {
	return pgsql.DB.Delete(&model.Tag{}, id).Error
}

// DeleteByExt 根据关联对象删除标签
func (d *TagDAO) DeleteByExt(extType int, extID int) error {
	return pgsql.DB.Model(&model.Tag{}).
		Where("ext_type = ? AND ext_id = ?", extType, extID).
		Update("status", 2).Error
}

