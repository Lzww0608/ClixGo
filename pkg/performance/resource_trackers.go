/*
* @Author: Lzww0608
* @Date: 2025-06-04 10:05:00
* @LastEditors: Lzww0608
* @LastEditTime: =2025-6-4 12:47:25
* @Description: 资源追踪器实现，用于支持内存泄漏检测
 */

package performance

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// GoroutineTracker goroutine追踪器
// 追踪goroutine的创建、销毁和状态变化
type GoroutineTracker struct {
	mu             sync.RWMutex
	startTime      time.Time
	initialCount   int
	maxCount       int
	totalCreated   int64
	lastCheckTime  time.Time
	lastCheckCount int
}

// NewGoroutineTracker 创建新的goroutine追踪器
func NewGoroutineTracker() *GoroutineTracker {
	return &GoroutineTracker{
		startTime:      time.Now(),
		initialCount:   runtime.NumGoroutine(),
		maxCount:       runtime.NumGoroutine(),
		lastCheckTime:  time.Now(),
		lastCheckCount: runtime.NumGoroutine(),
	}
}

// UpdateMetrics 更新指标
func (gt *GoroutineTracker) UpdateMetrics() {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	current := runtime.NumGoroutine()

	// 更新最大值
	if current > gt.maxCount {
		gt.maxCount = current
	}

	// 估算创建的goroutine总数（简化算法）
	if current > gt.lastCheckCount {
		atomic.AddInt64(&gt.totalCreated, int64(current-gt.lastCheckCount))
	}

	gt.lastCheckTime = time.Now()
	gt.lastCheckCount = current
}

// GetStats 获取统计信息
func (gt *GoroutineTracker) GetStats() map[string]interface{} {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	current := runtime.NumGoroutine()
	return map[string]interface{}{
		"current":       current,
		"initial":       gt.initialCount,
		"max":           gt.maxCount,
		"growth":        current - gt.initialCount,
		"total_created": atomic.LoadInt64(&gt.totalCreated),
		"uptime":        time.Since(gt.startTime),
	}
}

// MemoryTracker 内存追踪器
// 追踪内存分配、释放和使用模式
type MemoryTracker struct {
	mu           sync.RWMutex
	startTime    time.Time
	initialStats runtime.MemStats
	maxHeapAlloc uint64
	maxHeapSys   uint64
	gcCount      uint32
	lastGCPause  time.Duration
}

// NewMemoryTracker 创建新的内存追踪器
func NewMemoryTracker() *MemoryTracker {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &MemoryTracker{
		startTime:    time.Now(),
		initialStats: memStats,
		maxHeapAlloc: memStats.HeapAlloc,
		maxHeapSys:   memStats.HeapSys,
		gcCount:      memStats.NumGC,
	}
}

// UpdateMetrics 更新指标
func (mt *MemoryTracker) UpdateMetrics() {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 更新最大值
	if memStats.HeapAlloc > mt.maxHeapAlloc {
		mt.maxHeapAlloc = memStats.HeapAlloc
	}
	if memStats.HeapSys > mt.maxHeapSys {
		mt.maxHeapSys = memStats.HeapSys
	}

	// 更新GC信息
	if memStats.NumGC > mt.gcCount {
		mt.gcCount = memStats.NumGC
		if memStats.NumGC > 0 {
			mt.lastGCPause = time.Duration(memStats.PauseNs[(memStats.NumGC+255)%256])
		}
	}
}

// GetStats 获取统计信息
func (mt *MemoryTracker) GetStats() map[string]interface{} {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"heap_alloc_mb":     float64(memStats.HeapAlloc) / 1024 / 1024,
		"heap_sys_mb":       float64(memStats.HeapSys) / 1024 / 1024,
		"max_heap_alloc_mb": float64(mt.maxHeapAlloc) / 1024 / 1024,
		"max_heap_sys_mb":   float64(mt.maxHeapSys) / 1024 / 1024,
		"initial_heap_mb":   float64(mt.initialStats.HeapAlloc) / 1024 / 1024,
		"heap_growth_mb":    float64(memStats.HeapAlloc-mt.initialStats.HeapAlloc) / 1024 / 1024,
		"gc_count":          memStats.NumGC,
		"gc_count_growth":   memStats.NumGC - mt.initialStats.NumGC,
		"last_gc_pause_ms":  mt.lastGCPause.Nanoseconds() / 1e6,
		"total_alloc_mb":    float64(memStats.TotalAlloc) / 1024 / 1024,
		"stack_sys_mb":      float64(memStats.StackSys) / 1024 / 1024,
		"uptime":            time.Since(mt.startTime),
	}
}

// TimerTracker 定时器追踪器
// 追踪Timer和Ticker的创建和销毁
type TimerTracker struct {
	mu           sync.RWMutex
	activeTimers int32
	totalCreated int64
	maxActive    int32
	startTime    time.Time
}

// NewTimerTracker 创建新的定时器追踪器
func NewTimerTracker() *TimerTracker {
	return &TimerTracker{
		startTime: time.Now(),
	}
}

// RecordTimerCreated 记录定时器创建
func (tt *TimerTracker) RecordTimerCreated() {
	newActive := atomic.AddInt32(&tt.activeTimers, 1)
	atomic.AddInt64(&tt.totalCreated, 1)

	// 更新最大活跃数
	tt.mu.Lock()
	if newActive > tt.maxActive {
		tt.maxActive = newActive
	}
	tt.mu.Unlock()
}

// RecordTimerStopped 记录定时器停止
func (tt *TimerTracker) RecordTimerStopped() {
	atomic.AddInt32(&tt.activeTimers, -1)
}

// GetActiveTimerCount 获取活跃定时器数量
func (tt *TimerTracker) GetActiveTimerCount() int {
	return int(atomic.LoadInt32(&tt.activeTimers))
}

// GetStats 获取统计信息
func (tt *TimerTracker) GetStats() map[string]interface{} {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	active := atomic.LoadInt32(&tt.activeTimers)
	total := atomic.LoadInt64(&tt.totalCreated)

	return map[string]interface{}{
		"active":        active,
		"max_active":    tt.maxActive,
		"total_created": total,
		"uptime":        time.Since(tt.startTime),
	}
}

// UpdateMetrics 更新指标（定期调用以刷新统计）
func (tt *TimerTracker) UpdateMetrics() {
	// 定时器追踪主要依赖主动记录，这里预留接口
	// 可以在这里添加一些启发式检测逻辑
}
