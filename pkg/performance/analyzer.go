/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-03 19:35:00
* @Description: 性能分析器的核心实现
 */

package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

// TaskPerformanceAnalyzer 任务执行性能分析器
// 提供实时性能监控、指标收集、告警和分析功能
type TaskPerformanceAnalyzer struct {
	mu             sync.RWMutex            // 读写锁保护并发访问
	isRunning      bool                    // 分析器运行状态
	isStopping     bool                    // 分析器停止状态
	ctx            context.Context         // 上下文控制
	cancel         context.CancelFunc      // 取消函数
	config         AnalyzerConfig          // 分析器配置
	metrics        map[string]*TaskMetrics // 任务指标存储
	systemBaseline *SystemMetrics          // 系统基线指标
	updateChan     chan TaskMetrics        // 指标更新通道（外部）
	errorChan      chan error              // 错误通道（外部）
	alertChan      chan PerformanceAlert   // 告警通道（外部）

	// 添加goroutine管理
	activeGoroutines sync.WaitGroup // 活跃goroutine计数

	// 内部通道，用于安全发送
	internalUpdateChan chan TaskMetrics      // 内部指标更新通道
	internalErrorChan  chan error            // 内部错误通道
	internalAlertChan  chan PerformanceAlert // 内部告警通道

	// 原子标志防止向已关闭通道发送数据
	channelsClosed int32 // 0: 开放, 1: 已关闭

	// 确保通道只关闭一次
	closeOnce sync.Once // 关闭保护
}

// AnalyzerConfig 性能分析器配置
// 定义分析器的运行参数和告警阈值
type AnalyzerConfig struct {
	SampleInterval  time.Duration   `json:"sample_interval"`  // 采样间隔
	Timeout         time.Duration   `json:"timeout"`          // 操作超时时间
	MaxHistory      int             `json:"max_history"`      // 最大历史记录数
	EnableAlerts    bool            `json:"enable_alerts"`    // 启用告警
	AlertThresholds AlertThresholds `json:"alert_thresholds"` // 告警阈值
	EnableProfiling bool            `json:"enable_profiling"` // 启用性能剖析
}

// AlertThresholds 告警阈值配置
// 定义各种性能指标的告警触发条件
type AlertThresholds struct {
	CPUUsagePercent float64 `json:"cpu_usage_percent"` // CPU使用率阈值
	MemoryUsageMB   float64 `json:"memory_usage_mb"`   // 内存使用阈值(MB)
	ExecutionTimeMs int64   `json:"execution_time_ms"` // 执行时间阈值(毫秒)
	GoroutineCount  int     `json:"goroutine_count"`   // 协程数量阈值
	GCPauseMs       float64 `json:"gc_pause_ms"`       // GC暂停时间阈值(毫秒)
}

// TaskMetrics 任务性能指标
// 包含任务执行过程中的完整性能数据
type TaskMetrics struct {
	TaskID         string                 `json:"task_id"`         // 任务唯一标识
	TaskName       string                 `json:"task_name"`       // 任务名称
	StartTime      time.Time              `json:"start_time"`      // 开始时间
	EndTime        time.Time              `json:"end_time"`        // 结束时间
	Duration       time.Duration          `json:"duration"`        // 执行时长
	CPUUsage       CPUMetrics             `json:"cpu_usage"`       // CPU使用指标
	MemoryUsage    MemoryMetrics          `json:"memory_usage"`    // 内存使用指标
	SystemMetrics  SystemMetrics          `json:"system_metrics"`  // 系统指标
	RuntimeMetrics RuntimeMetrics         `json:"runtime_metrics"` // 运行时指标
	CustomMetrics  map[string]interface{} `json:"custom_metrics"`  // 自定义指标
	Timestamp      time.Time              `json:"timestamp"`       // 时间戳
}

