/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-24 20:23:44
* @Description: 终端用户界面管理器的实现
 */

package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	layoutMgr  *layoutManager

	// Step 5: 主题管理集成
	themeManager *ThemeManager
	themesDir    string
	configDir    string
}

// NewUIManager 创建新的UI管理器
func NewUIManager(config UIConfig) (*UIManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	app := tview.NewApplication()

	// 设置主题和配置目录
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".clixgo")
	themesDir := filepath.Join(configDir, "themes")
	configFile := filepath.Join(configDir, "theme.json")

	ui := &UIManager{
		app:       app,
		panels:    make(map[string]*Panel),
		ctx:       ctx,
		cancel:    cancel,
		keyBinds:  make(map[tcell.Key]KeyHandler),
		mouseMode: config.MouseEnabled,
		configDir: configDir,
		themesDir: themesDir,
	}

	// 初始化主题管理器
	themeManager, err := NewThemeManager(themesDir, configFile)
	if err != nil {
		logger.Warn("主题管理器初始化失败，使用默认主题", zap.Error(err))
		// 继续使用默认配置
	} else {
		ui.themeManager = themeManager
		// 添加主题变更监听器
		themeManager.AddWatcher(ui)

		// 应用当前主题到配置
		if err := ui.applyCurrentTheme(&config); err != nil {
			logger.Warn("应用当前主题失败", zap.Error(err))
		}
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
		zap.Duration("refresh_rate", config.RefreshRate),
		zap.String("themes_dir", themesDir))

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
		// 处理主题相关快捷键
		switch event.Key() {
		case tcell.KeyF9:
			if err := ui.ToggleTheme(); err != nil {
				logger.Error("切换主题失败", zap.Error(err))
			}
			return nil
		case tcell.KeyF10:
			if err := ui.NextTheme(); err != nil {
				logger.Error("切换到下一个主题失败", zap.Error(err))
			}
			return nil
		case tcell.KeyF11:
			if err := ui.PrevTheme(); err != nil {
				logger.Error("切换到上一个主题失败", zap.Error(err))
			}
			return nil
		case tcell.KeyF12:
			ui.showThemeSelector()
			return nil
		}

		// 原有的快捷键处理
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
		case tcell.KeyF2:
			ui.ToggleSidebar()
			return nil
		case tcell.KeyF3:
			ui.FocusSidebar()
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

	// 关闭主题管理器
	if ui.themeManager != nil {
		ui.themeManager.Close()
	}

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

	// 检查是否为自定义布局模式，如果是则保持用户设置
	currentMode := ui.layout.mode
	isCustomLayout := currentMode == LayoutCustom || currentMode == LayoutFloating

	if isCustomLayout {
		// 自定义布局模式，应用相应的布局逻辑
		ui.applyCustomLayout(currentMode)
		return
	}

	// 自动布局模式
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

	// 检查是否为自定义布局模式，如果是则保持用户设置
	currentMode := ui.layout.mode
	isCustomLayout := currentMode == LayoutCustom || currentMode == LayoutFloating

	if isCustomLayout {
		// 自定义布局模式，应用相应的布局逻辑
		ui.applyCustomLayoutWithSidebar(currentMode)
		return
	}

	// 创建包含侧边栏的布局（自动模式）
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

// applyCustomLayoutWithSidebar 应用自定义布局（带侧边栏）
func (ui *UIManager) applyCustomLayoutWithSidebar(layoutMode LayoutMode) {
	switch layoutMode {
	case LayoutCustom:
		ui.layoutCustomWithSidebar()
	case LayoutFloating:
		ui.layoutFloatingWithSidebar()
	default:
		// 回退到网格布局
		ui.layoutGridWithSidebar()
	}
}

// layoutCustomWithSidebar 自定义布局+侧边栏
func (ui *UIManager) layoutCustomWithSidebar() {
	panelCount := len(ui.layout.panels)
	if panelCount == 0 {
		ui.layoutWelcomeWithSidebar()
		return
	}

	// 自定义布局：根据面板的Position和Size属性进行精确定位
	// 这里暂时使用网格布局作为基础，后续可以扩展为真正的自由定位
	panelArea := tview.NewFlex()

	if panelCount == 1 {
		// 单面板自定义布局
		panelArea.SetDirection(tview.FlexColumn)
		panel := ui.layout.panels[0]
		panelArea.AddItem(panel.Content, 0, 1, panel.Active)
	} else if panelCount == 2 {
		// 双面板自定义布局：可以是垂直或水平分割
		panelArea.SetDirection(tview.FlexRow) // 水平分割
		for _, panel := range ui.layout.panels {
			panelArea.AddItem(panel.Content, 0, 1, panel.Active)
		}
	} else {
		// 多面板自定义布局：使用网格作为基础
		ui.layoutCustomGrid(panelArea)
	}

	// 侧边栏 + 自定义面板区域
	ui.layout.mainArea.SetDirection(tview.FlexColumn)
	ui.layout.mainArea.AddItem(ui.sidebar.List, ui.sidebar.GetWidth(), 0, false)
	ui.layout.mainArea.AddItem(panelArea, 0, 1, true)
}

