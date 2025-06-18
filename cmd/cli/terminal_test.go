/*
* @Author: Lzww0608
* @Date: 2025-6-16 12:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-16 12:00:00
* @Description: 终端CLI功能基础测试 - 验证PTY功能、会话管理、进程管理和数据I/O
 */

package cli

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/terminal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestSessionManager 设置测试用的SessionManager
func setupTestSessionManager(t *testing.T) *terminal.SessionManager {
	// 初始化logger
	err := logger.InitLogger()
	require.NoError(t, err, "Failed to initialize logger")

	config := terminal.DefaultConfig
	sessionManager := terminal.NewSessionManager(config)
	require.NotNil(t, sessionManager, "SessionManager should not be nil")

	return sessionManager
}

// TestSessionManager_CreateSession 测试会话创建功能
func TestSessionManager_CreateSession(t *testing.T) {
	sessionManager := setupTestSessionManager(t)
	defer sessionManager.Shutdown()

	tests := []struct {
		name        string
		sessionName string
		expectError bool
	}{
		{
			name:        "Create session with custom name",
			sessionName: "test-session",
			expectError: false,
		},
		{
			name:        "Create session with empty name",
			sessionName: "",
			expectError: false,
		},
		{
			name:        "Create session with special characters",
			sessionName: "dev-env_2024",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime := time.Now()
			session, err := sessionManager.CreateSession(tt.sessionName)
			createDuration := time.Since(startTime)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, session)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, session)
				assert.NotEmpty(t, session.ID)
				assert.NotEmpty(t, session.Name)
				assert.Equal(t, terminal.SessionActive, session.Status)
				assert.True(t, len(session.Windows) > 0, "Session should have at least one window")

				// 验证性能 - 会话创建应该在100ms内完成
				assert.Less(t, createDuration, 100*time.Millisecond, "Session creation should be fast")

				t.Logf("Session created: ID=%s, Name=%s, Duration=%v",
					session.ID[:8], session.Name, createDuration)
			}
		})
	}
}

// TestSessionManager_ListSessions 测试会话列表功能
func TestSessionManager_ListSessions(t *testing.T) {
	sessionManager := setupTestSessionManager(t)
	defer sessionManager.Shutdown()

	// 初始状态应该没有会话
	sessions := sessionManager.ListSessions()
	assert.Equal(t, 0, len(sessions), "Should start with no sessions")

	// 创建多个会话
	sessionNames := []string{"session1", "session2", "session3"}
	createdSessions := make([]*terminal.Session, 0, len(sessionNames))

	for _, name := range sessionNames {
		session, err := sessionManager.CreateSession(name)
		require.NoError(t, err)
		require.NotNil(t, session)
		createdSessions = append(createdSessions, session)
	}

	// 验证会话列表
	sessions = sessionManager.ListSessions()
	assert.Equal(t, len(sessionNames), len(sessions), "Should have created sessions")

	// 验证每个会话都存在
	sessionMap := make(map[string]*terminal.Session)
	for _, session := range sessions {
		sessionMap[session.Name] = session
	}

	for _, expectedName := range sessionNames {
		session, exists := sessionMap[expectedName]
		assert.True(t, exists, "Session %s should exist", expectedName)
		assert.Equal(t, expectedName, session.Name)
		assert.Equal(t, terminal.SessionActive, session.Status)
	}
}

// TestSessionManager_AttachDetachSession 测试会话附加和分离
func TestSessionManager_AttachDetachSession(t *testing.T) {
	sessionManager := setupTestSessionManager(t)
	defer sessionManager.Shutdown()

	// 创建测试会话
	session, err := sessionManager.CreateSession("attach-test")
	require.NoError(t, err)
	require.NotNil(t, session)

	// 测试附加会话
	err = sessionManager.AttachSession(session.ID)
	assert.NoError(t, err, "Should be able to attach to session")

	// 测试分离会话
	err = sessionManager.DetachSession(session.ID)
	assert.NoError(t, err, "Should be able to detach from session")

	// 测试附加不存在的会话
	err = sessionManager.AttachSession("non-existent-id")
	assert.Error(t, err, "Should fail to attach to non-existent session")
}

