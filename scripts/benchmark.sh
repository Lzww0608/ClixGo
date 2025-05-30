#!/bin/bash

echo "🚀 运行性能基准测试..."

# 运行基准测试
go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./...

echo "📊 生成性能报告..."

# CPU性能分析
echo "=== CPU 性能分析 ==="
if [ -f cpu.prof ]; then
    go tool pprof -text cpu.prof 2>/dev/null | head -20 || echo "CPU 分析文件为空或无法读取"
fi

# 内存性能分析
echo "=== 内存 性能分析 ==="
if [ -f mem.prof ]; then
    go tool pprof -text mem.prof 2>/dev/null | head -20 || echo "内存分析文件为空或无法读取"
fi

# 生成火焰图（需要安装 go-torch）
if command -v go-torch &> /dev/null; then
    echo "🔥 生成火焰图..."
    go-torch -b cpu.prof -f cpu_flame.svg 2>/dev/null || echo "火焰图生成失败"
    go-torch -alloc_space mem.prof -f mem_flame.svg 2>/dev/null || echo "内存火焰图生成失败"
fi

echo "✅ 性能分析完成"
