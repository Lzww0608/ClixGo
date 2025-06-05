/*
* @Author: Lzww0608
* @Date: 2025-06-05 11:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-05 20:19:33
* @Description: 内存优化监控器使用示例 - 展示pprof集成和内存优化功能
 */

package main

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/performance"
	"go.uber.org/zap"
)

func main() {
	// 创建日志记录器
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 配置内存监控器
	config := performance.MemoryMonitorConfig{
		MonitorInterval:        3 * time.Second,
		BaselineWarmupTime:     2 * time.Second,
		MaxSnapshots:           50,
		EnablePprof:            true,
		PprofAddress:           ":6060",
		EnableAutoOptimization: true,
		OptimizationInterval:   10 * time.Second,
		ProfileCollectionDepth: 10,

		// 告警阈值设置
		MemoryGrowthThresholdMB: 30.0,
		HeapGrowthThresholdMB:   20.0,
		GCPressureThreshold:     0.05,
		FragmentationThreshold:  0.25,

		// 优化参数
		AutoGCTriggerThresholdMB: 50.0,
		MemoryReleaseThresholdMB: 40.0,
		MaxOptimizationRetries:   5,
	}

	// 创建并启动内存监控器
	monitor := performance.NewMemoryMonitor(config, logger)

	fmt.Println("🚀 启动内存监控器...")
	err := monitor.Start()
	if err != nil {
		logger.Fatal("启动内存监控器失败", zap.Error(err))
	}
	defer monitor.Stop()

	// 启动监控goroutine
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监控告警
	wg.Add(1)
	go func() {
		defer wg.Done()
		monitorAlerts(ctx, monitor, logger)
	}()

	// 监控优化建议
	wg.Add(1)
	go func() {
		defer wg.Done()
		monitorOptimizations(ctx, monitor, logger)
	}()

	// 监控错误
	wg.Add(1)
	go func() {
		defer wg.Done()
		monitorErrors(ctx, monitor, logger)
	}()

	// 显示启动信息
	fmt.Printf("📊 pprof服务器已启动: http://localhost%s\n", config.PprofAddress)
	fmt.Println("📋 可用的profiling端点:")
	fmt.Println("   - http://localhost:6060/debug/pprof/")
	fmt.Println("   - http://localhost:6060/debug/pprof/heap")
	fmt.Println("   - http://localhost:6060/debug/pprof/goroutine")
	fmt.Println("   - http://localhost:6060/debug/pprof/allocs")
	fmt.Println()

	fmt.Println("⏳ 等待基线建立...")
	time.Sleep(3 * time.Second)

	// 显示基线信息
	baseline := monitor.GetBaseline()
	if baseline != nil {
		fmt.Printf("📊 内存基线信息:\n")
		fmt.Printf("  - 堆内存分配: %.2f MB\n", baseline.HeapAllocMB)
		fmt.Printf("  - 堆系统内存: %.2f MB\n", baseline.HeapSysMB)
		fmt.Printf("  - 栈内存: %.2f MB\n", baseline.StackInuseMB)
		fmt.Printf("  - GC CPU比例: %.4f\n", baseline.GCCPUFraction)
		fmt.Printf("  - Goroutine数量: %d\n", baseline.GoroutineCount)
		fmt.Println()
	}

	// 演示内存使用场景
	fmt.Println("🧪 开始内存使用演示...")

	// 场景1: 正常内存分配
	fmt.Println("\n1️⃣ 正常内存分配模式")
	simulateNormalMemoryUsage()
	time.Sleep(5 * time.Second)

	// 场景2: 内存泄漏模拟
	fmt.Println("\n2️⃣ 模拟内存泄漏")
	leakedMemory := simulateMemoryLeak()
	time.Sleep(8 * time.Second)

	// 场景3: 内存碎片化
	fmt.Println("\n3️⃣ 模拟内存碎片化")
	fragmentedMemory := simulateMemoryFragmentation()
	time.Sleep(8 * time.Second)

	// 场景4: 大量小对象分配
	fmt.Println("\n4️⃣ 大量小对象分配")
	simulateSmallObjectAllocation()
	time.Sleep(5 * time.Second)

	// 强制执行优化
	fmt.Println("\n🔧 强制执行内存优化...")
	err = monitor.ForceOptimization()
	if err != nil {
		logger.Error("强制优化失败", zap.Error(err))
	}
	time.Sleep(3 * time.Second)

	// 显示监控统计
	showMonitoringStats(monitor, logger)

	// 清理分配的内存
	fmt.Println("\n🧹 清理分配的内存...")
	_ = leakedMemory     // 保持引用避免编译器错误
	_ = fragmentedMemory // 保持引用避免编译器错误
	leakedMemory = nil
	fragmentedMemory = nil
	runtime.GC()
	time.Sleep(2 * time.Second)

	// 最终统计
	fmt.Println("\n📈 最终内存统计:")
	showFinalStats(monitor, baseline)

	fmt.Println("\n✅ 演示完成")
	fmt.Println("💡 提示: 可以使用以下工具分析内存:")
	fmt.Println("   go tool pprof http://localhost:6060/debug/pprof/heap")
	fmt.Println("   go tool pprof http://localhost:6060/debug/pprof/allocs")

	// 保持程序运行一段时间供外部工具分析
	fmt.Println("\n⏰ 保持服务运行30秒供pprof分析...")
	time.Sleep(30 * time.Second)

	// 停止监控
	cancel()

	// 等待所有goroutine结束
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("✅ 所有监控goroutine已安全停止")
	case <-time.After(5 * time.Second):
		fmt.Println("⚠️  等待监控goroutine停止超时")
	}
}

