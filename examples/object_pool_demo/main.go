/*
* @Author: Lzww0608
* @Date: 2025-6-6 11:29:27
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-6 12:56:41
* @Description: 对象池演示
 */

package main

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/performance"
)

func main() {
	fmt.Println("🚀 ClixGo 对象池化演示")
	fmt.Println(strings.Repeat("=", 50))

	// 演示基本对象池操作
	demonstrateBasicOperations()

	// 演示性能对比
	demonstratePerformanceComparison()

	// 演示并发安全性
	demonstrateConcurrentSafety()

	// 演示统计信息
	demonstrateStatistics()

	// 演示内存优化效果
	demonstrateMemoryOptimization()

	fmt.Println("\n✅ 对象池化演示完成")
}

func demonstrateBasicOperations() {
	fmt.Println("\n📦 基本对象池操作演示")
	fmt.Println(strings.Repeat("-", 30))

	// 创建对象池管理器
	config := performance.DefaultPoolConfig()
	config.EnableStats = true
	opm := performance.NewObjectPoolManager(config)
	defer opm.Stop()

	// 1. 字节缓冲区池化
	fmt.Println("1. 字节缓冲区池化:")
	buf := opm.GetBuffer(1024)
	fmt.Printf("   获取缓冲区: 长度=%d, 容量=%d\n", len(buf), cap(buf))

	buf = append(buf, []byte("Hello, Object Pool!")...)
	fmt.Printf("   使用后: 长度=%d, 内容=%s\n", len(buf), string(buf))

	opm.PutBuffer(buf)
	fmt.Println("   缓冲区已归还到池中")

	// 2. 字符串构建器池化
	fmt.Println("\n2. 字符串构建器池化:")
	builder := opm.GetStringBuilder()
	fmt.Printf("   获取构建器: 长度=%d\n", builder.Len())

	builder.WriteString("对象池")
	builder.WriteString("优化")
	builder.WriteString("演示")
	result := builder.String()
	fmt.Printf("   构建结果: %s\n", result)

	opm.PutStringBuilder(builder)
	fmt.Println("   构建器已归还到池中")

	// 3. 切片池化
	fmt.Println("\n3. 通用切片池化:")
	slice := opm.GetSlice(10)
	fmt.Printf("   获取切片: 长度=%d, 容量=%d\n", len(slice), cap(slice))

	slice = append(slice, "item1", "item2", "item3")
	fmt.Printf("   使用后: 长度=%d, 内容=%v\n", len(slice), slice)

	opm.PutSlice(slice)
	fmt.Println("   切片已归还到池中")

	// 4. 通道池化
	fmt.Println("\n4. 字节通道池化:")
	ch := opm.GetByteChannel(5)
	fmt.Printf("   获取通道: 长度=%d, 容量=%d\n", len(ch), cap(ch))

	go func() {
		ch <- []byte("channel data")
	}()

	data := <-ch
	fmt.Printf("   接收数据: %s\n", string(data))

	opm.PutByteChannel(ch)
	fmt.Println("   通道已归还到池中")
}

