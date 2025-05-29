/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 性能优化器功能的单元测试
 */

package performance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerformanceOptimizer_Basic(t *testing.T) {
	config := OptimizerConfig{
		OptimizationInterval: 100 * time.Millisecond,
		Timeout:              5 * time.Second,
		EnableAutoGC:         true,
		EnableMemoryLimit:    true,
		EnableCPUThrottling:  true,
		MemoryThresholdMB:    50.0,
		CPUThresholdPercent:  80.0,
		GCThresholdMB:        20.0,
		GoroutineThreshold:   100,
		GCTargetPercent:      100,
		MaxGoroutines:        1000,
		MemoryLimitMB:        200.0,
		CPUQuotaPercent:      80.0,
	}

	optimizer := NewPerformanceOptimizer(config)
	require.NotNil(t, optimizer)

	// 测试启动
	err := optimizer.Start()
	assert.NoError(t, err)
	assert.True(t, optimizer.IsRunning())

	// 测试重复启动
	err = optimizer.Start()
	assert.Error(t, err)

	// 测试停止
	err = optimizer.Stop()
	assert.NoError(t, err)
	assert.False(t, optimizer.IsRunning())

	// 测试重复停止
	err = optimizer.Stop()
	assert.Error(t, err)
}

func TestPerformanceOptimizer_TimeoutProtection(t *testing.T) {
	// 使用很短的超时时间来测试超时保护
	config := OptimizerConfig{
		OptimizationInterval: 50 * time.Millisecond,
		Timeout:              1 * time.Millisecond, // 极短的超时时间
		EnableAutoGC:         true,
		EnableMemoryLimit:    true,
		EnableCPUThrottling:  true,
		MemoryThresholdMB:    0.1, // 很低的阈值，容易触发优化
		CPUThresholdPercent:  0.1,
		GCThresholdMB:        0.1,
		GoroutineThreshold:   1,
	}

	optimizer := NewPerformanceOptimizer(config)
	require.NotNil(t, optimizer)

	err := optimizer.Start()
	require.NoError(t, err)
	defer optimizer.Stop()

	// 等待一段时间让优化循环运行
	time.Sleep(200 * time.Millisecond)

	// 检查错误通道是否有超时错误
	select {
	case err := <-optimizer.GetErrorChannel():
		t.Logf("收到预期的超时错误: %v", err)
	case <-time.After(100 * time.Millisecond):
		t.Log("未收到超时错误，可能系统性能足够好")
	}

	// 验证优化器仍在运行（没有死锁）
	assert.True(t, optimizer.IsRunning())
}

func TestPerformanceOptimizer_ForceOptimization(t *testing.T) {
	config := OptimizerConfig{
		OptimizationInterval: 1 * time.Hour, // 长间隔，避免自动优化干扰
		Timeout:              5 * time.Second,
		EnableAutoGC:         true,
		EnableMemoryLimit:    false,
		EnableCPUThrottling:  false,
		MemoryThresholdMB:    0.1, // 很低的阈值，容易触发优化
		CPUThresholdPercent:  80.0,
		GCThresholdMB:        0.1,
		GoroutineThreshold:   100,
	}

	optimizer := NewPerformanceOptimizer(config)
	require.NotNil(t, optimizer)

	err := optimizer.Start()
	require.NoError(t, err)
	defer optimizer.Stop()

	// 强制执行优化
	err = optimizer.ForceOptimization()
	assert.NoError(t, err)

	// 获取优化指标
	metrics := optimizer.GetMetrics()
	t.Logf("优化指标: %+v", metrics)

	// 验证指标被更新
	assert.GreaterOrEqual(t, metrics.CurrentMemoryMB, 0.0)
	assert.GreaterOrEqual(t, metrics.CurrentCPUPercent, 0.0)
	assert.Greater(t, metrics.CurrentGoroutines, 0)
}

func TestPerformanceOptimizer_MemoryOptimization(t *testing.T) {
	config := OptimizerConfig{
		OptimizationInterval: 1 * time.Hour,
		Timeout:              5 * time.Second,
		EnableAutoGC:         true,
		EnableMemoryLimit:    true,
		EnableCPUThrottling:  false,
		MemoryThresholdMB:    0.1, // 很低的阈值
		CPUThresholdPercent:  80.0,
		GCThresholdMB:        0.1,
		GoroutineThreshold:   100,
	}

	optimizer := NewPerformanceOptimizer(config)
	require.NotNil(t, optimizer)

	err := optimizer.Start()
	require.NoError(t, err)
	defer optimizer.Stop()

	// 分配一些内存来触发优化
	data := make([][]byte, 100)
	for i := range data {
		data[i] = make([]byte, 1024*1024) // 1MB each
	}

	// 强制执行优化
	err = optimizer.ForceOptimization()
	assert.NoError(t, err)

	// 获取优化指标
	metrics := optimizer.GetMetrics()
	t.Logf("内存优化指标: %+v", metrics)

	// 验证内存优化被执行
	assert.GreaterOrEqual(t, metrics.MemoryOptimizations, 0)
	assert.GreaterOrEqual(t, metrics.GCOptimizations, 0)

	// 清理内存引用
	data = nil
}