// CPUMetrics CPU使用指标
// 记录CPU在用户态和系统态的使用情况
type CPUMetrics struct {
	UserPercent   float64   `json:"user_percent"`   // 用户态CPU使用率
	SystemPercent float64   `json:"system_percent"` // 系统态CPU使用率
	TotalPercent  float64   `json:"total_percent"`  // 总CPU使用率
	LoadAverage   []float64 `json:"load_average"`   // 负载平均值
}

// MemoryMetrics 内存使用指标
// 记录进程和系统的内存使用情况
type MemoryMetrics struct {
	RSS         uint64  `json:"rss"`          // 常驻内存集
	VMS         uint64  `json:"vms"`          // 虚拟内存大小
	Swap        uint64  `json:"swap"`         // 交换内存
	UsedPercent float64 `json:"used_percent"` // 内存使用百分比
	AvailableMB uint64  `json:"available_mb"` // 可用内存(MB)
}

// SystemMetrics 系统指标
// 记录整个系统的资源使用情况
type SystemMetrics struct {
	TotalCPUPercent    float64 `json:"total_cpu_percent"`    // 系统总CPU使用率
	TotalMemoryUsedMB  uint64  `json:"total_memory_used_mb"` // 系统总内存使用(MB)
	TotalMemoryPercent float64 `json:"total_memory_percent"` // 系统内存使用百分比
	DiskIOReadMB       float64 `json:"disk_io_read_mb"`      // 磁盘读取(MB)
	DiskIOWriteMB      float64 `json:"disk_io_write_mb"`     // 磁盘写入(MB)
	NetworkInMB        float64 `json:"network_in_mb"`        // 网络接收(MB)
	NetworkOutMB       float64 `json:"network_out_mb"`       // 网络发送(MB)
}

// RuntimeMetrics Go运行时指标
// 记录Go程序的运行时状态和垃圾回收信息
type RuntimeMetrics struct {
	GoroutineCount int     `json:"goroutine_count"` // 协程数量
	HeapAllocMB    float64 `json:"heap_alloc_mb"`   // 堆内存分配(MB)
	HeapSysMB      float64 `json:"heap_sys_mb"`     // 堆系统内存(MB)
	GCCount        uint32  `json:"gc_count"`        // GC次数
	GCPauseMs      float64 `json:"gc_pause_ms"`     // GC暂停时间(毫秒)
	NextGCMB       float64 `json:"next_gc_mb"`      // 下次GC阈值(MB)
}

// PerformanceAlert 性能告警
// 当性能指标超过阈值时生成的告警信息
type PerformanceAlert struct {
	ID         string    `json:"id"`          // 告警唯一标识
	Type       string    `json:"type"`        // 告警类型
	Severity   string    `json:"severity"`    // 严重程度
	Message    string    `json:"message"`     // 告警消息
	TaskID     string    `json:"task_id"`     // 相关任务ID
	MetricName string    `json:"metric_name"` // 指标名称
	Value      float64   `json:"value"`       // 当前值
	Threshold  float64   `json:"threshold"`   // 阈值
	Timestamp  time.Time `json:"timestamp"`   // 告警时间
}

// TaskExecutionContext 任务执行上下文
// 跟踪单个任务执行过程中的状态和指标
type TaskExecutionContext struct {
	TaskID     string            // 任务ID
	TaskName   string            // 任务名称
	StartTime  time.Time         // 开始时间
	process    *process.Process  // 进程信息
	initialMem *runtime.MemStats // 初始内存状态
	samples    []TaskMetrics     // 采样数据
	mu         sync.RWMutex      // 读写锁
}

