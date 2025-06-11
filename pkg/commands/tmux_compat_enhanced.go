/*
* @Author: Lzww0608
* @Date: 2025-6-11 11:09:41
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-11 11:09:44
* @Description: 增强版tmux兼容层 - 基于现有架构优化的终端多路复用器兼容实现
 */

package commands

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

// EnhancedTmuxCompatLayer 增强版tmux兼容层
type EnhancedTmuxCompatLayer struct {
	parser         *ModernParser
	sessionManager *terminal.SessionManager
	logger         Logger
	prefixKey      string
	keyBindings    map[string]*KeyBinding
	commandAliases map[string]string
	mutex          sync.RWMutex
}

// NewEnhancedTmuxCompatLayer 创建增强版tmux兼容层
func NewEnhancedTmuxCompatLayer(sessionManager *terminal.SessionManager, logger Logger) *EnhancedTmuxCompatLayer {
	parser := NewModernParser(logger)

	layer := &EnhancedTmuxCompatLayer{
		parser:         parser,
		sessionManager: sessionManager,
		logger:         logger,
		prefixKey:      "C-b",
		keyBindings:    make(map[string]*KeyBinding),
		commandAliases: make(map[string]string),
	}

	// 初始化tmux兼容命令
	layer.initTmuxCommands()

	// 初始化tmux兼容快捷键
	layer.initTmuxKeyBindings()

	// 初始化命令别名
	layer.initCommandAliases()

	return layer
}

// initTmuxCommands 初始化tmux兼容命令
func (e *EnhancedTmuxCompatLayer) initTmuxCommands() {
	// 会话管理命令
	e.parser.RegisterCommand(&EnhancedNewSessionCommand{
		sessionManager: e.sessionManager,
		logger:         e.logger,
	})

	e.parser.RegisterCommand(&EnhancedAttachSessionCommand{
		sessionManager: e.sessionManager,
		logger:         e.logger,
	})

	e.parser.RegisterCommand(&EnhancedListSessionsCommand{
		sessionManager: e.sessionManager,
		logger:         e.logger,
	})

	e.parser.RegisterCommand(&EnhancedKillSessionCommand{
		sessionManager: e.sessionManager,
		logger:         e.logger,
	})

	// 窗口管理命令
	e.parser.RegisterCommand(&EnhancedNewWindowCommand{
		sessionManager: e.sessionManager,
		logger:         e.logger,
	})

	// 面板管理命令
	e.parser.RegisterCommand(&EnhancedSplitWindowCommand{
		sessionManager: e.sessionManager,
		logger:         e.logger,
	})
}

