-- ============================================================================
-- P3.3 order_messages.urgent / urged_at：QQ 加急
--
-- 动机：
--   订单聊天里偶尔会有"对方不看消息"的卡顿场景（卖家不上线 / 买家忘了付款）；
--   引入"加急"机制——把一条订单消息一键由 bot 私聊推到对方 QQ（绑过 QQ 的普通账号
--   或本身就是 QQ 用户的孤儿旗下号）。**严格私聊**：群里 @ 会泄露订单内容，不做群兜底。
--   每条消息独立 urgent 状态：默认 false，加急一次置 true，
--   urged_at 记录时间用于限流（同一对话每 5min 最多加急 1 次）。
--
-- 不引入"加急历史表"——单条订单消息只能加急 1 次（重复加急 = 拒绝），
-- 所以 urgent + urged_at 两个字段足够；查询审计走 service_token_audit
-- + zap log 即可。
--
-- 应用：
--   docker exec -i <hfut-postgres> psql -U postgres -d graduation_project < migrate_order_message_urgent.sql
-- ============================================================================

ALTER TABLE order_messages
    ADD COLUMN IF NOT EXISTS urgent   BOOLEAN     NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS urged_at TIMESTAMPTZ NULL;

COMMENT ON COLUMN order_messages.urgent IS 'P3.3 加急标记：被点过加急的消息会展示红色徽章；同条消息只能加急 1 次';
COMMENT ON COLUMN order_messages.urged_at IS 'P3.3 加急时间戳；用于限流与前端徽章 hover tooltip';
