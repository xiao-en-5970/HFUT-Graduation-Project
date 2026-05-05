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
// category: 0 不过滤；1 二手买卖；2 有偿求助
func (s *GoodStore) List(ctx context.Context, viewerSchoolID uint, page, pageSize int, keyword string, sort string, category int16) ([]*model.Good, int64, error) {
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
	if category == constant.GoodsCategoryNormal || category == constant.GoodsCategoryHelp {
		q = q.Where("goods_category = ?", category)
	}
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

// ownList 为 true 时表示查看自己的商品列表，不按 viewer 学校过滤，避免 JWT 未带学籍或与商品 school_id 不一致时列表为空。
func (s *GoodStore) ListByUserID(ctx context.Context, userID uint, viewerSchoolID uint, includeOffShelf bool, ownList bool, page, pageSize int) ([]*model.Good, int64, error) {
	return s.ListByUserIDs(ctx, []uint{userID}, viewerSchoolID, includeOffShelf, ownList, page, pageSize)
}

// ListByUserIDs 同 ListByUserID，但 user_id 接受一组——用于"账号集"语义下的"我的商品"
// 列表：当 caller 在看自己的列表时把它和旗下号的商品合并起来按时间倒序展示。
//
// 上层使用：
//
//	if targetUserID == callerUserID {
//	    ids, _ := GetAccountIDsForOps(ctx, callerUserID)
//	    list, total := dao.Good().ListByUserIDs(ctx, ids.AllIDs, ...)  // 合并视图
//	} else {
//	    dao.Good().ListByUserID(ctx, targetUserID, ...)                // 看别人不聚合
//	}
//
// userIDs 空时返回空 list、total=0（不报错）。
func (s *GoodStore) ListByUserIDs(ctx context.Context, userIDs []uint, viewerSchoolID uint, includeOffShelf bool, ownList bool, page, pageSize int) ([]*model.Good, int64, error) {
	if len(userIDs) == 0 {
		return nil, 0, nil
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := pgsql.DB.WithContext(ctx).Model(&model.Good{}).
		Where("user_id IN ? AND status = ?", userIDs, constant.StatusValid)
	if !includeOffShelf {
		q = q.Where("good_status = ?", GoodStatusOnSale)
	}
	if !ownList {
		q = applyGoodSchoolVisibility(q, viewerSchoolID)
	}
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

// FindLikelyDuplicates 找该用户最近 7 天内、同 category、title 互相包含的、还在售的商品。
//
// 用途：bot 调 BotPublishGood 上架前先查一次——同一 QQ 用户反复发"出鞋架 6元"
// 这种顶帖行为不该一直重复创建商品记录。命中（即返回非空列表）时上层应该返回 409 +
// 已存在的 good_id 让 bot 提示用户"你之前发过类似的"。
//
// 双向 ILIKE 包含：让"出鞋架 6元" / "三层鞋架" / "宿舍鞋架不锈钢" 都能互相命中——
// 实际复刻"用户反复重发"的 case 比严格 fuzzy similarity 更重要。
//
// 不参与去重判定的字段：价格、地点、图片——重发可能改价改图，但用户语义还是同一个商品。
func (s *GoodStore) FindLikelyDuplicates(ctx context.Context, userID int, category int16, title string) ([]*model.Good, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, nil
	}
	var out []*model.Good
	err := pgsql.DB.WithContext(ctx).
		Where("user_id = ? AND goods_category = ?", userID, category).
		Where("status = ? AND good_status = ?", constant.StatusValid, GoodStatusOnSale).
		Where("created_at > NOW() - INTERVAL '7 days'").
		// 双向 substring：title 是新的、t.title 是老的；任一方包含另一方都视为重复
		Where("title ILIKE ? OR ? ILIKE '%' || title || '%'", "%"+title+"%", title).
		Order("id DESC").
		Limit(5).
		Find(&out).Error
	return out, err
}

func (s *GoodStore) IsOwnedByUser(ctx context.Context, id uint, userID uint) (bool, error) {
	var count int64
	err := pgsql.DB.WithContext(ctx).Model(&model.Good{}).
		Where("id = ? AND user_id = ? AND status = ?", id, userID, constant.StatusValid).
		Count(&count).Error
	return count > 0, err
}

// IsOwnedByOneOf 是不是这组 user_id 中**任一个**拥有该商品。
//
// 用于"账号集"权限模型：主账号 ops 一个商品时，IDs 是 [主账号 id, 旗下账号 id]——
// 商品 owner 是其中任一个就放行，对应 SKILL.md "主账号可以读写旗下账号全部数据"。
//
// IDs 空 / 长度 1 时退化为简单等值查询，不影响行为。
func (s *GoodStore) IsOwnedByOneOf(ctx context.Context, id uint, userIDs []uint) (bool, error) {
	if len(userIDs) == 0 {
		return false, nil
	}
	var count int64
	err := pgsql.DB.WithContext(ctx).Model(&model.Good{}).
		Where("id = ? AND user_id IN ? AND status = ?", id, userIDs, constant.StatusValid).
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

// AutoOffShelfExpired 把所有 has_deadline=true 且 deadline<=now 的在售商品批量下架。
// 返回被下架的条数，供调度器日志。
func (s *GoodStore) AutoOffShelfExpired(ctx context.Context) (int64, error) {
	res := pgsql.DB.WithContext(ctx).Model(&model.Good{}).
		Where("good_status = ? AND has_deadline = ? AND deadline IS NOT NULL AND deadline <= NOW()",
			GoodStatusOnSale, true).
		UpdateColumn("good_status", GoodStatusOffShelf)
	return res.RowsAffected, res.Error
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
