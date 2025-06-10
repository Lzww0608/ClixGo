/*
* @Author: Lzww0608
* @Date: 2025-6-10 10:42:17
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-10 10:42:19
* @Description: 简化的终端服务实现 - 基于现有终端包的服务层封装
 */

package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/interfaces"
	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

// SimpleTerminalService 简化的终端服务实现
type SimpleTerminalService struct {
	sessionManager *terminal.SessionManager
	logger         interfaces.Logger
	mu             sync.RWMutex
	healthStatus   ServiceHealth
}

// NewSimpleTerminalService 创建简化的终端服务实例
func NewSimpleTerminalService(logger interfaces.Logger) *SimpleTerminalService {
	config := terminal.DefaultConfig
	sessionManager := terminal.NewSessionManager(config)

	service := &SimpleTerminalService{
		sessionManager: sessionManager,
		logger:         logger,
		healthStatus: ServiceHealth{
			ServiceName: "terminal_service",
			Status:      "healthy",
			Message:     "Terminal service initialized successfully",
			CheckedAt:   time.Now(),
		},
	}

	logger.Info("简化终端服务已初始化")
	return service
}

// CreateSession 创建新会话
func (t *SimpleTerminalService) CreateSession(name string) (interfaces.Session, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.logger.Debug(fmt.Sprintf("创建新会话: %s", name))

	session, err := t.sessionManager.CreateSession(name)
	if err != nil {
		t.logger.Error(fmt.Sprintf("创建会话失败: %v", err))
		return nil, fmt.Errorf("failed to create session %s: %w", name, err)
	}

	// 返回简单的会话包装器
	wrappedSession := &simpleSessionWrapper{
		sessionID: session.ID,
		name:      session.Name,
		createdAt: session.CreatedAt,
		manager:   t.sessionManager,
		logger:    t.logger,
	}

	t.logger.Info(fmt.Sprintf("会话创建成功: %s (ID: %s)", name, session.ID))
	return wrappedSession, nil
}

// GetSession 获取会话
func (t *SimpleTerminalService) GetSession(id string) (interfaces.Session, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	session, err := t.sessionManager.GetSession(id)
	if err != nil {
		return nil, fmt.Errorf("session %s not found: %w", id, err)
	}

	wrappedSession := &simpleSessionWrapper{
		sessionID: session.ID,
		name:      session.Name,
		createdAt: session.CreatedAt,
		manager:   t.sessionManager,
		logger:    t.logger,
	}

	return wrappedSession, nil
}

// ListSessions 列出所有会话
func (t *SimpleTerminalService) ListSessions() ([]interfaces.Session, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	sessions := t.sessionManager.ListSessions()
	wrappedSessions := make([]interfaces.Session, len(sessions))

	for i, session := range sessions {
		wrappedSessions[i] = &simpleSessionWrapper{
			sessionID: session.ID,
			name:      session.Name,
			createdAt: session.CreatedAt,
			manager:   t.sessionManager,
			logger:    t.logger,
		}
	}

	return wrappedSessions, nil
}

// CloseSession 关闭会话
func (t *SimpleTerminalService) CloseSession(id string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.logger.Debug(fmt.Sprintf("关闭会话: %s", id))

	err := t.sessionManager.KillSession(id)
	if err != nil {
		t.logger.Error(fmt.Sprintf("关闭会话失败: %v", err))
		return fmt.Errorf("failed to close session %s: %w", id, err)
	}

	t.logger.Info(fmt.Sprintf("会话已关闭: %s", id))
	return nil
}

// CreateWindow 创建新窗口
func (t *SimpleTerminalService) CreateWindow(sessionID, name string) (interfaces.Window, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	window, err := t.sessionManager.CreateWindow(sessionID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create window %s in session %s: %w", name, sessionID, err)
	}

	wrappedWindow := &simpleWindowWrapper{
		windowID:  window.ID,
		name:      window.Name,
		sessionID: sessionID,
		manager:   t.sessionManager,
		logger:    t.logger,
	}

	t.logger.Info(fmt.Sprintf("窗口创建成功: %s (Session: %s)", name, sessionID))
	return wrappedWindow, nil
}

// GetWindow 获取窗口
func (t *SimpleTerminalService) GetWindow(sessionID, windowID string) (interfaces.Window, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	session, err := t.sessionManager.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session %s not found: %w", sessionID, err)
	}

	// 在会话的窗口中查找指定ID的窗口
	for _, window := range session.Windows {
		if window.ID == windowID {
			wrappedWindow := &simpleWindowWrapper{
				windowID:  window.ID,
				name:      window.Name,
				sessionID: sessionID,
				manager:   t.sessionManager,
				logger:    t.logger,
			}
			return wrappedWindow, nil
		}
	}

	return nil, fmt.Errorf("window %s not found in session %s", windowID, sessionID)
}

