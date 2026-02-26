-- 中文智能分词全文检索迁移（从 simple 升级到 chinese_zh）
-- 若 create.sql 已使用 chinese_zh，则无需执行此脚本
-- 依赖: zhparser 扩展，安装见 doc/ZHPARSER_SETUP.md

CREATE EXTENSION IF NOT EXISTS zhparser;
DROP TEXT SEARCH CONFIGURATION IF EXISTS chinese_zh;
CREATE TEXT SEARCH CONFIGURATION chinese_zh (PARSER = zhparser);
ALTER TEXT SEARCH CONFIGURATION chinese_zh ADD MAPPING FOR n,v,a,i,e,l,j WITH simple;

DROP INDEX IF EXISTS idx_articles_search;
ALTER TABLE articles
    DROP COLUMN IF EXISTS search_vector;
ALTER TABLE articles
    ADD COLUMN search_vector tsvector
        GENERATED ALWAYS AS (
            setweight(to_tsvector('chinese_zh', coalesce(title, '')), 'A') ||
            setweight(to_tsvector('chinese_zh', coalesce(content, '')), 'B')
            ) STORED;
CREATE INDEX idx_articles_search ON articles USING GIN (search_vector);
