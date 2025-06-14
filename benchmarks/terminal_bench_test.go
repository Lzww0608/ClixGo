/*
* @Author: Lzww0608
* @Date: 2025-6-1 20:50:44
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-14 11:02:19
* @Description: 终端性能基准测试 - 用于性能对比和回归检测
 */

package benchmarks

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

// TestMain 在测试开始前初始化logger
func TestMain(m *testing.M) {
	// 初始化日志系统
	logger.InitLogger()
	defer logger.Close()

	// 运行测试
	code := m.Run()
	os.Exit(code)
}

// BenchmarkSessionCreation 会话创建性能基准测试
func BenchmarkSessionCreation(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	manager := terminal.NewSessionManager(config)
	defer manager.Shutdown()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sessionName := fmt.Sprintf("bench-session-%d", i)
		session, err := manager.CreateSession(sessionName)
		if err != nil {
			b.Fatalf("创建会话失败: %v", err)
		}
		_ = session // 避免编译器优化
	}

	// 报告性能统计
	stats := manager.GetPerformanceStats()
	b.ReportMetric(float64(stats.AvgCreateTime.Nanoseconds()), "avg_create_ns")
	b.ReportMetric(float64(stats.MemoryUsageMB), "memory_mb")
}

// BenchmarkStartupTime 启动时间基准测试
func BenchmarkStartupTime(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		startTime := time.Now()

		manager := terminal.NewSessionManager(config)

		setupTime := time.Since(startTime)
		b.ReportMetric(float64(setupTime.Nanoseconds()), "startup_ns")

		manager.Shutdown()
	}
}

// BenchmarkMemoryUsage 内存使用基准测试
func BenchmarkMemoryUsage(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		manager := terminal.NewSessionManager(config)

		runtime.GC()
		runtime.ReadMemStats(&m2)

		memoryUsed := m2.Alloc - m1.Alloc
		b.ReportMetric(float64(memoryUsed), "memory_bytes")

		manager.Shutdown()
	}
}

// BenchmarkConcurrentSessions 并发会话创建基准测试
func BenchmarkConcurrentSessions(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	manager := terminal.NewSessionManager(config)
	defer manager.Shutdown()

	concurrency := []int{1, 10, 50, 100}

	for _, c := range concurrency {
		b.Run(fmt.Sprintf("Concurrency-%d", c), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(c)

				startTime := time.Now()

				for j := 0; j < c; j++ {
					go func(index int) {
						defer wg.Done()
						sessionName := fmt.Sprintf("concurrent-session-%d-%d", i, index)
						_, err := manager.CreateSession(sessionName)
						if err != nil {
							b.Errorf("创建会话失败: %v", err)
						}
					}(j)
				}

				wg.Wait()
				duration := time.Since(startTime)
				b.ReportMetric(float64(duration.Nanoseconds()/int64(c)), "avg_session_create_ns")
			}
		})
	}
}

// BenchmarkObjectPoolEfficiency 对象池效率基准测试
func BenchmarkObjectPoolEfficiency(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	manager := terminal.NewSessionManager(config)
	defer manager.Shutdown()

	bufferSizes := []int{1024, 4096, 16384}

	for _, size := range bufferSizes {
		b.Run(fmt.Sprintf("BufferSize-%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			objectPool := manager.GetObjectPool()
			for i := 0; i < b.N; i++ {
				// 测试对象池的获取和归还
				buffer := objectPool.GetBuffer(size)
				objectPool.PutBuffer(buffer)
			}

			// 报告对象池统计
			stats := manager.GetPerformanceStats()
			b.ReportMetric(float64(stats.BufferPoolHits), "pool_hits")
			b.ReportMetric(float64(stats.BufferPoolMisses), "pool_misses")
		})
	}
}

// BenchmarkCommandExecution 命令执行性能基准测试
func BenchmarkCommandExecution(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	manager := terminal.NewSessionManager(config)
	defer manager.Shutdown()

	// 创建测试会话
	session, err := manager.CreateSession("bench-command-session")
	if err != nil {
		b.Fatalf("创建会话失败: %v", err)
	}

	commands := []string{
		"echo hello",
		"ls -la",
		"pwd",
		"date",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for _, cmd := range commands {
		b.Run(fmt.Sprintf("Command-%s", cmd), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				startTime := time.Now()

				// 创建新窗口执行命令
				_, err := manager.CreateWindow(session.ID, fmt.Sprintf("cmd-window-%d", i))
				if err != nil {
					b.Fatalf("创建窗口失败: %v", err)
				}

				executionTime := time.Since(startTime)
				b.ReportMetric(float64(executionTime.Nanoseconds()), "execution_ns")
			}
		})
	}
}

