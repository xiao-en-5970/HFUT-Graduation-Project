package dao

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
)

// BotServiceTokenStore 是 bot_service_tokens 表的 DAO。
//
// 关于"明文 vs hash"：上层 service 层负责 hash 计算 + 明文生成，DAO 不接触明文。
type BotServiceTokenStore struct{}

func BotServiceToken() *BotServiceTokenStore { return &BotServiceTokenStore{} }

// Create 落库一条新 token 记录（hash 由上层算好）。
func (s *BotServiceTokenStore) Create(ctx context.Context, t *model.BotServiceToken) error {
	return pgsql.DB.WithContext(ctx).Create(t).Error
}

// FindActiveByHash 根据 token hash 查"当前有效"的记录：未作废且未过期。
//
// 没找到返回 (nil, nil)；上层据此决定 401 不通过。
func (s *BotServiceTokenStore) FindActiveByHash(ctx context.Context, hash string) (*model.BotServiceToken, error) {
	var t model.BotServiceToken
	now := time.Now()
	err := pgsql.DB.WithContext(ctx).
		Where("token_hash = ?", hash).
		Where("revoked_at IS NULL").
		Where("(expires_at IS NULL OR expires_at > ?)", now).
		First(&t).Error
	if err != nil {
		// gorm.ErrRecordNotFound 视作"没找到/无效"，由上层判断
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// TouchLastUsed 把 LastUsedAt 刷成当前时间。鉴权成功后调，失败不调。
func (s *BotServiceTokenStore) TouchLastUsed(ctx context.Context, id uint) error {
	return pgsql.DB.WithContext(ctx).
		Model(&model.BotServiceToken{}).
		Where("id = ?", id).
		UpdateColumn("last_used_at", time.Now()).Error
}

// Revoke 主动作废一条 token。
func (s *BotServiceTokenStore) Revoke(ctx context.Context, id uint) error {
	now := time.Now()
	return pgsql.DB.WithContext(ctx).
		Model(&model.BotServiceToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		UpdateColumn("revoked_at", now).Error
}

// List 列出所有 token 给 admin 查看（不返回 hash 字段，结构体本身就 json:"-"）。
func (s *BotServiceTokenStore) List(ctx context.Context) ([]*model.BotServiceToken, error) {
	var ts []*model.BotServiceToken
	err := pgsql.DB.WithContext(ctx).Order("id DESC").Find(&ts).Error
	return ts, err
}

// isNotFound 判断是不是 gorm 的 record not found。
func isNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
