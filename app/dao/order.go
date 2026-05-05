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

// FindActiveBuyerOrderForGoods 同一买家、同一商品下尚未终态且订单行未删除的记录，用于「我想要」复用会话
func (s *OrderStore) FindActiveBuyerOrderForGoods(ctx context.Context, buyerID uint, goodsID uint) (*model.Order, error) {
	o := &model.Order{}
	err := pgsql.DB.WithContext(ctx).Model(&model.Order{}).
		Where("user_id = ? AND goods_id = ? AND status = ?", buyerID, goodsID, constant.StatusValid).
		Where("order_status IN ?", []int16{
			constant.OrderStatusAwaitBuyerLocation,
			constant.OrderStatusAwaitSellerPaymentConfirm,
			constant.OrderStatusFulfillment,
			constant.OrderStatusPendingBuyerConfirm,
		}).
		Order("id DESC").
		First(o).Error
	return o, err
}

func (s *OrderStore) ListByUserID(ctx context.Context, userID uint, page, pageSize int) ([]*model.Order, int64, error) {
	return s.ListByUserIDs(ctx, []uint{userID}, page, pageSize)
}

// ListByUserIDs 同 ListByUserID，但 buyer user_id 接受一组——给"账号集"语义下
// "我作为买家的订单"列表用：caller 的主账号 + 旗下号都可能下过单，合并展示。
func (s *OrderStore) ListByUserIDs(ctx context.Context, userIDs []uint, page, pageSize int) ([]*model.Order, int64, error) {
	if len(userIDs) == 0 {
		return nil, 0, nil
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := pgsql.DB.WithContext(ctx).Model(&model.Order{}).
		Where("user_id IN ?", userIDs).
		Where("status = ?", constant.StatusValid)
	var total int64
	q.Count(&total)
	var list []*model.Order
	err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

func (s *OrderStore) ListBySellerID(ctx context.Context, sellerID uint, page, pageSize int) ([]*model.Order, int64, error) {
	return s.ListBySellerIDs(ctx, []uint{sellerID}, page, pageSize)
}

// ListBySellerIDs 同 ListBySellerID，但 seller user_id 接受一组——给"账号集"语义下
// "我作为卖家的订单"列表用：caller 的主账号 + 旗下号都可能挂过商品，合并展示对应订单。
func (s *OrderStore) ListBySellerIDs(ctx context.Context, sellerIDs []uint, page, pageSize int) ([]*model.Order, int64, error) {
	if len(sellerIDs) == 0 {
		return nil, 0, nil
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	// 通过 goods 表关联：订单的 goods 属于 sellerIDs 中任一个
	q := pgsql.DB.WithContext(ctx).Model(&model.Order{}).
		Joins("JOIN goods ON goods.id = orders.goods_id").
		Where("goods.user_id IN ?", sellerIDs).
		Where("orders.status = ?", constant.StatusValid)
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
