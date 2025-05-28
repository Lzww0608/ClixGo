# ClixGo 会话持久化系统

## 概述

ClixGo 会话持久化系统提供了完整的终端会话保存和恢复功能，类似于 tmux-resurrect 的功能，但专为 ClixGo 终端多路复用器设计。该系统能够保存会话的完整状态，包括窗口布局、面板配置、缓冲区内容和环境变量等。

## 功能特性

### 🔄 核心功能
- **会话状态保存/恢复** - 完整保存会话的所有状态信息
- **进程状态快照** - 记录进程信息和工作目录
- **缓冲区历史记录** - 保存终端输出历史（可配置行数）
- **环境变量保存** - 保存会话相关的环境变量
- **自动清理** - 自动清理旧快照，避免磁盘空间浪费

### 📊 保存内容
- 会话基本信息（ID、名称、状态、时间戳）
- 窗口配置（名称、索引、布局、尺寸）
- 面板详情（位置、大小、命令、工作目录）
- 缓冲区内容（终端输出历史）
- 光标位置和状态
- 环境变量和元数据

### 🚀 高级特性
- **自动保存** - 支持定时自动保存会话
- **版本管理** - 保留多个历史快照
- **增量清理** - 智能清理旧快照
- **并发安全** - 支持多线程安全操作
- **JSON格式** - 人类可读的存储格式

## 快速开始

### 基本使用

```go
package main

import (
    "github.com/Lzww0608/ClixGo/pkg/terminal"
    "github.com/Lzww0608/ClixGo/pkg/logger"
)

func main() {
    // 初始化日志系统
    logger.InitLogger()
    defer logger.Close()
    
    // 创建会话管理器
    sm := terminal.NewSessionManager(terminal.DefaultConfig)
    
    // 创建会话
    session, _ := sm.CreateSession("my_session")
    
    // 保存会话
    err := sm.SaveSession(session.ID, "")
    if err != nil {
        panic(err)
    }
    
    // 加载会话
    loadedSession, err := sm.LoadSessionByName("my_session")
    if err != nil {
        panic(err)
    }
}
```

### 配置持久化管理器

```go
config := &terminal.PersistenceConfig{
    DataDir:         "/path/to/sessions",
    AutoSave:        true,
    SaveInterval:    time.Minute * 5,
    MaxSnapshots:    10,
    CompressData:    false,
    SaveBufferLines: 1000,
    SaveHistory:     true,
    SaveEnvironment: true,
}

pm, err := terminal.NewPersistenceManager(config)
```

## API 参考

### PersistenceManager

#### 创建管理器
```go
func NewPersistenceManager(config *PersistenceConfig) (*PersistenceManager, error)
```

#### 保存会话
```go
func (pm *PersistenceManager) SaveSession(session *Session) error
```

#### 加载会话
```go
func (pm *PersistenceManager) LoadSession(sessionName string) (*SessionSnapshot, error)
```

#### 恢复会话
```go
func (pm *PersistenceManager) RestoreSession(snapshot *SessionSnapshot, sm *SessionManager) (*Session, error)
```

#### 列出快照
```go
func (pm *PersistenceManager) ListSnapshots() ([]string, error)
```

#### 删除快照
```go
func (pm *PersistenceManager) DeleteSnapshot(filename string) error
```

### SessionManager 扩展方法

#### 保存会话（按ID）
```go
func (sm *SessionManager) SaveSession(sessionID string, filepath string) error
```

#### 保存会话（按名称）
```go
func (sm *SessionManager) SaveSessionByName(sessionName string) error
```

#### 加载会话（按名称）
```go
func (sm *SessionManager) LoadSessionByName(sessionName string) (*Session, error)
```

#### 列出已保存的会话
```go
func (sm *SessionManager) ListSavedSessions() ([]string, error)
```

#### 删除已保存的会话
```go
func (sm *SessionManager) DeleteSavedSession(sessionName string) error
```

#### 自动保存
```go
func (sm *SessionManager) AutoSaveSession(sessionID string, interval time.Duration)
```

