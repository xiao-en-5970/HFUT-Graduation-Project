# 中文智能分词全文检索（zhparser）

## 概述

全文检索固定使用 zhparser 中文智能分词。需先安装扩展再执行 `create.sql`。

---

## Docker 部署（postgres:18.1-alpine）

### 1. 构建带 zhparser 的镜像

```bash
cd deploy
docker build -f Dockerfile.postgres-zhparser -t postgres:18.1-alpine-zhparser .
```

### 2. 若已有数据（从原 postgres 容器迁移）

```bash
# 导出数据（在原 postgres 容器或本机执行）
docker exec <原postgres容器名> pg_dumpall -U postgres > backup.sql

# 停止并删除原容器
docker stop <原postgres容器名>
docker rm <原postgres容器名>

# 用新镜像启动，挂载同一数据卷（或新建卷）
docker run -d \
  --name postgres \
  -e POSTGRES_PASSWORD=你的密码 \
  -v /var/lib/postgresql/data:/var/lib/postgresql/data \
  postgres:18.1-alpine-zhparser

# 若用新卷，需先恢复数据：
# docker exec -i postgres psql -U postgres < backup.sql
```

### 3. 新建数据库

```bash
docker run -d \
  --name postgres \
  -e POSTGRES_PASSWORD=你的密码 \
  -v /var/lib/postgresql/data:/var/lib/postgresql/data \
  postgres:18.1-alpine-zhparser
```

### 4. 创建扩展（首次连接后）

```bash
docker exec -it postgres psql -U postgres -d <你的库名>

-- 执行
CREATE EXTENSION zhparser;
```

然后执行 `package/sql/create.sql` 或 `zhparser_search.sql` 完成配置。

> **若 PG 18 编译失败**：zhparser 官方目前主要支持到 PG 16，可改用 `zhparser/zhparser:alpine-16` 镜像，并用 `pg_dump`/
`pg_restore` 迁移数据。

---

## 非 Docker 安装

### Ubuntu / Debian

```bash
# 按 PostgreSQL 版本替换 15
sudo apt install postgresql-15-zhparser
```

### 从源码安装

1. 安装 SCWS：
   `wget -q -O - https://github.com/hightman/scws/archive/1.2.2.tar.gz | tar xzf - && cd scws-1.2.2 && ./configure && make && sudo make install`
2. 编译 zhparser：
   `git clone https://github.com/amutu/zhparser.git && cd zhparser && SCWS_HOME=/usr/local make && sudo make install`
