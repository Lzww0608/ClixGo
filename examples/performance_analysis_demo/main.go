/*
* @Author: Lzww0608
* @Date: 2025-6-17 16:50:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-17 16:50:00
* @Description: Phase 1.3 任务1.2 - 性能分析演示程序
 */

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

// PerformanceAnalysis 性能分析结构
type PerformanceAnalysis struct {
	StartupTimeNS       int64                  `json:"startup_time_ns"`
	MemoryUsageBytes    int64                  `json:"memory_usage_bytes"`
	SessionCreateTimeNS int64                  `json:"session_create_time_ns"`
	ShutdownTimeNS      int64                  `json:"shutdown_time_ns"`
	GoroutineCount      int                    `json:"goroutine_count"`
	ObjectPoolStats     map[string]interface{} `json:"object_pool_stats"`
	LeakDetectorStats   map[string]interface{} `json:"leak_detector_stats"`
	Timestamp           time.Time              `json:"timestamp"`
}

func main() {
	// 初始化日志系统
	logger.InitLogger()
	defer logger.Close()

	fmt.Println("=== ClixGo Phase 1.3 性能分析演示 ===")
	fmt.Println("正在执行任务1.2: 性能分析深化...")

	// 执行性能分析
	analysis := performAnalysis()

	// 输出分析结果
	printAnalysisResults(analysis)

	// 保存分析报告
	saveAnalysisReport(analysis)

	fmt.Println("\n✅ 任务1.2完成: 性能分析深化成功")
}

func performAnalysis() *PerformanceAnalysis {
	analysis := &PerformanceAnalysis{
		Timestamp: time.Now(),
	}

	fmt.Println("\n📊 开始性能分析...")

	// 1. 测试启动时间
	fmt.Print("  🚀 测试启动时间... ")
	startTime := time.Now()
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}
	manager := terminal.NewSessionManager(config)
	analysis.StartupTimeNS = time.Since(startTime).Nanoseconds()
	fmt.Printf("%.2fms\n", float64(analysis.StartupTimeNS)/1e6)

	// 2. 测试内存使用
	fmt.Print("  💾 测试内存使用... ")
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// 创建一些会话进行测试
	sessionCount := 10
	sessions := make([]*terminal.Session, 0, sessionCount)

	sessionCreateStart := time.Now()
	for i := 0; i < sessionCount; i++ {
		sessionName := fmt.Sprintf("analysis-session-%d", i)
		session, err := manager.CreateSession(sessionName)
		if err != nil {
			fmt.Printf("警告: 创建会话失败: %v\n", err)
			continue
		}
		sessions = append(sessions, session)
	}
	analysis.SessionCreateTimeNS = time.Since(sessionCreateStart).Nanoseconds() / int64(len(sessions))

	runtime.GC()
	runtime.ReadMemStats(&m2)
	analysis.MemoryUsageBytes = int64(m2.Alloc - m1.Alloc)
	analysis.GoroutineCount = runtime.NumGoroutine()
	fmt.Printf("%.2fMB\n", float64(analysis.MemoryUsageBytes)/1024/1024)

	// 3. 获取对象池统计
	fmt.Print("  🔄 分析对象池效率... ")
	objectPool := manager.GetObjectPool()
	if objectPool != nil {
		stats := manager.GetPerformanceStats()
		analysis.ObjectPoolStats = map[string]interface{}{
			"buffer_pool_hits":   stats.BufferPoolHits,
			"buffer_pool_misses": stats.BufferPoolMisses,
			"avg_create_time_ns": stats.AvgCreateTime.Nanoseconds(),
			"memory_usage_mb":    stats.MemoryUsageMB,
		}

		if stats.BufferPoolHits+stats.BufferPoolMisses > 0 {
			hitRate := float64(stats.BufferPoolHits) / float64(stats.BufferPoolHits+stats.BufferPoolMisses) * 100
			fmt.Printf("命中率 %.1f%%\n", hitRate)
		} else {
			fmt.Println("无数据")
		}
	} else {
		fmt.Println("未启用")
	}

	// 4. 获取内存泄漏检测器统计
	fmt.Print("  🔍 检查内存泄漏检测... ")
	leakDetector := manager.GetLeakDetector()
	if leakDetector != nil {
		analysis.LeakDetectorStats = map[string]interface{}{
			"enabled":           true,
			"check_interval_s":  30,
			"baseline_warmup_s": 60,
		}
		fmt.Println("运行中")
	} else {
		analysis.LeakDetectorStats = map[string]interface{}{
			"enabled": false,
		}
		fmt.Println("未启用")
	}

	// 5. 测试关闭时间
	fmt.Print("  🛑 测试关闭时间... ")
	shutdownStart := time.Now()
	err := manager.Shutdown()
	if err != nil {
		fmt.Printf("关闭失败: %v\n", err)
	} else {
		analysis.ShutdownTimeNS = time.Since(shutdownStart).Nanoseconds()
		fmt.Printf("%.2fms\n", float64(analysis.ShutdownTimeNS)/1e6)
	}

	return analysis
}

