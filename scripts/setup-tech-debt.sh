#!/bin/bash

# ClixGo 技术债务处理一键设置脚本
# 作者: ClixGo 开发团队
# 用途: 快速初始化技术债务处理所需的目录结构、脚本和配置文件

set -e

echo "🚀 ClixGo 技术债务处理环境设置"
echo "================================="

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 函数定义
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 检查必要工具
check_dependencies() {
    echo "🔍 检查依赖工具..."
    
    if ! command -v go &> /dev/null; then
        print_error "Go 未安装，请先安装 Go 1.23+"
        exit 1
    fi
    
    GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+')
    if [[ "$(printf '%s\n' "1.23" "$GO_VERSION" | sort -V | head -n1)" != "1.23" ]]; then
        print_warning "推荐使用 Go 1.23+，当前版本: $GO_VERSION"
    fi
    
    print_success "Go 环境检查通过"
}

# 创建目录结构
create_directories() {
    echo "📁 创建目录结构..."
    
    directories=(
        "scripts"
        ".github/workflows"
        "docs"
        "pkg/errors"
        "pkg/middleware"
        "pkg/container"
    )
    
    for dir in "${directories[@]}"; do
        mkdir -p "$dir"
        print_success "创建目录: $dir"
    done
}

# 创建测试脚本
create_test_scripts() {
    echo "📝 创建测试脚本..."
    
    # 测试脚本
    cat > scripts/test.sh << 'EOF'
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
EOF
    chmod +x scripts/test.sh
    print_success "创建测试脚本: scripts/test.sh"
    
    # 基准测试脚本
    cat > scripts/benchmark.sh << 'EOF'
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
EOF
    chmod +x scripts/benchmark.sh
    print_success "创建基准测试脚本: scripts/benchmark.sh"
}

# 创建质量检查脚本
create_quality_scripts() {
    echo "🔍 创建质量检查脚本..."
    
    cat > scripts/quality-check.sh << 'EOF'
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
EOF
    chmod +x scripts/quality-check.sh
    print_success "创建质量检查脚本: scripts/quality-check.sh"
}

# 创建GitHub Actions配置
create_github_actions() {
    echo "🔧 创建GitHub Actions配置..."
    
    cat > .github/workflows/test.yml << 'EOF'
name: 测试和覆盖率检查

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v4
    
    - name: 设置Go环境
      uses: actions/setup-go@v4
      with:
        go-version: '1.23'
        
    - name: 缓存Go模块
      uses: actions/cache@v3
      with:
        path: |
          ~/go/pkg/mod
          ~/.cache/go-build
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        
    - name: 安装依赖
      run: go mod download
      
    - name: 运行测试
      run: |
        chmod +x scripts/test.sh
        ./scripts/test.sh
      
    - name: 上传覆盖率报告
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
        flags: unittests
        name: codecov-umbrella
      continue-on-error: true

  quality:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v4
    
    - name: 设置Go环境
      uses: actions/setup-go@v4
      with:
        go-version: '1.23'
        
    - name: 安装质量检查工具
      run: |
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
        
    - name: 运行质量检查
      run: |
        chmod +x scripts/quality-check.sh
        ./scripts/quality-check.sh
EOF
    print_success "创建GitHub Actions配置: .github/workflows/test.yml"
}

# 创建Git hooks
create_git_hooks() {
    echo "🪝 创建Git hooks..."
    
    if [ -d .git ]; then
        cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash

echo "🚀 运行预提交检查..."

# 运行质量检查
if [ -f scripts/quality-check.sh ]; then
    if ! ./scripts/quality-check.sh; then
        echo "❌ 质量检查失败，请修复后重新提交"
        exit 1
    fi
else
    echo "⚠️  质量检查脚本不存在，跳过检查"
fi

echo "✅ 预提交检查通过"
EOF
        chmod +x .git/hooks/pre-commit
        print_success "创建Git pre-commit hook"
    else
        print_warning "不是Git仓库，跳过Git hooks创建"
    fi
}

