package sync

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 创建临时测试目录
func setupTestDir(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "sync-test")
	require.NoError(t, err, "创建临时目录失败")

	// 返回清理函数
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// TestNewSyncManager 测试创建同步管理器
func TestNewSyncManager(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	sm := NewSyncManager(tmpDir)
	assert.NotNil(t, sm, "同步管理器不应为nil")
	assert.Equal(t, tmpDir, sm.syncDir, "syncDir应正确设置")
	assert.Empty(t, sm.operations, "初始operations应为空")
	assert.False(t, sm.offline, "初始offline状态应为false")
}

// TestOfflineMode 测试离线模式
func TestOfflineMode(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	sm := NewSyncManager(tmpDir)

	assert.False(t, sm.IsOffline(), "初始应为在线模式")

	sm.SetOffline(true)
	assert.True(t, sm.IsOffline(), "应成功设置为离线模式")

	sm.SetOffline(false)
	assert.False(t, sm.IsOffline(), "应成功重设为在线模式")
}

// TestCreateOperation 测试创建操作
func TestCreateOperation(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	sm := NewSyncManager(tmpDir)

	// 创建一个测试操作
	testData := map[string]string{"key": "value"}
	op := sm.CreateOperation(OperationCreate, "test-entity", testData)

	assert.NotNil(t, op, "创建的操作不应为nil")
	assert.NotEmpty(t, op.ID, "操作ID不应为空")
	assert.Equal(t, OperationCreate, op.Type, "操作类型应正确设置")
	assert.Equal(t, "test-entity", op.Entity, "操作实体应正确设置")
	assert.Equal(t, testData, op.Data, "操作数据应正确设置")
	assert.Equal(t, "pending", op.Status, "初始状态应为pending")
	assert.Empty(t, op.Error, "初始错误应为空")

	// 验证操作是否已添加到管理器
	assert.Len(t, sm.operations, 1, "操作应添加到管理器")

	// 验证操作文件是否已创建
	files, err := os.ReadDir(tmpDir)
	assert.NoError(t, err, "读取目录不应出错")
	assert.Len(t, files, 1, "应该创建一个操作文件")

	// 读取并验证文件内容
	fileName := filepath.Join(tmpDir, op.ID+".json")
	fileData, err := os.ReadFile(fileName)
	assert.NoError(t, err, "读取文件不应出错")

	var savedOp Operation
	err = json.Unmarshal(fileData, &savedOp)
	assert.NoError(t, err, "解析操作不应出错")
	assert.Equal(t, op.ID, savedOp.ID, "保存的操作ID应匹配")
	assert.Equal(t, op.Type, savedOp.Type, "保存的操作类型应匹配")
}

// TestExecuteOperation 测试执行操作
func TestExecuteOperation(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	sm := NewSyncManager(tmpDir)

	// 创建测试操作
	op := sm.CreateOperation(OperationCreate, "test-entity", nil)

	// 测试成功执行
	err := sm.ExecuteOperation(context.Background(), op, func(ctx context.Context, op *Operation) error {
		return nil // 成功
	})

	assert.NoError(t, err, "执行成功的操作不应返回错误")
	assert.Equal(t, "completed", op.Status, "成功执行的操作状态应为completed")
	assert.Empty(t, op.Error, "成功执行的操作不应有错误")

	// 创建另一个操作测试失败情况
	failOp := sm.CreateOperation(OperationUpdate, "test-entity", nil)

	// 测试执行失败
	err = sm.ExecuteOperation(context.Background(), failOp, func(ctx context.Context, op *Operation) error {
		return assert.AnError
	})

	assert.Error(t, err, "执行失败的操作应返回错误")
	assert.Equal(t, "failed", failOp.Status, "执行失败的操作状态应为failed")
	assert.Contains(t, failOp.Error, "assert.AnError", "错误信息应被记录")

	// 验证文件是否更新
	fileName := filepath.Join(tmpDir, failOp.ID+".json")
	fileData, err := os.ReadFile(fileName)
	assert.NoError(t, err, "读取文件不应出错")

	var savedOp Operation
	err = json.Unmarshal(fileData, &savedOp)
	assert.NoError(t, err, "解析操作不应出错")
	assert.Equal(t, "failed", savedOp.Status, "保存的操作状态应更新")
}

