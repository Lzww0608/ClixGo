/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-12 11:00:00
* @Description: 网络监控和管理的CLI命令定义
 */

package cli

import (
	"fmt"
	"net"
	"strconv"
	"strings"
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
		Short: "🌐 网络工具",
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
// 用法: network ping <target>
//
// 返回:
//   - *cobra.Command: 配置完整的ping命令
func createPingCommand() *cobra.Command {
	pingCmd := &cobra.Command{
		Use:   "ping <target>",
		Short: "🏓 测试网络连接",
		Long: `向指定目标发送ICMP包测试网络连通性

示例:
  clixgo network ping google.com              # 基础ping测试
  clixgo network ping 8.8.8.8 -c 10          # 发送10个包
  clixgo network ping github.com -t 3s       # 设置3秒超时

支持的目标格式:
  • 域名: google.com, github.com
  • IPv4: 8.8.8.8, 192.168.1.1
  • IPv6: 2001:4860:4860::8888`,
		Args: cobra.ExactArgs(1),
		RunE: executePingCommand,
	}

	pingCmd.Flags().Int("count", 4, "发送的ping包数量 (1-100)")
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
	target := strings.TrimSpace(args[0])

	// 验证目标地址
	if target == "" {
		return fmt.Errorf("❌ 错误: 目标地址不能为空\n\n💡 提示: 请提供要ping的目标，例如:\n  clixgo network ping google.com")
	}

	// 验证目标地址格式
	if err := validateNetworkTarget(target); err != nil {
		return fmt.Errorf("❌ 错误: %v\n\n💡 建议:\n  • 检查域名拼写是否正确\n  • 确认IP地址格式是否有效\n  • 尝试使用知名服务如 google.com 或 8.8.8.8", err)
	}

	packetCount, _ := cmd.Flags().GetInt("count")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	// 验证参数范围
	if packetCount < 1 || packetCount > 100 {
		return fmt.Errorf("❌ 错误: 包数量超出范围 (%d)\n\n💡 提示: 包数量应在 1-100 之间", packetCount)
	}

	if timeout < time.Millisecond || timeout > 30*time.Second {
		return fmt.Errorf("❌ 错误: 超时时间超出范围 (%v)\n\n💡 提示: 超时时间应在 1ms-30s 之间", timeout)
	}

	fmt.Printf("🏓 正在ping %s (%d个包，超时%v)...\n\n", target, packetCount, timeout)

	result, err := network.Ping(target, packetCount, timeout)
	if err != nil {
		return fmt.Errorf("❌ Ping失败: %v\n\n💡 建议:\n  • 检查网络连接是否正常\n  • 确认目标地址是否可达\n  • 尝试增加超时时间", err)
	}

	fmt.Printf("✅ Ping完成\n%s", result)
	return nil
}

// createTracerouteCommand 创建traceroute命令
//
// traceroute命令用于跟踪数据包到达目标地址的网络路径
//
// 用法: network traceroute <target>
//
// 返回:
//   - *cobra.Command: 配置完整的traceroute命令
func createTracerouteCommand() *cobra.Command {
	tracerouteCmd := &cobra.Command{
		Use:   "traceroute <target>",
		Short: "🛤️  跟踪网络路径",
		Long: `跟踪数据包到达目标地址经过的所有网络节点

示例:
  clixgo network traceroute google.com        # 跟踪到google的路径
  clixgo network traceroute 8.8.8.8 -m 20    # 限制最大20跳

注意: 
  • 此操作可能需要一些时间完成
  • 某些路由器可能不响应traceroute请求
  • 结果显示每一跳的IP地址和响应时间`,
		Args: cobra.ExactArgs(1),
		RunE: executeTracerouteCommand,
	}

	tracerouteCmd.Flags().IntP("max-hops", "m", 30, "最大跳数 (1-64)")

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
	target := strings.TrimSpace(args[0])

	// 验证目标地址
	if target == "" {
		return fmt.Errorf("❌ 错误: 目标地址不能为空\n\n💡 提示: 请提供要跟踪的目标，例如:\n  clixgo network traceroute google.com")
	}

	if err := validateNetworkTarget(target); err != nil {
		return fmt.Errorf("❌ 错误: %v\n\n💡 建议:\n  • 检查域名拼写是否正确\n  • 确认IP地址格式是否有效", err)
	}

	maxHops, _ := cmd.Flags().GetInt("max-hops")

	// 验证跳数范围
	if maxHops < 1 || maxHops > 64 {
		return fmt.Errorf("❌ 错误: 最大跳数超出范围 (%d)\n\n💡 提示: 跳数应在 1-64 之间", maxHops)
	}

	fmt.Printf("🛤️  正在跟踪到 %s 的路径 (最大%d跳)...\n\n", target, maxHops)

	result, err := network.Traceroute(target, maxHops)
	if err != nil {
		return fmt.Errorf("❌ 路径跟踪失败: %v\n\n💡 建议:\n  • 检查网络连接是否正常\n  • 某些网络可能阻止traceroute\n  • 尝试使用不同的目标地址", err)
	}

	fmt.Printf("✅ 路径跟踪完成\n%s", result)
	return nil
}

// createDNSCommand 创建DNS查询命令
//
// # DNS命令用于查询域名对应的IP地址
//
// 用法: network dns <domain>
//
// 返回:
//   - *cobra.Command: 配置完整的DNS查询命令
func createDNSCommand() *cobra.Command {
	dnsCmd := &cobra.Command{
		Use:   "dns <domain>",
		Short: "🔍 DNS查询",
		Long: `查询指定域名的IP地址记录

示例:
  clixgo network dns google.com              # 查询A记录
  clixgo network dns github.com              # 查询域名解析
  clixgo network dns example.org             # 查询IP地址

支持查询类型:
  • A记录 (IPv4地址)
  • AAAA记录 (IPv6地址)
  • 自动检测最佳DNS服务器`,
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
	domain := strings.TrimSpace(args[0])

	// 验证域名
	if domain == "" {
		return fmt.Errorf("❌ 错误: 域名不能为空\n\n💡 提示: 请提供要查询的域名，例如:\n  clixgo network dns google.com")
	}

	if err := validateDomainName(domain); err != nil {
		return fmt.Errorf("❌ 错误: %v\n\n💡 建议:\n  • 检查域名格式是否正确\n  • 确认域名拼写无误\n  • 域名示例: google.com, github.com", err)
	}

	fmt.Printf("🔍 正在查询域名 %s...\n\n", domain)

	addresses, err := network.DNSLookup(domain)
	if err != nil {
		return fmt.Errorf("❌ DNS查询失败: %v\n\n💡 建议:\n  • 检查网络连接是否正常\n  • 确认域名是否存在\n  • 尝试使用公共DNS服务器", err)
	}

	if len(addresses) == 0 {
		fmt.Printf("ℹ️  域名 %s 没有找到IP地址记录\n", domain)
		return nil
	}

	fmt.Printf("✅ 域名 %s 解析结果:\n", domain)
	for i, addr := range addresses {
		var addrType string
		if net.ParseIP(addr).To4() != nil {
			addrType = "IPv4"
		} else {
			addrType = "IPv6"
		}
		fmt.Printf("  [%d] %s (%s)\n", i+1, addr, addrType)
	}

	return nil
}

// validateNetworkTarget 验证网络目标地址（域名或IP）
func validateNetworkTarget(target string) error {
	// 尝试解析为IP地址
	if ip := net.ParseIP(target); ip != nil {
		return nil
	}

	// 验证域名格式
	return validateDomainName(target)
}

// validateDomainName 验证域名格式
func validateDomainName(domain string) error {
	if len(domain) == 0 {
		return fmt.Errorf("域名不能为空")
	}

	if len(domain) > 253 {
		return fmt.Errorf("域名过长 (最大253字符)")
	}

	// 基本域名格式检查
	if strings.Contains(domain, "..") {
		return fmt.Errorf("域名格式无效: 包含连续的点")
	}

	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return fmt.Errorf("域名格式无效: 不能以点开头或结尾")
	}

	// 检查是否包含有效字符
	for _, char := range domain {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '.' || char == '-') {
			return fmt.Errorf("域名格式无效: 包含无效字符 '%c'", char)
		}
	}

	return nil
}