// NewTaskPerformanceAnalyzer 创建新的性能分析器
//
// 参数:
//   - config: 分析器配置
//
// 返回:
//   - *TaskPerformanceAnalyzer: 新创建的性能分析器实例
//
// 该函数会设置默认配置值并初始化所有必要的通道和数据结构
func NewTaskPerformanceAnalyzer(config AnalyzerConfig) *TaskPerformanceAnalyzer {
	analysisContext, cancelFunction := context.WithCancel(context.Background())

	// 设置默认值
	if config.SampleInterval == 0 {
		config.SampleInterval = 1 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxHistory == 0 {
		config.MaxHistory = 1000
	}

	performanceAnalyzer := &TaskPerformanceAnalyzer{
		ctx:                analysisContext,
		cancel:             cancelFunction,
		config:             config,
		metrics:            make(map[string]*TaskMetrics),
		updateChan:         make(chan TaskMetrics, 100),
		errorChan:          make(chan error, 10),
		alertChan:          make(chan PerformanceAlert, 50),
		internalUpdateChan: make(chan TaskMetrics, 100),
		internalErrorChan:  make(chan error, 10),
		internalAlertChan:  make(chan PerformanceAlert, 50),
	}

	return performanceAnalyzer
}

// Start 启动性能分析器
//
// 返回:
//   - error: 启动错误，nil表示成功
//
// 该函数会启动后台goroutine进行指标收集和通道转发
func (tpa *TaskPerformanceAnalyzer) Start() error {
	tpa.mu.Lock()
	defer tpa.mu.Unlock()

	if tpa.isRunning {
		return fmt.Errorf("性能分析器已在运行中")
	}

	tpa.isRunning = true
	tpa.isStopping = false

	// 启动通道转发goroutine
	tpa.activeGoroutines.Add(1)
	go func() {
		defer tpa.activeGoroutines.Done()
		tpa.channelForwarder()
	}()

	// 收集系统基线指标，使用WaitGroup跟踪
	tpa.activeGoroutines.Add(1)
	go func() {
		defer tpa.activeGoroutines.Done()
		tpa.collectSystemBaseline()
	}()

	return nil
}

// Stop 停止性能分析器（简化版本，避免竞态条件）
func (tpa *TaskPerformanceAnalyzer) Stop() error {
	// 首先设置原子标志，防止向通道发送新数据
	atomic.StoreInt32(&tpa.channelsClosed, 1)

	tpa.mu.Lock()

	if !tpa.isRunning {
		tpa.mu.Unlock()
		return fmt.Errorf("性能分析器未在运行")
	}

	// 标记为正在停止，防止新的goroutine启动
	tpa.isStopping = true
	tpa.isRunning = false

	// 取消上下文，通知所有goroutine停止
	tpa.cancel()

	tpa.mu.Unlock()

	// 等待所有活跃的goroutine退出
	// 使用超时防止无限等待
	done := make(chan struct{})
	go func() {
		tpa.activeGoroutines.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 所有goroutine已退出
	case <-time.After(5 * time.Second):
		// 超时，强制继续
	}

	// 不显式关闭通道，让垃圾回收器处理
	// 这避免了竞态条件，因为通道仍然可以接收数据，只是没有人会读取

	return nil
}

// StartTaskAnalysis 开始任务性能分析
func (tpa *TaskPerformanceAnalyzer) StartTaskAnalysis(taskID, taskName string) (*TaskExecutionContext, error) {
	tpa.mu.RLock()
	isRunning := tpa.isRunning
	isStopping := tpa.isStopping
	tpa.mu.RUnlock()

	if !isRunning || isStopping {
		return nil, fmt.Errorf("性能分析器未运行或正在停止")
	}

	// 获取当前进程
	pid := int32(runtime.GOMAXPROCS(0))
	proc, err := process.NewProcess(pid)
	if err != nil {
		// 如果无法获取进程信息，继续但记录错误
		tpa.mu.RLock()
		isRunning = tpa.isRunning
		tpa.mu.RUnlock()

		if isRunning {
			tpa.safeSendError(fmt.Errorf("获取进程信息失败: %w", err))
		}
	}

	// 获取初始内存统计
	var initialMem runtime.MemStats
	runtime.ReadMemStats(&initialMem)

	ctx := &TaskExecutionContext{
		TaskID:     taskID,
		TaskName:   taskName,
		StartTime:  time.Now(),
		process:    proc,
		initialMem: &initialMem,
		samples:    make([]TaskMetrics, 0),
	}

	// 启动采样协程，并使用WaitGroup跟踪
	tpa.activeGoroutines.Add(1)
	go func() {
		defer tpa.activeGoroutines.Done()
		tpa.sampleTaskMetrics(ctx)
	}()

	return ctx, nil
}

// FinishTaskAnalysis 完成任务性能分析
func (tpa *TaskPerformanceAnalyzer) FinishTaskAnalysis(ctx *TaskExecutionContext) *TaskMetrics {
	if ctx == nil {
		return nil
	}

	endTime := time.Now()
	duration := endTime.Sub(ctx.StartTime)

	// 收集最终指标
	finalMetrics := tpa.collectCurrentMetrics(ctx.TaskID, ctx.TaskName)
	finalMetrics.StartTime = ctx.StartTime
	finalMetrics.EndTime = endTime
	finalMetrics.Duration = duration

	// 存储指标
	tpa.mu.Lock()
	tpa.metrics[ctx.TaskID] = &finalMetrics
	tpa.mu.Unlock()

	// 发送更新通知（检查分析器是否仍在运行）
	tpa.mu.RLock()
	isRunning := tpa.isRunning
	tpa.mu.RUnlock()

	if isRunning {
		tpa.safeSendUpdate(finalMetrics)
	}

	// 检查告警
	if tpa.config.EnableAlerts && isRunning {
		tpa.checkAlerts(finalMetrics)
	}

	return &finalMetrics
}

// sampleTaskMetrics 采样任务指标
func (tpa *TaskPerformanceAnalyzer) sampleTaskMetrics(ctx *TaskExecutionContext) {
	ticker := time.NewTicker(tpa.config.SampleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-tpa.ctx.Done():
			return
		case <-ticker.C:
			// 使用超时上下文防止死锁
			sampleCtx, cancel := context.WithTimeout(tpa.ctx, tpa.config.Timeout)
			metrics := tpa.collectCurrentMetricsWithContext(sampleCtx, ctx.TaskID, ctx.TaskName)
			cancel()

			if metrics.TaskID != "" {
				ctx.mu.Lock()
				ctx.samples = append(ctx.samples, metrics)
				// 限制样本数量
				if len(ctx.samples) > tpa.config.MaxHistory {
					ctx.samples = ctx.samples[len(ctx.samples)-tpa.config.MaxHistory:]
				}
				ctx.mu.Unlock()
			}
		}
	}
}

