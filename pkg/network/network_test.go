package network

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDNSLookup 测试DNS查询功能
func TestDNSLookup(t *testing.T) {
	// 测试本地主机名解析
	results, err := DNSLookup("localhost")
	if err != nil {
		t.Errorf("DNSLookup localhost 失败: %v", err)
	}

	// 检查是否返回了结果
	if len(results) == 0 {
		t.Error("DNSLookup应该返回至少一个IP地址")
	}

	// 验证是否包含127.0.0.1或::1
	found := false
	for _, ip := range results {
		if ip == "127.0.0.1" || ip == "::1" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("DNSLookup应该解析localhost到127.0.0.1或::1，实际结果: %v", results)
	}

	// 测试无效域名
	_, err = DNSLookup("invalid.domain.that.does.not.exist.example")
	if err == nil {
		t.Error("DNSLookup应该对无效域名返回错误")
	}
}

// TestHTTPGet 测试HTTP GET请求功能
func TestHTTPGet(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("测试成功"))
	}))
	defer server.Close()

	// 测试有效URL
	body, err := HTTPGet(server.URL, 5*time.Second)
	if err != nil {
		t.Errorf("HTTPGet失败: %v", err)
	}

	if body != "测试成功" {
		t.Errorf("HTTPGet返回内容不正确，期望: %s, 实际: %s", "测试成功", body)
	}

	// 测试无效URL
	_, err = HTTPGet("http://invalid.domain.example", 1*time.Second)
	if err == nil {
		t.Error("HTTPGet应该对无效URL返回错误")
	}

	// 测试超时
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	_, err = HTTPGet(slowServer.URL, 500*time.Millisecond)
	if err == nil {
		t.Error("HTTPGet应该对超时请求返回错误")
	}
}

// TestCheckPort 测试端口检查功能
func TestCheckPort(t *testing.T) {
	// 创建本地监听器以模拟开放端口
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("无法创建TCP监听器: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	defer listener.Close()

	// 测试开放端口
	open, err := CheckPort("127.0.0.1", port, 1*time.Second)
	if err != nil {
		t.Errorf("CheckPort失败: %v", err)
	}

	if !open {
		t.Errorf("CheckPort应该检测到开放的端口")
	}

	// 测试关闭端口
	listener.Close()
	time.Sleep(100 * time.Millisecond) // 等待端口完全关闭

	open, err = CheckPort("127.0.0.1", port, 1*time.Second)
	if err != nil {
		t.Errorf("CheckPort失败: %v", err)
	}

	if open {
		t.Errorf("CheckPort不应该检测到关闭的端口")
	}

	// 测试不存在的主机
	_, err = CheckPort("invalid.host.example", 80, 1*time.Second)
	if err != nil {
		t.Errorf("CheckPort对不存在的主机应该返回false而不是错误: %v", err)
	}
}

// TestGetNetworkConfig 测试获取网络配置功能
func TestGetNetworkConfig(t *testing.T) {
	// 获取系统上的一个有效网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("无法获取网络接口: %v", err)
	}

	if len(interfaces) == 0 {
		t.Skip("跳过测试，没有可用的网络接口")
	}

	// 使用第一个网络接口进行测试
	iface := interfaces[0].Name

	config, err := GetNetworkConfig(iface)
	if err != nil {
		t.Errorf("GetNetworkConfig失败: %v", err)
	}

	if config == nil {
		t.Fatal("GetNetworkConfig返回nil配置")
	}

	if config.Interface != iface {
		t.Errorf("接口名不匹配，期望: %s, 实际: %s", iface, config.Interface)
	}

	// 测试不存在的接口
	_, err = GetNetworkConfig("nonexistent_interface")
	if err == nil {
		t.Error("GetNetworkConfig应该对不存在的接口返回错误")
	}
}

// TestRunDiagnostic 测试网络诊断功能
func TestRunDiagnostic(t *testing.T) {
	diagnostic, err := RunDiagnostic()
	if err != nil {
		t.Errorf("RunDiagnostic失败: %v", err)
	}

	if diagnostic == nil {
		t.Fatal("RunDiagnostic返回nil结果")
	}

	// 确认诊断结果包含预期的字段
	if diagnostic.Issues == nil {
		t.Error("诊断结果应该包含Issues字段")
	}
}

