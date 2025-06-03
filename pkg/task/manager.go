/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-03 19:15:00
* @Description: 任务管理器的核心实现，提供任务创建、执行、监控等功能
 */

package task

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// TaskStatus 表示任务状态枚举
// 定义任务在生命周期中的各种状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 待执行状态
	TaskStatusRunning   TaskStatus = "running"   // 正在执行状态
	TaskStatusComplete  TaskStatus = "complete"  // 执行完成状态
	TaskStatusFailed    TaskStatus = "failed"    // 执行失败状态
	TaskStatusCancelled TaskStatus = "cancelled" // 已取消状态
)

// Task 表示一个后台任务
// 包含任务的完整生命周期信息和执行状态
type Task struct {
	ID          string      `json:"id"`                    // 任务唯一标识符
	Name        string      `json:"name"`                  // 任务名称
	Description string      `json:"description"`           // 任务描述
	Status      TaskStatus  `json:"status"`                // 任务当前状态
	Progress    float64     `json:"progress"`              // 任务进度（0.0-1.0）
	Result      string      `json:"result"`                // 任务执行结果
	Error       string      `json:"error,omitempty"`       // 错误信息（如果有）
	CreatedAt   time.Time   `json:"created_at"`            // 任务创建时间
	StartedAt   *time.Time  `json:"started_at,omitempty"`  // 任务开始时间
	FinishedAt  *time.Time  `json:"finished_at,omitempty"` // 任务完成时间
	Metadata    interface{} `json:"metadata,omitempty"`    // 任务元数据
}

// TaskManager 后台任务管理器
// 负责任务的创建、执行、监控、持久化和订阅通知
type TaskManager struct {
	mu          sync.RWMutex            // 读写锁保护并发访问
	tasks       map[string]*Task        // 任务存储映射
	subscribers map[string][]chan *Task // 任务订阅者映射
	logger      *zap.Logger             // 日志记录器
	storePath   string                  // 持久化存储路径
}

// NewTaskManager 创建新的任务管理器
//
// 参数:
//   - logger: 日志记录器，用于记录任务管理器的操作日志
//   - storePath: 任务持久化存储文件路径
//
// 返回:
//   - *TaskManager: 创建的任务管理器实例
//   - error: 创建过程中的错误，nil表示成功
//
// 该函数会自动加载持久化的任务并启动定期保存服务
func NewTaskManager(logger *zap.Logger, storePath string) (*TaskManager, error) {
	if err := os.MkdirAll(filepath.Dir(storePath), 0755); err != nil {
		return nil, errors.Wrap(err, "创建存储目录失败")
	}

	taskManager := &TaskManager{
		tasks:       make(map[string]*Task),
		subscribers: make(map[string][]chan *Task),
		logger:      logger,
		storePath:   storePath,
	}

	// 加载持久化的任务
	if err := taskManager.loadTasks(); err != nil {
		return nil, errors.Wrap(err, "加载任务失败")
	}

	// 启动定期保存
	go taskManager.periodicSave()

	return taskManager, nil
}

// CreateTask 创建新的后台任务
//
// 参数:
//   - name: 任务名称，用于标识任务类型
//   - description: 任务描述，详细说明任务内容
//   - metadata: 任务元数据，可以存储任务相关的额外信息
//
// 返回:
//   - *Task: 创建的任务对象
//   - error: 创建过程中的错误，nil表示成功
//
// 新创建的任务状态为Pending，需要调用StartTask来执行
func (tm *TaskManager) CreateTask(name, description string, metadata interface{}) (*Task, error) {
	newTask := &Task{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Status:      TaskStatusPending,
		CreatedAt:   time.Now(),
		Metadata:    metadata,
	}

	tm.mu.Lock()
	tm.tasks[newTask.ID] = newTask
	tm.mu.Unlock()

	tm.notifySubscribers(newTask)
	return newTask, tm.saveTasks()
}

