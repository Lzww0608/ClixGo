/*
* @Author: Lzww0608
* @Date: 2025-6-10 10:38:56
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-10 10:38:59
* @Description: 服务注册中心 - 实现服务发现和管理
 */

package services

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/interfaces"
)

// ServiceMetadata 服务元数据
type ServiceMetadata struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	Interface    reflect.Type           `json:"-"`
	Tags         []string               `json:"tags"`
	Properties   map[string]interface{} `json:"properties"`
	RegisteredAt time.Time              `json:"registered_at"`
	LastAccessed time.Time              `json:"last_accessed"`
	AccessCount  int64                  `json:"access_count"`
}

// ServiceHealth 服务健康状态
type ServiceHealth struct {
	ServiceName string    `json:"service_name"`
	Status      string    `json:"status"` // healthy, unhealthy, unknown
	Message     string    `json:"message"`
	CheckedAt   time.Time `json:"checked_at"`
}

// HealthChecker 健康检查器接口
type HealthChecker interface {
	CheckHealth(ctx context.Context) ServiceHealth
}

// ServiceRegistry 服务注册中心
type ServiceRegistry struct {
	services     map[string]*ServiceMetadata
	instances    map[string]interface{}
	healthStatus map[string]ServiceHealth
	watchers     map[string][]chan ServiceEvent
	mu           sync.RWMutex
	logger       interfaces.Logger
	ctx          context.Context
	cancel       context.CancelFunc
}

// ServiceEvent 服务事件
type ServiceEvent struct {
	Type        string      `json:"type"` // registered, unregistered, health_changed
	ServiceName string      `json:"service_name"`
	Metadata    interface{} `json:"metadata"`
	Timestamp   time.Time   `json:"timestamp"`
}

// NewServiceRegistry 创建服务注册中心
func NewServiceRegistry(logger interfaces.Logger) *ServiceRegistry {
	ctx, cancel := context.WithCancel(context.Background())

	registry := &ServiceRegistry{
		services:     make(map[string]*ServiceMetadata),
		instances:    make(map[string]interface{}),
		healthStatus: make(map[string]ServiceHealth),
		watchers:     make(map[string][]chan ServiceEvent),
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
	}

	// 启动健康检查协程
	go registry.startHealthCheck()

	return registry
}

// RegisterService 注册服务
func (r *ServiceRegistry) RegisterService(name string, instance interface{}, metadata *ServiceMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[name]; exists {
		return fmt.Errorf("service %s already registered", name)
	}

	// 设置默认元数据
	if metadata == nil {
		metadata = &ServiceMetadata{
			Name:        name,
			Version:     "1.0.0",
			Description: fmt.Sprintf("Service %s", name),
			Tags:        []string{},
			Properties:  make(map[string]interface{}),
		}
	}

	metadata.Interface = reflect.TypeOf(instance)
	metadata.RegisteredAt = time.Now()
	metadata.LastAccessed = time.Now()

	r.services[name] = metadata
	r.instances[name] = instance

	// 初始化健康状态
	r.healthStatus[name] = ServiceHealth{
		ServiceName: name,
		Status:      "healthy",
		Message:     "Service registered successfully",
		CheckedAt:   time.Now(),
	}

	r.logger.Info(fmt.Sprintf("服务已注册: %s", name))

	// 发送注册事件
	r.notifyWatchers(name, ServiceEvent{
		Type:        "registered",
		ServiceName: name,
		Metadata:    metadata,
		Timestamp:   time.Now(),
	})

	return nil
}

// UnregisterService 注销服务
func (r *ServiceRegistry) UnregisterService(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[name]; !exists {
		return fmt.Errorf("service %s not found", name)
	}

	delete(r.services, name)
	delete(r.instances, name)
	delete(r.healthStatus, name)

	r.logger.Info(fmt.Sprintf("服务已注销: %s", name))

	// 发送注销事件
	r.notifyWatchers(name, ServiceEvent{
		Type:        "unregistered",
		ServiceName: name,
		Timestamp:   time.Now(),
	})

	return nil
}

// GetService 获取服务实例
func (r *ServiceRegistry) GetService(name string) (interface{}, *ServiceMetadata, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	metadata, exists := r.services[name]
	if !exists {
		return nil, nil, fmt.Errorf("service %s not found", name)
	}

	instance, exists := r.instances[name]
	if !exists {
		return nil, nil, fmt.Errorf("service %s instance not found", name)
	}

	// 更新访问统计
	metadata.LastAccessed = time.Now()
	metadata.AccessCount++

	return instance, metadata, nil
}

