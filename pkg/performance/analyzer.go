/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 性能分析器的核心实现
 */

package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

// TaskPerformanceAnalyzer 任务执行性能分析器
type TaskPerformanceAnalyzer struct {
	mu             sync.RWMutex
	isRunning      bool
	ctx            context.Context
	cancel         context.CancelFunc
	config         AnalyzerConfig
	metrics        map[string]*TaskMetrics
	systemBaseline *SystemMetrics
	updateChan     chan TaskMetrics
	errorChan      chan error
	alertChan      chan PerformanceAlert
}

// AnalyzerConfig 性能分析器配置
type AnalyzerConfig struct {
	SampleInterval  time.Duration   `json:"sample_interval"`  // 采样间隔
	Timeout         time.Duration   `json:"timeout"`          // 操作超时时间
	MaxHistory      int             `json:"max_history"`      // 最大历史记录数
	EnableAlerts    bool            `json:"enable_alerts"`    // 启用告警
	AlertThresholds AlertThresholds `json:"alert_thresholds"` // 告警阈值
	EnableProfiling bool            `json:"enable_profiling"` // 启用性能剖析
}

// AlertThresholds 告警阈值配置
type AlertThresholds struct {
	CPUUsagePercent float64 `json:"cpu_usage_percent"` // CPU使用率阈值
	MemoryUsageMB   float64 `json:"memory_usage_mb"`   // 内存使用阈值(MB)
	ExecutionTimeMs int64   `json:"execution_time_ms"` // 执行时间阈值(毫秒)
	GoroutineCount  int     `json:"goroutine_count"`   // 协程数量阈值
	GCPauseMs       float64 `json:"gc_pause_ms"`       // GC暂停时间阈值(毫秒)
}

