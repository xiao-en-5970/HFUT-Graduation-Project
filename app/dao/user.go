package dao

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
)

type UserDAO struct{}

// 确保 UserDAO 实现了 UserDAOInterface 接口
var _ UserDAOInterface = (*UserDAO)(nil)

// NewUserDAO 创建用户 DAO
func NewUserDAO() *UserDAO {
	return &UserDAO{}
}

// Create 创建用户
func (d *UserDAO) Create(user *model.User) error {
	return pgsql.DB.Create(user).Error
}

// GetByID 根据 ID 获取用户
func (d *UserDAO) GetByID(id uint) (*model.User, error) {
	var user model.User
	err := pgsql.DB.Preload("School").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (d *UserDAO) GetByUsername(username string) (*model.User, error) {
	var user model.User
	err := pgsql.DB.Preload("School").Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update 更新用户
func (d *UserDAO) Update(user *model.User) error {
	return pgsql.DB.Model(user).Updates(user).Error
}

// Delete 删除用户（软删除）
func (d *UserDAO) Delete(id uint) error {
	return pgsql.DB.Delete(&model.User{}, id).Error
}

// List 获取用户列表
func (d *UserDAO) List(page, pageSize int, schoolID *uint) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	query := pgsql.DB.Model(&model.User{}).Preload("School")

	if schoolID != nil {
		query = query.Where("school_id = ?", *schoolID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&users).Error
	return users, total, err
}