// ListServices 列出所有服务
func (r *ServiceRegistry) ListServices() map[string]*ServiceMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*ServiceMetadata)
	for name, metadata := range r.services {
		// 深拷贝避免外部修改
		copied := *metadata
		result[name] = &copied
	}

	return result
}

// FindServicesByTag 根据标签查找服务
func (r *ServiceRegistry) FindServicesByTag(tag string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var services []string
	for name, metadata := range r.services {
		for _, t := range metadata.Tags {
			if t == tag {
				services = append(services, name)
				break
			}
		}
	}

	return services
}

// WatchService 监听服务事件
func (r *ServiceRegistry) WatchService(serviceName string) <-chan ServiceEvent {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch := make(chan ServiceEvent, 10)
	r.watchers[serviceName] = append(r.watchers[serviceName], ch)

	return ch
}

// GetServiceHealth 获取服务健康状态
func (r *ServiceRegistry) GetServiceHealth(name string) (ServiceHealth, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	health, exists := r.healthStatus[name]
	if !exists {
		return ServiceHealth{}, fmt.Errorf("service %s not found", name)
	}

	return health, nil
}

// UpdateServiceHealth 更新服务健康状态
func (r *ServiceRegistry) UpdateServiceHealth(name string, health ServiceHealth) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[name]; !exists {
		return fmt.Errorf("service %s not found", name)
	}

	oldHealth := r.healthStatus[name]
	r.healthStatus[name] = health

	// 如果状态发生变化，发送事件
	if oldHealth.Status != health.Status {
		r.notifyWatchers(name, ServiceEvent{
			Type:        "health_changed",
			ServiceName: name,
			Metadata:    health,
			Timestamp:   time.Now(),
		})
	}

	return nil
}

// GetServiceStatistics 获取服务统计信息
func (r *ServiceRegistry) GetServiceStatistics() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := map[string]interface{}{
		"total_services":     len(r.services),
		"healthy_services":   0,
		"unhealthy_services": 0,
		"services_by_tag":    make(map[string]int),
	}

	// 统计健康状态
	for _, health := range r.healthStatus {
		if health.Status == "healthy" {
			stats["healthy_services"] = stats["healthy_services"].(int) + 1
		} else {
			stats["unhealthy_services"] = stats["unhealthy_services"].(int) + 1
		}
	}

	// 统计标签分布
	tagCount := make(map[string]int)
	for _, metadata := range r.services {
		for _, tag := range metadata.Tags {
			tagCount[tag]++
		}
	}
	stats["services_by_tag"] = tagCount

	return stats
}

// Dispose 释放资源
func (r *ServiceRegistry) Dispose() error {
	r.cancel()

	r.mu.Lock()
	defer r.mu.Unlock()

	// 关闭所有监听器
	for _, watchers := range r.watchers {
		for _, ch := range watchers {
			close(ch)
		}
	}

	// 清理所有数据
	r.services = make(map[string]*ServiceMetadata)
	r.instances = make(map[string]interface{})
	r.healthStatus = make(map[string]ServiceHealth)
	r.watchers = make(map[string][]chan ServiceEvent)

	r.logger.Info("服务注册中心已关闭")
	return nil
}

// startHealthCheck 启动健康检查
func (r *ServiceRegistry) startHealthCheck() {
	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (r *ServiceRegistry) performHealthCheck() {
	r.mu.RLock()
	services := make(map[string]interface{})
	for name, instance := range r.instances {
		services[name] = instance
	}
	r.mu.RUnlock()

	for name, instance := range services {
		if checker, ok := instance.(HealthChecker); ok {
			health := checker.CheckHealth(r.ctx)
			r.UpdateServiceHealth(name, health)
		}
	}
}

// notifyWatchers 通知监听器
func (r *ServiceRegistry) notifyWatchers(serviceName string, event ServiceEvent) {
	if watchers, exists := r.watchers[serviceName]; exists {
		for _, ch := range watchers {
			select {
			case ch <- event:
			default:
				// 如果通道满了，跳过这个监听器
			}
		}
	}

	// 也通知通配符监听器
	if watchers, exists := r.watchers["*"]; exists {
		for _, ch := range watchers {
			select {
			case ch <- event:
			default:
			}
		}
	}
}
