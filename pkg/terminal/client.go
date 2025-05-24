package terminal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"go.uber.org/zap"
)

// TerminalClient 终端客户端
type TerminalClient struct {
	conn      net.Conn
	sessionID string
	config    *TerminalConfig
	running   bool
	keyMode   bool // 是否在快捷键模式
}

// NewTerminalClient 创建终端客户端
func NewTerminalClient(config *TerminalConfig) *TerminalClient {
	if config == nil {
		config = DefaultConfig
	}

	return &TerminalClient{
		config:  config,
		running: false,
		keyMode: false,
	}
}

// Connect 连接到服务器
func (tc *TerminalClient) Connect() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp"
	}
	socketPath := fmt.Sprintf("%s/.clixgo/terminal/clixgo-terminal.sock", homeDir)

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("连接服务器失败: %v", err)
	}

	tc.conn = conn
	return nil
}

// Disconnect 断开连接
func (tc *TerminalClient) Disconnect() error {
	tc.running = false
	if tc.conn != nil {
		return tc.conn.Close()
	}
	return nil
}

// CreateSession 创建新会话
func (tc *TerminalClient) CreateSession(name string) error {
	response, err := tc.sendCommand(Command{
		Type: CmdCreateSession,
		Payload: map[string]interface{}{
			"name": name,
		},
	})
	if err != nil {
		return err
	}

	if errMsg, ok := response["error"].(string); ok {
		return fmt.Errorf(errMsg)
	}

	sessionID, _ := response["session_id"].(string)
	tc.sessionID = sessionID

	logger.Info("Session created", zap.String("session_id", sessionID))
	return nil
}

// AttachSession 连接到会话
func (tc *TerminalClient) AttachSession(sessionIdentifier string) error {
	// 首先尝试按ID连接
	response, err := tc.sendCommand(Command{
		Type: CmdAttachSession,
		Payload: map[string]interface{}{
			"session_id": sessionIdentifier,
		},
	})

	// 如果按ID连接失败，尝试按名称连接
	if err != nil || response["error"] != nil {
		response, err = tc.sendCommand(Command{
			Type: CmdAttachSession,
			Payload: map[string]interface{}{
				"session_name": sessionIdentifier,
			},
		})
	}

	if err != nil {
		return err
	}

	if errMsg, ok := response["error"].(string); ok {
		return fmt.Errorf(errMsg)
	}

	session, ok := response["session"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid session data")
	}

	tc.sessionID = session["id"].(string)
	logger.Info("Session attached", zap.String("session_id", tc.sessionID))

	return nil
}

// DetachSession 断开会话
func (tc *TerminalClient) DetachSession() error {
	if tc.sessionID == "" {
		return fmt.Errorf("no active session")
	}

	response, err := tc.sendCommand(Command{
		Type:    CmdDetachSession,
		Payload: nil,
	})
	if err != nil {
		return err
	}

	if errMsg, ok := response["error"].(string); ok {
		return fmt.Errorf(errMsg)
	}

	logger.Info("Session detached", zap.String("session_id", tc.sessionID))
	tc.sessionID = ""

	return nil
}

// StartInteractiveMode 启动交互模式
func (tc *TerminalClient) StartInteractiveMode() error {
	if tc.sessionID == "" {
		return fmt.Errorf("no active session")
	}

	tc.running = true

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 显示欢迎信息
	tc.showWelcome()

	// 启动输入处理
	go tc.handleInput()

	// 等待退出信号
	for tc.running {
		select {
		case sig := <-sigChan:
			logger.Info("Received signal", zap.String("signal", sig.String()))
			tc.running = false
		case <-time.After(time.Second):
			// 定期检查连接状态
			if tc.conn == nil {
				tc.running = false
			}
		}
	}

	return nil
}

