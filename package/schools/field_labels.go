package schools

// FormFieldLabel 表单字段的中英文展示名
type FormFieldLabel struct {
	Key    string `json:"key"`
	LabelZh string `json:"label_zh"`
	LabelEn string `json:"label_en"`
}

// 标准表单字段的中英文标签，供前端展示
var formFieldLabels = map[string]struct{ Zh, En string }{
	"username": {"学号", "Student ID"},
	"password": {"密码", "Password"},
	"captcha":  {"验证码", "Captcha"},
}

// BuildFormFieldsWithLabels 将 form_fields 键列表转为含中英文标签的对象列表
func BuildFormFieldsWithLabels(keys []string) []FormFieldLabel {
	if len(keys) == 0 {
		keys = []string{"username", "password"}
	}
	var out []FormFieldLabel
	for _, k := range keys {
		labels, ok := formFieldLabels[k]
		if !ok {
			labels = struct{ Zh, En string }{k, k}
		}
		out = append(out, FormFieldLabel{
			Key:     k,
			LabelZh: labels.Zh,
			LabelEn: labels.En,
		})
	}
	return out
}