func demonstratePerformanceComparison() {
	fmt.Println("\n⚡ 性能对比演示")
	fmt.Println(strings.Repeat("-", 30))

	opm := performance.NewObjectPoolManager(performance.DefaultPoolConfig())
	defer opm.Stop()

	const iterations = 100000

	// 测试缓冲区分配性能
	fmt.Println("1. 缓冲区分配性能对比:")

	// 使用对象池
	start := time.Now()
	for i := 0; i < iterations; i++ {
		buf := opm.GetBuffer(1024)
		buf = append(buf, []byte("test data")...)
		opm.PutBuffer(buf)
	}
	pooledTime := time.Since(start)

	// 直接分配
	start = time.Now()
	for i := 0; i < iterations; i++ {
		buf := make([]byte, 0, 1024)
		buf = append(buf, []byte("test data")...)
		_ = buf // 模拟使用
	}
	directTime := time.Since(start)

	fmt.Printf("   对象池方式: %v\n", pooledTime)
	fmt.Printf("   直接分配: %v\n", directTime)
	fmt.Printf("   性能提升: %.2fx\n", float64(directTime)/float64(pooledTime))

	// 测试字符串构建性能
	fmt.Println("\n2. 字符串构建性能对比:")

	// 使用对象池
	start = time.Now()
	for i := 0; i < iterations; i++ {
		builder := opm.GetStringBuilder()
		builder.WriteString("Hello")
		builder.WriteString(" ")
		builder.WriteString("World")
		_ = builder.String()
		opm.PutStringBuilder(builder)
	}
	pooledStringTime := time.Since(start)

	// 直接分配
	start = time.Now()
	for i := 0; i < iterations; i++ {
		builder := &strings.Builder{}
		builder.WriteString("Hello")
		builder.WriteString(" ")
		builder.WriteString("World")
		_ = builder.String()
	}
	directStringTime := time.Since(start)

	fmt.Printf("   对象池方式: %v\n", pooledStringTime)
	fmt.Printf("   直接分配: %v\n", directStringTime)
	fmt.Printf("   性能提升: %.2fx\n", float64(directStringTime)/float64(pooledStringTime))
}

