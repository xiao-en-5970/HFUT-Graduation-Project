package response

// AuthorProfile 作者简要信息（用于文章、评论等关联展示）
//
// QQ 旗下号语义：旗下账号是"发布渠道标签"——主账号通过 QQ 渠道发布的内容仍然可识别地
// 标到主账号身上。所以非孤儿旗下号作为作者时，会额外带 FromUserID + FromUsername，
// 前端拼成形如"username（来自用户 xxx）"展示——既保留了"通过 QQ 发布"这个语义，
// 又让主账号身份可见。详见 QQ-bot/skill/bot/SKILL.md "数据聚合 / 操作权限"段。
//
// 取值情况：
//
//	作者是普通账号                  ID=作者id      Username=作者用户名     FromUserID/Username=空
//	作者是非孤儿 QQ 旗下号           ID=旗下号id   Username=旗下号用户名   FromUserID=主账号id  FromUsername=主账号用户名
//	作者是孤儿 QQ 旗下号             ID=旗下号id   Username=旗下号用户名   FromUserID/Username=空（无主账号）
type AuthorProfile struct {
	ID           uint   `json:"id"`
	Username     string `json:"username"`
	Avatar       string `json:"avatar"`
	FromUserID   uint   `json:"from_user_id,omitempty"`
	FromUsername string `json:"from_username,omitempty"`
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

	// QQ 旗下账号信息（详见 QQ-bot/skill/bot/SKILL.md "数据聚合 / 操作权限"）。
	// 字段不为零时表示主账号当前挂着一个 QQ 旗下账号。
	// 前端在"我的商品 / 我的提问 / 通知"等列表里通过 item.user_id == QQChildUserID
	// 判断条目"来自 QQ"，给条目加 tag 标识——这是 P2b 阶段的数据聚合契约。
	QQChildUserID   uint   `json:"qq_child_user_id,omitempty"`
	QQChildQQNumber string `json:"qq_child_qq_number,omitempty"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
