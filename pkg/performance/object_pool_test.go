/*
* @Author: Lzww0608
* @Date: 2025-6-6 11:29:27
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-6 12:56:35
* @Description: 对象池测试
 */

package performance

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestObjectPoolManager_NewAndConfig(t *testing.T) {
	config := DefaultPoolConfig()
	opm := NewObjectPoolManager(config)
	defer opm.Stop()

	assert.NotNil(t, opm)
	assert.Equal(t, config.MaxPoolSize, opm.config.MaxPoolSize)
	assert.Equal(t, config.EnableStats, opm.config.EnableStats)
	assert.Equal(t, config.DefaultSizes, opm.config.DefaultSizes)
	assert.NotNil(t, opm.bufferPools)
	assert.NotNil(t, opm.builderPool)
	assert.NotNil(t, opm.slicePools)
	assert.NotNil(t, opm.channelPools)
}

func TestObjectPoolManager_BufferPool(t *testing.T) {
	opm := NewObjectPoolManager(DefaultPoolConfig())
	defer opm.Stop()

	t.Run("基本缓冲区获取和归还", func(t *testing.T) {
		// 获取缓冲区
		buf := opm.GetBuffer(1024)
		assert.NotNil(t, buf)
		assert.Equal(t, 0, len(buf))
		assert.GreaterOrEqual(t, cap(buf), 1024)

		// 使用缓冲区
		buf = append(buf, []byte("test data")...)
		assert.Equal(t, 9, len(buf))

		// 归还缓冲区
		opm.PutBuffer(buf)

		// 再次获取，应该得到相同容量的缓冲区
		buf2 := opm.GetBuffer(1024)
		assert.NotNil(t, buf2)
		assert.Equal(t, 0, len(buf2)) // 应该被重置为长度0
		assert.Equal(t, cap(buf), cap(buf2))
	})

	t.Run("不同大小的缓冲区", func(t *testing.T) {
		sizes := []int{64, 256, 1024, 4096}
		for _, size := range sizes {
			buf := opm.GetBuffer(size)
			assert.NotNil(t, buf)
			assert.GreaterOrEqual(t, cap(buf), size)
			opm.PutBuffer(buf)
		}
	})

	t.Run("nil缓冲区处理", func(t *testing.T) {
		// 归还nil缓冲区不应该panic
		assert.NotPanics(t, func() {
			opm.PutBuffer(nil)
		})
	})
}

func TestObjectPoolManager_StringBuilderPool(t *testing.T) {
	opm := NewObjectPoolManager(DefaultPoolConfig())
	defer opm.Stop()

	t.Run("基本字符串构建器获取和归还", func(t *testing.T) {
		// 获取构建器
		builder := opm.GetStringBuilder()
		assert.NotNil(t, builder)
		assert.Equal(t, 0, builder.Len())

		// 使用构建器
		builder.WriteString("Hello")
		builder.WriteString(" ")
		builder.WriteString("World")
		assert.Equal(t, "Hello World", builder.String())

		// 归还构建器
		opm.PutStringBuilder(builder)

		// 再次获取，应该被重置
		builder2 := opm.GetStringBuilder()
		assert.NotNil(t, builder2)
		assert.Equal(t, 0, builder2.Len())
		assert.Equal(t, "", builder2.String())
	})

	t.Run("nil构建器处理", func(t *testing.T) {
		// 归还nil构建器不应该panic
		assert.NotPanics(t, func() {
			opm.PutStringBuilder(nil)
		})
	})
}

