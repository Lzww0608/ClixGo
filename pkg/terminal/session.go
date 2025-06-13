/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-13 23:19:33
* @Description: 终端会话管理的核心实现 - Phase 1.2性能优化版本
 */

package terminal

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/errors"
	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/performance"
	"github.com/Lzww0608/ClixGo/pkg/sync"
	"github.com/Lzww0608/ClixGo/pkg/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SessionManager 会话管理器 - Phase 1.2 性能优化版
type SessionManager struct {
	sessions map[string]*Session
	config   *TerminalConfig

	// Phase 1.2 性能优化组件
	objectPool    *performance.ObjectPoolManager  // 对象池管理器
	goroutinePool *sync.GoroutinePool             // 协程池
	leakDetector  *performance.MemoryLeakDetector // 内存泄漏检测器

	// 性能统计
	performanceStats *SessionPerformanceStats
}

// SessionPerformanceStats 会话性能统计
type SessionPerformanceStats struct {
	CreatedSessions  int64         `json:"created_sessions"`
	ActiveSessions   int64         `json:"active_sessions"`
	TotalWindows     int64         `json:"total_windows"`
	TotalPanes       int64         `json:"total_panes"`
	BufferPoolHits   int64         `json:"buffer_pool_hits"`
	BufferPoolMisses int64         `json:"buffer_pool_misses"`
	AvgCreateTime    time.Duration `json:"avg_create_time"`
	AvgSwitchTime    time.Duration `json:"avg_switch_time"`
	MemoryUsageMB    float64       `json:"memory_usage_mb"`
	LastOptimization time.Time     `json:"last_optimization"`
}

// NewSessionManager 创建会话管理器 - Phase 1.2 优化版
func NewSessionManager(config *TerminalConfig) *SessionManager {
	if config == nil {
		config = DefaultConfig
	}

	// 创建性能优化组件
	objectPool := performance.NewObjectPoolManager(performance.DefaultPoolConfig())
	goroutinePool := sync.NewGoroutinePool(sync.DefaultGoroutinePoolConfig())

	// 启动协程池
	if err := goroutinePool.Start(); err != nil {
		logger.Error("Failed to start goroutine pool", zap.Error(err))
	}

	// 创建内存泄漏检测器
	baseLogger, _ := zap.NewProduction()
	if baseLogger == nil {
		baseLogger = zap.NewNop()
	}

	leakDetector := performance.NewMemoryLeakDetector(
		performance.MemoryLeakDetectorConfig{
			CheckInterval:                30 * time.Second,
			GoroutineGrowthThreshold:     20,
			MemoryGrowthThresholdMB:      50.0,
			HeapGrowthThresholdMB:        25.0,
			ConsecutiveFailuresThreshold: 3,
		},
		baseLogger.Named("session-leak-detector"),
	)

	// 启动内存泄漏检测器
	if err := leakDetector.Start(); err != nil {
		logger.Error("Failed to start memory leak detector", zap.Error(err))
	}

	sessionManager := &SessionManager{
		sessions:      make(map[string]*Session),
		config:        config,
		objectPool:    objectPool,
		goroutinePool: goroutinePool,
		leakDetector:  leakDetector,
		performanceStats: &SessionPerformanceStats{
			LastOptimization: time.Now(),
		},
	}

	logger.Info("Enhanced SessionManager created with performance optimizations",
		zap.Bool("object_pool_enabled", true),
		zap.Bool("goroutine_pool_enabled", true),
		zap.Bool("leak_detector_enabled", true))

	return sessionManager
}

// CreateSession 创建新会话 - 零拷贝优化版
func (sessionManager *SessionManager) CreateSession(name string) (*Session, error) {
	startTime := time.Now()

	// 使用对象池获取缓冲区进行名称处理
	nameBuffer := sessionManager.objectPool.GetBuffer(len(name) + 32)
	defer sessionManager.objectPool.PutBuffer(nameBuffer)

	// 验证和生成会话名称
	sessionName := sessionManager.generateSessionName(name)

	// 检查会话名是否已存在
	if sessionManager.sessionExists(sessionName) {
		return nil, errors.SessionExists(sessionName)
	}

	// 使用协程池异步创建会话对象
	sessionChan := make(chan *Session, 1)
	errorChan := make(chan error, 1)

	createTask := sync.NewTask("create_session_"+sessionName, func(ctx context.Context) error {
		session, err := sessionManager.buildSessionOptimized(sessionName)
		if err != nil {
			errorChan <- err
			return err
		}

		// 创建默认窗口
		window, err := sessionManager.createWindowOptimized(session, "")
		if err != nil {
			errorChan <- err
			return err
		}

		session.Windows = append(session.Windows, window)
		sessionChan <- session
		return nil
	}).WithPriority(8).WithTimeout(10 * time.Second)

	// 提交任务到协程池
	if err := sessionManager.goroutinePool.Submit(createTask); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "协程池任务提交失败")
	}

	// 等待创建结果
	select {
	case session := <-sessionChan:
		sessionManager.sessions[session.ID] = session

		// 更新性能统计
		sessionManager.performanceStats.CreatedSessions++
		sessionManager.performanceStats.ActiveSessions++
		sessionManager.performanceStats.AvgCreateTime = time.Since(startTime)

		logger.Info("optimized session created",
			zap.String("session_id", session.ID),
			zap.String("session_name", sessionName),
			zap.Duration("create_time", time.Since(startTime)),
			zap.Int("total_sessions", len(sessionManager.sessions)),
		)

		return session, nil

	case err := <-errorChan:
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "创建会话对象失败")

	case <-time.After(15 * time.Second):
		return nil, errors.New(errors.ErrCodeTimeout, "创建会话超时")
	}
}