// layoutFloatingWithSidebar 浮动布局+侧边栏
func (ui *UIManager) layoutFloatingWithSidebar() {
	panelCount := len(ui.layout.panels)
	if panelCount == 0 {
		ui.layoutWelcomeWithSidebar()
		return
	}

	// 浮动布局：面板可以重叠，按ZIndex排序
	// 这里暂时使用垂直布局作为基础，后续可以扩展为真正的浮动窗口
	panelArea := tview.NewFlex().SetDirection(tview.FlexColumn)

	// 按ZIndex排序面板
	sortedPanels := make([]*Panel, len(ui.layout.panels))
	copy(sortedPanels, ui.layout.panels)

	// 简单排序（按ZIndex）
	for i := 0; i < len(sortedPanels)-1; i++ {
		for j := i + 1; j < len(sortedPanels); j++ {
			if sortedPanels[i].ZIndex > sortedPanels[j].ZIndex {
				sortedPanels[i], sortedPanels[j] = sortedPanels[j], sortedPanels[i]
			}
		}
	}

	for _, panel := range sortedPanels {
		panelArea.AddItem(panel.Content, 0, 1, panel.Active)
	}

	// 侧边栏 + 浮动面板区域
	ui.layout.mainArea.SetDirection(tview.FlexColumn)
	ui.layout.mainArea.AddItem(ui.sidebar.List, ui.sidebar.GetWidth(), 0, false)
	ui.layout.mainArea.AddItem(panelArea, 0, 1, true)
}

// layoutCustomGrid 自定义网格布局
func (ui *UIManager) layoutCustomGrid(panelArea *tview.Flex) {
	panelCount := len(ui.layout.panels)
	if panelCount <= 2 {
		return
	}

	// 计算网格尺寸
	cols := 2
	rows := (panelCount + 1) / 2

	panelArea.SetDirection(tview.FlexRow)

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
}

// applyCustomLayout 应用自定义布局（不带侧边栏）
func (ui *UIManager) applyCustomLayout(layoutMode LayoutMode) {
	switch layoutMode {
	case LayoutCustom:
		ui.layoutCustom()
	case LayoutFloating:
		ui.layoutFloating()
	default:
		// 回退到网格布局
		ui.layoutGrid()
	}
}

// layoutCustom 自定义布局
func (ui *UIManager) layoutCustom() {
	panelCount := len(ui.layout.panels)
	if panelCount == 0 {
		// 无面板，显示欢迎信息
		welcome := tview.NewTextView()
		welcome.SetText("Welcome to ClixGo Terminal\nPress Ctrl+N to create a new panel")
		welcome.SetTextAlign(tview.AlignCenter)
		welcome.SetBorder(true)
		welcome.SetTitle("ClixGo")
		ui.layout.mainArea.AddItem(welcome, 0, 1, true)
		return
	}

	if panelCount == 1 {
		// 单面板自定义布局
		panel := ui.layout.panels[0]
		ui.layout.mainArea.AddItem(panel.Content, 0, 1, true)
	} else if panelCount == 2 {
		// 双面板自定义布局：水平分割
		ui.layout.mainArea.SetDirection(tview.FlexRow)
		for _, panel := range ui.layout.panels {
			ui.layout.mainArea.AddItem(panel.Content, 0, 1, panel.Active)
		}
	} else {
		// 多面板自定义布局：使用网格
		ui.layoutGrid()
	}
}

