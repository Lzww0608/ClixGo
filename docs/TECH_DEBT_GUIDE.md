# 技术债务处理实施指南

## 📋 总览
本指南提供了 ClixGo 项目技术债务处理的详细实施步骤，包括具体的命令、工具使用方法和代码示例。

## 🧪 测试覆盖率提升详细指南

### 当前状态分析
```bash
# 生成测试覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -1

# 生成HTML覆盖率报告
go tool cover -html=coverage.out -o coverage.html

# 查看各模块覆盖率详情
go tool cover -func=coverage.out | grep -v "100.0%"
```

### 第一阶段：核心模块测试补齐

#### 1. pkg/config 模块测试实施
```bash
# 创建测试文件
touch pkg/config/config_test.go
```

**测试模板：**
```go
package config

import (
    "io/ioutil"
    "os"
    "path/filepath"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestConfigLoad(t *testing.T) {
    // 创建临时配置文件
    tempDir, err := ioutil.TempDir("", "config-test")
    require.NoError(t, err)
    defer os.RemoveAll(tempDir)
    
    configFile := filepath.Join(tempDir, "config.yaml")
    configContent := `
debug: true
log_level: info
terminal:
  default_shell: /bin/bash
  max_sessions: 10
`
    err = ioutil.WriteFile(configFile, []byte(configContent), 0644)
    require.NoError(t, err)
    
    // 测试配置加载
    config, err := LoadConfig(configFile)
    assert.NoError(t, err)
    assert.True(t, config.Debug)
    assert.Equal(t, "info", config.LogLevel)
    assert.Equal(t, "/bin/bash", config.Terminal.DefaultShell)
    assert.Equal(t, 10, config.Terminal.MaxSessions)
}

func TestConfigValidation(t *testing.T) {
    tests := []struct {
        name        string
        config      Config
        expectError bool
    }{
        {
            name: "有效配置",
            config: Config{
                LogLevel: "info",
                Terminal: TerminalConfig{
                    DefaultShell: "/bin/bash",
                    MaxSessions: 5,
                },
            },
            expectError: false,
        },
        {
            name: "无效日志级别",
            config: Config{
                LogLevel: "invalid",
            },
            expectError: true,
        },
        {
            name: "会话数过大",
            config: Config{
                Terminal: TerminalConfig{
                    MaxSessions: 1000,
                },
            },
            expectError: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.Validate()
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

#### 2. pkg/terminal 模块测试实施
```bash
# 创建测试文件
touch pkg/terminal/terminal_test.go
touch pkg/terminal/session_test.go
touch pkg/terminal/pty_test.go
```

**PTY测试示例：**
```go
package terminal

