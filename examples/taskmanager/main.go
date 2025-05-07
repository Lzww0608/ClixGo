package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/task"
	"go.uber.org/zap"
)

func main() {
	fmt.Println("开始任务管理测试...")

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
		return
	}

	// 创建任务
	t, err := tm.CreateTask("测试任务", "简单测试任务", nil)
	if err != nil {
		logger.Fatal("创建任务失败", zap.Error(err))
		return
	}
	fmt.Printf("已创建任务: ID=%s, 名称=%s\n", t.ID, t.Name)

	// 使用WaitGroup确保等待任务完成
	var wg sync.WaitGroup
	wg.Add(1)

	// 启动任务
	ctx := context.Background()
	err = tm.StartTask(ctx, t.ID, func(ctx context.Context, t *task.Task) error {
		defer wg.Done() // 任务完成时通知主线程

		fmt.Println("任务已启动，开始执行...")

		// 只有5个简单步骤
		for i := 1; i <= 5; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				// 直接打印进度，不依赖通知机制
				progress := float64(i) / 5.0
				fmt.Printf("更新任务进度: %.0f%%\n", progress*100)

				// 更新任务进度
				err := tm.UpdateTaskProgress(t.ID, progress)
				if err != nil {
					fmt.Printf("更新进度出错: %v\n", err)
				}

				// 模拟工作
				time.Sleep(500 * time.Millisecond)
			}
		}

		fmt.Println("任务处理完成")
		return nil
	})

	if err != nil {
		logger.Fatal("启动任务失败", zap.Error(err))
		return
	}

	// 监控任务进度
	go func() {
		for {
			taskInfo, err := tm.GetTask(t.ID)
			if err != nil {
				fmt.Printf("获取任务失败: %v\n", err)
				return
			}

			fmt.Printf("当前状态: %s, 进度: %.0f%%\n", taskInfo.Status, taskInfo.Progress*100)

			if taskInfo.Status == task.TaskStatusComplete ||
				taskInfo.Status == task.TaskStatusFailed ||
				taskInfo.Status == task.TaskStatusCancelled {
				break
			}

			time.Sleep(200 * time.Millisecond)
		}
	}()

	// 等待任务完成
	fmt.Println("等待任务完成...")
	wg.Wait()

	// 获取最终状态
	taskInfo, err := tm.GetTask(t.ID)
	if err != nil {
		logger.Fatal("获取任务失败", zap.Error(err))
		return
	}

	fmt.Printf("任务最终状态: %s, 进度: %.0f%%\n", taskInfo.Status, taskInfo.Progress*100)
	fmt.Println("测试完成!")
}