// initTmuxKeyBindings 初始化tmux兼容快捷键
func (e *EnhancedTmuxCompatLayer) initTmuxKeyBindings() {
	// tmux默认快捷键绑定
	tmuxBindings := map[string]*KeyBinding{
		"d":       {Key: "d", Command: "detach-client", Args: []string{}},
		"c":       {Key: "c", Command: "new-window", Args: []string{}},
		"&":       {Key: "&", Command: "confirm-before", Args: []string{"-p", "kill-window #W? (y/n)", "kill-window"}},
		"x":       {Key: "x", Command: "confirm-before", Args: []string{"-p", "kill-pane #P? (y/n)", "kill-pane"}},
		"\"":      {Key: "\"", Command: "split-window", Args: []string{}},
		"%":       {Key: "%", Command: "split-window", Args: []string{"-h"}},
		"o":       {Key: "o", Command: "select-pane", Args: []string{"-t", ":.+"}},
		"n":       {Key: "n", Command: "next-window", Args: []string{}},
		"p":       {Key: "p", Command: "previous-window", Args: []string{}},
		"l":       {Key: "l", Command: "last-window", Args: []string{}},
		"s":       {Key: "s", Command: "choose-tree", Args: []string{}},
		"w":       {Key: "w", Command: "choose-window", Args: []string{}},
		"t":       {Key: "t", Command: "clock-mode", Args: []string{}},
		"?":       {Key: "?", Command: "list-keys", Args: []string{}},
		":":       {Key: ":", Command: "command-prompt", Args: []string{}},
		"[":       {Key: "[", Command: "copy-mode", Args: []string{}},
		"]":       {Key: "]", Command: "paste-buffer", Args: []string{}},
		"Space":   {Key: "Space", Command: "next-layout", Args: []string{}},
		"M-Up":    {Key: "M-Up", Command: "resize-pane", Args: []string{"-U", "5"}},
		"M-Down":  {Key: "M-Down", Command: "resize-pane", Args: []string{"-D", "5"}},
		"M-Left":  {Key: "M-Left", Command: "resize-pane", Args: []string{"-L", "5"}},
		"M-Right": {Key: "M-Right", Command: "resize-pane", Args: []string{"-R", "5"}},
		"Up":      {Key: "Up", Command: "select-pane", Args: []string{"-U"}},
		"Down":    {Key: "Down", Command: "select-pane", Args: []string{"-D"}},
		"Left":    {Key: "Left", Command: "select-pane", Args: []string{"-L"}},
		"Right":   {Key: "Right", Command: "select-pane", Args: []string{"-R"}},
		"C-Up":    {Key: "C-Up", Command: "resize-pane", Args: []string{"-U"}},
		"C-Down":  {Key: "C-Down", Command: "resize-pane", Args: []string{"-D"}},
		"C-Left":  {Key: "C-Left", Command: "resize-pane", Args: []string{"-L"}},
		"C-Right": {Key: "C-Right", Command: "resize-pane", Args: []string{"-R"}},
	}

	// 数字键绑定 (0-9)
	for i := 0; i <= 9; i++ {
		key := strconv.Itoa(i)
		tmuxBindings[key] = &KeyBinding{
			Key:     key,
			Command: "select-window",
			Args:    []string{"-t", ":" + key},
		}
	}

	// 功能键绑定 (F1-F12)
	for i := 1; i <= 12; i++ {
		key := fmt.Sprintf("F%d", i)
		tmuxBindings[key] = &KeyBinding{
			Key:     key,
			Command: "select-window",
			Args:    []string{"-t", ":" + strconv.Itoa(i-1)},
		}
	}

	e.keyBindings = tmuxBindings
}

// initCommandAliases 初始化命令别名
func (e *EnhancedTmuxCompatLayer) initCommandAliases() {
	// tmux常用命令别名
	aliases := map[string]string{
		"new":       "new-session",
		"attach":    "attach-session",
		"att":       "attach-session",
		"detach":    "detach-client",
		"det":       "detach-client",
		"ls":        "list-sessions",
		"list":      "list-sessions",
		"lss":       "list-sessions",
		"kill":      "kill-session",
		"neww":      "new-window",
		"movew":     "move-window",
		"renamew":   "rename-window",
		"linkw":     "link-window",
		"unlinkw":   "unlink-window",
		"splitw":    "split-window",
		"joinp":     "join-pane",
		"movep":     "move-pane",
		"swapp":     "swap-pane",
		"resizep":   "resize-pane",
		"selectp":   "select-pane",
		"lastp":     "last-pane",
		"nextl":     "next-layout",
		"prevl":     "previous-layout",
		"rotatew":   "rotate-window",
		"setw":      "set-window-option",
		"showw":     "show-window-options",
		"set":       "set-option",
		"show":      "show-options",
		"bind":      "bind-key",
		"unbind":    "unbind-key",
		"source":    "source-file",
		"refresh":   "refresh-client",
		"info":      "show-messages",
		"capture":   "capture-pane",
		"clearhist": "clear-history",
	}

	e.commandAliases = aliases
}

// ParseTmuxCommand 解析tmux命令
func (e *EnhancedTmuxCompatLayer) ParseTmuxCommand(input string) (*CommandList, error) {
	if input == "" {
		return nil, fmt.Errorf("empty command input")
	}

	// 处理命令别名
	input = e.expandAliases(input)

	// 使用现有的解析器
	return e.parser.Parse(input)
}

