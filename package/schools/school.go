package schools

import "context"

// LoginResult 学校端登录结果
type LoginResult struct {
	Success   bool                   // 是否验证成功
	Message   string                 // 失败时的错误信息
	StudentID string                 // 学号（成功时）
	Name      string                 // 姓名（如有）
	CertInfo  map[string]interface{} // 学生信息 JSON，来自学校端接口
}

// School 学校登录接口，封装与学校端的认证对接
type School interface {
	Code() string // 学校代码，如 hfut
	Name() string // 学校名称
	// Login 登录，captcha/captchaToken 为验证码时必填（由前端先调 GetCaptcha 获取）
	Login(ctx context.Context, username, password, captcha, captchaToken string) (*LoginResult, error)
}

// SchoolWithCaptcha 需要验证码的学校，实现此接口
type SchoolWithCaptcha interface {
	School
	// GetCaptcha 获取验证码图片及 token，token 用于后续 Login
	GetCaptcha(ctx context.Context) (image []byte, token string, err error)
}
