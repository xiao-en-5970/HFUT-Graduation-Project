package response

// AuthorProfile 作者简要信息（用于文章、评论等关联展示）
type AuthorProfile struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

// UserProfile 用户公开身份信息（供他人查看，仅非删用户）
type UserProfile struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	Avatar      string `json:"avatar"`
	Background  string `json:"background"`
	FollowCount int    `json:"follow_count"`
	FansCount   int    `json:"fans_count"`
	CreatedAt   string `json:"created_at"` // 注册时间
}

// UserInfo 当前用户完整信息（info 接口返回，含绑定信息等）
type UserInfo struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	Avatar      string `json:"avatar"`
	Background  string `json:"background"`
	FollowCount int    `json:"follow_count"`
	FansCount   int    `json:"fans_count"`
	Role        int16  `json:"role"`   // 1普通 2管理员 3超管 4匿名
	Status      int16  `json:"status"` // 1正常 2禁用
	SchoolID    uint   `json:"school_id"`
	SchoolName  string `json:"school_name,omitempty"` // 学校名称，未绑定时为空
	BindQQ      string `json:"bind_qq"`
	BindWX      string `json:"bind_wx"`
	BindPhone   string `json:"bind_phone"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}
