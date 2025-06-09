/*
* @Author: Lzww0608
* @Date: 2025-6-9 15:42:05
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-09 15:42:05
* @Description: 资源释放接口
 */

package interfaces

// Disposer 资源释放接口
// 实现此接口的服务可以在容器销毁时自动清理资源
type Disposer interface {
	// Dispose 释放资源
	Dispose() error
}

// AsyncDisposer 异步资源释放接口
type AsyncDisposer interface {
	// DisposeAsync 异步释放资源
	DisposeAsync() <-chan error
}

// ResourceManager 资源管理器接口
type ResourceManager interface {
	Disposer

	// IsDisposed 检查是否已释放
	IsDisposed() bool

	// AddResource 添加需要管理的资源
	AddResource(resource Disposer) error

	// RemoveResource 移除资源
	RemoveResource(resource Disposer) error

	// GetResourceCount 获取管理的资源数量
	GetResourceCount() int
}
