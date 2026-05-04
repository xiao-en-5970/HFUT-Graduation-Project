-- ============================================================================
-- 服务间互信 token: 给 QQ-bot 等服务调 hfut /api/v1/bot/* 用的鉴权凭证
--
-- 设计要点（参考 QQ-bot 仓库 skill/bot/SKILL.md）:
--   - 不存明文，只存 sha256 hex；明文仅在创建时返回一次给管理员妥善保管
--   - 支持滚动作废: revoked_at 置时间 = 立即失效；可发新的、保留旧的"过渡期"双 token 共存
--   - 支持过期时间(可选): expires_at 不传则无限期
--   - 记录 last_used_at: 让 admin 看到哪些 token 还在用、哪些可以清理
-- ============================================================================

CREATE TABLE IF NOT EXISTS bot_service_tokens
(
    id           SERIAL PRIMARY KEY,
    name         VARCHAR(64)  NOT NULL,
    description  VARCHAR(255),
    token_hash   VARCHAR(255) NOT NULL UNIQUE,
    created_by   INTEGER REFERENCES users (id),
    created_at   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at   TIMESTAMP,
    revoked_at   TIMESTAMP,
    last_used_at TIMESTAMP
);

COMMENT ON TABLE bot_service_tokens IS 'service-to-service 鉴权 token；用于 QQ-bot 等服务调 hfut /api/v1/bot/*';
COMMENT ON COLUMN bot_service_tokens.name IS '可读名称(如 "qq-bot-prod"），辅助管理员区分';
COMMENT ON COLUMN bot_service_tokens.description IS '备注说明';
COMMENT ON COLUMN bot_service_tokens.token_hash IS 'sha256(明文) hex；明文不入库';
COMMENT ON COLUMN bot_service_tokens.created_by IS '创建该 token 的管理员 user_id';
COMMENT ON COLUMN bot_service_tokens.expires_at IS '过期时间，NULL=不过期';
COMMENT ON COLUMN bot_service_tokens.revoked_at IS '主动作废时间；NULL=有效，非 NULL=立即失效';
COMMENT ON COLUMN bot_service_tokens.last_used_at IS '最后一次成功鉴权的时间，方便清理冷 token';

CREATE INDEX IF NOT EXISTS idx_bot_service_tokens_revoked_expires
    ON bot_service_tokens (revoked_at, expires_at);
