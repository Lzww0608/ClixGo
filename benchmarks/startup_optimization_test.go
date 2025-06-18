/*
 * @Author: Lzww0608
 * @Date: 2025-6-18 20:40:00
 * @LastEditors: Lzww0608
 * @LastEditTime: 2025-6-18 20:40:00
 * @Description: Phase 1.3 任务1.4 - 启动优化基准测试
 */

package benchmarks

import (
	"fmt"
	"testing"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

// BenchmarkStartupOptimization 启动优化基准测试
func BenchmarkStartupOptimization(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	b.Run("Original-SessionManager", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		var totalStartupTime int64

		for i := 0; i < b.N; i++ {
			startTime := time.Now()

			manager := terminal.NewSessionManager(config)

			startupTime := time.Since(startTime)
			totalStartupTime += startupTime.Nanoseconds()

			b.ReportMetric(float64(startupTime.Nanoseconds()), "startup_ns")

			// 立即关闭避免资源积累
			manager.Shutdown()
		}

		avgStartupTime := totalStartupTime / int64(b.N)
		b.ReportMetric(float64(avgStartupTime), "avg_startup_ns")

		// 检查是否达到目标 (<30ms = 30,000,000 ns)
		if avgStartupTime < 30000000 {
			b.Logf("✅ 启动性能达标: %.2fms < 30ms", float64(avgStartupTime)/1000000)
		} else {
			b.Logf("❌ 启动性能未达标: %.2fms >= 30ms", float64(avgStartupTime)/1000000)
		}
	})

	b.Run("Fast-SessionManager", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		var totalStartupTime int64

		for i := 0; i < b.N; i++ {
			startTime := time.Now()

			fastManager := terminal.NewFastSessionManager(config)

			startupTime := time.Since(startTime)
			totalStartupTime += startupTime.Nanoseconds()

			b.ReportMetric(float64(startupTime.Nanoseconds()), "startup_ns")

			// 立即关闭避免资源积累
			fastManager.Shutdown()
		}

		avgStartupTime := totalStartupTime / int64(b.N)
		b.ReportMetric(float64(avgStartupTime), "avg_startup_ns")

		// 检查是否达到目标 (<30ms = 30,000,000 ns)
		if avgStartupTime < 30000000 {
			b.Logf("✅ 快速启动性能达标: %.2fms < 30ms", float64(avgStartupTime)/1000000)
		} else {
			b.Logf("❌ 快速启动性能未达标: %.2fms >= 30ms", float64(avgStartupTime)/1000000)
		}
	})
}

// BenchmarkLazyInitialization 延迟初始化基准测试
func BenchmarkLazyInitialization(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	b.Run("FirstSessionCreation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			fastManager := terminal.NewFastSessionManager(config)

			startTime := time.Now()

			// 第一次创建会话会触发延迟初始化
			session, err := fastManager.CreateSession(fmt.Sprintf("test-session-%d", i))
			if err != nil {
				b.Fatalf("创建会话失败: %v", err)
			}

			initTime := time.Since(startTime)
			b.ReportMetric(float64(initTime.Nanoseconds()), "first_session_create_ns")

			// 清理
			fastManager.KillSession(session.ID)
			fastManager.Shutdown()
		}
	})

	b.Run("SubsequentSessionCreation", func(b *testing.B) {
		fastManager := terminal.NewFastSessionManager(config)

		// 先创建一个会话触发初始化
		firstSession, err := fastManager.CreateSession("init-session")
		if err != nil {
			b.Fatalf("初始化会话创建失败: %v", err)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			startTime := time.Now()

			// 后续会话创建不会触发初始化
			session, err := fastManager.CreateSession(fmt.Sprintf("subsequent-session-%d", i))
			if err != nil {
				b.Fatalf("创建后续会话失败: %v", err)
			}

			createTime := time.Since(startTime)
			b.ReportMetric(float64(createTime.Nanoseconds()), "subsequent_session_create_ns")

			// 清理会话
			fastManager.KillSession(session.ID)
		}

		// 最终清理
		fastManager.KillSession(firstSession.ID)
		fastManager.Shutdown()
	})
}

// BenchmarkMemoryUsageComparison 内存使用对比测试
func BenchmarkMemoryUsageComparison(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	b.Run("Original-MemoryUsage", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			manager := terminal.NewSessionManager(config)

			// 创建一个会话测试内存使用
			session, err := manager.CreateSession("memory-test")
			if err != nil {
				b.Fatalf("创建会话失败: %v", err)
			}

			// 清理
			manager.KillSession(session.ID)
			manager.Shutdown()
		}
	})

	b.Run("Fast-MemoryUsage", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			fastManager := terminal.NewFastSessionManager(config)

			// 创建一个会话测试内存使用
			session, err := fastManager.CreateSession("memory-test")
			if err != nil {
				b.Fatalf("创建会话失败: %v", err)
			}

			// 清理
			fastManager.KillSession(session.ID)
			fastManager.Shutdown()
		}
	})
}