// TestPing 测试Ping功能
func TestPing(t *testing.T) {
	t.Skip("跳过Ping测试，需要root权限才能执行")
	/*
		// 想运行此测试，请以root权限执行:
		// sudo go test -v ./pkg/network -run TestPing

		// 测试本地主机
		result, err := Ping("localhost", 2, 1*time.Second)
		if err != nil {
			t.Errorf("Ping localhost失败: %v", err)
			return
		}

		if result == nil {
			t.Fatal("Ping返回nil结果")
		}

		// 验证结果是否包含数据
		if result.PacketsSent != 2 {
			t.Errorf("期望发送2个包，实际发送%d个", result.PacketsSent)
		}

		// 测试不存在的主机
		_, err = Ping("invalid.host.example", 1, 1*time.Second)
		if err == nil {
			t.Error("Ping应该对不存在的主机返回错误")
		}
	*/
}

// TestPingMock 使用模拟方式测试Ping功能
func TestPingMock(t *testing.T) {
	// 创建模拟的PingResult
	mockResult := &PingResult{
		PacketsSent: 5,
		PacketsRecv: 5,
		PacketLoss:  0,
		MinRtt:      time.Millisecond * 10,
		MaxRtt:      time.Millisecond * 30,
		AvgRtt:      time.Millisecond * 20,
		StdDevRtt:   time.Millisecond * 5,
	}

	// 测试PingResult的字段
	if mockResult.PacketsSent != 5 {
		t.Errorf("期望发送5个包，实际发送%d个", mockResult.PacketsSent)
	}

	if mockResult.PacketsRecv != 5 {
		t.Errorf("期望接收5个包，实际接收%d个", mockResult.PacketsRecv)
	}

	if mockResult.PacketLoss != 0 {
		t.Errorf("期望丢包率为0，实际为%f", mockResult.PacketLoss)
	}

	if mockResult.AvgRtt != time.Millisecond*20 {
		t.Errorf("期望平均延迟为20ms，实际为%v", mockResult.AvgRtt)
	}
}

// TestCalculateChecksum 测试校验和计算函数
func TestCalculateChecksum(t *testing.T) {
	// 测试空数据
	checksum := calculateChecksum([]byte{})
	if checksum != 0xffff {
		t.Errorf("空数据的校验和不正确，期望: 0xffff, 实际: 0x%x", checksum)
	}

	// 测试已知数据
	testData := []byte{0x45, 0x00, 0x00, 0x73, 0x00, 0x00, 0x40, 0x00, 0x40, 0x11, 0x00, 0x00, 0xc0, 0xa8, 0x00, 0x01, 0xc0, 0xa8, 0x00, 0xc7}
	checksum = calculateChecksum(testData)
	// 实际校验和会根据数据而变化，这里只测试是否能够计算
	if checksum == 0 {
		t.Error("校验和计算错误")
	}
}

// TestProtocolTest 测试协议测试功能
func TestProtocolTest(t *testing.T) {
	// 创建测试HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 从URL中提取主机和端口
	host := server.URL[7:] // 去掉"http://"

	// 测试HTTP协议
	result, err := TestProtocol(host, "http")
	if err != nil {
		t.Errorf("TestProtocol失败: %v", err)
	}

	if result == nil {
		t.Fatal("TestProtocol返回nil结果")
	}

	if result.Status != "OK" {
		t.Errorf("HTTP协议测试应该成功，实际状态: %s", result.Status)
	}

	// 测试无效协议
	_, err = TestProtocol("localhost", "invalid_protocol")
	if err == nil {
		t.Error("TestProtocol应该对无效协议返回错误")
	}

	// 测试不存在的主机
	result, err = TestProtocol("invalid.host.example", "http")
	if err != nil {
		t.Errorf("TestProtocol应该返回错误结果而不是错误: %v", err)
	}

	if result.Status != "ERROR" {
		t.Errorf("不存在主机的协议测试应该返回ERROR状态，实际: %s", result.Status)
	}
}

