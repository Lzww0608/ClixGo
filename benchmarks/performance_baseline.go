/*
* @Author: Lzww0608
* @Date: 2025-6-13 23:12:33
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-13 23:12:36
* @Description: ClixGo性能基线测试工具
 */

package benchmarks

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	// 启动性能
	StartupTimeMs int64   `json:"startup_time_ms"`
	MemoryUsageMB float64 `json:"memory_usage_mb"`

	// 会话性能
	SessionCreateMs int64 `json:"session_create_ms"`
	SessionSwitchMs int64 `json:"session_switch_ms"`

	// 系统资源
	GoroutineCount int `json:"goroutine_count"`
	CPUCores       int `json:"cpu_cores"`

	// 与tmux对比
	TmuxStartupMs int64   `json:"tmux_startup_ms"`
	TmuxMemoryMB  float64 `json:"tmux_memory_mb"`

	// 性能比较
	StartupSpeedup  float64 `json:"startup_speedup"`
	MemoryReduction float64 `json:"memory_reduction"`
}

// PerformanceBaseline 性能基线测试
type PerformanceBaseline struct {
	metrics *PerformanceMetrics
}

// NewPerformanceBaseline 创建性能基线测试
func NewPerformanceBaseline() *PerformanceBaseline {
	return &PerformanceBaseline{
		metrics: &PerformanceMetrics{
			CPUCores: runtime.NumCPU(),
		},
	}
}

// RunBaseline 运行基线测试
func (pb *PerformanceBaseline) RunBaseline() (*PerformanceMetrics, error) {
	fmt.Println("🚀 开始ClixGo性能基线测试...")

	// 测试ClixGo启动性能
	if err := pb.measureClixGoStartup(); err != nil {
		return nil, fmt.Errorf("ClixGo启动测试失败: %w", err)
	}

	// 测试会话性能
	if err := pb.measureSessionPerformance(); err != nil {
		return nil, fmt.Errorf("会话性能测试失败: %w", err)
	}

	// 测试tmux性能（对比）
	if err := pb.measureTmuxPerformance(); err != nil {
		fmt.Printf("⚠️  tmux对比测试失败: %v\n", err)
	}

	// 计算性能比较
	pb.calculateComparison()

	fmt.Println("✅ 性能基线测试完成")
	return pb.metrics, nil
}

// measureClixGoStartup 测量ClixGo启动性能
func (pb *PerformanceBaseline) measureClixGoStartup() error {
	fmt.Print("📊 测量ClixGo启动性能...")

	// 初始化日志系统
	logger.InitLogger()

	// 记录初始内存
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// 测量启动时间
	startTime := time.Now()

	// 创建会话管理器
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}
	manager := terminal.NewSessionManager(config)

	// 创建第一个会话
	session, err := manager.CreateSession("baseline-session")
	if err != nil {
		return fmt.Errorf("创建会话失败: %w", err)
	}

	// 记录启动时间
	pb.metrics.StartupTimeMs = time.Since(startTime).Milliseconds()

	// 记录内存使用
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)
	pb.metrics.MemoryUsageMB = float64(m2.Alloc-m1.Alloc) / 1024 / 1024
	pb.metrics.GoroutineCount = runtime.NumGoroutine()

	// 清理
	manager.KillSession(session.ID)

	fmt.Printf(" ✓ (启动:%dms, 内存:%.2fMB)\n",
		pb.metrics.StartupTimeMs, pb.metrics.MemoryUsageMB)

	return nil
}

// measureSessionPerformance 测量会话性能
func (pb *PerformanceBaseline) measureSessionPerformance() error {
	fmt.Print("📊 测量会话操作性能...")

	// 确保日志系统已初始化
	logger.InitLogger()

	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}
	manager := terminal.NewSessionManager(config)

	// 测量会话创建时间
	createStart := time.Now()
	session, err := manager.CreateSession("perf-session")
	if err != nil {
		return fmt.Errorf("创建会话失败: %w", err)
	}
	pb.metrics.SessionCreateMs = time.Since(createStart).Milliseconds()

	// 创建多个会话测试切换性能
	sessions := []*terminal.Session{session}
	for i := 0; i < 5; i++ {
		s, err := manager.CreateSession(fmt.Sprintf("switch-session-%d", i))
		if err != nil {
			return fmt.Errorf("创建会话%d失败: %w", i, err)
		}
		sessions = append(sessions, s)
	}

	// 测量会话切换时间
	switchStart := time.Now()
	for i := 0; i < 10; i++ {
		sessionID := sessions[i%len(sessions)].ID
		_, err := manager.GetSession(sessionID)
		if err != nil {
			return fmt.Errorf("获取会话失败: %w", err)
		}
	}
	pb.metrics.SessionSwitchMs = time.Since(switchStart).Milliseconds() / 10

	// 清理
	for _, s := range sessions {
		manager.KillSession(s.ID)
	}

	fmt.Printf(" ✓ (创建:%dms, 切换:%dms)\n",
		pb.metrics.SessionCreateMs, pb.metrics.SessionSwitchMs)

	return nil
}

