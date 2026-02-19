package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"gorm.io/gorm"
)

type CollectStore struct{}

func (s *CollectStore) Create(ctx context.Context, c *model.Collect) (uint, error) {
	err := pgsql.DB.WithContext(ctx).Create(c).Error
	if err != nil {
		return 0, err
	}
	return c.ID, nil
}

// GetByID 按 ID 获取收藏夹
func (s *CollectStore) GetByID(ctx context.Context, id uint) (*model.Collect, error) {
	c := &model.Collect{}
	err := pgsql.DB.WithContext(ctx).Where("id = ? AND status = ?", id, constant.StatusValid).First(c).Error
	return c, err
}

// GetDefaultByUserID 获取用户的默认收藏夹，不存在则返回 nil
func (s *CollectStore) GetDefaultByUserID(ctx context.Context, userID uint) (*model.Collect, error) {
	c := &model.Collect{}
	uid := int(userID)
	err := pgsql.DB.WithContext(ctx).
		Where("user_id = ? AND is_default = ? AND status = ?", uid, true, constant.StatusValid).
		First(c).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return c, err
}

// GetByIDAndUser 校验收藏夹存在且属于用户
func (s *CollectStore) GetByIDAndUser(ctx context.Context, id uint, userID uint) (*model.Collect, error) {
	c := &model.Collect{}
	uid := int(userID)
	err := pgsql.DB.WithContext(ctx).
		Where("id = ? AND user_id = ? AND status = ?", id, uid, constant.StatusValid).
		First(c).Error
	return c, err
}

// ListByUserID 列出用户的所有收藏夹
func (s *CollectStore) ListByUserID(ctx context.Context, userID uint) ([]*model.Collect, error) {
	uid := int(userID)
	var list []*model.Collect
	err := pgsql.DB.WithContext(ctx).
		Where("user_id = ? AND status = ?", uid, constant.StatusValid).
		Order("is_default DESC, created_at ASC").
		Find(&list).Error
	return list, err
}