// createHTTPCommand 创建HTTP请求命令
func createHTTPCommand() *cobra.Command {
	httpCmd := &cobra.Command{
		Use:   "http <url>",
		Short: "🌐 HTTP请求测试",
		Long: `向指定URL发送HTTP GET请求并显示响应信息

示例:
  clixgo network http https://google.com        # 基础HTTP请求
  clixgo network http https://api.github.com -t 15s    # 设置超时时间
  clixgo network http http://example.com        # 测试HTTP网站

支持的URL格式:
  • HTTPS: https://example.com
  • HTTP: http://example.com  
  • 带端口: https://example.com:8080
  • 带路径: https://api.github.com/users`,
		Args: cobra.ExactArgs(1),
		RunE: executeHTTPCommand,
	}

	httpCmd.Flags().DurationP("timeout", "t", 10*time.Second, "请求超时时间 (1s-60s)")

	return httpCmd
}

// executeHTTPCommand 执行HTTP请求命令的业务逻辑
func executeHTTPCommand(cmd *cobra.Command, args []string) error {
	url := strings.TrimSpace(args[0])

	// 验证URL
	if url == "" {
		return fmt.Errorf("❌ 错误: URL不能为空\n\n💡 提示: 请提供要测试的URL，例如:\n  clixgo network http https://google.com")
	}

	if err := validateURL(url); err != nil {
		return fmt.Errorf("❌ 错误: %v\n\n💡 建议:\n  • 确保URL格式正确 (http://或https://)\n  • 检查域名拼写是否正确\n  • URL示例: https://google.com", err)
	}

	timeout, _ := cmd.Flags().GetDuration("timeout")

	// 验证超时时间
	if timeout < time.Second || timeout > 60*time.Second {
		return fmt.Errorf("❌ 错误: 超时时间超出范围 (%v)\n\n💡 提示: 超时时间应在 1s-60s 之间", timeout)
	}

	fmt.Printf("🌐 正在请求 %s (超时%v)...\n\n", url, timeout)

	response, err := network.HTTPGet(url, timeout)
	if err != nil {
		return fmt.Errorf("❌ HTTP请求失败: %v\n\n💡 建议:\n  • 检查网络连接是否正常\n  • 确认URL是否可访问\n  • 尝试增加超时时间\n  • 检查是否需要代理设置", err)
	}

	fmt.Printf("✅ HTTP请求完成\n%s", response)
	return nil
}

