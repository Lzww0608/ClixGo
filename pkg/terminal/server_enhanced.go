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

// EnhancedTerminalServer 增强的终端服务器
type EnhancedTerminalServer struct {
	config         *TerminalConfig
	sessionManager *SessionManager
	monitor        *PerformanceMonitor
	ptyManager     *SimplePTYManager
	uiRenderer     *UIRenderer
	listener       net.Listener
	clients        map[string]*ClientConnection
	socketPath     string
	running        bool
	mutex          sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	stats          *ServerStats
}

// ServerStats 服务器统计信息
type ServerStats struct {
	StartTime        time.Time           `json:"start_time"`
	TotalConnections int64               `json:"total_connections"`
	ActiveClients    int                 `json:"active_clients"`
	CommandsHandled  int64               `json:"commands_handled"`
	ErrorCount       int64               `json:"error_count"`
	PerformanceData  *PerformanceMetrics `json:"performance_data"`
	mutex            sync.RWMutex
}

// NewEnhancedTerminalServer 创建增强的终端服务器
func NewEnhancedTerminalServer(config *TerminalConfig) *EnhancedTerminalServer {
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
	socketPath := filepath.Join(socketDir, "clixgo-terminal-enhanced.sock")

	sessionManager := NewSessionManager(config)

	server := &EnhancedTerminalServer{
		config:         config,
		sessionManager: sessionManager,
		clients:        make(map[string]*ClientConnection),
		socketPath:     socketPath,
		running:        false,
		ctx:            ctx,
		cancel:         cancel,
		stats:          &ServerStats{StartTime: time.Now()},
	}

	// 创建组件
	server.monitor = NewPerformanceMonitor(config, sessionManager)
	server.ptyManager = NewSimplePTYManager(config)
	server.uiRenderer = NewUIRenderer(120, 40, nil)

	return server
}

// Start 启动增强服务器
func (es *EnhancedTerminalServer) Start() error {
	es.mutex.Lock()
	defer es.mutex.Unlock()

	if es.running {
		return fmt.Errorf("enhanced server is already running")
	}

	// 删除已存在的socket文件
	if err := os.Remove(es.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing socket: %v", err)
	}

	// 创建Unix domain socket监听器
	listener, err := net.Listen("unix", es.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create listener: %v", err)
	}

	es.listener = listener
	es.running = true
	es.stats.StartTime = time.Now()

	logger.Info("Enhanced terminal server started",
		zap.String("socket", es.socketPath),
		zap.Bool("monitoring", true),
		zap.Bool("pty_support", true))

	// 启动后台组件
	go es.acceptConnections()

	if es.config.AutoSave {
		go es.autoSave()
	}

	// 启动性能监控器
	if err := es.monitor.Start(); err != nil {
		logger.Warn("Failed to start performance monitor", zap.Error(err))
	}

	// 启动统计收集
	go es.collectStats()

	return nil
}

// Stop 停止增强服务器
func (es *EnhancedTerminalServer) Stop() error {
	es.mutex.Lock()
	defer es.mutex.Unlock()

	if !es.running {
		return fmt.Errorf("enhanced server is not running")
	}

	es.cancel()
	es.running = false

	// 停止性能监控器
	if es.monitor != nil {
		es.monitor.Stop()
	}

	// 关闭所有客户端连接
	for _, client := range es.clients {
		client.Conn.Close()
	}

	// 关闭监听器
	if es.listener != nil {
		es.listener.Close()
	}

	// 删除socket文件
	os.Remove(es.socketPath)

	logger.Info("Enhanced terminal server stopped",
		zap.Duration("uptime", time.Since(es.stats.StartTime)))

	return nil
}

// acceptConnections 接受客户端连接
func (es *EnhancedTerminalServer) acceptConnections() {
	for {
		select {
		case <-es.ctx.Done():
			return
		default:
			conn, err := es.listener.Accept()
			if err != nil {
				if es.running {
					logger.Error("Failed to accept connection", zap.Error(err))
					es.incrementErrorCount()
				}
				continue
			}

			es.incrementTotalConnections()
			go es.handleClient(conn)
		}
	}
}