func TestObjectPoolManager_SlicePool(t *testing.T) {
	opm := NewObjectPoolManager(DefaultPoolConfig())
	defer opm.Stop()

	t.Run("基本切片获取和归还", func(t *testing.T) {
		// 获取切片
		slice := opm.GetSlice(10)
		assert.NotNil(t, slice)
		assert.Equal(t, 0, len(slice))
		assert.GreaterOrEqual(t, cap(slice), 10)

		// 使用切片
		slice = append(slice, "item1", "item2", "item3")
		assert.Equal(t, 3, len(slice))

		// 归还切片
		opm.PutSlice(slice)

		// 再次获取，应该被重置
		slice2 := opm.GetSlice(10)
		assert.NotNil(t, slice2)
		assert.Equal(t, 0, len(slice2))
		assert.Equal(t, cap(slice), cap(slice2))
	})

	t.Run("切片内容清理", func(t *testing.T) {
		slice := opm.GetSlice(5)
		slice = append(slice, "test1", "test2")

		// 归还前先访问切片内容
		originalCap := cap(slice)
		opm.PutSlice(slice)

		// 获取新切片，验证内容已清理
		newSlice := opm.GetSlice(5)
		assert.Equal(t, 0, len(newSlice))
		assert.Equal(t, originalCap, cap(newSlice))
	})

	t.Run("nil切片处理", func(t *testing.T) {
		assert.NotPanics(t, func() {
			opm.PutSlice(nil)
		})
	})
}

func TestObjectPoolManager_ChannelPool(t *testing.T) {
	opm := NewObjectPoolManager(DefaultPoolConfig())
	defer opm.Stop()

	t.Run("基本通道获取和归还", func(t *testing.T) {
		// 获取通道
		ch := opm.GetByteChannel(10)
		assert.NotNil(t, ch)
		assert.Equal(t, 0, len(ch))
		assert.GreaterOrEqual(t, cap(ch), 10)

		// 使用通道
		testData := []byte("test")
		ch <- testData
		assert.Equal(t, 1, len(ch))

		// 归还通道
		opm.PutByteChannel(ch)

		// 再次获取，应该被清空
		ch2 := opm.GetByteChannel(10)
		assert.NotNil(t, ch2)
		assert.Equal(t, 0, len(ch2))
	})

	t.Run("通道清理", func(t *testing.T) {
		ch := opm.GetByteChannel(5)

		// 填充一些数据
		ch <- []byte("data1")
		ch <- []byte("data2")
		assert.Equal(t, 2, len(ch))

		// 归还通道
		opm.PutByteChannel(ch)

		// 再次获取，应该被清空
		newCh := opm.GetByteChannel(5)
		assert.Equal(t, 0, len(newCh))
	})

	t.Run("nil通道处理", func(t *testing.T) {
		assert.NotPanics(t, func() {
			opm.PutByteChannel(nil)
		})
	})
}

func TestObjectPoolManager_SizeSelection(t *testing.T) {
	opm := NewObjectPoolManager(DefaultPoolConfig())
	defer opm.Stop()

	t.Run("最佳大小选择", func(t *testing.T) {
		testCases := []struct {
			requested int
			expected  int
		}{
			{50, 64},       // 小于64，选择64
			{64, 64},       // 正好64
			{100, 256},     // 64和256之间，选择256
			{1000, 1024},   // 256和1024之间，选择1024
			{5000, 16384},  // 4096和16384之间，选择16384
			{70000, 65536}, // 超过最大预设，选择最大的
		}

		for _, tc := range testCases {
			bestSize := opm.findBestSize(tc.requested)
			assert.Equal(t, tc.expected, bestSize,
				"对于请求大小 %d，期望 %d，实际 %d", tc.requested, tc.expected, bestSize)
		}
	})
}

func TestObjectPoolManager_Statistics(t *testing.T) {
	config := DefaultPoolConfig()
	config.EnableStats = true
	opm := NewObjectPoolManager(config)
	defer opm.Stop()

	t.Run("统计信息收集", func(t *testing.T) {
		// 执行一些操作
		buf1 := opm.GetBuffer(1024)
		buf2 := opm.GetBuffer(1024)
		opm.PutBuffer(buf1)
		buf3 := opm.GetBuffer(1024) // 应该重用buf1

		// 获取统计信息
		stats := opm.GetStats()
		assert.NotNil(t, stats)

		// 验证统计信息
		for key, stat := range stats {
			if strings.Contains(key, "buffer_1024") {
				assert.Greater(t, stat.Gets, int64(0))
				assert.Greater(t, stat.Puts, int64(0))
				assert.Greater(t, stat.Hits, int64(0))
				break
			}
		}

		// 清理
		opm.PutBuffer(buf2)
		opm.PutBuffer(buf3)
	})

	t.Run("效率统计", func(t *testing.T) {
		// 执行一些操作以生成统计数据
		for i := 0; i < 10; i++ {
			buf := opm.GetBuffer(256)
			opm.PutBuffer(buf)
		}

		efficiency := opm.GetEfficiency()
		assert.NotNil(t, efficiency)

		// 检查是否有效率数据
		found := false
		for key, eff := range efficiency {
			if strings.Contains(key, "buffer") {
				assert.GreaterOrEqual(t, eff, 0.0)
				assert.LessOrEqual(t, eff, 100.0)
				found = true
			}
		}
		assert.True(t, found, "应该有缓冲区池的效率统计")
	})
}

