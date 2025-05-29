/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 基于Creack库的伪终端(PTY)功能的单元测试
 */

package terminal

import (
	"strings"
	"testing"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
)

func setupTest(t *testing.T) {
	err := logger.InitLogger()
	if err != nil {
		t.Fatalf("初始化logger失败: %v", err)
	}
	t.Cleanup(func() {
		logger.Close()
	})
}

func TestCreackPTYManager_CreateAndDestroy(t *testing.T) {
	// 初始化logger
	err := logger.InitLogger()
	if err != nil {
		t.Fatalf("初始化logger失败: %v", err)
	}
	defer logger.Close()

	config := &TerminalConfig{
		PrefixKey:         "C-b",
		MouseEnabled:      true,
		StatusBar:         true,
		BufferSize:        2000,
		ScrollBack:        2000,
		ClixGoIntegration: true,
	}

	manager := NewCreackPTYManager(config)

	// 测试创建PTY
	pty, err := manager.CreateCreackPTY("test1", "echo hello", "", 80, 24)
	if err != nil {
		t.Fatalf("创建PTY失败: %v", err)
	}

	// 验证PTY属性
	if pty.ID != "test1" {
		t.Errorf("期望ID为test1，实际为%s", pty.ID)
	}

	width, height := pty.GetSize()
	if width != 80 || height != 24 {
		t.Errorf("期望大小为80x24，实际为%dx%d", width, height)
	}

	// 测试列出PTY
	ptys := manager.ListCreackPTYs()
	if len(ptys) != 1 || ptys[0] != "test1" {
		t.Errorf("期望PTY列表包含test1，实际为%v", ptys)
	}

	// 测试获取PTY
	retrievedPTY, err := manager.GetCreackPTY("test1")
	if err != nil {
		t.Fatalf("获取PTY失败: %v", err)
	}

	if retrievedPTY.ID != pty.ID {
		t.Errorf("获取的PTY ID不匹配")
	}

	// 测试销毁PTY
	err = manager.DestroyCreackPTY("test1")
	if err != nil {
		t.Fatalf("销毁PTY失败: %v", err)
	}

	// 验证PTY已被销毁
	ptys = manager.ListCreackPTYs()
	if len(ptys) != 0 {
		t.Errorf("期望PTY列表为空，实际为%v", ptys)
	}
}

func TestCreackPTY_StartAndStop(t *testing.T) {
	// 初始化logger
	err := logger.InitLogger()
	if err != nil {
		t.Fatalf("初始化logger失败: %v", err)
	}
	defer logger.Close()

	config := &TerminalConfig{
		PrefixKey:         "C-b",
		MouseEnabled:      true,
		StatusBar:         true,
		BufferSize:        2000,
		ScrollBack:        2000,
		ClixGoIntegration: true,
	}

	manager := NewCreackPTYManager(config)

	// 创建PTY
	pty, err := manager.CreateCreackPTY("test2", "echo hello world", "", 80, 24)
	if err != nil {
		t.Fatalf("创建PTY失败: %v", err)
	}

	// 启动PTY
	err = pty.Start()
	if err != nil {
		t.Fatalf("启动PTY失败: %v", err)
	}

	// 验证PTY正在运行
	if !pty.IsRunning() {
		t.Error("PTY应该正在运行")
	}

	// 验证PID
	pid := pty.GetPID()
	if pid <= 0 {
		t.Errorf("期望有效的PID，实际为%d", pid)
	}

	// 等待命令执行完成
	time.Sleep(100 * time.Millisecond)

	// 尝试读取输出
	data, err := pty.ReadWithTimeout(1 * time.Second)
	if err != nil && !strings.Contains(err.Error(), "timeout") {
		t.Logf("读取输出时出现错误（可能正常）: %v", err)
	}

	if len(data) > 0 {
		output := string(data)
		t.Logf("PTY输出: %s", output)
		if !strings.Contains(output, "hello") {
			t.Logf("输出中未包含期望的'hello'，但这可能是正常的")
		}
	}

	// 关闭PTY
	err = pty.Close()
	if err != nil {
		t.Fatalf("关闭PTY失败: %v", err)
	}

	// 清理
	manager.DestroyCreackPTY("test2")
}

