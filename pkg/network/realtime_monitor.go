package network

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RealtimeNetworkMonitor 实时网络资源监控器
type RealtimeNetworkMonitor struct {
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	isRunning    bool
	updateChan   chan NetworkResourceSnapshot
	errorChan    chan error
	config       RealtimeMonitorConfig
	lastSnapshot *NetworkResourceSnapshot
	history      []NetworkResourceSnapshot
	maxHistory   int
	alertManager *AlertManager
}

// RealtimeMonitorConfig 实时监控配置
type RealtimeMonitorConfig struct {
	UpdateInterval   time.Duration   `json:"update_interval"`   // 更新间隔
	Timeout          time.Duration   `json:"timeout"`           // 操作超时时间
	MaxHistory       int             `json:"max_history"`       // 最大历史记录数
	EnableAlerts     bool            `json:"enable_alerts"`     // 是否启用告警
	AlertThresholds  AlertThresholds `json:"alert_thresholds"`  // 告警阈值
	MonitoredTargets []string        `json:"monitored_targets"` // 监控目标
	Interfaces       []string        `json:"interfaces"`        // 监控的网络接口
}

// AlertThresholds 告警阈值配置
type AlertThresholds struct {
	LatencyMs         float64 `json:"latency_ms"`          // 延迟阈值(毫秒)
	PacketLossPercent float64 `json:"packet_loss_percent"` // 丢包率阈值(百分比)
	BandwidthMbps     float64 `json:"bandwidth_mbps"`      // 带宽使用阈值(Mbps)
	ConnectionCount   int     `json:"connection_count"`    // 连接数阈值
	ErrorRate         float64 `json:"error_rate"`          // 错误率阈值(百分比)
}

// NetworkResourceSnapshot 网络资源快照
type NetworkResourceSnapshot struct {
	Timestamp        time.Time                 `json:"timestamp"`
	Interfaces       map[string]InterfaceStats `json:"interfaces"`
	Connections      ConnectionSummary         `json:"connections"`
	TargetLatencies  map[string]LatencyStats   `json:"target_latencies"`
	SystemResources  SystemNetworkResources    `json:"system_resources"`
	Alerts           []Alert                   `json:"alerts"`
	PerformanceScore float64                   `json:"performance_score"`
}

// InterfaceStats 网络接口统计信息
type InterfaceStats struct {
	Name             string    `json:"name"`
	IsUp             bool      `json:"is_up"`
	Speed            uint64    `json:"speed"` // 接口速度(bps)
	MTU              int       `json:"mtu"`
	BytesIn          uint64    `json:"bytes_in"`
	BytesOut         uint64    `json:"bytes_out"`
	PacketsIn        uint64    `json:"packets_in"`
	PacketsOut       uint64    `json:"packets_out"`
	ErrorsIn         uint64    `json:"errors_in"`
	ErrorsOut        uint64    `json:"errors_out"`
	DropsIn          uint64    `json:"drops_in"`
	DropsOut         uint64    `json:"drops_out"`
	BandwidthInMbps  float64   `json:"bandwidth_in_mbps"`
	BandwidthOutMbps float64   `json:"bandwidth_out_mbps"`
	Utilization      float64   `json:"utilization"` // 使用率(百分比)
	LastUpdate       time.Time `json:"last_update"`
}

// ConnectionSummary 连接汇总信息
type ConnectionSummary struct {
	Total       int            `json:"total"`
	TCP         int            `json:"tcp"`
	UDP         int            `json:"udp"`
	Established int            `json:"established"`
	Listen      int            `json:"listen"`
	TimeWait    int            `json:"time_wait"`
	CloseWait   int            `json:"close_wait"`
	ByState     map[string]int `json:"by_state"`
	TopPorts    []PortUsage    `json:"top_ports"`
}

// PortUsage 端口使用情况
type PortUsage struct {
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	Connections int    `json:"connections"`
	Service     string `json:"service,omitempty"`
}

// LatencyStats 延迟统计信息
type LatencyStats struct {
	Target      string        `json:"target"`
	MinLatency  time.Duration `json:"min_latency"`
	MaxLatency  time.Duration `json:"max_latency"`
	AvgLatency  time.Duration `json:"avg_latency"`
	PacketLoss  float64       `json:"packet_loss"`
	Jitter      time.Duration `json:"jitter"`
	IsReachable bool          `json:"is_reachable"`
	LastCheck   time.Time     `json:"last_check"`
}

