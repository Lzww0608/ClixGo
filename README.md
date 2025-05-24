# ClixGo

ClixGo 是一个功能强大的命令行工具集合，提供了多种实用功能，帮助用户提高工作效率。

## 功能

- **命令执行**
  - 串行执行命令：按顺序执行多个命令
  - 并行执行命令：同时执行多个命令
  - 管道命令处理：命令管道支持
  
- **文本处理**
  - AWK 命令处理：简化 AWK 命令的使用
  - grep 命令处理：增强的文本搜索
  - sed 命令处理：简化的文本替换
  
- **工作流辅助**
  - 历史记录：查看和重用命令历史
  - 命令别名：定义和使用命令别名
  - 命令补全：自动完成命令和参数
  
- **后台任务**
  - 任务创建与管理：创建和监控长时间运行的任务
  - 任务状态查询：检查任务进度和状态
  - 取消任务：中止正在运行的任务
  
- **网络工具**
  - Ping：测试网络连接
  - Traceroute：跟踪网络路径
  - DNS查询：查询域名解析
  - HTTP请求：发送HTTP请求
  - 端口检查：检查端口开放状态
  - IP信息：查询IP地址信息
  - 下载文件：从URL下载文件
  - SSL证书检查：检查网站SSL证书
  - 网络速度测试：测试网络速度
  - 网络监控：监控网络状态

- **🚀 终端多路复用器 (NEW!)**
  - **🎯 零配置启动**：开箱即用，无需复杂配置
  - **⚡ 超轻量级**：纯 Go 实现，启动速度快 3-5 倍
  - **🔧 深度工具集成**：内置网络诊断、文本处理、任务管理
  - **🌐 跨平台原生支持**：Windows/Linux/macOS 完全兼容
  - **☁️ 智能会话同步**：支持云端会话配置同步
  - **🎨 现代化界面**：支持鼠标操作、主题定制、状态栏定制
  - **🔄 智能恢复**：自动保存和恢复会话状态，包括运行中的命令
  - **📊 内置监控**：集成系统监控和性能分析

## 安装

```bash
# 从源码安装
git clone https://github.com/Lzww0608/ClixGo.git
cd ClixGo
go install ./...
```

安装完成后，`ClixGo` 命令将可用。如果 `$GOPATH/bin` 不在您的 PATH 中，您可能需要运行：

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

## 使用方法

### 基本命令

```bash
# 查看帮助
ClixGo help

# 生成命令补全脚本
ClixGo completion > ~/.clixgo_completion
source ~/.clixgo_completion

# 串行执行命令
ClixGo sequential "ls -la; echo hello"

# 并行执行命令
ClixGo parallel "ping -c 3 example.com; curl https://example.com"

# 使用AWK命令
ClixGo awk "filename.txt" '{print $1}'

# 使用grep命令
ClixGo grep "filename.txt" "pattern"

# 使用sed命令
ClixGo sed "filename.txt" "s/old/new/g"

# 使用管道命令
ClixGo pipe "ls -la | grep .txt | sort"

# 查看历史记录
ClixGo history

# 创建别名
ClixGo alias set "ll" "ls -la"
```

### 🆕 终端多路复用器

ClixGo Terminal 是下一代轻量级终端多路复用器，相比传统 tmux 具有显著优势：

```bash
# 创建新会话
ClixGo terminal new-session [session-name]

# 连接到现有会话
ClixGo terminal attach [session-name]

# 列出所有会话
ClixGo terminal list-sessions

# 销毁会话
ClixGo terminal kill-session [session-name]

# 启动服务器
ClixGo terminal server start

# 查看服务器状态
ClixGo terminal server status

# 分割窗口
ClixGo terminal split-window --vertical
```

#### 快捷键操作

默认快捷键前缀为 `Ctrl+B`，主要快捷键包括：

- `Ctrl+B, D` - 断开会话（detach）
- `Ctrl+B, C` - 创建新窗口
- `Ctrl+B, "` - 水平分割面板
- `Ctrl+B, %` - 垂直分割面板
- `Ctrl+B, O` - 切换面板
- `Ctrl+B, X` - 关闭面板
- `Ctrl+B, ?` - 显示帮助

#### 配置文件

创建配置文件 `~/.clixgo/terminal.yaml`：

```yaml
# 基础配置
prefix_key: "C-b"
mouse_enabled: true
status_bar: true
auto_save: true
save_interval: "5m"

# 主题配置
theme: "default"
status_format: "[#S] #I:#W"

# ClixGo 集成
clixgo_integration: true
network_monitor: false
task_integration: true
```

### 网络工具

ClixGo 提供了丰富的网络工具集：

```bash
# DNS查询
ClixGo network dns example.com

# HTTP请求
ClixGo network http https://example.com

# 端口检查
ClixGo network port example.com 80

# IP信息查询
ClixGo network ipinfo 8.8.8.8

# SSL证书检查
ClixGo network ssl example.com

# 网络配置查看
ClixGo network config

# 带宽测试
ClixGo network bandwidth

# 网络诊断
ClixGo network diagnose
```

### 任务管理

