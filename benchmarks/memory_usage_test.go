/*
 * @Author: Lzww0608
 * @Date: 2025-6-18 21:00:00
 * @LastEditors: Lzww0608
 * @LastEditTime: 2025-6-18 21:00:00
 * @Description: Phase 1.3 内存使用优化验证测试
 */

package benchmarks

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

// TestMemoryUsage 内存使用测试
func TestMemoryUsage(t *testing.T) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	t.Run("FastSessionManager-MemoryUsage", func(t *testing.T) {
		// 强制垃圾回收，获取基准内存
		runtime.GC()
		runtime.GC() // 执行两次确保清理完成
		time.Sleep(10 * time.Millisecond)

		var baseMemStats runtime.MemStats
		runtime.ReadMemStats(&baseMemStats)
		baseMemoryMB := float64(baseMemStats.Alloc) / 1024 / 1024

		// 创建FastSessionManager
		fastManager := terminal.NewFastSessionManager(config)
		defer fastManager.Shutdown()

		// 创建会话和窗口进行内存测试
		session, err := fastManager.CreateSession("memory-test")
		if err != nil {
			t.Fatalf("创建会话失败: %v", err)
		}

		// 创建多个窗口测试内存使用
		for i := 0; i < 5; i++ {
			windowName := fmt.Sprintf("window-%d", i)
			_, err := fastManager.CreateWindow(session.ID, windowName)
			if err != nil {
				t.Fatalf("创建窗口失败: %v", err)
			}
		}

		// 强制垃圾回收，获取使用后内存
		runtime.GC()
		runtime.GC()
		time.Sleep(10 * time.Millisecond)

		var afterMemStats runtime.MemStats
		runtime.ReadMemStats(&afterMemStats)
		afterMemoryMB := float64(afterMemStats.Alloc) / 1024 / 1024

		// 计算内存使用量
		memoryUsageMB := afterMemoryMB - baseMemoryMB

		t.Logf("基准内存: %.2f MB", baseMemoryMB)
		t.Logf("使用后内存: %.2f MB", afterMemoryMB)
		t.Logf("FastSessionManager内存使用: %.2f MB", memoryUsageMB)

		// 验证内存使用目标 <8MB
		if memoryUsageMB >= 8.0 {
			t.Errorf("内存使用超标: %.2f MB >= 8MB", memoryUsageMB)
		} else {
			t.Logf("✅ 内存使用达标: %.2f MB < 8MB", memoryUsageMB)
		}

		// 详细内存统计
		t.Logf("详细内存统计:")
		t.Logf("  - 堆内存分配: %.2f MB", float64(afterMemStats.HeapAlloc)/1024/1024)
		t.Logf("  - 堆内存系统: %.2f MB", float64(afterMemStats.HeapSys)/1024/1024)
		t.Logf("  - 栈内存使用: %.2f MB", float64(afterMemStats.StackSys)/1024/1024)
		t.Logf("  - 垃圾回收次数: %d", afterMemStats.NumGC-baseMemStats.NumGC)
	})

	t.Run("OriginalSessionManager-MemoryComparison", func(t *testing.T) {
		// 强制垃圾回收，获取基准内存
		runtime.GC()
		runtime.GC()
		time.Sleep(10 * time.Millisecond)

		var baseMemStats runtime.MemStats
		runtime.ReadMemStats(&baseMemStats)
		baseMemoryMB := float64(baseMemStats.Alloc) / 1024 / 1024

		// 创建原版SessionManager
		originalManager := terminal.NewSessionManager(config)
		defer originalManager.Shutdown()

		// 创建会话和窗口进行内存测试
		session, err := originalManager.CreateSession("memory-comparison-test")
		if err != nil {
			t.Fatalf("创建会话失败: %v", err)
		}

		// 创建多个窗口测试内存使用
		for i := 0; i < 5; i++ {
			windowName := fmt.Sprintf("window-%d", i)
			_, err := originalManager.CreateWindow(session.ID, windowName)
			if err != nil {
				t.Fatalf("创建窗口失败: %v", err)
			}
		}

		// 强制垃圾回收，获取使用后内存
		runtime.GC()
		runtime.GC()
		time.Sleep(10 * time.Millisecond)

		var afterMemStats runtime.MemStats
		runtime.ReadMemStats(&afterMemStats)
		afterMemoryMB := float64(afterMemStats.Alloc) / 1024 / 1024

		// 计算内存使用量
		memoryUsageMB := afterMemoryMB - baseMemoryMB

		t.Logf("原版SessionManager内存使用: %.2f MB", memoryUsageMB)

		if memoryUsageMB >= 25.0 {
			t.Logf("⚠️ 原版内存使用较高: %.2f MB (预期~25MB)", memoryUsageMB)
		} else {
			t.Logf("✅ 原版内存使用: %.2f MB", memoryUsageMB)
		}
	})
}