// showWelcome 显示欢迎信息
func (tc *TerminalClient) showWelcome() {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│                   ClixGo Terminal                           │")
	fmt.Println("│              轻量级终端多路复用器                              │")
	fmt.Println("├─────────────────────────────────────────────────────────────┤")
	fmt.Printf("│ 会话ID: %-47s │\n", tc.sessionID[:8]+"...")
	fmt.Println("├─────────────────────────────────────────────────────────────┤")
	fmt.Println("│ 快捷键:                                                      │")
	fmt.Printf("│ %-59s │\n", tc.config.PrefixKey+" d    - 断开会话")
	fmt.Printf("│ %-59s │\n", tc.config.PrefixKey+" c    - 创建新窗口")
	fmt.Printf("│ %-59s │\n", tc.config.PrefixKey+" \"    - 水平分割面板")
	fmt.Printf("│ %-59s │\n", tc.config.PrefixKey+" %    - 垂直分割面板")
	fmt.Printf("│ %-59s │\n", tc.config.PrefixKey+" o    - 切换面板")
	fmt.Printf("│ %-59s │\n", tc.config.PrefixKey+" x    - 关闭面板")
	fmt.Printf("│ %-59s │\n", tc.config.PrefixKey+" ?    - 显示帮助")
	fmt.Println("│ Ctrl+C        - 退出                                        │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
	fmt.Println("\n开始输入命令或使用快捷键...")
}

// handleInput 处理用户输入
func (tc *TerminalClient) handleInput() {
	scanner := bufio.NewScanner(os.Stdin)

	for tc.running && scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			continue
		}

		// 检查是否是快捷键
		if tc.handleShortcut(input) {
			continue
		}

		// 检查是否是内置命令
		if tc.handleBuiltinCommand(input) {
			continue
		}

		// 发送按键到活动面板
		tc.sendKeys(input)
	}
}

// handleShortcut 处理快捷键
func (tc *TerminalClient) handleShortcut(input string) bool {
	// 简化处理，只检查以 Ctrl+B 开头的命令
	if !strings.HasPrefix(input, "C-b") && !strings.HasPrefix(input, "Ctrl+B") {
		return false
	}

	// 解析快捷键
	parts := strings.Fields(input)
	if len(parts) < 2 {
		return false
	}

	key := parts[1]

	switch key {
	case "d":
		tc.handleDetach()
	case "c":
		tc.handleCreateWindow()
	case "\"":
		tc.handleSplitPane("horizontal")
	case "%":
		tc.handleSplitPane("vertical")
	case "o":
		tc.handleSwitchPane()
	case "x":
		tc.handleClosePane()
	case "?":
		tc.handleShowHelp()
	default:
		fmt.Printf("未知快捷键: %s\n", key)
		return false
	}

	return true
}

// handleBuiltinCommand 处理内置命令
func (tc *TerminalClient) handleBuiltinCommand(input string) bool {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false
	}

	command := parts[0]

	switch command {
	case "exit", "quit":
		tc.running = false
		return true
	case "detach":
		tc.handleDetach()
		return true
	case "help":
		tc.handleShowHelp()
		return true
	case "status":
		tc.handleShowStatus()
		return true
	case "clear":
		fmt.Print("\033[2J\033[H") // 清屏
		return true
	default:
		return false
	}
}

// handleDetach 处理断开会话
func (tc *TerminalClient) handleDetach() {
	fmt.Println("正在断开会话...")
	if err := tc.DetachSession(); err != nil {
		fmt.Printf("断开失败: %v\n", err)
	} else {
		fmt.Println("会话已断开")
		tc.running = false
	}
}

// handleCreateWindow 处理创建窗口
func (tc *TerminalClient) handleCreateWindow() {
	fmt.Print("输入新窗口名称 (回车使用默认): ")
	scanner := bufio.NewScanner(os.Stdin)
	var name string
	if scanner.Scan() {
		name = strings.TrimSpace(scanner.Text())
	}

	response, err := tc.sendCommand(Command{
		Type: CmdCreateWindow,
		Payload: map[string]interface{}{
			"name": name,
		},
	})
	if err != nil {
		fmt.Printf("创建窗口失败: %v\n", err)
		return
	}

	if errMsg, ok := response["error"].(string); ok {
		fmt.Printf("创建窗口失败: %s\n", errMsg)
		return
	}

	fmt.Println("新窗口创建成功")
}

// handleSplitPane 处理分割面板
func (tc *TerminalClient) handleSplitPane(direction string) {
	response, err := tc.sendCommand(Command{
		Type: CmdSplitPane,
		Payload: map[string]interface{}{
			"window_index": 0, // 当前窗口
			"direction":    direction,
		},
	})
	if err != nil {
		fmt.Printf("分割面板失败: %v\n", err)
		return
	}

	if errMsg, ok := response["error"].(string); ok {
		fmt.Printf("分割面板失败: %s\n", errMsg)
		return
	}

	fmt.Printf("面板已分割 (%s)\n", direction)
}

