# 故障排除指南

## crypto/sha3 错误

### 问题描述
IDE 显示错误：`package crypto/sha3 is not in std (/Users/dp/.goenv/versions/1.23.7/src/crypto/sha3)`

### 原因
这是一个 IDE lint 工具的配置问题，不是代码问题：
- 项目实际使用的是 Go 1.24.11（通过 toolchain）
- IDE 的 lint 工具可能使用了 goenv 的 1.23.7
- Go 1.23.7 的安装中 `crypto/sha3` 位于 vendor 目录，不在标准库位置

### 验证
代码可以正常编译和运行：
```bash
go build ./...
# 编译成功，无错误
```

### 解决方案

#### 方案 1：配置 IDE 使用正确的 Go 版本（推荐）

**VS Code / Cursor:**
1. 打开设置 (Cmd+,)
2. 搜索 `go.goroot`
3. 设置为空（让 IDE 自动检测）或指向正确的 Go 安装路径
4. 重启 IDE

**GoLand:**
1. Preferences → Go → GOROOT
2. 选择正确的 Go SDK（1.24.11）

#### 方案 2：更新 goenv 并安装 Go 1.24.11

```bash
# 更新 goenv
brew update && brew upgrade goenv

# 安装 Go 1.24.11
goenv install 1.24.11

# 设置项目使用 1.24.11
cd /Users/dp/Documents/go-proj/private/HFUT-Graduation-Project
goenv local 1.24.11
```

#### 方案 3：忽略 lint 错误（临时方案）

如果代码可以正常编译运行，可以暂时忽略 IDE 的 lint 错误。这不会影响程序的运行。

### 验证修复

运行以下命令确认一切正常：
```bash
go build ./...
go test ./...
```

如果编译通过，说明代码没有问题，只是 IDE 配置需要调整。

