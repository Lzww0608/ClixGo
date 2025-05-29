/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 终端性能监控功能的实现
 */

package terminal

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"go.uber.org/zap"
)

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	config          *TerminalConfig
	sessionManager  *SessionManager
	metrics         *PerformanceMetrics
	running         bool
	interval        time.Duration
	mutex           sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	alertThresholds *AlertThresholds
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	// 系统资源
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    int64   `json:"memory_usage"`
	GoroutineCount int     `json:"goroutine_count"`

	// 终端相关
	ActiveSessions int `json:"active_sessions"`
	TotalWindows   int `json:"total_windows"`
	TotalPanes     int `json:"total_panes"`
	ActivePTYs     int `json:"active_ptys"`

	// 性能统计
	AvgResponseTime time.Duration `json:"avg_response_time"`
	TotalCommands   int64         `json:"total_commands"`
	ErrorCount      int64         `json:"error_count"`

	// 网络统计
	ConnectionCount int   `json:"connection_count"`
	DataTransferred int64 `json:"data_transferred"`

	// 时间戳
	Timestamp time.Time     `json:"timestamp"`
	Uptime    time.Duration `json:"uptime"`

	mutex sync.RWMutex
}

// AlertThresholds 警报阈值
type AlertThresholds struct {
	MaxCPUUsage     float64       `json:"max_cpu_usage"`
	MaxMemoryUsage  int64         `json:"max_memory_usage"`
	MaxGoroutines   int           `json:"max_goroutines"`
	MaxResponseTime time.Duration `json:"max_response_time"`
	MaxErrorRate    float64       `json:"max_error_rate"`
}

// Alert 警报
type Alert struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Severity  string    `json:"severity"`
	Timestamp time.Time `json:"timestamp"`
}

// 默认警报阈值
var DefaultAlertThresholds = &AlertThresholds{
	MaxCPUUsage:     80.0,              // 80% CPU使用率
	MaxMemoryUsage:  100 * 1024 * 1024, // 100MB 内存使用
	MaxGoroutines:   1000,              // 1000个goroutines
	MaxResponseTime: time.Second * 5,   // 5秒响应时间
	MaxErrorRate:    5.0,               // 5% 错误率
}

// NewPerformanceMonitor 创建性能监控器
func NewPerformanceMonitor(config *TerminalConfig, sessionManager *SessionManager) *PerformanceMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &PerformanceMonitor{
		config:          config,
		sessionManager:  sessionManager,
		metrics:         NewPerformanceMetrics(),
		running:         false,
		interval:        time.Second * 5, // 默认5秒监控间隔
		ctx:             ctx,
		cancel:          cancel,
		alertThresholds: DefaultAlertThresholds,
	}
}

// NewPerformanceMetrics 创建性能指标
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		Timestamp: time.Now(),
	}
}

// Start 启动性能监控
func (pm *PerformanceMonitor) Start() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.running {
		return fmt.Errorf("performance monitor is already running")
	}

	pm.running = true
	pm.metrics.Timestamp = time.Now()

	// 启动监控goroutine
	go pm.monitor()

	logger.Info("Performance monitor started",
		zap.Duration("interval", pm.interval))

	return nil
}

// Stop 停止性能监控
func (pm *PerformanceMonitor) Stop() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if !pm.running {
		return fmt.Errorf("performance monitor is not running")
	}

	pm.running = false
	pm.cancel()

	logger.Info("Performance monitor stopped")
	return nil
}

// monitor 监控主循环
func (pm *PerformanceMonitor) monitor() {
	ticker := time.NewTicker(pm.interval)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.collectMetrics(startTime)
			pm.checkAlerts()
		}
	}
}

// collectMetrics 收集性能指标
func (pm *PerformanceMonitor) collectMetrics(startTime time.Time) {
	pm.metrics.mutex.Lock()
	defer pm.metrics.mutex.Unlock()

	// 更新时间戳和运行时间
	pm.metrics.Timestamp = time.Now()
	pm.metrics.Uptime = time.Since(startTime)

	// 收集系统资源指标
	pm.collectSystemMetrics()

	// 收集终端相关指标
	pm.collectTerminalMetrics()

	// 记录日志（每分钟记录一次详细信息）
	if int(pm.metrics.Uptime.Seconds())%60 == 0 {
		pm.logMetrics()
	}
}

