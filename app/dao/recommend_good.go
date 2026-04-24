package dao

import (
	"context"
	"strings"

	"github.com/lib/pq"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"gorm.io/gorm"
)

// GoodRecommendParams 商品推荐召回参数
type GoodRecommendParams struct {
	ViewerSchoolID    uint
	TopTagNames       []string
	TopAuthorIDs      []int
	ExcludeIDs        []int
	Limit             int
	FreshnessDecay    float64
	PopularityCollect int
	PopularityLike    int
	PopularityView    int // goods 表没有 view_count，保留参数占位兼容 config
	RequireInterest   bool
}

// GoodRecommendCandidates 商品推荐候选：在售商品，支持兴趣池/探索池两种模式
func (s *GoodStore) GoodRecommendCandidates(ctx context.Context, p GoodRecommendParams) ([]*model.Good, error) {
	if p.Limit < 1 || p.Limit > 500 {
		p.Limit = 60
	}
	if p.PopularityCollect <= 0 {
		p.PopularityCollect = 10
	}
	if p.PopularityLike <= 0 {
		p.PopularityLike = 5
	}

	q := pgsql.DB.WithContext(ctx).Model(&model.Good{}).
		Where("goods.status = ? AND goods.good_status = ?", constant.StatusValid, GoodStatusOnSale)
	q = applySchoolVisibilityTable(q, p.ViewerSchoolID, "goods.school_id")
	if len(p.ExcludeIDs) > 0 {
		q = q.Where("goods.id NOT IN ?", p.ExcludeIDs)
	}

	hasTags := len(p.TopTagNames) > 0
	hasAuthors := len(p.TopAuthorIDs) > 0

	if p.RequireInterest {
		if !hasTags && !hasAuthors {
			return nil, nil
		}
		whereParts := []string{}
		whereArgs := []interface{}{}
		if hasTags {
			whereParts = append(whereParts, "EXISTS (SELECT 1 FROM tags WHERE tags.ext_id = goods.id AND tags.ext_type = 2 AND tags.status = 1 AND tags.name = ANY(?))")
			whereArgs = append(whereArgs, pq.StringArray(p.TopTagNames))
		}
		if hasAuthors {
			whereParts = append(whereParts, "goods.user_id = ANY(?)")
			whereArgs = append(whereArgs, pq.Int64Array(toInt64Slice(p.TopAuthorIDs)))
		}
		q = q.Where("("+strings.Join(whereParts, " OR ")+")", whereArgs...)
	}

	scoreParts := []string{}
	args := []interface{}{}
	popExpr := "(goods.collect_count*? + goods.like_count*?)"
	if p.FreshnessDecay > 0 {
		scoreParts = append(scoreParts, "("+popExpr+" * (1.0 / (1 + EXTRACT(EPOCH FROM (now() - goods.created_at))/86400/?)) * 0.3)")
		args = append(args, p.PopularityCollect, p.PopularityLike, p.FreshnessDecay)
	} else {
		scoreParts = append(scoreParts, "("+popExpr+" * 0.3)")
		args = append(args, p.PopularityCollect, p.PopularityLike)
	}
	if hasTags {
		scoreParts = append(scoreParts, "((SELECT COUNT(*) FROM tags WHERE tags.ext_id = goods.id AND tags.ext_type = 2 AND tags.status = 1 AND tags.name = ANY(?)) * 3.0)")
		args = append(args, pq.StringArray(p.TopTagNames))
	}
	if hasAuthors {
		scoreParts = append(scoreParts, "((CASE WHEN goods.user_id = ANY(?) THEN 1 ELSE 0 END) * 4.0)")
		args = append(args, pq.Int64Array(toInt64Slice(p.TopAuthorIDs)))
	}

	scoreExpr := strings.Join(scoreParts, " + ")

	var list []*model.Good
	err := q.Order(gorm.Expr(scoreExpr+" DESC, goods.created_at DESC", args...)).
		Limit(p.Limit).Find(&list).Error
	return list, err
}
