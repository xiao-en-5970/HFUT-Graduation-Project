-- P5 个人展示页 / 社交关注（B 站风格）：
--   1) users.nickname           展示名（默认空，前端 fallback 到 username）；
--                               旗下号（account_type=2）由 bot 周期同步 QQ 昵称。
--   2) users.bio                个性签名（B 站风格"一句话介绍"）；普通用户可在 app 自填。
--   3) users.qq_avatar_url      旗下号的 QQ 头像 URL（永久 CDN）。如果非空且
--                               users.avatar 为空，则 author / profile 接口对外展示这个；
--                               用户自己上传 avatar 后以 avatar 为准。
--   4) follow                   user_id -> follow_id 关注关系；老表已存在，本次补：
--                               (a) (user_id, follow_id) 唯一索引（防并发重复关注）
--                               (b) follow_id 反向索引（粉丝列表查询）
--                               (c) status 索引（过滤 valid）
--   5) 一次性回填 users.follow_count / users.fans_count 与 follow 表对齐——
--      历史数据可能有累计漂移。
--
-- 设计动机：QQ-bot/skill/bot/SKILL.md "个人展示页"段 + AuthorProfile 注释。
-- QQ 旗下号是"渠道标签"，展示时优先用 QQ 昵称/头像让用户更易识别；普通用户
-- 名字/头像走自己上传。前后端通过 author.nickname + author.avatar 两个字段统一渲染，
-- 不再区分"username 后缀来自"的旧文案。
--
-- 应用：
--   docker exec -i <hfut-postgres 容器名> psql -U postgres -d graduation_project < migrate_profile_social.sql
--
-- 反向迁移：把 ADD COLUMN/CREATE INDEX 换成 DROP COLUMN IF EXISTS / DROP INDEX IF EXISTS。

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS nickname      VARCHAR(64),
    ADD COLUMN IF NOT EXISTS bio           VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS qq_avatar_url VARCHAR(255) NOT NULL DEFAULT '';

COMMENT ON COLUMN users.nickname IS
    '展示名；旗下号由 bot 定期同步 QQ 群名片/昵称，普通用户可自行设置。NULL 时前端 fallback 到 username。';
COMMENT ON COLUMN users.bio IS
    '个性签名/一句话介绍（B 站风格）。空字符串表示用户未填写。';
COMMENT ON COLUMN users.qq_avatar_url IS
    '旗下号的 QQ 头像 CDN URL（如 https://q.qlogo.cn/headimg_dl?...）。avatar 为空时 author 展示用这个。';

-- follow 关系表（旧表已存在）：补关键索引
-- 老 schema 用 nullable int—— uniqueIndex 不能直接套 NULL 行（PG 多个 NULL 视作不等），
-- 历史上未生效。本次直接加 partial unique index：仅在 (user_id, follow_id) 都不为 NULL 时唯一。
CREATE UNIQUE INDEX IF NOT EXISTS uq_follow_user_target
    ON follow (user_id, follow_id)
    WHERE user_id IS NOT NULL AND follow_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_follow_target ON follow (follow_id);
CREATE INDEX IF NOT EXISTS idx_follow_status ON follow (status);

-- 一次性把 users.follow_count / users.fans_count 跟 follow 表对齐——
-- status=1 只算有效关注。
--
-- follow_count = "user 关注了多少人" = WHERE user_id = user.id
-- fans_count   = "user 被多少人关注" = WHERE follow_id = user.id
--
-- 实现：三段 CTE 全部在 UPDATE 主查询的 FROM 子句一级 JOIN，避免 PG 老版本对
-- "CTE 在派生子查询内被引用" 解析报 `missing FROM-clause entry`。
WITH fc AS (SELECT user_id::bigint AS uid, COUNT(*) AS c
            FROM follow
            WHERE status = 1
              AND user_id IS NOT NULL
            GROUP BY user_id),
     fa AS (SELECT follow_id::bigint AS uid, COUNT(*) AS c
            FROM follow
            WHERE status = 1
              AND follow_id IS NOT NULL
            GROUP BY follow_id),
     counts AS (SELECT u.id,
                       COALESCE(fc.c, 0) AS new_follow_count,
                       COALESCE(fa.c, 0) AS new_fans_count
                FROM users u
                         LEFT JOIN fc ON fc.uid = u.id::bigint
                         LEFT JOIN fa ON fa.uid = u.id::bigint)
UPDATE users u
SET follow_count = c.new_follow_count,
    fans_count   = c.new_fans_count
FROM counts c
WHERE u.id = c.id
  AND (u.follow_count IS DISTINCT FROM c.new_follow_count
    OR u.fans_count IS DISTINCT FROM c.new_fans_count);
