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

// TaskStatus 表示任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusComplete  TaskStatus = "complete"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Task 表示一个后台任务
type Task struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Status      TaskStatus  `json:"status"`
	Progress    float64     `json:"progress"`
	Result      string      `json:"result"`
	Error       string      `json:"error,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	FinishedAt  *time.Time  `json:"finished_at,omitempty"`
	Metadata    interface{} `json:"metadata,omitempty"`
}

// TaskManager 管理后台任务
type TaskManager struct {
	mu          sync.RWMutex
	tasks       map[string]*Task
	subscribers map[string][]chan *Task
	logger      *zap.Logger
	storePath   string
}

// NewTaskManager 创建任务管理器
func NewTaskManager(logger *zap.Logger, storePath string) (*TaskManager, error) {
	if err := os.MkdirAll(filepath.Dir(storePath), 0755); err != nil {
		return nil, errors.Wrap(err, "创建存储目录失败")
	}

	tm := &TaskManager{
		tasks:       make(map[string]*Task),
		subscribers: make(map[string][]chan *Task),
		logger:      logger,
		storePath:   storePath,
	}

	// 加载持久化的任务
	if err := tm.loadTasks(); err != nil {
		return nil, errors.Wrap(err, "加载任务失败")
	}

	// 启动定期保存
	go tm.periodicSave()

	return tm, nil
}

// CreateTask 创建新任务
func (tm *TaskManager) CreateTask(name, description string, metadata interface{}) (*Task, error) {
	task := &Task{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Status:      TaskStatusPending,
		CreatedAt:   time.Now(),
		Metadata:    metadata,
	}

	tm.mu.Lock()
	tm.tasks[task.ID] = task
	tm.mu.Unlock()

	tm.notifySubscribers(task)
	return task, tm.saveTasks()
}

// StartTask 启动任务
func (tm *TaskManager) StartTask(ctx context.Context, taskID string, fn func(context.Context, *Task) error) error {
	// 获取并更新任务状态，使用锁保护
	tm.mu.Lock()
	task, ok := tm.tasks[taskID]
	if !ok {
		tm.mu.Unlock()
		return errors.New("任务不存在")
	}

	if task.Status != TaskStatusPending {
		tm.mu.Unlock()
		return errors.New("任务状态不正确")
	}

	// 更新任务状态
	now := time.Now()
	task.Status = TaskStatusRunning
	task.StartedAt = &now
	task.Progress = 0.0 // 确保进度初始化为0

	// 创建任务副本，避免并发修改
	taskCopy := *task
	tm.mu.Unlock()

	// 通知订阅者
	tm.notifySubscribers(task)

	// 在后台执行任务
	go func() {
		// 使用任务副本调用执行函数
		err := fn(ctx, &taskCopy)

		// 任务完成后，更新原始任务的状态
		tm.mu.Lock()
		// 重新获取任务，确保更新的是最新状态
		task, ok := tm.tasks[taskID]
		if !ok {
			tm.mu.Unlock()
			if tm.logger != nil {
				tm.logger.Error("任务完成后未找到任务", zap.String("task_id", taskID))
			}
			return
		}

		now := time.Now()
		task.FinishedAt = &now
		if err != nil {
			task.Status = TaskStatusFailed
			task.Error = err.Error()
		} else {
			task.Status = TaskStatusComplete
			task.Progress = 1.0 // 确保进度设为100%
		}
		tm.mu.Unlock()

		tm.notifySubscribers(task)
		tm.saveTasks()
	}()

	return nil
}

// UpdateTaskProgress 更新任务进度
func (tm *TaskManager) UpdateTaskProgress(taskID string, progress float64) error {
	// 使用读写锁保护对任务的访问
	tm.mu.Lock()
	task, ok := tm.tasks[taskID]
	if !ok {
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
	if task.Status != TaskStatusRunning {
		tm.mu.Unlock()
		return errors.New("任务不在运行状态，无法更新进度")
	}

	// 更新进度
	task.Progress = progress

	// 创建任务副本用于通知
	taskCopy := *task
	tm.mu.Unlock()

	// 通知订阅者（使用任务副本）
	tm.notifySubscribers(&taskCopy)
	return nil
}

// CancelTask 取消任务
func (tm *TaskManager) CancelTask(taskID string) error {
	tm.mu.Lock()

	task, ok := tm.tasks[taskID]
	if !ok {
		tm.mu.Unlock()
		return errors.New("任务不存在")
	}

	if task.Status != TaskStatusRunning && task.Status != TaskStatusPending {
		tm.mu.Unlock()
		return errors.New("任务无法取消")
	}

	now := time.Now()
	task.Status = TaskStatusCancelled
	task.FinishedAt = &now

	// 创建任务副本用于通知
	taskCopy := *task

	// 解锁，避免在持有写锁的情况下调用saveTasks
	tm.mu.Unlock()

	// 在解锁后保存任务和通知订阅者
	err := tm.saveTasks()
	tm.notifySubscribers(&taskCopy)

	return err
}

// GetTask 获取任务信息
func (tm *TaskManager) GetTask(taskID string) (*Task, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	task, ok := tm.tasks[taskID]
	if !ok {
		return nil, errors.New("任务不存在")
	}

	// 返回任务副本，避免外部修改影响内部状态
	taskCopy := *task
	return &taskCopy, nil
}

// ListTasks 列出所有任务
func (tm *TaskManager) ListTasks() []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		// 创建任务副本
		taskCopy := *task
		tasks = append(tasks, &taskCopy)
	}
	return tasks
}

// SubscribeTask 订阅任务更新
func (tm *TaskManager) SubscribeTask(taskID string) chan *Task {
	ch := make(chan *Task, 10)
	tm.mu.Lock()
	tm.subscribers[taskID] = append(tm.subscribers[taskID], ch)
	tm.mu.Unlock()

	// 立即发送当前状态
	task, err := tm.GetTask(taskID)
	if err == nil {
		select {
		case ch <- task:
			// 成功发送
		default:
			// 如果通道已满，忽略
		}
	}

	return ch
}

// UnsubscribeTask 取消订阅任务更新
func (tm *TaskManager) UnsubscribeTask(taskID string, ch chan *Task) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	subs := tm.subscribers[taskID]
	for i, sub := range subs {
		if sub == ch {
			tm.subscribers[taskID] = append(subs[:i], subs[i+1:]...)
			close(sub)
			break
		}
	}
}

// notifySubscribers 通知订阅者
func (tm *TaskManager) notifySubscribers(task *Task) {
	tm.mu.RLock()
	subs := tm.subscribers[task.ID]
	// 创建一个任务副本，防止并发修改导致的问题
	taskCopy := *task
	tm.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- &taskCopy:
			// 成功发送
		default:
			// 如果通道已满，记录日志但不阻塞
			if tm.logger != nil {
				tm.logger.Warn("通知订阅者失败：通道已满", zap.String("task_id", task.ID))
			}
		}
	}
}

// loadTasks 从文件加载任务
func (tm *TaskManager) loadTasks() error {
	data, err := os.ReadFile(tm.storePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "读取任务文件失败")
	}

	var tasks map[string]*Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return errors.Wrap(err, "解析任务数据失败")
	}

	tm.mu.Lock()
	tm.tasks = tasks
	tm.mu.Unlock()

	return nil
}

// saveTasks 保存任务到文件
func (tm *TaskManager) saveTasks() error {
	// 首先创建任务的一个深拷贝
	tm.mu.RLock()
	tasksCopy := make(map[string]*Task, len(tm.tasks))
	for id, task := range tm.tasks {
		taskCopy := *task
		tasksCopy[id] = &taskCopy
	}
	tm.mu.RUnlock()

	// 在不持有锁的情况下进行序列化
	data, err := json.MarshalIndent(tasksCopy, "", "  ")
	if err != nil {
		return errors.Wrap(err, "序列化任务数据失败")
	}

	if err := os.WriteFile(tm.storePath, data, 0644); err != nil {
		return errors.Wrap(err, "写入任务文件失败")
	}

	return nil
}

// periodicSave 定期保存任务状态
func (tm *TaskManager) periodicSave() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if err := tm.saveTasks(); err != nil {
			tm.logger.Error("保存任务状态失败", zap.Error(err))
		}
	}
}
