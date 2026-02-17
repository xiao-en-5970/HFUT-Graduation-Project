# 部署说明

## 服务器首次配置

### 1. 安装 Docker

```bash
curl -fsSL https://get.docker.com | sh
sudo systemctl enable docker
sudo systemctl start docker
```

### 2. 确保服务器有 Python 3

部署脚本会运行 `python3` 生成 Docker 兼容的 `.env.docker`。Ubuntu/Debian 通常已预装；若无则执行 `apt install python3`。

### 3. 配置服务器 .env

将本地 `.env` 拷贝到服务器 `/opt/app/.env`（首次部署或配置变更时执行一次）：

```bash
scp .env root@47.94.197.213:/opt/app/.env
ssh root@47.94.197.213 "mkdir -p /opt/app && chmod 600 /opt/app/.env"
```

> **重要**：`.env` 必须包含 `DB_HOST`、`DB_USER`、`DB_NAME`、`DB_PASSWORD` 等变量，且不能为空，否则容器会连不上数据库。

### 4. 配置 GitHub Secrets

仓库 **Settings** → **Secrets and variables** → **Actions** → **New repository secret**：

| Secret 名称        | 必填 | 说明 |
|--------------------|------|------|
| DEPLOY_HOST        | ✓ | 服务器 IP，如 `47.94.197.213` |
| DEPLOY_USER        | ✓ | SSH 用户名，如 `root` |
| DEPLOY_SSH_KEY     | ✓* | 私钥完整内容，或使用 DEPLOY_SSH_KEY_B64 |
| DEPLOY_SSH_KEY_B64 | * | 私钥的 Base64 编码：`base64 -w 0 ~/.ssh/deploy_key`（Linux） |
| GH_ACTIONS_READ_TOKEN |  | 可选。仅部署时若遇 403，添加 PAT（需 actions:read） |

### 5. 在服务器添加部署公钥

本机生成专用密钥对（仅用于部署）：

```bash
ssh-keygen -t ed25519 -C "deploy" -f ~/.ssh/deploy_key -N ""
cat ~/.ssh/deploy_key.pub
```

SSH 登录服务器，将上面输出的公钥追加到 `~/.ssh/authorized_keys`：

```bash
echo "公钥内容" >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

将 `~/.ssh/deploy_key`（私钥）完整复制到 GitHub Secret `DEPLOY_SSH_KEY`。

---

## CI/CD 流程

### 推送时（自动）

推送到 `main` / `master` 后自动执行：

1. **代码检查**：`go vet` + `go build`，确认可编译
2. **构建镜像**：编译 Linux 二进制 → 构建 Docker 镜像 → 上传为 Artifact

### 手动部署

需要部署时：

1. 打开 **Actions** → 选择 **「CI/CD」**
2. 点击 **Run workflow** → 选择分支
3. **完整流程**（默认）：不勾选「仅部署」，执行 检查 → 构建 → 部署
4. **仅部署**：勾选「仅部署」可复用最近一次构建；若遇 403，需在仓库 Settings → Actions → General 将 Workflow permissions 设为 Read and write，或添加 GH_ACTIONS_READ_TOKEN secret

---

## 手动部署（无 Actions）

1. 在 Actions 最新一次成功运行中下载 **apiserver-image** 产物  
2. 重命名为 `apiserver.tar.gz`  
3. 在本地执行：

```bash
# 上传镜像和 env 转换脚本到服务器
scp apiserver.tar.gz .github/scripts/env-to-docker.py root@47.94.197.213:/tmp/

# SSH 登录并部署（脚本生成 Docker 兼容的 .env.docker）
ssh root@47.94.197.213
python3 /tmp/env-to-docker.py
docker load < /tmp/apiserver.tar.gz
docker stop apiserver 2>/dev/null; docker rm apiserver 2>/dev/null
docker run -d --name apiserver --restart unless-stopped -p 8081:8081 \
  -e SERVER_PORT=8081 -e SERVER_HOST=0.0.0.0 \
  --env-file /opt/app/.env.docker apiserver:latest
```

> **说明**：`--env-file` 要求无行内空格，否则会报 `variable contains whitespaces`。若 `.env` 无空格可直接用 `--env-file /opt/app/.env`，否则需先用上述 Python 生成 `.env.docker`。

---

## 容器启动失败 / 连接数据库失败

若报错 `dial tcp 127.0.0.1:5432: connection refused` 或 `user=postgres database=graduation_project`：

- **原因**：容器未收到环境变量，使用了默认值（localhost）
- **处理**：确认 `/opt/app/.env` 存在且包含 `DB_HOST`、`DB_USER`、`DB_NAME`、`DB_PASSWORD` 等，执行 `cat /opt/app/.env | grep DB_HOST` 应有输出。修复后重新部署。

---

## SSH 认证失败排查

若出现 `unable to authenticate, attempted methods [none publickey]`：

1. **私钥格式**：确认 `DEPLOY_SSH_KEY` 包含首尾两行：
   ```
   -----BEGIN OPENSSH PRIVATE KEY-----
   ...
   -----END OPENSSH PRIVATE KEY-----
   ```

2. **公钥在服务器**：登录服务器执行 `cat ~/.ssh/authorized_keys`，确认有对应公钥。

3. **密钥对应**：私钥和公钥必须来自同一对，用 `ssh-keygen -y -f deploy_key` 验证。

4. **权限**：`~/.ssh` 为 700，`authorized_keys` 为 600。

---

## 部署与数据库访问

容器通过公网 IP `47.94.197.213` 访问 PostgreSQL 和 Redis。需确保：

1. **1Panel**：PostgreSQL、Redis 容器已映射端口 5432、6379
2. **防火墙/安全组**：开放 5432、6379 入站（建议仅允许本机或可信 IP）
3. **监听地址**：PostgreSQL `listen_addresses='*'`，Redis `bind 0.0.0.0`

若容器无法连接，可尝试在 `.env` 中改用 `DB_HOST=172.17.0.1`、`REDIS_HOST=172.17.0.1`（Docker 网桥网关）。
