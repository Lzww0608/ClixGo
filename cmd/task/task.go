package task

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/Lzww0608/ClixGo/pkg/task"
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

	storePath := filepath.Join(homeDir, ".gocli", "tasks.json")
	taskManager, err = task.NewTaskManager(logger, storePath)
	if err != nil {
		logger.Fatal("初始化任务管理器失败", zap.Error(err))
	}
}

// Command 返回任务管理命令
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "后台任务管理",
		Long:  "管理长时间运行的后台任务，包括创建、监控、取消等操作",
	}

	cmd.AddCommand(
		createCommand(),
		listCommand(),
		statusCommand(),
		cancelCommand(),
		watchCommand(),
	)

	return cmd
}

// createCommand 创建任务命令
func createCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [name] [description]",
		Short: "创建新任务",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			task, err := taskManager.CreateTask(args[0], args[1], nil)
			if err != nil {
				return err
			}

			fmt.Printf("任务已创建，ID: %s\n", task.ID)
			return nil
		},
	}

	return cmd
}

// listCommand 列出任务命令
func listCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出所有任务",
		RunE: func(cmd *cobra.Command, args []string) error {
			tasks := taskManager.ListTasks()
			if len(tasks) == 0 {
				fmt.Println("没有任务")
				return nil
			}

			fmt.Println("任务列表:")
			for _, t := range tasks {
				fmt.Printf("ID: %s\n", t.ID)
				fmt.Printf("  名称: %s\n", t.Name)
				fmt.Printf("  描述: %s\n", t.Description)
				fmt.Printf("  状态: %s\n", t.Status)
				fmt.Printf("  进度: %.1f%%\n", t.Progress*100)
				if t.Error != "" {
					fmt.Printf("  错误: %s\n", t.Error)
				}
				fmt.Printf("  创建时间: %s\n", t.CreatedAt.Format(time.RFC3339))
				if t.StartedAt != nil {
					fmt.Printf("  开始时间: %s\n", t.StartedAt.Format(time.RFC3339))
				}
				if t.FinishedAt != nil {
					fmt.Printf("  完成时间: %s\n", t.FinishedAt.Format(time.RFC3339))
				}
				fmt.Println()
			}

			return nil
		},
	}

	return cmd
}

// statusCommand 查看任务状态命令
func statusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [task-id]",
		Short: "查看任务状态",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			task, err := taskManager.GetTask(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("任务状态:\n")
			fmt.Printf("ID: %s\n", task.ID)
			fmt.Printf("名称: %s\n", task.Name)
			fmt.Printf("描述: %s\n", task.Description)
			fmt.Printf("状态: %s\n", task.Status)
			fmt.Printf("进度: %.1f%%\n", task.Progress*100)
			if task.Error != "" {
				fmt.Printf("错误: %s\n", task.Error)
			}
			fmt.Printf("创建时间: %s\n", task.CreatedAt.Format(time.RFC3339))
			if task.StartedAt != nil {
				fmt.Printf("开始时间: %s\n", task.StartedAt.Format(time.RFC3339))
			}
			if task.FinishedAt != nil {
				fmt.Printf("完成时间: %s\n", task.FinishedAt.Format(time.RFC3339))
			}

			return nil
		},
	}

	return cmd
}

// cancelCommand 取消任务命令
func cancelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel [task-id]",
		Short: "取消任务",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := taskManager.CancelTask(args[0]); err != nil {
				return err
			}

			fmt.Println("任务已取消")
			return nil
		},
	}

	return cmd
}

// watchCommand 监控任务命令
func watchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch [task-id]",
		Short: "监控任务进度",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			updates := taskManager.SubscribeTask(args[0])
			defer taskManager.UnsubscribeTask(args[0], updates)

			fmt.Println("正在监控任务进度...")
			for {
				select {
				case t := <-updates:
					fmt.Printf("\r进度: %.1f%% | 状态: %s", t.Progress*100, t.Status)
					if t.Status == task.TaskStatusComplete || t.Status == task.TaskStatusFailed || t.Status == task.TaskStatusCancelled {
						fmt.Println()
						if t.Error != "" {
							fmt.Printf("错误: %s\n", t.Error)
						}
						return nil
					}
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		},
	}

	return cmd
} 