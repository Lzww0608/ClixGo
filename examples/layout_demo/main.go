package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/terminal/ui"
	"github.com/gdamore/tcell/v2"
)

func main() {
	fmt.Println("ClixGo 布局管理增强功能演示")
	fmt.Println("=============================")

	// 创建UI管理器
	config := ui.DefaultUIConfig
	config.MouseEnabled = true

	uiManager, err := ui.NewUIManager(config)
	if err != nil {
		log.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 启用高级布局功能
	if err := uiManager.EnableAdvancedLayoutFeatures(); err != nil {
		log.Fatalf("启用高级布局功能失败: %v", err)
	}

	// 创建可调整大小的面板
	uiManager.CreateResizablePanel("panel1", "面板 1 (可调整)", true, true)
	uiManager.CreateResizablePanel("panel2", "面板 2 (可调整)", true, true)
	uiManager.CreateResizablePanel("panel3", "面板 3 (固定)", false, false)

	// 添加内容到面板
	uiManager.WriteToPanel("panel1", "这是面板1的内容\n")
	uiManager.WriteToPanel("panel1", "支持调整大小和拖拽\n")
	uiManager.WriteToPanel("panel1", "按F4切换布局模式\n")

	uiManager.WriteToPanel("panel2", "这是面板2的内容\n")
	uiManager.WriteToPanel("panel2", "也支持调整大小和拖拽\n")
	uiManager.WriteToPanel("panel2", "按F5保存当前布局\n")

	uiManager.WriteToPanel("panel3", "这是面板3的内容\n")
	uiManager.WriteToPanel("panel3", "这个面板是固定的\n")
	uiManager.WriteToPanel("panel3", "不能调整大小或拖拽\n")

	// 获取布局管理器
	layoutMgr := uiManager.GetLayoutManager()

	// 演示面板操作
	go func() {
		time.Sleep(1 * time.Second)

		// 调整面板1的大小
		newSize := ui.Size{Width: 100, Height: 30}
		if err := layoutMgr.ResizePanel("panel1", newSize); err != nil {
			uiManager.WriteToPanel("panel1", fmt.Sprintf("调整大小失败: %v\n", err))
		} else {
			uiManager.WriteToPanel("panel1", "面板大小已调整为100x30\n")
		}

		time.Sleep(1 * time.Second)

		// 移动面板2
		newPos := ui.Position{X: 10, Y: 5}
		if err := layoutMgr.MovePanel("panel2", newPos); err != nil {
			uiManager.WriteToPanel("panel2", fmt.Sprintf("移动失败: %v\n", err))
		} else {
			uiManager.WriteToPanel("panel2", "面板已移动到位置(10,5)\n")
		}

		time.Sleep(1 * time.Second)

		// 设置面板约束
		constraints := ui.Constraints{
			FixedWidth:  true,
			FixedHeight: false,
			AspectRatio: 2.0,
		}
		if err := layoutMgr.SetPanelConstraints("panel1", constraints); err != nil {
			uiManager.WriteToPanel("panel1", fmt.Sprintf("设置约束失败: %v\n", err))
		} else {
			uiManager.WriteToPanel("panel1", "已设置宽度固定，宽高比2:1约束\n")
		}
	}()

	// 设置自定义输入处理
	uiManager.SetCustomInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF4:
			uiManager.ToggleLayoutMode()
			return nil
		case tcell.KeyF5:
			uiManager.SaveCurrentLayout("demo_layout")
			uiManager.WriteToPanel("panel1", "布局已保存为demo_layout\n")
			return nil
		case tcell.KeyF6:
			if err := uiManager.LoadLastLayout(); err != nil {
				uiManager.WriteToPanel("panel1", fmt.Sprintf("加载布局失败: %v\n", err))
			} else {
				uiManager.WriteToPanel("panel1", "布局已加载\n")
			}
			return nil
		case tcell.KeyF7:
			uiManager.ResetAllLayouts()
			uiManager.WriteToPanel("panel1", "布局已重置\n")
			return nil
		case tcell.KeyEscape:
			uiManager.Quit()
			return nil
		}
		return event
	})

	// 显示帮助信息
	helpText := `
布局管理增强功能演示

快捷键:
  F4  - 切换布局模式 (单面板 → 垂直 → 水平 → 网格 → 自定义 → 浮动)
  F5  - 保存当前布局
  F6  - 加载保存的布局
  F7  - 重置所有布局
  Esc - 退出程序

功能特性:
- 可调整大小的面板
- 面板拖拽功能
- 布局约束设置
- 布局保存/加载
- 多种布局模式
- 自定义和浮动布局
`

	// 创建帮助面板
	uiManager.CreatePanel("help", "帮助信息")
	uiManager.WriteToPanel("help", helpText)

	fmt.Println("启动演示程序...")
	fmt.Println("按Esc退出")

	// 启动UI
	if err := uiManager.Start(); err != nil {
		log.Fatalf("启动UI失败: %v", err)
	}
}