// expandAliases 展开命令别名
func (e *EnhancedTmuxCompatLayer) expandAliases(input string) string {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return input
	}

	// 检查第一个单词是否是别名
	if alias, exists := e.commandAliases[parts[0]]; exists {
		parts[0] = alias
		return strings.Join(parts, " ")
	}

	return input
}

// HandleKeyBinding 处理快捷键绑定
func (e *EnhancedTmuxCompatLayer) HandleKeyBinding(key string) (*CommandList, error) {
	e.mutex.RLock()
	binding, exists := e.keyBindings[key]
	e.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("key binding not found: %s", key)
	}

	// 构建命令字符串
	cmdStr := binding.Command
	if len(binding.Args) > 0 {
		cmdStr += " " + strings.Join(binding.Args, " ")
	}

	// 解析并返回命令
	return e.ParseTmuxCommand(cmdStr)
}

// =========================== 增强版命令实现 ===========================

// EnhancedNewSessionCommand 增强版新建会话命令
type EnhancedNewSessionCommand struct {
	sessionManager *terminal.SessionManager
	logger         Logger
}

func (c *EnhancedNewSessionCommand) Name() string {
	return "new-session"
}

func (c *EnhancedNewSessionCommand) Description() string {
	return "Create a new terminal session (tmux compatible)"
}

func (c *EnhancedNewSessionCommand) Usage() string {
	return "new-session [-d] [-s session-name] [-n window-name] [-c start-directory] [shell-command]"
}

func (c *EnhancedNewSessionCommand) ArgumentSpecs() []ArgumentSpec {
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
			Name:        "window-name",
			ShortFlag:   "n",
			Type:        ArgString,
			Description: "Name of the initial window",
		},
		{
			Name:        "start-directory",
			ShortFlag:   "c",
			Type:        ArgString,
			Description: "Working directory for the new session",
		},
		{
			Name:        "shell-command",
			Type:        ArgString,
			Description: "Shell command to run in the new session",
		},
	}
}

func (c *EnhancedNewSessionCommand) Validate(args *Arguments) error {
	if sessionName, exists := args.Flags["session-name"]; exists {
		if name, ok := sessionName.(string); ok && name == "" {
			return fmt.Errorf("session name cannot be empty")
		}
	}
	return nil
}

func (c *EnhancedNewSessionCommand) Execute(ctx *Context, args *Arguments) error {
	// 获取会话名称
	sessionName := ""
	if name, exists := args.Flags["session-name"]; exists {
		if nameStr, ok := name.(string); ok {
			sessionName = nameStr
		}
	}

	// 创建新会话
	session, err := c.sessionManager.CreateSession(sessionName)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	c.logger.Info("Created new session: %s (ID: %s)", session.Name, session.ID)

	// 更新上下文
	ctx.Session = &Session{
		ID:      session.ID,
		Name:    session.Name,
		Windows: []*Window{},
		Active:  0,
		Created: session.CreatedAt.Unix(),
	}

	// 处理detached模式
	if detached, exists := args.Flags["detached"]; exists {
		if detachedBool, ok := detached.(bool); ok && detachedBool {
			c.logger.Info("Session created in detached mode")
			return nil
		}
	}

	// 连接到新会话
	if err := c.sessionManager.AttachSession(session.ID); err != nil {
		return fmt.Errorf("failed to attach to session: %w", err)
	}

	c.logger.Info("Attached to session: %s", session.Name)
	return nil
}

// EnhancedAttachSessionCommand 增强版连接会话命令
type EnhancedAttachSessionCommand struct {
	sessionManager *terminal.SessionManager
	logger         Logger
}

func (c *EnhancedAttachSessionCommand) Name() string {
	return "attach-session"
}

