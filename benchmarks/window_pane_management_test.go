/*
* @Author: Lzww0608
* @Date: 2025-6-17 18:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-17 18:00:00
* @Description: Phase 1.3 任务1.5和1.6 - 窗口和面板管理功能测试
 */

package benchmarks

import (
	"fmt"
	"testing"

	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

// TestWindowManagement 测试窗口管理功能
func TestWindowManagement(t *testing.T) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	sessionManager := terminal.NewSessionManager(config)
	defer sessionManager.Shutdown()

	// 创建会话
	session, err := sessionManager.CreateSession("test-window-management")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}

	// 测试创建多个窗口
	_, err = sessionManager.CreateWindow(session.ID, "window-1")
	if err != nil {
		t.Fatalf("创建窗口1失败: %v", err)
	}

	_, err = sessionManager.CreateWindow(session.ID, "window-2")
	if err != nil {
		t.Fatalf("创建窗口2失败: %v", err)
	}

	_, err = sessionManager.CreateWindow(session.ID, "window-3")
	if err != nil {
		t.Fatalf("创建窗口3失败: %v", err)
	}

	// 测试窗口导航
	t.Run("NextWindow", func(t *testing.T) {
		// 当前应该在window3 (index 2)
		err := sessionManager.NextWindow(session.ID)
		if err != nil {
			t.Errorf("NextWindow失败: %v", err)
		}

		// 应该循环到window1 (index 0)
		updatedSession, _ := sessionManager.GetSession(session.ID)
		if updatedSession.ActiveWindow != 0 {
			t.Errorf("期望ActiveWindow为0，实际为%d", updatedSession.ActiveWindow)
		}
	})

	t.Run("PreviousWindow", func(t *testing.T) {
		// 当前在window1 (index 0)
		err := sessionManager.PreviousWindow(session.ID)
		if err != nil {
			t.Errorf("PreviousWindow失败: %v", err)
		}

		// 应该循环到最后一个窗口 (index 3，因为会话默认创建了一个窗口)
		updatedSession, _ := sessionManager.GetSession(session.ID)
		expectedIndex := len(updatedSession.Windows) - 1
		if updatedSession.ActiveWindow != expectedIndex {
			t.Errorf("期望ActiveWindow为%d，实际为%d", expectedIndex, updatedSession.ActiveWindow)
		}
	})

	t.Run("SwitchWindow", func(t *testing.T) {
		err := sessionManager.SwitchWindow(session.ID, 1)
		if err != nil {
			t.Errorf("SwitchWindow失败: %v", err)
		}

		updatedSession, _ := sessionManager.GetSession(session.ID)
		if updatedSession.ActiveWindow != 1 {
			t.Errorf("期望ActiveWindow为1，实际为%d", updatedSession.ActiveWindow)
		}
	})

	t.Logf("窗口管理测试完成 - 窗口数量: %d", len(session.Windows))
}

// TestPaneManagement 测试面板管理功能
func TestPaneManagement(t *testing.T) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	sessionManager := terminal.NewSessionManager(config)
	defer sessionManager.Shutdown()

	// 创建会话和窗口
	session, err := sessionManager.CreateSession("test-pane-management")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}

	_, err = sessionManager.CreateWindow(session.ID, "test-window")
	if err != nil {
		t.Fatalf("创建窗口失败: %v", err)
	}

	// 测试面板分割
	t.Run("SplitPane", func(t *testing.T) {
		// 垂直分割
		pane1, err := sessionManager.SplitPane(session.ID, 0, "vertical")
		if err != nil {
			t.Errorf("垂直分割失败: %v", err)
		}

		// 水平分割
		pane2, err := sessionManager.SplitPane(session.ID, 0, "horizontal")
		if err != nil {
			t.Errorf("水平分割失败: %v", err)
		}

		updatedSession, _ := sessionManager.GetSession(session.ID)
		windowPanes := updatedSession.Windows[0].Panes
		expectedPanes := 3 // 原始面板 + 2个新面板
		if len(windowPanes) != expectedPanes {
			t.Errorf("期望面板数量为%d，实际为%d", expectedPanes, len(windowPanes))
		}

		t.Logf("创建的面板: pane1=%s, pane2=%s", pane1.ID, pane2.ID)
	})

	t.Run("PaneNavigation", func(t *testing.T) {
		// 测试下一个面板
		err := sessionManager.NextPane(session.ID, 0)
		if err != nil {
			t.Errorf("NextPane失败: %v", err)
		}

		// 测试上一个面板
		err = sessionManager.PreviousPane(session.ID, 0)
		if err != nil {
			t.Errorf("PreviousPane失败: %v", err)
		}

		// 测试直接切换面板
		err = sessionManager.SwitchPane(session.ID, 0, 1)
		if err != nil {
			t.Errorf("SwitchPane失败: %v", err)
		}

		updatedSession, _ := sessionManager.GetSession(session.ID)
		if updatedSession.Windows[0].ActivePane != 1 {
			t.Errorf("期望ActivePane为1，实际为%d", updatedSession.Windows[0].ActivePane)
		}
	})

	t.Run("ResizePane", func(t *testing.T) {
		// 测试面板大小调整
		err := sessionManager.ResizePane(session.ID, 0, 0, "right", 10)
		if err != nil {
			t.Errorf("ResizePane失败: %v", err)
		}

		err = sessionManager.ResizePane(session.ID, 0, 0, "down", 5)
		if err != nil {
			t.Errorf("ResizePane失败: %v", err)
		}

		t.Log("面板大小调整测试完成")
	})

	updatedSession, _ := sessionManager.GetSession(session.ID)
	t.Logf("面板管理测试完成 - 面板数量: %d", len(updatedSession.Windows[0].Panes))
}