func printAnalysisResults(analysis *PerformanceAnalysis) {
	fmt.Println("\n📈 性能分析结果:")
	fmt.Println("================")

	fmt.Printf("🚀 启动时间:     %.3f ms\n", float64(analysis.StartupTimeNS)/1e6)
	fmt.Printf("💾 内存使用:     %.2f MB\n", float64(analysis.MemoryUsageBytes)/1024/1024)
	fmt.Printf("⚡ 会话创建:     %.3f ms/session\n", float64(analysis.SessionCreateTimeNS)/1e6)
	fmt.Printf("🛑 关闭时间:     %.3f ms\n", float64(analysis.ShutdownTimeNS)/1e6)
	fmt.Printf("🔄 协程数量:     %d\n", analysis.GoroutineCount)

	if analysis.ObjectPoolStats != nil {
		fmt.Println("\n🔄 对象池统计:")
		if hits, ok := analysis.ObjectPoolStats["buffer_pool_hits"]; ok {
			fmt.Printf("   命中次数:     %v\n", hits)
		}
		if misses, ok := analysis.ObjectPoolStats["buffer_pool_misses"]; ok {
			fmt.Printf("   失误次数:     %v\n", misses)
		}
		if avgTime, ok := analysis.ObjectPoolStats["avg_create_time_ns"]; ok {
			fmt.Printf("   平均创建时间: %.3f ms\n", float64(avgTime.(int64))/1e6)
		}
	}

	fmt.Println("\n🎯 性能目标对比:")
	fmt.Println("================")

	// 与ROADMAP目标对比
	targetStartup := 30.0 // ms
	targetMemory := 8.0   // MB

	startupMs := float64(analysis.StartupTimeNS) / 1e6
	memoryMB := float64(analysis.MemoryUsageBytes) / 1024 / 1024

	fmt.Printf("启动时间: %.3fms / %.0fms 目标 ", startupMs, targetStartup)
	if startupMs < targetStartup {
		fmt.Printf("✅ (超越 %.1fx)\n", targetStartup/startupMs)
	} else {
		fmt.Printf("❌ (超出 %.1fx)\n", startupMs/targetStartup)
	}

	fmt.Printf("内存使用: %.2fMB / %.0fMB 目标 ", memoryMB, targetMemory)
	if memoryMB < targetMemory {
		fmt.Printf("✅ (节省 %.1fx)\n", targetMemory/memoryMB)
	} else {
		fmt.Printf("❌ (超出 %.1fx)\n", memoryMB/targetMemory)
	}
}

func saveAnalysisReport(analysis *PerformanceAnalysis) {
	reportFile := "performance_analysis_report.json"

	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		fmt.Printf("❌ 保存报告失败: %v\n", err)
		return
	}

	err = os.WriteFile(reportFile, data, 0644)
	if err != nil {
		fmt.Printf("❌ 写入文件失败: %v\n", err)
		return
	}

	fmt.Printf("\n📄 性能报告已保存至: %s\n", reportFile)
}
