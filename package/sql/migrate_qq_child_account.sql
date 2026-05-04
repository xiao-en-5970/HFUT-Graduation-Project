-- ============================================================================
-- QQ 旗下账号 + 商品面议 + 学校 QQ 群映射 + 提问关闭状态
--
-- 配套：app/dao/model/user.go / good.go / school.go / article.go
-- 设计文档：QQ-bot 仓库的 skill/bot/SKILL.md
-- ============================================================================

-- 1) users 表：加 account_type / parent_user_id / qq_number 三个字段
--    旗下账号是 bot 通过 QQ 群消息为某个未注册的 QQ 用户自动创建的"半身份"账号，
--    账号本身不可登录、能力受限；用户后续 app 注册并主动绑定 QQ 后，
--    旗下账号 parent_user_id 设置为主账号 ID，挂载完成。
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS account_type SMALLINT NOT NULL DEFAULT 1;
COMMENT ON COLUMN users.account_type IS '账号类型 1=normal 正常账号(默认) 2=qq_child QQ 旗下账号(不可登录)';

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS parent_user_id INTEGER REFERENCES users (id);
COMMENT ON COLUMN users.parent_user_id IS '主账号 ID；旗下账号挂在哪个主账号下，孤儿(未绑主账号)与正常账号为 NULL';
CREATE INDEX IF NOT EXISTS idx_users_parent_user_id
    ON users (parent_user_id) WHERE parent_user_id IS NOT NULL;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS qq_number VARCHAR(32);
COMMENT ON COLUMN users.qq_number IS 'QQ 旗下账号绑定的 QQ 号；正常账号此字段为 NULL（注意区别于 bind_qq——bind_qq 是主账号自报的 QQ；qq_number 是旗下账号绑定的 QQ）';

-- 同一 QQ 号在 status=1 范围内只允许有一个有效旗下账号；旧的 status<>1 的不冲突，方便软删后重建
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_qq_number_active
    ON users (qq_number) WHERE qq_number IS NOT NULL AND status = 1;

-- ============================================================================

-- 2) goods 表：加 negotiable 面议标记
--    保留原 price 字段（int 分），negotiable=true 时 price 字段被业务层忽略，前端展示"面议"
ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS negotiable BOOLEAN NOT NULL DEFAULT FALSE;
COMMENT ON COLUMN goods.negotiable IS '面议标记；TRUE 时 price 字段被忽略，前端展示"面议"';

-- ============================================================================

-- 3) schools 表：加 qq_groups 字段，记录映射到该学校的 QQ 群号
--    bot 在这些群里识别到的"未注册 QQ 用户"会自动创建旗下账号、归属该学校
ALTER TABLE schools
    ADD COLUMN IF NOT EXISTS qq_groups BIGINT[] NOT NULL DEFAULT '{}';
COMMENT ON COLUMN schools.qq_groups IS '映射到该学校的 QQ 群号列表；bot 在这些群里识别到 QQ 用户后会创建归属本校的旗下账号';

-- ============================================================================

-- 4) articles 表：复用 status 字段表达"已关闭"
--    现有 status: 1=正常 2=禁用 3=草稿
--    新增取值: 4=已关闭（用户主动关闭提问，停止接受回答）
--    不需要改字段类型，仅业务层认这个值；这里更新注释方便 DBA / SRE 排查
COMMENT ON COLUMN articles.status IS '状态 1=正常 2=禁用 3=草稿 4=已关闭(提问主动关闭)';
