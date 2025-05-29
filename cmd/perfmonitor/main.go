package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/performance"
	"github.com/Lzww0608/ClixGo/pkg/task"
	"go.uber.org/zap"
)

var (
	// 性能分析器配置
	analyzerSampleInterval = flag.Duration("analyzer-interval", 1*time.Second, "性能分析采样间隔")
	analyzerTimeout        = flag.Duration("analyzer-timeout", 30*time.Second, "性能分析超时时间")
	analyzerMaxHistory     = flag.Int("analyzer-history", 100, "最大历史记录数")
	analyzerEnableAlerts   = flag.Bool("analyzer-alerts", true, "启用性能告警")

	// 优化器配置
	optimizerInterval     = flag.Duration("optimizer-interval", 30*time.Second, "性能优化间隔")
	optimizerTimeout      = flag.Duration("optimizer-timeout", 10*time.Second, "优化操作超时时间")
	optimizerEnableGC     = flag.Bool("optimizer-gc", true, "启用自动垃圾回收优化")
	optimizerEnableMemory = flag.Bool("optimizer-memory", true, "启用内存优化")
	optimizerEnableCPU    = flag.Bool("optimizer-cpu", true, "启用CPU优化")

	// 阈值配置
	memoryThreshold    = flag.Float64("memory-threshold", 100.0, "内存使用阈值(MB)")
	cpuThreshold       = flag.Float64("cpu-threshold", 80.0, "CPU使用阈值(%)")
	gcThreshold        = flag.Float64("gc-threshold", 50.0, "GC触发阈值(MB)")
	goroutineThreshold = flag.Int("goroutine-threshold", 1000, "协程数量阈值")

	// 运行模式
	mode             = flag.String("mode", "monitor", "运行模式: monitor(监控), analyze(分析), optimize(优化), demo(演示)")
	demoTaskCount    = flag.Int("demo-tasks", 5, "演示模式下的任务数量")
	demoTaskDuration = flag.Duration("demo-duration", 10*time.Second, "演示任务运行时间")

	// 输出配置
	outputFormat = flag.String("output", "text", "输出格式: text, json")
	outputFile   = flag.String("output-file", "", "输出文件路径")
	verbose      = flag.Bool("verbose", false, "详细输出")
)

func main() {
	flag.Parse()

	// 创建日志器
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("创建日志器失败:", err)
	}
	defer logger.Sync()

	// 根据模式运行
	switch *mode {
	case "monitor":
		runMonitorMode(logger)
	case "analyze":
		runAnalyzeMode(logger)
	case "optimize":
		runOptimizeMode(logger)
	case "demo":
		runDemoMode(logger)
	default:
		fmt.Printf("未知模式: %s\n", *mode)
		flag.Usage()
		os.Exit(1)
	}
}

// runMonitorMode 运行监控模式
func runMonitorMode(logger *zap.Logger) {
	fmt.Println("🔍 启动性能监控模式...")

	// 创建性能分析器
	analyzerConfig := performance.AnalyzerConfig{
		SampleInterval: *analyzerSampleInterval,
		Timeout:        *analyzerTimeout,
		MaxHistory:     *analyzerMaxHistory,
		EnableAlerts:   *analyzerEnableAlerts,
		AlertThresholds: performance.AlertThresholds{
			CPUUsagePercent: *cpuThreshold,
			MemoryUsageMB:   *memoryThreshold,
			ExecutionTimeMs: int64(analyzerTimeout.Milliseconds()),
			GoroutineCount:  *goroutineThreshold,
			GCPauseMs:       10.0,
		},
	}

	analyzer := performance.NewTaskPerformanceAnalyzer(analyzerConfig)

	// 启动分析器
	if err := analyzer.Start(); err != nil {
		logger.Fatal("启动性能分析器失败", zap.Error(err))
	}
	defer analyzer.Stop()

	// 创建性能优化器
	optimizerConfig := performance.OptimizerConfig{
		OptimizationInterval: *optimizerInterval,
		Timeout:              *optimizerTimeout,
		EnableAutoGC:         *optimizerEnableGC,
		EnableMemoryLimit:    *optimizerEnableMemory,
		EnableCPUThrottling:  *optimizerEnableCPU,
		MemoryThresholdMB:    *memoryThreshold,
		CPUThresholdPercent:  *cpuThreshold,
		GCThresholdMB:        *gcThreshold,
		GoroutineThreshold:   *goroutineThreshold,
	}

	optimizer := performance.NewPerformanceOptimizer(optimizerConfig)

	// 启动优化器
	if err := optimizer.Start(); err != nil {
		logger.Fatal("启动性能优化器失败", zap.Error(err))
	}
	defer optimizer.Stop()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动监控循环
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	fmt.Println("✅ 性能监控已启动，按 Ctrl+C 退出")

	for {
		select {
		case <-sigChan:
			fmt.Println("\n🛑 收到退出信号，正在停止...")
			return
		case <-ticker.C:
			// 显示当前状态
			displayMonitorStatus(analyzer, optimizer)
		case alert := <-analyzer.GetAlertChannel():
			displayAlert("性能分析", alert.Type, alert.Message)
		case alert := <-optimizer.GetAlertChannel():
			displayOptimizationAlert("性能优化", alert.Type, alert.Message)
		case err := <-analyzer.GetErrorChannel():
			logger.Warn("性能分析器错误", zap.Error(err))
		case err := <-optimizer.GetErrorChannel():
			logger.Warn("性能优化器错误", zap.Error(err))
		}
	}
}