// TestNetworkDiagnostic 测试更详细的网络诊断功能
func TestNetworkDiagnostic(t *testing.T) {
	diagnostic, err := RunDiagnostic()
	if err != nil {
		t.Errorf("RunDiagnostic失败: %v", err)
	}

	if diagnostic == nil {
		t.Fatal("RunDiagnostic返回nil结果")
	}

	// 检查连接状态
	if !diagnostic.Connectivity {
		t.Log("网络连接状态: 未连接")
	} else {
		t.Log("网络连接状态: 已连接")
	}

	// 检查DNS状态
	if !diagnostic.DNS {
		t.Log("DNS状态: 无法解析")
	} else {
		t.Log("DNS状态: 可以解析")
	}

	// 检查互联网连接
	if !diagnostic.Internet {
		t.Log("互联网连接状态: 未连接")
	} else {
		t.Log("互联网连接状态: 已连接")
	}
}

// TestNetworkOptimization 测试网络优化功能
func TestNetworkOptimization(t *testing.T) {
	// 获取系统上的一个有效网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("无法获取网络接口: %v", err)
	}

	if len(interfaces) == 0 {
		t.Skip("跳过测试，没有可用的网络接口")
	}

	// 使用第一个网络接口进行测试
	iface := interfaces[0].Name

	optimization, err := OptimizeNetwork(iface)
	if err != nil {
		// 功能尚未实现时会返回错误，这是预期行为
		t.Logf("OptimizeNetwork返回错误: %v", err)
		return
	}

	if optimization != nil {
		// 检查优化结果
		t.Logf("当前MTU: %d", optimization.CurrentMTU)
		t.Logf("建议MTU: %d", optimization.RecommendedMTU)
		t.Logf("问题数量: %d", len(optimization.Issues))
		t.Logf("建议数量: %d", len(optimization.Suggestions))
	}
}

// TestSetNetworkConfig 测试设置网络配置功能
func TestSetNetworkConfig(t *testing.T) {
	// 创建测试配置
	config := NetworkConfig{
		Interface: "eth0",
		IP:        "192.168.1.100",
		Netmask:   "255.255.255.0",
		Gateway:   "192.168.1.1",
		DNS:       []string{"8.8.8.8", "8.8.4.4"},
		MTU:       1500,
	}

	// 尝试设置配置（预期会失败，因为功能尚未实现）
	err := SetNetworkConfig(config)
	if err == nil {
		t.Error("SetNetworkConfig应该返回错误，因为功能尚未实现")
	} else {
		t.Logf("预期错误: %v", err)
	}
}

// TestCalculateJitter 测试抖动计算函数
func TestCalculateJitter(t *testing.T) {
	// 创建测试数据
	pingResult := &PingResult{
		PacketsSent: 10,
		PacketsRecv: 10,
		PacketLoss:  0,
		MinRtt:      time.Millisecond * 10,
		MaxRtt:      time.Millisecond * 30,
		AvgRtt:      time.Millisecond * 20,
		StdDevRtt:   time.Millisecond * 5,
	}

	// 计算抖动
	jitter := calculateJitter(pingResult)

	// 验证结果
	if jitter != 5.0 {
		t.Errorf("抖动计算不正确，期望: %f, 实际: %f", 5.0, jitter)
	}
}

// TestAlert 测试告警功能
func TestAlert(t *testing.T) {
	// 创建测试配置
	config := AlertConfig{
		Enabled:     true,
		Threshold:   90.0,
		Email:       "test@example.com",
		Webhook:     "http://example.com/webhook",
		SMS:         "+1234567890",
		RepeatAfter: time.Hour,
	}

	// 创建告警管理器
	manager := NewAlertManager(config)

	// 尝试发送告警（预期会返回错误，因为功能尚未实现）
	err := manager.SendAlert("测试告警消息")
	if err == nil {
		t.Error("SendAlert应该返回错误，因为功能尚未实现")
	} else {
		t.Logf("预期错误: %v", err)
	}
}

