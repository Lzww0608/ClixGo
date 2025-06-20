/*
* @Author: Lzww0608
* @Date: 2025-06-18 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-18 10:00:00
* @Description: TUI-PTY集成层 - 整合UIManager和SessionManager，实现终端复用TUI界面
 */

package terminal

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/terminal/ui"
	"github.com/gdamore/tcell/v2"
	"go.uber.org/zap"
)

// TerminalTUI TUI-PTY集成管理器
type TerminalTUI struct {
	// 核心组件
	uiManager      *ui.UIManager
	sessionManager *SessionManager
	ptyManager     *CreackPTYManager

	// 面板-PTY映射
	panelPTYMap map[string]*CreackPTY // panelID -> PTY
	ptyPanelMap map[string]string     // ptyID -> panelID
	sessionMap  map[string]*Session   // sessionID -> Session

	// 控制和状态
	ctx             context.Context
	cancel          context.CancelFunc
	mutex           sync.RWMutex
	running         bool
	config          *TerminalConfig
	activeSessionID string

	// 键盘处理
	prefixKey     tcell.Key
	prefixPressed bool
}

// TerminalTUIConfig TUI配置
type TerminalTUIConfig struct {
	UIConfig       ui.UIConfig
	TerminalConfig *TerminalConfig
	PrefixKey      string // 默认 "C-b"
	AutoResize     bool
	RefreshRate    time.Duration
}