// handleClient 处理客户端连接
func (es *EnhancedTerminalServer) handleClient(conn net.Conn) {
	defer conn.Close()

	clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())
	client := &ClientConnection{
		ID:         clientID,
		Conn:       conn,
		LastActive: time.Now(),
	}

	es.mutex.Lock()
	es.clients[clientID] = client
	es.mutex.Unlock()

	defer func() {
		es.mutex.Lock()
		delete(es.clients, clientID)
		es.mutex.Unlock()
	}()

	logger.Info("Enhanced client connected",
		zap.String("client_id", clientID),
		zap.Int("total_clients", len(es.clients)))

	// 处理客户端消息
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-es.ctx.Done():
			return
		default:
			var cmd Command
			if err := decoder.Decode(&cmd); err != nil {
				logger.Error("Failed to decode command", zap.Error(err))
				es.incrementErrorCount()
				return
			}

			client.LastActive = time.Now()
			response := es.handleEnhancedCommand(client, &cmd)
			es.incrementCommandsHandled()

			if err := encoder.Encode(response); err != nil {
				logger.Error("Failed to send response", zap.Error(err))
				es.incrementErrorCount()
				return
			}
		}
	}
}

// handleEnhancedCommand 处理增强的客户端命令
func (es *EnhancedTerminalServer) handleEnhancedCommand(client *ClientConnection, cmd *Command) interface{} {
	switch cmd.Type {
	case "get_performance_metrics":
		return es.handleGetPerformanceMetrics(client, cmd.Payload)
	case "get_server_stats":
		return es.handleGetServerStats(client, cmd.Payload)
	case "render_ui":
		return es.handleRenderUI(client, cmd.Payload)
	case "create_pty":
		return es.handleCreatePTY(client, cmd.Payload)
	case "get_pty_output":
		return es.handleGetPTYOutput(client, cmd.Payload)
	case "send_pty_input":
		return es.handleSendPTYInput(client, cmd.Payload)
	default:
		// 回退到基本命令处理
		return es.handleBasicCommand(client, cmd)
	}
}

// handleGetPerformanceMetrics 处理获取性能指标请求
func (es *EnhancedTerminalServer) handleGetPerformanceMetrics(client *ClientConnection, payload interface{}) interface{} {
	if es.monitor == nil {
		return map[string]interface{}{
			"error": "performance monitor not available",
		}
	}

	metrics := es.monitor.GetMetrics()
	summary := es.monitor.GetSummary()

	return map[string]interface{}{
		"success": true,
		"metrics": metrics,
		"summary": summary,
	}
}

// handleGetServerStats 处理获取服务器统计信息请求
func (es *EnhancedTerminalServer) handleGetServerStats(client *ClientConnection, payload interface{}) interface{} {
	es.stats.mutex.RLock()
	defer es.stats.mutex.RUnlock()

	if es.monitor != nil {
		es.stats.PerformanceData = es.monitor.GetMetrics()
	}

	es.stats.ActiveClients = len(es.clients)

	return map[string]interface{}{
		"success": true,
		"stats":   es.stats,
	}
}

// handleRenderUI 处理UI渲染请求
func (es *EnhancedTerminalServer) handleRenderUI(client *ClientConnection, payload interface{}) interface{} {
	data, ok := payload.(map[string]interface{})
	if !ok {
		return map[string]interface{}{
			"error": "invalid payload for render_ui",
		}
	}

	sessionID, exists := data["session_id"].(string)
	if !exists {
		return map[string]interface{}{
			"error": "session_id required",
		}
	}

	session, err := es.sessionManager.GetSession(sessionID)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("session not found: %v", err),
		}
	}

	if len(session.Windows) == 0 {
		return map[string]interface{}{
			"error": "no windows in session",
		}
	}

	// 渲染当前活动窗口
	window := session.Windows[session.ActiveWindow]
	output := es.uiRenderer.RenderWindow(window)

	return map[string]interface{}{
		"success": true,
		"output":  output,
		"window":  window.Name,
	}
}

// handleCreatePTY 处理创建PTY请求
func (es *EnhancedTerminalServer) handleCreatePTY(client *ClientConnection, payload interface{}) interface{} {
	data, ok := payload.(map[string]interface{})
	if !ok {
		return map[string]interface{}{
			"error": "invalid payload for create_pty",
		}
	}

	ptyID, _ := data["pty_id"].(string)
	command, _ := data["command"].(string)
	workingDir, _ := data["working_dir"].(string)
	width, _ := data["width"].(float64)
	height, _ := data["height"].(float64)

	if ptyID == "" {
		ptyID = fmt.Sprintf("pty-%d", time.Now().UnixNano())
	}

	pty, err := es.ptyManager.CreateSimplePTY(ptyID, command, workingDir, int(width), int(height))
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("failed to create PTY: %v", err),
		}
	}

	if err := pty.Start(); err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("failed to start PTY: %v", err),
		}
	}

	return map[string]interface{}{
		"success": true,
		"pty_id":  ptyID,
		"pid":     pty.GetPID(),
	}
}

