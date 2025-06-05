/*
* @Author: Lzww0608
* @Date: 2025-06-04 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-5 20:34:10
* @Description: 内存泄漏检测器核心实现，提供runtime级别的泄漏监控
 */

package performance

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" // 导入pprof HTTP handlers
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// MemoryLeakDetector 内存泄漏检测器
// 提供goroutine、内存分配、定时器等资源的泄漏检测
type MemoryLeakDetector struct {
	mu               sync.RWMutex             // 读写锁保护并发访问
	isRunning        bool                     // 检测器运行状态
	ctx              context.Context          // 上下文控制
	cancel           context.CancelFunc       // 取消函数
	config           MemoryLeakDetectorConfig // 检测器配置
	logger           *zap.Logger              // 日志记录器
	baseline         *ResourceBaseline        // 资源基线
	snapshots        []ResourceSnapshot       // 历史快照
	alertChan        chan MemoryLeakAlert     // 告警通道
	leakDetectedChan chan LeakDetectionResult // 泄漏检测结果通道
	errorChan        chan error               // 错误通道
	activeGoroutines sync.WaitGroup           // 活跃goroutine计数
	shutdownOnce     sync.Once                // 确保只关闭一次
	channelsClosed   int32                    // 通道关闭标志

	// 资源追踪器
	goroutineTracker *GoroutineTracker // goroutine追踪器
	memoryTracker    *MemoryTracker    // 内存追踪器
	timerTracker     *TimerTracker     // 定时器追踪器
}

// MemoryLeakDetectorConfig 内存泄漏检测器配置
type MemoryLeakDetectorConfig struct {
	CheckInterval      time.Duration `json:"check_interval"`       // 检查间隔
	BaselineWarmupTime time.Duration `json:"baseline_warmup_time"` // 基线预热时间
	MaxSnapshots       int           `json:"max_snapshots"`        // 最大快照数量
	EnablePprof        bool          `json:"enable_pprof"`         // 启用pprof
	PprofPort          int           `json:"pprof_port"`           // pprof端口

	// 告警阈值
	GoroutineGrowthThreshold int     `json:"goroutine_growth_threshold"` // goroutine增长阈值
	MemoryGrowthThresholdMB  float64 `json:"memory_growth_threshold_mb"` // 内存增长阈值(MB)
	HeapGrowthThresholdMB    float64 `json:"heap_growth_threshold_mb"`   // 堆内存增长阈值(MB)
	GCFrequencyThreshold     int     `json:"gc_frequency_threshold"`     // GC频率阈值
	TimerGrowthThreshold     int     `json:"timer_growth_threshold"`     // 定时器增长阈值

	// 检测灵敏度
	ConsecutiveFailuresThreshold int           `json:"consecutive_failures_threshold"` // 连续失败阈值
	StabilityCheckDuration       time.Duration `json:"stability_check_duration"`       // 稳定性检查持续时间
}

// ResourceBaseline 资源基线
type ResourceBaseline struct {
	Timestamp      time.Time `json:"timestamp"`
	GoroutineCount int       `json:"goroutine_count"`
	HeapAllocMB    float64   `json:"heap_alloc_mb"`
	HeapSysMB      float64   `json:"heap_sys_mb"`
	StackSysMB     float64   `json:"stack_sys_mb"`
	GCCount        uint32    `json:"gc_count"`
	TimerCount     int       `json:"timer_count"`
	ActiveCGoCalls int64     `json:"active_cgo_calls"`
}

// ResourceSnapshot 资源快照
type ResourceSnapshot struct {
	Timestamp           time.Time                `json:"timestamp"`
	GoroutineCount      int                      `json:"goroutine_count"`
	GoroutineProfiles   []GoroutineProfile       `json:"goroutine_profiles"`
	HeapAllocMB         float64                  `json:"heap_alloc_mb"`
	HeapSysMB           float64                  `json:"heap_sys_mb"`
	StackSysMB          float64                  `json:"stack_sys_mb"`
	GCCount             uint32                   `json:"gc_count"`
	GCPauseTotalMs      float64                  `json:"gc_pause_total_ms"`
	TimerCount          int                      `json:"timer_count"`
	ActiveCGoCalls      int64                    `json:"active_cgo_calls"`
	TopMemoryAllocators []MemoryAllocatorProfile `json:"top_memory_allocators"`
	SuspiciousPatterns  []SuspiciousPattern      `json:"suspicious_patterns"`
}

