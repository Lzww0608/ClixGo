/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-3 18:52:54
* @Description: 终端用户界面的核心实现
 */

package terminal

import (
	"fmt"
	"strings"
	"sync"
)

// UIRenderer UI渲染器
type UIRenderer struct {
	width     int
	height    int
	buffer    [][]rune
	statusBar string
	keyHelp   []string
	mutex     sync.RWMutex
	theme     *UITheme
}

// UITheme UI主题
type UITheme struct {
	BorderStyle    string
	StatusBarStyle string
	HelpStyle      string
	ActiveStyle    string
	InactiveStyle  string
}

// PaneLayout 面板布局信息
type PaneLayout struct {
	X      int
	Y      int
	Width  int
	Height int
	Active bool
	Title  string
	Buffer []string
}

// DefaultTheme 默认主题
var DefaultTheme = &UITheme{
	BorderStyle:    "┌─┐│└┘",
	StatusBarStyle: "\033[44;97m", // 蓝色背景，白色文字
	HelpStyle:      "\033[90m",    // 灰色
	ActiveStyle:    "\033[32m",    // 绿色
	InactiveStyle:  "\033[37m",    // 白色
}

// NewUIRenderer 创建UI渲染器
func NewUIRenderer(width, height int, theme *UITheme) *UIRenderer {
	if theme == nil {
		theme = DefaultTheme
	}

	ui := &UIRenderer{
		width:  width,
		height: height,
		buffer: make([][]rune, height),
		theme:  theme,
		keyHelp: []string{
			"C-b d: 断开会话",
			"C-b c: 创建窗口",
			"C-b \": 水平分割",
			"C-b %: 垂直分割",
			"C-b o: 切换面板",
			"C-b x: 关闭面板",
			"C-b ?: 帮助",
		},
	}

	// 初始化缓冲区
	for i := range ui.buffer {
		ui.buffer[i] = make([]rune, width)
		for j := range ui.buffer[i] {
			ui.buffer[i][j] = ' '
		}
	}

	return ui
}

// Resize 调整UI大小
func (ui *UIRenderer) Resize(width, height int) {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	ui.width = width
	ui.height = height

	// 重新分配缓冲区
	ui.buffer = make([][]rune, height)
	for i := range ui.buffer {
		ui.buffer[i] = make([]rune, width)
		for j := range ui.buffer[i] {
			ui.buffer[i][j] = ' '
		}
	}
}

// SetStatusBar 设置状态栏
func (ui *UIRenderer) SetStatusBar(status string) {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()
	ui.statusBar = status
}

// ClearScreen 清屏
func (ui *UIRenderer) ClearScreen() {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	for i := range ui.buffer {
		for j := range ui.buffer[i] {
			ui.buffer[i][j] = ' '
		}
	}
}

// RenderWindow 渲染窗口
func (ui *UIRenderer) RenderWindow(window *Window) string {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	// 清空缓冲区
	ui.ClearScreen()

	// 计算面板布局
	layouts := ui.calculatePaneLayouts(window)

	// 渲染每个面板
	for _, layout := range layouts {
		ui.renderPane(&layout)
	}

	// 渲染状态栏
	ui.renderStatusBar(window)

	// 渲染帮助信息
	ui.renderHelpInfo()

	// 生成输出字符串
	return ui.generateOutput()
}

// calculatePaneLayouts 计算窗口中所有面板的布局信息
//
// 参数:
//   - window: 包含要布局的面板的窗口对象
//
// 返回:
//   - []PaneLayout: 每个面板的布局信息列表
//
// 该函数根据窗口的布局类型，计算每个面板的位置、尺寸和状态信息
func (ui *UIRenderer) calculatePaneLayouts(window *Window) []PaneLayout {
	if len(window.Panes) == 0 {
		return []PaneLayout{}
	}

	layouts := make([]PaneLayout, 0, len(window.Panes))
	availableHeight := ui.height - 3 // 预留状态栏和帮助行

	switch window.Layout {
	case LayoutEven:
		layouts = ui.layoutEven(window.Panes, ui.width, availableHeight)
	case LayoutMainVertical:
		layouts = ui.layoutMainVertical(window.Panes, ui.width, availableHeight)
	case LayoutMainHorizontal:
		layouts = ui.layoutMainHorizontal(window.Panes, ui.width, availableHeight)
	case LayoutTiled:
		layouts = ui.layoutTiled(window.Panes, ui.width, availableHeight)
	default:
		layouts = ui.layoutEven(window.Panes, ui.width, availableHeight)
	}

	// 设置活动状态和获取缓冲区内容
	for i, pane := range window.Panes {
		if i < len(layouts) {
			layouts[i].Active = (i == window.ActivePane)
			layouts[i].Title = fmt.Sprintf("Pane %d", i)
			layouts[i].Buffer = ui.getPaneBuffer(pane)
		}
	}

	return layouts
}

