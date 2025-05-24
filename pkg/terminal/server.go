package terminal

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"go.uber.org/zap"
)

// TerminalServer 终端服务器
type TerminalServer struct {
	config         *TerminalConfig
	sessionManager *SessionManager
	listener       net.Listener
	clients        map[string]*ClientConnection
	socketPath     string
	running        bool
	mutex          sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
}

// ClientConnection 客户端连接
type ClientConnection struct {
	ID         string
	Conn       net.Conn
	SessionID  string
	LastActive time.Time
}

// NewTerminalServer 创建终端服务器
func NewTerminalServer(config *TerminalConfig) *TerminalServer {
	if config == nil {
		config = DefaultConfig
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建socket路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp"
	}
	socketDir := filepath.Join(homeDir, ".clixgo", "terminal")
	os.MkdirAll(socketDir, 0755)
	socketPath := filepath.Join(socketDir, "clixgo-terminal.sock")

	server := &TerminalServer{
		config:         config,
		sessionManager: NewSessionManager(config),
		clients:        make(map[string]*ClientConnection),
		socketPath:     socketPath,
		running:        false,
		ctx:            ctx,
		cancel:         cancel,
	}

	return server
}

// Start 启动服务器
func (ts *TerminalServer) Start() error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	if ts.running {
		return fmt.Errorf("server is already running")
	}

	// 删除已存在的socket文件
	if err := os.Remove(ts.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing socket: %v", err)
	}

	// 创建Unix domain socket监听器
	listener, err := net.Listen("unix", ts.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create listener: %v", err)
	}

	ts.listener = listener
	ts.running = true

	logger.Info("Terminal server started", zap.String("socket", ts.socketPath))

	// 启动后台goroutine处理连接
	go ts.acceptConnections()

	// 启动自动保存goroutine
	if ts.config.AutoSave {
		go ts.autoSave()
	}

	return nil
}

// Stop 停止服务器
func (ts *TerminalServer) Stop() error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	if !ts.running {
		return fmt.Errorf("server is not running")
	}

	ts.cancel()
	ts.running = false

	// 关闭所有客户端连接
	for _, client := range ts.clients {
		client.Conn.Close()
	}

	// 关闭监听器
	if ts.listener != nil {
		ts.listener.Close()
	}

	// 删除socket文件
	os.Remove(ts.socketPath)

	logger.Info("Terminal server stopped")
	return nil
}

// acceptConnections 接受客户端连接
func (ts *TerminalServer) acceptConnections() {
	for {
		select {
		case <-ts.ctx.Done():
			return
		default:
			conn, err := ts.listener.Accept()
			if err != nil {
				if ts.running {
					logger.Error("Failed to accept connection", zap.Error(err))
				}
				continue
			}

			go ts.handleClient(conn)
		}
	}
}

// handleClient 处理客户端连接
func (ts *TerminalServer) handleClient(conn net.Conn) {
	defer conn.Close()

	clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())
	client := &ClientConnection{
		ID:         clientID,
		Conn:       conn,
		LastActive: time.Now(),
	}

	ts.mutex.Lock()
	ts.clients[clientID] = client
	ts.mutex.Unlock()

	defer func() {
		ts.mutex.Lock()
		delete(ts.clients, clientID)
		ts.mutex.Unlock()
	}()

	logger.Info("Client connected", zap.String("client_id", clientID))

	// 处理客户端消息
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-ts.ctx.Done():
			return
		default:
			var cmd Command
			if err := decoder.Decode(&cmd); err != nil {
				logger.Error("Failed to decode command", zap.Error(err))
				return
			}

			client.LastActive = time.Now()
			response := ts.handleCommand(client, &cmd)

			if err := encoder.Encode(response); err != nil {
				logger.Error("Failed to send response", zap.Error(err))
				return
			}
		}
	}
}

