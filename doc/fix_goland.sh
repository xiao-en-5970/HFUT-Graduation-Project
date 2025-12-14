#!/bin/bash
# Goland IDE 修复脚本

echo "🔧 修复 Goland IDE 配置..."

# 1. 清理 Go 模块缓存
echo "📦 清理并重新下载依赖..."
cd "$(dirname "$0")"
go clean -modcache
go mod download
go mod tidy

# 2. 验证编译
echo "✅ 验证代码编译..."
go build ./...

if [ $? -eq 0 ]; then
    echo "✅ 代码编译成功！"
    echo ""
    echo "📝 接下来请在 Goland 中执行："
    echo "   1. File → Invalidate Caches... → Invalidate and Restart"
    echo "   2. File → Settings → Go → GOROOT → 选择正确的 Go SDK"
    echo "   3. File → Settings → Go → Go Modules → 确保已启用"
else
    echo "❌ 编译失败，请检查错误信息"
    exit 1
fi