import (
    "context"
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestPTYCreation(t *testing.T) {
    pty, err := NewPTY(&PTYConfig{
        Command: []string{"/bin/echo", "hello"},
        WorkDir: "/tmp",
        Env:     []string{"PATH=/bin:/usr/bin"},
    })
    require.NoError(t, err)
    defer pty.Close()
    
    assert.NotNil(t, pty.Process())
    assert.True(t, pty.IsRunning())
}

func TestPTYSignalHandling(t *testing.T) {
    pty, err := NewPTY(&PTYConfig{
        Command: []string{"/bin/sleep", "10"},
    })
    require.NoError(t, err)
    defer pty.Close()
    
    // 测试信号发送
    err = pty.SendSignal(syscall.SIGTERM)
    assert.NoError(t, err)
    
    // 等待进程结束
    done := make(chan bool)
    go func() {
        pty.Wait()
        done <- true
    }()
    
    select {
    case <-done:
        // 进程正确终止
    case <-time.After(5 * time.Second):
        t.Fatal("进程未在预期时间内终止")
    }
}

func TestSessionPersistence(t *testing.T) {
    sessionID := "test-session-" + time.Now().Format("20060102150405")
    
    // 创建会话
    session, err := NewSession(&SessionConfig{
        ID:      sessionID,
        Command: []string{"/bin/bash"},
        WorkDir: "/tmp",
    })
    require.NoError(t, err)
    
    // 保存会话状态
    err = session.Save()
    assert.NoError(t, err)
    
    // 模拟重启后恢复
    session.Close()
    
    restoredSession, err := RestoreSession(sessionID)
    assert.NoError(t, err)
    assert.Equal(t, sessionID, restoredSession.ID())
    
    // 清理
    defer restoredSession.Close()
    defer os.Remove(filepath.Join(GetSessionDir(), sessionID+".json"))
}
```

#### 3. pkg/ui 模块测试实施
```bash
# 创建测试文件
touch pkg/ui/ui_test.go
touch pkg/ui/layout_test.go
touch pkg/ui/events_test.go
```

**UI测试示例：**
```go
package ui

import (
    "testing"
    "github.com/gdamore/tcell/v2"
    "github.com/rivo/tview"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLayoutCreation(t *testing.T) {
    app := tview.NewApplication()
    ui := NewUI(app)
    
    layout := ui.CreateLayout()
    assert.NotNil(t, layout)
    
    // 测试默认分割
    assert.Equal(t, 2, layout.GetItemCount()) // 主面板 + 状态栏
}

func TestPanelSplit(t *testing.T) {
    app := tview.NewApplication()
    ui := NewUI(app)
    
    // 创建面板
    panel1 := ui.CreatePanel("panel1")
    panel2 := ui.CreatePanel("panel2")
    
    // 测试水平分割
    splitView := ui.SplitHorizontal(panel1, panel2, 0.5)
    assert.NotNil(t, splitView)
    
    // 测试垂直分割
    splitView = ui.SplitVertical(panel1, panel2, 0.7)
    assert.NotNil(t, splitView)
}

func TestKeyboardEvents(t *testing.T) {
    app := tview.NewApplication()
    ui := NewUI(app)
    
    eventReceived := false
    ui.SetKeyHandler(func(event *tcell.EventKey) {
        if event.Key() == tcell.KeyCtrlC {
            eventReceived = true
        }
    })
    
    // 模拟按键事件
    event := tcell.NewEventKey(tcell.KeyCtrlC, 'c', tcell.ModCtrl)
    ui.HandleEvent(event)
    
    assert.True(t, eventReceived, "应该接收到Ctrl+C事件")
}
```

### 测试自动化设置

#### 1. 创建测试运行脚本
```bash
# 创建 scripts/test.sh
cat > scripts/test.sh << 'EOF'
#!/bin/bash

set -e

echo "🧪 运行单元测试..."
go test -v ./...

echo "📊 生成覆盖率报告..."
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

echo "📈 当前覆盖率："
go tool cover -func=coverage.out | tail -1

echo "🌐 生成HTML报告..."
go tool cover -html=coverage.out -o coverage.html
echo "覆盖率报告已生成：coverage.html"

# 检查覆盖率门槛
COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | sed 's/%//')
THRESHOLD=90

if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
    echo "❌ 覆盖率 $COVERAGE% 低于目标 $THRESHOLD%"
    exit 1
else
    echo "✅ 覆盖率 $COVERAGE% 达到目标"
fi
EOF

chmod +x scripts/test.sh
```

#### 2. 设置GitHub Actions
```yaml
# .github/workflows/test.yml
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
    - uses: actions/checkout@v3
    
    - name: 设置Go环境
      uses: actions/setup-go@v3
      with:
        go-version: 1.23
        
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
      run: ./scripts/test.sh
      
    - name: 上传覆盖率报告
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
        flags: unittests
        name: codecov-umbrella
```

## 🏗️ 性能基准测试实施

### 1. 创建基准测试框架
```go
// pkg/performance/benchmark_test.go
package performance

import (
    "testing"
    "time"
)

func BenchmarkTerminalCreation(b *testing.B) {
    for i := 0; i < b.N; i++ {
        terminal, err := NewTerminal(&Config{
            Shell: "/bin/bash",
        })
        if err != nil {
            b.Fatal(err)
        }
        terminal.Close()
    }
}

func BenchmarkUIRendering(b *testing.B) {
    ui := NewUI()
    defer ui.Close()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        ui.Render()
    }
}

func BenchmarkTextProcessing(b *testing.B) {
    text := generateLargeText(10000) // 生成10k字符文本
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = ProcessText(text)
    }
}

// 内存分配基准测试
func BenchmarkMemoryAllocation(b *testing.B) {
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        data := make([]byte, 1024)
        _ = processData(data)
    }
}
```

### 2. 性能监控脚本
```bash
# scripts/benchmark.sh
#!/bin/bash

echo "🚀 运行性能基准测试..."

# 运行基准测试
go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./...

echo "📊 生成性能报告..."

