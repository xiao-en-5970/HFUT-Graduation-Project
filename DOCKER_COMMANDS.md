# Docker 构建和运行命令

## 1. 构建 Docker 镜像

```bash
# 在项目根目录执行
docker build -t hfut-graduation-project:latest -f package/docker/apiserver/Dockerfile .
```

或者使用新的 Dockerfile：

```bash
docker build -t hfut-graduation-project:latest -f package/docker/apiserver/Dockerfile.new .
```

## 2. 运行 Docker 容器

### 基础运行命令（端口映射到 8082）

```bash
docker run -d \
  --name hfut-api \
  -p 8082:8080 \
  -e SERVER_PORT=8080 \
  -e DB_HOST=host.docker.internal \
  -e DB_PORT=5432 \
  -e DB_USER=postgres \
  -e DB_PASSWORD=postgres \
  -e DB_NAME=graduation_project \
  -e REDIS_HOST=host.docker.internal \
  -e REDIS_PORT=6379 \
  hfut-graduation-project:latest
```

### 如果 PostgreSQL 和 Redis 也在 Docker 容器中（推荐）

```bash
# 先确保 PostgreSQL 和 Redis 容器正在运行
docker run -d \
  --name hfut-api \
  -p 8082:8080 \
  --network hfut-network \
  -e SERVER_PORT=8080 \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e DB_USER=postgres \
  -e DB_PASSWORD=postgres \
  -e DB_NAME=graduation_project \
  -e REDIS_HOST=redis \
  -e REDIS_PORT=6379 \
  hfut-graduation-project:latest
```

### 使用环境变量文件（.env）

```bash
docker run -d \
  --name hfut-api \
  -p 8082:8080 \
  --env-file .env \
  hfut-graduation-project:latest
```

### 完整命令（包含所有常用选项）

```bash
docker run -d \
  --name hfut-api \
  --restart unless-stopped \
  -p 8082:8080 \
  --network hfut-network \
  -e SERVER_HOST=0.0.0.0 \
  -e SERVER_PORT=8080 \
  -e SERVER_MODE=release \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e DB_USER=postgres \
  -e DB_PASSWORD=postgres \
  -e DB_NAME=graduation_project \
  -e DB_SSLMODE=disable \
  -e DB_TIMEZONE=PRC \
  -e REDIS_HOST=redis \
  -e REDIS_PORT=6379 \
  -e REDIS_PASSWORD= \
  -e REDIS_DB=0 \
  -e LOG_LEVEL=info \
  -e JWT_SECRET=your-secret-key-change-in-production \
  -e JWT_EXPIRE_HOUR=24 \
  hfut-graduation-project:latest
```

## 3. 查看容器状态和日志

```bash
# 查看运行中的容器
docker ps

# 查看容器日志
docker logs hfut-api

# 实时查看日志
docker logs -f hfut-api

# 查看容器详细信息
docker inspect hfut-api
```

## 4. 停止和删除容器

```bash
# 停止容器
docker stop hfut-api

# 启动已停止的容器
docker start hfut-api

# 删除容器
docker rm hfut-api

# 强制删除运行中的容器
docker rm -f hfut-api
```

## 5. 进入容器调试

```bash
# 进入运行中的容器
docker exec -it hfut-api sh

# 在容器中执行命令
docker exec hfut-api ls -la /app
```

## 6. 端口说明

- **容器内部端口**: 8080（应用默认端口）
- **主机映射端口**: 8082（你指定的端口）
- **访问地址**: `http://localhost:8082`

如果需要修改容器内部端口为 8082，需要设置环境变量：
```bash
-e SERVER_PORT=8082
```
同时修改 EXPOSE 和端口映射：
```bash
-p 8082:8082
```

## 7. 网络配置

如果使用 Docker Compose 或需要连接到其他容器：

```bash
# 创建网络（如果不存在）
docker network create hfut-network

# 查看网络
docker network ls

# 将容器连接到网络
docker network connect hfut-network hfut-api
```

## 8. 一键构建和运行

```bash
# 构建镜像
docker build -t hfut-graduation-project:latest -f package/docker/apiserver/Dockerfile .

# 停止并删除旧容器（如果存在）
docker rm -f hfut-api 2>/dev/null || true

# 运行新容器
docker run -d \
  --name hfut-api \
  --restart unless-stopped \
  -p 8082:8080 \
  --network hfut-network \
  -e SERVER_PORT=8080 \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e DB_USER=postgres \
  -e DB_PASSWORD=postgres \
  -e DB_NAME=graduation_project \
  -e REDIS_HOST=redis \
  -e REDIS_PORT=6379 \
  hfut-graduation-project:latest

# 查看日志
docker logs -f hfut-api
```


