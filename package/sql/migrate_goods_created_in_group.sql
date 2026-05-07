-- P2c.3：给 goods 表加 created_in_group_id —— 商品被 bot 通过 QQ 群上架时所在的群号。
--
-- 动机：
--   原本 RequestOffShelfFromOrphan（孤儿商品 "请求下架"）只能用 users.created_in_group_id
--   定位"在哪个群 @ 卖家"。但有几个问题：
--     1) 存量孤儿 users.created_in_group_id 为 NULL，前端调用直接报"商品创建群信息缺失"
--     2) 用户可能在群 A 注册旗下号、之后在群 B 实际发了商品——卖家在群 B 而不是群 A 活跃
--   解决：商品本身记录"创建于哪个群"，比 user.first-seen 更精准，且新数据不会缺。
--
-- 字段语义：
--   created_in_group_id  bot 上架商品时来源 QQ 群号；非 bot 路径（管理员 / app 用户）发的商品 NULL。
--                        请求下架时优先用本字段，缺失时再回退到 owner.created_in_group_id，
--                        都缺失时 fallback 到 SendPrivate（如 bot 是该 QQ 好友）。
--
-- 数据迁移：
--   ALTER 后所有历史商品 created_in_group_id=NULL；新 bot 上架的商品自动写入。
--   旧数据无须回填——RequestOffShelfFromOrphan 三级 fallback 设计已经能应付历史数据。
--
-- 应用：
--   docker exec -i <hfut-postgres 容器名> psql -U postgres -d graduation_project < migrate_goods_created_in_group.sql

ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS created_in_group_id BIGINT NULL;

COMMENT ON COLUMN goods.created_in_group_id IS
    'bot 通过 QQ 群上架本商品时所在的群号；非 bot 路径上架的商品 NULL。请求下架时优先用本字段定位 @ 卖家位置（详见 QQ-bot/skill/bot/orphan.md "请求下架"段）';
