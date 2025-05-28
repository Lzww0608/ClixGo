package terminal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"go.uber.org/zap"
)

// PersistenceManager 会话持久化管理器
type PersistenceManager struct {
	dataDir    string
	autoSave   bool
	saveFormat string // json, gob
}

// SessionSnapshot 会话快照
type SessionSnapshot struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Status       SessionStatus          `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	LastActive   time.Time              `json:"last_active"`
	SavedAt      time.Time              `json:"saved_at"`
	Windows      []*WindowSnapshot      `json:"windows"`
	ActiveWindow int                    `json:"active_window"`
	Environment  map[string]string      `json:"environment"`
	WorkingDir   string                 `json:"working_dir"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// WindowSnapshot 窗口快照
type WindowSnapshot struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Index      int             `json:"index"`
	Panes      []*PaneSnapshot `json:"panes"`
	ActivePane int             `json:"active_pane"`
	Layout     Layout          `json:"layout"`
	CreatedAt  time.Time       `json:"created_at"`
	Size       *WindowSize     `json:"size"`
}

// PaneSnapshot 面板快照
type PaneSnapshot struct {
	ID          string            `json:"id"`
	Index       int               `json:"index"`
	X           int               `json:"x"`
	Y           int               `json:"y"`
	Width       int               `json:"width"`
	Height      int               `json:"height"`
	Command     string            `json:"command"`
	WorkingDir  string            `json:"working_dir"`
	ProcessID   int               `json:"process_id"`
	ProcessName string            `json:"process_name"`
	Active      bool              `json:"active"`
	CreatedAt   time.Time         `json:"created_at"`
	LastOutput  time.Time         `json:"last_output"`
	BufferLines []string          `json:"buffer_lines"`
	CursorPos   *CursorPosition   `json:"cursor_pos"`
	Environment map[string]string `json:"environment"`
	History     []string          `json:"history"`
}

// WindowSize 窗口尺寸
type WindowSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// CursorPosition 光标位置
type CursorPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// PersistenceConfig 持久化配置
type PersistenceConfig struct {
	DataDir         string        `json:"data_dir"`
	AutoSave        bool          `json:"auto_save"`
	SaveInterval    time.Duration `json:"save_interval"`
	MaxSnapshots    int           `json:"max_snapshots"`
	CompressData    bool          `json:"compress_data"`
	SaveBufferLines int           `json:"save_buffer_lines"`
	SaveHistory     bool          `json:"save_history"`
	SaveEnvironment bool          `json:"save_environment"`
}

// NewPersistenceManager 创建持久化管理器
func NewPersistenceManager(config *PersistenceConfig) (*PersistenceManager, error) {
	if config == nil {
		config = DefaultPersistenceConfig()
	}

	// 确保数据目录存在
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	pm := &PersistenceManager{
		dataDir:    config.DataDir,
		autoSave:   config.AutoSave,
		saveFormat: "json",
	}

	logger.Info("持久化管理器初始化完成",
		zap.String("data_dir", config.DataDir),
		zap.Bool("auto_save", config.AutoSave))

	return pm, nil
}

// DefaultPersistenceConfig 默认持久化配置
func DefaultPersistenceConfig() *PersistenceConfig {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".clixgo", "sessions")

	return &PersistenceConfig{
		DataDir:         dataDir,
		AutoSave:        true,
		SaveInterval:    time.Minute * 5,
		MaxSnapshots:    10,
		CompressData:    false,
		SaveBufferLines: 1000,
		SaveHistory:     true,
		SaveEnvironment: true,
	}
}

// SaveSession 保存会话快照
func (pm *PersistenceManager) SaveSession(session *Session) error {
	if session == nil {
		return fmt.Errorf("会话为空")
	}

	logger.Info("开始保存会话快照",
		zap.String("session_id", session.ID),
		zap.String("session_name", session.Name))

	// 创建会话快照
	snapshot, err := pm.createSessionSnapshot(session)
	if err != nil {
		return fmt.Errorf("创建会话快照失败: %w", err)
	}

	// 生成文件路径
	filename := fmt.Sprintf("%s_%s.json", session.Name, time.Now().Format("20060102_150405"))
	filepath := filepath.Join(pm.dataDir, filename)

	// 序列化快照
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化会话快照失败: %w", err)
	}

	// 写入文件
	if err := ioutil.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("写入会话快照文件失败: %w", err)
	}

	logger.Info("会话快照保存成功",
		zap.String("session_id", session.ID),
		zap.String("filepath", filepath),
		zap.Int("data_size", len(data)))

	// 清理旧快照
	go pm.cleanupOldSnapshots(session.Name)

	return nil
}

