package dao

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"gorm.io/gorm"
)

type ArticleStore struct{}

func (s *ArticleStore) Create(ctx context.Context, a *model.Article) (uint, error) {
	err := pgsql.DB.Create(a).Error
	if err != nil {
		return 0, err
	}
	return a.ID, nil
}

func (s *ArticleStore) GetByID(ctx context.Context, id uint) (*model.Article, error) {
	a := &model.Article{}
	err := pgsql.DB.Where("id = ? AND status = ?", id, constant.StatusValid).First(a).Error
	return a, err
}

// GetByIDWithSchool 按ID获取，学校隔离（含公开）
func (s *ArticleStore) GetByIDWithSchool(ctx context.Context, id uint, schoolID uint) (*model.Article, error) {
	a := &model.Article{}
	q := pgsql.DB.Where("id = ? AND status = ?", id, constant.StatusValid)
	q = applySchoolVisibility(q, schoolID)
	err := q.First(a).Error
	return a, err
}

// GetByIDWithSchoolAndType 按ID获取，学校+类型隔离（仅 status=1）
func (s *ArticleStore) GetByIDWithSchoolAndType(ctx context.Context, id uint, schoolID uint, articleType int) (*model.Article, error) {
	a := &model.Article{}
	q := pgsql.DB.Where("id = ? AND status = ? AND type = ?", id, constant.StatusValid, articleType)
	q = applySchoolVisibility(q, schoolID)
	err := q.First(a).Error
	return a, err
}

// GetByIDWithSchoolOrPublicAndType 按ID获取，学校或公开（school_id=0 或 = schoolID）
func (s *ArticleStore) GetByIDWithSchoolOrPublicAndType(ctx context.Context, id uint, viewerSchoolID uint, articleType int) (*model.Article, error) {
	a := &model.Article{}
	q := pgsql.DB.Where("id = ? AND status = ? AND type = ?", id, constant.StatusValid, articleType)
	q = applySchoolVisibility(q, viewerSchoolID)
	err := q.First(a).Error
	return a, err
}

// applySchoolVisibility 应用学校可见性：viewerSchoolID=0 仅公开(school_id=0)，>0 公开或本校
func applySchoolVisibility(q *gorm.DB, viewerSchoolID uint) *gorm.DB {
	if viewerSchoolID == 0 {
		return q.Where("school_id = 0")
	}
	return q.Where("school_id = 0 OR school_id = ?", int(viewerSchoolID))
}

// VisibilityMode 学校可见性筛选：public=仅公开(school_id=0)，my_school=仅本校，all=公开+本校
const (
	VisibilityPublic   = "public"
	VisibilityMySchool = "my_school"
	VisibilityAll      = "all"
)

// SortMode 排序模式：relevance=相关度，popularity=热度，combined=相关度+热度加权；latest=发布时间最新；updated_at=最近更新
const (
	SortRelevance  = "relevance"
	SortPopularity = "popularity"
	SortCombined   = "combined"
	SortLatest     = "latest"
	SortUpdatedAt  = "updated_at"
)

// listOrderClause 列表/用户列表：created_at 默认；updated_at 按最近更新时间
func listOrderClause(sort string) string {
	if strings.TrimSpace(sort) == SortUpdatedAt {
		return "updated_at DESC"
	}
	return "created_at DESC"
}

func applyVisibility(q *gorm.DB, mode string, viewerSchoolID uint) *gorm.DB {
	switch mode {
	case VisibilityPublic:
		return q.Where("school_id = 0 OR school_id IS NULL")
	case VisibilityMySchool:
		return q.Where("school_id = ?", int(viewerSchoolID))
	case VisibilityAll:
		fallthrough
	default:
		return applySchoolVisibility(q, viewerSchoolID)
	}
}

// searchConfig 全文检索配置名（create.sql 默认 COPY simple；可选 zhparser_search.sql 升级）
const searchConfig = "chinese_zh"

// escapeLikePattern 转义 LIKE 通配符，避免用户输入 % _ \ 被当作模式
func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// applyArticleKeywordFilter 全文检索 OR 标题/正文子串 ILIKE（与 plainto_tsquery 互补，支持「含关键词即可命中」）
func applyArticleKeywordFilter(q *gorm.DB, keyword string) *gorm.DB {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return q
	}
	pat := "%" + escapeLikePattern(keyword) + "%"
	return q.Where(
		`(search_vector @@ plainto_tsquery(?, ?)) OR (COALESCE(title, '') ILIKE ? ESCAPE '\\') OR (COALESCE(content, '') ILIKE ? ESCAPE '\\')`,
		searchConfig, keyword, pat, pat,
	)
}