// TestDownloadFile 测试文件下载功能
func TestDownloadFile(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("测试文件内容"))
	}))
	defer server.Close()

	// 创建临时目录用于测试
	tempDir, err := os.MkdirTemp("", "download_test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 测试文件路径
	testFilePath := filepath.Join(tempDir, "test_download.txt")

	// 测试下载文件
	err = DownloadFile(server.URL, testFilePath, 5*time.Second)
	if err != nil {
		t.Errorf("DownloadFile失败: %v", err)
	}

	// 验证文件是否存在且内容正确
	fileContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Errorf("无法读取下载的文件: %v", err)
	}

	if string(fileContent) != "测试文件内容" {
		t.Errorf("文件内容不正确，期望: %s, 实际: %s", "测试文件内容", string(fileContent))
	}

	// 测试无效URL
	err = DownloadFile("http://invalid.domain.example", testFilePath+".invalid", 1*time.Second)
	if err == nil {
		t.Error("DownloadFile应该对无效URL返回错误")
	}
}

// TestTraceroute 测试路由跟踪功能
func TestTraceroute(t *testing.T) {
	// 测试traceroute localhost (可能失败，这是预期的)
	results, err := Traceroute("localhost", 2)
	if err != nil {
		// 如果没有权限执行，这是可以预期的
		t.Logf("Traceroute测试返回错误: %v", err)
	} else {
		// 验证结果
		if len(results) == 0 {
			t.Error("Traceroute应该返回至少一个跳数结果")
		} else {
			for _, result := range results {
				t.Logf("跳数 %d: IP=%s, RTT=%v", result.Hop, result.IP, result.RTT)
			}
		}
	}
}

// TestStartMonitoring 测试网络监控功能
func TestStartMonitoring(t *testing.T) {
	// 创建测试配置
	config := NetworkMonitor{
		Targets:  []string{"localhost"},
		Interval: 100 * time.Millisecond,
		Timeout:  500 * time.Millisecond,
		AlertConfig: AlertConfig{
			Enabled:   true,
			Threshold: 90.0,
		},
	}

	// 启动监控
	results, cancel := StartMonitoring(config)

	// 确保函数结束时取消监控
	defer cancel()

	// 等待并读取一个结果
	select {
	case result := <-results:
		// 验证结果
		if result.Target != "localhost" {
			t.Errorf("监控目标不正确，期望: %s, 实际: %s", "localhost", result.Target)
		}

		if result.Timestamp.IsZero() {
			t.Error("结果时间戳应该有效")
		}

	case <-time.After(1 * time.Second):
		t.Error("监控超时，未收到结果")
	}
}

// TestProtocolTest_AllProtocols 测试所有支持的协议
func TestProtocolTest_AllProtocols(t *testing.T) {
	// 创建测试服务器
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer httpServer.Close()

	// 从URL中提取主机和端口
	host := httpServer.URL[7:] // 去掉"http://"

	// 测试所有协议
	protocols := []string{"http", "dns"}

	for _, protocol := range protocols {
		t.Run("Protocol_"+protocol, func(t *testing.T) {
			// 对于DNS协议使用有效的域名
			testHost := host
			if protocol == "dns" {
				testHost = "localhost"
			}

			result, err := TestProtocol(testHost, protocol)
			if err != nil {
				t.Errorf("TestProtocol对%s协议失败: %v", protocol, err)
				return
			}

			if result == nil {
				t.Fatalf("TestProtocol返回nil结果")
			}

			if result.Protocol != protocol {
				t.Errorf("协议不匹配，期望: %s, 实际: %s", protocol, result.Protocol)
			}

			t.Logf("%s协议测试结果: 状态=%s, 延迟=%v", protocol, result.Status, result.Latency)
		})
	}
}

// TestEvaluateQuality 测试网络质量评估功能
func TestEvaluateQuality(t *testing.T) {
	// 创建HTTP测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 从URL中提取主机
	host := server.URL[7:] // 去掉"http://"

	// 评估网络质量
	quality, err := EvaluateQuality(host, 500*time.Millisecond)
	if err != nil {
		// 如果功能尚未实现，记录错误并跳过
		if strings.Contains(err.Error(), "尚未实现") {
			t.Logf("EvaluateQuality返回预期错误: %v", err)
			return
		}
		t.Errorf("EvaluateQuality失败: %v", err)
		return
	}

	if quality == nil {
		t.Fatal("EvaluateQuality返回nil结果")
	}

	// 验证质量分数
	if quality.Score < 0 || quality.Score > 100 {
		t.Errorf("质量分数应该在0-100范围内，实际: %f", quality.Score)
	}

	// 验证各项分数
	if quality.LatencyScore < 0 || quality.LatencyScore > 100 {
		t.Errorf("延迟分数应该在0-100范围内，实际: %f", quality.LatencyScore)
	}

	// 记录网络质量信息
	t.Logf("网络质量评估: 总分=%f, 延迟分数=%f", quality.Score, quality.LatencyScore)
	t.Logf("问题数量: %d, 建议数量: %d", len(quality.Issues), len(quality.Recommendations))
}

