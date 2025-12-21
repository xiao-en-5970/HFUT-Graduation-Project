package dao

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
)

type OrderDAO struct{}

// 确保 OrderDAO 实现了 OrderDAOInterface 接口
var _ OrderDAOInterface = (*OrderDAO)(nil)

// NewOrderDAO 创建订单 DAO
func NewOrderDAO() *OrderDAO {
	return &OrderDAO{}
}

// Create 创建订单
func (d *OrderDAO) Create(order *model.Order) error {
	return pgsql.DB.Create(order).Error
}

// GetByID 根据 ID 获取订单
func (d *OrderDAO) GetByID(id uint) (*model.Order, error) {
	var order model.Order
	err := pgsql.DB.Preload("User").Preload("User.School").Preload("Good").Preload("Good.User").
		Where("status = ?", 1).First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// Update 更新订单
func (d *OrderDAO) Update(order *model.Order) error {
	return pgsql.DB.Model(order).Updates(order).Error
}

// Delete 删除订单（软删除）
func (d *OrderDAO) Delete(id uint) error {
	return pgsql.DB.Model(&model.Order{}).Where("id = ?", id).Update("status", 2).Error
}

// List 获取订单列表
func (d *OrderDAO) List(page, pageSize int, userID *uint, goodsID *uint, orderStatus *int8) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	query := pgsql.DB.Model(&model.Order{}).
		Where("status = ?", 1).
		Preload("User").Preload("User.School").Preload("Good").Preload("Good.User")

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if goodsID != nil {
		query = query.Where("goods_id = ?", *goodsID)
	}
	if orderStatus != nil {
		query = query.Where("order_status = ?", *orderStatus)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&orders).Error
	return orders, total, err
}

