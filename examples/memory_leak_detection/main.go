/*
* @Author: Lzww0608
* @Date: 2025-06-04 10:15:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-4 12:50:46
* @Description: 内存泄漏检测器使用示例
 */

package main

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/performance"
	"go.uber.org/zap"
)

func main() {
	// 创建日志记录器
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 配置内存泄漏检测器
	config := performance.MemoryLeakDetectorConfig{
		CheckInterval:            5 * time.Second,
		BaselineWarmupTime:       3 * time.Second,
		MaxSnapshots:             20,
		EnablePprof:              false,
		GoroutineGrowthThreshold: 10,
		MemoryGrowthThresholdMB:  50.0,
		HeapGrowthThresholdMB:    30.0,
		GCFrequencyThreshold:     20,
		TimerGrowthThreshold:     5,
	}

	// 创建并启动检测器
	detector := performance.NewMemoryLeakDetector(config, logger)

	fmt.Println("🔍 启动内存泄漏检测器...")
	err := detector.Start()
	if err != nil {
		logger.Fatal("启动检测器失败", zap.Error(err))
	}
	defer detector.Stop()

	// 启动监控goroutine
	go monitorAlerts(detector, logger)
	go monitorLeaks(detector, logger)
	go monitorErrors(detector, logger)

	fmt.Println("⏳ 等待基线建立...")
	time.Sleep(4 * time.Second)

	// 显示基线信息
	baseline := detector.GetBaseline()
	if baseline != nil {
		fmt.Printf("📊 基线信息:\n")
		fmt.Printf("  - Goroutine数量: %d\n", baseline.GoroutineCount)
		fmt.Printf("  - 堆内存: %.2f MB\n", baseline.HeapAllocMB)
		fmt.Printf("  - GC次数: %d\n", baseline.GCCount)
		fmt.Printf("  - 定时器数量: %d\n", baseline.TimerCount)
	}

	// 演示各种泄漏场景
	fmt.Println("\n🧪 开始泄漏检测演示...")

	// 1. Goroutine泄漏演示
	fmt.Println("\n1️⃣ 模拟Goroutine泄漏...")
	simulateGoroutineLeak()

	time.Sleep(6 * time.Second)

	// 2. 内存泄漏演示
	fmt.Println("\n2️⃣ 模拟内存泄漏...")
	simulateMemoryLeak()

	time.Sleep(6 * time.Second)

	// 3. 定时器泄漏演示
	fmt.Println("\n3️⃣ 模拟定时器泄漏...")
	simulateTimerLeak()

	time.Sleep(6 * time.Second)

	// 强制执行检查
	fmt.Println("\n🔍 执行强制检查...")
	result, err := detector.ForceCheck()
	if err != nil {
		logger.Error("强制检查失败", zap.Error(err))
	} else {
		fmt.Printf("检查结果: 是否有泄漏=%v, 类型=%s, 置信度=%.2f\n",
			result.HasLeak, result.LeakType, result.Confidence)
	}

	// 显示历史快照
	snapshots := detector.GetSnapshots()
	fmt.Printf("\n📈 收集了 %d 个快照\n", len(snapshots))

	if len(snapshots) > 0 {
		latest := snapshots[len(snapshots)-1]
		fmt.Printf("最新快照:\n")
		fmt.Printf("  - Goroutine数量: %d\n", latest.GoroutineCount)
		fmt.Printf("  - 堆内存: %.2f MB\n", latest.HeapAllocMB)
		fmt.Printf("  - GC次数: %d\n", latest.GCCount)
		fmt.Printf("  - 可疑模式: %d 个\n", len(latest.SuspiciousPatterns))
	}

	fmt.Println("\n✅ 演示完成，等待最后的检测...")
	time.Sleep(3 * time.Second)
}

// simulateGoroutineLeak 模拟goroutine泄漏
func simulateGoroutineLeak() {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建一些永不退出的goroutine（模拟泄漏）
	for i := 0; i < 15; i++ {
		go func(id int) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second): // 长时间运行
				return
			}
		}(i)
	}

	// 延迟取消，模拟泄漏一段时间
	go func() {
		time.Sleep(20 * time.Second)
		cancel()
	}()
}

// simulateMemoryLeak 模拟内存泄漏
func simulateMemoryLeak() {
	// 创建一个全局变量来持有内存引用（模拟泄漏）
	var leakedData [][]byte

	// 分配大量内存
	for i := 0; i < 80; i++ {
		data := make([]byte, 1024*1024) // 1MB
		leakedData = append(leakedData, data)
	}

	// 延迟释放内存
	go func() {
		time.Sleep(15 * time.Second)
		leakedData = nil
		runtime.GC()
	}()
}

// simulateTimerLeak 模拟定时器泄漏
func simulateTimerLeak() {
	var timers []*time.Timer

	// 创建一些定时器但不停止它们（模拟泄漏）
	for i := 0; i < 10; i++ {
		timer := time.NewTimer(30 * time.Second)
		timers = append(timers, timer)
	}

	// 延迟停止定时器
	go func() {
		time.Sleep(15 * time.Second)
		for _, timer := range timers {
			timer.Stop()
		}
	}()
}

// monitorAlerts 监控告警
func monitorAlerts(detector *performance.MemoryLeakDetector, logger *zap.Logger) {
	for alert := range detector.GetAlertChannel() {
		fmt.Printf("\n🚨 内存泄漏告警!\n")
		fmt.Printf("  ID: %s\n", alert.ID)
		fmt.Printf("  类型: %s\n", alert.Type)
		fmt.Printf("  严重程度: %s\n", alert.Severity)
		fmt.Printf("  标题: %s\n", alert.Title)
		fmt.Printf("  描述: %s\n", alert.Description)
		fmt.Printf("  证据数量: %d\n", len(alert.Evidence))
		fmt.Printf("  建议数量: %d\n", len(alert.Suggestions))

		if len(alert.Evidence) > 0 {
			fmt.Printf("  主要证据: %s\n", alert.Evidence[0])
		}
		if len(alert.Suggestions) > 0 {
			fmt.Printf("  主要建议: %s\n", alert.Suggestions[0])
		}
	}
}

// monitorLeaks 监控泄漏检测结果
func monitorLeaks(detector *performance.MemoryLeakDetector, logger *zap.Logger) {
	for result := range detector.GetLeakDetectedChannel() {
		fmt.Printf("\n🔍 检测到泄漏!\n")
		fmt.Printf("  类型: %s\n", result.LeakType)
		fmt.Printf("  置信度: %.2f\n", result.Confidence)
		fmt.Printf("  描述: %s\n", result.Description)
		fmt.Printf("  影响资源: %v\n", result.AffectedResources)
	}
}

// monitorErrors 监控错误
func monitorErrors(detector *performance.MemoryLeakDetector, logger *zap.Logger) {
	for err := range detector.GetErrorChannel() {
		logger.Warn("检测器错误", zap.Error(err))
	}
}
