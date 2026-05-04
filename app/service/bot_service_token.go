package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
)

// botServiceTokenPlainBytes 明文 token 的字节长度（hex 后字符数 = 2*N）。
// 32 字节 → 64 hex 字符，跟主流 secret 强度一致。
const botServiceTokenPlainBytes = 32

// CreateBotServiceTokenReq 创建 token 的入参。
type CreateBotServiceTokenReq struct {
	Name          string `json:"name"`            // 必填，可读名称
	Description   string `json:"description"`     // 可选
	ExpiresInDays int    `json:"expires_in_days"` // 0 = 不过期
}

// CreateBotServiceTokenResp 一次性返回的明文 token + 元信息。
//
// **明文 Token 仅在创建时返回一次**，调用方务必妥善保存；之后任何接口都查不到原文。
type CreateBotServiceTokenResp struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Token       string     `json:"token"` // 明文，一次性
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// CreateBotServiceToken 由 admin 调用，落库一条新的 service token，并把明文一次性返回。
//
// 实施细节：
//   - 用 crypto/rand 生成 32 字节随机数，hex encode → 64 字符明文
//   - 数据库只存 sha256(明文) hex；明文保存职责在调用方
func CreateBotServiceToken(ctx context.Context, creatorUserID *int, req CreateBotServiceTokenReq) (*CreateBotServiceTokenResp, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("name 不能为空")
	}

	// 1) 生成明文
	plain, err := generatePlainToken(botServiceTokenPlainBytes)
	if err != nil {
		return nil, fmt.Errorf("生成 token 失败: %w", err)
	}
	hash := hashToken(plain)

	// 2) 落库
	t := &model.BotServiceToken{
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		TokenHash:   hash,
		CreatedBy:   creatorUserID,
	}
	if req.ExpiresInDays > 0 {
		exp := time.Now().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
		t.ExpiresAt = &exp
	}
	if err := dao.BotServiceToken().Create(ctx, t); err != nil {
		return nil, fmt.Errorf("落库失败: %w", err)
	}

	return &CreateBotServiceTokenResp{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Token:       plain,
		CreatedAt:   t.CreatedAt,
		ExpiresAt:   t.ExpiresAt,
	}, nil
}

// ListBotServiceTokens 返回所有 token 元信息（含 last_used_at / revoked_at），不含明文 / hash。
func ListBotServiceTokens(ctx context.Context) ([]*model.BotServiceToken, error) {
	return dao.BotServiceToken().List(ctx)
}

// RevokeBotServiceToken 主动作废一条 token；幂等（重复调返回 nil）。
func RevokeBotServiceToken(ctx context.Context, id uint) error {
	return dao.BotServiceToken().Revoke(ctx, id)
}

// VerifyBotServiceToken 给 middleware 用：校验明文 token 是否对应一条有效记录。
//
// 返回非 nil 的 model 表示鉴权通过；nil 表示无效（hash 不存在 / 已作废 / 已过期）。
// 鉴权通过时同步刷一下 LastUsedAt（任何错误只 log，不影响 OK 路径）。
func VerifyBotServiceToken(ctx context.Context, plain string) (*model.BotServiceToken, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return nil, nil
	}
	hash := hashToken(plain)
	t, err := dao.BotServiceToken().FindActiveByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, nil
	}
	// 异步刷 last_used_at 即可——这里同步刷在事务外，写一行 SQL 没什么影响
	_ = dao.BotServiceToken().TouchLastUsed(ctx, t.ID)
	return t, nil
}

// generatePlainToken 生成 hex 编码的 random token。
func generatePlainToken(byteLen int) (string, error) {
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// hashToken 把明文 token 用 sha256 hash 成 hex。
//
// 我们故意用最直白的 sha256 而不是 bcrypt——因为：
//   - 这是 service token，长度 64 hex（256 位熵），暴力破解不现实，不需要"慢哈希"防字典攻击
//   - 中间件每个请求都要 hash 一次，bcrypt 太慢
//   - sha256 + 高熵 = 实际上跟 bcrypt+短密码 安全等级相当
func hashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
