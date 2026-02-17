# 部署说明

## 服务器首次配置

### 1. 安装 Docker

```bash
curl -fsSL https://get.docker.com | sh
sudo systemctl enable docker
sudo systemctl start docker
```

### 2. 配置宿主机 .env

在宿主机创建 `/.env` 并填入环境变量（可参考项目根目录 `.env.example`）：

```bash
# 在宿主机创建 /.env，每行 KEY=VALUE 格式，填写 SERVER_HOST、DB_HOST、REDIS_HOST、JWT_SECRET 等（见 .env.example）
# 部署时会过滤掉 # 注释行，仅保留 KEY=VALUE 传给容器；含空格的 value 需用双引号
```

> **重要**：`.env` 含敏感信息，不提交到 Git。需手动上传到宿主机 `/.env`。

### 3. 配置 GitHub Secrets

仓库 **Settings** → **Secrets and variables** → **Actions** → **New repository secret**：

| Secret 名称        | 必填 | 说明 |
|--------------------|------|------|
| DEPLOY_HOST        | ✓ | 服务器 IP，如 `47.94.197.213` |
| DEPLOY_USER        | ✓ | SSH 用户名，如 `root` |
| DEPLOY_SSH_KEY     | ✓* | 私钥完整内容，或使用 DEPLOY_SSH_KEY_B64 |
| DEPLOY_SSH_KEY_B64 | * | 私钥的 Base64 编码：`base64 -w 0 ~/.ssh/deploy_key`（Linux） |
| GH_ACTIONS_READ_TOKEN |  | 可选。手动部署时若遇 403，添加 PAT（需 actions:read） |
| GH_ACTIONS_READ_TOKEN |  | 可选。仅部署时若遇 403，添加 PAT（需 actions:read） |

### 4. 在服务器添加部署公钥

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

推送到 `main` / `master` 后**自动执行构建**：

1. **编译**：Go build 生成 Linux 二进制
2. **构建镜像**：Docker build 并上传为 Artifact

**部署**需手动触发：Actions → CI/CD → Run workflow → Run workflow

### 手动部署

需要部署时：

1. 打开 **Actions** → 选择 **「CI/CD」**
2. 点击 **Run workflow** → 选择分支
3. 执行部署（复用最近一次 push 的构建镜像）；若遇 403，需在仓库 Settings → Actions → General 将 Workflow permissions 设为 Read and write，或添加 GH_ACTIONS_READ_TOKEN secret

---

## 手动部署（无 Actions）

1. 在 Actions 最新一次成功构建中下载 **apiserver-image** 产物
2. 重命名为 `apiserver.tar.gz`
3. 在本地执行：

```bash
scp apiserver.tar.gz root@47.94.197.213:/tmp/
ssh root@47.94.197.213 '
  grep -E "^[A-Za-z_][A-Za-z0-9_]*=" /.env > /tmp/.env.docker
  docker load < /tmp/apiserver.tar.gz
  docker stop apiserver 2>/dev/null; docker rm apiserver 2>/dev/null
  docker run -d --name apiserver --restart unless-stopped -p 8081:8081 --env-file /tmp/.env.docker apiserver:latest
'
```

---

## 容器启动失败 / 连接数据库失败

若报错 `dial tcp 127.0.0.1:5432: connection refused`：

- **原因**：`/.env` 不存在或 `DB_HOST` 等配置错误
- **处理**：确认宿主机 `/.env` 存在且 `DB_HOST` 指向正确地址，参考 `.env.example`

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

若容器无法连接，可在 `/.env` 中改用 `DB_HOST=172.17.0.1`、`REDIS_HOST=172.17.0.1`（Docker 网桥网关）。