// TestGetPendingOperations 测试获取待处理操作
func TestGetPendingOperations(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	sm := NewSyncManager(tmpDir)

	// 创建各种状态的操作
	pendingOp1 := sm.CreateOperation(OperationCreate, "entity1", nil)
	pendingOp2 := sm.CreateOperation(OperationUpdate, "entity2", nil)

	// 将一个操作更改为completed状态
	completedOp := sm.CreateOperation(OperationDelete, "entity3", nil)
	err := sm.ExecuteOperation(context.Background(), completedOp, func(ctx context.Context, op *Operation) error {
		return nil
	})
	assert.NoError(t, err)

	// 将一个操作更改为failed状态
	failedOp := sm.CreateOperation(OperationUpdate, "entity4", nil)
	err = sm.ExecuteOperation(context.Background(), failedOp, func(ctx context.Context, op *Operation) error {
		return assert.AnError
	})
	assert.Error(t, err)

	// 获取待处理操作
	pendingOps := sm.GetPendingOperations()
	assert.Len(t, pendingOps, 2, "应有2个待处理操作")

	// 查找特定操作
	foundPending1 := false
	foundPending2 := false

	for _, op := range pendingOps {
		if op.ID == pendingOp1.ID {
			foundPending1 = true
		}
		if op.ID == pendingOp2.ID {
			foundPending2 = true
		}
	}

	assert.True(t, foundPending1, "待处理操作应包含pendingOp1")
	assert.True(t, foundPending2, "待处理操作应包含pendingOp2")
}

// TestGetFailedOperations 测试获取失败操作
func TestGetFailedOperations(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	sm := NewSyncManager(tmpDir)

	// 创建一个成功操作
	successOp := sm.CreateOperation(OperationCreate, "entity1", nil)
	err := sm.ExecuteOperation(context.Background(), successOp, func(ctx context.Context, op *Operation) error {
		return nil
	})
	assert.NoError(t, err)

	// 创建两个失败操作
	failedOp1 := sm.CreateOperation(OperationUpdate, "entity2", nil)
	err = sm.ExecuteOperation(context.Background(), failedOp1, func(ctx context.Context, op *Operation) error {
		return assert.AnError
	})
	assert.Error(t, err)

	failedOp2 := sm.CreateOperation(OperationDelete, "entity3", nil)
	err = sm.ExecuteOperation(context.Background(), failedOp2, func(ctx context.Context, op *Operation) error {
		return assert.AnError
	})
	assert.Error(t, err)

	// 获取失败操作
	failedOps := sm.GetFailedOperations()
	assert.Len(t, failedOps, 2, "应有2个失败操作")

	// 查找特定操作
	foundFailed1 := false
	foundFailed2 := false

	for _, op := range failedOps {
		if op.ID == failedOp1.ID {
			foundFailed1 = true
		}
		if op.ID == failedOp2.ID {
			foundFailed2 = true
		}
	}

	assert.True(t, foundFailed1, "失败操作应包含failedOp1")
	assert.True(t, foundFailed2, "失败操作应包含failedOp2")
}

// TestRetryOperation 测试重试操作
func TestRetryOperation(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	sm := NewSyncManager(tmpDir)

	// 创建一个失败操作
	failedOp := sm.CreateOperation(OperationUpdate, "entity", nil)
	err := sm.ExecuteOperation(context.Background(), failedOp, func(ctx context.Context, op *Operation) error {
		return assert.AnError
	})
	assert.Error(t, err)
	assert.Equal(t, "failed", failedOp.Status)

	// 重试操作（成功）
	err = sm.RetryOperation(context.Background(), failedOp, func(ctx context.Context, op *Operation) error {
		return nil // 现在成功
	})

	assert.NoError(t, err, "重试成功的操作不应返回错误")
	assert.Equal(t, "completed", failedOp.Status, "重试成功的操作状态应为completed")
	assert.Empty(t, failedOp.Error, "重试成功的操作不应有错误")

	// 验证文件更新
	fileName := filepath.Join(tmpDir, failedOp.ID+".json")
	fileData, err := os.ReadFile(fileName)
	assert.NoError(t, err)

	var savedOp Operation
	err = json.Unmarshal(fileData, &savedOp)
	assert.NoError(t, err)
	assert.Equal(t, "completed", savedOp.Status, "文件中操作状态应更新为completed")

	// 测试重试仍失败的情况
	stillFailOp := sm.CreateOperation(OperationDelete, "entity", nil)
	err = sm.ExecuteOperation(context.Background(), stillFailOp, func(ctx context.Context, op *Operation) error {
		return assert.AnError
	})
	assert.Error(t, err)

	// 重试但仍失败
	err = sm.RetryOperation(context.Background(), stillFailOp, func(ctx context.Context, op *Operation) error {
		return assert.AnError
	})

	assert.Error(t, err, "重试失败的操作应返回错误")
	assert.Equal(t, "failed", stillFailOp.Status, "重试失败的操作状态应为failed")
	assert.Contains(t, stillFailOp.Error, "assert.AnError", "新错误信息应被记录")
}