// simulateNormalMemoryUsage 模拟正常内存使用
func simulateNormalMemoryUsage() {
	data := make([][]byte, 20)
	for i := 0; i < 20; i++ {
		data[i] = make([]byte, 1024*1024) // 1MB
		time.Sleep(100 * time.Millisecond)
	}
	// 自动释放
}

// simulateMemoryLeak 模拟内存泄漏
func simulateMemoryLeak() [][]byte {
	var leakedData [][]byte

	// 分配内存但不释放
	for i := 0; i < 60; i++ {
		data := make([]byte, 1024*1024) // 1MB
		leakedData = append(leakedData, data)
		if i%10 == 0 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	return leakedData
}

// simulateMemoryFragmentation 模拟内存碎片化
func simulateMemoryFragmentation() []interface{} {
	var fragments []interface{}

	// 分配不同大小的内存块造成碎片化
	sizes := []int{128, 1024, 4096, 16384, 65536, 262144}

	for i := 0; i < 200; i++ {
		size := sizes[i%len(sizes)]
		data := make([]byte, size)
		fragments = append(fragments, data)

		// 随机释放一些内存块
		if i%5 == 0 && len(fragments) > 10 {
			fragments = fragments[1:]
		}

		if i%20 == 0 {
			time.Sleep(50 * time.Millisecond)
		}
	}

	return fragments
}

// simulateSmallObjectAllocation 模拟大量小对象分配
func simulateSmallObjectAllocation() {
	type SmallObject struct {
		ID   int
		Data [64]byte
	}

	objects := make([]*SmallObject, 10000)
	for i := 0; i < 10000; i++ {
		objects[i] = &SmallObject{
			ID: i,
		}

		if i%1000 == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// 触发GC
	runtime.GC()
}

// monitorAlerts 监控告警
func monitorAlerts(ctx context.Context, monitor *performance.MemoryMonitor, logger *zap.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case alert := <-monitor.GetAlertChannel():
			fmt.Printf("\n🚨 内存告警!\n")
			fmt.Printf("  ID: %s\n", alert.ID)
			fmt.Printf("  类型: %s\n", alert.Type)
			fmt.Printf("  严重程度: %s\n", alert.Severity)
			fmt.Printf("  标题: %s\n", alert.Title)
			fmt.Printf("  描述: %s\n", alert.Description)

			if len(alert.Evidence) > 0 {
				fmt.Printf("  主要证据: %s\n", alert.Evidence[0])
			}
			if len(alert.Suggestions) > 0 {
				fmt.Printf("  建议: %s\n", alert.Suggestions[0])
			}

			// 显示关键指标
			if alert.MetricValues != nil {
				fmt.Printf("  关键指标:\n")
				for key, value := range alert.MetricValues {
					fmt.Printf("    %s: %.2f\n", key, value)
				}
			}
			fmt.Println()
		}
	}
}