# 创建测试文件模板
create_test_templates() {
    echo "📝 创建测试文件模板..."
    
    # 如果config模块没有测试文件，创建一个
    if [ ! -f pkg/config/config_test.go ] && [ -d pkg/config ]; then
        cat > pkg/config/config_test.go << 'EOF'
package config

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestConfigBasic(t *testing.T) {
    // TODO: 实现配置加载测试
    assert.True(t, true, "基础测试模板")
}

// TODO: 添加更多配置测试
// func TestConfigLoad(t *testing.T) { ... }
// func TestConfigValidation(t *testing.T) { ... }
// func TestConfigSave(t *testing.T) { ... }
EOF
        print_success "创建配置模块测试模板: pkg/config/config_test.go"
    fi
    
    # 为其他模块创建测试文件模板的脚本
    cat > scripts/create-test-template.sh << 'EOF'
#!/bin/bash

if [ $# -eq 0 ]; then
    echo "用法: $0 <包路径>"
    echo "示例: $0 pkg/terminal"
    exit 1
fi

PACKAGE_PATH=$1
PACKAGE_NAME=$(basename "$PACKAGE_PATH")
TEST_FILE="$PACKAGE_PATH/${PACKAGE_NAME}_test.go"

if [ -f "$TEST_FILE" ]; then
    echo "测试文件已存在: $TEST_FILE"
    exit 1
fi

mkdir -p "$PACKAGE_PATH"

cat > "$TEST_FILE" << EOL
package $PACKAGE_NAME

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func Test${PACKAGE_NAME^}Basic(t *testing.T) {
    // TODO: 实现 $PACKAGE_NAME 模块的基础测试
    assert.True(t, true, "基础测试模板")
}

// TODO: 添加更多测试函数
// func Test${PACKAGE_NAME^}Function1(t *testing.T) { ... }
// func Test${PACKAGE_NAME^}Function2(t *testing.T) { ... }
EOL

echo "✅ 创建测试文件模板: $TEST_FILE"
EOF
    chmod +x scripts/create-test-template.sh
    print_success "创建测试模板生成脚本: scripts/create-test-template.sh"
}

# 安装测试依赖
install_test_dependencies() {
    echo "📦 安装测试依赖..."
    
    # 检查go.mod是否存在
    if [ ! -f go.mod ]; then
        print_warning "go.mod不存在，跳过依赖安装"
        return
    fi
    
    # 添加testify依赖（如果还没有）
    if ! grep -q "github.com/stretchr/testify" go.mod; then
        echo "安装testify测试框架..."
        go get github.com/stretchr/testify@latest
        print_success "安装testify完成"
    fi
    
    # 整理依赖
    go mod tidy
    print_success "依赖整理完成"
}

# 创建技术债务跟踪文件
create_tracking_files() {
    echo "📋 创建技术债务跟踪文件..."
    
    cat > TECH_DEBT_PROGRESS.md << 'EOF'
# 技术债务处理进度跟踪

## 📊 总体进度
- [ ] 测试覆盖率达到90%+
- [ ] 架构重构完成
- [ ] 性能基准测试建立
- [ ] 文档完善

## 🧪 测试覆盖率进度 (当前: 25.3%)

### 第一阶段：核心模块 (Week 1-2)
- [ ] pkg/config 模块测试 (优先级：高)
- [ ] pkg/terminal 模块测试 (优先级：高)  
- [ ] pkg/ui 模块测试 (优先级：高)

### 第二阶段：功能模块 (Week 3-4)
- [ ] pkg/network 模块测试 (优先级：中)
- [ ] pkg/performance 模块测试 (优先级：中)
- [ ] pkg/task 模块测试 (优先级：中)

### 第三阶段：命令行工具 (Week 5-6)
- [ ] cmd/cli 测试
- [ ] cmd/perfmonitor 测试
- [ ] cmd/netmonitor 测试
- [ ] cmd/task 测试

## 🏗️ 架构重构进度

### 错误处理标准化
- [ ] 创建标准错误类型
- [ ] 实现错误处理中间件
- [ ] 统一错误码规范

### 日志系统统一
- [ ] 定义统一日志接口
- [ ] 实现分层日志管理
- [ ] 日志聚合和分析

### 依赖注入改进
- [ ] 实现IoC容器
- [ ] 接口与实现分离
- [ ] 配置驱动的依赖注入

## 📝 更新日志

### $(date '+%Y-%m-%d')
- ✅ 初始化技术债务处理环境
- ✅ 创建测试脚本和质量检查工具
- ✅ 设置GitHub Actions CI/CD

---
*使用 scripts/update-progress.sh 来更新此文件*
EOF
    print_success "创建技术债务跟踪文件: TECH_DEBT_PROGRESS.md"
}

# 主函数
main() {
    echo "开始设置技术债务处理环境..."
    echo
    
    check_dependencies
    create_directories
    create_test_scripts
    create_quality_scripts
    create_github_actions
    create_git_hooks
    create_test_templates
    install_test_dependencies
    create_tracking_files
    
    echo
    echo "🎉 技术债务处理环境设置完成！"
    echo
    echo "📋 下一步操作建议："
    echo "1. 运行 ./scripts/test.sh 检查当前测试状态"
    echo "2. 运行 ./scripts/quality-check.sh 进行代码质量检查"
    echo "3. 使用 ./scripts/create-test-template.sh <包路径> 为模块创建测试文件"
    echo "4. 查看 TECH_DEBT_PROGRESS.md 跟踪处理进度"
    echo "5. 参考 docs/TECH_DEBT_GUIDE.md 获取详细实施指导"
    echo
    echo "🚀 开始您的技术债务处理之旅吧！"
}

# 运行主函数
main "$@" 