// TestSessionManager_KillSession 测试会话终止
func TestSessionManager_KillSession(t *testing.T) {
	sessionManager := setupTestSessionManager(t)
	defer sessionManager.Shutdown()

	// 创建测试会话
	session, err := sessionManager.CreateSession("kill-test")
	require.NoError(t, err)
	require.NotNil(t, session)

	// 验证会话存在
	sessions := sessionManager.ListSessions()
	assert.Equal(t, 1, len(sessions))

	// 终止会话
	err = sessionManager.KillSession(session.ID)
	assert.NoError(t, err, "Should be able to kill session")

	// 验证会话被移除
	sessions = sessionManager.ListSessions()
	assert.Equal(t, 0, len(sessions), "Session should be removed after kill")

	// 测试终止不存在的会话
	err = sessionManager.KillSession("non-existent-id")
	assert.Error(t, err, "Should fail to kill non-existent session")
}

// TestWindowManagement 测试窗口管理功能
func TestWindowManagement(t *testing.T) {
	sessionManager := setupTestSessionManager(t)
	defer sessionManager.Shutdown()

	// 创建测试会话
	session, err := sessionManager.CreateSession("window-test")
	require.NoError(t, err)
	require.NotNil(t, session)

	// 验证默认窗口
	assert.True(t, len(session.Windows) > 0, "Session should have default window")

	// 创建新窗口
	window, err := sessionManager.CreateWindow(session.ID, "test-window")
	assert.NoError(t, err, "Should be able to create window")
	assert.NotNil(t, window)
	assert.Equal(t, "test-window", window.Name)
	assert.NotEmpty(t, window.ID)

	// 验证窗口被添加到会话
	updatedSession, err := sessionManager.GetSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(updatedSession.Windows), "Session should have 2 windows")

	// 测试窗口切换
	err = sessionManager.SwitchWindow(session.ID, 1)
	assert.NoError(t, err, "Should be able to switch window")

	// 测试关闭窗口
	err = sessionManager.CloseWindow(session.ID, 1)
	assert.NoError(t, err, "Should be able to close window")

	// 验证窗口被移除
	updatedSession, err = sessionManager.GetSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(updatedSession.Windows), "Session should have 1 window after close")
}

// TestPaneManagement 测试面板管理功能
func TestPaneManagement(t *testing.T) {
	sessionManager := setupTestSessionManager(t)
	defer sessionManager.Shutdown()

	// 创建测试会话
	session, err := sessionManager.CreateSession("pane-test")
	require.NoError(t, err)
	require.NotNil(t, session)

	// 获取默认窗口
	require.True(t, len(session.Windows) > 0)
	window := session.Windows[0]
	require.True(t, len(window.Panes) > 0, "Window should have default pane")

	// 测试面板分割
	pane, err := sessionManager.SplitPane(session.ID, 0, "horizontal")
	assert.NoError(t, err, "Should be able to split pane horizontally")
	assert.NotNil(t, pane)
	assert.NotEmpty(t, pane.ID)

	// 验证面板被添加
	updatedSession, err := sessionManager.GetSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(updatedSession.Windows[0].Panes), "Window should have 2 panes")

	// 测试垂直分割
	pane2, err := sessionManager.SplitPane(session.ID, 0, "vertical")
	assert.NoError(t, err, "Should be able to split pane vertically")
	assert.NotNil(t, pane2)

	// 验证面板数量
	updatedSession, err = sessionManager.GetSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, len(updatedSession.Windows[0].Panes), "Window should have 3 panes")

	// 测试面板切换
	err = sessionManager.SwitchPane(session.ID, 0, 1)
	assert.NoError(t, err, "Should be able to switch pane")

	// 测试面板大小调整
	err = sessionManager.ResizePane(session.ID, 0, 1, "up", 5)
	assert.NoError(t, err, "Should be able to resize pane")
}