// fuzzyRelevanceExpr 混合相关分：ts_rank + 标题/正文子串命中加成（参数顺序：config, keyword, likePat, likePat）
func fuzzyRelevanceExpr() string {
	return `GREATEST(
  COALESCE(ts_rank(search_vector, plainto_tsquery(?, ?)), 0),
  CASE WHEN COALESCE(title, '') ILIKE ? ESCAPE '\\' THEN 0.25 ELSE 0 END,
  CASE WHEN COALESCE(content, '') ILIKE ? ESCAPE '\\' THEN 0.12 ELSE 0 END
)`
}

// GetByIDWithSchoolAndTypeAllowDraft 按ID获取，学校+类型，允许 status=1 或 3
func (s *ArticleStore) GetByIDWithSchoolAndTypeAllowDraft(ctx context.Context, id uint, schoolID uint, articleType int) (*model.Article, error) {
	a := &model.Article{}
	q := pgsql.DB.Where("id = ? AND type = ? AND status IN ?", id, articleType, []int16{constant.StatusValid, constant.StatusDraft})
	q = applySchoolVisibility(q, schoolID)
	err := q.First(a).Error
	return a, err
}

func (s *ArticleStore) GetByIDIncludeDeleted(ctx context.Context, id uint) (*model.Article, error) {
	a := &model.Article{}
	err := pgsql.DB.Where("id = ?", id).First(a).Error
	return a, err
}

// Update 全量保存，ID=0 会触发 INSERT。文章更新请用 UpdateColumns
func (s *ArticleStore) Update(ctx context.Context, a *model.Article) error {
	return pgsql.DB.Save(a).Error
}

func (s *ArticleStore) UpdateColumns(ctx context.Context, id uint, updates map[string]interface{}) error {
	return pgsql.DB.Model(&model.Article{}).Where("id = ?", id).Updates(updates).Error
}

func (s *ArticleStore) SoftDelete(ctx context.Context, id uint) error {
	return pgsql.DB.Model(&model.Article{}).Where("id = ?", id).Update("status", constant.StatusInvalid).Error
}

func (s *ArticleStore) Restore(ctx context.Context, id uint) error {
	return pgsql.DB.Model(&model.Article{}).Where("id = ?", id).Update("status", constant.StatusValid).Error
}

func (s *ArticleStore) UpdateImages(ctx context.Context, id uint, images []string) error {
	return pgsql.DB.Model(&model.Article{}).Where("id = ?", id).
		Updates(map[string]interface{}{"images": pq.StringArray(images), "image_count": len(images)}).Error
}

// UpdateCollectCount 增减 collect_count，delta 为 +1 或 -1
func (s *ArticleStore) UpdateCollectCount(ctx context.Context, id uint, delta int) error {
	return pgsql.DB.WithContext(ctx).Model(&model.Article{}).Where("id = ?", id).
		UpdateColumn("collect_count", gorm.Expr("GREATEST(0, collect_count + ?)", delta)).Error
}

// UpdateLikeCount 增减 like_count，delta 为 +1 或 -1
func (s *ArticleStore) UpdateLikeCount(ctx context.Context, id uint, delta int) error {
	return pgsql.DB.WithContext(ctx).Model(&model.Article{}).Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("GREATEST(0, like_count + ?)", delta)).Error
}

// UpdateCollectCountDB 事务内增减 collect_count
func (s *ArticleStore) UpdateCollectCountDB(db *gorm.DB, id uint, delta int) error {
	return db.Model(&model.Article{}).Where("id = ?", id).
		UpdateColumn("collect_count", gorm.Expr("GREATEST(0, collect_count + ?)", delta)).Error
}

// UpdateLikeCountDB 事务内增减 like_count
func (s *ArticleStore) UpdateLikeCountDB(db *gorm.DB, id uint, delta int) error {
	return db.Model(&model.Article{}).Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("GREATEST(0, like_count + ?)", delta)).Error
}

