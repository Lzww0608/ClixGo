/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-20 11:06:38
* @Description: 终端用户界面管理器的实现
 */

package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.uber.org/zap"
)

// UIManager 管理终端UI渲染
type UIManager struct {
	app        *tview.Application
	screen     tcell.Screen
	layout     *Layout
	statusBar  *StatusBar
	sidebar    *Sidebar // 添加侧边栏
	panels     map[string]*Panel
	activePane string
	mutex      sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	keyBinds   map[tcell.Key]KeyHandler
	mouseMode  bool
}

// NewUIManager 创建新的UI管理器
func NewUIManager(config UIConfig) (*UIManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	app := tview.NewApplication()

	ui := &UIManager{
		app:       app,
		panels:    make(map[string]*Panel),
		ctx:       ctx,
		cancel:    cancel,
		keyBinds:  make(map[tcell.Key]KeyHandler),
		mouseMode: config.MouseEnabled,
	}

	// 初始化布局
	if err := ui.initLayout(config); err != nil {
		cancel()
		return nil, fmt.Errorf("初始化布局失败: %w", err)
	}

	// 设置按键绑定
	ui.setupKeyBindings()

	// 设置鼠标支持
	if config.MouseEnabled {
		ui.enableMouse()
	}

	logger.Info("UI管理器初始化完成",
		zap.Bool("mouse_enabled", config.MouseEnabled),
		zap.Duration("refresh_rate", config.RefreshRate))

	return ui, nil
}

// initLayout 初始化布局
func (ui *UIManager) initLayout(config UIConfig) error {
	// 创建主布局
	root := tview.NewFlex().SetDirection(tview.FlexRow)
	mainArea := tview.NewFlex().SetDirection(tview.FlexColumn)

	// 创建状态栏
	statusBar := ui.createStatusBar(config.StatusBarStyle)

	// 设置布局
	root.AddItem(mainArea, 0, 1, true)
	root.AddItem(statusBar.view, 1, 0, false)

	ui.layout = &Layout{
		root:      root,
		mainArea:  mainArea,
		statusBar: statusBar.view,
		mode:      LayoutSingle,
		panels:    make([]*Panel, 0),
	}

	ui.statusBar = statusBar

	// 设置应用根组件
	ui.app.SetRoot(root, true)

	return nil
}

// createStatusBar 创建状态栏
func (ui *UIManager) createStatusBar(style StatusBarStyle) *StatusBar {
	view := tview.NewTextView()
	view.SetBackgroundColor(DefaultTheme.StatusBar)
	view.SetTextColor(DefaultTheme.StatusText)
	view.SetBorder(false)
	view.SetDynamicColors(true)

	statusBar := &StatusBar{
		view:    view,
		style:   tcell.StyleDefault.Background(DefaultTheme.StatusBar).Foreground(DefaultTheme.StatusText),
		visible: true,
	}

	// 初始状态栏内容
	statusBar.UpdateStatus("ClixGo Terminal", "Ready", "")

	return statusBar
}

// setupKeyBindings 设置按键绑定
func (ui *UIManager) setupKeyBindings() {
	ui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			ui.Quit()
			return nil
		case tcell.KeyCtrlD:
			ui.Detach()
			return nil
		case tcell.KeyCtrlN:
			ui.CreatePanel("new_panel", "New Panel")
			return nil
		case tcell.KeyCtrlW:
			ui.CloseActivePanel()
			return nil
		case tcell.KeyTab:
			ui.NextPanel()
			return nil
		case tcell.KeyCtrlH:
			ui.SplitHorizontal()
			return nil
		case tcell.KeyCtrlV:
			ui.SplitVertical()
			return nil
		case tcell.KeyF1:
			ui.ShowHelp()
			return nil
		}

		// 检查自定义按键绑定
		if handler, exists := ui.keyBinds[event.Key()]; exists {
			return handler(event)
		}

		return event
	})
}

// enableMouse 启用鼠标支持
func (ui *UIManager) enableMouse() {
	ui.app.EnableMouse(true)
	logger.Debug("鼠标支持已启用")
}

// Start 启动UI
func (ui *UIManager) Start() error {
	logger.Info("启动UI渲染系统")

	// 启动状态栏更新协程
	go ui.statusBarUpdater()

	// 运行应用
	if err := ui.app.Run(); err != nil {
		return fmt.Errorf("UI运行失败: %w", err)
	}

	return nil
}

