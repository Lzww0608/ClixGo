/*
* @Author: Lzww0608
* @Date: 2025-6-11 11:14:01
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-11 11:14:05
* @Description: 增强版命令解析器 - 扩展现有parser.go以支持tmux兼容性
 */

package commands

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// EnhancedParser 增强版命令解析器，扩展现有ModernParser
type EnhancedParser struct {
	*ModernParser
	tmuxAliases map[string]string      // tmux命令别名映射
	formatVars  map[string]string      // tmux格式变量
	keyBindings map[string]*KeyBinding // 快捷键绑定
	prefixKey   string                 // 前缀键
	tmuxMode    bool                   // tmux兼容模式
	mutex       sync.RWMutex
}

// NewEnhancedParser 创建增强版解析器
func NewEnhancedParser(logger Logger) *EnhancedParser {
	baseParser := NewModernParser(logger)

	enhanced := &EnhancedParser{
		ModernParser: baseParser,
		tmuxAliases:  make(map[string]string),
		formatVars:   make(map[string]string),
		keyBindings:  make(map[string]*KeyBinding),
		prefixKey:    "C-b",
		tmuxMode:     true,
	}

	// 初始化tmux兼容性
	enhanced.initTmuxCompatibility()

	return enhanced
}

// initTmuxCompatibility 初始化tmux兼容性设置
func (p *EnhancedParser) initTmuxCompatibility() {
	// 初始化tmux命令别名
	p.initTmuxAliases()

	// 初始化tmux格式变量
	p.initTmuxFormatVars()

	// 初始化tmux快捷键绑定
	p.initTmuxKeyBindings()
}

// initTmuxAliases 初始化tmux命令别名映射
func (p *EnhancedParser) initTmuxAliases() {
	aliases := map[string]string{
		// 会话管理
		"new":     "new-session",
		"attach":  "attach-session",
		"att":     "attach-session",
		"detach":  "detach-client",
		"det":     "detach-client",
		"ls":      "list-sessions",
		"list":    "list-sessions",
		"lss":     "list-sessions",
		"kill":    "kill-session",
		"has":     "has-session",
		"refresh": "refresh-client",

		// 窗口管理
		"neww":    "new-window",
		"movew":   "move-window",
		"renamew": "rename-window",
		"linkw":   "link-window",
		"unlinkw": "unlink-window",
		"findw":   "find-window",
		"nextw":   "next-window",
		"prevw":   "previous-window",
		"selectw": "select-window",
		"lastw":   "last-window",
		"lsw":     "list-windows",
		"killw":   "kill-window",

		// 面板管理
		"splitw":   "split-window",
		"joinp":    "join-pane",
		"movep":    "move-pane",
		"swapp":    "swap-pane",
		"resizep":  "resize-pane",
		"selectp":  "select-pane",
		"lastp":    "last-pane",
		"killp":    "kill-pane",
		"lsp":      "list-panes",
		"displayp": "display-panes",

		// 布局管理
		"nextl":   "next-layout",
		"prevl":   "previous-layout",
		"selectl": "select-layout",
		"rotatew": "rotate-window",

		// 配置管理
		"setw":   "set-window-option",
		"showw":  "show-window-options",
		"set":    "set-option",
		"show":   "show-options",
		"bind":   "bind-key",
		"unbind": "unbind-key",
		"source": "source-file",

		// 复制模式
		"copy":    "copy-mode",
		"paste":   "paste-buffer",
		"saveb":   "save-buffer",
		"loadb":   "load-buffer",
		"lsb":     "list-buffers",
		"deleteb": "delete-buffer",

		// 其他
		"info":      "show-messages",
		"capture":   "capture-pane",
		"clearhist": "clear-history",
		"clock":     "clock-mode",
		"choose":    "choose-tree",
		"choosew":   "choose-window",
		"confirms":  "confirm-before",
		"display":   "display-message",
	}

	p.tmuxAliases = aliases
}