// layoutFloating 浮动布局
func (ui *UIManager) layoutFloating() {
	panelCount := len(ui.layout.panels)
	if panelCount == 0 {
		// 无面板，显示欢迎信息
		welcome := tview.NewTextView()
		welcome.SetText("Welcome to ClixGo Terminal\nPress Ctrl+N to create a new panel")
		welcome.SetTextAlign(tview.AlignCenter)
		welcome.SetBorder(true)
		welcome.SetTitle("ClixGo")
		ui.layout.mainArea.AddItem(welcome, 0, 1, true)
		return
	}

	// 浮动布局：按ZIndex排序面板
	ui.layout.mainArea.SetDirection(tview.FlexColumn)

	// 按ZIndex排序面板
	sortedPanels := make([]*Panel, len(ui.layout.panels))
	copy(sortedPanels, ui.layout.panels)

	// 简单排序（按ZIndex）
	for i := 0; i < len(sortedPanels)-1; i++ {
		for j := i + 1; j < len(sortedPanels); j++ {
			if sortedPanels[i].ZIndex > sortedPanels[j].ZIndex {
				sortedPanels[i], sortedPanels[j] = sortedPanels[j], sortedPanels[i]
			}
		}
	}

	for _, panel := range sortedPanels {
		ui.layout.mainArea.AddItem(panel.Content, 0, 1, panel.Active)
	}
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

// ToggleFocusBetweenSidebarAndPanels 在侧边栏和面板之间切换焦点
func (ui *UIManager) ToggleFocusBetweenSidebarAndPanels() {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	if ui.sidebar == nil || !ui.sidebar.IsVisible() {
		ui.nextPanelInternal() // 使用内部方法避免死锁
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

// nextPanelInternal 内部面板切换方法，不获取锁
func (ui *UIManager) nextPanelInternal() {
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

// ====== Step 4: 布局管理增强 ======

// layoutManager 内嵌的布局管理器
type layoutManager struct {
	ui            *UIManager
	dragState     *DragState
	resizeHandles map[string][]*ResizeHandle
	savedLayouts  map[string]LayoutConfig
	mutex         sync.RWMutex
}

// newLayoutManager 创建布局管理器
func (ui *UIManager) newLayoutManager() *layoutManager {
	return &layoutManager{
		ui:            ui,
		dragState:     &DragState{},
		resizeHandles: make(map[string][]*ResizeHandle),
		savedLayouts:  make(map[string]LayoutConfig),
	}
}

// GetLayoutManager 获取布局管理器
func (ui *UIManager) GetLayoutManager() LayoutManager {
	if ui.layoutMgr == nil {
		ui.layoutMgr = ui.newLayoutManager()
	}
	return ui.layoutMgr
}

// ApplyLayout 应用布局配置
func (lm *layoutManager) ApplyLayout(config LayoutConfig) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	ui := lm.ui
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	// 设置布局模式
	ui.layout.mode = config.Mode

	// 应用侧边栏设置
	if ui.sidebar != nil {
		ui.sidebar.SetVisible(config.SidebarVisible)
		if config.SidebarWidth > 0 {
			ui.sidebar.SetWidth(config.SidebarWidth)
		}
	}

	// 应用面板布局
	for _, panelConfig := range config.PanelLayouts {
		if panel, exists := ui.panels[panelConfig.PanelID]; exists {
			panel.Position = panelConfig.Position
			panel.Size = panelConfig.Size
			panel.ZIndex = panelConfig.ZIndex
			panel.Constraints = &panelConfig.Constraints
		}
	}

	// 更新布局
	ui.updateLayoutWithSidebar()

	logger.Info("布局配置已应用",
		zap.String("layout_name", config.Name),
		zap.Int("layout_mode", int(config.Mode)),
		zap.Int("panel_count", len(config.PanelLayouts)))

	return nil
}

// GetCurrentLayout 获取当前布局配置
func (lm *layoutManager) GetCurrentLayout() LayoutConfig {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	ui := lm.ui
	ui.mutex.RLock()
	defer ui.mutex.RUnlock()

	config := LayoutConfig{
		Name:           "current",
		Mode:           ui.layout.mode,
		SidebarVisible: ui.sidebar != nil && ui.sidebar.IsVisible(),
		SidebarWidth:   0,
		PanelLayouts:   make([]PanelLayoutConfig, 0, len(ui.panels)),
		CreatedAt:      time.Now(),
		LastModified:   time.Now(),
	}

	if ui.sidebar != nil {
		config.SidebarWidth = ui.sidebar.GetWidth()
	}

	// 收集面板布局信息
	for _, panel := range ui.panels {
		panelConfig := PanelLayoutConfig{
			PanelID:  panel.ID,
			Position: panel.Position,
			Size:     panel.Size,
			ZIndex:   panel.ZIndex,
		}
		if panel.Constraints != nil {
			panelConfig.Constraints = *panel.Constraints
		}
		config.PanelLayouts = append(config.PanelLayouts, panelConfig)
	}

	return config
}

// ResetLayout 重置布局
func (lm *layoutManager) ResetLayout() error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	ui := lm.ui
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	// 重置所有面板到默认状态
	for _, panel := range ui.panels {
		panel.Position = Position{X: 0, Y: 0}
		panel.Size = Size{Width: 80, Height: 24}
		panel.ZIndex = 0
		panel.Constraints = nil
		panel.Resizable = true
		panel.Draggable = true
	}

	// 重置布局模式
	if len(ui.panels) <= 1 {
		ui.layout.mode = LayoutSingle
	} else if len(ui.panels) == 2 {
		ui.layout.mode = LayoutVertical
	} else {
		ui.layout.mode = LayoutGrid
	}

	// 更新布局
	ui.updateLayout()

	logger.Info("布局已重置")
	return nil
}

// ResizePanel 调整面板大小
func (lm *layoutManager) ResizePanel(panelID string, newSize Size) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	ui := lm.ui
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	panel, exists := ui.panels[panelID]
	if !exists {
		return fmt.Errorf("面板不存在: %s", panelID)
	}

	if !panel.Resizable {
		return fmt.Errorf("面板不可调整大小: %s", panelID)
	}

	// 检查尺寸约束
	if err := lm.validateSize(panel, newSize); err != nil {
		return fmt.Errorf("尺寸验证失败: %w", err)
	}

	oldSize := panel.Size
	panel.Size = newSize

	// 触发布局事件
	lm.emitLayoutEvent(LayoutEventPanelResized, panelID, oldSize, newSize)

	// 更新布局
	ui.updateLayoutWithSidebar()

	logger.Debug("面板大小已调整",
		zap.String("panel_id", panelID),
		zap.Int("old_width", oldSize.Width),
		zap.Int("old_height", oldSize.Height),
		zap.Int("new_width", newSize.Width),
		zap.Int("new_height", newSize.Height))

	return nil
}

// MovePanel 移动面板
func (lm *layoutManager) MovePanel(panelID string, newPos Position) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	ui := lm.ui
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	panel, exists := ui.panels[panelID]
	if !exists {
		return fmt.Errorf("面板不存在: %s", panelID)
	}

	if !panel.Draggable {
		return fmt.Errorf("面板不可拖拽: %s", panelID)
	}

	// 检查位置约束
	if err := lm.validatePosition(panel, newPos); err != nil {
		return fmt.Errorf("位置验证失败: %w", err)
	}

	oldPos := panel.Position
	panel.Position = newPos

	// 触发布局事件
	lm.emitLayoutEvent(LayoutEventPanelMoved, panelID, oldPos, newPos)

	// 更新布局
	ui.updateLayoutWithSidebar()

	logger.Debug("面板位置已移动",
		zap.String("panel_id", panelID),
		zap.Int("old_x", oldPos.X),
		zap.Int("old_y", oldPos.Y),
		zap.Int("new_x", newPos.X),
		zap.Int("new_y", newPos.Y))

	return nil
}

