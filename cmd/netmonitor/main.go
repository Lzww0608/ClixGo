/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 网络监控工具主程序
 */

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/network"
)

func main() {
	// 命令行参数
	var (
		updateInterval = flag.Duration("interval", 2*time.Second, "监控更新间隔")
		timeout        = flag.Duration("timeout", 5*time.Second, "操作超时时间")
		maxHistory     = flag.Int("history", 100, "最大历史记录数")
		enableAlerts   = flag.Bool("alerts", true, "启用告警")
		targets        = flag.String("targets", "8.8.8.8,1.1.1.1", "监控目标，逗号分隔")
		interfaces     = flag.String("interfaces", "", "监控的网络接口，逗号分隔（空表示所有）")
		uiMode         = flag.Bool("ui", true, "启用图形界面模式")

		// 告警阈值
		latencyThreshold    = flag.Float64("latency-threshold", 100.0, "延迟告警阈值(毫秒)")
		packetLossThreshold = flag.Float64("packetloss-threshold", 5.0, "丢包率告警阈值(百分比)")
		bandwidthThreshold  = flag.Float64("bandwidth-threshold", 100.0, "带宽使用告警阈值(Mbps)")
		connectionThreshold = flag.Int("connection-threshold", 1000, "连接数告警阈值")
		errorRateThreshold  = flag.Float64("error-threshold", 1.0, "错误率告警阈值(百分比)")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "ClixGo 实时网络资源监控工具\n\n")
		fmt.Fprintf(os.Stderr, "用法: %s [选项]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  %s -targets=8.8.8.8,1.1.1.1 -interval=1s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -ui=false -targets=google.com\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -interfaces=eth0,wlan0 -alerts=true\n", os.Args[0])
	}

	flag.Parse()

	// 解析目标列表
	var targetList []string
	if *targets != "" {
		targetList = strings.Split(*targets, ",")
		for i, target := range targetList {
			targetList[i] = strings.TrimSpace(target)
		}
	}

	// 解析接口列表
	var interfaceList []string
	if *interfaces != "" {
		interfaceList = strings.Split(*interfaces, ",")
		for i, iface := range interfaceList {
			interfaceList[i] = strings.TrimSpace(iface)
		}
	}

	// 创建监控配置
	config := network.RealtimeMonitorConfig{
		UpdateInterval:   *updateInterval,
		Timeout:          *timeout,
		MaxHistory:       *maxHistory,
		EnableAlerts:     *enableAlerts,
		MonitoredTargets: targetList,
		Interfaces:       interfaceList,
		AlertThresholds: network.AlertThresholds{
			LatencyMs:         *latencyThreshold,
			PacketLossPercent: *packetLossThreshold,
			BandwidthMbps:     *bandwidthThreshold,
			ConnectionCount:   *connectionThreshold,
			ErrorRate:         *errorRateThreshold,
		},
	}

	// 创建监控器
	monitor := network.NewRealtimeNetworkMonitor(config)

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("启动ClixGo实时网络监控器...\n")
	fmt.Printf("监控目标: %v\n", targetList)
	fmt.Printf("监控接口: %v\n", interfaceList)
	fmt.Printf("更新间隔: %v\n", *updateInterval)
	fmt.Printf("超时时间: %v\n", *timeout)
	fmt.Printf("告警启用: %v\n", *enableAlerts)
	fmt.Printf("界面模式: %v\n", *uiMode)
	fmt.Println("按 Ctrl+C 退出")
	fmt.Println()

	if *uiMode {
		// 启动图形界面模式
		runUIMode(monitor, sigChan)
	} else {
		// 启动命令行模式
		runConsoleMode(monitor, sigChan)
	}
}

// runUIMode 运行图形界面模式
func runUIMode(monitor *network.RealtimeNetworkMonitor, sigChan chan os.Signal) {
	// 创建UI
	ui := network.NewRealtimeNetworkUI(monitor)

	// 在单独的协程中处理信号
	go func() {
		<-sigChan
		fmt.Println("\n收到退出信号，正在关闭...")
		ui.Stop()
	}()

	// 启动UI（这会阻塞直到UI退出）
	if err := ui.Start(); err != nil {
		log.Fatalf("启动UI失败: %v", err)
	}

	fmt.Println("监控器已停止")
}

