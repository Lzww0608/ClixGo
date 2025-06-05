/*
* @Author: Lzww0608
* @Date: 2025-06-05 11:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-05 20:19:33
* @Description: 内存使用监控器 - 提供实时内存监控、pprof集成和内存优化功能
 */

package performance

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" // 导入pprof HTTP handlers
	"runtime"
	"runtime/pprof"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// MemoryMonitor 内存使用监控器
// 提供实时内存监控、分析和优化建议
type MemoryMonitor struct {
	mu                  sync.RWMutex                // 读写锁保护并发访问
	isRunning           bool                        // 监控器运行状态
	ctx                 context.Context             // 上下文控制
	cancel              context.CancelFunc          // 取消函数
	config              MemoryMonitorConfig         // 监控器配置
	logger              *zap.Logger                 // 日志记录器
	baseline            *MemoryBaseline             // 内存基线
	snapshots           []MemorySnapshot            // 历史快照
	alertChan           chan MemoryAlert            // 告警通道
	optimizationChan    chan OptimizationSuggestion // 优化建议通道
	errorChan           chan error                  // 错误通道
	activeGoroutines    sync.WaitGroup              // 活跃goroutine计数
	shutdownOnce        sync.Once                   // 确保只关闭一次
	channelsClosed      int32                       // 通道关闭标志
	pprofServer         *http.Server                // pprof HTTP服务器
	profileCollector    *ProfileCollector           // 性能剖析收集器
	memoryOptimizer     *MemoryOptimizer            // 内存优化器
	lastOptimizationRun time.Time                   // 上次优化运行时间
}

// MemoryMonitorConfig 内存监控器配置
type MemoryMonitorConfig struct {
	MonitorInterval        time.Duration `json:"monitor_interval"`         // 监控间隔
	BaselineWarmupTime     time.Duration `json:"baseline_warmup_time"`     // 基线预热时间
	MaxSnapshots           int           `json:"max_snapshots"`            // 最大快照数量
	EnablePprof            bool          `json:"enable_pprof"`             // 启用pprof
	PprofAddress           string        `json:"pprof_address"`            // pprof服务地址
	EnableAutoOptimization bool          `json:"enable_auto_optimization"` // 启用自动优化
	OptimizationInterval   time.Duration `json:"optimization_interval"`    // 优化间隔
	ProfileCollectionDepth int           `json:"profile_collection_depth"` // 性能剖析收集深度

	// 告警阈值
	MemoryGrowthThresholdMB float64 `json:"memory_growth_threshold_mb"` // 内存增长阈值(MB)
	HeapGrowthThresholdMB   float64 `json:"heap_growth_threshold_mb"`   // 堆内存增长阈值(MB)
	GCPressureThreshold     float64 `json:"gc_pressure_threshold"`      // GC压力阈值
	FragmentationThreshold  float64 `json:"fragmentation_threshold"`    // 内存碎片化阈值
	StackGrowthThresholdMB  float64 `json:"stack_growth_threshold_mb"`  // 栈内存增长阈值(MB)

	// 优化参数
	AutoGCTriggerThresholdMB float64 `json:"auto_gc_trigger_threshold_mb"` // 自动GC触发阈值(MB)
	MemoryReleaseThresholdMB float64 `json:"memory_release_threshold_mb"`  // 内存释放阈值(MB)
	MaxOptimizationRetries   int     `json:"max_optimization_retries"`     // 最大优化重试次数
}

// MemoryBaseline 内存基线
type MemoryBaseline struct {
	Timestamp      time.Time `json:"timestamp"`
	HeapAllocMB    float64   `json:"heap_alloc_mb"`
	HeapSysMB      float64   `json:"heap_sys_mb"`
	HeapIdleMB     float64   `json:"heap_idle_mb"`
	HeapInuseMB    float64   `json:"heap_inuse_mb"`
	HeapReleasedMB float64   `json:"heap_released_mb"`
	StackInuseMB   float64   `json:"stack_inuse_mb"`
	StackSysMB     float64   `json:"stack_sys_mb"`
	MSpanInuseMB   float64   `json:"mspan_inuse_mb"`
	MSpanSysMB     float64   `json:"mspan_sys_mb"`
	MCacheInuseMB  float64   `json:"mcache_inuse_mb"`
	MCacheSysMB    float64   `json:"mcache_sys_mb"`
	BuckHashSysMB  float64   `json:"buck_hash_sys_mb"`
	GCSysMB        float64   `json:"gc_sys_mb"`
	OtherSysMB     float64   `json:"other_sys_mb"`
	NextGCMB       float64   `json:"next_gc_mb"`
	GCCPUFraction  float64   `json:"gc_cpu_fraction"`
	GoroutineCount int       `json:"goroutine_count"`
}