// validateURL 验证URL格式
func validateURL(url string) error {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("URL格式无效: 必须以 http:// 或 https:// 开头")
	}

	// 移除协议前缀进行进一步验证
	urlWithoutProtocol := strings.TrimPrefix(url, "https://")
	urlWithoutProtocol = strings.TrimPrefix(urlWithoutProtocol, "http://")

	if len(urlWithoutProtocol) == 0 {
		return fmt.Errorf("URL格式无效: 缺少域名部分")
	}

	// 分离域名和路径
	parts := strings.Split(urlWithoutProtocol, "/")
	domain := parts[0]

	// 分离域名和端口
	hostPort := strings.Split(domain, ":")
	host := hostPort[0]

	// 验证端口（如果存在）
	if len(hostPort) > 1 {
		port, err := strconv.Atoi(hostPort[1])
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("URL格式无效: 端口号无效")
		}
	}

	// 验证主机名
	if host == "" {
		return fmt.Errorf("URL格式无效: 主机名不能为空")
	}

	return validateDomainName(host)
}

// createPortCheckCommand 创建端口检查命令
func createPortCheckCommand() *cobra.Command {
	portCmd := &cobra.Command{
		Use:   "port <host> <port>",
		Short: "🔌 检查端口连通性",
		Long: `检查指定主机的端口是否开放和可连接

示例:
  clixgo network port google.com 80          # 检查HTTP端口
  clixgo network port 192.168.1.1 22 -t 3s  # 检查SSH端口
  clixgo network port github.com 443        # 检查HTTPS端口
  clixgo network port localhost 8080        # 检查本地服务

常用端口:
  • 22 - SSH
  • 80 - HTTP
  • 443 - HTTPS
  • 3306 - MySQL
  • 5432 - PostgreSQL`,
		Args: cobra.ExactArgs(2),
		RunE: executePortCheckCommand,
	}

	portCmd.Flags().DurationP("timeout", "t", 5*time.Second, "连接超时时间 (1s-30s)")

	return portCmd
}