func (c *EnhancedAttachSessionCommand) Description() string {
	return "Attach to an existing session (tmux compatible)"
}

func (c *EnhancedAttachSessionCommand) Usage() string {
	return "attach-session [-d] [-r] [-t target-session]"
}

func (c *EnhancedAttachSessionCommand) ArgumentSpecs() []ArgumentSpec {
	return []ArgumentSpec{
		{
			Name:        "detach-others",
			ShortFlag:   "d",
			Type:        ArgBool,
			Default:     false,
			Description: "Detach other clients attached to the session",
		},
		{
			Name:        "read-only",
			ShortFlag:   "r",
			Type:        ArgBool,
			Default:     false,
			Description: "Attach in read-only mode",
		},
		{
			Name:        "target-session",
			ShortFlag:   "t",
			Type:        ArgString,
			Description: "Target session name or ID",
		},
	}
}

func (c *EnhancedAttachSessionCommand) Validate(args *Arguments) error {
	return nil
}

func (c *EnhancedAttachSessionCommand) Execute(ctx *Context, args *Arguments) error {
	targetSession := ""
	if target, exists := args.Flags["target-session"]; exists {
		if targetStr, ok := target.(string); ok {
			targetSession = targetStr
		}
	}

	// 如果没有指定目标会话，使用最近的会话
	if targetSession == "" {
		sessions := c.sessionManager.ListSessions()
		if len(sessions) == 0 {
			return fmt.Errorf("no sessions to attach to")
		}
		targetSession = sessions[0].ID
	}

	// 尝试按名称查找会话
	var session *terminal.Session
	var err error

	if session, err = c.sessionManager.GetSessionByName(targetSession); err != nil {
		// 如果按名称找不到，尝试按ID查找
		if session, err = c.sessionManager.GetSession(targetSession); err != nil {
			return fmt.Errorf("session not found: %s", targetSession)
		}
	}

	// 连接到会话
	if err := c.sessionManager.AttachSession(session.ID); err != nil {
		return fmt.Errorf("failed to attach to session: %w", err)
	}

	c.logger.Info("Attached to session: %s (ID: %s)", session.Name, session.ID)

	// 更新上下文
	ctx.Session = &Session{
		ID:      session.ID,
		Name:    session.Name,
		Windows: []*Window{},
		Active:  session.ActiveWindow,
		Created: session.CreatedAt.Unix(),
	}

	return nil
}

// EnhancedListSessionsCommand 增强版列出会话命令
type EnhancedListSessionsCommand struct {
	sessionManager *terminal.SessionManager
	logger         Logger
}

func (c *EnhancedListSessionsCommand) Name() string {
	return "list-sessions"
}

func (c *EnhancedListSessionsCommand) Description() string {
	return "List all sessions (tmux compatible)"
}

func (c *EnhancedListSessionsCommand) Usage() string {
	return "list-sessions [-F format]"
}

func (c *EnhancedListSessionsCommand) ArgumentSpecs() []ArgumentSpec {
	return []ArgumentSpec{
		{
			Name:        "format",
			ShortFlag:   "F",
			Type:        ArgString,
			Default:     "#{session_name}: #{session_windows} windows (created #{session_created})",
			Description: "Output format",
		},
	}
}

func (c *EnhancedListSessionsCommand) Validate(args *Arguments) error {
	return nil
}

func (c *EnhancedListSessionsCommand) Execute(ctx *Context, args *Arguments) error {
	format := "#{session_name}: #{session_windows} windows (created #{session_created})"
	if f, exists := args.Flags["format"]; exists {
		if formatStr, ok := f.(string); ok {
			format = formatStr
		}
	}

	sessions := c.sessionManager.ListSessions()
	if len(sessions) == 0 {
		c.logger.Info("No sessions found")
		return nil
	}

	c.logger.Info("Sessions:")
	for _, session := range sessions {
		output := c.formatSessionOutput(session, format)
		c.logger.Info(output)
	}

	return nil
}

