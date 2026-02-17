# 部署说明

## 服务器首次配置

### 1. 安装 Docker

```bash
curl -fsSL https://get.docker.com | sh
sudo systemctl enable docker
sudo systemctl start docker
```

### 2. 创建应用目录和 .env

```bash
sudo mkdir -p /opt/app
sudo cp .env.example /opt/app/.env
# 编辑 /opt/app/.env 填入生产环境配置（数据库、Redis、JWT 等）
sudo chmod 600 /opt/app/.env
```

### 3. 配置 GitHub Secrets

在仓库 Settings → Secrets and variables → Actions 中添加：

| Secret 名称        | 说明                         |
|--------------------|------------------------------|
| DEPLOY_HOST        | 服务器 IP，如 `47.94.197.213` |
| DEPLOY_USER        | SSH 用户名，如 `root`         |
| DEPLOY_SSH_KEY     | SSH 私钥完整内容              |

### 4. 确保 SSH 可登录

- 将 GitHub Actions 使用的公钥加入服务器 `~/.ssh/authorized_keys`
- 或使用已有密钥对，将私钥填入 `DEPLOY_SSH_KEY`

## 部署流程

推送代码到 `main` 或 `master` 分支后，GitHub Actions 将自动：

1. 编译 Linux 二进制（`go build -o build/app main.go`）
2. 构建 Docker 镜像
3. 通过 SCP 将镜像传到服务器
4. 在服务器上 `docker load` 并启动容器

应用监听 **8081** 端口，容器自动重启（`--restart unless-stopped`）。

## 手动部署（备用）

```bash
# 本地编译
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/app main.go

# 构建镜像
docker build -t apiserver:latest .

# 传到服务器后
docker load < apiserver.tar.gz
docker run -d --name apiserver -p 8081:8081 --env-file /opt/app/.env apiserver:latest
```