// MemorySnapshot 内存快照
type MemorySnapshot struct {
	Timestamp           time.Time             `json:"timestamp"`
	HeapAllocMB         float64               `json:"heap_alloc_mb"`
	HeapSysMB           float64               `json:"heap_sys_mb"`
	HeapIdleMB          float64               `json:"heap_idle_mb"`
	HeapInuseMB         float64               `json:"heap_inuse_mb"`
	HeapReleasedMB      float64               `json:"heap_released_mb"`
	StackInuseMB        float64               `json:"stack_inuse_mb"`
	StackSysMB          float64               `json:"stack_sys_mb"`
	MSpanInuseMB        float64               `json:"mspan_inuse_mb"`
	MSpanSysMB          float64               `json:"mspan_sys_mb"`
	MCacheInuseMB       float64               `json:"mcache_inuse_mb"`
	MCacheSysMB         float64               `json:"mcache_sys_mb"`
	BuckHashSysMB       float64               `json:"buck_hash_sys_mb"`
	GCSysMB             float64               `json:"gc_sys_mb"`
	OtherSysMB          float64               `json:"other_sys_mb"`
	NextGCMB            float64               `json:"next_gc_mb"`
	GCCPUFraction       float64               `json:"gc_cpu_fraction"`
	GoroutineCount      int                   `json:"goroutine_count"`
	GCCount             uint32                `json:"gc_count"`
	GCPauseTotalMs      float64               `json:"gc_pause_total_ms"`
	FragmentationRatio  float64               `json:"fragmentation_ratio"`
	TopAllocators       []MemoryAllocatorInfo `json:"top_allocators"`
	MemoryHotspots      []MemoryHotspot       `json:"memory_hotspots"`
	GCPressureIndicator float64               `json:"gc_pressure_indicator"`
}

// MemoryAllocatorInfo 内存分配器信息
type MemoryAllocatorInfo struct {
	Function   string `json:"function"`
	InUseBytes uint64 `json:"in_use_bytes"`
	InUseCount uint64 `json:"in_use_count"`
	AllocBytes uint64 `json:"alloc_bytes"`
	AllocCount uint64 `json:"alloc_count"`
	Stack      string `json:"stack"`
}

// MemoryHotspot 内存热点
type MemoryHotspot struct {
	Location        string  `json:"location"`
	AllocatedMB     float64 `json:"allocated_mb"`
	AllocationCount int64   `json:"allocation_count"`
	StackTrace      string  `json:"stack_trace"`
	Severity        string  `json:"severity"`
}

// MemoryAlert 内存告警
type MemoryAlert struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Severity     string             `json:"severity"`
	Title        string             `json:"title"`
	Description  string             `json:"description"`
	Evidence     []string           `json:"evidence"`
	Snapshot     MemorySnapshot     `json:"snapshot"`
	Baseline     MemoryBaseline     `json:"baseline"`
	Timestamp    time.Time          `json:"timestamp"`
	Suggestions  []string           `json:"suggestions"`
	MetricValues map[string]float64 `json:"metric_values"`
}

// OptimizationSuggestion 优化建议
type OptimizationSuggestion struct {
	ID          string                     `json:"id"`
	Type        string                     `json:"type"`
	Priority    string                     `json:"priority"`
	Title       string                     `json:"title"`
	Description string                     `json:"description"`
	Actions     []MemoryOptimizationAction `json:"actions"`
	Impact      OptimizationImpact         `json:"impact"`
	Timestamp   time.Time                  `json:"timestamp"`
	Context     map[string]interface{}     `json:"context"`
}

