// Package botinternal 封装 hfut 反向调用 bot 的"internal HTTP API"。
//
// 鉴权（跟 bot → hfut 方向**对称**的 service-to-service JWT）：
//
//   - 共享 HS256 secret = config.BotServiceJWTSecret（env BOT_SERVICE_JWT_SECRET）；
//     跟 bot 那边 HFUT_API_JWT_SECRET 是同一个值，2 个调用方向共用 1 个 secret
//   - 每次请求自签 60s 有效期的 JWT 放 X-Service-Token 头
//   - iss = "HFUT-Graduation-Project-hfut"——故意跟反向 (HFUT-Graduation-Project-bot) 不同，
//     防止把 hfut 这边的 token 拿去反向调用 bot 的 internal API（即便 secret 共享）
//
// 使用前必须调 Init()——bootstrap 启动期就调一次；Init 失败时 Default 为 nil，
// service 层调用方自行兜底（QQ 绑定流程返"系统繁忙"）。
//
// 错误语义：
//   - HTTP 4xx + 信封 code=4xx → ClientError（含 HTTPStatus + Message）
//   - 网络错 / HTTP 5xx → 普通 error
//   - send-private 时对方不是好友：bot 那边返 502 + message 含"好友"——上层用
//     errors.Is(err, ErrBotNotFriend) 判别
package botinternal

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
)

// hfutToBotIssuer 跟 bot 端 internal API server 校验的 iss 值对齐。
//
// 把 bot 那边自签的 token (iss=HFUT-Graduation-Project-bot) 拿来反向调用 bot
// 的 internal API 时会被拒——这样即便 secret 共享，方向也不可混用。
const hfutToBotIssuer = "HFUT-Graduation-Project-hfut"

// serviceTokenHeader 头名跟 bot 端对齐。
const serviceTokenHeader = "X-Service-Token"

// internalServiceTokenTTL 每次自签 token 的有效期；60s 跟反向方向一致。
const internalServiceTokenTTL = 60 * time.Second

// hfutToBotClaims JWT claims；跟 bot 端 hfutToBotClaims 字段对齐。
type hfutToBotClaims struct {
	Service string `json:"service"`
	jwt.RegisteredClaims
}

// ErrBotNotFriend QQ 不是 bot 好友——上层应该让用户先加 bot 为好友再继续绑定流程。
var ErrBotNotFriend = errors.New("botinternal: 目标 QQ 不是 bot 的好友")

// ErrBotUnavailable bot 整体不可达 / 内部服务挂了 / Token 配错了。
var ErrBotUnavailable = errors.New("botinternal: bot 服务不可达")

// ClientError bot 那边返回的非 200 响应（包括 4xx/5xx 但能解析信封的情况）。
type ClientError struct {
	HTTPStatus int
	Code       int    // 信封 code（跟 HTTP 一致）
	Message    string // 人类可读
}

func (e *ClientError) Error() string {
	return fmt.Sprintf("botinternal: HTTP %d / %s", e.HTTPStatus, e.Message)
}

// Client 反向调用 bot 的客户端。单例，bootstrap 启动期初始化一次。
type Client struct {
	baseURL    string
	jwtSecret  []byte
	httpClient *http.Client
}

// Default 全局单例；不可用时为 nil。service 层调用前先判 nil。
var Default *Client

// Init 按 config.BotInternalAPIURL + config.BotServiceJWTSecret 构造 Default 单例。
//
// 二者任一缺失 → 不构造，Default 仍为 nil（安全降级，QQ 绑定流程拒绝）。
//
// URL 防呆校验：
//   - 必须以 http:// 或 https:// 开头——之前踩过坑：env 写成 "qq-bot-server:8090"
//     （缺 scheme），http.NewRequest 按相对路径处理直接 502。这里启动期就拒绝。
//   - 末尾的 / 自动剥掉，调用方拼 path 时不重复
func Init() error {
	url := strings.TrimRight(strings.TrimSpace(config.BotInternalAPIURL), "/")
	secret := strings.TrimSpace(config.BotServiceJWTSecret)
	if url == "" || secret == "" {
		Default = nil
		return nil // 不算错；让调用方按 Default == nil 兜底
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		Default = nil
		return fmt.Errorf("botinternal: BOT_INTERNAL_API_URL 必须以 http:// 或 https:// 开头，当前值 %q", url)
	}
	Default = &Client{
		baseURL:   url,
		jwtSecret: []byte(secret),
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // bot 那边 get_friend_list 可能略慢
		},
	}
	return nil
}

