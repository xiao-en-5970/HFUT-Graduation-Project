package controller

import (
	"github.com/gin-gonic/gin"
)

// UserControllerInterface 用户控制器接口
type UserControllerInterface interface {
	Register(ctx *gin.Context)
	Login(ctx *gin.Context)
	GetByID(ctx *gin.Context)
	Update(ctx *gin.Context)
	List(ctx *gin.Context)
	Info(ctx *gin.Context)
}

// ArticleControllerInterface 文章控制器接口
type ArticleControllerInterface interface {
	Create(ctx *gin.Context)
	GetByID(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	List(ctx *gin.Context)
}

// CommentControllerInterface 评论控制器接口
type CommentControllerInterface interface {
	Create(ctx *gin.Context)
	List(ctx *gin.Context)
	Delete(ctx *gin.Context)
}

// LikeControllerInterface 点赞控制器接口
type LikeControllerInterface interface {
	ToggleLike(ctx *gin.Context)
	IsLiked(ctx *gin.Context)
}

// GoodControllerInterface 商品控制器接口
type GoodControllerInterface interface {
	Create(ctx *gin.Context)
	GetByID(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	List(ctx *gin.Context)
}

// TagControllerInterface 标签控制器接口
type TagControllerInterface interface {
	Create(ctx *gin.Context)
	GetByExt(ctx *gin.Context)
}

// SchoolControllerInterface 学校控制器接口
type SchoolControllerInterface interface {
	Create(ctx *gin.Context)
	GetByID(ctx *gin.Context)
	List(ctx *gin.Context)
}

// CollectControllerInterface 收藏控制器接口
type CollectControllerInterface interface {
	Create(ctx *gin.Context)
	GetByID(ctx *gin.Context)
	Delete(ctx *gin.Context)
	DeleteByExt(ctx *gin.Context)
	List(ctx *gin.Context)
	IsCollected(ctx *gin.Context)
}

// FollowControllerInterface 关注控制器接口
type FollowControllerInterface interface {
	Follow(ctx *gin.Context)
	Unfollow(ctx *gin.Context)
	GetFollowingList(ctx *gin.Context)
	GetFollowersList(ctx *gin.Context)
	GetFollowCount(ctx *gin.Context)
	IsFollowing(ctx *gin.Context)
}

// OrderControllerInterface 订单控制器接口
type OrderControllerInterface interface {
	Create(ctx *gin.Context)
	GetByID(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	List(ctx *gin.Context)
}
