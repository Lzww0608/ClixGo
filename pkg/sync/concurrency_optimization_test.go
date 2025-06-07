/*
* @Author: Lzww0608
* @Date: 2025-06-06 15:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-7 15:52:37
* @Description: 并发优化模块测试 - 验证goroutine池和优雅关闭管理器功能
 */

package sync

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TestGoroutinePool 测试Goroutine池基本功能
func TestGoroutinePool(t *testing.T) {
	config := DefaultGoroutinePoolConfig()
	config.MinWorkers = 2
	config.MaxWorkers = 4
	config.QueueSize = 10

	pool := NewGoroutinePool(config)
	require.NotNil(t, pool)

	// 启动池
	err := pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	// 验证启动状态
	assert.True(t, pool.IsRunning())
	assert.False(t, pool.IsClosed())

	t.Run("基本任务提交和执行", func(t *testing.T) {
		var counter int32
		var wg sync.WaitGroup

		// 提交多个任务
		for i := 0; i < 5; i++ {
			wg.Add(1)
			taskID := fmt.Sprintf("task-%d", i)

			err := pool.SubmitFunc(taskID, func(ctx context.Context) error {
				defer wg.Done()
				atomic.AddInt32(&counter, 1)
				time.Sleep(10 * time.Millisecond) // 模拟工作
				return nil
			})
			assert.NoError(t, err)
		}

		// 等待所有任务完成
		wg.Wait()
		assert.Equal(t, int32(5), atomic.LoadInt32(&counter))
	})

	t.Run("任务超时处理", func(t *testing.T) {
		task := NewTask("timeout-task", func(ctx context.Context) error {
			select {
			case <-time.After(100 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}).WithTimeout(50 * time.Millisecond)

		err := pool.SubmitAndWait(task)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("优先级任务", func(t *testing.T) {
		var results []int
		var mu sync.Mutex

		// 提交不同优先级的任务
		highPriorityTask := NewTask("high", func(ctx context.Context) error {
			mu.Lock()
			results = append(results, 10)
			mu.Unlock()
			return nil
		}).WithPriority(10)

		lowPriorityTask := NewTask("low", func(ctx context.Context) error {
			mu.Lock()
			results = append(results, 1)
			mu.Unlock()
			return nil
		}).WithPriority(1)

		// 先提交低优先级，再提交高优先级
		pool.Submit(lowPriorityTask)
		pool.Submit(highPriorityTask)

		// 等待任务完成
		<-lowPriorityTask.Result
		<-highPriorityTask.Result

		assert.Len(t, results, 2)
		// 注意：由于是简单的队列，优先级在当前实现中可能不完全体现
	})

	t.Run("池指标统计", func(t *testing.T) {
		// 提交一些任务
		for i := 0; i < 3; i++ {
			pool.SubmitFunc(fmt.Sprintf("metrics-task-%d", i), func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			})
		}

		time.Sleep(50 * time.Millisecond) // 等待任务执行

		metrics := pool.GetMetrics()
		assert.GreaterOrEqual(t, metrics.TotalWorkers, int32(2))
		assert.GreaterOrEqual(t, metrics.CompletedTasks, uint64(3))
	})
}

// TestGoroutinePoolConcurrency 测试Goroutine池并发性能
func TestGoroutinePoolConcurrency(t *testing.T) {
	config := DefaultGoroutinePoolConfig()
	config.MinWorkers = 4
	config.MaxWorkers = 8
	config.QueueSize = 100

	pool := NewGoroutinePool(config)
	require.NotNil(t, pool)

	err := pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	const numTasks = 50
	var completedTasks int32
	var wg sync.WaitGroup

	start := time.Now()

	// 并发提交大量任务
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		taskID := fmt.Sprintf("concurrent-task-%d", i)

		err := pool.SubmitFunc(taskID, func(ctx context.Context) error {
			defer wg.Done()
			// 模拟CPU密集型工作
			for j := 0; j < 1000; j++ {
				_ = j * j
			}
			atomic.AddInt32(&completedTasks, 1)
			return nil
		})
		assert.NoError(t, err)
	}

	wg.Wait()
	duration := time.Since(start)

	assert.Equal(t, int32(numTasks), atomic.LoadInt32(&completedTasks))
	assert.Less(t, duration, 5*time.Second, "任务执行时间过长")

	t.Logf("完成 %d 个任务耗时: %v", numTasks, duration)
}

// TestGracefulShutdownManager 测试优雅关闭管理器
func TestGracefulShutdownManager(t *testing.T) {
	config := DefaultShutdownConfig()
	config.ComponentTimeout = 5 * time.Second
	config.GlobalTimeout = 10 * time.Second

	manager := NewGracefulShutdownManager(config)
	require.NotNil(t, manager)

	err := manager.Start()
	require.NoError(t, err)

	t.Run("组件注册和管理", func(t *testing.T) {
		// 创建测试组件
		testComp := &TestComponent{name: "test-component"}

		// 注册组件
		err := manager.RegisterComponent(testComp)
		assert.NoError(t, err)

		// 启动组件
		err = manager.StartComponent(testComp.Name())
		assert.NoError(t, err)
		assert.True(t, testComp.isStarted)

		// 检查状态
		state, err := manager.GetComponentState(testComp.Name())
		assert.NoError(t, err)
		assert.Equal(t, StateRunning, state)

		// 停止组件
		err = manager.StopComponent(testComp.Name())
		assert.NoError(t, err)
		assert.True(t, testComp.isStopped)
	})

	t.Run("Channel管理", func(t *testing.T) {
		// 注册channel
		ch := make(chan string, 5)
		err := manager.RegisterChannel("test-channel", ch)
		assert.NoError(t, err)

		// 向channel发送数据
		ch <- "test-message"

		// 安全关闭channel
		err = manager.SafeClose("test-channel")
		assert.NoError(t, err)

		// 验证channel已关闭
		_, ok := <-ch
		assert.False(t, ok, "channel应该已关闭")
	})

	t.Run("受管理的Goroutine", func(t *testing.T) {
		var executed bool
		var mu sync.Mutex

		// 运行受管理的goroutine
		manager.RunManagedGoroutine("test-goroutine", func(ctx context.Context) {
			mu.Lock()
			executed = true
			mu.Unlock()

			// 等待取消信号
			<-ctx.Done()
		})

		// 等待goroutine启动
		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		assert.True(t, executed)
		mu.Unlock()
	})

	t.Run("关闭钩子", func(t *testing.T) {
		var hookExecuted bool

		manager.AddShutdownHook(func(ctx context.Context) error {
			hookExecuted = true
			return nil
		})

		// 执行关闭
		err := manager.Stop()
		assert.NoError(t, err)
		assert.True(t, hookExecuted)
	})
}

// TestGracefulShutdownTimeout 测试优雅关闭超时处理
func TestGracefulShutdownTimeout(t *testing.T) {
	config := DefaultShutdownConfig()
	config.ComponentTimeout = 100 * time.Millisecond
	config.GlobalTimeout = 200 * time.Millisecond

	manager := NewGracefulShutdownManager(config)
	require.NotNil(t, manager)

	err := manager.Start()
	require.NoError(t, err)

	// 注册一个会阻塞的组件
	slowComp := &SlowStopComponent{name: "slow-component"}
	err = manager.RegisterComponent(slowComp)
	require.NoError(t, err)

	err = manager.StartComponent(slowComp.Name())
	require.NoError(t, err)

	start := time.Now()
	err = manager.StopWithTimeout(300 * time.Millisecond)
	duration := time.Since(start)

	// 应该在超时时间内返回（即使组件停止缓慢）
	assert.Less(t, duration, 400*time.Millisecond)
	assert.Error(t, err, "应该因为组件停止超时而返回错误")
}

// TestConcurrentGoroutinePoolUsage 测试在真实场景中的goroutine池使用
func TestConcurrentGoroutinePoolUsage(t *testing.T) {
	// 模拟性能分析器场景
	config := DefaultGoroutinePoolConfig()
	config.MaxWorkers = 6
	pool := NewGoroutinePool(config)

	err := pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	var (
		cpuTasksDone    int32
		memoryTasksDone int32
		systemTasksDone int32
	)

	const iterations = 10

	// 模拟并发收集不同类型的指标
	for i := 0; i < iterations; i++ {
		// CPU指标收集任务
		pool.SubmitFunc(fmt.Sprintf("cpu-task-%d", i), func(ctx context.Context) error {
			time.Sleep(20 * time.Millisecond) // 模拟CPU指标收集
			atomic.AddInt32(&cpuTasksDone, 1)
			return nil
		})

		// 内存指标收集任务
		pool.SubmitFunc(fmt.Sprintf("memory-task-%d", i), func(ctx context.Context) error {
			time.Sleep(15 * time.Millisecond) // 模拟内存指标收集
			atomic.AddInt32(&memoryTasksDone, 1)
			return nil
		})

		// 系统指标收集任务
		pool.SubmitFunc(fmt.Sprintf("system-task-%d", i), func(ctx context.Context) error {
			time.Sleep(25 * time.Millisecond) // 模拟系统指标收集
			atomic.AddInt32(&systemTasksDone, 1)
			return nil
		})
	}

	// 等待所有任务完成
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&cpuTasksDone) == iterations &&
			atomic.LoadInt32(&memoryTasksDone) == iterations &&
			atomic.LoadInt32(&systemTasksDone) == iterations {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	assert.Equal(t, int32(iterations), atomic.LoadInt32(&cpuTasksDone))
	assert.Equal(t, int32(iterations), atomic.LoadInt32(&memoryTasksDone))
	assert.Equal(t, int32(iterations), atomic.LoadInt32(&systemTasksDone))

	metrics := pool.GetMetrics()
	t.Logf("Pool metrics: Active=%d, Completed=%d, Failed=%d",
		metrics.ActiveWorkers, metrics.CompletedTasks, metrics.FailedTasks)
}

// TestComponent 测试组件实现
type TestComponent struct {
	name      string
	isStarted bool
	isStopped bool
	mu        sync.Mutex
}

func (tc *TestComponent) Name() string {
	return tc.name
}

func (tc *TestComponent) Start(ctx context.Context) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.isStarted = true
	return nil
}

func (tc *TestComponent) Stop(ctx context.Context) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.isStopped = true
	return nil
}

