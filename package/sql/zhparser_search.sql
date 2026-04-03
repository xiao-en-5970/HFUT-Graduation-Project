-- 中文智能分词全文检索（zhparser）：在已用 create.sql 默认 simple 分词的基础上升级
-- 若未安装 zhparser 或未执行本脚本，全文检索仍可用（english/simple 行为）
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
