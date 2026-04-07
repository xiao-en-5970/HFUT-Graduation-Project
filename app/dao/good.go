package dao

import (
	"context"
	"strings"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"gorm.io/gorm"
)

// GoodStatus 商品状态：1在售 2下架 3已售出
const (
	GoodStatusOnSale   = 1
	GoodStatusOffShelf = 2
	GoodStatusSold     = 3
)

type GoodStore struct{}

func applyGoodSchoolVisibility(q *gorm.DB, viewerSchoolID uint) *gorm.DB {
	if viewerSchoolID == 0 {
		return q.Where("school_id = 0 OR school_id IS NULL")
	}
	return q.Where("school_id = 0 OR school_id IS NULL OR school_id = ?", int(viewerSchoolID))
}

func (s *GoodStore) Create(ctx context.Context, g *model.Good) (uint, error) {
	err := pgsql.DB.WithContext(ctx).Create(g).Error
	return g.ID, err
}

func (s *GoodStore) GetByID(ctx context.Context, id uint) (*model.Good, error) {
	g := &model.Good{}
	err := pgsql.DB.WithContext(ctx).Where("id = ? AND status = ?", id, constant.StatusValid).First(g).Error
	return g, err
}

func (s *GoodStore) GetByIDWithSchool(ctx context.Context, id uint, viewerSchoolID uint) (*model.Good, error) {
	g := &model.Good{}
	q := pgsql.DB.WithContext(ctx).Where("id = ? AND status = ? AND good_status = ?", id, constant.StatusValid, GoodStatusOnSale)
	q = applyGoodSchoolVisibility(q, viewerSchoolID)
	err := q.First(g).Error
	return g, err
}

func (s *GoodStore) GetByIDWithSchoolAllowOffShelf(ctx context.Context, id uint, viewerSchoolID uint) (*model.Good, error) {
	g := &model.Good{}
	q := pgsql.DB.WithContext(ctx).Where("id = ? AND status = ?", id, constant.StatusValid)
	q = applyGoodSchoolVisibility(q, viewerSchoolID)
	err := q.First(g).Error
	return g, err
}

// GoodListSort 与 GET /goods 的 sort 参数：空/newest=上架时间；updated_at=最近更新
const GoodListSortUpdatedAt = "updated_at"

func goodListOrderClause(sort string) string {
	if strings.TrimSpace(sort) == GoodListSortUpdatedAt {
		return "updated_at DESC"
	}
	return "created_at DESC"
}

// List 在售商品分页；keyword 非空时标题模糊匹配（ILIKE）；sort 见 GoodListSortUpdatedAt
func (s *GoodStore) List(ctx context.Context, viewerSchoolID uint, page, pageSize int, keyword string, sort string) ([]*model.Good, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := pgsql.DB.WithContext(ctx).Model(&model.Good{}).
		Where("status = ? AND good_status = ?", constant.StatusValid, GoodStatusOnSale)
	q = applyGoodSchoolVisibility(q, viewerSchoolID)
	kw := strings.TrimSpace(keyword)
	if kw != "" {
		q = q.Where("title ILIKE ?", "%"+kw+"%")
	}
	var total int64
	q.Count(&total)
	var list []*model.Good
	err := q.Order(goodListOrderClause(sort)).Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

func (s *GoodStore) ListByUserID(ctx context.Context, userID uint, viewerSchoolID uint, includeOffShelf bool, page, pageSize int) ([]*model.Good, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := pgsql.DB.WithContext(ctx).Model(&model.Good{}).
		Where("user_id = ? AND status = ?", userID, constant.StatusValid)
	if !includeOffShelf {
		q = q.Where("good_status = ?", GoodStatusOnSale)
	}
	q = applyGoodSchoolVisibility(q, viewerSchoolID)
	var total int64
	q.Count(&total)
	var list []*model.Good
	err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

func (s *GoodStore) UpdateColumns(ctx context.Context, id uint, updates map[string]interface{}) error {
	return pgsql.DB.WithContext(ctx).Model(&model.Good{}).Where("id = ?", id).Updates(updates).Error
}

// DecrementStockAfterSale 成交后库存-1；库存为 0 时标记已售出。须在事务内调用。
func (s *GoodStore) DecrementStockAfterSale(ctx context.Context, tx *gorm.DB, id uint) error {
	res := tx.Model(&model.Good{}).Where("id = ? AND stock >= ?", id, 1).
		UpdateColumn("stock", gorm.Expr("stock - ?", 1))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	var g model.Good
	if err := tx.Where("id = ?", id).First(&g).Error; err != nil {
		return err
	}
	if g.Stock == 0 {
		return tx.Model(&model.Good{}).Where("id = ?", id).Update("good_status", GoodStatusSold).Error
	}
	return nil
}

func (s *GoodStore) IsOwnedByUser(ctx context.Context, id uint, userID uint) (bool, error) {
	var count int64
	err := pgsql.DB.WithContext(ctx).Model(&model.Good{}).
		Where("id = ? AND user_id = ? AND status = ?", id, userID, constant.StatusValid).
		Count(&count).Error
	return count > 0, err
}

func (s *GoodStore) UpdateLikeCountDB(tx *gorm.DB, goodID uint, delta int) error {
	return tx.Model(&model.Good{}).Where("id = ?", goodID).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
}

func (s *GoodStore) UpdateCollectCountDB(tx *gorm.DB, goodID uint, delta int) error {
	return tx.Model(&model.Good{}).Where("id = ?", goodID).
		UpdateColumn("collect_count", gorm.Expr("collect_count + ?", delta)).Error
}

// GetByIDAdmin 管理端：按主键取商品（不限制学校、上下架）
func (s *GoodStore) GetByIDAdmin(ctx context.Context, id uint) (*model.Good, error) {
	g := &model.Good{}
	err := pgsql.DB.WithContext(ctx).Where("id = ?", id).First(g).Error
	return g, err
}

// ListAllForAdmin 管理端：全站商品分页，可选按学校筛选；includeInvalid=false 仅 status=正常
func (s *GoodStore) ListAllForAdmin(ctx context.Context, page, pageSize int, schoolID uint, includeInvalid bool) ([]*model.Good, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := pgsql.DB.WithContext(ctx).Model(&model.Good{})
	if !includeInvalid {
		q = q.Where("status = ?", constant.StatusValid)
	}
	if schoolID > 0 {
		q = q.Where("school_id = ?", int(schoolID))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.Good
	err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}