// LoadSession 加载会话快照
func (pm *PersistenceManager) LoadSession(sessionName string) (*SessionSnapshot, error) {
	logger.Info("开始加载会话快照", zap.String("session_name", sessionName))

	// 查找最新的快照文件
	filepath, err := pm.findLatestSnapshot(sessionName)
	if err != nil {
		return nil, fmt.Errorf("查找会话快照失败: %w", err)
	}

	// 读取文件
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("读取会话快照文件失败: %w", err)
	}

	// 反序列化快照
	var snapshot SessionSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("反序列化会话快照失败: %w", err)
	}

	logger.Info("会话快照加载成功",
		zap.String("session_name", sessionName),
		zap.String("filepath", filepath),
		zap.Time("saved_at", snapshot.SavedAt))

	return &snapshot, nil
}

// RestoreSession 恢复会话
func (pm *PersistenceManager) RestoreSession(snapshot *SessionSnapshot, sm *SessionManager) (*Session, error) {
	if snapshot == nil {
		return nil, fmt.Errorf("快照为空")
	}

	logger.Info("开始恢复会话",
		zap.String("session_id", snapshot.ID),
		zap.String("session_name", snapshot.Name))

	// 创建新会话
	session := &Session{
		ID:           snapshot.ID,
		Name:         snapshot.Name,
		Status:       SessionActive,
		CreatedAt:    snapshot.CreatedAt,
		LastActive:   time.Now(),
		Windows:      make([]*Window, 0),
		ActiveWindow: snapshot.ActiveWindow,
	}

	// 恢复窗口
	for _, windowSnapshot := range snapshot.Windows {
		window, err := pm.restoreWindow(windowSnapshot, session, sm)
		if err != nil {
			logger.Warn("恢复窗口失败",
				zap.String("window_id", windowSnapshot.ID),
				zap.Error(err))
			continue
		}
		session.Windows = append(session.Windows, window)
	}

	// 设置活动窗口
	if snapshot.ActiveWindow >= 0 && snapshot.ActiveWindow < len(session.Windows) {
		session.ActiveWindow = snapshot.ActiveWindow
	}

	logger.Info("会话恢复完成",
		zap.String("session_id", session.ID),
		zap.Int("windows_count", len(session.Windows)))

	return session, nil
}

// createSessionSnapshot 创建会话快照
func (pm *PersistenceManager) createSessionSnapshot(session *Session) (*SessionSnapshot, error) {
	session.mutex.RLock()
	defer session.mutex.RUnlock()

	snapshot := &SessionSnapshot{
		ID:           session.ID,
		Name:         session.Name,
		Status:       session.Status,
		CreatedAt:    session.CreatedAt,
		LastActive:   session.LastActive,
		SavedAt:      time.Now(),
		ActiveWindow: session.ActiveWindow,
		Environment:  make(map[string]string),
		Metadata:     make(map[string]interface{}),
	}

	// 获取环境变量
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			snapshot.Environment[parts[0]] = parts[1]
		}
	}

	// 获取工作目录
	if wd, err := os.Getwd(); err == nil {
		snapshot.WorkingDir = wd
	}

	// 创建窗口快照
	for _, window := range session.Windows {
		windowSnapshot, err := pm.createWindowSnapshot(window)
		if err != nil {
			logger.Warn("创建窗口快照失败",
				zap.String("window_id", window.ID),
				zap.Error(err))
			continue
		}
		snapshot.Windows = append(snapshot.Windows, windowSnapshot)
	}

	return snapshot, nil
}

