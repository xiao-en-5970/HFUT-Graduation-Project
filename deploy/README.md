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
# 从项目根目录复制 .env.example，编辑后保存为 /opt/app/.env
sudo chmod 600 /opt/app/.env
```

### 3. 配置 GitHub Secrets

仓库 **Settings** → **Secrets and variables** → **Actions** → **New repository secret**：

| Secret 名称        | 值 |
|--------------------|----|
| DEPLOY_HOST        | `47.94.197.213` |
| DEPLOY_USER        | `root`（或你的 SSH 用户名） |
| DEPLOY_SSH_KEY     | 私钥完整内容（见下方说明） |
| DEPLOY_SSH_KEY_B64 | （可选）若 DEPLOY_SSH_KEY 认证失败，用此方式：`cat ~/.ssh/deploy_key \| base64 -w0` 的输出 |

**DEPLOY_SSH_KEY 正确填写方式**：在终端执行 `cat ~/.ssh/deploy_key`，整段复制（含首尾 `-----BEGIN/END-----`），不要增减空格或换行。若仍报 Permission denied，改用 `DEPLOY_SSH_KEY_B64`。

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

推送到 `main` / `master` 后自动执行：

1. **代码检查**：`go vet` + `go build`，确认可编译
2. **构建镜像**：编译 Linux 二进制 → 构建 Docker 镜像 → 上传为 Artifact

### 手动部署

需要部署时：

1. 打开 **Actions** → 选择 **「手动部署」**
2. 点击 **Run workflow** → **Run workflow**
3. 会编译、构建镜像并部署到服务器

---

## 手动部署（无 Actions）

1. 在 Actions 最新一次成功运行中下载 **apiserver-image** 产物  
2. 重命名为 `apiserver.tar.gz`  
3. 在本地执行：

```bash
# 上传到服务器
scp apiserver.tar.gz root@47.94.197.213:/tmp/

# SSH 登录并部署
ssh root@47.94.197.213
docker load < /tmp/apiserver.tar.gz
docker stop apiserver 2>/dev/null; docker rm apiserver 2>/dev/null
docker run -d --name apiserver --restart unless-stopped -p 8081:8081 \
  -e SERVER_PORT=8081 -e SERVER_HOST=0.0.0.0 \
  --env-file /opt/app/.env apiserver:latest
```

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
