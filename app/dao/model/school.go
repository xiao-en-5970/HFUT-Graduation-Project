package model

import "time"

// School 学校表
type School struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      *string   `gorm:"type:varchar(50)" json:"name"`                        // 学校名称
	Code      *string   `gorm:"type:varchar(32);uniqueIndex" json:"code"`            // 学校代码，如 hfut，用于对接 package/schools
	LoginURL  *string   `gorm:"column:login_url;type:varchar(255)" json:"login_url"` // 登录地址
	UserCount int       `gorm:"column:user_count;default:0" json:"user_count"`       // 用户数量
	Status    int16     `gorm:"type:smallint;default:1" json:"status"`               // 1:正常 2:禁用
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (School) TableName() string {
	return "schools"
}
