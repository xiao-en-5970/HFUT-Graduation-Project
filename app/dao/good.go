package dao

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
)

type GoodDAO struct{}

// 确保 GoodDAO 实现了 GoodDAOInterface 接口
var _ GoodDAOInterface = (*GoodDAO)(nil)

// NewGoodDAO 创建商品 DAO
func NewGoodDAO() *GoodDAO {
	return &GoodDAO{}
}

// Create 创建商品
func (d *GoodDAO) Create(good *model.Good) error {
	return pgsql.DB.Create(good).Error
}

// GetByID 根据 ID 获取商品
func (d *GoodDAO) GetByID(id uint) (*model.Good, error) {
	var good model.Good
	err := pgsql.DB.Preload("User").Preload("User.School").First(&good, id).Error
	if err != nil {
		return nil, err
	}
	return &good, nil
}

// Update 更新商品
func (d *GoodDAO) Update(good *model.Good) error {
	return pgsql.DB.Model(good).Updates(good).Error
}

// Delete 删除商品（软删除）
func (d *GoodDAO) Delete(id uint) error {
	return pgsql.DB.Delete(&model.Good{}, id).Error
}

// List 获取商品列表
func (d *GoodDAO) List(page, pageSize int, userID *uint, goodStatus *int, status *int8, keyword string, minPrice, maxPrice *int) ([]model.Good, int64, error) {
	var goods []model.Good
	var total int64

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	query := pgsql.DB.Model(&model.Good{}).Preload("User").Preload("User.School")

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if goodStatus != nil {
		query = query.Where("good_status = ?", *goodStatus)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if keyword != "" {
		query = query.Where("title LIKE ? OR content LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if minPrice != nil {
		query = query.Where("price >= ?", *minPrice)
	}
	if maxPrice != nil {
		query = query.Where("price <= ?", *maxPrice)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&goods).Error
	return goods, total, err
}

