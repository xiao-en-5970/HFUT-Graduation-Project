-- ============================================================================
-- wipe_business_data.sql
--
-- 上线前删档脚本：清空所有用户产生的业务数据，重置自增 ID，保留：
--   - schools 整张表（学校元数据、QQ 群映射、登录配置）
--   - users 表中 role IN (2,3) 的管理员 / 超级管理员账号
--   - users 表中 username = '__order_official__' 的系统订单官方账号
--
-- ----------------------------------------------------------------------------
-- 用法（容器外）：
--     psql "postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME" \
--          -v ON_ERROR_STOP=1 -f wipe_business_data.sql
--
-- 用法（在 docker compose 里跑 postgres 服务）：
--     cat package/sql/wipe_business_data.sql | docker exec -i <pg-container> \
--          psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1
--
-- ----------------------------------------------------------------------------
-- 操作语义：
--   - 整段包在 BEGIN…COMMIT 事务里，失败自动回滚，数据不留半截
--   - 业务子表用 TRUNCATE ... RESTART IDENTITY CASCADE 一次性清空 + 复位序列
--   - users 用 DELETE WHERE （保留管理员 + 系统账号），再 setval 复位序列到
--     当前最大 id + 1，避免后续注册时 id 跳号
--   - schools 不动，所有 schools.id 引用仍然合法
--
-- 安全提示：
--   - 跑之前**务必备份**：pg_dump > backup.sql
--   - 这个脚本会**清除所有用户、所有帖子、所有商品、所有订单、所有指标历史**，
--     不可逆。在生产库跑前确认你真的在演练前的"灰度准备"阶段。
-- ============================================================================

BEGIN;

-- 显式锁住要清的表，避免业务进程并发写入（开发环境单实例可忽略）
LOCK TABLE users IN ACCESS EXCLUSIVE MODE;

-- ----------------------------------------------------------------------------
-- 1) 业务子表：TRUNCATE + RESTART IDENTITY + CASCADE
--
-- TRUNCATE 比逐张 DELETE 快几个数量级（不走 WAL 单行记录），同时 RESTART IDENTITY
-- 把每张表的自增 SERIAL/IDENTITY 序列重置回 1。CASCADE 自动级联清掉任何外键
-- 引用——不需要按 FK 拓扑顺序手动排列，PostgreSQL 自动处理。
--
-- 列出来的所有表都是"用户产生 + 运维生成"的内容，wipe 后应该是初始空状态。
-- ----------------------------------------------------------------------------
TRUNCATE TABLE
    -- 订单链
    order_message_reads,
    order_messages,
    orders,
    -- 推荐 / 行为
    user_behaviors,
    user_locations,
    -- 收藏
    collect_item,
    collect,
    -- 内容互动
    likes,
    comments,
    -- 内容主体（articles 自引用 parent_id，靠 CASCADE 一次性清掉）
    articles,
    -- 社交
    follow,
    -- 通知
    notifications,
    -- 标签
    tags,
    -- 商品
    goods,
    -- 学校认证（注意：这是用户对学校的认证，不是 schools 自身）
    user_cert,
    -- 指标 / 事件 / 审计
    metric_minute,
    bot_dispatch_event,
    service_token_audit
    RESTART IDENTITY CASCADE;

-- ----------------------------------------------------------------------------
-- 2) users：保留管理员（role=2/3）+ 系统订单账号
--
-- 旗下账号 (parent_user_id) 自引用：因为单条 DELETE 一次性把所有匹配行都标记删除，
-- FK 约束在语句结束时统一校验，旗下号 + 主号会一起消失，不会出现"指向不存在主号"
-- 的中间态。
-- ----------------------------------------------------------------------------
DELETE
FROM users
WHERE role NOT IN (2, 3)
  AND username <> '__order_official__';

-- 重置 users.id 序列到剩余行最大 id 之后；setval 第三参 false 表示"下一次
-- nextval 直接返回这个值"，避开"max(id) 已被占用 → 新插入冲突"的边界
SELECT setval(
               pg_get_serial_sequence('users', 'id'),
               COALESCE((SELECT MAX(id) FROM users), 0) + 1,
               false
       );

-- ----------------------------------------------------------------------------
-- 3) sanity check —— 打印剩余行数；用于人眼确认 wipe 完整
-- ----------------------------------------------------------------------------
SELECT 'users_keep' AS table_name, COUNT(*) AS rows
FROM users
UNION ALL
SELECT 'schools_keep', COUNT(*)
FROM schools
UNION ALL
SELECT 'articles', COUNT(*)
FROM articles
UNION ALL
SELECT 'comments', COUNT(*)
FROM comments
UNION ALL
SELECT 'likes', COUNT(*)
FROM likes
UNION ALL
SELECT 'goods', COUNT(*)
FROM goods
UNION ALL
SELECT 'tags', COUNT(*)
FROM tags
UNION ALL
SELECT 'collect', COUNT(*)
FROM collect
UNION ALL
SELECT 'collect_item', COUNT(*)
FROM collect_item
UNION ALL
SELECT 'follow', COUNT(*)
FROM follow
UNION ALL
SELECT 'orders', COUNT(*)
FROM orders
UNION ALL
SELECT 'order_messages', COUNT(*)
FROM order_messages
UNION ALL
SELECT 'order_message_reads', COUNT(*)
FROM order_message_reads
UNION ALL
SELECT 'notifications', COUNT(*)
FROM notifications
UNION ALL
SELECT 'user_cert', COUNT(*)
FROM user_cert
UNION ALL
SELECT 'user_behaviors', COUNT(*)
FROM user_behaviors
UNION ALL
SELECT 'user_locations', COUNT(*)
FROM user_locations
UNION ALL
SELECT 'metric_minute', COUNT(*)
FROM metric_minute
UNION ALL
SELECT 'bot_dispatch_event', COUNT(*)
FROM bot_dispatch_event
UNION ALL
SELECT 'service_token_audit', COUNT(*)
FROM service_token_audit
ORDER BY table_name;

COMMIT;

-- ----------------------------------------------------------------------------
-- 结束。预期输出：
--   users_keep            = 管理员数 + 1（__order_official__）
--   schools_keep          = 当前已配置的学校数（>=1）
--   其它所有表             = 0
-- 如果其它表 > 0，说明有未列入 TRUNCATE 的子表或种子数据被 CASCADE 错过——
-- 不正常，回滚检查。
-- ============================================================================
