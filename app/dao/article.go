package dao

import (
	"context"
	"strings"

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

// GetByIDWithSchool 按ID获取，学校隔离
func (s *ArticleStore) GetByIDWithSchool(ctx context.Context, id uint, schoolID uint) (*model.Article, error) {
	a := &model.Article{}
	q := pgsql.DB.Where("id = ? AND status = ?", id, constant.StatusValid)
	if schoolID > 0 {
		q = q.Where("school_id = ?", schoolID)
	}
	err := q.First(a).Error
	return a, err
}

// GetByIDWithSchoolAndType 按ID获取，学校+类型隔离
func (s *ArticleStore) GetByIDWithSchoolAndType(ctx context.Context, id uint, schoolID uint, articleType int) (*model.Article, error) {
	a := &model.Article{}
	q := pgsql.DB.Where("id = ? AND status = ? AND type = ?", id, constant.StatusValid, articleType)
	if schoolID > 0 {
		q = q.Where("school_id = ?", schoolID)
	}
	err := q.First(a).Error
	return a, err
}

func (s *ArticleStore) GetByIDIncludeDeleted(ctx context.Context, id uint) (*model.Article, error) {
	a := &model.Article{}
	err := pgsql.DB.Where("id = ?", id).First(a).Error
	return a, err
}

func (s *ArticleStore) Update(ctx context.Context, a *model.Article) error {
	return pgsql.DB.Save(a).Error
}

func (s *ArticleStore) UpdateColumns(ctx context.Context, id uint, updates map[string]interface{}) error {
	return pgsql.DB.Model(&model.Article{}).Where("id = ?", id).Updates(updates).Error
}

func (s *ArticleStore) SoftDelete(ctx context.Context, id uint) error {
	return pgsql.DB.Model(&model.Article{}).Where("id = ?", id).Update("status", constant.StatusInvalid).Error
}

func (s *ArticleStore) UpdateImages(ctx context.Context, id uint, images []string) error {
	return pgsql.DB.Model(&model.Article{}).Where("id = ?", id).
		Updates(map[string]interface{}{"images": images, "image_count": len(images)}).Error
}

// List 按学校+类型分页列出，类型隔离+学校隔离
func (s *ArticleStore) List(ctx context.Context, schoolID uint, articleType int, page, pageSize int) ([]*model.Article, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	var total int64
	q := pgsql.DB.Model(&model.Article{}).Where("status = ? AND publish_status = ? AND type = ?", constant.StatusValid, 2, articleType)
	if schoolID > 0 {
		q = q.Where("school_id = ?", schoolID)
	}
	q.Count(&total)
	var list []*model.Article
	err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

// Search 全文检索：按类型+学校隔离，相关度+点赞量+收藏量排序
func (s *ArticleStore) Search(ctx context.Context, schoolID uint, articleType int, keyword string, page, pageSize int) ([]*model.Article, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return s.List(ctx, schoolID, articleType, page, pageSize)
	}
	q := pgsql.DB.Model(&model.Article{}).
		Where("status = ? AND publish_status = ? AND type = ?", constant.StatusValid, 2, articleType).
		Where("search_vector @@ plainto_tsquery('simple', ?)", keyword)
	if schoolID > 0 {
		q = q.Where("school_id = ?", schoolID)
	}
	var total int64
	q.Count(&total)
	var list []*model.Article
	// 排序：ts_rank 相关度 + 点赞量*0.01 + 收藏量*0.01
	err := q.Order(gorm.Expr("ts_rank(search_vector, plainto_tsquery('simple', ?)) + like_count*0.01 + collect_count*0.01 DESC", keyword)).
		Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

func (s *ArticleStore) ExistsAndOwnedBy(ctx context.Context, id uint, userID uint) (bool, error) {
	var count int64
	err := pgsql.DB.Model(&model.Article{}).Where("id = ? AND user_id = ?", id, int(userID)).Count(&count).Error
	return count > 0, err
}

// ExistsAndOwnedByWithSchool 校验存在、归属用户且同校
func (s *ArticleStore) ExistsAndOwnedByWithSchool(ctx context.Context, id uint, userID uint, schoolID uint) (bool, error) {
	var count int64
	q := pgsql.DB.Model(&model.Article{}).Where("id = ? AND user_id = ?", id, int(userID))
	if schoolID > 0 {
		q = q.Where("school_id = ?", schoolID)
	}
	err := q.Count(&count).Error
	return count > 0, err
}

// ExistsAndOwnedByWithSchoolAndType 校验存在、归属用户、同校且类型匹配
func (s *ArticleStore) ExistsAndOwnedByWithSchoolAndType(ctx context.Context, id uint, userID uint, schoolID uint, articleType int) (bool, error) {
	var count int64
	q := pgsql.DB.Model(&model.Article{}).Where("id = ? AND user_id = ? AND type = ?", id, int(userID), articleType)
	if schoolID > 0 {
		q = q.Where("school_id = ?", schoolID)
	}
	err := q.Count(&count).Error
	return count > 0, err
}

// ListByParentID 按父文章ID分页列出子文章（回答），学校隔离
func (s *ArticleStore) ListByParentID(ctx context.Context, parentID uint, schoolID uint, childType int, page, pageSize int) ([]*model.Article, int64, error) {
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
	if schoolID > 0 {
		q = q.Where("school_id = ?", schoolID)
	}
	q.Count(&total)
	var list []*model.Article
	err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}
