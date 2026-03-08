package schools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/redis"
)

const captchaTTL = 5 * time.Minute
const captchaKeyPrefix = "school_captcha:"

// StoreCaptchaSession 存储验证码会话（cookie 等），返回 token
func StoreCaptchaSession(ctx context.Context, schoolCode, cookieStr string) (string, error) {
	if redis.Client == nil {
		return "", fmt.Errorf("Redis 未初始化，无法存储验证码会话")
	}
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	key := captchaKeyPrefix + schoolCode + ":" + hex.EncodeToString(token)
	if err := redis.Client.Set(ctx, key, cookieStr, captchaTTL).Err(); err != nil {
		return "", fmt.Errorf("存储验证码会话失败: %w", err)
	}
	return hex.EncodeToString(token), nil
}

// GetCaptchaSession 获取验证码会话，获取后删除（一次性使用）
func GetCaptchaSession(ctx context.Context, schoolCode, token string) (string, error) {
	if redis.Client == nil {
		return "", fmt.Errorf("Redis 未初始化，无法获取验证码会话")
	}
	if token == "" {
		return "", fmt.Errorf("验证码 token 不能为空")
	}
	key := captchaKeyPrefix + schoolCode + ":" + token
	cookieStr, err := redis.Client.Get(ctx, key).Result()
	if err != nil {
		if err == goredis.Nil {
			return "", fmt.Errorf("验证码会话无效或已过期，请重新获取验证码")
		}
		return "", fmt.Errorf("验证码会话无效或已过期: %w", err)
	}
	_ = redis.Client.Del(ctx, key).Err()
	return cookieStr, nil
}
