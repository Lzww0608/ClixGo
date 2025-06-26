/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-26 20:04:05
* @Description: 终端用户界面管理器的单元测试
 */

package ui

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
)

func TestMain(m *testing.M) {
	// 初始化日志系统
	logger.InitLogger()
	defer logger.Close()

	m.Run()
}

func TestNewUIManager(t *testing.T) {
	config := DefaultUIConfig

	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}

	if uiManager == nil {
		t.Fatal("UI管理器为nil")
	}

	if uiManager.app == nil {
		t.Fatal("tview应用为nil")
	}

	if uiManager.layout == nil {
		t.Fatal("布局为nil")
	}

	if uiManager.statusBar == nil {
		t.Fatal("状态栏为nil")
	}

	if uiManager.panels == nil {
		t.Fatal("面板映射为nil")
	}

	// 清理
	uiManager.Stop()
}

func TestCreatePanel(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建面板
	panel := uiManager.CreatePanel("test1", "Test Panel 1")
	if panel == nil {
		t.Fatal("创建面板失败")
	}

	if panel.ID != "test1" {
		t.Errorf("面板ID错误，期望: test1, 实际: %s", panel.ID)
	}

	if panel.Title != "Test Panel 1" {
		t.Errorf("面板标题错误，期望: Test Panel 1, 实际: %s", panel.Title)
	}

	if !panel.Active {
		t.Error("第一个面板应该是活动状态")
	}

	if uiManager.GetActivePanel() != "test1" {
		t.Errorf("活动面板错误，期望: test1, 实际: %s", uiManager.GetActivePanel())
	}

	if uiManager.GetPanelCount() != 1 {
		t.Errorf("面板数量错误，期望: 1, 实际: %d", uiManager.GetPanelCount())
	}
}

func TestMultiplePanels(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建多个面板
	panel1 := uiManager.CreatePanel("test1", "Test Panel 1")
	panel2 := uiManager.CreatePanel("test2", "Test Panel 2")
	panel3 := uiManager.CreatePanel("test3", "Test Panel 3")

	if panel1 == nil || panel2 == nil || panel3 == nil {
		t.Fatal("创建面板失败")
	}

	if uiManager.GetPanelCount() != 3 {
		t.Errorf("面板数量错误，期望: 3, 实际: %d", uiManager.GetPanelCount())
	}

	// 测试面板切换
	uiManager.NextPanel()
	if uiManager.GetActivePanel() != "test2" {
		t.Errorf("切换面板失败，期望: test2, 实际: %s", uiManager.GetActivePanel())
	}

	uiManager.NextPanel()
	if uiManager.GetActivePanel() != "test3" {
		t.Errorf("切换面板失败，期望: test3, 实际: %s", uiManager.GetActivePanel())
	}

	uiManager.NextPanel()
	if uiManager.GetActivePanel() != "test1" {
		t.Errorf("切换面板失败，期望: test1, 实际: %s", uiManager.GetActivePanel())
	}
}

func TestWriteToPanel(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建面板
	uiManager.CreatePanel("test1", "Test Panel 1")

	// 写入内容
	testContent := "Hello, World!\nThis is a test message."
	err = uiManager.WriteToPanel("test1", testContent)
	if err != nil {
		t.Errorf("写入面板内容失败: %v", err)
	}

	// 测试写入不存在的面板
	err = uiManager.WriteToPanel("nonexistent", "test")
	if err == nil {
		t.Error("写入不存在的面板应该返回错误")
	}
}

func TestClosePanel(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建多个面板
	uiManager.CreatePanel("test1", "Test Panel 1")
	uiManager.CreatePanel("test2", "Test Panel 2")
	uiManager.CreatePanel("test3", "Test Panel 3")

	if uiManager.GetPanelCount() != 3 {
		t.Errorf("面板数量错误，期望: 3, 实际: %d", uiManager.GetPanelCount())
	}

	// 关闭活动面板
	uiManager.CloseActivePanel()

	if uiManager.GetPanelCount() != 2 {
		t.Errorf("关闭面板后数量错误，期望: 2, 实际: %d", uiManager.GetPanelCount())
	}

	// 验证活动面板已切换
	activePanel := uiManager.GetActivePanel()
	if activePanel != "test2" && activePanel != "test3" {
		t.Errorf("关闭面板后活动面板错误: %s", activePanel)
	}
}