// NewTerminalTUI 创建TUI-PTY集成管理器
func NewTerminalTUI(config *TerminalTUIConfig) (*TerminalTUI, error) {
	if config == nil {
		config = DefaultTerminalTUIConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建UI管理器
	uiManager, err := ui.NewUIManager(config.UIConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("创建UI管理器失败: %w", err)
	}

	// 创建会话管理器
	sessionManager := NewSessionManager(config.TerminalConfig)

	// 创建PTY管理器
	ptyManager := NewCreackPTYManager(config.TerminalConfig)

	tui := &TerminalTUI{
		uiManager:      uiManager,
		sessionManager: sessionManager,
		ptyManager:     ptyManager,
		panelPTYMap:    make(map[string]*CreackPTY),
		ptyPanelMap:    make(map[string]string),
		sessionMap:     make(map[string]*Session),
		ctx:            ctx,
		cancel:         cancel,
		config:         config.TerminalConfig,
		prefixKey:      parsePrefixKey(config.PrefixKey),
	}

	// 设置自定义输入处理
	tui.setupInputCapture()

	logger.Info("TUI-PTY集成管理器创建成功",
		zap.String("prefix_key", config.PrefixKey),
		zap.Bool("auto_resize", config.AutoResize))

	return tui, nil
}

// DefaultTerminalTUIConfig 默认TUI配置
func DefaultTerminalTUIConfig() *TerminalTUIConfig {
	return &TerminalTUIConfig{
		UIConfig:       ui.DefaultUIConfig,
		TerminalConfig: DefaultConfig,
		PrefixKey:      "C-b",
		AutoResize:     true,
		RefreshRate:    time.Millisecond * 50, // 20 FPS
	}
}

// Start 启动TUI界面
func (tui *TerminalTUI) Start() error {
	tui.mutex.Lock()
	tui.running = true
	tui.mutex.Unlock()

	logger.Info("启动TUI-PTY集成终端界面")

	// 创建默认会话
	if err := tui.createDefaultSession(); err != nil {
		return fmt.Errorf("创建默认会话失败: %w", err)
	}

	// 启动输出监控
	go tui.outputMonitor()

	// 启动状态更新
	go tui.statusUpdater()

	// 启动UI（阻塞）
	return tui.uiManager.Start()
}

// Stop 停止TUI界面
func (tui *TerminalTUI) Stop() {
	tui.mutex.Lock()
	defer tui.mutex.Unlock()

	if !tui.running {
		return
	}

	tui.running = false
	tui.cancel()

	logger.Info("停止TUI-PTY集成终端界面")

	// 清理所有PTY
	for ptyID, pty := range tui.panelPTYMap {
		if err := pty.Close(); err != nil {
			logger.Warn("关闭PTY失败", zap.String("pty_id", ptyID), zap.Error(err))
		}
	}

	// 停止UI管理器
	tui.uiManager.Stop()
}

// CreateSession 创建新会话
func (tui *TerminalTUI) CreateSession(name string) (*Session, error) {
	session, err := tui.sessionManager.CreateSession(name)
	if err != nil {
		return nil, err
	}

	tui.mutex.Lock()
	tui.sessionMap[session.ID] = session
	tui.activeSessionID = session.ID
	tui.mutex.Unlock()

	// 为会话的第一个窗口创建面板
	if len(session.Windows) > 0 {
		window := session.Windows[0]
		if len(window.Panes) > 0 {
			pane := window.Panes[0]
			if err := tui.createPanelForPane(session, window, pane); err != nil {
				logger.Error("为面板创建UI失败", zap.Error(err))
			}
		}
	}

	logger.Info("创建TUI会话",
		zap.String("session_id", session.ID),
		zap.String("session_name", session.Name))

	return session, nil
}

// createPanelForPane 为会话面板创建UI面板
func (tui *TerminalTUI) createPanelForPane(session *Session, window *Window, pane *Pane) error {
	panelID := fmt.Sprintf("panel_%s_%s_%s", session.ID[:8], window.ID[:8], pane.ID[:8])
	panelTitle := fmt.Sprintf("%s:%s [%s]", session.Name, window.Name, pane.Command)

	// 创建UI面板
	panel := tui.uiManager.CreatePanel(panelID, panelTitle)
	if panel == nil {
		return fmt.Errorf("创建UI面板失败")
	}

	// 创建PTY
	pty, err := tui.ptyManager.CreateCreackPTY(
		pane.ID,
		pane.Command,
		pane.WorkingDir,
		80, 24) // 初始大小
	if err != nil {
		return fmt.Errorf("创建PTY失败: %w", err)
	}

	// 启动PTY
	if err := pty.Start(); err != nil {
		tui.ptyManager.DestroyCreackPTY(pane.ID)
		return fmt.Errorf("启动PTY失败: %w", err)
	}

	// 建立映射关系
	tui.mutex.Lock()
	tui.panelPTYMap[panelID] = pty
	tui.ptyPanelMap[pane.ID] = panelID
	tui.mutex.Unlock()

	// 更新面板状态
	tui.uiManager.WriteToPanel(panelID, fmt.Sprintf("Terminal ready [PID: %d]\n", pty.GetPID()))

	logger.Info("为会话面板创建PTY绑定",
		zap.String("panel_id", panelID),
		zap.String("pty_id", pane.ID),
		zap.Int("pid", pty.GetPID()))

	return nil
}

// outputMonitor 监控PTY输出并更新UI面板
func (tui *TerminalTUI) outputMonitor() {
	ticker := time.NewTicker(10 * time.Millisecond) // 高频更新
	defer ticker.Stop()

	for {
		select {
		case <-tui.ctx.Done():
			return
		case <-ticker.C:
			tui.updatePanelOutputs()
		}
	}
}

// updatePanelOutputs 更新所有面板的输出
func (tui *TerminalTUI) updatePanelOutputs() {
	tui.mutex.RLock()
	panelPTYs := make(map[string]*CreackPTY)
	for panelID, pty := range tui.panelPTYMap {
		panelPTYs[panelID] = pty
	}
	tui.mutex.RUnlock()

	for panelID, pty := range panelPTYs {
		if !pty.IsRunning() {
			continue
		}

		// 读取PTY输出
		data, err := pty.ReadWithTimeout(1 * time.Millisecond)
		if err != nil {
			if !strings.Contains(err.Error(), "timeout") {
				logger.Debug("读取PTY输出失败",
					zap.String("panel_id", panelID),
					zap.Error(err))
			}
			continue
		}

		if len(data) > 0 {
			// 处理ANSI转义序列
			processedOutput := tui.processAnsiOutput(string(data))

			// 更新UI面板
			if err := tui.uiManager.WriteToPanel(panelID, processedOutput); err != nil {
				logger.Debug("更新面板输出失败",
					zap.String("panel_id", panelID),
					zap.Error(err))
			}
		}
	}
}

// processAnsiOutput 处理ANSI转义序列
func (tui *TerminalTUI) processAnsiOutput(output string) string {
	// 基本的ANSI转义序列处理
	// tview支持基本的颜色代码

	// 移除一些控制字符
	output = strings.ReplaceAll(output, "\r\n", "\n")
	output = strings.ReplaceAll(output, "\r", "\n")

	return output
}

// statusUpdater 更新状态栏
func (tui *TerminalTUI) statusUpdater() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-tui.ctx.Done():
			return
		case <-ticker.C:
			tui.updateStatusBar()
		}
	}
}

