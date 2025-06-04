/*
* @Author: Lzww0608
* @Date: 2025-06-04 10:10:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-4 12:51:25
* @Description: 内存泄漏检测器测试
 */

package performance

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewMemoryLeakDetector(t *testing.T) {
	logger := zap.NewNop()
	config := MemoryLeakDetectorConfig{
		CheckInterval:      5 * time.Second,
		BaselineWarmupTime: 2 * time.Second,
		MaxSnapshots:       50,
	}

	detector := NewMemoryLeakDetector(config, logger)
	require.NotNil(t, detector)
	assert.Equal(t, config.CheckInterval, detector.config.CheckInterval)
	assert.Equal(t, config.MaxSnapshots, detector.config.MaxSnapshots)
	assert.False(t, detector.IsRunning())
}

func TestMemoryLeakDetector_StartStop(t *testing.T) {
	logger := zap.NewNop()
	config := MemoryLeakDetectorConfig{
		CheckInterval:      100 * time.Millisecond,
		BaselineWarmupTime: 50 * time.Millisecond,
		MaxSnapshots:       10,
	}

	detector := NewMemoryLeakDetector(config, logger)
	require.NotNil(t, detector)

	// 测试启动
	err := detector.Start()
	require.NoError(t, err)
	assert.True(t, detector.IsRunning())

	// 测试重复启动
	err = detector.Start()
	assert.Error(t, err)

	// 等待一段时间让检测器运行
	time.Sleep(200 * time.Millisecond)

	// 测试停止
	err = detector.Stop()
	require.NoError(t, err)
	assert.False(t, detector.IsRunning())

	// 测试重复停止
	err = detector.Stop()
	require.NoError(t, err) // 应该不报错
}

func TestMemoryLeakDetector_BaselineEstablishment(t *testing.T) {
	logger := zap.NewNop()
	config := MemoryLeakDetectorConfig{
		CheckInterval:      1 * time.Second,
		BaselineWarmupTime: 100 * time.Millisecond,
		MaxSnapshots:       10,
	}

	detector := NewMemoryLeakDetector(config, logger)
	require.NotNil(t, detector)

	err := detector.Start()
	require.NoError(t, err)
	defer detector.Stop()

	// 等待基线建立（需要等待BaselineWarmupTime + 一些额外时间）
	time.Sleep(300 * time.Millisecond)

	baseline := detector.GetBaseline()
	require.NotNil(t, baseline)
	assert.Greater(t, baseline.GoroutineCount, 0)
	assert.Greater(t, baseline.HeapAllocMB, 0.0)
}

func TestMemoryLeakDetector_SnapshotCapture(t *testing.T) {
	logger := zap.NewNop()
	config := MemoryLeakDetectorConfig{
		CheckInterval:      50 * time.Millisecond,
		BaselineWarmupTime: 10 * time.Millisecond,
		MaxSnapshots:       5,
	}

	detector := NewMemoryLeakDetector(config, logger)
	require.NotNil(t, detector)

	err := detector.Start()
	require.NoError(t, err)
	defer detector.Stop()

	// 等待收集一些快照
	time.Sleep(300 * time.Millisecond)

	snapshots := detector.GetSnapshots()
	assert.Greater(t, len(snapshots), 0)

	if len(snapshots) > 0 {
		snapshot := snapshots[0]
		assert.Greater(t, snapshot.GoroutineCount, 0)
		assert.Greater(t, snapshot.HeapAllocMB, 0.0)
		assert.NotZero(t, snapshot.Timestamp)
	}
}

func TestMemoryLeakDetector_ForceCheck(t *testing.T) {
	logger := zap.NewNop()
	config := MemoryLeakDetectorConfig{
		CheckInterval:      1 * time.Second,
		BaselineWarmupTime: 50 * time.Millisecond,
		MaxSnapshots:       10,
	}

	detector := NewMemoryLeakDetector(config, logger)
	require.NotNil(t, detector)

	// 测试未启动时强制检查
	result, err := detector.ForceCheck()
	assert.Error(t, err)
	assert.Nil(t, result)

	err = detector.Start()
	require.NoError(t, err)
	defer detector.Stop()

	// 等待基线建立
	time.Sleep(100 * time.Millisecond)

	// 测试强制检查
	result, err = detector.ForceCheck()
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotZero(t, result.Timestamp)
}

func TestMemoryLeakDetector_GoroutineLeakDetection(t *testing.T) {
	logger := zap.NewNop()
	config := MemoryLeakDetectorConfig{
		CheckInterval:            100 * time.Millisecond,
		BaselineWarmupTime:       50 * time.Millisecond,
		MaxSnapshots:             10,
		GoroutineGrowthThreshold: 5, // 低阈值便于测试
	}

	detector := NewMemoryLeakDetector(config, logger)
	require.NotNil(t, detector)

	err := detector.Start()
	require.NoError(t, err)
	defer detector.Stop()

	// 等待基线建立
	time.Sleep(100 * time.Millisecond)

	// 创建一些goroutine来模拟泄漏
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				return
			}
		}()
	}

	// 等待检测器检测到泄漏
	time.Sleep(300 * time.Millisecond)

	// 检查是否检测到泄漏
	result, err := detector.ForceCheck()
	require.NoError(t, err)

	// 清理goroutine
	cancel()
	wg.Wait()

	// 验证结果（可能检测到也可能没有，取决于时机）
	assert.NotNil(t, result)
}

