package request

// UserRegisterRequest 用户注册请求
type UserRegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=255"`
	SchoolID *uint  `json:"school_id"`
}

// UserLoginRequest 用户登录请求
type UserLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserUpdateRequest 用户更新请求
type UserUpdateRequest struct {
	Avatar     string `json:"avatar"`
	Background string `json:"background"`
	BindQQ     string `json:"bind_qq"`
	BindWX     string `json:"bind_wx"`
	BindPhone  string `json:"bind_phone"`
	SchoolID   *uint  `json:"school_id"`
}

// ArticleCreateRequest 创建文章请求
type ArticleCreateRequest struct {
	Title         string `json:"title" binding:"required,max=255"`
	Content       string `json:"content" binding:"required"`
	PublishStatus int8   `json:"publish_status"` // 1:私密 2:公开
	Type          int    `json:"type"`           // 1:普通文章 2:提问 3:回答
}

// ArticleUpdateRequest 更新文章请求
type ArticleUpdateRequest struct {
	Title         string `json:"title" binding:"max=255"`
	Content       string `json:"content"`
	PublishStatus *int8  `json:"publish_status"`
	Status        *int8  `json:"status"`
}

// ArticleListRequest 文章列表请求
type ArticleListRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	Type     *int   `form:"type"`
	Status   *int8  `form:"status"`
	UserID   *uint  `form:"user_id"`
	Keyword  string `form:"keyword"`
}

// CommentCreateRequest 创建评论请求
type CommentCreateRequest struct {
	ExtType  int      `json:"ext_type" binding:"required"` // 1:articles 2:goods
	ExtID    int      `json:"ext_id" binding:"required"`
	ParentID *uint    `json:"parent_id"`
	ReplyID  *uint    `json:"reply_id"`
	Content  string   `json:"content" binding:"required"`
	Images   []string `json:"images"`
}

// CommentListRequest 评论列表请求
type CommentListRequest struct {
	Page    int  `form:"page" binding:"min=1"`
	PageSize int `form:"page_size" binding:"min=1,max=100"`
	ExtType int  `form:"ext_type" binding:"required"`
	ExtID   int  `form:"ext_id" binding:"required"`
}

// LikeCreateRequest 创建点赞请求
type LikeCreateRequest struct {
	ExtType int    `json:"ext_type" binding:"required"` // 1:articles 2:comments 3:goods
	ExtID   int    `json:"ext_id" binding:"required"`
	Images  []string `json:"images"`
}

// GoodCreateRequest 创建商品请求
type GoodCreateRequest struct {
	Title      string   `json:"title" binding:"required,max=255"`
	Content    string   `json:"content" binding:"required"`
	Price      int      `json:"price" binding:"min=0"`
	Images     []string `json:"images"`
	GoodStatus int      `json:"good_status"` // 1:在售 2:下架
}

// GoodUpdateRequest 更新商品请求
type GoodUpdateRequest struct {
	Title      string   `json:"title" binding:"max=255"`
	Content    string   `json:"content"`
	Price      *int     `json:"price"`
	Images     []string `json:"images"`
	GoodStatus *int     `json:"good_status"`
	Status     *int8    `json:"status"`
}

// GoodListRequest 商品列表请求
type GoodListRequest struct {
	Page      int    `form:"page" binding:"min=1"`
	PageSize  int    `form:"page_size" binding:"min=1,max=100"`
	GoodStatus *int   `form:"good_status"`
	Status    *int8   `form:"status"`
	UserID    *uint   `form:"user_id"`
	Keyword   string  `form:"keyword"`
	MinPrice  *int    `form:"min_price"`
	MaxPrice  *int    `form:"max_price"`
}

// TagCreateRequest 创建标签请求
type TagCreateRequest struct {
	Name    string `json:"name" binding:"required,max=255"`
	ExtType int    `json:"ext_type" binding:"required"` // 1:articles 2:goods
	ExtID   int    `json:"ext_id" binding:"required"`
}

// SchoolCreateRequest 创建学校请求
type SchoolCreateRequest struct {
	Name     string `json:"name" binding:"required,max=50"`
	LoginURL string `json:"login_url" binding:"max=255"`
}
