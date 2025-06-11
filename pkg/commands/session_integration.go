/*
* @Author: Lzww0608
* @Date: 2025-6-11 11:15:01
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-11 11:15:04
* @Description: 会话集成层 - 连接tmux兼容命令与现有terminal.SessionManager
 */

package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

// SessionIntegrationLayer 会话集成层
type SessionIntegrationLayer struct {
	sessionManager *terminal.SessionManager
	logger         Logger
}

// NewSessionIntegrationLayer 创建会话集成层
func NewSessionIntegrationLayer(sessionManager *terminal.SessionManager, logger Logger) *SessionIntegrationLayer {
	return &SessionIntegrationLayer{
		sessionManager: sessionManager,
		logger:         logger,
	}
}

// TmuxCommandAdapter tmux命令适配器接口
type TmuxCommandAdapter struct {
	integration *SessionIntegrationLayer
	logger      Logger
}

// NewTmuxCommandAdapter 创建tmux命令适配器
func NewTmuxCommandAdapter(integration *SessionIntegrationLayer, logger Logger) *TmuxCommandAdapter {
	return &TmuxCommandAdapter{
		integration: integration,
		logger:      logger,
	}
}

// 实现Command接口的tmux命令

// TmuxNewSessionCommand tmux new-session命令
type TmuxNewSessionCommand struct {
	adapter *TmuxCommandAdapter
}

func (cmd *TmuxNewSessionCommand) Execute(ctx *Context, args *Arguments) error {
	return cmd.adapter.ExecuteNewSession(ctx, args)
}

func (cmd *TmuxNewSessionCommand) Validate(args *Arguments) error {
	return nil // 基础验证，可根据需要扩展
}

func (cmd *TmuxNewSessionCommand) Usage() string {
	return "new-session [-d] [-s session-name] [shell-command]"
}

func (cmd *TmuxNewSessionCommand) Name() string {
	return "new-session"
}

func (cmd *TmuxNewSessionCommand) Description() string {
	return "Create a new session"
}

func (cmd *TmuxNewSessionCommand) ArgumentSpecs() []ArgumentSpec {
	return []ArgumentSpec{
		{
			Name:        "detached",
			ShortFlag:   "d",
			Type:        ArgBool,
			Default:     false,
			Description: "Do not attach to the new session",
		},
		{
			Name:        "session-name",
			ShortFlag:   "s",
			Type:        ArgString,
			Description: "Name of the new session",
		},
		{
			Name:        "shell-command",
			Type:        ArgString,
			Description: "Command to run in the new session",
		},
	}
}

// TmuxAttachSessionCommand tmux attach-session命令
type TmuxAttachSessionCommand struct {
	adapter *TmuxCommandAdapter
}

func (cmd *TmuxAttachSessionCommand) Execute(ctx *Context, args *Arguments) error {
	return cmd.adapter.ExecuteAttachSession(ctx, args)
}

func (cmd *TmuxAttachSessionCommand) Validate(args *Arguments) error {
	return nil
}

func (cmd *TmuxAttachSessionCommand) Usage() string {
	return "attach-session [-t target-session]"
}

func (cmd *TmuxAttachSessionCommand) Name() string {
	return "attach-session"
}

func (cmd *TmuxAttachSessionCommand) Description() string {
	return "Attach to an existing session"
}

func (cmd *TmuxAttachSessionCommand) ArgumentSpecs() []ArgumentSpec {
	return []ArgumentSpec{
		{
			Name:        "target-session",
			ShortFlag:   "t",
			Type:        ArgString,
			Description: "Target session name or ID",
		},
	}
}

// TmuxListSessionsCommand tmux list-sessions命令
type TmuxListSessionsCommand struct {
	adapter *TmuxCommandAdapter
}

func (cmd *TmuxListSessionsCommand) Execute(ctx *Context, args *Arguments) error {
	return cmd.adapter.ExecuteListSessions(ctx, args)
}

func (cmd *TmuxListSessionsCommand) Validate(args *Arguments) error {
	return nil
}

func (cmd *TmuxListSessionsCommand) Usage() string {
	return "list-sessions"
}

func (cmd *TmuxListSessionsCommand) Name() string {
	return "list-sessions"
}

