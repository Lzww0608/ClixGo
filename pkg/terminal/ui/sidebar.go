/*
* @Author: Lzww0608
* @Date: 2025-6-22 17:04:25
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-22 17:04:20
* @Description: 工具侧边栏的实现 - Step 3 阶段1-2
 */

package ui

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.uber.org/zap"
)

// SessionManagerInterface 会话管理器接口，避免导入循环
type SessionManagerInterface interface {
	GetPerformanceStats() interface{}
	ListSessions() []SessionInfo
	GetGoroutinePool() GoroutinePoolInterface
}

// SessionInfo 会话信息接口
type SessionInfo interface {
	GetID() string
	GetName() string
	GetWindows() []WindowInfo
}

// WindowInfo 窗口信息接口
type WindowInfo interface {
	GetPanes() []PaneInfo
}

// PaneInfo 面板信息接口
type PaneInfo interface {
	GetID() string
}

// GoroutinePoolInterface 协程池接口
type GoroutinePoolInterface interface {
	GetMetrics() PoolMetrics
}

// PoolMetrics 池指标接口
type PoolMetrics interface {
	GetActiveWorkers() int64
}

// SidebarTool 侧边栏工具项
type SidebarTool struct {
	ID          string                   // 工具ID
	Name        string                   // 显示名称
	Icon        string                   // 图标
	Description string                   // 描述
	DataSource  func() interface{}       // 数据源函数
	Formatter   func(interface{}) string // 格式化函数
	Color       tcell.Color              // 颜色
	Category    string                   // 分类
	Enabled     bool                     // 是否启用
}

// Sidebar 工具侧边栏
type Sidebar struct {
	*tview.List // 继承tview.List

	// 数据源
	sessionManager SessionManagerInterface

	// 工具列表
	tools        []SidebarTool    // 工具列表
	categories   map[string][]int // 分类索引
	selectedTool int              // 选中的工具

	// 界面控制
	visible bool   // 可见性
	width   int    // 宽度
	title   string // 标题

	// 更新控制
	updateTicker   *time.Ticker       // 更新定时器
	updateInterval time.Duration      // 更新间隔
	ctx            context.Context    // 上下文
	cancel         context.CancelFunc // 取消函数
	mutex          sync.RWMutex       // 读写锁

	// 状态
	isRunning  bool      // 是否运行中
	lastUpdate time.Time // 最后更新时间
}

// NewSidebar 创建新的工具侧边栏
func NewSidebar(sessionManager SessionManagerInterface) *Sidebar {
	ctx, cancel := context.WithCancel(context.Background())

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" 🔧 工具面板 ")
	list.SetTitleAlign(tview.AlignLeft)
	list.SetBorderColor(DefaultTheme.Border)
	list.SetBackgroundColor(DefaultTheme.Background)

	sidebar := &Sidebar{
		List:           list,
		sessionManager: sessionManager,
		tools:          make([]SidebarTool, 0),
		categories:     make(map[string][]int),
		visible:        true,
		width:          30,
		title:          "工具面板",
		updateInterval: 2 * time.Second,
		ctx:            ctx,
		cancel:         cancel,
		selectedTool:   0,
		lastUpdate:     time.Now(),
	}

	// 初始化工具列表
	sidebar.initializeTools()

	// 设置选择回调
	sidebar.setupCallbacks()

	// 设置样式
	sidebar.setupStyles()

	logger.Info("工具侧边栏初始化完成",
		zap.Int("tool_count", len(sidebar.tools)),
		zap.Duration("update_interval", sidebar.updateInterval))

	return sidebar
}