// updateStatusBar 更新状态栏信息
func (tui *TerminalTUI) updateStatusBar() {
	tui.mutex.RLock()
	sessionCount := len(tui.sessionMap)
	panelCount := len(tui.panelPTYMap)
	activeSession := tui.activeSessionID
	tui.mutex.RUnlock()

	// 获取活动会话名称
	sessionName := "none"
	if activeSession != "" {
		if session, exists := tui.sessionMap[activeSession]; exists {
			sessionName = session.Name
		}
	}

	// 获取性能统计
	stats := tui.sessionManager.GetPerformanceStats()

	statusLeft := fmt.Sprintf("ClixGo [%s]", sessionName)
	statusCenter := fmt.Sprintf("Sessions: %d | Panels: %d | Memory: %.1fMB",
		sessionCount, panelCount, stats.MemoryUsageMB)
	statusRight := time.Now().Format("15:04:05")

	logger.Debug("状态栏更新",
		zap.String("left", statusLeft),
		zap.String("center", statusCenter),
		zap.String("right", statusRight))
}

// setupInputCapture 设置自定义输入捕获
func (tui *TerminalTUI) setupInputCapture() {
	// 使用UIManager的自定义输入捕获接口
	tui.uiManager.SetCustomInputCapture(tui.handleKeyEvent)
	logger.Debug("设置TUI输入捕获")
}

// handleKeyEvent 处理键盘事件
func (tui *TerminalTUI) handleKeyEvent(event *tcell.EventKey) *tcell.EventKey {
	// 检查是否是prefix键
	if event.Key() == tui.prefixKey && !tui.prefixPressed {
		tui.prefixPressed = true
		logger.Debug("Prefix键按下", zap.String("key", event.Name()))
		return nil // 消费这个事件
	}

	// 如果prefix键被按下，处理命令
	if tui.prefixPressed {
		tui.prefixPressed = false
		return tui.handlePrefixCommand(event)
	}

	// 转发到活动的PTY
	return tui.forwardInputToPTY(event)
}

// handlePrefixCommand 处理prefix命令
func (tui *TerminalTUI) handlePrefixCommand(event *tcell.EventKey) *tcell.EventKey {
	switch event.Rune() {
	case 'c': // 创建新窗口
		tui.createNewPanel()
	case '"': // 水平分割
		tui.splitHorizontal()
	case '%': // 垂直分割
		tui.splitVertical()
	case 'o': // 切换面板
		tui.uiManager.NextPanel()
	case 'x': // 关闭面板
		tui.closeActivePanel()
	case 'd': // 分离会话
		tui.detachSession()
	case '?': // 显示帮助
		tui.showHelp()
	default:
		// 未知命令，忽略
		logger.Debug("未知prefix命令", zap.String("key", string(event.Rune())))
	}

	return nil // 消费事件
}

// forwardInputToPTY 转发输入到PTY
func (tui *TerminalTUI) forwardInputToPTY(event *tcell.EventKey) *tcell.EventKey {
	// 获取活动面板
	activePanelID := tui.uiManager.GetActivePanel()
	if activePanelID == "" {
		return event // 没有活动面板，传递给UI管理器
	}

	// 获取对应的PTY
	tui.mutex.RLock()
	pty, exists := tui.panelPTYMap[activePanelID]
	tui.mutex.RUnlock()

	if !exists || !pty.IsRunning() {
		return event // 没有PTY或PTY未运行，传递给UI管理器
	}

	// 转换键盘事件为字节
	data := tui.eventToBytes(event)
	if len(data) > 0 {
		if err := pty.Write(data); err != nil {
			logger.Error("向PTY写入数据失败",
				zap.String("panel_id", activePanelID),
				zap.Error(err))
		}
	}

	return nil // 消费事件
}

