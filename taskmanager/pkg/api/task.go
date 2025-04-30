package api

import (
	"context"
	"time"

	"github.com/yourusername/gocli/taskmanager/internal/task"
)

// TaskAPI 提供任务管理的API接口
type TaskAPI struct {
	tm *task.TaskManager
}

// NewTaskAPI 创建新的TaskAPI实例
func NewTaskAPI(tm *task.TaskManager) *TaskAPI {
	return &TaskAPI{tm: tm}
}

// CreateTask 创建新任务
func (api *TaskAPI) CreateTask(name, description string, metadata interface{}) (*task.Task, error) {
	return api.tm.CreateTask(name, description, metadata)
}

// StartTask 启动任务
func (api *TaskAPI) StartTask(ctx context.Context, taskID string, fn func(context.Context, *task.Task) error) error {
	return api.tm.StartTask(ctx, taskID, fn)
}

// UpdateTaskProgress 更新任务进度
func (api *TaskAPI) UpdateTaskProgress(taskID string, progress float64) error {
	return api.tm.UpdateTaskProgress(taskID, progress)
}

// CancelTask 取消任务
func (api *TaskAPI) CancelTask(taskID string) error {
	return api.tm.CancelTask(taskID)
}

// GetTask 获取任务信息
func (api *TaskAPI) GetTask(taskID string) (*task.Task, error) {
	return api.tm.GetTask(taskID)
}

// ListTasks 列出所有任务
func (api *TaskAPI) ListTasks() []*task.Task {
	return api.tm.ListTasks()
}

// SubscribeTask 订阅任务更新
func (api *TaskAPI) SubscribeTask(taskID string) <-chan *task.Task {
	return api.tm.SubscribeTask(taskID)
}

// UnsubscribeTask 取消订阅任务更新
func (api *TaskAPI) UnsubscribeTask(taskID string, ch <-chan *task.Task) {
	api.tm.UnsubscribeTask(taskID, ch)
} 