# ClixGo

ClixGo 是一个功能丰富的命令行工具集合，提供了多种实用功能，帮助用户提高工作效率。

## 功能

- **串行执行命令**：按顺序执行多个命令
- **并行执行命令**：同时执行多个命令
- **AWK 命令处理**：简化 AWK 命令的使用
- **grep 命令处理**：增强的文本搜索
- **sed 命令处理**：简化的文本替换
- **管道命令处理**：命令管道支持
- **历史记录**：查看和重用命令历史
- **命令别名**：定义和使用命令别名
- **命令补全**：自动完成命令和参数
- **后台任务管理**：管理长时间运行的任务

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

### 作为库使用

您可以在自己的 Go 程序中导入并使用 ClixGo 的组件：

```go
import "github.com/Lzww0608/ClixGo/pkg/task"

// 创建任务管理器
tm, err := task.NewTaskManager(logger, "tasks.json")
if err != nil {
    log.Fatal(err)
}

// 创建任务
task, err := tm.CreateTask("任务名称", "任务描述", nil)
if err != nil {
    log.Fatal(err)
}

// 启动任务
err = tm.StartTask(ctx, task.ID, func(ctx context.Context, t *task.Task) error {
    // 实现任务逻辑
    return nil
})
```

## 项目结构

```
ClixGo/
├── cmd/                 # 命令行工具
│   ├── cli/            # 主命令行接口
│   └── task/           # 任务管理命令
├── examples/           # 示例程序
│   └── task/           # 任务管理示例
├── internal/           # 内部包
├── pkg/                # 公共包
│   ├── alias/          # 命令别名
│   ├── completion/     # 命令补全
│   ├── config/         # 配置管理
│   ├── logger/         # 日志管理
│   └── task/           # 任务管理
├── go.mod              # 依赖管理
└── README.md           # 文档
```

## 依赖

- github.com/spf13/cobra - 命令行框架
- github.com/spf13/viper - 配置管理
- go.uber.org/zap - 日志管理
- github.com/google/uuid - 生成唯一ID
- github.com/pkg/errors - 错误处理

## 贡献

欢迎提交 Issue 和 Pull Request 贡献代码。

## 许可证

MIT License 