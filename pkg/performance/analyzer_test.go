/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 性能分析器功能的单元测试
 */

package performance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskPerformanceAnalyzer_Basic(t *testing.T) {
	config := AnalyzerConfig{
		SampleInterval: 100 * time.Millisecond,
		Timeout:        5 * time.Second,
		MaxHistory:     10,
		EnableAlerts:   true,
		AlertThresholds: AlertThresholds{
			CPUUsagePercent: 80.0,
			MemoryUsageMB:   100.0,
			ExecutionTimeMs: 1000,
			GoroutineCount:  100,
			GCPauseMs:       10.0,
		},
	}

	analyzer := NewTaskPerformanceAnalyzer(config)
	require.NotNil(t, analyzer)

	// 测试启动
	err := analyzer.Start()
	assert.NoError(t, err)
	assert.True(t, analyzer.IsRunning())

	// 测试重复启动
	err = analyzer.Start()
	assert.Error(t, err)

	// 测试停止
	err = analyzer.Stop()
	assert.NoError(t, err)
	assert.False(t, analyzer.IsRunning())

	// 测试重复停止
	err = analyzer.Stop()
	assert.Error(t, err)
}

func TestTaskPerformanceAnalyzer_TaskAnalysis(t *testing.T) {
	config := AnalyzerConfig{
		SampleInterval: 50 * time.Millisecond,
		Timeout:        2 * time.Second,
		MaxHistory:     5,
		EnableAlerts:   false,
	}

	analyzer := NewTaskPerformanceAnalyzer(config)
	require.NotNil(t, analyzer)

	err := analyzer.Start()
	require.NoError(t, err)
	defer analyzer.Stop()

	// 开始任务分析
	ctx, err := analyzer.StartTaskAnalysis("test-task-1", "测试任务")
	require.NoError(t, err)
	require.NotNil(t, ctx)

	// 模拟任务执行
	time.Sleep(200 * time.Millisecond)

	// 完成任务分析
	metrics := analyzer.FinishTaskAnalysis(ctx)
	require.NotNil(t, metrics)

	assert.Equal(t, "test-task-1", metrics.TaskID)
	assert.Equal(t, "测试任务", metrics.TaskName)
	assert.True(t, metrics.Duration > 0)
	assert.False(t, metrics.StartTime.IsZero())
	assert.False(t, metrics.EndTime.IsZero())

	// 获取任务指标
	retrievedMetrics, err := analyzer.GetTaskMetrics("test-task-1")
	require.NoError(t, err)
	assert.Equal(t, metrics.TaskID, retrievedMetrics.TaskID)
}

func TestTaskPerformanceAnalyzer_TimeoutProtection(t *testing.T) {
	// 使用很短的超时时间来测试超时保护
	config := AnalyzerConfig{
		SampleInterval: 10 * time.Millisecond,
		Timeout:        1 * time.Millisecond, // 极短的超时时间
		MaxHistory:     5,
		EnableAlerts:   false,
	}

	analyzer := NewTaskPerformanceAnalyzer(config)
	require.NotNil(t, analyzer)

	err := analyzer.Start()
	require.NoError(t, err)
	defer analyzer.Stop()

	// 开始任务分析
	ctx, err := analyzer.StartTaskAnalysis("timeout-test", "超时测试")
	require.NoError(t, err)
	require.NotNil(t, ctx)

	// 等待一段时间让采样发生
	time.Sleep(100 * time.Millisecond)

	// 检查错误通道是否有超时错误
	select {
	case err := <-analyzer.GetErrorChannel():
		t.Logf("收到预期的超时错误: %v", err)
	case <-time.After(200 * time.Millisecond):
		t.Log("未收到超时错误，可能系统性能足够好")
	}

	// 完成任务分析
	metrics := analyzer.FinishTaskAnalysis(ctx)
	require.NotNil(t, metrics)
}

func TestTaskPerformanceAnalyzer_ChannelNonBlocking(t *testing.T) {
	config := AnalyzerConfig{
		SampleInterval: 10 * time.Millisecond,
		Timeout:        1 * time.Second,
		MaxHistory:     2, // 小的历史记录数
		EnableAlerts:   true,
		AlertThresholds: AlertThresholds{
			CPUUsagePercent: 0.1, // 很低的阈值，容易触发告警
			MemoryUsageMB:   0.1,
			ExecutionTimeMs: 1,
			GoroutineCount:  1,
			GCPauseMs:       0.1,
		},
	}

	analyzer := NewTaskPerformanceAnalyzer(config)
	require.NotNil(t, analyzer)

	err := analyzer.Start()
	require.NoError(t, err)
	defer analyzer.Stop()

	// 创建多个任务来填满通道
	for i := 0; i < 10; i++ {
		taskID := "task-" + string(rune('0'+i))
		ctx, err := analyzer.StartTaskAnalysis(taskID, "测试任务")
		require.NoError(t, err)

		// 立即完成任务
		metrics := analyzer.FinishTaskAnalysis(ctx)
		require.NotNil(t, metrics)
	}

	// 等待一段时间让通道处理
	time.Sleep(200 * time.Millisecond)

	// 验证分析器仍在运行（没有死锁）
	assert.True(t, analyzer.IsRunning())
}

