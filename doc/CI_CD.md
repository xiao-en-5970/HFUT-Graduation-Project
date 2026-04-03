# CI/CD 说明（GitHub Actions）

## Node.js 20 弃用告警

GitHub 将逐步把 Actions 里运行 JavaScript action 的 Node 版本从 20 迁到 24（约 2026-06 起默认 24）。

本仓库 `.github/workflows/deploy.yml` 已做：

- 使用较新的 action 主版本：`actions/checkout@v6`、`actions/setup-go@v6`、`docker/login-action@v4`、
  `docker/build-push-action@v7`（通常基于 Node 24 或兼容新运行时）。
- 工作流顶层设置 `FORCE_JAVASCRIPT_ACTIONS_TO_NODE24: true`，在官方尚未全部切换前显式使用 Node 24 运行 JS action。

若仍出现告警，可到 [GitHub Changelog](https://github.blog/changelog/2025-09-19-deprecation-of-node-20-on-github-actions-runners/)
查看最新说明，并把各 action 升级到 `releases/latest` 对应 tag。

## Artifact storage quota（无法上传 Artifact）

错误示例：`Failed to CreateArtifact: Artifact storage quota has been hit`

### 本仓库已处理：`docker/build-push-action` 隐式上传

`docker/build-push-action` v6+ 在 **post** 阶段默认把 **build record** 通过 `GitHubArtifact.upload()` 传到 **GitHub
Artifacts**
（环境变量 `DOCKER_BUILD_RECORD_UPLOAD` 默认为开启）。这与是否手写 `actions/upload-artifact` 无关，仍会占用 Artifact 存储。

本工作流在 **build** 任务中已设置 `DOCKER_BUILD_RECORD_UPLOAD: "false"`：镜像仍会正常构建并推送到 GHCR，仅不再上传 build
record。

### 账号仍满额时（其它仓库或历史 artifact）

- 额度按 **组织或用户** 汇总，约每 6–12 小时重算。

建议：

1. **清理**：GitHub → 仓库 **Settings** → **Actions** → **Artifacts**（或组织级存储管理）删除旧 artifact。
2. **缩短保留**：缩短 artifact / log 保留天数。
3. **扩容
   **：[Billing / Actions 存储说明](https://docs.github.com/en/billing/managing-billing-for-github-actions/about-billing-for-github-actions#calculating-minute-and-storage-spending)。

本仓库 `setup-go` 已设 `cache: false`，可减少 **缓存** 占用。

## 部署时 `docker pull` 很久后 SSH 断线（Broken pipe）

`docker pull` 常持续数分钟，此阶段 **SSH 隧道上几乎只有 Docker 与 registry 的流量**，中间 NAT/防火墙可能把连接当空闲掐掉，客户端报
`client_loop: send disconnect: Broken pipe`。

部署步骤里已对 `ssh` 使用 **`ServerAliveInterval` / `ServerAliveCountMax`**（见 `deploy.yml`），让 Runner
周期性发保活，减轻该问题。若仍偶发，可在部署机 `sshd_config` 中配置 `ClientAliveInterval`，或排查宿主机/云安全组对长连接的限制。

## 相关链接

- [About billing for GitHub Actions](https://docs.github.com/en/billing/managing-billing-for-github-actions/about-billing-for-github-actions)
