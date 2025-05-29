/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 伪终端(PTY)功能的核心实现
 */

package terminal

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"go.uber.org/zap"
)

// PTY 伪终端结构
type PTY struct {
	Master  *os.File
	Slave   *os.File
	Process *os.Process
	Command *exec.Cmd
	Width   int
	Height  int
	input   chan []byte
	output  chan []byte
	running bool
	mutex   sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// PTYManager PTY管理器
type PTYManager struct {
	ptys   map[string]*PTY
	mutex  sync.RWMutex
	config *TerminalConfig
}

// NewPTYManager 创建PTY管理器
func NewPTYManager(config *TerminalConfig) *PTYManager {
	return &PTYManager{
		ptys:   make(map[string]*PTY),
		config: config,
	}
}

// CreatePTY 创建新的PTY
func (pm *PTYManager) CreatePTY(id, command, workingDir string, width, height int) (*PTY, error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if _, exists := pm.ptys[id]; exists {
		return nil, fmt.Errorf("PTY with id %s already exists", id)
	}

	pty, err := pm.createPTY(command, workingDir, width, height)
	if err != nil {
		return nil, err
	}

	pm.ptys[id] = pty
	logger.Info("PTY created", zap.String("id", id), zap.String("command", command))

	return pty, nil
}

// GetPTY 获取PTY
func (pm *PTYManager) GetPTY(id string) (*PTY, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	pty, exists := pm.ptys[id]
	if !exists {
		return nil, fmt.Errorf("PTY not found: %s", id)
	}

	return pty, nil
}

// DestroyPTY 销毁PTY
func (pm *PTYManager) DestroyPTY(id string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pty, exists := pm.ptys[id]
	if !exists {
		return fmt.Errorf("PTY not found: %s", id)
	}

	if err := pty.Close(); err != nil {
		logger.Error("Failed to close PTY", zap.Error(err))
	}

	delete(pm.ptys, id)
	logger.Info("PTY destroyed", zap.String("id", id))

	return nil
}

// createPTY 内部创建PTY方法
func (pm *PTYManager) createPTY(command, workingDir string, width, height int) (*PTY, error) {
	// 创建主从伪终端文件描述符
	master, slave, err := ptyOpen()
	if err != nil {
		return nil, fmt.Errorf("failed to open pty: %v", err)
	}

	// 设置终端大小
	if err := setWinSize(slave.Fd(), width, height); err != nil {
		master.Close()
		slave.Close()
		return nil, fmt.Errorf("failed to set window size: %v", err)
	}

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
		fmt.Sprintf("TERM=%s", getTerminalType()),
		fmt.Sprintf("COLUMNS=%d", width),
		fmt.Sprintf("LINES=%d", height),
	)

	// 连接标准输入输出到slave
	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setctty: true,
		Setsid:  true,
	}

	ctx, cancel := context.WithCancel(context.Background())

	pty := &PTY{
		Master:  master,
		Slave:   slave,
		Command: cmd,
		Width:   width,
		Height:  height,
		input:   make(chan []byte, 1024),
		output:  make(chan []byte, 1024),
		running: false,
		ctx:     ctx,
		cancel:  cancel,
	}

	return pty, nil
}

// Start 启动PTY进程
func (pty *PTY) Start() error {
	pty.mutex.Lock()
	defer pty.mutex.Unlock()

	if pty.running {
		return fmt.Errorf("PTY is already running")
	}

	// 启动进程
	if err := pty.Command.Start(); err != nil {
		return fmt.Errorf("failed to start command: %v", err)
	}

	pty.Process = pty.Command.Process
	pty.running = true

	// 关闭slave端（父进程不需要）
	pty.Slave.Close()

	// 启动I/O处理goroutines
	go pty.handleInput()
	go pty.handleOutput()
	go pty.waitForExit()

	logger.Info("PTY process started", zap.Int("pid", pty.Process.Pid))
	return nil
}

// Write 向PTY写入数据
func (pty *PTY) Write(data []byte) error {
	pty.mutex.RLock()
	defer pty.mutex.RUnlock()

	if !pty.running {
		return fmt.Errorf("PTY is not running")
	}

	select {
	case pty.input <- data:
		return nil
	case <-pty.ctx.Done():
		return fmt.Errorf("PTY is closed")
	default:
		return fmt.Errorf("input buffer full")
	}
}

// Read 从PTY读取数据
func (pty *PTY) Read() ([]byte, error) {
	select {
	case data := <-pty.output:
		return data, nil
	case <-pty.ctx.Done():
		return nil, fmt.Errorf("PTY is closed")
	}
}

// Resize 调整PTY大小
func (pty *PTY) Resize(width, height int) error {
	pty.mutex.Lock()
	defer pty.mutex.Unlock()

	if !pty.running {
		return fmt.Errorf("PTY is not running")
	}

	if err := setWinSize(pty.Master.Fd(), width, height); err != nil {
		return err
	}

	pty.Width = width
	pty.Height = height

	// 发送SIGWINCH信号给进程组
	if pty.Process != nil {
		syscall.Kill(-pty.Process.Pid, syscall.SIGWINCH)
	}

	return nil
}

