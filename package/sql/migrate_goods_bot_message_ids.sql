-- migrate_goods_bot_message_ids.sql
--
-- 为 QQ-bot 上架链路增加"涉及到的 QQ 消息 ID 集合"列，让"用户在群里回复自己的
-- 上架消息说『已出』"能精确定位到具体 good，而不必走模糊匹配 + 反问消歧。
--
-- 字段说明：
--   bot_message_ids BIGINT[]
--     bot 识别到一次 publish_good 时，把这次上架涉及到的全部 QQ message_id（外层
--     消息 ID + Kimi 返回的 image_message_ids / source_message_ids 合集去重）一起
--     写进来。后续 reply 段 data.id 命中数组里任何一个就视为命中这条商品。
--
-- 兼容：旧 goods 行 default '{}'，不影响现有查询；GIN 索引用 ANY 查找 O(1) 命中。

ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS bot_message_ids BIGINT[] NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_goods_bot_msg_ids
    ON goods USING gin (bot_message_ids);

COMMENT ON COLUMN goods.bot_message_ids IS
    'bot 上架时关联的全部 QQ message_id 集合（外层 + 图文段）；用户 reply 这些消息说"已出"时按此反查定位 good';
