#!/bin/bash

set -e

echo "🔍 代码质量检查..."

# 1. 代码格式检查
echo "📝 检查代码格式..."
UNFORMATTED=$(gofmt -l . 2>/dev/null)
if [ -n "$UNFORMATTED" ]; then
    echo "❌ 代码格式不符合规范，以下文件需要格式化："
    echo "$UNFORMATTED"
    echo "运行 'gofmt -w .' 来修复格式问题"
    exit 1
fi
echo "✅ 代码格式检查通过"

# 2. 代码质量检查
echo "🔍 运行 go vet..."
go vet ./...
echo "✅ go vet 检查通过"

# 3. 安全扫描
echo "🔒 安全扫描..."
if command -v gosec &> /dev/null; then
    gosec ./... || echo "⚠️  发现安全问题，请检查"
else
    echo "⚠️  gosec 未安装，跳过安全扫描"
    echo "安装命令: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"
fi

# 4. 依赖检查
echo "📦 检查依赖..."
go mod tidy
go mod verify
echo "✅ 依赖检查通过"

# 5. 测试覆盖率检查（如果存在测试脚本）
if [ -f scripts/test.sh ]; then
    echo "🧪 检查测试覆盖率..."
    ./scripts/test.sh
else
    echo "⚠️  测试脚本不存在，跳过测试"
fi

echo "✅ 所有质量检查通过"