// TestLayoutManagement 测试布局管理功能
func TestLayoutManagement(t *testing.T) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	sessionManager := terminal.NewSessionManager(config)
	defer sessionManager.Shutdown()

	// 创建会话和窗口
	session, err := sessionManager.CreateSession("test-layout-management")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}

	_, err = sessionManager.CreateWindow(session.ID, "test-window")
	if err != nil {
		t.Fatalf("创建窗口失败: %v", err)
	}

	// 创建多个面板用于测试布局
	for i := 0; i < 3; i++ {
		_, err := sessionManager.SplitPane(session.ID, 0, "vertical")
		if err != nil {
			t.Fatalf("创建面板%d失败: %v", i, err)
		}
	}

	// 测试不同布局
	layouts := []terminal.Layout{
		terminal.LayoutEven,
		terminal.LayoutMainVertical,
		terminal.LayoutMainHorizontal,
		terminal.LayoutTiled,
	}

	for _, layout := range layouts {
		t.Run(string(layout), func(t *testing.T) {
			err := sessionManager.SetLayout(session.ID, 0, layout)
			if err != nil {
				t.Errorf("设置布局%s失败: %v", layout, err)
			}

			updatedSession, _ := sessionManager.GetSession(session.ID)
			if updatedSession.Windows[0].Layout != layout {
				t.Errorf("期望布局为%s，实际为%s", layout, updatedSession.Windows[0].Layout)
			}

			// 验证面板位置和大小是否合理
			panes := updatedSession.Windows[0].Panes
			for i, pane := range panes {
				if pane.Width <= 0 || pane.Height <= 0 {
					t.Errorf("面板%d的大小无效: Width=%d, Height=%d", i, pane.Width, pane.Height)
				}
				if pane.X < 0 || pane.Y < 0 {
					t.Errorf("面板%d的位置无效: X=%d, Y=%d", i, pane.X, pane.Y)
				}
			}

			t.Logf("布局%s测试完成 - 面板数量: %d", layout, len(panes))
		})
	}
}

// BenchmarkWindowOperations 窗口操作性能基准测试
func BenchmarkWindowOperations(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	sessionManager := terminal.NewSessionManager(config)
	defer sessionManager.Shutdown()

	// 创建会话
	session, err := sessionManager.CreateSession("benchmark-windows")
	if err != nil {
		b.Fatalf("创建会话失败: %v", err)
	}

	// 创建多个窗口
	for i := 0; i < 10; i++ {
		_, err := sessionManager.CreateWindow(session.ID, fmt.Sprintf("window-%d", i))
		if err != nil {
			b.Fatalf("创建窗口失败: %v", err)
		}
	}

	b.ResetTimer()

	b.Run("NextWindow", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := sessionManager.NextWindow(session.ID)
			if err != nil {
				b.Errorf("NextWindow失败: %v", err)
			}
		}
	})

	b.Run("PreviousWindow", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := sessionManager.PreviousWindow(session.ID)
			if err != nil {
				b.Errorf("PreviousWindow失败: %v", err)
			}
		}
	})

	b.Run("SwitchWindow", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			windowIndex := i % 10
			err := sessionManager.SwitchWindow(session.ID, windowIndex)
			if err != nil {
				b.Errorf("SwitchWindow失败: %v", err)
			}
		}
	})
}

// BenchmarkPaneOperations 面板操作性能基准测试
func BenchmarkPaneOperations(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	sessionManager := terminal.NewSessionManager(config)
	defer sessionManager.Shutdown()

	// 创建会话和窗口
	session, err := sessionManager.CreateSession("benchmark-panes")
	if err != nil {
		b.Fatalf("创建会话失败: %v", err)
	}

	_, err = sessionManager.CreateWindow(session.ID, "test-window")
	if err != nil {
		b.Fatalf("创建窗口失败: %v", err)
	}

	// 创建多个面板
	for i := 0; i < 8; i++ {
		direction := "vertical"
		if i%2 == 1 {
			direction = "horizontal"
		}
		_, err := sessionManager.SplitPane(session.ID, 0, direction)
		if err != nil {
			b.Fatalf("创建面板失败: %v", err)
		}
	}

	b.ResetTimer()

	b.Run("NextPane", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := sessionManager.NextPane(session.ID, 0)
			if err != nil {
				b.Errorf("NextPane失败: %v", err)
			}
		}
	})

	b.Run("ResizePane", func(b *testing.B) {
		directions := []string{"up", "down", "left", "right"}
		for i := 0; i < b.N; i++ {
			direction := directions[i%4]
			amount := (i % 10) + 1
			err := sessionManager.ResizePane(session.ID, 0, 0, direction, amount)
			if err != nil {
				b.Errorf("ResizePane失败: %v", err)
			}
		}
	})

	b.Run("SetLayout", func(b *testing.B) {
		layouts := []terminal.Layout{
			terminal.LayoutEven,
			terminal.LayoutMainVertical,
			terminal.LayoutMainHorizontal,
			terminal.LayoutTiled,
		}
		for i := 0; i < b.N; i++ {
			layout := layouts[i%4]
			err := sessionManager.SetLayout(session.ID, 0, layout)
			if err != nil {
				b.Errorf("SetLayout失败: %v", err)
			}
		}
	})
}
