package schools

import (
	"context"
	"sync"
)

var (
	registry = make(map[string]School)
	mu       sync.RWMutex
)

// Register 注册学校实现
func Register(s School) {
	mu.Lock()
	defer mu.Unlock()
	registry[s.Code()] = s
}

// Get 按学校代码获取实现
func Get(code string) (School, bool) {
	mu.RLock()
	defer mu.RUnlock()
	s, ok := registry[code]
	return s, ok
}

// Login 统一入口：按学校代码调用对应登录
func Login(ctx context.Context, schoolCode, username, password string) (*LoginResult, error) {
	s, ok := Get(schoolCode)
	if !ok {
		return &LoginResult{Success: false, Message: "不支持的学校"}, nil
	}
	return s.Login(ctx, username, password)
}
