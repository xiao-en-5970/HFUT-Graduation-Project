package model

import (
	"time"
)

// User 用户模型
type User struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Username   string    `gorm:"type:varchar(50);not null;uniqueIndex" json:"username"`
	Password   string    `gorm:"type:varchar(255);not null" json:"-"`
	SchoolID   *uint     `gorm:"index" json:"school_id"`
	School     *School   `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	BindQQ     string    `gorm:"type:varchar(128)" json:"bind_qq,omitempty"`
	BindWX     string    `gorm:"type:varchar(128)" json:"bind_wx,omitempty"`
	BindPhone  string    `gorm:"type:varchar(20)" json:"bind_phone,omitempty"`
	Status     int8      `gorm:"default:1" json:"status"` // 1:正常 2:禁用
	Role       int8      `gorm:"default:1" json:"role"`   // 1:普通用户 2:管理员 3:超级管理员 4:匿名用户
	Avatar     string    `gorm:"type:varchar(255)" json:"avatar,omitempty"`
	Background string    `gorm:"type:varchar(255)" json:"background,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
