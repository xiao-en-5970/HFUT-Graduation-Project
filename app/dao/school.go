package dao

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"gorm.io/gorm"
)

type SchoolDAO struct{}

// 确保 SchoolDAO 实现了 SchoolDAOInterface 接口
var _ SchoolDAOInterface = (*SchoolDAO)(nil)

// NewSchoolDAO 创建学校 DAO
func NewSchoolDAO() *SchoolDAO {
	return &SchoolDAO{}
}

// Create 创建学校
func (d *SchoolDAO) Create(school *model.School) error {
	return pgsql.DB.Create(school).Error
}

// GetByID 根据 ID 获取学校
func (d *SchoolDAO) GetByID(id uint) (*model.School, error) {
	var school model.School
	err := pgsql.DB.Where("status = ?", 1).First(&school, id).Error
	if err != nil {
		return nil, err
	}
	return &school, nil
}

// Update 更新学校
func (d *SchoolDAO) Update(school *model.School) error {
	return pgsql.DB.Model(school).Updates(school).Error
}

// List 获取学校列表
func (d *SchoolDAO) List() ([]model.School, error) {
	var schools []model.School
	err := pgsql.DB.Where("status = ?", 1).Order("created_at DESC").Find(&schools).Error
	return schools, err
}

// IncrementUserCount 增加用户数
func (d *SchoolDAO) IncrementUserCount(id uint) error {
	return pgsql.DB.Model(&model.School{}).Where("id = ?", id).UpdateColumn("user_count", gorm.Expr("user_count + ?", 1)).Error
}

// Delete 删除学校（软删除）
func (d *SchoolDAO) Delete(id uint) error {
	return pgsql.DB.Model(&model.School{}).Where("id = ?", id).Update("status", 2).Error
}

