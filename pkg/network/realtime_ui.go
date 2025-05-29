/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 实时网络监控用户界面的实现
 */

package network

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// RealtimeNetworkUI 实时网络监控UI
type RealtimeNetworkUI struct {
	app              *tview.Application
	monitor          *RealtimeNetworkMonitor
	mainFlex         *tview.Flex
	interfaceTable   *tview.Table
	connectionTable  *tview.Table
	latencyTable     *tview.Table
	alertList        *tview.List
	statusBar        *tview.TextView
	performanceGauge *tview.TextView
	systemInfo       *tview.TextView
	isRunning        bool
	updateTicker     *time.Ticker
}

// NewRealtimeNetworkUI 创建新的实时网络监控UI
func NewRealtimeNetworkUI(monitor *RealtimeNetworkMonitor) *RealtimeNetworkUI {
	ui := &RealtimeNetworkUI{
		app:     tview.NewApplication(),
		monitor: monitor,
	}

	ui.setupUI()
	ui.setupKeyBindings()

	return ui
}

// setupUI 设置UI布局
func (ui *RealtimeNetworkUI) setupUI() {
	// 创建各个组件
	ui.createInterfaceTable()
	ui.createConnectionTable()
	ui.createLatencyTable()
	ui.createAlertList()
	ui.createStatusBar()
	ui.createPerformanceGauge()
	ui.createSystemInfo()

	// 创建主布局
	ui.createMainLayout()
}

// createInterfaceTable 创建网络接口表格
func (ui *RealtimeNetworkUI) createInterfaceTable() {
	ui.interfaceTable = tview.NewTable()
	ui.interfaceTable.SetBorders(true).
		SetTitle(" 网络接口统计 ").
		SetTitleAlign(tview.AlignLeft)

	// 设置表头
	headers := []string{"接口", "状态", "MTU", "接收(MB)", "发送(MB)", "带宽入(Mbps)", "带宽出(Mbps)", "使用率%", "错误"}
	for i, header := range headers {
		ui.interfaceTable.SetCell(0, i, tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false))
	}

	ui.interfaceTable.SetFixed(1, 0)
}

// createConnectionTable 创建连接统计表格
func (ui *RealtimeNetworkUI) createConnectionTable() {
	ui.connectionTable = tview.NewTable()
	ui.connectionTable.SetBorders(true).
		SetTitle(" 连接统计 ").
		SetTitleAlign(tview.AlignLeft)

	// 设置表头
	headers := []string{"协议", "总数", "已建立", "监听", "等待", "关闭等待"}
	for i, header := range headers {
		ui.connectionTable.SetCell(0, i, tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false))
	}

	ui.connectionTable.SetFixed(1, 0)
}

// createLatencyTable 创建延迟统计表格
func (ui *RealtimeNetworkUI) createLatencyTable() {
	ui.latencyTable = tview.NewTable()
	ui.latencyTable.SetBorders(true).
		SetTitle(" 目标延迟统计 ").
		SetTitleAlign(tview.AlignLeft)

	// 设置表头
	headers := []string{"目标", "状态", "平均延迟", "最小延迟", "最大延迟", "丢包率%", "抖动"}
	for i, header := range headers {
		ui.latencyTable.SetCell(0, i, tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false))
	}

	ui.latencyTable.SetFixed(1, 0)
}

// createAlertList 创建告警列表
func (ui *RealtimeNetworkUI) createAlertList() {
	ui.alertList = tview.NewList()
	ui.alertList.SetTitle(" 实时告警 ").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)

	ui.alertList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		// 处理告警确认
		ui.acknowledgeAlert(index)
	})
}

// createStatusBar 创建状态栏
func (ui *RealtimeNetworkUI) createStatusBar() {
	ui.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false).
		SetText("[green]实时网络监控 - 按 'q' 退出, 'r' 重启监控, 'p' 暂停/恢复[white]")
}

// createPerformanceGauge 创建性能指示器
func (ui *RealtimeNetworkUI) createPerformanceGauge() {
	ui.performanceGauge = tview.NewTextView()
	ui.performanceGauge.SetTitle(" 网络性能评分 ").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)
	ui.performanceGauge.SetDynamicColors(true)
}

