/*
* @Author: Lzww0608
* @Date: 2025-6-22 17:40:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-22 17:40:00
* @Description: Step 4 布局管理增强的单元测试
 */

package ui

import (
	"fmt"
	"testing"
)

// ====== Step 4: 布局管理器基础测试 ======

func TestLayoutManagerCreation(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 测试获取布局管理器
	layoutMgr := uiManager.GetLayoutManager()
	if layoutMgr == nil {
		t.Fatal("布局管理器不应该为nil")
	}

	// 再次获取应该返回同一个实例
	layoutMgr2 := uiManager.GetLayoutManager()
	if layoutMgr != layoutMgr2 {
		t.Error("多次获取布局管理器应该返回同一个实例")
	}
}

func TestCreateResizablePanel(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 测试创建可调整大小的面板
	err = uiManager.CreateResizablePanel("test_panel", "Test Panel", true, true)
	if err != nil {
		t.Fatalf("创建可调整面板失败: %v", err)
	}

	// 验证面板属性
	panel, exists := uiManager.panels["test_panel"]
	if !exists {
		t.Fatal("面板未创建")
	}

	if !panel.Resizable {
		t.Error("面板应该可调整大小")
	}

	if !panel.Draggable {
		t.Error("面板应该可拖拽")
	}

	if panel.MinSize.Width != 20 || panel.MinSize.Height != 5 {
		t.Errorf("最小尺寸错误，期望: 20x5, 实际: %dx%d", panel.MinSize.Width, panel.MinSize.Height)
	}

	// 测试重复创建
	err = uiManager.CreateResizablePanel("test_panel", "Test Panel 2", false, false)
	if err == nil {
		t.Error("重复创建面板应该失败")
	}
}

func TestEnableAdvancedLayoutFeatures(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建普通面板
	uiManager.CreatePanel("panel1", "Panel 1")
	uiManager.CreatePanel("panel2", "Panel 2")

	// 启用高级布局功能
	err = uiManager.EnableAdvancedLayoutFeatures()
	if err != nil {
		t.Fatalf("启用高级布局功能失败: %v", err)
	}

	// 验证面板已启用高级功能
	for _, panel := range uiManager.panels {
		if !panel.Resizable {
			t.Errorf("面板 %s 应该可调整大小", panel.ID)
		}
		if !panel.Draggable {
			t.Errorf("面板 %s 应该可拖拽", panel.ID)
		}
	}
}

// ====== Step 4: 布局配置测试 ======

func TestLayoutConfigOperations(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	layoutMgr := uiManager.GetLayoutManager()

	// 创建一些面板
	uiManager.CreateResizablePanel("panel1", "Panel 1", true, true)
	uiManager.CreateResizablePanel("panel2", "Panel 2", true, true)

	// 测试获取当前布局
	currentLayout := layoutMgr.GetCurrentLayout()
	if currentLayout.Name != "current" {
		t.Errorf("当前布局名称错误，期望: current, 实际: %s", currentLayout.Name)
	}

	if len(currentLayout.PanelLayouts) != 2 {
		t.Errorf("面板布局数量错误，期望: 2, 实际: %d", len(currentLayout.PanelLayouts))
	}

	// 测试保存布局
	err = layoutMgr.SaveLayout("test_layout")
	if err != nil {
		t.Fatalf("保存布局失败: %v", err)
	}

	// 测试列出布局
	layouts := layoutMgr.ListLayouts()
	if len(layouts) != 1 {
		t.Errorf("布局数量错误，期望: 1, 实际: %d", len(layouts))
	}

	if layouts[0] != "test_layout" {
		t.Errorf("布局名称错误，期望: test_layout, 实际: %s", layouts[0])
	}

	// 测试加载布局
	err = layoutMgr.LoadLayout("test_layout")
	if err != nil {
		t.Fatalf("加载布局失败: %v", err)
	}

	// 测试删除布局
	err = layoutMgr.DeleteLayout("test_layout")
	if err != nil {
		t.Fatalf("删除布局失败: %v", err)
	}

	// 验证布局已删除
	layouts = layoutMgr.ListLayouts()
	if len(layouts) != 0 {
		t.Errorf("删除后布局数量应该为0，实际: %d", len(layouts))
	}
}

