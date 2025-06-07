/*
* @Author: Lzww0608
* @Date: 2025-6-6 23:47:45
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-7 15:52:27
* @Description: 高级并发优化演示 - 展示ClixGo项目中的并发模型优化技术
 */

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/network"
	clixgosync "github.com/Lzww0608/ClixGo/pkg/sync"
	"go.uber.org/zap"
)

// DemoConfig 演示配置
type DemoConfig struct {
	// 演示持续时间
	DemoDuration time.Duration
	// 并发任务数量
	ConcurrentTasks int
	// 是否启用详细日志
	VerboseLogging bool
	// 是否启用网络监控
	EnableNetworkMonitor bool
	// 任务提交间隔
	TaskSubmitInterval time.Duration
	// 任务执行超时
	TaskTimeout time.Duration
}

// main 主函数 - 高级并发优化演示
func main() {
	config := DemoConfig{
		DemoDuration:         30 * time.Second, // 缩短到30秒避免长时间运行
		ConcurrentTasks:      20,               // 减少并发任务数
		VerboseLogging:       false,            // 减少日志输出
		EnableNetworkMonitor: true,
		TaskSubmitInterval:   200 * time.Millisecond, // 降低提交频率
		TaskTimeout:          5 * time.Second,        // 任务超时
	}

	fmt.Println("🚀 启动ClixGo高级并发优化演示")

	// 创建演示
	demo, err := createConcurrencyDemo(config)
	if err != nil {
		log.Fatal("创建演示失败:", err)
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动演示
	if err := demo.Start(); err != nil {
		log.Fatal("启动演示失败:", err)
	}

	// 等待信号或演示结束
	go func() {
		time.Sleep(config.DemoDuration)
		sigChan <- syscall.SIGTERM
	}()

	// 等待停止信号
	sig := <-sigChan
	fmt.Printf("接收到停止信号: %s\n", sig.String())

	// 停止演示
	if err := demo.Stop(); err != nil {
		log.Printf("停止演示失败: %v", err)
	}

	fmt.Println("🎉 高级并发优化演示完成！")
}

// ConcurrencyDemo 并发演示结构
type ConcurrencyDemo struct {
	logger          *zap.Logger
	shutdownManager *clixgosync.GracefulShutdownManager
	goroutinePool   *clixgosync.GoroutinePool
	networkMonitor  *network.OptimizedRealtimeNetworkMonitor
	config          DemoConfig
	startTime       time.Time
	taskCounter     uint64
	successCounter  uint64
}

// createConcurrencyDemo 创建并发演示
func createConcurrencyDemo(config DemoConfig) (*ConcurrencyDemo, error) {
	// 初始化日志器
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, fmt.Errorf("初始化日志器失败: %w", err)
	}

	// 配置优雅关闭管理器
	shutdownConfig := clixgosync.DefaultShutdownConfig()
	shutdownConfig.ComponentTimeout = 30 * time.Second
	shutdownConfig.GlobalTimeout = 60 * time.Second
	shutdownConfig.ParallelShutdown = true

	// 配置goroutine池 - 优化配置避免队列满载
	poolConfig := clixgosync.DefaultGoroutinePoolConfig()
	poolConfig.MinWorkers = 3
	poolConfig.MaxWorkers = 10
	poolConfig.QueueSize = 100 // 减小队列大小
	poolConfig.WorkerNamePrefix = "demo-worker"

	demo := &ConcurrencyDemo{
		logger:          logger,
		shutdownManager: clixgosync.NewGracefulShutdownManager(shutdownConfig),
		goroutinePool:   clixgosync.NewGoroutinePool(poolConfig),
		config:          config,
		startTime:       time.Now(),
	}

	return demo, nil
}

// Start 启动演示
func (cd *ConcurrencyDemo) Start() error {
	cd.logger.Info("启动并发优化演示",
		zap.Duration("duration", cd.config.DemoDuration),
		zap.Int("concurrent_tasks", cd.config.ConcurrentTasks),
	)

	// 启动优雅关闭管理器
	if err := cd.shutdownManager.Start(); err != nil {
		return fmt.Errorf("启动优雅关闭管理器失败: %w", err)
	}

	// 启动goroutine池
	if err := cd.goroutinePool.Start(); err != nil {
		return fmt.Errorf("启动goroutine池失败: %w", err)
	}

	// 启动网络监控器（如果启用）
	if cd.config.EnableNetworkMonitor {
		if err := cd.startNetworkMonitor(); err != nil {
			cd.logger.Error("启动网络监控器失败", zap.Error(err))
		}
	}

	// 启动演示任务
	cd.startDemoTasks()

	// 启动监控任务
	cd.startMonitoring()

	return nil
}

// Stop 停止演示
func (cd *ConcurrencyDemo) Stop() error {
	cd.logger.Info("停止并发优化演示")

	// 打印最终统计
	cd.printFinalStats()

	// 先停止网络监控器（避免嵌套关闭超时）
	if cd.networkMonitor != nil {
		if err := cd.networkMonitor.Stop(); err != nil {
			cd.logger.Warn("停止网络监控器失败", zap.Error(err))
		}
	}

	// 使用优雅关闭管理器停止其他组件
	return cd.shutdownManager.StopWithTimeout(30 * time.Second)
}

// startNetworkMonitor 启动网络监控器
func (cd *ConcurrencyDemo) startNetworkMonitor() error {
	networkConfig := network.RealtimeMonitorConfig{
		UpdateInterval:   3 * time.Second,
		Timeout:          10 * time.Second,
		MaxHistory:       20,
		EnableAlerts:     true,
		MonitoredTargets: []string{"8.8.8.8", "1.1.1.1"},
	}

	cd.networkMonitor = network.NewOptimizedRealtimeNetworkMonitor(networkConfig)
	return cd.networkMonitor.Start()
}