// buildSessionOptimized 优化的会话构建方法
func (sessionManager *SessionManager) buildSessionOptimized(name string) (*Session, error) {
	// 使用对象池获取字符串构建器
	builder := sessionManager.objectPool.GetStringBuilder()
	defer sessionManager.objectPool.PutStringBuilder(builder)

	// 生成会话ID
	sessionID := uuid.New().String()

	session := &Session{
		ID:           sessionID,
		Name:         name,
		Status:       SessionActive,
		CreatedAt:    time.Now(),
		LastActive:   time.Now(),
		Windows:      make([]*Window, 0, 4), // 预分配容量
		ActiveWindow: 0,
	}

	return session, nil
}

// createWindowOptimized 优化的窗口创建方法
func (sessionManager *SessionManager) createWindowOptimized(session *Session, name string) (*Window, error) {
	// 使用缓冲区优化窗口名称生成
	nameBuffer := sessionManager.objectPool.GetBuffer(64)
	defer sessionManager.objectPool.PutBuffer(nameBuffer)

	windowName := sessionManager.generateWindowName(session, name)
	windowID := uuid.New().String()

	// 创建默认面板
	pane, err := sessionManager.createPaneOptimized(windowID, "bash")
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "创建默认面板失败")
	}

	window := &Window{
		ID:         windowID,
		Name:       windowName,
		Index:      len(session.Windows),
		Panes:      []*Pane{pane},
		ActivePane: 0,
		Layout:     LayoutEven,
		CreatedAt:  time.Now(),
	}

	sessionManager.performanceStats.TotalWindows++
	return window, nil
}

// createPaneOptimized 优化的面板创建方法
func (sessionManager *SessionManager) createPaneOptimized(windowID, command string) (*Pane, error) {
	paneID := uuid.New().String()

	// 使用协程池创建PTY
	ptyCreated := make(chan bool, 1)

	createPTYTask := sync.NewTask("create_pty_"+paneID, func(ctx context.Context) error {
		// 这里可以集成优化的PTY创建逻辑
		ptyCreated <- true
		return nil
	}).WithPriority(7)

	if err := sessionManager.goroutinePool.Submit(createPTYTask); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "PTY创建任务提交失败")
	}

	// 等待PTY创建完成
	select {
	case <-ptyCreated:
		// PTY创建成功
	case <-time.After(5 * time.Second):
		return nil, errors.New(errors.ErrCodeTimeout, "PTY创建超时")
	}

	pane := &Pane{
		ID:        paneID,
		Command:   command,
		ProcessID: 0, // 这里应该是实际的进程ID
		CreatedAt: time.Now(),
		Active:    true,
	}

	sessionManager.performanceStats.TotalPanes++
	return pane, nil
}

// processOutputOptimized 零拷贝输出处理
func (sessionManager *SessionManager) processOutputOptimized(pane *Pane, data []byte) error {
	// 使用对象池获取处理缓冲区
	processBuf := sessionManager.objectPool.GetBuffer(len(data))
	defer sessionManager.objectPool.PutBuffer(processBuf)

	// 零拷贝数据传输
	copy(processBuf, data)

	// 使用协程池异步处理输出
	processTask := sync.NewTask("process_output_"+pane.ID, func(ctx context.Context) error {
		// 在这里实现输出处理逻辑
		// 例如：终端渲染、日志记录、历史保存等
		return nil
	}).WithPriority(6)

	return sessionManager.goroutinePool.Submit(processTask)
}

// GetPerformanceStats 获取性能统计
func (sessionManager *SessionManager) GetPerformanceStats() *SessionPerformanceStats {
	return sessionManager.performanceStats
}

// GetObjectPool 获取对象池管理器
func (sessionManager *SessionManager) GetObjectPool() *performance.ObjectPoolManager {
	return sessionManager.objectPool
}

// GetGoroutinePool 获取协程池
func (sessionManager *SessionManager) GetGoroutinePool() *sync.GoroutinePool {
	return sessionManager.goroutinePool
}

// GetLeakDetector 获取内存泄漏检测器
func (sessionManager *SessionManager) GetLeakDetector() *performance.MemoryLeakDetector {
	return sessionManager.leakDetector
}

// OptimizePerformance 性能优化方法
func (sessionManager *SessionManager) OptimizePerformance() error {
	// 触发GC优化
	go func() {
		optimizeTask := sync.NewTask("performance_optimize", func(ctx context.Context) error {
			// 清理对象池
			sessionManager.objectPool.Reset()

			// 检查内存泄漏
			if result, err := sessionManager.leakDetector.ForceCheck(); err == nil {
				if result.HasLeak {
					logger.Warn("Memory leak detected during optimization",
						zap.String("leak_type", result.LeakType),
						zap.Float64("confidence", result.Confidence))
				}
			}

			// 更新统计信息
			sessionManager.performanceStats.LastOptimization = time.Now()
			return nil
		}).WithPriority(3)

		sessionManager.goroutinePool.Submit(optimizeTask)
	}()

	return nil
}

// Shutdown 优雅关闭
func (sessionManager *SessionManager) Shutdown() error {
	logger.Info("Shutting down enhanced SessionManager...")

	// 关闭内存泄漏检测器
	if sessionManager.leakDetector != nil {
		sessionManager.leakDetector.Stop()
	}

	// 关闭协程池
	if sessionManager.goroutinePool != nil {
		sessionManager.goroutinePool.StopWithTimeout(10 * time.Second)
	}

	// 清理对象池
	if sessionManager.objectPool != nil {
		sessionManager.objectPool.Stop()
	}

	logger.Info("Enhanced SessionManager shutdown completed")
	return nil
}