// SetPanelConstraints 设置面板约束
func (lm *layoutManager) SetPanelConstraints(panelID string, constraints Constraints) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	ui := lm.ui
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	panel, exists := ui.panels[panelID]
	if !exists {
		return fmt.Errorf("面板不存在: %s", panelID)
	}

	panel.Constraints = &constraints

	logger.Debug("面板约束已设置",
		zap.String("panel_id", panelID),
		zap.Bool("fixed_width", constraints.FixedWidth),
		zap.Bool("fixed_height", constraints.FixedHeight))

	return nil
}

// StartDrag 开始拖拽
func (lm *layoutManager) StartDrag(panelID string, startPos Position, dragType DragType) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	ui := lm.ui
	ui.mutex.RLock()
	panel, exists := ui.panels[panelID]
	ui.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("面板不存在: %s", panelID)
	}

	if dragType == DragMove && !panel.Draggable {
		return fmt.Errorf("面板不可拖拽: %s", panelID)
	}

	if dragType == DragResize && !panel.Resizable {
		return fmt.Errorf("面板不可调整大小: %s", panelID)
	}

	lm.dragState = &DragState{
		Active:     true,
		PanelID:    panelID,
		StartPos:   startPos,
		CurrentPos: startPos,
		Offset:     Position{X: 0, Y: 0},
		Type:       dragType,
	}

	// 触发拖拽开始事件
	lm.emitLayoutEvent(LayoutEventDragStarted, panelID, nil, dragType)

	logger.Debug("拖拽已开始",
		zap.String("panel_id", panelID),
		zap.Int("drag_type", int(dragType)),
		zap.Int("start_x", startPos.X),
		zap.Int("start_y", startPos.Y))

	return nil
}

// UpdateDrag 更新拖拽
func (lm *layoutManager) UpdateDrag(currentPos Position) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	if !lm.dragState.Active {
		return fmt.Errorf("没有活动的拖拽操作")
	}

	lm.dragState.CurrentPos = currentPos
	lm.dragState.Offset = Position{
		X: currentPos.X - lm.dragState.StartPos.X,
		Y: currentPos.Y - lm.dragState.StartPos.Y,
	}

	ui := lm.ui
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	panel, exists := ui.panels[lm.dragState.PanelID]
	if !exists {
		return fmt.Errorf("拖拽的面板不存在: %s", lm.dragState.PanelID)
	}

	// 根据拖拽类型更新面板
	switch lm.dragState.Type {
	case DragMove:
		newPos := Position{
			X: panel.Position.X + lm.dragState.Offset.X,
			Y: panel.Position.Y + lm.dragState.Offset.Y,
		}
		if err := lm.validatePosition(panel, newPos); err == nil {
			panel.Position = newPos
		}

	case DragResize:
		newSize := Size{
			Width:  panel.Size.Width + lm.dragState.Offset.X,
			Height: panel.Size.Height + lm.dragState.Offset.Y,
		}
		if err := lm.validateSize(panel, newSize); err == nil {
			panel.Size = newSize
		}
	}

	// 更新布局
	ui.updateLayoutWithSidebar()

	return nil
}