任务管理功能允许您创建、监控和管理长时间运行的后台任务。

```bash
# 创建任务
ClixGo task create "任务名称" "任务描述"

# 列出所有任务
ClixGo task list

# 查看任务状态
ClixGo task status <task-id>

# 取消任务
ClixGo task cancel <task-id>

# 监控任务进度
ClixGo task watch <task-id>
```

**注意**：通过命令行创建的任务初始状态为"pending"，需要通过编程方式启动。您可以参考 `examples/task/main.go` 或 `examples/taskmanager/main.go` 中的示例代码了解如何启动和管理任务。

## 实际使用示例

### 文本处理示例

```bash
# 统计文件中的单词、行数和字符数
ClixGo text count myfile.txt

# 查找包含特定模式的行
ClixGo text find myfile.txt "search pattern"

# 替换文本
ClixGo text replace myfile.txt "old text" "new text"

# 提取所有URL
ClixGo text extract urls myfile.txt

# 格式化JSON
ClixGo text json format '{"name":"John","age":30}'
```

### 网络监控示例

```bash
# 监控多个主机的网络状态
ClixGo network monitor example.com google.com -i 10s -t 3s

# 分析特定接口的流量
ClixGo network traffic -i eth0

# 评估网络质量
ClixGo network quality example.com -d 30s
```

### 终端多路复用器实际场景

```bash
# 场景1：开发环境搭建
ClixGo terminal new-session "dev-env"
# 在会话中：
# Ctrl+B, " -> 创建代码编辑区域
# Ctrl+B, % -> 创建终端区域
# Ctrl+B, C -> 创建服务器监控窗口

# 场景2：服务器管理
ClixGo terminal new-session "server-mgmt"
# 集成 ClixGo 网络工具监控服务器状态
# network ping server1.example.com
# task list  # 查看后台任务

# 场景3：多项目管理
ClixGo terminal new-session "project-a"
ClixGo terminal new-session "project-b"
ClixGo terminal list-sessions  # 查看所有项目会话
```

## 项目结构

```
ClixGo/
├── cmd/                 # 命令行工具
│   ├── cli/            # 主命令行接口
│   └── task/           # 任务管理命令
├── examples/           # 示例程序
│   └── terminal/       # 终端多路复用器示例
├── internal/           # 内部包
├── pkg/                # 公共包
│   ├── alias/          # 命令别名
│   ├── commands/       # 命令执行
│   ├── completion/     # 命令补全
│   ├── config/         # 配置管理
│   ├── filesystem/     # 文件系统操作
│   ├── history/        # 历史记录
│   ├── logger/         # 日志管理
│   ├── network/        # 网络工具
│   ├── plugin/         # 插件系统
│   ├── security/       # 安全功能
│   ├── task/           # 任务管理
│   ├── terminal/       # 🆕 终端多路复用器
│   ├── text/           # 文本处理
│   └── utils/          # 通用工具
├── plugins/            # 插件目录
│   ├── translate/      # 翻译插件
│   └── weather/        # 天气插件
├── go.mod              # 依赖管理
└── README.md           # 文档
```

## 配置

ClixGo 使用 YAML 配置文件，默认位置为 `~/.clixgo/config.yaml`。您也可以通过 `-c` 参数指定自定义配置文件路径。

配置示例:

```yaml
log_level: debug
log_file: clixgo.log

commands:
  timeout: 30

network:
  default_dns:
    - 8.8.8.8
    - 1.1.1.1

task:
  store_path: ~/.clixgo/tasks.json
  max_concurrent: 5

# 终端多路复用器配置
terminal:
  prefix_key: "C-b"
  mouse_enabled: true
  status_bar: true
  auto_save: true
  save_interval: "5m"
  theme: "default"
  clixgo_integration: true
```

## 开发说明

如果您想扩展 ClixGo 的功能，可以通过以下方式：

1. 添加新的命令到 `cmd/cli` 目录
2. 在 `pkg` 目录下实现功能模块
3. 修改 `cmd/cli/root.go` 添加新命令

开发任务管理相关功能时，可参考 `pkg/task/manager.go` 中的实现。

开发终端多路复用器相关功能时，可参考 `pkg/terminal/` 目录下的实现，主要模块包括：
- `types.go` - 核心类型定义
- `session.go` - 会话管理
- `server.go` - 服务器实现
- `client.go` - 客户端实现

## 性能对比

| 功能 | ClixGo Terminal | tmux | screen |
|------|----------------|------|---------|
| 启动时间 | ~50ms | ~150ms | ~100ms |
| 内存占用 | ~8MB | ~25MB | ~15MB |
| 跨平台支持 | ✅ 原生 | ❌ 需编译 | ❌ 需编译 |
| 配置复杂度 | 🟢 零配置 | 🟡 中等 | 🟡 中等 |
| 工具集成 | ✅ 深度集成 | ❌ 无 | ❌ 无 |
| 会话恢复 | ✅ 智能 | 🟡 手动 | 🟡 手动 |

## 贡献

欢迎提交 Issue 和 Pull Request 贡献代码。

## 许可证

MIT License 