// handleCommand 处理客户端命令
func (ts *TerminalServer) handleCommand(client *ClientConnection, cmd *Command) interface{} {
	switch cmd.Type {
	case CmdCreateSession:
		return ts.handleCreateSession(client, cmd.Payload)
	case CmdAttachSession:
		return ts.handleAttachSession(client, cmd.Payload)
	case CmdDetachSession:
		return ts.handleDetachSession(client, cmd.Payload)
	case CmdListSessions:
		return ts.handleListSessions(client, cmd.Payload)
	case CmdCreateWindow:
		return ts.handleCreateWindow(client, cmd.Payload)
	case CmdCloseWindow:
		return ts.handleCloseWindow(client, cmd.Payload)
	case CmdSplitPane:
		return ts.handleSplitPane(client, cmd.Payload)
	case CmdClosePane:
		return ts.handleClosePane(client, cmd.Payload)
	case CmdSwitchWindow:
		return ts.handleSwitchWindow(client, cmd.Payload)
	case CmdSwitchPane:
		return ts.handleSwitchPane(client, cmd.Payload)
	case CmdSendKeys:
		return ts.handleSendKeys(client, cmd.Payload)
	case CmdRename:
		return ts.handleRename(client, cmd.Payload)
	case CmdKillSession:
		return ts.handleKillSession(client, cmd.Payload)
	default:
		return map[string]interface{}{
			"error": fmt.Sprintf("unknown command type: %s", cmd.Type),
		}
	}
}

// handleCreateSession 处理创建会话命令
func (ts *TerminalServer) handleCreateSession(client *ClientConnection, payload interface{}) interface{} {
	data, ok := payload.(map[string]interface{})
	if !ok {
		return map[string]interface{}{"error": "invalid payload"}
	}

	name, _ := data["name"].(string)
	session, err := ts.sessionManager.CreateSession(name)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	client.SessionID = session.ID
	logger.Info("Session created", zap.String("session_id", session.ID), zap.String("name", session.Name))

	return map[string]interface{}{
		"success":    true,
		"session_id": session.ID,
		"session":    session,
	}
}

// handleAttachSession 处理连接会话命令
func (ts *TerminalServer) handleAttachSession(client *ClientConnection, payload interface{}) interface{} {
	data, ok := payload.(map[string]interface{})
	if !ok {
		return map[string]interface{}{"error": "invalid payload"}
	}

	sessionID, _ := data["session_id"].(string)
	if sessionID == "" {
		sessionName, _ := data["session_name"].(string)
		if sessionName != "" {
			session, err := ts.sessionManager.GetSessionByName(sessionName)
			if err != nil {
				return map[string]interface{}{"error": err.Error()}
			}
			sessionID = session.ID
		}
	}

	if sessionID == "" {
		return map[string]interface{}{"error": "session_id or session_name required"}
	}

	err := ts.sessionManager.AttachSession(sessionID)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	client.SessionID = sessionID
	session, _ := ts.sessionManager.GetSession(sessionID)

	logger.Info("Session attached", zap.String("session_id", sessionID))

	return map[string]interface{}{
		"success": true,
		"session": session,
	}
}

// handleDetachSession 处理断开会话命令
func (ts *TerminalServer) handleDetachSession(client *ClientConnection, payload interface{}) interface{} {
	if client.SessionID == "" {
		return map[string]interface{}{"error": "no active session"}
	}

	err := ts.sessionManager.DetachSession(client.SessionID)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	logger.Info("Session detached", zap.String("session_id", client.SessionID))
	client.SessionID = ""

	return map[string]interface{}{
		"success": true,
	}
}

// handleListSessions 处理列出会话命令
func (ts *TerminalServer) handleListSessions(client *ClientConnection, payload interface{}) interface{} {
	sessions := ts.sessionManager.ListSessions()
	return map[string]interface{}{
		"success":  true,
		"sessions": sessions,
	}
}

