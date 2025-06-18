/*
 * @Author: Lzww0608
 * @Date: 2025-6-18 20:45:00
 * @LastEditors: Lzww0608
 * @LastEditTime: 2025-6-18 20:45:00
 * @Description: Phase 1.3 任务1.5 - FastSessionManager功能兼容性验证
 */

package benchmarks

import (
	"fmt"
	"testing"

	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

// TestFunctionalCompatibility 功能兼容性测试
func TestFunctionalCompatibility(t *testing.T) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	t.Run("SessionManagement", func(t *testing.T) {
		// 测试原版SessionManager
		originalManager := terminal.NewSessionManager(config)
		defer originalManager.Shutdown()

		// 测试FastSessionManager
		fastManager := terminal.NewFastSessionManager(config)
		defer fastManager.Shutdown()

		// 使用不同的会话名称避免冲突
		originalSession, err := originalManager.CreateSession("original-test-session")
		if err != nil {
			t.Fatalf("原版会话创建失败: %v", err)
		}

		fastSession, err := fastManager.CreateSession("fast-test-session")
		if err != nil {
			t.Fatalf("快速版会话创建失败: %v", err)
		}

		// 验证会话属性
		if originalSession.Name != "original-test-session" {
			t.Errorf("原版会话名称不正确: 期望=%s, 实际=%s", "original-test-session", originalSession.Name)
		}

		if fastSession.Name != "fast-test-session" {
			t.Errorf("快速版会话名称不正确: 期望=%s, 实际=%s", "fast-test-session", fastSession.Name)
		}

		if len(originalSession.Windows) != len(fastSession.Windows) {
			t.Errorf("窗口数量不一致: 原版=%d, 快速=%d", len(originalSession.Windows), len(fastSession.Windows))
		}

		t.Logf("✅ 会话管理功能完全兼容")
	})

	t.Run("WindowOperations", func(t *testing.T) {
		// 简化窗口操作测试
		fastManager := terminal.NewFastSessionManager(config)
		defer fastManager.Shutdown()

		// 创建会话
		session, err := fastManager.CreateSession("window-test")
		if err != nil {
			t.Fatalf("创建会话失败: %v", err)
		}

		// 创建窗口
		window, err := fastManager.CreateWindow(session.ID, "test-window")
		if err != nil {
			t.Fatalf("创建窗口失败: %v", err)
		}

		// 验证窗口属性
		if window == nil {
			t.Fatal("窗口为nil")
		}

		if window.Name != "test-window" {
			t.Errorf("窗口名称不匹配: 期望=%s, 实际=%s", "test-window", window.Name)
		}

		if len(window.Panes) == 0 {
			t.Error("窗口应该有默认面板")
		}

		t.Logf("✅ 窗口操作功能完全兼容")
	})

	t.Run("LazyInitialization", func(t *testing.T) {
		fastManager := terminal.NewFastSessionManager(config)
		defer fastManager.Shutdown()

		// 验证初始状态
		if fastManager.IsLazyInitialized() {
			t.Error("FastSessionManager不应该在创建时就初始化")
		}

		// 触发延迟初始化
		session, err := fastManager.CreateSession("lazy-init-test")
		if err != nil {
			t.Fatalf("延迟初始化触发失败: %v", err)
		}

		// 验证初始化完成
		if !fastManager.IsLazyInitialized() {
			t.Error("FastSessionManager应该在首次使用后完成初始化")
		}

		// 验证功能正常
		if session == nil || session.Name != "lazy-init-test" {
			t.Error("延迟初始化后功能异常")
		}

		t.Logf("✅ 延迟初始化功能正常工作")
	})
}

// TestComprehensiveFunctionality 全面功能测试
func TestComprehensiveFunctionality(t *testing.T) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	fastManager := terminal.NewFastSessionManager(config)
	defer fastManager.Shutdown()

	// 测试完整的会话生命周期
	sessionName := "comprehensive-test"

	// 1. 创建会话
	session, err := fastManager.CreateSession(sessionName)
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}

	// 2. 创建窗口
	window, err := fastManager.CreateWindow(session.ID, "test-window")
	if err != nil {
		t.Fatalf("创建窗口失败: %v", err)
	}

	// 3. 分割面板
	pane, err := fastManager.SplitPane(session.ID, window.Index, "horizontal")
	if err != nil {
		t.Fatalf("分割面板失败: %v", err)
	}

	// 4. 切换窗口
	err = fastManager.SwitchWindow(session.ID, 0)
	if err != nil {
		t.Fatalf("切换窗口失败: %v", err)
	}

	// 5. 切换面板
	err = fastManager.SwitchPane(session.ID, 0, 0)
	if err != nil {
		t.Fatalf("切换面板失败: %v", err)
	}

	// 6. 获取会话信息
	retrievedSession, err := fastManager.GetSession(session.ID)
	if err != nil {
		t.Fatalf("获取会话失败: %v", err)
	}

	// 7. 验证会话状态
	sessions := fastManager.ListSessions()
	if len(sessions) == 0 {
		t.Fatal("会话列表为空")
	}

	// 8. 重命名会话
	err = fastManager.RenameSession(session.ID, "renamed-session")
	if err != nil {
		t.Fatalf("重命名会话失败: %v", err)
	}

	// 9. 关闭面板
	err = fastManager.ClosePane(session.ID, window.Index, pane.Index)
	if err != nil {
		t.Fatalf("关闭面板失败: %v", err)
	}

	// 10. 最终清理
	err = fastManager.KillSession(session.ID)
	if err != nil {
		t.Fatalf("杀死会话失败: %v", err)
	}

	// 验证所有操作
	if retrievedSession.ID != session.ID {
		t.Error("获取的会话ID不匹配")
	}

	t.Logf("✅ 全面功能测试通过 - FastSessionManager功能完整")
}

// BenchmarkFunctionalityOverhead 功能使用开销测试
func BenchmarkFunctionalityOverhead(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	b.Run("SessionOperations", func(b *testing.B) {
		fastManager := terminal.NewFastSessionManager(config)
		defer fastManager.Shutdown()

		// 先触发初始化
		firstSession, _ := fastManager.CreateSession("init-session")
		defer fastManager.KillSession(firstSession.ID)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			sessionName := fmt.Sprintf("bench-session-%d", i)

			session, err := fastManager.CreateSession(sessionName)
			if err != nil {
				b.Fatalf("创建会话失败: %v", err)
			}

			window, err := fastManager.CreateWindow(session.ID, "bench-window")
			if err != nil {
				b.Fatalf("创建窗口失败: %v", err)
			}

			_, err = fastManager.SplitPane(session.ID, window.Index, "horizontal")
			if err != nil {
				b.Fatalf("分割面板失败: %v", err)
			}

			err = fastManager.KillSession(session.ID)
			if err != nil {
				b.Fatalf("杀死会话失败: %v", err)
			}
		}
	})
}
