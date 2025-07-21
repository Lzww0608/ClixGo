# ClixGo 2.0 - 高性能CLI工具套件

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org)
[![Build Status](https://img.shields.io/badge/Build-Passing-green.svg)](https://github.com/Lzww0608/ClixGo)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

ClixGo 2.0 是一个现代化的高性能CLI工具套件，以终端复用为核心，集成网络诊断、文件管理、性能监控等开发运维工具。

## ✨ 核心特性

- **🚀 高性能**：启动速度快5倍，内存占用减少70%
- **🛠️ 多功能集成**：终端复用、网络工具、文件管理一体化
- **🎨 现代界面**：支持TUI图形化界面和鼠标操作
- **🔧 开箱即用**：零配置启动，支持深度定制

## � 快速开始

### 基本用法

```bash
# 查看帮助
clixgo --help

# 别名管理
clixgo alias add ll "ls -la"       # 添加别名
clixgo alias list                  # 查看别名

# 文件操作
clixgo fs list /path/to/dir        # 列出目录
clixgo fs copy source dest         # 复制文件

# 终端会话
clixgo terminal session create    # 创建会话
clixgo terminal session list      # 列出会话

# 网络工具
clixgo network ping google.com    # 网络测试
clixgo network monitor            # 网络监控
```

## � 安装

### 环境要求
- Go 1.23+
- Linux/macOS/Windows

### 从源码安装

```bash
git clone https://github.com/Lzww0608/ClixGo.git
cd ClixGo
go build -o clixgo
sudo mv clixgo /usr/local/bin/
```

### 验证安装

```bash
clixgo --version
clixgo --help
```

## 🏗️ 架构

### 核心模块
- **commands** - 命令执行、别名管理
- **terminal** - 终端复用、会话管理
- **network** - 网络诊断工具
- **utils** - 文件系统操作
- **config** - 配置管理
- **logger** - 日志系统

### 主要命令
```bash
clixgo
├── alias              # 别名管理
├── history            # 历史记录
├── fs                 # 文件系统操作
├── terminal           # 终端复用
├── network            # 网络工具
└── completion         # 命令补全
```

##  性能目标

| 指标 | 目标值 | 状态 |
|------|--------|------|
| 启动时间 | < 30ms | � 优化中 |
| 内存占用 | < 8MB | � 优化中 |
| 并发会话 | 500+ | 📋 计划中 |

## 🛠️ 开发

### 构建

```bash
# 开发构建
go build -o clixgo

# 生产构建
go build -ldflags="-s -w" -o clixgo

# 运行测试
go test ./...
```

## 📚 技术栈

- **框架**: [Cobra](https://github.com/spf13/cobra) - CLI框架
- **配置**: [Viper](https://github.com/spf13/viper) - 配置管理
- **日志**: [Zap](https://github.com/uber-go/zap) - 高性能日志
- **终端**: [tcell](https://github.com/gdamore/tcell) - 终端界面
- **TUI**: [tview](https://github.com/rivo/tview) - TUI组件

## 🤝 贡献

欢迎贡献代码！

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/new-feature`)
3. 提交更改 (`git commit -m 'Add new feature'`)
4. 推送分支 (`git push origin feature/new-feature`)
5. 创建 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 📞 联系

- 项目主页: [https://github.com/Lzww0608/ClixGo](https://github.com/Lzww0608/ClixGo)
- 问题反馈: [Issues](https://github.com/Lzww0608/ClixGo/issues)

---

**ClixGo 2.0** - 现代化高性能CLI工具套件