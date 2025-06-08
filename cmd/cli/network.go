/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-8 18:07:01
* @Description: 网络监控和管理的CLI命令定义
 */

package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/network"
	"github.com/spf13/cobra"
)

// NewNetworkCmd 创建网络工具命令组
//
// 该函数创建一个包含多个网络诊断和测试子命令的命令组，包括：
// - ping: 网络连通性测试
// - traceroute: 网络路径跟踪
// - dns: DNS查询
// - http: HTTP请求测试
// - port: 端口连通性检查
// - ipinfo: IP地址信息查询
// - download: 文件下载
// - ssl: SSL证书检查
// - speedtest: 网络速度测试
// - monitor: 实时网络监控
//
// 返回:
//   - *cobra.Command: 配置完整的网络工具命令组
func NewNetworkCmd() *cobra.Command {
	networkCmd := &cobra.Command{
		Use:   "network",
		Short: "网络工具",
		Long:  `提供各种网络诊断和测试功能，支持连通性检查、性能测试、证书验证等`,
	}

	// 添加所有网络相关子命令
	networkCmd.AddCommand(createPingCommand())
	networkCmd.AddCommand(createTracerouteCommand())
	networkCmd.AddCommand(createDNSCommand())
	networkCmd.AddCommand(createHTTPCommand())
	networkCmd.AddCommand(createPortCheckCommand())
	networkCmd.AddCommand(createIPInfoCommand())
	networkCmd.AddCommand(createDownloadCommand())
	networkCmd.AddCommand(createSSLCheckCommand())
	networkCmd.AddCommand(createSpeedTestCommand())
	networkCmd.AddCommand(createNetworkMonitorCommand())

	return networkCmd
}

// createPingCommand 创建ping命令
//
// ping命令用于测试网络连通性，支持自定义包数量和超时时间
//
// 用法: network ping <目标地址> [--count 包数量] [--timeout 超时时间]
//
// 返回:
//   - *cobra.Command: 配置完整的ping命令
func createPingCommand() *cobra.Command {
	pingCmd := &cobra.Command{
		Use:   "ping <目标地址>",
		Short: "测试网络连接",
		Long: `向指定目标发送ICMP包测试网络连通性
		
示例:
  clixgo network ping google.com
  clixgo network ping 8.8.8.8 --count 10 --timeout 3s`,
		Args: cobra.ExactArgs(1),
		RunE: executePingCommand,
	}

	// 配置命令行标志
	pingCmd.Flags().Int("count", 4, "发送的ping包数量")
	pingCmd.Flags().DurationP("timeout", "t", 5*time.Second, "每个包的超时时间")

	return pingCmd
}

// executePingCommand 执行ping命令的业务逻辑
//
// 参数:
//   - cmd: cobra命令实例
//   - args: 命令行参数，args[0]为目标地址
//
// 返回:
//   - error: 执行过程中的错误，nil表示成功
func executePingCommand(cmd *cobra.Command, args []string) error {
	targetAddress := args[0]
	packetCount, _ := cmd.Flags().GetInt("count")
	timeoutDuration, _ := cmd.Flags().GetDuration("timeout")

	pingResult, err := network.Ping(targetAddress, packetCount, timeoutDuration)
	if err != nil {
		return fmt.Errorf("ping操作失败: %w", err)
	}

	fmt.Println(pingResult)
	return nil
}

// createTracerouteCommand 创建traceroute命令
//
// traceroute命令用于跟踪数据包到达目标地址的网络路径
//
// 用法: network traceroute <目标地址> [--max-hops 最大跳数]
//
// 返回:
//   - *cobra.Command: 配置完整的traceroute命令
func createTracerouteCommand() *cobra.Command {
	tracerouteCmd := &cobra.Command{
		Use:   "traceroute <目标地址>",
		Short: "跟踪网络路径",
		Long: `跟踪数据包到达目标地址经过的所有网络节点
		
示例:
  clixgo network traceroute google.com
  clixgo network traceroute 8.8.8.8 --max-hops 20`,
		Args: cobra.ExactArgs(1),
		RunE: executeTracerouteCommand,
	}

	tracerouteCmd.Flags().IntP("max-hops", "m", 30, "允许的最大跳数")

	return tracerouteCmd
}

