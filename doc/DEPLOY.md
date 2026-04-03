# 部署说明

CI/CD 使用 GitHub Actions：打 tag 时先通过代码检查，再将源码 **rsync 到部署机**，在部署机上 **`docker build`**（根目录多阶段
`Dockerfile`）并 **`docker run`**。镜像**不经过** GitHub Container Registry；构建产物与 Docker **层缓存**均在部署机本地。

详见 `doc/CI_CD.md`。

## Secrets 配置

在 Repo → Settings → Secrets and variables → Actions 中配置：

| Secret                                  | 说明                                  |
|-----------------------------------------|-------------------------------------|
| `DEPLOY_HOST`                           | 部署服务器 IP 或域名                        |
| `DEPLOY_USER`                           | SSH 登录用户名                           |
| `DEPLOY_SSH_KEY` 或 `DEPLOY_SSH_KEY_B64` | SSH 私钥（后者为 base64 编码）               |
| `DEPLOY_PATH`（可选）                       | 远程源码目录；不填则使用 `$HOME/hfut-apiserver` |

## 部署机要求

- 安装 Docker（建议已启用 BuildKit）
- 创建 `/.env`，包含运行所需环境变量（工作流会 `grep` 出 `KEY=VALUE` 行注入容器）
- 能通过 SSH 访问（使用上述 `DEPLOY_*` 配置）

## 本地镜像名

- 构建：`apiserver:<tag>`（与 Git tag 一致，如 `v1.0.0`），并打 `apiserver:latest`
- 运行：使用 `apiserver:<tag>` 启动容器

## 首次使用

1. 配置上述 Secrets（无需 PAT / GHCR）
2. 在部署机准备 `/.env` 与 Docker
3. 推送 tag 触发部署；可在 Actions 日志中查看 rsync 与 `docker build` 输出