func TestLayoutConfigWithSidebar(t *testing.T) {
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

	layoutMgr := uiManager.GetLayoutManager()

	// 获取包含侧边栏的布局配置
	currentLayout := layoutMgr.GetCurrentLayout()
	if !currentLayout.SidebarVisible {
		t.Error("侧边栏应该可见")
	}

	if currentLayout.SidebarWidth != sidebar.GetWidth() {
		t.Errorf("侧边栏宽度错误，期望: %d, 实际: %d", sidebar.GetWidth(), currentLayout.SidebarWidth)
	}
}

// ====== Step 4: 面板操作测试 ======

func TestResizePanel(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	layoutMgr := uiManager.GetLayoutManager()

	// 创建可调整大小的面板
	uiManager.CreateResizablePanel("test_panel", "Test Panel", true, true)

	// 测试调整面板大小
	newSize := Size{Width: 100, Height: 30}
	err = layoutMgr.ResizePanel("test_panel", newSize)
	if err != nil {
		t.Fatalf("调整面板大小失败: %v", err)
	}

	// 验证面板大小已更新
	panel := uiManager.panels["test_panel"]
	if panel.Size.Width != newSize.Width || panel.Size.Height != newSize.Height {
		t.Errorf("面板大小未正确更新，期望: %dx%d, 实际: %dx%d",
			newSize.Width, newSize.Height, panel.Size.Width, panel.Size.Height)
	}

	// 测试调整不存在的面板
	err = layoutMgr.ResizePanel("nonexistent", newSize)
	if err == nil {
		t.Error("调整不存在的面板应该失败")
	}

	// 测试调整不可调整大小的面板
	uiManager.CreateResizablePanel("fixed_panel", "Fixed Panel", false, true)
	err = layoutMgr.ResizePanel("fixed_panel", newSize)
	if err == nil {
		t.Error("调整不可调整大小的面板应该失败")
	}
}

func TestMovePanel(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	layoutMgr := uiManager.GetLayoutManager()

	// 创建可拖拽的面板
	uiManager.CreateResizablePanel("test_panel", "Test Panel", true, true)

	// 测试移动面板
	newPos := Position{X: 10, Y: 5}
	err = layoutMgr.MovePanel("test_panel", newPos)
	if err != nil {
		t.Fatalf("移动面板失败: %v", err)
	}

	// 验证面板位置已更新
	panel := uiManager.panels["test_panel"]
	if panel.Position.X != newPos.X || panel.Position.Y != newPos.Y {
		t.Errorf("面板位置未正确更新，期望: (%d,%d), 实际: (%d,%d)",
			newPos.X, newPos.Y, panel.Position.X, panel.Position.Y)
	}

	// 测试移动不可拖拽的面板
	uiManager.CreateResizablePanel("fixed_panel", "Fixed Panel", true, false)
	err = layoutMgr.MovePanel("fixed_panel", newPos)
	if err == nil {
		t.Error("移动不可拖拽的面板应该失败")
	}
}

func TestSetPanelConstraints(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	layoutMgr := uiManager.GetLayoutManager()

	// 创建面板
	uiManager.CreateResizablePanel("test_panel", "Test Panel", true, true)

	// 设置约束
	constraints := Constraints{
		FixedWidth:   true,
		FixedHeight:  false,
		AspectRatio:  1.5,
		AlignX:       AlignCenter,
		AlignY:       AlignStart,
		MarginLeft:   5,
		MarginTop:    3,
		MarginRight:  5,
		MarginBottom: 3,
	}

	err = layoutMgr.SetPanelConstraints("test_panel", constraints)
	if err != nil {
		t.Fatalf("设置面板约束失败: %v", err)
	}

	// 验证约束已设置
	panel := uiManager.panels["test_panel"]
	if panel.Constraints == nil {
		t.Fatal("面板约束未设置")
	}

	if panel.Constraints.FixedWidth != constraints.FixedWidth {
		t.Errorf("固定宽度约束错误，期望: %t, 实际: %t",
			constraints.FixedWidth, panel.Constraints.FixedWidth)
	}

	if panel.Constraints.AspectRatio != constraints.AspectRatio {
		t.Errorf("宽高比约束错误，期望: %.2f, 实际: %.2f",
			constraints.AspectRatio, panel.Constraints.AspectRatio)
	}
}

// ====== Step 4: 拖拽操作测试 ======

