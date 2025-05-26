package terminal

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/creack/pty"
	"go.uber.org/zap"
	"golang.org/x/term"
)

// CreackPTY 基于creack/pty库的PTY实现
type CreackPTY struct {
	ID          string
	Command     *exec.Cmd
	Process     *os.Process
	PTYFile     *os.File
	Width       int
	Height      int
	WorkingDir  string
	Environment []string
	running     bool
	mutex       sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	outputChan  chan []byte
	errorChan   chan error
	exitChan    chan int
}

// CreackPTYManager 基于creack/pty的PTY管理器
type CreackPTYManager struct {
	ptys   map[string]*CreackPTY
	mutex  sync.RWMutex
	config *TerminalConfig
}

// NewCreackPTYManager 创建基于creack/pty的PTY管理器
func NewCreackPTYManager(config *TerminalConfig) *CreackPTYManager {
	return &CreackPTYManager{
		ptys:   make(map[string]*CreackPTY),
		config: config,
	}
}

// CreateCreackPTY 创建新的CreackPTY
func (pm *CreackPTYManager) CreateCreackPTY(id, command, workingDir string, width, height int) (*CreackPTY, error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if _, exists := pm.ptys[id]; exists {
		return nil, fmt.Errorf("PTY with id %s already exists", id)
	}

	ptyInstance, err := pm.createCreackPTY(id, command, workingDir, width, height)
	if err != nil {
		return nil, err
	}

	pm.ptys[id] = ptyInstance
	logger.Info("Creack PTY created", zap.String("id", id), zap.String("command", command))

	return ptyInstance, nil
}

// GetCreackPTY 获取CreackPTY
func (pm *CreackPTYManager) GetCreackPTY(id string) (*CreackPTY, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	ptyInstance, exists := pm.ptys[id]
	if !exists {
		return nil, fmt.Errorf("Creack PTY not found: %s", id)
	}

	return ptyInstance, nil
}

// DestroyCreackPTY 销毁CreackPTY
func (pm *CreackPTYManager) DestroyCreackPTY(id string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	ptyInstance, exists := pm.ptys[id]
	if !exists {
		return fmt.Errorf("Creack PTY not found: %s", id)
	}

	if err := ptyInstance.Close(); err != nil {
		logger.Error("Failed to close Creack PTY", zap.Error(err))
	}

	delete(pm.ptys, id)
	logger.Info("Creack PTY destroyed", zap.String("id", id))

	return nil
}

// ListCreackPTYs 列出所有PTY
func (pm *CreackPTYManager) ListCreackPTYs() []string {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	ids := make([]string, 0, len(pm.ptys))
	for id := range pm.ptys {
		ids = append(ids, id)
	}
	return ids
}

// createCreackPTY 内部创建CreackPTY方法
func (pm *CreackPTYManager) createCreackPTY(id, command, workingDir string, width, height int) (*CreackPTY, error) {
	var cmd *exec.Cmd

	// 解析命令
	if command == "" {
		// 如果没有指定命令，启动交互式shell
		shellPath := os.Getenv("SHELL")
		if shellPath == "" {
			// 尝试常见的shell路径
			for _, path := range []string{"/bin/bash", "/usr/bin/bash", "/bin/sh", "/usr/bin/sh"} {
				if _, err := os.Stat(path); err == nil {
					shellPath = path
					break
				}
			}
			if shellPath == "" {
				return nil, fmt.Errorf("no shell found")
			}
		}
		cmd = exec.Command(shellPath)
	} else {
		// 如果指定了命令，使用shell执行
		shellPath := "/bin/bash"
		if _, err := os.Stat(shellPath); err != nil {
			shellPath = "/usr/bin/bash"
			if _, err := os.Stat(shellPath); err != nil {
				shellPath = "/bin/sh"
			}
		}
		cmd = exec.Command(shellPath, "-c", command)
	}

	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// 设置环境变量
	env := append(os.Environ(),
		fmt.Sprintf("TERM=%s", getTerminalType()),
		fmt.Sprintf("COLUMNS=%d", width),
		fmt.Sprintf("LINES=%d", height),
		"PS1=\\u@\\h:\\w\\$ ", // 设置提示符
	)
	cmd.Env = env

	ctx, cancel := context.WithCancel(context.Background())

	ptyInstance := &CreackPTY{
		ID:          id,
		Command:     cmd,
		Width:       width,
		Height:      height,
		WorkingDir:  workingDir,
		Environment: env,
		running:     false,
		ctx:         ctx,
		cancel:      cancel,
		outputChan:  make(chan []byte, 1024),
		errorChan:   make(chan error, 10),
		exitChan:    make(chan int, 1),
	}

	return ptyInstance, nil
}