// AttachToSession 连接到会话
func (t *SimpleTerminalService) AttachToSession(sessionID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	err := t.sessionManager.AttachSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to attach to session %s: %w", sessionID, err)
	}

	t.logger.Info(fmt.Sprintf("已连接到会话: %s", sessionID))
	return nil
}

// DetachFromSession 断开会话连接
func (t *SimpleTerminalService) DetachFromSession(sessionID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	err := t.sessionManager.DetachSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to detach from session %s: %w", sessionID, err)
	}

	t.logger.Info(fmt.Sprintf("已断开会话连接: %s", sessionID))
	return nil
}

// CheckHealth 实现健康检查接口
func (t *SimpleTerminalService) CheckHealth(ctx context.Context) ServiceHealth {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// 检查SessionManager状态
	sessions := t.sessionManager.ListSessions()
	activeCount := 0
	for _, session := range sessions {
		if session.Status == terminal.SessionActive {
			activeCount++
		}
	}

	health := ServiceHealth{
		ServiceName: "terminal_service",
		Status:      "healthy",
		Message:     fmt.Sprintf("Terminal service running with %d active sessions", activeCount),
		CheckedAt:   time.Now(),
	}

	// 如果活跃会话过多，标记为警告
	if activeCount > 50 {
		health.Status = "warning"
		health.Message = fmt.Sprintf("High session count: %d active sessions", activeCount)
	}

	t.healthStatus = health
	return health
}

// Dispose 释放资源
func (t *SimpleTerminalService) Dispose() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.logger.Info("释放终端服务资源")

	// 关闭所有会话
	sessions := t.sessionManager.ListSessions()
	for _, session := range sessions {
		if err := t.sessionManager.KillSession(session.ID); err != nil {
			t.logger.Error(fmt.Sprintf("关闭会话失败: %v", err))
		}
	}

	return nil
}

// simpleSessionWrapper 简单的Session接口包装器
type simpleSessionWrapper struct {
	sessionID string
	name      string
	createdAt time.Time
	manager   *terminal.SessionManager
	logger    interfaces.Logger
}

func (s *simpleSessionWrapper) ID() string {
	return s.sessionID
}

func (s *simpleSessionWrapper) Name() string {
	return s.name
}

func (s *simpleSessionWrapper) SetName(name string) error {
	err := s.manager.RenameSession(s.sessionID, name)
	if err == nil {
		s.name = name
	}
	return err
}

func (s *simpleSessionWrapper) CreatedAt() time.Time {
	return s.createdAt
}

func (s *simpleSessionWrapper) IsActive() bool {
	session, err := s.manager.GetSession(s.sessionID)
	if err != nil {
		return false
	}
	return session.Status == terminal.SessionActive
}

func (s *simpleSessionWrapper) Close() error {
	return s.manager.KillSession(s.sessionID)
}

func (s *simpleSessionWrapper) Resize(width, height int) error {
	// 简化实现，仅记录调整大小请求
	s.logger.Debug(fmt.Sprintf("调整会话 %s 大小: %dx%d", s.sessionID, width, height))
	return nil
}

func (s *simpleSessionWrapper) Write(data []byte) (int, error) {
	// 简化实现，返回写入的字节数
	s.logger.Debug(fmt.Sprintf("向会话 %s 写入 %d 字节", s.sessionID, len(data)))
	return len(data), nil
}

func (s *simpleSessionWrapper) Read(data []byte) (int, error) {
	// 简化实现，返回空读取
	return 0, fmt.Errorf("read not implemented in simple wrapper")
}

// simpleWindowWrapper 简单的Window接口包装器
type simpleWindowWrapper struct {
	windowID  string
	name      string
	sessionID string
	manager   *terminal.SessionManager
	logger    interfaces.Logger
}

func (w *simpleWindowWrapper) ID() string {
	return w.windowID
}

func (w *simpleWindowWrapper) SessionID() string {
	return w.sessionID
}

func (w *simpleWindowWrapper) Name() string {
	return w.name
}

func (w *simpleWindowWrapper) SetName(name string) error {
	// 需要先找到窗口索引
	session, err := w.manager.GetSession(w.sessionID)
	if err != nil {
		return err
	}

	for i, window := range session.Windows {
		if window.ID == w.windowID {
			err := w.manager.RenameWindow(w.sessionID, i, name)
			if err == nil {
				w.name = name
			}
			return err
		}
	}

	return fmt.Errorf("window not found")
}

func (w *simpleWindowWrapper) IsActive() bool {
	session, err := w.manager.GetSession(w.sessionID)
	if err != nil {
		return false
	}

	if len(session.Windows) > session.ActiveWindow {
		return session.Windows[session.ActiveWindow].ID == w.windowID
	}

	return false
}

