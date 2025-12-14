# Goland IDE 配置指南

## 解决 "未解析的引用" 问题

如果 Goland 显示 `StatusBadRequest` 等标准库常量无法识别，请按以下步骤操作：

### 方法 1：重新索引项目（推荐）

1. **File → Invalidate Caches...**
2. 选择 **Invalidate and Restart**
3. 等待重新索引完成

### 方法 2：配置 Go SDK

1. **Preferences / Settings** (Cmd+,)
2. 导航到 **Go → GOROOT**
3. 确保选择了正确的 Go SDK（1.24.11）
4. 如果没有，点击 **+** 添加 SDK
5. 选择路径：`/Users/dp/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.11.darwin-arm64`

### 方法 3：配置 Go Modules

1. **Preferences / Settings → Go → Go Modules**
2. 确保 **Enable Go modules integration** 已勾选
3. **Proxy** 设置为：`https://proxy.golang.org,direct`
4. 点击 **Apply**

### 方法 4：重新下载依赖

在终端中运行：
```bash
cd /Users/dp/Documents/go-proj/private/HFUT-Graduation-Project
go mod download
go mod tidy
```

### 方法 5：检查导入别名

确保代码中正确导入：
```go
import (
    "net/http"  // 标准库，不需要路径
    // ...
)
```

使用方式：
```go
http.StatusBadRequest  // 正确
StatusBadRequest       // 错误，需要 http. 前缀
```

### 方法 6：重启 Goland

有时简单的重启可以解决索引问题：
1. **File → Exit**
2. 重新打开 Goland
3. 打开项目

### 验证修复

1. 打开 `app/controller/good.go`
2. 将光标放在 `http.StatusBadRequest` 上
3. 按 **Cmd+B** (Go to Declaration)
4. 应该能跳转到 `net/http` 包的定义

### 如果问题仍然存在

1. 检查 **File → Project Structure → SDKs** 中是否有正确的 Go SDK
2. 检查 **File → Settings → Go → Build Tags & Vendoring** 中的配置
3. 尝试删除 `.idea` 目录并重新导入项目（注意：会丢失 IDE 配置）