func (cmd *TmuxListSessionsCommand) Description() string {
	return "List all sessions"
}

func (cmd *TmuxListSessionsCommand) ArgumentSpecs() []ArgumentSpec {
	return []ArgumentSpec{}
}

// TmuxKillSessionCommand tmux kill-session命令
type TmuxKillSessionCommand struct {
	adapter *TmuxCommandAdapter
}

func (cmd *TmuxKillSessionCommand) Execute(ctx *Context, args *Arguments) error {
	return cmd.adapter.ExecuteKillSession(ctx, args)
}

func (cmd *TmuxKillSessionCommand) Validate(args *Arguments) error {
	return nil
}

func (cmd *TmuxKillSessionCommand) Usage() string {
	return "kill-session [-t target-session]"
}

func (cmd *TmuxKillSessionCommand) Name() string {
	return "kill-session"
}

func (cmd *TmuxKillSessionCommand) Description() string {
	return "Kill a session"
}

func (cmd *TmuxKillSessionCommand) ArgumentSpecs() []ArgumentSpec {
	return []ArgumentSpec{
		{
			Name:        "target-session",
			ShortFlag:   "t",
			Type:        ArgString,
			Description: "Target session name or ID",
		},
	}
}

// TmuxNewWindowCommand tmux new-window命令
type TmuxNewWindowCommand struct {
	adapter *TmuxCommandAdapter
}

func (cmd *TmuxNewWindowCommand) Execute(ctx *Context, args *Arguments) error {
	return cmd.adapter.ExecuteNewWindow(ctx, args)
}

func (cmd *TmuxNewWindowCommand) Validate(args *Arguments) error {
	return nil
}

func (cmd *TmuxNewWindowCommand) Usage() string {
	return "new-window [-n window-name] [-t target-session] [shell-command]"
}

func (cmd *TmuxNewWindowCommand) Name() string {
	return "new-window"
}

func (cmd *TmuxNewWindowCommand) Description() string {
	return "Create a new window"
}

func (cmd *TmuxNewWindowCommand) ArgumentSpecs() []ArgumentSpec {
	return []ArgumentSpec{
		{
			Name:        "window-name",
			ShortFlag:   "n",
			Type:        ArgString,
			Description: "Name of the new window",
		},
		{
			Name:        "target-session",
			ShortFlag:   "t",
			Type:        ArgString,
			Description: "Target session name or ID",
		},
		{
			Name:        "shell-command",
			Type:        ArgString,
			Description: "Command to run in the new window",
		},
	}
}

// 适配器实现方法

// ExecuteNewSession 执行new-session命令
func (adapter *TmuxCommandAdapter) ExecuteNewSession(ctx *Context, args *Arguments) error {
	sessionName := ""
	if name, exists := args.Flags["session-name"]; exists {
		sessionName = name.(string)
	}

	detached := false
	if d, exists := args.Flags["detached"]; exists {
		detached = d.(bool)
	}

	adapter.logger.Debug("Creating new session: name=%s, detached=%v", sessionName, detached)

	session, err := adapter.integration.sessionManager.CreateSession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}

	adapter.logger.Info("Session created: id=%s, name=%s", session.ID, session.Name)

	if !detached {
		// 如果不是分离模式，自动附加到会话
		if err := adapter.integration.sessionManager.AttachSession(session.ID); err != nil {
			return fmt.Errorf("failed to attach to session: %v", err)
		}
		adapter.logger.Debug("Attached to session: %s", session.ID)
	}

	return nil
}

// ExecuteAttachSession 执行attach-session命令
func (adapter *TmuxCommandAdapter) ExecuteAttachSession(ctx *Context, args *Arguments) error {
	targetSession := ""
	if target, exists := args.Flags["target-session"]; exists {
		targetSession = target.(string)
	}

	// 如果没有指定目标会话，尝试附加到第一个可用会话
	if targetSession == "" {
		sessions := adapter.integration.sessionManager.ListSessions()
		if len(sessions) == 0 {
			return fmt.Errorf("no sessions available")
		}
		targetSession = sessions[0].ID
	}

	// 尝试通过名称查找会话
	var sessionID string
	if session, err := adapter.integration.sessionManager.GetSessionByName(targetSession); err == nil {
		sessionID = session.ID
	} else {
		// 尝试直接作为ID使用
		sessionID = targetSession
	}

	adapter.logger.Debug("Attaching to session: %s", sessionID)

	if err := adapter.integration.sessionManager.AttachSession(sessionID); err != nil {
		return fmt.Errorf("failed to attach to session: %v", err)
	}

	adapter.logger.Info("Attached to session: %s", sessionID)
	return nil
}

