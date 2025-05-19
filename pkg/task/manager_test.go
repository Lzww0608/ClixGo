package task

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 创建临时测试目录
func setupTestDir(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "task-test")
	require.NoError(t, err, "创建临时目录失败")

	// 返回清理函数
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// TestNewTaskManager 测试创建任务管理器
func TestNewTaskManager(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	storePath := filepath.Join(tmpDir, "tasks.json")
	logger := zaptest.NewLogger(t)

	tm, err := NewTaskManager(logger, storePath)
	assert.NoError(t, err, "创建任务管理器不应出错")
	assert.NotNil(t, tm, "任务管理器不应为nil")
	assert.Empty(t, tm.tasks, "初始任务列表应为空")
	assert.Equal(t, storePath, tm.storePath, "存储路径应正确设置")
}

// TestCreateTask 测试创建任务
func TestCreateTask(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	storePath := filepath.Join(tmpDir, "tasks.json")
	logger := zaptest.NewLogger(t)

	tm, err := NewTaskManager(logger, storePath)
	require.NoError(t, err)

	// 创建测试任务
	metadata := map[string]string{"key": "value"}
	task, err := tm.CreateTask("测试任务", "这是一个测试任务", metadata)

	assert.NoError(t, err, "创建任务不应出错")
	assert.NotNil(t, task, "创建的任务不应为nil")
	assert.NotEmpty(t, task.ID, "任务ID不应为空")
	assert.Equal(t, "测试任务", task.Name, "任务名称应正确设置")
	assert.Equal(t, "这是一个测试任务", task.Description, "任务描述应正确设置")
	assert.Equal(t, TaskStatusPending, task.Status, "初始任务状态应为pending")
	assert.Equal(t, metadata, task.Metadata, "任务元数据应正确设置")
	assert.Equal(t, 0.0, task.Progress, "初始任务进度应为0")

	// 验证任务是否已添加到管理器
	assert.Len(t, tm.tasks, 1, "任务应添加到管理器")
}

// TestStartTask 测试启动任务
func TestStartTask(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	storePath := filepath.Join(tmpDir, "tasks.json")
	logger := zaptest.NewLogger(t)

	tm, err := NewTaskManager(logger, storePath)
	require.NoError(t, err)

	// 创建测试任务
	task, err := tm.CreateTask("测试任务", "这是一个测试任务", nil)
	require.NoError(t, err)

	// 启动任务（成功完成）
	err = tm.StartTask(context.Background(), task.ID, func(ctx context.Context, task *Task) error {
		// 模拟任务进度更新
		for i := 0; i <= 10; i++ {
			tm.UpdateTaskProgress(task.ID, float64(i)/10.0)
			if i < 10 {
				time.Sleep(10 * time.Millisecond)
			}
		}
		return nil
	})

	assert.NoError(t, err, "启动任务不应出错")

	// 等待任务完成
	time.Sleep(200 * time.Millisecond)

	// 获取更新后的任务状态
	updatedTask, err := tm.GetTask(task.ID)
	assert.NoError(t, err)
	assert.Equal(t, TaskStatusComplete, updatedTask.Status, "任务状态应为complete")
	assert.Equal(t, 1.0, updatedTask.Progress, "任务进度应为100%")
	assert.NotNil(t, updatedTask.StartedAt, "任务应有开始时间")
	assert.NotNil(t, updatedTask.FinishedAt, "任务应有结束时间")
}

// TestStartTaskFailed 测试任务执行失败
func TestStartTaskFailed(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	storePath := filepath.Join(tmpDir, "tasks.json")
	logger := zaptest.NewLogger(t)

	tm, err := NewTaskManager(logger, storePath)
	require.NoError(t, err)

	// 创建测试任务
	task, err := tm.CreateTask("测试任务", "这是一个测试任务", nil)
	require.NoError(t, err)

	// 启动任务（执行失败）
	err = tm.StartTask(context.Background(), task.ID, func(ctx context.Context, task *Task) error {
		return assert.AnError
	})

	assert.NoError(t, err, "启动任务不应报错，错误应在执行函数中处理")

	// 等待任务完成
	time.Sleep(50 * time.Millisecond)

	// 获取更新后的任务状态
	updatedTask, err := tm.GetTask(task.ID)
	assert.NoError(t, err)
	assert.Equal(t, TaskStatusFailed, updatedTask.Status, "任务状态应为failed")
	assert.Contains(t, updatedTask.Error, "assert.AnError", "任务应包含错误信息")
}