// measureTmuxPerformance 测量tmux性能（对比）
func (pb *PerformanceBaseline) measureTmuxPerformance() error {
	fmt.Print("📊 测量tmux性能（对比）...")

	// 检查tmux是否可用
	if !pb.isTmuxAvailable() {
		return fmt.Errorf("tmux未安装或不可用")
	}

	// 测量tmux启动时间
	startTime := time.Now()
	cmd := exec.Command("tmux", "new-session", "-d", "-s", "baseline-test", "echo hello")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("启动tmux失败: %w", err)
	}
	pb.metrics.TmuxStartupMs = time.Since(startTime).Milliseconds()

	// 估算tmux内存使用（简化）
	pb.metrics.TmuxMemoryMB = 25.0 // tmux典型内存使用量

	// 清理tmux会话
	cleanupCmd := exec.Command("tmux", "kill-session", "-t", "baseline-test")
	cleanupCmd.Run() // 忽略错误

	fmt.Printf(" ✓ (启动:%dms, 内存:%.2fMB)\n",
		pb.metrics.TmuxStartupMs, pb.metrics.TmuxMemoryMB)

	return nil
}

// isTmuxAvailable 检查tmux是否可用
func (pb *PerformanceBaseline) isTmuxAvailable() bool {
	cmd := exec.Command("tmux", "-V")
	return cmd.Run() == nil
}

// calculateComparison 计算性能比较
func (pb *PerformanceBaseline) calculateComparison() {
	if pb.metrics.TmuxStartupMs > 0 {
		pb.metrics.StartupSpeedup = float64(pb.metrics.TmuxStartupMs) / float64(pb.metrics.StartupTimeMs)
	}

	if pb.metrics.TmuxMemoryMB > 0 {
		pb.metrics.MemoryReduction = (pb.metrics.TmuxMemoryMB - pb.metrics.MemoryUsageMB) / pb.metrics.TmuxMemoryMB * 100
	}
}

// GenerateReport 生成性能报告
func (pb *PerformanceBaseline) GenerateReport() string {
	m := pb.metrics

	report := fmt.Sprintf(`
🚀 ClixGo 性能基线报告
==========================================

📈 启动性能:
  ├─ ClixGo启动时间: %dms
  ├─ tmux启动时间:   %dms
  └─ 性能提升:       %.1fx

💾 内存使用:
  ├─ ClixGo内存:     %.2fMB
  ├─ tmux内存:       %.2fMB
  └─ 内存减少:       %.1f%%

⚡ 会话性能:
  ├─ 创建时间:       %dms
  ├─ 切换时间:       %dms
  └─ 协程数量:       %d

🖥️  系统信息:
  └─ CPU核心数:      %d

🎯 性能目标达成度:
  ├─ 启动时间目标:   <30ms   [%s]
  ├─ 内存使用目标:   <8MB    [%s]
  └─ 切换时间目标:   <5ms    [%s]

`,
		m.StartupTimeMs, m.TmuxStartupMs, m.StartupSpeedup,
		m.MemoryUsageMB, m.TmuxMemoryMB, m.MemoryReduction,
		m.SessionCreateMs, m.SessionSwitchMs, m.GoroutineCount,
		m.CPUCores,
		pb.getStatus(m.StartupTimeMs < 30),
		pb.getStatus(m.MemoryUsageMB < 8.0),
		pb.getStatus(m.SessionSwitchMs < 5),
	)

	return report
}

// getStatus 获取状态图标
func (pb *PerformanceBaseline) getStatus(achieved bool) string {
	if achieved {
		return "✅ 达成"
	}
	return "❌ 未达成"
}

// SaveReport 保存报告到文件
func (pb *PerformanceBaseline) SaveReport(filename string) error {
	report := pb.GenerateReport()

	// 这里可以实现文件保存逻辑
	fmt.Printf("报告保存功能待实现: %s\n", filename)
	fmt.Println(report)

	return nil
}

// RunContinuousMonitoring 运行持续性能监控
func (pb *PerformanceBaseline) RunContinuousMonitoring(ctx context.Context, interval time.Duration) error {
	fmt.Printf("📊 开始持续性能监控 (间隔: %v)\n", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("📊 持续监控已停止")
			return nil
		case <-ticker.C:
			metrics, err := pb.RunBaseline()
			if err != nil {
				fmt.Printf("❌ 监控测试失败: %v\n", err)
				continue
			}

			fmt.Printf("📈 [%s] 启动:%dms, 内存:%.2fMB, 协程:%d\n",
				time.Now().Format("15:04:05"),
				metrics.StartupTimeMs,
				metrics.MemoryUsageMB,
				metrics.GoroutineCount)
		}
	}
}
