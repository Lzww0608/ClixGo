/*
* @Author: Lzww0608
* @Date: 2025-6-1 20:51:10
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-1 21:21:34
* @Description: 终端性能基准测试
 */

package benchmarks

import (
	"fmt"
	"testing"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/terminal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TestMain 在测试开始前初始化logger（如果未定义）
func init() {
	// 确保logger已初始化（避免与terminal_bench_test.go中的TestMain冲突）
	logger.InitLogger()
}

// UI渲染性能基准测试
func BenchmarkUIRendering(b *testing.B) {
	config := ui.UIConfig{
		MouseEnabled: false,
		RefreshRate:  time.Millisecond * 16,
		StatusBarStyle: ui.StatusBarStyle{
			Format:    "%s | %s | %s",
			ShowTime:  true,
			ShowStats: true,
		},
	}

	manager, err := ui.NewUIManager(config)
	if err != nil {
		b.Fatalf("创建UI管理器失败: %v", err)
	}
	defer manager.Stop()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 模拟UI更新操作
		panelID := fmt.Sprintf("render_panel_%d", i)
		panel := manager.CreatePanel(panelID, "Test Panel")
		if panel != nil {
			manager.WriteToPanel(panelID, "test content")
		}
	}
}

// 面板操作性能基准测试
func BenchmarkPanelOperations(b *testing.B) {
	config := ui.UIConfig{
		MouseEnabled: false,
		RefreshRate:  time.Millisecond * 16,
		StatusBarStyle: ui.StatusBarStyle{
			Format:    "%s | %s | %s",
			ShowTime:  true,
			ShowStats: true,
		},
	}

	manager, err := ui.NewUIManager(config)
	if err != nil {
		b.Fatalf("创建UI管理器失败: %v", err)
	}
	defer manager.Stop()

	b.Run("面板创建", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			panelID := fmt.Sprintf("panel_%d", i)
			panel := manager.CreatePanel(panelID, "Test Panel")
			if panel == nil {
				b.Fatalf("创建面板失败")
			}
		}
	})

	b.Run("面板切换", func(b *testing.B) {
		// 预创建一些面板
		panelCount := 10
		for i := 0; i < panelCount; i++ {
			panelID := fmt.Sprintf("switch_panel_%d", i)
			manager.CreatePanel(panelID, "Switch Panel")
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// 使用NextPanel进行切换
			manager.NextPanel()
		}
	})

	b.Run("面板写入", func(b *testing.B) {
		// 创建测试面板
		testPanel := manager.CreatePanel("write_test_panel", "Write Test")
		if testPanel == nil {
			b.Fatalf("创建测试面板失败")
		}

		testContent := "benchmark test content\n"

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			err := manager.WriteToPanel("write_test_panel", testContent)
			if err != nil {
				b.Fatalf("写入面板失败: %v", err)
			}
		}
	})
}

// 事件处理性能基准测试
func BenchmarkEventHandling(b *testing.B) {
	screen := tcell.NewSimulationScreen("UTF-8")
	err := screen.Init()
	if err != nil {
		b.Fatalf("初始化屏幕失败: %v", err)
	}
	defer screen.Fini()

	app := tview.NewApplication()
	app.SetScreen(screen)

	config := ui.UIConfig{
		MouseEnabled: true,
		RefreshRate:  time.Millisecond * 16,
	}

	uiManager, err := ui.NewUIManager(config)
	if err != nil {
		b.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 模拟不同类型的事件
	keyEvents := []tcell.Key{
		tcell.KeyCtrlC,
		tcell.KeyCtrlN,
		tcell.KeyCtrlP,
		tcell.KeyTab,
		tcell.KeyEnter,
		tcell.KeyEscape,
	}

	b.Run("键盘事件", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := keyEvents[i%len(keyEvents)]
			event := tcell.NewEventKey(key, 0, tcell.ModNone)
			// 模拟键盘事件处理
			_ = event
		}
	})

	b.Run("鼠标事件", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			x := i % 80
			y := i % 24
			event := tcell.NewEventMouse(x, y, tcell.Button1, tcell.ModNone)
			// 模拟鼠标事件处理
			_ = event
		}
	})
}

// 大量数据渲染性能基准测试
func BenchmarkLargeDataRendering(b *testing.B) {
	screen := tcell.NewSimulationScreen("UTF-8")
	err := screen.Init()
	if err != nil {
		b.Fatalf("初始化屏幕失败: %v", err)
	}
	defer screen.Fini()

	screen.SetSize(120, 40) // 设置较大的屏幕尺寸

	app := tview.NewApplication()
	app.SetScreen(screen)

	// 测试不同数据量的渲染性能
	dataSizes := []int{100, 1000, 10000, 100000}

	for _, size := range dataSizes {
		b.Run(fmt.Sprintf("数据量_%d", size), func(b *testing.B) {
			// 生成测试数据
			data := make([]string, size)
			for i := 0; i < size; i++ {
				data[i] = fmt.Sprintf("Line %d: This is test data for performance benchmarking", i)
			}

			// 创建文本视图
			textView := tview.NewTextView()
			textView.SetDynamicColors(true)
			textView.SetScrollable(true)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				textView.Clear()
				for _, line := range data {
					fmt.Fprintln(textView, line)
				}
				textView.Draw(screen)
			}
		})
	}
}