// EndDrag 结束拖拽
func (lm *layoutManager) EndDrag() error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	if !lm.dragState.Active {
		return fmt.Errorf("没有活动的拖拽操作")
	}

	panelID := lm.dragState.PanelID
	dragType := lm.dragState.Type

	lm.dragState.Active = false

	// 触发拖拽结束事件
	lm.emitLayoutEvent(LayoutEventDragEnded, panelID, nil, dragType)

	logger.Debug("拖拽已结束",
		zap.String("panel_id", panelID),
		zap.Int("drag_type", int(dragType)))

	return nil
}

// SaveLayout 保存布局
func (lm *layoutManager) SaveLayout(name string) error {
	if name == "" {
		return fmt.Errorf("布局名称不能为空")
	}

	// 先获取当前布局配置（不加锁）
	config := lm.GetCurrentLayout()
	config.Name = name
	config.CreatedAt = time.Now()
	config.LastModified = time.Now()

	// 然后加锁保存
	lm.mutex.Lock()
	lm.savedLayouts[name] = config
	lm.mutex.Unlock()

	logger.Info("布局已保存",
		zap.String("layout_name", name),
		zap.Int("panel_count", len(config.PanelLayouts)))

	return nil
}

// LoadLayout 加载布局
func (lm *layoutManager) LoadLayout(name string) error {
	lm.mutex.RLock()
	config, exists := lm.savedLayouts[name]
	lm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("布局不存在: %s", name)
	}

	return lm.ApplyLayout(config)
}

// ListLayouts 列出所有布局
func (lm *layoutManager) ListLayouts() []string {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	layouts := make([]string, 0, len(lm.savedLayouts))
	for name := range lm.savedLayouts {
		layouts = append(layouts, name)
	}

	return layouts
}

// DeleteLayout 删除布局
func (lm *layoutManager) DeleteLayout(name string) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	if _, exists := lm.savedLayouts[name]; !exists {
		return fmt.Errorf("布局不存在: %s", name)
	}

	delete(lm.savedLayouts, name)

	logger.Info("布局已删除", zap.String("layout_name", name))
	return nil
}

// ====== Step 4: 辅助方法 ======

// validateSize 验证面板尺寸
func (lm *layoutManager) validateSize(panel *Panel, size Size) error {
	// 检查最小尺寸
	if size.Width < panel.MinSize.Width || size.Height < panel.MinSize.Height {
		return fmt.Errorf("尺寸小于最小值 (%dx%d)", panel.MinSize.Width, panel.MinSize.Height)
	}

	// 检查最大尺寸
	if panel.MaxSize.Width > 0 && size.Width > panel.MaxSize.Width {
		return fmt.Errorf("宽度超过最大值 %d", panel.MaxSize.Width)
	}
	if panel.MaxSize.Height > 0 && size.Height > panel.MaxSize.Height {
		return fmt.Errorf("高度超过最大值 %d", panel.MaxSize.Height)
	}

	// 检查宽高比约束
	if panel.Constraints != nil && panel.Constraints.AspectRatio > 0 {
		ratio := float64(size.Width) / float64(size.Height)
		expectedRatio := panel.Constraints.AspectRatio
		tolerance := 0.1 // 10%容差

		if ratio < expectedRatio-tolerance || ratio > expectedRatio+tolerance {
			return fmt.Errorf("宽高比不符合约束 %.2f", expectedRatio)
		}
	}

	return nil
}

// validatePosition 验证面板位置
func (lm *layoutManager) validatePosition(panel *Panel, pos Position) error {
	// 检查边界
	if pos.X < 0 || pos.Y < 0 {
		return fmt.Errorf("位置不能为负数")
	}

	// 检查边距约束
	if panel.Constraints != nil {
		if pos.X < panel.Constraints.MarginLeft {
			return fmt.Errorf("超出左边距约束")
		}
		if pos.Y < panel.Constraints.MarginTop {
			return fmt.Errorf("超出上边距约束")
		}
	}

	return nil
}

// emitLayoutEvent 发射布局事件
func (lm *layoutManager) emitLayoutEvent(eventType LayoutEventType, panelID string, oldValue, newValue interface{}) {
	event := LayoutEvent{
		Type:      eventType,
		PanelID:   panelID,
		OldValue:  oldValue,
		NewValue:  newValue,
		Timestamp: time.Now(),
	}

	logger.Debug("布局事件已发射",
		zap.Int("event_type", int(eventType)),
		zap.String("panel_id", panelID))

	// TODO: 实现事件监听器机制
	_ = event
}