// createSystemInfo 创建系统信息面板
func (ui *RealtimeNetworkUI) createSystemInfo() {
	ui.systemInfo = tview.NewTextView()
	ui.systemInfo.SetTitle(" 系统资源 ").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true)
	ui.systemInfo.SetDynamicColors(true)
}

// createMainLayout 创建主布局
func (ui *RealtimeNetworkUI) createMainLayout() {
	// 顶部信息面板
	topFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.performanceGauge, 0, 1, false).
		AddItem(ui.systemInfo, 0, 1, false)

	// 中间表格区域
	middleFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.interfaceTable, 0, 2, true).
		AddItem(ui.connectionTable, 0, 1, false)

	// 底部区域
	bottomFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.latencyTable, 0, 2, false).
		AddItem(ui.alertList, 0, 1, false)

	// 主布局
	ui.mainFlex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topFlex, 8, 0, false).
		AddItem(middleFlex, 0, 2, true).
		AddItem(bottomFlex, 0, 1, false).
		AddItem(ui.statusBar, 1, 0, false)

	ui.app.SetRoot(ui.mainFlex, true)
}

// setupKeyBindings 设置键盘绑定
func (ui *RealtimeNetworkUI) setupKeyBindings() {
	ui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q', 'Q':
			ui.Stop()
			return nil
		case 'r', 'R':
			ui.restartMonitoring()
			return nil
		case 'p', 'P':
			ui.togglePause()
			return nil
		case 'c', 'C':
			ui.clearAlerts()
			return nil
		}

		switch event.Key() {
		case tcell.KeyEscape:
			ui.Stop()
			return nil
		case tcell.KeyTab:
			ui.switchFocus()
			return nil
		}

		return event
	})
}

// Start 启动UI
func (ui *RealtimeNetworkUI) Start() error {
	if ui.isRunning {
		return fmt.Errorf("UI已在运行中")
	}

	ui.isRunning = true

	// 启动监控器
	if err := ui.monitor.Start(); err != nil {
		return fmt.Errorf("启动监控器失败: %w", err)
	}

	// 启动更新协程
	go ui.updateLoop()

	// 启动UI
	return ui.app.Run()
}

// Stop 停止UI
func (ui *RealtimeNetworkUI) Stop() {
	if !ui.isRunning {
		return
	}

	ui.isRunning = false

	// 停止更新定时器
	if ui.updateTicker != nil {
		ui.updateTicker.Stop()
	}

	// 停止监控器
	ui.monitor.Stop()

	// 停止应用
	ui.app.Stop()
}

// updateLoop 更新循环
func (ui *RealtimeNetworkUI) updateLoop() {
	ui.updateTicker = time.NewTicker(1 * time.Second)
	defer ui.updateTicker.Stop()

	for ui.isRunning {
		select {
		case <-ui.updateTicker.C:
			ui.updateDisplay()
		case snapshot := <-ui.monitor.GetUpdateChannel():
			ui.updateWithSnapshot(snapshot)
		case err := <-ui.monitor.GetErrorChannel():
			ui.showError(err)
		}
	}
}

// updateDisplay 更新显示
func (ui *RealtimeNetworkUI) updateDisplay() {
	if !ui.isRunning {
		return
	}

	snapshot := ui.monitor.GetCurrentSnapshot()
	if snapshot != nil {
		ui.updateWithSnapshot(*snapshot)
	}

	ui.app.QueueUpdateDraw(func() {
		// 强制重绘
	})
}

// updateWithSnapshot 使用快照更新显示
func (ui *RealtimeNetworkUI) updateWithSnapshot(snapshot NetworkResourceSnapshot) {
	ui.app.QueueUpdateDraw(func() {
		ui.updateInterfaceTable(snapshot.Interfaces)
		ui.updateConnectionTable(snapshot.Connections)
		ui.updateLatencyTable(snapshot.TargetLatencies)
		ui.updateAlertList(snapshot.Alerts)
		ui.updatePerformanceGauge(snapshot.PerformanceScore)
		ui.updateSystemInfo(snapshot.SystemResources)
		ui.updateStatusBar(snapshot.Timestamp)
	})
}

