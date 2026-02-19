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
comment on column articles.status is '1:正常 2:禁用';
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
comment on column orders.order_status is '1:待支付 2:已支付 3:已发货 4:已收货 5:已取消';

-- 为没有 status 字段的表添加 status 字段
ALTER TABLE schools ADD COLUMN IF NOT EXISTS status smallint NOT NULL DEFAULT 1;
comment on column schools.status is '1:正常 2:禁用';


ALTER TABLE follow ADD COLUMN IF NOT EXISTS status smallint NOT NULL DEFAULT 1;
comment on column follow.status is '1:正常 2:禁用';

ALTER TABLE goods ADD COLUMN IF NOT EXISTS school_id integer REFERENCES schools(id);
comment on column goods.school_id is '学校ID';

ALTER TABLE articles ADD COLUMN IF NOT EXISTS school_id integer REFERENCES schools(id);
comment on column articles.school_id is '学校ID';

ALTER TABLE articles ADD COLUMN IF NOT EXISTS parent_id integer REFERENCES articles(id);
comment on column articles.parent_id is '父文章ID，仅回答类型(type=3)使用，指向提问';

-- 全文检索：tsvector 倒排索引，用于标题、正文搜索
ALTER TABLE articles ADD COLUMN IF NOT EXISTS search_vector tsvector
  GENERATED ALWAYS AS (setweight(to_tsvector('simple', coalesce(title,'')), 'A') ||
                       setweight(to_tsvector('simple', coalesce(content,'')), 'B')) STORED;
CREATE INDEX IF NOT EXISTS idx_articles_search ON articles USING GIN (search_vector);