// handleSwitchPane 处理切换面板
func (tc *TerminalClient) handleSwitchPane() {
	fmt.Print("输入面板索引: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return
	}

	var paneIndex int
	if _, err := fmt.Sscanf(scanner.Text(), "%d", &paneIndex); err != nil {
		fmt.Printf("无效的面板索引: %v\n", err)
		return
	}

	response, err := tc.sendCommand(Command{
		Type: CmdSwitchPane,
		Payload: map[string]interface{}{
			"window_index": 0, // 当前窗口
			"pane_index":   paneIndex,
		},
	})
	if err != nil {
		fmt.Printf("切换面板失败: %v\n", err)
		return
	}

	if errMsg, ok := response["error"].(string); ok {
		fmt.Printf("切换面板失败: %s\n", errMsg)
		return
	}

	fmt.Printf("已切换到面板 %d\n", paneIndex)
}

// handleClosePane 处理关闭面板
func (tc *TerminalClient) handleClosePane() {
	fmt.Print("输入要关闭的面板索引: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return
	}

	var paneIndex int
	if _, err := fmt.Sscanf(scanner.Text(), "%d", &paneIndex); err != nil {
		fmt.Printf("无效的面板索引: %v\n", err)
		return
	}

	response, err := tc.sendCommand(Command{
		Type: CmdClosePane,
		Payload: map[string]interface{}{
			"window_index": 0, // 当前窗口
			"pane_index":   paneIndex,
		},
	})
	if err != nil {
		fmt.Printf("关闭面板失败: %v\n", err)
		return
	}

	if errMsg, ok := response["error"].(string); ok {
		fmt.Printf("关闭面板失败: %s\n", errMsg)
		return
	}

	fmt.Printf("面板 %d 已关闭\n", paneIndex)
}

// handleShowHelp 显示帮助信息
func (tc *TerminalClient) handleShowHelp() {
	fmt.Println("\n=== ClixGo Terminal 帮助 ===")
	fmt.Println("\n快捷键:")
	for _, binding := range tc.config.KeyBindings {
		fmt.Printf("  %-12s - %s\n", binding.Key, binding.Command)
	}

	fmt.Println("\n内置命令:")
	fmt.Println("  help         - 显示此帮助信息")
	fmt.Println("  status       - 显示会话状态")
	fmt.Println("  detach       - 断开当前会话")
	fmt.Println("  clear        - 清屏")
	fmt.Println("  exit/quit    - 退出客户端")

	fmt.Println("\nClixGo 集成命令:")
	fmt.Println("  network ping <host>    - 网络诊断")
	fmt.Println("  text count <file>      - 文本统计")
	fmt.Println("  task list              - 任务管理")
	fmt.Println()
}

// handleShowStatus 显示状态信息
func (tc *TerminalClient) handleShowStatus() {
	response, err := tc.sendCommand(Command{
		Type:    CmdListSessions,
		Payload: nil,
	})
	if err != nil {
		fmt.Printf("获取状态失败: %v\n", err)
		return
	}

	if errMsg, ok := response["error"].(string); ok {
		fmt.Printf("获取状态失败: %s\n", errMsg)
		return
	}

	sessions, ok := response["sessions"].([]interface{})
	if !ok {
		fmt.Println("无效的响应格式")
		return
	}

	fmt.Printf("\n=== 会话状态 ===\n")
	fmt.Printf("活动会话数: %d\n", len(sessions))
	fmt.Printf("当前会话ID: %s\n", tc.sessionID)
	fmt.Printf("客户端配置: %s\n", tc.config.Theme)
	fmt.Println()
}

// sendKeys 发送按键到活动面板
func (tc *TerminalClient) sendKeys(keys string) {
	response, err := tc.sendCommand(Command{
		Type: CmdSendKeys,
		Payload: map[string]interface{}{
			"keys": keys,
		},
	})
	if err != nil {
		fmt.Printf("发送按键失败: %v\n", err)
		return
	}

	if errMsg, ok := response["error"].(string); ok {
		fmt.Printf("发送按键失败: %s\n", errMsg)
		return
	}

	// 模拟命令输出（实际应该从面板获取）
	fmt.Printf("$ %s\n", keys)
	fmt.Printf("命令已发送到活动面板\n")
}

// sendCommand 发送命令到服务器
func (tc *TerminalClient) sendCommand(cmd Command) (map[string]interface{}, error) {
	if tc.conn == nil {
		return nil, fmt.Errorf("not connected to server")
	}

	encoder := json.NewEncoder(tc.conn)
	decoder := json.NewDecoder(tc.conn)

	if err := encoder.Encode(cmd); err != nil {
		return nil, fmt.Errorf("发送命令失败: %v", err)
	}

	var response map[string]interface{}
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("接收响应失败: %v", err)
	}

	return response, nil
}
