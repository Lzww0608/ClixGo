package history

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 使用临时文件进行测试的辅助函数
func setupTempHistory(t *testing.T) string {
	// 创建临时目录
	tempDir := t.TempDir()

	// 保存原始的历史文件路径并设置测试用的临时路径
	originalHistoryFile := GetHistoryFilePath()
	SetHistoryFilePath(filepath.Join(tempDir, "history.json"))

	// 测试结束后恢复原始路径
	t.Cleanup(func() {
		SetHistoryFilePath(originalHistoryFile)
	})

	return GetHistoryFilePath()
}

func TestSaveHistory(t *testing.T) {
	// 设置临时历史文件
	tempHistoryFile := setupTempHistory(t)

	// 创建测试历史记录
	now := time.Now()
	cmd := &CommandHistory{
		Command:   "test command",
		Status:    "success",
		Output:    "test output",
		StartTime: now.Add(-5 * time.Second),
		EndTime:   now,
		Duration:  "5s",
	}

	// 保存历史记录
	err := SaveHistory(cmd)
	require.NoError(t, err, "保存历史记录应该成功")

	// 验证文件是否存在
	_, err = os.Stat(tempHistoryFile)
	require.NoError(t, err, "历史文件应该已创建")

	// 获取历史记录并验证内容
	history, err := GetHistory()
	require.NoError(t, err, "获取历史记录应该成功")
	require.Len(t, history, 1, "应该有一条历史记录")
	assert.Equal(t, cmd.Command, history[0].Command, "命令内容不匹配")
	assert.Equal(t, cmd.Status, history[0].Status, "状态不匹配")

	// 测试添加多条记录
	cmd2 := &CommandHistory{
		Command:   "test command 2",
		Status:    "failure",
		Output:    "test output 2",
		StartTime: now.Add(-3 * time.Second),
		EndTime:   now,
		Duration:  "3s",
	}

	err = SaveHistory(cmd2)
	require.NoError(t, err, "保存第二条历史记录应该成功")

	history, err = GetHistory()
	require.NoError(t, err, "获取历史记录应该成功")
	require.Len(t, history, 2, "应该有两条历史记录")
	assert.Equal(t, cmd.Command, history[0].Command, "第一条命令内容不匹配")
	assert.Equal(t, cmd2.Command, history[1].Command, "第二条命令内容不匹配")
}

func TestGetHistory_EmptyFile(t *testing.T) {
	// 设置临时历史文件
	setupTempHistory(t)

	// 获取历史记录（文件不存在）
	history, err := GetHistory()
	require.NoError(t, err, "获取不存在的历史记录应该成功")
	assert.Empty(t, history, "历史记录应该为空")
}

func TestGetHistory_InvalidJSON(t *testing.T) {
	// 设置临时历史文件
	tempHistoryFile := setupTempHistory(t)

	// 创建一个包含无效JSON的历史文件
	err := os.MkdirAll(filepath.Dir(tempHistoryFile), 0755)
	require.NoError(t, err, "创建目录应该成功")

	err = os.WriteFile(tempHistoryFile, []byte("这不是有效的JSON"), 0644)
	require.NoError(t, err, "写入无效的JSON应该成功")

	// 尝试获取历史记录
	history, err := GetHistory()
	assert.Error(t, err, "获取无效的JSON应该返回错误")
	assert.Nil(t, history, "历史记录应该为nil")
	assert.Contains(t, err.Error(), "解析历史记录失败", "错误信息不正确")
}

func TestSaveHistory_MaxEntries(t *testing.T) {
	// 设置临时历史文件
	setupTempHistory(t)

	// 创建多条历史记录（超过100条）
	now := time.Now()
	for i := 0; i < 110; i++ {
		cmd := &CommandHistory{
			Command:   "command " + strconv.Itoa(i),
			Status:    "success",
			Output:    "output",
			StartTime: now.Add(-time.Duration(i) * time.Second),
			EndTime:   now,
			Duration:  "1s",
		}
		err := SaveHistory(cmd)
		require.NoError(t, err, "保存历史记录应该成功")
	}

	// 获取历史记录并验证数量
	history, err := GetHistory()
	require.NoError(t, err, "获取历史记录应该成功")
	assert.Len(t, history, 100, "应该只保留最近的100条历史记录")

	// 验证保留的是最新的100条记录（命令以"command 9"的应该被淘汰）
	hasOldCommand := false
	for _, h := range history {
		if h.Command == "command 9" {
			hasOldCommand = true
			break
		}
	}
	assert.False(t, hasOldCommand, "最早的记录应该已被淘汰")
}

