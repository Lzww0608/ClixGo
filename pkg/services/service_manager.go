/*
* @Author: Lzww0608
* @Date: 2025-6-10 10:57:49
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-10 10:57:52
* @Description: 服务管理器 - 整合服务注册中心和依赖注入容器
 */

package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/container"
	"github.com/Lzww0608/ClixGo/pkg/interfaces"
)

// ServiceManager 服务管理器
type ServiceManager struct {
	container *container.Container
	registry  *ServiceRegistry
	logger    interfaces.Logger
	mu        sync.RWMutex
	services  map[string]ServiceInfo
	started   bool
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name      string                 `json:"name"`
	Instance  interface{}            `json:"-"`
	Metadata  *ServiceMetadata       `json:"metadata"`
	Config    map[string]interface{} `json:"config"`
	StartedAt time.Time              `json:"started_at"`
	Status    string                 `json:"status"` // starting, running, stopping, stopped, error
	Error     string                 `json:"error,omitempty"`
}

// NewServiceManager 创建服务管理器
func NewServiceManager(logger interfaces.Logger) *ServiceManager {
	registry := NewServiceRegistry(logger)
	container := container.NewContainer()

	manager := &ServiceManager{
		container: container,
		registry:  registry,
		logger:    logger,
		services:  make(map[string]ServiceInfo),
		started:   false,
	}

	// 注册基础服务
	manager.registerCoreServices()

	logger.Info("服务管理器已初始化")
	return manager
}

// registerCoreServices 注册核心服务
func (sm *ServiceManager) registerCoreServices() {
	// 注册日志服务
	sm.container.RegisterInstance("logger", sm.logger)

	// 注册服务注册中心
	sm.container.RegisterInstance("service_registry", sm.registry)

	// 注册终端服务工厂
	sm.container.RegisterSingleton("terminal_service", func(c *container.Container) (interface{}, error) {
		logger, err := c.Get("logger")
		if err != nil {
			return nil, fmt.Errorf("failed to get logger: %w", err)
		}

		return NewSimpleTerminalService(logger.(interfaces.Logger)), nil
	}, "logger")
}

// RegisterService 注册服务
func (sm *ServiceManager) RegisterService(name string, factory container.ServiceFactory, scope container.ServiceScope, dependencies ...string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 在容器中注册服务
	if err := sm.container.Register(name, factory, scope, dependencies...); err != nil {
		return fmt.Errorf("failed to register service in container: %w", err)
	}

	// 记录服务信息
	sm.services[name] = ServiceInfo{
		Name:      name,
		Status:    "registered",
		Config:    make(map[string]interface{}),
		StartedAt: time.Time{},
	}

	sm.logger.Info(fmt.Sprintf("服务已注册: %s", name))
	return nil
}

// RegisterServiceInstance 直接注册服务实例
func (sm *ServiceManager) RegisterServiceInstance(name string, instance interface{}, metadata *ServiceMetadata) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 在容器中注册实例
	if err := sm.container.RegisterInstance(name, instance); err != nil {
		return fmt.Errorf("failed to register instance in container: %w", err)
	}

	// 在注册中心注册服务
	if err := sm.registry.RegisterService(name, instance, metadata); err != nil {
		return fmt.Errorf("failed to register service in registry: %w", err)
	}

	// 记录服务信息
	sm.services[name] = ServiceInfo{
		Name:      name,
		Instance:  instance,
		Metadata:  metadata,
		Status:    "registered",
		Config:    make(map[string]interface{}),
		StartedAt: time.Time{},
	}

	sm.logger.Info(fmt.Sprintf("服务实例已注册: %s", name))
	return nil
}

// StartService 启动服务
func (sm *ServiceManager) StartService(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	info, exists := sm.services[name]
	if !exists {
		return fmt.Errorf("service %s not found", name)
	}

	if info.Status == "running" {
		return fmt.Errorf("service %s is already running", name)
	}

	info.Status = "starting"
	info.StartedAt = time.Now()
	sm.services[name] = info

	sm.logger.Info(fmt.Sprintf("正在启动服务: %s", name))

	// 从容器获取服务实例
	instance, err := sm.container.Get(name)
	if err != nil {
		info.Status = "error"
		info.Error = err.Error()
		sm.services[name] = info
		return fmt.Errorf("failed to get service instance: %w", err)
	}

	// 更新服务信息
	info.Instance = instance
	info.Status = "running"
	info.Error = ""
	sm.services[name] = info

	sm.logger.Info(fmt.Sprintf("服务启动成功: %s", name))
	return nil
}

// StopService 停止服务
func (sm *ServiceManager) StopService(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	info, exists := sm.services[name]
	if !exists {
		return fmt.Errorf("service %s not found", name)
	}

	if info.Status != "running" {
		return fmt.Errorf("service %s is not running", name)
	}

	info.Status = "stopping"
	sm.services[name] = info

	sm.logger.Info(fmt.Sprintf("正在停止服务: %s", name))

	// 如果服务实现了Disposer接口，调用Dispose方法
	if disposer, ok := info.Instance.(interfaces.Disposer); ok {
		if err := disposer.Dispose(); err != nil {
			sm.logger.Error(fmt.Sprintf("服务释放失败: %v", err))
		}
	}

	// 从注册中心注销服务
	if err := sm.registry.UnregisterService(name); err != nil {
		sm.logger.Error(fmt.Sprintf("从注册中心注销服务失败: %v", err))
	}

	info.Status = "stopped"
	info.Instance = nil
	sm.services[name] = info

	sm.logger.Info(fmt.Sprintf("服务已停止: %s", name))
	return nil
}

