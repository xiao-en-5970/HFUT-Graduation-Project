package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"gorm.io/gorm"
)

type OrderMessageStore struct{}

func (s *OrderMessageStore) Create(ctx context.Context, m *model.OrderMessage) error {
	return pgsql.DB.WithContext(ctx).Create(m).Error
}

func (s *OrderMessageStore) CreateTx(ctx context.Context, tx *gorm.DB, m *model.OrderMessage) error {
	return tx.WithContext(ctx).Create(m).Error
}

func (s *OrderMessageStore) ListByOrderID(ctx context.Context, orderID uint, page, pageSize int) ([]*model.OrderMessage, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize
	q := pgsql.DB.WithContext(ctx).Model(&model.OrderMessage{}).Where("order_id = ?", orderID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.OrderMessage
	err := q.Order("created_at ASC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}