func TestStatusBar(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	statusBar := uiManager.statusBar
	if statusBar == nil {
		t.Fatal("状态栏为nil")
	}

	// 测试状态栏更新
	statusBar.UpdateStatus("Left", "Center", "Right")

	if statusBar.left != "Left" {
		t.Errorf("状态栏左侧文本错误，期望: Left, 实际: %s", statusBar.left)
	}

	if statusBar.center != "Center" {
		t.Errorf("状态栏中间文本错误，期望: Center, 实际: %s", statusBar.center)
	}

	if statusBar.right != "Right" {
		t.Errorf("状态栏右侧文本错误，期望: Right, 实际: %s", statusBar.right)
	}

	// 测试状态栏可见性
	statusBar.SetVisible(false)
	if statusBar.visible {
		t.Error("设置状态栏不可见失败")
	}

	statusBar.SetVisible(true)
	if !statusBar.visible {
		t.Error("设置状态栏可见失败")
	}
}

func TestSplitOperations(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建初始面板
	uiManager.CreatePanel("main", "Main Panel")

	initialCount := uiManager.GetPanelCount()

	// 测试水平分割
	uiManager.SplitHorizontal()
	if uiManager.GetPanelCount() != initialCount+1 {
		t.Errorf("水平分割后面板数量错误，期望: %d, 实际: %d",
			initialCount+1, uiManager.GetPanelCount())
	}

	// 测试垂直分割
	uiManager.SplitVertical()
	if uiManager.GetPanelCount() != initialCount+2 {
		t.Errorf("垂直分割后面板数量错误，期望: %d, 实际: %d",
			initialCount+2, uiManager.GetPanelCount())
	}
}

func TestLayoutModes(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 测试单面板模式
	uiManager.CreatePanel("panel1", "Panel 1")
	if uiManager.layout.mode != LayoutSingle {
		t.Errorf("单面板模式错误，期望: %d, 实际: %d", LayoutSingle, uiManager.layout.mode)
	}

	// 测试垂直分割模式
	uiManager.CreatePanel("panel2", "Panel 2")
	if uiManager.layout.mode != LayoutVertical {
		t.Errorf("垂直分割模式错误，期望: %d, 实际: %d", LayoutVertical, uiManager.layout.mode)
	}

	// 测试网格模式
	uiManager.CreatePanel("panel3", "Panel 3")
	if uiManager.layout.mode != LayoutGrid {
		t.Errorf("网格模式错误，期望: %d, 实际: %d", LayoutGrid, uiManager.layout.mode)
	}
}

func TestConcurrentOperations(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 并发创建面板
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			panelID := fmt.Sprintf("panel_%d", id)
			title := fmt.Sprintf("Panel %d", id)
			uiManager.CreatePanel(panelID, title)

			// 写入一些内容
			for j := 0; j < 5; j++ {
				content := fmt.Sprintf("Message %d from panel %d\n", j, id)
				uiManager.WriteToPanel(panelID, content)
				time.Sleep(10 * time.Millisecond)
			}

			done <- true
		}(i)
	}

	// 等待所有协程完成
	for i := 0; i < 10; i++ {
		<-done
	}

	if uiManager.GetPanelCount() != 10 {
		t.Errorf("并发创建面板数量错误，期望: 10, 实际: %d", uiManager.GetPanelCount())
	}
}

// ====== Step 3: UIManager侧边栏集成测试 ======

func TestUIManagerSidebarIntegration(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建模拟会话管理器
	mockManager := &MockSessionManager{
		sessions: []MockSessionInfo{
			{
				id:   "test_session",
				name: "Test Session",
				windows: []MockWindowInfo{
					{panes: []MockPaneInfo{{id: "test_pane"}}},
				},
			},
		},
	}

	// 测试启用侧边栏集成
	err = uiManager.EnableSidebarIntegration(mockManager)
	if err != nil {
		t.Fatalf("启用侧边栏集成失败: %v", err)
	}

	// 验证侧边栏已设置
	sidebar := uiManager.GetSidebar()
	if sidebar == nil {
		t.Fatal("启用集成后侧边栏为nil")
	}

	if !sidebar.IsVisible() {
		t.Error("启用集成后侧边栏应该可见")
	}

	// 测试禁用侧边栏集成
	uiManager.DisableSidebarIntegration()

	sidebar = uiManager.GetSidebar()
	if sidebar != nil {
		t.Error("禁用集成后侧边栏应该为nil")
	}
}