// createWindowSnapshot 创建窗口快照
func (pm *PersistenceManager) createWindowSnapshot(window *Window) (*WindowSnapshot, error) {
	window.mutex.RLock()
	defer window.mutex.RUnlock()

	snapshot := &WindowSnapshot{
		ID:         window.ID,
		Name:       window.Name,
		Index:      window.Index,
		ActivePane: window.ActivePane,
		Layout:     window.Layout,
		CreatedAt:  window.CreatedAt,
		Size: &WindowSize{
			Width:  80, // 默认值，实际应该从终端获取
			Height: 24,
		},
	}

	// 创建面板快照
	for _, pane := range window.Panes {
		paneSnapshot, err := pm.createPaneSnapshot(pane)
		if err != nil {
			logger.Warn("创建面板快照失败",
				zap.String("pane_id", pane.ID),
				zap.Error(err))
			continue
		}
		snapshot.Panes = append(snapshot.Panes, paneSnapshot)
	}

	return snapshot, nil
}

// createPaneSnapshot 创建面板快照
func (pm *PersistenceManager) createPaneSnapshot(pane *Pane) (*PaneSnapshot, error) {
	pane.mutex.RLock()
	defer pane.mutex.RUnlock()

	snapshot := &PaneSnapshot{
		ID:          pane.ID,
		Index:       pane.Index,
		X:           pane.X,
		Y:           pane.Y,
		Width:       pane.Width,
		Height:      pane.Height,
		Command:     pane.Command,
		WorkingDir:  pane.WorkingDir,
		ProcessID:   pane.ProcessID,
		Active:      pane.Active,
		CreatedAt:   pane.CreatedAt,
		LastOutput:  pane.LastOutput,
		Environment: make(map[string]string),
		History:     make([]string, 0),
	}

	// 获取进程名称
	if pane.Process != nil {
		snapshot.ProcessName = pane.Command
	}

	// 获取缓冲区内容
	if pane.Buffer != nil {
		pane.Buffer.mutex.RLock()
		for i, line := range pane.Buffer.Lines {
			if i >= 1000 { // 限制保存的行数
				break
			}
			snapshot.BufferLines = append(snapshot.BufferLines, string(line))
		}
		snapshot.CursorPos = &CursorPosition{
			X: pane.Buffer.CursorX,
			Y: pane.Buffer.CursorY,
		}
		pane.Buffer.mutex.RUnlock()
	}

	return snapshot, nil
}

// restoreWindow 恢复窗口
func (pm *PersistenceManager) restoreWindow(snapshot *WindowSnapshot, session *Session, sm *SessionManager) (*Window, error) {
	window := &Window{
		ID:         snapshot.ID,
		Name:       snapshot.Name,
		Index:      snapshot.Index,
		Panes:      make([]*Pane, 0),
		ActivePane: snapshot.ActivePane,
		Layout:     snapshot.Layout,
		CreatedAt:  snapshot.CreatedAt,
	}

	// 恢复面板
	for _, paneSnapshot := range snapshot.Panes {
		pane, err := pm.restorePane(paneSnapshot, window, sm)
		if err != nil {
			logger.Warn("恢复面板失败",
				zap.String("pane_id", paneSnapshot.ID),
				zap.Error(err))
			continue
		}
		window.Panes = append(window.Panes, pane)
	}

	return window, nil
}

// restorePane 恢复面板
func (pm *PersistenceManager) restorePane(snapshot *PaneSnapshot, window *Window, sm *SessionManager) (*Pane, error) {
	pane := &Pane{
		ID:         snapshot.ID,
		Index:      snapshot.Index,
		X:          snapshot.X,
		Y:          snapshot.Y,
		Width:      snapshot.Width,
		Height:     snapshot.Height,
		Command:    snapshot.Command,
		WorkingDir: snapshot.WorkingDir,
		ProcessID:  snapshot.ProcessID,
		Active:     snapshot.Active,
		CreatedAt:  snapshot.CreatedAt,
		LastOutput: snapshot.LastOutput,
	}

	// 恢复缓冲区
	if len(snapshot.BufferLines) > 0 {
		pane.Buffer = &Buffer{
			Lines:    make([][]rune, 0),
			MaxLines: 2000,
			CursorX:  0,
			CursorY:  0,
		}

		if snapshot.CursorPos != nil {
			pane.Buffer.CursorX = snapshot.CursorPos.X
			pane.Buffer.CursorY = snapshot.CursorPos.Y
		}

		for _, line := range snapshot.BufferLines {
			pane.Buffer.Lines = append(pane.Buffer.Lines, []rune(line))
		}
	}

	// 注意：这里不恢复实际的进程，只恢复状态信息
	// 实际的进程恢复需要在更高层处理

	return pane, nil
}

