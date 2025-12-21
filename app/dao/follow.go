package dao

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
)

type FollowDAO struct{}

// 确保 FollowDAO 实现了 FollowDAOInterface 接口
var _ FollowDAOInterface = (*FollowDAO)(nil)

// NewFollowDAO 创建关注 DAO
func NewFollowDAO() *FollowDAO {
	return &FollowDAO{}
}

// Create 创建关注关系
func (d *FollowDAO) Create(follow *model.Follow) error {
	return pgsql.DB.Create(follow).Error
}

// GetByID 根据 ID 获取关注关系
func (d *FollowDAO) GetByID(id uint) (*model.Follow, error) {
	var follow model.Follow
	err := pgsql.DB.Where("status = ?", 1).Preload("User").Preload("User.School").Preload("Followed").Preload("Followed.School").First(&follow, id).Error
	if err != nil {
		return nil, err
	}
	return &follow, nil
}

// GetByUserAndFollow 根据用户和被关注者获取关注关系
func (d *FollowDAO) GetByUserAndFollow(userID uint, followID uint) (*model.Follow, error) {
	var follow model.Follow
	err := pgsql.DB.Where("user_id = ? AND follow_id = ? AND status = ?", userID, followID, 1).
		Preload("User").Preload("User.School").Preload("Followed").Preload("Followed.School").
		First(&follow).Error
	if err != nil {
		return nil, err
	}
	return &follow, nil
}

// Delete 删除关注关系（软删除）
func (d *FollowDAO) Delete(id uint) error {
	return pgsql.DB.Model(&model.Follow{}).Where("id = ?", id).Update("status", 2).Error
}

// DeleteByUserAndFollow 根据用户和被关注者删除关注关系（软删除）
func (d *FollowDAO) DeleteByUserAndFollow(userID uint, followID uint) error {
	return pgsql.DB.Model(&model.Follow{}).Where("user_id = ? AND follow_id = ?", userID, followID).Update("status", 2).Error
}

// ListFollowing 获取关注列表（我关注的人）
func (d *FollowDAO) ListFollowing(page, pageSize int, userID uint) ([]model.Follow, int64, error) {
	var follows []model.Follow
	var total int64

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	query := pgsql.DB.Model(&model.Follow{}).
		Where("user_id = ? AND status = ?", userID, 1).
		Preload("Followed").Preload("Followed.School")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&follows).Error
	return follows, total, err
}

// ListFollowers 获取粉丝列表（关注我的人）
func (d *FollowDAO) ListFollowers(page, pageSize int, userID uint) ([]model.Follow, int64, error) {
	var follows []model.Follow
	var total int64

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	query := pgsql.DB.Model(&model.Follow{}).
		Where("follow_id = ? AND status = ?", userID, 1).
		Preload("User").Preload("User.School")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&follows).Error
	return follows, total, err
}

// CountFollowing 统计关注数量（我关注的人数）
func (d *FollowDAO) CountFollowing(userID uint) (int64, error) {
	var count int64
	err := pgsql.DB.Model(&model.Follow{}).
		Where("user_id = ? AND status = ?", userID, 1).
		Count(&count).Error
	return count, err
}

// CountFollowers 统计粉丝数量（关注我的人数）
func (d *FollowDAO) CountFollowers(userID uint) (int64, error) {
	var count int64
	err := pgsql.DB.Model(&model.Follow{}).
		Where("follow_id = ? AND status = ?", userID, 1).
		Count(&count).Error
	return count, err
}

