#!/bin/bash

# ClixGo 增强基准测试脚本
# 基于ROADMAP性能目标的全面性能基准测试

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 配置
BENCHMARK_TIME=${BENCHMARK_TIME:-"3s"}
OUTPUT_DIR="benchmarks/results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
RESULTS_FILE="$OUTPUT_DIR/benchmark_$TIMESTAMP.txt"
JSON_FILE="$OUTPUT_DIR/benchmark_$TIMESTAMP.json"
BASELINE_FILE="$OUTPUT_DIR/baseline.txt"

# 性能目标（来自ROADMAP）
TARGET_STARTUP_TIME_MS=50
TARGET_MEMORY_MB=8
TARGET_CPU_PERCENT=1
TARGET_TERMINAL_CREATION_MS=10
TARGET_SESSION_SWITCH_MS=5
TARGET_UI_FPS=60

echo -e "${BLUE}🚀 ClixGo 性能基准测试框架${NC}"
echo -e "${CYAN}================================================${NC}"
echo -e "时间戳: $TIMESTAMP"
echo -e "基准测试时间: $BENCHMARK_TIME"
echo -e "结果目录: $OUTPUT_DIR"
echo ""

# 创建结果目录
mkdir -p "$OUTPUT_DIR"

# 清理旧的性能分析文件
cleanup_profiles() {
    echo -e "${YELLOW}🧹 清理旧的性能分析文件...${NC}"
    rm -f *.prof *.svg 2>/dev/null || true
}

# 运行基准测试套件
run_benchmark_suite() {
    local suite_name=$1
    local pattern=$2
    local description=$3
    
    echo -e "${PURPLE}📊 运行 $description 基准测试...${NC}"
    echo "========================================" >> "$RESULTS_FILE"
    echo "基准测试套件: $suite_name" >> "$RESULTS_FILE"
    echo "描述: $description" >> "$RESULTS_FILE"
    echo "时间: $(date)" >> "$RESULTS_FILE"
    echo "========================================" >> "$RESULTS_FILE"
    
    # 运行基准测试并保存结果
    go test -bench="$pattern" \
           -benchtime="$BENCHMARK_TIME" \
           -benchmem \
           -cpuprofile="cpu_${suite_name}.prof" \
           -memprofile="mem_${suite_name}.prof" \
           -blockprofile="block_${suite_name}.prof" \
           -timeout=10m \
           ./... 2>&1 | tee -a "$RESULTS_FILE"
    
    echo "" >> "$RESULTS_FILE"
}

# 分析性能数据
analyze_performance() {
    echo -e "${CYAN}📈 分析性能数据...${NC}"
    
    for prof_file in *.prof; do
        if [ -f "$prof_file" ]; then
            echo -e "${YELLOW}分析 $prof_file...${NC}"
            
            # 生成top 10分析
            echo "=== $prof_file Top 10 ===" >> "$RESULTS_FILE"
            go tool pprof -text "$prof_file" 2>/dev/null | head -20 >> "$RESULTS_FILE" || echo "无法分析 $prof_file" >> "$RESULTS_FILE"
            echo "" >> "$RESULTS_FILE"
            
            # 生成SVG火焰图（如果有go-torch）
            if command -v go-torch &> /dev/null; then
                svg_name="${prof_file%.prof}.svg"
                if [[ "$prof_file" == *"cpu"* ]]; then
                    go-torch -b "$prof_file" -f "$svg_name" 2>/dev/null || true
                elif [[ "$prof_file" == *"mem"* ]]; then
                    go-torch -alloc_space "$prof_file" -f "$svg_name" 2>/dev/null || true
                fi
                
                if [ -f "$svg_name" ]; then
                    echo -e "  ${GREEN}✓${NC} 生成火焰图: $svg_name"
                fi
            fi
        fi
    done
}

