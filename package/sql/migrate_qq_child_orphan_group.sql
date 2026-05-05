-- P2c.1：给 QQ 旗下账号持久化"创建群"——孤儿账号被回复时 bot 把 app 端的回复
-- 转发回创建群里那个 QQ 用户。
--
-- 字段语义：
--   created_in_group_id  bot 第一次在该群里见到这个 QQ 用户时所在的群号；
--                        旗下号一旦被绑回主账号、所有 inbound 走 P2b 重定向，
--                        这个字段就不再被使用——但保留作为审计 / 万一解绑后又
--                        变孤儿时仍能继续用。
--   仅 account_type=2（QQ 旗下号）时填；普通账号 NULL。
--
-- 数据迁移：
--   字段 ALTER 后，所有已存在的旗下账号 created_in_group_id = NULL；
--   下次 bot 调 BotUpsertQQChild 时不会写入（因为已有记录走 update path 不写）；
--   这意味着**改动前已经存在的孤儿**仍然没有创建群信息——它们的 inbound 通知
--   会被 P2c.2 跳过转发（fallback 到 dropped + log warn），新创建的旗下号不受影响。
--
-- 应用：
--   docker exec -i <hfut-postgres 容器名> psql -U postgres -d graduation_project < migrate_qq_child_orphan_group.sql

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS created_in_group_id BIGINT NULL;

COMMENT ON COLUMN users.created_in_group_id IS
    'QQ 旗下账号被 bot 创建时所在的 QQ 群号；普通账号为 NULL。用于孤儿 inbound 通知转发回原群（详见 QQ-bot/skill/bot/SKILL.md "孤儿旗下账号特殊行为"）';
