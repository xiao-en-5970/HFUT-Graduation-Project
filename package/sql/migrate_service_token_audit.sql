-- ============================================================================
-- P3.4 service_token_audit：bot → hfut 服务调用的审计日志
--
-- 动机：
--   现状只有 zap log 能看到 bot 调 hfut 的请求，定位攻击 / 误用比较麻烦。
--   把每次 X-Bot-Service-Token 通过 BotServiceAuth 中间件的请求落到一张表里，
--   方便后续按 service / jti / 时间窗口审计与排障。
--
-- 表语义：
--   每行 = 一次成功通过 bot service auth 的请求；middleware 异步写，不影响请求链路。
--   auth 失败的请求**不写**（401 已在 ZapLogger 里完整记录）。
--
-- 体量预估：
--   bot 一群 = 5~50 次/分钟（峰值），单天 ~5w 行。30 天 ~150w 行——可接受；
--   后续可加 cron 任务按 created_at 删除 30 天前数据（同 OSS 镜像清理一并做）。
--
-- 应用：
--   docker exec -i <hfut-postgres> psql -U postgres -d graduation_project < migrate_service_token_audit.sql
-- ============================================================================

CREATE TABLE IF NOT EXISTS service_token_audit
(
    id          BIGSERIAL PRIMARY KEY,
    service     VARCHAR(64)  NOT NULL,
    jti         VARCHAR(64)  NOT NULL,
    method      VARCHAR(8)   NOT NULL,
    path        VARCHAR(255) NOT NULL,
    status_code INT          NOT NULL,
    remote_ip   VARCHAR(64),
    duration_ms INT          NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_service_token_audit_service_created
    ON service_token_audit (service, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_service_token_audit_jti
    ON service_token_audit (jti);

CREATE INDEX IF NOT EXISTS idx_service_token_audit_created
    ON service_token_audit (created_at DESC);

COMMENT ON TABLE service_token_audit IS 'bot service-to-service 调用审计日志（详见 QQ-bot/skill/bot/SKILL.md "P3.4 限流/审计"）';
COMMENT ON COLUMN service_token_audit.service IS 'bot 自报的服务名，来自 X-Bot-Service-Token JWT 的 service 字段';
COMMENT ON COLUMN service_token_audit.jti IS 'JWT jti——bot 每次请求随机生成，跨日志串起一次端到端调用';
COMMENT ON COLUMN service_token_audit.status_code IS '响应 HTTP status code（middleware Next() 后采集）';
COMMENT ON COLUMN service_token_audit.duration_ms IS '本次请求耗时（毫秒）';