// runAnalyzeMode 运行分析模式
func runAnalyzeMode(logger *zap.Logger) {
	fmt.Println("📊 启动性能分析模式...")

	// 创建任务管理器
	taskManager, err := task.NewTaskManager(logger, "/tmp/clixgo_tasks.json")
	if err != nil {
		logger.Fatal("创建任务管理器失败", zap.Error(err))
	}

	// 创建性能分析器
	analyzerConfig := performance.AnalyzerConfig{
		SampleInterval: *analyzerSampleInterval,
		Timeout:        *analyzerTimeout,
		MaxHistory:     *analyzerMaxHistory,
		EnableAlerts:   *analyzerEnableAlerts,
		AlertThresholds: performance.AlertThresholds{
			CPUUsagePercent: *cpuThreshold,
			MemoryUsageMB:   *memoryThreshold,
			ExecutionTimeMs: int64(analyzerTimeout.Milliseconds()),
			GoroutineCount:  *goroutineThreshold,
			GCPauseMs:       10.0,
		},
	}

	analyzer := performance.NewTaskPerformanceAnalyzer(analyzerConfig)

	// 启动分析器
	if err := analyzer.Start(); err != nil {
		logger.Fatal("启动性能分析器失败", zap.Error(err))
	}
	defer analyzer.Stop()

	// 创建示例任务
	task1, err := taskManager.CreateTask("CPU密集型任务", "模拟CPU密集型计算", nil)
	if err != nil {
		logger.Fatal("创建任务失败", zap.Error(err))
	}

	task2, err := taskManager.CreateTask("内存密集型任务", "模拟内存密集型操作", nil)
	if err != nil {
		logger.Fatal("创建任务失败", zap.Error(err))
	}

	// 分析任务1
	fmt.Printf("🔍 开始分析任务: %s\n", task1.Name)
	ctx1, err := analyzer.StartTaskAnalysis(task1.ID, task1.Name)
	if err != nil {
		logger.Fatal("开始任务分析失败", zap.Error(err))
	}

	// 启动任务1
	taskManager.StartTask(context.Background(), task1.ID, func(ctx context.Context, t *task.Task) error {
		// 模拟CPU密集型工作
		for i := 0; i < 1000000; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				// CPU密集型计算
				_ = i * i * i
			}

			if i%100000 == 0 {
				taskManager.UpdateTaskProgress(t.ID, float64(i)/1000000.0)
			}
		}
		return nil
	})

	// 等待任务1完成
	time.Sleep(3 * time.Second)
	metrics1 := analyzer.FinishTaskAnalysis(ctx1)

	// 分析任务2
	fmt.Printf("🔍 开始分析任务: %s\n", task2.Name)
	ctx2, err := analyzer.StartTaskAnalysis(task2.ID, task2.Name)
	if err != nil {
		logger.Fatal("开始任务分析失败", zap.Error(err))
	}

	// 启动任务2
	taskManager.StartTask(context.Background(), task2.ID, func(ctx context.Context, t *task.Task) error {
		// 模拟内存密集型工作
		data := make([][]byte, 100)
		for i := 0; i < 100; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				data[i] = make([]byte, 1024*1024) // 1MB
				taskManager.UpdateTaskProgress(t.ID, float64(i)/100.0)
				time.Sleep(50 * time.Millisecond)
			}
		}
		return nil
	})

	// 等待任务2完成
	time.Sleep(6 * time.Second)
	metrics2 := analyzer.FinishTaskAnalysis(ctx2)

	// 输出分析结果
	fmt.Println("\n📈 任务性能分析结果:")
	displayTaskMetrics("CPU密集型任务", metrics1)
	displayTaskMetrics("内存密集型任务", metrics2)

	// 保存结果到文件
	if *outputFile != "" {
		saveAnalysisResults(*outputFile, map[string]*performance.TaskMetrics{
			task1.ID: metrics1,
			task2.ID: metrics2,
		})
	}
}