// CreateResizablePanel 创建可调整大小的面板
func (ui *UIManager) CreateResizablePanel(id, title string, resizable, draggable bool) error {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	if _, exists := ui.panels[id]; exists {
		return fmt.Errorf("面板已存在: %s", id)
	}

	content := tview.NewTextView()
	content.SetBorder(true)
	content.SetTitle(title)
	content.SetDynamicColors(true)
	content.SetScrollable(true)

	panel := &Panel{
		ID:         id,
		Title:      title,
		Content:    content,
		Border:     true,
		Active:     len(ui.panels) == 0,
		Position:   Position{X: 0, Y: 0},
		Size:       Size{Width: 80, Height: 24},
		LastUpdate: time.Now(),
		MaxLines:   1000,
		AutoScroll: true,

		// Step 4: 增强字段
		Resizable:    resizable,
		Draggable:    draggable,
		MinSize:      Size{Width: 20, Height: 5},
		MaxSize:      Size{Width: 200, Height: 50},
		OriginalSize: Size{Width: 80, Height: 24},
		ZIndex:       0,
		Constraints:  nil,
	}

	ui.panels[id] = panel
	ui.layout.panels = append(ui.layout.panels, panel)

	if panel.Active {
		ui.activePane = id
	}

	ui.updateLayoutWithSidebar()

	logger.Info("可调整面板已创建",
		zap.String("panel_id", id),
		zap.String("title", title),
		zap.Bool("resizable", resizable),
		zap.Bool("draggable", draggable))

	return nil
}

// EnableAdvancedLayoutFeatures 启用高级布局功能
func (ui *UIManager) EnableAdvancedLayoutFeatures() error {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	// 初始化布局管理器
	if ui.layoutMgr == nil {
		ui.layoutMgr = ui.newLayoutManager()
	}

	// 为现有面板启用高级功能
	for _, panel := range ui.panels {
		panel.Resizable = true
		panel.Draggable = true
		panel.MinSize = Size{Width: 20, Height: 5}
		panel.MaxSize = Size{Width: 200, Height: 50}
		panel.OriginalSize = panel.Size
	}

	// 设置高级键盘绑定
	ui.setupAdvancedKeyBindings()

	logger.Info("高级布局功能已启用")
	return nil
}

// setupAdvancedKeyBindings 设置高级键盘绑定
func (ui *UIManager) setupAdvancedKeyBindings() {
	// 扩展现有的键盘绑定
	originalCapture := ui.app.GetInputCapture()

	ui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Step 4: 高级布局快捷键
		switch event.Key() {
		case tcell.KeyF4:
			ui.ToggleLayoutMode()
			return nil
		case tcell.KeyF5:
			ui.SaveCurrentLayout("auto_save")
			return nil
		case tcell.KeyF6:
			ui.LoadLastLayout()
			return nil
		case tcell.KeyF7:
			ui.ResetAllLayouts()
			return nil
		}

		// Ctrl组合键
		if event.Key() == tcell.KeyCtrlR {
			ui.TogglePanelResizable()
			return nil
		}
		if event.Key() == tcell.KeyCtrlT {
			ui.TogglePanelDraggable()
			return nil
		}

		// 调用原有处理
		if originalCapture != nil {
			return originalCapture(event)
		}

		return event
	})
}

// ToggleLayoutMode 切换布局模式
func (ui *UIManager) ToggleLayoutMode() {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	currentMode := ui.layout.mode
	var newMode LayoutMode

	// 在不同布局模式间循环
	switch currentMode {
	case LayoutSingle, LayoutSingleWithSidebar:
		newMode = LayoutVertical
	case LayoutVertical, LayoutVerticalWithSidebar:
		newMode = LayoutHorizontal
	case LayoutHorizontal, LayoutHorizontalWithSidebar:
		newMode = LayoutGrid
	case LayoutGrid, LayoutGridWithSidebar:
		newMode = LayoutCustom
	case LayoutCustom:
		newMode = LayoutFloating
	case LayoutFloating:
		newMode = LayoutSingle
	default:
		newMode = LayoutSingle
	}

	// 如果有侧边栏，转换为对应的侧边栏模式
	if ui.sidebar != nil && ui.sidebar.IsVisible() {
		switch newMode {
		case LayoutSingle:
			newMode = LayoutSingleWithSidebar
		case LayoutVertical:
			newMode = LayoutVerticalWithSidebar
		case LayoutHorizontal:
			newMode = LayoutHorizontalWithSidebar
		case LayoutGrid:
			newMode = LayoutGridWithSidebar
		}
	}

	ui.layout.mode = newMode

	// 直接应用新的布局模式，不让自动布局逻辑覆盖
	if ui.sidebar != nil && ui.sidebar.IsVisible() {
		ui.applyCustomLayoutWithSidebar(newMode)
	} else {
		ui.applyCustomLayout(newMode)
	}

	logger.Info("布局模式已切换",
		zap.Int("old_mode", int(currentMode)),
		zap.Int("new_mode", int(newMode)))
}

// SaveCurrentLayout 保存当前布局
func (ui *UIManager) SaveCurrentLayout(name string) error {
	if ui.layoutMgr == nil {
		ui.layoutMgr = ui.newLayoutManager()
	}
	return ui.layoutMgr.SaveLayout(name)
}

