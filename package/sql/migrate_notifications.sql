-- 站内通知表
-- type:
--   1 = 点赞了你的帖子/提问/回答/商品
--   2 = 点赞了你的评论
--   3 = 评论了你的帖子/提问/回答/商品（顶层评论）
--   4 = 回复了你的评论
--   5 = 官方通知（from_user_id=0）
-- target_type: 1帖子 2提问 3回答 4商品 5评论
-- target_id:  目标文章/商品/评论的 ID（type=5 官方通知时允许为 0）
-- ref_id:     次级定位 ID（如 target_type=5 评论 → ref_id 记录评论所属文章/商品 ID；
--                           type=4 回复评论 → ref_id 记录所属文章/商品 ID）
-- ref_ext_type: ref_id 的归属 ext_type（与 target_type 同值域）
-- title / summary / image: 页面展示用的冗余字段，防止后续目标被删后消息也无法展示
CREATE TABLE IF NOT EXISTS notifications
(
    id           BIGSERIAL PRIMARY KEY,
    user_id      INTEGER   NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    from_user_id INTEGER   NOT NULL DEFAULT 0, -- 0 为系统/官方
    type         SMALLINT  NOT NULL,
    target_type  SMALLINT  NOT NULL DEFAULT 0,
    target_id    INTEGER   NOT NULL DEFAULT 0,
    ref_ext_type SMALLINT  NOT NULL DEFAULT 0,
    ref_id       INTEGER   NOT NULL DEFAULT 0,
    title        VARCHAR(255)       DEFAULT '',
    summary      VARCHAR(512)       DEFAULT '',
    image        VARCHAR(512)       DEFAULT '',
    is_read      BOOLEAN   NOT NULL DEFAULT FALSE,
    status       SMALLINT  NOT NULL DEFAULT 1, -- 1正常 2禁用
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE notifications IS '站内通知：点赞/评论/回复/官方通知';
COMMENT ON COLUMN notifications.user_id IS '接收者用户 ID';
COMMENT ON COLUMN notifications.from_user_id IS '触发者用户 ID，0 表示系统/官方';
COMMENT ON COLUMN notifications.type IS '1赞作品 2赞评论 3评论 4回复评论 5官方通知';
COMMENT ON COLUMN notifications.target_type IS '1帖子 2提问 3回答 4商品 5评论（与 ext_type 同）';
COMMENT ON COLUMN notifications.ref_ext_type IS '当 target 为评论时，记录评论所属对象的 ext_type';
COMMENT ON COLUMN notifications.ref_id IS '当 target 为评论时，记录评论所属对象的 ID';

-- 列表接口按接收者 + 时间查询
CREATE INDEX IF NOT EXISTS idx_notifications_user_created
    ON notifications (user_id, status, created_at DESC);

-- 未读计数 / 分类未读按 type 过滤
CREATE INDEX IF NOT EXISTS idx_notifications_user_type_read
    ON notifications (user_id, type, is_read);

-- 插入 0 号「官方」用户（若不存在）
-- password 不是合法 bcrypt 哈希，登录比对不会成功，等效于「禁止登录」。
-- status=1 表示账号正常（否则批量 author 查询会把它过滤掉）。
INSERT INTO users (id, username, password, school_id, status, role, avatar, created_at, updated_at)
VALUES (0, '官方', '!locked-cannot-login!', 0, 1, 3 /* 超级管理员 */, '', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- 保证 users 序列不会再分配 0 号 / 或者撞到现有最大 id
-- GREATEST 里 +1 是因为 setval 取 last_value，下次 nextval 会 +1
SELECT setval('users_id_seq', GREATEST((SELECT MAX(id) FROM users), 1));
