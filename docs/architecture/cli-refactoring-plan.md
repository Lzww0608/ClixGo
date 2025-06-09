# CLI模块重构计划 - 架构优化 Phase 2

## 📋 概述

根据依赖关系分析，`cmd/cli`模块是项目中耦合度最高的模块，拥有13个内部依赖。本计划旨在通过重构CLI模块来降低耦合度，提高代码的可维护性和可测试性。

## 🎯 目标

1. **降低耦合度**: 将CLI模块的依赖从13个减少到5个以下
2. **提高可测试性**: 通过依赖注入使CLI组件更容易进行单元测试
3. **增强可维护性**: 清晰的职责分离和模块边界
4. **保持功能完整性**: 确保重构后所有CLI功能正常工作

## 📊 当前状态分析

### 当前依赖关系
```
cmd/cli 依赖:
- cmd/task
- pkg/alias
- pkg/commands
- pkg/completion
- pkg/config
- pkg/filesystem
- pkg/history
- pkg/logger
- pkg/network
- pkg/security
- pkg/terminal
- pkg/text
- pkg/utils
```

### 问题识别
1. **直接依赖过多**: CLI直接依赖了大量业务模块
2. **职责混乱**: CLI既处理命令解析又直接调用业务逻辑
3. **测试困难**: 大量依赖使得单元测试复杂
4. **扩展性差**: 添加新功能需要修改CLI核心代码

## 🏗️ 重构策略

### 1. 分层架构设计

```
┌─────────────────────────────────────┐
│           CLI Layer                 │
│  (命令解析、参数验证、输出格式化)      │
└─────────────────────────────────────┘
                    │
┌─────────────────────────────────────┐
│        Application Layer            │
│     (应用服务、业务流程编排)          │
└─────────────────────────────────────┘
                    │
┌─────────────────────────────────────┐
│         Service Layer               │
│    (业务服务、领域逻辑)               │
└─────────────────────────────────────┘
                    │
┌─────────────────────────────────────┐
│      Infrastructure Layer          │
│   (数据访问、外部系统集成)            │
└─────────────────────────────────────┘
```

### 2. 服务抽象

#### 2.1 应用服务接口
```go
// pkg/interfaces/application.go
type ApplicationService interface {
    // 终端管理
    CreateTerminalSession(name string) error
    ListTerminalSessions() ([]SessionInfo, error)
    
    // 网络诊断
    RunNetworkDiagnostics(target string) (*NetworkReport, error)
    
    // 任务管理
    CreateTask(name, command string) (string, error)
    GetTaskStatus(id string) (*TaskStatus, error)
    
    // 文件操作
    ProcessFiles(pattern string, operation FileOperation) error
}
```

#### 2.2 CLI服务接口
```go
// pkg/interfaces/cli.go
type CLIService interface {
    // 命令注册
    RegisterCommand(cmd Command) error
    
    // 命令执行
    ExecuteCommand(args []string) error
    
    // 帮助系统
    ShowHelp(command string) error
    
    // 自动补全
    GetCompletions(partial string) ([]string, error)
}
```

### 3. 依赖注入重构

#### 3.1 CLI容器配置
```go
// cmd/cli/container.go
func SetupCLIContainer() *container.Container {
    c := container.NewContainer()
    
    // 注册核心服务
    c.RegisterSingleton("logger", createLogger)
    c.RegisterSingleton("config", createConfig)
    
    // 注册应用服务
    c.RegisterSingleton("terminal_app", createTerminalApp)
    c.RegisterSingleton("network_app", createNetworkApp)
    c.RegisterSingleton("task_app", createTaskApp)
    
    // 注册CLI服务
    c.RegisterSingleton("cli", createCLIService)
    
    return c
}
```

#### 3.2 命令重构
```go
// cmd/cli/commands/terminal.go
type TerminalCommand struct {
    app interfaces.TerminalApplicationService
    logger interfaces.Logger
}

func NewTerminalCommand(app interfaces.TerminalApplicationService, logger interfaces.Logger) *TerminalCommand {
    return &TerminalCommand{app: app, logger: logger}
}

func (t *TerminalCommand) Execute(args []string) error {
    // 只处理CLI逻辑，业务逻辑委托给应用服务
    return t.app.CreateSession(args[0])
}
```

## 📅 实施计划

### Week 14: 接口定义和应用服务层 (第1周)

#### Day 1-2: 定义应用服务接口
- [ ] 创建 `pkg/interfaces/application.go`
- [ ] 定义终端应用服务接口
- [ ] 定义网络应用服务接口
- [ ] 定义任务应用服务接口
- [ ] 定义文件应用服务接口

#### Day 3-4: 实现应用服务
- [ ] 创建 `pkg/application/terminal_service.go`
- [ ] 创建 `pkg/application/network_service.go`
- [ ] 创建 `pkg/application/task_service.go`
- [ ] 创建 `pkg/application/file_service.go`

#### Day 5-7: 应用服务测试
- [ ] 编写应用服务单元测试
- [ ] 集成测试验证
- [ ] 性能基准测试

### Week 15: CLI层重构 (第2周)

#### Day 1-3: CLI框架重构
- [ ] 重构 `cmd/cli/main.go` 使用依赖注入
- [ ] 创建 `cmd/cli/container.go` 容器配置
- [ ] 重构命令注册机制

#### Day 4-5: 命令重构
- [ ] 重构终端相关命令
- [ ] 重构网络相关命令
- [ ] 重构任务相关命令

#### Day 6-7: 测试和验证
- [ ] CLI功能测试
- [ ] 集成测试
- [ ] 性能回归测试

## 🧪 测试策略

### 1. 单元测试
```go
func TestTerminalCommand_Execute(t *testing.T) {
    // 使用mock应用服务
    mockApp := &MockTerminalApp{}
    mockLogger := &MockLogger{}
    
    cmd := NewTerminalCommand(mockApp, mockLogger)
    
    err := cmd.Execute([]string{"test-session"})
    assert.NoError(t, err)
    assert.True(t, mockApp.CreateSessionCalled)
}
```

### 2. 集成测试
```go
func TestCLI_Integration(t *testing.T) {
    container := SetupTestContainer()
    cli, _ := container.Get("cli")
    
    err := cli.ExecuteCommand([]string{"terminal", "create", "test"})
    assert.NoError(t, err)
}
```

### 3. 端到端测试
```bash
#!/bin/bash
# 测试CLI命令的完整流程
./clixgo terminal create test-session
./clixgo terminal list
./clixgo terminal attach test-session
```
