package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

type UserStore struct {
}

func (s *UserStore) Create(ctx context.Context, user *model.User) (uint, error) {
	err := pgsql.DB.Create(user).Error
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}

func (s *UserStore) Update(ctx context.Context, user *model.User) error {
	return pgsql.DB.Save(user).Error
}
func (s *UserStore) UpdateStatus(ctx context.Context, id uint, status int16) error {
	return pgsql.DB.Model(&model.User{}).Where("id = ?", id).UpdateColumn("status", status).Error
}

func (s *UserStore) UpdateRole(ctx context.Context, id uint, role int16) error {
	return pgsql.DB.Model(&model.User{}).Where("id = ?", id).UpdateColumn("role", role).Error
}

// List 分页列出用户（管理员用），statusFilter: 0全部 1正常 2禁用
func (s *UserStore) List(ctx context.Context, page, pageSize int, statusFilter int16) ([]*model.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := pgsql.DB.WithContext(ctx).Model(&model.User{})
	if statusFilter > 0 {
		q = q.Where("status = ?", statusFilter)
	}
	var total int64
	q.Count(&total)
	var list []*model.User
	err := q.Order("id DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

func (s *UserStore) GetByID(ctx context.Context, id uint) (*model.User, error) {
	user := &model.User{}
	err := pgsql.DB.Where("id = ?", id).First(user).Error
	return user, err
}

func (s *UserStore) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	user := &model.User{}
	err := pgsql.DB.Where("username = ?", username).First(user).Error
	return user, err
}

func (s *UserStore) UpdateSchoolByID(ctx context.Context, userID uint, schoolID uint) error {
	return pgsql.DB.Model(&model.User{}).Where("id = ?", userID).Update("school_id", schoolID).Error
}

func (s *UserStore) UpdateAvatarByID(ctx context.Context, userID uint, avatar string) error {
	return pgsql.DB.Model(&model.User{}).Where("id = ?", userID).Update("avatar", avatar).Error
}

func (s *UserStore) UpdateBackgroundByID(ctx context.Context, userID uint, background string) error {
	return pgsql.DB.Model(&model.User{}).Where("id = ?", userID).Update("background", background).Error
}
