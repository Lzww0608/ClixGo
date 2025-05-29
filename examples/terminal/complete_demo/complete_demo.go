/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 终端功能完整示例程序
 */

package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

// 全局超时配置
const (
	DefaultTimeout = 5 * time.Second
	QuickTimeout   = 2 * time.Second
	LongTimeout    = 10 * time.Second
)

// TimeoutError 超时错误
type TimeoutError struct {
	Operation string
	Duration  time.Duration
}

func (e TimeoutError) Error() string {
	return fmt.Sprintf("操作 '%s' 超时 (%.2fs)", e.Operation, e.Duration.Seconds())
}

// withTimeout 包装函数以添加超时机制
func withTimeout(name string, timeout time.Duration, fn func() error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- fn()
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return TimeoutError{Operation: name, Duration: timeout}
	}
}

func main() {
	// 检查是否有参数指定运行模式
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "enhanced":
			runEnhancedDemo()
		case "interactive":
			runInteractiveDemo()
		default:
			showUsage()
		}
	} else {
		runCompleteDemo()
	}
}

func showUsage() {
	fmt.Println("ClixGo 终端多路复用器演示程序")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  go run complete_demo.go                 - 运行完整演示")
	fmt.Println("  go run complete_demo.go enhanced        - 运行增强功能演示")
	fmt.Println("  go run complete_demo.go interactive     - 运行交互式演示")
	fmt.Println()
}

func runEnhancedDemo() {
	fmt.Println("🚀 === ClixGo 增强功能演示 ===")

	// 初始化日志
	logger.InitLogger()

	fmt.Println("\n运行增强功能演示...")
	demonstrateEnhancedFeatures()

	fmt.Println("\n✅ 增强功能演示完成！")
}

func runCompleteDemo() {
	fmt.Println("🚀 === ClixGo 完整演示程序 ===")

	// 初始化日志
	logger.InitLogger()

	// 第一部分：基础功能演示
	fmt.Println("\n📋 第一部分：基础功能演示")
	if err := withTimeout("基础功能演示", LongTimeout, func() error {
		demonstrateBasicFeatures()
		return nil
	}); err != nil {
		fmt.Printf("⚠️  基础功能演示超时: %v\n", err)
	}

	// 第二部分：增强功能演示
	fmt.Println("\n⚡ 第二部分：增强功能演示")
	if err := withTimeout("增强功能演示", LongTimeout, func() error {
		demonstrateEnhancedFeatures()
		return nil
	}); err != nil {
		fmt.Printf("⚠️  增强功能演示超时: %v\n", err)
	}

	// 第三部分：性能监控演示
	fmt.Println("\n📊 第三部分：性能监控演示")
	if err := withTimeout("性能监控演示", LongTimeout, func() error {
		demonstratePerformanceMonitoring()
		return nil
	}); err != nil {
		fmt.Printf("⚠️  性能监控演示超时: %v\n", err)
	}

	// 第四部分：实际使用场景演示
	fmt.Println("\n️  第四部分：实际使用场景演示")
	if err := withTimeout("实际场景演示", DefaultTimeout, func() error {
		demonstrateRealWorldScenarios()
		return nil
	}); err != nil {
		fmt.Printf("⚠️  实际场景演示超时: %v\n", err)
	}

	fmt.Println("\n✅ 完整演示结束！")
	showNextSteps()
}

