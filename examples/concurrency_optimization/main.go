/*
* @Author: Lzww0608
* @Date: 2025-06-06 15:30:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-7 15:52:32
* @Description: 并发优化示例 - 展示如何优化ClixGo项目的并发性能
 */

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/sync"
	"go.uber.org/zap"
)

// OptimizedPerformanceAnalyzer 优化后的性能分析器
type OptimizedPerformanceAnalyzer struct {
	name            string
	ctx             context.Context
	cancel          context.CancelFunc
	shutdownManager *sync.GracefulShutdownManager
	goroutinePool   *sync.GoroutinePool
	metricsChannel  chan PerformanceMetrics
	state           int32
	logger          *zap.Logger
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	Timestamp      time.Time
	CPUUsage       float64
	MemoryUsage    uint64
	GoroutineCount int
	TasksCompleted uint64
}

// NewOptimizedPerformanceAnalyzer 创建优化的性能分析器
func NewOptimizedPerformanceAnalyzer(logger *zap.Logger) *OptimizedPerformanceAnalyzer {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建优雅关闭管理器
	shutdownConfig := sync.DefaultShutdownConfig()
	shutdownConfig.ComponentTimeout = 10 * time.Second
	shutdownConfig.GlobalTimeout = 30 * time.Second
	shutdownManager := sync.NewGracefulShutdownManager(shutdownConfig)

	analyzer := &OptimizedPerformanceAnalyzer{
		name:            "optimized-performance-analyzer",
		ctx:             ctx,
		cancel:          cancel,
		shutdownManager: shutdownManager,
		metricsChannel:  make(chan PerformanceMetrics, 100),
		logger:          logger,
	}

	return analyzer
}

// Name 组件名称
func (opa *OptimizedPerformanceAnalyzer) Name() string {
	return opa.name
}

// Start 启动分析器
func (opa *OptimizedPerformanceAnalyzer) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&opa.state, 0, 1) {
		return fmt.Errorf("分析器已启动")
	}

	opa.logger.Info("启动优化的性能分析器")

	// 启动优雅关闭管理器
	if err := opa.shutdownManager.Start(); err != nil {
		return fmt.Errorf("启动关闭管理器失败: %w", err)
	}

	// 创建独立的goroutine池用于指标收集
	poolConfig := sync.DefaultGoroutinePoolConfig()
	poolConfig.MinWorkers = 2
	poolConfig.MaxWorkers = 6
	poolConfig.WorkerNamePrefix = "metrics-worker"

	opa.goroutinePool = sync.NewGoroutinePool(poolConfig)
	if err := opa.goroutinePool.Start(); err != nil {
		return fmt.Errorf("启动goroutine池失败: %w", err)
	}

	// 注册组件到关闭管理器
	opa.shutdownManager.RegisterComponent(opa)

	// 注册指标channel
	opa.shutdownManager.RegisterChannel("metrics", opa.metricsChannel)

	// 启动指标收集任务
	opa.startMetricsCollection()

	// 启动指标处理器
	opa.shutdownManager.RunManagedGoroutine("metrics-processor", opa.processMetrics)

	return nil
}

// Stop 停止分析器
func (opa *OptimizedPerformanceAnalyzer) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&opa.state, 1, 0) {
		return nil // 已经停止
	}

	opa.logger.Info("停止优化的性能分析器")

	// 优雅关闭
	if err := opa.shutdownManager.StopWithTimeout(20 * time.Second); err != nil {
		opa.logger.Error("关闭管理器停止失败", zap.Error(err))
	}

	// 停止goroutine池
	if opa.goroutinePool != nil {
		if err := opa.goroutinePool.StopWithTimeout(10 * time.Second); err != nil {
			opa.logger.Error("goroutine池停止失败", zap.Error(err))
		}
	}

	opa.cancel()
	return nil
}

// State 组件状态
func (opa *OptimizedPerformanceAnalyzer) State() sync.ComponentState {
	if atomic.LoadInt32(&opa.state) == 1 {
		return sync.StateRunning
	}
	return sync.StateStopped
}

