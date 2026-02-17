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

### Push 到 main 分支

仅执行 **代码检查**（`go vet` + 编译验证），不编译、不构建、不部署。

### 打 tag 后（如 v1.0.0）

触发完整流水线：

1. **代码检查** → **编译 & 镜像构建** → **部署**（需手动审批）

打 tag 命令：

```bash
git tag v1.0.0
git push origin v1.0.0
```

（tag 需以 `v` 开头，如 `v1.0.0`、`v2.1.3`）

### 手动审批部署

流水线在「部署」步骤前暂停后：

1. 打开 **Actions** → 找到该次 tag 的流水线运行
2. 点击 **Review deployments**
3. 选择 **production** → 点击 **Approve and deploy**

### 首次配置 Environment（生产环境审批）

1. 打开仓库首页，点击 **Settings**
2. 左侧菜单找到 **Environments**，点击进入
3. 若列表为空，点击 **New environment**；若已有 `production`（首次运行流水线后自动创建），则直接点击进入
4. 在 `production` 页面中，找到 **Environment protection rules** 区域
5. 勾选 **Required reviewers**
6. 在弹出框中搜索并添加审批人（可添加自己的 GitHub 用户名），点击 **Add**
7. 点击 **Save protection rules** 保存

配置后，每次流水线运行到「部署」步骤时会显示 **Review deployments** 按钮，需你进入 Actions 页面批准后才会继续执行。

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
  docker image prune -f
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