func (c *EnhancedListSessionsCommand) formatSessionOutput(session *terminal.Session, format string) string {
	// 替换tmux格式变量
	output := format
	output = strings.ReplaceAll(output, "#{session_name}", session.Name)
	output = strings.ReplaceAll(output, "#{session_id}", session.ID[:8])
	output = strings.ReplaceAll(output, "#{session_windows}", strconv.Itoa(len(session.Windows)))
	output = strings.ReplaceAll(output, "#{session_created}", session.CreatedAt.Format("Mon Jan _2 15:04:05 2006"))
	output = strings.ReplaceAll(output, "#{session_activity}", session.LastActive.Format("Mon Jan _2 15:04:05 2006"))
	output = strings.ReplaceAll(output, "#{session_attached}", strconv.Itoa(1)) // 简化，总是显示1

	return output
}

// EnhancedKillSessionCommand 增强版删除会话命令
type EnhancedKillSessionCommand struct {
	sessionManager *terminal.SessionManager
	logger         Logger
}

func (c *EnhancedKillSessionCommand) Name() string {
	return "kill-session"
}

func (c *EnhancedKillSessionCommand) Description() string {
	return "Kill a session (tmux compatible)"
}

func (c *EnhancedKillSessionCommand) Usage() string {
	return "kill-session [-t target-session] [-a]"
}

func (c *EnhancedKillSessionCommand) ArgumentSpecs() []ArgumentSpec {
	return []ArgumentSpec{
		{
			Name:        "target-session",
			ShortFlag:   "t",
			Type:        ArgString,
			Description: "Target session name or ID",
		},
		{
			Name:        "all",
			ShortFlag:   "a",
			Type:        ArgBool,
			Default:     false,
			Description: "Kill all sessions except the current one",
		},
	}
}

func (c *EnhancedKillSessionCommand) Validate(args *Arguments) error {
	return nil
}

func (c *EnhancedKillSessionCommand) Execute(ctx *Context, args *Arguments) error {
	killAll := false
	if all, exists := args.Flags["all"]; exists {
		if allBool, ok := all.(bool); ok {
			killAll = allBool
		}
	}

	if killAll {
		// 杀死所有会话
		sessions := c.sessionManager.ListSessions()
		for _, session := range sessions {
			if err := c.sessionManager.KillSession(session.ID); err != nil {
				c.logger.Error("Failed to kill session %s: %v", session.Name, err)
			} else {
				c.logger.Info("Killed session: %s", session.Name)
			}
		}
		return nil
	}

	// 杀死指定会话
	targetSession := ""
	if target, exists := args.Flags["target-session"]; exists {
		if targetStr, ok := target.(string); ok {
			targetSession = targetStr
		}
	}

	if targetSession == "" {
		if ctx.Session == nil {
			return fmt.Errorf("no target session specified")
		}
		targetSession = ctx.Session.ID
	}

	// 尝试按名称查找会话
	var session *terminal.Session
	var err error

	if session, err = c.sessionManager.GetSessionByName(targetSession); err != nil {
		// 如果按名称找不到，尝试按ID查找
		if session, err = c.sessionManager.GetSession(targetSession); err != nil {
			return fmt.Errorf("session not found: %s", targetSession)
		}
	}

	// 杀死会话
	if err := c.sessionManager.KillSession(session.ID); err != nil {
		return fmt.Errorf("failed to kill session: %w", err)
	}

	c.logger.Info("Killed session: %s", session.Name)
	return nil
}

// EnhancedNewWindowCommand 增强版新建窗口命令
type EnhancedNewWindowCommand struct {
	sessionManager *terminal.SessionManager
	logger         Logger
}

func (c *EnhancedNewWindowCommand) Name() string {
	return "new-window"
}

func (c *EnhancedNewWindowCommand) Description() string {
	return "Create a new window (tmux compatible)"
}

func (c *EnhancedNewWindowCommand) Usage() string {
	return "new-window [-n window-name] [-t target-session] [-c start-directory] [shell-command]"
}

