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
func (s *UserStore) Delete(ctx context.Context, user *model.User) error {
	return pgsql.DB.Update("status", constant.StatusInvalid).Error
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