// SystemNetworkResources 系统网络资源信息
type SystemNetworkResources struct {
	OpenFiles       int     `json:"open_files"`
	MaxOpenFiles    int     `json:"max_open_files"`
	SocketBuffers   int     `json:"socket_buffers"`
	NetworkThreads  int     `json:"network_threads"`
	MemoryUsageMB   float64 `json:"memory_usage_mb"`
	CPUUsagePercent float64 `json:"cpu_usage_percent"`
}

// Alert 告警信息
type Alert struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Severity     string    `json:"severity"`
	Message      string    `json:"message"`
	Target       string    `json:"target,omitempty"`
	Value        float64   `json:"value"`
	Threshold    float64   `json:"threshold"`
	Timestamp    time.Time `json:"timestamp"`
	Acknowledged bool      `json:"acknowledged"`
}

// NewRealtimeNetworkMonitor 创建新的实时网络监控器
func NewRealtimeNetworkMonitor(config RealtimeMonitorConfig) *RealtimeNetworkMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	// 设置默认值
	if config.UpdateInterval == 0 {
		config.UpdateInterval = 2 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.MaxHistory == 0 {
		config.MaxHistory = 100
	}

	monitor := &RealtimeNetworkMonitor{
		ctx:        ctx,
		cancel:     cancel,
		updateChan: make(chan NetworkResourceSnapshot, 10),
		errorChan:  make(chan error, 10),
		config:     config,
		history:    make([]NetworkResourceSnapshot, 0, config.MaxHistory),
		maxHistory: config.MaxHistory,
	}

	if config.EnableAlerts {
		alertConfig := AlertConfig{
			Enabled:     true,
			Threshold:   config.AlertThresholds.LatencyMs,
			RepeatAfter: 5 * time.Minute,
		}
		monitor.alertManager = NewAlertManager(alertConfig)
	}

	return monitor
}

// Start 启动实时监控
func (m *RealtimeNetworkMonitor) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("监控器已在运行中")
	}

	m.isRunning = true

	// 启动监控协程
	go m.monitorLoop()

	return nil
}

// Stop 停止实时监控
func (m *RealtimeNetworkMonitor) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return fmt.Errorf("监控器未在运行")
	}

	m.isRunning = false

	// 先取消上下文，让监控循环退出
	m.cancel()

	// 等待一小段时间让监控循环完全退出
	time.Sleep(50 * time.Millisecond)

	// 安全地关闭通道
	select {
	case <-m.updateChan:
		// 清空通道
	default:
	}
	close(m.updateChan)

	select {
	case <-m.errorChan:
		// 清空通道
	default:
	}
	close(m.errorChan)

	return nil
}

// GetUpdateChannel 获取更新通道
func (m *RealtimeNetworkMonitor) GetUpdateChannel() <-chan NetworkResourceSnapshot {
	return m.updateChan
}

// GetErrorChannel 获取错误通道
func (m *RealtimeNetworkMonitor) GetErrorChannel() <-chan error {
	return m.errorChan
}

// GetCurrentSnapshot 获取当前快照
func (m *RealtimeNetworkMonitor) GetCurrentSnapshot() *NetworkResourceSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.lastSnapshot == nil {
		return nil
	}

	// 返回副本以避免并发修改
	snapshot := *m.lastSnapshot
	return &snapshot
}

// GetHistory 获取历史记录
func (m *RealtimeNetworkMonitor) GetHistory() []NetworkResourceSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回副本
	history := make([]NetworkResourceSnapshot, len(m.history))
	copy(history, m.history)
	return history
}

