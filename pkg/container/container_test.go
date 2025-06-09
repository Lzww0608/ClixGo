/*
* @Author: Lzww0608
* @Date: 2025-6-9 15:41:55
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-09 15:41:55
* @Description: 容器测试
 */

package container

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试用接口和实现
type TestService interface {
	GetMessage() string
}

type testServiceImpl struct {
	message string
}

func (t *testServiceImpl) GetMessage() string {
	return t.message
}

func (t *testServiceImpl) Dispose() error {
	// 实现Disposer接口
	return nil
}

type TestDependentService interface {
	GetServiceMessage() string
}

type testDependentServiceImpl struct {
	testService TestService
}

func (t *testDependentServiceImpl) GetServiceMessage() string {
	return "Dependent: " + t.testService.GetMessage()
}

func TestContainer_RegisterAndGet(t *testing.T) {
	container := NewContainer()

	// 注册服务
	err := container.RegisterSingleton("test_service", func(c *Container) (interface{}, error) {
		return &testServiceImpl{message: "Hello World"}, nil
	})
	require.NoError(t, err)

	// 获取服务
	service, err := container.Get("test_service")
	require.NoError(t, err)
	assert.NotNil(t, service)

	// 类型断言并测试功能
	testService, ok := service.(TestService)
	assert.True(t, ok)
	assert.Equal(t, "Hello World", testService.GetMessage())
}

func TestContainer_Singleton(t *testing.T) {
	container := NewContainer()

	counter := 0
	err := container.RegisterSingleton("counter", func(c *Container) (interface{}, error) {
		counter++
		return &testServiceImpl{message: fmt.Sprintf("Instance %d", counter)}, nil
	})
	require.NoError(t, err)

	// 多次获取应该返回同一个实例
	service1, err := container.Get("counter")
	require.NoError(t, err)

	service2, err := container.Get("counter")
	require.NoError(t, err)

	// 应该是同一个实例
	assert.Same(t, service1, service2)
	assert.Equal(t, 1, counter) // 只创建了一次
}

func TestContainer_Transient(t *testing.T) {
	container := NewContainer()

	counter := 0
	err := container.RegisterTransient("counter", func(c *Container) (interface{}, error) {
		counter++
		return &testServiceImpl{message: fmt.Sprintf("Instance %d", counter)}, nil
	})
	require.NoError(t, err)

	// 多次获取应该返回不同的实例
	service1, err := container.Get("counter")
	require.NoError(t, err)

	service2, err := container.Get("counter")
	require.NoError(t, err)

	// 应该是不同的实例
	assert.NotSame(t, service1, service2)
	assert.Equal(t, 2, counter) // 创建了两次
}

func TestContainer_DependencyInjection(t *testing.T) {
	container := NewContainer()

	// 注册基础服务（无依赖）
	err := container.RegisterSingleton("test_service", func(c *Container) (interface{}, error) {
		return &testServiceImpl{message: "Base Service"}, nil
	})
	require.NoError(t, err)

	// 注册依赖服务（依赖test_service）
	err = container.RegisterSingleton("dependent_service", func(c *Container) (interface{}, error) {
		testService, err := SafeGet[TestService](c, "test_service")
		if err != nil {
			return nil, err
		}
		return &testDependentServiceImpl{testService: testService}, nil
	}, "test_service")
	require.NoError(t, err)

	// 获取依赖服务
	service, err := container.Get("dependent_service")
	require.NoError(t, err)

	dependentService, ok := service.(TestDependentService)
	assert.True(t, ok)
	assert.Equal(t, "Dependent: Base Service", dependentService.GetServiceMessage())
}

