/*
* @Author: Lzww0608
* @Date: 2025-6-17 17:30:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-17 17:30:00
* @Description: Phase 1.3 任务1.4 - 关闭性能优化测试
 */

package benchmarks

import (
	"fmt"
	"testing"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

// BenchmarkShutdownOptimization 关闭性能优化基准测试
func BenchmarkShutdownOptimization(b *testing.B) {
	// 测试不同的关闭超时时间
	timeouts := []time.Duration{
		1 * time.Second,  // 快速关闭
		3 * time.Second,  // 平衡关闭
		5 * time.Second,  // 安全关闭
		10 * time.Second, // 当前默认
	}

	for _, timeout := range timeouts {
		b.Run(fmt.Sprintf("Timeout-%dms", timeout.Milliseconds()), func(b *testing.B) {
			benchmarkShutdownWithTimeout(b, timeout)
		})
	}
}

// benchmarkShutdownWithTimeout 测试指定超时时间的关闭性能
func benchmarkShutdownWithTimeout(b *testing.B, timeout time.Duration) {
	b.ResetTimer()
	b.ReportAllocs()

	var totalShutdownTime int64

	for i := 0; i < b.N; i++ {
		// 创建SessionManager
		config := &terminal.TerminalConfig{
			BufferSize: 1000,
			ScrollBack: 1000,
		}

		sessionManager := terminal.NewSessionManager(config)

		// 创建一个会话来模拟实际使用
		sessionName := fmt.Sprintf("test-session-%d-%d", time.Now().UnixNano(), i)
		session, err := sessionManager.CreateSession(sessionName)
		if err != nil {
			b.Fatalf("创建会话失败: %v", err)
		}

		// 创建一个窗口
		_, err = sessionManager.CreateWindow(session.ID, "test-window")
		if err != nil {
			b.Fatalf("创建窗口失败: %v", err)
		}

		// 测量关闭时间
		startTime := time.Now()

		// 使用优化后的关闭方法
		err = shutdownWithTimeout(sessionManager, timeout)
		if err != nil {
			b.Errorf("关闭失败: %v", err)
		}

		shutdownTime := time.Since(startTime)
		totalShutdownTime += shutdownTime.Nanoseconds()

		b.StopTimer()
		// 清理
		time.Sleep(10 * time.Millisecond) // 短暂等待，确保资源完全释放
		b.StartTimer()
	}

	// 报告平均关闭时间
	if b.N > 0 {
		avgShutdownTime := time.Duration(totalShutdownTime / int64(b.N))
		b.ReportMetric(float64(avgShutdownTime.Milliseconds()), "ms/shutdown")
	}
}

// shutdownWithTimeout 优化的关闭方法，支持自定义超时
func shutdownWithTimeout(sessionManager *terminal.SessionManager, timeout time.Duration) error {
	// 获取组件
	goroutinePool := sessionManager.GetGoroutinePool()
	leakDetector := sessionManager.GetLeakDetector()
	objectPool := sessionManager.GetObjectPool()

	// 创建关闭完成通道
	done := make(chan error, 1)

	go func() {
		// 关闭内存泄漏检测器
		if leakDetector != nil {
			leakDetector.Stop()
		}

		// 关闭协程池（使用自定义超时）
		if goroutinePool != nil {
			goroutinePool.StopWithTimeout(timeout)
		}

		// 清理对象池
		if objectPool != nil {
			objectPool.Stop()
		}

		done <- nil
	}()

	// 等待关闭完成或超时
	select {
	case err := <-done:
		return err
	case <-time.After(timeout + 1*time.Second): // 额外1秒缓冲
		return fmt.Errorf("关闭超时")
	}
}

// BenchmarkOptimizedShutdown 优化后的关闭基准测试
func BenchmarkOptimizedShutdown(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	var totalShutdownTime int64

	for i := 0; i < b.N; i++ {
		// 创建SessionManager
		config := &terminal.TerminalConfig{
			BufferSize: 1000,
			ScrollBack: 1000,
		}

		sessionManager := terminal.NewSessionManager(config)

		// 创建一个会话来模拟实际使用
		sessionName := fmt.Sprintf("optimized-session-%d-%d", time.Now().UnixNano(), i)
		session, err := sessionManager.CreateSession(sessionName)
		if err != nil {
			b.Fatalf("创建会话失败: %v", err)
		}

		// 创建一个窗口
		_, err = sessionManager.CreateWindow(session.ID, "test-window")
		if err != nil {
			b.Fatalf("创建窗口失败: %v", err)
		}

		// 测量优化后的关闭时间（使用3秒超时）
		startTime := time.Now()

		err = shutdownWithTimeout(sessionManager, 3*time.Second)
		if err != nil {
			b.Errorf("关闭失败: %v", err)
		}

		shutdownTime := time.Since(startTime)
		totalShutdownTime += shutdownTime.Nanoseconds()

		b.StopTimer()
		time.Sleep(10 * time.Millisecond)
		b.StartTimer()
	}

	// 报告平均关闭时间
	if b.N > 0 {
		avgShutdownTime := time.Duration(totalShutdownTime / int64(b.N))
		b.ReportMetric(float64(avgShutdownTime.Milliseconds()), "ms/shutdown")
	}
}
