/*
* @Author: Lzww0608
* @Date: 2025-6-9 15:42:10
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-09 15:42:10
* @Description: 容器实现
 */

package container

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/Lzww0608/ClixGo/pkg/interfaces"
)

// ServiceScope 服务作用域
type ServiceScope string

const (
	// Singleton 单例模式 - 整个容器生命周期中只创建一次
	Singleton ServiceScope = "singleton"

	// Transient 瞬态模式 - 每次请求都创建新实例
	Transient ServiceScope = "transient"

	// Scoped 作用域模式 - 在特定作用域内是单例
	Scoped ServiceScope = "scoped"
)

// ServiceFactory 服务工厂函数类型
type ServiceFactory func(c *Container) (interface{}, error)

// ServiceDescriptor 服务描述符
type ServiceDescriptor struct {
	Name         string         // 服务名称
	Type         reflect.Type   // 服务类型
	Factory      ServiceFactory // 工厂函数
	Scope        ServiceScope   // 作用域
	Instance     interface{}    // 缓存的实例(用于单例)
	Dependencies []string       // 依赖列表
}

// Container 依赖注入容器
type Container struct {
	services  map[string]*ServiceDescriptor
	instances map[string]interface{}
	mu        sync.RWMutex
	parent    *Container                        // 父容器(用于分层容器)
	scopes    map[string]map[string]interface{} // 作用域实例缓存
}

// NewContainer 创建新的容器
func NewContainer() *Container {
	return &Container{
		services:  make(map[string]*ServiceDescriptor),
		instances: make(map[string]interface{}),
		scopes:    make(map[string]map[string]interface{}),
	}
}

// NewChildContainer 创建子容器
func (c *Container) NewChildContainer() *Container {
	child := NewContainer()
	child.parent = c
	return child
}

// Register 注册服务
func (c *Container) Register(name string, factory ServiceFactory, scope ServiceScope, dependencies ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.services[name]; exists {
		return fmt.Errorf("service %s already registered", name)
	}

	c.services[name] = &ServiceDescriptor{
		Name:         name,
		Factory:      factory,
		Scope:        scope,
		Dependencies: dependencies,
	}

	return nil
}

// RegisterSingleton 注册单例服务
func (c *Container) RegisterSingleton(name string, factory ServiceFactory, dependencies ...string) error {
	return c.Register(name, factory, Singleton, dependencies...)
}

// RegisterTransient 注册瞬态服务
func (c *Container) RegisterTransient(name string, factory ServiceFactory, dependencies ...string) error {
	return c.Register(name, factory, Transient, dependencies...)
}

// RegisterInstance 注册实例
func (c *Container) RegisterInstance(name string, instance interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.services[name] = &ServiceDescriptor{
		Name:     name,
		Type:     reflect.TypeOf(instance),
		Scope:    Singleton,
		Instance: instance,
	}

	c.instances[name] = instance
	return nil
}

// Get 获取服务实例
func (c *Container) Get(name string) (interface{}, error) {
	return c.GetWithScope(name, "")
}

// GetWithScope 在指定作用域内获取服务实例
func (c *Container) GetWithScope(name string, scopeID string) (interface{}, error) {
	c.mu.RLock()
	descriptor, exists := c.services[name]
	if !exists && c.parent != nil {
		c.mu.RUnlock()
		return c.parent.GetWithScope(name, scopeID)
	}
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	// 检查循环依赖
	if err := c.checkCircularDependency(name, make(map[string]bool)); err != nil {
		return nil, err
	}

	return c.createInstance(descriptor, scopeID)
}

// createInstance 创建服务实例
func (c *Container) createInstance(descriptor *ServiceDescriptor, scopeID string) (interface{}, error) {
	switch descriptor.Scope {
	case Singleton:
		return c.getSingletonInstance(descriptor)
	case Scoped:
		return c.getScopedInstance(descriptor, scopeID)
	case Transient:
		return c.createTransientInstance(descriptor)
	default:
		return nil, fmt.Errorf("unknown service scope: %s", descriptor.Scope)
	}
}

// getSingletonInstance 获取单例实例
func (c *Container) getSingletonInstance(descriptor *ServiceDescriptor) (interface{}, error) {
	// 先检查是否已存在实例
	c.mu.RLock()
	if descriptor.Instance != nil {
		instance := descriptor.Instance
		c.mu.RUnlock()
		return instance, nil
	}

	if instance, exists := c.instances[descriptor.Name]; exists {
		c.mu.RUnlock()
		return instance, nil
	}
	c.mu.RUnlock()

	// 释放锁后创建实例，避免死锁
	instance, err := descriptor.Factory(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create singleton instance for %s: %w", descriptor.Name, err)
	}

	// 重新获取写锁来存储实例
	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查，防止并发创建
	if descriptor.Instance != nil {
		return descriptor.Instance, nil
	}

	if existingInstance, exists := c.instances[descriptor.Name]; exists {
		return existingInstance, nil
	}

	descriptor.Instance = instance
	c.instances[descriptor.Name] = instance
	return instance, nil
}