// signToken 临时签一个 60s 有效期的 service JWT；每次请求都重新签（HS256 几个微秒）。
func (c *Client) signToken() (string, error) {
	now := time.Now()
	jti, err := newJTI()
	if err != nil {
		return "", fmt.Errorf("生成 jti 失败: %w", err)
	}
	claims := hfutToBotClaims{
		Service: "hfut-api",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    hfutToBotIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(internalServiceTokenTTL)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(c.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("签名失败: %w", err)
	}
	return signed, nil
}

// newJTI 16 字节 hex（128 bit 熵），让 bot 端按需 log/审计每次调用。
func newJTI() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// envelope bot internal API 的统一信封（跟 bot 端一致）
type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// CheckFriend 询问 bot：指定 QQ 是不是它的好友。
//
// noCache=true 强制刷新 NapCat 缓存（用户刚加好友的几秒内可能没同步）。
func (c *Client) CheckFriend(ctx context.Context, qq int64, noCache bool) (bool, error) {
	body := map[string]interface{}{"qq_number": qq, "no_cache": noCache}
	var out struct {
		IsFriend bool `json:"is_friend"`
	}
	if err := c.doJSON(ctx, "/internal/qq/check-friend", body, &out); err != nil {
		return false, err
	}
	return out.IsFriend, nil
}

// SendPrivate 让 bot 给指定 QQ 发一条文本私聊。
//
// 错误：
//   - 对方不是好友 → ErrBotNotFriend（上层应提示用户加 bot 好友）
//   - bot/NapCat 整体不可达 → ErrBotUnavailable
func (c *Client) SendPrivate(ctx context.Context, qq int64, text string) error {
	body := map[string]interface{}{"qq_number": qq, "text": text}
	if err := c.doJSON(ctx, "/internal/qq/send-private", body, nil); err != nil {
		// bot 端返 502 表示 NapCat 拒发——典型场景就是不是好友 / 临时会话不允许
		// 这里把"含好友"的 502 转 ErrBotNotFriend；其它 502 当 ErrBotUnavailable
		var ce *ClientError
		if errors.As(err, &ce) && ce.HTTPStatus == http.StatusBadGateway {
			if strings.Contains(ce.Message, "好友") {
				return ErrBotNotFriend
			}
			return ErrBotUnavailable
		}
		return err
	}
	return nil
}

// SendGroup 让 bot 在群里发消息。
//
// qq != 0：发"@user + text"组合；qq == 0：发纯文本群消息。
func (c *Client) SendGroup(ctx context.Context, groupID, qq int64, text string) error {
	body := map[string]interface{}{"group_id": groupID, "qq_number": qq, "text": text}
	return c.doJSON(ctx, "/internal/qq/send-group", body, nil)
}

// doJSON 发 POST + JSON 请求，按信封解响应；out=nil 时只校验信封 code。
func (c *Client) doJSON(ctx context.Context, path string, reqBody interface{}, out interface{}) error {
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("botinternal: 请求体序列化失败: %w", err)
	}
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("botinternal: 创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	token, err := c.signToken()
	if err != nil {
		return fmt.Errorf("botinternal: 签 service token 失败: %w", err)
	}
	req.Header.Set(serviceTokenHeader, token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBotUnavailable, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("botinternal: 读响应失败: %w", err)
	}

	var env envelope
	if jerr := json.Unmarshal(respBody, &env); jerr != nil {
		// 信封解不出 + HTTP 不 OK：把 raw 截短带回去
		if resp.StatusCode/100 != 2 {
			return fmt.Errorf("botinternal: HTTP %d 且信封解析失败: %s", resp.StatusCode, truncate(string(respBody), 200))
		}
		return fmt.Errorf("botinternal: 信封解析失败: %w", jerr)
	}

	if env.Code != 200 || resp.StatusCode/100 != 2 {
		return &ClientError{HTTPStatus: resp.StatusCode, Code: env.Code, Message: env.Message}
	}

	if out != nil && len(env.Data) > 0 && string(env.Data) != "null" {
		if jerr := json.Unmarshal(env.Data, out); jerr != nil {
			return fmt.Errorf("botinternal: data 解析失败: %w", jerr)
		}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
