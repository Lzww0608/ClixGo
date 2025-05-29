/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 性能优化器的核心实现
 */

package performance

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// PerformanceOptimizer 性能优化器
type PerformanceOptimizer struct {
	mu                sync.RWMutex
	isRunning         bool
	ctx               context.Context
	cancel            context.CancelFunc
	config            OptimizerConfig
	lastOptimization  time.Time
	optimizationCount int
	metrics           OptimizationMetrics
	alertChan         chan OptimizationAlert
	errorChan         chan error
}

// OptimizerConfig 优化器配置
type OptimizerConfig struct {
	OptimizationInterval time.Duration `json:"optimization_interval"` // 优化间隔
	Timeout              time.Duration `json:"timeout"`               // 操作超时时间
	EnableAutoGC         bool          `json:"enable_auto_gc"`        // 启用自动GC
	EnableMemoryLimit    bool          `json:"enable_memory_limit"`   // 启用内存限制
	EnableCPUThrottling  bool          `json:"enable_cpu_throttling"` // 启用CPU节流

	// 阈值配置
	MemoryThresholdMB   float64 `json:"memory_threshold_mb"`   // 内存阈值(MB)
	CPUThresholdPercent float64 `json:"cpu_threshold_percent"` // CPU阈值(%)
	GCThresholdMB       float64 `json:"gc_threshold_mb"`       // GC阈值(MB)
	GoroutineThreshold  int     `json:"goroutine_threshold"`   // 协程数量阈值

	// 优化策略
	GCTargetPercent int     `json:"gc_target_percent"` // GC目标百分比
	MaxGoroutines   int     `json:"max_goroutines"`    // 最大协程数
	MemoryLimitMB   float64 `json:"memory_limit_mb"`   // 内存限制(MB)
	CPUQuotaPercent float64 `json:"cpu_quota_percent"` // CPU配额百分比
}

// OptimizationMetrics 优化指标
type OptimizationMetrics struct {
	TotalOptimizations   int       `json:"total_optimizations"`    // 总优化次数
	GCOptimizations      int       `json:"gc_optimizations"`       // GC优化次数
	MemoryOptimizations  int       `json:"memory_optimizations"`   // 内存优化次数
	CPUOptimizations     int       `json:"cpu_optimizations"`      // CPU优化次数
	LastOptimizationTime time.Time `json:"last_optimization_time"` // 最后优化时间

	// 优化效果
	MemorySavedMB   float64 `json:"memory_saved_mb"`   // 节省的内存(MB)
	CPUSavedPercent float64 `json:"cpu_saved_percent"` // 节省的CPU(%)
	GCPauseReduced  float64 `json:"gc_pause_reduced"`  // 减少的GC暂停时间(ms)

	// 当前状态
	CurrentMemoryMB   float64 `json:"current_memory_mb"`   // 当前内存使用(MB)
	CurrentCPUPercent float64 `json:"current_cpu_percent"` // 当前CPU使用(%)
	CurrentGoroutines int     `json:"current_goroutines"`  // 当前协程数
}

// OptimizationAlert 优化告警
type OptimizationAlert struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Action    string    `json:"action"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Timestamp time.Time `json:"timestamp"`
}

// OptimizationAction 优化动作
type OptimizationAction struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Timestamp   time.Time              `json:"timestamp"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
}

// NewPerformanceOptimizer 创建新的性能优化器
func NewPerformanceOptimizer(config OptimizerConfig) *PerformanceOptimizer {
	ctx, cancel := context.WithCancel(context.Background())

	// 设置默认值
	if config.OptimizationInterval == 0 {
		config.OptimizationInterval = 30 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	if config.MemoryThresholdMB == 0 {
		config.MemoryThresholdMB = 100.0
	}
	if config.CPUThresholdPercent == 0 {
		config.CPUThresholdPercent = 80.0
	}
	if config.GCThresholdMB == 0 {
		config.GCThresholdMB = 50.0
	}
	if config.GoroutineThreshold == 0 {
		config.GoroutineThreshold = 1000
	}
	if config.GCTargetPercent == 0 {
		config.GCTargetPercent = 100
	}
	if config.MaxGoroutines == 0 {
		config.MaxGoroutines = 10000
	}

	optimizer := &PerformanceOptimizer{
		ctx:       ctx,
		cancel:    cancel,
		config:    config,
		alertChan: make(chan OptimizationAlert, 50),
		errorChan: make(chan error, 10),
	}

	return optimizer
}

// Start 启动性能优化器
func (po *PerformanceOptimizer) Start() error {
	po.mu.Lock()
	defer po.mu.Unlock()

	if po.isRunning {
		return fmt.Errorf("性能优化器已在运行中")
	}

	po.isRunning = true

	// 启动优化循环
	go po.optimizationLoop()

	return nil
}

// Stop 停止性能优化器
func (po *PerformanceOptimizer) Stop() error {
	po.mu.Lock()
	defer po.mu.Unlock()

	if !po.isRunning {
		return fmt.Errorf("性能优化器未在运行")
	}

	po.isRunning = false
	po.cancel()

	// 等待一小段时间让协程退出
	time.Sleep(100 * time.Millisecond)

	// 安全关闭通道
	close(po.alertChan)
	close(po.errorChan)

	return nil
}

// optimizationLoop 优化循环
func (po *PerformanceOptimizer) optimizationLoop() {
	ticker := time.NewTicker(po.config.OptimizationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-po.ctx.Done():
			return
		case <-ticker.C:
			// 使用超时上下文防止死锁
			optimizeCtx, cancel := context.WithTimeout(po.ctx, po.config.Timeout)
			po.performOptimization(optimizeCtx)
			cancel()
		}
	}
}