func TestDragOperations(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	layoutMgr := uiManager.GetLayoutManager()

	// 创建可拖拽的面板
	uiManager.CreateResizablePanel("test_panel", "Test Panel", true, true)

	// 测试开始拖拽
	startPos := Position{X: 10, Y: 10}
	err = layoutMgr.StartDrag("test_panel", startPos, DragMove)
	if err != nil {
		t.Fatalf("开始拖拽失败: %v", err)
	}

	// 测试更新拖拽
	currentPos := Position{X: 15, Y: 12}
	err = layoutMgr.UpdateDrag(currentPos)
	if err != nil {
		t.Fatalf("更新拖拽失败: %v", err)
	}

	// 测试结束拖拽
	err = layoutMgr.EndDrag()
	if err != nil {
		t.Fatalf("结束拖拽失败: %v", err)
	}

	// 测试在没有活动拖拽时更新
	err = layoutMgr.UpdateDrag(currentPos)
	if err == nil {
		t.Error("在没有活动拖拽时更新应该失败")
	}

	// 测试在没有活动拖拽时结束
	err = layoutMgr.EndDrag()
	if err == nil {
		t.Error("在没有活动拖拽时结束应该失败")
	}
}

func TestDragResize(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	layoutMgr := uiManager.GetLayoutManager()

	// 创建可调整大小的面板
	uiManager.CreateResizablePanel("test_panel", "Test Panel", true, true)

	// 测试拖拽调整大小
	startPos := Position{X: 80, Y: 24}
	err = layoutMgr.StartDrag("test_panel", startPos, DragResize)
	if err != nil {
		t.Fatalf("开始拖拽调整大小失败: %v", err)
	}

	// 更新拖拽
	currentPos := Position{X: 90, Y: 30}
	err = layoutMgr.UpdateDrag(currentPos)
	if err != nil {
		t.Fatalf("更新拖拽调整大小失败: %v", err)
	}

	// 结束拖拽
	err = layoutMgr.EndDrag()
	if err != nil {
		t.Fatalf("结束拖拽调整大小失败: %v", err)
	}
}

func TestDragConstraints(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	layoutMgr := uiManager.GetLayoutManager()

	// 创建不可拖拽的面板
	uiManager.CreateResizablePanel("fixed_panel", "Fixed Panel", false, false)

	// 测试拖拽不可移动的面板
	startPos := Position{X: 10, Y: 10}
	err = layoutMgr.StartDrag("fixed_panel", startPos, DragMove)
	if err == nil {
		t.Error("拖拽不可移动的面板应该失败")
	}

	// 测试拖拽调整不可调整大小的面板
	err = layoutMgr.StartDrag("fixed_panel", startPos, DragResize)
	if err == nil {
		t.Error("拖拽调整不可调整大小的面板应该失败")
	}

	// 测试拖拽不存在的面板
	err = layoutMgr.StartDrag("nonexistent", startPos, DragMove)
	if err == nil {
		t.Error("拖拽不存在的面板应该失败")
	}
}

// ====== Step 4: 布局重置测试 ======

func TestResetLayout(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	layoutMgr := uiManager.GetLayoutManager()

	// 创建面板并修改其属性
	uiManager.CreateResizablePanel("panel1", "Panel 1", true, true)
	uiManager.CreateResizablePanel("panel2", "Panel 2", true, true)

	// 修改面板属性
	panel1 := uiManager.panels["panel1"]
	panel1.Position = Position{X: 50, Y: 50}
	panel1.Size = Size{Width: 120, Height: 40}
	panel1.ZIndex = 5

	// 重置布局
	err = layoutMgr.ResetLayout()
	if err != nil {
		t.Fatalf("重置布局失败: %v", err)
	}

	// 验证面板已重置
	if panel1.Position.X != 0 || panel1.Position.Y != 0 {
		t.Errorf("面板位置未重置，期望: (0,0), 实际: (%d,%d)",
			panel1.Position.X, panel1.Position.Y)
	}

	if panel1.Size.Width != 80 || panel1.Size.Height != 24 {
		t.Errorf("面板大小未重置，期望: 80x24, 实际: %dx%d",
			panel1.Size.Width, panel1.Size.Height)
	}

	if panel1.ZIndex != 0 {
		t.Errorf("面板层级未重置，期望: 0, 实际: %d", panel1.ZIndex)
	}

	// 验证布局模式已重置
	expectedMode := LayoutVertical // 2个面板应该是垂直布局
	if uiManager.layout.mode != expectedMode {
		t.Errorf("布局模式未正确重置，期望: %d, 实际: %d",
			expectedMode, uiManager.layout.mode)
	}
}

