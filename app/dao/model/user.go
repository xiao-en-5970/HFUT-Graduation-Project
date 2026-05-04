package model

import "time"

// 账号类型常量。
//
// AccountTypeNormal 正常注册账号，可登录、参与所有业务。
// AccountTypeQQChild QQ 旗下账号，由 bot 在群消息中为未注册的 QQ 用户自动创建：
//   - 不可独立登录（password 留空）
//   - parent_user_id 为 NULL 时是"孤儿"，等待用户后续 app 注册并主动绑定 QQ 后挂上来
//   - 主账号最多 1 个 QQ 旗下账号（业务层强制 1:1）
//   - 详细行为见 QQ-bot 仓库 skill/bot/SKILL.md 的"旗下账号"段
const (
	AccountTypeNormal  int16 = 1
	AccountTypeQQChild int16 = 2
)

// User 用户表
type User struct {
	ID       uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Username string `gorm:"type:varchar(50);not null;uniqueIndex" json:"username"`
	Password string `gorm:"type:varchar(255);not null" json:"password"`
	SchoolID uint   `gorm:"column:school_id;index" json:"school_id"`

	// QQ 旗下账号机制：详见上面 AccountType 常量注释。注意 BindQQ 与 QQNumber 字段
	// 含义不同——BindQQ 是主账号自报的 QQ（仅展示用），QQNumber 是 QQ 旗下账号绑定的真实 QQ 号。
	// 一对主账号 X + 旗下账号 child 的关系：X.bind_qq 可填可不填；child.qq_number=Y, child.parent_user_id=X.id
	AccountType  int16   `gorm:"column:account_type;type:smallint;not null;default:1" json:"account_type"`
	ParentUserID *int    `gorm:"column:parent_user_id;index" json:"parent_user_id,omitempty"`
	QQNumber     *string `gorm:"column:qq_number;type:varchar(32)" json:"qq_number,omitempty"`

	BindQQ      string    `gorm:"column:bind_qq;type:varchar(128)" json:"bind_qq"`
	BindWX      string    `gorm:"column:bind_wx;type:varchar(128)" json:"bind_wx"`
	BindPhone   string    `gorm:"column:bind_phone;type:varchar(20)" json:"bind_phone"`
	Status      int16     `gorm:"type:smallint;default:1" json:"status"` // 1:正常 2:禁用
	Role        int16     `gorm:"type:smallint;default:1" json:"role"`   // 1:普通用户 2:管理员 3:超级管理员 4:匿名用户
	Avatar      string    `gorm:"type:varchar(255)" json:"avatar"`       // 用户头像
	Background  string    `gorm:"type:varchar(255)" json:"background"`   // 用户背景
	FollowCount int       `gorm:"column:follow_count;not null;default:0" json:"follow_count"`
	FansCount   int       `gorm:"column:fans_count;not null;default:0" json:"fans_count"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// IsQQChild 当前账号是否是 QQ 旗下账号（不可登录、能力受限）。
func (u *User) IsQQChild() bool {
	return u != nil && u.AccountType == AccountTypeQQChild
}

// IsOrphanQQChild 当前账号是否是孤儿 QQ 旗下账号（旗下账号且没挂主账号）。
//
// 孤儿账号的 app 行为受额外限制（详见 skill/bot/SKILL.md 的"孤儿旗下账号的特殊行为"段）：
//   - 商品对话不开放，只展示"通过 QQ 联系"告示 + 已出按钮
//   - 提问的回复转发回创建群、不进 app 通知
func (u *User) IsOrphanQQChild() bool {
	return u.IsQQChild() && u.ParentUserID == nil
}

func (User) TableName() string {
	return "users"
}
