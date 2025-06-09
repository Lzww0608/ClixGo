/*
* @Author: Lzww0608
* @Date: 2025-6-9 15:42:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-09 15:42:00
* @Description: 容器辅助函数
 */

package container

import (
	"fmt"

	"github.com/Lzww0608/ClixGo/pkg/interfaces"
)

// 类型安全的服务获取函数

// GetLogger 获取日志服务
func GetLogger(c *Container) (interfaces.Logger, error) {
	service, err := c.Get("logger")
	if err != nil {
		return nil, err
	}

	if logger, ok := service.(interfaces.Logger); ok {
		return logger, nil
	}
	return nil, fmt.Errorf("service logger is not of expected type")
}

// GetErrorHandler 获取错误处理服务
func GetErrorHandler(c *Container) (interfaces.ErrorHandler, error) {
	service, err := c.Get("error_handler")
	if err != nil {
		return nil, err
	}

	if handler, ok := service.(interfaces.ErrorHandler); ok {
		return handler, nil
	}
	return nil, fmt.Errorf("service error_handler is not of expected type")
}

// GetTerminalService 获取终端服务
func GetTerminalService(c *Container) (interfaces.TerminalService, error) {
	service, err := c.Get("terminal_service")
	if err != nil {
		return nil, err
	}

	if terminalService, ok := service.(interfaces.TerminalService); ok {
		return terminalService, nil
	}
	return nil, fmt.Errorf("service terminal_service is not of expected type")
}

// GetNetworkService 获取网络服务
func GetNetworkService(c *Container) (interfaces.NetworkService, error) {
	service, err := c.Get("network_service")
	if err != nil {
		return nil, err
	}

	if networkService, ok := service.(interfaces.NetworkService); ok {
		return networkService, nil
	}
	return nil, fmt.Errorf("service network_service is not of expected type")
}

// GetPerformanceService 获取性能监控服务
func GetPerformanceService(c *Container) (interfaces.PerformanceService, error) {
	service, err := c.Get("performance_service")
	if err != nil {
		return nil, err
	}

	if performanceService, ok := service.(interfaces.PerformanceService); ok {
		return performanceService, nil
	}
	return nil, fmt.Errorf("service performance_service is not of expected type")
}

// GetTaskService 获取任务服务
func GetTaskService(c *Container) (interfaces.TaskService, error) {
	service, err := c.Get("task_service")
	if err != nil {
		return nil, err
	}

	if taskService, ok := service.(interfaces.TaskService); ok {
		return taskService, nil
	}
	return nil, fmt.Errorf("service task_service is not of expected type")
}

// MustGetLogger 获取日志服务 (panic on error)
func MustGetLogger(c *Container) interfaces.Logger {
	logger, err := GetLogger(c)
	if err != nil {
		panic(fmt.Sprintf("Failed to get logger service: %v", err))
	}
	return logger
}

// MustGetErrorHandler 获取错误处理服务 (panic on error)
func MustGetErrorHandler(c *Container) interfaces.ErrorHandler {
	handler, err := GetErrorHandler(c)
	if err != nil {
		panic(fmt.Sprintf("Failed to get error handler service: %v", err))
	}
	return handler
}

// MustGetTerminalService 获取终端服务 (panic on error)
func MustGetTerminalService(c *Container) interfaces.TerminalService {
	service, err := GetTerminalService(c)
	if err != nil {
		panic(fmt.Sprintf("Failed to get terminal service: %v", err))
	}
	return service
}

// MustGetNetworkService 获取网络服务 (panic on error)
func MustGetNetworkService(c *Container) interfaces.NetworkService {
	service, err := GetNetworkService(c)
	if err != nil {
		panic(fmt.Sprintf("Failed to get network service: %v", err))
	}
	return service
}

// MustGetPerformanceService 获取性能监控服务 (panic on error)
func MustGetPerformanceService(c *Container) interfaces.PerformanceService {
	service, err := GetPerformanceService(c)
	if err != nil {
		panic(fmt.Sprintf("Failed to get performance service: %v", err))
	}
	return service
}

// MustGetTaskService 获取任务服务 (panic on error)
func MustGetTaskService(c *Container) interfaces.TaskService {
	service, err := GetTaskService(c)
	if err != nil {
		panic(fmt.Sprintf("Failed to get task service: %v", err))
	}
	return service
}

// SafeGet 安全获取服务，返回error而不是panic
func SafeGet[T any](c *Container, serviceName string) (T, error) {
	var zero T

	service, err := c.Get(serviceName)
	if err != nil {
		return zero, err
	}

	if typedService, ok := service.(T); ok {
		return typedService, nil
	}

	return zero, fmt.Errorf("service %s is not of expected type %T", serviceName, zero)
}

// TryGet 尝试获取服务，如果失败返回nil
func TryGet[T any](c *Container, serviceName string) T {
	var zero T

	service, err := SafeGet[T](c, serviceName)
	if err != nil {
		return zero
	}

	return service
}

// GetOrCreate 获取或创建服务
func GetOrCreate[T any](c *Container, serviceName string, factory func() T) T {
	// 先尝试获取服务
	service, err := SafeGet[T](c, serviceName)
	if err == nil {
		return service
	}

	// 如果服务不存在，创建并注册
	newService := factory()
	err = c.RegisterInstance(serviceName, newService)
	if err != nil {
		// 如果注册失败，直接返回创建的实例
		return newService
	}

	return newService
}
