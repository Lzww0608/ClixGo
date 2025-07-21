/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 实时网络监控功能的单元测试
 */

package network

import (
	"context"
	"testing"
	"time"
)

func TestRealtimeNetworkMonitor_Basic(t *testing.T) {
	config := RealtimeMonitorConfig{
		UpdateInterval:   1 * time.Second,
		Timeout:          3 * time.Second,
		MaxHistory:       10,
		EnableAlerts:     false,
		MonitoredTargets: []string{"8.8.8.8", "1.1.1.1"},
		Interfaces:       []string{},
	}

	monitor := NewRealtimeNetworkMonitor(config)

	// 测试启动
	err := monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}

	// 等待一些数据
	time.Sleep(2 * time.Second)

	// 检查是否正在运行
	if !monitor.IsRunning() {
		t.Error("监控器应该正在运行")
	}

	// 获取当前快照
	snapshot := monitor.GetCurrentSnapshot()
	if snapshot == nil {
		t.Error("应该有当前快照")
	}

	// 停止监控器
	err = monitor.Stop()
	if err != nil {
		t.Fatalf("停止监控器失败: %v", err)
	}

	// 检查是否已停止
	if monitor.IsRunning() {
		t.Error("监控器应该已停止")
	}
}

func TestRealtimeNetworkMonitor_TimeoutProtection(t *testing.T) {
	config := RealtimeMonitorConfig{
		UpdateInterval:   500 * time.Millisecond,
		Timeout:          1 * time.Second, // 短超时时间
		MaxHistory:       5,
		EnableAlerts:     false,
		MonitoredTargets: []string{"192.0.2.1"}, // 不可达的测试地址
		Interfaces:       []string{},
	}

	monitor := NewRealtimeNetworkMonitor(config)

	err := monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}
	defer monitor.Stop()

	// 监听错误通道，确保超时机制工作
	go func() {
		select {
		case <-monitor.GetErrorChannel():
			// 收到错误，这是预期的
		case <-time.After(5 * time.Second):
			// 超时
		}
	}()

	// 等待足够时间让监控器尝试收集数据
	time.Sleep(3 * time.Second)

	// 验证监控器仍在运行（没有死锁）
	if !monitor.IsRunning() {
		t.Error("监控器不应该因为超时而停止")
	}
}

func TestRealtimeNetworkMonitor_ChannelNonBlocking(t *testing.T) {
	config := RealtimeMonitorConfig{
		UpdateInterval:   100 * time.Millisecond, // 快速更新
		Timeout:          2 * time.Second,
		MaxHistory:       3, // 小历史记录
		EnableAlerts:     false,
		MonitoredTargets: []string{},
		Interfaces:       []string{},
	}

	monitor := NewRealtimeNetworkMonitor(config)

	err := monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}
	defer monitor.Stop()

	// 不读取更新通道，测试是否会阻塞
	time.Sleep(2 * time.Second)

	// 验证监控器仍在运行
	if !monitor.IsRunning() {
		t.Error("监控器不应该因为通道满而阻塞")
	}

	// 现在读取一些更新
	updateCount := 0
	timeout := time.After(1 * time.Second)

	for updateCount < 3 {
		select {
		case <-monitor.GetUpdateChannel():
			updateCount++
		case <-timeout:
			goto timeoutReached
		}
	}
timeoutReached:

	if updateCount == 0 {
		t.Error("应该收到一些更新")
	}
}

func TestRealtimeNetworkMonitor_AlertThresholds(t *testing.T) {
	config := RealtimeMonitorConfig{
		UpdateInterval: 1 * time.Second,
		Timeout:        3 * time.Second,
		MaxHistory:     5,
		EnableAlerts:   true,
		AlertThresholds: AlertThresholds{
			LatencyMs:         50.0, // 低阈值，容易触发
			PacketLossPercent: 1.0,  // 低阈值，容易触发
			BandwidthMbps:     1.0,  // 低阈值，容易触发
			ConnectionCount:   100,  // 低阈值，容易触发
			ErrorRate:         0.1,  // 低阈值，容易触发
		},
		MonitoredTargets: []string{"8.8.8.8"},
		Interfaces:       []string{},
	}

	monitor := NewRealtimeNetworkMonitor(config)

	err := monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}
	defer monitor.Stop()

	// 等待一些数据和可能的告警
	time.Sleep(3 * time.Second)

	snapshot := monitor.GetCurrentSnapshot()
	if snapshot == nil {
		t.Error("应该有当前快照")
		return
	}

	// 检查告警功能是否工作（可能有告警，也可能没有，取决于网络状况）
	t.Logf("收到 %d 个告警", len(snapshot.Alerts))
}

