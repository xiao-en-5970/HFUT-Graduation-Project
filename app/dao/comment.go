package dao

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"gorm.io/gorm"
)

type CommentDAO struct{}

// 确保 CommentDAO 实现了 CommentDAOInterface 接口
var _ CommentDAOInterface = (*CommentDAO)(nil)

// NewCommentDAO 创建评论 DAO
func NewCommentDAO() *CommentDAO {
	return &CommentDAO{}
}

// Create 创建评论
func (d *CommentDAO) Create(comment *model.Comment) error {
	return pgsql.DB.Create(comment).Error
}

// GetByID 根据 ID 获取评论
func (d *CommentDAO) GetByID(id uint) (*model.Comment, error) {
	var comment model.Comment
	err := pgsql.DB.Preload("User").Preload("Parent").Preload("Reply").First(&comment, id).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// Update 更新评论
func (d *CommentDAO) Update(comment *model.Comment) error {
	return pgsql.DB.Model(comment).Updates(comment).Error
}

// Delete 删除评论（软删除）
func (d *CommentDAO) Delete(id uint) error {
	return pgsql.DB.Delete(&model.Comment{}, id).Error
}

// List 获取评论列表（按 ext_type 和 ext_id）
func (d *CommentDAO) List(page, pageSize int, extType int, extID int) ([]model.Comment, int64, error) {
	var comments []model.Comment
	var total int64

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	query := pgsql.DB.Model(&model.Comment{}).
		Preload("User").
		Where("ext_type = ? AND ext_id = ? AND type = ? AND status = ?", extType, extID, 1, 1) // 只获取顶层评论且状态正常

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&comments).Error
	return comments, total, err
}

// GetReplies 获取评论的回复列表
func (d *CommentDAO) GetReplies(parentID uint) ([]model.Comment, error) {
	var replies []model.Comment
	err := pgsql.DB.Preload("User").Preload("Reply").
		Where("parent_id = ? AND status = ?", parentID, 1).
		Order("created_at ASC").
		Find(&replies).Error
	return replies, err
}

// IncrementLikeCount 增加点赞数
func (d *CommentDAO) IncrementLikeCount(id uint) error {
	return pgsql.DB.Model(&model.Comment{}).Where("id = ?", id).UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
}

// DecrementLikeCount 减少点赞数
func (d *CommentDAO) DecrementLikeCount(id uint) error {
	return pgsql.DB.Model(&model.Comment{}).Where("id = ?", id).UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error
}

