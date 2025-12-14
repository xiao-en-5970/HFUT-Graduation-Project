package dao

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"
)

// UserDAOInterface 用户 DAO 接口
type UserDAOInterface interface {
	Create(user *model.User) error
	GetByID(id uint) (*model.User, error)
	GetByUsername(username string) (*model.User, error)
	Update(user *model.User) error
	Delete(id uint) error
	List(page, pageSize int, schoolID *uint) ([]model.User, int64, error)
}

// SchoolDAOInterface 学校 DAO 接口
type SchoolDAOInterface interface {
	Create(school *model.School) error
	GetByID(id uint) (*model.School, error)
	Update(school *model.School) error
	List() ([]model.School, error)
	IncrementUserCount(id uint) error
}

// ArticleDAOInterface 文章 DAO 接口
type ArticleDAOInterface interface {
	Create(article *model.Article) error
	GetByID(id uint) (*model.Article, error)
	Update(article *model.Article) error
	Delete(id uint) error
	List(page, pageSize int, userID *uint, articleType *int, status *int8, keyword string) ([]model.Article, int64, error)
	IncrementViewCount(id uint) error
	IncrementLikeCount(id uint) error
	DecrementLikeCount(id uint) error
}

// CommentDAOInterface 评论 DAO 接口
type CommentDAOInterface interface {
	Create(comment *model.Comment) error
	GetByID(id uint) (*model.Comment, error)
	Update(comment *model.Comment) error
	Delete(id uint) error
	List(page, pageSize int, extType int, extID int) ([]model.Comment, int64, error)
	GetReplies(parentID uint) ([]model.Comment, error)
	IncrementLikeCount(id uint) error
	DecrementLikeCount(id uint) error
}

// LikeDAOInterface 点赞 DAO 接口
type LikeDAOInterface interface {
	Create(like *model.Like) error
	GetByUserAndExt(userID uint, extType int, extID int) (*model.Like, error)
	Delete(userID uint, extType int, extID int) error
	CountByExt(extType int, extID int) (int64, error)
}

// GoodDAOInterface 商品 DAO 接口
type GoodDAOInterface interface {
	Create(good *model.Good) error
	GetByID(id uint) (*model.Good, error)
	Update(good *model.Good) error
	Delete(id uint) error
	List(page, pageSize int, userID *uint, goodStatus *int, status *int8, keyword string, minPrice, maxPrice *int) ([]model.Good, int64, error)
}

// TagDAOInterface 标签 DAO 接口
type TagDAOInterface interface {
	Create(tag *model.Tag) error
	GetByID(id uint) (*model.Tag, error)
	GetByExt(extType int, extID int) ([]model.Tag, error)
	Delete(id uint) error
	DeleteByExt(extType int, extID int) error
}