// Stop 停止UI
func (ui *UIManager) Stop() {
	logger.Info("停止UI渲染系统")
	ui.cancel()
	ui.app.Stop()
}

// Quit 退出应用
func (ui *UIManager) Quit() {
	logger.Info("退出UI应用")
	ui.Stop()
}

// Detach 分离会话
func (ui *UIManager) Detach() {
	logger.Info("分离UI会话")
	ui.Stop()
}

// statusBarUpdater 状态栏更新器
func (ui *UIManager) statusBarUpdater() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ui.ctx.Done():
			return
		case <-ticker.C:
			ui.updateStatusBar()
		}
	}
}

// updateStatusBar 更新状态栏
func (ui *UIManager) updateStatusBar() {
	ui.mutex.RLock()
	panelCount := len(ui.panels)
	activePanel := ui.activePane
	ui.mutex.RUnlock()

	currentTime := time.Now().Format("15:04:05")
	center := fmt.Sprintf("Panels: %d | Active: %s", panelCount, activePanel)

	ui.statusBar.UpdateStatus("ClixGo Terminal", center, currentTime)
}

// CreatePanel 创建新面板
func (ui *UIManager) CreatePanel(id, title string) *Panel {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	// 检查面板是否已存在
	if _, exists := ui.panels[id]; exists {
		logger.Warn("面板已存在", zap.String("panel_id", id))
		return ui.panels[id]
	}

	// 创建面板内容
	content := tview.NewTextView()
	content.SetBorder(true)
	content.SetTitle(title)
	content.SetBorderColor(DefaultTheme.Border)
	content.SetTitleColor(DefaultTheme.Foreground)
	content.SetBackgroundColor(DefaultTheme.Background)
	content.SetTextColor(DefaultTheme.Foreground)
	content.SetDynamicColors(true)
	content.SetScrollable(true)
	content.SetWrap(true)

	panel := &Panel{
		ID:         id,
		Title:      title,
		Content:    content,
		Border:     true,
		Active:     false,
		LastUpdate: time.Now(),
		MaxLines:   1000,
		AutoScroll: true,
	}

	ui.panels[id] = panel
	ui.layout.panels = append(ui.layout.panels, panel)

	// 如果是第一个面板，设置为活动面板
	if len(ui.panels) == 1 {
		ui.activePane = id
		panel.Active = true
		ui.setActivePanel(panel)
	}

	// 更新布局
	ui.updateLayout()

	logger.Info("创建新面板",
		zap.String("panel_id", id),
		zap.String("title", title))

	return panel
}

// setActivePanel 设置活动面板
func (ui *UIManager) setActivePanel(panel *Panel) {
	// 重置所有面板的活动状态
	for _, p := range ui.panels {
		p.Active = false
		p.Content.SetBorderColor(DefaultTheme.Border)
	}

	// 设置当前面板为活动状态
	panel.Active = true
	panel.Content.SetBorderColor(DefaultTheme.ActiveBorder)
	ui.activePane = panel.ID

	// 设置焦点
	ui.app.SetFocus(panel.Content)
}

// updateLayout 更新布局
func (ui *UIManager) updateLayout() {
	ui.layout.mainArea.Clear()

	switch len(ui.layout.panels) {
	case 0:
		// 无面板，显示欢迎信息
		welcome := tview.NewTextView()
		welcome.SetText("Welcome to ClixGo Terminal\nPress Ctrl+N to create a new panel")
		welcome.SetTextAlign(tview.AlignCenter)
		welcome.SetBorder(true)
		welcome.SetTitle("ClixGo")
		ui.layout.mainArea.AddItem(welcome, 0, 1, true)

	case 1:
		// 单面板模式
		panel := ui.layout.panels[0]
		ui.layout.mainArea.AddItem(panel.Content, 0, 1, true)
		ui.layout.mode = LayoutSingle

	case 2:
		// 垂直分割模式
		ui.layout.mainArea.SetDirection(tview.FlexColumn)
		for _, panel := range ui.layout.panels {
			ui.layout.mainArea.AddItem(panel.Content, 0, 1, panel.Active)
		}
		ui.layout.mode = LayoutVertical

	default:
		// 网格模式
		ui.layoutGrid()
		ui.layout.mode = LayoutGrid
	}
}

