package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/Lzww0608/ClixGo/pkg/task"
)

func main() {
	// 初始化日志
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}

	// 创建任务管理器
	tm, err := task.NewTaskManager(logger, "tasks.json")
	if err != nil {
		logger.Fatal("初始化任务管理器失败", zap.Error(err))
	}

	// 创建一个长时间运行的任务
	t, err := tm.CreateTask("数据处理", "处理大量数据的示例任务", nil)
	if err != nil {
		logger.Fatal("创建任务失败", zap.Error(err))
	}

	// 订阅任务更新
	updates := tm.SubscribeTask(t.ID)
	defer tm.UnsubscribeTask(t.ID, updates)

	// 在后台监控任务进度
	go func() {
		for taskUpdate := range updates {
			fmt.Printf("\r任务进度: %.1f%% | 状态: %s", taskUpdate.Progress*100, taskUpdate.Status)
			if taskUpdate.Status == task.TaskStatusComplete || taskUpdate.Status == task.TaskStatusFailed {
				fmt.Println()
				if taskUpdate.Error != "" {
					fmt.Printf("错误: %s\n", taskUpdate.Error)
				}
				return
			}
		}
	}()

	// 启动任务
	ctx := context.Background()
	err = tm.StartTask(ctx, t.ID, func(ctx context.Context, t *task.Task) error {
		// 模拟长时间运行的任务
		totalSteps := 100
		for i := 0; i < totalSteps; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				// 模拟工作
				time.Sleep(100 * time.Millisecond)

				// 随机模拟错误
				if rand.Float64() < 0.01 {
					return fmt.Errorf("随机错误在步骤 %d", i)
				}

				// 更新进度
				progress := float64(i+1) / float64(totalSteps)
				if err := tm.UpdateTaskProgress(t.ID, progress); err != nil {
					logger.Error("更新进度失败", zap.Error(err))
				}
			}
		}
		return nil
	})

	if err != nil {
		logger.Fatal("启动任务失败", zap.Error(err))
	}

	// 等待任务完成
	taskInfo, err := tm.GetTask(t.ID)
	if err != nil {
		logger.Fatal("获取任务失败", zap.Error(err))
	}

	for taskInfo.Status != task.TaskStatusComplete && taskInfo.Status != task.TaskStatusFailed {
		time.Sleep(100 * time.Millisecond)
		taskInfo, err = tm.GetTask(t.ID)
		if err != nil {
			logger.Fatal("获取任务失败", zap.Error(err))
		}
	}

	fmt.Println("\n任务完成!")
}
