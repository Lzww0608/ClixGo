/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 终端会话持久化功能的单元测试
 */

package terminal

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// 初始化日志系统
	logger.InitLogger()
	defer logger.Close()

	m.Run()
}

func TestNewPersistenceManager(t *testing.T) {
	// 添加延迟防止死锁
	time.Sleep(100 * time.Millisecond)

	// 创建临时目录
	tempDir := t.TempDir()

	config := &PersistenceConfig{
		DataDir:         tempDir,
		AutoSave:        true,
		SaveInterval:    time.Minute * 5,
		MaxSnapshots:    10,
		CompressData:    false,
		SaveBufferLines: 1000,
		SaveHistory:     true,
		SaveEnvironment: true,
	}

	pm, err := NewPersistenceManager(config)
	require.NoError(t, err)
	assert.NotNil(t, pm)
	assert.Equal(t, tempDir, pm.dataDir)
	assert.True(t, pm.autoSave)

	// 验证目录是否创建
	_, err = os.Stat(tempDir)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
}

func TestDefaultPersistenceConfig(t *testing.T) {
	time.Sleep(100 * time.Millisecond)

	config := DefaultPersistenceConfig()
	assert.NotNil(t, config)
	assert.True(t, config.AutoSave)
	assert.Equal(t, time.Minute*5, config.SaveInterval)
	assert.Equal(t, 10, config.MaxSnapshots)
	assert.Equal(t, 1000, config.SaveBufferLines)
	assert.True(t, config.SaveHistory)
	assert.True(t, config.SaveEnvironment)

	time.Sleep(100 * time.Millisecond)
}

func TestSaveAndLoadSession(t *testing.T) {
	time.Sleep(100 * time.Millisecond)

	// 创建临时目录
	tempDir := t.TempDir()

	config := &PersistenceConfig{
		DataDir:         tempDir,
		AutoSave:        false,
		SaveInterval:    time.Minute * 5,
		MaxSnapshots:    10,
		CompressData:    false,
		SaveBufferLines: 1000,
		SaveHistory:     true,
		SaveEnvironment: true,
	}

	pm, err := NewPersistenceManager(config)
	require.NoError(t, err)

	// 创建测试会话
	session := createTestSession(t)

	time.Sleep(100 * time.Millisecond)

	// 保存会话
	err = pm.SaveSession(session)
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// 加载会话
	snapshot, err := pm.LoadSession(session.Name)
	assert.NoError(t, err)
	assert.NotNil(t, snapshot)
	assert.Equal(t, session.ID, snapshot.ID)
	assert.Equal(t, session.Name, snapshot.Name)
	assert.Equal(t, len(session.Windows), len(snapshot.Windows))

	time.Sleep(100 * time.Millisecond)
}

func TestRestoreSession(t *testing.T) {
	time.Sleep(100 * time.Millisecond)

	// 创建临时目录
	tempDir := t.TempDir()

	config := &PersistenceConfig{
		DataDir:         tempDir,
		AutoSave:        false,
		SaveInterval:    time.Minute * 5,
		MaxSnapshots:    10,
		CompressData:    false,
		SaveBufferLines: 1000,
		SaveHistory:     true,
		SaveEnvironment: true,
	}

	pm, err := NewPersistenceManager(config)
	require.NoError(t, err)

	// 创建测试会话
	originalSession := createTestSession(t)

	time.Sleep(100 * time.Millisecond)

	// 保存会话
	err = pm.SaveSession(originalSession)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// 加载快照
	snapshot, err := pm.LoadSession(originalSession.Name)
	require.NoError(t, err)

	// 创建会话管理器
	sm := NewSessionManager(DefaultConfig)

	time.Sleep(100 * time.Millisecond)

	// 恢复会话
	restoredSession, err := pm.RestoreSession(snapshot, sm)
	assert.NoError(t, err)
	assert.NotNil(t, restoredSession)
	assert.Equal(t, originalSession.ID, restoredSession.ID)
	assert.Equal(t, originalSession.Name, restoredSession.Name)
	assert.Equal(t, len(originalSession.Windows), len(restoredSession.Windows))

	time.Sleep(100 * time.Millisecond)
}

