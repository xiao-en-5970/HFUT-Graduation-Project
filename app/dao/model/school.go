package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// FormFieldItem 单个表单字段，每学校可自定义 label
type FormFieldItem struct {
	Key     string `json:"key"`
	LabelZh string `json:"label_zh"`
	LabelEn string `json:"label_en"`
}

// FormFieldsJSON 表单字段列表，每学校独立配置
type FormFieldsJSON []FormFieldItem

func (f FormFieldsJSON) Value() (driver.Value, error) {
	if f == nil {
		return "[]", nil
	}
	return json.Marshal(f)
}

func (f *FormFieldsJSON) Scan(value interface{}) error {
	if value == nil {
		*f = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("form_fields must be []byte")
	}
	// 兼容旧格式 ["username","password"] 与 新格式 [{"key":...,"label_zh":...,"label_en":...}]
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	var out []FormFieldItem
	for _, r := range raw {
		var s string
		if err := json.Unmarshal(r, &s); err == nil {
			out = append(out, defaultFormFieldLabel(s))
			continue
		}
		var obj FormFieldItem
		if err := json.Unmarshal(r, &obj); err != nil {
			continue
		}
		if obj.Key == "" && (obj.LabelZh != "" || obj.LabelEn != "") {
			continue
		}
		if obj.Key == "" {
			continue
		}
		if obj.LabelZh == "" {
			obj.LabelZh = obj.Key
		}
		if obj.LabelEn == "" {
			obj.LabelEn = obj.Key
		}
		out = append(out, obj)
	}
	*f = out
	return nil
}

// HasKey 是否包含某字段（如 captcha）
func (f FormFieldsJSON) HasKey(key string) bool {
	for _, item := range f {
		if item.Key == key {
			return true
		}
	}
	return false
}

func defaultFormFieldLabel(key string) FormFieldItem {
	labels := map[string]struct{ Zh, En string }{
		"username": {"学号", "Student ID"},
		"password": {"密码", "Password"},
		"captcha":  {"验证码", "Captcha"},
	}
	l, ok := labels[key]
	if !ok {
		return FormFieldItem{Key: key, LabelZh: key, LabelEn: key}
	}
	return FormFieldItem{Key: key, LabelZh: l.Zh, LabelEn: l.En}
}

// School 学校表
type School struct {
	ID         uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       *string        `gorm:"type:varchar(50)" json:"name"`
	Code       *string        `gorm:"type:varchar(32);uniqueIndex" json:"code"`
	LoginURL   *string        `gorm:"column:login_url;type:varchar(512)" json:"login_url"`
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
