/*
* @Author: Lzww0608
* @Date: 2025-6-6 23:47:35
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-8 20:05:34
* @Description: 优化版实时网络监控器 - 集成goroutine池和优雅关闭管理器
 */

package network

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	clixgosync "github.com/Lzww0608/ClixGo/pkg/sync"
	"go.uber.org/zap"
)

// OptimizedRealtimeNetworkMonitor 优化版实时网络监控器
// 使用goroutine池和优雅关闭管理器来提升并发性能
type OptimizedRealtimeNetworkMonitor struct {
	// 基础配置
	config RealtimeMonitorConfig
	logger *zap.Logger

	// 并发优化组件
	goroutinePool   *clixgosync.GoroutinePool
	shutdownManager *clixgosync.GracefulShutdownManager

	// 数据管理
	lastSnapshot *NetworkResourceSnapshot
	history      []NetworkResourceSnapshot
	historyMu    sync.RWMutex

	// 通道管理
	updateChan chan NetworkResourceSnapshot
	errorChan  chan error

	// 状态管理
	isRunning int32 // 使用原子操作
	startTime time.Time

	// 性能指标
	snapshotCount    uint64
	totalProcessTime time.Duration
	processTimeMu    sync.RWMutex
}

// NewOptimizedRealtimeNetworkMonitor 创建优化版网络监控器
func NewOptimizedRealtimeNetworkMonitor(config RealtimeMonitorConfig) *OptimizedRealtimeNetworkMonitor {
	logger, _ := zap.NewProduction()
	if logger == nil {
		logger = zap.NewNop()
	}

	// 设置默认值
	if config.UpdateInterval == 0 {
		config.UpdateInterval = 5 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxHistory == 0 {
		config.MaxHistory = 100
	}

	// 配置goroutine池
	poolConfig := clixgosync.DefaultGoroutinePoolConfig()
	poolConfig.MinWorkers = 3
	poolConfig.MaxWorkers = 12
	poolConfig.QueueSize = 200
	poolConfig.WorkerNamePrefix = "net-monitor"

	// 配置优雅关闭管理器
	shutdownConfig := clixgosync.DefaultShutdownConfig()
	shutdownConfig.ComponentTimeout = 30 * time.Second
	shutdownConfig.ParallelShutdown = true

	monitor := &OptimizedRealtimeNetworkMonitor{
		config:          config,
		logger:          logger,
		goroutinePool:   clixgosync.NewGoroutinePool(poolConfig),
		shutdownManager: clixgosync.NewGracefulShutdownManager(shutdownConfig),
		history:         make([]NetworkResourceSnapshot, 0, config.MaxHistory),
		updateChan:      make(chan NetworkResourceSnapshot, 50),
		errorChan:       make(chan error, 25),
		startTime:       time.Now(),
	}

	return monitor
}

// Start 启动优化版网络监控器
func (monitor *OptimizedRealtimeNetworkMonitor) Start() error {
	if !atomic.CompareAndSwapInt32(&monitor.isRunning, 0, 1) {
		return fmt.Errorf("网络监控器已在运行中")
	}

	monitor.logger.Info("启动优化版网络监控器")

	// 启动优雅关闭管理器
	if err := monitor.shutdownManager.Start(); err != nil {
		atomic.StoreInt32(&monitor.isRunning, 0)
		return fmt.Errorf("启动优雅关闭管理器失败: %w", err)
	}

	// 启动goroutine池
	if err := monitor.goroutinePool.Start(); err != nil {
		atomic.StoreInt32(&monitor.isRunning, 0)
		monitor.shutdownManager.Stop()
		return fmt.Errorf("启动goroutine池失败: %w", err)
	}

	// 注册组件到优雅关闭管理器
	poolComponent := &NetworkComponent{
		name: "goroutine-pool",
		pool: monitor.goroutinePool,
	}
	if err := monitor.shutdownManager.RegisterComponent(poolComponent); err != nil {
		monitor.logger.Error("注册goroutine池组件失败", zap.Error(err))
	}

	// 注册通道到优雅关闭管理器
	monitor.shutdownManager.RegisterChannel("update-chan", monitor.updateChan)
	monitor.shutdownManager.RegisterChannel("error-chan", monitor.errorChan)

	// 启动监控循环
	monitor.shutdownManager.RunManagedGoroutine("monitor-loop", monitor.monitorLoopOptimized)

	// 启动性能统计
	monitor.shutdownManager.RunManagedGoroutine("performance-stats", monitor.performanceStatsLoop)

	monitor.logger.Info("优化版网络监控器启动成功",
		zap.Duration("update_interval", monitor.config.UpdateInterval),
		zap.Int32("min_workers", int32(monitor.goroutinePool.GetMetrics().TotalWorkers)),
	)

	return nil
}

// Stop 停止优化版网络监控器
func (monitor *OptimizedRealtimeNetworkMonitor) Stop() error {
	if !atomic.CompareAndSwapInt32(&monitor.isRunning, 1, 0) {
		return fmt.Errorf("网络监控器未在运行")
	}

	monitor.logger.Info("停止优化版网络监控器")

	// 使用更短的超时时间，避免嵌套超时
	if err := monitor.shutdownManager.StopWithTimeout(10 * time.Second); err != nil {
		monitor.logger.Warn("优雅关闭超时，强制关闭", zap.Error(err))
	}

	// 强制关闭通道，确保等待的goroutine能够退出
	close(monitor.updateChan)
	close(monitor.errorChan)

	monitor.logger.Info("优化版网络监控器已停止")
	return nil
}

// monitorLoopOptimized 优化版监控循环
func (monitor *OptimizedRealtimeNetworkMonitor) monitorLoopOptimized(ctx context.Context) {
	ticker := time.NewTicker(monitor.config.UpdateInterval)
	defer ticker.Stop()

	monitor.logger.Info("监控循环已启动", zap.Duration("interval", monitor.config.UpdateInterval))

	for {
		select {
		case <-ctx.Done():
			monitor.logger.Info("监控循环接收到停止信号")
			return
		case <-ticker.C:
			// 使用goroutine池执行快照收集
			monitor.goroutinePool.SubmitFunc(
				fmt.Sprintf("snapshot-%d", atomic.AddUint64(&monitor.snapshotCount, 1)),
				func(poolCtx context.Context) error {
					return monitor.collectSnapshotAsync(poolCtx)
				})
		}
	}
}

// collectSnapshotAsync 异步收集网络快照
func (monitor *OptimizedRealtimeNetworkMonitor) collectSnapshotAsync(ctx context.Context) error {
	startTime := time.Now()
	defer func() {
		processDuration := time.Since(startTime)
		monitor.processTimeMu.Lock()
		monitor.totalProcessTime += processDuration
		monitor.processTimeMu.Unlock()
	}()

	// 使用超时上下文
	collectCtx, cancel := context.WithTimeout(ctx, monitor.config.Timeout)
	defer cancel()

	snapshot, err := monitor.collectSnapshotOptimized(collectCtx)
	if err != nil {
		monitor.sendErrorAsync(fmt.Errorf("收集网络快照失败: %w", err))
		return err
	}

	// 更新历史记录
	monitor.updateHistoryOptimized(snapshot)

	// 发送更新通知
	monitor.sendUpdateAsync(snapshot)

	return nil
}

// collectSnapshotOptimized 优化版快照收集
func (monitor *OptimizedRealtimeNetworkMonitor) collectSnapshotOptimized(ctx context.Context) (NetworkResourceSnapshot, error) {
	snapshot := NetworkResourceSnapshot{
		Timestamp:       time.Now(),
		Interfaces:      make(map[string]InterfaceStats),
		TargetLatencies: make(map[string]LatencyStats),
		Alerts:          make([]Alert, 0),
	}

	// 使用并发收集器
	type collector struct {
		interfaces       map[string]InterfaceStats
		connections      ConnectionSummary
		targetLatencies  map[string]LatencyStats
		systemResources  SystemNetworkResources
		interfacesReady  int32
		connectionsReady int32
		latenciesReady   int32
		resourcesReady   int32
	}

	coll := &collector{
		interfaces:      make(map[string]InterfaceStats),
		targetLatencies: make(map[string]LatencyStats),
	}

	// 并发收集接口统计
	monitor.goroutinePool.SubmitFunc("collect-interfaces", func(poolCtx context.Context) error {
		if interfaces, err := monitor.collectInterfaceStatsOptimized(poolCtx); err == nil {
			coll.interfaces = interfaces
			atomic.StoreInt32(&coll.interfacesReady, 1)
		}
		return nil
	})

	// 并发收集连接统计
	monitor.goroutinePool.SubmitFunc("collect-connections", func(poolCtx context.Context) error {
		if connections, err := monitor.collectConnectionStatsOptimized(poolCtx); err == nil {
			coll.connections = connections
			atomic.StoreInt32(&coll.connectionsReady, 1)
		}
		return nil
	})

	// 并发收集延迟统计（如果有目标）
	if len(monitor.config.MonitoredTargets) > 0 {
		monitor.goroutinePool.SubmitFunc("collect-latencies", func(poolCtx context.Context) error {
			if latencies, err := monitor.collectTargetLatenciesOptimized(poolCtx); err == nil {
				coll.targetLatencies = latencies
				atomic.StoreInt32(&coll.latenciesReady, 1)
			}
			return nil
		})
	} else {
		atomic.StoreInt32(&coll.latenciesReady, 1) // 标记为已完成
	}

	// 并发收集系统资源
	monitor.goroutinePool.SubmitFunc("collect-resources", func(poolCtx context.Context) error {
		if resources, err := monitor.collectSystemResourcesOptimized(poolCtx); err == nil {
			coll.systemResources = resources
			atomic.StoreInt32(&coll.resourcesReady, 1)
		}
		return nil
	})

	// 等待所有收集任务完成
	deadline := time.Now().Add(monitor.config.Timeout)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&coll.interfacesReady) == 1 &&
			atomic.LoadInt32(&coll.connectionsReady) == 1 &&
			atomic.LoadInt32(&coll.latenciesReady) == 1 &&
			atomic.LoadInt32(&coll.resourcesReady) == 1 {
			break
		}
		select {
		case <-ctx.Done():
			return snapshot, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}

	// 组装结果
	snapshot.Interfaces = coll.interfaces
	snapshot.Connections = coll.connections
	snapshot.TargetLatencies = coll.targetLatencies
	snapshot.SystemResources = coll.systemResources

	// 计算性能评分
	snapshot.PerformanceScore = monitor.calculatePerformanceScoreOptimized(snapshot)

	// 检查告警
	if monitor.config.EnableAlerts {
		snapshot.Alerts = monitor.checkAlertsOptimized(snapshot)
	}

	return snapshot, nil
}