// executePortCheckCommand 执行端口检查命令的业务逻辑
func executePortCheckCommand(cmd *cobra.Command, args []string) error {
	host := strings.TrimSpace(args[0])
	portStr := strings.TrimSpace(args[1])

	// 验证主机地址
	if host == "" {
		return fmt.Errorf("❌ 错误: 主机地址不能为空\n\n💡 提示: 请提供要检查的主机，例如:\n  clixgo network port google.com 80")
	}

	if err := validateNetworkTarget(host); err != nil {
		return fmt.Errorf("❌ 错误: %v\n\n💡 建议:\n  • 检查主机地址拼写是否正确\n  • 确认域名或IP地址格式是否有效", err)
	}

	// 验证端口号
	if portStr == "" {
		return fmt.Errorf("❌ 错误: 端口号不能为空\n\n💡 提示: 请提供要检查的端口号，例如:\n  clixgo network port google.com 80")
	}

	port, err := parsePortNumber(portStr)
	if err != nil {
		return fmt.Errorf("❌ 错误: %v\n\n💡 建议:\n  • 端口号应为1-65535之间的数字\n  • 常用端口: 22(SSH), 80(HTTP), 443(HTTPS)", err)
	}

	timeout, _ := cmd.Flags().GetDuration("timeout")

	// 验证超时时间
	if timeout < time.Second || timeout > 30*time.Second {
		return fmt.Errorf("❌ 错误: 超时时间超出范围 (%v)\n\n💡 提示: 超时时间应在 1s-30s 之间", timeout)
	}

	fmt.Printf("🔌 正在检查 %s:%d 的连通性 (超时%v)...\n\n", host, port, timeout)

	isOpen, err := network.CheckPort(host, port, timeout)
	if err != nil {
		return fmt.Errorf("❌ 端口检查失败: %v\n\n💡 建议:\n  • 检查网络连接是否正常\n  • 确认目标主机是否可达\n  • 端口可能被防火墙阻止", err)
	}

	displayPortCheckResult(host, port, isOpen)
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
func displayPortCheckResult(host string, port int, isOpen bool) {
	if isOpen {
		fmt.Printf("✅ 端口检查成功\n")
		fmt.Printf("🔌 %s:%d - 🟢 开放\n", host, port)

		// 显示常见服务信息
		if serviceName := getServiceByPort(port); serviceName != "" {
			fmt.Printf("📋 常见服务: %s\n", serviceName)
		}
	} else {
		fmt.Printf("❌ 端口不可达\n")
		fmt.Printf("🔌 %s:%d - 🔴 关闭/超时\n", host, port)

		fmt.Printf("\n💡 可能原因:\n")
		fmt.Printf("  • 端口未开放或服务未运行\n")
		fmt.Printf("  • 被防火墙或安全组阻止\n")
		fmt.Printf("  • 网络连接问题\n")
	}
}

// getServiceByPort 根据端口号获取常见服务名称
func getServiceByPort(port int) string {
	services := map[int]string{
		21:   "FTP",
		22:   "SSH",
		23:   "Telnet",
		25:   "SMTP",
		53:   "DNS",
		80:   "HTTP",
		110:  "POP3",
		143:  "IMAP",
		443:  "HTTPS",
		993:  "IMAPS",
		995:  "POP3S",
		3306: "MySQL",
		5432: "PostgreSQL",
		6379: "Redis",
		8080: "HTTP Alternative",
		9200: "Elasticsearch",
	}

	return services[port]
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