// TestPerformanceStats 测试性能统计功能
func TestPerformanceStats(t *testing.T) {
	sessionManager := setupTestSessionManager(t)
	defer sessionManager.Shutdown()

	// 获取初始统计
	initialStats := sessionManager.GetPerformanceStats()
	assert.NotNil(t, initialStats)

	// 创建几个会话来生成统计数据
	for i := 0; i < 3; i++ {
		session, err := sessionManager.CreateSession("")
		require.NoError(t, err)
		require.NotNil(t, session)
	}

	// 获取更新后的统计
	updatedStats := sessionManager.GetPerformanceStats()
	assert.NotNil(t, updatedStats)

	// 验证统计数据更新
	assert.Equal(t, int64(3), updatedStats.CreatedSessions, "Should have created 3 sessions")
	assert.Equal(t, int64(3), updatedStats.ActiveSessions, "Should have 3 active sessions")
	assert.True(t, updatedStats.AvgCreateTime > 0, "Average create time should be positive")

	t.Logf("Performance Stats: Created=%d, Active=%d, AvgCreateTime=%v",
		updatedStats.CreatedSessions, updatedStats.ActiveSessions, updatedStats.AvgCreateTime)
}

// TestSessionPersistence 测试会话持久化功能
func TestSessionPersistence(t *testing.T) {
	sessionManager := setupTestSessionManager(t)
	defer sessionManager.Shutdown()

	// 创建测试会话
	session, err := sessionManager.CreateSession("persistence-test")
	require.NoError(t, err)
	require.NotNil(t, session)

	// 保存会话
	err = sessionManager.SaveSessionByName(session.Name)
	assert.NoError(t, err, "Should be able to save session")

	// 列出保存的会话
	savedSessions, err := sessionManager.ListSavedSessions()
	assert.NoError(t, err, "Should be able to list saved sessions")
	assert.Contains(t, savedSessions, session.Name, "Saved session should be in list")

	// 加载会话
	loadedSession, err := sessionManager.LoadSessionByName(session.Name)
	assert.NoError(t, err, "Should be able to load session")
	assert.NotNil(t, loadedSession)
	assert.Equal(t, session.Name, loadedSession.Name)

	// 删除保存的会话
	err = sessionManager.DeleteSavedSession(session.Name)
	assert.NoError(t, err, "Should be able to delete saved session")

	// 验证会话被删除
	savedSessions, err = sessionManager.ListSavedSessions()
	assert.NoError(t, err)
	assert.NotContains(t, savedSessions, session.Name, "Deleted session should not be in list")
}

// TestConcurrentOperations 测试并发操作
func TestConcurrentOperations(t *testing.T) {
	sessionManager := setupTestSessionManager(t)
	defer sessionManager.Shutdown()

	const numGoroutines = 5
	const sessionsPerGoroutine = 3

	// 使用通道收集结果
	resultChan := make(chan error, numGoroutines*sessionsPerGoroutine)

	// 启动多个goroutine并发创建会话
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < sessionsPerGoroutine; j++ {
				sessionName := fmt.Sprintf("concurrent-session-%d-%d", goroutineID, j)
				_, err := sessionManager.CreateSession(sessionName)
				resultChan <- err
			}
		}(i)
	}

	// 收集所有结果
	successCount := 0
	for i := 0; i < numGoroutines*sessionsPerGoroutine; i++ {
		err := <-resultChan
		if err == nil {
			successCount++
		} else {
			t.Logf("Concurrent operation failed: %v", err)
		}
	}

	// 验证大部分操作成功
	expectedSuccess := numGoroutines * sessionsPerGoroutine
	assert.Equal(t, expectedSuccess, successCount, "All concurrent operations should succeed")

	// 验证会话列表
	sessions := sessionManager.ListSessions()
	assert.Equal(t, expectedSuccess, len(sessions), "Should have all created sessions")
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	sessionManager := setupTestSessionManager(t)
	defer sessionManager.Shutdown()

	// 测试无效的会话ID
	_, err := sessionManager.GetSession("invalid-session-id")
	assert.Error(t, err, "Should fail with invalid session ID")

	// 测试无效的窗口索引
	session, err := sessionManager.CreateSession("error-test")
	require.NoError(t, err)

	err = sessionManager.SwitchWindow(session.ID, 999)
	assert.Error(t, err, "Should fail with invalid window index")

	// 测试无效的面板索引
	err = sessionManager.SwitchPane(session.ID, 0, 999)
	assert.Error(t, err, "Should fail with invalid pane index")

	// 测试重复会话名称
	_, err = sessionManager.CreateSession("error-test")
	assert.Error(t, err, "Should fail with duplicate session name")
}