## 配置选项

### PersistenceConfig

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `DataDir` | string | `~/.clixgo/sessions` | 快照存储目录 |
| `AutoSave` | bool | `true` | 是否启用自动保存 |
| `SaveInterval` | time.Duration | `5分钟` | 自动保存间隔 |
| `MaxSnapshots` | int | `10` | 每个会话最大快照数 |
| `CompressData` | bool | `false` | 是否压缩数据 |
| `SaveBufferLines` | int | `1000` | 保存的缓冲区行数 |
| `SaveHistory` | bool | `true` | 是否保存命令历史 |
| `SaveEnvironment` | bool | `true` | 是否保存环境变量 |

## 数据结构

### SessionSnapshot
```go
type SessionSnapshot struct {
    ID           string                `json:"id"`
    Name         string                `json:"name"`
    Status       SessionStatus         `json:"status"`
    CreatedAt    time.Time             `json:"created_at"`
    LastActive   time.Time             `json:"last_active"`
    SavedAt      time.Time             `json:"saved_at"`
    Windows      []*WindowSnapshot     `json:"windows"`
    ActiveWindow int                   `json:"active_window"`
    Environment  map[string]string     `json:"environment"`
    WorkingDir   string                `json:"working_dir"`
    Metadata     map[string]interface{} `json:"metadata"`
}
```

### WindowSnapshot
```go
type WindowSnapshot struct {
    ID         string           `json:"id"`
    Name       string           `json:"name"`
    Index      int              `json:"index"`
    Panes      []*PaneSnapshot  `json:"panes"`
    ActivePane int              `json:"active_pane"`
    Layout     Layout           `json:"layout"`
    CreatedAt  time.Time        `json:"created_at"`
    Size       *WindowSize      `json:"size"`
}
```

### PaneSnapshot
```go
type PaneSnapshot struct {
    ID           string            `json:"id"`
    Index        int               `json:"index"`
    X            int               `json:"x"`
    Y            int               `json:"y"`
    Width        int               `json:"width"`
    Height       int               `json:"height"`
    Command      string            `json:"command"`
    WorkingDir   string            `json:"working_dir"`
    ProcessID    int               `json:"process_id"`
    ProcessName  string            `json:"process_name"`
    Active       bool              `json:"active"`
    CreatedAt    time.Time         `json:"created_at"`
    LastOutput   time.Time         `json:"last_output"`
    BufferLines  []string          `json:"buffer_lines"`
    CursorPos    *CursorPosition   `json:"cursor_pos"`
    Environment  map[string]string `json:"environment"`
    History      []string          `json:"history"`
}
```

## 使用示例

### 演示程序

运行完整的交互式演示：

```bash
go run ./examples/persistence_demo/main.go
```

演示程序功能：
- 📝 列出当前会话
- 💾 保存会话
- 📂 加载会话
- 💾 列出已保存的会话
- 🗑️ 删除已保存的会话
- 🔍 查看会话详情
- ⏰ 演示自动保存
- 📝 创建新会话

### 自动保存示例

```go
// 启动自动保存（每5分钟保存一次）
go sm.AutoSaveSession(sessionID, 5*time.Minute)
```

### 批量操作示例

```go
// 保存所有活动会话
sessions := sm.ListSessions()
for _, session := range sessions {
    err := sm.SaveSession(session.ID, "")
    if err != nil {
        log.Printf("保存会话 %s 失败: %v", session.Name, err)
    }
}

// 列出并加载所有已保存的会话
savedSessions, _ := sm.ListSavedSessions()
for _, sessionName := range savedSessions {
    session, err := sm.LoadSessionByName(sessionName)
    if err != nil {
        log.Printf("加载会话 %s 失败: %v", sessionName, err)
    } else {
        log.Printf("成功加载会话: %s", session.Name)
    }
}
```

## 最佳实践

### 1. 定期保存
```go
// 设置合理的自动保存间隔
config.SaveInterval = time.Minute * 5  // 5分钟自动保存
```

