-- ============================================================================
-- bot_service_tokens 表已废弃。
--
-- 历史：原计划在数据库里维护 service-to-service 鉴权 token 元数据（创建/作废/列出），
-- 让 admin 在 web 后台管理。
--
-- 现状：方案改为"共享 secret + bot 自签短期 JWT"——bot 跟 hfut 都拿同一个 HS256 secret，
-- bot 每次请求自签 60s 有效期 JWT，hfut 端用同一 secret 验签即放行，**不需要 DB 维护**。
-- 旋转 secret = 让所有旧 token 失效；不再有 admin 创建/作废 token 的概念。
--
-- 详细设计见 QQ-bot 仓库 skill/bot/SKILL.md。
--
-- 这条 migration 把废弃表删掉，幂等：表不存在也不报错。
-- ============================================================================

DROP TABLE IF EXISTS bot_service_tokens;
