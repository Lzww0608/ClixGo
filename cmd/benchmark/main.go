/*
* @Author: Lzww0608
* @Date: 2025-6-13 23:08:37
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-14 11:03:17
* @Description: ClixGo性能基线测试命令行工具
 */

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Lzww0608/ClixGo/benchmarks"
	"github.com/Lzww0608/ClixGo/pkg/logger"
)

func main() {
	// 命令行参数
	var (
		mode     = flag.String("mode", "single", "运行模式: single(单次测试) 或 monitor(持续监控)")
		interval = flag.Duration("interval", 30*time.Second, "监控间隔时间")
		output   = flag.String("output", "", "输出文件路径")
		format   = flag.String("format", "text", "输出格式: text 或 json")
		verbose  = flag.Bool("verbose", false, "详细输出")
		help     = flag.Bool("help", false, "显示帮助信息")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// 初始化日志系统
	logger.InitLogger()
	defer logger.Close()

	fmt.Println("🚀 ClixGo Performance Baseline Tool")
	fmt.Printf("模式: %s\n", *mode)

	baseline := benchmarks.NewPerformanceBaseline()

	switch *mode {
	case "single":
		runSingleTest(baseline, *output, *format, *verbose)
	case "monitor":
		runContinuousMonitoring(baseline, *interval, *verbose)
	default:
		fmt.Printf("❌ 未知的运行模式: %s\n", *mode)
		fmt.Println("支持的模式: single, monitor")
		os.Exit(1)
	}
}

// runSingleTest 运行单次性能测试
func runSingleTest(baseline *benchmarks.PerformanceBaseline, output, format string, verbose bool) {
	fmt.Println("📊 执行单次性能基线测试...")

	metrics, err := baseline.RunBaseline()
	if err != nil {
		fmt.Printf("❌ 性能测试失败: %v\n", err)
		os.Exit(1)
	}

	// 根据格式输出结果
	switch format {
	case "json":
		outputJSON(metrics, output, verbose)
	default:
		outputText(baseline, output, verbose)
	}

	// 检查性能目标达成情况
	checkPerformanceGoals(metrics)
}

// runContinuousMonitoring 运行持续监控
func runContinuousMonitoring(baseline *benchmarks.PerformanceBaseline, interval time.Duration, _ bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\n📈 接收到停止信号，正在优雅退出...")
		cancel()
	}()

	// 运行持续监控
	if err := baseline.RunContinuousMonitoring(ctx, interval); err != nil {
		if err != context.Canceled {
			fmt.Printf("❌ 持续监控失败: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("✅ 持续监控已停止")
}

// outputJSON 输出JSON格式结果
func outputJSON(metrics *benchmarks.PerformanceMetrics, output string, verbose bool) {
	data, err := json.MarshalIndent(map[string]any{
		"timestamp":           time.Now().Unix(),
		"performance_metrics": metrics,
		"performance_goals": map[string]any{
			"startup_time_ms_target":    30,
			"memory_usage_mb_target":    8.0,
			"startup_speedup_target":    5.0,
			"memory_reduction_target":   70.0,
			"startup_time_achieved":     metrics.StartupTimeMs < 30,
			"memory_usage_achieved":     metrics.MemoryUsageMB < 8.0,
			"startup_speedup_achieved":  metrics.StartupSpeedup >= 5.0,
			"memory_reduction_achieved": metrics.MemoryReduction >= 70.0,
		},
	}, "", "  ")
	if err != nil {
		fmt.Printf("❌ JSON序列化失败: %v\n", err)
		return
	}

	if output != "" {
		err := os.WriteFile(output, data, 0644)
		if err != nil {
			fmt.Printf("❌ 写入文件失败: %v\n", err)
			return
		}
		fmt.Printf("📄 JSON报告已保存到: %s\n", output)
	}

	if verbose || output == "" {
		fmt.Printf("📊 JSON格式结果:\n%s\n", string(data))
	}
}

// outputText 输出文本格式结果
func outputText(baseline *benchmarks.PerformanceBaseline, output string, verbose bool) {
	report := baseline.GenerateReport()

	if output != "" {
		err := os.WriteFile(output, []byte(report), 0644)
		if err != nil {
			fmt.Printf("❌ 写入文件失败: %v\n", err)
			return
		}
		fmt.Printf("📄 文本报告已保存到: %s\n", output)
	}

	if verbose || output == "" {
		fmt.Println(report)
	}
}

// checkPerformanceGoals 检查性能目标达成情况
func checkPerformanceGoals(metrics *benchmarks.PerformanceMetrics) {
	fmt.Println("\n🎯 性能目标检查:")

	goals := []struct {
		name     string
		achieved bool
		actual   any
		target   any
		unit     string
	}{
		{"启动时间", metrics.StartupTimeMs < 30, metrics.StartupTimeMs, "< 30", "ms"},
		{"内存占用", metrics.MemoryUsageMB < 8.0, fmt.Sprintf("%.2f", metrics.MemoryUsageMB), "< 8.0", "MB"},
		{"启动加速", metrics.StartupSpeedup >= 5.0, fmt.Sprintf("%.1f", metrics.StartupSpeedup), ">= 5.0", "x"},
		{"内存减少", metrics.MemoryReduction >= 70.0, fmt.Sprintf("%.1f", metrics.MemoryReduction), ">= 70.0", "%"},
	}

	achievedCount := 0
	for _, goal := range goals {
		status := "❌"
		if goal.achieved {
			status = "✅"
			achievedCount++
		}
		fmt.Printf("  %s %s: %v%s (目标: %v%s)\n",
			status, goal.name, goal.actual, goal.unit, goal.target, goal.unit)
	}

	fmt.Printf("\n📈 总体达成度: %d/%d (%.1f%%)\n",
		achievedCount, len(goals), float64(achievedCount)/float64(len(goals))*100)

	// 如果所有目标都达成，设置成功退出码
	if achievedCount == len(goals) {
		fmt.Println("🎉 所有性能目标均已达成！")
	} else {
		fmt.Printf("⚠️  还有 %d 个性能目标未达成，需要进一步优化\n", len(goals)-achievedCount)
	}
}

// showHelp 显示帮助信息
func showHelp() {
	fmt.Println(`ClixGo Performance Baseline Tool

用法:
  benchmark [选项]

选项:
  -mode string
        运行模式: single(单次测试) 或 monitor(持续监控) (默认 "single")
  -interval duration
        监控模式下的间隔时间 (默认 30s)
  -output string
        输出文件路径 (可选)
  -format string
        输出格式: text 或 json (默认 "text")
  -verbose
        启用详细输出
  -help
        显示此帮助信息

示例:
  # 运行单次测试
  benchmark

  # 运行单次测试并保存报告
  benchmark -output performance_report.txt

  # 运行持续监控，每60秒检查一次
  benchmark -mode monitor -interval 60s

  # 输出JSON格式结果
  benchmark -format json -output metrics.json

性能目标:
  - 启动时间 < 30ms
  - 内存占用 < 8MB
  - 相比tmux启动速度提升 >= 5x
  - 相比tmux内存减少 >= 70%`)
}
