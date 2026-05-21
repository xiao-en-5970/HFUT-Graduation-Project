-- migrate_bot_runtime_config.sql
--
-- 把 QQ-bot 运行时"群相关配置"从 bot 端 env 搬到 hfut DB，由管理后台编辑。
--
-- 设计：单张 KV 表（key 唯一，value 是 jsonb），将来加配置项不用改 schema。
-- bot 启动 + 每 60s 调 GET /api/v1/bot/runtime-config 一次性拉全部 keys 进内存，
-- 内存里用 atomic.Pointer 替换，热生效；env 仍作为兜底（拉不到时用 env 值）。
--
-- 已支持的 keys（value 类型见 default）：
--
--   auto_reply_whitelist  jsonb array of int64  启用自动监听的 QQ 群号列表
--   ops_group_ids         jsonb array of int64  运维通知 / 运维查询群（NotifyOps* / @bot 提问）
--   silent_mode           jsonb bool            灰度静默：非运维群 + 私聊全部静默不发
--
-- 学校认证群（schools.qq_groups）原本就在 DB 里，不进 bot_runtime_config，
-- 但本迁移顺便给学校列表添 qq_groups 字段提示——admin UI 这次会补上编辑入口。

CREATE TABLE IF NOT EXISTS bot_runtime_config
(
    key        VARCHAR(64) PRIMARY KEY,
    value      JSONB     NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by INTEGER REFERENCES users (id),
    comment    TEXT
);

COMMENT ON TABLE bot_runtime_config IS 'QQ-bot 运行时配置 KV 表，由管理后台维护，bot 周期拉取';
COMMENT ON COLUMN bot_runtime_config.key IS '配置项名（snake_case）';
COMMENT ON COLUMN bot_runtime_config.value IS 'JSON 值；数组 / 布尔 / 数字均可，由 bot 侧按 key 解读';
COMMENT ON COLUMN bot_runtime_config.updated_by IS '最近一次修改者；NULL=种子或迁移写入';
COMMENT ON COLUMN bot_runtime_config.comment IS '面向 admin 的字段说明（admin UI 表单提示）';

-- 种子数据：默认空白名单 / 仅一个内置运维群 / 关静默
INSERT INTO bot_runtime_config (key, value, comment)
VALUES ('auto_reply_whitelist', '[]'::jsonb,
        '启用自动监听的 QQ 群号列表。空列表 = bot 不在任何群里自动识别，仅响应 @bot 命令'),
       ('ops_group_ids', '[
         1084352497
       ]'::jsonb,
        '运维通知 / 运维查询群。bot 上架成功 / 群接入申请等通知会广播；群内 @bot 会进运维 SQL 查询路径'),
       ('silent_mode', 'false'::jsonb,
        '灰度静默模式。开启后非运维群的群消息、所有私聊、群文件上传都被静默；运维群 + hfut 落库照常')
ON CONFLICT (key) DO NOTHING;
