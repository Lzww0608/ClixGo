/*
* @Author: Lzww0608
* @Date: 2025-06-06 14:30:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-7 15:52:49
* @Description: 优雅关闭管理器 - 统一管理goroutine生命周期和channel通信
 */

package sync

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// ShutdownHook 关闭钩子函数类型
type ShutdownHook func(ctx context.Context) error

// ComponentState 组件状态
type ComponentState int32

const (
	StateUnknown ComponentState = iota
	StateStarting
	StateRunning
	StateStopping
	StateStopped
	StateError
)

// String 返回组件状态的字符串表示
func (s ComponentState) String() string {
	switch s {
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	case StateStopped:
		return "stopped"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// Component 可管理的组件接口
type Component interface {
	// Start 启动组件
	Start(ctx context.Context) error
	// Stop 停止组件
	Stop(ctx context.Context) error
	// Name 组件名称
	Name() string
	// State 组件状态
	State() ComponentState
}

// ManagedComponent 被管理的组件
type ManagedComponent struct {
	component Component
	state     int32 // 使用原子操作的状态
	startTime time.Time
	stopTime  time.Time
	mu        sync.RWMutex
}

// ShutdownConfig 优雅关闭配置
type ShutdownConfig struct {
	// 组件关闭超时时间
	ComponentTimeout time.Duration
	// 全局关闭超时时间
	GlobalTimeout time.Duration
	// 是否并行关闭组件
	ParallelShutdown bool
	// 关闭重试次数
	RetryCount int
	// 重试间隔
	RetryInterval time.Duration
	// 是否启用详细日志
	VerboseLogging bool
	// 强制关闭前的宽限期
	GracePeriod time.Duration
}

// DefaultShutdownConfig 默认关闭配置
func DefaultShutdownConfig() ShutdownConfig {
	return ShutdownConfig{
		ComponentTimeout: 30 * time.Second,
		GlobalTimeout:    60 * time.Second,
		ParallelShutdown: true,
		RetryCount:       3,
		RetryInterval:    time.Second,
		VerboseLogging:   true,
		GracePeriod:      5 * time.Second,
	}
}

// GracefulShutdownManager 优雅关闭管理器
type GracefulShutdownManager struct {
	config ShutdownConfig
	ctx    context.Context
	cancel context.CancelFunc

	// 组件管理
	components   map[string]*ManagedComponent
	componentsMu sync.RWMutex

	// 关闭钩子
	hooks   []ShutdownHook
	hooksMu sync.RWMutex

	// 状态管理
	state     int32     // 管理器状态
	startTime time.Time // 启动时间
	stopTime  time.Time // 停止时间

	// Goroutine 管理
	activeGoroutines sync.WaitGroup
	goroutinePool    *GoroutinePool

	// Channel 管理
	channels   map[string]interface{}
	channelsMu sync.RWMutex

	logger     *zap.Logger
	shutdownCh chan struct{}
	once       sync.Once
}

// NewGracefulShutdownManager 创建优雅关闭管理器
func NewGracefulShutdownManager(config ShutdownConfig) *GracefulShutdownManager {
	ctx, cancel := context.WithCancel(context.Background())

	logger, _ := zap.NewProduction()
	if logger == nil {
		logger = zap.NewNop()
	}

	// 创建内置的goroutine池
	poolConfig := DefaultGoroutinePoolConfig()
	poolConfig.MinWorkers = 2
	poolConfig.MaxWorkers = 10

	manager := &GracefulShutdownManager{
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		components:    make(map[string]*ManagedComponent),
		hooks:         make([]ShutdownHook, 0),
		channels:      make(map[string]interface{}),
		logger:        logger,
		shutdownCh:    make(chan struct{}),
		startTime:     time.Now(),
		goroutinePool: NewGoroutinePool(poolConfig),
	}

	atomic.StoreInt32(&manager.state, int32(StateStarting))

	return manager
}

// Start 启动管理器
func (gsm *GracefulShutdownManager) Start() error {
	if !atomic.CompareAndSwapInt32(&gsm.state, int32(StateStarting), int32(StateRunning)) {
		return fmt.Errorf("管理器已启动或状态不正确")
	}

	// 启动内置goroutine池
	if err := gsm.goroutinePool.Start(); err != nil {
		atomic.StoreInt32(&gsm.state, int32(StateError))
		return fmt.Errorf("启动goroutine池失败: %w", err)
	}

	gsm.logger.Info("优雅关闭管理器已启动")
	return nil
}

// Stop 停止管理器
func (gsm *GracefulShutdownManager) Stop() error {
	return gsm.StopWithTimeout(gsm.config.GlobalTimeout)
}

// StopWithTimeout 带超时的停止管理器
func (gsm *GracefulShutdownManager) StopWithTimeout(timeout time.Duration) error {
	if !atomic.CompareAndSwapInt32(&gsm.state, int32(StateRunning), int32(StateStopping)) {
		currentState := ComponentState(atomic.LoadInt32(&gsm.state))
		return fmt.Errorf("管理器状态不正确，当前状态: %s", currentState.String())
	}

	gsm.once.Do(func() {
		gsm.stopTime = time.Now()
		close(gsm.shutdownCh)
	})

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	gsm.logger.Info("开始优雅关闭", zap.Duration("timeout", timeout))

	// 执行关闭流程
	if err := gsm.performShutdown(ctx); err != nil {
		gsm.logger.Error("优雅关闭失败", zap.Error(err))
		atomic.StoreInt32(&gsm.state, int32(StateError))
		return err
	}

	atomic.StoreInt32(&gsm.state, int32(StateStopped))
	gsm.logger.Info("优雅关闭完成", zap.Duration("total_time", time.Since(gsm.stopTime)))
	return nil
}

// RegisterComponent 注册组件
func (gsm *GracefulShutdownManager) RegisterComponent(component Component) error {
	gsm.componentsMu.Lock()
	defer gsm.componentsMu.Unlock()

	name := component.Name()
	if _, exists := gsm.components[name]; exists {
		return fmt.Errorf("组件 %s 已存在", name)
	}

	managed := &ManagedComponent{
		component: component,
		state:     int32(StateUnknown),
	}

	gsm.components[name] = managed
	gsm.logger.Debug("注册组件", zap.String("component", name))
	return nil
}

// UnregisterComponent 注销组件
func (gsm *GracefulShutdownManager) UnregisterComponent(name string) error {
	gsm.componentsMu.Lock()
	defer gsm.componentsMu.Unlock()

	if _, exists := gsm.components[name]; !exists {
		return fmt.Errorf("组件 %s 不存在", name)
	}

	delete(gsm.components, name)
	gsm.logger.Debug("注销组件", zap.String("component", name))
	return nil
}

// StartComponent 启动指定组件
func (gsm *GracefulShutdownManager) StartComponent(name string) error {
	gsm.componentsMu.RLock()
	managed, exists := gsm.components[name]
	gsm.componentsMu.RUnlock()

	if !exists {
		return fmt.Errorf("组件 %s 不存在", name)
	}

	if !atomic.CompareAndSwapInt32(&managed.state, int32(StateUnknown), int32(StateStarting)) {
		currentState := ComponentState(atomic.LoadInt32(&managed.state))
		return fmt.Errorf("组件 %s 状态不正确，当前状态: %s", name, currentState.String())
	}

	managed.mu.Lock()
	managed.startTime = time.Now()
	managed.mu.Unlock()

	ctx, cancel := context.WithTimeout(gsm.ctx, gsm.config.ComponentTimeout)
	defer cancel()

	if err := managed.component.Start(ctx); err != nil {
		atomic.StoreInt32(&managed.state, int32(StateError))
		return fmt.Errorf("启动组件 %s 失败: %w", name, err)
	}

	atomic.StoreInt32(&managed.state, int32(StateRunning))
	gsm.logger.Info("组件启动成功", zap.String("component", name))
	return nil
}

// StopComponent 停止指定组件
func (gsm *GracefulShutdownManager) StopComponent(name string) error {
	gsm.componentsMu.RLock()
	managed, exists := gsm.components[name]
	gsm.componentsMu.RUnlock()

	if !exists {
		return fmt.Errorf("组件 %s 不存在", name)
	}

	return gsm.stopManagedComponent(managed, gsm.config.ComponentTimeout)
}

// AddShutdownHook 添加关闭钩子
func (gsm *GracefulShutdownManager) AddShutdownHook(hook ShutdownHook) {
	gsm.hooksMu.Lock()
	defer gsm.hooksMu.Unlock()

	gsm.hooks = append(gsm.hooks, hook)
}

// RegisterChannel 注册channel用于管理
func (gsm *GracefulShutdownManager) RegisterChannel(name string, ch interface{}) error {
	gsm.channelsMu.Lock()
	defer gsm.channelsMu.Unlock()

	if _, exists := gsm.channels[name]; exists {
		return fmt.Errorf("channel %s 已存在", name)
	}

	gsm.channels[name] = ch
	gsm.logger.Debug("注册channel", zap.String("channel", name))
	return nil
}

// SafeClose 安全关闭channel
func (gsm *GracefulShutdownManager) SafeClose(name string) error {
	gsm.channelsMu.Lock()
	defer gsm.channelsMu.Unlock()

	ch, exists := gsm.channels[name]
	if !exists {
		return fmt.Errorf("channel %s 不存在", name)
	}

	defer func() {
		if r := recover(); r != nil {
			gsm.logger.Warn("关闭channel时发生panic",
				zap.String("channel", name),
				zap.Any("panic", r))
		}
	}()

	// 使用反射关闭不同类型的channel
	switch v := ch.(type) {
	case chan struct{}:
		close(v)
	case chan error:
		close(v)
	case chan []byte:
		close(v)
	case chan string:
		close(v)
	default:
		return fmt.Errorf("不支持的channel类型: %T", ch)
	}

	delete(gsm.channels, name)
	gsm.logger.Debug("安全关闭channel", zap.String("channel", name))
	return nil
}

// SubmitTask 提交任务到内置goroutine池
func (gsm *GracefulShutdownManager) SubmitTask(id string, fn TaskFunc) error {
	if gsm.goroutinePool == nil {
		return fmt.Errorf("goroutine池未初始化")
	}

	return gsm.goroutinePool.SubmitFunc(id, fn)
}

// RunManagedGoroutine 运行受管理的goroutine
func (gsm *GracefulShutdownManager) RunManagedGoroutine(name string, fn func(ctx context.Context)) {
	gsm.activeGoroutines.Add(1)

	go func() {
		defer func() {
			gsm.activeGoroutines.Done()
			if r := recover(); r != nil {
				gsm.logger.Error("受管理的goroutine发生panic",
					zap.String("goroutine", name),
					zap.Any("panic", r))
			}
		}()

		gsm.logger.Debug("启动受管理的goroutine", zap.String("goroutine", name))
		fn(gsm.ctx)
		gsm.logger.Debug("受管理的goroutine退出", zap.String("goroutine", name))
	}()
}

// GetComponentState 获取组件状态
func (gsm *GracefulShutdownManager) GetComponentState(name string) (ComponentState, error) {
	gsm.componentsMu.RLock()
	defer gsm.componentsMu.RUnlock()

	managed, exists := gsm.components[name]
	if !exists {
		return StateUnknown, fmt.Errorf("组件 %s 不存在", name)
	}

	return ComponentState(atomic.LoadInt32(&managed.state)), nil
}

// GetState 获取管理器状态
func (gsm *GracefulShutdownManager) GetState() ComponentState {
	return ComponentState(atomic.LoadInt32(&gsm.state))
}

// Context 获取管理器上下文
func (gsm *GracefulShutdownManager) Context() context.Context {
	return gsm.ctx
}

// ShutdownChannel 获取关闭信号channel
func (gsm *GracefulShutdownManager) ShutdownChannel() <-chan struct{} {
	return gsm.shutdownCh
}

// performShutdown 执行关闭流程
func (gsm *GracefulShutdownManager) performShutdown(ctx context.Context) error {
	var wg sync.WaitGroup
	var errs []error
	var errsMu sync.Mutex

	// 1. 执行关闭钩子
	gsm.logger.Debug("执行关闭钩子")
	if err := gsm.executeShutdownHooks(ctx); err != nil {
		gsm.logger.Error("执行关闭钩子失败", zap.Error(err))
		errsMu.Lock()
		errs = append(errs, err)
		errsMu.Unlock()
	}

	// 2. 停止所有组件
	gsm.logger.Debug("停止所有组件")
	if gsm.config.ParallelShutdown {
		gsm.componentsMu.RLock()
		for _, managed := range gsm.components {
			wg.Add(1)
			go func(m *ManagedComponent) {
				defer wg.Done()
				if err := gsm.stopManagedComponent(m, gsm.config.ComponentTimeout); err != nil {
					errsMu.Lock()
					errs = append(errs, err)
					errsMu.Unlock()
				}
			}(managed)
		}
		gsm.componentsMu.RUnlock()
		wg.Wait()
	} else {
		gsm.componentsMu.RLock()
		for _, managed := range gsm.components {
			if err := gsm.stopManagedComponent(managed, gsm.config.ComponentTimeout); err != nil {
				errs = append(errs, err)
			}
		}
		gsm.componentsMu.RUnlock()
	}

	// 3. 关闭所有注册的channel
	gsm.logger.Debug("关闭所有注册的channel")
	gsm.channelsMu.Lock()
	for name := range gsm.channels {
		if err := gsm.SafeClose(name); err != nil {
			gsm.logger.Warn("关闭channel失败", zap.String("channel", name), zap.Error(err))
		}
	}
	gsm.channelsMu.Unlock()

	// 4. 停止goroutine池
	gsm.logger.Debug("停止goroutine池")
	if gsm.goroutinePool != nil {
		if err := gsm.goroutinePool.StopWithTimeout(gsm.config.ComponentTimeout); err != nil {
			errsMu.Lock()
			errs = append(errs, fmt.Errorf("停止goroutine池失败: %w", err))
			errsMu.Unlock()
		}
	}

	// 5. 等待所有受管理的goroutine退出
	gsm.logger.Debug("等待受管理的goroutine退出")
	done := make(chan struct{})
	go func() {
		gsm.activeGoroutines.Wait()
		close(done)
	}()

	select {
	case <-done:
		gsm.logger.Debug("所有受管理的goroutine已退出")
	case <-ctx.Done():
		gsm.logger.Warn("等待goroutine退出超时")
		errsMu.Lock()
		errs = append(errs, fmt.Errorf("等待goroutine退出超时"))
		errsMu.Unlock()
	}

	// 6. 最终取消上下文
	gsm.cancel()

	if len(errs) > 0 {
		return fmt.Errorf("关闭过程中发生 %d 个错误: %v", len(errs), errs)
	}

	return nil
}

// executeShutdownHooks 执行关闭钩子
func (gsm *GracefulShutdownManager) executeShutdownHooks(ctx context.Context) error {
	gsm.hooksMu.RLock()
	hooks := make([]ShutdownHook, len(gsm.hooks))
	copy(hooks, gsm.hooks)
	gsm.hooksMu.RUnlock()

	var errs []error
	for i, hook := range hooks {
		if err := hook(ctx); err != nil {
			gsm.logger.Error("执行关闭钩子失败", zap.Int("hook_index", i), zap.Error(err))
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("执行关闭钩子失败: %v", errs)
	}

	return nil
}

// stopManagedComponent 停止被管理的组件
func (gsm *GracefulShutdownManager) stopManagedComponent(managed *ManagedComponent, timeout time.Duration) error {
	currentState := ComponentState(atomic.LoadInt32(&managed.state))

	// 只停止正在运行的组件
	if currentState != StateRunning {
		gsm.logger.Debug("跳过非运行状态的组件",
			zap.String("component", managed.component.Name()),
			zap.String("state", currentState.String()))
		return nil
	}

	if !atomic.CompareAndSwapInt32(&managed.state, int32(StateRunning), int32(StateStopping)) {
		return nil // 其他goroutine已经在停止这个组件
	}

	managed.mu.Lock()
	managed.stopTime = time.Now()
	managed.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	name := managed.component.Name()
	gsm.logger.Debug("停止组件", zap.String("component", name))

	// 尝试多次停止组件
	var lastErr error
	for i := 0; i < gsm.config.RetryCount; i++ {
		if err := managed.component.Stop(ctx); err != nil {
			lastErr = err
			gsm.logger.Warn("停止组件失败，准备重试",
				zap.String("component", name),
				zap.Int("attempt", i+1),
				zap.Error(err))

			if i < gsm.config.RetryCount-1 {
				time.Sleep(gsm.config.RetryInterval)
			}
		} else {
			atomic.StoreInt32(&managed.state, int32(StateStopped))
			gsm.logger.Info("组件停止成功", zap.String("component", name))
			return nil
		}
	}

	atomic.StoreInt32(&managed.state, int32(StateError))
	return fmt.Errorf("停止组件 %s 失败（重试 %d 次）: %w", name, gsm.config.RetryCount, lastErr)
}

// GetRuntimeInfo 获取运行时信息
func (gsm *GracefulShutdownManager) GetRuntimeInfo() map[string]interface{} {
	info := make(map[string]interface{})

	// 基本信息
	info["state"] = gsm.GetState().String()
	info["start_time"] = gsm.startTime
	if !gsm.stopTime.IsZero() {
		info["stop_time"] = gsm.stopTime
		info["uptime"] = gsm.stopTime.Sub(gsm.startTime)
	} else {
		info["uptime"] = time.Since(gsm.startTime)
	}

	// 组件信息
	gsm.componentsMu.RLock()
	components := make(map[string]string)
	for name, managed := range gsm.components {
		components[name] = ComponentState(atomic.LoadInt32(&managed.state)).String()
	}
	gsm.componentsMu.RUnlock()
	info["components"] = components

	// Channel信息
	gsm.channelsMu.RLock()
	channelNames := make([]string, 0, len(gsm.channels))
	for name := range gsm.channels {
		channelNames = append(channelNames, name)
	}
	gsm.channelsMu.RUnlock()
	info["channels"] = channelNames

	// Goroutine池信息
	if gsm.goroutinePool != nil {
		info["goroutine_pool"] = gsm.goroutinePool.GetMetrics()
	}

	// 系统信息
	info["num_goroutines"] = runtime.NumGoroutine()
	info["num_cpu"] = runtime.NumCPU()

	return info
}