func (w *simpleWindowWrapper) Panes() []interfaces.Pane {
	session, err := w.manager.GetSession(w.sessionID)
	if err != nil {
		return []interfaces.Pane{}
	}

	for _, window := range session.Windows {
		if window.ID == w.windowID {
			panes := make([]interfaces.Pane, len(window.Panes))
			for i, pane := range window.Panes {
				panes[i] = &simplePaneWrapper{
					paneID:     pane.ID,
					windowID:   w.windowID,
					sessionID:  w.sessionID,
					command:    pane.Command,
					workingDir: pane.WorkingDir,
					processID:  pane.ProcessID,
					active:     pane.Active,
					manager:    w.manager,
					logger:     w.logger,
				}
			}
			return panes
		}
	}

	return []interfaces.Pane{}
}

func (w *simpleWindowWrapper) CreatePane() (interfaces.Pane, error) {
	// 使用SessionManager的SplitPane功能
	session, err := w.manager.GetSession(w.sessionID)
	if err != nil {
		return nil, err
	}

	windowIndex := -1
	for i, window := range session.Windows {
		if window.ID == w.windowID {
			windowIndex = i
			break
		}
	}

	if windowIndex == -1 {
		return nil, fmt.Errorf("window not found")
	}

	pane, err := w.manager.SplitPane(w.sessionID, windowIndex, "vertical")
	if err != nil {
		return nil, err
	}

	return &simplePaneWrapper{
		paneID:     pane.ID,
		windowID:   w.windowID,
		sessionID:  w.sessionID,
		command:    pane.Command,
		workingDir: pane.WorkingDir,
		processID:  pane.ProcessID,
		active:     pane.Active,
		manager:    w.manager,
		logger:     w.logger,
	}, nil
}

func (w *simpleWindowWrapper) ClosePane(paneID string) error {
	session, err := w.manager.GetSession(w.sessionID)
	if err != nil {
		return err
	}

	windowIndex := -1
	paneIndex := -1

	for i, window := range session.Windows {
		if window.ID == w.windowID {
			windowIndex = i
			for j, pane := range window.Panes {
				if pane.ID == paneID {
					paneIndex = j
					break
				}
			}
			break
		}
	}

	if windowIndex == -1 || paneIndex == -1 {
		return fmt.Errorf("pane not found")
	}

	return w.manager.ClosePane(w.sessionID, windowIndex, paneIndex)
}

func (w *simpleWindowWrapper) SwitchPane(paneID string) error {
	session, err := w.manager.GetSession(w.sessionID)
	if err != nil {
		return err
	}

	windowIndex := -1
	paneIndex := -1

	for i, window := range session.Windows {
		if window.ID == w.windowID {
			windowIndex = i
			for j, pane := range window.Panes {
				if pane.ID == paneID {
					paneIndex = j
					break
				}
			}
			break
		}
	}

	if windowIndex == -1 || paneIndex == -1 {
		return fmt.Errorf("pane not found")
	}

	return w.manager.SwitchPane(w.sessionID, windowIndex, paneIndex)
}

// simplePaneWrapper 简单的Pane接口包装器
type simplePaneWrapper struct {
	paneID     string
	windowID   string
	sessionID  string
	command    string
	workingDir string
	processID  int
	active     bool
	manager    *terminal.SessionManager
	logger     interfaces.Logger
}

func (p *simplePaneWrapper) ID() string {
	return p.paneID
}

func (p *simplePaneWrapper) WindowID() string {
	return p.windowID
}

func (p *simplePaneWrapper) Command() string {
	return p.command
}

func (p *simplePaneWrapper) IsActive() bool {
	return p.active
}

func (p *simplePaneWrapper) ProcessID() int {
	return p.processID
}

func (p *simplePaneWrapper) WorkingDirectory() string {
	return p.workingDir
}

func (p *simplePaneWrapper) SetWorkingDirectory(dir string) error {
	p.workingDir = dir
	return nil
}

func (p *simplePaneWrapper) Write(data []byte) (int, error) {
	p.logger.Debug(fmt.Sprintf("向面板 %s 写入 %d 字节", p.paneID, len(data)))
	return len(data), nil
}

func (p *simplePaneWrapper) Read(data []byte) (int, error) {
	return 0, fmt.Errorf("read not implemented in simple wrapper")
}

func (p *simplePaneWrapper) Kill() error {
	// 通过关闭面板来终止进程
	session, err := p.manager.GetSession(p.sessionID)
	if err != nil {
		return err
	}

	windowIndex := -1
	paneIndex := -1

	for i, window := range session.Windows {
		if window.ID == p.windowID {
			windowIndex = i
			for j, pane := range window.Panes {
				if pane.ID == p.paneID {
					paneIndex = j
					break
				}
			}
			break
		}
	}

	if windowIndex == -1 || paneIndex == -1 {
		return fmt.Errorf("pane not found")
	}

	return p.manager.ClosePane(p.sessionID, windowIndex, paneIndex)
}