// performOptimization 执行优化
func (po *PerformanceOptimizer) performOptimization(ctx context.Context) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	actions := make([]OptimizationAction, 0)

	// 收集当前指标
	currentMetrics, err := po.collectCurrentMetrics(ctx)
	if err != nil {
		// 检查优化器是否仍在运行，避免向已关闭的通道发送数据
		po.mu.RLock()
		isRunning := po.isRunning
		po.mu.RUnlock()

		if isRunning {
			select {
			case po.errorChan <- fmt.Errorf("收集当前指标失败: %w", err):
			default:
			}
		}
		return
	}

	// 更新当前状态
	po.mu.Lock()
	po.metrics.CurrentMemoryMB = currentMetrics.MemoryMB
	po.metrics.CurrentCPUPercent = currentMetrics.CPUPercent
	po.metrics.CurrentGoroutines = currentMetrics.GoroutineCount
	po.mu.Unlock()

	// 并发执行各种优化策略

	// 内存优化
	if po.config.EnableAutoGC && currentMetrics.MemoryMB > po.config.MemoryThresholdMB {
		wg.Add(1)
		go func() {
			defer wg.Done()
			action := po.optimizeMemory(ctx, currentMetrics)
			mu.Lock()
			actions = append(actions, action)
			mu.Unlock()
		}()
	}

	// CPU优化
	if po.config.EnableCPUThrottling && currentMetrics.CPUPercent > po.config.CPUThresholdPercent {
		wg.Add(1)
		go func() {
			defer wg.Done()
			action := po.optimizeCPU(ctx, currentMetrics)
			mu.Lock()
			actions = append(actions, action)
			mu.Unlock()
		}()
	}

	// GC优化
	if po.config.EnableAutoGC && currentMetrics.HeapMB > po.config.GCThresholdMB {
		wg.Add(1)
		go func() {
			defer wg.Done()
			action := po.optimizeGC(ctx, currentMetrics)
			mu.Lock()
			actions = append(actions, action)
			mu.Unlock()
		}()
	}

	// 协程优化
	if currentMetrics.GoroutineCount > po.config.GoroutineThreshold {
		wg.Add(1)
		go func() {
			defer wg.Done()
			action := po.optimizeGoroutines(ctx, currentMetrics)
			mu.Lock()
			actions = append(actions, action)
			mu.Unlock()
		}()
	}

	// 等待所有优化任务完成或超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 所有优化任务完成
	case <-ctx.Done():
		// 超时
		// 检查优化器是否仍在运行，避免向已关闭的通道发送数据
		po.mu.RLock()
		isRunning := po.isRunning
		po.mu.RUnlock()

		if isRunning {
			select {
			case po.errorChan <- fmt.Errorf("优化任务超时"):
			default:
			}
		}
		return
	}

	// 更新优化统计
	po.updateOptimizationStats(actions)

	// 发送告警（如果需要）
	po.checkOptimizationAlerts(currentMetrics, actions)
}

