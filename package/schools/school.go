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

// LoginOptions 登录所需配置，从 schools 表读取，禁止写死
type LoginOptions struct {
	LoginURL   string // 登录页 URL
	CaptchaURL string // 验证码 URL（用于推导 cas_base 等）
}

// School 学校登录接口，封装与学校端的认证对接
type School interface {
	Code() string // 学校代码，如 hfut
	Name() string // 学校名称
	// Login 登录，opts 从 schools 表配置传入，captcha/captchaToken 为验证码时必填
	Login(ctx context.Context, username, password, captcha, captchaToken string, opts *LoginOptions) (*LoginResult, error)
}

// CaptchaOptions 验证码获取所需配置，从 schools 表读取，禁止写死
type CaptchaOptions struct {
	LoginURL   string // 登录页 URL，用于获取 session cookie
	CaptchaURL string // 验证码图片 URL
}

// SchoolWithCaptcha 需要验证码的学校，实现此接口
type SchoolWithCaptcha interface {
	School
	// GetCaptcha 获取验证码图片及 token，opts 从 schools 表配置传入，禁止使用硬编码
	GetCaptcha(ctx context.Context, opts *CaptchaOptions) (image []byte, token string, err error)
}