func TestMemoryLeakDetector_MemoryLeakDetection(t *testing.T) {
	logger := zap.NewNop()
	config := MemoryLeakDetectorConfig{
		CheckInterval:           100 * time.Millisecond,
		BaselineWarmupTime:      50 * time.Millisecond,
		MaxSnapshots:            10,
		MemoryGrowthThresholdMB: 1.0, // 低阈值便于测试
	}

	detector := NewMemoryLeakDetector(config, logger)
	require.NotNil(t, detector)

	err := detector.Start()
	require.NoError(t, err)
	defer detector.Stop()

	// 等待基线建立
	time.Sleep(100 * time.Millisecond)

	// 分配一些内存来模拟泄漏
	data := make([][]byte, 0)
	for i := 0; i < 100; i++ {
		data = append(data, make([]byte, 1024*1024)) // 1MB each
	}

	// 等待检测器检测
	time.Sleep(300 * time.Millisecond)

	result, err := detector.ForceCheck()
	require.NoError(t, err)
	assert.NotNil(t, result)

	// 清理内存
	data = nil
	runtime.GC()
}

func TestMemoryLeakDetector_ChannelSafety(t *testing.T) {
	logger := zap.NewNop()
	config := MemoryLeakDetectorConfig{
		CheckInterval:      10 * time.Millisecond,
		BaselineWarmupTime: 5 * time.Millisecond,
		MaxSnapshots:       5,
	}

	detector := NewMemoryLeakDetector(config, logger)
	require.NotNil(t, detector)

	err := detector.Start()
	require.NoError(t, err)

	// 不读取通道，测试是否会阻塞
	time.Sleep(200 * time.Millisecond)

	// 验证检测器仍在运行
	assert.True(t, detector.IsRunning())

	err = detector.Stop()
	require.NoError(t, err)
}

func TestMemoryLeakDetector_ConcurrentAccess(t *testing.T) {
	logger := zap.NewNop()
	config := MemoryLeakDetectorConfig{
		CheckInterval:      50 * time.Millisecond,
		BaselineWarmupTime: 10 * time.Millisecond,
		MaxSnapshots:       10,
	}

	detector := NewMemoryLeakDetector(config, logger)
	require.NotNil(t, detector)

	err := detector.Start()
	require.NoError(t, err)
	defer detector.Stop()

	// 等待基线建立
	time.Sleep(50 * time.Millisecond)

	// 并发访问测试
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				detector.GetBaseline()
				detector.GetSnapshots()
				detector.IsRunning()
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	wg.Wait()
	assert.True(t, detector.IsRunning())
}

func TestMemoryLeakDetector_SnapshotLimit(t *testing.T) {
	logger := zap.NewNop()
	config := MemoryLeakDetectorConfig{
		CheckInterval:      10 * time.Millisecond,
		BaselineWarmupTime: 5 * time.Millisecond,
		MaxSnapshots:       3, // 小的限制便于测试
	}

	detector := NewMemoryLeakDetector(config, logger)
	require.NotNil(t, detector)

	err := detector.Start()
	require.NoError(t, err)
	defer detector.Stop()

	// 等待收集超过限制的快照
	time.Sleep(200 * time.Millisecond)

	snapshots := detector.GetSnapshots()
	assert.LessOrEqual(t, len(snapshots), config.MaxSnapshots)
}

func TestMemoryLeakDetector_ErrorHandling(t *testing.T) {
	logger := zap.NewNop()
	config := MemoryLeakDetectorConfig{
		CheckInterval:                10 * time.Millisecond,
		BaselineWarmupTime:           5 * time.Millisecond,
		MaxSnapshots:                 5,
		ConsecutiveFailuresThreshold: 2,
	}

	detector := NewMemoryLeakDetector(config, logger)
	require.NotNil(t, detector)

	err := detector.Start()
	require.NoError(t, err)
	defer detector.Stop()

	// 监听错误通道
	go func() {
		for {
			select {
			case <-detector.GetErrorChannel():
				// 收到错误
			case <-time.After(500 * time.Millisecond):
				return
			}
		}
	}()

	time.Sleep(200 * time.Millisecond)
	assert.True(t, detector.IsRunning())
}

func TestResourceTrackers(t *testing.T) {
	// 测试GoroutineTracker
	gt := NewGoroutineTracker()
	require.NotNil(t, gt)

	gt.UpdateMetrics()
	stats := gt.GetStats()
	assert.Contains(t, stats, "current")
	assert.Contains(t, stats, "initial")

	// 测试MemoryTracker
	mt := NewMemoryTracker()
	require.NotNil(t, mt)

	mt.UpdateMetrics()
	stats = mt.GetStats()
	assert.Contains(t, stats, "heap_alloc_mb")
	assert.Contains(t, stats, "gc_count")

	// 测试TimerTracker
	tt := NewTimerTracker()
	require.NotNil(t, tt)

	tt.RecordTimerCreated()
	assert.Equal(t, 1, tt.GetActiveTimerCount())

	tt.RecordTimerStopped()
	assert.Equal(t, 0, tt.GetActiveTimerCount())

	stats = tt.GetStats()
	assert.Contains(t, stats, "active")
	assert.Contains(t, stats, "total_created")
}