func (c *EnhancedNewWindowCommand) ArgumentSpecs() []ArgumentSpec {
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
			Description: "Target session for the new window",
		},
		{
			Name:        "start-directory",
			ShortFlag:   "c",
			Type:        ArgString,
			Description: "Working directory for the new window",
		},
	}
}

func (c *EnhancedNewWindowCommand) Validate(args *Arguments) error {
	return nil
}

func (c *EnhancedNewWindowCommand) Execute(ctx *Context, args *Arguments) error {
	// 获取目标会话
	targetSession := ""
	if target, exists := args.Flags["target-session"]; exists {
		if targetStr, ok := target.(string); ok {
			targetSession = targetStr
		}
	}

	if targetSession == "" && ctx.Session != nil {
		targetSession = ctx.Session.ID
	}

	if targetSession == "" {
		return fmt.Errorf("no target session specified")
	}

	// 获取会话
	var session *terminal.Session
	var err error

	if session, err = c.sessionManager.GetSession(targetSession); err != nil {
		if session, err = c.sessionManager.GetSessionByName(targetSession); err != nil {
			return fmt.Errorf("session not found: %s", targetSession)
		}
	}

	// 获取窗口名称
	windowName := ""
	if name, exists := args.Flags["window-name"]; exists {
		if nameStr, ok := name.(string); ok {
			windowName = nameStr
		}
	}

	// 创建新窗口
	window, err := c.sessionManager.CreateWindow(session.ID, windowName)
	if err != nil {
		return fmt.Errorf("failed to create window: %w", err)
	}

	c.logger.Info("Created new window: %s in session %s", window.Name, session.Name)

	// 更新上下文
	if ctx.Session != nil && ctx.Session.ID == session.ID {
		ctx.Window = &Window{
			ID:     window.ID,
			Name:   window.Name,
			Panes:  []*Pane{},
			Active: 0,
			Index:  window.Index,
		}
	}

	return nil
}

// EnhancedSplitWindowCommand 增强版分割窗口命令
type EnhancedSplitWindowCommand struct {
	sessionManager *terminal.SessionManager
	logger         Logger
}

func (c *EnhancedSplitWindowCommand) Name() string {
	return "split-window"
}

func (c *EnhancedSplitWindowCommand) Description() string {
	return "Split a window into panes (tmux compatible)"
}

func (c *EnhancedSplitWindowCommand) Usage() string {
	return "split-window [-h] [-v] [-t target-pane] [-c start-directory] [shell-command]"
}

func (c *EnhancedSplitWindowCommand) ArgumentSpecs() []ArgumentSpec {
	return []ArgumentSpec{
		{
			Name:        "horizontal",
			ShortFlag:   "h",
			Type:        ArgBool,
			Default:     false,
			Description: "Split horizontally",
		},
		{
			Name:        "vertical",
			ShortFlag:   "v",
			Type:        ArgBool,
			Default:     true,
			Description: "Split vertically (default)",
		},
		{
			Name:        "target-pane",
			ShortFlag:   "t",
			Type:        ArgString,
			Description: "Target pane to split",
		},
		{
			Name:        "start-directory",
			ShortFlag:   "c",
			Type:        ArgString,
			Description: "Working directory for the new pane",
		},
	}
}

func (c *EnhancedSplitWindowCommand) Validate(args *Arguments) error {
	return nil
}

func (c *EnhancedSplitWindowCommand) Execute(ctx *Context, args *Arguments) error {
	// 确定分割方向
	horizontal := false
	if h, exists := args.Flags["horizontal"]; exists {
		if hBool, ok := h.(bool); ok {
			horizontal = hBool
		}
	}

	direction := "vertical"
	if horizontal {
		direction = "horizontal"
	}

	c.logger.Info("Splitting window %s", direction)

	// 在真实实现中，这里会调用SessionManager的分割面板功能
	// 目前简单记录操作

	return nil
}
