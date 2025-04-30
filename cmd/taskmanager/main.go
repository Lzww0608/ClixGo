package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/Lzww0608/ClixGo/internal/task"
)

var (
	taskManager *task.TaskManager
	logger      *zap.Logger
)

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化任务管理器
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Fatal("获取用户目录失败", zap.Error(err))
	}

	storePath := filepath.Join(homeDir, ".taskmanager", "tasks.json")
	taskManager, err = task.NewTaskManager(logger, storePath)
	if err != nil {
		logger.Fatal("初始化任务管理器失败", zap.Error(err))
	}
}

func main() {
	cmd := &cobra.Command{
		Use:   "taskmanager",
		Short: "后台任务管理系统",
		Long:  "一个功能完整的后台任务管理系统，支持任务创建、监控、取消等功能",
	}

	cmd.AddCommand(
		createCommand(),
		listCommand(),
		statusCommand(),
		cancelCommand(),
		watchCommand(),
	)

	if err := cmd.Execute(); err != nil {
		logger.Fatal("执行命令失败", zap.Error(err))
	}
} 