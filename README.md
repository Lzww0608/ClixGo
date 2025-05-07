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

## 安装

```bash
# 从源码安装
git clone https://github.com/Lzww0608/ClixGo.git
cd ClixGo
go install ./...
```

## 使用方法

### 基本命令

```bash
# 查看帮助
gocli help

# 生成命令补全脚本
gocli completion > ~/.gocli_completion
source ~/.gocli_completion

# 串行执行命令
gocli sequential "ls -la; echo hello"

# 并行执行命令
gocli parallel "ping -c 3 example.com; curl https://example.com"

# 使用AWK命令
gocli awk "filename.txt" '{print $1}'

# 使用grep命令
gocli grep "filename.txt" "pattern"

# 使用sed命令
gocli sed "filename.txt" "s/old/new/g"

# 使用管道命令
gocli pipe "ls -la | grep .txt | sort"

# 查看历史记录
gocli history

# 创建别名
gocli alias set "ll" "ls -la"

# 使用网络工具
gocli network ping example.com
gocli network dns example.com
gocli network http https://example.com
```

### 任务管理

任务管理功能允许您创建、监控和管理长时间运行的后台任务。

```bash
# 创建任务
gocli task create "任务名称" "任务描述"

# 列出所有任务
gocli task list

# 查看任务状态
gocli task status <task-id>

# 取消任务
gocli task cancel <task-id>

# 监控任务进度
gocli task watch <task-id>
```

## 项目结构

```
ClixGo/
├── cmd/                 # 命令行工具
│   ├── cli/            # 主命令行接口
│   └── task/           # 任务管理命令
├── examples/           # 示例程序
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
│   ├── text/           # 文本处理
│   └── utils/          # 通用工具
├── plugins/            # 插件目录
│   ├── translate/      # 翻译插件
│   └── weather/        # 天气插件
├── go.mod              # 依赖管理
└── README.md           # 文档
```

## 配置

ClixGo 使用 YAML 配置文件，默认位置为 `~/.gocli/config.yaml`。您也可以通过 `-c` 参数指定自定义配置文件路径。

配置示例:

```yaml
log_level: debug
log_file: gocli.log

commands:
  timeout: 30
```

## 贡献

欢迎提交 Issue 和 Pull Request 贡献代码。

## 许可证

MIT License 