// StartTask 启动任务执行
//
// 参数:
//   - ctx: 上下文，用于控制任务执行的取消和超时
//   - taskID: 要启动的任务ID
//   - fn: 任务执行函数，接收上下文和任务对象
//
// 返回:
//   - error: 启动错误，nil表示成功
//
// 该函数会在后台goroutine中异步执行任务，并自动更新任务状态
func (tm *TaskManager) StartTask(ctx context.Context, taskID string, fn func(context.Context, *Task) error) error {
	// 获取并更新任务状态，使用锁保护
	tm.mu.Lock()
	currentTask, taskExists := tm.tasks[taskID]
	if !taskExists {
		tm.mu.Unlock()
		return errors.New("任务不存在")
	}

	if currentTask.Status != TaskStatusPending {
		tm.mu.Unlock()
		return errors.New("任务状态不正确")
	}

	// 更新任务状态
	startTime := time.Now()
	currentTask.Status = TaskStatusRunning
	currentTask.StartedAt = &startTime
	currentTask.Progress = 0.0 // 确保进度初始化为0

	// 创建任务副本，避免并发修改
	taskCopy := *currentTask
	tm.mu.Unlock()

	// 通知订阅者
	tm.notifySubscribers(currentTask)

	// 在后台执行任务
	go func() {
		// 使用任务副本调用执行函数
		executionError := fn(ctx, &taskCopy)

		// 任务完成后，更新原始任务的状态
		tm.mu.Lock()
		// 重新获取任务，确保更新的是最新状态
		currentTask, taskExists := tm.tasks[taskID]
		if !taskExists {
			tm.mu.Unlock()
			if tm.logger != nil {
				tm.logger.Error("任务完成后未找到任务", zap.String("task_id", taskID))
			}
			return
		}

		finishTime := time.Now()
		currentTask.FinishedAt = &finishTime
		if executionError != nil {
			currentTask.Status = TaskStatusFailed
			currentTask.Error = executionError.Error()
		} else {
			currentTask.Status = TaskStatusComplete
			currentTask.Progress = 1.0 // 确保进度设为100%
		}
		tm.mu.Unlock()

		tm.notifySubscribers(currentTask)
		tm.saveTasks()
	}()

	return nil
}

// UpdateTaskProgress 更新任务执行进度
//
// 参数:
//   - taskID: 任务ID
//   - progress: 进度值（0.0-1.0）
//
// 返回:
//   - error: 更新错误，nil表示成功
//
// 只有处于运行状态的任务才能更新进度
func (tm *TaskManager) UpdateTaskProgress(taskID string, progress float64) error {
	// 使用读写锁保护对任务的访问
	tm.mu.Lock()
	currentTask, taskExists := tm.tasks[taskID]
	if !taskExists {
		tm.mu.Unlock()
		return errors.New("任务不存在")
	}

	// 确保进度在有效范围内
	if progress < 0 {
		progress = 0
	} else if progress > 1 {
		progress = 1
	}

	// 只有在运行状态才能更新进度
	if currentTask.Status != TaskStatusRunning {
		tm.mu.Unlock()
		return errors.New("任务不在运行状态，无法更新进度")
	}

	// 更新进度
	currentTask.Progress = progress

	// 创建任务副本用于通知
	taskCopy := *currentTask
	tm.mu.Unlock()

	// 通知订阅者（使用任务副本）
	tm.notifySubscribers(&taskCopy)
	return nil
}

// CancelTask 取消任务执行
//
// 参数:
//   - taskID: 要取消的任务ID
//
// 返回:
//   - error: 取消错误，nil表示成功
//
// 只有待执行或正在执行的任务才能被取消
func (tm *TaskManager) CancelTask(taskID string) error {
	tm.mu.Lock()

	targetTask, taskExists := tm.tasks[taskID]
	if !taskExists {
		tm.mu.Unlock()
		return errors.New("任务不存在")
	}

	if targetTask.Status != TaskStatusRunning && targetTask.Status != TaskStatusPending {
		tm.mu.Unlock()
		return errors.New("任务无法取消")
	}

	cancelTime := time.Now()
	targetTask.Status = TaskStatusCancelled
	targetTask.FinishedAt = &cancelTime

	// 创建任务副本用于通知
	taskCopy := *targetTask

	// 解锁，避免在持有写锁的情况下调用saveTasks
	tm.mu.Unlock()

	// 在解锁后保存任务和通知订阅者
	saveError := tm.saveTasks()
	tm.notifySubscribers(&taskCopy)

	return saveError
}

// GetTask 获取指定任务的详细信息
//
// 参数:
//   - taskID: 任务ID
//
// 返回:
//   - *Task: 任务对象副本
//   - error: 获取错误，nil表示成功
//
// 返回的是任务副本，避免外部修改影响内部状态
func (tm *TaskManager) GetTask(taskID string) (*Task, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	targetTask, taskExists := tm.tasks[taskID]
	if !taskExists {
		return nil, errors.New("任务不存在")
	}

	// 返回任务副本，避免外部修改影响内部状态
	taskCopy := *targetTask
	return &taskCopy, nil
}

