package terminal

import (
	"context"
	"io"
	"os"
	"sync"
	"time"
)

// SessionStatus 会话状态
type SessionStatus string

const (
	SessionActive    SessionStatus = "active"
	SessionDetached  SessionStatus = "detached"
	SessionDestroyed SessionStatus = "destroyed"
)

// Session 会话结构
type Session struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Status       SessionStatus `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
	LastActive   time.Time     `json:"last_active"`
	Windows      []*Window     `json:"windows"`
	ActiveWindow int           `json:"active_window"`
	mutex        sync.RWMutex
}

// Window 窗口结构
type Window struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Index      int       `json:"index"`
	Panes      []*Pane   `json:"panes"`
	ActivePane int       `json:"active_pane"`
	Layout     Layout    `json:"layout"`
	CreatedAt  time.Time `json:"created_at"`
	mutex      sync.RWMutex
}

// Pane 面板结构
type Pane struct {
	ID         string      `json:"id"`
	Index      int         `json:"index"`
	X          int         `json:"x"`
	Y          int         `json:"y"`
	Width      int         `json:"width"`
	Height     int         `json:"height"`
	Command    string      `json:"command"`
	WorkingDir string      `json:"working_dir"`
	Process    *os.Process `json:"-"`
	ProcessID  int         `json:"process_id"`
	Input      io.Writer   `json:"-"`
	Output     io.Reader   `json:"-"`
	Buffer     *Buffer     `json:"-"`
	Active     bool        `json:"active"`
	CreatedAt  time.Time   `json:"created_at"`
	LastOutput time.Time   `json:"last_output"`
	mutex      sync.RWMutex
}

// Layout 布局类型
type Layout string

const (
	LayoutMainVertical   Layout = "main-vertical"
	LayoutMainHorizontal Layout = "main-horizontal"
	LayoutEven           Layout = "even"
	LayoutTiled          Layout = "tiled"
)

// Buffer 终端缓冲区
type Buffer struct {
	Lines    [][]rune `json:"lines"`
	MaxLines int      `json:"max_lines"`
	CursorX  int      `json:"cursor_x"`
	CursorY  int      `json:"cursor_y"`
	mutex    sync.RWMutex
}

// KeyBinding 快捷键绑定
type KeyBinding struct {
	Key     string   `json:"key"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// TerminalConfig 终端配置
type TerminalConfig struct {
	// 基础配置
	PrefixKey    string        `yaml:"prefix_key" json:"prefix_key"`
	MouseEnabled bool          `yaml:"mouse_enabled" json:"mouse_enabled"`
	StatusBar    bool          `yaml:"status_bar" json:"status_bar"`
	AutoSave     bool          `yaml:"auto_save" json:"auto_save"`
	SaveInterval time.Duration `yaml:"save_interval" json:"save_interval"`

	// 显示配置
	Theme        string `yaml:"theme" json:"theme"`
	StatusFormat string `yaml:"status_format" json:"status_format"`
	WindowFormat string `yaml:"window_format" json:"window_format"`

	// 缓冲区配置
	BufferSize int `yaml:"buffer_size" json:"buffer_size"`
	ScrollBack int `yaml:"scroll_back" json:"scroll_back"`

	// 快捷键配置
	KeyBindings []KeyBinding `yaml:"key_bindings" json:"key_bindings"`

	// 集成配置
	ClixGoIntegration bool `yaml:"clixgo_integration" json:"clixgo_integration"`
	NetworkMonitor    bool `yaml:"network_monitor" json:"network_monitor"`
	TaskIntegration   bool `yaml:"task_integration" json:"task_integration"`
}

// Server 服务器结构
type Server struct {
	Sessions   map[string]*Session `json:"sessions"`
	Config     *TerminalConfig     `json:"config"`
	SocketPath string              `json:"socket_path"`
	Running    bool                `json:"running"`
	mutex      sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// Client 客户端结构
type Client struct {
	SessionID  string          `json:"session_id"`
	Connected  bool            `json:"connected"`
	Config     *TerminalConfig `json:"config"`
	Input      chan []byte     `json:"-"`
	Output     chan []byte     `json:"-"`
	SocketPath string          `json:"socket_path"`
	mutex      sync.RWMutex
}

// Command 终端命令
type Command struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// 命令类型常量
const (
	CmdCreateSession = "create_session"
	CmdAttachSession = "attach_session"
	CmdDetachSession = "detach_session"
	CmdListSessions  = "list_sessions"
	CmdCreateWindow  = "create_window"
	CmdCloseWindow   = "close_window"
	CmdSplitPane     = "split_pane"
	CmdClosePane     = "close_pane"
	CmdSwitchWindow  = "switch_window"
	CmdSwitchPane    = "switch_pane"
	CmdResizePane    = "resize_pane"
	CmdSendKeys      = "send_keys"
	CmdCopyMode      = "copy_mode"
	CmdPasteBuffer   = "paste_buffer"
	CmdSetLayout     = "set_layout"
	CmdRename        = "rename"
	CmdKillSession   = "kill_session"
)

// 默认配置
var DefaultConfig = &TerminalConfig{
	PrefixKey:         "C-b",
	MouseEnabled:      true,
	StatusBar:         true,
	AutoSave:          true,
	SaveInterval:      time.Minute * 5,
	Theme:             "default",
	StatusFormat:      "[#S] #I:#W",
	WindowFormat:      "#I:#W",
	BufferSize:        2000,
	ScrollBack:        2000,
	ClixGoIntegration: true,
	NetworkMonitor:    false,
	TaskIntegration:   true,
	KeyBindings: []KeyBinding{
		{Key: "C-b d", Command: "detach_session"},
		{Key: "C-b c", Command: "create_window"},
		{Key: "C-b &", Command: "close_window"},
		{Key: "C-b \"", Command: "split_pane", Args: []string{"horizontal"}},
		{Key: "C-b %", Command: "split_pane", Args: []string{"vertical"}},
		{Key: "C-b x", Command: "close_pane"},
		{Key: "C-b o", Command: "switch_pane"},
		{Key: "C-b n", Command: "next_window"},
		{Key: "C-b p", Command: "previous_window"},
		{Key: "C-b l", Command: "last_window"},
		{Key: "C-b [", Command: "copy_mode"},
		{Key: "C-b ]", Command: "paste_buffer"},
		{Key: "C-b z", Command: "zoom_pane"},
		{Key: "C-b Space", Command: "next_layout"},
		{Key: "C-b ?", Command: "list_keys"},
	},
}
