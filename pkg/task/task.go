package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TaskStatus 表示任务状态
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusRunning   TaskStatus = "running"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
	StatusCancelled TaskStatus = "cancelled"
)

// Task 表示一个后台任务
type Task struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Status      TaskStatus  `json:"status"`
	StartTime   time.Time   `json:"start_time"`
	EndTime     time.Time   `json:"end_time,omitempty"`
	Progress    float64     `json:"progress"`
	Error       string      `json:"error,omitempty"`
	Result      interface{} `json:"result,omitempty"`
	Logs        []string    `json:"logs"`
	Context     context.Context
	CancelFunc  context.CancelFunc
	mu          sync.RWMutex
}

// TaskManager 管理后台任务
type TaskManager struct {
	tasks     map[string]*Task
	mu        sync.RWMutex
	taskDir   string
	notifyCh  chan *Task
}

// NewTaskManager 创建新的任务管理器
func NewTaskManager(taskDir string) *TaskManager {
	tm := &TaskManager{
		tasks:    make(map[string]*Task),
		taskDir:  taskDir,
		notifyCh: make(chan *Task, 100),
	}

	// 加载现有任务
	if err := tm.loadTasks(); err != nil {
		fmt.Printf("加载任务失败: %v\n", err)
	}

	return tm
}

// CreateTask 创建新任务
func (tm *TaskManager) CreateTask(name string) *Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	task := &Task{
		ID:         uuid.New().String(),
		Name:       name,
		Status:     StatusPending,
		StartTime:  time.Now(),
		Progress:   0,
		Context:    ctx,
		CancelFunc: cancel,
		Logs:       make([]string, 0),
	}

	tm.tasks[task.ID] = task
	if err := tm.saveTask(task); err != nil {
		fmt.Printf("保存任务失败: %v\n", err)
	}

	return task
}

// StartTask 启动任务
func (tm *TaskManager) StartTask(task *Task, fn func(ctx context.Context, task *Task) error) {
	tm.mu.Lock()
	task.Status = StatusRunning
	tm.mu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				tm.mu.Lock()
				task.Status = StatusFailed
				task.Error = fmt.Sprintf("任务崩溃: %v", r)
				tm.mu.Unlock()
			}
		}()

		err := fn(task.Context, task)
		tm.mu.Lock()
		if err != nil {
			task.Status = StatusFailed
			task.Error = err.Error()
		} else {
			task.Status = StatusCompleted
		}
		task.EndTime = time.Now()
		tm.mu.Unlock()

		if err := tm.saveTask(task); err != nil {
			fmt.Printf("保存任务失败: %v\n", err)
		}

		tm.notifyCh <- task
	}()
}

// CancelTask 取消任务
func (tm *TaskManager) CancelTask(id string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, ok := tm.tasks[id]
	if !ok {
		return fmt.Errorf("任务 %s 不存在", id)
	}

	if task.Status != StatusRunning {
		return fmt.Errorf("任务 %s 不在运行中", id)
	}

	task.CancelFunc()
	task.Status = StatusCancelled
	task.EndTime = time.Now()

	if err := tm.saveTask(task); err != nil {
		return fmt.Errorf("保存任务失败: %v", err)
	}

	return nil
}

// GetTask 获取任务
func (tm *TaskManager) GetTask(id string) (*Task, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	task, ok := tm.tasks[id]
	if !ok {
		return nil, fmt.Errorf("任务 %s 不存在", id)
	}

	return task, nil
}

// GetTasks 获取所有任务
func (tm *TaskManager) GetTasks() []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// UpdateTaskProgress 更新任务进度
func (tm *TaskManager) UpdateTaskProgress(task *Task, progress float64) {
	tm.mu.Lock()
	task.Progress = progress
	tm.mu.Unlock()

	if err := tm.saveTask(task); err != nil {
		fmt.Printf("保存任务失败: %v\n", err)
	}
}

// AddTaskLog 添加任务日志
func (tm *TaskManager) AddTaskLog(task *Task, log string) {
	tm.mu.Lock()
	task.Logs = append(task.Logs, log)
	tm.mu.Unlock()

	if err := tm.saveTask(task); err != nil {
		fmt.Printf("保存任务失败: %v\n", err)
	}
}

// SetTaskResult 设置任务结果
func (tm *TaskManager) SetTaskResult(task *Task, result interface{}) {
	tm.mu.Lock()
	task.Result = result
	tm.mu.Unlock()

	if err := tm.saveTask(task); err != nil {
		fmt.Printf("保存任务失败: %v\n", err)
	}
}

// NotifyChannel 获取任务通知通道
func (tm *TaskManager) NotifyChannel() <-chan *Task {
	return tm.notifyCh
}

// loadTasks 加载所有任务
func (tm *TaskManager) loadTasks() error {
	if err := os.MkdirAll(tm.taskDir, 0755); err != nil {
		return fmt.Errorf("创建任务目录失败: %v", err)
	}

	files, err := os.ReadDir(tm.taskDir)
	if err != nil {
		return fmt.Errorf("读取任务目录失败: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Join(tm.taskDir, file.Name()))
		if err != nil {
			fmt.Printf("读取任务文件 %s 失败: %v\n", file.Name(), err)
			continue
		}

		var task Task
		if err := json.Unmarshal(data, &task); err != nil {
			fmt.Printf("解析任务文件 %s 失败: %v\n", file.Name(), err)
			continue
		}

		tm.tasks[task.ID] = &task
	}

	return nil
}

// saveTask 保存任务
func (tm *TaskManager) saveTask(task *Task) error {
	if err := os.MkdirAll(tm.taskDir, 0755); err != nil {
		return fmt.Errorf("创建任务目录失败: %v", err)
	}

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化任务失败: %v", err)
	}

	filename := filepath.Join(tm.taskDir, task.ID+".json")
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("写入任务文件失败: %v", err)
	}

	return nil
} 