// collectCurrentMetrics 收集当前指标
func (tpa *TaskPerformanceAnalyzer) collectCurrentMetrics(taskID, taskName string) TaskMetrics {
	ctx, cancel := context.WithTimeout(context.Background(), tpa.config.Timeout)
	defer cancel()
	return tpa.collectCurrentMetricsWithContext(ctx, taskID, taskName)
}

// collectCurrentMetricsWithContext 使用上下文控制收集当前指标（并发安全版本）
func (tpa *TaskPerformanceAnalyzer) collectCurrentMetricsWithContext(ctx context.Context, taskID, taskName string) TaskMetrics {
	// 创建本地指标结构体，避免并发访问问题
	metrics := TaskMetrics{
		TaskID:        taskID,
		TaskName:      taskName,
		Timestamp:     time.Now(),
		CustomMetrics: make(map[string]interface{}),
	}

	// 使用独立的结构体来收集各种指标，避免并发访问问题
	type metricsContainer struct {
		mu       sync.Mutex
		cpu      CPUMetrics
		memory   MemoryMetrics
		system   SystemMetrics
		runtime  RuntimeMetrics
		cpuReady bool
		memReady bool
		sysReady bool
		rtReady  bool
	}

	container := &metricsContainer{}
	var wg sync.WaitGroup

	// 并发收集各种指标，使用超时防止死锁

	// 收集CPU指标
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 检查是否正在停止
		tpa.mu.RLock()
		isStopping := tpa.isStopping
		tpa.mu.RUnlock()

		if isStopping {
			return
		}

		result, err := tpa.collectCPUMetrics(ctx)
		if err != nil {
			// 检查分析器是否仍在运行，避免向已关闭的通道发送数据
			tpa.mu.RLock()
			isRunning := tpa.isRunning
			tpa.mu.RUnlock()

			if isRunning {
				tpa.safeSendError(fmt.Errorf("收集CPU指标失败: %w", err))
			}
			return
		}
		container.mu.Lock()
		container.cpu = result
		container.cpuReady = true
		container.mu.Unlock()
	}()

	// 收集内存指标
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 检查是否正在停止
		tpa.mu.RLock()
		isStopping := tpa.isStopping
		tpa.mu.RUnlock()

		if isStopping {
			return
		}

		result, err := tpa.collectMemoryMetrics(ctx)
		if err != nil {
			// 检查分析器是否仍在运行，避免向已关闭的通道发送数据
			tpa.mu.RLock()
			isRunning := tpa.isRunning
			tpa.mu.RUnlock()

			if isRunning {
				tpa.safeSendError(fmt.Errorf("收集内存指标失败: %w", err))
			}
			return
		}
		container.mu.Lock()
		container.memory = result
		container.memReady = true
		container.mu.Unlock()
	}()

	// 收集系统指标
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 检查是否正在停止
		tpa.mu.RLock()
		isStopping := tpa.isStopping
		tpa.mu.RUnlock()

		if isStopping {
			return
		}

		result, err := tpa.collectSystemMetrics(ctx)
		if err != nil {
			// 检查分析器是否仍在运行，避免向已关闭的通道发送数据
			tpa.mu.RLock()
			isRunning := tpa.isRunning
			tpa.mu.RUnlock()

			if isRunning {
				tpa.safeSendError(fmt.Errorf("收集系统指标失败: %w", err))
			}
			return
		}
		container.mu.Lock()
		container.system = result
		container.sysReady = true
		container.mu.Unlock()
	}()

	// 收集运行时指标（现在也用锁保护）
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 检查是否正在停止
		tpa.mu.RLock()
		isStopping := tpa.isStopping
		tpa.mu.RUnlock()

		if isStopping {
			return
		}

		result := tpa.collectRuntimeMetrics()
		container.mu.Lock()
		container.runtime = result
		container.rtReady = true
		container.mu.Unlock()
	}()

	// 等待所有收集任务完成或超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 所有任务完成，安全地复制数据到结果中
		container.mu.Lock()
		if container.cpuReady {
			metrics.CPUUsage = container.cpu
		}
		if container.memReady {
			metrics.MemoryUsage = container.memory
		}
		if container.sysReady {
			metrics.SystemMetrics = container.system
		}
		if container.rtReady {
			metrics.RuntimeMetrics = container.runtime
		}
		container.mu.Unlock()

	case <-ctx.Done():
		// 超时，返回部分数据
		// 检查分析器是否仍在运行，避免向已关闭的通道发送数据
		tpa.mu.RLock()
		isRunning := tpa.isRunning
		tpa.mu.RUnlock()

		if isRunning {
			tpa.safeSendError(fmt.Errorf("指标收集超时"))
		}

		// 即使超时，也尝试获取已收集的数据
		container.mu.Lock()
		if container.cpuReady {
			metrics.CPUUsage = container.cpu
		}
		if container.memReady {
			metrics.MemoryUsage = container.memory
		}
		if container.sysReady {
			metrics.SystemMetrics = container.system
		}
		if container.rtReady {
			metrics.RuntimeMetrics = container.runtime
		}
		container.mu.Unlock()
	}

	return metrics
}