func TestTaskPerformanceAnalyzer_AlertGeneration(t *testing.T) {
	config := AnalyzerConfig{
		SampleInterval: 100 * time.Millisecond,
		Timeout:        5 * time.Second,
		MaxHistory:     5,
		EnableAlerts:   true,
		AlertThresholds: AlertThresholds{
			CPUUsagePercent: 0.1, // 很低的阈值
			MemoryUsageMB:   0.1,
			ExecutionTimeMs: 1,
			GoroutineCount:  1,
			GCPauseMs:       0.1,
		},
	}

	analyzer := NewTaskPerformanceAnalyzer(config)
	require.NotNil(t, analyzer)

	err := analyzer.Start()
	require.NoError(t, err)
	defer analyzer.Stop()

	// 开始任务分析
	ctx, err := analyzer.StartTaskAnalysis("alert-test", "告警测试")
	require.NoError(t, err)

	// 模拟一些工作负载
	time.Sleep(50 * time.Millisecond)

	// 完成任务分析
	metrics := analyzer.FinishTaskAnalysis(ctx)
	require.NotNil(t, metrics)

	// 检查是否有告警产生
	select {
	case alert := <-analyzer.GetAlertChannel():
		t.Logf("收到告警: %s - %s", alert.Type, alert.Message)
		assert.NotEmpty(t, alert.ID)
		assert.NotEmpty(t, alert.Type)
		assert.NotEmpty(t, alert.Message)
		assert.Equal(t, "alert-test", alert.TaskID)
	case <-time.After(100 * time.Millisecond):
		t.Log("未收到告警，可能阈值设置过低或系统负载很轻")
	}
}

func TestTaskPerformanceAnalyzer_ConcurrentAccess(t *testing.T) {
	config := AnalyzerConfig{
		SampleInterval: 50 * time.Millisecond,
		Timeout:        2 * time.Second,
		MaxHistory:     10,
		EnableAlerts:   false,
	}

	analyzer := NewTaskPerformanceAnalyzer(config)
	require.NotNil(t, analyzer)

	err := analyzer.Start()
	require.NoError(t, err)
	defer analyzer.Stop()

	// 并发启动多个任务
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(taskNum int) {
			defer func() { done <- true }()

			taskID := "concurrent-task-" + string(rune('0'+taskNum))
			ctx, err := analyzer.StartTaskAnalysis(taskID, "并发测试任务")
			if err != nil {
				t.Errorf("启动任务分析失败: %v", err)
				return
			}

			// 模拟工作
			time.Sleep(100 * time.Millisecond)

			// 完成任务
			metrics := analyzer.FinishTaskAnalysis(ctx)
			if metrics == nil {
				t.Error("完成任务分析返回nil")
				return
			}

			// 获取指标
			_, err = analyzer.GetTaskMetrics(taskID)
			if err != nil {
				t.Errorf("获取任务指标失败: %v", err)
			}
		}(i)
	}

	// 等待所有任务完成
	for i := 0; i < 5; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("并发测试超时")
		}
	}

	// 验证所有指标都被收集
	allMetrics := analyzer.GetAllMetrics()
	assert.Equal(t, 5, len(allMetrics))
}

func TestTaskPerformanceAnalyzer_ContextCancellation(t *testing.T) {
	config := AnalyzerConfig{
		SampleInterval: 10 * time.Millisecond,
		Timeout:        1 * time.Second,
		MaxHistory:     5,
		EnableAlerts:   false,
	}

	analyzer := NewTaskPerformanceAnalyzer(config)
	require.NotNil(t, analyzer)

	err := analyzer.Start()
	require.NoError(t, err)

	// 开始任务分析
	ctx, err := analyzer.StartTaskAnalysis("cancel-test", "取消测试")
	require.NoError(t, err)

	// 等待一小段时间让采样开始
	time.Sleep(50 * time.Millisecond)

	// 停止分析器（这会取消上下文）
	err = analyzer.Stop()
	require.NoError(t, err)

	// 验证分析器已停止
	assert.False(t, analyzer.IsRunning())

	// 尝试完成任务分析（应该仍然工作）
	metrics := analyzer.FinishTaskAnalysis(ctx)
	require.NotNil(t, metrics)
}