// updateInterfaceTable 更新接口表格
func (ui *RealtimeNetworkUI) updateInterfaceTable(interfaces map[string]InterfaceStats) {
	// 清除现有数据（保留表头）
	for row := ui.interfaceTable.GetRowCount() - 1; row > 0; row-- {
		ui.interfaceTable.RemoveRow(row)
	}

	// 按接口名排序
	var names []string
	for name := range interfaces {
		names = append(names, name)
	}
	sort.Strings(names)

	row := 1
	for _, name := range names {
		iface := interfaces[name]

		status := "DOWN"
		statusColor := tcell.ColorRed
		if iface.IsUp {
			status = "UP"
			statusColor = tcell.ColorGreen
		}

		// 计算MB
		bytesInMB := float64(iface.BytesIn) / 1024 / 1024
		bytesOutMB := float64(iface.BytesOut) / 1024 / 1024

		// 错误统计
		totalErrors := iface.ErrorsIn + iface.ErrorsOut + iface.DropsIn + iface.DropsOut

		cells := []struct {
			text  string
			color tcell.Color
		}{
			{name, tcell.ColorWhite},
			{status, statusColor},
			{strconv.Itoa(iface.MTU), tcell.ColorWhite},
			{fmt.Sprintf("%.2f", bytesInMB), tcell.ColorLightCyan},
			{fmt.Sprintf("%.2f", bytesOutMB), tcell.ColorLightCyan},
			{fmt.Sprintf("%.2f", iface.BandwidthInMbps), tcell.ColorYellow},
			{fmt.Sprintf("%.2f", iface.BandwidthOutMbps), tcell.ColorYellow},
			{fmt.Sprintf("%.1f", iface.Utilization), ui.getUtilizationColor(iface.Utilization)},
			{strconv.FormatUint(totalErrors, 10), ui.getErrorColor(totalErrors)},
		}

		for col, cell := range cells {
			ui.interfaceTable.SetCell(row, col, tview.NewTableCell(cell.text).
				SetTextColor(cell.color).
				SetAlign(tview.AlignCenter))
		}

		row++
	}
}

// updateConnectionTable 更新连接表格
func (ui *RealtimeNetworkUI) updateConnectionTable(connections ConnectionSummary) {
	// 清除现有数据（保留表头）
	for row := ui.connectionTable.GetRowCount() - 1; row > 0; row-- {
		ui.connectionTable.RemoveRow(row)
	}

	// TCP行
	ui.connectionTable.SetCell(1, 0, tview.NewTableCell("TCP").SetTextColor(tcell.ColorWhite))
	ui.connectionTable.SetCell(1, 1, tview.NewTableCell(strconv.Itoa(connections.TCP)).SetTextColor(tcell.ColorLightCyan))
	ui.connectionTable.SetCell(1, 2, tview.NewTableCell(strconv.Itoa(connections.Established)).SetTextColor(tcell.ColorGreen))
	ui.connectionTable.SetCell(1, 3, tview.NewTableCell(strconv.Itoa(connections.Listen)).SetTextColor(tcell.ColorYellow))
	ui.connectionTable.SetCell(1, 4, tview.NewTableCell(strconv.Itoa(connections.TimeWait)).SetTextColor(tcell.ColorOrange))
	ui.connectionTable.SetCell(1, 5, tview.NewTableCell(strconv.Itoa(connections.CloseWait)).SetTextColor(tcell.ColorRed))

	// UDP行
	ui.connectionTable.SetCell(2, 0, tview.NewTableCell("UDP").SetTextColor(tcell.ColorWhite))
	ui.connectionTable.SetCell(2, 1, tview.NewTableCell(strconv.Itoa(connections.UDP)).SetTextColor(tcell.ColorLightCyan))

	// 总计行
	ui.connectionTable.SetCell(3, 0, tview.NewTableCell("总计").SetTextColor(tcell.ColorYellow))
	ui.connectionTable.SetCell(3, 1, tview.NewTableCell(strconv.Itoa(connections.Total)).SetTextColor(tcell.ColorYellow))
	ui.connectionTable.SetCell(3, 2, tview.NewTableCell("-").SetTextColor(tcell.ColorGray))
	ui.connectionTable.SetCell(3, 3, tview.NewTableCell("-").SetTextColor(tcell.ColorGray))
	ui.connectionTable.SetCell(3, 4, tview.NewTableCell("-").SetTextColor(tcell.ColorGray))
	ui.connectionTable.SetCell(3, 5, tview.NewTableCell("-").SetTextColor(tcell.ColorGray))
}

