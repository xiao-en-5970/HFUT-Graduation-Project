-- 迁移：为 likes 表添加联合唯一约束，保证幂等性
-- 若 create.sql 已包含新结构，无需执行
-- 若有重复数据 (user_id, ext_id, ext_type)，需先清理后再执行

CREATE UNIQUE INDEX IF NOT EXISTS uk_likes_user_ext ON likes (user_id, ext_id, ext_type);
