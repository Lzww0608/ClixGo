/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 终端用户界面相关的类型定义
 */

package ui

import (
	"context"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// UIManager 管理终端UI渲染
type UIManager struct {
	app        *tview.Application
	screen     tcell.Screen
	layout     *Layout
	statusBar  *StatusBar
	panels     map[string]*Panel
	activePane string
	mutex      sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	keyBinds   map[tcell.Key]KeyHandler
	mouseMode  bool
}

// Layout 布局管理器
type Layout struct {
	root      *tview.Flex
	mainArea  *tview.Flex
	statusBar *tview.TextView
	mode      LayoutMode
	panels    []*Panel
}

// LayoutMode 布局模式
type LayoutMode int

const (
	LayoutSingle     LayoutMode = iota // 单面板
	LayoutVertical                     // 垂直分割
	LayoutHorizontal                   // 水平分割
	LayoutGrid                         // 网格布局
)

// Panel 面板
type Panel struct {
	ID         string
	Title      string
	Content    *tview.TextView
	Border     bool
	Active     bool
	Position   Position
	Size       Size
	LastUpdate time.Time
	ScrollPos  int
	MaxLines   int
	AutoScroll bool
}

// Position 位置信息
type Position struct {
	X, Y int
}

// Size 尺寸信息
type Size struct {
	Width, Height int
}

// StatusBar 状态栏
type StatusBar struct {
	view    *tview.TextView
	left    string
	center  string
	right   string
	style   tcell.Style
	visible bool
}

// KeyHandler 按键处理函数
type KeyHandler func(event *tcell.EventKey) *tcell.EventKey

// MouseHandler 鼠标处理函数
type MouseHandler func(event *tcell.EventMouse) *tcell.EventMouse

// UIConfig UI配置
type UIConfig struct {
	Theme          Theme
	StatusBarStyle StatusBarStyle
	PanelStyle     PanelStyle
	KeyBindings    map[string]string
	MouseEnabled   bool
	RefreshRate    time.Duration
}

// Theme 主题配置
type Theme struct {
	Background   tcell.Color
	Foreground   tcell.Color
	Border       tcell.Color
	ActiveBorder tcell.Color
	StatusBar    tcell.Color
	StatusText   tcell.Color
}

// StatusBarStyle 状态栏样式
type StatusBarStyle struct {
	Format    string
	ShowTime  bool
	ShowStats bool
}

// PanelStyle 面板样式
type PanelStyle struct {
	BorderStyle  tcell.Style
	TitleStyle   tcell.Style
	ContentStyle tcell.Style
}

// UIEvent UI事件
type UIEvent struct {
	Type      EventType
	Panel     string
	Data      interface{}
	Timestamp time.Time
}

// EventType 事件类型
type EventType int

const (
	EventPanelFocus EventType = iota
	EventPanelResize
	EventPanelClose
	EventKeyPress
	EventMouseClick
	EventRefresh
)

// 默认主题
var DefaultTheme = Theme{
	Background:   tcell.ColorBlack,
	Foreground:   tcell.ColorWhite,
	Border:       tcell.ColorGray,
	ActiveBorder: tcell.ColorBlue,
	StatusBar:    tcell.ColorDarkBlue,
	StatusText:   tcell.ColorWhite,
}

// 默认配置
var DefaultUIConfig = UIConfig{
	Theme:        DefaultTheme,
	MouseEnabled: true,
	RefreshRate:  time.Millisecond * 100,
	KeyBindings: map[string]string{
		"Ctrl+C": "quit",
		"Ctrl+D": "detach",
		"Ctrl+N": "new_panel",
		"Ctrl+W": "close_panel",
		"Tab":    "next_panel",
		"Ctrl+H": "split_horizontal",
		"Ctrl+V": "split_vertical",
	},
}