// ListTasks 列出所有任务
//
// 返回:
//   - []*Task: 所有任务的副本列表
//
// 返回的任务列表是副本，可以安全地在外部使用
func (tm *TaskManager) ListTasks() []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	taskList := make([]*Task, 0, len(tm.tasks))
	for _, originalTask := range tm.tasks {
		// 创建任务副本
		taskCopy := *originalTask
		taskList = append(taskList, &taskCopy)
	}
	return taskList
}

// SubscribeTask 订阅指定任务的状态更新
//
// 参数:
//   - taskID: 要订阅的任务ID
//
// 返回:
//   - chan *Task: 接收任务更新的通道
//
// 通道缓冲大小为10，订阅者需要及时消费以避免阻塞
func (tm *TaskManager) SubscribeTask(taskID string) chan *Task {
	updateChannel := make(chan *Task, 10)
	tm.mu.Lock()
	tm.subscribers[taskID] = append(tm.subscribers[taskID], updateChannel)
	tm.mu.Unlock()

	// 立即发送当前状态
	currentTask, err := tm.GetTask(taskID)
	if err == nil {
		select {
		case updateChannel <- currentTask:
			// 成功发送
		default:
			// 如果通道已满，忽略
		}
	}

	return updateChannel
}

// UnsubscribeTask 取消订阅任务更新
//
// 参数:
//   - taskID: 任务ID
//   - ch: 要取消的订阅通道
//
// 该函数会关闭指定的通道并从订阅列表中移除
func (tm *TaskManager) UnsubscribeTask(taskID string, ch chan *Task) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	subscriberList := tm.subscribers[taskID]
	for index, subscriber := range subscriberList {
		if subscriber == ch {
			tm.subscribers[taskID] = append(subscriberList[:index], subscriberList[index+1:]...)
			close(subscriber)
			break
		}
	}
}

// notifySubscribers 向所有订阅者发送任务更新通知
//
// 参数:
//   - task: 要发送的任务对象
//
// 该函数会非阻塞地向所有订阅者发送通知，如果通道已满则跳过
func (tm *TaskManager) notifySubscribers(task *Task) {
	tm.mu.RLock()
	subscriberList := tm.subscribers[task.ID]
	// 创建一个任务副本，防止并发修改导致的问题
	taskCopy := *task
	tm.mu.RUnlock()

	for _, subscriberChannel := range subscriberList {
		select {
		case subscriberChannel <- &taskCopy:
			// 成功发送
		default:
			// 如果通道已满，记录日志但不阻塞
			if tm.logger != nil {
				tm.logger.Warn("通知订阅者失败：通道已满", zap.String("task_id", task.ID))
			}
		}
	}
}

// loadTasks 从持久化文件加载任务数据
//
// 返回:
//   - error: 加载错误，nil表示成功
//
// 如果文件不存在则忽略，如果文件损坏则返回错误
func (tm *TaskManager) loadTasks() error {
	fileData, err := os.ReadFile(tm.storePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "读取任务文件失败")
	}

	var loadedTasks map[string]*Task
	if err := json.Unmarshal(fileData, &loadedTasks); err != nil {
		return errors.Wrap(err, "解析任务数据失败")
	}

	tm.mu.Lock()
	tm.tasks = loadedTasks
	tm.mu.Unlock()

	return nil
}

// saveTasks 将任务数据保存到持久化文件
//
// 返回:
//   - error: 保存错误，nil表示成功
//
// 该函数会创建任务的深拷贝后再序列化，避免持有锁的时间过长
func (tm *TaskManager) saveTasks() error {
	// 首先创建任务的一个深拷贝
	tm.mu.RLock()
	taskCopyMap := make(map[string]*Task, len(tm.tasks))
	for taskID, originalTask := range tm.tasks {
		taskCopy := *originalTask
		taskCopyMap[taskID] = &taskCopy
	}
	tm.mu.RUnlock()

	// 在不持有锁的情况下进行序列化
	serializedData, err := json.MarshalIndent(taskCopyMap, "", "  ")
	if err != nil {
		return errors.Wrap(err, "序列化任务数据失败")
	}

	if err := os.WriteFile(tm.storePath, serializedData, 0644); err != nil {
		return errors.Wrap(err, "写入任务文件失败")
	}

	return nil
}

// periodicSave 定期保存任务状态到持久化文件
// 该函数运行在独立的goroutine中，每分钟保存一次任务状态
func (tm *TaskManager) periodicSave() {
	saveTicker := time.NewTicker(1 * time.Minute)
	defer saveTicker.Stop()

	for range saveTicker.C {
		if err := tm.saveTasks(); err != nil {
			tm.logger.Error("保存任务状态失败", zap.Error(err))
		}
	}
}