// collectInterfaceStatsOptimized 优化版接口统计收集
func (monitor *OptimizedRealtimeNetworkMonitor) collectInterfaceStatsOptimized(ctx context.Context) (map[string]InterfaceStats, error) {
	interfaces := make(map[string]InterfaceStats)

	netInterfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range netInterfaces {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 如果配置了特定接口，只监控这些接口
		if len(monitor.config.Interfaces) > 0 {
			found := false
			for _, configIface := range monitor.config.Interfaces {
				if iface.Name == configIface {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		stats := InterfaceStats{
			Name:       iface.Name,
			IsUp:       iface.Flags&net.FlagUp != 0,
			MTU:        iface.MTU,
			LastUpdate: time.Now(),
		}

		// 获取流量统计（简化版）
		if runtime.GOOS == "linux" {
			// 在实际实现中，这里会读取 /proc/net/dev
			// 为了演示，我们使用模拟数据
			stats.BytesIn = 1024 * 1024 // 1MB
			stats.BytesOut = 512 * 1024 // 512KB
			stats.PacketsIn = 1000
			stats.PacketsOut = 800
		}

		// 计算带宽使用率
		monitor.historyMu.RLock()
		lastSnapshot := monitor.lastSnapshot
		monitor.historyMu.RUnlock()

		if lastSnapshot != nil {
			if lastStats, exists := lastSnapshot.Interfaces[iface.Name]; exists {
				timeDiff := stats.LastUpdate.Sub(lastStats.LastUpdate).Seconds()
				if timeDiff > 0 {
					bytesDiffIn := float64(stats.BytesIn - lastStats.BytesIn)
					bytesDiffOut := float64(stats.BytesOut - lastStats.BytesOut)

					stats.BandwidthInMbps = (bytesDiffIn * 8) / (timeDiff * 1024 * 1024)
					stats.BandwidthOutMbps = (bytesDiffOut * 8) / (timeDiff * 1024 * 1024)

					// 计算使用率（假设千兆网卡）
					if stats.Speed > 0 {
						totalBandwidth := stats.BandwidthInMbps + stats.BandwidthOutMbps
						maxBandwidth := float64(stats.Speed) / (1024 * 1024)
						stats.Utilization = (totalBandwidth / maxBandwidth) * 100
					}
				}
			}
		}

		interfaces[iface.Name] = stats
	}

	return interfaces, nil
}

// collectConnectionStatsOptimized 优化版连接统计收集
func (monitor *OptimizedRealtimeNetworkMonitor) collectConnectionStatsOptimized(_ context.Context) (ConnectionSummary, error) {
	// 简化版连接统计收集
	summary := ConnectionSummary{
		Total:       100,
		TCP:         80,
		UDP:         20,
		Established: 60,
		Listen:      10,
		TimeWait:    5,
		CloseWait:   5,
		ByState:     make(map[string]int),
		TopPorts:    make([]PortUsage, 0),
	}

	summary.ByState["ESTABLISHED"] = summary.Established
	summary.ByState["LISTEN"] = summary.Listen
	summary.ByState["TIME_WAIT"] = summary.TimeWait
	summary.ByState["CLOSE_WAIT"] = summary.CloseWait

	// 模拟一些常用端口
	summary.TopPorts = append(summary.TopPorts,
		PortUsage{Port: 80, Protocol: "tcp", Connections: 20, Service: "http"},
		PortUsage{Port: 443, Protocol: "tcp", Connections: 25, Service: "https"},
		PortUsage{Port: 22, Protocol: "tcp", Connections: 3, Service: "ssh"},
	)

	return summary, nil
}

// collectTargetLatenciesOptimized 优化版目标延迟收集
func (monitor *OptimizedRealtimeNetworkMonitor) collectTargetLatenciesOptimized(ctx context.Context) (map[string]LatencyStats, error) {
	latencies := make(map[string]LatencyStats)

	for _, target := range monitor.config.MonitoredTargets {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 简化版延迟统计
		stats := LatencyStats{
			Target:      target,
			MinLatency:  10 * time.Millisecond,
			MaxLatency:  100 * time.Millisecond,
			AvgLatency:  30 * time.Millisecond,
			PacketLoss:  0.0,
			Jitter:      5 * time.Millisecond,
			IsReachable: true,
			LastCheck:   time.Now(),
		}

		latencies[target] = stats
	}

	return latencies, nil
}

// collectSystemResourcesOptimized 优化版系统资源收集
func (monitor *OptimizedRealtimeNetworkMonitor) collectSystemResourcesOptimized(_ context.Context) (SystemNetworkResources, error) {
	// 简化版系统资源收集
	return SystemNetworkResources{
		OpenFiles:       100,
		MaxOpenFiles:    65536,
		SocketBuffers:   50,
		NetworkThreads:  10,
		MemoryUsageMB:   25.5,
		CPUUsagePercent: 2.3,
	}, nil
}

// 辅助方法
func (monitor *OptimizedRealtimeNetworkMonitor) calculatePerformanceScoreOptimized(snapshot NetworkResourceSnapshot) float64 {
	// 简化版性能评分计算
	score := 100.0

	// 基于延迟降分
	for _, latency := range snapshot.TargetLatencies {
		if latency.AvgLatency > 100*time.Millisecond {
			score -= 10
		}
		if latency.PacketLoss > 0 {
			score -= latency.PacketLoss * 50
		}
	}

	// 基于连接数降分
	if snapshot.Connections.Total > 500 {
		score -= 5
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (monitor *OptimizedRealtimeNetworkMonitor) checkAlertsOptimized(snapshot NetworkResourceSnapshot) []Alert {
	alerts := make([]Alert, 0)

	// 检查延迟告警
	for target, latency := range snapshot.TargetLatencies {
		if monitor.config.AlertThresholds.LatencyMs > 0 &&
			latency.AvgLatency.Seconds()*1000 > monitor.config.AlertThresholds.LatencyMs {
			alerts = append(alerts, Alert{
				ID:        fmt.Sprintf("latency-%s-%d", target, time.Now().Unix()),
				Type:      "latency",
				Severity:  "warning",
				Message:   fmt.Sprintf("高延迟告警: %s 平均延迟 %.2fms", target, latency.AvgLatency.Seconds()*1000),
				Target:    target,
				Value:     latency.AvgLatency.Seconds() * 1000,
				Threshold: monitor.config.AlertThresholds.LatencyMs,
				Timestamp: time.Now(),
			})
		}
	}

	return alerts
}

func (monitor *OptimizedRealtimeNetworkMonitor) updateHistoryOptimized(snapshot NetworkResourceSnapshot) {
	monitor.historyMu.Lock()
	defer monitor.historyMu.Unlock()

	monitor.lastSnapshot = &snapshot
	monitor.history = append(monitor.history, snapshot)

	// 限制历史记录数量
	if len(monitor.history) > monitor.config.MaxHistory {
		monitor.history = monitor.history[len(monitor.history)-monitor.config.MaxHistory:]
	}
}

func (monitor *OptimizedRealtimeNetworkMonitor) sendUpdateAsync(snapshot NetworkResourceSnapshot) {
	// 检查是否仍在运行
	if atomic.LoadInt32(&monitor.isRunning) == 0 {
		return
	}

	monitor.goroutinePool.SubmitFunc("send-update", func(ctx context.Context) error {
		defer func() {
			if r := recover(); r != nil {
				// 忽略向已关闭通道发送数据的panic
			}
		}()

		select {
		case monitor.updateChan <- snapshot:
		case <-ctx.Done():
		case <-time.After(100 * time.Millisecond):
			// 防止阻塞
		}
		return nil
	})
}

func (monitor *OptimizedRealtimeNetworkMonitor) sendErrorAsync(err error) {
	// 检查是否仍在运行
	if atomic.LoadInt32(&monitor.isRunning) == 0 {
		return
	}

	monitor.goroutinePool.SubmitFunc("send-error", func(ctx context.Context) error {
		defer func() {
			if r := recover(); r != nil {
				// 忽略向已关闭通道发送数据的panic
			}
		}()

		select {
		case monitor.errorChan <- err:
		case <-ctx.Done():
		case <-time.After(100 * time.Millisecond):
			// 防止阻塞
		}
		return nil
	})
}

// performanceStatsLoop 性能统计循环
func (monitor *OptimizedRealtimeNetworkMonitor) performanceStatsLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 输出性能统计
			poolMetrics := monitor.goroutinePool.GetMetrics()

			monitor.processTimeMu.RLock()
			avgProcessTime := time.Duration(0)
			if monitor.snapshotCount > 0 {
				avgProcessTime = time.Duration(int64(monitor.totalProcessTime) / int64(monitor.snapshotCount))
			}
			monitor.processTimeMu.RUnlock()

			monitor.logger.Info("网络监控器性能统计",
				zap.Uint64("snapshot_count", atomic.LoadUint64(&monitor.snapshotCount)),
				zap.Duration("avg_process_time", avgProcessTime),
				zap.Int32("active_workers", poolMetrics.ActiveWorkers),
				zap.Int32("pending_tasks", poolMetrics.PendingTasks),
				zap.Uint64("completed_tasks", poolMetrics.CompletedTasks),
			)
		}
	}
}

// 公共方法
func (monitor *OptimizedRealtimeNetworkMonitor) IsRunning() bool {
	return atomic.LoadInt32(&monitor.isRunning) == 1
}

func (monitor *OptimizedRealtimeNetworkMonitor) GetUpdateChannel() <-chan NetworkResourceSnapshot {
	return monitor.updateChan
}

func (monitor *OptimizedRealtimeNetworkMonitor) GetErrorChannel() <-chan error {
	return monitor.errorChan
}

func (monitor *OptimizedRealtimeNetworkMonitor) GetCurrentSnapshot() *NetworkResourceSnapshot {
	monitor.historyMu.RLock()
	defer monitor.historyMu.RUnlock()
	if monitor.lastSnapshot == nil {
		return nil
	}
	snapshot := *monitor.lastSnapshot
	return &snapshot
}

func (monitor *OptimizedRealtimeNetworkMonitor) GetHistory() []NetworkResourceSnapshot {
	monitor.historyMu.RLock()
	defer monitor.historyMu.RUnlock()
	history := make([]NetworkResourceSnapshot, len(monitor.history))
	copy(history, monitor.history)
	return history
}

func (monitor *OptimizedRealtimeNetworkMonitor) GetPoolMetrics() clixgosync.PoolMetrics {
	return monitor.goroutinePool.GetMetrics()
}

// NetworkComponent 网络组件适配器
type NetworkComponent struct {
	name string
	pool *clixgosync.GoroutinePool
}

func (nc *NetworkComponent) Name() string {
	return nc.name
}

func (nc *NetworkComponent) Start(ctx context.Context) error {
	return nc.pool.Start()
}

func (nc *NetworkComponent) Stop(ctx context.Context) error {
	return nc.pool.Stop()
}

func (nc *NetworkComponent) State() clixgosync.ComponentState {
	if nc.pool.IsRunning() {
		return clixgosync.StateRunning
	}
	return clixgosync.StateStopped
}