// UI状态更新性能基准测试
func BenchmarkUIStateUpdates(b *testing.B) {
	screen := tcell.NewSimulationScreen("UTF-8")
	err := screen.Init()
	if err != nil {
		b.Fatalf("初始化屏幕失败: %v", err)
	}
	defer screen.Fini()

	app := tview.NewApplication()
	app.SetScreen(screen)

	config := ui.UIConfig{
		MouseEnabled: true,
		RefreshRate:  time.Millisecond * 16,
	}

	uiManager, err := ui.NewUIManager(config)
	if err != nil {
		b.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	b.Run("状态栏更新", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// 模拟状态栏更新
			status := fmt.Sprintf("状态 %d - CPU: %d%% Memory: %dMB", i, i%100, (i%1000)+100)
			_ = status // 模拟更新操作
		}
	})

	b.Run("面板内容更新", func(b *testing.B) {
		panel := uiManager.CreatePanel("update_test", "Update Test")
		if panel == nil {
			b.Fatalf("创建测试面板失败")
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			content := fmt.Sprintf("Update %d: %s", i, time.Now().Format("15:04:05.000"))
			uiManager.WriteToPanel(panel.ID, content)
		}
	})

	b.Run("多面板同步更新", func(b *testing.B) {
		panelCount := 5
		panels := make([]string, panelCount)

		for i := 0; i < panelCount; i++ {
			panelID := fmt.Sprintf("sync_panel_%d", i)
			panels[i] = panelID
			uiManager.CreatePanel(panelID, fmt.Sprintf("Sync Panel %d", i))
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for j, panelID := range panels {
				content := fmt.Sprintf("Panel %d Update %d", j, i)
				uiManager.WriteToPanel(panelID, content)
			}
		}
	})
}

// 颜色和样式处理性能基准测试
func BenchmarkUIStyleProcessing(b *testing.B) {
	screen := tcell.NewSimulationScreen("UTF-8")
	err := screen.Init()
	if err != nil {
		b.Fatalf("初始化屏幕失败: %v", err)
	}
	defer screen.Fini()

	b.Run("颜色解析", func(b *testing.B) {
		colorStrings := []string{
			"[red]Red Text[white]",
			"[green]Green Text[white]",
			"[blue]Blue Text[white]",
			"[yellow]Yellow Text[white]",
			"[#ff00ff]Custom Color[white]",
		}

		textView := tview.NewTextView()
		textView.SetDynamicColors(true)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			colorStr := colorStrings[i%len(colorStrings)]
			fmt.Fprint(textView, colorStr)
		}
	})

	b.Run("样式应用", func(b *testing.B) {
		textView := tview.NewTextView()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			textView.SetTextColor(tcell.ColorWhite)
			textView.SetBackgroundColor(tcell.ColorBlack)
			textView.SetBorderColor(tcell.ColorBlue)
			textView.SetTitleColor(tcell.ColorYellow)
		}
	})
}

// FPS测试基准（模拟60FPS渲染）
func BenchmarkFPSRendering(b *testing.B) {
	screen := tcell.NewSimulationScreen("UTF-8")
	err := screen.Init()
	if err != nil {
		b.Fatalf("初始化屏幕失败: %v", err)
	}
	defer screen.Fini()

	app := tview.NewApplication()
	app.SetScreen(screen)

	config := ui.UIConfig{
		MouseEnabled: true,
		RefreshRate:  time.Millisecond * 16,
	}

	uiManager, err := ui.NewUIManager(config)
	if err != nil {
		b.Fatalf("创建UI管理器失败: %v", err)
	}
	defer uiManager.Stop()

	// 目标是60FPS，即每帧约16.67ms
	targetFrameTime := time.Millisecond * 16

	b.Run("60FPS模拟", func(b *testing.B) {
		frameCount := 0
		startTime := time.Now()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			frameStart := time.Now()

			// 执行一帧的渲染操作（模拟）
			panelID := fmt.Sprintf("fps_panel_%d", i%10)
			panel := uiManager.CreatePanel(panelID, "FPS Test")
			if panel != nil {
				content := fmt.Sprintf("Frame %d", i)
				uiManager.WriteToPanel(panelID, content)
			}
			frameCount++

			// 模拟帧时间控制
			frameTime := time.Since(frameStart)
			if frameTime < targetFrameTime {
				time.Sleep(targetFrameTime - frameTime)
			}
		}

		totalTime := time.Since(startTime)
		actualFPS := float64(frameCount) / totalTime.Seconds()

		b.Logf("实际FPS: %.2f, 目标FPS: 60", actualFPS)
	})
}

// 内存分配优化基准测试
func BenchmarkUIMemoryOptimization(b *testing.B) {
	b.Run("字符串池化", func(b *testing.B) {
		stringPool := make([]string, 0, 1000)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// 模拟字符串重用
			if i < len(stringPool) {
				_ = stringPool[i]
			} else {
				str := fmt.Sprintf("Pooled string %d", i)
				if len(stringPool) < cap(stringPool) {
					stringPool = append(stringPool, str)
				}
				_ = str
			}
		}
	})

	b.Run("缓冲区重用", func(b *testing.B) {
		bufferPool := make([][]byte, 0, 100)
		bufferSize := 1024

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var buffer []byte

			// 尝试从池中获取缓冲区
			if len(bufferPool) > 0 {
				buffer = bufferPool[len(bufferPool)-1]
				bufferPool = bufferPool[:len(bufferPool)-1]
				// 重置缓冲区
				buffer = buffer[:0]
			} else {
				buffer = make([]byte, 0, bufferSize)
			}

			// 使用缓冲区
			for j := 0; j < 10; j++ {
				buffer = append(buffer, byte(j))
			}

			// 归还到池中
			if len(bufferPool) < cap(bufferPool) {
				bufferPool = append(bufferPool, buffer)
			}
		}
	})
}
