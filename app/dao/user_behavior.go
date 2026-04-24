package dao

import (
	"context"
	"time"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
)

type UserBehaviorStore struct{}

// Record 异步落库用户行为：不阻塞主接口；调用方应用 goroutine 包一层（service 层已处理）
func (s *UserBehaviorStore) Record(ctx context.Context, ub *model.UserBehavior) error {
	return pgsql.DB.WithContext(ctx).Create(ub).Error
}

// ListRecent 近 sinceDays 天用户行为，按时间降序（供画像聚合使用，通常 limit <= 500）
func (s *UserBehaviorStore) ListRecent(ctx context.Context, userID uint, sinceDays int, limit int) ([]*model.UserBehavior, error) {
	if sinceDays < 1 {
		sinceDays = 30
	}
	if limit < 1 || limit > 2000 {
		limit = 500
	}
	since := time.Now().AddDate(0, 0, -sinceDays)
	var list []*model.UserBehavior
	err := pgsql.DB.WithContext(ctx).
		Where("user_id = ? AND created_at >= ?", int(userID), since).
		Order("created_at DESC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

// RecentViewedIDs 近 sinceDays 天用户曾 view/comment/like/collect 过的 (ext_type, ext_id) 列表，用于推荐流去重
// 返回按 ext_id 去重后的结果；若 extType=0 则不按类型过滤
func (s *UserBehaviorStore) RecentViewedIDs(ctx context.Context, userID uint, extType int, sinceDays int) ([]int, error) {
	if sinceDays < 1 {
		sinceDays = 7
	}
	since := time.Now().AddDate(0, 0, -sinceDays)
	q := pgsql.DB.WithContext(ctx).Model(&model.UserBehavior{}).
		Where("user_id = ? AND created_at >= ? AND ext_id > 0", int(userID), since)
	if extType > 0 {
		q = q.Where("ext_type = ?", extType)
	}
	// 仅正向信号引发「已看过」去重（unlike/uncollect 不算）
	q = q.Where("action IN ?", []int{1, 2, 4, 6})
	var ids []int
	err := q.Distinct("ext_id").Pluck("ext_id", &ids).Error
	return ids, err
}

// TagAggregate 单个 tag 的聚合得分（按 weight 加权 + 时间衰减后）
type TagAggregate struct {
	Name  string
	Score float64
}

// AuthorAggregate 单个作者的聚合得分
type AuthorAggregate struct {
	AuthorID int
	Score    float64
}

// AggregateTopTags 按 user_behaviors 中的 (ext_type, ext_id) 反查 tags 表，
// 按行为 weight 聚合 tag_name 的总得分，返回 top-N。tagExtType: 1=articles 2=goods（两类 tag 都会聚合）
// sinceDays: 统计近 N 天；decayDays: 时间衰减半衰期
func (s *UserBehaviorStore) AggregateTopTags(ctx context.Context, userID uint, sinceDays int, decayDays float64, limit int) ([]TagAggregate, error) {
	if sinceDays < 1 {
		sinceDays = 30
	}
	if limit < 1 || limit > 200 {
		limit = 20
	}
	since := time.Now().AddDate(0, 0, -sinceDays)

	// 时间衰减：decay_factor = 1/(1+ age_days / decay_days)
	decayExpr := "1"
	if decayDays > 0 {
		decayExpr = "(1.0 / (1 + EXTRACT(EPOCH FROM (now() - ub.created_at))/86400/?))"
	}

	// ext_type 映射：user_behaviors.ext_type ∈ {1,2,3,4} → tags.ext_type ∈ {1(articles),2(goods)}
	// 1/2/3 → 1 (articles)；4 → 2 (goods)
	sqlStr := `
SELECT t.name AS name, SUM(ub.weight * ` + decayExpr + `) AS score
FROM user_behaviors ub
JOIN tags t ON t.ext_id = ub.ext_id
  AND t.ext_type = CASE WHEN ub.ext_type = 4 THEN 2 ELSE 1 END
  AND t.status = 1
WHERE ub.user_id = ? AND ub.created_at >= ? AND ub.ext_id > 0
  AND ub.action IN (1, 2, 4, 6)
GROUP BY t.name
ORDER BY score DESC
LIMIT ?`

	type row struct {
		Name  string
		Score float64
	}
	var rows []row
	var args []interface{}
	if decayDays > 0 {
		args = append(args, decayDays)
	}
	args = append(args, int(userID), since, limit)
	if err := pgsql.DB.WithContext(ctx).Raw(sqlStr, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]TagAggregate, len(rows))
	for i, r := range rows {
		out[i] = TagAggregate{Name: r.Name, Score: r.Score}
	}
	return out, nil
}

// AggregateTopAuthors 按 user_behaviors 反查 articles/goods 作者 ID，聚合 top-N
func (s *UserBehaviorStore) AggregateTopAuthors(ctx context.Context, userID uint, sinceDays int, decayDays float64, limit int) ([]AuthorAggregate, error) {
	if sinceDays < 1 {
		sinceDays = 30
	}
	if limit < 1 || limit > 200 {
		limit = 20
	}
	since := time.Now().AddDate(0, 0, -sinceDays)

	decayExpr := "1"
	if decayDays > 0 {
		decayExpr = "(1.0 / (1 + EXTRACT(EPOCH FROM (now() - ub.created_at))/86400/?))"
	}

	// UNION：articles（ext_type 1/2/3）→ articles.user_id；goods（ext_type=4）→ goods.user_id
	sqlStr := `
WITH hits AS (
  SELECT a.user_id AS author_id, ub.weight * ` + decayExpr + ` AS s
  FROM user_behaviors ub
  JOIN articles a ON a.id = ub.ext_id AND a.status = 1
  WHERE ub.user_id = ? AND ub.created_at >= ? AND ub.ext_id > 0
    AND ub.ext_type IN (1, 2, 3)
    AND ub.action IN (1, 2, 4, 6)
    AND a.user_id IS NOT NULL
  UNION ALL
  SELECT g.user_id AS author_id, ub.weight * ` + decayExpr + ` AS s
  FROM user_behaviors ub
  JOIN goods g ON g.id = ub.ext_id AND g.status = 1
  WHERE ub.user_id = ? AND ub.created_at >= ? AND ub.ext_id > 0
    AND ub.ext_type = 4
    AND ub.action IN (1, 2, 4, 6)
    AND g.user_id IS NOT NULL
)
SELECT author_id, SUM(s) AS score
FROM hits
WHERE author_id IS NOT NULL AND author_id <> ?
GROUP BY author_id
ORDER BY score DESC
LIMIT ?`

	type row struct {
		AuthorID int     `gorm:"column:author_id"`
		Score    float64 `gorm:"column:score"`
	}
	var rows []row
	var args []interface{}
	if decayDays > 0 {
		args = append(args, decayDays)
	}
	args = append(args, int(userID), since)
	if decayDays > 0 {
		args = append(args, decayDays)
	}
	args = append(args, int(userID), since, int(userID), limit)
	if err := pgsql.DB.WithContext(ctx).Raw(sqlStr, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]AuthorAggregate, len(rows))
	for i, r := range rows {
		out[i] = AuthorAggregate{AuthorID: r.AuthorID, Score: r.Score}
	}
	return out, nil
}

// CountRecent 近 sinceDays 天用户产生的行为条数（用于冷启动判定）
func (s *UserBehaviorStore) CountRecent(ctx context.Context, userID uint, sinceDays int) (int64, error) {
	if sinceDays < 1 {
		sinceDays = 30
	}
	since := time.Now().AddDate(0, 0, -sinceDays)
	var cnt int64
	err := pgsql.DB.WithContext(ctx).Model(&model.UserBehavior{}).
		Where("user_id = ? AND created_at >= ?", int(userID), since).
		Count(&cnt).Error
	return cnt, err
}