// runConsoleMode 运行命令行模式
func runConsoleMode(monitor *network.RealtimeNetworkMonitor, sigChan chan os.Signal) {
	// 启动监控器
	if err := monitor.Start(); err != nil {
		log.Fatalf("启动监控器失败: %v", err)
	}
	defer monitor.Stop()

	// 启动数据显示协程
	go func() {
		for {
			select {
			case snapshot := <-monitor.GetUpdateChannel():
				displaySnapshot(snapshot)
			case err := <-monitor.GetErrorChannel():
				fmt.Printf("错误: %v\n", err)
			}
		}
	}()

	// 等待退出信号
	<-sigChan
	fmt.Println("\n收到退出信号，正在关闭...")

	// 显示最终统计
	stats := monitor.GetStatistics()
	fmt.Printf("\n最终统计信息:\n")
	for key, value := range stats {
		fmt.Printf("  %s: %v\n", key, value)
	}

	fmt.Println("监控器已停止")
}

// displaySnapshot 显示快照信息（命令行模式）
func displaySnapshot(snapshot network.NetworkResourceSnapshot) {
	fmt.Printf("\n=== 网络监控快照 [%s] ===\n", snapshot.Timestamp.Format("15:04:05"))

	// 显示性能评分
	fmt.Printf("性能评分: %.1f/100\n", snapshot.PerformanceScore)

	// 显示接口信息
	if len(snapshot.Interfaces) > 0 {
		fmt.Printf("\n网络接口:\n")
		for name, iface := range snapshot.Interfaces {
			status := "DOWN"
			if iface.IsUp {
				status = "UP"
			}
			fmt.Printf("  %s: %s, MTU=%d, 带宽=%.2f/%.2f Mbps, 使用率=%.1f%%\n",
				name, status, iface.MTU,
				iface.BandwidthInMbps, iface.BandwidthOutMbps, iface.Utilization)
		}
	}

	// 显示连接信息
	fmt.Printf("\n连接统计:\n")
	fmt.Printf("  总连接: %d (TCP: %d, UDP: %d)\n",
		snapshot.Connections.Total, snapshot.Connections.TCP, snapshot.Connections.UDP)
	fmt.Printf("  TCP状态: 已建立=%d, 监听=%d, 等待=%d, 关闭等待=%d\n",
		snapshot.Connections.Established, snapshot.Connections.Listen,
		snapshot.Connections.TimeWait, snapshot.Connections.CloseWait)

	// 显示延迟信息
	if len(snapshot.TargetLatencies) > 0 {
		fmt.Printf("\n目标延迟:\n")
		for target, latency := range snapshot.TargetLatencies {
			status := "不可达"
			if latency.IsReachable {
				status = "可达"
			}
			avgMs := float64(latency.AvgLatency.Nanoseconds()) / 1e6
			fmt.Printf("  %s: %s, 延迟=%.2fms, 丢包=%.2f%%\n",
				target, status, avgMs, latency.PacketLoss)
		}
	}

	// 显示告警
	if len(snapshot.Alerts) > 0 {
		fmt.Printf("\n告警 (%d):\n", len(snapshot.Alerts))
		for _, alert := range snapshot.Alerts {
			fmt.Printf("  [%s] %s: %s\n",
				strings.ToUpper(alert.Severity), alert.Type, alert.Message)
		}
	}

	// 显示系统资源
	fmt.Printf("\n系统资源:\n")
	fmt.Printf("  打开文件: %d/%d\n",
		snapshot.SystemResources.OpenFiles, snapshot.SystemResources.MaxOpenFiles)
	fmt.Printf("  网络线程: %d\n", snapshot.SystemResources.NetworkThreads)
	fmt.Printf("  内存使用: %.2f MB\n", snapshot.SystemResources.MemoryUsageMB)

	fmt.Printf("\n" + strings.Repeat("-", 60))
}