// getScopedInstance 获取作用域实例
func (c *Container) getScopedInstance(descriptor *ServiceDescriptor, scopeID string) (interface{}, error) {
	if scopeID == "" {
		scopeID = "default"
	}

	// 先检查是否已存在实例
	c.mu.RLock()
	if scopeInstances, exists := c.scopes[scopeID]; exists {
		if instance, exists := scopeInstances[descriptor.Name]; exists {
			c.mu.RUnlock()
			return instance, nil
		}
	}
	c.mu.RUnlock()

	// 释放锁后创建实例，避免死锁
	instance, err := descriptor.Factory(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create scoped instance for %s: %w", descriptor.Name, err)
	}

	// 重新获取写锁来存储实例
	c.mu.Lock()
	defer c.mu.Unlock()

	// 确保作用域存在
	if _, exists := c.scopes[scopeID]; !exists {
		c.scopes[scopeID] = make(map[string]interface{})
	}

	// 双重检查，防止并发创建
	if existingInstance, exists := c.scopes[scopeID][descriptor.Name]; exists {
		return existingInstance, nil
	}

	c.scopes[scopeID][descriptor.Name] = instance
	return instance, nil
}

// createTransientInstance 创建瞬态实例
func (c *Container) createTransientInstance(descriptor *ServiceDescriptor) (interface{}, error) {
	return descriptor.Factory(c)
}

// checkCircularDependency 检查循环依赖
func (c *Container) checkCircularDependency(serviceName string, visited map[string]bool) error {
	if visited[serviceName] {
		return fmt.Errorf("circular dependency detected for service: %s", serviceName)
	}

	c.mu.RLock()
	descriptor, exists := c.services[serviceName]
	c.mu.RUnlock()

	if !exists {
		return nil
	}

	visited[serviceName] = true

	for _, dep := range descriptor.Dependencies {
		if err := c.checkCircularDependency(dep, visited); err != nil {
			return err
		}
	}

	delete(visited, serviceName)
	return nil
}

// Dispose 释放容器资源
func (c *Container) Dispose() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 释放所有实现了Disposer接口的服务
	for name, instance := range c.instances {
		if disposer, ok := instance.(interfaces.Disposer); ok {
			if err := disposer.Dispose(); err != nil {
				// 记录错误但继续释放其他资源
				fmt.Printf("Error disposing service %s: %v\n", name, err)
			}
		}
	}

	// 清理作用域实例
	for scopeID, scopeInstances := range c.scopes {
		for name, instance := range scopeInstances {
			if disposer, ok := instance.(interfaces.Disposer); ok {
				if err := disposer.Dispose(); err != nil {
					fmt.Printf("Error disposing scoped service %s in scope %s: %v\n", name, scopeID, err)
				}
			}
		}
	}

	// 清空缓存
	c.services = make(map[string]*ServiceDescriptor)
	c.instances = make(map[string]interface{})
	c.scopes = make(map[string]map[string]interface{})

	return nil
}

// ClearScope 清理指定作用域
func (c *Container) ClearScope(scopeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if scopeInstances, exists := c.scopes[scopeID]; exists {
		for name, instance := range scopeInstances {
			if disposer, ok := instance.(interfaces.Disposer); ok {
				if err := disposer.Dispose(); err != nil {
					fmt.Printf("Error disposing scoped service %s in scope %s: %v\n", name, scopeID, err)
				}
			}
		}
		delete(c.scopes, scopeID)
	}

	return nil
}

// GetServiceNames 获取所有已注册的服务名称
func (c *Container) GetServiceNames() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.services))
	for name := range c.services {
		names = append(names, name)
	}

	return names
}

// HasService 检查服务是否已注册
func (c *Container) HasService(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.services[name]
	if !exists && c.parent != nil {
		return c.parent.HasService(name)
	}

	return exists
}

// BuildServiceGraph 构建服务依赖图
func (c *Container) BuildServiceGraph() map[string][]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	graph := make(map[string][]string)
	for name, descriptor := range c.services {
		graph[name] = append([]string{}, descriptor.Dependencies...)
	}

	return graph
}

// ValidateContainer 验证容器配置
func (c *Container) ValidateContainer() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 检查所有依赖是否都已注册
	for serviceName, descriptor := range c.services {
		for _, dep := range descriptor.Dependencies {
			if !c.hasServiceInChain(dep) {
				return fmt.Errorf("service %s depends on unregistered service %s", serviceName, dep)
			}
		}

		// 检查循环依赖
		if err := c.checkCircularDependency(serviceName, make(map[string]bool)); err != nil {
			return err
		}
	}

	return nil
}

// hasServiceInChain 检查服务是否在容器链中存在
func (c *Container) hasServiceInChain(name string) bool {
	current := c
	for current != nil {
		if _, exists := current.services[name]; exists {
			return true
		}
		current = current.parent
	}
	return false
}