// executeTracerouteCommand 执行traceroute命令的业务逻辑
//
// 参数:
//   - cmd: cobra命令实例
//   - args: 命令行参数，args[0]为目标地址
//
// 返回:
//   - error: 执行过程中的错误，nil表示成功
func executeTracerouteCommand(cmd *cobra.Command, args []string) error {
	targetAddress := args[0]
	maxHops, _ := cmd.Flags().GetInt("max-hops")

	traceResult, err := network.Traceroute(targetAddress, maxHops)
	if err != nil {
		return fmt.Errorf("路径跟踪失败: %w", err)
	}

	fmt.Println(traceResult)
	return nil
}

// createDNSCommand 创建DNS查询命令
//
// # DNS命令用于查询域名对应的IP地址
//
// 用法: network dns <域名>
//
// 返回:
//   - *cobra.Command: 配置完整的DNS查询命令
func createDNSCommand() *cobra.Command {
	dnsCmd := &cobra.Command{
		Use:   "dns <域名>",
		Short: "DNS查询",
		Long: `查询指定域名的IP地址记录
		
示例:
  clixgo network dns google.com
  clixgo network dns github.com`,
		Args: cobra.ExactArgs(1),
		RunE: executeDNSCommand,
	}

	return dnsCmd
}

// executeDNSCommand 执行DNS查询命令的业务逻辑
//
// 参数:
//   - cmd: cobra命令实例
//   - args: 命令行参数，args[0]为要查询的域名
//
// 返回:
//   - error: 执行过程中的错误，nil表示成功
func executeDNSCommand(cmd *cobra.Command, args []string) error {
	domainName := args[0]

	ipAddresses, err := network.DNSLookup(domainName)
	if err != nil {
		return fmt.Errorf("DNS查询失败: %w", err)
	}

	fmt.Printf("域名 %s 对应的IP地址:\n", domainName)
	for _, ipAddress := range ipAddresses {
		fmt.Printf("  %s\n", ipAddress)
	}

	return nil
}

// createHTTPCommand 创建HTTP请求命令
//
// # HTTP命令用于发送HTTP GET请求并获取响应
//
// 用法: network http <URL> [--timeout 超时时间]
//
// 返回:
//   - *cobra.Command: 配置完整的HTTP请求命令
func createHTTPCommand() *cobra.Command {
	httpCmd := &cobra.Command{
		Use:   "http <URL>",
		Short: "HTTP请求",
		Long: `向指定URL发送HTTP GET请求并显示响应信息
		
示例:
  clixgo network http https://google.com
  clixgo network http https://api.github.com --timeout 15s`,
		Args: cobra.ExactArgs(1),
		RunE: executeHTTPCommand,
	}

	httpCmd.Flags().DurationP("timeout", "t", 10*time.Second, "请求超时时间")

	return httpCmd
}

// executeHTTPCommand 执行HTTP请求命令的业务逻辑
//
// 参数:
//   - cmd: cobra命令实例
//   - args: 命令行参数，args[0]为目标URL
//
// 返回:
//   - error: 执行过程中的错误，nil表示成功
func executeHTTPCommand(cmd *cobra.Command, args []string) error {
	targetURL := args[0]
	timeoutDuration, _ := cmd.Flags().GetDuration("timeout")

	httpResponse, err := network.HTTPGet(targetURL, timeoutDuration)
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %w", err)
	}

	fmt.Println(httpResponse)
	return nil
}

// createPortCheckCommand 创建端口检查命令
//
// 端口检查命令用于测试指定主机的端口是否开放
//
// 用法: network port <主机地址> <端口号> [--timeout 超时时间]
//
// 返回:
//   - *cobra.Command: 配置完整的端口检查命令
func createPortCheckCommand() *cobra.Command {
	portCmd := &cobra.Command{
		Use:   "port <主机地址> <端口号>",
		Short: "检查端口",
		Long: `检查指定主机的端口是否开放和可连接
		
示例:
  clixgo network port google.com 80
  clixgo network port 192.168.1.1 22 --timeout 3s`,
		Args: cobra.ExactArgs(2),
		RunE: executePortCheckCommand,
	}

	portCmd.Flags().DurationP("timeout", "t", 5*time.Second, "连接超时时间")

	return portCmd
}

// executePortCheckCommand 执行端口检查命令的业务逻辑
//
// 参数:
//   - cmd: cobra命令实例
//   - args: 命令行参数，args[0]为主机地址，args[1]为端口号
//
// 返回:
//   - error: 执行过程中的错误，nil表示成功
func executePortCheckCommand(cmd *cobra.Command, args []string) error {
	hostAddress := args[0]
	portString := args[1]

	// 解析端口号
	portNumber, err := parsePortNumber(portString)
	if err != nil {
		return err
	}

	timeoutDuration, _ := cmd.Flags().GetDuration("timeout")

	isPortOpen, err := network.CheckPort(hostAddress, portNumber, timeoutDuration)
	if err != nil {
		return fmt.Errorf("端口检查失败: %w", err)
	}

	displayPortCheckResult(hostAddress, portNumber, isPortOpen)
	return nil
}