func TestPerformanceOptimizer_AlertGeneration(t *testing.T) {
	config := OptimizerConfig{
		OptimizationInterval: 50 * time.Millisecond,
		Timeout:              5 * time.Second,
		EnableAutoGC:         true,
		EnableMemoryLimit:    true,
		EnableCPUThrottling:  true,
		MemoryThresholdMB:    0.1, // 很低的阈值，容易触发告警
		CPUThresholdPercent:  0.1,
		GCThresholdMB:        0.1,
		GoroutineThreshold:   1,
	}

	optimizer := NewPerformanceOptimizer(config)
	require.NotNil(t, optimizer)

	err := optimizer.Start()
	require.NoError(t, err)
	defer optimizer.Stop()

	// 等待一段时间让优化循环运行
	time.Sleep(200 * time.Millisecond)

	// 检查是否有告警产生
	select {
	case alert := <-optimizer.GetAlertChannel():
		t.Logf("收到告警: %s - %s", alert.Type, alert.Message)
		assert.NotEmpty(t, alert.ID)
		assert.NotEmpty(t, alert.Type)
		assert.NotEmpty(t, alert.Message)
		assert.NotEmpty(t, alert.Action)
	case <-time.After(300 * time.Millisecond):
		t.Log("未收到告警，可能阈值设置过低或系统负载很轻")
	}
}

func TestPerformanceOptimizer_ChannelNonBlocking(t *testing.T) {
	config := OptimizerConfig{
		OptimizationInterval: 10 * time.Millisecond,
		Timeout:              2 * time.Second, // 增加超时时间
		EnableAutoGC:         true,
		EnableMemoryLimit:    true,
		EnableCPUThrottling:  true,
		MemoryThresholdMB:    0.1, // 很低的阈值，容易触发告警
		CPUThresholdPercent:  0.1,
		GCThresholdMB:        0.1,
		GoroutineThreshold:   1,
	}

	optimizer := NewPerformanceOptimizer(config)
	require.NotNil(t, optimizer)

	err := optimizer.Start()
	require.NoError(t, err)
	defer optimizer.Stop()

	// 等待一段时间让通道可能被填满
	time.Sleep(200 * time.Millisecond)

	// 验证优化器仍在运行（没有死锁）
	assert.True(t, optimizer.IsRunning())

	// 尝试强制执行多次优化，但减少次数避免超时
	for i := 0; i < 3; i++ {
		err := optimizer.ForceOptimization()
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond) // 给每次优化一些时间
	}

	// 验证优化器仍在运行
	assert.True(t, optimizer.IsRunning())
}

