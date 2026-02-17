package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

type UserStore struct {
}

func (s *UserStore) Create(ctx context.Context, user *model.User) error {
	return pgsql.DB.Create(user).Error
}

func (s *UserStore) Update(ctx context.Context, user *model.User) error {
	return pgsql.DB.Save(user).Error
}
func (s *UserStore) Delete(ctx context.Context, user *model.User) error {
	return pgsql.DB.Update("status", constant.StatusInvalid).Error
}

func (s *UserStore) GetByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	err := pgsql.DB.Where("id = ?", id).First(&user).Error
	return &user, err
}

func (s *UserStore) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := pgsql.DB.Where("username = ?", username).First(&user).Error
	return &user, err
}

func (s *UserStore) UpdateColumn(ctx context.Context, column string, value interface{}) error {
	return pgsql.DB.Update(column, value).Error
}
