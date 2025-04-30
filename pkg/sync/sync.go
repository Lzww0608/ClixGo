package sync

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

// OperationType 表示操作类型
type OperationType string

const (
	OperationCreate OperationType = "create"
	OperationUpdate OperationType = "update"
	OperationDelete OperationType = "delete"
)

// Operation 表示一个操作
type Operation struct {
	ID        string        `json:"id"`
	Type      OperationType `json:"type"`
	Entity    string        `json:"entity"`
	Data      interface{}   `json:"data"`
	Timestamp time.Time     `json:"timestamp"`
	Status    string        `json:"status"`
	Error     string        `json:"error,omitempty"`
}

// SyncManager 管理数据同步
type SyncManager struct {
	operations []*Operation
	mu         sync.RWMutex
	syncDir    string
	offline    bool
}

// NewSyncManager 创建新的同步管理器
func NewSyncManager(syncDir string) *SyncManager {
	sm := &SyncManager{
		operations: make([]*Operation, 0),
		syncDir:    syncDir,
		offline:    false,
	}

	// 加载现有操作
	if err := sm.loadOperations(); err != nil {
		fmt.Printf("加载操作失败: %v\n", err)
	}

	return sm
}

// SetOffline 设置离线模式
func (sm *SyncManager) SetOffline(offline bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.offline = offline
}

// IsOffline 检查是否处于离线模式
func (sm *SyncManager) IsOffline() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.offline
}

// CreateOperation 创建新操作
func (sm *SyncManager) CreateOperation(opType OperationType, entity string, data interface{}) *Operation {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	op := &Operation{
		ID:        uuid.New().String(),
		Type:      opType,
		Entity:    entity,
		Data:      data,
		Timestamp: time.Now(),
		Status:    "pending",
	}

	sm.operations = append(sm.operations, op)
	if err := sm.saveOperation(op); err != nil {
		fmt.Printf("保存操作失败: %v\n", err)
	}

	return op
}

// ExecuteOperation 执行操作
func (sm *SyncManager) ExecuteOperation(ctx context.Context, op *Operation, fn func(ctx context.Context, op *Operation) error) error {
	sm.mu.Lock()
	op.Status = "executing"
	sm.mu.Unlock()

	if err := sm.saveOperation(op); err != nil {
		return fmt.Errorf("保存操作失败: %v", err)
	}

	err := fn(ctx, op)

	sm.mu.Lock()
	if err != nil {
		op.Status = "failed"
		op.Error = err.Error()
	} else {
		op.Status = "completed"
	}
	sm.mu.Unlock()

	if err := sm.saveOperation(op); err != nil {
		return fmt.Errorf("保存操作失败: %v", err)
	}

	return err
}

// GetPendingOperations 获取待处理的操作
func (sm *SyncManager) GetPendingOperations() []*Operation {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	pending := make([]*Operation, 0)
	for _, op := range sm.operations {
		if op.Status == "pending" {
			pending = append(pending, op)
		}
	}

	return pending
}

// GetFailedOperations 获取失败的操作
func (sm *SyncManager) GetFailedOperations() []*Operation {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	failed := make([]*Operation, 0)
	for _, op := range sm.operations {
		if op.Status == "failed" {
			failed = append(failed, op)
		}
	}

	return failed
}

// RetryOperation 重试操作
func (sm *SyncManager) RetryOperation(ctx context.Context, op *Operation, fn func(ctx context.Context, op *Operation) error) error {
	sm.mu.Lock()
	op.Status = "retrying"
	op.Error = ""
	sm.mu.Unlock()

	if err := sm.saveOperation(op); err != nil {
		return fmt.Errorf("保存操作失败: %v", err)
	}

	err := fn(ctx, op)

	sm.mu.Lock()
	if err != nil {
		op.Status = "failed"
		op.Error = err.Error()
	} else {
		op.Status = "completed"
	}
	sm.mu.Unlock()

	if err := sm.saveOperation(op); err != nil {
		return fmt.Errorf("保存操作失败: %v", err)
	}

	return err
}

// ClearOperations 清除操作
func (sm *SyncManager) ClearOperations() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.operations = make([]*Operation, 0)

	// 删除所有操作文件
	files, err := os.ReadDir(sm.syncDir)
	if err != nil {
		return fmt.Errorf("读取同步目录失败: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if err := os.Remove(filepath.Join(sm.syncDir, file.Name())); err != nil {
			fmt.Printf("删除操作文件 %s 失败: %v\n", file.Name(), err)
		}
	}

	return nil
}

// loadOperations 加载所有操作
func (sm *SyncManager) loadOperations() error {
	if err := os.MkdirAll(sm.syncDir, 0755); err != nil {
		return fmt.Errorf("创建同步目录失败: %v", err)
	}

	files, err := os.ReadDir(sm.syncDir)
	if err != nil {
		return fmt.Errorf("读取同步目录失败: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Join(sm.syncDir, file.Name()))
		if err != nil {
			fmt.Printf("读取操作文件 %s 失败: %v\n", file.Name(), err)
			continue
		}

		var op Operation
		if err := json.Unmarshal(data, &op); err != nil {
			fmt.Printf("解析操作文件 %s 失败: %v\n", file.Name(), err)
			continue
		}

		sm.operations = append(sm.operations, &op)
	}

	return nil
}

// saveOperation 保存操作
func (sm *SyncManager) saveOperation(op *Operation) error {
	if err := os.MkdirAll(sm.syncDir, 0755); err != nil {
		return fmt.Errorf("创建同步目录失败: %v", err)
	}

	data, err := json.MarshalIndent(op, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化操作失败: %v", err)
	}

	filename := filepath.Join(sm.syncDir, op.ID+".json")
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("写入操作文件失败: %v", err)
	}

	return nil
} 