// collectCPUMetrics 收集CPU指标
func (tpa *TaskPerformanceAnalyzer) collectCPUMetrics(ctx context.Context) (CPUMetrics, error) {
	var metrics CPUMetrics

	// 获取CPU使用率
	cpuPercents, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		return metrics, err
	}

	if len(cpuPercents) > 0 {
		metrics.TotalPercent = cpuPercents[0]
	}

	// 获取负载平均值（仅Linux/Unix）
	if runtime.GOOS != "windows" {
		// 这里可以添加负载平均值的获取逻辑
		// 由于gopsutil的load包在某些环境下可能有问题，我们暂时跳过
	}

	return metrics, nil
}

// collectMemoryMetrics 收集内存指标
func (tpa *TaskPerformanceAnalyzer) collectMemoryMetrics(ctx context.Context) (MemoryMetrics, error) {
	var metrics MemoryMetrics

	// 获取系统内存信息
	vmem, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return metrics, err
	}

	metrics.UsedPercent = vmem.UsedPercent
	metrics.AvailableMB = vmem.Available / 1024 / 1024

	// 获取当前进程内存信息
	pid := int32(runtime.GOMAXPROCS(0))
	proc, err := process.NewProcessWithContext(ctx, pid)
	if err == nil {
		if memInfo, err := proc.MemoryInfoWithContext(ctx); err == nil {
			metrics.RSS = memInfo.RSS
			metrics.VMS = memInfo.VMS
		}
	}

	return metrics, nil
}

