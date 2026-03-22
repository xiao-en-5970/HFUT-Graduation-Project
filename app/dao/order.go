package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"gorm.io/gorm"
)

type OrderStore struct{}

func (s *OrderStore) Create(ctx context.Context, o *model.Order) (uint, error) {
	err := pgsql.DB.WithContext(ctx).Create(o).Error
	return o.ID, err
}

func (s *OrderStore) GetByID(ctx context.Context, id uint) (*model.Order, error) {
	o := &model.Order{}
	err := pgsql.DB.WithContext(ctx).Where("id = ?", id).First(o).Error
	return o, err
}

func (s *OrderStore) ListByUserID(ctx context.Context, userID uint, page, pageSize int) ([]*model.Order, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := pgsql.DB.WithContext(ctx).Model(&model.Order{}).Where("user_id = ?", userID)
	var total int64
	q.Count(&total)
	var list []*model.Order
	err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

func (s *OrderStore) ListBySellerID(ctx context.Context, sellerID uint, page, pageSize int) ([]*model.Order, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	// 通过 goods 表关联：订单的 goods 属于 seller
	q := pgsql.DB.WithContext(ctx).Model(&model.Order{}).
		Joins("JOIN goods ON goods.id = orders.goods_id").
		Where("goods.user_id = ?", sellerID)
	var total int64
	q.Count(&total)
	var list []*model.Order
	err := q.Order("orders.created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

func (s *OrderStore) UpdateOrderStatus(ctx context.Context, id uint, orderStatus int16) error {
	return pgsql.DB.WithContext(ctx).Model(&model.Order{}).Where("id = ?", id).Update("order_status", orderStatus).Error
}

func (s *OrderStore) UpdateColumns(ctx context.Context, id uint, updates map[string]interface{}) error {
	return pgsql.DB.WithContext(ctx).Model(&model.Order{}).Where("id = ?", id).Updates(updates).Error
}

func (s *OrderStore) UpdateColumnsTx(ctx context.Context, tx *gorm.DB, id uint, updates map[string]interface{}) error {
	return tx.Model(&model.Order{}).Where("id = ?", id).Updates(updates).Error
}

// ListAllForAdmin 管理端订单列表（可选按商品所属学校筛选）
func (s *OrderStore) ListAllForAdmin(ctx context.Context, page, pageSize int, schoolID uint, includeInvalid bool) ([]*model.Order, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := pgsql.DB.WithContext(ctx).Model(&model.Order{}).Joins("LEFT JOIN goods ON goods.id = orders.goods_id")
	if !includeInvalid {
		q = q.Where("orders.status = ?", constant.StatusValid)
	}
	if schoolID > 0 {
		q = q.Where("goods.school_id = ?", int(schoolID))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.Order
	err := q.Order("orders.created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}
