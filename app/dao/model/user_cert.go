package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// CertInfoJSON 学生认证信息 JSON，来自学校端接口
type CertInfoJSON map[string]interface{}

func (c CertInfoJSON) Value() (driver.Value, error) {
	if c == nil {
		return "{}", nil
	}
	return json.Marshal(c)
}

func (c *CertInfoJSON) Scan(value interface{}) error {
	if value == nil {
		*c = make(map[string]interface{})
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("cert_info must be []byte")
	}
	return json.Unmarshal(b, c)
}

// UserCert 用户学校认证表
type UserCert struct {
	ID        uint         `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint         `gorm:"column:user_id;not null;uniqueIndex:idx_user_school" json:"user_id"`
	SchoolID  uint         `gorm:"column:school_id;not null;uniqueIndex:idx_user_school" json:"school_id"`
	CertInfo  CertInfoJSON `gorm:"column:cert_info;type:jsonb;not null" json:"cert_info"`
	CreatedAt time.Time    `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time    `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserCert) TableName() string {
	return "user_cert"
}