// getPaneBuffer 获取面板的缓冲区内容以供渲染
//
// 参数:
//   - pane: 要获取缓冲区内容的面板对象
//
// 返回:
//   - []string: 面板缓冲区的文本行列表
//
// 如果面板缓冲区为空，则返回包含命令提示符的默认内容
func (ui *UIRenderer) getPaneBuffer(pane *Pane) []string {
	if pane.Buffer == nil || len(pane.Buffer.Lines) == 0 {
		return []string{"$ " + pane.Command}
	}

	buffer := make([]string, 0, len(pane.Buffer.Lines))
	for _, line := range pane.Buffer.Lines {
		buffer = append(buffer, string(line))
	}

	return buffer
}

// layoutEven 计算均匀分布的面板布局
//
// 参数:
//   - panes: 要布局的面板列表
//   - width: 可用的总宽度
//   - height: 可用的总高度
//
// 返回:
//   - []PaneLayout: 每个面板的布局信息
//
// 该布局将所有面板水平均匀分布，每个面板具有相同的宽度
func (ui *UIRenderer) layoutEven(panes []*Pane, width, height int) []PaneLayout {
	if len(panes) == 0 {
		return []PaneLayout{}
	}

	layouts := make([]PaneLayout, len(panes))
	paneWidth := width / len(panes)

	for i := range layouts {
		layouts[i] = PaneLayout{
			X:      i * paneWidth,
			Y:      0,
			Width:  paneWidth,
			Height: height,
		}
	}

	return layouts
}

// layoutMainVertical 计算主垂直分割布局
//
// 参数:
//   - panes: 要布局的面板列表
//   - width: 可用的总宽度
//   - height: 可用的总高度
//
// 返回:
//   - []PaneLayout: 每个面板的布局信息
//
// 该布局将第一个面板作为主面板占据左侧2/3宽度，
// 其余面板垂直排列在右侧1/3宽度区域
func (ui *UIRenderer) layoutMainVertical(panes []*Pane, width, height int) []PaneLayout {
	if len(panes) == 0 {
		return []PaneLayout{}
	}

	layouts := make([]PaneLayout, len(panes))

	if len(panes) == 1 {
		layouts[0] = PaneLayout{
			X: 0, Y: 0, Width: width, Height: height,
		}
		return layouts
	}

	mainPaneWidth := width * 2 / 3
	sidePaneWidth := width - mainPaneWidth
	sidePaneHeight := height / (len(panes) - 1)

	// 主面板（占据左侧大部分空间）
	layouts[0] = PaneLayout{
		X: 0, Y: 0, Width: mainPaneWidth, Height: height,
	}

	// 侧面板（垂直排列在右侧）
	for i := 1; i < len(panes); i++ {
		layouts[i] = PaneLayout{
			X:      mainPaneWidth,
			Y:      (i - 1) * sidePaneHeight,
			Width:  sidePaneWidth,
			Height: sidePaneHeight,
		}
	}

	return layouts
}

// layoutMainHorizontal 计算主水平分割布局
//
// 参数:
//   - panes: 要布局的面板列表
//   - width: 可用的总宽度
//   - height: 可用的总高度
//
// 返回:
//   - []PaneLayout: 每个面板的布局信息
//
// 该布局将第一个面板作为主面板占据上方2/3高度，
// 其余面板水平排列在下方1/3高度区域
func (ui *UIRenderer) layoutMainHorizontal(panes []*Pane, width, height int) []PaneLayout {
	if len(panes) == 0 {
		return []PaneLayout{}
	}

	layouts := make([]PaneLayout, len(panes))

	if len(panes) == 1 {
		layouts[0] = PaneLayout{
			X: 0, Y: 0, Width: width, Height: height,
		}
		return layouts
	}

	mainPaneHeight := height * 2 / 3
	sidePaneHeight := height - mainPaneHeight
	sidePaneWidth := width / (len(panes) - 1)

	// 主面板（占据上方大部分空间）
	layouts[0] = PaneLayout{
		X: 0, Y: 0, Width: width, Height: mainPaneHeight,
	}

	// 侧面板（水平排列在下方）
	for i := 1; i < len(panes); i++ {
		layouts[i] = PaneLayout{
			X:      (i - 1) * sidePaneWidth,
			Y:      mainPaneHeight,
			Width:  sidePaneWidth,
			Height: sidePaneHeight,
		}
	}

	return layouts
}