// GoroutineProfile goroutine分析
type GoroutineProfile struct {
	ID       uint64        `json:"id"`
	State    string        `json:"state"`
	Function string        `json:"function"`
	Stack    string        `json:"stack"`
	Runtime  time.Duration `json:"runtime"`
}

// MemoryAllocatorProfile 内存分配器分析
type MemoryAllocatorProfile struct {
	Function string  `json:"function"`
	AllocMB  float64 `json:"alloc_mb"`
	Count    int64   `json:"count"`
	Stack    string  `json:"stack"`
}

// SuspiciousPattern 可疑模式
type SuspiciousPattern struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	Evidence    string    `json:"evidence"`
	Timestamp   time.Time `json:"timestamp"`
}

// MemoryLeakAlert 内存泄漏告警
type MemoryLeakAlert struct {
	ID          string           `json:"id"`
	Type        string           `json:"type"`
	Severity    string           `json:"severity"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Evidence    []string         `json:"evidence"`
	Snapshot    ResourceSnapshot `json:"snapshot"`
	Baseline    ResourceBaseline `json:"baseline"`
	Timestamp   time.Time        `json:"timestamp"`
	Suggestions []string         `json:"suggestions"`
}

// LeakDetectionResult 泄漏检测结果
type LeakDetectionResult struct {
	HasLeak           bool             `json:"has_leak"`
	LeakType          string           `json:"leak_type"`
	Confidence        float64          `json:"confidence"`
	Description       string           `json:"description"`
	AffectedResources []string         `json:"affected_resources"`
	Snapshot          ResourceSnapshot `json:"snapshot"`
	Timestamp         time.Time        `json:"timestamp"`
}

// NewMemoryLeakDetector 创建新的内存泄漏检测器
func NewMemoryLeakDetector(config MemoryLeakDetectorConfig, logger *zap.Logger) *MemoryLeakDetector {
	ctx, cancel := context.WithCancel(context.Background())

	// 设置默认配置
	if config.CheckInterval == 0 {
		config.CheckInterval = 30 * time.Second
	}
	if config.BaselineWarmupTime == 0 {
		config.BaselineWarmupTime = 60 * time.Second
	}
	if config.MaxSnapshots == 0 {
		config.MaxSnapshots = 100
	}
	if config.GoroutineGrowthThreshold == 0 {
		config.GoroutineGrowthThreshold = 50
	}
	if config.MemoryGrowthThresholdMB == 0 {
		config.MemoryGrowthThresholdMB = 100.0
	}
	if config.HeapGrowthThresholdMB == 0 {
		config.HeapGrowthThresholdMB = 50.0
	}
	if config.ConsecutiveFailuresThreshold == 0 {
		config.ConsecutiveFailuresThreshold = 3
	}
	if config.StabilityCheckDuration == 0 {
		config.StabilityCheckDuration = 5 * time.Minute
	}

	detector := &MemoryLeakDetector{
		ctx:              ctx,
		cancel:           cancel,
		config:           config,
		logger:           logger,
		snapshots:        make([]ResourceSnapshot, 0, config.MaxSnapshots),
		alertChan:        make(chan MemoryLeakAlert, 50),
		leakDetectedChan: make(chan LeakDetectionResult, 20),
		errorChan:        make(chan error, 10),
		goroutineTracker: NewGoroutineTracker(),
		memoryTracker:    NewMemoryTracker(),
		timerTracker:     NewTimerTracker(),
	}

	return detector
}

// Start 启动内存泄漏检测器
func (mld *MemoryLeakDetector) Start() error {
	mld.mu.Lock()
	defer mld.mu.Unlock()

	if mld.isRunning {
		return fmt.Errorf("内存泄漏检测器已在运行")
	}

	mld.isRunning = true

	// 启动pprof服务（如果启用）
	if mld.config.EnablePprof {
		go mld.startPprofServer()
	}

	// 等待基线预热
	go func() {
		time.Sleep(mld.config.BaselineWarmupTime)
		mld.establishBaseline()
	}()

	// 启动检测循环
	mld.activeGoroutines.Add(1)
	go func() {
		defer mld.activeGoroutines.Done()
		mld.detectionLoop()
	}()

	mld.logger.Info("内存泄漏检测器已启动",
		zap.Duration("check_interval", mld.config.CheckInterval),
		zap.Duration("baseline_warmup", mld.config.BaselineWarmupTime))

	return nil
}

// Stop 停止内存泄漏检测器
func (mld *MemoryLeakDetector) Stop() error {
	mld.shutdownOnce.Do(func() {
		atomic.StoreInt32(&mld.channelsClosed, 1)

		mld.mu.Lock()
		if !mld.isRunning {
			mld.mu.Unlock()
			return
		}
		mld.isRunning = false
		mld.cancel()
		mld.mu.Unlock()

		// 等待所有goroutine退出
		done := make(chan struct{})
		go func() {
			mld.activeGoroutines.Wait()
			close(done)
		}()

		select {
		case <-done:
			// 正常退出
		case <-time.After(10 * time.Second):
			// 超时强制退出
			mld.logger.Warn("内存泄漏检测器停止超时")
		}

		mld.logger.Info("内存泄漏检测器已停止")
	})

	return nil
}

// establishBaseline 建立资源基线
func (mld *MemoryLeakDetector) establishBaseline() {
	// 强制执行GC以获得稳定的基线
	runtime.GC()
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	baseline := &ResourceBaseline{
		Timestamp:      time.Now(),
		GoroutineCount: runtime.NumGoroutine(),
		HeapAllocMB:    float64(memStats.HeapAlloc) / 1024 / 1024,
		HeapSysMB:      float64(memStats.HeapSys) / 1024 / 1024,
		StackSysMB:     float64(memStats.StackSys) / 1024 / 1024,
		GCCount:        memStats.NumGC,
		TimerCount:     mld.timerTracker.GetActiveTimerCount(),
		ActiveCGoCalls: runtime.NumCgoCall(),
	}

	mld.mu.Lock()
	mld.baseline = baseline
	mld.mu.Unlock()

	mld.logger.Info("资源基线已建立",
		zap.Int("goroutines", baseline.GoroutineCount),
		zap.Float64("heap_mb", baseline.HeapAllocMB),
		zap.Uint32("gc_count", baseline.GCCount))
}

// detectionLoop 检测循环
func (mld *MemoryLeakDetector) detectionLoop() {
	ticker := time.NewTicker(mld.config.CheckInterval)
	defer ticker.Stop()

	consecutiveFailures := 0

	for {
		select {
		case <-mld.ctx.Done():
			return
		case <-ticker.C:
			snapshot, err := mld.captureResourceSnapshot()
			if err != nil {
				consecutiveFailures++
				mld.safeSendError(errors.Wrap(err, "捕获资源快照失败"))

				if consecutiveFailures >= mld.config.ConsecutiveFailuresThreshold {
					mld.safeSendError(fmt.Errorf("连续%d次快照失败", consecutiveFailures))
				}
				continue
			}

			consecutiveFailures = 0
			mld.addSnapshot(snapshot)

			// 执行泄漏检测
			result := mld.detectMemoryLeaks(snapshot)
			if result.HasLeak {
				mld.safeSendLeakResult(result)
				mld.generateAlert(snapshot, result)
			}
		}
	}
}

// captureResourceSnapshot 捕获资源快照
func (mld *MemoryLeakDetector) captureResourceSnapshot() (ResourceSnapshot, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	snapshot := ResourceSnapshot{
		Timestamp:      time.Now(),
		GoroutineCount: runtime.NumGoroutine(),
		HeapAllocMB:    float64(memStats.HeapAlloc) / 1024 / 1024,
		HeapSysMB:      float64(memStats.HeapSys) / 1024 / 1024,
		StackSysMB:     float64(memStats.StackSys) / 1024 / 1024,
		GCCount:        memStats.NumGC,
		GCPauseTotalMs: float64(memStats.PauseTotalNs) / 1e6,
		TimerCount:     mld.timerTracker.GetActiveTimerCount(),
		ActiveCGoCalls: runtime.NumCgoCall(),
	}

	// 收集goroutine分析（采样，避免性能影响）
	if mld.shouldCollectGoroutineProfiles() {
		profiles, err := mld.collectGoroutineProfiles()
		if err != nil {
			mld.logger.Warn("收集goroutine分析失败", zap.Error(err))
		} else {
			snapshot.GoroutineProfiles = profiles
		}
	}

	// 收集内存分配器分析（采样）
	if mld.shouldCollectMemoryProfiles() {
		allocators, err := mld.collectMemoryAllocatorProfiles()
		if err != nil {
			mld.logger.Warn("收集内存分配器分析失败", zap.Error(err))
		} else {
			snapshot.TopMemoryAllocators = allocators
		}
	}

	// 检测可疑模式
	patterns := mld.detectSuspiciousPatterns(snapshot)
	snapshot.SuspiciousPatterns = patterns

	return snapshot, nil
}

// shouldCollectGoroutineProfiles 判断是否应该收集goroutine分析
func (mld *MemoryLeakDetector) shouldCollectGoroutineProfiles() bool {
	// 每5分钟收集一次详细分析，或者当检测到goroutine增长时
	mld.mu.RLock()
	defer mld.mu.RUnlock()

	if mld.baseline == nil {
		return false
	}

	currentCount := runtime.NumGoroutine()
	growth := currentCount - mld.baseline.GoroutineCount

	return growth > mld.config.GoroutineGrowthThreshold/2 ||
		time.Now().Unix()%300 == 0 // 每5分钟
}

// shouldCollectMemoryProfiles 判断是否应该收集内存分析
func (mld *MemoryLeakDetector) shouldCollectMemoryProfiles() bool {
	// 每10分钟收集一次详细分析，或者当检测到内存增长时
	mld.mu.RLock()
	defer mld.mu.RUnlock()

	if mld.baseline == nil {
		return false
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	currentHeapMB := float64(memStats.HeapAlloc) / 1024 / 1024
	growth := currentHeapMB - mld.baseline.HeapAllocMB

	return growth > mld.config.HeapGrowthThresholdMB/2 ||
		time.Now().Unix()%600 == 0 // 每10分钟
}

// collectGoroutineProfiles 收集goroutine分析
func (mld *MemoryLeakDetector) collectGoroutineProfiles() ([]GoroutineProfile, error) {
	profile := pprof.Lookup("goroutine")
	if profile == nil {
		return nil, fmt.Errorf("无法获取goroutine profile")
	}

	var buf strings.Builder
	if err := profile.WriteTo(&buf, 2); err != nil {
		return nil, err
	}

	// 解析profile内容（简化版本）
	profiles := make([]GoroutineProfile, 0)
	lines := strings.Split(buf.String(), "\n")

	for i, line := range lines {
		if strings.Contains(line, "goroutine") && strings.Contains(line, "[") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				profile := GoroutineProfile{
					State:    parts[2],
					Function: "unknown",
					Stack:    line,
				}

				// 获取下一行的函数信息
				if i+1 < len(lines) {
					profile.Function = strings.TrimSpace(lines[i+1])
				}

				profiles = append(profiles, profile)
			}
		}
	}

	// 按状态排序并限制数量
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].State < profiles[j].State
	})

	if len(profiles) > 50 {
		profiles = profiles[:50]
	}

	return profiles, nil
}

// collectMemoryAllocatorProfiles 收集内存分配器分析
func (mld *MemoryLeakDetector) collectMemoryAllocatorProfiles() ([]MemoryAllocatorProfile, error) {
	profile := pprof.Lookup("heap")
	if profile == nil {
		return nil, fmt.Errorf("无法获取heap profile")
	}

	var buf strings.Builder
	if err := profile.WriteTo(&buf, 0); err != nil {
		return nil, err
	}

	// 简化的heap profile解析
	allocators := make([]MemoryAllocatorProfile, 0)
	lines := strings.Split(buf.String(), "\n")

	for _, line := range lines {
		if strings.Contains(line, "#") && strings.Contains(line, "bytes") {
			allocator := MemoryAllocatorProfile{
				Function: line,
				Stack:    line,
			}
			allocators = append(allocators, allocator)
		}
	}

	// 限制数量
	if len(allocators) > 20 {
		allocators = allocators[:20]
	}

	return allocators, nil
}

// detectSuspiciousPatterns 检测可疑模式
func (mld *MemoryLeakDetector) detectSuspiciousPatterns(snapshot ResourceSnapshot) []SuspiciousPattern {
	patterns := make([]SuspiciousPattern, 0)

	mld.mu.RLock()
	baseline := mld.baseline
	mld.mu.RUnlock()

	if baseline == nil {
		return patterns
	}

	// 检测goroutine泄漏模式
	goroutineGrowth := snapshot.GoroutineCount - baseline.GoroutineCount
	if goroutineGrowth > mld.config.GoroutineGrowthThreshold {
		patterns = append(patterns, SuspiciousPattern{
			Type:        "goroutine_leak",
			Description: fmt.Sprintf("Goroutine数量异常增长 %d -> %d (+%d)", baseline.GoroutineCount, snapshot.GoroutineCount, goroutineGrowth),
			Severity:    "high",
			Evidence:    fmt.Sprintf("增长超过阈值 %d", mld.config.GoroutineGrowthThreshold),
			Timestamp:   snapshot.Timestamp,
		})
	}

	// 检测内存泄漏模式
	memoryGrowth := snapshot.HeapAllocMB - baseline.HeapAllocMB
	if memoryGrowth > mld.config.MemoryGrowthThresholdMB {
		patterns = append(patterns, SuspiciousPattern{
			Type:        "memory_leak",
			Description: fmt.Sprintf("堆内存异常增长 %.2fMB -> %.2fMB (+%.2fMB)", baseline.HeapAllocMB, snapshot.HeapAllocMB, memoryGrowth),
			Severity:    "high",
			Evidence:    fmt.Sprintf("增长超过阈值 %.2fMB", mld.config.MemoryGrowthThresholdMB),
			Timestamp:   snapshot.Timestamp,
		})
	}

	// 检测GC异常模式
	gcGrowth := snapshot.GCCount - baseline.GCCount
	if gcGrowth > uint32(mld.config.GCFrequencyThreshold) {
		patterns = append(patterns, SuspiciousPattern{
			Type:        "gc_pressure",
			Description: fmt.Sprintf("GC频率异常 %d -> %d (+%d)", baseline.GCCount, snapshot.GCCount, gcGrowth),
			Severity:    "medium",
			Evidence:    fmt.Sprintf("GC次数增长超过阈值 %d", mld.config.GCFrequencyThreshold),
			Timestamp:   snapshot.Timestamp,
		})
	}

	return patterns
}

// detectMemoryLeaks 检测内存泄漏
func (mld *MemoryLeakDetector) detectMemoryLeaks(snapshot ResourceSnapshot) LeakDetectionResult {
	result := LeakDetectionResult{
		HasLeak:           false,
		Confidence:        0.0,
		AffectedResources: make([]string, 0),
		Snapshot:          snapshot,
		Timestamp:         time.Now(),
	}

	mld.mu.RLock()
	baseline := mld.baseline
	snapshotHistory := make([]ResourceSnapshot, len(mld.snapshots))
	copy(snapshotHistory, mld.snapshots)
	mld.mu.RUnlock()

	if baseline == nil || len(snapshotHistory) < 3 {
		return result
	}

	// 分析趋势
	trends := mld.analyzeTrends(snapshotHistory)

	// 检测各类泄漏
	leakTypes := []string{}
	confidence := 0.0

	// Goroutine泄漏检测
	if trends.GoroutineGrowthRate > 0 && snapshot.GoroutineCount > baseline.GoroutineCount+mld.config.GoroutineGrowthThreshold {
		leakTypes = append(leakTypes, "goroutine")
		confidence += 0.3
		result.AffectedResources = append(result.AffectedResources, "goroutines")
	}

	// 内存泄漏检测
	if trends.MemoryGrowthRate > 0 && snapshot.HeapAllocMB > baseline.HeapAllocMB+mld.config.MemoryGrowthThresholdMB {
		leakTypes = append(leakTypes, "memory")
		confidence += 0.4
		result.AffectedResources = append(result.AffectedResources, "heap_memory")
	}

	// 定时器泄漏检测
	if trends.TimerGrowthRate > 0 && snapshot.TimerCount > baseline.TimerCount+mld.config.TimerGrowthThreshold {
		leakTypes = append(leakTypes, "timer")
		confidence += 0.2
		result.AffectedResources = append(result.AffectedResources, "timers")
	}

	// 可疑模式增加置信度
	for _, pattern := range snapshot.SuspiciousPatterns {
		if pattern.Severity == "high" {
			confidence += 0.1
		}
	}

	if len(leakTypes) > 0 && confidence > 0.5 {
		result.HasLeak = true
		result.LeakType = strings.Join(leakTypes, ",")
		result.Confidence = confidence
		result.Description = fmt.Sprintf("检测到%s泄漏，置信度: %.2f", result.LeakType, confidence)
	}

	return result
}

// TrendAnalysis 趋势分析
type TrendAnalysis struct {
	GoroutineGrowthRate float64
	MemoryGrowthRate    float64
	TimerGrowthRate     float64
	GCPressureRate      float64
}

// analyzeTrends 分析趋势
func (mld *MemoryLeakDetector) analyzeTrends(snapshots []ResourceSnapshot) TrendAnalysis {
	if len(snapshots) < 2 {
		return TrendAnalysis{}
	}

	// 取最近的快照进行趋势分析
	recent := snapshots[len(snapshots)-1]
	older := snapshots[0]

	timeDiff := recent.Timestamp.Sub(older.Timestamp).Seconds()
	if timeDiff <= 0 {
		return TrendAnalysis{}
	}

	return TrendAnalysis{
		GoroutineGrowthRate: float64(recent.GoroutineCount-older.GoroutineCount) / timeDiff,
		MemoryGrowthRate:    (recent.HeapAllocMB - older.HeapAllocMB) / timeDiff,
		TimerGrowthRate:     float64(recent.TimerCount-older.TimerCount) / timeDiff,
		GCPressureRate:      float64(recent.GCCount-older.GCCount) / timeDiff,
	}
}

// generateAlert 生成告警
func (mld *MemoryLeakDetector) generateAlert(snapshot ResourceSnapshot, result LeakDetectionResult) {
	alert := MemoryLeakAlert{
		ID:          fmt.Sprintf("leak_%s_%d", result.LeakType, time.Now().Unix()),
		Type:        result.LeakType,
		Severity:    mld.calculateSeverity(result.Confidence),
		Title:       fmt.Sprintf("检测到%s内存泄漏", result.LeakType),
		Description: result.Description,
		Evidence:    mld.buildEvidence(snapshot, result),
		Snapshot:    snapshot,
		Timestamp:   time.Now(),
		Suggestions: mld.generateSuggestions(result.LeakType),
	}

	mld.mu.RLock()
	if mld.baseline != nil {
		alert.Baseline = *mld.baseline
	}
	mld.mu.RUnlock()

	mld.safeSendAlert(alert)
}

// calculateSeverity 计算严重程度
func (mld *MemoryLeakDetector) calculateSeverity(confidence float64) string {
	if confidence >= 0.8 {
		return "critical"
	} else if confidence >= 0.6 {
		return "high"
	} else if confidence >= 0.4 {
		return "medium"
	}
	return "low"
}

// buildEvidence 构建证据
func (mld *MemoryLeakDetector) buildEvidence(snapshot ResourceSnapshot, result LeakDetectionResult) []string {
	evidence := make([]string, 0)

	mld.mu.RLock()
	baseline := mld.baseline
	mld.mu.RUnlock()

	if baseline != nil {
		evidence = append(evidence, fmt.Sprintf("Goroutine增长: %d -> %d", baseline.GoroutineCount, snapshot.GoroutineCount))
		evidence = append(evidence, fmt.Sprintf("堆内存增长: %.2fMB -> %.2fMB", baseline.HeapAllocMB, snapshot.HeapAllocMB))
		evidence = append(evidence, fmt.Sprintf("GC次数增长: %d -> %d", baseline.GCCount, snapshot.GCCount))
	}

	for _, pattern := range snapshot.SuspiciousPatterns {
		evidence = append(evidence, pattern.Description)
	}

	return evidence
}

// generateSuggestions 生成建议
func (mld *MemoryLeakDetector) generateSuggestions(leakType string) []string {
	suggestions := make([]string, 0)

	if strings.Contains(leakType, "goroutine") {
		suggestions = append(suggestions, "检查是否有未正确关闭的goroutine")
		suggestions = append(suggestions, "确保所有goroutine都有退出机制")
		suggestions = append(suggestions, "使用context.Context控制goroutine生命周期")
	}

	if strings.Contains(leakType, "memory") {
		suggestions = append(suggestions, "检查是否有循环引用")
		suggestions = append(suggestions, "确保大对象及时释放")
		suggestions = append(suggestions, "使用sync.Pool复用对象")
	}

	if strings.Contains(leakType, "timer") {
		suggestions = append(suggestions, "确保所有Timer都调用了Stop()")
		suggestions = append(suggestions, "检查Ticker是否正确停止")
	}

	suggestions = append(suggestions, "使用go tool pprof进行详细分析")
	suggestions = append(suggestions, "启用runtime/trace进行追踪")

	return suggestions
}

// addSnapshot 添加快照
func (mld *MemoryLeakDetector) addSnapshot(snapshot ResourceSnapshot) {
	mld.mu.Lock()
	defer mld.mu.Unlock()

	mld.snapshots = append(mld.snapshots, snapshot)

	// 保持快照数量在限制内
	if len(mld.snapshots) > mld.config.MaxSnapshots {
		// 删除最旧的快照
		copy(mld.snapshots, mld.snapshots[1:])
		mld.snapshots = mld.snapshots[:len(mld.snapshots)-1]
	}
}

// startPprofServer 启动pprof服务器
func (mld *MemoryLeakDetector) startPprofServer() {
	if !mld.config.EnablePprof {
		return
	}

	// 使用net/http/pprof包提供pprof服务
	addr := fmt.Sprintf(":%d", mld.config.PprofPort)

	mld.logger.Info("启动pprof服务器", zap.String("address", addr))

	// 启动HTTP服务器，使用默认的pprof handlers
	// net/http/pprof包会自动注册到默认的ServeMux

	// 启动HTTP服务器
	server := &http.Server{
		Addr:    addr,
		Handler: nil, // 使用默认的ServeMux，pprof已自动注册
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			mld.safeSendError(fmt.Errorf("pprof服务器启动失败: %w", err))
		}
	}()
}

// 安全发送方法
func (mld *MemoryLeakDetector) safeSendAlert(alert MemoryLeakAlert) {
	if atomic.LoadInt32(&mld.channelsClosed) == 1 {
		return
	}

	select {
	case mld.alertChan <- alert:
	case <-mld.ctx.Done():
	default:
		// 通道满，丢弃告警
	}
}

func (mld *MemoryLeakDetector) safeSendLeakResult(result LeakDetectionResult) {
	if atomic.LoadInt32(&mld.channelsClosed) == 1 {
		return
	}

	select {
	case mld.leakDetectedChan <- result:
	case <-mld.ctx.Done():
	default:
		// 通道满，丢弃结果
	}
}

func (mld *MemoryLeakDetector) safeSendError(err error) {
	if atomic.LoadInt32(&mld.channelsClosed) == 1 {
		return
	}

	select {
	case mld.errorChan <- err:
	case <-mld.ctx.Done():
	default:
		// 通道满，丢弃错误
	}
}

// 获取通道的方法
func (mld *MemoryLeakDetector) GetAlertChannel() <-chan MemoryLeakAlert {
	return mld.alertChan
}

func (mld *MemoryLeakDetector) GetLeakDetectedChannel() <-chan LeakDetectionResult {
	return mld.leakDetectedChan
}

func (mld *MemoryLeakDetector) GetErrorChannel() <-chan error {
	return mld.errorChan
}

// GetCurrentSnapshot 获取当前资源快照
func (mld *MemoryLeakDetector) GetCurrentSnapshot() (*ResourceSnapshot, error) {
	if !mld.IsRunning() {
		return nil, fmt.Errorf("检测器未运行")
	}

	snapshot, err := mld.captureResourceSnapshot()
	if err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// GetBaseline 获取资源基线
func (mld *MemoryLeakDetector) GetBaseline() *ResourceBaseline {
	mld.mu.RLock()
	defer mld.mu.RUnlock()

	if mld.baseline == nil {
		return nil
	}

	baseline := *mld.baseline
	return &baseline
}

// GetSnapshots 获取历史快照
func (mld *MemoryLeakDetector) GetSnapshots() []ResourceSnapshot {
	mld.mu.RLock()
	defer mld.mu.RUnlock()

	snapshots := make([]ResourceSnapshot, len(mld.snapshots))
	copy(snapshots, mld.snapshots)
	return snapshots
}

// IsRunning 检查是否运行中
func (mld *MemoryLeakDetector) IsRunning() bool {
	mld.mu.RLock()
	defer mld.mu.RUnlock()
	return mld.isRunning
}

// ForceCheck 强制执行检查
func (mld *MemoryLeakDetector) ForceCheck() (*LeakDetectionResult, error) {
	if !mld.IsRunning() {
		return nil, fmt.Errorf("检测器未运行")
	}

	snapshot, err := mld.captureResourceSnapshot()
	if err != nil {
		return nil, err
	}

	mld.addSnapshot(snapshot)
	result := mld.detectMemoryLeaks(snapshot)

	if result.HasLeak {
		mld.generateAlert(snapshot, result)
	}

	return &result, nil
}
