#!/bin/bash

# ClixGo 性能回归检测脚本
# 比较两个基准测试结果，检测性能回归

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 默认回归阈值
DEFAULT_TIME_THRESHOLD=10    # 时间回归阈值 10%
DEFAULT_MEMORY_THRESHOLD=15  # 内存回归阈值 15%
DEFAULT_ALLOC_THRESHOLD=20   # 分配回归阈值 20%

# 变量
BASELINE_FILE=""
CURRENT_FILE=""
TIME_THRESHOLD=$DEFAULT_TIME_THRESHOLD
MEMORY_THRESHOLD=$DEFAULT_MEMORY_THRESHOLD
ALLOC_THRESHOLD=$DEFAULT_ALLOC_THRESHOLD
OUTPUT_FILE=""
VERBOSE=false

# 使用说明
usage() {
    cat << EOF
ClixGo 性能回归检测脚本

用法: $0 [选项] <基线文件> <当前文件>

参数:
  <基线文件>           基线基准测试结果文件
  <当前文件>           当前基准测试结果文件

选项:
  --time-threshold <num>    时间回归阈值百分比 (默认: $DEFAULT_TIME_THRESHOLD%)
  --memory-threshold <num>  内存回归阈值百分比 (默认: $DEFAULT_MEMORY_THRESHOLD%)
  --alloc-threshold <num>   分配回归阈值百分比 (默认: $DEFAULT_ALLOC_THRESHOLD%)
  --output <file>           输出比较结果到文件
  --verbose, -v             详细输出
  --help, -h                显示帮助信息

示例:
  $0 baseline.txt current.txt
  $0 --time-threshold 5 --verbose baseline.txt current.txt
  $0 --output regression_report.txt baseline.txt current.txt

EOF
}

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --time-threshold)
                TIME_THRESHOLD="$2"
                shift 2
                ;;
            --memory-threshold)
                MEMORY_THRESHOLD="$2"
                shift 2
                ;;
            --alloc-threshold)
                ALLOC_THRESHOLD="$2"
                shift 2
                ;;
            --output)
                OUTPUT_FILE="$2"
                shift 2
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --help|-h)
                usage
                exit 0
                ;;
            -*)
                echo -e "${RED}错误: 未知选项 $1${NC}"
                usage
                exit 1
                ;;
            *)
                if [ -z "$BASELINE_FILE" ]; then
                    BASELINE_FILE="$1"
                elif [ -z "$CURRENT_FILE" ]; then
                    CURRENT_FILE="$1"
                else
                    echo -e "${RED}错误: 太多参数${NC}"
                    usage
                    exit 1
                fi
                shift
                ;;
        esac
    done
    
    # 检查必需参数
    if [ -z "$BASELINE_FILE" ] || [ -z "$CURRENT_FILE" ]; then
        echo -e "${RED}错误: 需要指定基线文件和当前文件${NC}"
        usage
        exit 1
    fi
    
    # 检查文件是否存在
    if [ ! -f "$BASELINE_FILE" ]; then
        echo -e "${RED}错误: 基线文件不存在: $BASELINE_FILE${NC}"
        exit 1
    fi
    
    if [ ! -f "$CURRENT_FILE" ]; then
        echo -e "${RED}错误: 当前文件不存在: $CURRENT_FILE${NC}"
        exit 1
    fi
}

# 输出函数
output() {
    local message="$1"
    echo -e "$message"
    if [ ! -z "$OUTPUT_FILE" ]; then
        echo -e "$message" | sed 's/\x1b\[[0-9;]*m//g' >> "$OUTPUT_FILE"
    fi
}

# 详细输出函数
verbose_output() {
    if [ "$VERBOSE" = true ]; then
        output "$1"
    fi
}

# 提取基准测试数据
extract_benchmark_data() {
    local file="$1"
    local benchmark_name="$2"
    
    # 提取基准测试行，格式: BenchmarkName-4 1000 1234 ns/op 456 B/op 7 allocs/op
    grep "^$benchmark_name" "$file" | head -1
}

