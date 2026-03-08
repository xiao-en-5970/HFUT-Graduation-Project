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
	// Login 仅需账号密码，内部处理验证码等
	Login(ctx context.Context, username, password string) (*LoginResult, error)
}