// collectSystemMetrics 收集系统资源指标
func (pm *PerformanceMonitor) collectSystemMetrics() {
	// 获取内存统计
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	pm.metrics.MemoryUsage = int64(m.Alloc)
	pm.metrics.GoroutineCount = runtime.NumGoroutine()

	// CPU使用率需要更复杂的计算，这里简化处理
	// 在实际应用中可以使用第三方库如 gopsutil
	pm.metrics.CPUUsage = pm.estimateCPUUsage()
}

// collectTerminalMetrics 收集终端相关指标
func (pm *PerformanceMonitor) collectTerminalMetrics() {
	sessions := pm.sessionManager.ListSessions()
	pm.metrics.ActiveSessions = len(sessions)

	totalWindows := 0
	totalPanes := 0
	activePTYs := 0

	for _, session := range sessions {
		totalWindows += len(session.Windows)
		for _, window := range session.Windows {
			totalPanes += len(window.Panes)
			for _, pane := range window.Panes {
				if pane.ProcessID > 0 {
					activePTYs++
				}
			}
		}
	}

	pm.metrics.TotalWindows = totalWindows
	pm.metrics.TotalPanes = totalPanes
	pm.metrics.ActivePTYs = activePTYs
}

// estimateCPUUsage 估算CPU使用率
func (pm *PerformanceMonitor) estimateCPUUsage() float64 {
	// 这是一个简化的CPU使用率估算
	// 基于goroutine数量和活动会话数量
	baseUsage := float64(pm.metrics.GoroutineCount) / 100.0
	sessionUsage := float64(pm.metrics.ActiveSessions) * 5.0

	usage := baseUsage + sessionUsage
	if usage > 100.0 {
		usage = 100.0
	}

	return usage
}

// checkAlerts 检查警报条件
func (pm *PerformanceMonitor) checkAlerts() {
	alerts := pm.evaluateAlerts()

	for _, alert := range alerts {
		pm.handleAlert(alert)
	}
}

// evaluateAlerts 评估警报条件
func (pm *PerformanceMonitor) evaluateAlerts() []Alert {
	pm.metrics.mutex.RLock()
	defer pm.metrics.mutex.RUnlock()

	var alerts []Alert

	// CPU使用率警报
	if pm.metrics.CPUUsage > pm.alertThresholds.MaxCPUUsage {
		alerts = append(alerts, Alert{
			Type:      "cpu_usage",
			Message:   "高CPU使用率",
			Value:     pm.metrics.CPUUsage,
			Threshold: pm.alertThresholds.MaxCPUUsage,
			Severity:  pm.getSeverity(pm.metrics.CPUUsage, pm.alertThresholds.MaxCPUUsage),
			Timestamp: time.Now(),
		})
	}

	// 内存使用警报
	if pm.metrics.MemoryUsage > pm.alertThresholds.MaxMemoryUsage {
		alerts = append(alerts, Alert{
			Type:      "memory_usage",
			Message:   "高内存使用",
			Value:     float64(pm.metrics.MemoryUsage),
			Threshold: float64(pm.alertThresholds.MaxMemoryUsage),
			Severity:  pm.getSeverity(float64(pm.metrics.MemoryUsage), float64(pm.alertThresholds.MaxMemoryUsage)),
			Timestamp: time.Now(),
		})
	}

	// Goroutine数量警报
	if pm.metrics.GoroutineCount > pm.alertThresholds.MaxGoroutines {
		alerts = append(alerts, Alert{
			Type:      "goroutine_count",
			Message:   "Goroutine数量过多",
			Value:     float64(pm.metrics.GoroutineCount),
			Threshold: float64(pm.alertThresholds.MaxGoroutines),
			Severity:  pm.getSeverity(float64(pm.metrics.GoroutineCount), float64(pm.alertThresholds.MaxGoroutines)),
			Timestamp: time.Now(),
		})
	}

	return alerts
}

// getSeverity 获取警报严重程度
func (pm *PerformanceMonitor) getSeverity(value, threshold float64) string {
	ratio := value / threshold

	if ratio >= 2.0 {
		return "critical"
	} else if ratio >= 1.5 {
		return "high"
	} else if ratio >= 1.2 {
		return "medium"
	} else {
		return "low"
	}
}