// ListAdmin 管理员列表，可含已删除，不筛 publish_status
func (s *ArticleStore) ListAdmin(ctx context.Context, schoolID uint, articleType int, includeInvalid bool, page, pageSize int) ([]*model.Article, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := pgsql.DB.WithContext(ctx).Model(&model.Article{}).Where("type = ?", articleType)
	if !includeInvalid {
		q = q.Where("status = ?", constant.StatusValid)
	}
	if schoolID > 0 {
		q = q.Where("school_id = ?", schoolID)
	}
	var total int64
	q.Count(&total)
	var list []*model.Article
	err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

// List 按学校+类型分页列出，类型隔离+学校可见性（viewerSchoolID=0 仅公开，>0 公开或本校）。sort 空或非法：created_at；SortUpdatedAt：updated_at
func (s *ArticleStore) List(ctx context.Context, viewerSchoolID uint, articleType int, page, pageSize int, sort string) ([]*model.Article, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	var total int64
	q := pgsql.DB.Model(&model.Article{}).Where("status = ? AND publish_status = ? AND type = ?", constant.StatusValid, 2, articleType)
	q = applySchoolVisibility(q, viewerSchoolID)
	q.Count(&total)
	var list []*model.Article
	err := q.Order(listOrderClause(sort)).Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

// ListByUserID 按用户 ID 分页列出文章，onlyPublic=true 仅公开(publish_status=2)，false 含私密(1,2)
func (s *ArticleStore) ListByUserID(ctx context.Context, userID uint, articleType int, onlyPublic bool, viewerSchoolID uint, page, pageSize int, sort string) ([]*model.Article, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := pgsql.DB.WithContext(ctx).Model(&model.Article{}).
		Where("user_id = ? AND status = ? AND type = ?", int(userID), constant.StatusValid, articleType)
	if onlyPublic {
		q = q.Where("publish_status = ?", 2)
	} else {
		q = q.Where("publish_status IN ?", []int16{1, 2})
	}
	q = applySchoolVisibility(q, viewerSchoolID)
	var total int64
	q.Count(&total)
	var list []*model.Article
	err := q.Order(listOrderClause(sort)).Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

// Search 全文检索：按类型+学校可见性，相关度+点赞量+收藏量排序；sort=updated_at 时按最近更新
func (s *ArticleStore) Search(ctx context.Context, viewerSchoolID uint, articleType int, keyword string, page, pageSize int, sort string) ([]*model.Article, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return s.List(ctx, viewerSchoolID, articleType, page, pageSize, sort)
	}
	q := pgsql.DB.Model(&model.Article{}).
		Where("status = ? AND publish_status = ? AND type = ?", constant.StatusValid, 2, articleType)
	q = applyArticleKeywordFilter(q, keyword)
	q = applySchoolVisibility(q, viewerSchoolID)
	pat := "%" + escapeLikePattern(keyword) + "%"
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.Article
	var err error
	switch strings.TrimSpace(sort) {
	case SortUpdatedAt:
		err = q.Order("updated_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	case SortLatest:
		err = q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	default:
		err = q.Order(gorm.Expr(fuzzyRelevanceExpr()+" + like_count*0.01 + collect_count*0.01 DESC", searchConfig, keyword, pat, pat)).
			Limit(pageSize).Offset(offset).Find(&list).Error
	}
	return list, total, err
}

// AggregateSearchParams 聚合搜索参数
type AggregateSearchParams struct {
	Keyword       string // 关键词，空则不做全文检索
	Type          int    // 0=全部 1=帖子 2=提问 3=回答
	Visibility    string // public|my_school|all
	ViewerSchool  uint   // 当前用户学校ID
	TimeRange     string // 7d|30d|90d|all
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	Sort          string // relevance|popularity|combined|latest|updated_at
	Page          int
	PageSize      int

	// 排序权重，由 service 从 config 填入
	WeightCollect        int     // 收藏权重
	WeightLike           int     // 点赞权重
	WeightView           int     // 浏览权重
	InteractionDecayDays float64 // 互动分衰减半衰期（天），互动分 *= 1/(1+距今天数/此值)，0=不衰减
	CombinedRelevance    float64 // combined：相关度系数
	CombinedPopularity   float64 // combined：热度系数
}

// AggregateSearch 聚合搜索：帖子+提问+回答，支持筛选与排序
func (s *ArticleStore) AggregateSearch(ctx context.Context, p AggregateSearchParams) ([]*model.Article, int64, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 || p.PageSize > 100 {
		p.PageSize = 20
	}
	offset := (p.Page - 1) * p.PageSize
	keyword := strings.TrimSpace(p.Keyword)

	q := pgsql.DB.WithContext(ctx).Model(&model.Article{}).
		Where("status = ? AND publish_status = ?", constant.StatusValid, 2)
	// 类型筛选
	if p.Type > 0 {
		q = q.Where("type = ?", p.Type)
	}
	q = applyVisibility(q, p.Visibility, p.ViewerSchool)
	// 时间筛选
	if p.CreatedAfter != nil {
		q = q.Where("created_at >= ?", p.CreatedAfter)
	}
	if p.CreatedBefore != nil {
		q = q.Where("created_at <= ?", p.CreatedBefore)
	}
	if p.CreatedAfter == nil && p.CreatedBefore == nil && p.TimeRange != "" && p.TimeRange != "all" {
		now := time.Now()
		switch p.TimeRange {
		case "7d":
			q = q.Where("created_at >= ?", now.AddDate(0, 0, -7))
		case "30d":
			q = q.Where("created_at >= ?", now.AddDate(0, 0, -30))
		case "90d":
			q = q.Where("created_at >= ?", now.AddDate(0, 0, -90))
		}
	}
	// 全文检索 + 标题/正文子串模糊（search_vector 已含 title A + content B）
	q = applyArticleKeywordFilter(q, keyword)

	var total int64
	// 独立 Session 计数，避免 Count 与后续 Order/Limit/Find 复用同一链导致 total 或分页异常
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 权重默认值
	wc, wl, wv := p.WeightCollect, p.WeightLike, p.WeightView
	if wc == 0 {
		wc = 10
	}
	if wl == 0 {
		wl = 5
	}
	if wv == 0 {
		wv = 1
	}
	decayDays := p.InteractionDecayDays
	// 互动分衰减系数：1/(1+距今天数/decayDays)，decayDays<=0 表示不衰减
	interactionDecay := "1"
	if decayDays > 0 {
		interactionDecay = fmt.Sprintf("(1.0 / (1 + EXTRACT(EPOCH FROM (now() - created_at))/86400/%f))", decayDays)
	}
	combRel := p.CombinedRelevance
	if combRel <= 0 {
		combRel = 100
	}
	combPop := p.CombinedPopularity
	if combPop <= 0 {
		combPop = 0.01
	}

	// 热度 = (收藏*Wc+点赞*Wl+浏览*Wv)*互动衰减；互动衰减=1/(1+距今天数/decayDays)
	popExpr := fmt.Sprintf("(collect_count*%d + like_count*%d + view_count*%d) * %s", wc, wl, wv, interactionDecay)

	hasKeyword := keyword != ""
	var likePat string
	if hasKeyword {
		likePat = "%" + escapeLikePattern(keyword) + "%"
	}
	switch p.Sort {
	case SortUpdatedAt:
		q = q.Order("updated_at DESC")
	case SortLatest:
		q = q.Order("created_at DESC")
	case SortRelevance:
		if hasKeyword {
			q = q.Order(gorm.Expr(fuzzyRelevanceExpr()+" DESC", searchConfig, keyword, likePat, likePat))
		} else {
			q = q.Order(gorm.Expr(popExpr + " DESC, created_at DESC"))
		}
	case SortPopularity:
		q = q.Order(gorm.Expr(popExpr + " DESC, created_at DESC"))
	case SortCombined:
		fallthrough
	default:
		if hasKeyword {
			combined := fmt.Sprintf("(%s)*%f + (%s)*%f DESC", fuzzyRelevanceExpr(), combRel, popExpr, combPop)
			q = q.Order(gorm.Expr(combined, searchConfig, keyword, likePat, likePat))
		} else {
			q = q.Order(gorm.Expr(popExpr + " DESC, created_at DESC"))
		}
	}

	var list []*model.Article
	err := q.Limit(p.PageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

func (s *ArticleStore) ExistsAndOwnedBy(ctx context.Context, id uint, userID uint) (bool, error) {
	var count int64
	err := pgsql.DB.Model(&model.Article{}).Where("id = ? AND user_id = ? AND status = ?", id, int(userID), constant.StatusValid).Count(&count).Error
	return count > 0, err
}

// ExistsAndOwnedByWithSchool 校验存在、归属用户且可见（同校或公开）
func (s *ArticleStore) ExistsAndOwnedByWithSchool(ctx context.Context, id uint, userID uint, schoolID uint) (bool, error) {
	var count int64
	q := pgsql.DB.Model(&model.Article{}).Where("id = ? AND user_id = ? AND status = ?", id, int(userID), constant.StatusValid)
	q = applySchoolVisibility(q, schoolID)
	err := q.Count(&count).Error
	return count > 0, err
}

// ExistsAndOwnedByWithSchoolAndType 校验存在、归属用户、可见且类型匹配（仅 status=1 正常）
func (s *ArticleStore) ExistsAndOwnedByWithSchoolAndType(ctx context.Context, id uint, userID uint, schoolID uint, articleType int) (bool, error) {
	var count int64
	q := pgsql.DB.Model(&model.Article{}).Where("id = ? AND user_id = ? AND type = ? AND status = ?", id, int(userID), articleType, constant.StatusValid)
	q = applySchoolVisibility(q, schoolID)
	err := q.Count(&count).Error
	return count > 0, err
}

// IsOwnedByUserForOSS 校验文章归属用户（含草稿），用于 OSS 路径权限校验
func (s *ArticleStore) IsOwnedByUserForOSS(ctx context.Context, articleID uint, userID uint) (bool, error) {
	var count int64
	err := pgsql.DB.WithContext(ctx).Model(&model.Article{}).
		Where("id = ? AND user_id = ? AND status IN ?", articleID, int(userID), []int16{constant.StatusValid, constant.StatusDraft}).
		Count(&count).Error
	return count > 0, err
}

// ExistsAndOwnedByWithSchoolAndTypeAllowDraft 校验存在、归属、可见、类型，允许 status=1 或 3（草稿可编辑）
func (s *ArticleStore) ExistsAndOwnedByWithSchoolAndTypeAllowDraft(ctx context.Context, id uint, userID uint, schoolID uint, articleType int) (bool, error) {
	var count int64
	q := pgsql.DB.Model(&model.Article{}).Where("id = ? AND user_id = ? AND type = ?", id, int(userID), articleType).
		Where("status IN ?", []int16{constant.StatusValid, constant.StatusDraft})
	q = applySchoolVisibility(q, schoolID)
	err := q.Count(&count).Error
	return count > 0, err
}

// ListDrafts 草稿列表，按用户汇总，type=0 全部 1帖子 2提问 3回答（viewerSchoolID 用于过滤本人可编辑的草稿）
func (s *ArticleStore) ListDrafts(ctx context.Context, userID uint, viewerSchoolID uint, articleType int, page, pageSize int) ([]*model.Article, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := pgsql.DB.WithContext(ctx).Model(&model.Article{}).Where("status = ? AND user_id = ?", constant.StatusDraft, int(userID))
	if articleType > 0 {
		q = q.Where("type = ?", articleType)
	}
	q = applySchoolVisibility(q, viewerSchoolID)
	var total int64
	q.Count(&total)
	var list []*model.Article
	err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

// PublishDraft 草稿发布为正式文章，返回是否更新成功
func (s *ArticleStore) PublishDraft(ctx context.Context, id uint, userID uint) (bool, error) {
	result := pgsql.DB.WithContext(ctx).Model(&model.Article{}).
		Where("id = ? AND user_id = ? AND status = ?", id, int(userID), constant.StatusDraft).
		Update("status", constant.StatusValid)
	return result.RowsAffected > 0, result.Error
}

// ListByParentID 按父文章ID分页列出子文章（回答），按父提问的 school_id 过滤
func (s *ArticleStore) ListByParentID(ctx context.Context, parentID uint, questionSchoolID *int, childType int, page, pageSize int) ([]*model.Article, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	var total int64
	q := pgsql.DB.Model(&model.Article{}).
		Where("status = ? AND publish_status = ? AND type = ? AND parent_id = ?", constant.StatusValid, 2, childType, parentID)
	if questionSchoolID == nil || *questionSchoolID == 0 {
		q = q.Where("school_id = 0")
	} else {
		q = q.Where("school_id = ?", *questionSchoolID)
	}
	q.Count(&total)
	var list []*model.Article
	err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}