# CPU性能分析
echo "=== CPU 性能分析 ==="
go tool pprof -text cpu.prof | head -20

# 内存性能分析
echo "=== 内存 性能分析 ==="
go tool pprof -text mem.prof | head -20

# 生成火焰图（需要安装 go-torch）
if command -v go-torch &> /dev/null; then
    echo "🔥 生成火焰图..."
    go-torch -b cpu.prof -f cpu_flame.svg
    go-torch -alloc_space mem.prof -f mem_flame.svg
fi

echo "✅ 性能分析完成"
```

## 🔧 架构重构实施指南

### 1. 错误处理标准化

#### 创建标准错误类型
```go
// pkg/errors/errors.go
package errors

import (
    "fmt"
    "time"
)

// 错误码常量
const (
    ErrCodeInternal     = "INTERNAL_ERROR"
    ErrCodeValidation   = "VALIDATION_ERROR"
    ErrCodeNotFound     = "NOT_FOUND"
    ErrCodePermission   = "PERMISSION_DENIED"
    ErrCodeNetwork      = "NETWORK_ERROR"
    ErrCodeTimeout      = "TIMEOUT_ERROR"
)

// ClixGoError 标准错误类型
type ClixGoError struct {
    Code     string                 `json:"code"`
    Message  string                 `json:"message"`
    Context  map[string]interface{} `json:"context,omitempty"`
    Cause    error                  `json:"cause,omitempty"`
    Time     time.Time              `json:"time"`
    Stack    string                 `json:"stack,omitempty"`
}

func (e *ClixGoError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// New 创建新的标准错误
func New(code, message string) *ClixGoError {
    return &ClixGoError{
        Code:    code,
        Message: message,
        Time:    time.Now(),
        Context: make(map[string]interface{}),
    }
}

// Wrap 包装现有错误
func Wrap(err error, code, message string) *ClixGoError {
    return &ClixGoError{
        Code:    code,
        Message: message,
        Cause:   err,
        Time:    time.Now(),
        Context: make(map[string]interface{}),
    }
}

// WithContext 添加上下文信息
func (e *ClixGoError) WithContext(key string, value interface{}) *ClixGoError {
    e.Context[key] = value
    return e
}
```

#### 错误处理中间件
```go
// pkg/middleware/error.go
package middleware

import (
    "context"
    "runtime/debug"
    
    "github.com/Lzww0608/ClixGo/pkg/errors"
    "github.com/Lzww0608/ClixGo/pkg/logger"
)

// ErrorHandler 错误处理中间件
type ErrorHandler struct {
    logger logger.Logger
}

func NewErrorHandler(log logger.Logger) *ErrorHandler {
    return &ErrorHandler{logger: log}
}

// Recover 恢复panic并转换为标准错误
func (h *ErrorHandler) Recover() {
    if r := recover(); r != nil {
        stack := string(debug.Stack())
        
        var err *errors.ClixGoError
        switch v := r.(type) {
        case error:
            err = errors.Wrap(v, errors.ErrCodeInternal, "系统内部错误")
        case string:
            err = errors.New(errors.ErrCodeInternal, v)
        default:
            err = errors.New(errors.ErrCodeInternal, "未知错误")
        }
        
        err.Stack = stack
        h.logger.Error("panic recovered", 
            logger.String("error", err.Error()),
            logger.String("stack", stack),
        )
    }
}

// HandleError 统一错误处理
func (h *ErrorHandler) HandleError(ctx context.Context, err error) error {
    if err == nil {
        return nil
    }
    
    var clixErr *errors.ClixGoError
    switch v := err.(type) {
    case *errors.ClixGoError:
        clixErr = v
    default:
        clixErr = errors.Wrap(err, errors.ErrCodeInternal, "内部处理错误")
    }
    
    // 记录错误日志
    h.logger.Error("operation failed",
        logger.String("code", clixErr.Code),
        logger.String("message", clixErr.Message),
        logger.Any("context", clixErr.Context),
    )
    
    return clixErr
}
```

### 2. 日志系统统一

#### 标准日志接口
```go
// pkg/logger/interface.go
package logger

import "context"

type Field interface {
    Key() string
    Value() interface{}
}

type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    Fatal(msg string, fields ...Field)
    
    WithFields(fields map[string]interface{}) Logger
    WithContext(ctx context.Context) Logger
}

// 便捷的Field构造函数
func String(key, value string) Field {
    return &stringField{key: key, value: value}
}