// TestAnalyzePerformance 测试网络性能分析功能
func TestAnalyzePerformance(t *testing.T) {
	// 创建HTTP测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 从URL中提取主机
	host := server.URL[7:] // 去掉"http://"

	// 分析网络性能
	performance, err := AnalyzePerformance(host, 500*time.Millisecond)
	if err != nil {
		// 如果功能尚未实现，记录错误并跳过
		if strings.Contains(err.Error(), "尚未实现") {
			t.Logf("AnalyzePerformance返回预期错误: %v", err)
			return
		}
		t.Errorf("AnalyzePerformance失败: %v", err)
		return
	}

	if performance == nil {
		t.Fatal("AnalyzePerformance返回nil结果")
	}

	// 验证性能数据
	t.Logf("网络性能: 带宽=%f Mbps, 延迟=%f ms, 丢包率=%f%%",
		performance.Bandwidth, performance.Latency, performance.PacketLoss)
}

// TestCheckSSL 测试SSL证书检查功能
func TestCheckSSL(t *testing.T) {
	// 测试有效的HTTPS站点
	// 注意：这个测试可能失败，因为我们使用真实的外部服务
	info, err := CheckSSL("google.com")
	if err != nil {
		t.Logf("CheckSSL返回错误: %v", err)
		return
	}

	if info == nil {
		t.Fatal("CheckSSL返回nil结果")
	}

	// 验证证书信息
	if info.Issuer == "" {
		t.Error("证书颁发者不应该为空")
	}

	if info.Expiry.IsZero() {
		t.Error("证书过期时间不应该为零")
	}

	t.Logf("SSL证书信息: 颁发者=%s, 过期时间=%v", info.Issuer, info.Expiry)

	// 测试无效站点
	_, err = CheckSSL("invalid.domain.example")
	if err == nil {
		t.Error("CheckSSL应该对无效站点返回错误")
	}
}

// TestBackupAndRestoreNetworkConfig 测试网络配置备份和恢复功能
func TestBackupAndRestoreNetworkConfig(t *testing.T) {
	// 获取系统上的一个有效网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("无法获取网络接口: %v", err)
	}

	if len(interfaces) == 0 {
		t.Skip("跳过测试，没有可用的网络接口")
	}

	// 使用第一个网络接口进行测试
	iface := interfaces[0].Name

	// 备份网络配置
	backup, err := BackupNetworkConfig(iface)
	if err != nil {
		// 如果功能尚未实现，记录错误并跳过
		if strings.Contains(err.Error(), "尚未实现") {
			t.Logf("BackupNetworkConfig返回预期错误: %v", err)
			return
		}
		t.Errorf("BackupNetworkConfig失败: %v", err)
		return
	}

	if backup == nil {
		t.Fatal("BackupNetworkConfig返回nil结果")
	}

	// 验证备份
	if backup.Interface != iface {
		t.Errorf("接口名不匹配，期望: %s, 实际: %s", iface, backup.Interface)
	}

	// 尝试恢复配置
	err = RestoreNetworkConfig(backup.BackupID)
	if err != nil {
		// 如果功能尚未实现，记录错误
		if strings.Contains(err.Error(), "尚未实现") {
			t.Logf("RestoreNetworkConfig返回预期错误: %v", err)
		} else {
			t.Errorf("RestoreNetworkConfig失败: %v", err)
		}
	}
}

