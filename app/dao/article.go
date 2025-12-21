package dao

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"gorm.io/gorm"
)

type ArticleDAO struct{}

// 确保 ArticleDAO 实现了 ArticleDAOInterface 接口
var _ ArticleDAOInterface = (*ArticleDAO)(nil)

// NewArticleDAO 创建文章 DAO
func NewArticleDAO() *ArticleDAO {
	return &ArticleDAO{}
}

// Create 创建文章
func (d *ArticleDAO) Create(article *model.Article) error {
	return pgsql.DB.Create(article).Error
}

// GetByID 根据 ID 获取文章
func (d *ArticleDAO) GetByID(id uint) (*model.Article, error) {
	var article model.Article
	err := pgsql.DB.Preload("User").Preload("User.School").First(&article, id).Error
	if err != nil {
		return nil, err
	}
	return &article, nil
}

// Update 更新文章
func (d *ArticleDAO) Update(article *model.Article) error {
	return pgsql.DB.Model(article).Updates(article).Error
}

// Delete 删除文章（软删除）
func (d *ArticleDAO) Delete(id uint) error {
	return pgsql.DB.Delete(&model.Article{}, id).Error
}

// List 获取文章列表
func (d *ArticleDAO) List(page, pageSize int, userID *uint, articleType *int, status *int8, keyword string) ([]model.Article, int64, error) {
	var articles []model.Article
	var total int64

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	query := pgsql.DB.Model(&model.Article{}).Preload("User").Preload("User.School")

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if articleType != nil {
		query = query.Where("type = ?", *articleType)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if keyword != "" {
		query = query.Where("title LIKE ? OR content LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&articles).Error
	return articles, total, err
}

// IncrementViewCount 增加浏览次数
func (d *ArticleDAO) IncrementViewCount(id uint) error {
	return pgsql.DB.Model(&model.Article{}).Where("id = ?", id).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

// IncrementLikeCount 增加点赞数
func (d *ArticleDAO) IncrementLikeCount(id uint) error {
	return pgsql.DB.Model(&model.Article{}).Where("id = ?", id).UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
}

// DecrementLikeCount 减少点赞数
func (d *ArticleDAO) DecrementLikeCount(id uint) error {
	return pgsql.DB.Model(&model.Article{}).Where("id = ?", id).UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error
}

// IncrementCollectCount 增加收藏数
func (d *ArticleDAO) IncrementCollectCount(id uint) error {
	return pgsql.DB.Model(&model.Article{}).Where("id = ?", id).UpdateColumn("collect_count", gorm.Expr("collect_count + ?", 1)).Error
}

// DecrementCollectCount 减少收藏数
func (d *ArticleDAO) DecrementCollectCount(id uint) error {
	return pgsql.DB.Model(&model.Article{}).Where("id = ?", id).UpdateColumn("collect_count", gorm.Expr("GREATEST(collect_count - 1, 0)")).Error
}

