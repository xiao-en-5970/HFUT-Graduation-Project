-- 用户行为采集表：用于个性化推荐（兴趣画像 + 去重已浏览）
-- ext_type: 1帖子 2提问 3回答 4商品
-- action:   1=view(浏览) 2=like 3=unlike 4=collect 5=uncollect 6=comment 7=search
-- weight:   动作权重（推荐系统聚合画像时使用），不同 action 预设不同默认值
-- keyword:  仅 action=7 (search) 时使用；存搜索关键词用于未来扩展
CREATE TABLE IF NOT EXISTS user_behaviors
(
    id         BIGSERIAL PRIMARY KEY,
    user_id    INTEGER  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    ext_type   SMALLINT NOT NULL,
    ext_id     INTEGER  NOT NULL DEFAULT 0,
    action     SMALLINT NOT NULL,
    weight     REAL     NOT NULL DEFAULT 1.0,
    keyword    VARCHAR(128),
    created_at TIMESTAMP         DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE user_behaviors IS '用户行为流水，用于个性化推荐的画像构建与去重';
COMMENT ON COLUMN user_behaviors.ext_type IS '1帖子 2提问 3回答 4商品';
COMMENT ON COLUMN user_behaviors.action IS '1=view 2=like 3=unlike 4=collect 5=uncollect 6=comment 7=search';
COMMENT ON COLUMN user_behaviors.weight IS '该行为的分数权重（示例：view=1 like=5 collect=8 comment=3 search=2）';
COMMENT ON COLUMN user_behaviors.keyword IS '仅 action=7 search 时使用';

-- 用户近 N 天画像聚合：按 user_id + created_at DESC 扫描
CREATE INDEX IF NOT EXISTS idx_user_behaviors_user_created
    ON user_behaviors (user_id, created_at DESC);

-- 快速判断「用户是否浏览过某内容」（反向去重 / 画像构建中 item 聚合）
CREATE INDEX IF NOT EXISTS idx_user_behaviors_user_ext
    ON user_behaviors (user_id, ext_type, ext_id);
