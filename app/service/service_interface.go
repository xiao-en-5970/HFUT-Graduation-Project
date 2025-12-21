package service

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
)

// UserServiceInterface 用户服务接口
type UserServiceInterface interface {
	Register(req *request.UserRegisterRequest) (*response.LoginResponse, error)
	Login(req *request.UserLoginRequest) (*response.LoginResponse, error)
	GetByID(id uint) (*response.UserResponse, error)
	Update(id uint, req *request.UserUpdateRequest) (*response.UserResponse, error)
	List(page, pageSize int, schoolID *uint) (*response.PageResponse, error)
	GetCurrentUser(id uint) (*response.UserResponse, error)
}

// ArticleServiceInterface 文章服务接口
type ArticleServiceInterface interface {
	Create(userID uint, req *request.ArticleCreateRequest) (*response.ArticleResponse, error)
	GetByID(id uint) (*response.ArticleResponse, error)
	Update(userID, articleID uint, req *request.ArticleUpdateRequest) (*response.ArticleResponse, error)
	Delete(userID, articleID uint) error
	List(req *request.ArticleListRequest) (*response.PageResponse, error)
}

// CommentServiceInterface 评论服务接口
type CommentServiceInterface interface {
	Create(userID uint, req *request.CommentCreateRequest) (*response.CommentResponse, error)
	List(req *request.CommentListRequest) (*response.PageResponse, error)
	Delete(userID, commentID uint) error
}

// LikeServiceInterface 点赞服务接口
type LikeServiceInterface interface {
	ToggleLike(userID uint, req *request.LikeCreateRequest) (bool, error)
	IsLiked(userID uint, extType int, extID int) (bool, error)
}

// GoodServiceInterface 商品服务接口
type GoodServiceInterface interface {
	Create(userID uint, req *request.GoodCreateRequest) (*response.GoodResponse, error)
	GetByID(id uint) (*response.GoodResponse, error)
	Update(userID, goodID uint, req *request.GoodUpdateRequest) (*response.GoodResponse, error)
	Delete(userID, goodID uint) error
	List(req *request.GoodListRequest) (*response.PageResponse, error)
}

// TagServiceInterface 标签服务接口
type TagServiceInterface interface {
	Create(req *request.TagCreateRequest) (*response.TagResponse, error)
	GetByExt(extType int, extID int) ([]*response.TagResponse, error)
}

// SchoolServiceInterface 学校服务接口
type SchoolServiceInterface interface {
	Create(req *request.SchoolCreateRequest) (*response.SchoolResponse, error)
	GetByID(id uint) (*response.SchoolResponse, error)
	List() ([]*response.SchoolResponse, error)
}

// CollectServiceInterface 收藏服务接口
type CollectServiceInterface interface {
	Create(userID uint, req *request.CollectCreateRequest) (*response.CollectResponse, error)
	GetByID(id uint) (*response.CollectResponse, error)
	Delete(userID, collectID uint) error
	DeleteByExt(userID uint, req *request.CollectDeleteRequest) error
	List(req *request.CollectListRequest) (*response.PageResponse, error)
	IsCollected(userID uint, extType int, extID int) (bool, error)
}

// FollowServiceInterface 关注服务接口
type FollowServiceInterface interface {
	Follow(userID uint, followID uint) (*response.FollowResponse, error)
	Unfollow(userID uint, followID uint) error
	GetFollowingList(req *request.FollowListRequest) (*response.PageResponse, error)
	GetFollowersList(req *request.FollowListRequest) (*response.PageResponse, error)
	GetFollowCount(userID uint) (*response.FollowCountResponse, error)
	IsFollowing(userID uint, followID uint) (bool, error)
}

// OrderServiceInterface 订单服务接口
type OrderServiceInterface interface {
	Create(userID uint, req *request.OrderCreateRequest) (*response.OrderResponse, error)
	GetByID(id uint) (*response.OrderResponse, error)
	Update(userID, orderID uint, req *request.OrderUpdateRequest) (*response.OrderResponse, error)
	Delete(userID, orderID uint) error
	List(req *request.OrderListRequest) (*response.PageResponse, error)
}