// runOptimizeMode 运行优化模式
func runOptimizeMode(logger *zap.Logger) {
	fmt.Println("⚡ 启动性能优化模式...")

	// 创建性能优化器
	optimizerConfig := performance.OptimizerConfig{
		OptimizationInterval: *optimizerInterval,
		Timeout:              *optimizerTimeout,
		EnableAutoGC:         *optimizerEnableGC,
		EnableMemoryLimit:    *optimizerEnableMemory,
		EnableCPUThrottling:  *optimizerEnableCPU,
		MemoryThresholdMB:    *memoryThreshold,
		CPUThresholdPercent:  *cpuThreshold,
		GCThresholdMB:        *gcThreshold,
		GoroutineThreshold:   *goroutineThreshold,
	}

	optimizer := performance.NewPerformanceOptimizer(optimizerConfig)

	// 启动优化器
	if err := optimizer.Start(); err != nil {
		logger.Fatal("启动性能优化器失败", zap.Error(err))
	}
	defer optimizer.Stop()

	// 显示初始状态
	fmt.Println("📊 优化前状态:")
	displayOptimizerMetrics(optimizer.GetMetrics())

	// 创建一些负载来触发优化
	fmt.Println("🔄 创建系统负载...")
	createSystemLoad()

	// 强制执行优化
	fmt.Println("⚡ 执行性能优化...")
	if err := optimizer.ForceOptimization(); err != nil {
		logger.Fatal("执行优化失败", zap.Error(err))
	}

	// 等待优化完成
	time.Sleep(2 * time.Second)

	// 显示优化后状态
	fmt.Println("📊 优化后状态:")
	displayOptimizerMetrics(optimizer.GetMetrics())

	// 监控优化效果
	fmt.Println("🔍 监控优化效果 (10秒)...")
	monitorOptimizationEffects(optimizer, 10*time.Second)
}