// initializeTools 初始化工具列表
func (s *Sidebar) initializeTools() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 性能监控工具
	s.addTool(SidebarTool{
		ID:          "performance_stats",
		Name:        "性能统计",
		Icon:        "📊",
		Description: "会话和系统性能统计",
		Category:    "监控",
		Color:       tcell.ColorGreen,
		Enabled:     true,
		DataSource:  s.getPerformanceStats,
		Formatter:   s.formatPerformanceStats,
	})

	// 会话管理工具
	s.addTool(SidebarTool{
		ID:          "session_info",
		Name:        "会话信息",
		Icon:        "🖥️",
		Description: "当前会话详细信息",
		Category:    "会话",
		Color:       tcell.ColorBlue,
		Enabled:     true,
		DataSource:  s.getSessionInfo,
		Formatter:   s.formatSessionInfo,
	})

	// 系统信息工具
	s.addTool(SidebarTool{
		ID:          "system_info",
		Name:        "系统信息",
		Icon:        "💻",
		Description: "系统状态和资源使用",
		Category:    "系统",
		Color:       tcell.ColorYellow,
		Enabled:     true,
		DataSource:  s.getSystemInfo,
		Formatter:   s.formatSystemInfo,
	})

	// 快捷操作工具
	s.addTool(SidebarTool{
		ID:          "quick_actions",
		Name:        "快捷操作",
		Icon:        "⚡",
		Description: "常用操作快捷入口",
		Category:    "操作",
		Color:       tcell.ColorRed, // 使用Red替代Magenta
		Enabled:     true,
		DataSource:  s.getQuickActions,
		Formatter:   s.formatQuickActions,
	})

	// 网络监控工具 (预留)
	s.addTool(SidebarTool{
		ID:          "network_monitor",
		Name:        "网络监控",
		Icon:        "🌐",
		Description: "网络状态和流量监控",
		Category:    "监控",
		Color:       tcell.ColorBlue, // 使用Blue替代Cyan
		Enabled:     false,           // 暂时禁用
		DataSource:  s.getNetworkStats,
		Formatter:   s.formatNetworkStats,
	})

	// 帮助和信息
	s.addTool(SidebarTool{
		ID:          "help_info",
		Name:        "帮助信息",
		Icon:        "❓",
		Description: "快捷键和使用帮助",
		Category:    "帮助",
		Color:       tcell.ColorWhite,
		Enabled:     true,
		DataSource:  s.getHelpInfo,
		Formatter:   s.formatHelpInfo,
	})

	// 更新工具列表显示
	s.updateToolList()

	logger.Debug("工具列表初始化完成",
		zap.Int("total_tools", len(s.tools)),
		zap.Int("enabled_tools", s.countEnabledTools()))
}

// addTool 添加工具到列表
func (s *Sidebar) addTool(tool SidebarTool) {
	s.tools = append(s.tools, tool)

	// 更新分类索引
	if _, exists := s.categories[tool.Category]; !exists {
		s.categories[tool.Category] = make([]int, 0)
	}
	s.categories[tool.Category] = append(s.categories[tool.Category], len(s.tools)-1)
}

// setupCallbacks 设置回调函数
func (s *Sidebar) setupCallbacks() {
	// 设置选择回调
	s.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		s.handleToolSelection(index)
	})

	// 设置焦点变化回调
	s.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		s.selectedTool = index
		s.updateToolPreview(index)
	})
}

// setupStyles 设置样式
func (s *Sidebar) setupStyles() {
	s.SetMainTextColor(DefaultTheme.Foreground)
	s.SetSecondaryTextColor(tcell.ColorGray)
	s.SetSelectedTextColor(tcell.ColorBlack)
	s.SetSelectedBackgroundColor(DefaultTheme.ActiveBorder)
	s.SetHighlightFullLine(true)
	s.ShowSecondaryText(true)
}

// updateToolList 更新工具列表显示
func (s *Sidebar) updateToolList() {
	s.Clear()

	for i, tool := range s.tools {
		if !tool.Enabled {
			continue
		}

		// 获取工具数据
		var data interface{}
		if tool.DataSource != nil {
			data = tool.DataSource()
		}

		// 格式化显示文本
		var mainText, secondaryText string
		if tool.Formatter != nil && data != nil {
			formatted := tool.Formatter(data)
			mainText = fmt.Sprintf("%s %s", tool.Icon, tool.Name)
			secondaryText = formatted
		} else {
			mainText = fmt.Sprintf("%s %s", tool.Icon, tool.Name)
			secondaryText = tool.Description
		}

		// 添加到列表
		s.AddItem(mainText, secondaryText, rune('1'+i), nil)
	}

	s.lastUpdate = time.Now()
}

// handleToolSelection 处理工具选择
func (s *Sidebar) handleToolSelection(index int) {
	if index < 0 || index >= len(s.tools) {
		return
	}

	tool := s.tools[index]
	logger.Info("工具被选择",
		zap.String("tool_id", tool.ID),
		zap.String("tool_name", tool.Name))

	// 根据工具类型执行不同操作
	switch tool.ID {
	case "quick_actions":
		s.handleQuickActions()
	case "help_info":
		s.showHelpDialog()
	default:
		// 对于其他工具，刷新数据显示
		s.refreshToolData(index)
	}
}