// TestUpdateTaskProgress 测试更新任务进度
func TestUpdateTaskProgress(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	storePath := filepath.Join(tmpDir, "tasks.json")
	logger := zaptest.NewLogger(t)

	tm, err := NewTaskManager(logger, storePath)
	require.NoError(t, err)

	// 创建测试任务
	task, err := tm.CreateTask("测试任务", "这是一个测试任务", nil)
	require.NoError(t, err)

	// 启动任务
	err = tm.StartTask(context.Background(), task.ID, func(ctx context.Context, task *Task) error {
		// 不立即返回，以便测试可以更新进度
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)

	// 等待任务开始
	time.Sleep(20 * time.Millisecond)

	// 更新进度
	err = tm.UpdateTaskProgress(task.ID, 0.5)
	assert.NoError(t, err, "更新任务进度不应出错")

	// 获取更新后的任务
	updatedTask, err := tm.GetTask(task.ID)
	assert.NoError(t, err)
	assert.Equal(t, 0.5, updatedTask.Progress, "任务进度应为50%")

	// 等待任务完成
	time.Sleep(100 * time.Millisecond)
}

// TestCancelTask 测试取消任务
func TestCancelTask(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	storePath := filepath.Join(tmpDir, "tasks.json")
	logger := zaptest.NewLogger(t)

	tm, err := NewTaskManager(logger, storePath)
	require.NoError(t, err)

	// 创建测试任务
	task, err := tm.CreateTask("测试任务", "这是一个测试任务", nil)
	require.NoError(t, err)

	// 取消任务
	err = tm.CancelTask(task.ID)
	assert.NoError(t, err, "取消任务不应出错")

	// 获取更新后的任务
	updatedTask, err := tm.GetTask(task.ID)
	assert.NoError(t, err)
	assert.Equal(t, TaskStatusCancelled, updatedTask.Status, "任务状态应为cancelled")
	assert.NotNil(t, updatedTask.FinishedAt, "任务应有结束时间")
}

// TestGetTask 测试获取单个任务
func TestGetTask(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	storePath := filepath.Join(tmpDir, "tasks.json")
	logger := zaptest.NewLogger(t)

	tm, err := NewTaskManager(logger, storePath)
	require.NoError(t, err)

	// 创建测试任务
	task, err := tm.CreateTask("测试任务", "这是一个测试任务", nil)
	require.NoError(t, err)

	// 获取任务
	retrievedTask, err := tm.GetTask(task.ID)
	assert.NoError(t, err, "获取任务不应出错")
	assert.Equal(t, task.ID, retrievedTask.ID, "应返回正确的任务")

	// 获取不存在的任务
	_, err = tm.GetTask("不存在的ID")
	assert.Error(t, err, "获取不存在的任务应返回错误")
}

// TestListTasks 测试列出所有任务
func TestListTasks(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	storePath := filepath.Join(tmpDir, "tasks.json")
	logger := zaptest.NewLogger(t)

	tm, err := NewTaskManager(logger, storePath)
	require.NoError(t, err)

	// 创建多个测试任务
	task1, err := tm.CreateTask("任务1", "这是任务1", nil)
	require.NoError(t, err)

	task2, err := tm.CreateTask("任务2", "这是任务2", nil)
	require.NoError(t, err)

	// 列出所有任务
	tasks := tm.ListTasks()
	assert.Len(t, tasks, 2, "应返回2个任务")

	// 验证是否返回了正确的任务
	var found1, found2 bool
	for _, task := range tasks {
		if task.ID == task1.ID {
			found1 = true
		}
		if task.ID == task2.ID {
			found2 = true
		}
	}

	assert.True(t, found1, "任务列表应包含任务1")
	assert.True(t, found2, "任务列表应包含任务2")
}

// TestTaskSubscription 测试任务订阅
func TestTaskSubscription(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	storePath := filepath.Join(tmpDir, "tasks.json")
	logger := zaptest.NewLogger(t)

	tm, err := NewTaskManager(logger, storePath)
	require.NoError(t, err)

	// 创建测试任务
	task, err := tm.CreateTask("测试任务", "这是一个测试任务", nil)
	require.NoError(t, err)

	// 订阅任务更新
	ch := tm.SubscribeTask(task.ID)

	// 启动任务
	err = tm.StartTask(context.Background(), task.ID, func(ctx context.Context, task *Task) error {
		// 更新进度
		tm.UpdateTaskProgress(task.ID, 0.5)
		time.Sleep(20 * time.Millisecond)
		tm.UpdateTaskProgress(task.ID, 1.0)
		return nil
	})
	assert.NoError(t, err)

	// 收集更新
	var updates []*Task
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		timeout := time.After(200 * time.Millisecond)

		for {
			select {
			case update, ok := <-ch:
				if !ok {
					return
				}
				updates = append(updates, update)
			case <-timeout:
				return
			}
		}
	}()

	// 等待任务完成
	time.Sleep(100 * time.Millisecond)

	// 取消订阅
	tm.UnsubscribeTask(task.ID, ch)

	wg.Wait()

	// 验证收到的更新
	assert.GreaterOrEqual(t, len(updates), 3, "应至少收到3个更新（初始、进度50%、进度100%）")

	// 验证最后一个更新显示任务已完成
	lastUpdate := updates[len(updates)-1]
	assert.Equal(t, TaskStatusComplete, lastUpdate.Status, "最后一个更新应显示任务已完成")
}

