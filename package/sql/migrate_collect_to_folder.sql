-- 迁移脚本：将旧版 collect 表升级为收藏夹 + collect_item
-- 若数据库为新建（create.sql 已包含新结构），无需执行此脚本
-- 若有重要收藏数据，请先备份

-- 1. 新建 collect_item 表
CREATE TABLE IF NOT EXISTS collect_item (
    id SERIAL PRIMARY KEY,
    collect_id integer NOT NULL REFERENCES collect(id) ON DELETE CASCADE,
    ext_id integer NOT NULL,
    ext_type integer NOT NULL DEFAULT 1,
    status smallint NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(collect_id, ext_id, ext_type)
);
comment on table collect_item is '收藏表，收藏夹中的具体收藏项';
comment on column collect_item.ext_type is '关联类型 1:帖子 2:提问 3:回答 4:商品';

-- 2. 修改 collect 为收藏夹结构（需先迁移旧数据到 collect_item 或清空 collect）
ALTER TABLE collect ADD COLUMN IF NOT EXISTS name VARCHAR(100) DEFAULT '默认';
ALTER TABLE collect ADD COLUMN IF NOT EXISTS is_default BOOLEAN DEFAULT false;
-- 有旧数据时需先备份并处理，再执行：
-- ALTER TABLE collect DROP COLUMN IF EXISTS ext_id;
-- ALTER TABLE collect DROP COLUMN IF EXISTS ext_type;
