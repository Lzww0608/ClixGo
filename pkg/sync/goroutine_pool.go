/*
* @Author: Lzww0608
* @Date: 2025-6-6 23:47:11
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-7 15:52:19
* @Description: 高性能Goroutine池实现 - 优化并发任务执行和资源管理
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

// TaskFunc 表示要在goroutine池中执行的任务函数
type TaskFunc func(ctx context.Context) error

// Task 表示一个待执行的任务
type Task struct {
	ID       string        // 任务唯一标识
	Func     TaskFunc      // 任务执行函数
	Priority int           // 任务优先级 (0=最低, 10=最高)
	Timeout  time.Duration // 任务超时时间
	Created  time.Time     // 任务创建时间
	Result   chan error    // 结果通道
}

// NewTask 创建新的任务
func NewTask(id string, fn TaskFunc) *Task {
	return &Task{
		ID:       id,
		Func:     fn,
		Priority: 5,                // 默认中等优先级
		Timeout:  30 * time.Second, // 默认超时
		Created:  time.Now(),
		Result:   make(chan error, 1),
	}
}

// WithPriority 设置任务优先级
func (t *Task) WithPriority(priority int) *Task {
	if priority < 0 {
		priority = 0
	} else if priority > 10 {
		priority = 10
	}
	t.Priority = priority
	return t
}

// WithTimeout 设置任务超时
func (t *Task) WithTimeout(timeout time.Duration) *Task {
	t.Timeout = timeout
	return t
}

// GoroutinePoolConfig Goroutine池配置
type GoroutinePoolConfig struct {
	MinWorkers       int               // 最小工作goroutine数量
	MaxWorkers       int               // 最大工作goroutine数量
	QueueSize        int               // 任务队列大小
	IdleTimeout      time.Duration     // 空闲goroutine超时时间
	TaskTimeout      time.Duration     // 默认任务超时时间
	EnableMetrics    bool              // 是否启用指标收集
	EnablePriority   bool              // 是否启用优先级队列
	PanicHandler     func(interface{}) // panic处理函数
	WorkerNamePrefix string            // 工作goroutine名称前缀
}

// DefaultGoroutinePoolConfig 默认配置
func DefaultGoroutinePoolConfig() GoroutinePoolConfig {
	return GoroutinePoolConfig{
		MinWorkers:       runtime.NumCPU(),
		MaxWorkers:       runtime.NumCPU() * 4,
		QueueSize:        1000,
		IdleTimeout:      30 * time.Second,
		TaskTimeout:      60 * time.Second,
		EnableMetrics:    true,
		EnablePriority:   true,
		WorkerNamePrefix: "clixgo-worker",
		PanicHandler: func(p interface{}) {
			// 默认panic处理：记录日志但不崩溃
			if logger := zap.L(); logger != nil {
				logger.Error("Goroutine池任务执行panic", zap.Any("panic", p))
			}
		},
	}
}

// PoolMetrics 池指标
type PoolMetrics struct {
	ActiveWorkers   int32         // 活跃工作goroutine数
	IdleWorkers     int32         // 空闲工作goroutine数
	PendingTasks    int32         // 待处理任务数
	CompletedTasks  uint64        // 已完成任务数
	FailedTasks     uint64        // 失败任务数
	PanicCount      uint64        // panic计数
	TotalWorkers    int32         // 总工作goroutine数
	QueueCapacity   int32         // 队列容量
	AverageWaitTime time.Duration // 平均等待时间
	AverageExecTime time.Duration // 平均执行时间
	StartTime       time.Time     // 池启动时间
}

// GoroutinePool 高性能Goroutine池
type GoroutinePool struct {
	config GoroutinePoolConfig
	ctx    context.Context
	cancel context.CancelFunc

	// 任务队列
	taskQueue chan *Task

	// 工作goroutine管理
	workers       map[string]*worker // 工作goroutine映射
	workersMu     sync.RWMutex       // 工作goroutine保护锁
	activeWorkers int32              // 活跃工作goroutine数量
	idleWorkers   int32              // 空闲工作goroutine数量

	// 状态管理
	running int32 // 池运行状态 (0=停止, 1=运行)
	closed  int32 // 池关闭状态 (0=开启, 1=关闭)

	// 指标统计
	metrics   PoolMetrics
	metricsMu sync.RWMutex

	// 优雅关闭
	shutdownOnce sync.Once
	shutdownCh   chan struct{}

	logger *zap.Logger
}

// worker 工作goroutine
type worker struct {
	id         string
	pool       *GoroutinePool
	taskCh     chan *Task
	stopCh     chan struct{}
	lastActive time.Time
	tasksCount uint64
	created    time.Time
}

// NewGoroutinePool 创建新的Goroutine池
func NewGoroutinePool(config GoroutinePoolConfig) *GoroutinePool {
	ctx, cancel := context.WithCancel(context.Background())

	// 验证和调整配置
	if config.MinWorkers < 1 {
		config.MinWorkers = 1
	}
	if config.MaxWorkers < config.MinWorkers {
		config.MaxWorkers = config.MinWorkers
	}
	if config.QueueSize < 1 {
		config.QueueSize = 100
	}
	if config.IdleTimeout < time.Second {
		config.IdleTimeout = 30 * time.Second
	}
	if config.TaskTimeout < time.Second {
		config.TaskTimeout = 60 * time.Second
	}

	logger, _ := zap.NewProduction()
	if logger == nil {
		logger = zap.NewNop()
	}

	pool := &GoroutinePool{
		config:     config,
		ctx:        ctx,
		cancel:     cancel,
		taskQueue:  make(chan *Task, config.QueueSize),
		workers:    make(map[string]*worker),
		shutdownCh: make(chan struct{}),
		logger:     logger,
		metrics: PoolMetrics{
			StartTime:     time.Now(),
			QueueCapacity: int32(config.QueueSize),
		},
	}

	return pool
}

// Start 启动Goroutine池
func (gp *GoroutinePool) Start() error {
	if !atomic.CompareAndSwapInt32(&gp.running, 0, 1) {
		return fmt.Errorf("goroutine池已在运行中")
	}

	gp.logger.Info("启动Goroutine池",
		zap.Int("min_workers", gp.config.MinWorkers),
		zap.Int("max_workers", gp.config.MaxWorkers),
		zap.Int("queue_size", gp.config.QueueSize))

	// 启动最小数量的工作goroutine
	for i := 0; i < gp.config.MinWorkers; i++ {
		if err := gp.addWorker(); err != nil {
			gp.logger.Error("创建初始工作goroutine失败", zap.Error(err))
			return err
		}
	}

	// 启动任务分发器
	go gp.dispatcher()

	// 启动工作goroutine监控器
	go gp.workerMonitor()

	// 启动指标收集器
	if gp.config.EnableMetrics {
		go gp.metricsCollector()
	}

	return nil
}

// Stop 停止Goroutine池
func (gp *GoroutinePool) Stop() error {
	return gp.StopWithTimeout(30 * time.Second)
}

// StopWithTimeout 带超时的停止Goroutine池
func (gp *GoroutinePool) StopWithTimeout(timeout time.Duration) error {
	if !atomic.CompareAndSwapInt32(&gp.closed, 0, 1) {
		return fmt.Errorf("goroutine池已经关闭")
	}

	gp.shutdownOnce.Do(func() {
		gp.logger.Info("停止Goroutine池", zap.Duration("timeout", timeout))

		// 关闭任务队列，不再接受新任务
		close(gp.taskQueue)

		// 取消上下文，通知所有工作goroutine停止
		gp.cancel()

		// 等待工作goroutine退出，但设置最大等待时间
		start := time.Now()
		maxWait := time.Minute // 最多等待1分钟
		if timeout < maxWait {
			maxWait = timeout
		}

		for time.Since(start) < maxWait {
			gp.workersMu.RLock()
			count := len(gp.workers)
			gp.workersMu.RUnlock()

			if count == 0 {
				gp.logger.Info("所有工作goroutine已优雅退出")
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		// 如果仍有worker存在，强制清理
		gp.workersMu.Lock()
		if len(gp.workers) > 0 {
			gp.logger.Warn("强制清理剩余工作goroutine", zap.Int("count", len(gp.workers)))
			// 清空映射，重置计数器
			gp.workers = make(map[string]*worker)
			atomic.StoreInt32(&gp.activeWorkers, 0)
			atomic.StoreInt32(&gp.idleWorkers, 0)
			atomic.StoreInt32(&gp.metrics.TotalWorkers, 0)
		}
		gp.workersMu.Unlock()

		// 标记为停止
		atomic.StoreInt32(&gp.running, 0)
		close(gp.shutdownCh)
	})

	return nil
}

// Submit 提交任务到池中执行
func (gp *GoroutinePool) Submit(task *Task) error {
	if atomic.LoadInt32(&gp.closed) == 1 {
		return fmt.Errorf("goroutine池已关闭")
	}

	if atomic.LoadInt32(&gp.running) == 0 {
		return fmt.Errorf("goroutine池未运行")
	}

	// 设置默认超时时间
	if task.Timeout == 0 {
		task.Timeout = gp.config.TaskTimeout
	}

	select {
	case gp.taskQueue <- task:
		atomic.AddInt32(&gp.metrics.PendingTasks, 1)
		return nil
	case <-gp.ctx.Done():
		return fmt.Errorf("goroutine池已关闭")
	default:
		return fmt.Errorf("任务队列已满")
	}
}

// SubmitAndWait 提交任务并等待执行结果
func (gp *GoroutinePool) SubmitAndWait(task *Task) error {
	if err := gp.Submit(task); err != nil {
		return err
	}

	select {
	case err := <-task.Result:
		return err
	case <-gp.ctx.Done():
		return fmt.Errorf("goroutine池已关闭")
	}
}

// SubmitFunc 提交函数任务
func (gp *GoroutinePool) SubmitFunc(id string, fn TaskFunc) error {
	task := NewTask(id, fn)
	return gp.Submit(task)
}

// SubmitFuncAndWait 提交函数任务并等待结果
func (gp *GoroutinePool) SubmitFuncAndWait(id string, fn TaskFunc) error {
	task := NewTask(id, fn)
	return gp.SubmitAndWait(task)
}

// GetMetrics 获取池指标
func (gp *GoroutinePool) GetMetrics() PoolMetrics {
	gp.metricsMu.RLock()
	defer gp.metricsMu.RUnlock()

	// 更新实时指标
	metrics := gp.metrics
	metrics.ActiveWorkers = atomic.LoadInt32(&gp.activeWorkers)
	metrics.IdleWorkers = atomic.LoadInt32(&gp.idleWorkers)
	metrics.PendingTasks = atomic.LoadInt32(&gp.metrics.PendingTasks)
	metrics.TotalWorkers = atomic.LoadInt32(&gp.metrics.TotalWorkers)

	return metrics
}

// IsRunning 检查池是否正在运行
func (gp *GoroutinePool) IsRunning() bool {
	return atomic.LoadInt32(&gp.running) == 1
}

// IsClosed 检查池是否已关闭
func (gp *GoroutinePool) IsClosed() bool {
	return atomic.LoadInt32(&gp.closed) == 1
}

// dispatcher 任务分发器
func (gp *GoroutinePool) dispatcher() {
	defer func() {
		if r := recover(); r != nil {
			gp.logger.Error("任务分发器panic", zap.Any("panic", r))
			if gp.config.PanicHandler != nil {
				gp.config.PanicHandler(r)
			}
		}
	}()

	for {
		select {
		case task, ok := <-gp.taskQueue:
			if !ok {
				// 任务队列已关闭
				return
			}

			atomic.AddInt32(&gp.metrics.PendingTasks, -1)

			// 尝试分配给空闲工作goroutine
			if !gp.dispatchToIdleWorker(task) {
				// 没有空闲工作goroutine，尝试创建新的
				if gp.shouldCreateNewWorker() {
					if err := gp.addWorker(); err != nil {
						gp.logger.Error("创建新工作goroutine失败", zap.Error(err))
						// 创建失败，任务放回队列
						select {
						case gp.taskQueue <- task:
							atomic.AddInt32(&gp.metrics.PendingTasks, 1)
						default:
							// 队列满了，丢弃任务
							atomic.AddUint64(&gp.metrics.FailedTasks, 1)
							task.Result <- fmt.Errorf("无法创建工作goroutine且队列已满")
						}
						continue
					}
				}

				// 再次尝试分配给工作goroutine
				if !gp.dispatchToIdleWorker(task) {
					// 仍然无法分配，任务放回队列头部
					select {
					case gp.taskQueue <- task:
						atomic.AddInt32(&gp.metrics.PendingTasks, 1)
					default:
						// 队列满了，丢弃任务
						atomic.AddUint64(&gp.metrics.FailedTasks, 1)
						task.Result <- fmt.Errorf("所有工作goroutine忙碌且队列已满")
					}
				}
			}

		case <-gp.ctx.Done():
			return
		}
	}
}

// dispatchToIdleWorker 将任务分发给空闲的工作goroutine
func (gp *GoroutinePool) dispatchToIdleWorker(task *Task) bool {
	gp.workersMu.RLock()
	defer gp.workersMu.RUnlock()

	for _, w := range gp.workers {
		select {
		case w.taskCh <- task:
			return true
		default:
			// 工作goroutine忙碌，继续查找
		}
	}

	return false
}

// shouldCreateNewWorker 判断是否应该创建新的工作goroutine
func (gp *GoroutinePool) shouldCreateNewWorker() bool {
	currentWorkers := len(gp.workers)
	return currentWorkers < gp.config.MaxWorkers &&
		atomic.LoadInt32(&gp.idleWorkers) == 0
}

// addWorker 添加新的工作goroutine
func (gp *GoroutinePool) addWorker() error {
	gp.workersMu.Lock()
	defer gp.workersMu.Unlock()

	workerID := fmt.Sprintf("%s-%d", gp.config.WorkerNamePrefix, len(gp.workers)+1)

	w := &worker{
		id:         workerID,
		pool:       gp,
		taskCh:     make(chan *Task, 1),
		stopCh:     make(chan struct{}),
		lastActive: time.Now(),
		created:    time.Now(),
	}

	gp.workers[workerID] = w
	atomic.AddInt32(&gp.metrics.TotalWorkers, 1)
	atomic.AddInt32(&gp.idleWorkers, 1)

	go w.run()

	gp.logger.Debug("创建新工作goroutine", zap.String("worker_id", workerID))
	return nil
}

// removeWorker 移除工作goroutine
func (gp *GoroutinePool) removeWorker(workerID string) {
	gp.workersMu.Lock()
	defer gp.workersMu.Unlock()

	if _, exists := gp.workers[workerID]; exists {
		// 不要在这里关闭stopCh，worker已经退出了
		delete(gp.workers, workerID)
		atomic.AddInt32(&gp.metrics.TotalWorkers, -1)
		gp.logger.Debug("移除工作goroutine", zap.String("worker_id", workerID))
	}
}

// workerMonitor 工作goroutine监控器
func (gp *GoroutinePool) workerMonitor() {
	ticker := time.NewTicker(gp.config.IdleTimeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			gp.cleanupIdleWorkers()
		case <-gp.ctx.Done():
			return
		}
	}
}

// cleanupIdleWorkers 清理空闲的工作goroutine
func (gp *GoroutinePool) cleanupIdleWorkers() {
	gp.workersMu.Lock()
	defer gp.workersMu.Unlock()

	now := time.Now()
	minWorkers := gp.config.MinWorkers
	currentWorkers := len(gp.workers)

	if currentWorkers <= minWorkers {
		return
	}

	toRemove := make([]string, 0)
	for id, w := range gp.workers {
		if now.Sub(w.lastActive) > gp.config.IdleTimeout {
			toRemove = append(toRemove, id)
			if currentWorkers-len(toRemove) <= minWorkers {
				break
			}
		}
	}

	for _, id := range toRemove {
		if w, exists := gp.workers[id]; exists {
			close(w.stopCh) // 发送停止信号
			// worker自己会在退出时清理计数器，这里不删除映射
			gp.logger.Debug("发送停止信号给空闲工作goroutine", zap.String("worker_id", id))
		}
	}
}

// waitForWorkersShutdown 等待所有工作goroutine关闭（已废弃，在Stop中内联实现）

// metricsCollector 指标收集器
func (gp *GoroutinePool) metricsCollector() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			gp.updateMetrics()
		case <-gp.ctx.Done():
			return
		}
	}
}

// updateMetrics 更新指标
func (gp *GoroutinePool) updateMetrics() {
	// 这里可以添加更详细的指标计算逻辑
	// 例如计算平均等待时间、执行时间等
}

// run 工作goroutine主循环
func (w *worker) run() {
	defer func() {
		if r := recover(); r != nil {
			atomic.AddUint64(&w.pool.metrics.PanicCount, 1)
			w.pool.logger.Error("工作goroutine panic",
				zap.String("worker_id", w.id),
				zap.Any("panic", r))
			if w.pool.config.PanicHandler != nil {
				w.pool.config.PanicHandler(r)
			}
		}
		// 在panic后清理worker
		w.pool.removeWorker(w.id)
	}()

	// worker已经在addWorker中被标记为idle，这里不需要重复操作
	for {
		select {
		case task := <-w.taskCh:
			// 从idle转换为active
			atomic.AddInt32(&w.pool.idleWorkers, -1)
			atomic.AddInt32(&w.pool.activeWorkers, 1)
			w.lastActive = time.Now()
			w.tasksCount++

			w.executeTask(task)

			// 从active转换回idle
			atomic.AddInt32(&w.pool.activeWorkers, -1)
			atomic.AddInt32(&w.pool.idleWorkers, 1)

		case <-w.stopCh:
			// 退出时减去idle状态（worker创建时被标记为idle）
			atomic.AddInt32(&w.pool.idleWorkers, -1)
			return
		case <-w.pool.ctx.Done():
			// 退出时减去idle状态（worker创建时被标记为idle）
			atomic.AddInt32(&w.pool.idleWorkers, -1)
			return
		}
	}
}

// executeTask 执行任务
func (w *worker) executeTask(task *Task) {
	defer func() {
		if r := recover(); r != nil {
			atomic.AddUint64(&w.pool.metrics.PanicCount, 1)
			atomic.AddUint64(&w.pool.metrics.FailedTasks, 1)
			task.Result <- fmt.Errorf("任务执行panic: %v", r)
			if w.pool.config.PanicHandler != nil {
				w.pool.config.PanicHandler(r)
			}
		}
	}()

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(w.pool.ctx, task.Timeout)
	defer cancel()

	// 执行任务
	start := time.Now()
	err := task.Func(ctx)
	duration := time.Since(start)

	// 更新统计
	if err != nil {
		atomic.AddUint64(&w.pool.metrics.FailedTasks, 1)
	} else {
		atomic.AddUint64(&w.pool.metrics.CompletedTasks, 1)
	}

	// 发送结果
	select {
	case task.Result <- err:
	default:
		// 结果通道满，丢弃结果
	}

	w.pool.logger.Debug("任务执行完成",
		zap.String("task_id", task.ID),
		zap.String("worker_id", w.id),
		zap.Duration("duration", duration),
		zap.Error(err))
}