// layoutTiled 计算平铺网格布局
//
// 参数:
//   - panes: 要布局的面板列表
//   - width: 可用的总宽度
//   - height: 可用的总高度
//
// 返回:
//   - []PaneLayout: 每个面板的布局信息
//
// 该布局将面板排列成尽可能接近正方形的网格，
// 自动计算最优的行列数来容纳所有面板
func (ui *UIRenderer) layoutTiled(panes []*Pane, width, height int) []PaneLayout {
	if len(panes) == 0 {
		return []PaneLayout{}
	}

	layouts := make([]PaneLayout, len(panes))

	// 计算最优的列数（尽可能接近正方形网格）
	numColumns := 1
	for numColumns*numColumns < len(panes) {
		numColumns++
	}
	numRows := (len(panes) + numColumns - 1) / numColumns

	paneWidth := width / numColumns
	paneHeight := height / numRows

	for i := range layouts {
		columnIndex := i % numColumns
		rowIndex := i / numColumns

		layouts[i] = PaneLayout{
			X:      columnIndex * paneWidth,
			Y:      rowIndex * paneHeight,
			Width:  paneWidth,
			Height: paneHeight,
		}
	}

	return layouts
}

// renderPane 渲染单个面板
func (ui *UIRenderer) renderPane(layout *PaneLayout) {
	// 渲染边框
	ui.renderBorder(layout)

	// 渲染标题
	ui.renderPaneTitle(layout)

	// 渲染内容
	ui.renderPaneContent(layout)
}

// renderBorder 渲染边框
func (ui *UIRenderer) renderBorder(layout *PaneLayout) {
	x, y := layout.X, layout.Y
	w, h := layout.Width, layout.Height

	// 确保不越界
	if x >= ui.width || y >= ui.height {
		return
	}

	// 调整尺寸以适应缓冲区
	if x+w > ui.width {
		w = ui.width - x
	}
	if y+h > ui.height {
		h = ui.height - y
	}

	// 绘制边框
	borderRunes := []rune("┌─┐│└┘│")
	if len(borderRunes) < 7 {
		borderRunes = []rune("+-+|+-|")
	}

	// 顶边
	if y < ui.height && x < ui.width {
		ui.buffer[y][x] = borderRunes[0] // ┌
		for i := 1; i < w-1; i++ {
			if x+i < ui.width {
				ui.buffer[y][x+i] = borderRunes[1] // ─
			}
		}
		if x+w-1 < ui.width {
			ui.buffer[y][x+w-1] = borderRunes[2] // ┐
		}
	}

	// 左右边
	for i := 1; i < h-1; i++ {
		if y+i < ui.height {
			if x < ui.width {
				ui.buffer[y+i][x] = borderRunes[3] // │
			}
			if x+w-1 < ui.width {
				ui.buffer[y+i][x+w-1] = borderRunes[6] // │
			}
		}
	}

	// 底边
	if y+h-1 < ui.height && x < ui.width {
		ui.buffer[y+h-1][x] = borderRunes[4] // └
		for i := 1; i < w-1; i++ {
			if x+i < ui.width {
				ui.buffer[y+h-1][x+i] = borderRunes[1] // ─
			}
		}
		if x+w-1 < ui.width {
			ui.buffer[y+h-1][x+w-1] = borderRunes[5] // ┘
		}
	}
}

// renderPaneTitle 渲染面板标题
func (ui *UIRenderer) renderPaneTitle(layout *PaneLayout) {
	if layout.Y >= ui.height {
		return
	}

	title := layout.Title
	if layout.Active {
		title = "* " + title + " *"
	}

	// 限制标题长度
	maxTitleLen := layout.Width - 4
	if len(title) > maxTitleLen {
		title = title[:maxTitleLen]
	}

	// 渲染标题
	startX := layout.X + 2
	for i, r := range []rune(title) {
		if startX+i < ui.width && startX+i >= 0 {
			ui.buffer[layout.Y][startX+i] = r
		}
	}
}

// renderPaneContent 渲染面板内容
func (ui *UIRenderer) renderPaneContent(layout *PaneLayout) {
	contentY := layout.Y + 1
	contentHeight := layout.Height - 2
	contentWidth := layout.Width - 2

	// 渲染缓冲区内容
	for i, line := range layout.Buffer {
		if i >= contentHeight {
			break
		}

		lineY := contentY + i
		if lineY >= ui.height {
			break
		}

		// 限制行长度
		lineRunes := []rune(line)
		if len(lineRunes) > contentWidth {
			lineRunes = lineRunes[:contentWidth]
		}

		// 渲染行内容
		for j, r := range lineRunes {
			x := layout.X + 1 + j
			if x < ui.width && x >= 0 {
				ui.buffer[lineY][x] = r
			}
		}
	}
}

// renderStatusBar 渲染状态栏
func (ui *UIRenderer) renderStatusBar(window *Window) {
	statusY := ui.height - 2
	if statusY < 0 {
		return
	}

	status := ui.statusBar
	if status == "" {
		status = fmt.Sprintf("Window: %s | Panes: %d | Layout: %s",
			window.Name, len(window.Panes), window.Layout)
	}

	// 清空状态栏行
	for i := 0; i < ui.width; i++ {
		ui.buffer[statusY][i] = ' '
	}

	// 渲染状态文本
	statusRunes := []rune(status)
	for i, r := range statusRunes {
		if i >= ui.width {
			break
		}
		ui.buffer[statusY][i] = r
	}
}