// ====== Step 4: 高级功能测试 ======

func TestToggleLayoutMode(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建面板
	uiManager.CreatePanel("panel1", "Panel 1")
	uiManager.CreatePanel("panel2", "Panel 2")

	initialMode := uiManager.layout.mode
	t.Logf("初始布局模式: %d", int(initialMode))

	// 测试切换布局模式
	uiManager.ToggleLayoutMode()
	firstToggleMode := uiManager.layout.mode
	t.Logf("第一次切换后布局模式: %d", int(firstToggleMode))

	if firstToggleMode == initialMode {
		t.Errorf("布局模式应该已切换，初始: %d, 当前: %d", int(initialMode), int(firstToggleMode))
	}

	// 再次切换
	uiManager.ToggleLayoutMode()
	secondToggleMode := uiManager.layout.mode
	t.Logf("第二次切换后布局模式: %d", int(secondToggleMode))

	if secondToggleMode == firstToggleMode {
		t.Errorf("布局模式应该再次切换，第一次: %d, 第二次: %d", int(firstToggleMode), int(secondToggleMode))
	}

	// 验证布局模式循环
	expectedSequence := []LayoutMode{LayoutVertical, LayoutHorizontal, LayoutGrid}
	if len(expectedSequence) > 0 && initialMode == expectedSequence[0] {
		if firstToggleMode != expectedSequence[1] {
			t.Errorf("第一次切换布局模式错误，期望: %d, 实际: %d", int(expectedSequence[1]), int(firstToggleMode))
		}
		if len(expectedSequence) > 2 && secondToggleMode != expectedSequence[2] {
			t.Errorf("第二次切换布局模式错误，期望: %d, 实际: %d", int(expectedSequence[2]), int(secondToggleMode))
		}
	}
}

func TestTogglePanelProperties(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建面板并设为活动面板
	uiManager.CreateResizablePanel("test_panel", "Test Panel", true, true)
	uiManager.activePane = "test_panel"

	panel := uiManager.panels["test_panel"]

	// 测试切换可调整大小属性
	initialResizable := panel.Resizable
	uiManager.TogglePanelResizable()
	if panel.Resizable == initialResizable {
		t.Error("面板可调整大小属性应该已切换")
	}

	// 测试切换可拖拽属性
	initialDraggable := panel.Draggable
	uiManager.TogglePanelDraggable()
	if panel.Draggable == initialDraggable {
		t.Error("面板可拖拽属性应该已切换")
	}

	// 测试在没有活动面板时切换
	uiManager.activePane = ""
	uiManager.TogglePanelResizable() // 应该不会panic
	uiManager.TogglePanelDraggable() // 应该不会panic
}

func TestLayoutSaveAndLoad(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 创建面板
	uiManager.CreateResizablePanel("panel1", "Panel 1", true, true)
	uiManager.CreateResizablePanel("panel2", "Panel 2", true, true)

	// 保存当前布局
	err = uiManager.SaveCurrentLayout("test_layout")
	if err != nil {
		t.Fatalf("保存当前布局失败: %v", err)
	}

	// 保存自动保存布局（LoadLastLayout会加载这个）
	err = uiManager.SaveCurrentLayout("auto_save")
	if err != nil {
		t.Fatalf("保存自动保存布局失败: %v", err)
	}

	// 修改布局
	uiManager.ToggleLayoutMode()
	originalMode := uiManager.layout.mode

	// 加载布局
	err = uiManager.LoadLastLayout()
	if err != nil {
		t.Fatalf("加载布局失败: %v", err)
	}

	// 重置布局
	err = uiManager.ResetAllLayouts()
	if err != nil {
		t.Fatalf("重置布局失败: %v", err)
	}

	// 验证布局已重置
	if uiManager.layout.mode == originalMode {
		t.Error("布局应该已重置")
	}
}

// ====== Step 4: 验证方法测试 ======