// ExecuteListSessions 执行list-sessions命令
func (adapter *TmuxCommandAdapter) ExecuteListSessions(ctx *Context, args *Arguments) error {
	sessions := adapter.integration.sessionManager.ListSessions()

	if len(sessions) == 0 {
		fmt.Println("no server running")
		return nil
	}

	for _, session := range sessions {
		// 模拟tmux输出格式
		windowCount := len(session.Windows)
		status := "attached"
		if session.Status != terminal.SessionActive {
			status = "detached"
		}

		fmt.Printf("%s: %d windows (created %s) [%s]\n",
			session.Name,
			windowCount,
			session.CreatedAt.Format("Mon Jan _2 15:04:05 2006"),
			status,
		)
	}

	return nil
}

// ExecuteKillSession 执行kill-session命令
func (adapter *TmuxCommandAdapter) ExecuteKillSession(ctx *Context, args *Arguments) error {
	targetSession := ""
	if target, exists := args.Flags["target-session"]; exists {
		targetSession = target.(string)
	}

	if targetSession == "" {
		return fmt.Errorf("target session not specified")
	}

	// 尝试通过名称查找会话
	var sessionID string
	if session, err := adapter.integration.sessionManager.GetSessionByName(targetSession); err == nil {
		sessionID = session.ID
	} else {
		sessionID = targetSession
	}

	adapter.logger.Debug("Killing session: %s", sessionID)

	if err := adapter.integration.sessionManager.KillSession(sessionID); err != nil {
		return fmt.Errorf("failed to kill session: %v", err)
	}

	adapter.logger.Info("Session killed: %s", sessionID)
	return nil
}

// ExecuteNewWindow 执行new-window命令
func (adapter *TmuxCommandAdapter) ExecuteNewWindow(ctx *Context, args *Arguments) error {
	windowName := ""
	if name, exists := args.Flags["window-name"]; exists {
		windowName = name.(string)
	}

	targetSession := ""
	if target, exists := args.Flags["target-session"]; exists {
		targetSession = target.(string)
	}

	// 如果没有指定目标会话，使用当前会话
	var sessionID string
	if targetSession == "" {
		if ctx.Session != nil {
			sessionID = ctx.Session.ID
		} else {
			// 使用第一个可用会话
			sessions := adapter.integration.sessionManager.ListSessions()
			if len(sessions) == 0 {
				return fmt.Errorf("no sessions available")
			}
			sessionID = sessions[0].ID
		}
	} else {
		// 尝试通过名称查找会话
		if session, err := adapter.integration.sessionManager.GetSessionByName(targetSession); err == nil {
			sessionID = session.ID
		} else {
			sessionID = targetSession
		}
	}

	adapter.logger.Debug("Creating new window: session=%s, name=%s", sessionID, windowName)

	window, err := adapter.integration.sessionManager.CreateWindow(sessionID, windowName)
	if err != nil {
		return fmt.Errorf("failed to create window: %v", err)
	}

	adapter.logger.Info("Window created: id=%s, name=%s", window.ID, window.Name)
	return nil
}

// 数据类型转换辅助函数

// ConvertTerminalSessionToTmux 将terminal.Session转换为tmux格式的Session
func ConvertTerminalSessionToTmux(terminalSession *terminal.Session) *Session {
	if terminalSession == nil {
		return nil
	}

	// 转换窗口
	var windows []*Window
	for i, terminalWindow := range terminalSession.Windows {
		// Window.Active字段在commands包中是int类型，表示活跃面板索引
		windows = append(windows, &Window{
			ID:     terminalWindow.ID,
			Name:   terminalWindow.Name,
			Index:  i,
			Active: terminalWindow.ActivePane,
			Panes:  ConvertTerminalPanesToTmux(terminalWindow.Panes),
		})
	}

	return &Session{
		ID:      terminalSession.ID,
		Name:    terminalSession.Name,
		Windows: windows,
		Active:  terminalSession.ActiveWindow,
		Created: terminalSession.CreatedAt.Unix(),
	}
}