// handleCreateWindow 处理创建窗口命令
func (ts *TerminalServer) handleCreateWindow(client *ClientConnection, payload interface{}) interface{} {
	if client.SessionID == "" {
		return map[string]interface{}{"error": "no active session"}
	}

	data, ok := payload.(map[string]interface{})
	if !ok {
		return map[string]interface{}{"error": "invalid payload"}
	}

	name, _ := data["name"].(string)
	window, err := ts.sessionManager.CreateWindow(client.SessionID, name)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"success": true,
		"window":  window,
	}
}

// handleCloseWindow 处理关闭窗口命令
func (ts *TerminalServer) handleCloseWindow(client *ClientConnection, payload interface{}) interface{} {
	if client.SessionID == "" {
		return map[string]interface{}{"error": "no active session"}
	}

	data, ok := payload.(map[string]interface{})
	if !ok {
		return map[string]interface{}{"error": "invalid payload"}
	}

	windowIndex, ok := data["window_index"].(float64)
	if !ok {
		return map[string]interface{}{"error": "window_index required"}
	}

	err := ts.sessionManager.CloseWindow(client.SessionID, int(windowIndex))
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"success": true,
	}
}

// handleSplitPane 处理分割面板命令
func (ts *TerminalServer) handleSplitPane(client *ClientConnection, payload interface{}) interface{} {
	if client.SessionID == "" {
		return map[string]interface{}{"error": "no active session"}
	}

	data, ok := payload.(map[string]interface{})
	if !ok {
		return map[string]interface{}{"error": "invalid payload"}
	}

	windowIndex, ok := data["window_index"].(float64)
	if !ok {
		return map[string]interface{}{"error": "window_index required"}
	}

	direction, _ := data["direction"].(string)
	if direction == "" {
		direction = "vertical"
	}

	pane, err := ts.sessionManager.SplitPane(client.SessionID, int(windowIndex), direction)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"success": true,
		"pane":    pane,
	}
}

// handleClosePane 处理关闭面板命令
func (ts *TerminalServer) handleClosePane(client *ClientConnection, payload interface{}) interface{} {
	if client.SessionID == "" {
		return map[string]interface{}{"error": "no active session"}
	}

	data, ok := payload.(map[string]interface{})
	if !ok {
		return map[string]interface{}{"error": "invalid payload"}
	}

	windowIndex, ok := data["window_index"].(float64)
	if !ok {
		return map[string]interface{}{"error": "window_index required"}
	}

	paneIndex, ok := data["pane_index"].(float64)
	if !ok {
		return map[string]interface{}{"error": "pane_index required"}
	}

	err := ts.sessionManager.ClosePane(client.SessionID, int(windowIndex), int(paneIndex))
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"success": true,
	}
}

// handleSwitchWindow 处理切换窗口命令
func (ts *TerminalServer) handleSwitchWindow(client *ClientConnection, payload interface{}) interface{} {
	if client.SessionID == "" {
		return map[string]interface{}{"error": "no active session"}
	}

	data, ok := payload.(map[string]interface{})
	if !ok {
		return map[string]interface{}{"error": "invalid payload"}
	}

	windowIndex, ok := data["window_index"].(float64)
	if !ok {
		return map[string]interface{}{"error": "window_index required"}
	}

	err := ts.sessionManager.SwitchWindow(client.SessionID, int(windowIndex))
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"success": true,
	}
}

// handleSwitchPane 处理切换面板命令
func (ts *TerminalServer) handleSwitchPane(client *ClientConnection, payload interface{}) interface{} {
	if client.SessionID == "" {
		return map[string]interface{}{"error": "no active session"}
	}

	data, ok := payload.(map[string]interface{})
	if !ok {
		return map[string]interface{}{"error": "invalid payload"}
	}

	windowIndex, ok := data["window_index"].(float64)
	if !ok {
		return map[string]interface{}{"error": "window_index required"}
	}

	paneIndex, ok := data["pane_index"].(float64)
	if !ok {
		return map[string]interface{}{"error": "pane_index required"}
	}

	err := ts.sessionManager.SwitchPane(client.SessionID, int(windowIndex), int(paneIndex))
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"success": true,
	}
}