// LoadLastLayout 加载最后保存的布局
func (ui *UIManager) LoadLastLayout() error {
	if ui.layoutMgr == nil {
		ui.layoutMgr = ui.newLayoutManager()
	}
	return ui.layoutMgr.LoadLayout("auto_save")
}

// ResetAllLayouts 重置所有布局
func (ui *UIManager) ResetAllLayouts() error {
	if ui.layoutMgr == nil {
		ui.layoutMgr = ui.newLayoutManager()
	}
	return ui.layoutMgr.ResetLayout()
}

// TogglePanelResizable 切换活动面板的可调整大小属性
func (ui *UIManager) TogglePanelResizable() {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	if ui.activePane == "" {
		return
	}

	if panel, exists := ui.panels[ui.activePane]; exists {
		panel.Resizable = !panel.Resizable
		logger.Debug("面板可调整大小属性已切换",
			zap.String("panel_id", ui.activePane),
			zap.Bool("resizable", panel.Resizable))
	}
}

// TogglePanelDraggable 切换活动面板的可拖拽属性
func (ui *UIManager) TogglePanelDraggable() {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	if ui.activePane == "" {
		return
	}

	if panel, exists := ui.panels[ui.activePane]; exists {
		panel.Draggable = !panel.Draggable
		logger.Debug("面板可拖拽属性已切换",
			zap.String("panel_id", ui.activePane),
			zap.Bool("draggable", panel.Draggable))
	}
}

// ===== Step 5: 主题管理集成 =====

// OnThemeChanged 实现ThemeWatcher接口
func (ui *UIManager) OnThemeChanged(oldTheme, newTheme *EnhancedTheme) {
	logger.Info("主题变更通知",
		zap.String("old_theme", func() string {
			if oldTheme != nil {
				return oldTheme.Name
			}
			return "none"
		}()),
		zap.String("new_theme", newTheme.Name))

	// 应用新主题
	if err := ui.applyTheme(newTheme); err != nil {
		logger.Error("应用新主题失败", zap.Error(err))
		return
	}

	// 刷新UI
	ui.app.QueueUpdateDraw(func() {
		// 强制重绘所有组件
		ui.refreshAllComponents()
	})
}

// applyCurrentTheme 应用当前活动主题
func (ui *UIManager) applyCurrentTheme(config *UIConfig) error {
	if ui.themeManager == nil {
		return nil
	}

	theme, err := ui.themeManager.GetActiveTheme()
	if err != nil {
		return fmt.Errorf("获取当前主题失败: %w", err)
	}

	return ui.applyThemeToConfig(theme, config)
}

// applyTheme 应用主题到UI组件
func (ui *UIManager) applyTheme(theme *EnhancedTheme) error {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	// 应用主题到状态栏
	if ui.statusBar != nil {
		ui.statusBar.view.SetBackgroundColor(theme.Colors.StatusBar)
		ui.statusBar.view.SetTextColor(theme.Colors.StatusText)
		ui.statusBar.style = tcell.StyleDefault.
			Background(theme.Colors.StatusBar).
			Foreground(theme.Colors.StatusText)
	}

	// 应用主题到侧边栏
	if ui.sidebar != nil {
		ui.sidebar.List.SetBackgroundColor(theme.Colors.SidebarBg)
		ui.sidebar.List.SetMainTextColor(theme.Colors.SidebarText)
		ui.sidebar.List.SetSelectedBackgroundColor(theme.Colors.SidebarActive)
		ui.sidebar.List.SetSelectedTextColor(theme.Colors.SidebarText)
		ui.sidebar.List.SetBorderColor(theme.Colors.Border)
	}

	// 应用主题到所有面板
	for _, panel := range ui.panels {
		panel.Content.SetBackgroundColor(theme.Colors.PanelBg)
		panel.Content.SetTextColor(theme.Colors.PanelText)
		panel.Content.SetTitleColor(theme.Colors.PanelTitle)

		if panel.Active {
			panel.Content.SetBorderColor(theme.Colors.ActiveBorder)
		} else {
			panel.Content.SetBorderColor(theme.Colors.Border)
		}
	}

	// 应用主题到布局
	if ui.layout != nil && ui.layout.root != nil {
		ui.layout.root.SetBackgroundColor(theme.Colors.Background)
	}

	logger.Debug("主题应用完成", zap.String("theme", theme.Name))
	return nil
}

