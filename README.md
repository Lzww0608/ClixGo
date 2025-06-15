# ClixGo 2.0 - 增强型CLI工具套件

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org)
[![Build Status](https://img.shields.io/badge/Build-Passing-green.svg)](https://github.com/Lzww0608/ClixGo)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Architecture](https://img.shields.io/badge/Architecture-Simplified-brightgreen.svg)](ROADMAP.md)

ClixGo 2.0 是下一代高性能增强型CLI工具套件，以终端复用为核心，集成网络诊断、文本处理、性能监控等开发运维工具，提供统一的用户体验和卓越的性能表现。

## 🎯 项目愿景

构建现代化的CLI工具套件，实现：
- **🚀 极致性能**：启动速度快5倍，内存占用减少70%
- **🛠️ 工具集成**：一个工具解决多种开发运维需求
- **🎨 现代界面**：TUI图形化界面，支持鼠标和实时数据可视化
- **🔧 零配置启动**：开箱即用，同时支持深度定制

## 📊 当前状态 (Phase 1.2 已完成)

### ✅ 架构重构成果
- **模块精简**：从21个模块精简到13个核心模块 (减少38%)
- **入口统一**：从8个分散入口统一为2个 (减少75%)
- **编译成功**：生成11MB可执行文件，所有功能正常
- **依赖清晰**：模块间依赖关系明确，无循环依赖

### 🔧 当前可用功能

```bash
# 主要功能
./clixgo --help                    # 查看帮助信息
./clixgo --version                 # 查看版本信息

# 别名管理
./clixgo alias add ll "ls -la"     # 添加别名
./clixgo alias list                # 查看所有别名
./clixgo alias remove ll           # 删除别名

# 历史记录
./clixgo history list              # 查看命令历史
./clixgo history show 10           # 显示最近10条历史
./clixgo history clear             # 清空历史记录

# 文件系统操作
./clixgo fs list /path/to/dir      # 列出目录内容
./clixgo fs copy source dest       # 复制文件
./clixgo fs move source dest       # 移动文件
./clixgo fs delete file            # 删除文件

# 终端会话 (简化版)
./clixgo terminal session create  # 创建会话
./clixgo terminal session list    # 列出会话

# 命令补全
./clixgo completion bash > /etc/bash_completion.d/clixgo
./clixgo completion zsh > ~/.zsh/completions/_clixgo
```

## 🚀 安装

### 从源码安装

```bash
# 克隆仓库
git clone https://github.com/Lzww0608/ClixGo.git
cd ClixGo

# 编译安装
go build -o clixgo
sudo mv clixgo /usr/local/bin/

# 或者使用go install
go install
```

### 验证安装

```bash
# 检查版本
clixgo --version

# 查看帮助
clixgo --help

# 测试基础功能
clixgo alias list
clixgo history list
```

## 🏗️ 架构概览

### 核心模块结构

```
pkg/
├── commands/          # ✅ 命令执行、别名管理、补全
├── terminal/          # 🔄 终端复用、历史记录 (基础框架)
├── utils/             # ✅ 文件系统、工具函数
├── config/            # ✅ 配置管理
├── logger/            # ✅ 日志系统
├── errors/            # ✅ 错误处理
├── network/           # 📋 网络工具 (待完善)
├── performance/       # 📋 性能监控 (待完善)
├── sync/              # 📋 协程管理 (待完善)
├── task/              # 📋 任务管理 (待完善)
├── text/              # 📋 文本处理 (待完善)
└── ui/                # 📋 TUI界面 (待实现)
```

### 命令结构

```bash
clixgo
├── alias              # 别名管理
│   ├── add           # 添加别名
│   ├── remove        # 删除别名
│   └── list          # 列出别名
├── history            # 历史记录
│   ├── list          # 查看历史
│   ├── show          # 显示指定数量
│   └── clear         # 清空历史
├── fs                 # 文件系统
│   ├── list          # 列出文件
│   ├── copy          # 复制文件
│   ├── move          # 移动文件
│   └── delete        # 删除文件
├── terminal           # 终端复用
│   └── session       # 会话管理
└── completion         # 命令补全
    ├── bash          # bash补全
    └── zsh           # zsh补全
```

## 🔄 开发进展

### ✅ 已完成 (Phase 1.2)
- [x] 架构重构和模块精简
- [x] 功能迁移和整合
- [x] CLI命令结构统一
- [x] 编译问题修复
- [x] 基础功能验证

### 🔄 进行中 (Phase 1.3)
- [ ] Terminal模块PTY集成
- [ ] 真正的终端复用功能
- [ ] 性能基准测试
- [ ] 单元测试覆盖

### 📋 计划中 (Phase 2+)
- [ ] TUI界面开发 (tcell/tview)
- [ ] 数据可视化和监控
- [ ] 网络工具完善
- [ ] 智能化功能
- [ ] 插件生态

## 📈 性能目标

| 指标 | 目标值 | 当前状态 |
|------|--------|----------|
| **启动时间** | < 30ms | 📋 待测试 |
| **内存占用** | < 8MB | 📋 待测试 |
| **CPU空闲** | < 0.5% | 📋 待测试 |
| **并发会话** | 500+ | 📋 待实现 |

## 🛠️ 开发

### 环境要求

- Go 1.23+
- Linux/macOS/Windows
- Git

### 构建

```bash
# 开发构建
go build -o clixgo

# 生产构建
go build -ldflags="-s -w" -o clixgo

# 交叉编译
GOOS=linux GOARCH=amd64 go build -o clixgo-linux
GOOS=windows GOARCH=amd64 go build -o clixgo-windows.exe
GOOS=darwin GOARCH=amd64 go build -o clixgo-darwin
```

### 测试

```bash
# 运行测试
go test ./...

# 测试覆盖率
go test -cover ./...

# 基准测试
go test -bench=. ./...
```

## 📚 文档

- [ROADMAP.md](ROADMAP.md) - 详细开发路线图
- [PROGRESS_REPORT.md](PROGRESS_REPORT.md) - 项目进展报告
- [docs/](docs/) - 详细文档目录
  - [architecture/](docs/architecture/) - 架构设计文档
  - [api/](docs/api/) - API文档
  - [guides/](docs/guides/) - 使用指南

## 🤝 贡献

欢迎贡献代码！请查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解详细信息。

### 开发流程

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- [cobra](https://github.com/spf13/cobra) - CLI框架
- [viper](https://github.com/spf13/viper) - 配置管理
- [zap](https://github.com/uber-go/zap) - 日志系统
- [tcell](https://github.com/gdamore/tcell) - 终端界面 (计划中)
- [tview](https://github.com/rivo/tview) - TUI组件 (计划中)

## 📞 联系

- 项目主页: [https://github.com/Lzww0608/ClixGo](https://github.com/Lzww0608/ClixGo)
- 问题反馈: [Issues](https://github.com/Lzww0608/ClixGo/issues)
- 功能请求: [Feature Requests](https://github.com/Lzww0608/ClixGo/issues/new?template=feature_request.md)

---

**ClixGo 2.0** - 下一代增强型CLI工具套件，让命令行工作更高效、更现代、更智能！ 