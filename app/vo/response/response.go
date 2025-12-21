package response

import "github.com/xiao-en-5970/HFUT-Graduation-Project/app/model"

// Response 通用响应
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageResponse 分页响应
type PageResponse struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// UserResponse 用户响应
type UserResponse struct {
	ID         uint   `json:"id"`
	Username   string `json:"username"`
	SchoolID   *uint  `json:"school_id,omitempty"`
	School     *SchoolResponse `json:"school,omitempty"`
	Avatar     string `json:"avatar,omitempty"`
	Background string `json:"background,omitempty"`
	Status     int8   `json:"status"`
	Role       int8   `json:"role"`
	CreatedAt  string `json:"created_at"`
}

// SchoolResponse 学校响应
type SchoolResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	LoginURL  string `json:"login_url,omitempty"`
	UserCount int    `json:"user_count"`
	Status    int8   `json:"status"`
}

// ArticleResponse 文章响应
type ArticleResponse struct {
	ID            uint          `json:"id"`
	UserID        uint          `json:"user_id"`
	User          *UserResponse `json:"user,omitempty"`
	Title         string        `json:"title"`
	Content       string        `json:"content"`
	Status        int8          `json:"status"`
	PublishStatus int8          `json:"publish_status"`
	Type          int           `json:"type"`
	ViewCount     int           `json:"view_count"`
	LikeCount     int           `json:"like_count"`
	CollectCount  int           `json:"collect_count"`
	CreatedAt     string        `json:"created_at"`
	UpdatedAt     string        `json:"updated_at"`
}

// CommentResponse 评论响应
type CommentResponse struct {
	ID        uint            `json:"id"`
	UserID    uint            `json:"user_id"`
	User      *UserResponse   `json:"user,omitempty"`
	ExtType   int             `json:"ext_type"`
	ExtID     int             `json:"ext_id"`
	ParentID  *uint           `json:"parent_id,omitempty"`
	ReplyID   *uint           `json:"reply_id,omitempty"`
	Images    []string        `json:"images,omitempty"`
	Type      int             `json:"type"`
	Content   string          `json:"content"`
	Status    int8            `json:"status"`
	LikeCount int             `json:"like_count"`
	CreatedAt string          `json:"created_at"`
	Replies   []CommentResponse `json:"replies,omitempty"`
}

// GoodResponse 商品响应
type GoodResponse struct {
	ID         uint          `json:"id"`
	UserID     uint          `json:"user_id"`
	User       *UserResponse `json:"user,omitempty"`
	Title      string        `json:"title"`
	Images     []string      `json:"images,omitempty"`
	Content    string        `json:"content"`
	Status     int8          `json:"status"`
	GoodStatus int           `json:"good_status"`
	Price      int           `json:"price"`
	Stock      int           `json:"stock"` // 库存数量
	CreatedAt  string        `json:"created_at"`
	UpdatedAt  string        `json:"updated_at"`
}

