package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

type SchoolStore struct{}

func (s *SchoolStore) Create(ctx context.Context, school *model.School) (uint, error) {
	err := pgsql.DB.WithContext(ctx).Create(school).Error
	if err != nil {
		return 0, err
	}
	return school.ID, nil
}

func (s *SchoolStore) GetByID(ctx context.Context, id uint) (*model.School, error) {
	school := &model.School{}
	err := pgsql.DB.WithContext(ctx).Where("id = ?", id).First(school).Error
	return school, err
}

// GetByIDValid 仅获取 status=1 的学校
func (s *SchoolStore) GetByIDValid(ctx context.Context, id uint) (*model.School, error) {
	school := &model.School{}
	err := pgsql.DB.WithContext(ctx).Where("id = ? AND status = ?", id, constant.StatusValid).First(school).Error
	return school, err
}

func (s *SchoolStore) List(ctx context.Context, page, pageSize int, includeInvalid bool) ([]*model.School, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := pgsql.DB.WithContext(ctx).Model(&model.School{})
	if !includeInvalid {
		q = q.Where("status = ?", constant.StatusValid)
	}
	var total int64
	q.Count(&total)
	var list []*model.School
	err := q.Order("id ASC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

func (s *SchoolStore) UpdateStatus(ctx context.Context, id uint, status int16) error {
	return pgsql.DB.WithContext(ctx).Model(&model.School{}).Where("id = ?", id).UpdateColumn("status", status).Error
}

// Update 全量保存，ID=0 会触发 INSERT。学校更新请用 UpdateColumns
func (s *SchoolStore) Update(ctx context.Context, school *model.School) error {
	return pgsql.DB.WithContext(ctx).Save(school).Error
}

// UpdateColumns 部分字段更新，避免 Save 误触发 INSERT
func (s *SchoolStore) UpdateColumns(ctx context.Context, id uint, updates map[string]interface{}) error {
	return pgsql.DB.WithContext(ctx).Model(&model.School{}).Where("id = ?", id).Updates(updates).Error
}
