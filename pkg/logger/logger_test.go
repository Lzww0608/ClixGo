package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 测试初始化logger
func TestInitLogger(t *testing.T) {
	// 保存原始日志文件路径
	originalLogPath := GetLogPath()
	defer SetLogPath(originalLogPath)
	defer os.Remove(originalLogPath)
	defer Close()

	// 初始化logger
	err := InitLogger()
	require.NoError(t, err, "初始化日志应该成功")

	// 验证logger是否已初始化
	assert.NotNil(t, Log, "Log应该被初始化")

	// 测试日志记录
	Info("测试信息日志")
	Error("测试错误日志", zap.String("key", "value"))
	Debug("测试调试日志")
	Warn("测试警告日志")

	// 验证日志文件是否已创建
	_, err = os.Stat(originalLogPath)
	assert.NoError(t, err, "日志文件应该已创建")
}

// 测试使用未初始化的logger
func TestUninitializedLogger(t *testing.T) {
	// 备份当前logger状态
	oldLog := Log
	oldInitialized := initialized

	// 恢复测试后的状态
	defer func() {
		Log = oldLog
		initialized = oldInitialized
	}()

	// 重置logger
	Close()
	Log = nil
	initialized = false

	// 这些应该会panic，我们需要恢复
	defer func() {
		if r := recover(); r != nil {
			t.Log("预期的panic发生:", r)
		} else {
			t.Error("应该发生panic但没有")
		}
	}()

	// 尝试使用未初始化的logger
	Info("这应该导致panic")
}

// 测试日志级别是否正确工作
func TestLogLevels(t *testing.T) {
	// 创建临时目录用于测试
	tempDir := t.TempDir()
	tempLogFile := filepath.Join(tempDir, "test.log")

	// 保存和恢复原始设置
	originalLogPath := GetLogPath()
	defer func() {
		SetLogPath(originalLogPath)
		Close()
	}()

	// 设置测试日志路径
	SetLogPath(tempLogFile)
	err := InitLogger()
	require.NoError(t, err, "初始化日志应该成功")

	// 记录各种级别的日志
	Debug("这是调试日志")
	Info("这是信息日志")
	Warn("这是警告日志")
	Error("这是错误日志")

	// 关闭日志以确保内容被刷新到文件
	Close()

	// 读取日志文件内容
	logContent, err := os.ReadFile(tempLogFile)
	require.NoError(t, err, "应该能读取日志文件")

	logStr := string(logContent)

	// 验证日志内容
	assert.Contains(t, logStr, "这是调试日志", "Debug日志应该被记录")
	assert.Contains(t, logStr, "这是信息日志", "Info日志应该被记录")
	assert.Contains(t, logStr, "这是警告日志", "Warn日志应该被记录")
	assert.Contains(t, logStr, "这是错误日志", "Error日志应该被记录")
}

// 测试日志文件路径设置
func TestLogPathSetting(t *testing.T) {
	// 保存原始日志路径
	originalLogPath := GetLogPath()
	defer SetLogPath(originalLogPath)
	defer Close()

	// 设置新的日志路径
	testLogPath := "test_gocli.log"
	SetLogPath(testLogPath)
	defer os.Remove(testLogPath)

	// 验证路径已更改
	assert.Equal(t, testLogPath, GetLogPath(), "日志路径应该已更改")

	// 初始化logger并写入日志
	err := InitLogger()
	require.NoError(t, err, "初始化日志应该成功")
	Info("测试日志路径设置")

	// 验证日志文件是否在新位置创建
	_, err = os.Stat(testLogPath)
	assert.NoError(t, err, "日志文件应该在新位置创建")
}

// 测试重新初始化logger
func TestReinitLogger(t *testing.T) {
	// 保存原始日志路径
	originalLogPath := GetLogPath()
	defer SetLogPath(originalLogPath)
	defer Close()

	// 第一次初始化
	err := InitLogger()
	require.NoError(t, err, "第一次初始化日志应该成功")
	Info("第一次初始化")

	// 更改日志路径
	newLogPath := "reinit_test.log"
	SetLogPath(newLogPath)
	defer os.Remove(newLogPath)

	// 重新初始化
	err = InitLogger()
	require.NoError(t, err, "重新初始化日志应该成功")
	Info("重新初始化后")

	// 验证新日志文件是否创建
	_, err = os.Stat(newLogPath)
	assert.NoError(t, err, "重新初始化后应该创建新日志文件")
}

// 测试关闭日志
func TestCloseLogger(t *testing.T) {
	// 初始化logger
	err := InitLogger()
	require.NoError(t, err, "初始化日志应该成功")

	// 写入一些日志
	Info("关闭前的日志")

	// 关闭logger
	err = Close()
	require.NoError(t, err, "关闭日志应该成功")

	// 检查状态
	assert.False(t, initialized, "关闭后initialized应该为false")
	assert.Nil(t, logFile, "关闭后logFile应该为nil")

	// 再次关闭也不应该报错
	err = Close()
	assert.NoError(t, err, "重复关闭不应该报错")
}