// IsRunning 检查是否正在运行
func (m *RealtimeNetworkMonitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// monitorLoop 监控主循环
func (m *RealtimeNetworkMonitor) monitorLoop() {
	ticker := time.NewTicker(m.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			// 使用超时上下文防止死锁
			ctx, cancel := context.WithTimeout(m.ctx, m.config.Timeout)
			snapshot, err := m.collectSnapshot(ctx)
			cancel()

			if err != nil {
				select {
				case m.errorChan <- err:
				case <-m.ctx.Done():
					return
				default:
					// 错误通道满了，丢弃错误
				}
				continue
			}

			// 更新历史记录
			m.updateHistory(snapshot)

			// 发送更新
			select {
			case m.updateChan <- snapshot:
			case <-m.ctx.Done():
				return
			default:
				// 更新通道满了，丢弃旧数据
			}
		}
	}
}

// collectSnapshot 收集网络资源快照
func (m *RealtimeNetworkMonitor) collectSnapshot(ctx context.Context) (NetworkResourceSnapshot, error) {
	snapshot := NetworkResourceSnapshot{
		Timestamp:       time.Now(),
		Interfaces:      make(map[string]InterfaceStats),
		TargetLatencies: make(map[string]LatencyStats),
		Alerts:          make([]Alert, 0),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	// 并发收集各种数据，使用超时防止死锁

	// 收集接口统计信息
	wg.Add(1)
	go func() {
		defer wg.Done()
		interfaces, err := m.collectInterfaceStats(ctx)
		if err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("收集接口统计失败: %w", err))
			mu.Unlock()
			return
		}
		mu.Lock()
		snapshot.Interfaces = interfaces
		mu.Unlock()
	}()

	// 收集连接信息
	wg.Add(1)
	go func() {
		defer wg.Done()
		connections, err := m.collectConnectionStats(ctx)
		if err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("收集连接统计失败: %w", err))
			mu.Unlock()
			return
		}
		mu.Lock()
		snapshot.Connections = connections
		mu.Unlock()
	}()

	// 收集目标延迟信息
	if len(m.config.MonitoredTargets) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			latencies, err := m.collectTargetLatencies(ctx)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("收集延迟统计失败: %w", err))
				mu.Unlock()
				return
			}
			mu.Lock()
			snapshot.TargetLatencies = latencies
			mu.Unlock()
		}()
	}

	// 收集系统资源信息
	wg.Add(1)
	go func() {
		defer wg.Done()
		resources, err := m.collectSystemResources(ctx)
		if err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("收集系统资源失败: %w", err))
			mu.Unlock()
			return
		}
		mu.Lock()
		snapshot.SystemResources = resources
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
		return snapshot, fmt.Errorf("数据收集超时")
	}

	// 检查是否有错误
	if len(errors) > 0 {
		return snapshot, fmt.Errorf("部分数据收集失败: %v", errors)
	}

	// 计算性能评分
	snapshot.PerformanceScore = m.calculatePerformanceScore(snapshot)

	// 检查告警
	if m.config.EnableAlerts {
		snapshot.Alerts = m.checkAlerts(snapshot)
	}

	return snapshot, nil
}