// collectCurrentMetrics 收集当前指标
func (po *PerformanceOptimizer) collectCurrentMetrics(ctx context.Context) (*CurrentMetrics, error) {
	metrics := &CurrentMetrics{}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var collectError error

	// 收集内存指标
	wg.Add(1)
	go func() {
		defer wg.Done()
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		mu.Lock()
		metrics.MemoryMB = float64(memStats.Alloc) / 1024 / 1024
		metrics.HeapMB = float64(memStats.HeapAlloc) / 1024 / 1024
		metrics.GoroutineCount = runtime.NumGoroutine()
		mu.Unlock()
	}()

	// 收集CPU指标
	wg.Add(1)
	go func() {
		defer wg.Done()
		cpuPercents, err := cpu.PercentWithContext(ctx, time.Second, false)
		if err != nil {
			mu.Lock()
			if collectError == nil {
				collectError = err
			}
			mu.Unlock()
			return
		}

		if len(cpuPercents) > 0 {
			mu.Lock()
			metrics.CPUPercent = cpuPercents[0]
			mu.Unlock()
		}
	}()

	// 收集系统内存指标
	wg.Add(1)
	go func() {
		defer wg.Done()
		vmem, err := mem.VirtualMemoryWithContext(ctx)
		if err != nil {
			mu.Lock()
			if collectError == nil {
				collectError = err
			}
			mu.Unlock()
			return
		}

		mu.Lock()
		metrics.SystemMemoryPercent = vmem.UsedPercent
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
		return nil, fmt.Errorf("收集指标超时")
	}

	if collectError != nil {
		return nil, collectError
	}

	return metrics, nil
}

// CurrentMetrics 当前指标
type CurrentMetrics struct {
	MemoryMB            float64
	HeapMB              float64
	CPUPercent          float64
	GoroutineCount      int
	SystemMemoryPercent float64
}

// optimizeMemory 优化内存
func (po *PerformanceOptimizer) optimizeMemory(ctx context.Context, metrics *CurrentMetrics) OptimizationAction {
	action := OptimizationAction{
		Type:        "memory_optimization",
		Description: "内存优化",
		Parameters:  make(map[string]interface{}),
		Timestamp:   time.Now(),
	}

	beforeMB := metrics.MemoryMB

	// 强制垃圾回收
	runtime.GC()

	// 释放未使用的内存给操作系统
	debug.FreeOSMemory()

	// 收集优化后的内存使用
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	afterMB := float64(memStats.Alloc) / 1024 / 1024

	savedMB := beforeMB - afterMB
	action.Parameters["before_mb"] = beforeMB
	action.Parameters["after_mb"] = afterMB
	action.Parameters["saved_mb"] = savedMB
	action.Success = savedMB > 0

	if action.Success {
		po.mu.Lock()
		po.metrics.MemoryOptimizations++
		po.metrics.MemorySavedMB += savedMB
		po.mu.Unlock()
	}

	return action
}

// optimizeCPU 优化CPU
func (po *PerformanceOptimizer) optimizeCPU(ctx context.Context, metrics *CurrentMetrics) OptimizationAction {
	action := OptimizationAction{
		Type:        "cpu_optimization",
		Description: "CPU优化",
		Parameters:  make(map[string]interface{}),
		Timestamp:   time.Now(),
	}

	beforePercent := metrics.CPUPercent

	// 调整GOMAXPROCS（如果配置了CPU配额）
	if po.config.CPUQuotaPercent > 0 {
		maxProcs := runtime.GOMAXPROCS(0)
		targetProcs := int(float64(maxProcs) * po.config.CPUQuotaPercent / 100.0)
		if targetProcs < 1 {
			targetProcs = 1
		}
		if targetProcs < maxProcs {
			runtime.GOMAXPROCS(targetProcs)
			action.Parameters["max_procs_before"] = maxProcs
			action.Parameters["max_procs_after"] = targetProcs
			action.Success = true
		}
	}

	// 触发调度器让出CPU
	runtime.Gosched()

	action.Parameters["before_percent"] = beforePercent

	if action.Success {
		po.mu.Lock()
		po.metrics.CPUOptimizations++
		po.mu.Unlock()
	}

	return action
}

// optimizeGC 优化垃圾回收
func (po *PerformanceOptimizer) optimizeGC(ctx context.Context, metrics *CurrentMetrics) OptimizationAction {
	action := OptimizationAction{
		Type:        "gc_optimization",
		Description: "垃圾回收优化",
		Parameters:  make(map[string]interface{}),
		Timestamp:   time.Now(),
	}

	var beforeStats runtime.MemStats
	runtime.ReadMemStats(&beforeStats)

	// 调整GC目标百分比
	oldGCPercent := debug.SetGCPercent(po.config.GCTargetPercent)

	// 强制执行GC
	runtime.GC()

	var afterStats runtime.MemStats
	runtime.ReadMemStats(&afterStats)

	pauseReduced := float64(beforeStats.PauseNs[(beforeStats.NumGC+255)%256]-afterStats.PauseNs[(afterStats.NumGC+255)%256]) / 1e6

	action.Parameters["gc_percent_before"] = oldGCPercent
	action.Parameters["gc_percent_after"] = po.config.GCTargetPercent
	action.Parameters["pause_reduced_ms"] = pauseReduced
	action.Parameters["gc_count_before"] = beforeStats.NumGC
	action.Parameters["gc_count_after"] = afterStats.NumGC
	action.Success = true

	po.mu.Lock()
	po.metrics.GCOptimizations++
	if pauseReduced > 0 {
		po.metrics.GCPauseReduced += pauseReduced
	}
	po.mu.Unlock()

	return action
}