// startMetricsCollection 启动指标收集
func (opa *OptimizedPerformanceAnalyzer) startMetricsCollection() {
	// 定期提交指标收集任务到goroutine池
	opa.shutdownManager.RunManagedGoroutine("metrics-scheduler", func(ctx context.Context) {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 并发收集不同类型的指标
				opa.submitMetricsTask("cpu-metrics", opa.collectCPUMetrics)
				opa.submitMetricsTask("memory-metrics", opa.collectMemoryMetrics)
				opa.submitMetricsTask("runtime-metrics", opa.collectRuntimeMetrics)
			}
		}
	})
}

// submitMetricsTask 提交指标收集任务
func (opa *OptimizedPerformanceAnalyzer) submitMetricsTask(taskID string, collector func() interface{}) {
	task := sync.NewTask(taskID, func(ctx context.Context) error {
		defer func() {
			if r := recover(); r != nil {
				opa.logger.Error("指标收集任务panic",
					zap.String("task", taskID),
					zap.Any("panic", r))
			}
		}()

		// 执行指标收集
		result := collector()

		// 将结果发送到指标通道（非阻塞）
		select {
		case opa.metricsChannel <- PerformanceMetrics{
			Timestamp: time.Now(),
			// 这里简化处理，实际应该根据collector类型处理结果
		}:
		default:
			// 通道满了，丢弃数据
			opa.logger.Warn("指标通道已满，丢弃数据", zap.String("task", taskID))
		}

		return nil
	}).WithTimeout(2 * time.Second).WithPriority(7) // 高优先级

	if err := opa.goroutinePool.Submit(task); err != nil {
		opa.logger.Error("提交指标收集任务失败",
			zap.String("task", taskID),
			zap.Error(err))
	}
}

// collectCPUMetrics 收集CPU指标
func (opa *OptimizedPerformanceAnalyzer) collectCPUMetrics() interface{} {
	// 模拟CPU指标收集
	time.Sleep(10 * time.Millisecond)
	return map[string]interface{}{
		"usage": float64(runtime.NumGoroutine()) * 0.1,
		"cores": runtime.NumCPU(),
		"type":  "cpu",
	}
}

// collectMemoryMetrics 收集内存指标
func (opa *OptimizedPerformanceAnalyzer) collectMemoryMetrics() interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"alloc":       m.Alloc,
		"total_alloc": m.TotalAlloc,
		"sys":         m.Sys,
		"num_gc":      m.NumGC,
		"type":        "memory",
	}
}

// collectRuntimeMetrics 收集运行时指标
func (opa *OptimizedPerformanceAnalyzer) collectRuntimeMetrics() interface{} {
	return map[string]interface{}{
		"goroutines": runtime.NumGoroutine(),
		"cgocalls":   runtime.NumCgoCall(),
		"type":       "runtime",
	}
}

// processMetrics 处理指标数据
func (opa *OptimizedPerformanceAnalyzer) processMetrics(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case metrics := <-opa.metricsChannel:
			// 处理指标数据（这里只是简单打印）
			opa.logger.Debug("处理性能指标",
				zap.Time("timestamp", metrics.Timestamp),
				zap.Float64("cpu_usage", metrics.CPUUsage),
				zap.Uint64("memory_usage", metrics.MemoryUsage),
				zap.Int("goroutine_count", metrics.GoroutineCount))
		}
	}
}

// OptimizedTaskManager 优化的任务管理器
type OptimizedTaskManager struct {
	name            string
	shutdownManager *sync.GracefulShutdownManager
	taskPool        *sync.GoroutinePool
	completedTasks  uint64
	failedTasks     uint64
	state           int32
	logger          *zap.Logger
}

// NewOptimizedTaskManager 创建优化的任务管理器
func NewOptimizedTaskManager(logger *zap.Logger) *OptimizedTaskManager {
	shutdownConfig := sync.DefaultShutdownConfig()
	shutdownManager := sync.NewGracefulShutdownManager(shutdownConfig)

	return &OptimizedTaskManager{
		name:            "optimized-task-manager",
		shutdownManager: shutdownManager,
		logger:          logger,
	}
}

// Name 组件名称
func (otm *OptimizedTaskManager) Name() string {
	return otm.name
}

