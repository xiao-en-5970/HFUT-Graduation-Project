package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// FormFieldsJSON 表单字段列表，如 ["username","password","captcha"]
type FormFieldsJSON []string

func (f FormFieldsJSON) Value() (driver.Value, error) {
	if f == nil {
		return "[]", nil
	}
	return json.Marshal(f)
}

func (f *FormFieldsJSON) Scan(value interface{}) error {
	if value == nil {
		*f = []string{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("form_fields must be []byte")
	}
	return json.Unmarshal(b, f)
}

// School 学校表
type School struct {
	ID         uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       *string        `gorm:"type:varchar(50)" json:"name"`
	Code       *string        `gorm:"type:varchar(32);uniqueIndex" json:"code"`
	LoginURL   *string        `gorm:"column:login_url;type:varchar(255)" json:"login_url"`
	FormFields FormFieldsJSON `gorm:"column:form_fields;type:jsonb" json:"form_fields"`
	CaptchaURL *string        `gorm:"column:captcha_url;type:varchar(512)" json:"captcha_url"`
	UserCount  int            `gorm:"column:user_count;default:0" json:"user_count"`
	Status     int16          `gorm:"type:smallint;default:1" json:"status"`
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (School) TableName() string {
	return "schools"
}