// handleAlert 处理警报
func (pm *PerformanceMonitor) handleAlert(alert Alert) {
	logger.Warn("Performance alert",
		zap.String("type", alert.Type),
		zap.String("message", alert.Message),
		zap.Float64("value", alert.Value),
		zap.Float64("threshold", alert.Threshold),
		zap.String("severity", alert.Severity))

	// 根据严重程度采取不同行动
	switch alert.Severity {
	case "critical":
		pm.handleCriticalAlert(alert)
	case "high":
		pm.handleHighAlert(alert)
	case "medium":
		pm.handleMediumAlert(alert)
	}
}

// handleCriticalAlert 处理关键警报
func (pm *PerformanceMonitor) handleCriticalAlert(alert Alert) {
	logger.Error("Critical performance alert",
		zap.String("type", alert.Type),
		zap.Float64("value", alert.Value),
		zap.Float64("threshold", alert.Threshold))

	// 可以在这里实现自动化响应，如：
	// - 清理资源
	// - 关闭非活动会话
	// - 发送通知
}

// handleHighAlert 处理高级警报
func (pm *PerformanceMonitor) handleHighAlert(alert Alert) {
	logger.Warn("High performance alert",
		zap.String("type", alert.Type),
		zap.Float64("value", alert.Value),
		zap.Float64("threshold", alert.Threshold))
}

// handleMediumAlert 处理中级警报
func (pm *PerformanceMonitor) handleMediumAlert(alert Alert) {
	logger.Info("Medium performance alert",
		zap.String("type", alert.Type),
		zap.Float64("value", alert.Value),
		zap.Float64("threshold", alert.Threshold))
}

// logMetrics 记录性能指标
func (pm *PerformanceMonitor) logMetrics() {
	logger.Info("Performance metrics",
		zap.Float64("cpu_usage", pm.metrics.CPUUsage),
		zap.Int64("memory_usage_mb", pm.metrics.MemoryUsage/(1024*1024)),
		zap.Int("goroutines", pm.metrics.GoroutineCount),
		zap.Int("active_sessions", pm.metrics.ActiveSessions),
		zap.Int("total_windows", pm.metrics.TotalWindows),
		zap.Int("total_panes", pm.metrics.TotalPanes),
		zap.Int("active_ptys", pm.metrics.ActivePTYs),
		zap.Duration("uptime", pm.metrics.Uptime))
}

// GetMetrics 获取当前性能指标
func (pm *PerformanceMonitor) GetMetrics() *PerformanceMetrics {
	pm.metrics.mutex.RLock()
	defer pm.metrics.mutex.RUnlock()

	// 返回指标的副本
	metrics := *pm.metrics
	return &metrics
}

// SetInterval 设置监控间隔
func (pm *PerformanceMonitor) SetInterval(interval time.Duration) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.interval = interval
}

// SetAlertThresholds 设置警报阈值
func (pm *PerformanceMonitor) SetAlertThresholds(thresholds *AlertThresholds) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.alertThresholds = thresholds
}

// IsRunning 检查监控器是否运行中
func (pm *PerformanceMonitor) IsRunning() bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.running
}

// GetSummary 获取性能摘要
func (pm *PerformanceMonitor) GetSummary() map[string]interface{} {
	metrics := pm.GetMetrics()

	return map[string]interface{}{
		"status":           pm.getOverallStatus(),
		"cpu_usage":        fmt.Sprintf("%.1f%%", metrics.CPUUsage),
		"memory_usage":     fmt.Sprintf("%.1fMB", float64(metrics.MemoryUsage)/(1024*1024)),
		"active_sessions":  metrics.ActiveSessions,
		"total_components": metrics.TotalWindows + metrics.TotalPanes,
		"uptime":           pm.formatDuration(metrics.Uptime),
		"last_update":      metrics.Timestamp.Format("15:04:05"),
	}
}

// getOverallStatus 获取整体状态
func (pm *PerformanceMonitor) getOverallStatus() string {
	metrics := pm.GetMetrics()

	if metrics.CPUUsage > pm.alertThresholds.MaxCPUUsage ||
		metrics.MemoryUsage > pm.alertThresholds.MaxMemoryUsage ||
		metrics.GoroutineCount > pm.alertThresholds.MaxGoroutines {
		return "warning"
	}

	if metrics.CPUUsage > pm.alertThresholds.MaxCPUUsage*0.8 ||
		metrics.MemoryUsage > pm.alertThresholds.MaxMemoryUsage*8/10 {
		return "caution"
	}

	return "healthy"
}

// formatDuration 格式化时间长度
func (pm *PerformanceMonitor) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
}
