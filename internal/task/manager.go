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

	now := time.Now()
	task.Status = TaskStatusRunning
	task.StartedAt = &now
	tm.mu.Unlock()

	tm.notifySubscribers(task)

	// 在后台执行任务
	go func() {
		err := fn(ctx, task)
		tm.mu.Lock()
		now := time.Now()
		task.FinishedAt = &now
		if err != nil {
			task.Status = TaskStatusFailed
			task.Error = err.Error()
		} else {
			task.Status = TaskStatusComplete
		}
		tm.mu.Unlock()

		tm.notifySubscribers(task)
		tm.saveTasks()
	}()

	return nil
}

// UpdateTaskProgress 更新任务进度
func (tm *TaskManager) UpdateTaskProgress(taskID string, progress float64) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, ok := tm.tasks[taskID]
	if !ok {
		return errors.New("任务不存在")
	}

	task.Progress = progress
	tm.notifySubscribers(task)
	return nil
}

// CancelTask 取消任务
func (tm *TaskManager) CancelTask(taskID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, ok := tm.tasks[taskID]
	if !ok {
		return errors.New("任务不存在")
	}

	if task.Status != TaskStatusRunning && task.Status != TaskStatusPending {
		return errors.New("任务无法取消")
	}

	now := time.Now()
	task.Status = TaskStatusCancelled
	task.FinishedAt = &now

	tm.notifySubscribers(task)
	return tm.saveTasks()
}

// GetTask 获取任务信息
func (tm *TaskManager) GetTask(taskID string) (*Task, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	task, ok := tm.tasks[taskID]
	if !ok {
		return nil, errors.New("任务不存在")
	}

	return task, nil
}

// ListTasks 列出所有任务
func (tm *TaskManager) ListTasks() []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// SubscribeTask 订阅任务更新
func (tm *TaskManager) SubscribeTask(taskID string) chan *Task {
	ch := make(chan *Task, 10)
	tm.mu.Lock()
	tm.subscribers[taskID] = append(tm.subscribers[taskID], ch)
	tm.mu.Unlock()
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
	tm.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- task:
		default:
			// 如果通道已满，跳过
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
	tm.mu.RLock()
	data, err := json.MarshalIndent(tm.tasks, "", "  ")
	tm.mu.RUnlock()

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