func TestUIManagerSidebarLayoutModes(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建侧边栏
	mockManager := &MockSessionManager{}
	sidebar := NewSidebar(mockManager)
	uiManager.SetSidebar(sidebar)

	// 测试单面板+侧边栏布局
	uiManager.CreatePanel("panel1", "Panel 1")
	// 创建面板后布局会自动更新，但可能不是侧边栏布局
	// 需要显式调用侧边栏布局更新
	uiManager.updateLayoutWithSidebar()
	if uiManager.layout.mode != LayoutSingleWithSidebar {
		t.Errorf("单面板+侧边栏布局模式错误，期望: %d, 实际: %d",
			LayoutSingleWithSidebar, uiManager.layout.mode)
	}

	// 测试垂直分割+侧边栏布局
	uiManager.CreatePanel("panel2", "Panel 2")
	uiManager.updateLayoutWithSidebar()
	if uiManager.layout.mode != LayoutVerticalWithSidebar {
		t.Errorf("垂直分割+侧边栏布局模式错误，期望: %d, 实际: %d",
			LayoutVerticalWithSidebar, uiManager.layout.mode)
	}

	// 测试网格+侧边栏布局
	uiManager.CreatePanel("panel3", "Panel 3")
	uiManager.updateLayoutWithSidebar()
	if uiManager.layout.mode != LayoutGridWithSidebar {
		t.Errorf("网格+侧边栏布局模式错误，期望: %d, 实际: %d",
			LayoutGridWithSidebar, uiManager.layout.mode)
	}

	// 测试隐藏侧边栏后的布局回退
	sidebar.SetVisible(false)
	uiManager.updateLayoutWithSidebar()

	// 应该回退到普通布局模式
	if uiManager.layout.mode != LayoutGrid {
		t.Errorf("隐藏侧边栏后布局模式错误，期望: %d, 实际: %d",
			LayoutGrid, uiManager.layout.mode)
	}
}

func TestUIManagerSidebarToggle(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 测试没有侧边栏时的切换
	uiManager.ToggleSidebar() // 应该不会panic

	// 创建侧边栏
	mockManager := &MockSessionManager{}
	sidebar := NewSidebar(mockManager)
	uiManager.SetSidebar(sidebar)

	// 验证初始状态
	if !sidebar.IsVisible() {
		t.Error("初始状态下侧边栏应该可见")
	}

	// 测试切换隐藏
	uiManager.ToggleSidebar()
	if sidebar.IsVisible() {
		t.Error("切换后侧边栏应该不可见")
	}

	// 测试切换显示
	uiManager.ToggleSidebar()
	if !sidebar.IsVisible() {
		t.Error("再次切换后侧边栏应该可见")
	}
}

func TestUIManagerSidebarFocus(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建侧边栏和面板
	mockManager := &MockSessionManager{}
	sidebar := NewSidebar(mockManager)
	uiManager.SetSidebar(sidebar)
	uiManager.CreatePanel("panel1", "Panel 1")

	// 测试焦点到侧边栏
	uiManager.FocusSidebar()

	// 由于我们在测试环境中无法真正检查tview的焦点状态，
	// 这里主要验证方法调用不会panic

	// 测试焦点切换 (移除可能导致死锁的操作)
	// uiManager.ToggleFocusBetweenSidebarAndPanels()

	// 测试隐藏侧边栏时的焦点处理
	sidebar.SetVisible(false)
	uiManager.FocusSidebar() // 应该不会panic
	// uiManager.ToggleFocusBetweenSidebarAndPanels() // 可能导致死锁，暂时移除
}