func Int(key string, value int) Field {
    return &intField{key: key, value: value}
}

func Any(key string, value interface{}) Field {
    return &anyField{key: key, value: value}
}

// Field实现
type stringField struct {
    key   string
    value string
}

func (f *stringField) Key() string      { return f.key }
func (f *stringField) Value() interface{} { return f.value }

type intField struct {
    key   string
    value int
}

func (f *intField) Key() string      { return f.key }
func (f *intField) Value() interface{} { return f.value }

type anyField struct {
    key   string
    value interface{}
}

func (f *anyField) Key() string      { return f.key }
func (f *anyField) Value() interface{} { return f.value }
```

### 3. 依赖注入容器
```go
// pkg/container/container.go
package container

import (
    "fmt"
    "reflect"
    "sync"
)

type Container struct {
    services map[string]interface{}
    mu       sync.RWMutex
}

func New() *Container {
    return &Container{
        services: make(map[string]interface{}),
    }
}

// Register 注册服务
func (c *Container) Register(name string, service interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.services[name] = service
}

// RegisterSingleton 注册单例服务
func (c *Container) RegisterSingleton(name string, factory func() interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // 延迟创建
    c.services[name] = &singleton{factory: factory}
}

// Get 获取服务
func (c *Container) Get(name string) (interface{}, error) {
    c.mu.RLock()
    service, exists := c.services[name]
    c.mu.RUnlock()
    
    if !exists {
        return nil, fmt.Errorf("service '%s' not found", name)
    }
    
    if s, ok := service.(*singleton); ok {
        return s.getInstance(), nil
    }
    
    return service, nil
}

// Inject 自动注入依赖
func (c *Container) Inject(target interface{}) error {
    v := reflect.ValueOf(target)
    if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
        return fmt.Errorf("target must be a pointer to struct")
    }
    
    elem := v.Elem()
    typ := elem.Type()
    
    for i := 0; i < elem.NumField(); i++ {
        field := elem.Field(i)
        fieldType := typ.Field(i)
        
        // 检查inject标签
        if tag := fieldType.Tag.Get("inject"); tag != "" {
            if service, err := c.Get(tag); err == nil {
                if field.CanSet() && reflect.TypeOf(service).AssignableTo(field.Type()) {
                    field.Set(reflect.ValueOf(service))
                }
            }
        }
    }
    
    return nil
}

type singleton struct {
    factory  func() interface{}
    instance interface{}
    once     sync.Once
}

func (s *singleton) getInstance() interface{} {
    s.once.Do(func() {
        s.instance = s.factory()
    })
    return s.instance
}
```

## 📊 进度跟踪和质量门禁

### 1. 质量检查脚本
```bash
# scripts/quality-check.sh
#!/bin/bash

set -e

echo "🔍 代码质量检查..."

# 1. 代码格式检查
echo "📝 检查代码格式..."
if ! gofmt -l . | grep -q '^$'; then
    echo "❌ 代码格式不符合规范"
    gofmt -l .
    exit 1
fi

# 2. 代码质量检查
echo "🔍 运行go vet..."
go vet ./...

# 3. 安全扫描
echo "🔒 安全扫描..."
if command -v gosec &> /dev/null; then
    gosec ./...
fi

# 4. 依赖检查
echo "📦 检查依赖..."
go mod tidy
go mod verify

# 5. 测试覆盖率检查
echo "🧪 检查测试覆盖率..."
./scripts/test.sh

echo "✅ 所有质量检查通过"
```

### 2. 预提交钩子
```bash
# .git/hooks/pre-commit
#!/bin/bash

echo "🚀 运行预提交检查..."

# 运行质量检查
if ! ./scripts/quality-check.sh; then
    echo "❌ 质量检查失败，请修复后重新提交"
    exit 1
fi

echo "✅ 预提交检查通过"
```

这个详细的技术债务处理计划现在为您提供了：

1. **具体的执行步骤** - 每个任务都有明确的实施方法
2. **代码示例** - 提供了实际可用的测试代码模板
3. **自动化工具** - 包含脚本和CI/CD配置
4. **质量门禁** - 确保代码质量的检查机制
5. **进度跟踪** - 明确的里程碑和完成标准

您可以按照这个计划逐步实施，每完成一个阶段就能看到明显的改进效果。建议从测试覆盖率最低的模块开始，优先处理核心功能模块。 