# 提取性能指标
extract_metrics() {
    echo -e "${CYAN}📏 提取关键性能指标...${NC}"
    
    # 创建JSON格式的结果
    cat > "$JSON_FILE" << EOF
{
  "timestamp": "$TIMESTAMP",
  "benchmark_time": "$BENCHMARK_TIME",
  "performance_targets": {
    "startup_time_ms": $TARGET_STARTUP_TIME_MS,
    "memory_mb": $TARGET_MEMORY_MB,
    "cpu_percent": $TARGET_CPU_PERCENT,
    "terminal_creation_ms": $TARGET_TERMINAL_CREATION_MS,
    "session_switch_ms": $TARGET_SESSION_SWITCH_MS,
    "ui_fps": $TARGET_UI_FPS
  },
  "results": {
EOF

    # 提取终端创建时间
    terminal_creation_ns=$(grep "BenchmarkTerminalCreation" "$RESULTS_FILE" | awk '{print $3}' | head -1)
    if [ ! -z "$terminal_creation_ns" ]; then
        terminal_creation_ms=$(echo "scale=2; $terminal_creation_ns / 1000000" | bc 2>/dev/null || echo "0")
        echo "    \"terminal_creation_ms\": $terminal_creation_ms," >> "$JSON_FILE"
        
        # 检查是否达到目标
        if (( $(echo "$terminal_creation_ms <= $TARGET_TERMINAL_CREATION_MS" | bc -l 2>/dev/null || echo "0") )); then
            echo -e "  ${GREEN}✓${NC} 终端创建时间: ${terminal_creation_ms}ms (目标: ${TARGET_TERMINAL_CREATION_MS}ms)"
        else
            echo -e "  ${RED}✗${NC} 终端创建时间: ${terminal_creation_ms}ms (目标: ${TARGET_TERMINAL_CREATION_MS}ms)"
        fi
    fi
    
    # 提取会话切换时间
    session_switch_ns=$(grep "BenchmarkSessionSwitch" "$RESULTS_FILE" | awk '{print $3}' | head -1)
    if [ ! -z "$session_switch_ns" ]; then
        session_switch_ms=$(echo "scale=2; $session_switch_ns / 1000000" | bc 2>/dev/null || echo "0")
        echo "    \"session_switch_ms\": $session_switch_ms," >> "$JSON_FILE"
        
        if (( $(echo "$session_switch_ms <= $TARGET_SESSION_SWITCH_MS" | bc -l 2>/dev/null || echo "0") )); then
            echo -e "  ${GREEN}✓${NC} 会话切换时间: ${session_switch_ms}ms (目标: ${TARGET_SESSION_SWITCH_MS}ms)"
        else
            echo -e "  ${RED}✗${NC} 会话切换时间: ${session_switch_ms}ms (目标: ${TARGET_SESSION_SWITCH_MS}ms)"
        fi
    fi
    
    # 提取启动时间
    startup_ns=$(grep "BenchmarkStartupTime" "$RESULTS_FILE" | awk '{print $3}' | head -1)
    if [ ! -z "$startup_ns" ]; then
        startup_ms=$(echo "scale=2; $startup_ns / 1000000" | bc 2>/dev/null || echo "0")
        echo "    \"startup_time_ms\": $startup_ms" >> "$JSON_FILE"
        
        if (( $(echo "$startup_ms <= $TARGET_STARTUP_TIME_MS" | bc -l 2>/dev/null || echo "0") )); then
            echo -e "  ${GREEN}✓${NC} 启动时间: ${startup_ms}ms (目标: ${TARGET_STARTUP_TIME_MS}ms)"
        else
            echo -e "  ${RED}✗${NC} 启动时间: ${startup_ms}ms (目标: ${TARGET_STARTUP_TIME_MS}ms)"
        fi
    fi
    
    echo "  }" >> "$JSON_FILE"
    echo "}" >> "$JSON_FILE"
}

# 比较基线性能
compare_baseline() {
    if [ -f "$BASELINE_FILE" ]; then
        echo -e "${CYAN}📊 与基线性能比较...${NC}"
        echo "=== 基线性能比较 ===" >> "$RESULTS_FILE"
        
        # 这里可以添加更复杂的基线比较逻辑
        echo "基线文件: $BASELINE_FILE" >> "$RESULTS_FILE"
        echo "当前结果: $RESULTS_FILE" >> "$RESULTS_FILE"
        echo "" >> "$RESULTS_FILE"
    else
        echo -e "${YELLOW}⚠️ 未找到基线文件，将当前结果设为基线...${NC}"
        cp "$RESULTS_FILE" "$BASELINE_FILE"
    fi
}

# 生成HTML报告
generate_html_report() {
    local html_file="$OUTPUT_DIR/benchmark_$TIMESTAMP.html"
    
    echo -e "${CYAN}📄 生成HTML报告...${NC}"
    
    cat > "$html_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>ClixGo 性能基准测试报告</title>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 10px; border-radius: 5px; }
        .metric { margin: 10px 0; padding: 10px; border-left: 4px solid #007cba; }
        .success { border-left-color: #28a745; }
        .warning { border-left-color: #ffc107; }
        .danger { border-left-color: #dc3545; }
        .code { background-color: #f8f9fa; padding: 10px; border-radius: 5px; font-family: monospace; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🚀 ClixGo 性能基准测试报告</h1>
        <p><strong>时间戳:</strong> $TIMESTAMP</p>
        <p><strong>基准测试时间:</strong> $BENCHMARK_TIME</p>
    </div>
    
    <h2>📊 性能目标对比</h2>
    <table>
        <tr><th>指标</th><th>目标值</th><th>实际值</th><th>状态</th></tr>
        <tr><td>启动时间</td><td>&lt; ${TARGET_STARTUP_TIME_MS}ms</td><td>待测试</td><td>-</td></tr>
        <tr><td>内存占用</td><td>&lt; ${TARGET_MEMORY_MB}MB</td><td>待测试</td><td>-</td></tr>
        <tr><td>终端创建</td><td>&lt; ${TARGET_TERMINAL_CREATION_MS}ms</td><td>待测试</td><td>-</td></tr>
        <tr><td>会话切换</td><td>&lt; ${TARGET_SESSION_SWITCH_MS}ms</td><td>待测试</td><td>-</td></tr>
        <tr><td>UI渲染</td><td>&gt; ${TARGET_UI_FPS} FPS</td><td>待测试</td><td>-</td></tr>
    </table>
    
    <h2>📈 详细基准测试结果</h2>
    <div class="code">
        <pre>$(cat "$RESULTS_FILE" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')</pre>
    </div>
    
    <h2>🔗 相关文件</h2>
    <ul>
        <li><a href="benchmark_$TIMESTAMP.txt">原始结果文件</a></li>
        <li><a href="benchmark_$TIMESTAMP.json">JSON格式结果</a></li>
EOF

    # 添加火焰图链接
    for svg_file in *.svg; do
        if [ -f "$svg_file" ]; then
            echo "        <li><a href=\"$svg_file\">火焰图: $svg_file</a></li>" >> "$html_file"
            # 移动SVG文件到结果目录
            mv "$svg_file" "$OUTPUT_DIR/"
        fi
    done
    
    cat >> "$html_file" << EOF
    </ul>
    
    <footer style="margin-top: 50px; color: #666;">
        <p>报告生成时间: $(date)</p>
        <p>ClixGo 性能基准测试框架 v1.0</p>
    </footer>
</body>
</html>
EOF

    echo -e "  ${GREEN}✓${NC} HTML报告: $html_file"
}

# 移动分析文件到结果目录
move_analysis_files() {
    echo -e "${CYAN}📦 整理分析文件...${NC}"
    
    for file in *.prof; do
        if [ -f "$file" ]; then
            mv "$file" "$OUTPUT_DIR/"
            echo -e "  ${GREEN}✓${NC} 移动: $file -> $OUTPUT_DIR/"
        fi
    done
}

# 主执行流程
main() {
    cleanup_profiles
    
    echo -e "${BLUE}开始基准测试执行...${NC}"
    echo ""
    
    # 运行各种基准测试套件
    run_benchmark_suite "terminal" "BenchmarkTerminal" "终端模块性能测试"
    run_benchmark_suite "ui" "BenchmarkUI" "UI模块性能测试"
    run_benchmark_suite "network" "BenchmarkNetwork" "网络模块性能测试"
    run_benchmark_suite "performance" "BenchmarkPerformance" "性能监控模块测试"
    run_benchmark_suite "task" "BenchmarkTask" "任务管理模块测试"
    run_benchmark_suite "concurrent" "BenchmarkConcurrent" "并发性能测试"
    run_benchmark_suite "memory" "BenchmarkMemory" "内存性能测试"
    
    # 分析和报告
    analyze_performance
    extract_metrics
    compare_baseline
    generate_html_report
    move_analysis_files
    
    echo ""
    echo -e "${GREEN}✅ 基准测试完成！${NC}"
    echo -e "${CYAN}================================================${NC}"
    echo -e "结果文件: $RESULTS_FILE"
    echo -e "JSON文件: $JSON_FILE"
    echo -e "HTML报告: $OUTPUT_DIR/benchmark_$TIMESTAMP.html"
    echo ""
    echo -e "${YELLOW}💡 提示:${NC}"
    echo -e "  • 使用 'go tool pprof <profile_file>' 进行详细分析"
    echo -e "  • 查看HTML报告获取可视化结果"
    echo -e "  • 比较不同版本的JSON文件来跟踪性能变化"
    echo ""
}

# 检查依赖
check_dependencies() {
    echo -e "${CYAN}🔍 检查依赖...${NC}"
    
    if ! command -v go &> /dev/null; then
        echo -e "${RED}错误: 未找到go命令${NC}"
        exit 1
    fi
    
    if ! command -v bc &> /dev/null; then
        echo -e "${YELLOW}警告: 未找到bc命令，数值计算可能受限${NC}"
    fi
    
    if ! command -v go-torch &> /dev/null; then
        echo -e "${YELLOW}提示: 未找到go-torch，跳过火焰图生成${NC}"
        echo -e "      安装方法: go install github.com/uber/go-torch@latest"
    fi
    
    echo ""
}

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --time)
                BENCHMARK_TIME="$2"
                shift 2
                ;;
            --output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            --help|-h)
                echo "ClixGo 基准测试脚本"
                echo ""
                echo "用法: $0 [选项]"
                echo ""
                echo "选项:"
                echo "  --time <duration>    基准测试运行时间 (默认: 3s)"
                echo "  --output <dir>       输出目录 (默认: benchmarks/results)"
                echo "  --help, -h           显示帮助信息"
                echo ""
                exit 0
                ;;
            *)
                echo -e "${RED}未知参数: $1${NC}"
                exit 1
                ;;
        esac
    done
}

# 脚本入口
parse_args "$@"
check_dependencies
main 