// runDemoMode 运行演示模式
func runDemoMode(logger *zap.Logger) {
	fmt.Println("🎭 启动演示模式...")

	// 创建任务管理器
	taskManager, err := task.NewTaskManager(logger, "/tmp/clixgo_demo_tasks.json")
	if err != nil {
		logger.Fatal("创建任务管理器失败", zap.Error(err))
	}

	// 创建性能分析器
	analyzerConfig := performance.AnalyzerConfig{
		SampleInterval: 500 * time.Millisecond,
		Timeout:        *analyzerTimeout,
		MaxHistory:     50,
		EnableAlerts:   true,
		AlertThresholds: performance.AlertThresholds{
			CPUUsagePercent: 50.0, // 降低阈值以便演示
			MemoryUsageMB:   50.0,
			ExecutionTimeMs: 5000,
			GoroutineCount:  50,
			GCPauseMs:       5.0,
		},
	}

	analyzer := performance.NewTaskPerformanceAnalyzer(analyzerConfig)

	// 启动分析器
	if err := analyzer.Start(); err != nil {
		logger.Fatal("启动性能分析器失败", zap.Error(err))
	}
	defer analyzer.Stop()

	// 创建性能优化器
	optimizerConfig := performance.OptimizerConfig{
		OptimizationInterval: 5 * time.Second,
		Timeout:              5 * time.Second,
		EnableAutoGC:         true,
		EnableMemoryLimit:    true,
		EnableCPUThrottling:  true,
		MemoryThresholdMB:    30.0, // 降低阈值以便演示
		CPUThresholdPercent:  50.0,
		GCThresholdMB:        20.0,
		GoroutineThreshold:   50,
	}

	optimizer := performance.NewPerformanceOptimizer(optimizerConfig)

	// 启动优化器
	if err := optimizer.Start(); err != nil {
		logger.Fatal("启动性能优化器失败", zap.Error(err))
	}
	defer optimizer.Stop()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("🎬 开始演示，将创建 %d 个任务，每个运行 %v\n", *demoTaskCount, *demoTaskDuration)

	// 创建并运行演示任务
	for i := 0; i < *demoTaskCount; i++ {
		taskName := fmt.Sprintf("演示任务-%d", i+1)
		demoTask, err := taskManager.CreateTask(taskName, "演示用的混合负载任务", nil)
		if err != nil {
			logger.Error("创建演示任务失败", zap.Error(err))
			continue
		}

		// 开始性能分析
		ctx, err := analyzer.StartTaskAnalysis(demoTask.ID, demoTask.Name)
		if err != nil {
			logger.Error("开始任务分析失败", zap.Error(err))
			continue
		}

		// 启动任务
		taskManager.StartTask(context.Background(), demoTask.ID, func(taskCtx context.Context, t *task.Task) error {
			return runDemoTask(taskCtx, t, taskManager, *demoTaskDuration)
		})

		// 延迟启动下一个任务
		time.Sleep(1 * time.Second)

		// 在任务运行期间完成分析
		go func(analysisCtx *performance.TaskExecutionContext, taskID string) {
			time.Sleep(*demoTaskDuration + 1*time.Second)
			metrics := analyzer.FinishTaskAnalysis(analysisCtx)
			if metrics != nil {
				fmt.Printf("✅ 任务 %s 完成，执行时间: %v\n", taskID, metrics.Duration)
			}
		}(ctx, demoTask.ID)
	}

	// 监控演示过程
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	fmt.Println("🔍 监控演示过程，按 Ctrl+C 退出")

	for {
		select {
		case <-sigChan:
			fmt.Println("\n🛑 演示结束")
			return
		case <-ticker.C:
			displayDemoStatus(analyzer, optimizer)
		case alert := <-analyzer.GetAlertChannel():
			displayAlert("分析器", alert.Type, alert.Message)
		case alert := <-optimizer.GetAlertChannel():
			displayOptimizationAlert("优化器", alert.Type, alert.Message)
		}
	}
}

// runDemoTask 运行演示任务
func runDemoTask(ctx context.Context, t *task.Task, taskManager *task.TaskManager, duration time.Duration) error {
	startTime := time.Now()
	endTime := startTime.Add(duration)

	// 混合负载：CPU + 内存 + 协程
	data := make([][]byte, 0)

	for time.Now().Before(endTime) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// CPU密集型计算
			for i := 0; i < 10000; i++ {
				_ = i * i
			}

			// 内存分配
			if len(data) < 50 {
				data = append(data, make([]byte, 1024*1024)) // 1MB
			}

			// 启动一些短期协程
			for i := 0; i < 5; i++ {
				go func() {
					time.Sleep(100 * time.Millisecond)
				}()
			}

			// 更新进度
			elapsed := time.Since(startTime)
			progress := float64(elapsed) / float64(duration)
			if progress > 1.0 {
				progress = 1.0
			}
			taskManager.UpdateTaskProgress(t.ID, progress)

			time.Sleep(100 * time.Millisecond)
		}
	}

	return nil
}

// createSystemLoad 创建系统负载
func createSystemLoad() {
	// 分配内存
	data := make([][]byte, 100)
	for i := range data {
		data[i] = make([]byte, 1024*1024) // 1MB each
	}

	// 启动一些协程
	for i := 0; i < 50; i++ {
		go func() {
			for j := 0; j < 1000000; j++ {
				_ = j * j
			}
		}()
	}

	time.Sleep(1 * time.Second)
}

// 显示函数
func displayMonitorStatus(analyzer *performance.TaskPerformanceAnalyzer, optimizer *performance.PerformanceOptimizer) {
	fmt.Printf("\n⏰ %s - 系统状态:\n", time.Now().Format("15:04:05"))

	// 显示优化器指标
	optimizerMetrics := optimizer.GetMetrics()
	fmt.Printf("  💾 内存: %.2f MB | 🖥️  CPU: %.2f%% | 🔄 协程: %d\n",
		optimizerMetrics.CurrentMemoryMB,
		optimizerMetrics.CurrentCPUPercent,
		optimizerMetrics.CurrentGoroutines)

	fmt.Printf("  ⚡ 优化次数: %d | 💾 节省内存: %.2f MB | ⏱️  最后优化: %s\n",
		optimizerMetrics.TotalOptimizations,
		optimizerMetrics.MemorySavedMB,
		formatTime(optimizerMetrics.LastOptimizationTime))
}