// initTmuxFormatVars 初始化tmux格式变量
func (p *EnhancedParser) initTmuxFormatVars() {
	formatVars := map[string]string{
		// 会话变量
		"session_name":     "#{session_name}",
		"session_id":       "#{session_id}",
		"session_created":  "#{session_created}",
		"session_activity": "#{session_activity}",
		"session_attached": "#{session_attached}",
		"session_windows":  "#{session_windows}",
		"session_width":    "#{session_width}",
		"session_height":   "#{session_height}",

		// 窗口变量
		"window_name":          "#{window_name}",
		"window_id":            "#{window_id}",
		"window_index":         "#{window_index}",
		"window_active":        "#{window_active}",
		"window_bell_flag":     "#{window_bell_flag}",
		"window_activity_flag": "#{window_activity_flag}",
		"window_silence_flag":  "#{window_silence_flag}",
		"window_flags":         "#{window_flags}",
		"window_layout":        "#{window_layout}",
		"window_panes":         "#{window_panes}",
		"window_width":         "#{window_width}",
		"window_height":        "#{window_height}",

		// 面板变量
		"pane_id":              "#{pane_id}",
		"pane_index":           "#{pane_index}",
		"pane_active":          "#{pane_active}",
		"pane_current_command": "#{pane_current_command}",
		"pane_current_path":    "#{pane_current_path}",
		"pane_dead":            "#{pane_dead}",
		"pane_dead_status":     "#{pane_dead_status}",
		"pane_height":          "#{pane_height}",
		"pane_width":           "#{pane_width}",
		"pane_left":            "#{pane_left}",
		"pane_right":           "#{pane_right}",
		"pane_top":             "#{pane_top}",
		"pane_bottom":          "#{pane_bottom}",
		"pane_pid":             "#{pane_pid}",
		"pane_start_command":   "#{pane_start_command}",
		"pane_synchronized":    "#{pane_synchronized}",
		"pane_title":           "#{pane_title}",
		"pane_tty":             "#{pane_tty}",

		// 客户端变量
		"client_name":     "#{client_name}",
		"client_prefix":   "#{client_prefix}",
		"client_readonly": "#{client_readonly}",
		"client_session":  "#{client_session}",
		"client_termname": "#{client_termname}",
		"client_termtype": "#{client_termtype}",
		"client_tty":      "#{client_tty}",
		"client_utf8":     "#{client_utf8}",
		"client_width":    "#{client_width}",
		"client_height":   "#{client_height}",

		// 服务器变量
		"socket_path": "#{socket_path}",
		"start_time":  "#{start_time}",
		"version":     "#{version}",

		// 时间变量
		"client_activity": "#{client_activity}",
		"client_created":  "#{client_created}",
	}

	p.formatVars = formatVars
}