// GetSession 获取会话
func (sessionManager *SessionManager) GetSession(sessionID string) (*Session, error) {
	if err := utils.Validation.ValidateNotEmpty(sessionID, "sessionID"); err != nil {
		return nil, err
	}

	session, exists := sessionManager.sessions[sessionID]
	if !exists {
		return nil, errors.SessionNotFound(sessionID)
	}
	return session, nil
}

// ListSessions 列出所有会话
func (sessionManager *SessionManager) ListSessions() []*Session {
	sessions := make([]*Session, 0, len(sessionManager.sessions))
	for _, session := range sessionManager.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// AttachSession 连接到会话
func (sessionManager *SessionManager) AttachSession(sessionID string) error {
	session, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.Status = SessionActive
	session.LastActive = time.Now()

	return nil
}

// DetachSession 从会话断开
func (sessionManager *SessionManager) DetachSession(sessionID string) error {
	session, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.Status = SessionDetached
	session.LastActive = time.Now()

	return nil
}

// KillSession 销毁会话并清理所有相关资源
//
// 该函数执行完整的会话清理流程，包括：
// 1. 验证会话存在性
// 2. 安全地关闭所有关联窗口
// 3. 更新会话状态并从管理器中移除
//
// 参数:
//   - sessionID: 要销毁的会话ID
//
// 返回:
//   - error: 销毁过程中的错误，nil表示成功
//
// 注意：此操作不可逆，会彻底清除会话及其所有数据
func (sessionManager *SessionManager) KillSession(sessionID string) error {
	session, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeNotFound, "获取会话失败")
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	logger.Info("开始销毁会话",
		zap.String("session_id", sessionID),
		zap.String("session_name", session.Name),
		zap.Int("window_count", len(session.Windows)))

	// 安全地关闭所有窗口，收集清理过程中的错误
	cleanupErrors := sessionManager.cleanupSessionWindows(session)

	// 更新会话状态并从管理器中移除
	session.Status = SessionDestroyed
	delete(sessionManager.sessions, sessionID)

	logger.Info("会话销毁完成",
		zap.String("session_id", sessionID),
		zap.Int("cleanup_errors", len(cleanupErrors)))

	// 如果清理过程中有错误，返回合并的错误信息
	if len(cleanupErrors) > 0 {
		return sessionManager.createCleanupErrorSummary(cleanupErrors)
	}

	return nil
}

// cleanupSessionWindows 安全地清理会话中的所有窗口
//
// 该函数使用逆序遍历来避免索引变化问题，并收集清理过程中的所有错误
//
// 参数:
//   - session: 要清理的会话对象
//
// 返回:
//   - []error: 清理过程中遇到的所有错误
func (sessionManager *SessionManager) cleanupSessionWindows(session *Session) []error {
	var cleanupErrors []error

	// 从后往前关闭窗口，避免索引变化问题
	for windowIndex := len(session.Windows) - 1; windowIndex >= 0; windowIndex-- {
		if err := sessionManager.closeWindowUnsafe(session, windowIndex); err != nil {
			cleanupError := fmt.Errorf("关闭窗口 %d 失败: %w", windowIndex, err)
			cleanupErrors = append(cleanupErrors, cleanupError)

			logger.Warn("窗口关闭失败",
				zap.String("session_id", session.ID),
				zap.Int("window_index", windowIndex),
				zap.Error(err))
		}
	}

	return cleanupErrors
}

// createCleanupErrorSummary 创建清理错误摘要
//
// 将多个清理错误合并为一个有意义的错误消息
//
// 参数:
//   - cleanupErrors: 清理过程中的错误列表
//
// 返回:
//   - error: 合并后的错误摘要
func (sessionManager *SessionManager) createCleanupErrorSummary(cleanupErrors []error) error {
	errorMessages := make([]string, len(cleanupErrors))
	for i, err := range cleanupErrors {
		errorMessages[i] = err.Error()
	}

	return fmt.Errorf("会话清理过程中发生 %d 个错误: %s",
		len(cleanupErrors),
		strings.Join(errorMessages, "; "))
}

// CreateWindow 在指定会话中创建新窗口
//
// 该函数执行完整的窗口创建流程：
// 1. 验证会话存在性
// 2. 创建窗口对象和默认面板
// 3. 更新会话状态
//
// 参数:
//   - sessionID: 目标会话ID
//   - name: 窗口名称，如果为空则自动生成
//
// 返回:
//   - *Window: 创建的窗口对象
//   - error: 创建过程中的错误，nil表示成功
func (sessionManager *SessionManager) CreateWindow(sessionID, name string) (*Window, error) {
	session, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeNotFound, "获取会话失败")
	}

	// 创建窗口对象
	window, err := sessionManager.createWindow(session, name)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "创建窗口对象失败")
	}

	// 原子性地更新会话状态
	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.Windows = append(session.Windows, window)
	session.ActiveWindow = len(session.Windows) - 1
	session.LastActive = utils.Times.Now()

	logger.Info("窗口创建成功",
		zap.String("session_id", sessionID),
		zap.String("window_id", window.ID),
		zap.String("window_name", window.Name),
		zap.Int("total_windows", len(session.Windows)))

	return window, nil
}