func TestListSnapshots(t *testing.T) {
	time.Sleep(100 * time.Millisecond)

	// 创建临时目录
	tempDir := t.TempDir()

	config := &PersistenceConfig{
		DataDir:         tempDir,
		AutoSave:        false,
		SaveInterval:    time.Minute * 5,
		MaxSnapshots:    10,
		CompressData:    false,
		SaveBufferLines: 1000,
		SaveHistory:     true,
		SaveEnvironment: true,
	}

	pm, err := NewPersistenceManager(config)
	require.NoError(t, err)

	// 初始状态应该没有快照
	snapshots, err := pm.ListSnapshots()
	assert.NoError(t, err)
	assert.Empty(t, snapshots)

	time.Sleep(100 * time.Millisecond)

	// 创建并保存多个会话
	session1 := createTestSession(t)
	session1.Name = "test_session_1"

	session2 := createTestSession(t)
	session2.Name = "test_session_2"

	time.Sleep(100 * time.Millisecond)

	err = pm.SaveSession(session1)
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	err = pm.SaveSession(session2)
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// 列出快照
	snapshots, err = pm.ListSnapshots()
	assert.NoError(t, err)
	assert.Len(t, snapshots, 2)

	time.Sleep(100 * time.Millisecond)
}

func TestDeleteSnapshot(t *testing.T) {
	time.Sleep(100 * time.Millisecond)

	// 创建临时目录
	tempDir := t.TempDir()

	config := &PersistenceConfig{
		DataDir:         tempDir,
		AutoSave:        false,
		SaveInterval:    time.Minute * 5,
		MaxSnapshots:    10,
		CompressData:    false,
		SaveBufferLines: 1000,
		SaveHistory:     true,
		SaveEnvironment: true,
	}

	pm, err := NewPersistenceManager(config)
	require.NoError(t, err)

	// 创建并保存会话
	session := createTestSession(t)

	time.Sleep(100 * time.Millisecond)

	err = pm.SaveSession(session)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// 列出快照
	snapshots, err := pm.ListSnapshots()
	require.NoError(t, err)
	require.Len(t, snapshots, 1)

	// 删除快照
	err = pm.DeleteSnapshot(snapshots[0])
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// 验证快照已删除
	snapshots, err = pm.ListSnapshots()
	assert.NoError(t, err)
	assert.Empty(t, snapshots)

	time.Sleep(100 * time.Millisecond)
}

func TestGetSnapshotInfo(t *testing.T) {
	time.Sleep(100 * time.Millisecond)

	// 创建临时目录
	tempDir := t.TempDir()

	config := &PersistenceConfig{
		DataDir:         tempDir,
		AutoSave:        false,
		SaveInterval:    time.Minute * 5,
		MaxSnapshots:    10,
		CompressData:    false,
		SaveBufferLines: 1000,
		SaveHistory:     true,
		SaveEnvironment: true,
	}

	pm, err := NewPersistenceManager(config)
	require.NoError(t, err)

	// 创建并保存会话
	session := createTestSession(t)

	time.Sleep(100 * time.Millisecond)

	err = pm.SaveSession(session)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// 列出快照
	snapshots, err := pm.ListSnapshots()
	require.NoError(t, err)
	require.Len(t, snapshots, 1)

	// 获取快照信息
	snapshotInfo, err := pm.GetSnapshotInfo(snapshots[0])
	assert.NoError(t, err)
	assert.NotNil(t, snapshotInfo)
	assert.Equal(t, session.ID, snapshotInfo.ID)
	assert.Equal(t, session.Name, snapshotInfo.Name)

	time.Sleep(100 * time.Millisecond)
}

func TestSessionManagerPersistence(t *testing.T) {
	time.Sleep(100 * time.Millisecond)

	// 创建会话管理器
	sm := NewSessionManager(DefaultConfig)

	// 创建会话
	session, err := sm.CreateSession("test_persistence")
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// 创建窗口
	window, err := sm.CreateWindow(session.ID, "test_window")
	require.NoError(t, err)
	assert.NotNil(t, window)

	time.Sleep(100 * time.Millisecond)

	// 保存会话
	err = sm.SaveSession(session.ID, "")
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// 列出已保存的会话
	savedSessions, err := sm.ListSavedSessions()
	assert.NoError(t, err)
	assert.Contains(t, savedSessions, "test_persistence")

	time.Sleep(100 * time.Millisecond)

	// 删除当前会话
	err = sm.KillSession(session.ID)
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// 加载会话
	loadedSession, err := sm.LoadSessionByName("test_persistence")
	assert.NoError(t, err)
	assert.NotNil(t, loadedSession)
	assert.Equal(t, session.Name, loadedSession.Name)
	assert.Equal(t, 2, len(loadedSession.Windows))

	time.Sleep(100 * time.Millisecond)

	// 删除已保存的会话
	err = sm.DeleteSavedSession("test_persistence")
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// 验证会话已删除
	savedSessions, err = sm.ListSavedSessions()
	assert.NoError(t, err)
	assert.NotContains(t, savedSessions, "test_persistence")

	time.Sleep(100 * time.Millisecond)
}

