package model

import "time"

// User 用户表
type User struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username    string    `gorm:"type:varchar(50);not null;uniqueIndex" json:"username"`
	Password    string    `gorm:"type:varchar(255);not null" json:"password"`
	SchoolID    int       `gorm:"column:school_id;index" json:"school_id"`
	BindQQ      *string   `gorm:"column:bind_qq;type:varchar(128)" json:"bind_qq"`
	BindWX      *string   `gorm:"column:bind_wx;type:varchar(128)" json:"bind_wx"`
	BindPhone   *string   `gorm:"column:bind_phone;type:varchar(20)" json:"bind_phone"`
	Status      int16     `gorm:"type:smallint;default:1" json:"status"` // 1:正常 2:禁用
	Role        int16     `gorm:"type:smallint;default:1" json:"role"`   // 1:普通用户 2:管理员 3:超级管理员 4:匿名用户
	Avatar      *string   `gorm:"type:varchar(255)" json:"avatar"`       // 用户头像
	Background  *string   `gorm:"type:varchar(255)" json:"background"`   // 用户背景
	FollowCount int       `gorm:"column:follow_count;not null;default:0" json:"follow_count"`
	FansCount   int       `gorm:"column:fans_count;not null;default:0" json:"fans_count"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
