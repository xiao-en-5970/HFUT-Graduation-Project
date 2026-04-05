package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

type OrderMessageReadStore struct{}

// UpsertLastReadGreatest 将 last_read 更新为 GREATEST(原值, newVal)
func (s *OrderMessageReadStore) UpsertLastReadGreatest(ctx context.Context, userID, orderID, newVal uint) error {
	if newVal == 0 {
		return nil
	}
	return pgsql.DB.WithContext(ctx).Exec(`
INSERT INTO order_message_reads (user_id, order_id, last_read_message_id, updated_at)
VALUES (?, ?, ?, NOW())
ON CONFLICT (user_id, order_id) DO UPDATE SET
  last_read_message_id = GREATEST(order_message_reads.last_read_message_id, EXCLUDED.last_read_message_id),
  updated_at = NOW()
`, userID, orderID, newVal).Error
}

// UnreadCountsByUser 返回 (总未读条数, 按 order_id 的未读条数)
func (s *OrderMessageReadStore) UnreadCountsByUser(ctx context.Context, userID uint) (uint, map[uint]uint, error) {
	type row struct {
		OrderID uint `gorm:"column:order_id"`
		Cnt     int64
	}
	var rows []row
	uid := int(userID)
	mt := int16(constant.OrderMsgTypeOfficial)
	err := pgsql.DB.WithContext(ctx).Raw(`
SELECT om.order_id, COUNT(*)::bigint AS cnt
FROM order_messages om
INNER JOIN (
  SELECT o.id
  FROM orders o
  INNER JOIN goods g ON g.id = o.goods_id
  WHERE o.user_id = ? OR g.user_id = ?
) AS p ON p.id = om.order_id
LEFT JOIN order_message_reads r ON r.order_id = om.order_id AND r.user_id = ?
WHERE om.sender_id <> ?
  AND om.msg_type <> ?
  AND om.id > COALESCE(r.last_read_message_id, 0)
GROUP BY om.order_id
`, uid, uid, uid, uid, mt).Scan(&rows).Error
	if err != nil {
		return 0, nil, err
	}
	out := make(map[uint]uint, len(rows))
	var total uint
	for _, r := range rows {
		if r.Cnt > 0 {
			c := uint(r.Cnt)
			out[r.OrderID] = c
			total += c
		}
	}
	return total, out, nil
}
