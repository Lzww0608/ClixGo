/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 简化版伪终端(PTY)功能的实现
 */

package terminal

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"go.uber.org/zap"
)

// SimplePTY 简化的PTY实现
type SimplePTY struct {
	ID         string
	Command    *exec.Cmd
	Process    *os.Process
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stderr     io.ReadCloser
	Width      int
	Height     int
	running    bool
	mutex      sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	outputChan chan []byte
}

// SimplePTYManager 简化的PTY管理器
type SimplePTYManager struct {
	ptys   map[string]*SimplePTY
	mutex  sync.RWMutex
	config *TerminalConfig
}

// NewSimplePTYManager 创建简化PTY管理器
func NewSimplePTYManager(config *TerminalConfig) *SimplePTYManager {
	return &SimplePTYManager{
		ptys:   make(map[string]*SimplePTY),
		config: config,
	}
}

// CreateSimplePTY 创建简化PTY
func (pm *SimplePTYManager) CreateSimplePTY(id, command, workingDir string, width, height int) (*SimplePTY, error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if _, exists := pm.ptys[id]; exists {
		return nil, fmt.Errorf("PTY with id %s already exists", id)
	}

	pty, err := pm.createSimplePTY(id, command, workingDir, width, height)
	if err != nil {
		return nil, err
	}

	pm.ptys[id] = pty
	logger.Info("Simple PTY created", zap.String("id", id), zap.String("command", command))

	return pty, nil
}

// GetSimplePTY 获取简化PTY
func (pm *SimplePTYManager) GetSimplePTY(id string) (*SimplePTY, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	pty, exists := pm.ptys[id]
	if !exists {
		return nil, fmt.Errorf("Simple PTY not found: %s", id)
	}

	return pty, nil
}

// DestroySimplePTY 销毁简化PTY
func (pm *SimplePTYManager) DestroySimplePTY(id string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pty, exists := pm.ptys[id]
	if !exists {
		return fmt.Errorf("Simple PTY not found: %s", id)
	}

	if err := pty.Close(); err != nil {
		logger.Error("Failed to close Simple PTY", zap.Error(err))
	}

	delete(pm.ptys, id)
	logger.Info("Simple PTY destroyed", zap.String("id", id))

	return nil
}

// createSimplePTY 内部创建简化PTY方法
func (pm *SimplePTYManager) createSimplePTY(id, command, workingDir string, width, height int) (*SimplePTY, error) {
	// 解析命令
	if command == "" {
		command = os.Getenv("SHELL")
		if command == "" {
			command = "/bin/bash"
		}
	}

	// 创建命令
	cmd := exec.Command("/bin/bash", "-c", command)
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// 设置环境变量
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("TERM=%s", getTermType()),
		fmt.Sprintf("COLUMNS=%d", width),
		fmt.Sprintf("LINES=%d", height),
	)

	// 获取管道
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	pty := &SimplePTY{
		ID:         id,
		Command:    cmd,
		stdin:      stdin,
		stdout:     stdout,
		stderr:     stderr,
		Width:      width,
		Height:     height,
		running:    false,
		ctx:        ctx,
		cancel:     cancel,
		outputChan: make(chan []byte, 1024),
	}

	return pty, nil
}

// Start 启动简化PTY进程
func (pty *SimplePTY) Start() error {
	pty.mutex.Lock()
	defer pty.mutex.Unlock()

	if pty.running {
		return fmt.Errorf("Simple PTY is already running")
	}

	// 启动进程
	if err := pty.Command.Start(); err != nil {
		return fmt.Errorf("failed to start command: %v", err)
	}

	pty.Process = pty.Command.Process
	pty.running = true

	// 启动I/O处理goroutines
	go pty.handleOutput()
	go pty.handleError()
	go pty.waitForExit()

	logger.Info("Simple PTY process started", zap.String("id", pty.ID), zap.Int("pid", pty.Process.Pid))
	return nil
}

// Write 向简化PTY写入数据
func (pty *SimplePTY) Write(data []byte) error {
	pty.mutex.RLock()
	defer pty.mutex.RUnlock()

	if !pty.running {
		return fmt.Errorf("Simple PTY is not running")
	}

	if pty.stdin == nil {
		return fmt.Errorf("stdin is not available")
	}

	_, err := pty.stdin.Write(data)
	return err
}