// ConvertTerminalPanesToTmux 将terminal.Pane数组转换为tmux格式
func ConvertTerminalPanesToTmux(terminalPanes []*terminal.Pane) []*Pane {
	var panes []*Pane
	for _, terminalPane := range terminalPanes {
		panes = append(panes, &Pane{
			ID:     terminalPane.ID,
			Width:  terminalPane.Width,
			Height: terminalPane.Height,
			X:      terminalPane.X,
			Y:      terminalPane.Y,
			Active: terminalPane.Active,
		})
	}
	return panes
}

// ConvertTmuxSessionToTerminal 将tmux格式的Session转换为terminal.Session
func ConvertTmuxSessionToTerminal(tmuxSession *Session) *terminal.Session {
	if tmuxSession == nil {
		return nil
	}

	// 转换窗口
	var windows []*terminal.Window
	for _, tmuxWindow := range tmuxSession.Windows {
		windows = append(windows, &terminal.Window{
			ID:         tmuxWindow.ID,
			Name:       tmuxWindow.Name,
			Index:      tmuxWindow.Index,
			ActivePane: tmuxWindow.Active, // Active字段映射到ActivePane
			Panes:      ConvertTmuxPanesToTerminal(tmuxWindow.Panes),
			Layout:     terminal.LayoutEven, // 默认布局
			CreatedAt:  time.Now(),          // 使用当前时间
		})
	}

	return &terminal.Session{
		ID:           tmuxSession.ID,
		Name:         tmuxSession.Name,
		Windows:      windows,
		ActiveWindow: tmuxSession.Active,
		CreatedAt:    time.Unix(tmuxSession.Created, 0),
		LastActive:   time.Now(),
		Status:       terminal.SessionActive, // 默认为活跃状态
	}
}

// ConvertTmuxPanesToTerminal 将tmux格式的Pane数组转换为terminal.Pane
func ConvertTmuxPanesToTerminal(tmuxPanes []*Pane) []*terminal.Pane {
	var panes []*terminal.Pane
	for _, tmuxPane := range tmuxPanes {
		panes = append(panes, &terminal.Pane{
			ID:     tmuxPane.ID,
			Width:  tmuxPane.Width,
			Height: tmuxPane.Height,
			X:      tmuxPane.X,
			Y:      tmuxPane.Y,
			Active: tmuxPane.Active,
		})
	}
	return panes
}

// 工具函数

// parseSessionTarget 解析会话目标（支持名称和索引）
func parseSessionTarget(target string) (string, int, error) {
	if target == "" {
		return "", -1, fmt.Errorf("empty target")
	}

	// 如果是数字，作为索引处理
	if index, err := strconv.Atoi(target); err == nil {
		return "", index, nil
	}

	// 否则作为名称处理
	return target, -1, nil
}

// parseWindowTarget 解析窗口目标（格式：session:window）
func parseWindowTarget(target string) (string, string, error) {
	if target == "" {
		return "", "", fmt.Errorf("empty target")
	}

	parts := strings.Split(target, ":")
	if len(parts) == 1 {
		// 只有窗口标识
		return "", parts[0], nil
	} else if len(parts) == 2 {
		// 会话:窗口格式
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("invalid target format: %s", target)
}

// RegisterTmuxCommands 注册tmux兼容命令到解析器
func RegisterTmuxCommands(parser *EnhancedParser, integrationLayer *SessionIntegrationLayer, logger Logger) error {
	adapter := NewTmuxCommandAdapter(integrationLayer, logger)

	// 注册核心tmux命令
	commands := []Command{
		&TmuxNewSessionCommand{adapter: adapter},
		&TmuxAttachSessionCommand{adapter: adapter},
		&TmuxListSessionsCommand{adapter: adapter},
		&TmuxKillSessionCommand{adapter: adapter},
		&TmuxNewWindowCommand{adapter: adapter},
	}

	for _, cmd := range commands {
		if err := parser.RegisterCommand(cmd); err != nil {
			return fmt.Errorf("failed to register command %s: %v", cmd.Name(), err)
		}
	}

	return nil
}