// Start 启动CreackPTY进程
func (cp *CreackPTY) Start() error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if cp.running {
		return fmt.Errorf("Creack PTY is already running")
	}

	// 使用creack/pty启动命令
	ptmx, err := pty.StartWithSize(cp.Command, &pty.Winsize{
		Rows: uint16(cp.Height),
		Cols: uint16(cp.Width),
	})
	if err != nil {
		return fmt.Errorf("failed to start command with pty: %v", err)
	}

	cp.PTYFile = ptmx
	cp.Process = cp.Command.Process
	cp.running = true

	// 启动I/O处理goroutines
	go cp.handleOutput()
	go cp.handleSignals()
	go cp.waitForExit()

	logger.Info("Creack PTY process started",
		zap.String("id", cp.ID),
		zap.Int("pid", cp.Process.Pid),
		zap.Int("width", cp.Width),
		zap.Int("height", cp.Height))

	return nil
}

// Write 向CreackPTY写入数据
func (cp *CreackPTY) Write(data []byte) error {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()

	if !cp.running {
		return fmt.Errorf("Creack PTY is not running")
	}

	if cp.PTYFile == nil {
		return fmt.Errorf("PTY file is not available")
	}

	_, err := cp.PTYFile.Write(data)
	if err != nil {
		logger.Error("Failed to write to Creack PTY", zap.Error(err))
		return err
	}

	return nil
}

// Read 从CreackPTY读取数据
func (cp *CreackPTY) Read() ([]byte, error) {
	select {
	case data := <-cp.outputChan:
		return data, nil
	case err := <-cp.errorChan:
		return nil, err
	case <-cp.ctx.Done():
		return nil, fmt.Errorf("PTY context cancelled")
	case <-time.After(100 * time.Millisecond):
		return nil, fmt.Errorf("read timeout")
	}
}

// ReadWithTimeout 带超时的读取
func (cp *CreackPTY) ReadWithTimeout(timeout time.Duration) ([]byte, error) {
	select {
	case data := <-cp.outputChan:
		return data, nil
	case err := <-cp.errorChan:
		return nil, err
	case <-cp.ctx.Done():
		return nil, fmt.Errorf("PTY context cancelled")
	case <-time.After(timeout):
		return nil, fmt.Errorf("read timeout after %v", timeout)
	}
}

// Resize 调整CreackPTY大小
func (cp *CreackPTY) Resize(width, height int) error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if !cp.running {
		return fmt.Errorf("Creack PTY is not running")
	}

	if cp.PTYFile == nil {
		return fmt.Errorf("PTY file is not available")
	}

	// 使用creack/pty的Setsize函数
	err := pty.Setsize(cp.PTYFile, &pty.Winsize{
		Rows: uint16(height),
		Cols: uint16(width),
	})
	if err != nil {
		logger.Error("Failed to resize Creack PTY", zap.Error(err))
		return err
	}

	cp.Width = width
	cp.Height = height

	logger.Info("Creack PTY resized",
		zap.String("id", cp.ID),
		zap.Int("width", width),
		zap.Int("height", height))

	return nil
}

// Signal 向CreackPTY发送信号
func (cp *CreackPTY) Signal(sig os.Signal) error {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()

	if !cp.running || cp.Process == nil {
		return fmt.Errorf("Creack PTY is not running")
	}

	err := cp.Process.Signal(sig)
	if err != nil {
		logger.Error("Failed to send signal to Creack PTY", zap.Error(err))
		return err
	}

	logger.Info("Signal sent to Creack PTY",
		zap.String("id", cp.ID),
		zap.String("signal", sig.String()))

	return nil
}

// Close 关闭CreackPTY
func (cp *CreackPTY) Close() error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if !cp.running {
		return nil
	}

	cp.running = false
	cp.cancel()

	// 关闭PTY文件
	if cp.PTYFile != nil {
		if err := cp.PTYFile.Close(); err != nil {
			logger.Error("Failed to close PTY file", zap.Error(err))
		}
	}

	// 终止进程
	if cp.Process != nil {
		// 先尝试优雅关闭
		if err := cp.Process.Signal(syscall.SIGTERM); err != nil {
			logger.Warn("Failed to send SIGTERM", zap.Error(err))
		}

		// 等待一段时间后强制杀死
		done := make(chan error, 1)
		go func() {
			_, err := cp.Process.Wait()
			done <- err
		}()

		select {
		case <-done:
			// 进程已退出
		case <-time.After(5 * time.Second):
			// 超时，强制杀死
			if err := cp.Process.Kill(); err != nil {
				logger.Error("Failed to kill process", zap.Error(err))
			}
		}
	}

	// 关闭通道
	close(cp.outputChan)
	close(cp.errorChan)
	close(cp.exitChan)

	logger.Info("Creack PTY closed", zap.String("id", cp.ID))
	return nil
}