// updateLatencyTable 更新延迟表格
func (ui *RealtimeNetworkUI) updateLatencyTable(latencies map[string]LatencyStats) {
	// 清除现有数据（保留表头）
	for row := ui.latencyTable.GetRowCount() - 1; row > 0; row-- {
		ui.latencyTable.RemoveRow(row)
	}

	// 按目标名排序
	var targets []string
	for target := range latencies {
		targets = append(targets, target)
	}
	sort.Strings(targets)

	row := 1
	for _, target := range targets {
		latency := latencies[target]

		status := "不可达"
		statusColor := tcell.ColorRed
		if latency.IsReachable {
			status = "可达"
			statusColor = tcell.ColorGreen
		}

		avgLatencyMs := float64(latency.AvgLatency.Nanoseconds()) / 1e6
		minLatencyMs := float64(latency.MinLatency.Nanoseconds()) / 1e6
		maxLatencyMs := float64(latency.MaxLatency.Nanoseconds()) / 1e6
		jitterMs := float64(latency.Jitter.Nanoseconds()) / 1e6

		cells := []struct {
			text  string
			color tcell.Color
		}{
			{target, tcell.ColorWhite},
			{status, statusColor},
			{fmt.Sprintf("%.2fms", avgLatencyMs), ui.getLatencyColor(avgLatencyMs)},
			{fmt.Sprintf("%.2fms", minLatencyMs), tcell.ColorLightCyan},
			{fmt.Sprintf("%.2fms", maxLatencyMs), tcell.ColorLightCyan},
			{fmt.Sprintf("%.2f%%", latency.PacketLoss), ui.getPacketLossColor(latency.PacketLoss)},
			{fmt.Sprintf("%.2fms", jitterMs), tcell.ColorWhite},
		}

		for col, cell := range cells {
			ui.latencyTable.SetCell(row, col, tview.NewTableCell(cell.text).
				SetTextColor(cell.color).
				SetAlign(tview.AlignCenter))
		}

		row++
	}
}

// updateAlertList 更新告警列表
func (ui *RealtimeNetworkUI) updateAlertList(alerts []Alert) {
	ui.alertList.Clear()

	if len(alerts) == 0 {
		ui.alertList.AddItem("无告警", "", 0, nil)
		return
	}

	// 按时间倒序排列
	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].Timestamp.After(alerts[j].Timestamp)
	})

	for i, alert := range alerts {
		if i >= 10 { // 最多显示10个告警
			break
		}

		timeStr := alert.Timestamp.Format("15:04:05")
		mainText := fmt.Sprintf("[%s] %s", ui.getSeverityColor(alert.Severity), alert.Type)
		secondaryText := fmt.Sprintf("%s - %s", timeStr, alert.Message)

		ui.alertList.AddItem(mainText, secondaryText, 0, nil)
	}
}

// updatePerformanceGauge 更新性能指示器
func (ui *RealtimeNetworkUI) updatePerformanceGauge(score float64) {
	var color string
	var status string

	if score >= 90 {
		color = "green"
		status = "优秀"
	} else if score >= 75 {
		color = "yellow"
		status = "良好"
	} else if score >= 60 {
		color = "orange"
		status = "一般"
	} else {
		color = "red"
		status = "较差"
	}

	// 创建进度条
	barLength := 20
	filledLength := int(score / 100 * float64(barLength))
	bar := strings.Repeat("█", filledLength) + strings.Repeat("░", barLength-filledLength)

	text := fmt.Sprintf("[%s]评分: %.1f/100\n状态: %s\n\n进度: [%s]%s[white]\n\n更新时间: %s",
		color, score, status, color, bar, time.Now().Format("15:04:05"))

	ui.performanceGauge.SetText(text)
}