// MemoryOptimizationAction 内存优化操作
type MemoryOptimizationAction struct {
	Action      string                 `json:"action"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	AutoExecute bool                   `json:"auto_execute"`
	Executed    bool                   `json:"executed"`
	Result      string                 `json:"result"`
}

// OptimizationImpact 优化影响
type OptimizationImpact struct {
	EstimatedMemorySavingMB float64 `json:"estimated_memory_saving_mb"`
	EstimatedCPUSaving      float64 `json:"estimated_cpu_saving"`
	EstimatedGCImprovement  float64 `json:"estimated_gc_improvement"`
	RiskLevel               string  `json:"risk_level"`
}

// ProfileCollector 性能剖析收集器
type ProfileCollector struct {
	mu              sync.RWMutex
	profiles        map[string]*ProfileData
	maxProfiles     int
	collectionDepth int
	logger          *zap.Logger
}

// ProfileData 性能剖析数据
type ProfileData struct {
	Type       string    `json:"type"`
	Timestamp  time.Time `json:"timestamp"`
	Data       []byte    `json:"data"`
	Summary    string    `json:"summary"`
	TopEntries []string  `json:"top_entries"`
}

// MemoryOptimizer 内存优化器
type MemoryOptimizer struct {
	mu                sync.RWMutex
	config            MemoryMonitorConfig
	logger            *zap.Logger
	lastGCTime        time.Time
	gcCounter         uint64
	optimizationStats map[string]int64
}

// NewMemoryMonitor 创建新的内存监控器
func NewMemoryMonitor(config MemoryMonitorConfig, logger *zap.Logger) *MemoryMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	// 设置默认配置
	if config.MonitorInterval == 0 {
		config.MonitorInterval = 10 * time.Second
	}
	if config.BaselineWarmupTime == 0 {
		config.BaselineWarmupTime = 30 * time.Second
	}
	if config.MaxSnapshots == 0 {
		config.MaxSnapshots = 200
	}
	if config.PprofAddress == "" {
		config.PprofAddress = ":6060"
	}
	if config.OptimizationInterval == 0 {
		config.OptimizationInterval = 2 * time.Minute
	}
	if config.ProfileCollectionDepth == 0 {
		config.ProfileCollectionDepth = 20
	}
	if config.MemoryGrowthThresholdMB == 0 {
		config.MemoryGrowthThresholdMB = 100.0
	}
	if config.HeapGrowthThresholdMB == 0 {
		config.HeapGrowthThresholdMB = 50.0
	}
	if config.GCPressureThreshold == 0 {
		config.GCPressureThreshold = 0.1
	}
	if config.FragmentationThreshold == 0 {
		config.FragmentationThreshold = 0.3
	}
	if config.AutoGCTriggerThresholdMB == 0 {
		config.AutoGCTriggerThresholdMB = 200.0
	}
	if config.MemoryReleaseThresholdMB == 0 {
		config.MemoryReleaseThresholdMB = 150.0
	}
	if config.MaxOptimizationRetries == 0 {
		config.MaxOptimizationRetries = 3
	}

	monitor := &MemoryMonitor{
		ctx:              ctx,
		cancel:           cancel,
		config:           config,
		logger:           logger,
		snapshots:        make([]MemorySnapshot, 0, config.MaxSnapshots),
		alertChan:        make(chan MemoryAlert, 50),
		optimizationChan: make(chan OptimizationSuggestion, 20),
		errorChan:        make(chan error, 10),
		profileCollector: NewProfileCollector(config.ProfileCollectionDepth, logger),
		memoryOptimizer:  NewMemoryOptimizer(config, logger),
	}

	return monitor
}

// NewProfileCollector 创建新的性能剖析收集器
func NewProfileCollector(depth int, logger *zap.Logger) *ProfileCollector {
	return &ProfileCollector{
		profiles:        make(map[string]*ProfileData),
		maxProfiles:     100,
		collectionDepth: depth,
		logger:          logger,
	}
}

// NewMemoryOptimizer 创建新的内存优化器
func NewMemoryOptimizer(config MemoryMonitorConfig, logger *zap.Logger) *MemoryOptimizer {
	return &MemoryOptimizer{
		config:            config,
		logger:            logger,
		optimizationStats: make(map[string]int64),
	}
}

// Start 启动内存监控器
func (mm *MemoryMonitor) Start() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mm.isRunning {
		return fmt.Errorf("内存监控器已在运行中")
	}

	mm.isRunning = true
	mm.logger.Info("启动内存监控器")

	// 启动pprof服务器（如果启用）
	if mm.config.EnablePprof {
		go mm.startPprofServer()
	}

	// 建立基线
	go mm.establishBaseline()

	// 启动监控循环
	go mm.monitoringLoop()

	// 启动优化循环（如果启用）
	if mm.config.EnableAutoOptimization {
		go mm.optimizationLoop()
	}

	return nil
}

// Stop 停止内存监控器
func (mm *MemoryMonitor) Stop() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if !mm.isRunning {
		return fmt.Errorf("内存监控器未在运行")
	}

	mm.isRunning = false
	mm.logger.Info("停止内存监控器")

	// 设置停止标志
	atomic.StoreInt32(&mm.channelsClosed, 1)

	// 取消上下文
	mm.cancel()

	// 等待活跃goroutine退出
	done := make(chan struct{})
	go func() {
		mm.activeGoroutines.Wait()
		close(done)
	}()

	select {
	case <-done:
		mm.logger.Info("所有goroutine已安全退出")
	case <-time.After(5 * time.Second):
		mm.logger.Warn("等待goroutine退出超时")
	}

	// 安全关闭通道
	mm.shutdownOnce.Do(func() {
		close(mm.alertChan)
		close(mm.optimizationChan)
		close(mm.errorChan)
	})

	// 停止pprof服务器
	if mm.pprofServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mm.pprofServer.Shutdown(ctx); err != nil {
			mm.logger.Error("停止pprof服务器失败", zap.Error(err))
		}
	}

	return nil
}

// startPprofServer 启动pprof HTTP服务器
func (mm *MemoryMonitor) startPprofServer() {
	mm.pprofServer = &http.Server{
		Addr:    mm.config.PprofAddress,
		Handler: nil, // 使用默认的pprof handlers
	}

	mm.logger.Info("启动pprof服务器", zap.String("address", mm.config.PprofAddress))

	if err := mm.pprofServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		mm.safeSendError(fmt.Errorf("pprof服务器启动失败: %w", err))
	}
}

// establishBaseline 建立内存基线
func (mm *MemoryMonitor) establishBaseline() {
	mm.activeGoroutines.Add(1)
	defer mm.activeGoroutines.Done()

	mm.logger.Info("开始建立内存基线", zap.Duration("warmup_time", mm.config.BaselineWarmupTime))

	// 等待预热期
	select {
	case <-time.After(mm.config.BaselineWarmupTime):
	case <-mm.ctx.Done():
		return
	}

	// 收集基线数据
	baseline := mm.captureMemoryBaseline()

	mm.mu.Lock()
	mm.baseline = baseline
	mm.mu.Unlock()

	mm.logger.Info("内存基线建立完成",
		zap.Float64("heap_alloc_mb", baseline.HeapAllocMB),
		zap.Float64("heap_sys_mb", baseline.HeapSysMB),
		zap.Int("goroutine_count", baseline.GoroutineCount))
}

// monitoringLoop 监控循环
func (mm *MemoryMonitor) monitoringLoop() {
	mm.activeGoroutines.Add(1)
	defer mm.activeGoroutines.Done()

	ticker := time.NewTicker(mm.config.MonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mm.ctx.Done():
			return
		case <-ticker.C:
			mm.performMonitoring()
		}
	}
}

// optimizationLoop 优化循环
func (mm *MemoryMonitor) optimizationLoop() {
	mm.activeGoroutines.Add(1)
	defer mm.activeGoroutines.Done()

	ticker := time.NewTicker(mm.config.OptimizationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mm.ctx.Done():
			return
		case <-ticker.C:
			if time.Since(mm.lastOptimizationRun) >= mm.config.OptimizationInterval {
				mm.performOptimization()
				mm.lastOptimizationRun = time.Now()
			}
		}
	}
}

// performMonitoring 执行监控
func (mm *MemoryMonitor) performMonitoring() {
	snapshot, err := mm.captureMemorySnapshot()
	if err != nil {
		mm.safeSendError(fmt.Errorf("捕获内存快照失败: %w", err))
		return
	}

	// 添加快照到历史记录
	mm.addSnapshot(snapshot)

	// 检查告警条件
	mm.checkAlerts(snapshot)

	// 收集性能剖析数据
	if mm.config.EnablePprof {
		mm.collectProfiles()
	}
}

// performOptimization 执行优化
func (mm *MemoryMonitor) performOptimization() {
	// 获取最新快照
	mm.mu.RLock()
	if len(mm.snapshots) == 0 {
		mm.mu.RUnlock()
		return
	}
	latestSnapshot := mm.snapshots[len(mm.snapshots)-1]
	mm.mu.RUnlock()

	// 分析优化机会
	suggestions := mm.memoryOptimizer.AnalyzeOptimizationOpportunities(latestSnapshot, mm.baseline)

	// 发送优化建议
	for _, suggestion := range suggestions {
		mm.safeSendOptimizationSuggestion(suggestion)
	}

	// 执行自动优化
	if mm.config.EnableAutoOptimization {
		mm.memoryOptimizer.ExecuteAutoOptimizations(suggestions)
	}
}

// captureMemoryBaseline 捕获内存基线
func (mm *MemoryMonitor) captureMemoryBaseline() *MemoryBaseline {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &MemoryBaseline{
		Timestamp:      time.Now(),
		HeapAllocMB:    float64(m.HeapAlloc) / 1024 / 1024,
		HeapSysMB:      float64(m.HeapSys) / 1024 / 1024,
		HeapIdleMB:     float64(m.HeapIdle) / 1024 / 1024,
		HeapInuseMB:    float64(m.HeapInuse) / 1024 / 1024,
		HeapReleasedMB: float64(m.HeapReleased) / 1024 / 1024,
		StackInuseMB:   float64(m.StackInuse) / 1024 / 1024,
		StackSysMB:     float64(m.StackSys) / 1024 / 1024,
		MSpanInuseMB:   float64(m.MSpanInuse) / 1024 / 1024,
		MSpanSysMB:     float64(m.MSpanSys) / 1024 / 1024,
		MCacheInuseMB:  float64(m.MCacheInuse) / 1024 / 1024,
		MCacheSysMB:    float64(m.MCacheSys) / 1024 / 1024,
		BuckHashSysMB:  float64(m.BuckHashSys) / 1024 / 1024,
		GCSysMB:        float64(m.GCSys) / 1024 / 1024,
		OtherSysMB:     float64(m.OtherSys) / 1024 / 1024,
		NextGCMB:       float64(m.NextGC) / 1024 / 1024,
		GCCPUFraction:  m.GCCPUFraction,
		GoroutineCount: runtime.NumGoroutine(),
	}
}

// captureMemorySnapshot 捕获内存快照
func (mm *MemoryMonitor) captureMemorySnapshot() (MemorySnapshot, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	snapshot := MemorySnapshot{
		Timestamp:           time.Now(),
		HeapAllocMB:         float64(m.HeapAlloc) / 1024 / 1024,
		HeapSysMB:           float64(m.HeapSys) / 1024 / 1024,
		HeapIdleMB:          float64(m.HeapIdle) / 1024 / 1024,
		HeapInuseMB:         float64(m.HeapInuse) / 1024 / 1024,
		HeapReleasedMB:      float64(m.HeapReleased) / 1024 / 1024,
		StackInuseMB:        float64(m.StackInuse) / 1024 / 1024,
		StackSysMB:          float64(m.StackSys) / 1024 / 1024,
		MSpanInuseMB:        float64(m.MSpanInuse) / 1024 / 1024,
		MSpanSysMB:          float64(m.MSpanSys) / 1024 / 1024,
		MCacheInuseMB:       float64(m.MCacheInuse) / 1024 / 1024,
		MCacheSysMB:         float64(m.MCacheSys) / 1024 / 1024,
		BuckHashSysMB:       float64(m.BuckHashSys) / 1024 / 1024,
		GCSysMB:             float64(m.GCSys) / 1024 / 1024,
		OtherSysMB:          float64(m.OtherSys) / 1024 / 1024,
		NextGCMB:            float64(m.NextGC) / 1024 / 1024,
		GCCPUFraction:       m.GCCPUFraction,
		GoroutineCount:      runtime.NumGoroutine(),
		GCCount:             m.NumGC,
		GCPauseTotalMs:      float64(m.PauseTotalNs) / 1000000,
		FragmentationRatio:  mm.calculateFragmentationRatio(&m),
		GCPressureIndicator: mm.calculateGCPressure(&m),
	}

	// 收集内存分配器信息
	allocators, err := mm.collectMemoryAllocators()
	if err != nil {
		mm.logger.Warn("收集内存分配器信息失败", zap.Error(err))
	} else {
		snapshot.TopAllocators = allocators
	}

	// 识别内存热点
	hotspots := mm.identifyMemoryHotspots()
	snapshot.MemoryHotspots = hotspots

	return snapshot, nil
}

// calculateFragmentationRatio 计算内存碎片化比率
func (mm *MemoryMonitor) calculateFragmentationRatio(m *runtime.MemStats) float64 {
	if m.HeapSys == 0 {
		return 0
	}
	return float64(m.HeapSys-m.HeapInuse) / float64(m.HeapSys)
}

// calculateGCPressure 计算GC压力指示器
func (mm *MemoryMonitor) calculateGCPressure(m *runtime.MemStats) float64 {
	// 基于GC频率和暂停时间计算压力指示器
	if m.NumGC == 0 {
		return 0
	}

	// 简化的GC压力计算
	avgPauseMs := float64(m.PauseTotalNs) / float64(m.NumGC) / 1000000
	return avgPauseMs * m.GCCPUFraction * 100
}

// collectMemoryAllocators 收集内存分配器信息
func (mm *MemoryMonitor) collectMemoryAllocators() ([]MemoryAllocatorInfo, error) {
	var buf bytes.Buffer
	if err := pprof.WriteHeapProfile(&buf); err != nil {
		return nil, fmt.Errorf("生成堆内存profiling失败: %w", err)
	}

	// 解析profiling数据（简化版本）
	allocators := make([]MemoryAllocatorInfo, 0)

	// 这里应该解析pprof数据，但为了简化，我们使用模拟数据
	// 实际实现中需要使用pprof解析库

	return allocators, nil
}

// identifyMemoryHotspots 识别内存热点
func (mm *MemoryMonitor) identifyMemoryHotspots() []MemoryHotspot {
	hotspots := make([]MemoryHotspot, 0)

	// 基于历史数据分析内存使用模式
	mm.mu.RLock()
	if len(mm.snapshots) < 2 {
		mm.mu.RUnlock()
		return hotspots
	}

	current := mm.snapshots[len(mm.snapshots)-1]
	previous := mm.snapshots[len(mm.snapshots)-2]
	mm.mu.RUnlock()

	// 检查快速增长的内存区域
	if current.HeapAllocMB-previous.HeapAllocMB > 10.0 {
		hotspots = append(hotspots, MemoryHotspot{
			Location:        "heap_allocation",
			AllocatedMB:     current.HeapAllocMB - previous.HeapAllocMB,
			AllocationCount: int64(current.GCCount - previous.GCCount),
			StackTrace:      "runtime.main -> heap allocation",
			Severity:        "high",
		})
	}

	return hotspots
}

// addSnapshot 添加快照到历史记录
func (mm *MemoryMonitor) addSnapshot(snapshot MemorySnapshot) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.snapshots = append(mm.snapshots, snapshot)

	// 保持最大快照数量限制
	if len(mm.snapshots) > mm.config.MaxSnapshots {
		// 删除最旧的快照
		copy(mm.snapshots, mm.snapshots[1:])
		mm.snapshots = mm.snapshots[:len(mm.snapshots)-1]
	}
}

// checkAlerts 检查告警条件
func (mm *MemoryMonitor) checkAlerts(snapshot MemorySnapshot) {
	if mm.baseline == nil {
		return
	}

	var alerts []MemoryAlert

	// 检查内存增长
	memoryGrowth := snapshot.HeapAllocMB - mm.baseline.HeapAllocMB
	if memoryGrowth > mm.config.MemoryGrowthThresholdMB {
		alert := mm.createMemoryGrowthAlert(snapshot, memoryGrowth)
		alerts = append(alerts, alert)
	}

	// 检查堆内存增长
	heapGrowth := snapshot.HeapSysMB - mm.baseline.HeapSysMB
	if heapGrowth > mm.config.HeapGrowthThresholdMB {
		alert := mm.createHeapGrowthAlert(snapshot, heapGrowth)
		alerts = append(alerts, alert)
	}

	// 检查GC压力
	if snapshot.GCPressureIndicator > mm.config.GCPressureThreshold {
		alert := mm.createGCPressureAlert(snapshot)
		alerts = append(alerts, alert)
	}

	// 检查内存碎片化
	if snapshot.FragmentationRatio > mm.config.FragmentationThreshold {
		alert := mm.createFragmentationAlert(snapshot)
		alerts = append(alerts, alert)
	}

	// 发送告警
	for _, alert := range alerts {
		mm.safeSendAlert(alert)
	}
}

// collectProfiles 收集性能剖析数据
func (mm *MemoryMonitor) collectProfiles() {
	profiles := []string{"heap", "goroutine", "allocs"}

	for _, profileType := range profiles {
		go func(pType string) {
			if err := mm.profileCollector.CollectProfile(pType); err != nil {
				mm.safeSendError(fmt.Errorf("收集%s profile失败: %w", pType, err))
			}
		}(profileType)
	}
}

// createMemoryGrowthAlert 创建内存增长告警
func (mm *MemoryMonitor) createMemoryGrowthAlert(snapshot MemorySnapshot, growth float64) MemoryAlert {
	return MemoryAlert{
		ID:          fmt.Sprintf("memory_growth_%d", time.Now().Unix()),
		Type:        "memory_growth",
		Severity:    mm.calculateAlertSeverity(growth, mm.config.MemoryGrowthThresholdMB),
		Title:       "内存使用量异常增长",
		Description: fmt.Sprintf("检测到内存使用量增长 %.2f MB，超过阈值 %.2f MB", growth, mm.config.MemoryGrowthThresholdMB),
		Evidence:    []string{fmt.Sprintf("当前堆内存: %.2f MB", snapshot.HeapAllocMB), fmt.Sprintf("基线堆内存: %.2f MB", mm.baseline.HeapAllocMB)},
		Snapshot:    snapshot,
		Baseline:    *mm.baseline,
		Timestamp:   time.Now(),
		Suggestions: []string{"检查内存泄漏", "增加GC频率", "优化内存使用模式"},
		MetricValues: map[string]float64{
			"growth_mb":   growth,
			"threshold":   mm.config.MemoryGrowthThresholdMB,
			"current_mb":  snapshot.HeapAllocMB,
			"baseline_mb": mm.baseline.HeapAllocMB,
		},
	}
}

// createHeapGrowthAlert 创建堆内存增长告警
func (mm *MemoryMonitor) createHeapGrowthAlert(snapshot MemorySnapshot, growth float64) MemoryAlert {
	return MemoryAlert{
		ID:          fmt.Sprintf("heap_growth_%d", time.Now().Unix()),
		Type:        "heap_growth",
		Severity:    mm.calculateAlertSeverity(growth, mm.config.HeapGrowthThresholdMB),
		Title:       "堆内存系统使用量异常增长",
		Description: fmt.Sprintf("检测到堆系统内存增长 %.2f MB，超过阈值 %.2f MB", growth, mm.config.HeapGrowthThresholdMB),
		Evidence:    []string{fmt.Sprintf("当前堆系统内存: %.2f MB", snapshot.HeapSysMB), fmt.Sprintf("基线堆系统内存: %.2f MB", mm.baseline.HeapSysMB)},
		Snapshot:    snapshot,
		Baseline:    *mm.baseline,
		Timestamp:   time.Now(),
		Suggestions: []string{"检查内存分配模式", "调整GOGC参数", "分析内存热点"},
		MetricValues: map[string]float64{
			"growth_mb":   growth,
			"threshold":   mm.config.HeapGrowthThresholdMB,
			"current_mb":  snapshot.HeapSysMB,
			"baseline_mb": mm.baseline.HeapSysMB,
		},
	}
}

// createGCPressureAlert 创建GC压力告警
func (mm *MemoryMonitor) createGCPressureAlert(snapshot MemorySnapshot) MemoryAlert {
	return MemoryAlert{
		ID:          fmt.Sprintf("gc_pressure_%d", time.Now().Unix()),
		Type:        "gc_pressure",
		Severity:    mm.calculateAlertSeverity(snapshot.GCPressureIndicator, mm.config.GCPressureThreshold),
		Title:       "GC压力过高",
		Description: fmt.Sprintf("检测到GC压力指示器 %.2f，超过阈值 %.2f", snapshot.GCPressureIndicator, mm.config.GCPressureThreshold),
		Evidence:    []string{fmt.Sprintf("GC次数: %d", snapshot.GCCount), fmt.Sprintf("GC总暂停时间: %.2f ms", snapshot.GCPauseTotalMs)},
		Snapshot:    snapshot,
		Baseline:    *mm.baseline,
		Timestamp:   time.Now(),
		Suggestions: []string{"调整GOGC参数", "减少内存分配", "优化数据结构"},
		MetricValues: map[string]float64{
			"pressure":  snapshot.GCPressureIndicator,
			"threshold": mm.config.GCPressureThreshold,
			"gc_count":  float64(snapshot.GCCount),
		},
	}
}

// createFragmentationAlert 创建内存碎片化告警
func (mm *MemoryMonitor) createFragmentationAlert(snapshot MemorySnapshot) MemoryAlert {
	return MemoryAlert{
		ID:          fmt.Sprintf("fragmentation_%d", time.Now().Unix()),
		Type:        "fragmentation",
		Severity:    mm.calculateAlertSeverity(snapshot.FragmentationRatio, mm.config.FragmentationThreshold),
		Title:       "内存碎片化严重",
		Description: fmt.Sprintf("检测到内存碎片化比率 %.2f，超过阈值 %.2f", snapshot.FragmentationRatio, mm.config.FragmentationThreshold),
		Evidence:    []string{fmt.Sprintf("堆系统内存: %.2f MB", snapshot.HeapSysMB), fmt.Sprintf("堆使用内存: %.2f MB", snapshot.HeapInuseMB)},
		Snapshot:    snapshot,
		Baseline:    *mm.baseline,
		Timestamp:   time.Now(),
		Suggestions: []string{"执行GC", "调整内存分配策略", "考虑内存预分配"},
		MetricValues: map[string]float64{
			"fragmentation": snapshot.FragmentationRatio,
			"threshold":     mm.config.FragmentationThreshold,
			"heap_sys":      snapshot.HeapSysMB,
			"heap_inuse":    snapshot.HeapInuseMB,
		},
	}
}

// calculateAlertSeverity 计算告警严重程度
func (mm *MemoryMonitor) calculateAlertSeverity(value, threshold float64) string {
	ratio := value / threshold
	switch {
	case ratio >= 3.0:
		return "critical"
	case ratio >= 2.0:
		return "high"
	case ratio >= 1.5:
		return "medium"
	default:
		return "low"
	}
}

// safeSendAlert 安全发送告警
func (mm *MemoryMonitor) safeSendAlert(alert MemoryAlert) {
	if atomic.LoadInt32(&mm.channelsClosed) == 1 {
		return
	}

	select {
	case mm.alertChan <- alert:
	default:
		mm.logger.Warn("告警通道已满，丢弃告警", zap.String("alert_id", alert.ID))
	}
}

// safeSendOptimizationSuggestion 安全发送优化建议
func (mm *MemoryMonitor) safeSendOptimizationSuggestion(suggestion OptimizationSuggestion) {
	if atomic.LoadInt32(&mm.channelsClosed) == 1 {
		return
	}

	select {
	case mm.optimizationChan <- suggestion:
	default:
		mm.logger.Warn("优化建议通道已满，丢弃建议", zap.String("suggestion_id", suggestion.ID))
	}
}

// safeSendError 安全发送错误
func (mm *MemoryMonitor) safeSendError(err error) {
	if atomic.LoadInt32(&mm.channelsClosed) == 1 {
		return
	}

	select {
	case mm.errorChan <- err:
	default:
		mm.logger.Error("错误通道已满，丢弃错误", zap.Error(err))
	}
}

// GetAlertChannel 获取告警通道
func (mm *MemoryMonitor) GetAlertChannel() <-chan MemoryAlert {
	return mm.alertChan
}

// GetOptimizationChannel 获取优化建议通道
func (mm *MemoryMonitor) GetOptimizationChannel() <-chan OptimizationSuggestion {
	return mm.optimizationChan
}

// GetErrorChannel 获取错误通道
func (mm *MemoryMonitor) GetErrorChannel() <-chan error {
	return mm.errorChan
}

// GetCurrentSnapshot 获取当前内存快照
func (mm *MemoryMonitor) GetCurrentSnapshot() (*MemorySnapshot, error) {
	snapshot, err := mm.captureMemorySnapshot()
	return &snapshot, err
}

// GetBaseline 获取内存基线
func (mm *MemoryMonitor) GetBaseline() *MemoryBaseline {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.baseline
}

// GetSnapshots 获取历史快照
func (mm *MemoryMonitor) GetSnapshots() []MemorySnapshot {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	snapshots := make([]MemorySnapshot, len(mm.snapshots))
	copy(snapshots, mm.snapshots)
	return snapshots
}

// IsRunning 检查是否正在运行
func (mm *MemoryMonitor) IsRunning() bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.isRunning
}

// ForceOptimization 强制执行优化
func (mm *MemoryMonitor) ForceOptimization() error {
	if !mm.IsRunning() {
		return fmt.Errorf("内存监控器未运行")
	}

	go mm.performOptimization()
	return nil
}

// CollectProfile 收集性能剖析数据
func (pc *ProfileCollector) CollectProfile(profileType string) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	var buf bytes.Buffer
	var err error

	switch profileType {
	case "heap":
		err = pprof.WriteHeapProfile(&buf)
	case "goroutine":
		profile := pprof.Lookup("goroutine")
		if profile != nil {
			err = profile.WriteTo(&buf, 0)
		}
	case "allocs":
		profile := pprof.Lookup("allocs")
		if profile != nil {
			err = profile.WriteTo(&buf, 0)
		}
	default:
		return fmt.Errorf("不支持的profile类型: %s", profileType)
	}

	if err != nil {
		return fmt.Errorf("收集%s profile失败: %w", profileType, err)
	}

	// 生成摘要
	summary := fmt.Sprintf("%s profile collected at %s, size: %d bytes",
		profileType, time.Now().Format(time.RFC3339), buf.Len())

	profileData := &ProfileData{
		Type:       profileType,
		Timestamp:  time.Now(),
		Data:       buf.Bytes(),
		Summary:    summary,
		TopEntries: pc.extractTopEntries(buf.Bytes(), profileType),
	}

	// 存储profile数据
	key := fmt.Sprintf("%s_%d", profileType, time.Now().Unix())
	pc.profiles[key] = profileData

	// 限制存储的profile数量
	if len(pc.profiles) > pc.maxProfiles {
		pc.cleanupOldProfiles()
	}

	pc.logger.Debug("成功收集profile",
		zap.String("type", profileType),
		zap.String("key", key),
		zap.Int("data_size", len(profileData.Data)))

	return nil
}

// extractTopEntries 提取顶部条目
func (pc *ProfileCollector) extractTopEntries(data []byte, profileType string) []string {
	// 简化实现，实际应该解析pprof数据
	entries := make([]string, 0, pc.collectionDepth)

	switch profileType {
	case "heap":
		entries = append(entries, "main.allocateMemory: 50MB")
		entries = append(entries, "runtime.mallocgc: 30MB")
	case "goroutine":
		entries = append(entries, "main.worker: 100 goroutines")
		entries = append(entries, "net/http.(*conn).serve: 50 goroutines")
	}

	return entries
}

// cleanupOldProfiles 清理旧的profiles
func (pc *ProfileCollector) cleanupOldProfiles() {
	// 按时间戳排序并删除最旧的
	type profileEntry struct {
		key  string
		time time.Time
	}

	var entries []profileEntry
	for key, profile := range pc.profiles {
		entries = append(entries, profileEntry{key: key, time: profile.Timestamp})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].time.Before(entries[j].time)
	})

	// 删除超出限制的旧profiles
	deleteCount := len(entries) - pc.maxProfiles/2
	for i := 0; i < deleteCount; i++ {
		delete(pc.profiles, entries[i].key)
	}
}

// AnalyzeOptimizationOpportunities 分析优化机会
func (mo *MemoryOptimizer) AnalyzeOptimizationOpportunities(snapshot MemorySnapshot, baseline *MemoryBaseline) []OptimizationSuggestion {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	var suggestions []OptimizationSuggestion

	// 分析内存增长
	if baseline != nil && snapshot.HeapAllocMB > baseline.HeapAllocMB+mo.config.AutoGCTriggerThresholdMB {
		suggestion := OptimizationSuggestion{
			ID:          fmt.Sprintf("gc_optimization_%d", time.Now().Unix()),
			Type:        "gc_optimization",
			Priority:    "high",
			Title:       "建议执行垃圾回收",
			Description: fmt.Sprintf("内存使用量从基线的 %.2f MB 增长到 %.2f MB，建议执行GC", baseline.HeapAllocMB, snapshot.HeapAllocMB),
			Actions: []MemoryOptimizationAction{
				{
					Action:      "trigger_gc",
					Description: "执行强制垃圾回收",
					Parameters:  map[string]interface{}{"force": true},
					AutoExecute: true,
				},
			},
			Impact: OptimizationImpact{
				EstimatedMemorySavingMB: (snapshot.HeapAllocMB - baseline.HeapAllocMB) * 0.3,
				EstimatedCPUSaving:      0.0,
				EstimatedGCImprovement:  20.0,
				RiskLevel:               "low",
			},
			Timestamp: time.Now(),
			Context: map[string]interface{}{
				"current_heap_mb":  snapshot.HeapAllocMB,
				"baseline_heap_mb": baseline.HeapAllocMB,
				"growth_mb":        snapshot.HeapAllocMB - baseline.HeapAllocMB,
			},
		}
		suggestions = append(suggestions, suggestion)
	}

	// 分析内存碎片化
	if snapshot.FragmentationRatio > mo.config.FragmentationThreshold {
		suggestion := OptimizationSuggestion{
			ID:          fmt.Sprintf("defrag_optimization_%d", time.Now().Unix()),
			Type:        "defragmentation",
			Priority:    "medium",
			Title:       "建议整理内存碎片",
			Description: fmt.Sprintf("内存碎片化比率 %.2f 超过阈值 %.2f", snapshot.FragmentationRatio, mo.config.FragmentationThreshold),
			Actions: []MemoryOptimizationAction{
				{
					Action:      "memory_defrag",
					Description: "执行内存碎片整理",
					Parameters:  map[string]interface{}{"aggressive": false},
					AutoExecute: false,
				},
			},
			Impact: OptimizationImpact{
				EstimatedMemorySavingMB: snapshot.HeapSysMB * 0.1,
				EstimatedCPUSaving:      5.0,
				EstimatedGCImprovement:  15.0,
				RiskLevel:               "medium",
			},
			Timestamp: time.Now(),
			Context: map[string]interface{}{
				"fragmentation_ratio": snapshot.FragmentationRatio,
				"threshold":           mo.config.FragmentationThreshold,
			},
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

// ExecuteAutoOptimizations 执行自动优化
func (mo *MemoryOptimizer) ExecuteAutoOptimizations(suggestions []OptimizationSuggestion) {
	mo.mu.Lock()
	defer mo.mu.Unlock()

	for _, suggestion := range suggestions {
		for i, action := range suggestion.Actions {
			if action.AutoExecute && !action.Executed {
				mo.executeOptimizationAction(action)
				suggestion.Actions[i].Executed = true
				mo.optimizationStats[action.Action] = mo.optimizationStats[action.Action] + 1
			}
		}
	}
}

// executeOptimizationAction 执行优化操作
func (mo *MemoryOptimizer) executeOptimizationAction(action MemoryOptimizationAction) {
	switch action.Action {
	case "trigger_gc":
		mo.logger.Info("执行强制垃圾回收")
		runtime.GC()
		atomic.AddUint64(&mo.gcCounter, 1)
		mo.lastGCTime = time.Now()

	case "memory_defrag":
		mo.logger.Info("执行内存碎片整理")
		// 内存碎片整理的实现
		runtime.GC()
		runtime.GC() // 连续两次GC有助于内存整理

	default:
		mo.logger.Warn("未知的优化操作", zap.String("action", action.Action))
	}
}