// collectSystemMetrics 收集系统指标
func (tpa *TaskPerformanceAnalyzer) collectSystemMetrics(ctx context.Context) (SystemMetrics, error) {
	var metrics SystemMetrics

	// 获取系统CPU使用率
	cpuPercents, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err == nil && len(cpuPercents) > 0 {
		metrics.TotalCPUPercent = cpuPercents[0]
	}

	// 获取系统内存使用
	vmem, err := mem.VirtualMemoryWithContext(ctx)
	if err == nil {
		metrics.TotalMemoryUsedMB = vmem.Used / 1024 / 1024
		metrics.TotalMemoryPercent = vmem.UsedPercent
	}

	return metrics, nil
}

// collectRuntimeMetrics 收集Go运行时指标
func (tpa *TaskPerformanceAnalyzer) collectRuntimeMetrics() RuntimeMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return RuntimeMetrics{
		GoroutineCount: runtime.NumGoroutine(),
		HeapAllocMB:    float64(memStats.HeapAlloc) / 1024 / 1024,
		HeapSysMB:      float64(memStats.HeapSys) / 1024 / 1024,
		GCCount:        memStats.NumGC,
		GCPauseMs:      float64(memStats.PauseNs[(memStats.NumGC+255)%256]) / 1e6,
		NextGCMB:       float64(memStats.NextGC) / 1024 / 1024,
	}
}

// collectSystemBaseline 收集系统基线指标
func (tpa *TaskPerformanceAnalyzer) collectSystemBaseline() {
	ctx, cancel := context.WithTimeout(tpa.ctx, tpa.config.Timeout)
	defer cancel()

	baseline, err := tpa.collectSystemMetrics(ctx)
	if err != nil {
		tpa.safeSendError(fmt.Errorf("收集系统基线失败: %w", err))
		return
	}

	tpa.mu.Lock()
	tpa.systemBaseline = &baseline
	tpa.mu.Unlock()
}