// collectInterfaceStats 收集网络接口统计信息
func (m *RealtimeNetworkMonitor) collectInterfaceStats(ctx context.Context) (map[string]InterfaceStats, error) {
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
		if len(m.config.Interfaces) > 0 {
			found := false
			for _, configIface := range m.config.Interfaces {
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

		// 获取流量统计（Linux特定实现）
		if runtime.GOOS == "linux" {
			if err := m.getLinuxInterfaceStats(&stats); err != nil {
				// 记录错误但继续处理其他接口
				continue
			}
		}

		// 计算带宽使用率
		if m.lastSnapshot != nil {
			if lastStats, exists := m.lastSnapshot.Interfaces[iface.Name]; exists {
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

// getLinuxInterfaceStats 获取Linux系统的接口统计信息
func (m *RealtimeNetworkMonitor) getLinuxInterfaceStats(stats *InterfaceStats) error {
	// 读取 /proc/net/dev 文件
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.Contains(line, stats.Name+":") {
			fields := strings.Fields(line)
			if len(fields) >= 17 {
				// 解析统计数据
				if val, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					stats.BytesIn = val
				}
				if val, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
					stats.PacketsIn = val
				}
				if val, err := strconv.ParseUint(fields[3], 10, 64); err == nil {
					stats.ErrorsIn = val
				}
				if val, err := strconv.ParseUint(fields[4], 10, 64); err == nil {
					stats.DropsIn = val
				}
				if val, err := strconv.ParseUint(fields[9], 10, 64); err == nil {
					stats.BytesOut = val
				}
				if val, err := strconv.ParseUint(fields[10], 10, 64); err == nil {
					stats.PacketsOut = val
				}
				if val, err := strconv.ParseUint(fields[11], 10, 64); err == nil {
					stats.ErrorsOut = val
				}
				if val, err := strconv.ParseUint(fields[12], 10, 64); err == nil {
					stats.DropsOut = val
				}
			}
			break
		}
	}

	return nil
}

// collectConnectionStats 收集连接统计信息
func (m *RealtimeNetworkMonitor) collectConnectionStats(ctx context.Context) (ConnectionSummary, error) {
	summary := ConnectionSummary{
		ByState:  make(map[string]int),
		TopPorts: make([]PortUsage, 0),
	}

	// 检查上下文
	select {
	case <-ctx.Done():
		return summary, ctx.Err()
	default:
	}

	// 获取TCP连接
	if err := m.getTCPConnections(&summary); err != nil {
		return summary, err
	}

	// 获取UDP连接
	if err := m.getUDPConnections(&summary); err != nil {
		return summary, err
	}

	// 计算总数
	summary.Total = summary.TCP + summary.UDP

	return summary, nil
}

// getTCPConnections 获取TCP连接信息
func (m *RealtimeNetworkMonitor) getTCPConnections(summary *ConnectionSummary) error {
	if runtime.GOOS != "linux" {
		return nil // 暂时只支持Linux
	}

	data, err := os.ReadFile("/proc/net/tcp")
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	portCount := make(map[int]int)

	for i, line := range lines {
		if i == 0 || line == "" {
			continue // 跳过标题行和空行
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		// 解析状态
		state := fields[3]
		switch state {
		case "01":
			summary.Established++
			summary.ByState["ESTABLISHED"]++
		case "0A":
			summary.Listen++
			summary.ByState["LISTEN"]++
		case "06":
			summary.TimeWait++
			summary.ByState["TIME_WAIT"]++
		case "08":
			summary.CloseWait++
			summary.ByState["CLOSE_WAIT"]++
		}

		// 解析本地端口
		localAddr := fields[1]
		if colonIndex := strings.LastIndex(localAddr, ":"); colonIndex != -1 {
			portHex := localAddr[colonIndex+1:]
			if port, err := strconv.ParseInt(portHex, 16, 32); err == nil {
				portCount[int(port)]++
			}
		}

		summary.TCP++
	}

	// 生成端口使用排行
	summary.TopPorts = m.generateTopPorts(portCount, "tcp")

	return nil
}

// getUDPConnections 获取UDP连接信息
func (m *RealtimeNetworkMonitor) getUDPConnections(summary *ConnectionSummary) error {
	if runtime.GOOS != "linux" {
		return nil // 暂时只支持Linux
	}

	data, err := os.ReadFile("/proc/net/udp")
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 4 {
			summary.UDP++
		}
	}

	return nil
}

// generateTopPorts 生成端口使用排行
func (m *RealtimeNetworkMonitor) generateTopPorts(portCount map[int]int, protocol string) []PortUsage {
	type portStat struct {
		port  int
		count int
	}

	stats := make([]portStat, 0, len(portCount))
	for port, count := range portCount {
		stats = append(stats, portStat{port: port, count: count})
	}

	// 按连接数排序
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].count > stats[j].count
	})

	// 取前10个
	topPorts := make([]PortUsage, 0, 10)
	for i, stat := range stats {
		if i >= 10 {
			break
		}

		usage := PortUsage{
			Port:        stat.port,
			Protocol:    protocol,
			Connections: stat.count,
			Service:     m.getServiceName(stat.port),
		}
		topPorts = append(topPorts, usage)
	}

	return topPorts
}

// getServiceName 根据端口号获取服务名称
func (m *RealtimeNetworkMonitor) getServiceName(port int) string {
	wellKnownPorts := map[int]string{
		22:   "SSH",
		23:   "Telnet",
		25:   "SMTP",
		53:   "DNS",
		80:   "HTTP",
		110:  "POP3",
		143:  "IMAP",
		443:  "HTTPS",
		993:  "IMAPS",
		995:  "POP3S",
		3306: "MySQL",
		5432: "PostgreSQL",
		6379: "Redis",
		8080: "HTTP-Alt",
		9200: "Elasticsearch",
	}

	if service, exists := wellKnownPorts[port]; exists {
		return service
	}
	return ""
}