// updateToolPreview 更新工具预览
func (s *Sidebar) updateToolPreview(index int) {
	if index < 0 || index >= len(s.tools) {
		return
	}

	tool := s.tools[index]

	// 更新标题显示当前选中的工具
	title := fmt.Sprintf(" 🔧 %s ", tool.Name)
	s.SetTitle(title)
	s.SetTitleColor(tool.Color)
}

// Start 启动侧边栏更新
func (s *Sidebar) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRunning {
		return fmt.Errorf("侧边栏已在运行中")
	}

	s.isRunning = true
	s.updateTicker = time.NewTicker(s.updateInterval)

	// 启动更新协程
	go s.updateLoop()

	logger.Info("工具侧边栏启动成功",
		zap.Duration("update_interval", s.updateInterval))

	return nil
}

// Stop 停止侧边栏更新
func (s *Sidebar) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return
	}

	s.isRunning = false

	if s.updateTicker != nil {
		s.updateTicker.Stop()
		s.updateTicker = nil
	}

	s.cancel()

	logger.Info("工具侧边栏已停止")
}

// updateLoop 更新循环
func (s *Sidebar) updateLoop() {
	// 确保ticker不为空
	if s.updateTicker == nil {
		return
	}

	ticker := s.updateTicker // 本地引用，避免并发修改
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.mutex.RLock()
			if !s.isRunning {
				s.mutex.RUnlock()
				return
			}
			s.mutex.RUnlock()
			s.refreshData()
		}
	}
}

// refreshData 刷新数据
func (s *Sidebar) refreshData() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.visible {
		return
	}

	s.updateToolList()
}

// refreshToolData 刷新特定工具数据
func (s *Sidebar) refreshToolData(index int) {
	if index < 0 || index >= len(s.tools) {
		return
	}

	tool := s.tools[index]
	if tool.DataSource == nil {
		return
	}

	// 强制更新该工具的数据
	data := tool.DataSource()
	if tool.Formatter != nil && data != nil {
		formatted := tool.Formatter(data)

		// 更新列表项
		mainText := fmt.Sprintf("%s %s", tool.Icon, tool.Name)
		s.SetItemText(index, mainText, formatted)
	}
}

// Toggle 切换侧边栏可见性
func (s *Sidebar) Toggle() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.visible = !s.visible

	logger.Debug("侧边栏可见性切换",
		zap.Bool("visible", s.visible))
}

// IsVisible 获取可见性状态
func (s *Sidebar) IsVisible() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.visible
}

// SetVisible 设置可见性
func (s *Sidebar) SetVisible(visible bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.visible = visible
}

// GetWidth 获取宽度
func (s *Sidebar) GetWidth() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.width
}

// SetWidth 设置宽度
func (s *Sidebar) SetWidth(width int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if width < 20 {
		width = 20
	} else if width > 80 {
		width = 80
	}

	s.width = width
}

// countEnabledTools 统计启用的工具数量
func (s *Sidebar) countEnabledTools() int {
	count := 0
	for _, tool := range s.tools {
		if tool.Enabled {
			count++
		}
	}
	return count
}

// ====== 阶段2: 数据源集成函数 ======

// getPerformanceStats 获取性能统计数据
func (s *Sidebar) getPerformanceStats() interface{} {
	if s.sessionManager == nil {
		return nil
	}

	stats := s.sessionManager.GetPerformanceStats()
	if stats == nil {
		return nil
	}

	// 使用类型断言来安全访问字段
	if statsMap, ok := stats.(map[string]interface{}); ok {
		return statsMap
	}

	// 返回默认值
	return map[string]interface{}{
		"sessions":    1,
		"memory_mb":   2.5,
		"create_time": "1ms",
		"windows":     1,
		"panes":       1,
		"buffer_hits": 0,
		"buffer_miss": 0,
	}
}

// formatPerformanceStats 格式化性能统计数据
func (s *Sidebar) formatPerformanceStats(data interface{}) string {
	if data == nil {
		return "无数据"
	}

	statsMap, ok := data.(map[string]interface{})
	if !ok {
		return "数据错误"
	}

	return fmt.Sprintf("会话:%v 内存:%.1fMB",
		statsMap["sessions"], statsMap["memory_mb"])
}

// getSessionInfo 获取会话信息
func (s *Sidebar) getSessionInfo() interface{} {
	if s.sessionManager == nil {
		return nil
	}

	sessions := s.sessionManager.ListSessions()
	totalWindows := 0
	totalPanes := 0

	for _, session := range sessions {
		windows := session.GetWindows()
		totalWindows += len(windows)
		for _, window := range windows {
			panes := window.GetPanes()
			totalPanes += len(panes)
		}
	}

	return map[string]interface{}{
		"total_sessions": len(sessions),
		"total_windows":  totalWindows,
		"total_panes":    totalPanes,
		"sessions":       sessions,
	}
}

