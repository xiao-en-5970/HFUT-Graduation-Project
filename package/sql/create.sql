CREATE TABLE users (
                       id SERIAL PRIMARY KEY ,
                       username VARCHAR(50) NOT NULL UNIQUE ,
                       password VARCHAR(255) NOT NULL ,
                       school_id integer REFERENCES schools(id) ,
                       bind_qq VARCHAR(128) ,
                       bind_wx VARCHAR(128) ,
                       bind_phone VARCHAR(20),
                       status smallint DEFAULT 1 ,
                       role smallint DEFAULT 1 ,
                        avatar VARCHAR(255)  ,
                        background VARCHAR(255) ,
                       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                       updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

comment on column users.role is '1:普通用户 2:管理员 3:超级管理员 4:匿名用户';
comment on column users.status is '1:正常 2:禁用';
comment on column users.avatar is '用户头像';
comment on column users.background is '用户背景';
comment on column users.bind_phone is '绑定的手机号';
comment on column users.bind_qq is '绑定的QQ号';
comment on column users.bind_wx is '绑定的微信号';
comment on column users.school_id is '学校ID';
comment on column users.username is '用户名';
comment on column users.password is '密码';

ALTER TABLE users ADD COLUMN IF NOT EXISTS follow_count integer NOT NULL DEFAULT 0;
comment on column users.follow_count is '关注数量';
ALTER TABLE users ADD COLUMN IF NOT EXISTS fans_count integer NOT NULL DEFAULT 0;
comment on column users.fans_count is '粉丝数量';

CREATE TABLE schools (
                         id SERIAL PRIMARY KEY,
                         name VARCHAR(50),
                         created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                         updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                         login_url varchar(255),
                         user_count integer default 0
);

comment on column schools.name is '学校名称';
comment on column schools.login_url is '登录地址';
comment on column schools.user_count is '用户数量';

-- 插入 id=0 占位行，供 users.school_id=0 表示「未绑定」使用（FK 约束需要）
INSERT INTO schools (id, name) VALUES (0, '未绑定') ON CONFLICT (id) DO NOTHING;
SELECT setval(pg_get_serial_sequence('schools', 'id'), (SELECT COALESCE(MAX(id), 1) FROM schools));

create table articles (
            id SERIAL PRIMARY KEY,
            user_id integer REFERENCES users(id),
            title VARCHAR(255) NOT NULL,
            content TEXT NOT NULL,
            status smallint not null DEFAULT 1,
            publish_status smallint not null DEFAULT 1,
            type int not null DEFAULT 1,
            view_count integer not null DEFAULT 0,
            like_count integer not null DEFAULT 0,
            collect_count integer not null DEFAULT 0,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

comment on column articles.user_id is '用户ID';
comment on column articles.title is '文章标题';
comment on column articles.content is '文章内容';
comment on column articles.status is '1:正常 2:禁用 3:草稿';
comment on column articles.publish_status is '1:私密 2:公开';
comment on column articles.type is '1:普通文章 2:提问 3:回答';
comment on column articles.view_count is '浏览次数';
comment on column articles.like_count is '点赞/同问次数';
comment on column articles.collect_count is '收藏次数';


ALTER TABLE articles ADD COLUMN IF NOT EXISTS images varchar(255)[];
comment on column articles.images is '图片数组';


ALTER TABLE articles ADD COLUMN IF NOT EXISTS image_count integer NOT NULL DEFAULT 0;
comment on column articles.image_count is '图片数量';


create table comments (
            id SERIAL PRIMARY KEY,
            user_id integer REFERENCES users(id),
            ext_type integer not null DEFAULT 1,
            ext_id integer not null ,
            parent_id integer REFERENCES comments(id),
            reply_id integer REFERENCES comments(id),
            images varchar(255)[],
            type integer not null DEFAULT 1,
            content TEXT NOT NULL,
            status smallint not null DEFAULT 1,
            like_count integer not null DEFAULT 0,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

comment on column comments.user_id is '用户ID';
comment on column comments.ext_type is '关联类型 1:帖子 2:提问 3:回答 4:商品';
comment on column comments.ext_id is '关联ID';
comment on column comments.parent_id is '父评论ID';
comment on column comments.reply_id is '回复评论ID';
comment on column comments.images is '图片数组';
comment on column comments.type is '1:顶层评论 2:评论回复';
comment on column comments.content is '评论内容';
comment on column comments.status is '1:正常 2:禁用';
comment on column comments.like_count is '点赞次数';

create table likes (
            id SERIAL PRIMARY KEY,
            user_id integer REFERENCES users(id),
            ext_id integer not null,
            ext_type integer not null DEFAULT 1,
            images varchar(255)[],
            status smallint not null DEFAULT 1,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            UNIQUE(user_id, ext_id, ext_type)
);

comment on column likes.user_id is '用户ID';
comment on column likes.ext_id is '关联ID';
comment on column likes.ext_type is '关联类型 1:帖子 2:提问 3:回答 4:商品 5:评论';
comment on column likes.images is '图片数组';
comment on column likes.status is '1:正常 2:禁用';

create table goods (
            id SERIAL PRIMARY KEY,
            user_id integer REFERENCES users(id),
            title VARCHAR(255) NOT NULL,
            images varchar(255)[],
            content TEXT NOT NULL,
            status smallint not null DEFAULT 1,
            good_status int not null DEFAULT 1,
            price integer not null DEFAULT 0,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


comment on column goods.user_id is '用户ID';
comment on column goods.title is '商品名称';
comment on column goods.images is '图片数组';
comment on column goods.content is '商品内容';
comment on column goods.status is '1:正常 2:禁用';
comment on column goods.good_status is '1:在售 2:下架 3:已售出';
comment on column goods.price is '商品价格，单位分';

-- 添加库存数量字段
ALTER TABLE goods ADD COLUMN IF NOT EXISTS stock integer NOT NULL DEFAULT 0;
comment on column goods.stock is '库存数量';

ALTER TABLE goods ADD COLUMN IF NOT EXISTS end_time integer NOT NULL DEFAULT 0;
ALTER TABLE goods ADD COLUMN IF NOT EXISTS start_time integer NOT NULL DEFAULT 0;
comment on column goods.start_time is '开始时间';
comment on column goods.end_time is '结束时间';

ALTER TABLE goods ADD COLUMN IF NOT EXISTS marked_price integer NOT NULL DEFAULT 0;
comment on column goods.marked_price is '标价，单位分';

ALTER TABLE goods ADD COLUMN IF NOT EXISTS image_count integer NOT NULL DEFAULT 0;
comment on column goods.image_count is '图片数量';

ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS like_count integer NOT NULL DEFAULT 0;
ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS collect_count integer NOT NULL DEFAULT 0;
comment on column goods.like_count is '点赞次数';
comment on column goods.collect_count is '收藏次数';

create table tags (
            id SERIAL PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            ext_id integer not null,
            ext_type integer not null DEFAULT 1,
            status smallint not null DEFAULT 1,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

comment on column tags.name is '标签名称';
comment on column tags.ext_id is '关联ID';
comment on column tags.ext_type is '关联类型 1:articles 2:goods';
comment on column tags.status is '1:正常 2:禁用';


-- 收藏夹：不区分类型，可混合收藏帖子/提问/回答/商品
create table collect (
            id SERIAL PRIMARY KEY,
            user_id integer REFERENCES users(id),
            name VARCHAR(100) NOT NULL DEFAULT '默认',
            is_default boolean NOT NULL DEFAULT false,
            status smallint NOT NULL DEFAULT 1,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

comment on table collect is '收藏夹表，不区分类型，一个收藏夹可混合收藏多种内容';
comment on column collect.user_id is '用户ID';
comment on column collect.name is '收藏夹名称';
comment on column collect.is_default is '是否默认收藏夹，每用户一个';
comment on column collect.status is '1:正常 2:禁用';

-- 收藏表：关联收藏夹，每条收藏需标明类型（帖子/提问/回答/商品）
create table collect_item (
            id SERIAL PRIMARY KEY,
            collect_id integer NOT NULL REFERENCES collect(id) ON DELETE CASCADE,
            ext_id integer NOT NULL,
            ext_type integer NOT NULL DEFAULT 1,
            status smallint NOT NULL DEFAULT 1,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            UNIQUE(collect_id, ext_id, ext_type)
);

comment on table collect_item is '收藏表，关联收藏夹，每条收藏标明类型便于筛选展示';
comment on column collect_item.collect_id is '收藏夹ID';
comment on column collect_item.ext_id is '关联目标ID（如文章ID、商品ID）';
comment on column collect_item.ext_type is '类型 1:帖子 2:提问 3:回答 4:商品';
comment on column collect_item.status is '1:正常 2:禁用';

CREATE INDEX IF NOT EXISTS idx_collect_item_collect_ext ON collect_item(collect_id, ext_type);

create table follow (
            id SERIAL PRIMARY KEY,
            user_id integer REFERENCES users(id),
            follow_id integer REFERENCES users(id),
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

comment on column follow.user_id is '用户ID';
comment on column follow.follow_id is '关注用户ID';

create table orders (
            id SERIAL PRIMARY KEY,
            user_id integer REFERENCES users(id),
            goods_id integer REFERENCES goods(id),
            status smallint not null DEFAULT 1,
            order_status smallint not null DEFAULT 1,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

comment on column orders.user_id is '用户ID';
comment on column orders.goods_id is '商品ID';
comment on column orders.status is '1:正常 2:禁用';
comment on column orders.order_status is '1:待下单 2:正在派送 3:待买方确认收货 4:已完成 5:已取消（平台不经手资金）';

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS receiver_addr VARCHAR(512);
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS sender_addr VARCHAR(512);
COMMENT ON COLUMN orders.receiver_addr IS '收货地址';
COMMENT ON COLUMN orders.sender_addr IS '发货地址';

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS distance_meters integer;
COMMENT ON COLUMN orders.distance_meters IS '发货地与收货地步行规划距离（米），高德地图 API 计算';

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS receiver_lat DOUBLE PRECISION;
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS receiver_lng DOUBLE PRECISION;
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS sender_lat DOUBLE PRECISION;
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS sender_lng DOUBLE PRECISION;
COMMENT ON COLUMN orders.receiver_lat IS '收货地图选点纬度 GCJ-02';
COMMENT ON COLUMN orders.receiver_lng IS '收货地图选点经度 GCJ-02';
COMMENT ON COLUMN orders.sender_lat IS '发货地图选点纬度 GCJ-02';
COMMENT ON COLUMN orders.sender_lng IS '发货地图选点经度 GCJ-02';

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS buyer_agreed_at TIMESTAMP;
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS seller_agreed_at TIMESTAMP;
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS delivery_images VARCHAR(2048)[];
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS buyer_confirm_images VARCHAR(2048)[];
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS completed_at TIMESTAMP;
COMMENT ON COLUMN orders.buyer_agreed_at IS '买方同意开始线下交易/派送';
COMMENT ON COLUMN orders.seller_agreed_at IS '卖方同意开始派送';
COMMENT ON COLUMN orders.delivery_images IS '卖方确认送达时上传的凭证图 URL';
COMMENT ON COLUMN orders.buyer_confirm_images IS '买方确认收货时附加图 URL';
COMMENT ON COLUMN orders.completed_at IS '订单完成（确认收货）时间';

CREATE TABLE IF NOT EXISTS order_messages
(
    id         SERIAL PRIMARY KEY,
    order_id   INTEGER  NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    sender_id  INTEGER  NOT NULL REFERENCES users (id),
    msg_type   SMALLINT NOT NULL DEFAULT 1,
    content    TEXT,
    image_url  VARCHAR(1024),
    created_at TIMESTAMP         DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_order_messages_order_id ON order_messages (order_id);
COMMENT ON TABLE order_messages IS '订单内买卖双方聊天，不经手资金';
COMMENT ON COLUMN order_messages.msg_type IS '1:文字 2:图片';

-- 为没有 status 字段的表添加 status 字段
ALTER TABLE schools ADD COLUMN IF NOT EXISTS status smallint NOT NULL DEFAULT 1;
comment on column schools.status is '1:正常 2:禁用';


ALTER TABLE follow ADD COLUMN IF NOT EXISTS status smallint NOT NULL DEFAULT 1;
comment on column follow.status is '1:正常 2:禁用';

ALTER TABLE goods ADD COLUMN IF NOT EXISTS school_id integer REFERENCES schools(id);
comment on column goods.school_id is '学校ID';

ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS goods_type smallint NOT NULL DEFAULT 1;
ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS pickup_addr VARCHAR(512);
COMMENT ON COLUMN goods.goods_type IS '1:送货上门 2:自提 3:在线商品';
COMMENT ON COLUMN goods.pickup_addr IS '与 goods_addr 同步；自提类约定提货点，下单时可作为默认收货地址';

ALTER TABLE goods
    ADD COLUMN IF NOT EXISTS goods_addr VARCHAR(512);
COMMENT ON COLUMN goods.goods_addr IS '商品地址：发货地/自提点，用于默认卖方发货地址与自提说明（与 pickup_addr 写入时同步）';

ALTER TABLE articles ADD COLUMN IF NOT EXISTS school_id integer REFERENCES schools(id);
comment on column articles.school_id is '学校ID';

ALTER TABLE articles ADD COLUMN IF NOT EXISTS parent_id integer REFERENCES articles(id);
comment on column articles.parent_id is '父文章ID，仅回答类型(type=3)使用，指向提问';

-- 全文检索：tsvector 倒排索引（基于 simple 分词；需中文分词时安装 zhparser 后执行 package/sql/zhparser_search.sql）
DROP TEXT SEARCH CONFIGURATION IF EXISTS chinese_zh;
CREATE TEXT SEARCH CONFIGURATION chinese_zh (COPY = pg_catalog.simple);

-- search_vector：标题权重 A，正文权重 B
ALTER TABLE articles
    DROP COLUMN IF EXISTS search_vector;
ALTER TABLE articles
    ADD COLUMN search_vector tsvector
        GENERATED ALWAYS AS (
            setweight(to_tsvector('chinese_zh', coalesce(title, '')), 'A') ||
            setweight(to_tsvector('chinese_zh', coalesce(content, '')), 'B')
            ) STORED;
CREATE INDEX IF NOT EXISTS idx_articles_search ON articles USING GIN (search_vector);

-- 学校表增加 code 字段，用于对接 package/schools 登录模块（如 hfut）
ALTER TABLE schools
    ADD COLUMN IF NOT EXISTS code VARCHAR(32) UNIQUE;
COMMENT ON COLUMN schools.code IS '学校代码，如 hfut，用于 school-login';

-- 学校表单配置：form_fields 需填字段，captcha_url 验证码图片获取地址（空则用后端 GET /schools/:id/captcha）
ALTER TABLE schools ADD COLUMN IF NOT EXISTS form_fields jsonb DEFAULT '["username","password"]'::jsonb;
ALTER TABLE schools ADD COLUMN IF NOT EXISTS captcha_url VARCHAR(512);
-- login_url 需 512 以容纳 HFUT 等带长 service 参数的 CAS 登录地址
ALTER TABLE schools ALTER COLUMN login_url TYPE VARCHAR(512);
COMMENT ON COLUMN schools.form_fields IS '登录表单字段：username,password,captcha 等';
COMMENT ON COLUMN schools.captcha_url IS '验证码图片 URL，空则调用后端 GET /schools/:id/captcha';

-- info 接口配置（禁止写死）：eam_service_url 用于 CAS cookie 换取 EAM session，info_url 为学生信息页 base
ALTER TABLE schools ADD COLUMN IF NOT EXISTS eam_service_url VARCHAR(512);
ALTER TABLE schools ADD COLUMN IF NOT EXISTS info_url VARCHAR(512);
COMMENT ON COLUMN schools.eam_service_url IS 'EAM SSO 地址，CAS cookie 换取 EAM session 用';
COMMENT ON COLUMN schools.info_url IS '学生信息页 base URL，请求 /info/{code} 获取完整信息';

-- HFUT 需验证码，必须配置 login_url、captcha_url、eam_service_url、info_url（禁止写死）：
-- UPDATE schools SET form_fields = '[{"key":"username","label_zh":"学号","label_en":"Student ID"},{"key":"password","label_zh":"密码","label_en":"Password"},{"key":"captcha","label_zh":"验证码","label_en":"Captcha"}]'::jsonb,
--   login_url = 'https://cas.hfut.edu.cn/cas/login?service=https%3A%2F%2Fcas.hfut.edu.cn%2Fcas%2Foauth2.0%2FcallbackAuthorize%3Fclient_id%3DBsHfutEduPortal%26redirect_uri%3Dhttps%253A%252F%252Fone.hfut.edu.cn%252Fhome%252Findex%26response_type%3Dcode%26client_name%3DCasOAuthClient',
--   captcha_url = 'https://cas.hfut.edu.cn/cas/vercode'
-- WHERE code = 'hfut';

-- 用户认证表：记录用户在某学校的认证信息（学校信息门户 info 接口全部信息存入 cert_info）
CREATE TABLE IF NOT EXISTS user_cert
(
    id         SERIAL PRIMARY KEY,
    user_id    integer NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    school_id  integer NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
    cert_info  jsonb   NOT NULL DEFAULT '{}',
    status     smallint NOT NULL DEFAULT 1,
    created_at TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, school_id)
);

COMMENT ON TABLE user_cert IS '用户学校认证记录';
COMMENT ON COLUMN user_cert.cert_info IS '学生信息 JSON，来自学校信息门户 info 接口';
COMMENT ON COLUMN user_cert.status IS '1正常 2惰性删除';

-- 已有表增加 status 列（首次创建时 CREATE 已含，此 ALTER 用于旧表迁移）
ALTER TABLE user_cert ADD COLUMN IF NOT EXISTS status smallint NOT NULL DEFAULT 1;



UPDATE schools
SET form_fields = '[
  {"key":"username","label_zh":"学号","label_en":"Student ID"},
  {"key":"password","label_zh":"密码","label_en":"Password"},
  {"key":"captcha","label_zh":"验证码","label_en":"Captcha"}
]'::jsonb,
  login_url = 'https://cas.hfut.edu.cn/cas/login?service=https%3A%2F%2Fcas.hfut.edu.cn%2Fcas%2Foauth2.0%2FcallbackAuthorize%3Fclient_id%3DBsHfutEduPortal%26redirect_uri%3Dhttps%253A%252F%252Fone.hfut.edu.cn%252Fhome%252Findex%26response_type%3Dcode%26client_name%3DCasOAuthClient',
  captcha_url = 'https://cas.hfut.edu.cn/cas/vercode',
  eam_service_url = 'https://cas.hfut.edu.cn/cas/login?service=http://jxglstu.hfut.edu.cn/eams5-student/neusoft-sso/login',
  info_url = 'http://jxglstu.hfut.edu.cn/eams5-student/for-std/student-info'
WHERE code = 'hfut';