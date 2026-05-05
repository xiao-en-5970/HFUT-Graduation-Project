package dao

import (
	"context"
	"time"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
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
	q := pgsql.DB.WithContext(ctx).Model(&model.OrderMessage{}).Where("order_id = ? AND msg_type != ?", orderID, constant.OrderMsgTypeOfficial)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.OrderMessage
	err := q.Order("created_at ASC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

// MaxMessageIDByOrder 订单内非官方消息的最大 id（用于全部已读）
func (s *OrderMessageStore) MaxMessageIDByOrder(ctx context.Context, orderID uint) (uint, error) {
	var maxID uint
	err := pgsql.DB.WithContext(ctx).Raw(
		`SELECT COALESCE(MAX(id), 0) FROM order_messages WHERE order_id = ? AND msg_type != ?`,
		orderID, constant.OrderMsgTypeOfficial,
	).Scan(&maxID).Error
	return maxID, err
}

// GetByID 取一条订单消息——加急流程要按 (order_id, msg_id) 校验消息归属。
func (s *OrderMessageStore) GetByID(ctx context.Context, msgID uint) (*model.OrderMessage, error) {
	var m model.OrderMessage
	if err := pgsql.DB.WithContext(ctx).Where("id = ?", msgID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// MarkUrgent 把指定消息置为 urgent=true + urged_at=now()。
//
// 用 conditional UPDATE（WHERE urgent = false）保证幂等：第二次加急同一条消息
// RowsAffected = 0，service 层据此返回"已加急过"错误。
func (s *OrderMessageStore) MarkUrgent(ctx context.Context, msgID uint, now time.Time) (int64, error) {
	res := pgsql.DB.WithContext(ctx).Model(&model.OrderMessage{}).
		Where("id = ? AND urgent = ?", msgID, false).
		Updates(map[string]interface{}{
			"urgent":   true,
			"urged_at": now,
		})
	return res.RowsAffected, res.Error
}
