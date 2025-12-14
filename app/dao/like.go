package dao

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
)

type LikeDAO struct{}

// 确保 LikeDAO 实现了 LikeDAOInterface 接口
var _ LikeDAOInterface = (*LikeDAO)(nil)

// NewLikeDAO 创建点赞 DAO
func NewLikeDAO() *LikeDAO {
	return &LikeDAO{}
}

// Create 创建点赞
func (d *LikeDAO) Create(like *model.Like) error {
	return pgsql.DB.Create(like).Error
}

// GetByUserAndExt 根据用户和关联对象获取点赞
func (d *LikeDAO) GetByUserAndExt(userID uint, extType int, extID int) (*model.Like, error) {
	var like model.Like
	err := pgsql.DB.Where("user_id = ? AND ext_type = ? AND ext_id = ? AND status = ?", userID, extType, extID, 1).First(&like).Error
	if err != nil {
		return nil, err
	}
	return &like, nil
}

// Delete 删除点赞（软删除）
func (d *LikeDAO) Delete(userID uint, extType int, extID int) error {
	return pgsql.DB.Model(&model.Like{}).
		Where("user_id = ? AND ext_type = ? AND ext_id = ?", userID, extType, extID).
		Update("status", 2).Error
}

// CountByExt 统计关联对象的点赞数
func (d *LikeDAO) CountByExt(extType int, extID int) (int64, error) {
	var count int64
	err := pgsql.DB.Model(&model.Like{}).
		Where("ext_type = ? AND ext_id = ? AND status = ?", extType, extID, 1).
		Count(&count).Error
	return count, err
}

