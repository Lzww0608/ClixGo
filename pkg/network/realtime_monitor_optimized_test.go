/*
* @Author: Lzww0608
* @Date: 2025-6-6 23:48:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-7 15:52:23
* @Description: 优化版网络监控器测试 - 验证goroutine池和优雅关闭功能
 */

package network

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	clixgosync "github.com/Lzww0608/ClixGo/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOptimizedRealtimeNetworkMonitor 测试优化版网络监控器基本功能
func TestOptimizedRealtimeNetworkMonitor(t *testing.T) {
	config := RealtimeMonitorConfig{
		UpdateInterval:   2 * time.Second,
		Timeout:          5 * time.Second,
		MaxHistory:       10,
		EnableAlerts:     true,
		MonitoredTargets: []string{"8.8.8.8"},
	}

	monitor := NewOptimizedRealtimeNetworkMonitor(config)
	require.NotNil(t, monitor)

	t.Run("启动和停止", func(t *testing.T) {
		// 测试启动
		err := monitor.Start()
		require.NoError(t, err)
		assert.True(t, monitor.IsRunning())

		// 等待一段时间让监控器运行
		time.Sleep(3 * time.Second)

		// 测试停止
		err = monitor.Stop()
		require.NoError(t, err)
		assert.False(t, monitor.IsRunning())
	})

	t.Run("重复启动应该失败", func(t *testing.T) {
		err := monitor.Start()
		require.NoError(t, err)
		defer monitor.Stop()

		// 重复启动应该失败
		err = monitor.Start()
		assert.Error(t, err)
	})
}

// TestOptimizedNetworkMonitorConcurrency 测试并发性能
func TestOptimizedNetworkMonitorConcurrency(t *testing.T) {
	config := RealtimeMonitorConfig{
		UpdateInterval:   500 * time.Millisecond,
		Timeout:          2 * time.Second,
		MaxHistory:       50,
		EnableAlerts:     false,
		MonitoredTargets: []string{"8.8.8.8", "1.1.1.1"},
		Interfaces:       []string{"lo"}, // 只监控loopback接口
	}

	monitor := NewOptimizedRealtimeNetworkMonitor(config)
	require.NotNil(t, monitor)

	err := monitor.Start()
	require.NoError(t, err)
	defer monitor.Stop()

	t.Run("并发数据收集", func(t *testing.T) {
		var updateCount int32
		var errorCount int32
		var wg sync.WaitGroup

		// 启动数据收集goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			updateChan := monitor.GetUpdateChannel()
			errorChan := monitor.GetErrorChannel()

			timeout := time.After(10 * time.Second)
			for {
				select {
				case <-updateChan:
					atomic.AddInt32(&updateCount, 1)
				case <-errorChan:
					atomic.AddInt32(&errorCount, 1)
				case <-timeout:
					return
				}
			}
		}()

		// 等待数据收集
		time.Sleep(5 * time.Second)

		// 检查goroutine池指标
		poolMetrics := monitor.GetPoolMetrics()
		assert.GreaterOrEqual(t, poolMetrics.TotalWorkers, int32(3))
		assert.GreaterOrEqual(t, poolMetrics.CompletedTasks, uint64(5))

		wg.Wait()

		// 验证收集到了数据
		updates := atomic.LoadInt32(&updateCount)
		errors := atomic.LoadInt32(&errorCount)

		t.Logf("收到更新: %d, 错误: %d", updates, errors)
		assert.Greater(t, updates, int32(5), "应该收到多个更新")
	})

	t.Run("历史记录管理", func(t *testing.T) {
		// 等待足够的时间收集历史记录
		time.Sleep(3 * time.Second)

		history := monitor.GetHistory()
		assert.NotEmpty(t, history, "应该有历史记录")
		assert.LessOrEqual(t, len(history), config.MaxHistory, "历史记录不应超过最大限制")

		currentSnapshot := monitor.GetCurrentSnapshot()
		assert.NotNil(t, currentSnapshot, "应该有当前快照")
	})
}

// TestOptimizedNetworkMonitorResourceManagement 测试资源管理
func TestOptimizedNetworkMonitorResourceManagement(t *testing.T) {
	config := RealtimeMonitorConfig{
		UpdateInterval:   100 * time.Millisecond,
		Timeout:          1 * time.Second,
		MaxHistory:       5,
		EnableAlerts:     true,
		MonitoredTargets: []string{"127.0.0.1"},
	}

	t.Run("多个监控器并发运行", func(t *testing.T) {
		const numMonitors = 3
		monitors := make([]*OptimizedRealtimeNetworkMonitor, numMonitors)

		// 创建并启动多个监控器
		for i := 0; i < numMonitors; i++ {
			monitors[i] = NewOptimizedRealtimeNetworkMonitor(config)
			err := monitors[i].Start()
			require.NoError(t, err)
		}

		// 运行一段时间
		time.Sleep(2 * time.Second)

		// 验证所有监控器都在运行
		for i, monitor := range monitors {
			assert.True(t, monitor.IsRunning(), "监控器 %d 应该在运行", i)

			poolMetrics := monitor.GetPoolMetrics()
			assert.Greater(t, poolMetrics.TotalWorkers, int32(0), "监控器 %d 应该有活跃工作者", i)
		}

		// 停止所有监控器
		for i, monitor := range monitors {
			err := monitor.Stop()
			assert.NoError(t, err, "停止监控器 %d 应该成功", i)
			assert.False(t, monitor.IsRunning(), "监控器 %d 应该已停止", i)
		}
	})

	t.Run("优雅关闭测试", func(t *testing.T) {
		monitor := NewOptimizedRealtimeNetworkMonitor(config)

		err := monitor.Start()
		require.NoError(t, err)

		// 启动一些数据消费者
		var wg sync.WaitGroup
		stopChan := make(chan struct{})

		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				updateChan := monitor.GetUpdateChannel()

				for {
					select {
					case <-updateChan:
						// 处理更新
					case <-stopChan:
						return
					}
				}
			}(i)
		}

		// 运行一段时间
		time.Sleep(1 * time.Second)

		// 停止监控器
		startStop := time.Now()
		close(stopChan)
		err = monitor.Stop()
		stopDuration := time.Since(startStop)

		assert.NoError(t, err)
		assert.Less(t, stopDuration, 5*time.Second, "优雅关闭应该在合理时间内完成")

		wg.Wait()
	})
}