func TestValidateSize(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	layoutMgr := uiManager.newLayoutManager()

	// 创建带约束的面板
	panel := &Panel{
		MinSize: Size{Width: 20, Height: 5},
		MaxSize: Size{Width: 200, Height: 50},
		Constraints: &Constraints{
			AspectRatio: 2.0, // 宽高比 2:1
		},
	}

	// 测试有效尺寸
	validSize := Size{Width: 40, Height: 20}
	err = layoutMgr.validateSize(panel, validSize)
	if err != nil {
		t.Errorf("有效尺寸验证失败: %v", err)
	}

	// 测试小于最小尺寸
	tooSmall := Size{Width: 10, Height: 3}
	err = layoutMgr.validateSize(panel, tooSmall)
	if err == nil {
		t.Error("小于最小尺寸应该验证失败")
	}

	// 测试大于最大尺寸
	tooBig := Size{Width: 300, Height: 60}
	err = layoutMgr.validateSize(panel, tooBig)
	if err == nil {
		t.Error("大于最大尺寸应该验证失败")
	}

	// 测试宽高比不符合
	wrongRatio := Size{Width: 40, Height: 40} // 1:1 比例
	err = layoutMgr.validateSize(panel, wrongRatio)
	if err == nil {
		t.Error("错误宽高比应该验证失败")
	}
}

func TestValidatePosition(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	layoutMgr := uiManager.newLayoutManager()

	// 创建带约束的面板
	panel := &Panel{
		Constraints: &Constraints{
			MarginLeft: 5,
			MarginTop:  3,
		},
	}

	// 测试有效位置
	validPos := Position{X: 10, Y: 5}
	err = layoutMgr.validatePosition(panel, validPos)
	if err != nil {
		t.Errorf("有效位置验证失败: %v", err)
	}

	// 测试负数位置
	negativePos := Position{X: -5, Y: 10}
	err = layoutMgr.validatePosition(panel, negativePos)
	if err == nil {
		t.Error("负数位置应该验证失败")
	}

	// 测试超出边距约束
	tooClose := Position{X: 2, Y: 1}
	err = layoutMgr.validatePosition(panel, tooClose)
	if err == nil {
		t.Error("超出边距约束应该验证失败")
	}
}

// ====== Step 4: 错误处理测试 ======

func TestLayoutManagerErrorHandling(t *testing.T) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		t.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	layoutMgr := uiManager.GetLayoutManager()

	// 测试空布局名称
	err = layoutMgr.SaveLayout("")
	if err == nil {
		t.Error("空布局名称应该保存失败")
	}

	// 测试加载不存在的布局
	err = layoutMgr.LoadLayout("nonexistent")
	if err == nil {
		t.Error("加载不存在的布局应该失败")
	}

	// 测试删除不存在的布局
	err = layoutMgr.DeleteLayout("nonexistent")
	if err == nil {
		t.Error("删除不存在的布局应该失败")
	}

	// 测试操作不存在的面板
	err = layoutMgr.ResizePanel("nonexistent", Size{Width: 100, Height: 50})
	if err == nil {
		t.Error("调整不存在面板大小应该失败")
	}

	err = layoutMgr.MovePanel("nonexistent", Position{X: 10, Y: 10})
	if err == nil {
		t.Error("移动不存在面板应该失败")
	}

	err = layoutMgr.SetPanelConstraints("nonexistent", Constraints{})
	if err == nil {
		t.Error("设置不存在面板约束应该失败")
	}
}

// ====== Step 4: 性能测试 ======

func BenchmarkLayoutManager(b *testing.B) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		b.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	layoutMgr := uiManager.GetLayoutManager()

	// 创建面板
	for i := 0; i < 10; i++ {
		panelID := fmt.Sprintf("panel_%d", i)
		uiManager.CreateResizablePanel(panelID, fmt.Sprintf("Panel %d", i), true, true)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// 测试各种布局操作的性能
		layoutMgr.GetCurrentLayout()
		layoutMgr.SaveLayout("bench_layout")
		layoutMgr.LoadLayout("bench_layout")
		layoutMgr.ResetLayout()
	}
}

func BenchmarkDragOperations(b *testing.B) {
	config := DefaultUIConfig
	uiManager, err := NewUIManager(config)
	if err != nil {
		b.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	layoutMgr := uiManager.GetLayoutManager()
	uiManager.CreateResizablePanel("test_panel", "Test Panel", true, true)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		startPos := Position{X: i % 100, Y: i % 50}
		layoutMgr.StartDrag("test_panel", startPos, DragMove)
		layoutMgr.UpdateDrag(Position{X: startPos.X + 5, Y: startPos.Y + 3})
		layoutMgr.EndDrag()
	}
}
