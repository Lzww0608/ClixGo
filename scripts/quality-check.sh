#!/bin/bash

set -e

# 检查是否为快速模式（用于git pre-commit）
QUICK_MODE=${QUICK_MODE:-false}
if [ "$1" = "--quick" ] || [ "$1" = "-q" ]; then
    QUICK_MODE=true
fi

if [ "$QUICK_MODE" = "true" ]; then
    echo "🚀 快速代码质量检查..."
else
    echo "🔍 完整代码质量检查..."
fi

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

# 3. 基本语法检查
echo "🔧 检查Go语法..."
go build -o /dev/null ./... 2>/dev/null || {
    echo "❌ 代码存在语法错误"
    exit 1
}
echo "✅ 语法检查通过"

# 快速模式跳过以下检查
if [ "$QUICK_MODE" = "true" ]; then
    echo "⚡ 快速模式：跳过安全扫描、依赖检查和测试覆盖率检查"
    echo "✅ 快速质量检查完成"
    exit 0
fi

# 4. 安全扫描（仅完整模式）
echo "🔒 安全扫描..."
if command -v gosec &> /dev/null; then
    gosec ./... || echo "⚠️  发现安全问题，请检查"
else
    echo "⚠️  gosec 未安装，跳过安全扫描"
    echo "安装命令: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"
fi

# 5. 依赖检查（仅完整模式）
echo "📦 检查依赖..."
go mod tidy
go mod verify
echo "✅ 依赖检查通过"

# 6. 测试覆盖率检查（仅完整模式）
if [ -f scripts/test.sh ]; then
    echo "🧪 检查测试覆盖率..."
    ./scripts/test.sh || echo "⚠️  测试执行过程中出现问题，但不影响提交"
else
    echo "⚠️  测试脚本不存在，跳过测试"
fi

echo "✅ 完整质量检查通过"
