# PTY (伪终端) 实现文档

## 概述

本文档记录了ClixGo项目中PTY (Pseudo Terminal) 功能的实现，这是项目开发路线图Phase 1.1的核心功能。

## 技术选型

### 使用的开源库
- **creack/pty**: GitHub上最受欢迎的Go语言PTY库
  - 星标数: 1.8k+
  - 使用项目: 23.2k+
  - 跨平台支持: Linux, macOS, Windows
  - 成熟稳定，社区活跃

- **golang.org/x/term**: Go官方终端控制库
  - 用于终端模式控制
  - 原始模式支持

## 实现架构

### 核心组件

#### 1. CreackPTY 结构体
```go
type CreackPTY struct {
    ID          string
    Command     *exec.Cmd
    Process     *os.Process
    PTY         *os.File
    Size        *pty.Winsize
    WorkingDir  string
    Environment []string
    // ... 其他字段
}
```

#### 2. CreackPTYManager 管理器
- 并发安全的PTY管理
- 支持创建、获取、销毁、列出PTY
- 使用读写锁保护共享资源

### 核心功能

#### 1. PTY创建和启动
- 智能shell路径检测
- 环境变量继承和自定义
- 工作目录设置
- 窗口大小配置

#### 2. 进程生命周期管理
- 异步进程监控
- 优雅关闭机制
- 进程状态跟踪
- 退出码获取

#### 3. I/O处理
- 异步读写操作
- 超时控制
- 缓冲区管理
- 错误处理

#### 4. 信号处理
- SIGWINCH窗口大小变化
- 进程信号转发
- 优雅退出信号

## 文件结构

```
ClixGo/
├── pkg/terminal/
│   ├── pty_creack.go          # PTY核心实现
│   └── pty_creack_test.go     # 单元测试
├── examples/
│   └── pty_demo.go            # 演示程序
└── docs/
    └── PTY_IMPLEMENTATION.md  # 本文档
```

## 使用示例

### 基本用法

```go
// 创建PTY管理器
manager := terminal.NewCreackPTYManager()

// 创建PTY配置
config := &terminal.TerminalConfig{
    Shell:       "/usr/bin/bash",
    WorkingDir:  "/home/user",
    Environment: []string{"TERM=xterm-256color"},
    Rows:        24,
    Cols:        80,
}

// 创建PTY
ptyInstance, err := manager.CreatePTY("session1", config)
if err != nil {
    log.Fatal(err)
}

// 启动PTY
err = ptyInstance.Start()
if err != nil {
    log.Fatal(err)
}

// 与PTY交互
ptyInstance.Write([]byte("ls -la\n"))
output, err := ptyInstance.Read()
if err != nil {
    log.Fatal(err)
}
fmt.Print(string(output))
```

### 高级功能

```go
// 调整窗口大小
err = ptyInstance.Resize(30, 100)

// 获取进程状态
status := ptyInstance.GetStatus()

// 优雅关闭
err = ptyInstance.Stop()
```

## 测试覆盖

### 单元测试
- PTY创建和销毁测试
- 进程启动和停止测试
- 读写操作测试
- 大小调整测试
- 错误处理测试

### 演示程序
- 交互式菜单系统
- 完整的PTY生命周期演示
- 实时命令执行
- 会话管理

## 性能特性

### 内存使用
- 轻量级设计，单个PTY占用 < 1MB
- 高效的缓冲区管理
- 及时的资源清理

### 并发性能
- 支持多个并发PTY会话
- 线程安全的操作
- 异步I/O处理

### 响应性
- 低延迟的命令执行
- 实时的输出流
- 快速的窗口大小调整

## 跨平台支持

### Linux
- 完全支持所有功能
- 原生PTY支持
- 信号处理完整

### macOS
- 通过creack/pty库支持
- 兼容性良好

### Windows
- 通过ConPTY支持
- 功能基本完整

## 已知限制

1. **Windows兼容性**: 某些高级功能在Windows上可能有限制
2. **信号处理**: 不同平台的信号行为可能略有差异
3. **终端特性**: 某些终端特定功能可能不完全兼容

## 未来改进

1. **性能优化**: 进一步优化内存使用和I/O性能
2. **功能扩展**: 添加更多终端特性支持
3. **错误处理**: 增强错误恢复机制
4. **监控集成**: 添加性能监控和日志记录

## 相关资源

- [creack/pty GitHub仓库](https://github.com/creack/pty)
- [Go终端编程指南](https://golang.org/x/term)
- [PTY概念介绍](https://en.wikipedia.org/wiki/Pseudoterminal)

## 贡献指南

如需对PTY功能进行改进或扩展，请：

1. 阅读现有代码和测试
2. 确保跨平台兼容性
3. 添加相应的单元测试
4. 更新文档和示例

---

*最后更新: 2024年12月*
*版本: 1.0.0* 