// Read 从简化PTY读取数据
func (pty *SimplePTY) Read() ([]byte, error) {
	select {
	case data := <-pty.outputChan:
		return data, nil
	case <-pty.ctx.Done():
		return nil, fmt.Errorf("Simple PTY is closed")
	case <-time.After(time.Millisecond * 100):
		return nil, fmt.Errorf("read timeout")
	}
}

// Resize 调整简化PTY大小（简化实现）
func (pty *SimplePTY) Resize(width, height int) error {
	pty.mutex.Lock()
	defer pty.mutex.Unlock()

	pty.Width = width
	pty.Height = height

	// 简化实现：仅更新尺寸变量
	// 实际的终端大小调整需要更复杂的实现
	logger.Info("Simple PTY resized",
		zap.String("id", pty.ID),
		zap.Int("width", width),
		zap.Int("height", height))

	return nil
}

// Close 关闭简化PTY
func (pty *SimplePTY) Close() error {
	pty.mutex.Lock()
	defer pty.mutex.Unlock()

	if !pty.running {
		return nil
	}

	pty.running = false
	pty.cancel()

	// 关闭管道
	if pty.stdin != nil {
		pty.stdin.Close()
	}
	if pty.stdout != nil {
		pty.stdout.Close()
	}
	if pty.stderr != nil {
		pty.stderr.Close()
	}

	// 终止进程
	if pty.Process != nil {
		// 先尝试优雅关闭
		pty.Process.Signal(os.Interrupt)

		// 等待一段时间
		done := make(chan error, 1)
		go func() {
			_, err := pty.Process.Wait()
			done <- err
		}()

		select {
		case <-done:
			// 进程已退出
		case <-time.After(time.Second * 3):
			// 强制杀死进程
			pty.Process.Kill()
			pty.Process.Wait()
		}
	}

	// 关闭通道
	close(pty.outputChan)

	logger.Info("Simple PTY closed", zap.String("id", pty.ID))
	return nil
}

// IsRunning 检查简化PTY是否运行中
func (pty *SimplePTY) IsRunning() bool {
	pty.mutex.RLock()
	defer pty.mutex.RUnlock()
	return pty.running
}

// GetPID 获取进程ID
func (pty *SimplePTY) GetPID() int {
	pty.mutex.RLock()
	defer pty.mutex.RUnlock()
	if pty.Process != nil {
		return pty.Process.Pid
	}
	return -1
}

// handleOutput 处理标准输出
func (pty *SimplePTY) handleOutput() {
	buffer := make([]byte, 4096)

	for {
		select {
		case <-pty.ctx.Done():
			return
		default:
			if pty.stdout == nil {
				return
			}

			n, err := pty.stdout.Read(buffer)
			if err != nil {
				if err != io.EOF {
					logger.Error("Failed to read from stdout", zap.Error(err))
				}
				return
			}

			if n > 0 {
				data := make([]byte, n)
				copy(data, buffer[:n])

				select {
				case pty.outputChan <- data:
				case <-pty.ctx.Done():
					return
				default:
					// 如果输出缓冲区满了，丢弃最旧的数据
					select {
					case <-pty.outputChan:
					default:
					}
					pty.outputChan <- data
				}
			}
		}
	}
}

// handleError 处理标准错误
func (pty *SimplePTY) handleError() {
	buffer := make([]byte, 4096)

	for {
		select {
		case <-pty.ctx.Done():
			return
		default:
			if pty.stderr == nil {
				return
			}

			n, err := pty.stderr.Read(buffer)
			if err != nil {
				if err != io.EOF {
					logger.Error("Failed to read from stderr", zap.Error(err))
				}
				return
			}

			if n > 0 {
				data := make([]byte, n)
				copy(data, buffer[:n])

				// 将错误输出也发送到输出通道
				select {
				case pty.outputChan <- data:
				case <-pty.ctx.Done():
					return
				default:
					// 如果输出缓冲区满了，丢弃最旧的数据
					select {
					case <-pty.outputChan:
					default:
					}
					pty.outputChan <- data
				}
			}
		}
	}
}

// waitForExit 等待进程退出
func (pty *SimplePTY) waitForExit() {
	if pty.Command != nil {
		pty.Command.Wait()
		pty.mutex.Lock()
		pty.running = false
		pty.mutex.Unlock()
		logger.Info("Simple PTY process exited", zap.String("id", pty.ID), zap.Int("pid", pty.GetPID()))
	}
}

// getTermType 获取终端类型
func getTermType() string {
	if term := os.Getenv("TERM"); term != "" {
		return term
	}
	return "xterm-256color"
}