// checkAlerts 检查告警条件
func (tpa *TaskPerformanceAnalyzer) checkAlerts(metrics TaskMetrics) {
	alerts := make([]PerformanceAlert, 0)

	// 检查CPU使用率告警
	if metrics.CPUUsage.TotalPercent > tpa.config.AlertThresholds.CPUUsagePercent {
		alert := PerformanceAlert{
			ID:         fmt.Sprintf("cpu_%s_%d", metrics.TaskID, time.Now().Unix()),
			Type:       "cpu_usage",
			Severity:   "warning",
			Message:    fmt.Sprintf("任务 %s CPU使用率过高: %.2f%%", metrics.TaskName, metrics.CPUUsage.TotalPercent),
			TaskID:     metrics.TaskID,
			MetricName: "cpu_usage_percent",
			Value:      metrics.CPUUsage.TotalPercent,
			Threshold:  tpa.config.AlertThresholds.CPUUsagePercent,
			Timestamp:  time.Now(),
		}
		alerts = append(alerts, alert)
	}

	// 检查内存使用告警
	memUsageMB := float64(metrics.MemoryUsage.RSS) / 1024 / 1024
	if memUsageMB > tpa.config.AlertThresholds.MemoryUsageMB {
		alert := PerformanceAlert{
			ID:         fmt.Sprintf("memory_%s_%d", metrics.TaskID, time.Now().Unix()),
			Type:       "memory_usage",
			Severity:   "warning",
			Message:    fmt.Sprintf("任务 %s 内存使用过高: %.2f MB", metrics.TaskName, memUsageMB),
			TaskID:     metrics.TaskID,
			MetricName: "memory_usage_mb",
			Value:      memUsageMB,
			Threshold:  tpa.config.AlertThresholds.MemoryUsageMB,
			Timestamp:  time.Now(),
		}
		alerts = append(alerts, alert)
	}

	// 检查执行时间告警
	if metrics.Duration.Milliseconds() > tpa.config.AlertThresholds.ExecutionTimeMs {
		alert := PerformanceAlert{
			ID:         fmt.Sprintf("duration_%s_%d", metrics.TaskID, time.Now().Unix()),
			Type:       "execution_time",
			Severity:   "info",
			Message:    fmt.Sprintf("任务 %s 执行时间较长: %v", metrics.TaskName, metrics.Duration),
			TaskID:     metrics.TaskID,
			MetricName: "execution_time_ms",
			Value:      float64(metrics.Duration.Milliseconds()),
			Threshold:  float64(tpa.config.AlertThresholds.ExecutionTimeMs),
			Timestamp:  time.Now(),
		}
		alerts = append(alerts, alert)
	}

	// 检查协程数量告警
	if metrics.RuntimeMetrics.GoroutineCount > tpa.config.AlertThresholds.GoroutineCount {
		alert := PerformanceAlert{
			ID:         fmt.Sprintf("goroutine_%s_%d", metrics.TaskID, time.Now().Unix()),
			Type:       "goroutine_count",
			Severity:   "warning",
			Message:    fmt.Sprintf("任务 %s 协程数量过多: %d", metrics.TaskName, metrics.RuntimeMetrics.GoroutineCount),
			TaskID:     metrics.TaskID,
			MetricName: "goroutine_count",
			Value:      float64(metrics.RuntimeMetrics.GoroutineCount),
			Threshold:  float64(tpa.config.AlertThresholds.GoroutineCount),
			Timestamp:  time.Now(),
		}
		alerts = append(alerts, alert)
	}

	// 检查GC暂停时间告警
	if metrics.RuntimeMetrics.GCPauseMs > tpa.config.AlertThresholds.GCPauseMs {
		alert := PerformanceAlert{
			ID:         fmt.Sprintf("gc_pause_%s_%d", metrics.TaskID, time.Now().Unix()),
			Type:       "gc_pause",
			Severity:   "info",
			Message:    fmt.Sprintf("任务 %s GC暂停时间较长: %.2f ms", metrics.TaskName, metrics.RuntimeMetrics.GCPauseMs),
			TaskID:     metrics.TaskID,
			MetricName: "gc_pause_ms",
			Value:      metrics.RuntimeMetrics.GCPauseMs,
			Threshold:  tpa.config.AlertThresholds.GCPauseMs,
			Timestamp:  time.Now(),
		}
		alerts = append(alerts, alert)
	}

	// 发送告警
	for _, alert := range alerts {
		tpa.safeSendAlert(alert)
	}
}

// GetUpdateChannel 获取更新通道
func (tpa *TaskPerformanceAnalyzer) GetUpdateChannel() <-chan TaskMetrics {
	return tpa.updateChan
}

