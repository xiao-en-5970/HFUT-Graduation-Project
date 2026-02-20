# 日志栈追踪、收集与搜索方案

## 一、当前配置

项目使用 `go.uber.org/zap`，已启用：

- `zap.AddCaller()`：记录调用位置
- `zap.AddStacktrace(zapcore.ErrorLevel)`：**Error 及以上级别**自动附带栈追踪

## 二、栈追踪增强

### 2.1 通过环境变量配置栈追踪级别

通过 `LOG_STACKTRACE_LEVEL` 控制从哪个级别开始附带栈追踪（默认 `error`）：

```bash
# .env 或环境变量
LOG_STACKTRACE_LEVEL=error   # 仅 Error 及以上（默认）
LOG_STACKTRACE_LEVEL=warn    # Warn 及以上
LOG_STACKTRACE_LEVEL=info    # Info 及以上
LOG_STACKTRACE_LEVEL=debug   # 所有级别（调试用，生产慎用）
```

### 2.2 强制带栈记录

对关键异常希望每次都带栈时，可使用 `ErrorWithStack`：

```go
logger.ErrorWithStack(ctx, "数据库异常", zap.Error(err))
```

## 三、日志收集

### 3.1 输出方式

| 方式         | 适用场景       | 收集工具                     |
|------------|------------|--------------------------|
| **stdout** | Docker/K8s | 容器日志驱动、Fluentd、Filebeat  |
| **文件**     | 物理机/虚拟机    | Filebeat、Fluentd、rsyslog |
| **syslog** | 传统运维       | rsyslog、syslog-ng        |

### 3.2 推荐：JSON 输出到 stdout

生产环境建议使用 JSON 格式并输出到 stdout，便于：

- Docker 直接收集
- 由日志 Agent 统一采集
- 保留结构化字段，方便搜索

配置（`.env`）：

```bash
LOG_ENCODING=json
LOG_LEVEL=info
LOG_STACKTRACE_LEVEL=error   # 可选，栈追踪起始级别
```

### 3.3 文件输出（可选）

如需落盘，可增加文件 Core（轮转、按天切割等），由 Filebeat/Fluentd 采集该文件。

## 四、日志搜索方案

### 4.1 工具选型

| 方案                      | 特点                   | 适用规模 |
|-------------------------|----------------------|------|
| **Grafana Loki**        | 轻量、与 Grafana 集成好、成本低 | 中小型  |
| **ELK (Elasticsearch)** | 功能全、检索强              | 中大型  |
| **Graylog**             | 开箱即用、易部署             | 中小型  |
| **云厂商日志**               | 阿里云 SLS、腾讯云 CLS 等    | 云上部署 |

### 4.2 Loki 部署示例（轻量）

```yaml
# docker-compose.yml
services:
  loki:
    image: grafana/loki:2.9.0
    ports:
      - "3100:3100"
    command: -config.file=/etc/loki/local-config.yaml

  promtail:
    image: grafana/promtail:2.9.0
    volumes:
      - /var/log:/var/log
      - ./promtail-config.yml:/etc/promtail/config.yml
    command: -config.file=/etc/promtail/config.yml

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_AUTH_ANONYMOUS_ENABLED=true
```

应用以 JSON 输出到 stdout 时，由 Docker 日志驱动或 Fluentd 转发到 Loki 即可。

### 4.3 搜索友好字段设计

当前 Gin 中间件已有结构化字段，便于检索：

- `status`, `method`, `path`, `ip`, `latency`, `user_id`, `errors`

建议业务日志也使用 zap 字段，例如：

```go
logger.Error(ctx, "操作失败", zap.Error(err), zap.String("action", "create_post"), zap.Uint("id", id))
```

这样在 Loki/ELK 中可按 `action="create_post"`、`status>=500` 等过滤。

## 五、快速实施清单

1. **栈追踪**：通过 `LOG_STACKTRACE_LEVEL` 或 `ErrorWithStack` 控制
2. **生产输出**：`LOG_ENCODING=json` + stdout
3. **收集**：Docker 日志或 Filebeat/Fluentd 采集 stdout
4. **搜索**：部署 Loki/ELK，配置数据源和查询