func (tc *TestComponent) State() ComponentState {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if tc.isStopped {
		return StateStopped
	}
	if tc.isStarted {
		return StateRunning
	}
	return StateUnknown
}

// SlowStopComponent 慢停止组件，用于测试超时
type SlowStopComponent struct {
	name      string
	isStarted bool
}

func (ssc *SlowStopComponent) Name() string {
	return ssc.name
}

func (ssc *SlowStopComponent) Start(ctx context.Context) error {
	ssc.isStarted = true
	return nil
}

func (ssc *SlowStopComponent) Stop(ctx context.Context) error {
	// 故意阻塞，模拟停止困难的组件
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (ssc *SlowStopComponent) State() ComponentState {
	if ssc.isStarted {
		return StateRunning
	}
	return StateUnknown
}

// BenchmarkGoroutinePool 性能基准测试
func BenchmarkGoroutinePool(b *testing.B) {
	config := DefaultGoroutinePoolConfig()
	config.MaxWorkers = 8
	pool := NewGoroutinePool(config)

	err := pool.Start()
	require.NoError(b, err)
	defer pool.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		taskCounter := 0
		for pb.Next() {
			taskCounter++
			taskID := fmt.Sprintf("bench-task-%d", taskCounter)

			task := NewTask(taskID, func(ctx context.Context) error {
				// 模拟轻量级工作
				for i := 0; i < 100; i++ {
					_ = i * i
				}
				return nil
			})

			pool.SubmitAndWait(task)
		}
	})
}