// TestLoadOperations 测试加载操作
func TestLoadOperations(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// 创建一个同步管理器并添加一些操作
	sm1 := NewSyncManager(tmpDir)
	op1 := sm1.CreateOperation(OperationCreate, "entity1", nil)
	op2 := sm1.CreateOperation(OperationUpdate, "entity2", nil)

	// 执行其中一个操作
	err := sm1.ExecuteOperation(context.Background(), op2, func(ctx context.Context, op *Operation) error {
		return nil
	})
	assert.NoError(t, err)

	// 创建一个新的同步管理器，应该加载已有操作
	sm2 := NewSyncManager(tmpDir)

	// 验证操作是否被加载
	assert.Len(t, sm2.operations, 2, "应加载2个操作")

	// 检查特定操作是否存在及其状态
	var foundOp1, foundOp2 bool

	for _, op := range sm2.operations {
		if op.ID == op1.ID {
			foundOp1 = true
			assert.Equal(t, "pending", op.Status, "op1状态应为pending")
		}
		if op.ID == op2.ID {
			foundOp2 = true
			assert.Equal(t, "completed", op.Status, "op2状态应为completed")
		}
	}

	assert.True(t, foundOp1, "应加载op1")
	assert.True(t, foundOp2, "应加载op2")
}

// TestClearOperations 测试清除操作
func TestClearOperations(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	sm := NewSyncManager(tmpDir)

	// 创建一些测试操作
	sm.CreateOperation(OperationCreate, "entity1", nil)
	sm.CreateOperation(OperationUpdate, "entity2", nil)

	// 验证操作已存在
	assert.Len(t, sm.operations, 2, "应有2个操作")

	files, err := os.ReadDir(tmpDir)
	assert.NoError(t, err)
	assert.Len(t, files, 2, "应有2个操作文件")

	// 清除操作
	err = sm.ClearOperations()
	assert.NoError(t, err, "清除操作不应出错")

	// 验证操作已被清除
	assert.Empty(t, sm.operations, "操作列表应为空")

	files, err = os.ReadDir(tmpDir)
	assert.NoError(t, err)
	assert.Empty(t, files, "操作文件应被删除")
}

// TestInvalidOperationJSON 测试处理无效的操作JSON
func TestInvalidOperationJSON(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// 创建一个无效的JSON文件
	invalidJSON := []byte("{\"id\":\"test-id\",\"type\":\"create\",INVALID")
	fileName := filepath.Join(tmpDir, "invalid.json")
	err := os.WriteFile(fileName, invalidJSON, 0644)
	require.NoError(t, err)

	// 创建同步管理器，应忽略无效文件
	sm := NewSyncManager(tmpDir)

	// 验证无效操作被忽略
	assert.Empty(t, sm.operations, "无效操作应被忽略")

	// 添加一个有效操作
	sm.CreateOperation(OperationCreate, "entity", nil)
	assert.Len(t, sm.operations, 1, "有效操作应被添加")
}

// TestDirectoryCreation 测试目录创建
func TestDirectoryCreation(t *testing.T) {
	// 使用不存在的目录
	nonExistentDir := filepath.Join(os.TempDir(), "sync-test-"+time.Now().Format("20060102150405"))
	defer os.RemoveAll(nonExistentDir)

	// 确保目录不存在
	_, err := os.Stat(nonExistentDir)
	assert.True(t, os.IsNotExist(err), "测试前目录不应存在")

	// 创建使用不存在目录的同步管理器
	sm := NewSyncManager(nonExistentDir)

	// 添加操作以触发目录创建
	sm.CreateOperation(OperationCreate, "entity", nil)

	// 验证目录已创建
	_, err = os.Stat(nonExistentDir)
	assert.NoError(t, err, "目录应已创建")
	assert.True(t, !os.IsNotExist(err), "目录应存在")
}