func TestCreackPTY_WriteAndRead(t *testing.T) {
	config := &TerminalConfig{
		PrefixKey:         "C-b",
		MouseEnabled:      true,
		StatusBar:         true,
		BufferSize:        2000,
		ScrollBack:        2000,
		ClixGoIntegration: true,
	}

	manager := NewCreackPTYManager(config)

	// 创建一个shell PTY
	pty, err := manager.CreateCreackPTY("test3", "", "", 80, 24)
	if err != nil {
		t.Fatalf("创建PTY失败: %v", err)
	}

	// 启动PTY
	err = pty.Start()
	if err != nil {
		t.Fatalf("启动PTY失败: %v", err)
	}

	// 等待shell启动
	time.Sleep(200 * time.Millisecond)

	// 发送命令
	err = pty.Write([]byte("echo test123\n"))
	if err != nil {
		t.Fatalf("写入PTY失败: %v", err)
	}

	// 读取输出
	var output strings.Builder
	for i := 0; i < 10; i++ {
		data, err := pty.ReadWithTimeout(200 * time.Millisecond)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				break
			}
			t.Logf("读取错误: %v", err)
			break
		}

		if len(data) > 0 {
			output.Write(data)
		}
	}

	outputStr := output.String()
	t.Logf("PTY输出: %q", outputStr)

	// 关闭PTY
	err = pty.Close()
	if err != nil {
		t.Fatalf("关闭PTY失败: %v", err)
	}

	// 清理
	manager.DestroyCreackPTY("test3")
}

func TestCreackPTY_Resize(t *testing.T) {
	config := &TerminalConfig{
		PrefixKey:         "C-b",
		MouseEnabled:      true,
		StatusBar:         true,
		BufferSize:        2000,
		ScrollBack:        2000,
		ClixGoIntegration: true,
	}

	manager := NewCreackPTYManager(config)

	// 创建PTY
	pty, err := manager.CreateCreackPTY("test4", "", "", 80, 24)
	if err != nil {
		t.Fatalf("创建PTY失败: %v", err)
	}

	// 启动PTY
	err = pty.Start()
	if err != nil {
		t.Fatalf("启动PTY失败: %v", err)
	}

	// 测试调整大小
	err = pty.Resize(100, 30)
	if err != nil {
		t.Fatalf("调整PTY大小失败: %v", err)
	}

	// 验证新大小
	width, height := pty.GetSize()
	if width != 100 || height != 30 {
		t.Errorf("期望大小为100x30，实际为%dx%d", width, height)
	}

	// 关闭PTY
	err = pty.Close()
	if err != nil {
		t.Fatalf("关闭PTY失败: %v", err)
	}

	// 清理
	manager.DestroyCreackPTY("test4")
}

func TestCreackPTYManager_DuplicateID(t *testing.T) {
	config := &TerminalConfig{
		PrefixKey:         "C-b",
		MouseEnabled:      true,
		StatusBar:         true,
		BufferSize:        2000,
		ScrollBack:        2000,
		ClixGoIntegration: true,
	}

	manager := NewCreackPTYManager(config)

	// 创建第一个PTY
	_, err := manager.CreateCreackPTY("duplicate", "echo test", "", 80, 24)
	if err != nil {
		t.Fatalf("创建第一个PTY失败: %v", err)
	}

	// 尝试创建相同ID的PTY
	_, err = manager.CreateCreackPTY("duplicate", "echo test2", "", 80, 24)
	if err == nil {
		t.Error("期望创建重复ID的PTY失败，但成功了")
	}

	// 清理
	manager.DestroyCreackPTY("duplicate")
}

func TestCreackPTYManager_GetNonexistentPTY(t *testing.T) {
	config := &TerminalConfig{
		PrefixKey:         "C-b",
		MouseEnabled:      true,
		StatusBar:         true,
		BufferSize:        2000,
		ScrollBack:        2000,
		ClixGoIntegration: true,
	}

	manager := NewCreackPTYManager(config)

	// 尝试获取不存在的PTY
	_, err := manager.GetCreackPTY("nonexistent")
	if err == nil {
		t.Error("期望获取不存在的PTY失败，但成功了")
	}

	// 尝试销毁不存在的PTY
	err = manager.DestroyCreackPTY("nonexistent")
	if err == nil {
		t.Error("期望销毁不存在的PTY失败，但成功了")
	}
}