// renderHelpInfo 渲染帮助信息
func (ui *UIRenderer) renderHelpInfo() {
	helpY := ui.height - 1
	if helpY < 0 {
		return
	}

	helpText := strings.Join(ui.keyHelp[:3], " | ") // 只显示前3个
	if len(helpText) > ui.width-10 {
		helpText = helpText[:ui.width-10] + "..."
	}

	// 清空帮助行
	for i := 0; i < ui.width; i++ {
		ui.buffer[helpY][i] = ' '
	}

	// 渲染帮助文本
	helpRunes := []rune(helpText)
	for i, r := range helpRunes {
		if i >= ui.width {
			break
		}
		ui.buffer[helpY][i] = r
	}
}

// generateOutput 生成输出字符串
func (ui *UIRenderer) generateOutput() string {
	var output strings.Builder

	// 添加清屏和回到顶部
	output.WriteString("\033[2J\033[H")

	for i, row := range ui.buffer {
		if i > 0 {
			output.WriteRune('\n')
		}

		// 转换行为字符串，去除尾部空格
		line := strings.TrimRight(string(row), " ")
		output.WriteString(line)
	}

	return output.String()
}

// ShowHelp 显示帮助对话框
func (ui *UIRenderer) ShowHelp() string {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	// 创建帮助对话框
	dialogWidth := 60
	dialogHeight := 15
	startX := (ui.width - dialogWidth) / 2
	startY := (ui.height - dialogHeight) / 2

	// 保存原始缓冲区
	originalBuffer := make([][]rune, ui.height)
	for i := range originalBuffer {
		originalBuffer[i] = make([]rune, ui.width)
		copy(originalBuffer[i], ui.buffer[i])
	}

	// 绘制对话框背景
	for y := startY; y < startY+dialogHeight && y < ui.height; y++ {
		for x := startX; x < startX+dialogWidth && x < ui.width; x++ {
			if y >= 0 && x >= 0 {
				ui.buffer[y][x] = ' '
			}
		}
	}

	// 绘制对话框边框
	ui.renderDialogBorder(startX, startY, dialogWidth, dialogHeight)

	// 绘制帮助内容
	helpContent := []string{
		"ClixGo Terminal 快捷键帮助",
		"",
		"C-b d    - 断开会话",
		"C-b c    - 创建新窗口",
		"C-b \"    - 水平分割面板",
		"C-b %    - 垂直分割面板",
		"C-b o    - 切换面板",
		"C-b x    - 关闭面板",
		"C-b n    - 下一个窗口",
		"C-b p    - 上一个窗口",
		"C-b Space - 切换布局",
		"",
		"按任意键关闭",
	}

	for i, line := range helpContent {
		y := startY + 1 + i
		if y >= ui.height || y < 0 {
			continue
		}

		lineRunes := []rune(line)
		for j, r := range lineRunes {
			x := startX + 2 + j
			if x >= ui.width || x < 0 {
				break
			}
			ui.buffer[y][x] = r
		}
	}

	output := ui.generateOutput()

	// 恢复原始缓冲区
	ui.buffer = originalBuffer

	return output
}

// renderDialogBorder 渲染对话框边框
func (ui *UIRenderer) renderDialogBorder(x, y, width, height int) {
	// 顶边
	if y >= 0 && y < ui.height {
		if x >= 0 && x < ui.width {
			ui.buffer[y][x] = '┌'
		}
		for i := 1; i < width-1; i++ {
			if x+i >= 0 && x+i < ui.width {
				ui.buffer[y][x+i] = '─'
			}
		}
		if x+width-1 >= 0 && x+width-1 < ui.width {
			ui.buffer[y][x+width-1] = '┐'
		}
	}

	// 左右边
	for i := 1; i < height-1; i++ {
		if y+i >= 0 && y+i < ui.height {
			if x >= 0 && x < ui.width {
				ui.buffer[y+i][x] = '│'
			}
			if x+width-1 >= 0 && x+width-1 < ui.width {
				ui.buffer[y+i][x+width-1] = '│'
			}
		}
	}

	// 底边
	if y+height-1 >= 0 && y+height-1 < ui.height {
		if x >= 0 && x < ui.width {
			ui.buffer[y+height-1][x] = '└'
		}
		for i := 1; i < width-1; i++ {
			if x+i >= 0 && x+i < ui.width {
				ui.buffer[y+height-1][x+i] = '─'
			}
		}
		if x+width-1 >= 0 && x+width-1 < ui.width {
			ui.buffer[y+height-1][x+width-1] = '┘'
		}
	}
}