// initTmuxKeyBindings 初始化tmux快捷键绑定
func (p *EnhancedParser) initTmuxKeyBindings() {
	// 基础快捷键绑定
	basicBindings := map[string]*KeyBinding{
		"d": {Key: "d", Command: "detach-client"},
		"D": {Key: "D", Command: "choose-client"},
		"r": {Key: "r", Command: "refresh-client"},
		"t": {Key: "t", Command: "clock-mode"},
		"~": {Key: "~", Command: "show-messages"},
		"i": {Key: "i", Command: "display-message"},
		":": {Key: ":", Command: "command-prompt"},
		"?": {Key: "?", Command: "list-keys"},

		// 会话管理
		"s": {Key: "s", Command: "choose-tree"},
		"$": {Key: "$", Command: "command-prompt", Args: []string{"-I", "#S", "rename-session '%%'"}},
		"(": {Key: "(", Command: "switch-client", Args: []string{"-p"}},
		")": {Key: ")", Command: "switch-client", Args: []string{"-n"}},
		"L": {Key: "L", Command: "switch-client", Args: []string{"-l"}},

		// 窗口管理
		"c": {Key: "c", Command: "new-window"},
		"&": {Key: "&", Command: "confirm-before", Args: []string{"-p", "kill-window #W? (y/n)", "kill-window"}},
		",": {Key: ",", Command: "command-prompt", Args: []string{"-I", "#W", "rename-window '%%'"}},
		".": {Key: ".", Command: "command-prompt", Args: []string{"move-window -t '%%'"}},
		"f": {Key: "f", Command: "command-prompt", Args: []string{"find-window '%%'"}},
		"w": {Key: "w", Command: "choose-window"},
		"'": {Key: "'", Command: "command-prompt", Args: []string{"-p", "index", "select-window -t ':%%'"}},
		"n": {Key: "n", Command: "next-window"},
		"p": {Key: "p", Command: "previous-window"},
		"l": {Key: "l", Command: "last-window"},

		// 面板管理
		"\"": {Key: "\"", Command: "split-window"},
		"%":  {Key: "%", Command: "split-window", Args: []string{"-h"}},
		"o":  {Key: "o", Command: "select-pane", Args: []string{"-t", ":.+"}},
		";":  {Key: ";", Command: "last-pane"},
		"x":  {Key: "x", Command: "confirm-before", Args: []string{"-p", "kill-pane #P? (y/n)", "kill-pane"}},
		"!":  {Key: "!", Command: "break-pane"},
		"z":  {Key: "z", Command: "resize-pane", Args: []string{"-Z"}},
		"{":  {Key: "{", Command: "swap-pane", Args: []string{"-U"}},
		"}":  {Key: "}", Command: "swap-pane", Args: []string{"-D"}},
		"q":  {Key: "q", Command: "display-panes"},

		// 方向键面板选择
		"Up":    {Key: "Up", Command: "select-pane", Args: []string{"-U"}},
		"Down":  {Key: "Down", Command: "select-pane", Args: []string{"-D"}},
		"Left":  {Key: "Left", Command: "select-pane", Args: []string{"-L"}},
		"Right": {Key: "Right", Command: "select-pane", Args: []string{"-R"}},

		// Ctrl方向键面板调整大小
		"C-Up":    {Key: "C-Up", Command: "resize-pane", Args: []string{"-U"}},
		"C-Down":  {Key: "C-Down", Command: "resize-pane", Args: []string{"-D"}},
		"C-Left":  {Key: "C-Left", Command: "resize-pane", Args: []string{"-L"}},
		"C-Right": {Key: "C-Right", Command: "resize-pane", Args: []string{"-R"}},

		// Alt方向键面板调整大小（大步）
		"M-Up":    {Key: "M-Up", Command: "resize-pane", Args: []string{"-U", "5"}},
		"M-Down":  {Key: "M-Down", Command: "resize-pane", Args: []string{"-D", "5"}},
		"M-Left":  {Key: "M-Left", Command: "resize-pane", Args: []string{"-L", "5"}},
		"M-Right": {Key: "M-Right", Command: "resize-pane", Args: []string{"-R", "5"}},

		// 布局管理
		"Space": {Key: "Space", Command: "next-layout"},
		"M-1":   {Key: "M-1", Command: "select-layout", Args: []string{"even-horizontal"}},
		"M-2":   {Key: "M-2", Command: "select-layout", Args: []string{"even-vertical"}},
		"M-3":   {Key: "M-3", Command: "select-layout", Args: []string{"main-horizontal"}},
		"M-4":   {Key: "M-4", Command: "select-layout", Args: []string{"main-vertical"}},
		"M-5":   {Key: "M-5", Command: "select-layout", Args: []string{"tiled"}},

		// 复制粘贴
		"[": {Key: "[", Command: "copy-mode"},
		"]": {Key: "]", Command: "paste-buffer"},
		"#": {Key: "#", Command: "list-buffers"},
		"=": {Key: "=", Command: "choose-buffer"},
		"-": {Key: "-", Command: "delete-buffer"},

		// 其他功能
		"m":   {Key: "m", Command: "select-pane", Args: []string{"-m"}},
		"M":   {Key: "M", Command: "select-pane", Args: []string{"-M"}},
		"C-o": {Key: "C-o", Command: "rotate-window"},
		"C-z": {Key: "C-z", Command: "suspend-client"},
	}

	// 数字键绑定 (0-9)
	for i := 0; i <= 9; i++ {
		key := strconv.Itoa(i)
		basicBindings[key] = &KeyBinding{
			Key:     key,
			Command: "select-window",
			Args:    []string{"-t", ":" + key},
		}
	}

	// 功能键绑定 (F1-F12)
	for i := 1; i <= 12; i++ {
		key := fmt.Sprintf("F%d", i)
		basicBindings[key] = &KeyBinding{
			Key:     key,
			Command: "select-window",
			Args:    []string{"-t", ":" + strconv.Itoa(i-1)},
		}
	}

	p.keyBindings = basicBindings
}

