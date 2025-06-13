/*
* @Author: Lzww0608
* @Date: 2025-6-13 23:08:37
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-13 23:08:41
* @Description: ClixGo性能基线测试命令行工具
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Lzww0608/ClixGo/benchmarks"
)

func main() {
	var (
		continuous = flag.Bool("continuous", false, "运行持续性能监控")
		interval   = flag.Duration("interval", 30*time.Second, "持续监控间隔")
		output     = flag.String("output", "", "报告输出文件")
		help       = flag.Bool("help", false, "显示帮助信息")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	fmt.Println("🚀 ClixGo Performance Baseline Tool")
	fmt.Println("=====================================")

	baseline := benchmarks.NewPerformanceBaseline()

	if *continuous {
		fmt.Printf("📊 启动持续监控模式 (间隔: %v)\n", *interval)
		fmt.Println("按 Ctrl+C 停止监控...")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := baseline.RunContinuousMonitoring(ctx, *interval); err != nil {
			fmt.Printf("❌ 持续监控失败: %v\n", err)
			os.Exit(1)
		}
	} else {
		// 运行一次性基线测试
		metrics, err := baseline.RunBaseline()
		if err != nil {
			fmt.Printf("❌ 基线测试失败: %v\n", err)
			os.Exit(1)
		}

		// 显示报告
		report := baseline.GenerateReport()
		fmt.Println(report)

		// 保存报告到文件
		if *output != "" {
			if err := baseline.SaveReport(*output); err != nil {
				fmt.Printf("⚠️  保存报告失败: %v\n", err)
			} else {
				fmt.Printf("📄 报告已保存到: %s\n", *output)
			}
		}

		// 输出JSON格式的指标（便于CI/CD集成）
		fmt.Printf("\n📊 JSON指标:\n")
		fmt.Printf(`{
  "startup_time_ms": %d,
  "memory_usage_mb": %.2f,
  "session_create_ms": %d,
  "session_switch_ms": %d,
  "goroutine_count": %d,
  "performance_goals": {
    "startup_under_30ms": %t,
    "memory_under_8mb": %t,
    "switch_under_5ms": %t
  }
}
`,
			metrics.StartupTimeMs,
			metrics.MemoryUsageMB,
			metrics.SessionCreateMs,
			metrics.SessionSwitchMs,
			metrics.GoroutineCount,
			metrics.StartupTimeMs < 30,
			metrics.MemoryUsageMB < 8.0,
			metrics.SessionSwitchMs < 5,
		)

		// 设置退出码（便于CI/CD判断）
		if metrics.StartupTimeMs >= 30 || metrics.MemoryUsageMB >= 8.0 || metrics.SessionSwitchMs >= 5 {
			fmt.Println("\n⚠️  某些性能目标未达成")
			os.Exit(1)
		} else {
			fmt.Println("\n✅ 所有性能目标均已达成")
		}
	}
}

func showHelp() {
	fmt.Println(`ClixGo Performance Baseline Tool

用法:
  go run cmd/benchmark/main.go [选项]

选项:
  -continuous         运行持续性能监控模式
  -interval duration  持续监控的间隔时间 (默认: 30s)
  -output string      报告输出文件路径
  -help              显示此帮助信息

示例:
  # 运行一次基线测试
  go run cmd/benchmark/main.go

  # 保存报告到文件
  go run cmd/benchmark/main.go -output=baseline-report.txt

  # 持续监控（每10秒一次）
  go run cmd/benchmark/main.go -continuous -interval=10s

性能目标:
  启动时间: < 30ms
  内存使用: < 8MB
  切换时间: < 5ms
`)
}