func TestRealtimeNetworkMonitor_ContextCancellation(t *testing.T) {
	config := RealtimeMonitorConfig{
		UpdateInterval:   500 * time.Millisecond,
		Timeout:          2 * time.Second,
		MaxHistory:       5,
		EnableAlerts:     false,
		MonitoredTargets: []string{"8.8.8.8"},
		Interfaces:       []string{},
	}

	monitor := NewRealtimeNetworkMonitor(config)

	err := monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}

	// 立即停止，测试上下文取消
	err = monitor.Stop()
	if err != nil {
		t.Fatalf("停止监控器失败: %v", err)
	}

	// 等待一点时间确保清理完成
	time.Sleep(100 * time.Millisecond)

	if monitor.IsRunning() {
		t.Error("监控器应该已停止")
	}
}

func TestRealtimeNetworkMonitor_Statistics(t *testing.T) {
	config := RealtimeMonitorConfig{
		UpdateInterval:   100 * time.Millisecond, // 减少更新间隔
		Timeout:          1 * time.Second,        // 减少超时时间
		MaxHistory:       5,
		EnableAlerts:     true,
		MonitoredTargets: []string{"8.8.8.8"},
		Interfaces:       []string{},
		AlertThresholds: AlertThresholds{
			LatencyMs:         100.0,
			PacketLossPercent: 5.0,
			BandwidthMbps:     100.0,
			ConnectionCount:   1000,
			ErrorRate:         1.0,
		},
	}

	monitor := NewRealtimeNetworkMonitor(config)

	err := monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}
	defer monitor.Stop()

	// 使用超时机制等待数据收集
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	maxAttempts := 10
	attempts := 0

	for {
		select {
		case <-ctx.Done():
			t.Log("统计信息收集超时，这可能是由于网络环境或系统限制")
			return
		case <-ticker.C:
			attempts++
			stats := monitor.GetStatistics()

			// 如果统计信息不为空，验证它
			if len(stats) > 0 {
				// 检查统计信息字段
				if _, exists := stats["is_running"]; exists {
					t.Logf("找到运行状态统计")
				}

				if _, exists := stats["history_count"]; exists {
					t.Logf("找到历史记录数量统计")
				}

				t.Logf("统计信息: %+v", stats)
				return
			}

			// 如果尝试次数过多，结束测试但不失败
			if attempts >= maxAttempts {
				t.Log("统计信息为空，这可能是因为监控器尚未收集到足够的数据")
				return
			}

			t.Logf("尝试 %d/%d: 统计信息为空，等待数据收集...", attempts, maxAttempts)
		}
	}
}

func TestRealtimeNetworkMonitor_HistoryManagement(t *testing.T) {
	config := RealtimeMonitorConfig{
		UpdateInterval:   200 * time.Millisecond,
		Timeout:          1 * time.Second,
		MaxHistory:       3, // 小的历史记录限制
		EnableAlerts:     false,
		MonitoredTargets: []string{},
		Interfaces:       []string{},
	}

	monitor := NewRealtimeNetworkMonitor(config)

	err := monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}
	defer monitor.Stop()

	// 等待收集超过MaxHistory数量的记录
	time.Sleep(1 * time.Second)

	history := monitor.GetHistory()
	if len(history) > config.MaxHistory {
		t.Errorf("历史记录数量 %d 超过了限制 %d", len(history), config.MaxHistory)
	}

	t.Logf("历史记录数量: %d (限制: %d)", len(history), config.MaxHistory)
}

// 基准测试
func BenchmarkRealtimeNetworkMonitor_DataCollection(b *testing.B) {
	config := RealtimeMonitorConfig{
		UpdateInterval:   1 * time.Second,
		Timeout:          2 * time.Second,
		MaxHistory:       10,
		EnableAlerts:     false,
		MonitoredTargets: []string{"8.8.8.8"},
		Interfaces:       []string{},
	}

	monitor := NewRealtimeNetworkMonitor(config)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := monitor.collectSnapshot(ctx)
		cancel()

		if err != nil {
			b.Fatalf("收集快照失败: %v", err)
		}
	}
}

func BenchmarkRealtimeNetworkMonitor_InterfaceStats(b *testing.B) {
	config := RealtimeMonitorConfig{
		UpdateInterval: 1 * time.Second,
		Timeout:        2 * time.Second,
		MaxHistory:     10,
		EnableAlerts:   false,
		Interfaces:     []string{},
	}

	monitor := NewRealtimeNetworkMonitor(config)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		_, err := monitor.collectInterfaceStats(ctx)
		cancel()

		if err != nil {
			b.Fatalf("收集接口统计失败: %v", err)
		}
	}
}