// GetService 获取服务实例
func (sm *ServiceManager) GetService(name string) (interface{}, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.container.Get(name)
}

// GetServiceInfo 获取服务信息
func (sm *ServiceManager) GetServiceInfo(name string) (ServiceInfo, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	info, exists := sm.services[name]
	if !exists {
		return ServiceInfo{}, fmt.Errorf("service %s not found", name)
	}

	return info, nil
}

// ListServices 列出所有服务
func (sm *ServiceManager) ListServices() map[string]ServiceInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]ServiceInfo)
	for name, info := range sm.services {
		result[name] = info
	}

	return result
}

// StartAllServices 启动所有注册的服务
func (sm *ServiceManager) StartAllServices() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.started {
		return fmt.Errorf("services already started")
	}

	sm.logger.Info("开始启动所有服务")

	var errors []string
	for name := range sm.services {
		sm.mu.Unlock() // 临时释放锁以避免死锁
		if err := sm.StartService(name); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))
		}
		sm.mu.Lock() // 重新获取锁
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to start some services: %v", errors)
	}

	sm.started = true
	sm.logger.Info("所有服务启动完成")
	return nil
}

// StopAllServices 停止所有服务
func (sm *ServiceManager) StopAllServices() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.started {
		return nil
	}

	sm.logger.Info("开始停止所有服务")

	var errors []string
	for name := range sm.services {
		sm.mu.Unlock() // 临时释放锁以避免死锁
		if err := sm.StopService(name); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))
		}
		sm.mu.Lock() // 重新获取锁
	}

	// 释放容器资源
	if err := sm.container.Dispose(); err != nil {
		errors = append(errors, fmt.Sprintf("container disposal: %v", err))
	}

	// 释放注册中心资源
	if err := sm.registry.Dispose(); err != nil {
		errors = append(errors, fmt.Sprintf("registry disposal: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop some services: %v", errors)
	}

	sm.started = false
	sm.logger.Info("所有服务已停止")
	return nil
}

// GetServiceHealth 获取服务健康状态
func (sm *ServiceManager) GetServiceHealth(name string) (ServiceHealth, error) {
	return sm.registry.GetServiceHealth(name)
}

// WatchService 监听服务事件
func (sm *ServiceManager) WatchService(serviceName string) <-chan ServiceEvent {
	return sm.registry.WatchService(serviceName)
}

// GetServiceStatistics 获取服务统计信息
func (sm *ServiceManager) GetServiceStatistics() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := sm.registry.GetServiceStatistics()

	// 添加服务管理器的统计信息
	statusCount := make(map[string]int)
	for _, info := range sm.services {
		statusCount[info.Status]++
	}

	stats["service_manager"] = map[string]interface{}{
		"total_managed_services": len(sm.services),
		"services_by_status":     statusCount,
		"manager_started":        sm.started,
	}

	return stats
}

// PerformHealthCheck 执行健康检查
func (sm *ServiceManager) PerformHealthCheck(ctx context.Context) map[string]ServiceHealth {
	sm.mu.RLock()
	services := make(map[string]interface{})
	for name, info := range sm.services {
		if info.Instance != nil {
			services[name] = info.Instance
		}
	}
	sm.mu.RUnlock()

	results := make(map[string]ServiceHealth)

	for name, instance := range services {
		if checker, ok := instance.(HealthChecker); ok {
			health := checker.CheckHealth(ctx)
			results[name] = health
			sm.registry.UpdateServiceHealth(name, health)
		} else {
			// 为没有实现健康检查的服务创建默认健康状态
			health := ServiceHealth{
				ServiceName: name,
				Status:      "healthy",
				Message:     "Service is running (no health check implemented)",
				CheckedAt:   time.Now(),
			}
			results[name] = health
		}
	}

	return results
}

// Dispose 释放服务管理器资源
func (sm *ServiceManager) Dispose() error {
	return sm.StopAllServices()
}

// RestartService 重启服务
func (sm *ServiceManager) RestartService(name string) error {
	if err := sm.StopService(name); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// 等待一小段时间确保清理完成
	time.Sleep(100 * time.Millisecond)

	if err := sm.StartService(name); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	sm.logger.Info(fmt.Sprintf("服务重启成功: %s", name))
	return nil
}

// IsServiceRunning 检查服务是否正在运行
func (sm *ServiceManager) IsServiceRunning(name string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	info, exists := sm.services[name]
	return exists && info.Status == "running"
}

// GetServiceDependencies 获取服务依赖图
func (sm *ServiceManager) GetServiceDependencies() map[string][]string {
	return sm.container.BuildServiceGraph()
}