// Signal 向PTY进程发送信号
func (pty *PTY) Signal(sig os.Signal) error {
	pty.mutex.RLock()
	defer pty.mutex.RUnlock()

	if !pty.running || pty.Process == nil {
		return fmt.Errorf("PTY process is not running")
	}

	return pty.Process.Signal(sig)
}

// Close 关闭PTY
func (pty *PTY) Close() error {
	pty.mutex.Lock()
	defer pty.mutex.Unlock()

	if !pty.running {
		return nil
	}

	pty.running = false
	pty.cancel()

	// 终止进程
	if pty.Process != nil {
		// 先尝试优雅关闭
		pty.Process.Signal(syscall.SIGTERM)

		// 等待一段时间
		done := make(chan error, 1)
		go func() {
			_, err := pty.Process.Wait()
			done <- err
		}()

		select {
		case <-done:
			// 进程已退出
		case <-time.After(time.Second * 5):
			// 强制杀死进程
			pty.Process.Kill()
			pty.Process.Wait()
		}
	}

	// 关闭文件描述符
	if pty.Master != nil {
		pty.Master.Close()
	}
	if pty.Slave != nil {
		pty.Slave.Close()
	}

	// 关闭通道
	close(pty.input)
	close(pty.output)

	logger.Info("PTY closed")
	return nil
}

// IsRunning 检查PTY是否运行中
func (pty *PTY) IsRunning() bool {
	pty.mutex.RLock()
	defer pty.mutex.RUnlock()
	return pty.running
}

// GetPID 获取进程ID
func (pty *PTY) GetPID() int {
	pty.mutex.RLock()
	defer pty.mutex.RUnlock()
	if pty.Process != nil {
		return pty.Process.Pid
	}
	return -1
}

// handleInput 处理输入数据
func (pty *PTY) handleInput() {
	for {
		select {
		case data := <-pty.input:
			if _, err := pty.Master.Write(data); err != nil {
				logger.Error("Failed to write to PTY master", zap.Error(err))
				return
			}
		case <-pty.ctx.Done():
			return
		}
	}
}

// handleOutput 处理输出数据
func (pty *PTY) handleOutput() {
	buffer := make([]byte, 4096)

	for {
		select {
		case <-pty.ctx.Done():
			return
		default:
			n, err := pty.Master.Read(buffer)
			if err != nil {
				if err != io.EOF {
					logger.Error("Failed to read from PTY master", zap.Error(err))
				}
				return
			}

			if n > 0 {
				data := make([]byte, n)
				copy(data, buffer[:n])

				select {
				case pty.output <- data:
				case <-pty.ctx.Done():
					return
				default:
					// 如果输出缓冲区满了，丢弃最旧的数据
					select {
					case <-pty.output:
					default:
					}
					pty.output <- data
				}
			}
		}
	}
}

// waitForExit 等待进程退出
func (pty *PTY) waitForExit() {
	if pty.Command != nil {
		pty.Command.Wait()
		pty.mutex.Lock()
		pty.running = false
		pty.mutex.Unlock()
		logger.Info("PTY process exited", zap.Int("pid", pty.GetPID()))
	}
}

// 平台相关的PTY实现

// ptyOpen 打开PTY（Unix/Linux实现）
func ptyOpen() (*os.File, *os.File, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}

	// 解锁slave
	if err := unlockPT(master.Fd()); err != nil {
		master.Close()
		return nil, nil, err
	}

	// 获取slave路径
	slaveName, err := ptsName(master.Fd())
	if err != nil {
		master.Close()
		return nil, nil, err
	}

	// 打开slave
	slave, err := os.OpenFile(slaveName, os.O_RDWR, 0)
	if err != nil {
		master.Close()
		return nil, nil, err
	}

	return master, slave, nil
}

// 系统调用包装函数

// unlockPT 解锁PTY
func unlockPT(fd uintptr) error {
	var unlock int32 = 0
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock)))
	if errno != 0 {
		return errno
	}
	return nil
}

// ptsName 获取slave PTY名称
func ptsName(fd uintptr) (string, error) {
	var n int32
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n)))
	if errno != 0 {
		return "", errno
	}
	return fmt.Sprintf("/dev/pts/%d", n), nil
}

// winSize 定义终端窗口大小结构
type winSize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

// setWinSize 设置终端窗口大小
func setWinSize(fd uintptr, width, height int) error {
	ws := &winSize{
		Row:    uint16(height),
		Col:    uint16(width),
		Xpixel: uint16(width * 8),
		Ypixel: uint16(height * 16),
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCSWINSZ, uintptr(unsafe.Pointer(ws)))
	if errno != 0 {
		return errno
	}
	return nil
}

// getTerminalType 获取终端类型
func getTerminalType() string {
	if term := os.Getenv("TERM"); term != "" {
		return term
	}
	return "xterm-256color"
}