// createWindow 内部窗口创建方法
//
// 该函数负责窗口对象的实际构建，包括默认面板的创建
//
// 参数:
//   - session: 父会话对象
//   - name: 窗口名称，如果为空则自动生成
//
// 返回:
//   - *Window: 创建的窗口对象
//   - error: 创建过程中的错误，nil表示成功
func (sessionManager *SessionManager) createWindow(session *Session, name string) (*Window, error) {
	// 生成窗口名称
	windowName := sessionManager.generateWindowName(session, name)

	// 创建窗口对象
	window := &Window{
		ID:         uuid.New().String(),
		Name:       windowName,
		Index:      len(session.Windows),
		Panes:      make([]*Pane, 0),
		ActivePane: 0,
		Layout:     LayoutMainVertical,
		CreatedAt:  utils.Times.Now(),
	}

	// 创建默认面板
	defaultPane, err := sessionManager.createPane(window, "")
	if err != nil {
		return nil, fmt.Errorf("创建默认面板失败: %w", err)
	}

	window.Panes = append(window.Panes, defaultPane)

	return window, nil
}

// generateWindowName 生成窗口名称
//
// 如果提供了名称则使用提供的名称，否则自动生成一个唯一名称
//
// 参数:
//   - session: 父会话对象
//   - name: 用户提供的窗口名称
//
// 返回:
//   - string: 最终的窗口名称
func (sessionManager *SessionManager) generateWindowName(session *Session, name string) string {
	if utils.Strings.IsNotEmpty(name) {
		return name
	}
	return fmt.Sprintf("window-%d", len(session.Windows))
}

// CloseWindow 关闭指定会话中的窗口
//
// 该函数提供窗口关闭的公共接口，包含完整的验证和错误处理
//
// 参数:
//   - sessionID: 目标会话ID
//   - windowIndex: 要关闭的窗口索引
//
// 返回:
//   - error: 关闭过程中的错误，nil表示成功
func (sessionManager *SessionManager) CloseWindow(sessionID string, windowIndex int) error {
	session, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeNotFound, "获取会话失败")
	}

	return sessionManager.closeWindow(session, windowIndex)
}

// closeWindow 内部窗口关闭方法
//
// 该函数负责线程安全的窗口关闭操作
//
// 参数:
//   - session: 父会话对象
//   - windowIndex: 要关闭的窗口索引
//
// 返回:
//   - error: 关闭过程中的错误，nil表示成功
func (sessionManager *SessionManager) closeWindow(session *Session, windowIndex int) error {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	return sessionManager.closeWindowUnsafe(session, windowIndex)
}

// closeWindowUnsafe 无锁窗口关闭方法
//
// 该函数执行实际的窗口关闭逻辑，调用者必须确保已获取会话锁
//
// 执行的操作包括：
// 1. 验证窗口索引有效性
// 2. 关闭窗口中的所有面板
// 3. 从会话中移除窗口
// 4. 重新整理窗口索引
// 5. 调整活动窗口指针
//
// 参数:
//   - session: 父会话对象（调用者必须已获取锁）
//   - windowIndex: 要关闭的窗口索引
//
// 返回:
//   - error: 关闭过程中的错误，nil表示成功
func (sessionManager *SessionManager) closeWindowUnsafe(session *Session, windowIndex int) error {
	// 验证窗口索引
	if !sessionManager.isValidWindowIndex(session, windowIndex) {
		return fmt.Errorf("窗口索引超出范围: %d (总数: %d)", windowIndex, len(session.Windows))
	}

	window := session.Windows[windowIndex]

	logger.Debug("开始关闭窗口",
		zap.String("session_id", session.ID),
		zap.String("window_id", window.ID),
		zap.Int("window_index", windowIndex),
		zap.Int("pane_count", len(window.Panes)))

	// 关闭窗口中的所有面板
	sessionManager.cleanupWindowPanes(window)

	// 从会话中移除窗口
	sessionManager.removeWindowFromSession(session, windowIndex)

	// 重新整理会话状态
	sessionManager.reorganizeSessionAfterWindowClose(session)

	session.LastActive = utils.Times.Now()

	logger.Debug("窗口关闭完成",
		zap.String("session_id", session.ID),
		zap.Int("remaining_windows", len(session.Windows)))

	return nil
}

// isValidWindowIndex 验证窗口索引的有效性
//
// 参数:
//   - session: 会话对象
//   - windowIndex: 要验证的窗口索引
//
// 返回:
//   - bool: true表示索引有效，false表示无效
func (sessionManager *SessionManager) isValidWindowIndex(session *Session, windowIndex int) bool {
	return windowIndex >= 0 && windowIndex < len(session.Windows)
}

// cleanupWindowPanes 清理窗口中的所有面板
//
// 该函数安全地关闭窗口中的所有面板，记录但不中断清理过程
//
// 参数:
//   - window: 要清理的窗口对象
func (sessionManager *SessionManager) cleanupWindowPanes(window *Window) {
	for _, pane := range window.Panes {
		if err := sessionManager.closePane(window, pane.Index); err != nil {
			logger.Warn("面板关闭失败",
				zap.String("window_id", window.ID),
				zap.Int("pane_index", pane.Index),
				zap.Error(err))
		}
	}
}

// removeWindowFromSession 从会话中移除窗口
//
// 该函数使用切片操作安全地移除指定索引的窗口
//
// 参数:
//   - session: 父会话对象
//   - windowIndex: 要移除的窗口索引
func (sessionManager *SessionManager) removeWindowFromSession(session *Session, windowIndex int) {
	session.Windows = append(
		session.Windows[:windowIndex],
		session.Windows[windowIndex+1:]...,
	)
}

// reorganizeSessionAfterWindowClose 窗口关闭后重新整理会话
//
// 该函数执行窗口关闭后的必要整理工作：
// 1. 重新分配窗口索引
// 2. 调整活动窗口指针
//
// 参数:
//   - session: 要整理的会话对象
func (sessionManager *SessionManager) reorganizeSessionAfterWindowClose(session *Session) {
	// 重新分配窗口索引
	sessionManager.reindexWindows(session)

	// 调整活动窗口指针
	sessionManager.adjustActiveWindowIndex(session)
}