// TaskMetrics 任务性能指标
type TaskMetrics struct {
	TaskID         string                 `json:"task_id"`
	TaskName       string                 `json:"task_name"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	Duration       time.Duration          `json:"duration"`
	CPUUsage       CPUMetrics             `json:"cpu_usage"`
	MemoryUsage    MemoryMetrics          `json:"memory_usage"`
	SystemMetrics  SystemMetrics          `json:"system_metrics"`
	RuntimeMetrics RuntimeMetrics         `json:"runtime_metrics"`
	CustomMetrics  map[string]interface{} `json:"custom_metrics"`
	Timestamp      time.Time              `json:"timestamp"`
}

// CPUMetrics CPU使用指标
type CPUMetrics struct {
	UserPercent   float64   `json:"user_percent"`   // 用户态CPU使用率
	SystemPercent float64   `json:"system_percent"` // 系统态CPU使用率
	TotalPercent  float64   `json:"total_percent"`  // 总CPU使用率
	LoadAverage   []float64 `json:"load_average"`   // 负载平均值
}

// MemoryMetrics 内存使用指标
type MemoryMetrics struct {
	RSS         uint64  `json:"rss"`          // 常驻内存集
	VMS         uint64  `json:"vms"`          // 虚拟内存大小
	Swap        uint64  `json:"swap"`         // 交换内存
	UsedPercent float64 `json:"used_percent"` // 内存使用百分比
	AvailableMB uint64  `json:"available_mb"` // 可用内存(MB)
}

// SystemMetrics 系统指标
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
type RuntimeMetrics struct {
	GoroutineCount int     `json:"goroutine_count"` // 协程数量
	HeapAllocMB    float64 `json:"heap_alloc_mb"`   // 堆内存分配(MB)
	HeapSysMB      float64 `json:"heap_sys_mb"`     // 堆系统内存(MB)
	GCCount        uint32  `json:"gc_count"`        // GC次数
	GCPauseMs      float64 `json:"gc_pause_ms"`     // GC暂停时间(毫秒)
	NextGCMB       float64 `json:"next_gc_mb"`      // 下次GC阈值(MB)
}

// PerformanceAlert 性能告警
type PerformanceAlert struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Severity   string    `json:"severity"`
	Message    string    `json:"message"`
	TaskID     string    `json:"task_id"`
	MetricName string    `json:"metric_name"`
	Value      float64   `json:"value"`
	Threshold  float64   `json:"threshold"`
	Timestamp  time.Time `json:"timestamp"`
}

// TaskExecutionContext 任务执行上下文
type TaskExecutionContext struct {
	TaskID     string
	TaskName   string
	StartTime  time.Time
	process    *process.Process
	initialMem *runtime.MemStats
	samples    []TaskMetrics
	mu         sync.RWMutex
}

// NewTaskPerformanceAnalyzer 创建新的性能分析器
func NewTaskPerformanceAnalyzer(config AnalyzerConfig) *TaskPerformanceAnalyzer {
	ctx, cancel := context.WithCancel(context.Background())

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

	analyzer := &TaskPerformanceAnalyzer{
		ctx:        ctx,
		cancel:     cancel,
		config:     config,
		metrics:    make(map[string]*TaskMetrics),
		updateChan: make(chan TaskMetrics, 100),
		errorChan:  make(chan error, 10),
		alertChan:  make(chan PerformanceAlert, 50),
	}

	return analyzer
}

// Start 启动性能分析器
func (tpa *TaskPerformanceAnalyzer) Start() error {
	tpa.mu.Lock()
	defer tpa.mu.Unlock()

	if tpa.isRunning {
		return fmt.Errorf("性能分析器已在运行中")
	}

	tpa.isRunning = true

	// 收集系统基线指标
	go tpa.collectSystemBaseline()

	return nil
}

// Stop 停止性能分析器
func (tpa *TaskPerformanceAnalyzer) Stop() error {
	tpa.mu.Lock()
	defer tpa.mu.Unlock()

	if !tpa.isRunning {
		return fmt.Errorf("性能分析器未在运行")
	}

	tpa.isRunning = false
	tpa.cancel()

	// 等待一小段时间让协程退出
	time.Sleep(100 * time.Millisecond)

	// 安全关闭通道
	close(tpa.updateChan)
	close(tpa.errorChan)
	close(tpa.alertChan)

	return nil
}

// StartTaskAnalysis 开始任务性能分析
func (tpa *TaskPerformanceAnalyzer) StartTaskAnalysis(taskID, taskName string) (*TaskExecutionContext, error) {
	if !tpa.isRunning {
		return nil, fmt.Errorf("性能分析器未运行")
	}

	// 获取当前进程
	pid := int32(runtime.GOMAXPROCS(0))
	proc, err := process.NewProcess(pid)
	if err != nil {
		// 如果无法获取进程信息，继续但记录错误
		select {
		case tpa.errorChan <- fmt.Errorf("获取进程信息失败: %w", err):
		default:
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

	// 启动采样协程
	go tpa.sampleTaskMetrics(ctx)

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
		select {
		case tpa.updateChan <- finalMetrics:
		default:
			// 通道满了，丢弃数据
		}
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

// collectCurrentMetricsWithContext 带上下文收集当前指标
func (tpa *TaskPerformanceAnalyzer) collectCurrentMetricsWithContext(ctx context.Context, taskID, taskName string) TaskMetrics {
	metrics := TaskMetrics{
		TaskID:        taskID,
		TaskName:      taskName,
		Timestamp:     time.Now(),
		CustomMetrics: make(map[string]interface{}),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// 并发收集各种指标，使用超时防止死锁

	// 收集CPU指标
	wg.Add(1)
	go func() {
		defer wg.Done()
		cpuMetrics, err := tpa.collectCPUMetrics(ctx)
		if err != nil {
			// 检查分析器是否仍在运行，避免向已关闭的通道发送数据
			tpa.mu.RLock()
			isRunning := tpa.isRunning
			tpa.mu.RUnlock()

			if isRunning {
				select {
				case tpa.errorChan <- fmt.Errorf("收集CPU指标失败: %w", err):
				default:
				}
			}
			return
		}
		mu.Lock()
		metrics.CPUUsage = cpuMetrics
		mu.Unlock()
	}()

	// 收集内存指标
	wg.Add(1)
	go func() {
		defer wg.Done()
		memMetrics, err := tpa.collectMemoryMetrics(ctx)
		if err != nil {
			// 检查分析器是否仍在运行，避免向已关闭的通道发送数据
			tpa.mu.RLock()
			isRunning := tpa.isRunning
			tpa.mu.RUnlock()

			if isRunning {
				select {
				case tpa.errorChan <- fmt.Errorf("收集内存指标失败: %w", err):
				default:
				}
			}
			return
		}
		mu.Lock()
		metrics.MemoryUsage = memMetrics
		mu.Unlock()
	}()

	// 收集系统指标
	wg.Add(1)
	go func() {
		defer wg.Done()
		sysMetrics, err := tpa.collectSystemMetrics(ctx)
		if err != nil {
			// 检查分析器是否仍在运行，避免向已关闭的通道发送数据
			tpa.mu.RLock()
			isRunning := tpa.isRunning
			tpa.mu.RUnlock()

			if isRunning {
				select {
				case tpa.errorChan <- fmt.Errorf("收集系统指标失败: %w", err):
				default:
				}
			}
			return
		}
		mu.Lock()
		metrics.SystemMetrics = sysMetrics
		mu.Unlock()
	}()

	// 收集运行时指标
	wg.Add(1)
	go func() {
		defer wg.Done()
		runtimeMetrics := tpa.collectRuntimeMetrics()
		mu.Lock()
		metrics.RuntimeMetrics = runtimeMetrics
		mu.Unlock()
	}()

	// 等待所有收集任务完成或超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 所有任务完成
	case <-ctx.Done():
		// 超时，返回部分数据
		// 检查分析器是否仍在运行，避免向已关闭的通道发送数据
		tpa.mu.RLock()
		isRunning := tpa.isRunning
		tpa.mu.RUnlock()

		if isRunning {
			select {
			case tpa.errorChan <- fmt.Errorf("指标收集超时"):
			default:
			}
		}
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
		select {
		case tpa.errorChan <- fmt.Errorf("收集系统基线失败: %w", err):
		default:
		}
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
		select {
		case tpa.alertChan <- alert:
		default:
			// 告警通道满了，丢弃告警
		}
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