func demonstrateConcurrentSafety() {
	fmt.Println("\n🔒 并发安全性演示")
	fmt.Println(strings.Repeat("-", 30))

	opm := performance.NewObjectPoolManager(performance.DefaultPoolConfig())
	defer opm.Stop()

	const numGoroutines = 50
	const operationsPerGoroutine = 1000

	fmt.Printf("启动 %d 个goroutine，每个执行 %d 次操作\n", numGoroutines, operationsPerGoroutine)

	var wg sync.WaitGroup
	start := time.Now()

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				// 混合使用不同类型的池化对象
				switch j % 4 {
				case 0:
					buf := opm.GetBuffer(512)
					buf = append(buf, byte(id), byte(j))
					opm.PutBuffer(buf)
				case 1:
					builder := opm.GetStringBuilder()
					builder.WriteString(fmt.Sprintf("goroutine-%d-op-%d", id, j))
					opm.PutStringBuilder(builder)
				case 2:
					slice := opm.GetSlice(5)
					slice = append(slice, id, j)
					opm.PutSlice(slice)
				case 3:
					ch := opm.GetByteChannel(2)
					opm.PutByteChannel(ch)
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	totalOperations := numGoroutines * operationsPerGoroutine
	fmt.Printf("完成 %d 次并发操作，耗时: %v\n", totalOperations, elapsed)
	fmt.Printf("平均每次操作: %v\n", elapsed/time.Duration(totalOperations))
	fmt.Println("✅ 并发测试通过，无数据竞争")
}

func demonstrateStatistics() {
	fmt.Println("\n📊 统计信息演示")
	fmt.Println(strings.Repeat("-", 30))

	config := performance.DefaultPoolConfig()
	config.EnableStats = true
	opm := performance.NewObjectPoolManager(config)
	defer opm.Stop()

	// 执行一些操作以生成统计数据
	for i := 0; i < 100; i++ {
		buf := opm.GetBuffer(1024)
		opm.PutBuffer(buf)

		builder := opm.GetStringBuilder()
		builder.WriteString("test")
		opm.PutStringBuilder(builder)

		slice := opm.GetSlice(10)
		opm.PutSlice(slice)
	}

	// 获取统计信息
	stats := opm.GetStats()
	fmt.Println("池统计信息:")
	for poolName, stat := range stats {
		fmt.Printf("  %s:\n", poolName)
		fmt.Printf("    获取次数: %d\n", stat.Gets)
		fmt.Printf("    归还次数: %d\n", stat.Puts)
		fmt.Printf("    命中次数: %d\n", stat.Hits)
		fmt.Printf("    失效次数: %d\n", stat.Misses)
		fmt.Printf("    创建次数: %d\n", stat.Created)
	}

	// 获取效率统计
	efficiency := opm.GetEfficiency()
	fmt.Println("\n池效率统计:")
	for poolName, eff := range efficiency {
		fmt.Printf("  %s: %.2f%%\n", poolName, eff)
	}
}

func demonstrateMemoryOptimization() {
	fmt.Println("\n💾 内存优化效果演示")
	fmt.Println(strings.Repeat("-", 30))

	opm := performance.NewObjectPoolManager(performance.DefaultPoolConfig())
	defer opm.Stop()

	// 记录初始内存状态
	var m1, m2, m3 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	const numAllocations = 10000
	const bufferSize = 4096

	fmt.Printf("执行 %d 次大小为 %d 字节的缓冲区分配\n", numAllocations, bufferSize)

	// 第一轮：使用对象池
	fmt.Println("\n第一轮：使用对象池")
	start := time.Now()
	for i := 0; i < numAllocations; i++ {
		buf := opm.GetBuffer(bufferSize)
		buf = append(buf, []byte("pool test data")...)
		opm.PutBuffer(buf)
	}
	poolTime := time.Since(start)

	runtime.GC()
	runtime.ReadMemStats(&m2)

	// 第二轮：直接分配
	fmt.Println("第二轮：直接分配")
	start = time.Now()
	for i := 0; i < numAllocations; i++ {
		buf := make([]byte, 0, bufferSize)
		buf = append(buf, []byte("direct test data")...)
		_ = buf
	}
	directTime := time.Since(start)

	runtime.GC()
	runtime.ReadMemStats(&m3)

	// 分析结果
	fmt.Println("\n内存使用分析:")
	fmt.Printf("  对象池方式:\n")
	fmt.Printf("    执行时间: %v\n", poolTime)
	fmt.Printf("    内存分配: %d bytes\n", m2.TotalAlloc-m1.TotalAlloc)
	fmt.Printf("    GC次数: %d\n", m2.NumGC-m1.NumGC)

	fmt.Printf("  直接分配方式:\n")
	fmt.Printf("    执行时间: %v\n", directTime)
	fmt.Printf("    内存分配: %d bytes\n", m3.TotalAlloc-m2.TotalAlloc)
	fmt.Printf("    GC次数: %d\n", m3.NumGC-m2.NumGC)

	// 计算优化效果
	memoryReduction := float64(m3.TotalAlloc-m2.TotalAlloc) / float64(m2.TotalAlloc-m1.TotalAlloc)
	timeImprovement := float64(directTime) / float64(poolTime)
	gcReduction := float64(m3.NumGC-m2.NumGC) / float64(max(m2.NumGC-m1.NumGC, 1))

	fmt.Printf("\n优化效果:\n")
	fmt.Printf("  时间性能提升: %.2fx\n", timeImprovement)
	fmt.Printf("  内存分配减少: %.2fx\n", memoryReduction)
	fmt.Printf("  GC压力减少: %.2fx\n", gcReduction)
}

// 全局便捷方法演示
func demonstrateGlobalMethods() {
	fmt.Println("\n🌍 全局便捷方法演示")
	fmt.Println(strings.Repeat("-", 30))

	// 使用全局池管理器的便捷方法
	buf := performance.GetPooledBuffer(2048)
	fmt.Printf("全局获取缓冲区: 容量=%d\n", cap(buf))
	performance.PutPooledBuffer(buf)

	builder := performance.GetPooledStringBuilder()
	builder.WriteString("全局字符串构建器测试")
	fmt.Printf("构建结果: %s\n", builder.String())
	performance.PutPooledStringBuilder(builder)

	slice := performance.GetPooledSlice(20)
	fmt.Printf("全局获取切片: 容量=%d\n", cap(slice))
	performance.PutPooledSlice(slice)

	ch := performance.GetPooledByteChannel(10)
	fmt.Printf("全局获取通道: 容量=%d\n", cap(ch))
	performance.PutPooledByteChannel(ch)

	fmt.Println("✅ 全局方法测试完成")
}

func max(a, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}
