package terminal

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

// SessionManager 会话管理器
type SessionManager struct {
	sessions map[string]*Session
	config   *TerminalConfig
}

// NewSessionManager 创建会话管理器
func NewSessionManager(config *TerminalConfig) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		config:   config,
	}
}

// CreateSession 创建新会话
func (sm *SessionManager) CreateSession(name string) (*Session, error) {
	if name == "" {
		name = fmt.Sprintf("session-%d", len(sm.sessions))
	}

	// 检查会话名是否已存在
	for _, session := range sm.sessions {
		if session.Name == name {
			return nil, fmt.Errorf("session with name '%s' already exists", name)
		}
	}

	session := &Session{
		ID:           uuid.New().String(),
		Name:         name,
		Status:       SessionActive,
		CreatedAt:    time.Now(),
		LastActive:   time.Now(),
		Windows:      make([]*Window, 0),
		ActiveWindow: 0,
	}

	// 创建默认窗口
	window, err := sm.createWindow(session, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create default window: %v", err)
	}

	session.Windows = append(session.Windows, window)
	sm.sessions[session.ID] = session

	return session, nil
}

// GetSession 获取会话
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return session, nil
}

// ListSessions 列出所有会话
func (sm *SessionManager) ListSessions() []*Session {
	sessions := make([]*Session, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// AttachSession 连接到会话
func (sm *SessionManager) AttachSession(sessionID string) error {
	session, err := sm.GetSession(sessionID)
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
func (sm *SessionManager) DetachSession(sessionID string) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.Status = SessionDetached
	session.LastActive = time.Now()

	return nil
}

// KillSession 销毁会话
func (sm *SessionManager) KillSession(sessionID string) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	// 关闭所有窗口
	for _, window := range session.Windows {
		if err := sm.closeWindow(session, window.Index); err != nil {
			// 记录错误但继续
			fmt.Printf("Warning: failed to close window %d: %v\n", window.Index, err)
		}
	}

	session.Status = SessionDestroyed
	delete(sm.sessions, sessionID)

	return nil
}

// CreateWindow 创建新窗口
func (sm *SessionManager) CreateWindow(sessionID, name string) (*Window, error) {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	window, err := sm.createWindow(session, name)
	if err != nil {
		return nil, err
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.Windows = append(session.Windows, window)
	session.ActiveWindow = len(session.Windows) - 1
	session.LastActive = time.Now()

	return window, nil
}

// createWindow 内部创建窗口方法
func (sm *SessionManager) createWindow(session *Session, name string) (*Window, error) {
	if name == "" {
		name = fmt.Sprintf("window-%d", len(session.Windows))
	}

	window := &Window{
		ID:         uuid.New().String(),
		Name:       name,
		Index:      len(session.Windows),
		Panes:      make([]*Pane, 0),
		ActivePane: 0,
		Layout:     LayoutMainVertical,
		CreatedAt:  time.Now(),
	}

	// 创建默认面板
	pane, err := sm.createPane(window, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create default pane: %v", err)
	}

	window.Panes = append(window.Panes, pane)
	return window, nil
}

// CloseWindow 关闭窗口
func (sm *SessionManager) CloseWindow(sessionID string, windowIndex int) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	return sm.closeWindow(session, windowIndex)
}

// closeWindow 内部关闭窗口方法
func (sm *SessionManager) closeWindow(session *Session, windowIndex int) error {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	if windowIndex < 0 || windowIndex >= len(session.Windows) {
		return fmt.Errorf("window index out of range: %d", windowIndex)
	}

	window := session.Windows[windowIndex]

	// 关闭所有面板
	for _, pane := range window.Panes {
		if err := sm.closePane(window, pane.Index); err != nil {
			fmt.Printf("Warning: failed to close pane %d: %v\n", pane.Index, err)
		}
	}

	// 从会话中移除窗口
	session.Windows = append(session.Windows[:windowIndex], session.Windows[windowIndex+1:]...)

	// 重新索引窗口
	for i, w := range session.Windows {
		w.Index = i
	}

	// 调整活动窗口索引
	if session.ActiveWindow >= len(session.Windows) {
		session.ActiveWindow = len(session.Windows) - 1
	}
	if session.ActiveWindow < 0 {
		session.ActiveWindow = 0
	}

	session.LastActive = time.Now()
	return nil
}

// SwitchWindow 切换窗口
func (sm *SessionManager) SwitchWindow(sessionID string, windowIndex int) error {
	session, err := sm.GetSession(sessionID)
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
func (sm *SessionManager) SplitPane(sessionID string, windowIndex int, direction string) (*Pane, error) {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	if windowIndex < 0 || windowIndex >= len(session.Windows) {
		return nil, fmt.Errorf("window index out of range: %d", windowIndex)
	}

	window := session.Windows[windowIndex]

	// 创建新面板
	pane, err := sm.createPane(window, "")
	if err != nil {
		return nil, err
	}

	window.mutex.Lock()
	defer window.mutex.Unlock()

	window.Panes = append(window.Panes, pane)
	window.ActivePane = len(window.Panes) - 1

	// 重新计算布局
	sm.recalculateLayout(window)

	session.LastActive = time.Now()
	return pane, nil
}

// createPane 创建面板
func (sm *SessionManager) createPane(window *Window, command string) (*Pane, error) {
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
			MaxLines: sm.config.BufferSize,
			CursorX:  0,
			CursorY:  0,
		},
	}

	return pane, nil
}

