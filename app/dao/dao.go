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
	Delete(id uint) error
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
	IncrementCollectCount(id uint) error
	DecrementCollectCount(id uint) error
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
	DecrementStock(id uint) error
}

// TagDAOInterface 标签 DAO 接口
type TagDAOInterface interface {
	Create(tag *model.Tag) error
	GetByID(id uint) (*model.Tag, error)
	GetByExt(extType int, extID int) ([]model.Tag, error)
	Delete(id uint) error
	DeleteByExt(extType int, extID int) error
}

// CollectDAOInterface 收藏 DAO 接口
type CollectDAOInterface interface {
	Create(collect *model.Collect) error
	GetByID(id uint) (*model.Collect, error)
	GetByUserAndExt(userID uint, extType int, extID int) (*model.Collect, error)
	Delete(id uint) error
	DeleteByUserAndExt(userID uint, extType int, extID int) error
	List(page, pageSize int, userID *uint, extType *int, extID *int) ([]model.Collect, int64, error)
	CountByExt(extType int, extID int) (int64, error)
}

// FollowDAOInterface 关注 DAO 接口
type FollowDAOInterface interface {
	Create(follow *model.Follow) error
	GetByID(id uint) (*model.Follow, error)
	GetByUserAndFollow(userID uint, followID uint) (*model.Follow, error)
	Delete(id uint) error
	DeleteByUserAndFollow(userID uint, followID uint) error
	ListFollowing(page, pageSize int, userID uint) ([]model.Follow, int64, error)
	ListFollowers(page, pageSize int, userID uint) ([]model.Follow, int64, error)
	CountFollowing(userID uint) (int64, error)
	CountFollowers(userID uint) (int64, error)
}

// OrderDAOInterface 订单 DAO 接口
type OrderDAOInterface interface {
	Create(order *model.Order) error
	GetByID(id uint) (*model.Order, error)
	Update(order *model.Order) error
	Delete(id uint) error
	List(page, pageSize int, userID *uint, goodsID *uint, orderStatus *int8) ([]model.Order, int64, error)
}