// applyThemeToConfig 应用主题到配置
func (ui *UIManager) applyThemeToConfig(theme *EnhancedTheme, config *UIConfig) error {
	// 转换增强主题为标准主题
	config.Theme = Theme{
		Background:   theme.Colors.Background,
		Foreground:   theme.Colors.Foreground,
		Border:       theme.Colors.Border,
		ActiveBorder: theme.Colors.ActiveBorder,
		StatusBar:    theme.Colors.StatusBar,
		StatusText:   theme.Colors.StatusText,
	}

	// 应用组件特定配置
	config.StatusBarStyle.ShowTime = theme.Components.StatusBar.ShowTime
	config.StatusBarStyle.ShowStats = theme.Components.StatusBar.ShowStats
	config.StatusBarStyle.Format = theme.Components.StatusBar.Format

	return nil
}

// refreshAllComponents 刷新所有UI组件
func (ui *UIManager) refreshAllComponents() {
	// 刷新状态栏
	if ui.statusBar != nil {
		ui.updateStatusBar()
	}

	// 刷新侧边栏
	if ui.sidebar != nil {
		ui.sidebar.refreshData()
	}

	// 刷新布局
	ui.updateLayout()
}

// ===== 主题管理功能 =====

// SwitchTheme 切换到指定主题
func (ui *UIManager) SwitchTheme(themeName string) error {
	if ui.themeManager == nil {
		return fmt.Errorf("主题管理器未初始化")
	}

	return ui.themeManager.SetActiveTheme(themeName)
}

// NextTheme 切换到下一个主题
func (ui *UIManager) NextTheme() error {
	if ui.themeManager == nil {
		return fmt.Errorf("主题管理器未初始化")
	}

	return ui.themeManager.NextTheme()
}

// PrevTheme 切换到上一个主题
func (ui *UIManager) PrevTheme() error {
	if ui.themeManager == nil {
		return fmt.Errorf("主题管理器未初始化")
	}

	return ui.themeManager.PrevTheme()
}

// ToggleTheme 在默认和暗色主题间切换
func (ui *UIManager) ToggleTheme() error {
	if ui.themeManager == nil {
		return fmt.Errorf("主题管理器未初始化")
	}

	return ui.themeManager.ToggleTheme()
}

// GetAvailableThemes 获取可用主题列表
func (ui *UIManager) GetAvailableThemes() []string {
	if ui.themeManager == nil {
		return []string{"default"}
	}

	return ui.themeManager.ListThemes()
}

// GetCurrentTheme 获取当前主题名称
func (ui *UIManager) GetCurrentTheme() string {
	if ui.themeManager == nil {
		return "default"
	}

	theme, err := ui.themeManager.GetActiveTheme()
	if err != nil {
		return "default"
	}

	return theme.Name
}

// GetThemePreview 获取主题预览信息
func (ui *UIManager) GetThemePreview(themeName string) (string, error) {
	if ui.themeManager == nil {
		return "", fmt.Errorf("主题管理器未初始化")
	}

	return ui.themeManager.GetThemePreview(themeName)
}

// SaveCurrentTheme 保存当前主题配置
func (ui *UIManager) SaveCurrentTheme(name string) error {
	if ui.themeManager == nil {
		return fmt.Errorf("主题管理器未初始化")
	}

	// 获取当前主题
	currentTheme, err := ui.themeManager.GetActiveTheme()
	if err != nil {
		return fmt.Errorf("获取当前主题失败: %w", err)
	}

	// 创建副本并重命名
	newTheme := *currentTheme
	newTheme.Name = name
	newTheme.Description = fmt.Sprintf("基于 %s 的自定义主题", currentTheme.Name)
	newTheme.Author = "User"
	newTheme.CreatedAt = time.Now()

	return ui.themeManager.SaveTheme(&newTheme)
}

// DeleteTheme 删除主题
func (ui *UIManager) DeleteTheme(name string) error {
	if ui.themeManager == nil {
		return fmt.Errorf("主题管理器未初始化")
	}

	return ui.themeManager.DeleteTheme(name)
}

// showThemeSelector 显示主题选择器
func (ui *UIManager) showThemeSelector() {
	if ui.themeManager == nil {
		return
	}

	themes := ui.themeManager.ListThemes()
	currentTheme := ui.GetCurrentTheme()

	// 创建主题选择对话框
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle("选择主题 (Enter: 应用, Esc: 取消)")

	for i, theme := range themes {
		title := theme
		if theme == currentTheme {
			title = "* " + theme + " (当前)"
		}

		list.AddItem(title, "", rune('1'+i), func() {
			if err := ui.SwitchTheme(theme); err != nil {
				logger.Error("切换主题失败", zap.Error(err))
			}
			ui.app.SetRoot(ui.layout.root, true)
		})
	}

	// 设置选择处理
	list.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		themeName := themes[index]
		if err := ui.SwitchTheme(themeName); err != nil {
			logger.Error("切换主题失败", zap.Error(err))
		}
		ui.app.SetRoot(ui.layout.root, true)
	})

	// 设置退出处理
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			ui.app.SetRoot(ui.layout.root, true)
			return nil
		}
		return event
	})

	// 显示对话框
	ui.app.SetRoot(list, true)
	ui.app.SetFocus(list)
}