# 解析基准测试结果
parse_benchmark_result() {
    local line="$1"
    
    if [ -z "$line" ]; then
        echo "0 0 0 0 0 0"
        return
    fi
    
    # 提取各个字段
    local name=$(echo "$line" | awk '{print $1}')
    local iterations=$(echo "$line" | awk '{print $2}')
    local time_ns=$(echo "$line" | awk '{print $3}')
    local time_unit=$(echo "$line" | awk '{print $4}')
    local memory_bytes=$(echo "$line" | awk '{print $5}' | sed 's/B\/op//')
    local allocs=$(echo "$line" | awk '{print $7}' | sed 's/allocs\/op//')
    
    # 转换时间单位到纳秒
    case "$time_unit" in
        "ns/op")
            time_ns_normalized=$time_ns
            ;;
        "μs/op"|"us/op")
            time_ns_normalized=$(echo "$time_ns * 1000" | bc 2>/dev/null || echo "$time_ns")
            ;;
        "ms/op")
            time_ns_normalized=$(echo "$time_ns * 1000000" | bc 2>/dev/null || echo "$time_ns")
            ;;
        "s/op")
            time_ns_normalized=$(echo "$time_ns * 1000000000" | bc 2>/dev/null || echo "$time_ns")
            ;;
        *)
            time_ns_normalized=$time_ns
            ;;
    esac
    
    echo "$iterations $time_ns_normalized ${memory_bytes:-0} ${allocs:-0}"
}

# 计算百分比变化
calculate_percentage_change() {
    local baseline="$1"
    local current="$2"
    
    if [ "$baseline" = "0" ] || [ -z "$baseline" ]; then
        echo "N/A"
        return
    fi
    
    local change=$(echo "scale=2; (($current - $baseline) / $baseline) * 100" | bc 2>/dev/null || echo "0")
    echo "$change"
}

# 检查是否为回归
is_regression() {
    local change="$1"
    local threshold="$2"
    
    if [ "$change" = "N/A" ]; then
        return 1
    fi
    
    local is_positive=$(echo "$change > $threshold" | bc 2>/dev/null || echo "0")
    [ "$is_positive" = "1" ]
}

# 比较基准测试结果
compare_benchmark() {
    local benchmark_name="$1"
    local display_name="$2"
    
    verbose_output "${CYAN}比较基准测试: $display_name${NC}"
    
    # 提取数据
    local baseline_line=$(extract_benchmark_data "$BASELINE_FILE" "$benchmark_name")
    local current_line=$(extract_benchmark_data "$CURRENT_FILE" "$benchmark_name")
    
    verbose_output "  基线: $baseline_line"
    verbose_output "  当前: $current_line"
    
    # 解析结果
    local baseline_data=($(parse_benchmark_result "$baseline_line"))
    local current_data=($(parse_benchmark_result "$current_line"))
    
    local baseline_time=${baseline_data[1]}
    local current_time=${current_data[1]}
    local baseline_memory=${baseline_data[2]}
    local current_memory=${current_data[2]}
    local baseline_allocs=${baseline_data[3]}
    local current_allocs=${current_data[3]}
    
    # 计算变化
    local time_change=$(calculate_percentage_change "$baseline_time" "$current_time")
    local memory_change=$(calculate_percentage_change "$baseline_memory" "$current_memory")
    local alloc_change=$(calculate_percentage_change "$baseline_allocs" "$current_allocs")
    
    # 检查回归
    local time_regression=""
    local memory_regression=""
    local alloc_regression=""
    
    if is_regression "$time_change" "$TIME_THRESHOLD"; then
        time_regression="${RED}REGRESSION${NC}"
    elif [ "$time_change" != "N/A" ] && (( $(echo "$time_change < -5" | bc -l 2>/dev/null || echo "0") )); then
        time_regression="${GREEN}IMPROVED${NC}"
    else
        time_regression="${YELLOW}OK${NC}"
    fi
    
    if is_regression "$memory_change" "$MEMORY_THRESHOLD"; then
        memory_regression="${RED}REGRESSION${NC}"
    elif [ "$memory_change" != "N/A" ] && (( $(echo "$memory_change < -5" | bc -l 2>/dev/null || echo "0") )); then
        memory_regression="${GREEN}IMPROVED${NC}"
    else
        memory_regression="${YELLOW}OK${NC}"
    fi
    
    if is_regression "$alloc_change" "$ALLOC_THRESHOLD"; then
        alloc_regression="${RED}REGRESSION${NC}"
    elif [ "$alloc_change" != "N/A" ] && (( $(echo "$alloc_change < -5" | bc -l 2>/dev/null || echo "0") )); then
        alloc_regression="${GREEN}IMPROVED${NC}"
    else
        alloc_regression="${YELLOW}OK${NC}"
    fi
    
    # 输出结果
    output "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    output "${CYAN}📊 $display_name${NC}"
    output "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    if [ "$baseline_line" = "" ]; then
        output "  ${YELLOW}⚠️ 基线数据缺失${NC}"
    elif [ "$current_line" = "" ]; then
        output "  ${YELLOW}⚠️ 当前数据缺失${NC}"
    else
        output "  🕒 执行时间:   ${time_change}% ${time_regression}"
        output "  💾 内存使用:   ${memory_change}% ${memory_regression}"
        output "  🔢 内存分配:   ${alloc_change}% ${alloc_regression}"
        
        if [ "$VERBOSE" = true ]; then
            output ""
            output "  详细对比:"
            output "    时间: $baseline_time ns/op → $current_time ns/op"
            output "    内存: $baseline_memory B/op → $current_memory B/op"
            output "    分配: $baseline_allocs allocs/op → $current_allocs allocs/op"
        fi
    fi
    
    output ""
    
    # 返回是否有回归
    if is_regression "$time_change" "$TIME_THRESHOLD" || \
       is_regression "$memory_change" "$MEMORY_THRESHOLD" || \
       is_regression "$alloc_change" "$ALLOC_THRESHOLD"; then
        return 1
    else
        return 0
    fi
}