// layoutGrid 网格布局
func (ui *UIManager) layoutGrid() {
	panelCount := len(ui.layout.panels)
	if panelCount <= 2 {
		return
	}

	// 计算网格尺寸
	cols := 2
	rows := (panelCount + 1) / 2

	ui.layout.mainArea.SetDirection(tview.FlexRow)

	for row := 0; row < rows; row++ {
		rowFlex := tview.NewFlex().SetDirection(tview.FlexColumn)

		for col := 0; col < cols; col++ {
			index := row*cols + col
			if index < panelCount {
				panel := ui.layout.panels[index]
				rowFlex.AddItem(panel.Content, 0, 1, panel.Active)
			}
		}

		ui.layout.mainArea.AddItem(rowFlex, 0, 1, false)
	}
}

// WriteToPanel 向面板写入内容
func (ui *UIManager) WriteToPanel(panelID, content string) error {
	ui.mutex.RLock()
	panel, exists := ui.panels[panelID]
	ui.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("面板不存在: %s", panelID)
	}

	// 更新面板内容
	currentText := panel.Content.GetText(false)
	newText := currentText + content

	// 限制行数
	lines := strings.Split(newText, "\n")
	if len(lines) > panel.MaxLines {
		lines = lines[len(lines)-panel.MaxLines:]
		newText = strings.Join(lines, "\n")
	}

	panel.Content.SetText(newText)
	panel.LastUpdate = time.Now()

	// 自动滚动到底部
	if panel.AutoScroll {
		panel.Content.ScrollToEnd()
	}

	return nil
}

// NextPanel 切换到下一个面板
func (ui *UIManager) NextPanel() {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	if len(ui.panels) <= 1 {
		return
	}

	// 找到当前活动面板的索引
	currentIndex := -1
	for i, panel := range ui.layout.panels {
		if panel.ID == ui.activePane {
			currentIndex = i
			break
		}
	}

	// 切换到下一个面板
	nextIndex := (currentIndex + 1) % len(ui.layout.panels)
	nextPanel := ui.layout.panels[nextIndex]
	ui.setActivePanel(nextPanel)

	logger.Debug("切换面板",
		zap.String("from", ui.activePane),
		zap.String("to", nextPanel.ID))
}

// CloseActivePanel 关闭活动面板
func (ui *UIManager) CloseActivePanel() {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	if ui.activePane == "" || len(ui.panels) == 0 {
		return
	}

	// 删除面板
	delete(ui.panels, ui.activePane)

	// 从布局中移除
	for i, panel := range ui.layout.panels {
		if panel.ID == ui.activePane {
			ui.layout.panels = append(ui.layout.panels[:i], ui.layout.panels[i+1:]...)
			break
		}
	}

	logger.Info("关闭面板", zap.String("panel_id", ui.activePane))

	// 如果还有面板，激活第一个
	if len(ui.layout.panels) > 0 {
		ui.setActivePanel(ui.layout.panels[0])
	} else {
		ui.activePane = ""
	}

	// 更新布局
	ui.updateLayout()
}

// SplitHorizontal 水平分割
func (ui *UIManager) SplitHorizontal() {
	newID := fmt.Sprintf("panel_%d", time.Now().UnixNano())
	ui.CreatePanel(newID, "Split Panel")
	logger.Info("水平分割面板")
}

// SplitVertical 垂直分割
func (ui *UIManager) SplitVertical() {
	newID := fmt.Sprintf("panel_%d", time.Now().UnixNano())
	ui.CreatePanel(newID, "Split Panel")
	logger.Info("垂直分割面板")
}

// ShowHelp 显示帮助信息
func (ui *UIManager) ShowHelp() {
	helpText := `ClixGo Terminal UI Help

Key Bindings:
  Ctrl+C    - Quit application
  Ctrl+D    - Detach session
  Ctrl+N    - Create new panel
  Ctrl+W    - Close active panel
  Tab       - Switch to next panel
  Ctrl+H    - Split horizontal
  Ctrl+V    - Split vertical
  F1        - Show this help

Mouse Support:
  Click     - Focus panel
  Scroll    - Scroll panel content
`

	// 创建帮助面板
	ui.CreatePanel("help", "Help")
	ui.WriteToPanel("help", helpText)
}

// GetPanelCount 获取面板数量
func (ui *UIManager) GetPanelCount() int {
	ui.mutex.RLock()
	defer ui.mutex.RUnlock()
	return len(ui.panels)
}

// GetActivePanel 获取活动面板ID
func (ui *UIManager) GetActivePanel() string {
	ui.mutex.RLock()
	defer ui.mutex.RUnlock()
	return ui.activePane
}

// SetCustomInputCapture 设置自定义输入捕获
func (ui *UIManager) SetCustomInputCapture(capture func(*tcell.EventKey) *tcell.EventKey) {
	ui.app.SetInputCapture(capture)
	logger.Debug("设置自定义输入捕获")
}