// eventToBytes 将键盘事件转换为字节
func (tui *TerminalTUI) eventToBytes(event *tcell.EventKey) []byte {
	switch event.Key() {
	case tcell.KeyEnter:
		return []byte{'\n'}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		return []byte{0x7f} // DEL
	case tcell.KeyTab:
		return []byte{'\t'}
	case tcell.KeyEscape:
		return []byte{0x1b}
	case tcell.KeyCtrlC:
		return []byte{0x03}
	case tcell.KeyUp:
		return []byte{0x1b, '[', 'A'}
	case tcell.KeyDown:
		return []byte{0x1b, '[', 'B'}
	case tcell.KeyRight:
		return []byte{0x1b, '[', 'C'}
	case tcell.KeyLeft:
		return []byte{0x1b, '[', 'D'}
	case tcell.KeyRune:
		// 普通字符
		return []byte(string(event.Rune()))
	}

	return nil
}

// createDefaultSession 创建默认会话
func (tui *TerminalTUI) createDefaultSession() error {
	session, err := tui.CreateSession("main")
	if err != nil {
		return err
	}

	logger.Info("创建默认TUI会话", zap.String("session_id", session.ID))
	return nil
}

// createNewPanel 创建新面板
func (tui *TerminalTUI) createNewPanel() {
	panelID := fmt.Sprintf("panel_%d", time.Now().UnixNano())
	panel := tui.uiManager.CreatePanel(panelID, "New Terminal")

	if panel != nil {
		// 创建新的PTY
		pty, err := tui.ptyManager.CreateCreackPTY(
			panelID, "", "", 80, 24)
		if err != nil {
			logger.Error("创建新PTY失败", zap.Error(err))
			return
		}

		if err := pty.Start(); err != nil {
			logger.Error("启动新PTY失败", zap.Error(err))
			tui.ptyManager.DestroyCreackPTY(panelID)
			return
		}

		// 建立映射
		tui.mutex.Lock()
		tui.panelPTYMap[panelID] = pty
		tui.ptyPanelMap[panelID] = panelID
		tui.mutex.Unlock()

		logger.Info("创建新终端面板", zap.String("panel_id", panelID))
	}
}

// splitHorizontal 水平分割
func (tui *TerminalTUI) splitHorizontal() {
	tui.createNewPanel()
	logger.Info("水平分割面板")
}

// splitVertical 垂直分割
func (tui *TerminalTUI) splitVertical() {
	tui.createNewPanel()
	logger.Info("垂直分割面板")
}

// closeActivePanel 关闭活动面板
func (tui *TerminalTUI) closeActivePanel() {
	activePanelID := tui.uiManager.GetActivePanel()
	if activePanelID == "" {
		return
	}

	// 关闭对应的PTY
	tui.mutex.Lock()
	if pty, exists := tui.panelPTYMap[activePanelID]; exists {
		pty.Close()
		delete(tui.panelPTYMap, activePanelID)
		delete(tui.ptyPanelMap, activePanelID)
	}
	tui.mutex.Unlock()

	// 关闭UI面板
	tui.uiManager.CloseActivePanel()
	logger.Info("关闭活动面板", zap.String("panel_id", activePanelID))
}

// detachSession 分离会话
func (tui *TerminalTUI) detachSession() {
	tui.Stop()
	logger.Info("分离TUI会话")
}

// showHelp 显示帮助
func (tui *TerminalTUI) showHelp() {
	tui.uiManager.ShowHelp()
}

// parsePrefixKey 解析prefix键
func parsePrefixKey(prefix string) tcell.Key {
	switch prefix {
	case "C-a":
		return tcell.KeyCtrlA
	case "C-b":
		return tcell.KeyCtrlB
	case "C-x":
		return tcell.KeyCtrlX
	default:
		return tcell.KeyCtrlB // 默认
	}
}