// parsePortNumber 解析端口号字符串
//
// 参数:
//   - portString: 端口号字符串
//
// 返回:
//   - int: 解析后的端口号
//   - error: 解析失败时的错误
func parsePortNumber(portString string) (int, error) {
	portNumber, err := strconv.Atoi(portString)
	if err != nil {
		return 0, fmt.Errorf("无效的端口号 '%s': %w", portString, err)
	}

	if portNumber < 1 || portNumber > 65535 {
		return 0, fmt.Errorf("端口号 %d 超出有效范围 (1-65535)", portNumber)
	}

	return portNumber, nil
}

// displayPortCheckResult 显示端口检查结果
//
// 参数:
//   - hostAddress: 主机地址
//   - portNumber: 端口号
//   - isOpen: 端口是否开放
func displayPortCheckResult(hostAddress string, portNumber int, isOpen bool) {
	statusText := "关闭"
	if isOpen {
		statusText = "开放"
	}

	fmt.Printf("主机 %s 的端口 %d 状态: %s\n", hostAddress, portNumber, statusText)
}

// createIPInfoCommand 创建IP信息查询命令
//
// # IP信息命令用于查询指定IP地址的地理位置和ISP信息
//
// 用法: network ipinfo <IP地址>
//
// 返回:
//   - *cobra.Command: 配置完整的IP信息查询命令
func createIPInfoCommand() *cobra.Command {
	ipInfoCmd := &cobra.Command{
		Use:   "ipinfo <IP地址>",
		Short: "查询IP信息",
		Long: `查询指定IP地址的地理位置、ISP等详细信息
		
示例:
  clixgo network ipinfo 8.8.8.8
  clixgo network ipinfo 1.1.1.1`,
		Args: cobra.ExactArgs(1),
		RunE: executeIPInfoCommand,
	}

	return ipInfoCmd
}

// executeIPInfoCommand 执行IP信息查询命令的业务逻辑
//
// 参数:
//   - cmd: cobra命令实例
//   - args: 命令行参数，args[0]为要查询的IP地址
//
// 返回:
//   - error: 执行过程中的错误，nil表示成功
func executeIPInfoCommand(cmd *cobra.Command, args []string) error {
	ipAddress := args[0]

	ipInfo, err := network.GetIPInfo(ipAddress)
	if err != nil {
		return fmt.Errorf("IP信息查询失败: %w", err)
	}

	displayIPInformation(ipInfo)
	return nil
}

// displayIPInformation 格式化显示IP信息
//
// 参数:
//   - info: IP信息结构体
func displayIPInformation(info *network.IPInfo) {
	fmt.Printf("IP地址信息:\n")
	fmt.Printf("  IP地址: %s\n", info.IP)
	fmt.Printf("  国家:   %s\n", info.Country)
	fmt.Printf("  地区:   %s\n", info.Region)
	fmt.Printf("  城市:   %s\n", info.City)
	fmt.Printf("  ISP:    %s\n", info.ISP)
}

// createDownloadCommand 创建文件下载命令
//
// 下载命令用于从指定URL下载文件到本地
//
// 用法: network download <源URL> <目标文件路径> [--timeout 超时时间]
//
// 返回:
//   - *cobra.Command: 配置完整的文件下载命令
func createDownloadCommand() *cobra.Command {
	downloadCmd := &cobra.Command{
		Use:   "download <源URL> <目标文件路径>",
		Short: "下载文件",
		Long: `从指定URL下载文件到本地指定位置
		
示例:
  clixgo network download https://example.com/file.txt ./file.txt
  clixgo network download https://github.com/user/repo/archive/main.zip ./repo.zip --timeout 60s`,
		Args: cobra.ExactArgs(2),
		RunE: executeDownloadCommand,
	}

	downloadCmd.Flags().DurationP("timeout", "t", 30*time.Second, "下载超时时间")

	return downloadCmd
}

// executeDownloadCommand 执行文件下载命令的业务逻辑
//
// 参数:
//   - cmd: cobra命令实例
//   - args: 命令行参数，args[0]为源URL，args[1]为目标文件路径
//
// 返回:
//   - error: 执行过程中的错误，nil表示成功
func executeDownloadCommand(cmd *cobra.Command, args []string) error {
	sourceURL := args[0]
	targetFilePath := args[1]
	timeoutDuration, _ := cmd.Flags().GetDuration("timeout")

	err := network.DownloadFile(sourceURL, targetFilePath, timeoutDuration)
	if err != nil {
		return fmt.Errorf("文件下载失败: %w", err)
	}

	fmt.Printf("文件已成功下载到: %s\n", targetFilePath)
	return nil
}

