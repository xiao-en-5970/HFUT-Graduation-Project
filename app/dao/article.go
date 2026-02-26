package dao

import (
	"context"
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

// SortMode 排序模式：relevance=相关度(标题权重高于正文)，popularity=热度(收藏>点赞>浏览)，combined=综合
const (
	SortRelevance  = "relevance"
	SortPopularity = "popularity"
	SortCombined   = "combined"
)

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

// searchConfig 全文检索配置，固定使用中文智能分词
const searchConfig = "chinese_zh"

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

// List 按学校+类型分页列出，类型隔离+学校可见性（viewerSchoolID=0 仅公开，>0 公开或本校）
func (s *ArticleStore) List(ctx context.Context, viewerSchoolID uint, articleType int, page, pageSize int) ([]*model.Article, int64, error) {
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
	err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

// ListByUserID 按用户 ID 分页列出文章，onlyPublic=true 仅公开(publish_status=2)，false 含私密(1,2)
func (s *ArticleStore) ListByUserID(ctx context.Context, userID uint, articleType int, onlyPublic bool, viewerSchoolID uint, page, pageSize int) ([]*model.Article, int64, error) {
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
	err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

// Search 全文检索：按类型+学校可见性，相关度+点赞量+收藏量排序
func (s *ArticleStore) Search(ctx context.Context, viewerSchoolID uint, articleType int, keyword string, page, pageSize int) ([]*model.Article, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return s.List(ctx, viewerSchoolID, articleType, page, pageSize)
	}
	q := pgsql.DB.Model(&model.Article{}).
		Where("status = ? AND publish_status = ? AND type = ?", constant.StatusValid, 2, articleType).
		Where("search_vector @@ plainto_tsquery(?, ?)", searchConfig, keyword)
	q = applySchoolVisibility(q, viewerSchoolID)
	var total int64
	q.Count(&total)
	var list []*model.Article
	// 排序：ts_rank 相关度 + 点赞量*0.01 + 收藏量*0.01
	err := q.Order(gorm.Expr("ts_rank(search_vector, plainto_tsquery(?, ?)) + like_count*0.01 + collect_count*0.01 DESC", searchConfig, keyword)).
		Limit(pageSize).Offset(offset).Find(&list).Error
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
	Sort          string // relevance|popularity|combined
	Page          int
	PageSize      int
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
	// 全文检索（search_vector 已含 title A 权重 + content B 权重）
	if keyword != "" {
		q = q.Where("search_vector @@ plainto_tsquery(?, ?)", searchConfig, keyword)
	}

	var total int64
	q.Count(&total)

	// 排序：relevance=标题优先相关度；popularity=收藏*10+点赞*5+浏览；combined=相关度+热度
	hasKeyword := keyword != ""
	switch p.Sort {
	case SortRelevance:
		if hasKeyword {
			q = q.Order(gorm.Expr("ts_rank(search_vector, plainto_tsquery(?, ?)) DESC", searchConfig, keyword))
		} else {
			q = q.Order(gorm.Expr("(collect_count*10 + like_count*5 + view_count) DESC, created_at DESC"))
		}
	case SortPopularity:
		q = q.Order(gorm.Expr("(collect_count*10 + like_count*5 + view_count) DESC, created_at DESC"))
	case SortCombined:
		fallthrough
	default:
		if hasKeyword {
			q = q.Order(gorm.Expr("ts_rank(search_vector, plainto_tsquery(?, ?))*100 + (collect_count*10 + like_count*5 + view_count)*0.01 DESC", searchConfig, keyword))
		} else {
			q = q.Order(gorm.Expr("(collect_count*10 + like_count*5 + view_count) DESC, created_at DESC"))
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