// startDemoTasks 启动演示任务
func (cd *ConcurrencyDemo) startDemoTasks() {
	cd.shutdownManager.RunManagedGoroutine("task-submitter", func(ctx context.Context) {
		ticker := time.NewTicker(cd.config.TaskSubmitInterval)
		defer ticker.Stop()

		endTime := cd.startTime.Add(cd.config.DemoDuration)
		taskSubmitted := 0

		cd.logger.Info("开始任务提交",
			zap.Duration("interval", cd.config.TaskSubmitInterval),
			zap.Duration("duration", cd.config.DemoDuration))

		for {
			select {
			case <-ctx.Done():
				cd.logger.Info("任务提交器收到停止信号")
				return
			case <-ticker.C:
				if time.Now().After(endTime) {
					cd.logger.Info("演示时间结束，停止提交任务", zap.Int("total_submitted", taskSubmitted))
					return
				}

				// 减少任务执行时间，避免队列堆积
				if cd.submitTaskWithBackoff("cpu", 100*time.Millisecond, 7) {
					taskSubmitted++
				}
				if cd.submitTaskWithBackoff("io", 50*time.Millisecond, 5) {
					taskSubmitted++
				}
				if cd.submitTaskWithBackoff("network", 150*time.Millisecond, 8) {
					taskSubmitted++
				}

				// 定期输出提交统计
				if taskSubmitted%50 == 0 && taskSubmitted > 0 {
					cd.logger.Info("任务提交进度", zap.Int("submitted", taskSubmitted))
				}
			}
		}
	})
}

// submitTask 提交任务
func (cd *ConcurrencyDemo) submitTask(taskType string, duration time.Duration, priority int) {
	taskID := fmt.Sprintf("%s-task-%d", taskType, atomic.AddUint64(&cd.taskCounter, 1))

	task := clixgosync.NewTask(taskID, func(ctx context.Context) error {
		// 模拟不同类型的工作
		select {
		case <-time.After(duration):
			atomic.AddUint64(&cd.successCounter, 1)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}).WithPriority(priority).WithTimeout(cd.config.TaskTimeout)

	if err := cd.goroutinePool.Submit(task); err != nil {
		cd.logger.Error("提交任务失败", zap.Error(err))
	}
}

// submitTaskWithBackoff 带退避机制的任务提交
func (cd *ConcurrencyDemo) submitTaskWithBackoff(taskType string, duration time.Duration, priority int) bool {
	taskID := fmt.Sprintf("%s-task-%d", taskType, atomic.AddUint64(&cd.taskCounter, 1))

	task := clixgosync.NewTask(taskID, func(ctx context.Context) error {
		// 模拟不同类型的工作
		select {
		case <-time.After(duration):
			atomic.AddUint64(&cd.successCounter, 1)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}).WithPriority(priority).WithTimeout(cd.config.TaskTimeout)

	if err := cd.goroutinePool.Submit(task); err != nil {
		// 队列满时不记录错误，避免日志泛滥
		if cd.config.VerboseLogging {
			cd.logger.Debug("提交任务失败",
				zap.String("task_id", taskID),
				zap.String("task_type", taskType),
				zap.Error(err))
		}
		return false
	}

	return true
}

// startMonitoring 启动监控
func (cd *ConcurrencyDemo) startMonitoring() {
	// 监控goroutine池状态
	cd.shutdownManager.RunManagedGoroutine("pool-monitor", func(ctx context.Context) {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				metrics := cd.goroutinePool.GetMetrics()
				cd.logger.Info("Goroutine池状态",
					zap.Int32("active_workers", metrics.ActiveWorkers),
					zap.Int32("pending_tasks", metrics.PendingTasks),
					zap.Uint64("completed_tasks", metrics.CompletedTasks),
				)
			}
		}
	})

	// 监控网络状态（如果启用）
	if cd.config.EnableNetworkMonitor && cd.networkMonitor != nil {
		cd.shutdownManager.RunManagedGoroutine("network-monitor", func(ctx context.Context) {
			updateChan := cd.networkMonitor.GetUpdateChannel()

			for {
				select {
				case <-ctx.Done():
					return
				case snapshot := <-updateChan:
					cd.logger.Debug("网络快照更新",
						zap.Int("interfaces", len(snapshot.Interfaces)),
						zap.Float64("score", snapshot.PerformanceScore),
					)
				}
			}
		})
	}
}

// printFinalStats 打印最终统计
func (cd *ConcurrencyDemo) printFinalStats() {
	duration := time.Since(cd.startTime)
	totalTasks := atomic.LoadUint64(&cd.taskCounter)
	successTasks := atomic.LoadUint64(&cd.successCounter)

	successRate := float64(0)
	throughput := float64(0)
	if totalTasks > 0 {
		successRate = float64(successTasks) / float64(totalTasks) * 100
		throughput = float64(totalTasks) / duration.Seconds()
	}

	poolMetrics := cd.goroutinePool.GetMetrics()

	cd.logger.Info("演示最终统计",
		zap.Duration("总运行时间", duration),
		zap.Uint64("总任务数", totalTasks),
		zap.Uint64("成功任务数", successTasks),
		zap.Float64("成功率(%)", successRate),
		zap.Float64("吞吐量(任务/秒)", throughput),
		zap.Uint64("池完成任务", poolMetrics.CompletedTasks),
		zap.Duration("平均等待时间", poolMetrics.AverageWaitTime),
		zap.Duration("平均执行时间", poolMetrics.AverageExecTime),
	)
}