// GetErrorChannel 获取错误通道
func (tpa *TaskPerformanceAnalyzer) GetErrorChannel() <-chan error {
	return tpa.errorChan
}

// GetAlertChannel 获取告警通道
func (tpa *TaskPerformanceAnalyzer) GetAlertChannel() <-chan PerformanceAlert {
	return tpa.alertChan
}

// GetTaskMetrics 获取任务指标
func (tpa *TaskPerformanceAnalyzer) GetTaskMetrics(taskID string) (*TaskMetrics, error) {
	tpa.mu.RLock()
	defer tpa.mu.RUnlock()

	metrics, exists := tpa.metrics[taskID]
	if !exists {
		return nil, fmt.Errorf("任务指标不存在: %s", taskID)
	}

	// 返回副本
	metricsCopy := *metrics
	return &metricsCopy, nil
}

// GetAllMetrics 获取所有任务指标
func (tpa *TaskPerformanceAnalyzer) GetAllMetrics() map[string]*TaskMetrics {
	tpa.mu.RLock()
	defer tpa.mu.RUnlock()

	// 返回副本
	result := make(map[string]*TaskMetrics)
	for k, v := range tpa.metrics {
		metricsCopy := *v
		result[k] = &metricsCopy
	}

	return result
}

// IsRunning 检查是否正在运行
func (tpa *TaskPerformanceAnalyzer) IsRunning() bool {
	tpa.mu.RLock()
	defer tpa.mu.RUnlock()
	return tpa.isRunning
}

// safeSendError 安全发送错误到错误通道
func (tpa *TaskPerformanceAnalyzer) safeSendError(err error) {
	defer func() {
		if r := recover(); r != nil {
			// 捕获向已关闭通道发送数据的panic，静默忽略
		}
	}()

	if atomic.LoadInt32(&tpa.channelsClosed) == 1 {
		return // 通道已关闭，不发送
	}

	select {
	case tpa.internalErrorChan <- err:
	default:
		// 通道满，静默丢弃
	}
}

// safeSendUpdate 安全发送更新到更新通道
func (tpa *TaskPerformanceAnalyzer) safeSendUpdate(metrics TaskMetrics) {
	defer func() {
		if r := recover(); r != nil {
			// 捕获向已关闭通道发送数据的panic，静默忽略
		}
	}()

	if atomic.LoadInt32(&tpa.channelsClosed) == 1 {
		return // 通道已关闭，不发送
	}

	select {
	case tpa.internalUpdateChan <- metrics:
	default:
		// 通道满，静默丢弃
	}
}

// safeSendAlert 安全发送告警到告警通道
func (tpa *TaskPerformanceAnalyzer) safeSendAlert(alert PerformanceAlert) {
	defer func() {
		if r := recover(); r != nil {
			// 捕获向已关闭通道发送数据的panic，静默忽略
		}
	}()

	if atomic.LoadInt32(&tpa.channelsClosed) == 1 {
		return // 通道已关闭，不发送
	}

	select {
	case tpa.internalAlertChan <- alert:
	default:
		// 通道满，静默丢弃
	}
}

// channelForwarder 通道转发器，安全地将内部通道数据转发到外部通道
func (tpa *TaskPerformanceAnalyzer) channelForwarder() {
	for {
		select {
		case <-tpa.ctx.Done():
			return
		case update := <-tpa.internalUpdateChan:
			select {
			case tpa.updateChan <- update:
			case <-tpa.ctx.Done():
				return
			default:
				// 外部通道满，丢弃数据
			}
		case err := <-tpa.internalErrorChan:
			select {
			case tpa.errorChan <- err:
			case <-tpa.ctx.Done():
				return
			default:
				// 外部通道满，丢弃数据
			}
		case alert := <-tpa.internalAlertChan:
			select {
			case tpa.alertChan <- alert:
			case <-tpa.ctx.Done():
				return
			default:
				// 外部通道满，丢弃数据
			}
		}
	}
}