func displayDemoStatus(analyzer *performance.TaskPerformanceAnalyzer, optimizer *performance.PerformanceOptimizer) {
	fmt.Printf("\n🎭 %s - 演示状态:\n", time.Now().Format("15:04:05"))

	optimizerMetrics := optimizer.GetMetrics()
	fmt.Printf("  📊 当前状态: 内存 %.2f MB, CPU %.2f%%, 协程 %d\n",
		optimizerMetrics.CurrentMemoryMB,
		optimizerMetrics.CurrentCPUPercent,
		optimizerMetrics.CurrentGoroutines)

	allMetrics := analyzer.GetAllMetrics()
	fmt.Printf("  📈 已分析任务: %d 个\n", len(allMetrics))
}

func displayTaskMetrics(taskName string, metrics *performance.TaskMetrics) {
	if metrics == nil {
		fmt.Printf("  ❌ %s: 无数据\n", taskName)
		return
	}

	fmt.Printf("\n  📋 %s:\n", taskName)
	fmt.Printf("    ⏱️  执行时间: %v\n", metrics.Duration)
	fmt.Printf("    🖥️  CPU使用: %.2f%%\n", metrics.CPUUsage.TotalPercent)
	fmt.Printf("    💾 内存使用: %.2f MB (RSS: %d bytes)\n",
		float64(metrics.MemoryUsage.RSS)/1024/1024, metrics.MemoryUsage.RSS)
	fmt.Printf("    🔄 协程数量: %d\n", metrics.RuntimeMetrics.GoroutineCount)
	fmt.Printf("    🗑️  GC次数: %d, 暂停时间: %.2f ms\n",
		metrics.RuntimeMetrics.GCCount, metrics.RuntimeMetrics.GCPauseMs)
}

func displayOptimizerMetrics(metrics performance.OptimizationMetrics) {
	fmt.Printf("  💾 当前内存: %.2f MB\n", metrics.CurrentMemoryMB)
	fmt.Printf("  🖥️  当前CPU: %.2f%%\n", metrics.CurrentCPUPercent)
	fmt.Printf("  🔄 当前协程: %d\n", metrics.CurrentGoroutines)
	fmt.Printf("  ⚡ 总优化次数: %d\n", metrics.TotalOptimizations)
	fmt.Printf("  💾 内存优化: %d 次, 节省: %.2f MB\n",
		metrics.MemoryOptimizations, metrics.MemorySavedMB)
	fmt.Printf("  🗑️  GC优化: %d 次, 减少暂停: %.2f ms\n",
		metrics.GCOptimizations, metrics.GCPauseReduced)
	fmt.Printf("  🖥️  CPU优化: %d 次\n", metrics.CPUOptimizations)
}

func displayAlert(source, alertType, message string) {
	fmt.Printf("🚨 [%s] %s 告警: %s\n", source, alertType, message)
}

func displayOptimizationAlert(source, alertType, message string) {
	fmt.Printf("⚠️  [%s] %s 告警: %s\n", source, alertType, message)
}

func monitorOptimizationEffects(optimizer *performance.PerformanceOptimizer, duration time.Duration) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(duration)

	for {
		select {
		case <-timeout:
			fmt.Println("✅ 监控完成")
			return
		case <-ticker.C:
			metrics := optimizer.GetMetrics()
			fmt.Printf("  📊 内存: %.2f MB, CPU: %.2f%%, 协程: %d\n",
				metrics.CurrentMemoryMB,
				metrics.CurrentCPUPercent,
				metrics.CurrentGoroutines)
		case alert := <-optimizer.GetAlertChannel():
			displayOptimizationAlert("优化器", alert.Type, alert.Message)
		}
	}
}

func saveAnalysisResults(filename string, results map[string]*performance.TaskMetrics) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Printf("❌ 序列化结果失败: %v\n", err)
		return
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Printf("❌ 保存文件失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 分析结果已保存到: %s\n", filename)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "从未"
	}
	return t.Format("15:04:05")
}