// BenchmarkGoroutinePoolEfficiency 协程池效率基准测试
func BenchmarkGoroutinePoolEfficiency(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	manager := terminal.NewSessionManager(config)
	defer manager.Shutdown()

	b.ResetTimer()
	b.ReportAllocs()

	goroutinePool := manager.GetGoroutinePool()

	for i := 0; i < b.N; i++ {
		sessionName := fmt.Sprintf("pool-test-session-%d", i)

		startTime := time.Now()
		session, err := manager.CreateSession(sessionName)
		if err != nil {
			b.Fatalf("创建会话失败: %v", err)
		}
		duration := time.Since(startTime)

		b.ReportMetric(float64(duration.Nanoseconds()), "session_create_ns")
		_ = session // 避免编译器优化
	}

	// 报告协程池统计
	if goroutinePool != nil {
		metrics := goroutinePool.GetMetrics()
		b.ReportMetric(float64(metrics.CompletedTasks), "tasks_completed")
		b.ReportMetric(float64(metrics.ActiveWorkers), "active_workers")
	}
}

// BenchmarkMemoryFootprint 内存占用基准测试
func BenchmarkMemoryFootprint(b *testing.B) {
	sessionCounts := []int{1, 10, 50, 100}

	for _, count := range sessionCounts {
		b.Run(fmt.Sprintf("Sessions-%d", count), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var m1, m2 runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&m1)

				config := &terminal.TerminalConfig{
					BufferSize: 2000,
					ScrollBack: 2000,
				}
				manager := terminal.NewSessionManager(config)

				// 创建指定数量的会话
				for j := 0; j < count; j++ {
					sessionName := fmt.Sprintf("memory-test-session-%d-%d", i, j)
					_, err := manager.CreateSession(sessionName)
					if err != nil {
						b.Fatalf("创建会话失败: %v", err)
					}
				}

				runtime.GC()
				runtime.ReadMemStats(&m2)

				memoryPerSession := float64(m2.Alloc-m1.Alloc) / float64(count)
				b.ReportMetric(memoryPerSession, "memory_per_session_bytes")

				manager.Shutdown()
			}
		})
	}
}

// BenchmarkShutdownTime 关闭时间基准测试
func BenchmarkShutdownTime(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		config := &terminal.TerminalConfig{
			BufferSize: 2000,
			ScrollBack: 2000,
		}
		manager := terminal.NewSessionManager(config)

		// 创建一些会话和窗口
		for j := 0; j < 5; j++ {
			sessionName := fmt.Sprintf("shutdown-test-session-%d-%d", i, j)
			session, err := manager.CreateSession(sessionName)
			if err != nil {
				b.Fatalf("创建会话失败: %v", err)
			}

			// 为每个会话创建几个窗口
			for k := 0; k < 3; k++ {
				_, err := manager.CreateWindow(session.ID, fmt.Sprintf("window-%d", k))
				if err != nil {
					b.Fatalf("创建窗口失败: %v", err)
				}
			}
		}

		// 测量关闭时间
		startTime := time.Now()
		err := manager.Shutdown()
		if err != nil {
			b.Fatalf("关闭管理器失败: %v", err)
		}
		shutdownTime := time.Since(startTime)

		b.ReportMetric(float64(shutdownTime.Nanoseconds()), "shutdown_ns")
	}
}

// PerformanceReport 性能报告结构
type PerformanceReport struct {
	StartupTimeNS       int64   `json:"startup_time_ns"`
	MemoryUsageBytes    int64   `json:"memory_usage_bytes"`
	SessionCreateTimeNS int64   `json:"session_create_time_ns"`
	ShutdownTimeNS      int64   `json:"shutdown_time_ns"`
	CPUUsagePercent     float64 `json:"cpu_usage_percent"`
	GoroutineCount      int     `json:"goroutine_count"`
}

// RunPerformanceReport 运行完整的性能报告
func RunPerformanceReport(b *testing.B) {
	// 这个函数可以被用来生成完整的性能报告
	// 供CI/CD系统使用，用于性能回归检测

	report := &PerformanceReport{}

	// 测试启动时间
	startTime := time.Now()
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}
	manager := terminal.NewSessionManager(config)
	report.StartupTimeNS = time.Since(startTime).Nanoseconds()

	// 测试内存使用
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// 创建一些会话
	for i := 0; i < 10; i++ {
		sessionName := fmt.Sprintf("report-session-%d", i)
		_, err := manager.CreateSession(sessionName)
		if err != nil {
			b.Fatalf("创建会话失败: %v", err)
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)
	report.MemoryUsageBytes = int64(m2.Alloc - m1.Alloc)
	report.GoroutineCount = runtime.NumGoroutine()

	// 测试关闭时间
	shutdownStart := time.Now()
	err := manager.Shutdown()
	if err != nil {
		b.Fatalf("关闭管理器失败: %v", err)
	}
	report.ShutdownTimeNS = time.Since(shutdownStart).Nanoseconds()

	// 输出报告（实际应用中可以保存到文件或发送到监控系统）
	b.Logf("性能报告: 启动时间=%dns, 内存使用=%d字节, 关闭时间=%dns, 协程数=%d",
		report.StartupTimeNS, report.MemoryUsageBytes, report.ShutdownTimeNS, report.GoroutineCount)
}