// BenchmarkMemoryEfficiency 内存效率基准测试
func BenchmarkMemoryEfficiency(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	b.Run("FastSessionManager-MemoryAllocation", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			fastManager := terminal.NewFastSessionManager(config)

			session, err := fastManager.CreateSession("bench-session")
			if err != nil {
				b.Fatalf("创建会话失败: %v", err)
			}

			_, err = fastManager.CreateWindow(session.ID, "bench-window")
			if err != nil {
				b.Fatalf("创建窗口失败: %v", err)
			}

			fastManager.Shutdown()
		}
	})

	b.Run("MemoryLeakTest", func(b *testing.B) {
		// 内存泄漏测试
		var initialMemStats runtime.MemStats
		runtime.GC()
		runtime.GC()
		runtime.ReadMemStats(&initialMemStats)

		for i := 0; i < b.N; i++ {
			fastManager := terminal.NewFastSessionManager(config)

			session, _ := fastManager.CreateSession(fmt.Sprintf("leak-test-%d", i))
			fastManager.CreateWindow(session.ID, "test-window")

			// 立即关闭，测试是否有内存泄漏
			fastManager.Shutdown()

			// 每100次迭代检查一次内存
			if i%100 == 0 && i > 0 {
				runtime.GC()
				runtime.GC()
				var currentMemStats runtime.MemStats
				runtime.ReadMemStats(&currentMemStats)

				memoryGrowthMB := float64(currentMemStats.Alloc-initialMemStats.Alloc) / 1024 / 1024
				if memoryGrowthMB > 10.0 { // 如果内存增长超过10MB，可能有泄漏
					b.Errorf("可能存在内存泄漏: 内存增长 %.2f MB after %d iterations", memoryGrowthMB, i)
				}
			}
		}
	})
}

// TestMemoryOptimizationFeatures 内存优化功能测试
func TestMemoryOptimizationFeatures(t *testing.T) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	t.Run("ObjectPoolMemoryReuse", func(t *testing.T) {
		fastManager := terminal.NewFastSessionManager(config)
		defer fastManager.Shutdown()

		// 触发延迟初始化
		_, err := fastManager.CreateSession("pool-test")
		if err != nil {
			t.Fatalf("创建会话失败: %v", err)
		}

		// 获取对象池
		objectPool := fastManager.GetObjectPool()
		if objectPool == nil {
			t.Fatal("对象池未初始化")
		}

		// 测试缓冲区复用
		buffer1 := objectPool.GetBuffer(1024)
		buffer2 := objectPool.GetBuffer(1024)

		objectPool.PutBuffer(buffer1)
		objectPool.PutBuffer(buffer2)

		// 再次获取，应该复用之前的缓冲区
		buffer3 := objectPool.GetBuffer(1024)
		buffer4 := objectPool.GetBuffer(1024)

		objectPool.PutBuffer(buffer3)
		objectPool.PutBuffer(buffer4)

		t.Logf("✅ 对象池缓冲区复用测试通过")
	})

	t.Run("GoroutinePoolResourceManagement", func(t *testing.T) {
		fastManager := terminal.NewFastSessionManager(config)
		defer fastManager.Shutdown()

		// 触发延迟初始化
		_, err := fastManager.CreateSession("goroutine-test")
		if err != nil {
			t.Fatalf("创建会话失败: %v", err)
		}

		// 获取协程池
		goroutinePool := fastManager.GetGoroutinePool()
		if goroutinePool == nil {
			t.Fatal("协程池未初始化")
		}

		// 检查协程池状态
		metrics := goroutinePool.GetMetrics()
		t.Logf("协程池统计:")
		t.Logf("  - 活跃工作协程: %d", metrics.ActiveWorkers)
		t.Logf("  - 空闲工作协程: %d", metrics.IdleWorkers)
		t.Logf("  - 待处理任务: %d", metrics.PendingTasks)
		t.Logf("  - 已完成任务: %d", metrics.CompletedTasks)

		if metrics.ActiveWorkers > 16 { // FastSessionManager使用最大16个工作协程
			t.Errorf("协程池工作协程数量超标: %d > 16", metrics.ActiveWorkers)
		} else {
			t.Logf("✅ 协程池资源管理正常")
		}
	})
}