### 2. 限制快照数量
```go
// 避免磁盘空间浪费
config.MaxSnapshots = 10  // 每个会话最多保留10个快照
```

### 3. 选择性保存
```go
// 根据需要配置保存内容
config.SaveBufferLines = 500    // 只保存最近500行
config.SaveEnvironment = false  // 不保存环境变量（如果不需要）
```

### 4. 错误处理
```go
if err := sm.SaveSession(sessionID, ""); err != nil {
    log.Printf("保存会话失败: %v", err)
    // 实施重试逻辑或用户通知
}
```

### 5. 资源清理
```go
// 定期清理不需要的快照
err := sm.DeleteSavedSession("old_session")
if err != nil {
    log.Printf("删除旧会话失败: %v", err)
}
```

## 文件格式

### 快照文件命名
```
{session_name}_{timestamp}.json
```

示例：
```
development_20240101_120000.json
monitoring_20240101_120500.json
```

### 存储目录结构
```
~/.clixgo/sessions/
├── development_20240101_120000.json
├── development_20240101_120500.json
├── monitoring_20240101_120000.json
└── monitoring_20240101_120500.json
```

### JSON 格式示例
```json
{
  "id": "session-uuid",
  "name": "development",
  "status": "active",
  "created_at": "2024-01-01T12:00:00Z",
  "last_active": "2024-01-01T12:05:00Z",
  "saved_at": "2024-01-01T12:05:00Z",
  "windows": [
    {
      "id": "window-uuid",
      "name": "editor",
      "index": 0,
      "panes": [
        {
          "id": "pane-uuid",
          "index": 0,
          "x": 0,
          "y": 0,
          "width": 80,
          "height": 24,
          "command": "vim",
          "working_dir": "/home/user/project",
          "process_id": 12345,
          "active": true,
          "buffer_lines": ["line 1", "line 2"],
          "cursor_pos": {"x": 0, "y": 1}
        }
      ],
      "active_pane": 0,
      "layout": "main-vertical"
    }
  ],
  "active_window": 0,
  "environment": {
    "PATH": "/usr/bin:/bin",
    "HOME": "/home/user"
  },
  "working_dir": "/home/user"
}
```

## 故障排除

### 常见问题

1. **权限错误**
   ```
   错误: 创建数据目录失败: permission denied
   解决: 确保对 ~/.clixgo/sessions 目录有写权限
   ```

2. **磁盘空间不足**
   ```
   错误: 写入会话快照文件失败: no space left on device
   解决: 清理旧快照或增加磁盘空间
   ```

3. **快照损坏**
   ```
   错误: 反序列化会话快照失败: invalid character
   解决: 删除损坏的快照文件，重新保存
   ```

### 调试技巧

1. **启用详细日志**
   ```go
   logger.SetLevel(zap.DebugLevel)
   ```

2. **检查快照文件**
   ```bash
   ls -la ~/.clixgo/sessions/
   cat ~/.clixgo/sessions/session_name_*.json | jq .
   ```

3. **手动清理**
   ```bash
   rm ~/.clixgo/sessions/corrupted_*.json
   ```

## 性能考虑

### 内存使用
- 快照创建时会临时占用额外内存
- 大型会话（多窗口/面板）会增加内存使用
- 建议限制 `SaveBufferLines` 以控制内存使用

### 磁盘使用
- JSON 格式相对占用较多空间
- 可通过 `MaxSnapshots` 限制快照数量
- 考虑启用 `CompressData`（未来版本）

### 性能优化
- 异步保存避免阻塞主线程
- 增量保存减少 I/O 操作
- 智能清理避免磁盘空间浪费

## 路线图

### 短期目标
- [ ] 支持数据压缩
- [ ] 增量快照功能
- [ ] 快照加密支持

### 长期目标
- [ ] 远程快照存储
- [ ] 快照同步功能
- [ ] 可视化快照管理

## 贡献指南

欢迎贡献代码和建议！请遵循以下步骤：

1. Fork 项目
2. 创建功能分支
3. 添加测试用例
4. 提交 Pull Request

## 许可证

本项目采用 MIT 许可证。 