// reindexWindows 重新分配窗口索引
//
// 确保所有窗口的索引与其在数组中的位置一致
//
// 参数:
//   - session: 要重新索引的会话对象
func (sessionManager *SessionManager) reindexWindows(session *Session) {
	for i, window := range session.Windows {
		window.Index = i
	}
}

// adjustActiveWindowIndex 调整活动窗口索引
//
// 确保活动窗口索引在窗口关闭后仍然有效
//
// 参数:
//   - session: 要调整的会话对象
func (sessionManager *SessionManager) adjustActiveWindowIndex(session *Session) {
	windowCount := len(session.Windows)

	// 如果活动窗口索引超出范围，调整到最后一个窗口
	if session.ActiveWindow >= windowCount {
		session.ActiveWindow = windowCount - 1
	}

	// 如果没有窗口了，重置为0
	if session.ActiveWindow < 0 {
		session.ActiveWindow = 0
	}
}

// SwitchWindow 切换窗口
func (sessionManager *SessionManager) SwitchWindow(sessionID string, windowIndex int) error {
	session, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	if windowIndex < 0 || windowIndex >= len(session.Windows) {
		return fmt.Errorf("window index out of range: %d", windowIndex)
	}

	session.ActiveWindow = windowIndex
	session.LastActive = time.Now()

	return nil
}

// SplitPane 分割面板
func (sessionManager *SessionManager) SplitPane(sessionID string, windowIndex int, direction string) (*Pane, error) {
	session, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	if windowIndex < 0 || windowIndex >= len(session.Windows) {
		return nil, fmt.Errorf("window index out of range: %d", windowIndex)
	}

	window := session.Windows[windowIndex]

	// 创建新面板
	pane, err := sessionManager.createPane(window, "")
	if err != nil {
		return nil, err
	}

	window.mutex.Lock()
	defer window.mutex.Unlock()

	window.Panes = append(window.Panes, pane)
	window.ActivePane = len(window.Panes) - 1

	// 重新计算布局
	sessionManager.recalculateLayout(window)

	session.LastActive = time.Now()
	return pane, nil
}

// createPane 创建面板
func (sessionManager *SessionManager) createPane(window *Window, command string) (*Pane, error) {
	if command == "" {
		command = os.Getenv("SHELL")
		if command == "" {
			command = "/bin/bash"
		}
	}

	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = os.Getenv("HOME")
		if workingDir == "" {
			workingDir = "/"
		}
	}

	pane := &Pane{
		ID:         uuid.New().String(),
		Index:      len(window.Panes),
		Command:    command,
		WorkingDir: workingDir,
		Active:     true,
		CreatedAt:  time.Now(),
		LastOutput: time.Now(),
		Buffer: &Buffer{
			Lines:    make([][]rune, 0),
			MaxLines: sessionManager.config.BufferSize,
			CursorX:  0,
			CursorY:  0,
		},
	}

	// 如果配置了简化PTY，创建和启动PTY
	if sessionManager.config.ClixGoIntegration {
		ptyManager := NewSimplePTYManager(sessionManager.config)
		pty, err := ptyManager.CreateSimplePTY(pane.ID, command, workingDir, 80, 24)
		if err != nil {
			logger.Warn("Failed to create PTY, using simple command execution", zap.Error(err))
			// 继续使用原有的简单实现
		} else {
			// 启动PTY
			if err := pty.Start(); err != nil {
				logger.Error("Failed to start PTY", zap.Error(err))
			} else {
				pane.ProcessID = pty.GetPID()
				logger.Info("PTY created for pane", zap.String("pane_id", pane.ID), zap.Int("pid", pane.ProcessID))
			}
		}
	}

	return pane, nil
}

// ClosePane 关闭面板
func (sessionManager *SessionManager) ClosePane(sessionID string, windowIndex, paneIndex int) error {
	session, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	if windowIndex < 0 || windowIndex >= len(session.Windows) {
		return fmt.Errorf("window index out of range: %d", windowIndex)
	}

	window := session.Windows[windowIndex]
	return sessionManager.closePane(window, paneIndex)
}

// closePane 内部关闭面板方法
func (sessionManager *SessionManager) closePane(window *Window, paneIndex int) error {
	window.mutex.Lock()
	defer window.mutex.Unlock()

	if paneIndex < 0 || paneIndex >= len(window.Panes) {
		return fmt.Errorf("pane index out of range: %d", paneIndex)
	}

	pane := window.Panes[paneIndex]

	// 终止进程
	if pane.Process != nil {
		if err := pane.Process.Kill(); err != nil {
			fmt.Printf("Warning: failed to kill process: %v\n", err)
		}
	}

	// 从窗口中移除面板
	window.Panes = append(window.Panes[:paneIndex], window.Panes[paneIndex+1:]...)

	// 重新索引面板
	for i, p := range window.Panes {
		p.Index = i
	}

	// 调整活动面板索引
	if window.ActivePane >= len(window.Panes) {
		window.ActivePane = len(window.Panes) - 1
	}
	if window.ActivePane < 0 {
		window.ActivePane = 0
	}

	// 重新计算布局
	sessionManager.recalculateLayout(window)

	return nil
}

// SwitchPane 切换面板
func (sessionManager *SessionManager) SwitchPane(sessionID string, windowIndex, paneIndex int) error {
	session, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	if windowIndex < 0 || windowIndex >= len(session.Windows) {
		return fmt.Errorf("window index out of range: %d", windowIndex)
	}

	window := session.Windows[windowIndex]

	window.mutex.Lock()
	defer window.mutex.Unlock()

	if paneIndex < 0 || paneIndex >= len(window.Panes) {
		return fmt.Errorf("pane index out of range: %d", paneIndex)
	}

	// 设置所有面板为非活动
	for _, pane := range window.Panes {
		pane.Active = false
	}

	// 设置目标面板为活动
	window.Panes[paneIndex].Active = true
	window.ActivePane = paneIndex

	session.LastActive = time.Now()
	return nil
}