// IsRunning 检查CreackPTY是否正在运行
func (cp *CreackPTY) IsRunning() bool {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	return cp.running
}

// GetPID 获取进程ID
func (cp *CreackPTY) GetPID() int {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()

	if cp.Process != nil {
		return cp.Process.Pid
	}
	return -1
}

// GetSize 获取终端大小
func (cp *CreackPTY) GetSize() (int, int) {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	return cp.Width, cp.Height
}

// GetWorkingDir 获取工作目录
func (cp *CreackPTY) GetWorkingDir() string {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	return cp.WorkingDir
}

// SetRawMode 设置原始模式
func (cp *CreackPTY) SetRawMode() (*term.State, error) {
	if cp.PTYFile == nil {
		return nil, fmt.Errorf("PTY file is not available")
	}

	oldState, err := term.MakeRaw(int(cp.PTYFile.Fd()))
	if err != nil {
		return nil, fmt.Errorf("failed to set raw mode: %v", err)
	}

	return oldState, nil
}

// RestoreMode 恢复终端模式
func (cp *CreackPTY) RestoreMode(state *term.State) error {
	if cp.PTYFile == nil {
		return fmt.Errorf("PTY file is not available")
	}

	err := term.Restore(int(cp.PTYFile.Fd()), state)
	if err != nil {
		return fmt.Errorf("failed to restore terminal mode: %v", err)
	}

	return nil
}

// handleOutput 处理输出
func (cp *CreackPTY) handleOutput() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic in handleOutput", zap.Any("panic", r))
		}
	}()

	buffer := make([]byte, 4096)
	for {
		select {
		case <-cp.ctx.Done():
			return
		default:
			if cp.PTYFile == nil {
				return
			}

			n, err := cp.PTYFile.Read(buffer)
			if err != nil {
				if err == io.EOF {
					logger.Info("PTY output stream closed", zap.String("id", cp.ID))
					return
				}
				select {
				case cp.errorChan <- err:
				case <-cp.ctx.Done():
					return
				}
				continue
			}

			if n > 0 {
				data := make([]byte, n)
				copy(data, buffer[:n])

				select {
				case cp.outputChan <- data:
				case <-cp.ctx.Done():
					return
				}
			}
		}
	}
}

// handleSignals 处理信号
func (cp *CreackPTY) handleSignals() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic in handleSignals", zap.Any("panic", r))
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)
	defer signal.Stop(sigChan)

	for {
		select {
		case <-cp.ctx.Done():
			return
		case sig := <-sigChan:
			if sig == syscall.SIGWINCH {
				// 窗口大小改变信号
				if cp.PTYFile != nil {
					if err := pty.InheritSize(os.Stdin, cp.PTYFile); err != nil {
						logger.Warn("Failed to inherit size", zap.Error(err))
					}
				}
			}
		}
	}
}

// waitForExit 等待进程退出
func (cp *CreackPTY) waitForExit() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic in waitForExit", zap.Any("panic", r))
		}
	}()

	if cp.Process == nil {
		return
	}

	state, err := cp.Process.Wait()
	if err != nil {
		logger.Error("Process wait failed", zap.String("id", cp.ID), zap.Error(err))
		select {
		case cp.errorChan <- err:
		case <-cp.ctx.Done():
		}
		return
	}

	exitCode := 0
	if state != nil {
		exitCode = state.ExitCode()
	}

	logger.Info("Process exited",
		zap.String("id", cp.ID),
		zap.Int("exit_code", exitCode))

	cp.mutex.Lock()
	cp.running = false
	cp.mutex.Unlock()

	select {
	case cp.exitChan <- exitCode:
	case <-cp.ctx.Done():
	}
}

// WaitForExit 等待进程退出
func (cp *CreackPTY) WaitForExit() int {
	select {
	case exitCode := <-cp.exitChan:
		return exitCode
	case <-cp.ctx.Done():
		return -1
	}
}

// Copy 复制数据流
func (cp *CreackPTY) Copy(dst io.Writer, src io.Reader) error {
	if cp.PTYFile == nil {
		return fmt.Errorf("PTY file is not available")
	}

	_, err := io.Copy(dst, src)
	return err
}

// CopyToStdout 复制到标准输出
func (cp *CreackPTY) CopyToStdout() error {
	if cp.PTYFile == nil {
		return fmt.Errorf("PTY file is not available")
	}

	_, err := io.Copy(os.Stdout, cp.PTYFile)
	return err
}

// CopyFromStdin 从标准输入复制
func (cp *CreackPTY) CopyFromStdin() error {
	if cp.PTYFile == nil {
		return fmt.Errorf("PTY file is not available")
	}

	_, err := io.Copy(cp.PTYFile, os.Stdin)
	return err
}