// TestPerformanceBenchmark 性能基准测试
func TestPerformanceBenchmark(t *testing.T) {
	sessionManager := setupTestSessionManager(t)
	defer sessionManager.Shutdown()

	// 测试会话创建性能
	startTime := time.Now()
	const numSessions = 10

	for i := 0; i < numSessions; i++ {
		session, err := sessionManager.CreateSession("")
		require.NoError(t, err)
		require.NotNil(t, session)
	}

	totalDuration := time.Since(startTime)
	avgDuration := totalDuration / numSessions

	// 验证性能目标：每个会话创建应该在50ms内完成
	assert.Less(t, avgDuration, 50*time.Millisecond,
		"Average session creation should be under 50ms, got %v", avgDuration)

	t.Logf("Performance Benchmark: %d sessions created in %v (avg: %v per session)",
		numSessions, totalDuration, avgDuration)

	// 测试内存使用
	stats := sessionManager.GetPerformanceStats()
	t.Logf("Memory usage: %.2f MB", stats.MemoryUsageMB)
}

// BenchmarkSessionCreation 会话创建基准测试
func BenchmarkSessionCreation(b *testing.B) {
	sessionManager := setupTestSessionManager(&testing.T{})
	defer sessionManager.Shutdown()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sessionName := fmt.Sprintf("bench-session-%d", i)
			session, err := sessionManager.CreateSession(sessionName)
			if err != nil {
				b.Errorf("Failed to create session: %v", err)
			}
			if session == nil {
				b.Error("Session is nil")
			}
			i++
		}
	})
}

// BenchmarkWindowOperations 窗口操作基准测试
func BenchmarkWindowOperations(b *testing.B) {
	sessionManager := setupTestSessionManager(&testing.T{})
	defer sessionManager.Shutdown()

	// 创建测试会话
	session, err := sessionManager.CreateSession("bench-window-test")
	if err != nil {
		b.Fatalf("Failed to create session: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 创建窗口
		_, err := sessionManager.CreateWindow(session.ID, "")
		if err != nil {
			b.Errorf("Failed to create window: %v", err)
		}

		// 切换窗口
		windowIndex := len(session.Windows) - 1
		err = sessionManager.SwitchWindow(session.ID, windowIndex)
		if err != nil {
			b.Errorf("Failed to switch window: %v", err)
		}

		// 关闭窗口（保留至少一个窗口）
		if len(session.Windows) > 1 {
			err = sessionManager.CloseWindow(session.ID, windowIndex)
			if err != nil {
				b.Errorf("Failed to close window: %v", err)
			}
		}

		// 更新session引用
		session, err = sessionManager.GetSession(session.ID)
		if err != nil {
			b.Errorf("Failed to get session: %v", err)
		}
	}
}

// setupTestSessionManagerContext 为上下文测试设置SessionManager
func setupTestSessionManagerContext(ctx context.Context, t *testing.T) *terminal.SessionManager {
	sessionManager := setupTestSessionManager(t)

	// 监听上下文取消
	go func() {
		<-ctx.Done()
		sessionManager.Shutdown()
	}()

	return sessionManager
}

// TestContextCancellation 测试上下文取消
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sessionManager := setupTestSessionManagerContext(ctx, t)

	// 创建会话
	session, err := sessionManager.CreateSession("context-test")
	require.NoError(t, err)
	require.NotNil(t, session)

	// 取消上下文
	cancel()

	// 等待一小段时间让清理完成
	time.Sleep(100 * time.Millisecond)

	// 此时操作应该失败或者处理上下文取消
	_, err = sessionManager.CreateSession("after-cancel")
	// 根据实现，这可能成功或失败，主要是测试不会崩溃
	t.Logf("Operation after context cancel: %v", err)
}