// createSSLCheckCommand 创建SSL证书检查命令
//
// # SSL检查命令用于验证指定域名的SSL证书信息
//
// 用法: network ssl <域名或URL>
//
// 返回:
//   - *cobra.Command: 配置完整的SSL证书检查命令
func createSSLCheckCommand() *cobra.Command {
	sslCmd := &cobra.Command{
		Use:   "ssl <域名或URL>",
		Short: "检查SSL证书",
		Long: `检查指定域名或URL的SSL证书有效性和详细信息
		
示例:
  clixgo network ssl google.com
  clixgo network ssl https://github.com`,
		Args: cobra.ExactArgs(1),
		RunE: executeSSLCheckCommand,
	}

	return sslCmd
}

// executeSSLCheckCommand 执行SSL证书检查命令的业务逻辑
//
// 参数:
//   - cmd: cobra命令实例
//   - args: 命令行参数，args[0]为要检查的域名或URL
//
// 返回:
//   - error: 执行过程中的错误，nil表示成功
func executeSSLCheckCommand(cmd *cobra.Command, args []string) error {
	targetDomain := args[0]

	sslInfo, err := network.CheckSSL(targetDomain)
	if err != nil {
		return fmt.Errorf("SSL证书检查失败: %w", err)
	}

	displaySSLCertificateInfo(sslInfo)
	return nil
}

// displaySSLCertificateInfo 格式化显示SSL证书信息
//
// 参数:
//   - info: SSL证书信息结构体
func displaySSLCertificateInfo(info *network.SSLInfo) {
	fmt.Printf("SSL证书信息:\n")
	fmt.Printf("  证书颁发者: %s\n", info.Issuer)
	fmt.Printf("  有效期至:   %s\n", info.Expiry.Format("2006-01-02 15:04:05"))

	// 检查证书是否即将过期
	daysUntilExpiry := int(time.Until(info.Expiry).Hours() / 24)
	if daysUntilExpiry < 30 {
		fmt.Printf("  ⚠️  警告: 证书将在 %d 天后过期\n", daysUntilExpiry)
	} else {
		fmt.Printf("  ✅ 证书有效期还有 %d 天\n", daysUntilExpiry)
	}
}

// createSpeedTestCommand 创建网络速度测试命令
//
// 速度测试命令用于测试当前网络的上传和下载速度
//
// 用法: network speedtest
//
// 返回:
//   - *cobra.Command: 配置完整的网络速度测试命令
func createSpeedTestCommand() *cobra.Command {
	speedTestCmd := &cobra.Command{
		Use:   "speedtest",
		Short: "网络速度测试",
		Long: `测试当前网络连接的上传和下载速度
		
示例:
  clixgo network speedtest`,
		RunE: executeSpeedTestCommand,
	}

	return speedTestCmd
}

// executeSpeedTestCommand 执行网络速度测试命令的业务逻辑
//
// 参数:
//   - cmd: cobra命令实例
//   - args: 命令行参数（此命令不需要参数）
//
// 返回:
//   - error: 执行过程中的错误，nil表示成功
func executeSpeedTestCommand(cmd *cobra.Command, args []string) error {
	fmt.Println("正在进行网络速度测试，请稍候...")

	speedTestResult, err := network.NetworkSpeedTest()
	if err != nil {
		return fmt.Errorf("网络速度测试失败: %w", err)
	}

	displaySpeedTestResults(speedTestResult)
	return nil
}

// displaySpeedTestResults 格式化显示网络速度测试结果
//
// 参数:
//   - result: 速度测试结果结构体
func displaySpeedTestResults(result *network.SpeedTestResult) {
	fmt.Printf("网络速度测试结果:\n")
	fmt.Printf("  下载速度: %.2f Mbps\n", result.Download)
	fmt.Printf("  上传速度: %.2f Mbps\n", result.Upload)

	// 提供速度评价
	downloadSpeed := result.Download
	switch {
	case downloadSpeed >= 100:
		fmt.Printf("  评价: 🚀 极速网络\n")
	case downloadSpeed >= 25:
		fmt.Printf("  评价: ⚡ 高速网络\n")
	case downloadSpeed >= 5:
		fmt.Printf("  评价: 📶 中等速度\n")
	default:
		fmt.Printf("  评价: 🐌 较慢网络\n")
	}
}

