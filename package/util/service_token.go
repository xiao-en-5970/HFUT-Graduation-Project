// Package util 的 service_token.go 实现 service-to-service JWT 鉴权的**验签侧**。
//
// 设计模式：共享 secret + bot 端自签
//   - bot（QQ-bot 等）跟 hfut 共享同一个 HS256 secret（独立于 user 登录的 JWT secret）
//   - bot 每次请求自签 60s 有效期 JWT 当 X-Bot-Service-Token 头
//   - hfut 这里只需要"用 secret 验签 + 检 exp + 检 iss + 校验 service 名"，**不查任何 DB**
//   - secret 暴露场景下 rotate 即可：bot env 改成新 secret，hfut env 同步改，旧 token 全失效
//
// 跟 user 登录的 JWT 区别：
//   - secret 不同（独立 env，避免 rotate 牵连）
//   - iss 不同（这里 "HFUT-Graduation-Project-bot"，user 那边 "HFUT-Graduation-Project"）
package util

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
)

// botServiceTokenIssuer service token 专属 iss。bot 端签发时填这个值，hfut 端校验。
const botServiceTokenIssuer = "HFUT-Graduation-Project-bot"

// BotServiceTokenClaims service token 的 JWT claims。
//
// 字段简洁：只有 service 名 + 标准 RegisteredClaims（jti/iss/iat/exp）。
// 不带 owner_user_id——共享 secret 模式下没"owner admin"概念，bot 是独立 service-account。
type BotServiceTokenClaims struct {
	Service string `json:"service"` // bot 端自报的服务名（"qq-bot"），hfut 端用作 log / 审计
	jwt.RegisteredClaims
}

// getBotServiceJWTSecret 从 config 拿 bot 共享 secret。空时认为"未配置 service token 鉴权"，
// middleware 应该按 401 处理（bot 那边没配 secret 也根本来不到这一步）。
func getBotServiceJWTSecret() []byte {
	return []byte(config.BotServiceJWTSecret)
}

// ParseBotServiceToken 验签 + 检 exp + 校验 iss。
//
// 失败返回非 nil err；上层（middleware）据此决定 401。
//
// 防御点：明确只接 HMAC（防 alg=none 或非对称密钥混用攻击）。
func ParseBotServiceToken(token string) (*BotServiceTokenClaims, error) {
	if token == "" {
		return nil, errors.New("空 token")
	}
	if len(getBotServiceJWTSecret()) == 0 {
		return nil, errors.New("server: BotServiceJWTSecret 未配置")
	}
	parsed, err := jwt.ParseWithClaims(token, &BotServiceTokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return getBotServiceJWTSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*BotServiceTokenClaims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid bot service token")
	}
	if claims.Issuer != botServiceTokenIssuer {
		// user JWT 的 iss = "HFUT-Graduation-Project"，明确拒绝
		return nil, errors.New("token issuer mismatch (not a bot service token)")
	}
	return claims, nil
}