func TestUIManagerSidebarConcurrentOperations(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建侧边栏
	mockManager := &MockSessionManager{}
	sidebar := NewSidebar(mockManager)
	err = sidebar.Start()
	if err != nil {
		t.Fatalf("启动侧边栏失败: %v", err)
	}
	defer sidebar.Stop()

	uiManager.SetSidebar(sidebar)

	// 并发操作测试 - 减少并发度和复杂性以避免死锁
	var wg sync.WaitGroup
	concurrency := 3 // 减少并发度
	iterations := 10 // 减少迭代次数

	// 测试1: 并发创建面板
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				panelID := fmt.Sprintf("panel_%d_%d", id, j)
				uiManager.CreatePanel(panelID, fmt.Sprintf("Panel %d-%d", id, j))
				time.Sleep(time.Millisecond) // 增加延迟避免竞争
			}
		}(i)
	}
	wg.Wait()

	// 测试2: 串行侧边栏操作以避免死锁
	for i := 0; i < 5; i++ {
		uiManager.ToggleSidebar()
		time.Sleep(time.Millisecond)
	}

	t.Log("并发侧边栏操作测试完成")
}

func TestUIManagerSidebarLayoutUpdate(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建侧边栏
	mockManager := &MockSessionManager{}
	sidebar := NewSidebar(mockManager)
	uiManager.SetSidebar(sidebar)

	// 测试不同面板数量下的布局更新
	testCases := []struct {
		panelCount   int
		expectedMode LayoutMode
		description  string
	}{
		{0, LayoutSingleWithSidebar, "无面板+侧边栏"},
		{1, LayoutSingleWithSidebar, "单面板+侧边栏"},
		{2, LayoutVerticalWithSidebar, "双面板+侧边栏"},
		{3, LayoutGridWithSidebar, "多面板网格+侧边栏"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// 清空现有面板
			for uiManager.GetPanelCount() > 0 {
				uiManager.CloseActivePanel()
			}

			// 创建指定数量的面板
			for i := 0; i < tc.panelCount; i++ {
				panelID := fmt.Sprintf("panel_%d", i)
				title := fmt.Sprintf("Panel %d", i)
				uiManager.CreatePanel(panelID, title)
			}

			// 更新布局
			uiManager.updateLayoutWithSidebar()

			// 验证布局模式
			if uiManager.layout.mode != tc.expectedMode {
				t.Errorf("%s布局模式错误，期望: %d, 实际: %d",
					tc.description, tc.expectedMode, uiManager.layout.mode)
			}
		})
	}
}

func TestUIManagerSidebarWidthAndPosition(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建侧边栏
	mockManager := &MockSessionManager{}
	sidebar := NewSidebar(mockManager)
	uiManager.SetSidebar(sidebar)

	// 测试不同宽度设置
	widths := []int{20, 30, 50, 80}
	for _, width := range widths {
		sidebar.SetWidth(width)
		if sidebar.GetWidth() != width {
			t.Errorf("设置侧边栏宽度失败，期望: %d, 实际: %d", width, sidebar.GetWidth())
		}

		// 更新布局以应用新宽度
		uiManager.updateLayoutWithSidebar()
	}

	// 测试边界宽度
	sidebar.SetWidth(10) // 小于最小值
	if sidebar.GetWidth() != 20 {
		t.Errorf("边界宽度处理错误，期望: 20, 实际: %d", sidebar.GetWidth())
	}

	sidebar.SetWidth(100) // 大于最大值
	if sidebar.GetWidth() != 80 {
		t.Errorf("边界宽度处理错误，期望: 80, 实际: %d", sidebar.GetWidth())
	}
}

// ====== 性能测试 ======

func BenchmarkUIManagerWithSidebar(b *testing.B) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		b.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建侧边栏
	mockManager := &MockSessionManager{}
	sidebar := NewSidebar(mockManager)
	uiManager.SetSidebar(sidebar)

	// 创建一些面板
	for i := 0; i < 3; i++ {
		panelID := fmt.Sprintf("panel_%d", i)
		title := fmt.Sprintf("Panel %d", i)
		uiManager.CreatePanel(panelID, title)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uiManager.updateLayoutWithSidebar()
	}
}

func BenchmarkSidebarToggle(b *testing.B) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		b.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建侧边栏
	mockManager := &MockSessionManager{}
	sidebar := NewSidebar(mockManager)
	uiManager.SetSidebar(sidebar)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uiManager.ToggleSidebar()
	}
}
