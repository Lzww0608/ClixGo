# ClixGo 项目依赖关系分析报告

**日期**: 2025-06-09
**分析目标**: ClixGo 项目模块化重构和依赖关系优化

## 📊 项目概览

### 模块统计
- **总模块数**: 23个核心模块
- **应用层模块**: 5个 (cmd/*)
- **业务逻辑模块**: 13个 (pkg/*)
- **UI模块**: 2个 (terminal, ui)
- **基础设施模块**: 3个 (errors, logger, config)

### 依赖关系健康度
- ✅ **循环依赖**: 0个 (优秀)
- ⚠️ **高耦合模块**: 1个 (cmd/cli - 13个依赖)
- ✅ **核心模块稳定性**: 良好
- 🔧 **需要优化**: 接口抽象和依赖注入

## 🏗️ 架构分层分析

### 当前分层结构

```
应用层 (Application Layer)
├── cmd/cli (13依赖)
├── cmd/netmonitor (1依赖)
├── cmd/perfmonitor (2依赖)
├── cmd/task (1依赖)
└── cmd/testseg (0内部依赖)

业务层 (Business Layer)
├── pkg/commands (5依赖)
├── pkg/network (1依赖)
├── pkg/performance (1依赖)
├── pkg/task (0内部依赖)
├── pkg/terminal (3依赖)
└── pkg/ui (0内部依赖)

服务层 (Service Layer)
├── pkg/alias (2依赖)
├── pkg/history (0内部依赖)
├── pkg/filesystem (0内部依赖)
├── pkg/security (0内部依赖)
├── pkg/text (0内部依赖)
├── pkg/completion (0内部依赖)
├── pkg/plugin (0内部依赖)
└── pkg/middleware (2依赖)

基础设施层 (Infrastructure Layer)
├── pkg/errors (0依赖) ⭐️
├── pkg/logger (0依赖) ⭐️
├── pkg/config (0依赖) ⭐️
├── pkg/utils (1依赖)
└── pkg/sync (0依赖)
```

## 🔍 问题识别

### 1. 高耦合模块

#### cmd/cli (⚠️ 13个依赖)
**问题**: 违反单一职责原则，承担过多功能
```go
// 当前依赖 (过多)
import (
    "github.com/Lzww0608/ClixGo/pkg/alias"
    "github.com/Lzww0608/ClixGo/pkg/commands"
    "github.com/Lzww0608/ClixGo/pkg/completion"
    "github.com/Lzww0608/ClixGo/pkg/config"
    "github.com/Lzww0608/ClixGo/pkg/filesystem"
    "github.com/Lzww0608/ClixGo/pkg/history"
    "github.com/Lzww0608/ClixGo/pkg/logger"
    "github.com/Lzww0608/ClixGo/pkg/network"
    "github.com/Lzww0608/ClixGo/pkg/security"
    "github.com/Lzww0608/ClixGo/pkg/terminal"
    "github.com/Lzww0608/ClixGo/pkg/text"
    "github.com/Lzww0608/ClixGo/pkg/utils"
    "github.com/Lzww0608/ClixGo/cmd/task"
)
```

**影响**:
- 编译时间增长
- 测试复杂度增加
- 维护困难
- 违反依赖倒置原则

### 2. 被高度依赖的模块

#### pkg/logger (🎯 6个模块依赖)
```
pkg/logger ← [
    pkg/commands,
    pkg/middleware, 
    pkg/performance,
    pkg/terminal,
    pkg/terminal/ui,
    cmd/cli
]
```

#### pkg/errors (🎯 5个模块依赖)  
```
pkg/errors ← [
    pkg/alias,
    pkg/commands,
    pkg/middleware,
    pkg/terminal,
    pkg/utils
]
```

#### pkg/utils (🎯 4个模块依赖)
```
pkg/utils ← [
    pkg/alias,
    pkg/commands,
    pkg/terminal,
    cmd/cli
]
```

## 🚀 架构优化方案

### Phase 1: 接口抽象 (Week 1-2)

#### 1.1 核心接口定义
创建 `pkg/interfaces/` 包，定义核心接口：

```go
// pkg/interfaces/logger.go
type Logger interface {
    Info(msg string, fields ...zap.Field)
    Error(msg string, fields ...zap.Field)
    Debug(msg string, fields ...zap.Field)
    Warn(msg string, fields ...zap.Field)
    With(fields ...zap.Field) Logger
}

// pkg/interfaces/error_handler.go
type ErrorHandler interface {
    Handle(err error) error
    Wrap(err error, msg string) error
    New(msg string) error
    WithCode(code ErrorCode, msg string) error
}

// pkg/interfaces/utils.go
type Validator interface {
    ValidateNotEmpty(value string, name string) error
    ValidateInRange(value, min, max int, name string) error
}

type FileHelper interface {
    SafeWriteFile(filename string, data []byte) error
    EnsureDir(dir string) error
    GetFileSize(filename string) (int64, error)
}
```

#### 1.2 依赖注入容器
创建 `pkg/container/` 包：

```go
// pkg/container/container.go
type Container struct {
    services map[string]interface{}
    mu       sync.RWMutex
}

func (c *Container) Register(name string, service interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.services[name] = service
}

func (c *Container) Get(name string) (interface{}, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    if service, exists := c.services[name]; exists {
        return service, nil
    }
    return nil, fmt.Errorf("service %s not found", name)
}

// 类型安全的获取方法
func GetLogger(c *Container) (interfaces.Logger, error) {
    service, err := c.Get("logger")
    if err != nil {
        return nil, err
    }
    
    if logger, ok := service.(interfaces.Logger); ok {
        return logger, nil
    }
    return nil, fmt.Errorf("service logger is not of expected type")
}
```

### Phase 2: CLI模块重构 (Week 2-3)

#### 2.1 按功能域拆分CLI
```
cmd/cli/
├── core/           # 核心CLI功能
│   ├── root.go
│   └── version.go
├── terminal/       # 终端相关命令
│   ├── session.go
│   └── multiplexer.go
├── tools/          # 工具命令
│   ├── text.go
│   ├── network.go
│   └── filesystem.go
├── system/         # 系统管理命令
│   ├── security.go
│   └── history.go
└── app/            # 应用集成
    └── registry.go  # 命令注册器
```

#### 2.2 命令注册器模式
```go
// cmd/cli/app/registry.go
type CommandRegistry struct {
    container *container.Container
    commands  []CommandBuilder
}

type CommandBuilder interface {
    Build(container *container.Container) *cobra.Command
    Dependencies() []string
}

// cmd/cli/terminal/session_cmd.go
type SessionCommandBuilder struct{}

func (s *SessionCommandBuilder) Dependencies() []string {
    return []string{"terminal_service", "logger"}
}

func (s *SessionCommandBuilder) Build(c *container.Container) *cobra.Command {
    terminalService, _ := GetTerminalService(c)
    logger, _ := GetLogger(c)
    
    return &cobra.Command{
        Use:   "session",
        Short: "Manage terminal sessions",
        RunE: func(cmd *cobra.Command, args []string) error {
            return s.runSession(terminalService, logger, args)
        },
    }
}
```

### Phase 3: 服务层抽象 (Week 3-4)

#### 3.1 服务接口定义
```go
// pkg/interfaces/services.go
type TerminalService interface {
    CreateSession(name string) (Session, error)
    GetSession(id string) (Session, error)
    ListSessions() ([]Session, error)
    CloseSession(id string) error
}

type NetworkService interface {
    Ping(host string, timeout time.Duration) (time.Duration, error)
    CheckPort(host string, port int) (bool, error)
    MonitorBandwidth(interval time.Duration) (<-chan BandwidthInfo, error)
}

type PerformanceService interface {
    GetSystemInfo() (SystemInfo, error)
    MonitorResources(interval time.Duration) (<-chan ResourceInfo, error)
    OptimizeMemory() error
}

type TaskService interface {
    CreateTask(task Task) (string, error)
    GetTask(id string) (Task, error)
    ListTasks(filter TaskFilter) ([]Task, error)
    CancelTask(id string) error
}
```

#### 3.2 服务实现
```go
// pkg/terminal/service.go
type terminalService struct {
    logger interfaces.Logger
    config interfaces.Config
    sessions map[string]*Session
    mu       sync.RWMutex
}

func NewTerminalService(logger interfaces.Logger, config interfaces.Config) interfaces.TerminalService {
    return &terminalService{
        logger:   logger,
        config:   config,
        sessions: make(map[string]*Session),
    }
}

func (s *terminalService) CreateSession(name string) (Session, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    session := &Session{
        ID:        uuid.New().String(),
        Name:      name,
        CreatedAt: time.Now(),
    }
    
    s.sessions[session.ID] = session
    s.logger.Info("Session created", zap.String("id", session.ID), zap.String("name", name))
    
    return *session, nil
}
```

### Phase 4: 事件驱动架构 (Week 4-5)

#### 4.1 事件系统
```go
// pkg/events/event_bus.go
type EventBus interface {
    Subscribe(eventType string, handler EventHandler) error
    Unsubscribe(eventType string, handler EventHandler) error
    Publish(event Event) error
    PublishAsync(event Event) error
}

type Event interface {
    Type() string
    Timestamp() time.Time
    Data() interface{}
}

type EventHandler interface {
    Handle(event Event) error
}

// 具体事件类型
type SessionCreated struct {
    SessionID string    `json:"session_id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}

func (s SessionCreated) Type() string { return "session.created" }
func (s SessionCreated) Timestamp() time.Time { return s.CreatedAt }
func (s SessionCreated) Data() interface{} { return s }
```

#### 4.2 事件驱动的模块通信
```go
// pkg/terminal/event_handlers.go
type SessionEventHandler struct {
    logger    interfaces.Logger
    metrics   interfaces.MetricsCollector
}

func (h *SessionEventHandler) Handle(event events.Event) error {
    switch event.Type() {
    case "session.created":
        return h.handleSessionCreated(event)
    case "session.closed":
        return h.handleSessionClosed(event)
    default:
        return nil
    }
}

func (h *SessionEventHandler) handleSessionCreated(event events.Event) error {
    data := event.Data().(SessionCreated)
    h.logger.Info("Handling session creation", zap.String("session_id", data.SessionID))
    
    // 更新指标
    h.metrics.IncCounter("sessions.created.total")
    
    return nil
}
```