// handleGetPTYOutput 处理获取PTY输出请求
func (es *EnhancedTerminalServer) handleGetPTYOutput(client *ClientConnection, payload interface{}) interface{} {
	data, ok := payload.(map[string]interface{})
	if !ok {
		return map[string]interface{}{
			"error": "invalid payload for get_pty_output",
		}
	}

	ptyID, exists := data["pty_id"].(string)
	if !exists {
		return map[string]interface{}{
			"error": "pty_id required",
		}
	}

	pty, err := es.ptyManager.GetSimplePTY(ptyID)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("PTY not found: %v", err),
		}
	}

	output, err := pty.Read()
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("failed to read PTY output: %v", err),
		}
	}

	return map[string]interface{}{
		"success": true,
		"output":  string(output),
		"running": pty.IsRunning(),
	}
}

// handleSendPTYInput 处理发送PTY输入请求
func (es *EnhancedTerminalServer) handleSendPTYInput(client *ClientConnection, payload interface{}) interface{} {
	data, ok := payload.(map[string]interface{})
	if !ok {
		return map[string]interface{}{
			"error": "invalid payload for send_pty_input",
		}
	}

	ptyID, exists := data["pty_id"].(string)
	if !exists {
		return map[string]interface{}{
			"error": "pty_id required",
		}
	}

	input, exists := data["input"].(string)
	if !exists {
		return map[string]interface{}{
			"error": "input required",
		}
	}

	pty, err := es.ptyManager.GetSimplePTY(ptyID)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("PTY not found: %v", err),
		}
	}

	if err := pty.Write([]byte(input)); err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("failed to write to PTY: %v", err),
		}
	}

	return map[string]interface{}{
		"success": true,
	}
}

// handleBasicCommand 处理基本命令（回退到原始实现）
func (es *EnhancedTerminalServer) handleBasicCommand(client *ClientConnection, cmd *Command) interface{} {
	// 这里可以委托给原始的服务器实现
	// 简化实现，直接返回not implemented
	return map[string]interface{}{
		"error": fmt.Sprintf("command type '%s' not implemented in enhanced server", cmd.Type),
	}
}

// collectStats 收集服务器统计信息
func (es *EnhancedTerminalServer) collectStats() {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-es.ctx.Done():
			return
		case <-ticker.C:
			es.updateStats()
		}
	}
}

// updateStats 更新统计信息
func (es *EnhancedTerminalServer) updateStats() {
	es.stats.mutex.Lock()
	defer es.stats.mutex.Unlock()

	es.stats.ActiveClients = len(es.clients)

	if es.monitor != nil {
		es.stats.PerformanceData = es.monitor.GetMetrics()
	}
}

// autoSave 自动保存功能
func (es *EnhancedTerminalServer) autoSave() {
	if es.config.SaveInterval <= 0 {
		return
	}

	ticker := time.NewTicker(es.config.SaveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-es.ctx.Done():
			return
		case <-ticker.C:
			es.saveAllSessions()
		}
	}
}

// saveAllSessions 保存所有会话
func (es *EnhancedTerminalServer) saveAllSessions() {
	sessions := es.sessionManager.ListSessions()
	logger.Info("Auto-saving sessions", zap.Int("count", len(sessions)))

	// TODO: 实现实际的保存逻辑
}

// 统计信息相关方法
func (es *EnhancedTerminalServer) incrementTotalConnections() {
	es.stats.mutex.Lock()
	defer es.stats.mutex.Unlock()
	es.stats.TotalConnections++
}

func (es *EnhancedTerminalServer) incrementCommandsHandled() {
	es.stats.mutex.Lock()
	defer es.stats.mutex.Unlock()
	es.stats.CommandsHandled++
}

func (es *EnhancedTerminalServer) incrementErrorCount() {
	es.stats.mutex.Lock()
	defer es.stats.mutex.Unlock()
	es.stats.ErrorCount++
}

// 公共访问方法
func (es *EnhancedTerminalServer) IsRunning() bool {
	es.mutex.RLock()
	defer es.mutex.RUnlock()
	return es.running
}

func (es *EnhancedTerminalServer) GetSocketPath() string {
	return es.socketPath
}

func (es *EnhancedTerminalServer) GetSessionManager() *SessionManager {
	return es.sessionManager
}

func (es *EnhancedTerminalServer) GetPerformanceMonitor() *PerformanceMonitor {
	return es.monitor
}

func (es *EnhancedTerminalServer) GetStats() *ServerStats {
	es.stats.mutex.RLock()
	defer es.stats.mutex.RUnlock()

	// 返回副本
	stats := *es.stats
	return &stats
}
