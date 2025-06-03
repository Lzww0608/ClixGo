/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-3 18:50:39
* @Description: 同步机制的核心实现
 */

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
	"go.uber.org/zap"
)

// OperationType 表示操作类型枚举
// 用于标识不同类型的数据同步操作
type OperationType string

const (
	// OperationCreate 创建操作
	OperationCreate OperationType = "create"
	// OperationUpdate 更新操作
	OperationUpdate OperationType = "update"
	// OperationDelete 删除操作
	OperationDelete OperationType = "delete"
)

// Operation 表示一个数据同步操作
// 包含操作的完整元数据和执行状态信息
type Operation struct {
	ID        string        `json:"id"`              // 操作唯一标识符
	Type      OperationType `json:"type"`            // 操作类型
	Entity    string        `json:"entity"`          // 操作的目标实体
	Data      interface{}   `json:"data"`            // 操作数据负载
	Timestamp time.Time     `json:"timestamp"`       // 操作创建时间戳
	Status    string        `json:"status"`          // 操作执行状态
	Error     string        `json:"error,omitempty"` // 操作失败时的错误信息
}

// SyncManager 同步管理器
// 负责管理数据同步操作，支持离线模式和操作持久化
type SyncManager struct {
	operations []*Operation // 所有操作记录
	mu         sync.RWMutex // 读写锁保护并发访问
	syncDir    string       // 同步数据存储目录
	offline    bool         // 离线模式标志
	logger     *zap.Logger  // 结构化日志记录器
}

// NewSyncManager 创建新的同步管理器
//
// 参数:
//   - syncDir: 同步数据存储目录路径
//
// 返回:
//   - *SyncManager: 初始化完成的同步管理器实例
//
// 该函数会自动创建必要的存储目录，并尝试加载现有的操作记录
func NewSyncManager(syncDir string) *SyncManager {
	logger, err := zap.NewProduction()
	if err != nil {
		// 如果无法创建日志器，使用空日志器
		logger = zap.NewNop()
	}

	sm := &SyncManager{
		operations: make([]*Operation, 0),
		syncDir:    syncDir,
		offline:    false,
		logger:     logger,
	}

	// 加载现有操作
	if err := sm.loadOperations(); err != nil {
		sm.logger.Error("加载操作失败", zap.Error(err))
	}

	return sm
}

// SetOffline 设置同步管理器的离线模式状态
//
// 参数:
//   - offline: true表示进入离线模式，false表示恢复在线模式
//
// 在离线模式下，操作将被缓存到本地，等待网络恢复后同步
func (sm *SyncManager) SetOffline(offline bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.offline = offline
}

// IsOffline 检查同步管理器是否处于离线模式
//
// 返回:
//   - bool: true表示离线模式，false表示在线模式
func (sm *SyncManager) IsOffline() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.offline
}

// CreateOperation 创建新的同步操作
//
// 参数:
//   - opType: 操作类型（创建、更新、删除）
//   - entity: 操作的目标实体标识
//   - data: 操作的数据负载
//
// 返回:
//   - *Operation: 创建的操作实例
//
// 该函数会自动生成唯一ID、设置时间戳，并将操作保存到持久化存储
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
		sm.logger.Error("保存操作失败",
			zap.String("operation_id", op.ID),
			zap.Error(err))
	}

	return op
}

// ExecuteOperation 执行指定的同步操作
//
// 参数:
//   - ctx: 上下文，用于取消和超时控制
//   - op: 要执行的操作实例
//   - fn: 实际执行操作的函数，接收上下文和操作作为参数
//
// 返回:
//   - error: 执行过程中的错误，nil表示成功
//
// 该函数会更新操作状态，记录执行结果，并保存到持久化存储
func (sm *SyncManager) ExecuteOperation(ctx context.Context, op *Operation, fn func(ctx context.Context, op *Operation) error) error {
	sm.mu.Lock()
	op.Status = "executing"
	sm.mu.Unlock()

	if err := sm.saveOperation(op); err != nil {
		return fmt.Errorf("保存操作失败: %w", err)
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
		return fmt.Errorf("保存操作失败: %w", err)
	}

	return err
}

// GetPendingOperations 获取所有待处理的操作列表
//
// 返回:
//   - []*Operation: 状态为"pending"的操作列表
//
// 该函数返回的是操作的副本，调用者可以安全地遍历和处理
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

// GetFailedOperations 获取所有执行失败的操作列表
//
// 返回:
//   - []*Operation: 状态为"failed"的操作列表
//
// 这些操作可以通过RetryOperation函数进行重试
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

// RetryOperation 重试执行失败的操作
//
// 参数:
//   - ctx: 上下文，用于取消和超时控制
//   - op: 要重试的操作实例
//   - fn: 实际执行操作的函数
//
// 返回:
//   - error: 重试执行过程中的错误，nil表示成功
//
// 该函数会清除之前的错误信息，重新设置操作状态并执行
func (sm *SyncManager) RetryOperation(ctx context.Context, op *Operation, fn func(ctx context.Context, op *Operation) error) error {
	sm.mu.Lock()
	op.Status = "retrying"
	op.Error = ""
	sm.mu.Unlock()

	if err := sm.saveOperation(op); err != nil {
		return fmt.Errorf("保存操作失败: %w", err)
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
		return fmt.Errorf("保存操作失败: %w", err)
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
		return fmt.Errorf("读取同步目录失败: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if err := os.Remove(filepath.Join(sm.syncDir, file.Name())); err != nil {
			sm.logger.Warn("删除操作文件失败",
				zap.String("file", file.Name()),
				zap.Error(err))
		}
	}

	return nil
}

// loadOperations 加载所有操作
func (sm *SyncManager) loadOperations() error {
	if err := os.MkdirAll(sm.syncDir, 0755); err != nil {
		return fmt.Errorf("创建同步目录失败: %w", err)
	}

	files, err := os.ReadDir(sm.syncDir)
	if err != nil {
		return fmt.Errorf("读取同步目录失败: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Join(sm.syncDir, file.Name()))
		if err != nil {
			sm.logger.Warn("读取操作文件失败",
				zap.String("file", file.Name()),
				zap.Error(err))
			continue
		}

		var op Operation
		if err := json.Unmarshal(data, &op); err != nil {
			sm.logger.Warn("解析操作文件失败",
				zap.String("file", file.Name()),
				zap.Error(err))
			continue
		}

		sm.operations = append(sm.operations, &op)
	}

	return nil
}

// saveOperation 保存操作
func (sm *SyncManager) saveOperation(op *Operation) error {
	if err := os.MkdirAll(sm.syncDir, 0755); err != nil {
		return fmt.Errorf("创建同步目录失败: %w", err)
	}

	data, err := json.MarshalIndent(op, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化操作失败: %w", err)
	}

	filename := filepath.Join(sm.syncDir, op.ID+".json")
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("写入操作文件失败: %w", err)
	}

	return nil
}
