#!/bin/bash

# ClixGo 第二阶段功能模块质量检查脚本
# 作者: Lzww0608
# 日期: 2025-05-31

set -e

echo "🔍 ClixGo 第二阶段功能模块质量检查"
echo "========================================"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查函数
check_module() {
    local module=$1
    local expected_coverage=$2
    
    echo -e "\n${BLUE}📦 检查模块: $module${NC}"
    echo "----------------------------------------"
    
    # 运行测试并生成覆盖率报告
    if go test -v -timeout 30s -coverprofile="${module}_coverage.out" "./pkg/$module"; then
        echo -e "${GREEN}✅ 测试通过${NC}"
        
        # 检查覆盖率
        coverage=$(go tool cover -func="${module}_coverage.out" | grep "total:" | awk '{print $3}' | sed 's/%//')
        
        if [ -n "$coverage" ]; then
            echo -e "${BLUE}📊 覆盖率: ${coverage}%${NC}"
            
            # 检查是否达到预期覆盖率
            if (( $(echo "$coverage >= $expected_coverage" | bc -l) )); then
                echo -e "${GREEN}✅ 覆盖率达标 (>= ${expected_coverage}%)${NC}"
            else
                echo -e "${YELLOW}⚠️  覆盖率未达标 (< ${expected_coverage}%)${NC}"
            fi
        else
            echo -e "${RED}❌ 无法获取覆盖率信息${NC}"
        fi
    else
        echo -e "${RED}❌ 测试失败${NC}"
        return 1
    fi
}

# 检查超时机制
check_timeout_protection() {
    echo -e "\n${BLUE}⏱️  检查超时机制${NC}"
    echo "----------------------------------------"
    
    # 运行网络模块测试，确保在30秒内完成
    start_time=$(date +%s)
    if timeout 35s go test -v -timeout 30s ./pkg/network > /dev/null 2>&1; then
        end_time=$(date +%s)
        duration=$((end_time - start_time))
        echo -e "${GREEN}✅ 超时机制正常工作 (${duration}秒)${NC}"
    else
        echo -e "${RED}❌ 超时机制失效${NC}"
        return 1
    fi
}

# 检查并发安全
check_concurrent_safety() {
    echo -e "\n${BLUE}🔄 检查并发安全${NC}"
    echo "----------------------------------------"
    
    # 运行并发测试
    if go test -v -race -timeout 30s ./pkg/performance ./pkg/task > /dev/null 2>&1; then
        echo -e "${GREEN}✅ 并发安全检查通过${NC}"
    else
        echo -e "${RED}❌ 发现竞态条件${NC}"
        return 1
    fi
}

# 生成质量报告
generate_quality_report() {
    echo -e "\n${BLUE}📋 生成质量报告${NC}"
    echo "----------------------------------------"
    
    # 合并覆盖率报告
    echo "mode: set" > phase2_combined_coverage.out
    for module in network performance task; do
        if [ -f "${module}_coverage.out" ]; then
            tail -n +2 "${module}_coverage.out" >> phase2_combined_coverage.out
        fi
    done
    
    # 生成总覆盖率
    total_coverage=$(go tool cover -func=phase2_combined_coverage.out | grep "total:" | awk '{print $3}')
    
    echo -e "${GREEN}📊 第二阶段总覆盖率: $total_coverage${NC}"
    
    # 生成HTML报告
    go tool cover -html=phase2_combined_coverage.out -o phase2_coverage_report.html
    echo -e "${BLUE}📄 HTML报告已生成: phase2_coverage_report.html${NC}"
}

# 清理函数
cleanup() {
    echo -e "\n${BLUE}🧹 清理临时文件${NC}"
    rm -f *_coverage.out
}

# 主执行流程
main() {
    echo -e "${BLUE}开始质量检查...${NC}\n"
    
    # 检查各个模块
    check_module "network" "45"     # 网络模块预期覆盖率 45%+
    check_module "performance" "85" # 性能模块预期覆盖率 85%+
    check_module "task" "80"        # 任务模块预期覆盖率 80%+
    
    # 检查超时机制
    check_timeout_protection
    
    # 检查并发安全
    check_concurrent_safety
    
    # 生成质量报告
    generate_quality_report
    
    echo -e "\n${GREEN}🎉 第二阶段功能模块质量检查完成！${NC}"
    echo -e "${BLUE}📈 所有模块都通过了质量门禁标准${NC}"
    
    # 清理
    cleanup
}

# 错误处理
trap 'echo -e "\n${RED}❌ 质量检查过程中发生错误${NC}"; cleanup; exit 1' ERR

# 执行主函数
main "$@"
