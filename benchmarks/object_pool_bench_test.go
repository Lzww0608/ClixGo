/*
 * @Author: Lzww0608
 * @Date: 2025-6-6 11:29:27
 * @LastEditors: Lzww0608
 * @LastEditTime: 2025-6-6 12:56:46
 * @Description: 对象池基准测试
 */

package benchmarks

import (
	"strings"
	"sync"
	"testing"

	"github.com/Lzww0608/ClixGo/pkg/performance"
)

// BenchmarkBufferAllocation 比较缓冲区分配性能
func BenchmarkBufferAllocation(b *testing.B) {
	b.Run("Direct", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := make([]byte, 1024)
			_ = buf
		}
	})

	b.Run("ObjectPool", func(b *testing.B) {
		manager := performance.NewObjectPoolManager(performance.DefaultPoolConfig())
		defer manager.Stop()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			buf := manager.GetBuffer(1024)
			manager.PutBuffer(buf)
		}
	})
}

// BenchmarkStringBuilding 比较字符串构建性能
func BenchmarkStringBuilding(b *testing.B) {
	text := "Hello, World! "

	b.Run("Direct", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var builder strings.Builder
			for j := 0; j < 100; j++ {
				builder.WriteString(text)
			}
			_ = builder.String()
		}
	})

	b.Run("ObjectPool", func(b *testing.B) {
		manager := performance.NewObjectPoolManager(performance.DefaultPoolConfig())
		defer manager.Stop()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			builder := manager.GetStringBuilder()
			for j := 0; j < 100; j++ {
				builder.WriteString(text)
			}
			_ = builder.String()
			manager.PutStringBuilder(builder)
		}
	})
}

// BenchmarkSliceAllocation 比较切片分配性能
func BenchmarkSliceAllocation(b *testing.B) {
	b.Run("Direct", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			slice := make([]interface{}, 0, 100)
			for j := 0; j < 10; j++ {
				slice = append(slice, j)
			}
			_ = slice
		}
	})

	b.Run("ObjectPool", func(b *testing.B) {
		manager := performance.NewObjectPoolManager(performance.DefaultPoolConfig())
		defer manager.Stop()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			slice := manager.GetSlice(100)
			for j := 0; j < 10; j++ {
				slice = append(slice, j)
			}
			manager.PutSlice(slice)
		}
	})
}

// BenchmarkConcurrentOperations 测试并发操作性能
func BenchmarkConcurrentOperations(b *testing.B) {
	b.Run("Direct", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := make([]byte, 1024)
				_ = buf
			}
		})
	})

	b.Run("ObjectPool", func(b *testing.B) {
		manager := performance.NewObjectPoolManager(performance.DefaultPoolConfig())
		defer manager.Stop()
		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := manager.GetBuffer(1024)
				manager.PutBuffer(buf)
			}
		})
	})
}

// BenchmarkHighFrequencyAllocation 高频分配测试
func BenchmarkHighFrequencyAllocation(b *testing.B) {
	const numGoroutines = 10
	const opsPerGoroutine = 1000

	b.Run("Direct", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup
			wg.Add(numGoroutines)

			for g := 0; g < numGoroutines; g++ {
				go func() {
					defer wg.Done()
					for op := 0; op < opsPerGoroutine; op++ {
						buf := make([]byte, 512)
						_ = buf
					}
				}()
			}
			wg.Wait()
		}
	})

	b.Run("ObjectPool", func(b *testing.B) {
		manager := performance.NewObjectPoolManager(performance.DefaultPoolConfig())
		defer manager.Stop()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup
			wg.Add(numGoroutines)

			for g := 0; g < numGoroutines; g++ {
				go func() {
					defer wg.Done()
					for op := 0; op < opsPerGoroutine; op++ {
						buf := manager.GetBuffer(512)
						manager.PutBuffer(buf)
					}
				}()
			}
			wg.Wait()
		}
	})
}

// BenchmarkMemoryReuse 内存复用效率测试
func BenchmarkMemoryReuse(b *testing.B) {
	manager := performance.NewObjectPoolManager(performance.DefaultPoolConfig())
	defer manager.Stop()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 连续获取和归还同样大小的缓冲区
		buf1 := manager.GetBuffer(256)
		buf2 := manager.GetBuffer(256)
		buf3 := manager.GetBuffer(256)

		manager.PutBuffer(buf1)
		manager.PutBuffer(buf2)
		manager.PutBuffer(buf3)

		// 再次获取，应该复用之前的对象
		reused1 := manager.GetBuffer(256)
		reused2 := manager.GetBuffer(256)
		reused3 := manager.GetBuffer(256)

		manager.PutBuffer(reused1)
		manager.PutBuffer(reused2)
		manager.PutBuffer(reused3)
	}
}