// optimizeGoroutines 优化协程
func (po *PerformanceOptimizer) optimizeGoroutines(ctx context.Context, metrics *CurrentMetrics) OptimizationAction {
	action := OptimizationAction{
		Type:        "goroutine_optimization",
		Description: "协程优化",
		Parameters:  make(map[string]interface{}),
		Timestamp:   time.Now(),
	}

	beforeCount := metrics.GoroutineCount

	// 触发调度器，让空闲的协程有机会退出
	runtime.Gosched()
	runtime.GC() // GC可能会清理一些协程相关的资源

	afterCount := runtime.NumGoroutine()
	reduced := beforeCount - afterCount

	action.Parameters["before_count"] = beforeCount
	action.Parameters["after_count"] = afterCount
	action.Parameters["reduced_count"] = reduced
	action.Success = reduced > 0

	return action
}

// updateOptimizationStats 更新优化统计
func (po *PerformanceOptimizer) updateOptimizationStats(actions []OptimizationAction) {
	po.mu.Lock()
	defer po.mu.Unlock()

	successfulActions := 0
	for _, action := range actions {
		if action.Success {
			successfulActions++
		}
	}

	if successfulActions > 0 {
		po.metrics.TotalOptimizations++
		po.metrics.LastOptimizationTime = time.Now()
		po.lastOptimization = time.Now()
		po.optimizationCount++
	}
}

// checkOptimizationAlerts 检查优化告警
func (po *PerformanceOptimizer) checkOptimizationAlerts(metrics *CurrentMetrics, actions []OptimizationAction) {
	alerts := make([]OptimizationAlert, 0)

	// 检查内存使用告警
	if metrics.MemoryMB > po.config.MemoryThresholdMB {
		alert := OptimizationAlert{
			ID:        fmt.Sprintf("memory_high_%d", time.Now().Unix()),
			Type:      "memory_usage",
			Severity:  "warning",
			Message:   fmt.Sprintf("内存使用过高: %.2f MB", metrics.MemoryMB),
			Action:    "memory_optimization",
			Value:     metrics.MemoryMB,
			Threshold: po.config.MemoryThresholdMB,
			Timestamp: time.Now(),
		}
		alerts = append(alerts, alert)
	}

	// 检查CPU使用告警
	if metrics.CPUPercent > po.config.CPUThresholdPercent {
		alert := OptimizationAlert{
			ID:        fmt.Sprintf("cpu_high_%d", time.Now().Unix()),
			Type:      "cpu_usage",
			Severity:  "warning",
			Message:   fmt.Sprintf("CPU使用过高: %.2f%%", metrics.CPUPercent),
			Action:    "cpu_optimization",
			Value:     metrics.CPUPercent,
			Threshold: po.config.CPUThresholdPercent,
			Timestamp: time.Now(),
		}
		alerts = append(alerts, alert)
	}

	// 检查协程数量告警
	if metrics.GoroutineCount > po.config.GoroutineThreshold {
		alert := OptimizationAlert{
			ID:        fmt.Sprintf("goroutine_high_%d", time.Now().Unix()),
			Type:      "goroutine_count",
			Severity:  "warning",
			Message:   fmt.Sprintf("协程数量过多: %d", metrics.GoroutineCount),
			Action:    "goroutine_optimization",
			Value:     float64(metrics.GoroutineCount),
			Threshold: float64(po.config.GoroutineThreshold),
			Timestamp: time.Now(),
		}
		alerts = append(alerts, alert)
	}

	// 发送告警
	for _, alert := range alerts {
		select {
		case po.alertChan <- alert:
		default:
			// 告警通道满了，丢弃告警
		}
	}
}

// GetMetrics 获取优化指标
func (po *PerformanceOptimizer) GetMetrics() OptimizationMetrics {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.metrics
}

// GetAlertChannel 获取告警通道
func (po *PerformanceOptimizer) GetAlertChannel() <-chan OptimizationAlert {
	return po.alertChan
}

// GetErrorChannel 获取错误通道
func (po *PerformanceOptimizer) GetErrorChannel() <-chan error {
	return po.errorChan
}

// IsRunning 检查是否正在运行
func (po *PerformanceOptimizer) IsRunning() bool {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.isRunning
}

// ForceOptimization 强制执行优化
func (po *PerformanceOptimizer) ForceOptimization() error {
	if !po.isRunning {
		return fmt.Errorf("性能优化器未运行")
	}

	ctx, cancel := context.WithTimeout(po.ctx, po.config.Timeout)
	defer cancel()

	po.performOptimization(ctx)
	return nil
}
