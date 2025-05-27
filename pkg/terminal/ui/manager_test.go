package ui

import (
	"fmt"
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