// Start 启动任务管理器
func (otm *OptimizedTaskManager) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&otm.state, 0, 1) {
		return fmt.Errorf("任务管理器已启动")
	}

	otm.logger.Info("启动优化的任务管理器")

	// 启动关闭管理器
	if err := otm.shutdownManager.Start(); err != nil {
		return fmt.Errorf("启动关闭管理器失败: %w", err)
	}

	// 创建任务执行池
	poolConfig := sync.DefaultGoroutinePoolConfig()
	poolConfig.MinWorkers = 3
	poolConfig.MaxWorkers = 10
	poolConfig.QueueSize = 200
	poolConfig.WorkerNamePrefix = "task-worker"

	otm.taskPool = sync.NewGoroutinePool(poolConfig)
	if err := otm.taskPool.Start(); err != nil {
		return fmt.Errorf("启动任务池失败: %w", err)
	}

	return nil
}

// Stop 停止任务管理器
func (otm *OptimizedTaskManager) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&otm.state, 1, 0) {
		return nil
	}

	otm.logger.Info("停止优化的任务管理器")

	// 停止任务池
	if otm.taskPool != nil {
		if err := otm.taskPool.StopWithTimeout(15 * time.Second); err != nil {
			otm.logger.Error("任务池停止失败", zap.Error(err))
		}
	}

	// 停止关闭管理器
	if err := otm.shutdownManager.StopWithTimeout(10 * time.Second); err != nil {
		otm.logger.Error("关闭管理器停止失败", zap.Error(err))
	}

	return nil
}

// State 组件状态
func (otm *OptimizedTaskManager) State() sync.ComponentState {
	if atomic.LoadInt32(&otm.state) == 1 {
		return sync.StateRunning
	}
	return sync.StateStopped
}

// SubmitTask 提交任务
func (otm *OptimizedTaskManager) SubmitTask(taskName string, taskFunc func() error) error {
	if atomic.LoadInt32(&otm.state) != 1 {
		return fmt.Errorf("任务管理器未运行")
	}

	task := sync.NewTask(taskName, func(ctx context.Context) error {
		start := time.Now()
		defer func() {
			duration := time.Since(start)
			otm.logger.Debug("任务执行完成",
				zap.String("task", taskName),
				zap.Duration("duration", duration))
		}()

		err := taskFunc()
		if err != nil {
			atomic.AddUint64(&otm.failedTasks, 1)
			otm.logger.Error("任务执行失败",
				zap.String("task", taskName),
				zap.Error(err))
		} else {
			atomic.AddUint64(&otm.completedTasks, 1)
		}

		return err
	}).WithPriority(5).WithTimeout(30 * time.Second)

	return otm.taskPool.Submit(task)
}

// GetStatistics 获取统计信息
func (otm *OptimizedTaskManager) GetStatistics() map[string]interface{} {
	poolMetrics := otm.taskPool.GetMetrics()

	return map[string]interface{}{
		"completed_tasks": atomic.LoadUint64(&otm.completedTasks),
		"failed_tasks":    atomic.LoadUint64(&otm.failedTasks),
		"active_workers":  poolMetrics.ActiveWorkers,
		"pending_tasks":   poolMetrics.PendingTasks,
		"total_workers":   poolMetrics.TotalWorkers,
		"pool_completed":  poolMetrics.CompletedTasks,
		"pool_failed":     poolMetrics.FailedTasks,
	}
}

