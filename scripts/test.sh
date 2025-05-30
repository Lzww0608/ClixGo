#!/bin/bash

set -e

echo "🧪 运行单元测试..."
go test -v ./...

echo "📊 生成覆盖率报告..."
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

echo "📈 当前覆盖率："
COVERAGE_LINE=$(go tool cover -func=coverage.out | tail -1)
echo "$COVERAGE_LINE"

echo "🌐 生成HTML报告..."
go tool cover -html=coverage.out -o coverage.html
echo "覆盖率报告已生成：coverage.html"

# 检查覆盖率门槛
COVERAGE=$(echo "$COVERAGE_LINE" | awk '{print $3}' | sed 's/%//')
THRESHOLD=90

if command -v bc &> /dev/null; then
    if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
        echo "❌ 覆盖率 $COVERAGE% 低于目标 $THRESHOLD%"
        exit 1
    else
        echo "✅ 覆盖率 $COVERAGE% 达到目标"
    fi
else
    echo "⚠️  bc 命令未找到，跳过覆盖率检查"
fi