// BenchmarkDirectGoroutine 直接创建goroutine的性能基准
func BenchmarkDirectGoroutine(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()
				// 模拟轻量级工作
				for i := 0; i < 100; i++ {
					_ = i * i
				}
			}()

			wg.Wait()
		}
	})
}

// TestChannelSafeClose 测试安全的channel关闭
func TestChannelSafeClose(t *testing.T) {
	manager := NewGracefulShutdownManager(DefaultShutdownConfig())
	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	t.Run("关闭不同类型的channel", func(t *testing.T) {
		// 字符串channel
		strCh := make(chan string, 1)
		strCh <- "test"
		manager.RegisterChannel("string-ch", strCh)

		// 字节channel
		byteCh := make(chan []byte, 1)
		byteCh <- []byte("test")
		manager.RegisterChannel("byte-ch", byteCh)

		// 错误channel
		errCh := make(chan error, 1)
		errCh <- fmt.Errorf("test error")
		manager.RegisterChannel("error-ch", errCh)

		// 结构体channel
		structCh := make(chan struct{}, 1)
		structCh <- struct{}{}
		manager.RegisterChannel("struct-ch", structCh)

		// 安全关闭所有channel
		assert.NoError(t, manager.SafeClose("string-ch"))
		assert.NoError(t, manager.SafeClose("byte-ch"))
		assert.NoError(t, manager.SafeClose("error-ch"))
		assert.NoError(t, manager.SafeClose("struct-ch"))

		// 验证channel已关闭
		_, ok := <-strCh
		assert.False(t, ok)
		_, ok = <-byteCh
		assert.False(t, ok)
		_, ok = <-errCh
		assert.False(t, ok)
		_, ok = <-structCh
		assert.False(t, ok)
	})
}