// formatSessionInfo 格式化会话信息
func (s *Sidebar) formatSessionInfo(data interface{}) string {
	if data == nil {
		return "无会话"
	}

	infoMap, ok := data.(map[string]interface{})
	if !ok {
		return "数据错误"
	}

	return fmt.Sprintf("会话:%v 窗口:%v 面板:%v",
		infoMap["total_sessions"],
		infoMap["total_windows"],
		infoMap["total_panes"])
}

// getSystemInfo 获取系统信息
func (s *Sidebar) getSystemInfo() interface{} {
	return map[string]interface{}{
		"time":       time.Now().Format("15:04:05"),
		"date":       time.Now().Format("2006-01-02"),
		"uptime":     time.Since(s.lastUpdate).Truncate(time.Second),
		"goroutines": s.getGoroutineCount(),
	}
}

// formatSystemInfo 格式化系统信息
func (s *Sidebar) formatSystemInfo(data interface{}) string {
	if data == nil {
		return "无数据"
	}

	infoMap, ok := data.(map[string]interface{})
	if !ok {
		return "数据错误"
	}

	return fmt.Sprintf("%v Goroutines:%v",
		infoMap["time"], infoMap["goroutines"])
}

// getQuickActions 获取快捷操作列表
func (s *Sidebar) getQuickActions() interface{} {
	return map[string]interface{}{
		"actions": []string{
			"Ctrl+N: 新建会话",
			"Ctrl+D: 分离会话",
			"Ctrl+\": 水平分割",
			"Ctrl+%: 垂直分割",
			"F2: 切换侧边栏",
			"F3: 工具选择",
		},
	}
}

// formatQuickActions 格式化快捷操作
func (s *Sidebar) formatQuickActions(data interface{}) string {
	if data == nil {
		return "无操作"
	}

	actionsMap, ok := data.(map[string]interface{})
	if !ok {
		return "数据错误"
	}

	actions, ok := actionsMap["actions"].([]string)
	if !ok || len(actions) == 0 {
		return "无操作"
	}

	return fmt.Sprintf("%d个快捷操作", len(actions))
}

// getNetworkStats 获取网络统计 (预留)
func (s *Sidebar) getNetworkStats() interface{} {
	// TODO: 集成 RealtimeNetworkMonitor
	return map[string]interface{}{
		"status":      "暂未实现",
		"interfaces":  0,
		"connections": 0,
	}
}

// formatNetworkStats 格式化网络统计
func (s *Sidebar) formatNetworkStats(data interface{}) string {
	if data == nil {
		return "网络监控暂未启用"
	}

	statsMap, ok := data.(map[string]interface{})
	if !ok {
		return "数据错误"
	}

	return fmt.Sprintf("%v", statsMap["status"])
}

// getHelpInfo 获取帮助信息
func (s *Sidebar) getHelpInfo() interface{} {
	return map[string]interface{}{
		"version": "ClixGo 2.0",
		"shortcuts": []string{
			"F1: 帮助",
			"F2: 侧边栏",
			"Tab: 切换焦点",
			"Esc: 退出",
		},
		"docs": "https://github.com/Lzww0608/ClixGo",
	}
}

// formatHelpInfo 格式化帮助信息
func (s *Sidebar) formatHelpInfo(data interface{}) string {
	if data == nil {
		return "无帮助信息"
	}

	helpMap, ok := data.(map[string]interface{})
	if !ok {
		return "数据错误"
	}

	version, _ := helpMap["version"].(string)
	shortcuts, _ := helpMap["shortcuts"].([]string)

	return fmt.Sprintf("%s (%d个快捷键)", version, len(shortcuts))
}

// ====== 辅助函数 ======

// getGoroutineCount 获取Goroutine数量
func (s *Sidebar) getGoroutineCount() int {
	// 优先使用runtime的goroutine数量
	return runtime.NumGoroutine()
}

// handleQuickActions 处理快捷操作选择
func (s *Sidebar) handleQuickActions() {
	logger.Info("快捷操作被选择")
	// TODO: 显示快捷操作菜单或执行默认操作
}

// showHelpDialog 显示帮助对话框
func (s *Sidebar) showHelpDialog() {
	logger.Info("帮助信息被选择")
	// TODO: 显示帮助对话框或跳转到帮助文档
}