// TagResponse 标签响应
type TagResponse struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	ExtType int    `json:"ext_type"`
	ExtID   int    `json:"ext_id"`
	Status  int8   `json:"status"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

// ToUserResponse 转换用户模型为响应
func ToUserResponse(user *model.User) *UserResponse {
	if user == nil {
		return nil
	}
	resp := &UserResponse{
		ID:         user.ID,
		Username:   user.Username,
		SchoolID:   user.SchoolID,
		Avatar:     user.Avatar,
		Background: user.Background,
		Status:     user.Status,
		Role:       user.Role,
		CreatedAt:  user.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if user.School != nil {
		resp.School = ToSchoolResponse(user.School)
	}
	return resp
}

// ToSchoolResponse 转换学校模型为响应
func ToSchoolResponse(school *model.School) *SchoolResponse {
	if school == nil {
		return nil
	}
	return &SchoolResponse{
		ID:        school.ID,
		Name:      school.Name,
		LoginURL:  school.LoginURL,
		UserCount: school.UserCount,
		Status:    school.Status,
	}
}

// ToArticleResponse 转换文章模型为响应
func ToArticleResponse(article *model.Article) *ArticleResponse {
	if article == nil {
		return nil
	}
	resp := &ArticleResponse{
		ID:            article.ID,
		UserID:        article.UserID,
		Title:         article.Title,
		Content:       article.Content,
		Status:        article.Status,
		PublishStatus: article.PublishStatus,
		Type:          article.Type,
		ViewCount:     article.ViewCount,
		LikeCount:     article.LikeCount,
		CollectCount:  article.CollectCount,
		CreatedAt:     article.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:     article.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if article.User != nil {
		resp.User = ToUserResponse(article.User)
	}
	return resp
}

// ToCommentResponse 转换评论模型为响应
func ToCommentResponse(comment *model.Comment) *CommentResponse {
	if comment == nil {
		return nil
	}
	resp := &CommentResponse{
		ID:        comment.ID,
		UserID:    comment.UserID,
		ExtType:   comment.ExtType,
		ExtID:     comment.ExtID,
		ParentID:  comment.ParentID,
		ReplyID:   comment.ReplyID,
		Type:      comment.Type,
		Content:   comment.Content,
		Status:    comment.Status,
		LikeCount: comment.LikeCount,
		CreatedAt: comment.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if comment.Images != nil {
		resp.Images = comment.Images
	}
	if comment.User != nil {
		resp.User = ToUserResponse(comment.User)
	}
	return resp
}

// ToGoodResponse 转换商品模型为响应
func ToGoodResponse(good *model.Good) *GoodResponse {
	if good == nil {
		return nil
	}
	resp := &GoodResponse{
		ID:         good.ID,
		UserID:     good.UserID,
		Title:      good.Title,
		Content:    good.Content,
		Status:     good.Status,
		GoodStatus: good.GoodStatus,
		Price:      good.Price,
		Stock:      good.Stock,
		CreatedAt:  good.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:  good.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if good.Images != nil {
		resp.Images = good.Images
	}
	if good.User != nil {
		resp.User = ToUserResponse(good.User)
	}
	return resp
}

// ToTagResponse 转换标签模型为响应
func ToTagResponse(tag *model.Tag) *TagResponse {
	if tag == nil {
		return nil
	}
	return &TagResponse{
		ID:      tag.ID,
		Name:    tag.Name,
		ExtType: tag.ExtType,
		ExtID:   tag.ExtID,
		Status:  tag.Status,
	}
}

// CollectResponse 收藏响应
type CollectResponse struct {
	ID        uint          `json:"id"`
	UserID    uint          `json:"user_id"`
	User      *UserResponse `json:"user,omitempty"`
	ExtType   int           `json:"ext_type"`
	ExtID     int           `json:"ext_id"`
	Status    int8          `json:"status"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
}

// ToCollectResponse 转换收藏模型为响应
func ToCollectResponse(collect *model.Collect) *CollectResponse {
	if collect == nil {
		return nil
	}
	resp := &CollectResponse{
		ID:        collect.ID,
		UserID:    collect.UserID,
		ExtType:   collect.ExtType,
		ExtID:     collect.ExtID,
		Status:    collect.Status,
		CreatedAt: collect.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: collect.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if collect.User != nil {
		resp.User = ToUserResponse(collect.User)
	}
	return resp
}

// FollowResponse 关注响应
type FollowResponse struct {
	ID        uint          `json:"id"`
	UserID    uint          `json:"user_id"`
	User      *UserResponse `json:"user,omitempty"`      // 关注者信息
	FollowID  uint          `json:"follow_id"`
	Followed  *UserResponse `json:"followed,omitempty"`  // 被关注者信息
	Status    int8          `json:"status"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
}

// FollowCountResponse 关注数量响应
type FollowCountResponse struct {
	FollowingCount int64 `json:"following_count"` // 关注数量（我关注的人数）
	FollowersCount int64 `json:"followers_count"` // 粉丝数量（关注我的人数）
}

// ToFollowResponse 转换关注模型为响应
func ToFollowResponse(follow *model.Follow) *FollowResponse {
	if follow == nil {
		return nil
	}
	resp := &FollowResponse{
		ID:        follow.ID,
		UserID:    follow.UserID,
		FollowID:  follow.FollowID,
		Status:    follow.Status,
		CreatedAt: follow.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: follow.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if follow.User != nil {
		resp.User = ToUserResponse(follow.User)
	}
	if follow.Followed != nil {
		resp.Followed = ToUserResponse(follow.Followed)
	}
	return resp
}

// OrderResponse 订单响应
type OrderResponse struct {
	ID          uint          `json:"id"`
	UserID      uint          `json:"user_id"`
	User        *UserResponse `json:"user,omitempty"`
	GoodsID     uint          `json:"goods_id"`
	Good        *GoodResponse `json:"good,omitempty"`
	Status      int8          `json:"status"`
	OrderStatus int8          `json:"order_status"`
	CreatedAt   string        `json:"created_at"`
	UpdatedAt   string        `json:"updated_at"`
}

// ToOrderResponse 转换订单模型为响应
func ToOrderResponse(order *model.Order) *OrderResponse {
	if order == nil {
		return nil
	}
	resp := &OrderResponse{
		ID:          order.ID,
		UserID:      order.UserID,
		GoodsID:     order.GoodsID,
		Status:      order.Status,
		OrderStatus: order.OrderStatus,
		CreatedAt:   order.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   order.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if order.User != nil {
		resp.User = ToUserResponse(order.User)
	}
	if order.Good != nil {
		resp.Good = ToGoodResponse(order.Good)
	}
	return resp
}