// TestOptimizedNetworkMonitorPerformance 性能测试
func TestOptimizedNetworkMonitorPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过性能测试")
	}

	config := RealtimeMonitorConfig{
		UpdateInterval: 50 * time.Millisecond, // 高频更新
		Timeout:        1 * time.Second,
		MaxHistory:     100,
		EnableAlerts:   true,
		MonitoredTargets: []string{
			"8.8.8.8", "1.1.1.1", "208.67.222.222", "9.9.9.9",
		},
	}

	monitor := NewOptimizedRealtimeNetworkMonitor(config)
	require.NotNil(t, monitor)

	err := monitor.Start()
	require.NoError(t, err)
	defer monitor.Stop()

	t.Run("高频数据收集性能", func(t *testing.T) {
		var updateCount int64
		var totalLatency time.Duration
		var latencyMu sync.Mutex

		startTime := time.Now()
		testDuration := 10 * time.Second

		// 数据收集器
		go func() {
			updateChan := monitor.GetUpdateChannel()
			for {
				select {
				case snapshot := <-updateChan:
					now := time.Now()
					latency := now.Sub(snapshot.Timestamp)

					latencyMu.Lock()
					totalLatency += latency
					latencyMu.Unlock()

					atomic.AddInt64(&updateCount, 1)
				case <-time.After(testDuration + time.Second):
					return
				}
			}
		}()

		// 等待测试完成
		time.Sleep(testDuration)

		// 计算性能指标
		elapsed := time.Since(startTime)
		updates := atomic.LoadInt64(&updateCount)

		latencyMu.Lock()
		avgLatency := time.Duration(0)
		if updates > 0 {
			avgLatency = time.Duration(int64(totalLatency) / updates)
		}
		latencyMu.Unlock()

		poolMetrics := monitor.GetPoolMetrics()

		t.Logf("性能测试结果:")
		t.Logf("  运行时间: %v", elapsed)
		t.Logf("  总更新数: %d", updates)
		t.Logf("  更新频率: %.2f updates/sec", float64(updates)/elapsed.Seconds())
		t.Logf("  平均延迟: %v", avgLatency)
		t.Logf("  池完成任务: %d", poolMetrics.CompletedTasks)
		t.Logf("  池失败任务: %d", poolMetrics.FailedTasks)
		t.Logf("  平均等待时间: %v", poolMetrics.AverageWaitTime)
		t.Logf("  平均执行时间: %v", poolMetrics.AverageExecTime)

		// 性能断言
		assert.Greater(t, updates, int64(50), "应该收集到足够的更新")
		assert.Less(t, avgLatency, 100*time.Millisecond, "平均延迟应该较低")
		assert.Equal(t, uint64(0), poolMetrics.FailedTasks, "不应该有失败的任务")
	})
}

// BenchmarkOptimizedNetworkMonitor 基准测试
func BenchmarkOptimizedNetworkMonitor(b *testing.B) {
	config := RealtimeMonitorConfig{
		UpdateInterval:   100 * time.Millisecond,
		Timeout:          2 * time.Second,
		MaxHistory:       50,
		EnableAlerts:     false,
		MonitoredTargets: []string{"8.8.8.8"},
	}

	monitor := NewOptimizedRealtimeNetworkMonitor(config)
	err := monitor.Start()
	require.NoError(b, err)
	defer monitor.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		updateChan := monitor.GetUpdateChannel()
		for pb.Next() {
			select {
			case <-updateChan:
				// 消费更新
			case <-time.After(time.Second):
				// 超时保护
			}
		}
	})
}

// TestComponentIntegration 测试组件集成
func TestComponentIntegration(t *testing.T) {
	// 创建独立的优雅关闭管理器
	shutdownManager := clixgosync.NewGracefulShutdownManager(
		clixgosync.DefaultShutdownConfig(),
	)
	err := shutdownManager.Start()
	require.NoError(t, err)
	defer shutdownManager.Stop()

	// 创建网络监控器
	config := RealtimeMonitorConfig{
		UpdateInterval: time.Second,
		Timeout:        5 * time.Second,
		MaxHistory:     10,
		EnableAlerts:   false,
	}

	monitor := NewOptimizedRealtimeNetworkMonitor(config)

	// 创建组件适配器
	component := &NetworkComponent{
		name: "test-network-monitor",
		pool: monitor.goroutinePool,
	}

	// 注册到优雅关闭管理器
	err = shutdownManager.RegisterComponent(component)
	require.NoError(t, err)

	// 启动监控器
	err = monitor.Start()
	require.NoError(t, err)

	// 验证集成工作正常
	assert.True(t, monitor.IsRunning())
	assert.Equal(t, clixgosync.StateRunning, component.State())

	// 等待一些操作
	time.Sleep(2 * time.Second)

	// 验证监控器状态
	poolMetrics := monitor.GetPoolMetrics()
	assert.Greater(t, poolMetrics.TotalWorkers, int32(0))

	// 停止监控器
	err = monitor.Stop()
	require.NoError(t, err)
	assert.False(t, monitor.IsRunning())
}
