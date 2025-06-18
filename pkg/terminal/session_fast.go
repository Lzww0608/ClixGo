/*
 * @Author: Lzww0608
 * @Date: 2025-6-18 20:35:00
 * @LastEditors: Lzww0608
 * @LastEditTime: 2025-6-18 20:35:00
 * @Description: Phase 1.3 任务1.2 - 快速启动SessionManager (延迟初始化版本)
 */

package terminal

import (
	"sync"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/errors"
	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/performance"
	clixsync "github.com/Lzww0608/ClixGo/pkg/sync"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// FastSessionManager 快速启动的会话管理器 - 延迟初始化版本
type FastSessionManager struct {
	*SessionManager

	// 延迟初始化标志
	lazyInitOnce    sync.Once
	lazyInitialized bool
	lazyInitMutex   sync.RWMutex
}

// NewFastSessionManager 创建快速启动的会话管理器
func NewFastSessionManager(config *TerminalConfig) *FastSessionManager {
	if config == nil {
		config = DefaultConfig
	}

	// 仅创建基础结构，不启动任何后台服务
	sessionManager := &SessionManager{
		sessions: make(map[string]*Session),
		config:   config,
		performanceStats: &SessionPerformanceStats{
			LastOptimization: time.Now(),
		},
	}

	fastManager := &FastSessionManager{
		SessionManager:  sessionManager,
		lazyInitialized: false,
	}

	logger.Info("Fast SessionManager created - lazy initialization enabled",
		zap.Bool("object_pool_enabled", false),
		zap.Bool("goroutine_pool_enabled", false),
		zap.Bool("leak_detector_enabled", false))

	return fastManager
}

// ensureLazyInit 确保延迟初始化已完成
func (fsm *FastSessionManager) ensureLazyInit() {
	fsm.lazyInitMutex.RLock()
	if fsm.lazyInitialized {
		fsm.lazyInitMutex.RUnlock()
		return
	}
	fsm.lazyInitMutex.RUnlock()

	fsm.lazyInitOnce.Do(func() {
		startTime := time.Now()

		fsm.lazyInitMutex.Lock()
		defer fsm.lazyInitMutex.Unlock()

		// 初始化对象池 - 使用最小配置
		minPoolConfig := performance.PoolConfig{
			MaxPoolSize:     10,                // 减少池大小
			CleanupInterval: 10 * time.Minute,  // 增加清理间隔
			MaxIdleTime:     15 * time.Minute,  // 增加空闲时间
			EnableStats:     false,             // 禁用统计减少开销
			DefaultSizes:    []int{1024, 4096}, // 仅2种大小
		}
		fsm.objectPool = performance.NewObjectPoolManager(minPoolConfig)

		// 初始化协程池 - 使用最小配置
		minGoroutineConfig := clixsync.GoroutinePoolConfig{
			MinWorkers:  4,   // 减少到4个工作协程
			MaxWorkers:  16,  // 减少到16个
			QueueSize:   100, // 减少队列大小
			IdleTimeout: 30 * time.Second,
			TaskTimeout: 30 * time.Second,
		}
		fsm.goroutinePool = clixsync.NewGoroutinePool(minGoroutineConfig)

		// 启动协程池
		if err := fsm.goroutinePool.Start(); err != nil {
			logger.Error("Failed to start goroutine pool during lazy init", zap.Error(err))
		}

		// 可选：内存泄漏检测器 - 使用宽松配置
		// 检查配置中是否启用了内存泄漏检测（添加新字段或默认启用）
		enableLeakDetection := true // 默认启用，可以通过配置控制
		if enableLeakDetection {
			baseLogger, _ := zap.NewProduction()
			if baseLogger == nil {
				baseLogger = zap.NewNop()
			}

			leakDetectorConfig := performance.MemoryLeakDetectorConfig{
				CheckInterval:                60 * time.Second, // 增加检查间隔
				GoroutineGrowthThreshold:     50,               // 提高阈值
				MemoryGrowthThresholdMB:      100.0,            // 提高阈值
				HeapGrowthThresholdMB:        50.0,             // 提高阈值
				ConsecutiveFailuresThreshold: 5,                // 提高阈值
			}
			fsm.leakDetector = performance.NewMemoryLeakDetector(
				leakDetectorConfig,
				baseLogger.Named("fast-session-leak-detector"),
			)

			// 启动内存泄漏检测器
			if err := fsm.leakDetector.Start(); err != nil {
				logger.Error("Failed to start memory leak detector during lazy init", zap.Error(err))
			}
		}

		fsm.lazyInitialized = true

		initTime := time.Since(startTime)
		logger.Info("Fast SessionManager lazy initialization completed",
			zap.Duration("init_time", initTime),
			zap.Bool("object_pool_enabled", true),
			zap.Bool("goroutine_pool_enabled", true),
			zap.Bool("leak_detector_enabled", enableLeakDetection))
	})
}

// CreateSession 创建新会话 - 触发延迟初始化
func (fsm *FastSessionManager) CreateSession(name string) (*Session, error) {
	fsm.ensureLazyInit()

	// 使用同步版本的会话创建，避免协程池超时
	return fsm.createSessionSync(name)
}

// createSessionSync 同步版本的会话创建方法
func (fsm *FastSessionManager) createSessionSync(name string) (*Session, error) {
	startTime := time.Now()

	// 验证和生成会话名称
	sessionName := fsm.generateSessionName(name)

	// 检查会话名是否已存在
	if fsm.sessionExists(sessionName) {
		return nil, errors.SessionExists(sessionName)
	}

	// 同步创建会话对象
	session, err := fsm.buildSessionSync(sessionName)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "创建会话对象失败")
	}

	// 创建默认窗口
	window, err := fsm.createWindowSync(session, "")
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "创建默认窗口失败")
	}

	session.Windows = append(session.Windows, window)
	fsm.sessions[session.ID] = session

	// 更新性能统计
	fsm.performanceStats.CreatedSessions++
	fsm.performanceStats.ActiveSessions++
	fsm.performanceStats.AvgCreateTime = time.Since(startTime)

	logger.Info("fast session created synchronously",
		zap.String("session_id", session.ID),
		zap.String("session_name", sessionName),
		zap.Duration("create_time", time.Since(startTime)),
		zap.Int("total_sessions", len(fsm.sessions)),
	)

	return session, nil
}

