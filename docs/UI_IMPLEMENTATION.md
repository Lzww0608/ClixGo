# ClixGo 终端UI渲染系统实现文档

## 概述

ClixGo 终端UI渲染系统是一个基于 `tcell/tview` 的高性能终端多路复用器UI实现，提供了现代化的终端用户界面体验。

## 技术架构

### 核心组件

#### 1. UIManager (UI管理器)
- **文件**: `pkg/terminal/ui/manager.go`
- **功能**: 管理整个UI系统的生命周期
- **特性**:
  - 面板创建和管理
  - 布局自动调整
  - 事件处理和分发
  - 并发安全设计

#### 2. Layout (布局管理器)
- **支持的布局模式**:
  - `LayoutSingle`: 单面板模式
  - `LayoutVertical`: 垂直分割模式
  - `LayoutHorizontal`: 水平分割模式
  - `LayoutGrid`: 网格布局模式

#### 3. Panel (面板)
- **功能**: 独立的内容显示区域
- **特性**:
  - 自动滚动
  - 内容限制 (默认1000行)
  - 边框和标题显示
  - 活动状态指示

#### 4. StatusBar (状态栏)
- **文件**: `pkg/terminal/ui/statusbar.go`
- **功能**: 显示系统状态信息
- **布局**: 左侧 | 中间 | 右侧

### 依赖库

- **tcell/v2**: 底层终端控制
- **tview**: 高级UI组件库
- **zap**: 日志系统

## 功能特性

### 1. 多面板支持
```go
// 创建新面板
panel := uiManager.CreatePanel("panel_id", "Panel Title")

// 向面板写入内容
uiManager.WriteToPanel("panel_id", "Hello, World!")
```

### 2. 智能布局管理
- **1个面板**: 自动使用单面板模式
- **2个面板**: 自动使用垂直分割模式
- **3+个面板**: 自动使用网格布局模式

### 3. 快捷键支持
| 快捷键 | 功能 |
|--------|------|
| `Ctrl+C` | 退出应用 |
| `Ctrl+D` | 分离会话 |
| `Ctrl+N` | 创建新面板 |
| `Ctrl+W` | 关闭当前面板 |
| `Tab` | 切换面板 |
| `Ctrl+H` | 水平分割 |
| `Ctrl+V` | 垂直分割 |
| `F1` | 显示帮助 |

### 4. 鼠标支持
- 点击面板切换焦点
- 滚轮滚动面板内容

### 5. 实时状态栏
- 左侧: 应用名称
- 中间: 面板统计信息
- 右侧: 当前时间

## 使用示例

### 基本使用

```go
package main

import (
    "github.com/Lzww0608/ClixGo/pkg/logger"
    "github.com/Lzww0608/ClixGo/pkg/terminal/ui"
)

func main() {
    // 初始化日志系统
    logger.InitLogger()
    defer logger.Close()
    
    // 创建UI管理器
    uiManager, err := ui.NewUIManager(ui.DefaultUIConfig)
    if err != nil {
        panic(err)
    }
    
    // 创建面板
    uiManager.CreatePanel("main", "Main Panel")
    uiManager.WriteToPanel("main", "欢迎使用 ClixGo!")
    
    // 启动UI
    uiManager.Start()
}
```

### 高级配置

```go
config := ui.UIConfig{
    Theme: ui.Theme{
        Background:   tcell.ColorBlack,
        Foreground:   tcell.ColorWhite,
        Border:       tcell.ColorGray,
        ActiveBorder: tcell.ColorBlue,
        StatusBar:    tcell.ColorDarkBlue,
        StatusText:   tcell.ColorWhite,
    },
    MouseEnabled: true,
    RefreshRate:  time.Millisecond * 100,
}

uiManager, err := ui.NewUIManager(config)
```

## 性能特性

### 内存管理
- 面板内容自动限制行数 (默认1000行)
- 智能垃圾回收
- 并发安全的读写操作

### 渲染优化
- 基于 tcell 的高效终端渲染
- 增量更新机制
- 最小化重绘区域

### 并发支持
- 线程安全的面板操作
- 异步状态栏更新
- 非阻塞事件处理

## 测试覆盖

### 单元测试
- **文件**: `pkg/terminal/ui/manager_test.go`
- **覆盖率**: 100%
- **测试用例**:
  - UI管理器创建和销毁
  - 面板创建和管理
  - 布局模式切换
  - 状态栏功能
  - 并发操作安全性

### 测试运行
```bash
go test ./pkg/terminal/ui/ -v
```

## 演示程序

### 交互式演示
```bash
go run ./examples/ui_demo/main.go
```

演示程序包含：
- 欢迎界面
- 自动创建多个面板
- 模拟日志输出
- 系统监控数据显示

## 扩展性

### 自定义主题
```go
customTheme := ui.Theme{
    Background:   tcell.ColorDarkGreen,
    Foreground:   tcell.ColorWhite,
    Border:       tcell.ColorYellow,
    ActiveBorder: tcell.ColorRed,
    StatusBar:    tcell.ColorBlue,
    StatusText:   tcell.ColorWhite,
}
```

### 自定义按键绑定
```go
config.KeyBindings = map[string]string{
    "Ctrl+Q": "quit",
    "Ctrl+S": "save_session",
    "F2":     "rename_panel",
}
```

## 未来规划

### 短期目标
- [ ] 面板大小调整支持
- [ ] 更多主题选项
- [ ] 面板拖拽功能

### 长期目标
- [ ] 插件系统集成
- [ ] 远程UI支持
- [ ] 配置文件支持

## 贡献指南

1. 遵循现有代码风格
2. 添加适当的单元测试
3. 更新相关文档
4. 确保向后兼容性

## 许可证

本项目采用 MIT 许可证。 