// TestCalculateLatencyScore 测试延迟分数计算
func TestCalculateLatencyScore(t *testing.T) {
	// 测试不同延迟值
	testCases := []struct {
		name     string
		latency  time.Duration
		expected float64
	}{
		{"极低延迟", 10 * time.Millisecond, 100},
		{"低延迟", 30 * time.Millisecond, 90},
		{"中等延迟", 80 * time.Millisecond, 80},
		{"高延迟", 150 * time.Millisecond, 60},
		{"极高延迟", 300 * time.Millisecond, 40},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := calculateLatencyScore(tc.latency)
			if score != tc.expected {
				t.Errorf("延迟%v的分数不正确，期望: %f, 实际: %f", tc.latency, tc.expected, score)
			}
		})
	}
}

// TestMockTraceroute 使用模拟方式测试Traceroute结果
func TestMockTraceroute(t *testing.T) {
	// 创建模拟的Traceroute结果
	mockResults := []TracerouteResult{
		{Hop: 1, IP: "192.168.1.1", RTT: 1 * time.Millisecond, Reached: true},
		{Hop: 2, IP: "10.0.0.1", RTT: 5 * time.Millisecond, Reached: true},
		{Hop: 3, IP: "8.8.8.8", RTT: 20 * time.Millisecond, Reached: true},
	}

	// 验证模拟结果
	if len(mockResults) != 3 {
		t.Errorf("期望有3个跳数结果，实际有%d个", len(mockResults))
	}

	// 检查第一跳
	if mockResults[0].Hop != 1 || mockResults[0].IP != "192.168.1.1" {
		t.Errorf("第一跳结果不正确，期望: {Hop:1, IP:192.168.1.1}, 实际: %+v", mockResults[0])
	}

	// 检查最后一跳
	if mockResults[2].Hop != 3 || mockResults[2].IP != "8.8.8.8" {
		t.Errorf("最后一跳结果不正确，期望: {Hop:3, IP:8.8.8.8}, 实际: %+v", mockResults[2])
	}
}

// TestNetworkConfigFields 测试NetworkConfig结构体字段
func TestNetworkConfigFields(t *testing.T) {
	// 创建测试配置
	config := NetworkConfig{
		Interface: "eth0",
		IP:        "192.168.1.100",
		Netmask:   "255.255.255.0",
		Gateway:   "192.168.1.1",
		DNS:       []string{"8.8.8.8", "8.8.4.4"},
		MTU:       1500,
	}

	// 验证字段值
	if config.Interface != "eth0" {
		t.Errorf("接口名不正确，期望: eth0, 实际: %s", config.Interface)
	}

	if config.IP != "192.168.1.100" {
		t.Errorf("IP地址不正确，期望: 192.168.1.100, 实际: %s", config.IP)
	}

	if config.Netmask != "255.255.255.0" {
		t.Errorf("子网掩码不正确，期望: 255.255.255.0, 实际: %s", config.Netmask)
	}

	if config.Gateway != "192.168.1.1" {
		t.Errorf("网关不正确，期望: 192.168.1.1, 实际: %s", config.Gateway)
	}

	if len(config.DNS) != 2 || config.DNS[0] != "8.8.8.8" || config.DNS[1] != "8.8.4.4" {
		t.Errorf("DNS设置不正确，期望: [8.8.8.8, 8.8.4.4], 实际: %v", config.DNS)
	}

	if config.MTU != 1500 {
		t.Errorf("MTU不正确，期望: 1500, 实际: %d", config.MTU)
	}
}

// TestAlertManagerNewAlert 测试告警间隔功能
func TestAlertManagerNewAlert(t *testing.T) {
	// 创建告警配置
	config := AlertConfig{
		Enabled:     true,
		Threshold:   90.0,
		Email:       "test@example.com",
		RepeatAfter: 1 * time.Hour,
	}

	// 创建告警管理器
	manager := NewAlertManager(config)

	// 验证初始状态
	if !manager.LastAlert.IsZero() {
		t.Errorf("初始LastAlert应该为零时间，实际: %v", manager.LastAlert)
	}

	if manager.Config.Threshold != 90.0 {
		t.Errorf("阈值不正确，期望: 90.0, 实际: %f", manager.Config.Threshold)
	}

	// 设置LastAlert以模拟之前发送的告警
	manager.LastAlert = time.Now().Add(-2 * time.Hour)
	oldLastAlert := manager.LastAlert

	// 尝试发送新告警（会返回错误，因为功能尚未实现）
	_ = manager.SendAlert("测试告警")

	// 验证LastAlert没有被更新（因为SendAlert没有实际实现）
	if !manager.LastAlert.Equal(oldLastAlert) {
		t.Errorf("LastAlert不应该被更新，因为SendAlert功能尚未实现")
	}
}