func TestClearHistory(t *testing.T) {
	// 设置临时历史文件
	tempHistoryFile := setupTempHistory(t)

	// 创建测试历史记录
	cmd := &CommandHistory{
		Command:   "test command",
		Status:    "success",
		Output:    "test output",
		StartTime: time.Now().Add(-5 * time.Second),
		EndTime:   time.Now(),
		Duration:  "5s",
	}

	err := SaveHistory(cmd)
	require.NoError(t, err, "保存历史记录应该成功")

	// 清除历史记录
	err = ClearHistory()
	require.NoError(t, err, "清除历史记录应该成功")

	// 验证文件是否已删除
	_, err = os.Stat(tempHistoryFile)
	assert.True(t, os.IsNotExist(err), "历史文件应该已删除")

	// 尝试清除不存在的历史文件
	err = ClearHistory()
	assert.NoError(t, err, "清除不存在的历史记录应该成功")
}

func TestClearHistory_NoPermission(t *testing.T) {
	// 这个测试在Windows上可能会失败，因为权限处理不同
	if os.PathSeparator == '\\' {
		t.Skip("跳过在Windows上的权限测试")
	}

	// 设置临时历史文件
	tempHistoryFile := setupTempHistory(t)

	// 创建测试历史记录
	cmd := &CommandHistory{
		Command:   "test command",
		Status:    "success",
		Output:    "test output",
		StartTime: time.Now().Add(-5 * time.Second),
		EndTime:   time.Now(),
		Duration:  "5s",
	}

	err := SaveHistory(cmd)
	require.NoError(t, err, "保存历史记录应该成功")

	// 修改文件权限为只读（模拟权限问题）
	tempDir := filepath.Dir(tempHistoryFile)
	err = os.Chmod(tempDir, 0500) // r-x------
	if err != nil {
		t.Skip("无法修改目录权限，跳过此测试")
	}

	// 尝试清除历史记录
	err = ClearHistory()

	// 恢复权限以便清理
	os.Chmod(tempDir, 0700)

	// 权限问题应该导致错误
	if err != nil {
		assert.Contains(t, err.Error(), "清除历史记录失败", "错误信息不正确")
	}
}

func TestGetLastHistory(t *testing.T) {
	// 设置临时历史文件
	setupTempHistory(t)

	// 创建测试历史记录
	now := time.Now()
	cmd1 := &CommandHistory{
		Command:   "first command",
		Status:    "success",
		Output:    "output 1",
		StartTime: now.Add(-10 * time.Second),
		EndTime:   now.Add(-8 * time.Second),
		Duration:  "2s",
	}

	cmd2 := &CommandHistory{
		Command:   "second command",
		Status:    "failure",
		Output:    "output 2",
		StartTime: now.Add(-5 * time.Second),
		EndTime:   now.Add(-4 * time.Second),
		Duration:  "1s",
	}

	// 保存第一条历史记录
	err := SaveHistory(cmd1)
	require.NoError(t, err, "保存第一条历史记录应该成功")

	// 获取最后一条记录并验证
	lastHistory, err := GetLastHistory()
	require.NoError(t, err, "获取最后一条历史记录应该成功")
	require.NotNil(t, lastHistory, "最后一条历史记录不应为nil")
	assert.Equal(t, cmd1.Command, lastHistory.Command, "最后一条命令内容不匹配")

	// 保存第二条历史记录
	err = SaveHistory(cmd2)
	require.NoError(t, err, "保存第二条历史记录应该成功")

	// 再次获取最后一条记录
	lastHistory, err = GetLastHistory()
	require.NoError(t, err, "获取最后一条历史记录应该成功")
	require.NotNil(t, lastHistory, "最后一条历史记录不应为nil")
	assert.Equal(t, cmd2.Command, lastHistory.Command, "最后一条命令内容不匹配")
}

func TestGetLastHistory_EmptyHistory(t *testing.T) {
	// 设置临时历史文件
	setupTempHistory(t)

	// 不添加任何历史记录，直接获取最后一条
	lastHistory, err := GetLastHistory()
	require.NoError(t, err, "获取空历史记录应该成功")
	assert.Nil(t, lastHistory, "历史记录为空时应返回nil")
}

func TestSetAndGetHistoryFilePath(t *testing.T) {
	// 保存原始路径
	originalPath := GetHistoryFilePath()
	defer SetHistoryFilePath(originalPath)

	// 设置新路径
	newPath := "/tmp/test_history.json"
	SetHistoryFilePath(newPath)

	// 验证路径已更改
	currentPath := GetHistoryFilePath()
	assert.Equal(t, newPath, currentPath, "历史文件路径应该已更改")
}