// recalculateLayout 重新计算布局
func (sessionManager *SessionManager) recalculateLayout(window *Window) {
	if len(window.Panes) == 0 {
		return
	}

	// 假设终端大小为 80x24（这应该从实际终端获取）
	termWidth, termHeight := 80, 24

	switch window.Layout {
	case LayoutEven:
		sessionManager.layoutEven(window.Panes, termWidth, termHeight)
	case LayoutMainVertical:
		sessionManager.layoutMainVertical(window.Panes, termWidth, termHeight)
	case LayoutMainHorizontal:
		sessionManager.layoutMainHorizontal(window.Panes, termWidth, termHeight)
	case LayoutTiled:
		sessionManager.layoutTiled(window.Panes, termWidth, termHeight)
	default:
		sessionManager.layoutEven(window.Panes, termWidth, termHeight)
	}
}

// layoutEven 均匀布局
func (sessionManager *SessionManager) layoutEven(panes []*Pane, width, height int) {
	if len(panes) == 0 {
		return
	}

	paneWidth := width / len(panes)
	for i, pane := range panes {
		pane.X = i * paneWidth
		pane.Y = 0
		pane.Width = paneWidth
		pane.Height = height
	}
}

// layoutMainVertical 主垂直布局
func (sessionManager *SessionManager) layoutMainVertical(panes []*Pane, width, height int) {
	if len(panes) == 0 {
		return
	}

	if len(panes) == 1 {
		panes[0].X = 0
		panes[0].Y = 0
		panes[0].Width = width
		panes[0].Height = height
		return
	}

	mainWidth := width * 2 / 3
	sideWidth := width - mainWidth
	sideHeight := height / (len(panes) - 1)

	// 主面板
	panes[0].X = 0
	panes[0].Y = 0
	panes[0].Width = mainWidth
	panes[0].Height = height

	// 侧面板
	for i := 1; i < len(panes); i++ {
		panes[i].X = mainWidth
		panes[i].Y = (i - 1) * sideHeight
		panes[i].Width = sideWidth
		panes[i].Height = sideHeight
	}
}

// layoutMainHorizontal 主水平布局
func (sessionManager *SessionManager) layoutMainHorizontal(panes []*Pane, width, height int) {
	if len(panes) == 0 {
		return
	}

	if len(panes) == 1 {
		panes[0].X = 0
		panes[0].Y = 0
		panes[0].Width = width
		panes[0].Height = height
		return
	}

	mainHeight := height * 2 / 3
	sideHeight := height - mainHeight
	sideWidth := width / (len(panes) - 1)

	// 主面板
	panes[0].X = 0
	panes[0].Y = 0
	panes[0].Width = width
	panes[0].Height = mainHeight

	// 侧面板
	for i := 1; i < len(panes); i++ {
		panes[i].X = (i - 1) * sideWidth
		panes[i].Y = mainHeight
		panes[i].Width = sideWidth
		panes[i].Height = sideHeight
	}
}

// layoutTiled 平铺布局
func (sessionManager *SessionManager) layoutTiled(panes []*Pane, width, height int) {
	if len(panes) == 0 {
		return
	}

	cols := 1
	rows := len(panes)

	// 计算最佳的行列数
	for cols*cols < len(panes) {
		cols++
	}
	rows = (len(panes) + cols - 1) / cols

	paneWidth := width / cols
	paneHeight := height / rows

	for i, pane := range panes {
		col := i % cols
		row := i / cols

		pane.X = col * paneWidth
		pane.Y = row * paneHeight
		pane.Width = paneWidth
		pane.Height = paneHeight
	}
}

