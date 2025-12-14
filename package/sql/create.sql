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

drop table if exists comments;

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
comment on column comments.ext_type is '关联类型 1:articles 2:goods';
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
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

comment on column likes.user_id is '用户ID';
comment on column likes.ext_id is '关联ID';
comment on column likes.ext_type is '关联类型 1:articles 2:comments 3:goods';
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
comment on column goods.good_status is '1:在售 2:下架';
comment on column goods.price is '商品价格，单位分';

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




