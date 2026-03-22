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

说明：

- **Artifact** 与 **Actions 缓存（cache）** 是不同计费项；本工作流未使用 `actions/upload-artifact`，但若 **组织/账号**
  下其它仓库或历史运行占满 Artifact 额度，仍可能报错（部分步骤或依赖会隐式产生 artifact）。
- 额度按 **组织或用户** 汇总，约每 6–12 小时重算一次。

建议处理：

1. **清理**：GitHub → 仓库 **Settings** → **Actions** → **Artifacts**（或组织级 **Actions → Artifact and log storage**）删除旧
   artifact。
2. **缩短保留**：在组织/仓库策略里缩短 artifact / log 保留天数，减少占用。
3. **扩容**
   ：在 [Billing / Actions](https://docs.github.com/en/billing/managing-billing-for-github-actions/about-billing-for-github-actions#calculating-minute-and-storage-spending)
   中了解存储计费并视情况升级或购买额外存储。

本仓库 `setup-go` 已设 `cache: false`，可减少 **缓存** 占用，但不直接增加 **Artifact** 配额。

## 相关链接

- [About billing for GitHub Actions](https://docs.github.com/en/billing/managing-billing-for-github-actions/about-billing-for-github-actions)