// updateSystemInfo 更新系统信息
func (ui *RealtimeNetworkUI) updateSystemInfo(resources SystemNetworkResources) {
	text := fmt.Sprintf(`[cyan]打开文件: [white]%d/%d
[cyan]网络线程: [white]%d
[cyan]内存使用: [white]%.2f MB
[cyan]套接字缓冲: [white]%d

[yellow]监控状态:[white]
运行中: %v
历史记录: %d`,
		resources.OpenFiles,
		resources.MaxOpenFiles,
		resources.NetworkThreads,
		resources.MemoryUsageMB,
		resources.SocketBuffers,
		ui.monitor.IsRunning(),
		len(ui.monitor.GetHistory()))

	ui.systemInfo.SetText(text)
}

// updateStatusBar 更新状态栏
func (ui *RealtimeNetworkUI) updateStatusBar(timestamp time.Time) {
	status := "[green]运行中"
	if !ui.monitor.IsRunning() {
		status = "[red]已停止"
	}

	text := fmt.Sprintf("%s [white]| 最后更新: %s | 按键: [yellow]q[white]=退出 [yellow]r[white]=重启 [yellow]p[white]=暂停 [yellow]c[white]=清除告警",
		status, timestamp.Format("15:04:05"))

	ui.statusBar.SetText(text)
}

// 辅助方法：获取颜色
func (ui *RealtimeNetworkUI) getUtilizationColor(utilization float64) tcell.Color {
	if utilization > 80 {
		return tcell.ColorRed
	} else if utilization > 60 {
		return tcell.ColorOrange
	} else if utilization > 40 {
		return tcell.ColorYellow
	}
	return tcell.ColorGreen
}

func (ui *RealtimeNetworkUI) getErrorColor(errors uint64) tcell.Color {
	if errors > 100 {
		return tcell.ColorRed
	} else if errors > 10 {
		return tcell.ColorOrange
	} else if errors > 0 {
		return tcell.ColorYellow
	}
	return tcell.ColorGreen
}

func (ui *RealtimeNetworkUI) getLatencyColor(latencyMs float64) tcell.Color {
	if latencyMs > 100 {
		return tcell.ColorRed
	} else if latencyMs > 50 {
		return tcell.ColorOrange
	} else if latencyMs > 20 {
		return tcell.ColorYellow
	}
	return tcell.ColorGreen
}

func (ui *RealtimeNetworkUI) getPacketLossColor(packetLoss float64) tcell.Color {
	if packetLoss > 5 {
		return tcell.ColorRed
	} else if packetLoss > 1 {
		return tcell.ColorOrange
	} else if packetLoss > 0 {
		return tcell.ColorYellow
	}
	return tcell.ColorGreen
}

func (ui *RealtimeNetworkUI) getSeverityColor(severity string) string {
	switch severity {
	case "critical":
		return "red"
	case "warning":
		return "yellow"
	case "info":
		return "blue"
	default:
		return "white"
	}
}

// 交互方法
func (ui *RealtimeNetworkUI) restartMonitoring() {
	ui.monitor.Stop()
	time.Sleep(100 * time.Millisecond)
	ui.monitor.Start()
}

func (ui *RealtimeNetworkUI) togglePause() {
	if ui.monitor.IsRunning() {
		ui.monitor.Stop()
	} else {
		ui.monitor.Start()
	}
}

func (ui *RealtimeNetworkUI) clearAlerts() {
	ui.alertList.Clear()
	ui.alertList.AddItem("告警已清除", "", 0, nil)
}

func (ui *RealtimeNetworkUI) acknowledgeAlert(index int) {
	// 这里可以实现告警确认逻辑
	ui.alertList.SetItemText(index, "[green]已确认", "")
}

func (ui *RealtimeNetworkUI) switchFocus() {
	// 实现焦点切换逻辑
	current := ui.app.GetFocus()
	switch current {
	case ui.interfaceTable:
		ui.app.SetFocus(ui.connectionTable)
	case ui.connectionTable:
		ui.app.SetFocus(ui.latencyTable)
	case ui.latencyTable:
		ui.app.SetFocus(ui.alertList)
	case ui.alertList:
		ui.app.SetFocus(ui.interfaceTable)
	default:
		ui.app.SetFocus(ui.interfaceTable)
	}
}

func (ui *RealtimeNetworkUI) showError(err error) {
	ui.app.QueueUpdateDraw(func() {
		ui.statusBar.SetText(fmt.Sprintf("[red]错误: %v", err))
	})
}