// ====== 阶段3: 侧边栏布局管理 ======

// SetSidebar 设置侧边栏
func (ui *UIManager) SetSidebar(sidebar *Sidebar) {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	ui.sidebar = sidebar
	ui.layout.sidebar = sidebar

	// 更新布局以包含侧边栏
	ui.updateLayoutWithSidebar()

	logger.Info("侧边栏已设置")
}

// GetSidebar 获取侧边栏
func (ui *UIManager) GetSidebar() *Sidebar {
	ui.mutex.RLock()
	defer ui.mutex.RUnlock()
	return ui.sidebar
}

// ToggleSidebar 切换侧边栏显示
func (ui *UIManager) ToggleSidebar() {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	if ui.sidebar == nil {
		logger.Warn("侧边栏未初始化")
		return
	}

	ui.sidebar.Toggle()
	ui.updateLayoutWithSidebar()

	logger.Debug("侧边栏显示状态已切换",
		zap.Bool("visible", ui.sidebar.IsVisible()))
}

// updateLayoutWithSidebar 更新包含侧边栏的布局
func (ui *UIManager) updateLayoutWithSidebar() {
	if ui.sidebar == nil {
		ui.updateLayout() // 回退到原有布局
		return
	}

	ui.layout.mainArea.Clear()

	// 根据面板数量和侧边栏状态确定布局模式
	panelCount := len(ui.layout.panels)
	sidebarVisible := ui.sidebar.IsVisible()

	if !sidebarVisible {
		// 侧边栏不可见，使用普通布局
		ui.updateLayout()
		return
	}

	// 创建包含侧边栏的布局
	switch panelCount {
	case 0:
		ui.layoutWelcomeWithSidebar()
		ui.layout.mode = LayoutSingleWithSidebar
	case 1:
		ui.layoutSingleWithSidebar()
		ui.layout.mode = LayoutSingleWithSidebar
	case 2:
		ui.layoutVerticalWithSidebar()
		ui.layout.mode = LayoutVerticalWithSidebar
	default:
		ui.layoutGridWithSidebar()
		ui.layout.mode = LayoutGridWithSidebar
	}
}

// layoutWelcomeWithSidebar 欢迎界面+侧边栏布局
func (ui *UIManager) layoutWelcomeWithSidebar() {
	// 创建欢迎信息
	welcome := tview.NewTextView()
	welcome.SetText("Welcome to ClixGo Terminal\nPress F2 to toggle sidebar\nPress Ctrl+N to create a new panel")
	welcome.SetTextAlign(tview.AlignCenter)
	welcome.SetBorder(true)
	welcome.SetTitle("ClixGo")

	// 侧边栏 + 主内容区
	ui.layout.mainArea.SetDirection(tview.FlexColumn)
	ui.layout.mainArea.AddItem(ui.sidebar.List, ui.sidebar.GetWidth(), 0, false)
	ui.layout.mainArea.AddItem(welcome, 0, 1, true)
}

// layoutSingleWithSidebar 单面板+侧边栏布局
func (ui *UIManager) layoutSingleWithSidebar() {
	if len(ui.layout.panels) == 0 {
		return
	}

	panel := ui.layout.panels[0]

	// 侧边栏 + 单面板
	ui.layout.mainArea.SetDirection(tview.FlexColumn)
	ui.layout.mainArea.AddItem(ui.sidebar.List, ui.sidebar.GetWidth(), 0, false)
	ui.layout.mainArea.AddItem(panel.Content, 0, 1, true)
}

// layoutVerticalWithSidebar 垂直分割+侧边栏布局
func (ui *UIManager) layoutVerticalWithSidebar() {
	if len(ui.layout.panels) < 2 {
		return
	}

	// 创建主面板区域
	panelArea := tview.NewFlex().SetDirection(tview.FlexColumn)
	for _, panel := range ui.layout.panels {
		panelArea.AddItem(panel.Content, 0, 1, panel.Active)
	}

	// 侧边栏 + 面板区域
	ui.layout.mainArea.SetDirection(tview.FlexColumn)
	ui.layout.mainArea.AddItem(ui.sidebar.List, ui.sidebar.GetWidth(), 0, false)
	ui.layout.mainArea.AddItem(panelArea, 0, 1, true)
}