// collectTargetLatencies 收集目标延迟信息
func (m *RealtimeNetworkMonitor) collectTargetLatencies(ctx context.Context) (map[string]LatencyStats, error) {
	latencies := make(map[string]LatencyStats)

	for _, target := range m.config.MonitoredTargets {
		// 检查上下文
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 使用超时上下文进行ping测试
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		result, err := m.pingWithContext(pingCtx, target)
		cancel()

		stats := LatencyStats{
			Target:    target,
			LastCheck: time.Now(),
		}

		if err != nil {
			stats.IsReachable = false
		} else {
			stats.IsReachable = true
			stats.MinLatency = result.MinRtt
			stats.MaxLatency = result.MaxRtt
			stats.AvgLatency = result.AvgRtt
			stats.PacketLoss = result.PacketLoss
			stats.Jitter = result.StdDevRtt
		}

		latencies[target] = stats
	}

	return latencies, nil
}

// pingWithContext 带上下文的ping测试
func (m *RealtimeNetworkMonitor) pingWithContext(ctx context.Context, target string) (*PingResult, error) {
	// 使用现有的Ping函数，但添加上下文支持
	done := make(chan struct{})
	var result *PingResult
	var err error

	go func() {
		defer close(done)
		result, err = Ping(target, 3, 2*time.Second)
	}()

	select {
	case <-done:
		return result, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// collectSystemResources 收集系统网络资源信息
func (m *RealtimeNetworkMonitor) collectSystemResources(ctx context.Context) (SystemNetworkResources, error) {
	resources := SystemNetworkResources{}

	// 检查上下文
	select {
	case <-ctx.Done():
		return resources, ctx.Err()
	default:
	}

	// 获取打开文件数（Linux特定）
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/sys/fs/file-nr"); err == nil {
			fields := strings.Fields(string(data))
			if len(fields) >= 3 {
				if val, err := strconv.Atoi(fields[0]); err == nil {
					resources.OpenFiles = val
				}
				if val, err := strconv.Atoi(fields[2]); err == nil {
					resources.MaxOpenFiles = val
				}
			}
		}

		// 获取网络相关的线程数
		resources.NetworkThreads = runtime.NumGoroutine()
	}

	// 获取内存使用情况
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	resources.MemoryUsageMB = float64(memStats.Alloc) / 1024 / 1024

	return resources, nil
}

// calculatePerformanceScore 计算性能评分
func (m *RealtimeNetworkMonitor) calculatePerformanceScore(snapshot NetworkResourceSnapshot) float64 {
	score := 100.0

	// 基于延迟评分
	for _, latency := range snapshot.TargetLatencies {
		if latency.IsReachable {
			latencyMs := float64(latency.AvgLatency.Nanoseconds()) / 1e6
			if latencyMs > 100 {
				score -= 10
			} else if latencyMs > 50 {
				score -= 5
			}

			// 基于丢包率评分
			if latency.PacketLoss > 5 {
				score -= 20
			} else if latency.PacketLoss > 1 {
				score -= 10
			}
		} else {
			score -= 30 // 不可达扣分更多
		}
	}

	// 基于接口错误率评分
	for _, iface := range snapshot.Interfaces {
		if iface.IsUp {
			totalPackets := iface.PacketsIn + iface.PacketsOut
			totalErrors := iface.ErrorsIn + iface.ErrorsOut
			if totalPackets > 0 {
				errorRate := float64(totalErrors) / float64(totalPackets) * 100
				if errorRate > 1 {
					score -= 15
				} else if errorRate > 0.1 {
					score -= 5
				}
			}
		}
	}

	// 基于连接数评分
	if snapshot.Connections.Total > 10000 {
		score -= 10
	} else if snapshot.Connections.Total > 5000 {
		score -= 5
	}

	if score < 0 {
		score = 0
	}

	return score
}

