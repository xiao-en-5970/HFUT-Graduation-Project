// Package util 的 jwt.go：用户登录侧的 JWT 工具——双 token 模型。
//
// 设计要点：
//
//   - access token：短有效期，业务接口都用这个。客户端每次请求带它，过期后由 refresh
//     token 自动换发。**写死 5 分钟**，不走 env，避免不同环境配置漂移。
//   - refresh token：长有效期，仅用于换发 access token。客户端在 access 过期时调
//     /user/refresh 接口；**写死 30 天**，等同于"30 天不打开 app 才会强制重新登录"。
//   - 两类 token 共用同一个 HS256 secret（config.JWTSecret），但 claims 里的 typ 字段
//     不同，互相不能替代——防止把 refresh token 当 access 直接拿去打业务接口。
//
// 说明：bot service-to-service JWT（X-Bot-Service-Token / X-Service-Token）是另一套
// 独立体系，跟本文件无关，详见 package/botinternal 与 QQ-bot/utils/hfut。
package util

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
)

const (
	// AccessTokenTTL access token 有效期 —— 写死 5 分钟。
	//
	// 为什么不走 env：双 token 的安全前提之一就是 access 短到一旦泄漏窗口期极短，
	// 由部署方随意改长会破坏这一假设；客户端也只需要做一次"401 时换 token"逻辑。
	AccessTokenTTL = 5 * time.Minute

	// RefreshTokenTTL refresh token 有效期 —— 写死 30 天。
	//
	// 30 天对应"一个月不进 app 就重新登录"的体感；同时给运营留够空间做 token 黑名单
	// 时不至于强制全员重新登录。
	RefreshTokenTTL = 30 * 24 * time.Hour

	// tokenTypeAccess / tokenTypeRefresh：写在 claims.typ 里区分两类 token。
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"

	// JWTIssuer 用户 JWT 的 issuer；跟 service token 的 issuer 故意不同。
	JWTIssuer = "HFUT-Graduation-Project"
)

// getJWTSecret 获取 JWT 密钥
func getJWTSecret() []byte {
	return []byte(config.JWTSecret)
}

// Claims 用户 JWT 载荷。
//
// Type 字段是两类 token 的分流标记；Username 仅做 log/审计用，业务接口取用户 ID 应优先
// 通过 user_id（middleware.GetUserID）。
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Type     string `json:"typ"`
	jwt.RegisteredClaims
}

func generateToken(userID uint, username, typ string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		Type:     typ,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    JWTIssuer,
		},
	}
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tokenClaims.SignedString(getJWTSecret())
}

// GenerateAccessToken 生成 5 分钟有效期的 access token。
func GenerateAccessToken(userID uint, username string) (string, error) {
	return generateToken(userID, username, tokenTypeAccess, AccessTokenTTL)
}

// GenerateRefreshToken 生成 30 天有效期的 refresh token。
func GenerateRefreshToken(userID uint, username string) (string, error) {
	return generateToken(userID, username, tokenTypeRefresh, RefreshTokenTTL)
}

// GenerateTokenPair 一次拿到 access+refresh，登录 / 刷新都用它。
//
// expiresIn 是 access 剩余秒数，给前端用作"距离下次刷新还有多久"的提示——客户端可不依赖
// 这个值，被动 401 时再 refresh 即可。
func GenerateTokenPair(userID uint, username string) (access, refresh string, expiresIn int64, err error) {
	access, err = GenerateAccessToken(userID, username)
	if err != nil {
		return
	}
	refresh, err = GenerateRefreshToken(userID, username)
	if err != nil {
		return
	}
	expiresIn = int64(AccessTokenTTL.Seconds())
	return
}

// ErrTokenExpired access / refresh 任一过期都返这个 sentinel；调用方按需区分（中间件返
// 不同 code，refresh 接口直接拒）。
var ErrTokenExpired = errors.New("token expired")

// ErrTokenWrongType 把 access 当 refresh 或反之 —— 直接拒。
var ErrTokenWrongType = errors.New("token type mismatch")

func parseToken(token, expectedType string) (*Claims, error) {
	tc, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return getJWTSecret(), nil
	})
	if err != nil {
		// 区分过期：让中间件给前端 4011，方便前端触发刷新逻辑
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, err
	}
	claims, ok := tc.Claims.(*Claims)
	if !ok || !tc.Valid {
		return nil, errors.New("invalid token")
	}
	if expectedType != "" && claims.Type != "" && claims.Type != expectedType {
		return nil, ErrTokenWrongType
	}
	return claims, nil
}

// ParseAccessToken 解 access；type=refresh 会被拒。
//
// 容错：claims.Type 为空（极旧版本签的 token，没有 typ 字段）时**视为 access**，避免存量
// 用户被强制下线。新签的 token 都会显式带 typ。
func ParseAccessToken(token string) (*Claims, error) {
	return parseToken(token, tokenTypeAccess)
}

// ParseRefreshToken 解 refresh；非 refresh 类型会被拒。
//
// 不做"空 typ 视为 refresh"的容错——refresh 是新增能力，旧 token 没带这字段就不该被认作
// refresh，否则可能被恶意把 access 当 refresh 来无限刷。
func ParseRefreshToken(token string) (*Claims, error) {
	tc, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return getJWTSecret(), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, err
	}
	claims, ok := tc.Claims.(*Claims)
	if !ok || !tc.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.Type != tokenTypeRefresh {
		return nil, ErrTokenWrongType
	}
	return claims, nil
}

// 下面两个保留旧名以兼容现存调用方（若有）。

// GenerateToken 兼容入口——等价于 GenerateAccessToken。新代码请用 GenerateTokenPair。
func GenerateToken(userID uint, username string) (string, error) {
	return GenerateAccessToken(userID, username)
}

// ParseToken 兼容入口——等价于 ParseAccessToken。
func ParseToken(token string) (*Claims, error) {
	return ParseAccessToken(token)
}

// GetUserIDFromToken 从 token 中获取用户 ID
func GetUserIDFromToken(token string) (uint, error) {
	claims, err := ParseAccessToken(token)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}
