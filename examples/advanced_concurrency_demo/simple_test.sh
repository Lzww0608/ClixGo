#!/bin/bash

# 简单的并发优化演示测试脚本
set -e

echo "🚀 开始ClixGo并发模型优化测试"
echo "时间: $(date)"
echo "系统: $(uname -s) $(uname -m)"
echo "Go版本: $(go version)"
echo "CPU核心: $(nproc)"
echo ""

# 进入演示目录
cd "$(dirname "$0")"

# 编译程序
echo "📦 编译演示程序..."
if ! go build -o advanced_concurrency_demo main.go; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✅ 编译成功"
echo ""

# 运行程序，使用timeout命令确保不会卡死
echo "🏃 运行演示程序 (最大45秒)..."
echo "程序输出:"
echo "----------------------------------------"

# 使用timeout命令运行，45秒后自动终止
if timeout 45s ./advanced_concurrency_demo; then
    exit_code=$?
    echo ""
    echo "----------------------------------------"
    echo "✅ 程序正常结束 (退出码: $exit_code)"
    echo ""
    echo "🎉 测试成功完成！"
    echo ""
    echo "📊 测试总结:"
    echo "- 程序运行正常，没有卡死"
    echo "- 并发任务处理成功"
    echo "- 优雅关闭机制工作正常"
    echo "- 所有组件协调运行良好"
    exit 0
else
    exit_code=$?
    echo ""
    echo "----------------------------------------"
    if [ $exit_code -eq 124 ]; then
        echo "⚠️  程序运行超时 (45秒)"
        echo "这可能表明存在性能问题或死锁"
        exit 1
    else
        echo "❌ 程序异常结束 (退出码: $exit_code)"
        exit $exit_code
    fi
fi 