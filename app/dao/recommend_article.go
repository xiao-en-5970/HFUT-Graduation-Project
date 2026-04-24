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

// ArticleRecommendParams 推荐召回参数
// articleType: 0=全部 1帖 2问 3答，>0 时按 type 过滤
// 当 topTagNames 和 topAuthorIDs 均为空时，退化为纯热度+新鲜度召回（用作探索池或冷启动兴趣池）
type ArticleRecommendParams struct {
	ArticleType       int      // 0=全部 1帖 2问 3答
	ViewerSchoolID    uint     // 学校可见性
	TopTagNames       []string // 用户 top 标签名
	TopAuthorIDs      []int    // 用户 top 作者 ID
	ExcludeIDs        []int    // 要排除的文章 ID（已曝光/已浏览）
	Limit             int      // 返回数量上限
	FreshnessDecay    float64  // 新鲜度衰减半衰期（天），0=不衰减
	PopularityCollect int      // 热度权重-收藏
	PopularityLike    int      // 热度权重-点赞
	PopularityView    int      // 热度权重-浏览
	RequireInterest   bool     // 兴趣池开关：true 则必须命中 tag 或 author 其一；false 供探索池使用
}

// ArticleRecommendCandidates 文章推荐候选（帖/问/答）：
//   - RequireInterest=true：仅返回命中 tag 或 author 的文章，综合「命中分+热度×新鲜度」排序；
//   - RequireInterest=false：全库按「热度×新鲜度」排序（探索池 / 冷启动用）。
func (s *ArticleStore) ArticleRecommendCandidates(ctx context.Context, p ArticleRecommendParams) ([]*model.Article, error) {
	if p.Limit < 1 || p.Limit > 500 {
		p.Limit = 60
	}
	if p.PopularityCollect <= 0 {
		p.PopularityCollect = 10
	}
	if p.PopularityLike <= 0 {
		p.PopularityLike = 5
	}
	if p.PopularityView <= 0 {
		p.PopularityView = 1
	}

	q := pgsql.DB.WithContext(ctx).Model(&model.Article{}).
		Where("articles.status = ? AND articles.publish_status = ?", constant.StatusValid, 2)
	if p.ArticleType > 0 {
		q = q.Where("articles.type = ?", p.ArticleType)
	}
	q = applySchoolVisibilityTable(q, p.ViewerSchoolID, "articles.school_id")
	if len(p.ExcludeIDs) > 0 {
		q = q.Where("articles.id NOT IN ?", p.ExcludeIDs)
	}

	hasTags := len(p.TopTagNames) > 0
	hasAuthors := len(p.TopAuthorIDs) > 0

	// 兴趣池必须命中 tag 或 author 其一
	if p.RequireInterest {
		if !hasTags && !hasAuthors {
			return nil, nil
		}
		whereParts := []string{}
		whereArgs := []interface{}{}
		if hasTags {
			whereParts = append(whereParts, "EXISTS (SELECT 1 FROM tags WHERE tags.ext_id = articles.id AND tags.ext_type = 1 AND tags.status = 1 AND tags.name = ANY(?))")
			whereArgs = append(whereArgs, pq.StringArray(p.TopTagNames))
		}
		if hasAuthors {
			whereParts = append(whereParts, "articles.user_id = ANY(?)")
			whereArgs = append(whereArgs, pq.Int64Array(toInt64Slice(p.TopAuthorIDs)))
		}
		q = q.Where("("+strings.Join(whereParts, " OR ")+")", whereArgs...)
	}

	// 构造 score 表达式
	scoreParts := []string{}
	args := []interface{}{}
	popExpr := "(articles.collect_count*? + articles.like_count*? + articles.view_count*?)"
	if p.FreshnessDecay > 0 {
		scoreParts = append(scoreParts, "("+popExpr+" * (1.0 / (1 + EXTRACT(EPOCH FROM (now() - articles.created_at))/86400/?)) * 0.3)")
		args = append(args, p.PopularityCollect, p.PopularityLike, p.PopularityView, p.FreshnessDecay)
	} else {
		scoreParts = append(scoreParts, "("+popExpr+" * 0.3)")
		args = append(args, p.PopularityCollect, p.PopularityLike, p.PopularityView)
	}
	if hasTags {
		scoreParts = append(scoreParts, "((SELECT COUNT(*) FROM tags WHERE tags.ext_id = articles.id AND tags.ext_type = 1 AND tags.status = 1 AND tags.name = ANY(?)) * 3.0)")
		args = append(args, pq.StringArray(p.TopTagNames))
	}
	if hasAuthors {
		scoreParts = append(scoreParts, "((CASE WHEN articles.user_id = ANY(?) THEN 1 ELSE 0 END) * 4.0)")
		args = append(args, pq.Int64Array(toInt64Slice(p.TopAuthorIDs)))
	}

	scoreExpr := strings.Join(scoreParts, " + ")

	var list []*model.Article
	err := q.Order(gorm.Expr(scoreExpr+" DESC, articles.created_at DESC", args...)).
		Limit(p.Limit).Find(&list).Error
	return list, err
}

// applySchoolVisibilityTable 带表名限定的学校可见性过滤，用于子查询/JOIN 场景避免列名歧义
func applySchoolVisibilityTable(q *gorm.DB, viewerSchoolID uint, col string) *gorm.DB {
	if viewerSchoolID == 0 {
		return q.Where(col + " = 0 OR " + col + " IS NULL")
	}
	return q.Where(col+" = 0 OR "+col+" IS NULL OR "+col+" = ?", int(viewerSchoolID))
}

// toInt64Slice 将 []int 转为 []int64，用于 pq.Int64Array
func toInt64Slice(ids []int) []int64 {
	out := make([]int64, len(ids))
	for i, v := range ids {
		out[i] = int64(v)
	}
	return out
}