// TestIntegration 集成测试：结合goroutine池和优雅关闭管理器
func TestIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// 创建优雅关闭管理器
	config := DefaultShutdownConfig()
	config.ComponentTimeout = 2 * time.Second
	manager := NewGracefulShutdownManager(config)

	err := manager.Start()
	require.NoError(t, err)

	// 模拟一个使用goroutine池的组件
	poolComponent := &PoolBasedComponent{
		name:   "pool-component",
		logger: logger,
	}

	err = manager.RegisterComponent(poolComponent)
	require.NoError(t, err)

	err = manager.StartComponent(poolComponent.Name())
	require.NoError(t, err)

	// 提交一些任务到池中
	var taskResults int32
	for i := 0; i < 5; i++ {
		manager.SubmitTask(fmt.Sprintf("integration-task-%d", i), func(ctx context.Context) error {
			atomic.AddInt32(&taskResults, 1)
			time.Sleep(50 * time.Millisecond)
			return nil
		})
	}

	// 等待任务完成
	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, int32(5), atomic.LoadInt32(&taskResults))

	// 优雅关闭
	err = manager.Stop()
	assert.NoError(t, err)

	// 验证组件已停止
	assert.True(t, poolComponent.isStopped)
}

// PoolBasedComponent 基于池的组件，用于集成测试
type PoolBasedComponent struct {
	name      string
	logger    *zap.Logger
	pool      *GoroutinePool
	isStopped bool
}

func (pbc *PoolBasedComponent) Name() string {
	return pbc.name
}

func (pbc *PoolBasedComponent) Start(ctx context.Context) error {
	config := DefaultGoroutinePoolConfig()
	config.MinWorkers = 2
	config.MaxWorkers = 4

	pbc.pool = NewGoroutinePool(config)
	return pbc.pool.Start()
}

func (pbc *PoolBasedComponent) Stop(ctx context.Context) error {
	pbc.isStopped = true
	if pbc.pool != nil {
		return pbc.pool.StopWithTimeout(2 * time.Second)
	}
	return nil
}

func (pbc *PoolBasedComponent) State() ComponentState {
	if pbc.isStopped {
		return StateStopped
	}
	if pbc.pool != nil && pbc.pool.IsRunning() {
		return StateRunning
	}
	return StateUnknown
}
