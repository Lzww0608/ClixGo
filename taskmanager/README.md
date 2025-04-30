# 任务管理系统

一个功能完整的后台任务管理系统，支持任务创建、监控、取消等功能。

## 功能特点

- 任务状态管理（pending/running/complete/failed/cancelled）
- 任务进度追踪
- 任务取消功能
- 任务持久化存储
- 事件通知系统
- 并发安全
- 完整的错误处理

## 安装

```bash
go get github.com/yourusername/gocli/taskmanager
```

## 使用方法

### 命令行工具

```bash
# 创建新任务
taskmanager create "任务名称" "任务描述"

# 列出所有任务
taskmanager list

# 查看任务状态
taskmanager status <task-id>

# 取消任务
taskmanager cancel <task-id>

# 监控任务进度
taskmanager watch <task-id>
```

### API使用示例

```go
package main

import (
    "context"
    "log"

    "github.com/yourusername/gocli/taskmanager/internal/task"
    "go.uber.org/zap"
)

func main() {
    // 初始化日志
    logger, err := zap.NewDevelopment()
    if err != nil {
        log.Fatal(err)
    }

    // 创建任务管理器
    tm, err := task.NewTaskManager(logger, "tasks.json")
    if err != nil {
        log.Fatal(err)
    }

    // 创建任务
    t, err := tm.CreateTask("示例任务", "这是一个示例任务", nil)
    if err != nil {
        log.Fatal(err)
    }

    // 启动任务
    ctx := context.Background()
    err = tm.StartTask(ctx, t.ID, func(ctx context.Context, t *task.Task) error {
        // 执行任务
        return nil
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

## 项目结构

```
taskmanager/
├── cmd/                # 命令行工具
│   └── taskmanager/   # 主程序入口
├── internal/          # 内部包
│   └── task/         # 任务管理核心实现
├── pkg/              # 对外包
│   └── api/         # 对外API接口
└── examples/         # 示例程序
    └── simple/      # 简单示例
```

## 许可证

MIT 