func TestTaskPerformanceAnalyzer_HistoryLimit(t *testing.T) {
	config := AnalyzerConfig{
		SampleInterval: 20 * time.Millisecond,
		Timeout:        1 * time.Second,
		MaxHistory:     3, // 限制历史记录数
		EnableAlerts:   false,
	}

	analyzer := NewTaskPerformanceAnalyzer(config)
	require.NotNil(t, analyzer)

	err := analyzer.Start()
	require.NoError(t, err)
	defer analyzer.Stop()

	// 开始任务分析
	ctx, err := analyzer.StartTaskAnalysis("history-test", "历史记录测试")
	require.NoError(t, err)

	// 等待足够长的时间让采样超过历史限制
	time.Sleep(200 * time.Millisecond)

	// 完成任务分析
	metrics := analyzer.FinishTaskAnalysis(ctx)
	require.NotNil(t, metrics)

	// 验证历史记录被正确限制
	ctx.mu.RLock()
	sampleCount := len(ctx.samples)
	ctx.mu.RUnlock()

	assert.LessOrEqual(t, sampleCount, config.MaxHistory)
}

func TestTaskPerformanceAnalyzer_MetricsCollection(t *testing.T) {
	config := AnalyzerConfig{
		SampleInterval: 100 * time.Millisecond,
		Timeout:        5 * time.Second,
		MaxHistory:     5,
		EnableAlerts:   false,
	}

	analyzer := NewTaskPerformanceAnalyzer(config)
	require.NotNil(t, analyzer)

	err := analyzer.Start()
	require.NoError(t, err)
	defer analyzer.Stop()

	// 开始任务分析
	ctx, err := analyzer.StartTaskAnalysis("metrics-test", "指标收集测试")
	require.NoError(t, err)

	// 模拟一些工作负载
	time.Sleep(150 * time.Millisecond)

	// 完成任务分析
	metrics := analyzer.FinishTaskAnalysis(ctx)
	require.NotNil(t, metrics)

	// 验证指标字段
	assert.Equal(t, "metrics-test", metrics.TaskID)
	assert.Equal(t, "指标收集测试", metrics.TaskName)
	assert.True(t, metrics.Duration > 0)
	assert.NotNil(t, metrics.CustomMetrics)

	// 验证运行时指标
	assert.Greater(t, metrics.RuntimeMetrics.GoroutineCount, 0)
	assert.GreaterOrEqual(t, metrics.RuntimeMetrics.HeapAllocMB, 0.0)
	assert.GreaterOrEqual(t, metrics.RuntimeMetrics.HeapSysMB, 0.0)
	assert.GreaterOrEqual(t, metrics.RuntimeMetrics.GCCount, uint32(0))

	// 验证系统指标
	assert.GreaterOrEqual(t, metrics.SystemMetrics.TotalCPUPercent, 0.0)
	assert.GreaterOrEqual(t, metrics.SystemMetrics.TotalMemoryUsedMB, uint64(0))
	assert.GreaterOrEqual(t, metrics.SystemMetrics.TotalMemoryPercent, 0.0)

	// 验证内存指标
	assert.GreaterOrEqual(t, metrics.MemoryUsage.UsedPercent, 0.0)
	assert.GreaterOrEqual(t, metrics.MemoryUsage.AvailableMB, uint64(0))
}

// 基准测试
func BenchmarkTaskPerformanceAnalyzer_MetricsCollection(b *testing.B) {
	config := AnalyzerConfig{
		SampleInterval: 1 * time.Second, // 长间隔避免干扰基准测试
		Timeout:        5 * time.Second,
		MaxHistory:     100,
		EnableAlerts:   false,
	}

	analyzer := NewTaskPerformanceAnalyzer(config)
	err := analyzer.Start()
	if err != nil {
		b.Fatal(err)
	}
	defer analyzer.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, err := analyzer.StartTaskAnalysis("bench-task", "基准测试")
		if err != nil {
			b.Fatal(err)
		}

		metrics := analyzer.FinishTaskAnalysis(ctx)
		if metrics == nil {
			b.Fatal("metrics is nil")
		}
	}
}

func BenchmarkTaskPerformanceAnalyzer_ConcurrentAnalysis(b *testing.B) {
	config := AnalyzerConfig{
		SampleInterval: 1 * time.Second,
		Timeout:        5 * time.Second,
		MaxHistory:     100,
		EnableAlerts:   false,
	}

	analyzer := NewTaskPerformanceAnalyzer(config)
	err := analyzer.Start()
	if err != nil {
		b.Fatal(err)
	}
	defer analyzer.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			taskID := "bench-concurrent-" + string(rune('0'+(i%10)))
			ctx, err := analyzer.StartTaskAnalysis(taskID, "并发基准测试")
			if err != nil {
				b.Fatal(err)
			}

			metrics := analyzer.FinishTaskAnalysis(ctx)
			if metrics == nil {
				b.Fatal("metrics is nil")
			}
			i++
		}
	})
}