// createNetworkMonitorCommand 创建网络监控命令
//
// 网络监控命令用于实时监控网络状态和性能指标
//
// 用法: network monitor [--interval 更新间隔] [--duration 持续时间]
//
// 返回:
//   - *cobra.Command: 配置完整的网络监控命令
func createNetworkMonitorCommand() *cobra.Command {
	monitorCmd := &cobra.Command{
		Use:   "monitor",
		Short: "网络监控",
		Long: `实时监控网络状态，包括接口统计、连接数、延迟等信息
		
示例:
  clixgo network monitor
  clixgo network monitor --interval 1s --duration 60s`,
		RunE: executeNetworkMonitorCommand,
	}

	monitorCmd.Flags().DurationP("interval", "i", 2*time.Second, "监控数据更新间隔")
	monitorCmd.Flags().DurationP("duration", "d", 0, "监控持续时间，0表示持续监控")

	return monitorCmd
}

// executeNetworkMonitorCommand 执行网络监控命令的业务逻辑
//
// 参数:
//   - cmd: cobra命令实例
//   - args: 命令行参数（此命令不需要参数）
//
// 返回:
//   - error: 执行过程中的错误，nil表示成功
func executeNetworkMonitorCommand(cmd *cobra.Command, args []string) error {
	updateInterval, _ := cmd.Flags().GetDuration("interval")
	monitorDuration, _ := cmd.Flags().GetDuration("duration")

	fmt.Println("启动网络监控...")
	fmt.Printf("更新间隔: %v\n", updateInterval)
	if monitorDuration > 0 {
		fmt.Printf("监控时长: %v\n", monitorDuration)
	} else {
		fmt.Println("持续监控 (按 Ctrl+C 停止)")
	}
	fmt.Println("---")

	// 启动监控
	networkMonitorConfig := network.NetworkMonitor{
		Targets:  []string{"8.8.8.8", "1.1.1.1"}, // 默认监控目标
		Interval: updateInterval,
		Timeout:  5 * time.Second,
		AlertConfig: network.AlertConfig{
			Enabled:     false,
			Threshold:   100.0,
			RepeatAfter: time.Hour,
		},
	}

	results, cancel := network.StartMonitoring(networkMonitorConfig)
	defer cancel()

	// 根据持续时间设置停止定时器
	var stopTimer <-chan time.Time
	if monitorDuration > 0 {
		stopTimer = time.After(monitorDuration)
	}

	// 处理监控结果
	for {
		select {
		case result, ok := <-results:
			if !ok {
				return nil // 通道已关闭
			}
			displayNetworkMonitorResult(result)
		case <-stopTimer:
			if stopTimer != nil {
				return nil // 达到持续时间
			}
		}
	}

	return nil
}

// displayNetworkMonitorResult 显示网络监控结果
//
// 参数:
//   - result: 网络监控结果
func displayNetworkMonitorResult(result network.MonitorResult) {
	timestamp := result.Timestamp.Format("2006-01-02 15:04:05")

	if result.Error != nil {
		fmt.Printf("[%s] %s: 错误 - %v\n", timestamp, result.Target, result.Error)
	} else {
		latencyMs := float64(result.Latency.Nanoseconds()) / 1e6
		fmt.Printf("[%s] %s: 状态=%s, 延迟=%.2fms, 丢包率=%.2f%%\n",
			timestamp, result.Target, result.Status, latencyMs, result.PacketLoss)
	}
}

// createNetworkMonitorConfig 创建网络监控配置
//
// 参数:
//   - updateInterval: 更新间隔
//
// 返回:
//   - network.RealtimeMonitorConfig: 网络监控配置
func createNetworkMonitorConfig(updateInterval time.Duration) network.RealtimeMonitorConfig {
	return network.RealtimeMonitorConfig{
		UpdateInterval: updateInterval,
		Timeout:        5 * time.Second,
		MaxHistory:     50,
		EnableAlerts:   true,
		AlertThresholds: network.AlertThresholds{
			LatencyMs:         100.0, // 100ms延迟阈值
			PacketLossPercent: 5.0,   // 5%丢包率阈值
			BandwidthMbps:     80.0,  // 80Mbps带宽使用阈值
			ConnectionCount:   1000,  // 1000连接数阈值
			ErrorRate:         1.0,   // 1%错误率阈值
		},
		MonitoredTargets: []string{"8.8.8.8", "1.1.1.1"}, // 默认监控目标
	}
}