func TestContainer_CircularDependencyDetection(t *testing.T) {
	container := NewContainer()

	// 注册相互依赖的服务
	err := container.RegisterSingleton("service_a", func(c *Container) (interface{}, error) {
		// 依赖service_b
		_, err := c.Get("service_b")
		return &testServiceImpl{message: "Service A"}, err
	}, "service_b")
	require.NoError(t, err)

	err = container.RegisterSingleton("service_b", func(c *Container) (interface{}, error) {
		// 依赖service_a
		_, err := c.Get("service_a")
		return &testServiceImpl{message: "Service B"}, err
	}, "service_a")
	require.NoError(t, err)

	// 尝试获取服务应该检测到循环依赖
	_, err = container.Get("service_a")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestContainer_Scoped(t *testing.T) {
	container := NewContainer()

	counter := 0
	err := container.Register("scoped_service", func(c *Container) (interface{}, error) {
		counter++
		return &testServiceImpl{message: fmt.Sprintf("Scoped Instance %d", counter)}, nil
	}, Scoped)
	require.NoError(t, err)

	// 在同一作用域内获取
	service1, err := container.GetWithScope("scoped_service", "scope1")
	require.NoError(t, err)

	service2, err := container.GetWithScope("scoped_service", "scope1")
	require.NoError(t, err)

	// 在同一作用域内应该是同一个实例
	assert.Same(t, service1, service2)

	// 在不同作用域内获取
	service3, err := container.GetWithScope("scoped_service", "scope2")
	require.NoError(t, err)

	// 在不同作用域内应该是不同的实例
	assert.NotSame(t, service1, service3)
	assert.Equal(t, 2, counter) // 创建了两次（两个作用域）
}

func TestContainer_ValidateContainer(t *testing.T) {
	container := NewContainer()

	// 注册一个依赖不存在服务的服务
	err := container.RegisterSingleton("invalid_service", func(c *Container) (interface{}, error) {
		return &testServiceImpl{message: "Invalid"}, nil
	}, "non_existent_service")
	require.NoError(t, err)

	// 验证容器应该失败
	err = container.ValidateContainer()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "depends on unregistered service")
}

func TestContainer_ChildContainer(t *testing.T) {
	parent := NewContainer()
	child := parent.NewChildContainer()

	// 在父容器中注册服务
	err := parent.RegisterSingleton("parent_service", func(c *Container) (interface{}, error) {
		return &testServiceImpl{message: "Parent Service"}, nil
	})
	require.NoError(t, err)

	// 在子容器中注册服务
	err = child.RegisterSingleton("child_service", func(c *Container) (interface{}, error) {
		return &testServiceImpl{message: "Child Service"}, nil
	})
	require.NoError(t, err)

	// 子容器应该能访问父容器的服务
	service, err := child.Get("parent_service")
	require.NoError(t, err)
	testService, ok := service.(TestService)
	assert.True(t, ok)
	assert.Equal(t, "Parent Service", testService.GetMessage())

	// 父容器不应该能访问子容器的服务
	_, err = parent.Get("child_service")
	assert.Error(t, err)
}

func TestContainer_Dispose(t *testing.T) {
	container := NewContainer()

	// 注册实现Disposer接口的服务
	err := container.RegisterSingleton("disposable_service", func(c *Container) (interface{}, error) {
		return &testServiceImpl{message: "Disposable"}, nil
	})
	require.NoError(t, err)

	// 获取服务以确保它被创建
	_, err = container.Get("disposable_service")
	require.NoError(t, err)

	// 释放容器
	err = container.Dispose()
	assert.NoError(t, err)

	// 容器应该被清空
	assert.Empty(t, container.GetServiceNames())
}

func TestContainer_TypeSafeHelpers(t *testing.T) {
	container := NewContainer()

	// 注册服务
	err := container.RegisterSingleton("test_service", func(c *Container) (interface{}, error) {
		return &testServiceImpl{message: "Type Safe Test"}, nil
	})
	require.NoError(t, err)

	// 使用类型安全的获取方法
	service, err := SafeGet[TestService](container, "test_service")
	require.NoError(t, err)
	assert.Equal(t, "Type Safe Test", service.GetMessage())

	// 测试错误的类型
	_, err = SafeGet[TestDependentService](container, "test_service")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not of expected type")
}

func TestContainer_ServiceNames(t *testing.T) {
	container := NewContainer()

	// 注册多个服务
	services := []string{"service1", "service2", "service3"}
	for _, name := range services {
		err := container.RegisterSingleton(name, func(c *Container) (interface{}, error) {
			return &testServiceImpl{message: name}, nil
		})
		require.NoError(t, err)
	}

	// 获取服务名称列表
	names := container.GetServiceNames()
	assert.Len(t, names, 3)

	for _, name := range services {
		assert.Contains(t, names, name)
		assert.True(t, container.HasService(name))
	}

	assert.False(t, container.HasService("non_existent"))
}

// 基准测试
func BenchmarkContainer_GetSingleton(b *testing.B) {
	container := NewContainer()

	err := container.RegisterSingleton("benchmark_service", func(c *Container) (interface{}, error) {
		return &testServiceImpl{message: "Benchmark"}, nil
	})
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := container.Get("benchmark_service")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkContainer_GetTransient(b *testing.B) {
	container := NewContainer()

	err := container.RegisterTransient("benchmark_service", func(c *Container) (interface{}, error) {
		return &testServiceImpl{message: "Benchmark"}, nil
	})
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := container.Get("benchmark_service")
		if err != nil {
			b.Fatal(err)
		}
	}
}