# 主函数
main() {
    local regression_count=0
    local total_benchmarks=0
    
    output "${BLUE}🔍 ClixGo 性能回归检测${NC}"
    output "${CYAN}================================================${NC}"
    output "基线文件: $BASELINE_FILE"
    output "当前文件: $CURRENT_FILE"
    output "回归阈值: 时间 ${TIME_THRESHOLD}%, 内存 ${MEMORY_THRESHOLD}%, 分配 ${ALLOC_THRESHOLD}%"
    output ""
    
    # 定义要比较的基准测试
    declare -A benchmarks=(
        ["BenchmarkTerminalCreation"]="终端创建性能"
        ["BenchmarkSessionSwitch"]="会话切换性能"
        ["BenchmarkPTYOperations"]="PTY操作性能"
        ["BenchmarkStartupTime"]="启动时间性能"
        ["BenchmarkUIRendering"]="UI渲染性能"
        ["BenchmarkPanelOperations"]="面板操作性能"
        ["BenchmarkEventHandling"]="事件处理性能"
        ["BenchmarkConcurrentSessionManagement"]="并发会话管理"
        ["BenchmarkTerminalMemoryAllocation"]="终端内存分配"
        ["BenchmarkSessionPersistence"]="会话持久化"
        ["BenchmarkPerformanceOptimizer"]="性能优化器"
        ["BenchmarkTaskPerformanceAnalyzer"]="任务性能分析"
        ["BenchmarkRealtimeNetworkMonitor"]="实时网络监控"
        ["BenchmarkCommandCreation"]="命令创建性能"
        ["BenchmarkTaskCreation"]="任务创建性能"
    )
    
    # 比较每个基准测试
    for benchmark_name in "${!benchmarks[@]}"; do
        local display_name="${benchmarks[$benchmark_name]}"
        total_benchmarks=$((total_benchmarks + 1))
        
        if ! compare_benchmark "$benchmark_name" "$display_name"; then
            regression_count=$((regression_count + 1))
        fi
    done
    
    # 输出总结
    output "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    output "${CYAN}📋 总结报告${NC}"
    output "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    output "总基准测试数: $total_benchmarks"
    output "性能回归数量: $regression_count"
    
    if [ $regression_count -eq 0 ]; then
        output "${GREEN}✅ 未检测到性能回归！${NC}"
        exit_code=0
    else
        output "${RED}❌ 检测到 $regression_count 个性能回归${NC}"
        exit_code=1
    fi
    
    if [ ! -z "$OUTPUT_FILE" ]; then
        output ""
        output "详细报告已保存到: $OUTPUT_FILE"
    fi
    
    output "${CYAN}================================================${NC}"
    
    exit $exit_code
}

# 初始化输出文件
init_output_file() {
    if [ ! -z "$OUTPUT_FILE" ]; then
        cat > "$OUTPUT_FILE" << EOF
ClixGo 性能回归检测报告
=====================

生成时间: $(date)
基线文件: $BASELINE_FILE
当前文件: $CURRENT_FILE
回归阈值: 时间 ${TIME_THRESHOLD}%, 内存 ${MEMORY_THRESHOLD}%, 分配 ${ALLOC_THRESHOLD}%

EOF
    fi
}

# 检查依赖
check_dependencies() {
    if ! command -v bc &> /dev/null; then
        echo -e "${YELLOW}警告: 未找到bc命令，百分比计算可能受限${NC}"
    fi
}

# 脚本入口
parse_args "$@"
check_dependencies
init_output_file
main 