func demonstrateBasicFeatures() {
	fmt.Println("1. 创建基础配置...")
	config := &terminal.TerminalConfig{
		PrefixKey:         "C-b",
		MouseEnabled:      true,
		StatusBar:         true,
		AutoSave:          false,
		SaveInterval:      time.Minute * 5,
		Theme:             "default",
		BufferSize:        1000,
		ScrollBack:        1000,
		ClixGoIntegration: false,
		NetworkMonitor:    false,
		TaskIntegration:   false,
	}

	fmt.Println("2. 创建会话管理器...")
	sm := terminal.NewSessionManager(config)

	fmt.Println("3. 创建测试会话...")
	session, err := sm.CreateSession("demo-basic")
	if err != nil {
		log.Printf("创建会话失败: %v", err)
		return
	}
	fmt.Printf("   ✓ 会话创建成功: %s (ID: %s)\n", session.Name, session.ID[:8])

	fmt.Println("4. 测试窗口操作...")
	window, err := sm.CreateWindow(session.ID, "test-window")
	if err != nil {
		log.Printf("创建窗口失败: %v", err)
		return
	}
	fmt.Printf("   ✓ 窗口创建成功: %s\n", window.Name)

	fmt.Println("5. 测试面板分割...")
	for i := 0; i < 3; i++ {
		pane, err := sm.SplitPane(session.ID, 0, "vertical")
		if err != nil {
			log.Printf("分割面板失败: %v", err)
			continue
		}
		fmt.Printf("   ✓ 面板 %d 创建成功 (ID: %s)\n", i+1, pane.ID[:8])
	}

	fmt.Println("6. 测试UI渲染（带超时保护）...")
	err = withTimeout("UI渲染", QuickTimeout, func() error {
		ui := terminal.NewUIRenderer(80, 24, nil) // 使用较小的尺寸避免过多计算
		if len(session.Windows) > 0 {
			output := ui.RenderWindow(session.Windows[0])
			if len(output) > 500 { // 限制输出长度
				output = output[:500] + "..."
			}
			fmt.Printf("   ✓ UI渲染成功 (输出长度: %d 字符)\n", len(output))

			// 可选：输出一小部分渲染结果作为验证
			lines := strings.Split(output, "\n")
			if len(lines) > 5 {
				fmt.Printf("   预览前几行:\n")
				for i := 0; i < 3 && i < len(lines); i++ {
					if len(lines[i]) > 50 {
						fmt.Printf("     %s...\n", lines[i][:50])
					} else {
						fmt.Printf("     %s\n", lines[i])
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("   ⚠️  UI渲染超时: %v\n", err)
	}

	fmt.Println("   基础功能演示完成！")
}

func demonstrateEnhancedFeatures() {
	fmt.Println("1. 创建增强配置...")
	config := &terminal.TerminalConfig{
		PrefixKey:         "C-b",
		MouseEnabled:      true,
		StatusBar:         true,
		AutoSave:          true,
		SaveInterval:      time.Minute * 2,
		Theme:             "enhanced",
		BufferSize:        2000,
		ScrollBack:        2000,
		ClixGoIntegration: true,
		NetworkMonitor:    true,
		TaskIntegration:   true,
		KeyBindings: []terminal.KeyBinding{
			{Key: "C-b d", Command: "detach_session"},
			{Key: "C-b c", Command: "create_window"},
			{Key: "C-b Space", Command: "next_layout"},
		},
	}

	fmt.Println("2. 启动增强服务器...")
	server := terminal.NewEnhancedTerminalServer(config)

	var serverStarted bool
	err := withTimeout("服务器启动", DefaultTimeout, func() error {
		if err := server.Start(); err != nil {
			return err
		}
		serverStarted = true
		return nil
	})

	if err != nil {
		log.Printf("启动服务器失败: %v", err)
		return
	}

	if serverStarted {
		defer func() {
			withTimeout("服务器停止", QuickTimeout, func() error {
				return server.Stop()
			})
		}()
	}

	fmt.Printf("   ✓ 增强服务器启动成功，Socket: %s\n", server.GetSocketPath())

	// 等待服务器完全启动
	time.Sleep(time.Second)

	fmt.Println("3. 测试PTY功能...")
	if err := withTimeout("PTY功能测试", DefaultTimeout, func() error {
		demonstratePTYFeatures(server)
		return nil
	}); err != nil {
		fmt.Printf("   ⚠️  PTY功能测试超时: %v\n", err)
	}

	fmt.Println("4. 测试性能监控...")
	if err := withTimeout("性能监控测试", DefaultTimeout, func() error {
		demonstrateMonitoringFeatures(server)
		return nil
	}); err != nil {
		fmt.Printf("   ⚠️  性能监控测试超时: %v\n", err)
	}

	fmt.Println("5. 测试服务器统计...")
	if err := withTimeout("服务器统计测试", QuickTimeout, func() error {
		demonstrateServerStats(server)
		return nil
	}); err != nil {
		fmt.Printf("   ⚠️  服务器统计测试超时: %v\n", err)
	}

	fmt.Println("   增强功能演示完成！")
}

func demonstratePTYFeatures(server *terminal.EnhancedTerminalServer) {
	// 创建会话
	sm := server.GetSessionManager()
	session, err := sm.CreateSession("pty-demo")
	if err != nil {
		log.Printf("创建PTY会话失败: %v", err)
		return
	}

	fmt.Printf("   ✓ PTY会话创建: %s\n", session.Name)

	// 获取PTY管理器并创建PTY
	config := &terminal.TerminalConfig{
		ClixGoIntegration: true,
	}
	ptyManager := terminal.NewSimplePTYManager(config)

	testCommands := []string{
		"echo 'Hello from PTY 1'",
		"echo 'PTY 2: Date'; date",
		"echo 'PTY 3: Working directory'; pwd",
	}

	for i, cmd := range testCommands {
		// 使用超时保护每个PTY创建
		err := withTimeout(fmt.Sprintf("PTY-%d创建", i+1), QuickTimeout, func() error {
			ptyID := fmt.Sprintf("demo-pty-%d", i+1)
			pty, err := ptyManager.CreateSimplePTY(ptyID, cmd, "/tmp", 80, 24)
			if err != nil {
				return fmt.Errorf("创建PTY失败: %v", err)
			}

			if err := pty.Start(); err != nil {
				return fmt.Errorf("启动PTY失败: %v", err)
			}

			fmt.Printf("   ✓ PTY %d 创建并启动 (PID: %d)\n", i+1, pty.GetPID())

			// 等待一段时间让命令执行
			time.Sleep(100 * time.Millisecond)

			// 尝试读取输出（带超时）
			if data, err := pty.Read(); err == nil && len(data) > 0 {
				output := string(data)
				if len(output) > 100 {
					output = output[:100] + "..."
				}
				output = strings.ReplaceAll(output, "\n", " ")
				fmt.Printf("     输出: %s\n", output)
			}

			return nil
		})

		if err != nil {
			fmt.Printf("   ⚠️  PTY %d 操作超时: %v\n", i+1, err)
		}
	}
}

func demonstrateMonitoringFeatures(server *terminal.EnhancedTerminalServer) {
	monitor := server.GetPerformanceMonitor()
	if monitor == nil {
		fmt.Println("   ⚠️  性能监控器不可用")
		return
	}

	// 等待监控器收集一些数据
	time.Sleep(time.Second * 2)

	metrics := monitor.GetMetrics()
	summary := monitor.GetSummary()

	fmt.Printf("   ✓ CPU使用率: %.1f%%\n", metrics.CPUUsage)
	fmt.Printf("   ✓ 内存使用: %.1fMB\n", float64(metrics.MemoryUsage)/(1024*1024))
	fmt.Printf("   ✓ Goroutine数量: %d\n", metrics.GoroutineCount)
	fmt.Printf("   ✓ 活动会话: %d\n", metrics.ActiveSessions)
	fmt.Printf("   ✓ 总面板数: %d\n", metrics.TotalPanes)
	fmt.Printf("   ✓ 运行时间: %v\n", metrics.Uptime)
	fmt.Printf("   ✓ 整体状态: %s\n", summary["status"])
}

func demonstrateServerStats(server *terminal.EnhancedTerminalServer) {
	stats := server.GetStats()

	fmt.Printf("   ✓ 服务器启动时间: %s\n", stats.StartTime.Format("15:04:05"))
	fmt.Printf("   ✓ 总连接数: %d\n", stats.TotalConnections)
	fmt.Printf("   ✓ 当前客户端: %d\n", stats.ActiveClients)
	fmt.Printf("   ✓ 处理命令数: %d\n", stats.CommandsHandled)
	fmt.Printf("   ✓ 错误次数: %d\n", stats.ErrorCount)

	if stats.PerformanceData != nil {
		fmt.Printf("   ✓ 性能数据已集成\n")
	}
}

func demonstratePerformanceMonitoring() {
	fmt.Println("1. 创建独立的性能监控器...")

	config := &terminal.TerminalConfig{
		ClixGoIntegration: true,
		NetworkMonitor:    true,
	}

	sm := terminal.NewSessionManager(config)
	monitor := terminal.NewPerformanceMonitor(config, sm)

	fmt.Println("2. 启动性能监控...")
	if err := withTimeout("监控器启动", QuickTimeout, func() error {
		return monitor.Start()
	}); err != nil {
		log.Printf("启动监控器失败: %v", err)
		return
	}
	defer monitor.Stop()

	fmt.Println("3. 创建测试负载...")
	// 创建一些会话来生成监控数据
	for i := 0; i < 3; i++ {
		sessionName := fmt.Sprintf("monitor-test-%d", i+1)
		session, err := sm.CreateSession(sessionName)
		if err != nil {
			continue
		}

		// 为每个会话创建多个窗口和面板
		for j := 0; j < 2; j++ {
			windowName := fmt.Sprintf("window-%d", j+1)
			sm.CreateWindow(session.ID, windowName)
			sm.SplitPane(session.ID, j, "vertical")
		}
	}

	fmt.Println("4. 收集监控数据...")
	time.Sleep(time.Second * 3)

	fmt.Println("5. 分析性能指标...")
	err := withTimeout("性能指标分析", QuickTimeout, func() error {
		metrics := monitor.GetMetrics()

		fmt.Printf("   📊 系统资源:\n")
		fmt.Printf("     - CPU: %.1f%%\n", metrics.CPUUsage)
		fmt.Printf("     - 内存: %.1fMB\n", float64(metrics.MemoryUsage)/(1024*1024))
		fmt.Printf("     - Goroutines: %d\n", metrics.GoroutineCount)

		fmt.Printf("   📊 终端统计:\n")
		fmt.Printf("     - 活动会话: %d\n", metrics.ActiveSessions)
		fmt.Printf("     - 总窗口数: %d\n", metrics.TotalWindows)
		fmt.Printf("     - 总面板数: %d\n", metrics.TotalPanes)
		fmt.Printf("     - 活动PTY: %d\n", metrics.ActivePTYs)

		fmt.Printf("   📊 时间信息:\n")
		fmt.Printf("     - 运行时间: %v\n", metrics.Uptime)
		fmt.Printf("     - 最后更新: %s\n", metrics.Timestamp.Format("15:04:05"))

		return nil
	})

	if err != nil {
		fmt.Printf("   ⚠️  性能指标分析超时: %v\n", err)
	}

	// 测试警报系统
	fmt.Println("6. 测试警报系统...")
	if err := withTimeout("警报系统测试", QuickTimeout, func() error {
		testAlerts(monitor)
		return nil
	}); err != nil {
		fmt.Printf("   ⚠️  警报系统测试超时: %v\n", err)
	}

	fmt.Println("   性能监控演示完成！")
}

func testAlerts(monitor *terminal.PerformanceMonitor) {
	// 设置较低的阈值来触发警报
	lowThresholds := &terminal.AlertThresholds{
		MaxCPUUsage:     1.0,  // 1% CPU
		MaxMemoryUsage:  1024, // 1KB 内存
		MaxGoroutines:   10,   // 10个goroutines
		MaxResponseTime: time.Millisecond,
		MaxErrorRate:    0.1,
	}

	monitor.SetAlertThresholds(lowThresholds)

	fmt.Println("   ⚠️  设置低阈值以触发警报...")
	time.Sleep(time.Second * 2)

	// 恢复正常阈值
	monitor.SetAlertThresholds(terminal.DefaultAlertThresholds)
	fmt.Println("   ✓ 警报系统测试完成，阈值已恢复")
}

func demonstrateRealWorldScenarios() {
	fmt.Println("1. 开发环境场景...")
	if err := withTimeout("开发环境场景", QuickTimeout, func() error {
		developmentScenario()
		return nil
	}); err != nil {
		fmt.Printf("   ⚠️  开发环境场景超时: %v\n", err)
	}

	fmt.Println("2. 服务器管理场景...")
	if err := withTimeout("服务器管理场景", DefaultTimeout, func() error {
		serverManagementScenario()
		return nil
	}); err != nil {
		fmt.Printf("   ⚠️  服务器管理场景超时: %v\n", err)
	}

	fmt.Println("3. 多项目管理场景...")
	if err := withTimeout("多项目管理场景", QuickTimeout, func() error {
		multiProjectScenario()
		return nil
	}); err != nil {
		fmt.Printf("   ⚠️  多项目管理场景超时: %v\n", err)
	}
}

func developmentScenario() {
	fmt.Println("   场景：前端开发环境设置")

	config := &terminal.TerminalConfig{
		ClixGoIntegration: true,
		AutoSave:          true,
		SaveInterval:      time.Minute * 3,
		BufferSize:        5000,
	}

	sm := terminal.NewSessionManager(config)
	session, err := sm.CreateSession("frontend-dev")
	if err != nil {
		log.Printf("创建开发会话失败: %v", err)
		return
	}

	// 创建开发所需的窗口
	windows := []string{"editor", "server", "tests", "logs"}
	for _, windowName := range windows {
		window, err := sm.CreateWindow(session.ID, windowName)
		if err != nil {
			continue
		}
		fmt.Printf("   ✓ 创建 %s 窗口\n", window.Name)

		// 为某些窗口分割面板
		if windowName == "server" || windowName == "logs" {
			sm.SplitPane(session.ID, window.Index, "horizontal")
			fmt.Printf("     - 分割面板用于多任务\n")
		}
	}

	fmt.Printf("   ✓ 开发环境设置完成 (会话: %s, 窗口数: %d)\n",
		session.Name, len(session.Windows))
}

func serverManagementScenario() {
	fmt.Println("   场景：服务器管理和监控")

	config := &terminal.TerminalConfig{
		ClixGoIntegration: true,
		NetworkMonitor:    true,
		TaskIntegration:   true,
		AutoSave:          true,
	}

	server := terminal.NewEnhancedTerminalServer(config)
	if err := server.Start(); err != nil {
		log.Printf("启动服务器失败: %v", err)
		return
	}
	defer server.Stop()

	sm := server.GetSessionManager()
	session, err := sm.CreateSession("server-mgmt")
	if err != nil {
		log.Printf("创建服务器管理会话失败: %v", err)
		return
	}

	// 创建服务器管理相关窗口
	managementTasks := []string{"monitoring", "logs", "deployment", "backup"}
	for _, task := range managementTasks {
		window, err := sm.CreateWindow(session.ID, task)
		if err != nil {
			continue
		}
		fmt.Printf("   ✓ 创建 %s 管理窗口\n", window.Name)
	}

	// 启动性能监控
	monitor := server.GetPerformanceMonitor()
	time.Sleep(time.Second)

	metrics := monitor.GetMetrics()
	fmt.Printf("   📊 当前系统状态:\n")
	fmt.Printf("     - 内存使用: %.1fMB\n", float64(metrics.MemoryUsage)/(1024*1024))
	fmt.Printf("     - 活动会话: %d\n", metrics.ActiveSessions)
	fmt.Printf("     - 管理窗口: %d\n", len(session.Windows))
}

func multiProjectScenario() {
	fmt.Println("   场景：多项目并行开发")

	config := &terminal.TerminalConfig{
		ClixGoIntegration: true,
		AutoSave:          true,
		SaveInterval:      time.Minute * 5,
	}

	sm := terminal.NewSessionManager(config)

	// 创建多个项目会话
	projects := []string{"web-app", "api-service", "mobile-app", "data-pipeline"}

	for _, projectName := range projects {
		session, err := sm.CreateSession(projectName)
		if err != nil {
			continue
		}

		// 为每个项目创建标准化窗口
		standardWindows := []string{"code", "terminal", "tests"}
		for _, windowName := range standardWindows {
			sm.CreateWindow(session.ID, windowName)
		}

		fmt.Printf("   ✓ 项目 %s 环境设置完成\n", projectName)
	}

	sessions := sm.ListSessions()
	fmt.Printf("   📝 多项目管理总结:\n")
	fmt.Printf("     - 总项目数: %d\n", len(sessions))

	totalWindows := 0
	totalPanes := 0
	for _, session := range sessions {
		totalWindows += len(session.Windows)
		for _, window := range session.Windows {
			totalPanes += len(window.Panes)
		}
	}

	fmt.Printf("     - 总窗口数: %d\n", totalWindows)
	fmt.Printf("     - 总面板数: %d\n", totalPanes)
}

func runInteractiveDemo() {
	fmt.Println("🎮 === ClixGo 交互式演示 ===")
	fmt.Println("输入命令进行交互，输入 'help' 查看可用命令，输入 'quit' 退出")

	// 初始化系统
	logger.InitLogger()

	config := &terminal.TerminalConfig{
		ClixGoIntegration: true,
		NetworkMonitor:    true,
		AutoSave:          true,
		SaveInterval:      time.Minute * 5,
	}

	var server *terminal.EnhancedTerminalServer
	err := withTimeout("服务器启动", DefaultTimeout, func() error {
		server = terminal.NewEnhancedTerminalServer(config)
		return server.Start()
	})

	if err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
	defer server.Stop()

	fmt.Printf("服务器已启动: %s\n", server.GetSocketPath())

	// 交互循环
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("ClixGo> ")
		if !scanner.Scan() {
			break
		}

		command := strings.TrimSpace(scanner.Text())
		if command == "" {
			continue
		}

		// 带超时的命令处理
		shouldExit := false
		err := withTimeout("命令处理", DefaultTimeout, func() error {
			shouldExit = handleInteractiveCommand(command, server)
			return nil
		})

		if err != nil {
			fmt.Printf("命令处理超时: %v\n", err)
		}

		if shouldExit {
			break
		}
	}

	fmt.Println("交互式演示结束！")
}

func handleInteractiveCommand(command string, server *terminal.EnhancedTerminalServer) bool {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "help", "h":
		showInteractiveHelp()
	case "quit", "exit", "q":
		return true
	case "status":
		showServerStatus(server)
	case "sessions":
		showSessions(server)
	case "create":
		createSession(server, args)
	case "monitor":
		showMonitoringInfo(server)
	case "stats":
		showDetailedStats(server)
	case "test":
		runQuickTest(server)
	default:
		fmt.Printf("未知命令: %s (输入 'help' 查看帮助)\n", cmd)
	}

	return false
}

func showInteractiveHelp() {
	fmt.Println("可用命令:")
	fmt.Println("  help, h       - 显示此帮助信息")
	fmt.Println("  status        - 显示服务器状态")
	fmt.Println("  sessions      - 列出所有会话")
	fmt.Println("  create <name> - 创建新会话")
	fmt.Println("  monitor       - 显示性能监控信息")
	fmt.Println("  stats         - 显示详细统计信息")
	fmt.Println("  test          - 运行快速测试")
	fmt.Println("  quit, exit, q - 退出程序")
}

func showServerStatus(server *terminal.EnhancedTerminalServer) {
	fmt.Printf("服务器状态: %s\n", map[bool]string{true: "运行中", false: "已停止"}[server.IsRunning()])
	fmt.Printf("Socket路径: %s\n", server.GetSocketPath())

	if monitor := server.GetPerformanceMonitor(); monitor != nil {
		summary := monitor.GetSummary()
		fmt.Printf("整体状态: %s\n", summary["status"])
		fmt.Printf("运行时间: %s\n", summary["uptime"])
	}
}

func showSessions(server *terminal.EnhancedTerminalServer) {
	sessions := server.GetSessionManager().ListSessions()
	fmt.Printf("当前会话数: %d\n", len(sessions))

	for i, session := range sessions {
		fmt.Printf("  %d. %s (ID: %s, 窗口: %d)\n",
			i+1, session.Name, session.ID[:8], len(session.Windows))
	}
}

func createSession(server *terminal.EnhancedTerminalServer, args []string) {
	name := "interactive-session"
	if len(args) > 0 {
		name = args[0]
	}

	session, err := server.GetSessionManager().CreateSession(name)
	if err != nil {
		fmt.Printf("创建会话失败: %v\n", err)
		return
	}

	fmt.Printf("会话 '%s' 创建成功 (ID: %s)\n", session.Name, session.ID[:8])
}

func showMonitoringInfo(server *terminal.EnhancedTerminalServer) {
	monitor := server.GetPerformanceMonitor()
	if monitor == nil {
		fmt.Println("性能监控器不可用")
		return
	}

	metrics := monitor.GetMetrics()
	fmt.Println("性能监控信息:")
	fmt.Printf("  CPU使用率: %.1f%%\n", metrics.CPUUsage)
	fmt.Printf("  内存使用: %.1fMB\n", float64(metrics.MemoryUsage)/(1024*1024))
	fmt.Printf("  Goroutine数: %d\n", metrics.GoroutineCount)
	fmt.Printf("  活动会话: %d\n", metrics.ActiveSessions)
}

func showDetailedStats(server *terminal.EnhancedTerminalServer) {
	stats := server.GetStats()
	fmt.Println("详细统计信息:")
	fmt.Printf("  启动时间: %s\n", stats.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("  总连接数: %d\n", stats.TotalConnections)
	fmt.Printf("  当前客户端: %d\n", stats.ActiveClients)
	fmt.Printf("  处理命令数: %d\n", stats.CommandsHandled)
	fmt.Printf("  错误次数: %d\n", stats.ErrorCount)
	fmt.Printf("  运行时长: %v\n", time.Since(stats.StartTime))
}

func runQuickTest(server *terminal.EnhancedTerminalServer) {
	fmt.Println("运行快速测试...")

	// 创建测试会话
	sm := server.GetSessionManager()
	session, err := sm.CreateSession("quick-test")
	if err != nil {
		fmt.Printf("测试失败: %v\n", err)
		return
	}

	// 创建窗口和面板
	window, err := sm.CreateWindow(session.ID, "test-window")
	if err != nil {
		fmt.Printf("创建窗口失败: %v\n", err)
		return
	}

	pane, err := sm.SplitPane(session.ID, 0, "vertical")
	if err != nil {
		fmt.Printf("分割面板失败: %v\n", err)
		return
	}

	fmt.Printf("✓ 测试完成 - 会话: %s, 窗口: %s, 面板: %s\n",
		session.Name, window.Name, pane.ID[:8])
}

func showNextSteps() {
	fmt.Println("\n🎯 下一步操作建议:")
	fmt.Println("1. 运行交互式演示: go run complete_demo.go interactive")
	fmt.Println("2. 查看增强功能: go run complete_demo.go enhanced")
	fmt.Println("3. 集成到实际项目中，使用 ClixGo 终端多路复用器")
	fmt.Println("4. 自定义配置文件，优化使用体验")
	fmt.Println("5. 探索性能监控和警报功能")
	fmt.Println()
	fmt.Println("📖 参考文档: ClixGo/README.md")
	fmt.Println("🛠️  配置示例: ClixGo/examples/terminal/config.yaml")
	fmt.Println("📊 开发总结: ClixGo/DEVELOPMENT_SUMMARY.md")
}
