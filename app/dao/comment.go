package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"gorm.io/gorm"
)

type CommentStore struct{}

func (s *CommentStore) Create(ctx context.Context, c *model.Comment) (uint, error) {
	err := pgsql.DB.WithContext(ctx).Create(c).Error
	if err != nil {
		return 0, err
	}
	return c.ID, nil
}

// GetByID 按 ID 获取评论
func (s *CommentStore) GetByID(ctx context.Context, id uint) (*model.Comment, error) {
	c := &model.Comment{}
	err := pgsql.DB.WithContext(ctx).Where("id = ? AND status = ?", id, constant.StatusValid).First(c).Error
	return c, err
}

// ListTopByExt 按 ext_type+ext_id 分页列出顶层评论（type=1）
func (s *CommentStore) ListTopByExt(ctx context.Context, extType int, extID int, page, pageSize int) ([]*model.Comment, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	var total int64
	q := pgsql.DB.WithContext(ctx).Model(&model.Comment{}).
		Where("ext_type = ? AND ext_id = ? AND parent_id IS NULL AND status = ? AND type = ?",
			extType, extID, constant.StatusValid, constant.CommentTypeTop)
	q.Count(&total)
	var list []*model.Comment
	err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

// ListRepliesByParentID 按父评论 ID 分页列出回复（type=2）
func (s *CommentStore) ListRepliesByParentID(ctx context.Context, parentID uint, page, pageSize int) ([]*model.Comment, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	var total int64
	q := pgsql.DB.WithContext(ctx).Model(&model.Comment{}).
		Where("parent_id = ? AND status = ? AND type = ?", parentID, constant.StatusValid, constant.CommentTypeReply)
	q.Count(&total)
	var list []*model.Comment
	err := q.Order("created_at ASC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

// GetByIDs 批量按 ID 获取评论
func (s *CommentStore) GetByIDs(ctx context.Context, ids []uint) ([]*model.Comment, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []*model.Comment
	err := pgsql.DB.WithContext(ctx).
		Where("id IN ? AND status = ?", ids, constant.StatusValid).
		Find(&list).Error
	return list, err
}

// CountRepliesByParentIDs 批量统计每个 parentID 的回复数
func (s *CommentStore) CountRepliesByParentIDs(ctx context.Context, parentIDs []uint) (map[uint]int64, error) {
	result := make(map[uint]int64, len(parentIDs))
	if len(parentIDs) == 0 {
		return result, nil
	}
	type row struct {
		ParentID int   `gorm:"column:parent_id"`
		Cnt      int64 `gorm:"column:cnt"`
	}
	var rows []row
	err := pgsql.DB.WithContext(ctx).Model(&model.Comment{}).
		Select("parent_id, COUNT(*) AS cnt").
		Where("parent_id IN ? AND status = ? AND type = ?", parentIDs, constant.StatusValid, constant.CommentTypeReply).
		Group("parent_id").
		Find(&rows).Error
	if err != nil {
		return result, err
	}
	for _, r := range rows {
		result[uint(r.ParentID)] = r.Cnt
	}
	return result, nil
}

// ExistsByExtAndID 校验评论是否存在且属于指定 ext
func (s *CommentStore) ExistsByExtAndID(ctx context.Context, extType int, extID int, commentID uint) (bool, error) {
	var count int64
	err := pgsql.DB.WithContext(ctx).Model(&model.Comment{}).
		Where("id = ? AND ext_type = ? AND ext_id = ? AND status = ?", commentID, extType, extID, constant.StatusValid).
		Count(&count).Error
	return count > 0, err
}

// TopRepliesByParentIDs 为多个 parentID 取各自 like_count 最高的 N 条回复
func (s *CommentStore) TopRepliesByParentIDs(ctx context.Context, parentIDs []uint, limit int) (map[uint][]*model.Comment, error) {
	result := make(map[uint][]*model.Comment, len(parentIDs))
	if len(parentIDs) == 0 {
		return result, nil
	}
	if limit <= 0 {
		limit = 3
	}
	// 使用 LATERAL join 等价：先取全部回复，再程序内截取 top N
	var all []*model.Comment
	err := pgsql.DB.WithContext(ctx).
		Where("parent_id IN ? AND status = ? AND type = ?", parentIDs, constant.StatusValid, constant.CommentTypeReply).
		Order("like_count DESC, created_at ASC").
		Find(&all).Error
	if err != nil {
		return result, err
	}
	for _, c := range all {
		pid := uint(*c.ParentID)
		if len(result[pid]) < limit {
			result[pid] = append(result[pid], c)
		}
	}
	return result, nil
}

// UpdateLikeCountDB 事务内增减 like_count
func (s *CommentStore) UpdateLikeCountDB(db *gorm.DB, id uint, delta int) error {
	return db.Model(&model.Comment{}).Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("GREATEST(0, like_count + ?)", delta)).Error
}

// CountByExt 按 ext 统计评论数量（仅 status=1 的顶层评论）
func (s *CommentStore) CountByExt(ctx context.Context, extType int, extID int) (int64, error) {
	var count int64
	err := pgsql.DB.WithContext(ctx).Model(&model.Comment{}).
		Where("ext_type = ? AND ext_id = ? AND parent_id IS NULL AND status = ?", extType, extID, constant.StatusValid).
		Count(&count).Error
	return count, err
}