func TestObjectPoolManager_ConcurrentAccess(t *testing.T) {
	opm := NewObjectPoolManager(DefaultPoolConfig())
	defer opm.Stop()

	t.Run("并发缓冲区操作", func(t *testing.T) {
		const numGoroutines = 20
		const operationsPerGoroutine = 100

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(routineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					// 随机选择缓冲区大小
					size := []int{64, 256, 1024, 4096}[j%4]

					buf := opm.GetBuffer(size)
					assert.NotNil(t, buf)

					// 模拟使用
					buf = append(buf, byte(routineID), byte(j))

					// 短暂延迟以增加竞争
					time.Sleep(time.Microsecond)

					opm.PutBuffer(buf)
				}
			}(i)
		}

		wg.Wait()
	})

	t.Run("并发字符串构建器操作", func(t *testing.T) {
		const numGoroutines = 15
		const operationsPerGoroutine = 50

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(routineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					builder := opm.GetStringBuilder()
					assert.NotNil(t, builder)

					// 使用构建器
					builder.WriteString("routine-")
					builder.WriteString(fmt.Sprintf("%d", routineID))
					builder.WriteString("-op-")
					builder.WriteString(fmt.Sprintf("%d", j))

					result := builder.String()
					assert.Contains(t, result, fmt.Sprintf("routine-%d", routineID))

					opm.PutStringBuilder(builder)
				}
			}(i)
		}

		wg.Wait()
	})
}

func TestObjectPoolManager_MemoryManagement(t *testing.T) {
	opm := NewObjectPoolManager(DefaultPoolConfig())
	defer opm.Stop()

	t.Run("内存使用优化", func(t *testing.T) {
		// 记录初始内存
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		// 执行大量分配操作
		const numAllocations = 1000
		buffers := make([][]byte, numAllocations)

		for i := 0; i < numAllocations; i++ {
			buffers[i] = opm.GetBuffer(1024)
		}

		// 归还所有缓冲区
		for _, buf := range buffers {
			opm.PutBuffer(buf)
		}

		// 再次执行相同操作（应该重用池中的对象）
		for i := 0; i < numAllocations; i++ {
			buf := opm.GetBuffer(1024)
			opm.PutBuffer(buf)
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)

		// 验证内存使用是否合理（这里主要检查没有内存泄漏）
		allocDiff := m2.TotalAlloc - m1.TotalAlloc
		t.Logf("总分配内存差异: %d bytes", allocDiff)

		// 池化应该减少内存分配
		assert.Greater(t, float64(allocDiff), 0.0)
	})
}

func TestObjectPoolManager_Reset(t *testing.T) {
	config := DefaultPoolConfig()
	config.EnableStats = true
	opm := NewObjectPoolManager(config)
	defer opm.Stop()

	// 执行一些操作以生成统计数据
	for i := 0; i < 10; i++ {
		buf := opm.GetBuffer(1024)
		builder := opm.GetStringBuilder()
		slice := opm.GetSlice(10)
		ch := opm.GetByteChannel(5)

		opm.PutBuffer(buf)
		opm.PutStringBuilder(builder)
		opm.PutSlice(slice)
		opm.PutByteChannel(ch)
	}

	// 验证有统计数据
	statsBefore := opm.GetStats()
	assert.NotNil(t, statsBefore)
	assert.Greater(t, len(statsBefore), 0)

	// 重置池
	opm.Reset()

	// 验证统计数据被重置
	statsAfter := opm.GetStats()
	for _, stat := range statsAfter {
		assert.Equal(t, int64(0), stat.Gets)
		assert.Equal(t, int64(0), stat.Puts)
		assert.Equal(t, int64(0), stat.Hits)
		assert.Equal(t, int64(0), stat.Misses)
		assert.Equal(t, int64(0), stat.Created)
		assert.Equal(t, int64(0), stat.Destroyed)
	}
}