// checkAlerts 检查告警条件
func (m *RealtimeNetworkMonitor) checkAlerts(snapshot NetworkResourceSnapshot) []Alert {
	alerts := make([]Alert, 0)

	// 检查延迟告警
	for target, latency := range snapshot.TargetLatencies {
		if latency.IsReachable {
			latencyMs := float64(latency.AvgLatency.Nanoseconds()) / 1e6
			if latencyMs > m.config.AlertThresholds.LatencyMs {
				alert := Alert{
					ID:        fmt.Sprintf("latency_%s_%d", target, time.Now().Unix()),
					Type:      "latency",
					Severity:  "warning",
					Message:   fmt.Sprintf("目标 %s 延迟过高: %.2fms", target, latencyMs),
					Target:    target,
					Value:     latencyMs,
					Threshold: m.config.AlertThresholds.LatencyMs,
					Timestamp: time.Now(),
				}
				alerts = append(alerts, alert)
			}

			// 检查丢包告警
			if latency.PacketLoss > m.config.AlertThresholds.PacketLossPercent {
				alert := Alert{
					ID:        fmt.Sprintf("packetloss_%s_%d", target, time.Now().Unix()),
					Type:      "packet_loss",
					Severity:  "critical",
					Message:   fmt.Sprintf("目标 %s 丢包率过高: %.2f%%", target, latency.PacketLoss),
					Target:    target,
					Value:     latency.PacketLoss,
					Threshold: m.config.AlertThresholds.PacketLossPercent,
					Timestamp: time.Now(),
				}
				alerts = append(alerts, alert)
			}
		}
	}

	// 检查带宽使用告警
	for ifaceName, iface := range snapshot.Interfaces {
		totalBandwidth := iface.BandwidthInMbps + iface.BandwidthOutMbps
		if totalBandwidth > m.config.AlertThresholds.BandwidthMbps {
			alert := Alert{
				ID:        fmt.Sprintf("bandwidth_%s_%d", ifaceName, time.Now().Unix()),
				Type:      "bandwidth",
				Severity:  "warning",
				Message:   fmt.Sprintf("接口 %s 带宽使用过高: %.2f Mbps", ifaceName, totalBandwidth),
				Target:    ifaceName,
				Value:     totalBandwidth,
				Threshold: m.config.AlertThresholds.BandwidthMbps,
				Timestamp: time.Now(),
			}
			alerts = append(alerts, alert)
		}
	}

	// 检查连接数告警
	if snapshot.Connections.Total > m.config.AlertThresholds.ConnectionCount {
		alert := Alert{
			ID:        fmt.Sprintf("connections_%d", time.Now().Unix()),
			Type:      "connections",
			Severity:  "warning",
			Message:   fmt.Sprintf("连接数过多: %d", snapshot.Connections.Total),
			Value:     float64(snapshot.Connections.Total),
			Threshold: float64(m.config.AlertThresholds.ConnectionCount),
			Timestamp: time.Now(),
		}
		alerts = append(alerts, alert)
	}

	return alerts
}

// updateHistory 更新历史记录
func (m *RealtimeNetworkMonitor) updateHistory(snapshot NetworkResourceSnapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 更新当前快照
	m.lastSnapshot = &snapshot

	// 添加到历史记录
	m.history = append(m.history, snapshot)

	// 保持历史记录在限制范围内
	if len(m.history) > m.maxHistory {
		m.history = m.history[len(m.history)-m.maxHistory:]
	}
}

// GetStatistics 获取统计信息
func (m *RealtimeNetworkMonitor) GetStatistics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})

	if len(m.history) == 0 {
		return stats
	}

	// 计算平均性能评分
	totalScore := 0.0
	for _, snapshot := range m.history {
		totalScore += snapshot.PerformanceScore
	}
	stats["average_performance_score"] = totalScore / float64(len(m.history))

	// 计算告警统计
	alertCounts := make(map[string]int)
	for _, snapshot := range m.history {
		for _, alert := range snapshot.Alerts {
			alertCounts[alert.Type]++
		}
	}
	stats["alert_counts"] = alertCounts

	// 历史记录数量
	stats["history_count"] = len(m.history)
	stats["is_running"] = m.isRunning

	return stats
}