// buildSessionSync 同步版本的会话构建方法
func (fsm *FastSessionManager) buildSessionSync(name string) (*Session, error) {
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

// createWindowSync 同步版本的窗口创建方法
func (fsm *FastSessionManager) createWindowSync(session *Session, name string) (*Window, error) {
	windowName := fsm.generateWindowName(session, name)
	windowID := uuid.New().String()

	// 创建默认面板 - 同步版本
	pane, err := fsm.createPaneSync(windowID, "bash")
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

	fsm.performanceStats.TotalWindows++
	return window, nil
}

// createPaneSync 同步版本的面板创建方法
func (fsm *FastSessionManager) createPaneSync(windowID, command string) (*Pane, error) {
	paneID := uuid.New().String()

	pane := &Pane{
		ID:        paneID,
		Command:   command,
		ProcessID: 0, // 这里应该是实际的进程ID
		CreatedAt: time.Now(),
		Active:    true,
		// 设置默认尺寸
		Width:  80,
		Height: 24,
		X:      0,
		Y:      0,
	}

	fsm.performanceStats.TotalPanes++
	return pane, nil
}

// 继承SessionManager的辅助方法
func (fsm *FastSessionManager) generateSessionName(name string) string {
	return fsm.SessionManager.generateSessionName(name)
}

func (fsm *FastSessionManager) sessionExists(name string) bool {
	return fsm.SessionManager.sessionExists(name)
}

func (fsm *FastSessionManager) generateWindowName(session *Session, name string) string {
	return fsm.SessionManager.generateWindowName(session, name)
}

// CreateWindow 创建窗口 - 触发延迟初始化
func (fsm *FastSessionManager) CreateWindow(sessionID, name string) (*Window, error) {
	fsm.ensureLazyInit()
	return fsm.SessionManager.CreateWindow(sessionID, name)
}

// SplitPane 分割面板 - 触发延迟初始化
func (fsm *FastSessionManager) SplitPane(sessionID string, windowIndex int, direction string) (*Pane, error) {
	fsm.ensureLazyInit()
	return fsm.SessionManager.SplitPane(sessionID, windowIndex, direction)
}

// SwitchWindow 切换窗口 - 触发延迟初始化
func (fsm *FastSessionManager) SwitchWindow(sessionID string, windowIndex int) error {
	fsm.ensureLazyInit()
	return fsm.SessionManager.SwitchWindow(sessionID, windowIndex)
}

// SwitchPane 切换面板 - 触发延迟初始化
func (fsm *FastSessionManager) SwitchPane(sessionID string, windowIndex, paneIndex int) error {
	fsm.ensureLazyInit()
	return fsm.SessionManager.SwitchPane(sessionID, windowIndex, paneIndex)
}

// GetSession 获取会话 - 触发延迟初始化
func (fsm *FastSessionManager) GetSession(sessionID string) (*Session, error) {
	fsm.ensureLazyInit()
	return fsm.SessionManager.GetSession(sessionID)
}

// ListSessions 列出会话 - 触发延迟初始化
func (fsm *FastSessionManager) ListSessions() []*Session {
	fsm.ensureLazyInit()
	return fsm.SessionManager.ListSessions()
}

// RenameSession 重命名会话 - 触发延迟初始化
func (fsm *FastSessionManager) RenameSession(sessionID, newName string) error {
	fsm.ensureLazyInit()
	return fsm.SessionManager.RenameSession(sessionID, newName)
}

// ClosePane 关闭面板 - 触发延迟初始化
func (fsm *FastSessionManager) ClosePane(sessionID string, windowIndex, paneIndex int) error {
	fsm.ensureLazyInit()
	return fsm.SessionManager.ClosePane(sessionID, windowIndex, paneIndex)
}

// KillSession 杀死会话 - 触发延迟初始化
func (fsm *FastSessionManager) KillSession(sessionID string) error {
	fsm.ensureLazyInit()
	return fsm.SessionManager.KillSession(sessionID)
}

// GetPerformanceStats 获取性能统计 - 包含初始化状态
func (fsm *FastSessionManager) GetPerformanceStats() *SessionPerformanceStats {
	stats := fsm.SessionManager.GetPerformanceStats()

	fsm.lazyInitMutex.RLock()
	initialized := fsm.lazyInitialized
	fsm.lazyInitMutex.RUnlock()

	if !initialized {
		// 如果未初始化，返回基础统计信息
		return &SessionPerformanceStats{
			CreatedSessions:  0,
			ActiveSessions:   0,
			TotalWindows:     0,
			TotalPanes:       0,
			LastOptimization: time.Now(),
		}
	}

	return stats
}

// Shutdown 关闭会话管理器
func (fsm *FastSessionManager) Shutdown() error {
	fsm.lazyInitMutex.RLock()
	initialized := fsm.lazyInitialized
	fsm.lazyInitMutex.RUnlock()

	if !initialized {
		logger.Info("Fast SessionManager shutdown - was not initialized")
		return nil
	}

	return fsm.SessionManager.Shutdown()
}

// IsLazyInitialized 检查是否已完成延迟初始化
func (fsm *FastSessionManager) IsLazyInitialized() bool {
	fsm.lazyInitMutex.RLock()
	defer fsm.lazyInitMutex.RUnlock()
	return fsm.lazyInitialized
}