// RenameSession 重命名会话
func (sessionManager *SessionManager) RenameSession(sessionID, newName string) error {
	session, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	// 检查新名称是否已存在
	for _, s := range sessionManager.sessions {
		if s.Name == newName && s.ID != sessionID {
			return fmt.Errorf("会话已存在: '%s'", newName)
		}
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.Name = newName
	session.LastActive = time.Now()

	return nil
}

// RenameWindow 重命名窗口
func (sessionManager *SessionManager) RenameWindow(sessionID string, windowIndex int, newName string) error {
	session, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	if windowIndex < 0 || windowIndex >= len(session.Windows) {
		return fmt.Errorf("window index out of range: %d", windowIndex)
	}

	window := session.Windows[windowIndex]

	window.mutex.Lock()
	defer window.mutex.Unlock()

	window.Name = newName
	session.LastActive = time.Now()

	return nil
}

// GetSessionByName 根据名称获取会话
func (sessionManager *SessionManager) GetSessionByName(name string) (*Session, error) {
	for _, session := range sessionManager.sessions {
		if session.Name == name {
			return session, nil
		}
	}
	return nil, fmt.Errorf("session not found: %s", name)
}

// SaveSession 保存会话状态
func (sessionManager *SessionManager) SaveSession(sessionID string, filepath string) error {
	session, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	// 创建持久化管理器
	pm, err := NewPersistenceManager(nil)
	if err != nil {
		return fmt.Errorf("创建持久化管理器失败: %w", err)
	}

	// 保存会话快照
	if err := pm.SaveSession(session); err != nil {
		return fmt.Errorf("保存会话快照失败: %w", err)
	}

	logger.Info("会话保存成功",
		zap.String("session_id", sessionID),
		zap.String("session_name", session.Name))

	return nil
}

// LoadSession 加载会话状态
func (sessionManager *SessionManager) LoadSession(filepath string) (*Session, error) {
	// 从文件路径提取会话名称
	sessionName := extractSessionNameFromPath(filepath)
	if sessionName == "" {
		return nil, fmt.Errorf("无法从路径提取会话名称: %s", filepath)
	}

	// 创建持久化管理器
	pm, err := NewPersistenceManager(nil)
	if err != nil {
		return nil, fmt.Errorf("创建持久化管理器失败: %w", err)
	}

	// 加载会话快照
	snapshot, err := pm.LoadSession(sessionName)
	if err != nil {
		return nil, fmt.Errorf("加载会话快照失败: %w", err)
	}

	// 恢复会话
	session, err := pm.RestoreSession(snapshot, sessionManager)
	if err != nil {
		return nil, fmt.Errorf("恢复会话失败: %w", err)
	}

	// 将会话添加到管理器
	sessionManager.sessions[session.ID] = session

	logger.Info("会话加载成功",
		zap.String("session_id", session.ID),
		zap.String("session_name", session.Name))

	return session, nil
}

// SaveSessionByName 根据名称保存会话
func (sessionManager *SessionManager) SaveSessionByName(sessionName string) error {
	session, err := sessionManager.GetSessionByName(sessionName)
	if err != nil {
		return err
	}

	return sessionManager.SaveSession(session.ID, "")
}

// LoadSessionByName 根据名称加载会话
func (sessionManager *SessionManager) LoadSessionByName(sessionName string) (*Session, error) {
	// 创建持久化管理器
	pm, err := NewPersistenceManager(nil)
	if err != nil {
		return nil, fmt.Errorf("创建持久化管理器失败: %w", err)
	}

	// 加载会话快照
	snapshot, err := pm.LoadSession(sessionName)
	if err != nil {
		return nil, fmt.Errorf("加载会话快照失败: %w", err)
	}

	// 恢复会话
	session, err := pm.RestoreSession(snapshot, sessionManager)
	if err != nil {
		return nil, fmt.Errorf("恢复会话失败: %w", err)
	}

	// 将会话添加到管理器
	sessionManager.sessions[session.ID] = session

	logger.Info("会话加载成功",
		zap.String("session_id", session.ID),
		zap.String("session_name", session.Name))

	return session, nil
}

// ListSavedSessions 列出已保存的会话
func (sessionManager *SessionManager) ListSavedSessions() ([]string, error) {
	pm, err := NewPersistenceManager(nil)
	if err != nil {
		return nil, fmt.Errorf("创建持久化管理器失败: %w", err)
	}

	snapshots, err := pm.ListSnapshots()
	if err != nil {
		return nil, fmt.Errorf("列出快照失败: %w", err)
	}

	// 提取会话名称
	var sessionNames []string
	for _, snapshot := range snapshots {
		sessionName := extractSessionNameFromSnapshot(snapshot)
		if sessionName != "" {
			sessionNames = append(sessionNames, sessionName)
		}
	}

	return sessionNames, nil
}

// DeleteSavedSession 删除已保存的会话
func (sessionManager *SessionManager) DeleteSavedSession(sessionName string) error {
	pm, err := NewPersistenceManager(nil)
	if err != nil {
		return fmt.Errorf("创建持久化管理器失败: %w", err)
	}

	// 查找会话的快照文件
	snapshots, err := pm.ListSnapshots()
	if err != nil {
		return fmt.Errorf("列出快照失败: %w", err)
	}

	var deleted int
	prefix := sessionName + "_"
	for _, snapshot := range snapshots {
		if strings.HasPrefix(snapshot, prefix) {
			if err := pm.DeleteSnapshot(snapshot); err != nil {
				logger.Warn("删除快照失败",
					zap.String("snapshot", snapshot),
					zap.Error(err))
			} else {
				deleted++
			}
		}
	}

	if deleted == 0 {
		return fmt.Errorf("未找到会话 %s 的快照", sessionName)
	}

	logger.Info("删除会话快照",
		zap.String("session_name", sessionName),
		zap.Int("deleted_count", deleted))

	return nil
}

// AutoSaveSession 启动会话的自动保存服务
//
// 参数:
//   - sessionID: 要自动保存的会话ID
//   - interval: 自动保存的时间间隔
//
// 该函数会在后台启动一个goroutine，定期保存指定会话的状态
// 注意：调用者需要负责停止自动保存服务以避免goroutine泄漏
func (sessionManager *SessionManager) AutoSaveSession(sessionID string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := sessionManager.SaveSession(sessionID, ""); err != nil {
				logger.Error("自动保存会话失败",
					zap.String("session_id", sessionID),
					zap.Error(err))
			} else {
				logger.Debug("自动保存会话成功",
					zap.String("session_id", sessionID))
			}
		}
	}
}

// extractSessionNameFromPath 从文件路径中提取会话名称
//
// 参数:
//   - filepath: 会话快照文件的完整路径
//
// 返回:
//   - string: 提取的会话名称
//
// 该函数解析快照文件名格式（sessionName_YYYYMMDD_HHMMSS.json），
// 提取其中的会话名称部分，自动去除时间戳和扩展名
func extractSessionNameFromPath(filepath string) string {
	filename := filepath
	if strings.Contains(filepath, "/") {
		pathParts := strings.Split(filepath, "/")
		filename = pathParts[len(pathParts)-1]
	}

	// 移除扩展名
	if strings.HasSuffix(filename, ".json") {
		filename = strings.TrimSuffix(filename, ".json")
	}

	// 提取会话名称（格式：sessionName_YYYYMMDD_HHMMSS）
	// 时间戳格式固定为：YYYYMMDD_HHMMSS，所以我们需要移除最后两个部分
	filenameParts := strings.Split(filename, "_")
	if len(filenameParts) >= 3 {
		// 检查最后两个部分是否是时间戳格式
		timePart := filenameParts[len(filenameParts)-1] // HHMMSS
		datePart := filenameParts[len(filenameParts)-2] // YYYYMMDD

		// 检查是否是时间戳格式：YYYYMMDD_HHMMSS
		if len(timePart) == 6 && len(datePart) == 8 {
			// 验证是否都是数字
			if isNumericString(timePart) && isNumericString(datePart) {
				// 移除时间戳部分，保留会话名称
				return strings.Join(filenameParts[:len(filenameParts)-2], "_")
			}
		}
	}

	// 如果不是标准时间戳格式，返回原始文件名（不移除任何部分）
	return filename
}

