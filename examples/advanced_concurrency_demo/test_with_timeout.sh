#!/bin/bash

# 带超时机制的并发优化演示测试脚本
# 
# 功能：
# 1. 设置最大运行时间防止程序卡死
# 2. 捕获和处理各种信号
# 3. 提供详细的运行日志
# 4. 自动清理资源

set -e

# 配置
TIMEOUT_SECONDS=60          # 最大运行时间：60秒
DEMO_DIR="$(dirname "$0")"
LOG_FILE="${DEMO_DIR}/test_output.log"
PID_FILE="${DEMO_DIR}/demo.pid"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%H:%M:%S') $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%H:%M:%S') $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $(date '+%H:%M:%S') $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%H:%M:%S') $1" | tee -a "$LOG_FILE"
}

# 清理函数
cleanup() {
    # 只有在紧急情况下才清理
    if [ "$EMERGENCY_CLEANUP" = "true" ]; then
        log_warning "正在清理资源..."
        
        # 终止可能运行的演示程序
        if [ -f "$PID_FILE" ]; then
            local pid=$(cat "$PID_FILE" 2>/dev/null || echo "")
            if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
                log_warning "正在终止演示程序 (PID: $pid)"
                kill -TERM "$pid" 2>/dev/null || true
                sleep 2
                if kill -0 "$pid" 2>/dev/null; then
                    log_warning "强制终止演示程序"
                    kill -KILL "$pid" 2>/dev/null || true
                fi
            fi
            rm -f "$PID_FILE"
        fi
        
        # 清理可能的孤儿进程
        pkill -f "advanced_concurrency_demo" 2>/dev/null || true
        
        log_info "资源清理完成"
    fi
    
    # 停止后台超时监控
    kill "$TIMEOUT_PID" 2>/dev/null || true
}

# 信号处理
emergency_cleanup() {
    EMERGENCY_CLEANUP=true
    cleanup
}

trap emergency_cleanup INT TERM
trap cleanup EXIT

# 超时处理函数
check_timeout() {
    if [ -f "$PID_FILE" ]; then
        local pid=$(cat "$PID_FILE" 2>/dev/null || echo "")
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            log_error "程序运行超时 (${TIMEOUT_SECONDS}秒)，强制停止"
            EMERGENCY_CLEANUP=true
            cleanup
            exit 124  # timeout exit code
        fi
    fi
}

# 启动超时监控
(
    sleep "$TIMEOUT_SECONDS"
    check_timeout
) &
TIMEOUT_PID=$!

# 主测试函数
run_test() {
    log_info "开始并发模型优化演示测试"
    log_info "最大运行时间: ${TIMEOUT_SECONDS}秒"
    log_info "日志输出: $LOG_FILE"
    
    # 进入演示目录
    cd "$DEMO_DIR"
    
    # 确保go模块依赖
    log_info "检查Go模块依赖..."
    if ! go mod tidy 2>>"$LOG_FILE"; then
        log_error "Go模块依赖检查失败"
        return 1
    fi
    
    # 编译程序
    log_info "编译演示程序..."
    if ! go build -o advanced_concurrency_demo main.go 2>>"$LOG_FILE"; then
        log_error "编译失败"
        return 1
    fi
    
    # 运行演示程序
    log_info "启动演示程序..."
    ./advanced_concurrency_demo &
    DEMO_PID=$!
    echo "$DEMO_PID" > "$PID_FILE"
    
    log_info "演示程序已启动 (PID: $DEMO_PID)"
    
    # 监控程序运行状态
    local start_time=$(date +%s)
    local timeout_triggered=false
    
    while kill -0 "$DEMO_PID" 2>/dev/null; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [ $elapsed -ge $TIMEOUT_SECONDS ]; then
            log_warning "程序运行时间达到限制，发送终止信号"
            timeout_triggered=true
            kill -TERM "$DEMO_PID" 2>/dev/null || true
            sleep 5  # 给程序更多时间优雅关闭
            if kill -0 "$DEMO_PID" 2>/dev/null; then
                log_warning "强制终止程序"
                kill -KILL "$DEMO_PID" 2>/dev/null || true
            fi
            break
        fi
        
        # 每5秒输出一次状态
        if [ $((elapsed % 5)) -eq 0 ] && [ $elapsed -gt 0 ]; then
            log_info "程序运行中... (已运行 ${elapsed}秒/${TIMEOUT_SECONDS}秒)"
        fi
        
        sleep 1
    done
    
    # 等待程序正常结束，给优雅关闭留出时间
    wait "$DEMO_PID" 2>/dev/null
    local exit_code=$?
    
    # 程序结束后立即停止超时监控和清除PID文件
    kill "$TIMEOUT_PID" 2>/dev/null || true
    rm -f "$PID_FILE"
    
    # 如果程序是被信号终止的，给它一些时间完成清理
    if [ $exit_code -gt 128 ]; then
        log_info "程序收到信号终止，等待清理完成..."
        sleep 3
    fi
    
    local end_time=$(date +%s)
    local total_time=$((end_time - start_time))
    
    if [ $exit_code -eq 0 ]; then
        log_success "演示程序正常结束 (运行时间: ${total_time}秒, 退出码: $exit_code)"
        return 0
    elif [ "$timeout_triggered" = "true" ]; then
        log_warning "演示程序因超时被终止 (运行时间: ${total_time}秒, 退出码: $exit_code)"
        return 124  # timeout exit code
    else
        log_error "演示程序异常结束 (运行时间: ${total_time}秒, 退出码: $exit_code)"
        return $exit_code
    fi
}

# 主执行逻辑
main() {
    # 初始化日志文件
    echo "==================== 并发优化演示测试 $(date) ====================" > "$LOG_FILE"
    
    # 显示系统信息
    log_info "系统信息:"
    echo "  Go版本: $(go version)" | tee -a "$LOG_FILE"
    echo "  操作系统: $(uname -s)" | tee -a "$LOG_FILE"
    echo "  CPU核心数: $(nproc)" | tee -a "$LOG_FILE"
    echo "  内存信息: $(free -h | head -2 | tail -1)" | tee -a "$LOG_FILE"
    
    # 运行测试
    if run_test; then
        log_success "测试成功完成"
        log_info "详细日志请查看: $LOG_FILE"
        exit 0
    else
        local exit_code=$?
        log_error "测试失败 (退出码: $exit_code)"
        log_info "详细日志请查看: $LOG_FILE"
        exit $exit_code
    fi
}

# 检查参数
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --help, -h     显示此帮助信息"
    echo "  --timeout N    设置超时时间（秒，默认60）"
    echo ""
    echo "示例:"
    echo "  $0             # 运行测试（默认60秒超时）"
    echo "  $0 --timeout 30   # 运行测试（30秒超时）"
    exit 0
fi

if [ "$1" = "--timeout" ] && [ -n "$2" ]; then
    TIMEOUT_SECONDS="$2"
    log_info "自定义超时时间: ${TIMEOUT_SECONDS}秒"
fi

# 执行主逻辑
main 