package dao

import (
	"context"
	"errors"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"gorm.io/gorm"
)

type AppReleaseStore struct{}

// LatestValid 取该平台 status=valid 的最高 version_code 记录，找不到返 (nil, nil)。
//
// 用于公开接口 GET /api/v1/app/latest-version——前端启动时拉来跟本地版本比对。
func (s *AppReleaseStore) LatestValid(ctx context.Context, platform string) (*model.AppRelease, error) {
	var row model.AppRelease
	err := pgsql.DB.WithContext(ctx).
		Where("platform = ? AND status = ?", platform, constant.StatusValid).
		Order("version_code DESC").
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// GetByID 主键查询（任意 status，admin 用）。
func (s *AppReleaseStore) GetByID(ctx context.Context, id uint) (*model.AppRelease, error) {
	var row model.AppRelease
	err := pgsql.DB.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// Create 落库新版本。
//
// 上层应保证 (platform, version_code) 唯一——重复写会被 uniq 索引 reject，
// 错误以 *pgconn.PgError code=23505 形式上抛；service 层应捕获并转成 4xx。
func (s *AppReleaseStore) Create(ctx context.Context, r *model.AppRelease) error {
	return pgsql.DB.WithContext(ctx).Create(r).Error
}

// ListValid 按 version_code DESC 列出该平台所有 valid 记录；用于 service 自动清理。
func (s *AppReleaseStore) ListValid(ctx context.Context, platform string) ([]*model.AppRelease, error) {
	var list []*model.AppRelease
	err := pgsql.DB.WithContext(ctx).
		Where("platform = ? AND status = ?", platform, constant.StatusValid).
		Order("version_code DESC").
		Find(&list).Error
	return list, err
}

// ListAll admin 端分页：列出所有 status 的记录。
func (s *AppReleaseStore) ListAll(ctx context.Context, platform string, page, pageSize int) ([]*model.AppRelease, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	q := pgsql.DB.WithContext(ctx).Model(&model.AppRelease{})
	if platform != "" {
		q = q.Where("platform = ?", platform)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.AppRelease
	err := q.Order("version_code DESC, id DESC").Limit(pageSize).Offset(offset).Find(&list).Error
	return list, total, err
}

// HardDelete 物理删除——配合 OSS 文件删除使用（service 那一层先删 OSS 再删 DB，
// DB 删不掉时再补 OSS：避免"DB 没有但 OSS 还有"的孤儿文件）。
func (s *AppReleaseStore) HardDelete(ctx context.Context, id uint) error {
	return pgsql.DB.WithContext(ctx).Where("id = ?", id).Delete(&model.AppRelease{}).Error
}