// handleSendKeys 处理发送按键命令
func (ts *TerminalServer) handleSendKeys(client *ClientConnection, payload interface{}) interface{} {
	if client.SessionID == "" {
		return map[string]interface{}{"error": "no active session"}
	}

	data, ok := payload.(map[string]interface{})
	if !ok {
		return map[string]interface{}{"error": "invalid payload"}
	}

	keys, _ := data["keys"].(string)
	if keys == "" {
		return map[string]interface{}{"error": "keys required"}
	}

	// 这里应该将按键发送到活动面板
	// TODO: 实现按键发送逻辑

	return map[string]interface{}{
		"success": true,
	}
}

// handleRename 处理重命名命令
func (ts *TerminalServer) handleRename(client *ClientConnection, payload interface{}) interface{} {
	if client.SessionID == "" {
		return map[string]interface{}{"error": "no active session"}
	}

	data, ok := payload.(map[string]interface{})
	if !ok {
		return map[string]interface{}{"error": "invalid payload"}
	}

	target, _ := data["target"].(string)
	newName, _ := data["new_name"].(string)
	if newName == "" {
		return map[string]interface{}{"error": "new_name required"}
	}

	var err error
	switch target {
	case "session":
		err = ts.sessionManager.RenameSession(client.SessionID, newName)
	case "window":
		windowIndex, ok := data["window_index"].(float64)
		if !ok {
			return map[string]interface{}{"error": "window_index required for window rename"}
		}
		err = ts.sessionManager.RenameWindow(client.SessionID, int(windowIndex), newName)
	default:
		return map[string]interface{}{"error": "invalid target"}
	}

	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"success": true,
	}
}

// handleKillSession 处理销毁会话命令
func (ts *TerminalServer) handleKillSession(client *ClientConnection, payload interface{}) interface{} {
	data, ok := payload.(map[string]interface{})
	if !ok {
		return map[string]interface{}{"error": "invalid payload"}
	}

	sessionID, _ := data["session_id"].(string)
	if sessionID == "" {
		sessionID = client.SessionID
	}

	if sessionID == "" {
		return map[string]interface{}{"error": "session_id required"}
	}

	err := ts.sessionManager.KillSession(sessionID)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	if client.SessionID == sessionID {
		client.SessionID = ""
	}

	return map[string]interface{}{
		"success": true,
	}
}

// autoSave 自动保存会话状态
func (ts *TerminalServer) autoSave() {
	ticker := time.NewTicker(ts.config.SaveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ts.ctx.Done():
			return
		case <-ticker.C:
			ts.saveAllSessions()
		}
	}
}

// saveAllSessions 保存所有会话状态
func (ts *TerminalServer) saveAllSessions() {
	sessions := ts.sessionManager.ListSessions()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("Failed to get home directory", zap.Error(err))
		return
	}

	saveDir := filepath.Join(homeDir, ".clixgo", "terminal", "sessions")
	os.MkdirAll(saveDir, 0755)

	for _, session := range sessions {
		filePath := filepath.Join(saveDir, fmt.Sprintf("%s.json", session.ID))
		data, err := json.MarshalIndent(session, "", "  ")
		if err != nil {
			logger.Error("Failed to marshal session", zap.Error(err), zap.String("session_id", session.ID))
			continue
		}

		if err := os.WriteFile(filePath, data, 0644); err != nil {
			logger.Error("Failed to save session", zap.Error(err), zap.String("session_id", session.ID))
		}
	}
}

// IsRunning 检查服务器是否运行中
func (ts *TerminalServer) IsRunning() bool {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	return ts.running
}

// GetSocketPath 获取socket路径
func (ts *TerminalServer) GetSocketPath() string {
	return ts.socketPath
}

// GetClientCount 获取客户端连接数
func (ts *TerminalServer) GetClientCount() int {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	return len(ts.clients)
}

// GetSessionManager 获取会话管理器
func (ts *TerminalServer) GetSessionManager() *SessionManager {
	return ts.sessionManager
}