// ClosePane 关闭面板
func (sm *SessionManager) ClosePane(sessionID string, windowIndex, paneIndex int) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	if windowIndex < 0 || windowIndex >= len(session.Windows) {
		return fmt.Errorf("window index out of range: %d", windowIndex)
	}

	window := session.Windows[windowIndex]
	return sm.closePane(window, paneIndex)
}

// closePane 内部关闭面板方法
func (sm *SessionManager) closePane(window *Window, paneIndex int) error {
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
	sm.recalculateLayout(window)

	return nil
}

// SwitchPane 切换面板
func (sm *SessionManager) SwitchPane(sessionID string, windowIndex, paneIndex int) error {
	session, err := sm.GetSession(sessionID)
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
func (sm *SessionManager) recalculateLayout(window *Window) {
	if len(window.Panes) == 0 {
		return
	}

	// 假设终端大小为 80x24（这应该从实际终端获取）
	termWidth, termHeight := 80, 24

	switch window.Layout {
	case LayoutEven:
		sm.layoutEven(window.Panes, termWidth, termHeight)
	case LayoutMainVertical:
		sm.layoutMainVertical(window.Panes, termWidth, termHeight)
	case LayoutMainHorizontal:
		sm.layoutMainHorizontal(window.Panes, termWidth, termHeight)
	case LayoutTiled:
		sm.layoutTiled(window.Panes, termWidth, termHeight)
	default:
		sm.layoutEven(window.Panes, termWidth, termHeight)
	}
}

// layoutEven 均匀布局
func (sm *SessionManager) layoutEven(panes []*Pane, width, height int) {
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
func (sm *SessionManager) layoutMainVertical(panes []*Pane, width, height int) {
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
func (sm *SessionManager) layoutMainHorizontal(panes []*Pane, width, height int) {
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
func (sm *SessionManager) layoutTiled(panes []*Pane, width, height int) {
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
func (sm *SessionManager) RenameSession(sessionID, newName string) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	// 检查新名称是否已存在
	for _, s := range sm.sessions {
		if s.Name == newName && s.ID != sessionID {
			return fmt.Errorf("session with name '%s' already exists", newName)
		}
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.Name = newName
	session.LastActive = time.Now()

	return nil
}

// RenameWindow 重命名窗口
func (sm *SessionManager) RenameWindow(sessionID string, windowIndex int, newName string) error {
	session, err := sm.GetSession(sessionID)
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
func (sm *SessionManager) GetSessionByName(name string) (*Session, error) {
	for _, session := range sm.sessions {
		if session.Name == name {
			return session, nil
		}
	}
	return nil, fmt.Errorf("session not found: %s", name)
}

// SaveSession 保存会话状态
func (sm *SessionManager) SaveSession(sessionID string, filepath string) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	// 这里可以实现会话状态的序列化保存
	// 可以保存为 JSON 或其他格式
	_ = filepath
	_ = session

	return fmt.Errorf("not implemented yet")
}

// LoadSession 加载会话状态
func (sm *SessionManager) LoadSession(filepath string) (*Session, error) {
	// 这里可以实现会话状态的反序列化加载
	_ = filepath

	return nil, fmt.Errorf("not implemented yet")
}