// ListSnapshots 列出所有快照
func (pm *PersistenceManager) ListSnapshots() ([]string, error) {
	files, err := ioutil.ReadDir(pm.dataDir)
	if err != nil {
		return nil, fmt.Errorf("读取数据目录失败: %w", err)
	}

	var snapshots []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			snapshots = append(snapshots, file.Name())
		}
	}

	return snapshots, nil
}

// DeleteSnapshot 删除快照
func (pm *PersistenceManager) DeleteSnapshot(filename string) error {
	filepath := filepath.Join(pm.dataDir, filename)
	if err := os.Remove(filepath); err != nil {
		return fmt.Errorf("删除快照文件失败: %w", err)
	}

	logger.Info("快照删除成功", zap.String("filename", filename))
	return nil
}

// findLatestSnapshot 查找最新的快照文件
func (pm *PersistenceManager) findLatestSnapshot(sessionName string) (string, error) {
	files, err := ioutil.ReadDir(pm.dataDir)
	if err != nil {
		return "", fmt.Errorf("读取数据目录失败: %w", err)
	}

	var latestFile string
	var latestTime time.Time

	prefix := sessionName + "_"
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), prefix) && strings.HasSuffix(file.Name(), ".json") {
			if file.ModTime().After(latestTime) {
				latestTime = file.ModTime()
				latestFile = file.Name()
			}
		}
	}

	if latestFile == "" {
		return "", fmt.Errorf("未找到会话 %s 的快照", sessionName)
	}

	return filepath.Join(pm.dataDir, latestFile), nil
}

// cleanupOldSnapshots 清理旧快照
func (pm *PersistenceManager) cleanupOldSnapshots(sessionName string) {
	files, err := ioutil.ReadDir(pm.dataDir)
	if err != nil {
		logger.Error("读取数据目录失败", zap.Error(err))
		return
	}

	// 收集该会话的所有快照文件
	var sessionFiles []os.FileInfo
	prefix := sessionName + "_"
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), prefix) && strings.HasSuffix(file.Name(), ".json") {
			sessionFiles = append(sessionFiles, file)
		}
	}

	// 如果快照数量超过限制，删除最旧的
	maxSnapshots := 10 // 可配置
	if len(sessionFiles) > maxSnapshots {
		// 按修改时间排序
		for i := 0; i < len(sessionFiles)-1; i++ {
			for j := i + 1; j < len(sessionFiles); j++ {
				if sessionFiles[i].ModTime().After(sessionFiles[j].ModTime()) {
					sessionFiles[i], sessionFiles[j] = sessionFiles[j], sessionFiles[i]
				}
			}
		}

		// 删除最旧的文件
		for i := 0; i < len(sessionFiles)-maxSnapshots; i++ {
			filepath := filepath.Join(pm.dataDir, sessionFiles[i].Name())
			if err := os.Remove(filepath); err != nil {
				logger.Warn("删除旧快照失败",
					zap.String("filename", sessionFiles[i].Name()),
					zap.Error(err))
			} else {
				logger.Info("删除旧快照",
					zap.String("filename", sessionFiles[i].Name()))
			}
		}
	}
}

// GetSnapshotInfo 获取快照信息
func (pm *PersistenceManager) GetSnapshotInfo(filename string) (*SessionSnapshot, error) {
	filepath := filepath.Join(pm.dataDir, filename)

	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("读取快照文件失败: %w", err)
	}

	var snapshot SessionSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("解析快照文件失败: %w", err)
	}

	return &snapshot, nil
}