// monitorOptimizations 监控优化建议
func monitorOptimizations(ctx context.Context, monitor *performance.MemoryMonitor, logger *zap.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case suggestion := <-monitor.GetOptimizationChannel():
			fmt.Printf("\n💡 内存优化建议!\n")
			fmt.Printf("  ID: %s\n", suggestion.ID)
			fmt.Printf("  类型: %s\n", suggestion.Type)
			fmt.Printf("  优先级: %s\n", suggestion.Priority)
			fmt.Printf("  标题: %s\n", suggestion.Title)
			fmt.Printf("  描述: %s\n", suggestion.Description)

			fmt.Printf("  预期影响:\n")
			fmt.Printf("    内存节省: %.2f MB\n", suggestion.Impact.EstimatedMemorySavingMB)
			fmt.Printf("    CPU节省: %.2f%%\n", suggestion.Impact.EstimatedCPUSaving)
			fmt.Printf("    GC改善: %.2f%%\n", suggestion.Impact.EstimatedGCImprovement)
			fmt.Printf("    风险级别: %s\n", suggestion.Impact.RiskLevel)

			fmt.Printf("  优化操作:\n")
			for i, action := range suggestion.Actions {
				status := "待执行"
				if action.Executed {
					status = "已执行"
				}
				fmt.Printf("    %d. %s - %s [%s]\n", i+1, action.Action, action.Description, status)
			}
			fmt.Println()
		}
	}
}

// monitorErrors 监控错误
func monitorErrors(ctx context.Context, monitor *performance.MemoryMonitor, logger *zap.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-monitor.GetErrorChannel():
			fmt.Printf("\n❌ 监控错误: %v\n", err)
		}
	}
}

// showMonitoringStats 显示监控统计
func showMonitoringStats(monitor *performance.MemoryMonitor, logger *zap.Logger) {
	snapshots := monitor.GetSnapshots()

	fmt.Printf("📊 监控统计信息:\n")
	fmt.Printf("  收集快照数: %d\n", len(snapshots))

	if len(snapshots) > 0 {
		latest := snapshots[len(snapshots)-1]
		fmt.Printf("  最新快照:\n")
		fmt.Printf("    堆内存分配: %.2f MB\n", latest.HeapAllocMB)
		fmt.Printf("    堆系统内存: %.2f MB\n", latest.HeapSysMB)
		fmt.Printf("    内存碎片化比率: %.3f\n", latest.FragmentationRatio)
		fmt.Printf("    GC压力指示器: %.3f\n", latest.GCPressureIndicator)
		fmt.Printf("    GC次数: %d\n", latest.GCCount)
		fmt.Printf("    Goroutine数量: %d\n", latest.GoroutineCount)
		fmt.Printf("    内存热点数: %d\n", len(latest.MemoryHotspots))
	}
	fmt.Println()
}

// showFinalStats 显示最终统计
func showFinalStats(monitor *performance.MemoryMonitor, baseline *performance.MemoryBaseline) {
	snapshot, err := monitor.GetCurrentSnapshot()
	if err != nil {
		fmt.Printf("获取最终快照失败: %v\n", err)
		return
	}

	if baseline != nil {
		heapGrowth := snapshot.HeapAllocMB - baseline.HeapAllocMB
		goroutineGrowth := snapshot.GoroutineCount - baseline.GoroutineCount

		fmt.Printf("  内存变化 (vs 基线):\n")
		fmt.Printf("    堆内存变化: %+.2f MB\n", heapGrowth)
		fmt.Printf("    Goroutine变化: %+d\n", goroutineGrowth)
		fmt.Printf("    GC次数变化: %+d\n", int(snapshot.GCCount))
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("  当前Go运行时状态:\n")
	fmt.Printf("    Alloc: %.2f MB\n", float64(m.Alloc)/1024/1024)
	fmt.Printf("    TotalAlloc: %.2f MB\n", float64(m.TotalAlloc)/1024/1024)
	fmt.Printf("    Sys: %.2f MB\n", float64(m.Sys)/1024/1024)
	fmt.Printf("    NumGC: %d\n", m.NumGC)
	fmt.Printf("    GCCPUFraction: %.4f\n", m.GCCPUFraction)
}
