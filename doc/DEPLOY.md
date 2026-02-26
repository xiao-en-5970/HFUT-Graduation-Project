# 部署说明

CI/CD 使用 GitHub Actions，打 tag 时自动构建镜像并部署。镜像推送至 GitHub Container Registry (GHCR)，部署机通过
`docker pull` 拉取，**不再使用 Artifact 存储**。

## Secrets 配置

在 Repo → Settings → Secrets and variables → Actions 中配置：

| Secret                                  | 说明                                                  |
|-----------------------------------------|-----------------------------------------------------|
| `DEPLOY_HOST`                           | 部署服务器 IP 或域名                                        |
| `DEPLOY_USER`                           | SSH 登录用户名                                           |
| `DEPLOY_SSH_KEY` 或 `DEPLOY_SSH_KEY_B64` | SSH 私钥（后者为 base64 编码）                               |
| `GHCR_TOKEN`                            | GitHub PAT，需 `read:packages` 权限，用于部署机从 ghcr.io 拉取镜像 |

## 部署机要求

- 安装 Docker
- 创建 `/.env`，包含运行所需环境变量
- 能通过 SSH 访问（使用上述 DEPLOY_* 配置）

## 镜像地址

- 打 tag 时推送：`ghcr.io/<owner>/<repo>/apiserver:<tag>`
- 同时打 `latest`：`ghcr.io/<owner>/<repo>/apiserver:latest`

## 首次使用

1. 创建 PAT：GitHub → Settings → Developer settings → Personal access tokens，勾选 `read:packages`
2. 将 PAT 存入 Repo 的 `GHCR_TOKEN` Secret
3. 若镜像为私有，部署机需先 `docker login ghcr.io`（workflow 内会自动执行，使用 GHCR_TOKEN）