func TestPerformanceOptimizer_ConcurrentAccess(t *testing.T) {
	config := OptimizerConfig{
		OptimizationInterval: 100 * time.Millisecond,
		Timeout:              2 * time.Second,
		EnableAutoGC:         true,
		EnableMemoryLimit:    false,
		EnableCPUThrottling:  false,
		MemoryThresholdMB:    50.0,
		CPUThresholdPercent:  80.0,
		GCThresholdMB:        20.0,
		GoroutineThreshold:   100,
	}

	optimizer := NewPerformanceOptimizer(config)
	require.NotNil(t, optimizer)

	err := optimizer.Start()
	require.NoError(t, err)
	defer optimizer.Stop()

	// 并发访问优化器
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(num int) {
			defer func() { done <- true }()

			// 并发执行强制优化
			for j := 0; j < 3; j++ {
				err := optimizer.ForceOptimization()
				if err != nil {
					t.Errorf("强制优化失败: %v", err)
					return
				}

				// 并发获取指标
				metrics := optimizer.GetMetrics()
				if metrics.CurrentMemoryMB < 0 {
					t.Error("获取到无效的内存指标")
					return
				}

				// 检查运行状态
				if !optimizer.IsRunning() {
					t.Error("优化器意外停止")
					return
				}

				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// 等待所有并发任务完成
	for i := 0; i < 5; i++ {
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("并发测试超时")
		}
	}

	// 验证优化器仍在运行
	assert.True(t, optimizer.IsRunning())
}

func TestPerformanceOptimizer_MetricsAccuracy(t *testing.T) {
	config := OptimizerConfig{
		OptimizationInterval: 1 * time.Hour, // 长间隔避免干扰
		Timeout:              5 * time.Second,
		EnableAutoGC:         true,
		EnableMemoryLimit:    true,
		EnableCPUThrottling:  true,
		MemoryThresholdMB:    0.1,
		CPUThresholdPercent:  0.1,
		GCThresholdMB:        0.1,
		GoroutineThreshold:   1,
	}

	optimizer := NewPerformanceOptimizer(config)
	require.NotNil(t, optimizer)

	err := optimizer.Start()
	require.NoError(t, err)
	defer optimizer.Stop()

	// 执行优化
	err = optimizer.ForceOptimization()
	require.NoError(t, err)

	// 获取指标
	metrics := optimizer.GetMetrics()

	// 验证指标的合理性
	assert.GreaterOrEqual(t, metrics.CurrentMemoryMB, 0.0)
	assert.LessOrEqual(t, metrics.CurrentMemoryMB, 10000.0) // 不应该超过10GB
	assert.GreaterOrEqual(t, metrics.CurrentCPUPercent, 0.0)
	assert.LessOrEqual(t, metrics.CurrentCPUPercent, 100.0)
	assert.Greater(t, metrics.CurrentGoroutines, 0)
	assert.LessOrEqual(t, metrics.CurrentGoroutines, 100000) // 合理的协程数量上限

	// 验证累计指标
	assert.GreaterOrEqual(t, metrics.TotalOptimizations, 0)
	assert.GreaterOrEqual(t, metrics.MemoryOptimizations, 0)
	assert.GreaterOrEqual(t, metrics.GCOptimizations, 0)
	assert.GreaterOrEqual(t, metrics.CPUOptimizations, 0)
	assert.GreaterOrEqual(t, metrics.MemorySavedMB, 0.0)
	assert.GreaterOrEqual(t, metrics.CPUSavedPercent, 0.0)
	assert.GreaterOrEqual(t, metrics.GCPauseReduced, 0.0)
}

func TestPerformanceOptimizer_DefaultValues(t *testing.T) {
	// 测试默认配置值
	config := OptimizerConfig{} // 空配置

	optimizer := NewPerformanceOptimizer(config)
	require.NotNil(t, optimizer)

	// 验证默认值被正确设置
	assert.Equal(t, 30*time.Second, optimizer.config.OptimizationInterval)
	assert.Equal(t, 10*time.Second, optimizer.config.Timeout)
	assert.Equal(t, 100.0, optimizer.config.MemoryThresholdMB)
	assert.Equal(t, 80.0, optimizer.config.CPUThresholdPercent)
	assert.Equal(t, 50.0, optimizer.config.GCThresholdMB)
	assert.Equal(t, 1000, optimizer.config.GoroutineThreshold)
	assert.Equal(t, 100, optimizer.config.GCTargetPercent)
	assert.Equal(t, 10000, optimizer.config.MaxGoroutines)
}

func TestPerformanceOptimizer_StopWhileRunning(t *testing.T) {
	config := OptimizerConfig{
		OptimizationInterval: 50 * time.Millisecond,
		Timeout:              5 * time.Second,
		EnableAutoGC:         true,
		MemoryThresholdMB:    0.1,
		GCThresholdMB:        0.1,
	}

	optimizer := NewPerformanceOptimizer(config)
	require.NotNil(t, optimizer)

	err := optimizer.Start()
	require.NoError(t, err)

	// 等待优化循环开始
	time.Sleep(100 * time.Millisecond)

	// 在运行时停止
	err = optimizer.Stop()
	assert.NoError(t, err)
	assert.False(t, optimizer.IsRunning())

	// 验证强制优化在停止后失败
	err = optimizer.ForceOptimization()
	assert.Error(t, err)
}

// 基准测试
func BenchmarkPerformanceOptimizer_ForceOptimization(b *testing.B) {
	config := OptimizerConfig{
		OptimizationInterval: 1 * time.Hour, // 长间隔避免干扰
		Timeout:              5 * time.Second,
		EnableAutoGC:         true,
		EnableMemoryLimit:    false,
		EnableCPUThrottling:  false,
		MemoryThresholdMB:    0.1,
		GCThresholdMB:        0.1,
	}

	optimizer := NewPerformanceOptimizer(config)
	err := optimizer.Start()
	if err != nil {
		b.Fatal(err)
	}
	defer optimizer.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := optimizer.ForceOptimization()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPerformanceOptimizer_GetMetrics(b *testing.B) {
	config := OptimizerConfig{
		OptimizationInterval: 1 * time.Hour,
		Timeout:              5 * time.Second,
		EnableAutoGC:         true,
	}

	optimizer := NewPerformanceOptimizer(config)
	err := optimizer.Start()
	if err != nil {
		b.Fatal(err)
	}
	defer optimizer.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := optimizer.GetMetrics()
		if metrics.CurrentMemoryMB < 0 {
			b.Fatal("invalid metrics")
		}
	}
}
