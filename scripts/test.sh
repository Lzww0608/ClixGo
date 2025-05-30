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

# 检查覆盖率门槛（仅提醒，不阻止提交）
COVERAGE=$(echo "$COVERAGE_LINE" | awk '{print $3}' | sed 's/%//')
THRESHOLD=90

echo "📊 覆盖率分析："
echo "  当前覆盖率: $COVERAGE%"
echo "  目标覆盖率: $THRESHOLD%"

if command -v bc &> /dev/null; then
    if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
        echo "📝 提醒: 当前覆盖率 $COVERAGE% 低于目标 $THRESHOLD%"
        echo "🎯 建议: 在后续开发中逐步提高测试覆盖率"
        echo "✅ 允许继续提交（覆盖率要求已放宽）"
    else
        echo "🎉 太棒了! 覆盖率 $COVERAGE% 已达到目标 $THRESHOLD%"
    fi
else
    echo "⚠️  bc 命令未找到，跳过覆盖率数值检查"
fi

echo "✅ 测试检查完成（覆盖率不影响提交）"
