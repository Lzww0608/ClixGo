/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-24 20:23:40
* @Description: 终端用户界面相关的类型定义
 */

package ui

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// UIManager 在manager.go中定义

// Layout 布局管理器
type Layout struct {
	root      *tview.Flex
	mainArea  *tview.Flex
	statusBar *tview.TextView
	sidebar   *Sidebar // 添加侧边栏支持
	mode      LayoutMode
	panels    []*Panel
}

// LayoutMode 布局模式
type LayoutMode int

const (
	LayoutSingle                LayoutMode = iota // 单面板
	LayoutVertical                                // 垂直分割
	LayoutHorizontal                              // 水平分割
	LayoutGrid                                    // 网格布局
	LayoutSingleWithSidebar                       // 单面板+侧边栏
	LayoutVerticalWithSidebar                     // 垂直分割+侧边栏
	LayoutHorizontalWithSidebar                   // 水平分割+侧边栏
	LayoutGridWithSidebar                         // 网格布局+侧边栏
	LayoutCustom                                  // Step 4: 自定义布局
	LayoutFloating                                // Step 4: 浮动布局
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

	// Step 4: 布局管理增强字段
	Resizable    bool         // 是否可调整大小
	Draggable    bool         // 是否可拖拽
	MinSize      Size         // 最小尺寸
	MaxSize      Size         // 最大尺寸
	OriginalSize Size         // 原始尺寸（用于恢复）
	ZIndex       int          // Z轴层级（浮动布局用）
	Constraints  *Constraints // 布局约束
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

// Step 4: 新增类型定义

// Constraints 面板约束
type Constraints struct {
	FixedWidth   bool    // 固定宽度
	FixedHeight  bool    // 固定高度
	AspectRatio  float64 // 宽高比约束
	AlignX       Align   // 水平对齐
	AlignY       Align   // 垂直对齐
	MarginTop    int     // 上边距
	MarginBottom int     // 下边距
	MarginLeft   int     // 左边距
	MarginRight  int     // 右边距
}

// Align 对齐方式
type Align int

const (
	AlignStart   Align = iota // 起始对齐
	AlignCenter               // 居中对齐
	AlignEnd                  // 结束对齐
	AlignStretch              // 拉伸对齐
)

// LayoutConfig 布局配置
type LayoutConfig struct {
	Name           string                 // 布局名称
	Mode           LayoutMode             // 布局模式
	SidebarVisible bool                   // 侧边栏可见性
	SidebarWidth   int                    // 侧边栏宽度
	PanelLayouts   []PanelLayoutConfig    // 面板布局配置
	GridRows       int                    // 网格行数
	GridCols       int                    // 网格列数
	CustomSettings map[string]interface{} // 自定义设置
	CreatedAt      time.Time              // 创建时间
	LastModified   time.Time              // 最后修改时间
}

// PanelLayoutConfig 面板布局配置
type PanelLayoutConfig struct {
	PanelID     string      // 面板ID
	Position    Position    // 位置
	Size        Size        // 尺寸
	Constraints Constraints // 约束
	ZIndex      int         // 层级
}

// ResizeHandle 调整手柄
type ResizeHandle struct {
	PanelID  string     // 关联面板ID
	Type     ResizeType // 调整类型
	Position Position   // 手柄位置
	Size     Size       // 手柄大小
	Active   bool       // 是否激活
	Cursor   CursorType // 鼠标样式
}

// ResizeType 调整类型
type ResizeType int

const (
	ResizeNone ResizeType = iota
	ResizeN               // 北（上）
	ResizeS               // 南（下）
	ResizeE               // 东（右）
	ResizeW               // 西（左）
	ResizeNE              // 东北
	ResizeNW              // 西北
	ResizeSE              // 东南
	ResizeSW              // 西南
)

// CursorType 鼠标样式
type CursorType int

const (
	CursorDefault CursorType = iota
	CursorResize
	CursorMove
	CursorNSResize   // 南北调整
	CursorEWResize   // 东西调整
	CursorNESWResize // 东北-西南调整
	CursorNWSEResize // 西北-东南调整
)

// DragState 拖拽状态
type DragState struct {
	Active     bool     // 是否正在拖拽
	PanelID    string   // 拖拽的面板ID
	StartPos   Position // 开始位置
	CurrentPos Position // 当前位置
	Offset     Position // 偏移量
	Type       DragType // 拖拽类型
}

// DragType 拖拽类型
type DragType int

const (
	DragMove   DragType = iota // 移动
	DragResize                 // 调整大小
)

// LayoutManager 布局管理器接口
type LayoutManager interface {
	// 基础布局操作
	ApplyLayout(config LayoutConfig) error
	GetCurrentLayout() LayoutConfig
	ResetLayout() error

	// 面板操作
	ResizePanel(panelID string, newSize Size) error
	MovePanel(panelID string, newPos Position) error
	SetPanelConstraints(panelID string, constraints Constraints) error

	// 拖拽操作
	StartDrag(panelID string, startPos Position, dragType DragType) error
	UpdateDrag(currentPos Position) error
	EndDrag() error

	// 布局保存/恢复
	SaveLayout(name string) error
	LoadLayout(name string) error
	ListLayouts() []string
	DeleteLayout(name string) error
}

// LayoutEvent 布局事件
type LayoutEvent struct {
	Type      LayoutEventType // 事件类型
	PanelID   string          // 面板ID
	OldValue  interface{}     // 旧值
	NewValue  interface{}     // 新值
	Timestamp time.Time       // 时间戳
}

// LayoutEventType 布局事件类型
type LayoutEventType int

const (
	LayoutEventPanelMoved LayoutEventType = iota
	LayoutEventPanelResized
	LayoutEventLayoutChanged
	LayoutEventSidebarToggled
	LayoutEventDragStarted
	LayoutEventDragEnded
)
