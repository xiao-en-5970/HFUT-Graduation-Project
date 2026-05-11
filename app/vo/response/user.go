package response

// AuthorProfile 作者简要信息（用于文章、评论、商品列表等聚合接口的"作者卡片"）。
//
// 展示规则（前端通用 AuthorChip）：
//   - 优先用 Nickname，没设置 fallback 到 Username
//   - Avatar 走 oss.ToFullURL 已是完整 URL；前端拿到直接渲染，不需要再判断 QQ vs 上传
//
// AccountType + ParentUserID 描述账号关联关系（点击作者后跳个人展示页时用）：
//
//	普通账号                  AccountType=1, ParentUserID=0
//	非孤儿 QQ 旗下号           AccountType=2, ParentUserID=主账号id  ParentNickname=主账号展示名
//	孤儿 QQ 旗下号             AccountType=2, ParentUserID=0          ParentNickname=""
//
// 在个人展示页上，旗下号会渲染为 "QQ智能体" tag，且若有 ParentUserID 则展示"关联自「xxx」"
// 且 xxx 可点击跳到主账号的个人展示页。详见 QQ-bot/skill/bot/SKILL.md "个人展示页"。
//
// 兼容：保留 FromUserID/FromUsername 字段——老版前端仍按这两个字段渲染"来自 xxx"后缀，
// 但新版前端应当用 ParentUserID/ParentNickname + AccountType 综合判断（前缀展示
// "QQ智能体" tag、关联自主账号链接）。
type AuthorProfile struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`           // 登录名（旗下号是 qqXXXXX）；保留兼容
	Nickname string `json:"nickname,omitempty"` // 展示名（QQ 旗下号是 QQ 昵称/群名片，普通用户是自定义昵称）
	Avatar   string `json:"avatar"`             // 完整 URL（旗下号是 q.qlogo.cn，普通用户是 OSS）

	AccountType    int16  `json:"account_type"`              // 1=普通 2=QQ 旗下号
	ParentUserID   uint   `json:"parent_user_id,omitempty"`  // 旗下号挂靠的主账号 id
	ParentNickname string `json:"parent_nickname,omitempty"` // 主账号展示名

	// 历史字段——给老前端兼容。新前端用 ParentUserID/ParentNickname + AccountType。
	FromUserID   uint   `json:"from_user_id,omitempty"`
	FromUsername string `json:"from_username,omitempty"`
}

// UserProfile 用户个人展示页信息（供他人查看，仅非删用户）——B 站个人页风格。
//
// IsFollowing / IsFollowedBy 仅当 viewer 已登录且不是 self 时填充：
//
//	IsFollowing   viewer 是否关注了这个 user
//	IsFollowedBy  这个 user 是否关注了 viewer（拼成"互相关注"标签用）
//	IsSelf        viewer 就是这个 user 本人
//
// AccountType / ParentUserID / ParentNickname 取值规则同 AuthorProfile——旗下号在
// 个人展示页上额外渲染"QQ智能体"tag，且如挂主账号给出"关联自「xxx」"可点击的子卡片。
type UserProfile struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname,omitempty"`
	Avatar      string `json:"avatar"`
	Bio         string `json:"bio"`
	Background  string `json:"background"`
	FollowCount int    `json:"follow_count"`
	FansCount   int    `json:"fans_count"`
	CreatedAt   string `json:"created_at"` // 注册时间（YYYY-MM-DD HH:MM:SS）

	AccountType    int16  `json:"account_type"`
	ParentUserID   uint   `json:"parent_user_id,omitempty"`
	ParentNickname string `json:"parent_nickname,omitempty"`

	IsFollowing  bool `json:"is_following"`
	IsFollowedBy bool `json:"is_followed_by"`
	IsSelf       bool `json:"is_self"`
}

// UserInfo 当前用户完整信息（info 接口返回，含绑定信息等）
type UserInfo struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname,omitempty"`
	Avatar      string `json:"avatar"`
	Bio         string `json:"bio"`
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