// TestConcurrentOperations 测试并发操作
func TestConcurrentOperations(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	storePath := filepath.Join(tmpDir, "tasks.json")
	logger := zaptest.NewLogger(t)

	tm, err := NewTaskManager(logger, storePath)
	require.NoError(t, err)

	// 创建多个测试任务
	tasks := make([]*Task, 5)
	for i := 0; i < 5; i++ {
		task, err := tm.CreateTask("并发任务", "测试并发操作", nil)
		require.NoError(t, err)
		tasks[i] = task
	}

	// 并发启动任务
	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		go func(taskID string) {
			defer wg.Done()
			err := tm.StartTask(context.Background(), taskID, func(ctx context.Context, task *Task) error {
				// 更新多个进度
				for i := 0; i <= 10; i++ {
					tm.UpdateTaskProgress(task.ID, float64(i)/10.0)
					time.Sleep(5 * time.Millisecond)
				}
				return nil
			})
			assert.NoError(t, err)
		}(task.ID)
	}

	// 等待所有任务完成
	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	// 验证所有任务都成功完成
	for _, task := range tasks {
		updatedTask, err := tm.GetTask(task.ID)
		assert.NoError(t, err)
		assert.Equal(t, TaskStatusComplete, updatedTask.Status, "任务应成功完成")
		assert.Equal(t, 1.0, updatedTask.Progress, "任务进度应为100%")
	}
}

// TestPersistence 测试持久化
func TestPersistence(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	storePath := filepath.Join(tmpDir, "tasks.json")
	logger := zaptest.NewLogger(t)

	// 创建任务管理器并添加任务
	tm1, err := NewTaskManager(logger, storePath)
	require.NoError(t, err)

	task1, err := tm1.CreateTask("任务1", "这是任务1", nil)
	require.NoError(t, err)

	// 启动任务并等待完成
	err = tm1.StartTask(context.Background(), task1.ID, func(ctx context.Context, task *Task) error {
		return nil
	})
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	// 创建第二个任务管理器，应该加载已有任务
	tm2, err := NewTaskManager(logger, storePath)
	require.NoError(t, err)

	// 验证任务是否被加载
	loadedTask, err := tm2.GetTask(task1.ID)
	assert.NoError(t, err, "应找到保存的任务")
	assert.Equal(t, task1.ID, loadedTask.ID, "任务ID应匹配")
	assert.Equal(t, TaskStatusComplete, loadedTask.Status, "任务状态应为complete")
}

// TestProgressRaceCondition 测试进度更新的竞态条件
func TestProgressRaceCondition(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	storePath := filepath.Join(tmpDir, "tasks.json")
	logger := zaptest.NewLogger(t)

	tm, err := NewTaskManager(logger, storePath)
	require.NoError(t, err)

	// 创建测试任务
	task, err := tm.CreateTask("测试任务", "这是一个测试任务", nil)
	require.NoError(t, err)

	// 启动任务
	err = tm.StartTask(context.Background(), task.ID, func(ctx context.Context, task *Task) error {
		// 使用任务副本尝试直接修改进度
		task.Progress = 0.2
		time.Sleep(50 * time.Millisecond)
		return nil
	})
	require.NoError(t, err)

	// 同时从外部更新进度
	time.Sleep(10 * time.Millisecond) // 确保任务已开始
	err = tm.UpdateTaskProgress(task.ID, 0.5)
	assert.NoError(t, err)

	// 等待任务完成
	time.Sleep(100 * time.Millisecond)

	// 获取最终任务状态
	finalTask, err := tm.GetTask(task.ID)
	assert.NoError(t, err)
	assert.Equal(t, TaskStatusComplete, finalTask.Status, "任务应成功完成")

	// 进度应该是最后设置的0.5或1.0，不应该是0.2
	assert.NotEqual(t, 0.2, finalTask.Progress, "任务进度不应为0.2（来自任务副本的直接修改）")
}