// ParseTmuxCommand 解析tmux风格命令
func (p *EnhancedParser) ParseTmuxCommand(input string) (*CommandList, error) {
	if input == "" {
		return nil, fmt.Errorf("empty command input")
	}

	// 预处理：展开别名
	input = p.expandTmuxAliases(input)

	// 预处理：处理格式变量
	input = p.expandFormatVars(input)

	// 使用基础解析器解析
	return p.ModernParser.Parse(input)
}

// expandTmuxAliases 展开tmux命令别名
func (p *EnhancedParser) expandTmuxAliases(input string) string {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return input
	}

	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// 检查第一个单词是否是别名
	if alias, exists := p.tmuxAliases[parts[0]]; exists {
		parts[0] = alias
		p.logger.Debug("Expanded tmux alias: %s -> %s", input, strings.Join(parts, " "))
		return strings.Join(parts, " ")
	}

	return input
}

// expandFormatVars 展开tmux格式变量
func (p *EnhancedParser) expandFormatVars(input string) string {
	// 使用正则表达式查找格式变量
	re := regexp.MustCompile(`#\{([^}]+)\}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		// 提取变量名
		varName := match[2 : len(match)-1] // 去掉 #{ 和 }

		p.mutex.RLock()
		defer p.mutex.RUnlock()

		// 查找变量映射
		if replacement, exists := p.formatVars[varName]; exists {
			return replacement
		}

		// 如果找不到映射，返回原始字符串
		return match
	})
}

// HandleKeyBinding 处理快捷键绑定
func (p *EnhancedParser) HandleKeyBinding(key string) (*CommandList, error) {
	p.mutex.RLock()
	binding, exists := p.keyBindings[key]
	p.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("key binding not found: %s", key)
	}

	// 构建命令字符串
	cmdStr := binding.Command
	if len(binding.Args) > 0 {
		cmdStr += " " + strings.Join(binding.Args, " ")
	}

	p.logger.Debug("Handling key binding: %s -> %s", key, cmdStr)

	// 解析并返回命令
	return p.ParseTmuxCommand(cmdStr)
}

// RegisterTmuxAlias 注册新的tmux别名
func (p *EnhancedParser) RegisterTmuxAlias(alias, command string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.tmuxAliases[alias] = command
	p.logger.Debug("Registered tmux alias: %s -> %s", alias, command)
}

// RegisterKeyBinding 注册新的快捷键绑定
func (p *EnhancedParser) RegisterKeyBinding(key, command string, args ...string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.keyBindings[key] = &KeyBinding{
		Key:     key,
		Command: command,
		Args:    args,
	}
	p.logger.Debug("Registered key binding: %s -> %s %v", key, command, args)
}

// SetPrefixKey 设置前缀键
func (p *EnhancedParser) SetPrefixKey(prefix string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.prefixKey = prefix
	p.logger.Debug("Set prefix key: %s", prefix)
}

// GetPrefixKey 获取前缀键
func (p *EnhancedParser) GetPrefixKey() string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.prefixKey
}

// EnableTmuxMode 启用tmux兼容模式
func (p *EnhancedParser) EnableTmuxMode() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.tmuxMode = true
	p.logger.Info("Tmux compatibility mode enabled")
}

// DisableTmuxMode 禁用tmux兼容模式
func (p *EnhancedParser) DisableTmuxMode() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.tmuxMode = false
	p.logger.Info("Tmux compatibility mode disabled")
}

// IsTmuxMode 检查是否处于tmux兼容模式
func (p *EnhancedParser) IsTmuxMode() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.tmuxMode
}

// GetSupportedTmuxCommands 获取支持的tmux命令列表
func (p *EnhancedParser) GetSupportedTmuxCommands() []string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var commands []string
	for alias, command := range p.tmuxAliases {
		commands = append(commands, alias+" -> "+command)
	}

	return commands
}

// GetKeyBindings 获取所有快捷键绑定
func (p *EnhancedParser) GetKeyBindings() map[string]*KeyBinding {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// 返回副本避免并发修改
	result := make(map[string]*KeyBinding)
	for k, v := range p.keyBindings {
		result[k] = &KeyBinding{
			Key:     v.Key,
			Command: v.Command,
			Args:    append([]string{}, v.Args...), // 深度复制args
		}
	}

	return result
}
