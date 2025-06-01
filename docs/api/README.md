# API文档

本目录将包含ClixGo项目的API接口文档和使用说明。

## 📋 文档规划

### 🚧 待完善的API文档

#### 核心API
- **会话管理API** - 会话创建、管理、持久化接口
- **终端API** - PTY管理、进程控制接口  
- **UI组件API** - 界面组件和布局管理接口
- **插件API** - 插件开发和集成接口

#### 工具API
- **错误处理API** - 统一错误处理框架接口
- **日志API** - 结构化日志记录接口
- **工具函数API** - 通用工具函数库接口
- **配置API** - 配置管理和验证接口

#### 监控API
- **性能监控API** - 系统性能监控接口
- **网络监控API** - 网络状态监控接口
- **任务管理API** - 任务调度和管理接口

## 🛠️ API文档生成

### 自动生成工具
我们计划使用以下工具自动生成API文档：

```bash
# 生成Go文档
godoc -http=:6060

# 生成API文档
swag init

# 生成接口文档
go-swagger generate spec
```

### 文档格式
- **Go标准文档**: 使用godoc生成
- **OpenAPI规范**: 用于REST API文档
- **Markdown格式**: 便于阅读和维护
- **交互式文档**: 支持在线测试

## 📚 API设计原则

### 设计理念
- **一致性**: 统一的命名和错误处理
- **简洁性**: 简单易用的接口设计
- **可扩展性**: 支持未来功能扩展
- **向后兼容**: 保持API稳定性

### 命名规范
- **包名**: 小写，简洁明了
- **接口名**: 大写开头，动词+名词
- **方法名**: 大写开头，语义清晰
- **参数名**: 小写开头，描述性强

### 错误处理
- 使用统一的错误类型 `ClixGoError`
- 提供详细的错误码和消息
- 支持错误链和上下文信息
- 国际化错误消息

## 🔍 API概览

### 核心接口

#### 会话管理
```go
type SessionManager interface {
    CreateSession(name string, config *SessionConfig) (*Session, error)
    GetSession(id string) (*Session, error)
    ListSessions() ([]*Session, error)
    KillSession(id string) error
    SaveSession(id string) error
    LoadSession(path string) (*Session, error)
}
```

#### 终端管理
```go
type Terminal interface {
    Start(cmd string, args ...string) error
    Stop() error
    Write(data []byte) (int, error)
    Read(data []byte) (int, error)
    Resize(cols, rows int) error
    GetPID() int
}
```

#### UI组件
```go
type UIManager interface {
    CreatePanel(config PanelConfig) (*Panel, error)
    SplitPanel(id string, direction Direction) error
    ClosePanel(id string) error
    SwitchPanel(id string) error
    UpdateLayout() error
}
```

### 工具接口

#### 日志记录
```go
type Logger interface {
    Debug(msg string, fields ...zap.Field)
    Info(msg string, fields ...zap.Field)
    Warn(msg string, fields ...zap.Field)
    Error(msg string, fields ...zap.Field)
    Fatal(msg string, fields ...zap.Field)
}
```

#### 配置管理
```go
type ConfigManager interface {
    Load(path string) error
    Save(path string) error
    Get(key string) interface{}
    Set(key string, value interface{}) error
    Validate() error
}
```

## 📊 API状态

### 实现状态
| 模块 | 接口定义 | 实现完成 | 文档完成 | 测试覆盖 |
|------|----------|----------|----------|----------|
| 会话管理 | ✅ 完成 | ✅ 完成 | 🚧 进行中 | ✅ 87% |
| 终端管理 | ✅ 完成 | ✅ 完成 | 🚧 进行中 | ✅ 85% |
| UI组件 | ✅ 完成 | ✅ 完成 | 🚧 进行中 | ✅ 65% |
| 错误处理 | ✅ 完成 | ✅ 完成 | 🚧 进行中 | ✅ 92% |
| 日志系统 | ✅ 完成 | ✅ 完成 | 🚧 进行中 | ✅ 88% |
| 配置管理 | ✅ 完成 | ✅ 完成 | 🚧 进行中 | ✅ 76% |

### 版本兼容性
- **v0.1.x**: 基础API，实验性质
- **v0.2.x**: 稳定API，向后兼容
- **v1.0.x**: 正式API，长期支持

## 🚀 使用示例

### 基本用法
```go
// 创建会话管理器
manager := terminal.NewSessionManager()

// 创建新会话
session, err := manager.CreateSession("my-session", &terminal.SessionConfig{
    Shell: "/bin/bash",
    WorkDir: "/home/user",
})
if err != nil {
    log.Fatal(err)
}

// 创建窗口
window, err := session.CreateWindow("main")
if err != nil {
    log.Fatal(err)
}

// 执行命令
err = window.Execute("ls -la")
if err != nil {
    log.Fatal(err)
}
```

### 高级用法
```go
// 使用UI管理器
ui, err := ui.NewUIManager(ui.UIConfig{
    MouseEnabled: true,
    RefreshRate: time.Millisecond * 16,
})
if err != nil {
    log.Fatal(err)
}

// 创建分割面板
panel, err := ui.CreatePanel(ui.PanelConfig{
    Title: "Terminal",
    Border: true,
})
if err != nil {
    log.Fatal(err)
}

// 绑定会话到面板
err = panel.AttachSession(session)
if err != nil {
    log.Fatal(err)
}
```

## 🔗 相关资源

### 开发文档
- [架构设计](../architecture/) - 系统架构和设计
- [实现细节](../implementation/) - 技术实现方案
- [开发指南](../guides/) - 开发规范和流程

### 外部资源
- [Go文档规范](https://golang.org/doc/comment)
- [OpenAPI规范](https://swagger.io/specification/)
- [REST API设计指南](https://restfulapi.net/)

## 📅 开发计划

### 短期目标 (1-2周)
- [ ] 完成核心API文档编写
- [ ] 建立自动文档生成流程
- [ ] 添加API使用示例
- [ ] 完善接口注释

### 中期目标 (1个月)
- [ ] 生成完整的API参考文档
- [ ] 建立在线文档站点
- [ ] 添加交互式API测试
- [ ] 完善错误码文档

### 长期目标 (3个月)
- [ ] 建立API版本管理
- [ ] 完善向后兼容策略
- [ ] 建立API变更通知机制
- [ ] 社区API贡献指南

---