// layoutGridWithSidebar 网格+侧边栏布局
func (ui *UIManager) layoutGridWithSidebar() {
	panelCount := len(ui.layout.panels)
	if panelCount <= 2 {
		return
	}

	// 计算网格尺寸
	cols := 2
	rows := (panelCount + 1) / 2

	// 创建网格面板区域
	panelArea := tview.NewFlex().SetDirection(tview.FlexRow)

	for row := 0; row < rows; row++ {
		rowFlex := tview.NewFlex().SetDirection(tview.FlexColumn)

		for col := 0; col < cols; col++ {
			index := row*cols + col
			if index < panelCount {
				panel := ui.layout.panels[index]
				rowFlex.AddItem(panel.Content, 0, 1, panel.Active)
			}
		}

		panelArea.AddItem(rowFlex, 0, 1, false)
	}

	// 侧边栏 + 网格区域
	ui.layout.mainArea.SetDirection(tview.FlexColumn)
	ui.layout.mainArea.AddItem(ui.sidebar.List, ui.sidebar.GetWidth(), 0, false)
	ui.layout.mainArea.AddItem(panelArea, 0, 1, true)
}

// ====== 阶段4: 键盘绑定增强 ======

// setupSidebarKeyBindings 设置侧边栏按键绑定
func (ui *UIManager) setupSidebarKeyBindings() {
	// 扩展现有的setupKeyBindings方法
	originalCapture := ui.app.GetInputCapture()

	ui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// 侧边栏专用快捷键
		switch event.Key() {
		case tcell.KeyF2:
			ui.ToggleSidebar()
			return nil
		case tcell.KeyF3:
			ui.FocusSidebar()
			return nil
		}

		// 增强的Tab切换逻辑
		if event.Key() == tcell.KeyTab {
			if ui.sidebar != nil && ui.sidebar.IsVisible() {
				ui.ToggleFocusBetweenSidebarAndPanels()
				return nil
			}
		}

		// 调用原有的输入处理
		if originalCapture != nil {
			return originalCapture(event)
		}

		return event
	})

	logger.Debug("侧边栏键盘绑定已设置")
}

// FocusSidebar 聚焦到侧边栏
func (ui *UIManager) FocusSidebar() {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	if ui.sidebar == nil || !ui.sidebar.IsVisible() {
		logger.Debug("侧边栏不可用或不可见")
		return
	}

	ui.app.SetFocus(ui.sidebar.List)
	logger.Debug("焦点已切换到侧边栏")
}

// ToggleFocusBetweenSidebarAndPanels 在侧边栏和面板间切换焦点
func (ui *UIManager) ToggleFocusBetweenSidebarAndPanels() {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	if ui.sidebar == nil || !ui.sidebar.IsVisible() {
		ui.NextPanel() // 回退到普通的面板切换
		return
	}

	// 检查当前焦点
	focused := ui.app.GetFocus()

	if focused == ui.sidebar.List {
		// 当前在侧边栏，切换到活动面板
		if ui.activePane != "" {
			if panel, exists := ui.panels[ui.activePane]; exists {
				ui.app.SetFocus(panel.Content)
				logger.Debug("焦点从侧边栏切换到面板", zap.String("panel", ui.activePane))
			}
		}
	} else {
		// 当前在面板，切换到侧边栏
		ui.app.SetFocus(ui.sidebar.List)
		logger.Debug("焦点从面板切换到侧边栏")
	}
}

// EnableSidebarIntegration 启用侧边栏集成
func (ui *UIManager) EnableSidebarIntegration(sessionManager SessionManagerInterface) error {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	// 创建侧边栏
	sidebar := NewSidebar(sessionManager)
	if sidebar == nil {
		return fmt.Errorf("创建侧边栏失败")
	}

	// 设置侧边栏
	ui.sidebar = sidebar
	ui.layout.sidebar = sidebar

	// 启动侧边栏
	if err := sidebar.Start(); err != nil {
		return fmt.Errorf("启动侧边栏失败: %w", err)
	}

	// 设置键盘绑定
	ui.setupSidebarKeyBindings()

	// 更新布局
	ui.updateLayoutWithSidebar()

	logger.Info("侧边栏集成已启用")
	return nil
}

// DisableSidebarIntegration 禁用侧边栏集成
func (ui *UIManager) DisableSidebarIntegration() {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	if ui.sidebar != nil {
		ui.sidebar.Stop()
		ui.sidebar = nil
		ui.layout.sidebar = nil

		// 恢复普通布局
		ui.updateLayout()

		logger.Info("侧边栏集成已禁用")
	}
}
