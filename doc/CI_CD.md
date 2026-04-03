# CI/CD 说明（GitHub Actions）

## 流程概览

- **推分支**（`main` / `master`）：仅 **`go vet`** 静态分析，不编译。
- **打 tag**（`v*`）：在通过检查后，于 **Actions 上** `go build` 产出 **`build/app`**（Linux amd64），再 **rsync** 到部署机；部署机仅 **`docker build`**（`Dockerfile` 只 `COPY` 二进制与 `package/web/admin`），然后 **`docker run`**。
- **不再使用 GHCR**：镜像不在 Registry 里中转；部署机上的 Docker **层缓存**（`/var/lib/docker`）主要作用于 Ubuntu 基础层与静态资源，二进制随每次 tag 更新。

### 部署机要求

- 已安装 **Docker**（**无需**安装 Go；编译在 Actions 完成）。建议开启 **BuildKit**（工作流里已设 `DOCKER_BUILDKIT=1`）。
- 宿主机已按原约定准备 **`/.env`**（供 `grep` 出 `KEY=VALUE` 注入容器）。
- 可选 Secret **`DEPLOY_PATH`**：源码同步与构建目录；不填则使用部署用户 **`$HOME/hfut-apiserver`**。

### rsync 与 SSH 密钥

`rsync` 通过 SSH 连接远程时**不会**自动使用工作流里写入的 `~/.ssh/deploy_key`；本仓库已设置 `RSYNC_RSH` / `rsync -e`，与
`ssh -i ~/.ssh/deploy_key` 一致。若曾出现 `Permission denied (publickey)` 仅发生在 rsync 步骤，多为未指定 `-i` 导致。

### 所需 Secrets

| Secret                                  | 说明             |
|-----------------------------------------|----------------|
| `DEPLOY_HOST`                           | 部署机主机名或 IP     |
| `DEPLOY_USER`                           | SSH 用户         |
| `DEPLOY_SSH_KEY` 或 `DEPLOY_SSH_KEY_B64` | SSH 私钥         |
| `DEPLOY_PATH`（可选）                       | 远程目录绝对路径或留空用默认 |

**不再需要** `GHCR_TOKEN` 或 `packages:write`。

## Node.js 20 弃用告警

GitHub 将逐步把 Actions 里运行 JavaScript action 的 Node 版本从 20 迁到 24（约 2026-06 起默认 24）。

本仓库 `.github/workflows/deploy.yml` 已做：

- 使用较新的 action 主版本：`actions/checkout@v6`、`actions/setup-go@v6`。
- 工作流顶层设置 `FORCE_JAVASCRIPT_ACTIONS_TO_NODE24: true`。

若仍出现告警，可到 [GitHub Changelog](https://github.blog/changelog/2025-09-19-deprecation-of-node-20-on-github-actions-runners/)
查看最新说明，并把各 action 升级到 `releases/latest` 对应 tag。

## setup-go 与缓存

本仓库 `setup-go` 已设 `cache: false`，减轻 Actions 侧缓存占用；**Go 编译**在 **deploy job** 的 Runner 上执行一次。若希望改为在部署机本机 `go build`，需在服务器安装 Go，并在 `docker build` 前增加编译步骤（或改用多阶段 Dockerfile）。

## 相关链接

- [About billing for GitHub Actions](https://docs.github.com/en/billing/managing-billing-for-github-actions/about-billing-for-github-actions)