func TestCleanupOldSnapshots(t *testing.T) {
	time.Sleep(100 * time.Millisecond)

	// 创建临时目录
	tempDir := t.TempDir()

	config := &PersistenceConfig{
		DataDir:         tempDir,
		AutoSave:        false,
		SaveInterval:    time.Minute * 5,
		MaxSnapshots:    3, // 设置较小的限制用于测试
		CompressData:    false,
		SaveBufferLines: 1000,
		SaveHistory:     true,
		SaveEnvironment: true,
	}

	pm, err := NewPersistenceManager(config)
	require.NoError(t, err)

	// 创建测试会话
	session := createTestSession(t)
	session.Name = "cleanup_test"

	time.Sleep(100 * time.Millisecond)

	// 保存多个快照
	for i := 0; i < 5; i++ {
		err = pm.SaveSession(session)
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond) // 确保文件时间戳不同
	}

	time.Sleep(500 * time.Millisecond) // 等待清理完成

	// 验证只保留了最新的快照
	snapshots, err := pm.ListSnapshots()
	assert.NoError(t, err)

	// 计算该会话的快照数量
	sessionSnapshots := 0
	for _, snapshot := range snapshots {
		if filepath.Base(snapshot)[:len("cleanup_test")] == "cleanup_test" {
			sessionSnapshots++
		}
	}

	// 应该只保留最多3个快照
	assert.LessOrEqual(t, sessionSnapshots, 3)

	time.Sleep(100 * time.Millisecond)
}

func TestExtractSessionNameFunctions(t *testing.T) {
	time.Sleep(100 * time.Millisecond)

	// 测试从路径提取会话名称
	testCases := []struct {
		input    string
		expected string
	}{
		{"test_session_20240101_120000.json", "test_session"},
		{"/path/to/test_session_20240101_120000.json", "test_session"},
		{"complex_session_name_20240101_120000.json", "complex_session_name"},
		{"simple.json", "simple"},
		{"no_extension", "no_extension"},
	}

	for _, tc := range testCases {
		result := extractSessionNameFromPath(tc.input)
		assert.Equal(t, tc.expected, result, "输入: %s", tc.input)

		result2 := extractSessionNameFromSnapshot(filepath.Base(tc.input))
		assert.Equal(t, tc.expected, result2, "快照输入: %s", tc.input)
	}

	time.Sleep(100 * time.Millisecond)
}

// createTestSession 创建测试会话
func createTestSession(t *testing.T) *Session {
	session := &Session{
		ID:           "test-session-id",
		Name:         "test_session",
		Status:       SessionActive,
		CreatedAt:    time.Now(),
		LastActive:   time.Now(),
		Windows:      make([]*Window, 0),
		ActiveWindow: 0,
	}

	// 创建测试窗口
	window := &Window{
		ID:         "test-window-id",
		Name:       "test_window",
		Index:      0,
		Panes:      make([]*Pane, 0),
		ActivePane: 0,
		Layout:     LayoutMainVertical,
		CreatedAt:  time.Now(),
	}

	// 创建测试面板
	pane := &Pane{
		ID:         "test-pane-id",
		Index:      0,
		X:          0,
		Y:          0,
		Width:      80,
		Height:     24,
		Command:    "bash",
		WorkingDir: "/tmp",
		ProcessID:  12345,
		Active:     true,
		CreatedAt:  time.Now(),
		LastOutput: time.Now(),
		Buffer: &Buffer{
			Lines:    [][]rune{[]rune("test line 1"), []rune("test line 2")},
			MaxLines: 1000,
			CursorX:  0,
			CursorY:  1,
		},
	}

	window.Panes = append(window.Panes, pane)
	session.Windows = append(session.Windows, window)

	return session
}