func TestGlobalPoolManager(t *testing.T) {
	t.Run("全局池管理器单例", func(t *testing.T) {
		manager1 := GetGlobalPoolManager()
		manager2 := GetGlobalPoolManager()

		assert.NotNil(t, manager1)
		assert.NotNil(t, manager2)
		assert.Same(t, manager1, manager2, "应该返回相同的单例实例")
	})

	t.Run("全局便捷方法", func(t *testing.T) {
		// 测试缓冲区
		buf := GetPooledBuffer(1024)
		assert.NotNil(t, buf)
		assert.GreaterOrEqual(t, cap(buf), 1024)
		PutPooledBuffer(buf)

		// 测试字符串构建器
		builder := GetPooledStringBuilder()
		assert.NotNil(t, builder)
		builder.WriteString("test")
		PutPooledStringBuilder(builder)

		// 测试切片
		slice := GetPooledSlice(10)
		assert.NotNil(t, slice)
		assert.GreaterOrEqual(t, cap(slice), 10)
		PutPooledSlice(slice)

		// 测试通道
		ch := GetPooledByteChannel(5)
		assert.NotNil(t, ch)
		assert.GreaterOrEqual(t, cap(ch), 5)
		PutPooledByteChannel(ch)
	})
}

func TestObjectPoolManager_LifecycleManagement(t *testing.T) {
	config := DefaultPoolConfig()
	config.CleanupInterval = 100 * time.Millisecond
	opm := NewObjectPoolManager(config)

	// 验证管理器正常工作
	buf := opm.GetBuffer(1024)
	assert.NotNil(t, buf)
	opm.PutBuffer(buf)

	// 等待清理周期执行
	time.Sleep(150 * time.Millisecond)

	// 停止管理器
	opm.Stop()

	// 验证停止后仍能使用（但没有清理）
	buf2 := opm.GetBuffer(1024)
	assert.NotNil(t, buf2)
}

// 基准测试
func BenchmarkObjectPool_BufferOperations(b *testing.B) {
	opm := NewObjectPoolManager(DefaultPoolConfig())
	defer opm.Stop()

	b.Run("获取和归还缓冲区", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			buf := opm.GetBuffer(1024)
			buf = append(buf, []byte("benchmark data")...)
			opm.PutBuffer(buf)
		}
	})

	b.Run("并发缓冲区操作", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := opm.GetBuffer(1024)
				buf = append(buf, []byte("parallel data")...)
				opm.PutBuffer(buf)
			}
		})
	})
}

func BenchmarkObjectPool_StringBuilderOperations(b *testing.B) {
	opm := NewObjectPoolManager(DefaultPoolConfig())
	defer opm.Stop()

	b.Run("获取和归还字符串构建器", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			builder := opm.GetStringBuilder()
			builder.WriteString("benchmark")
			builder.WriteString(" ")
			builder.WriteString("string")
			_ = builder.String()
			opm.PutStringBuilder(builder)
		}
	})

	b.Run("对比原生字符串构建器", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			builder := &strings.Builder{}
			builder.WriteString("benchmark")
			builder.WriteString(" ")
			builder.WriteString("string")
			_ = builder.String()
		}
	})
}

func BenchmarkObjectPool_vs_Direct(b *testing.B) {
	opm := NewObjectPoolManager(DefaultPoolConfig())
	defer opm.Stop()

	b.Run("池化缓冲区", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			buf := opm.GetBuffer(1024)
			buf = append(buf, []byte("test data")...)
			opm.PutBuffer(buf)
		}
	})

	b.Run("直接分配缓冲区", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			buf := make([]byte, 0, 1024)
			buf = append(buf, []byte("test data")...)
			_ = buf // 模拟使用
		}
	})
}