func main() {
	// 创建logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("创建logger失败:", err)
	}
	defer logger.Sync()

	logger.Info("启动并发优化示例程序")

	// 创建主关闭管理器
	mainConfig := sync.DefaultShutdownConfig()
	mainConfig.GlobalTimeout = 60 * time.Second
	mainShutdownManager := sync.NewGracefulShutdownManager(mainConfig)

	if err := mainShutdownManager.Start(); err != nil {
		logger.Fatal("启动主关闭管理器失败", zap.Error(err))
	}

	// 创建并启动优化的性能分析器
	perfAnalyzer := NewOptimizedPerformanceAnalyzer(logger)
	if err := mainShutdownManager.RegisterComponent(perfAnalyzer); err != nil {
		logger.Fatal("注册性能分析器失败", zap.Error(err))
	}

	if err := mainShutdownManager.StartComponent(perfAnalyzer.Name()); err != nil {
		logger.Fatal("启动性能分析器失败", zap.Error(err))
	}

	// 创建并启动优化的任务管理器
	taskManager := NewOptimizedTaskManager(logger)
	if err := mainShutdownManager.RegisterComponent(taskManager); err != nil {
		logger.Fatal("注册任务管理器失败", zap.Error(err))
	}

	if err := mainShutdownManager.StartComponent(taskManager.Name()); err != nil {
		logger.Fatal("启动任务管理器失败", zap.Error(err))
	}

	// 提交一些示例任务
	go func() {
		time.Sleep(2 * time.Second) // 等待组件启动

		for i := 0; i < 20; i++ {
			taskID := i
			taskManager.SubmitTask(fmt.Sprintf("demo-task-%d", taskID), func() error {
				// 模拟不同复杂度的任务
				workTime := time.Duration(50+taskID*10) * time.Millisecond
				time.Sleep(workTime)

				// 模拟偶尔的任务失败
				if taskID%7 == 0 {
					return fmt.Errorf("模拟任务失败")
				}

				logger.Debug("任务完成", zap.Int("task_id", taskID))
				return nil
			})
		}
	}()

	// 定期打印统计信息
	mainShutdownManager.RunManagedGoroutine("stats-printer", func(ctx context.Context) {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats := taskManager.GetStatistics()
				logger.Info("任务统计",
					zap.Any("statistics", stats),
					zap.Int("goroutines", runtime.NumGoroutine()))
			}
		}
	})

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 注册信号channel
	mainShutdownManager.RegisterChannel("signal", sigChan)

	// 添加关闭钩子
	mainShutdownManager.AddShutdownHook(func(ctx context.Context) error {
		logger.Info("执行关闭钩子")
		return nil
	})

	logger.Info("程序运行中，按 Ctrl+C 优雅退出")

	// 等待退出信号
	<-sigChan
	logger.Info("收到退出信号，开始优雅关闭")

	// 优雅关闭
	if err := mainShutdownManager.StopWithTimeout(45 * time.Second); err != nil {
		logger.Error("优雅关闭失败", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("程序已优雅退出")
}

// DemoLoadTesting 负载测试演示
func DemoLoadTesting(logger *zap.Logger) {
	logger.Info("开始负载测试演示")

	// 创建goroutine池
	config := sync.DefaultGoroutinePoolConfig()
	config.MaxWorkers = 20
	config.QueueSize = 500
	pool := sync.NewGoroutinePool(config)

	if err := pool.Start(); err != nil {
		logger.Fatal("启动goroutine池失败", zap.Error(err))
	}
	defer pool.Stop()

	const numTasks = 100
	var (
		completedTasks int32
		failedTasks    int32
		wg             sync.WaitGroup
	)

	start := time.Now()

	// 提交大量任务
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		taskID := i

		task := sync.NewTask(fmt.Sprintf("load-task-%d", taskID), func(ctx context.Context) error {
			defer wg.Done()

			// 模拟不同类型的工作负载
			switch taskID % 3 {
			case 0:
				// CPU密集型任务
				result := 0
				for j := 0; j < 10000; j++ {
					result += j * j
				}
			case 1:
				// I/O模拟任务
				time.Sleep(20 * time.Millisecond)
			case 2:
				// 混合任务
				time.Sleep(10 * time.Millisecond)
				for j := 0; j < 5000; j++ {
					_ = j * j
				}
			}

			// 模拟偶尔的任务失败
			if taskID%13 == 0 {
				atomic.AddInt32(&failedTasks, 1)
				return fmt.Errorf("模拟任务失败")
			}

			atomic.AddInt32(&completedTasks, 1)
			return nil
		}).WithPriority(taskID % 10) // 不同优先级

		if err := pool.Submit(task); err != nil {
			logger.Error("提交任务失败", zap.Int("task_id", taskID), zap.Error(err))
			wg.Done()
		}
	}

	// 等待所有任务完成
	wg.Wait()
	duration := time.Since(start)

	// 获取池指标
	metrics := pool.GetMetrics()

	logger.Info("负载测试完成",
		zap.Duration("total_time", duration),
		zap.Int32("completed_tasks", atomic.LoadInt32(&completedTasks)),
		zap.Int32("failed_tasks", atomic.LoadInt32(&failedTasks)),
		zap.Uint64("pool_completed", metrics.CompletedTasks),
		zap.Uint64("pool_failed", metrics.FailedTasks),
		zap.Int32("max_workers", metrics.TotalWorkers),
		zap.Float64("tasks_per_second", float64(numTasks)/duration.Seconds()))
}
