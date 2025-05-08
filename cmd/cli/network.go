package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/network"
	"github.com/spf13/cobra"
)

func NewNetworkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "网络工具",
		Long:  `提供各种网络诊断和测试功能`,
	}

	// Ping命令
	pingCmd := &cobra.Command{
		Use:   "ping",
		Short: "测试网络连接",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			count, _ := cmd.Flags().GetInt("count")
			timeout, _ := cmd.Flags().GetDuration("timeout")
			result, err := network.Ping(args[0], count, timeout)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
	pingCmd.Flags().Int("count", 4, "发送的ping包数量")
	pingCmd.Flags().DurationP("timeout", "t", 5*time.Second, "超时时间")
	cmd.AddCommand(pingCmd)

	// Traceroute命令
	tracerouteCmd := &cobra.Command{
		Use:   "traceroute",
		Short: "跟踪网络路径",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			maxHops, _ := cmd.Flags().GetInt("max-hops")
			result, err := network.Traceroute(args[0], maxHops)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
	tracerouteCmd.Flags().IntP("max-hops", "m", 30, "最大跳数")
	cmd.AddCommand(tracerouteCmd)

	// DNS查询命令
	dnsCmd := &cobra.Command{
		Use:   "dns",
		Short: "DNS查询",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ips, err := network.DNSLookup(args[0])
			if err != nil {
				return err
			}
			for _, ip := range ips {
				fmt.Println(ip)
			}
			return nil
		},
	}
	cmd.AddCommand(dnsCmd)

	// HTTP请求命令
	httpCmd := &cobra.Command{
		Use:   "http",
		Short: "HTTP请求",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			timeout, _ := cmd.Flags().GetDuration("timeout")
			response, err := network.HTTPGet(args[0], timeout)
			if err != nil {
				return err
			}
			fmt.Println(response)
			return nil
		},
	}
	httpCmd.Flags().DurationP("timeout", "t", 10*time.Second, "超时时间")
	cmd.AddCommand(httpCmd)

	// 端口检查命令
	portCmd := &cobra.Command{
		Use:   "port",
		Short: "检查端口",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("无效的端口号: %v", err)
			}
			timeout, _ := cmd.Flags().GetDuration("timeout")
			open, err := network.CheckPort(args[0], port, timeout)
			if err != nil {
				return err
			}
			if open {
				fmt.Printf("端口 %d 是开放的\n", port)
			} else {
				fmt.Printf("端口 %d 是关闭的\n", port)
			}
			return nil
		},
	}
	portCmd.Flags().DurationP("timeout", "t", 5*time.Second, "超时时间")
	cmd.AddCommand(portCmd)

	// IP信息查询命令
	ipinfoCmd := &cobra.Command{
		Use:   "ipinfo",
		Short: "查询IP信息",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := network.GetIPInfo(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("IP: %s\n国家: %s\n地区: %s\n城市: %s\nISP: %s\n",
				info.IP, info.Country, info.Region, info.City, info.ISP)
			return nil
		},
	}
	cmd.AddCommand(ipinfoCmd)

	// 文件下载命令
	downloadCmd := &cobra.Command{
		Use:   "download",
		Short: "下载文件",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			timeout, _ := cmd.Flags().GetDuration("timeout")
			err := network.DownloadFile(args[0], args[1], timeout)
			if err != nil {
				return err
			}
			fmt.Printf("文件已下载到: %s\n", args[1])
			return nil
		},
	}
	downloadCmd.Flags().DurationP("timeout", "t", 30*time.Second, "超时时间")
	cmd.AddCommand(downloadCmd)

	// SSL证书检查命令
	sslCmd := &cobra.Command{
		Use:   "ssl",
		Short: "检查SSL证书",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := network.CheckSSL(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("证书颁发者: %s\n有效期至: %s\n",
				info.Issuer, info.Expiry.Format("2006-01-02"))
			return nil
		},
	}
	cmd.AddCommand(sslCmd)

	// 网络速度测试命令
	speedtestCmd := &cobra.Command{
		Use:   "speedtest",
		Short: "网络速度测试",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := network.NetworkSpeedTest()
			if err != nil {
				return err
			}
			fmt.Printf("下载速度: %.2f Mbps\n上传速度: %.2f Mbps\n",
				result.Download, result.Upload)
			return nil
		},
	}
	cmd.AddCommand(speedtestCmd)

	// 网络监控命令
	monitorCmd := &cobra.Command{
		Use:   "monitor",
		Short: "网络监控",
		RunE: func(cmd *cobra.Command, args []string) error {
			interval, _ := cmd.Flags().GetDuration("interval")
			timeout, _ := cmd.Flags().GetDuration("timeout")
			threshold, _ := cmd.Flags().GetFloat64("threshold")
			email, _ := cmd.Flags().GetString("email")
			webhook, _ := cmd.Flags().GetString("webhook")

			config := network.NetworkMonitor{
				Targets:  args,
				Interval: interval,
				Timeout:  timeout,
				AlertConfig: network.AlertConfig{
					Enabled:     email != "" || webhook != "",
					Threshold:   threshold,
					Email:       email,
					Webhook:     webhook,
					RepeatAfter: time.Hour,
				},
			}

			results, cancel := network.StartMonitoring(config)
			defer cancel()

			for result := range results {
				if result.Error != nil {
					fmt.Printf("[%s] %s: 错误 - %v\n", result.Timestamp.Format("2006-01-02 15:04:05"), result.Target, result.Error)
				} else {
					fmt.Printf("[%s] %s: 状态=%s, 延迟=%.2fms, 丢包率=%.2f%%\n",
						result.Timestamp.Format("2006-01-02 15:04:05"),
						result.Target,
						result.Status,
						result.Latency.Seconds()*1000,
						result.PacketLoss)
				}
			}

			return nil
		},
	}
	monitorCmd.Flags().DurationP("interval", "i", 5*time.Second, "监控间隔")
	monitorCmd.Flags().DurationP("timeout", "t", 2*time.Second, "超时时间")
	monitorCmd.Flags().Float64P("threshold", "T", 50.0, "告警阈值(ms)")
	monitorCmd.Flags().StringP("email", "e", "", "告警邮箱")
	monitorCmd.Flags().StringP("webhook", "w", "", "告警Webhook")
	cmd.AddCommand(monitorCmd)

	// 网络配置命令
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "网络配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			iface, _ := cmd.Flags().GetString("interface")
			config, err := network.GetNetworkConfig(iface)
			if err != nil {
				return err
			}

			fmt.Printf("网络接口: %s\n", config.Interface)
			fmt.Printf("IP地址: %s\n", config.IP)
			fmt.Printf("子网掩码: %s\n", config.Netmask)
			fmt.Printf("默认网关: %s\n", config.Gateway)
			fmt.Printf("DNS服务器: %v\n", config.DNS)
			fmt.Printf("MTU: %d\n", config.MTU)

			return nil
		},
	}
	configCmd.Flags().StringP("interface", "i", "", "网络接口名称")
	cmd.AddCommand(configCmd)

	// 带宽测试命令
	bandwidthCmd := &cobra.Command{
		Use:   "bandwidth",
		Short: "带宽测试",
		RunE: func(cmd *cobra.Command, args []string) error {
			server, _ := cmd.Flags().GetString("server")
			result, err := network.TestBandwidth(server)
			if err != nil {
				return err
			}

			fmt.Printf("下载速度: %.2f Mbps\n", result.DownloadSpeed)
			fmt.Printf("上传速度: %.2f Mbps\n", result.UploadSpeed)
			fmt.Printf("抖动: %.2f ms\n", result.Jitter)
			fmt.Printf("延迟: %.2f ms\n", result.Latency)

			return nil
		},
	}
	bandwidthCmd.Flags().StringP("server", "s", "", "测试服务器地址")
	cmd.AddCommand(bandwidthCmd)

	// 数据包捕获命令
	captureCmd := &cobra.Command{
		Use:   "capture",
		Short: "数据包捕获",
		RunE: func(cmd *cobra.Command, args []string) error {
			iface, _ := cmd.Flags().GetString("interface")
			filter, _ := cmd.Flags().GetString("filter")
			count, _ := cmd.Flags().GetInt("count")
			timeout, _ := cmd.Flags().GetDuration("timeout")

			config := network.PacketCapture{
				Interface: iface,
				Filter:    filter,
				Count:     count,
				Timeout:   timeout,
			}

			packets, cancel := network.StartPacketCapture(config)
			defer cancel()

			for packet := range packets {
				fmt.Printf("捕获到数据包: %v\n", packet)
			}

			return nil
		},
	}
	captureCmd.Flags().StringP("interface", "i", "", "网络接口名称")
	captureCmd.Flags().StringP("filter", "f", "", "过滤表达式")
	captureCmd.Flags().IntP("count", "c", 0, "捕获数量(0表示无限)")
	captureCmd.Flags().DurationP("timeout", "t", 0, "超时时间(0表示无限)")
	cmd.AddCommand(captureCmd)

	// 网络诊断命令
	diagnoseCmd := &cobra.Command{
		Use:   "diagnose",
		Short: "网络诊断",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := network.RunDiagnostic()
			if err != nil {
				return err
			}

			fmt.Println("网络诊断结果:")
			fmt.Printf("本地连接: %v\n", result.Connectivity)
			fmt.Printf("DNS解析: %v\n", result.DNS)
			fmt.Printf("网关连接: %v\n", result.Gateway)
			fmt.Printf("互联网连接: %v\n", result.Internet)

			if len(result.Issues) > 0 {
				fmt.Println("\n发现的问题:")
				for _, issue := range result.Issues {
					fmt.Printf("- %s\n", issue)
				}
			}

			return nil
		},
	}
	cmd.AddCommand(diagnoseCmd)

	// 协议测试命令
	protocolCmd := &cobra.Command{
		Use:   "protocol",
		Short: "协议测试",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("需要提供目标地址和协议")
			}

			result, err := network.TestProtocol(args[0], args[1])
			if err != nil {
				return err
			}

			fmt.Printf("协议: %s\n", result.Protocol)
			fmt.Printf("状态: %s\n", result.Status)
			if result.Error != nil {
				fmt.Printf("错误: %v\n", result.Error)
			} else {
				fmt.Printf("延迟: %.2f ms\n", result.Latency.Seconds()*1000)
			}

			return nil
		},
	}
	cmd.AddCommand(protocolCmd)

	// 性能分析命令
	performanceCmd := &cobra.Command{
		Use:   "performance",
		Short: "性能分析",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("需要提供目标地址")
			}

			duration, _ := cmd.Flags().GetDuration("duration")
			result, err := network.AnalyzePerformance(args[0], duration)
			if err != nil {
				return err
			}

			fmt.Println("网络性能分析结果:")
			fmt.Printf("带宽: %.2f Mbps\n", result.Bandwidth)
			fmt.Printf("延迟: %.2f ms\n", result.Latency)
			fmt.Printf("抖动: %.2f ms\n", result.Jitter)
			fmt.Printf("丢包率: %.2f%%\n", result.PacketLoss)
			fmt.Printf("吞吐量: %.2f Mbps\n", result.Throughput)
			fmt.Printf("重传率: %.2f%%\n", result.Retransmission)
			fmt.Printf("连接时间: %.2f ms\n", result.ConnectionTime)

			return nil
		},
	}
	performanceCmd.Flags().DurationP("duration", "d", 10*time.Second, "测试持续时间")
	cmd.AddCommand(performanceCmd)

	// 流量统计命令
	trafficCmd := &cobra.Command{
		Use:   "traffic",
		Short: "流量统计",
		RunE: func(cmd *cobra.Command, args []string) error {
			iface, _ := cmd.Flags().GetString("interface")
			interval, _ := cmd.Flags().GetDuration("interval")

			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					stats, err := network.GetTrafficStats(iface)
					if err != nil {
						fmt.Printf("错误: %v\n", err)
						continue
					}
					fmt.Printf("\n[%s] 接口 %s 流量统计:\n",
						stats.Timestamp.Format("2006-01-02 15:04:05"),
						stats.Interface)
					fmt.Printf("接收: %d 字节, %d 数据包\n", stats.BytesIn, stats.PacketsIn)
					fmt.Printf("发送: %d 字节, %d 数据包\n", stats.BytesOut, stats.PacketsOut)
					fmt.Printf("错误: 接收 %d, 发送 %d\n", stats.ErrorsIn, stats.ErrorsOut)
					fmt.Printf("丢包: 接收 %d, 发送 %d\n", stats.DropsIn, stats.DropsOut)
				case <-cmd.Context().Done():
					return nil
				}
			}
		},
	}
	trafficCmd.Flags().StringP("interface", "i", "", "网络接口名称")
	trafficCmd.Flags().DurationP("interval", "I", 1*time.Second, "统计间隔")
	cmd.AddCommand(trafficCmd)

	// 网络优化命令
	optimizeCmd := &cobra.Command{
		Use:   "optimize",
		Short: "网络优化",
		RunE: func(cmd *cobra.Command, args []string) error {
			iface, _ := cmd.Flags().GetString("interface")
			auto, _ := cmd.Flags().GetBool("auto")

			result, err := network.OptimizeNetwork(iface)
			if err != nil {
				return err
			}

			fmt.Println("网络优化分析:")
			fmt.Printf("当前MTU: %d\n", result.CurrentMTU)
			fmt.Printf("推荐MTU: %d\n", result.RecommendedMTU)
			fmt.Printf("当前DNS: %v\n", result.CurrentDNS)
			fmt.Printf("推荐DNS: %v\n", result.RecommendedDNS)
			fmt.Printf("当前缓冲区: %d\n", result.CurrentBuffer)
			fmt.Printf("推荐缓冲区: %d\n", result.RecommendedBuffer)

			if len(result.Issues) > 0 {
				fmt.Println("\n发现的问题:")
				for _, issue := range result.Issues {
					fmt.Printf("- %s\n", issue)
				}
			}

			if len(result.Suggestions) > 0 {
				fmt.Println("\n优化建议:")
				for _, suggestion := range result.Suggestions {
					fmt.Printf("- %s\n", suggestion)
				}
			}

			if auto {
				// 自动应用优化建议
				fmt.Println("\n正在应用优化建议...")
				// 这里需要实现自动优化功能
			}

			return nil
		},
	}
	optimizeCmd.Flags().StringP("interface", "i", "", "网络接口名称")
	optimizeCmd.Flags().BoolP("auto", "a", false, "自动应用优化建议")
	cmd.AddCommand(optimizeCmd)

	// 告警配置命令
	alertCmd := &cobra.Command{
		Use:   "alert",
		Short: "告警配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			email, _ := cmd.Flags().GetString("email")
			webhook, _ := cmd.Flags().GetString("webhook")
			sms, _ := cmd.Flags().GetString("sms")
			slack, _ := cmd.Flags().GetString("slack")
			threshold, _ := cmd.Flags().GetFloat64("threshold")
			repeat, _ := cmd.Flags().GetDuration("repeat")

			config := network.AlertConfig{
				Enabled:      true,
				Threshold:    threshold,
				Email:        email,
				Webhook:      webhook,
				SMS:          sms,
				SlackWebhook: slack,
				RepeatAfter:  repeat,
			}

			manager := network.NewAlertManager(config)
			// 测试告警
			if err := manager.SendAlert("测试告警消息"); err != nil {
				return err
			}

			fmt.Println("告警配置已更新并测试")
			return nil
		},
	}
	alertCmd.Flags().StringP("email", "e", "", "告警邮箱")
	alertCmd.Flags().StringP("webhook", "w", "", "告警Webhook")
	alertCmd.Flags().StringP("sms", "s", "", "告警手机号")
	alertCmd.Flags().StringP("slack", "S", "", "Slack Webhook")
	alertCmd.Flags().Float64P("threshold", "t", 50.0, "告警阈值(ms)")
	alertCmd.Flags().DurationP("repeat", "r", 1*time.Hour, "重复告警间隔")
	cmd.AddCommand(alertCmd)

	// 流量分析命令
	analyzeCmd := &cobra.Command{
		Use:   "analyze",
		Short: "流量分析",
		RunE: func(cmd *cobra.Command, args []string) error {
			iface, _ := cmd.Flags().GetString("interface")
			duration, _ := cmd.Flags().GetDuration("duration")

			result, err := network.AnalyzeTraffic(iface, duration)
			if err != nil {
				return err
			}

			fmt.Println("流量分析结果:")
			fmt.Printf("接口: %s\n", result.Interface)
			fmt.Printf("时间: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))

			fmt.Println("\n协议分布:")
			for proto, stats := range result.ProtocolStats {
				fmt.Printf("%s: %d 数据包, %d 字节 (%.2f%%)\n",
					proto, stats.Packets, stats.Bytes, stats.Percentage)
			}

			fmt.Println("\n连接状态:")
			fmt.Printf("总连接数: %d\n", result.ConnectionStats.TotalConnections)
			fmt.Printf("活动连接: %d\n", result.ConnectionStats.ActiveConnections)
			fmt.Printf("TCP连接: %d\n", result.ConnectionStats.TCPConnections)
			fmt.Printf("UDP连接: %d\n", result.ConnectionStats.UDPConnections)
			fmt.Printf("已建立: %d\n", result.ConnectionStats.Established)
			fmt.Printf("等待关闭: %d\n", result.ConnectionStats.TimeWait)
			fmt.Printf("关闭等待: %d\n", result.ConnectionStats.CloseWait)

			fmt.Println("\n带宽使用:")
			fmt.Printf("当前入站: %.2f Mbps\n", result.BandwidthUsage.CurrentIn)
			fmt.Printf("当前出站: %.2f Mbps\n", result.BandwidthUsage.CurrentOut)
			fmt.Printf("峰值入站: %.2f Mbps\n", result.BandwidthUsage.PeakIn)
			fmt.Printf("峰值出站: %.2f Mbps\n", result.BandwidthUsage.PeakOut)
			fmt.Printf("平均入站: %.2f Mbps\n", result.BandwidthUsage.AverageIn)
			fmt.Printf("平均出站: %.2f Mbps\n", result.BandwidthUsage.AverageOut)
			fmt.Printf("利用率: %.2f%%\n", result.BandwidthUsage.Utilization)

			return nil
		},
	}
	analyzeCmd.Flags().StringP("interface", "i", "", "网络接口名称")
	analyzeCmd.Flags().DurationP("duration", "d", 1*time.Minute, "分析持续时间")
	cmd.AddCommand(analyzeCmd)

	// 网络质量评估命令
	qualityCmd := &cobra.Command{
		Use:   "quality",
		Short: "网络质量评估",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			duration, _ := cmd.Flags().GetDuration("duration")

			result, err := network.EvaluateQuality(args[0], duration)
			if err != nil {
				return err
			}

			fmt.Println("网络质量评估结果:")
			fmt.Printf("总分: %.2f/100\n", result.Score)
			fmt.Printf("延迟评分: %.2f/100\n", result.LatencyScore)
			fmt.Printf("稳定性评分: %.2f/100\n", result.StabilityScore)
			fmt.Printf("速度评分: %.2f/100\n", result.SpeedScore)
			fmt.Printf("可靠性评分: %.2f/100\n", result.ReliabilityScore)

			if len(result.Issues) > 0 {
				fmt.Println("\n发现的问题:")
				for _, issue := range result.Issues {
					fmt.Printf("- %s\n", issue)
				}
			}

			if len(result.Recommendations) > 0 {
				fmt.Println("\n优化建议:")
				for _, rec := range result.Recommendations {
					fmt.Printf("- %s\n", rec)
				}
			}

			return nil
		},
	}
	qualityCmd.Flags().DurationP("duration", "d", 1*time.Minute, "评估持续时间")
	cmd.AddCommand(qualityCmd)

	// 网络配置备份命令
	backupCmd := &cobra.Command{
		Use:   "backup",
		Short: "网络配置备份",
		RunE: func(cmd *cobra.Command, args []string) error {
			iface, _ := cmd.Flags().GetString("interface")

			backup, err := network.BackupNetworkConfig(iface)
			if err != nil {
				return err
			}

			fmt.Println("网络配置已备份:")
			fmt.Printf("备份ID: %s\n", backup.BackupID)
			fmt.Printf("接口: %s\n", backup.Interface)
			fmt.Printf("时间: %s\n", backup.Timestamp.Format("2006-01-02 15:04:05"))

			return nil
		},
	}
	backupCmd.Flags().StringP("interface", "i", "", "网络接口名称")
	cmd.AddCommand(backupCmd)

	// 网络配置恢复命令
	restoreCmd := &cobra.Command{
		Use:   "restore",
		Short: "恢复网络配置",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := network.RestoreNetworkConfig(args[0]); err != nil {
				return err
			}

			fmt.Printf("已从备份 %s 恢复网络配置\n", args[0])
			return nil
		},
	}
	cmd.AddCommand(restoreCmd)

	return cmd
}