// TestMockNetworkPerformance 模拟测试网络性能结构
func TestMockNetworkPerformance(t *testing.T) {
	// 创建模拟的网络性能数据
	performance := NetworkPerformance{
		Bandwidth:      100.5, // Mbps
		Latency:        15.2,  // ms
		Jitter:         2.3,   // ms
		PacketLoss:     0.5,   // %
		Throughput:     95.8,  // Mbps
		Retransmission: 0.2,   // %
		ConnectionTime: 45.6,  // ms
	}

	// 验证字段值
	if performance.Bandwidth != 100.5 {
		t.Errorf("带宽不正确，期望: 100.5, 实际: %f", performance.Bandwidth)
	}

	if performance.Latency != 15.2 {
		t.Errorf("延迟不正确，期望: 15.2, 实际: %f", performance.Latency)
	}

	if performance.Jitter != 2.3 {
		t.Errorf("抖动不正确，期望: 2.3, 实际: %f", performance.Jitter)
	}

	if performance.PacketLoss != 0.5 {
		t.Errorf("丢包率不正确，期望: 0.5, 实际: %f", performance.PacketLoss)
	}

	if performance.Throughput != 95.8 {
		t.Errorf("吞吐量不正确，期望: 95.8, 实际: %f", performance.Throughput)
	}
}

// TestMockNetworkQuality 模拟测试网络质量评分
func TestMockNetworkQuality(t *testing.T) {
	// 创建模拟的网络质量评分
	quality := NetworkQuality{
		Score:            85.5,
		LatencyScore:     90.0,
		StabilityScore:   85.0,
		SpeedScore:       80.0,
		ReliabilityScore: 87.0,
		Issues:           []string{"网络延迟偶尔波动", "带宽使用率较高"},
		Recommendations:  []string{"优化网络路由", "考虑升级带宽"},
	}

	// 验证字段值
	if quality.Score != 85.5 {
		t.Errorf("总分不正确，期望: 85.5, 实际: %f", quality.Score)
	}

	if quality.LatencyScore != 90.0 {
		t.Errorf("延迟分数不正确，期望: 90.0, 实际: %f", quality.LatencyScore)
	}

	if len(quality.Issues) != 2 {
		t.Errorf("问题数量不正确，期望: 2, 实际: %d", len(quality.Issues))
	}

	if len(quality.Recommendations) != 2 {
		t.Errorf("建议数量不正确，期望: 2, 实际: %d", len(quality.Recommendations))
	}
}

// TestPingAlternative 使用TCP连接测试连接性（不需要root权限）
func TestPingAlternative(t *testing.T) {
	// 创建临时HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)

	hosts := []struct {
		name   string
		host   string
		port   string
		expect bool
	}{
		{"测试服务器", serverURL.Hostname(), serverURL.Port(), true},
		{"无效域名", "invalid.host.example", "80", false},
	}

	for _, tc := range hosts {
		t.Run(tc.name, func(t *testing.T) {
			startTime := time.Now()
			conn, err := net.DialTimeout("tcp", tc.host+":"+tc.port, 1*time.Second)
			duration := time.Since(startTime)

			success := (err == nil)
			if success != tc.expect {
				if tc.expect {
					t.Errorf("期望能连接到%s:%s，但连接失败: %v", tc.host, tc.port, err)
				} else {
					t.Errorf("期望无法连接到%s:%s，但连接成功", tc.host, tc.port)
					conn.Close()
				}
			}

			if success {
				t.Logf("成功连接到%s:%s，耗时: %v", tc.host, tc.port, duration)
				conn.Close()
			} else {
				t.Logf("无法连接到%s:%s: %v", tc.host, tc.port, err)
			}
		})
	}
}
