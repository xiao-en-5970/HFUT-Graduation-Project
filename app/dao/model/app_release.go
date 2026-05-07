package model

import "time"

// AppRelease app 版本元数据。
//
// 业务流详见 package/sql/migrate_app_releases.sql 文件头注释。
//
// 字段语义：
//
//	Platform      预留多端：android（目前唯一）；ios 暂不实现（App Store 走自身渠道）
//	VersionName   语义版本号字符串，跟 Android build.gradle::versionName 一致；前端展示用
//	VersionCode   单调递增整数版本号；前端比较以这个为准（避免 "1.10" vs "1.9" 字符串比较坑）
//	APKURL        七牛/本地 OSS 完整 URL；前端 Linking.openURL 跳浏览器下载安装
//	ReleaseNotes  发布说明；纯文本或 markdown，前端用 Text 渲染
//	ForceUpdate   true = 强制更新（弹窗只显示"更新"按钮）；用于线上重大 bug
//	Status        1=valid 前端可见；2=disabled 隐藏（OSS 文件保留以防回滚）
type AppRelease struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Platform     string    `gorm:"type:varchar(16);not null;default:'android';uniqueIndex:uniq_app_release_platform_versioncode,priority:1" json:"platform"`
	VersionName  string    `gorm:"column:version_name;type:varchar(32);not null" json:"version_name"`
	VersionCode  int       `gorm:"column:version_code;type:integer;not null;uniqueIndex:uniq_app_release_platform_versioncode,priority:2" json:"version_code"`
	APKURL       string    `gorm:"column:apk_url;type:varchar(512);not null" json:"apk_url"`
	ReleaseNotes string    `gorm:"column:release_notes;type:text;not null;default:''" json:"release_notes"`
	ForceUpdate  bool      `gorm:"column:force_update;type:boolean;not null;default:false" json:"force_update"`
	Status       int16     `gorm:"type:smallint;not null;default:1" json:"status"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (AppRelease) TableName() string {
	return "app_releases"
}