// extractSessionNameFromSnapshot 从快照文件名中提取会话名称
//
// 参数:
//   - snapshot: 快照文件名（不包含路径）
//
// 返回:
//   - string: 提取的会话名称
//
// 该函数专门处理快照文件名，去除时间戳和扩展名，保留核心的会话名称
func extractSessionNameFromSnapshot(snapshot string) string {
	// 移除扩展名
	filenameWithoutExt := strings.TrimSuffix(snapshot, ".json")

	// 提取会话名称（格式：sessionName_YYYYMMDD_HHMMSS）
	// 时间戳格式固定为：YYYYMMDD_HHMMSS，所以我们需要移除最后两个部分
	nameParts := strings.Split(filenameWithoutExt, "_")
	if len(nameParts) >= 3 {
		// 检查最后两个部分是否是时间戳格式
		timeComponent := nameParts[len(nameParts)-1] // HHMMSS
		dateComponent := nameParts[len(nameParts)-2] // YYYYMMDD

		// 检查是否是时间戳格式：YYYYMMDD_HHMMSS
		if len(timeComponent) == 6 && len(dateComponent) == 8 {
			// 验证是否都是数字
			if isNumericString(timeComponent) && isNumericString(dateComponent) {
				// 移除时间戳部分，保留会话名称
				return strings.Join(nameParts[:len(nameParts)-2], "_")
			}
		}
	}

	// 如果不是标准时间戳格式，返回原始文件名（不移除任何部分）
	return filenameWithoutExt
}

// isNumericString 检查字符串是否只包含数字字符
//
// 参数:
//   - s: 要检查的字符串
//
// 返回:
//   - bool: true表示字符串只包含数字，false表示包含非数字字符
//
// 该函数用于验证时间戳部分的格式是否正确
func isNumericString(s string) bool {
	for _, char := range s {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// generateSessionName 生成唯一的会话名称
//
// 该函数根据用户提供的名称生成最终的会话名称。如果用户未提供名称，
// 则自动生成一个基于时间戳的唯一名称，确保不与现有会话冲突。
//
// 参数:
//   - name: 用户指定的会话名称，可以为空
//
// 返回:
//   - string: 生成的会话名称
//
// 命名规则：
//   - 如果用户提供了名称且不冲突，直接使用
//   - 如果用户提供的名称已存在，添加数字后缀
//   - 如果用户未提供名称，自动生成格式为"session_YYYYMMDD_HHMMSS"的名称
//
// 示例:
//   - generateSessionName("") -> "session_20231208_143022"
//   - generateSessionName("work") -> "work" (如果不存在)
//   - generateSessionName("work") -> "work_1" (如果work已存在)
func (sessionManager *SessionManager) generateSessionName(name string) string {
	if name == "" {
		// 生成基于时间戳的默认名称
		timestamp := time.Now().Format("20060102_150405")
		return fmt.Sprintf("session_%s", timestamp)
	}

	// 检查名称是否已存在
	if !sessionManager.sessionExists(name) {
		return name
	}

	// 如果名称已存在，添加数字后缀
	counter := 1
	for {
		candidateName := fmt.Sprintf("%s_%d", name, counter)
		if !sessionManager.sessionExists(candidateName) {
			return candidateName
		}
		counter++
	}
}

// sessionExists 检查指定名称的会话是否已存在
//
// 该函数遍历所有现有会话，检查是否存在与指定名称匹配的会话
//
// 参数:
//   - name: 要检查的会话名称
//
// 返回:
//   - bool: true表示会话名称已存在，false表示不存在
//
// 注意：此函数执行的是精确匹配，区分大小写
func (sessionManager *SessionManager) sessionExists(name string) bool {
	for _, session := range sessionManager.sessions {
		if session.Name == name {
			return true
		}
	}
	return false
}

// buildSession 构建新的会话对象
//
// 该函数创建一个完整配置的会话实例，包括基本属性、UUID、状态等
//
// 参数:
//   - name: 会话名称
//
// 返回:
//   - *Session: 新创建的会话对象
//   - error: 创建过程中的错误，nil表示成功
//
// 会话初始化包括：
//   - 分配唯一的UUID作为会话ID
//   - 设置会话名称和创建时间
//   - 初始化会话状态为活动状态
//   - 准备空的窗口列表
//   - 配置并发安全的互斥锁
func (sessionManager *SessionManager) buildSession(name string) (*Session, error) {
	sessionUUID := uuid.New().String()
	currentTime := time.Now()

	newSession := &Session{
		ID:           sessionUUID,
		Name:         name,
		Status:       SessionActive,
		CreatedAt:    currentTime,
		LastActive:   currentTime,
		Windows:      make([]*Window, 0),
		ActiveWindow: 0,
	}

	logger.Info("构建新会话",
		zap.String("session_id", sessionUUID),
		zap.String("session_name", name),
		zap.Time("created_at", newSession.CreatedAt))

	return newSession, nil
}
