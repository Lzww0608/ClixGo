package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
)

// 设置测试环境
func setupTestEnvironment() {
	// 初始化logger避免空指针错误
	logger.InitLogger()
}

// 创建临时测试目录
func createTempDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "clixgo_test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	return tempDir
}

// TestExecuteCommand 测试单个命令执行
func TestExecuteCommand(t *testing.T) {
	setupTestEnvironment()

	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{
			name:    "空命令",
			command: "",
			wantErr: true,
		},
		{
			name:    "无效命令",
			command: "invalid_command_xyz",
			wantErr: true,
		},
		{
			name:    "有效命令",
			command: "echo test",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExecuteCommand(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestExecuteCommandsSequentially 测试命令串行执行
func TestExecuteCommandsSequentially(t *testing.T) {
	setupTestEnvironment()

	tests := []struct {
		name     string
		commands []string
		wantErr  bool
	}{
		{
			name:     "空命令列表",
			commands: []string{},
			wantErr:  false,
		},
		{
			name:     "单个有效命令",
			commands: []string{"echo test"},
			wantErr:  false,
		},
		{
			name:     "多个有效命令",
			commands: []string{"echo test1", "echo test2"},
			wantErr:  false,
		},
		{
			name:     "包含无效命令",
			commands: []string{"echo test", "invalid_command_xyz"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExecuteCommandsSequentially(tt.commands)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteCommandsSequentially() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestExecuteCommandsParallel 测试命令并行执行
func TestExecuteCommandsParallel(t *testing.T) {
	setupTestEnvironment()

	// 创建临时目录
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// 创建两个测试文件
	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")

	tests := []struct {
		name     string
		commands []string
		wantErr  bool
	}{
		{
			name:     "空命令列表",
			commands: []string{},
			wantErr:  false,
		},
		{
			name: "多个有效命令",
			commands: []string{
				"touch " + file1,
				"touch " + file2,
			},
			wantErr: false,
		},
		{
			name:     "包含无效命令",
			commands: []string{"echo test", "invalid_command_xyz"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 确保测试开始时文件不存在
			if len(tt.commands) > 0 && strings.Contains(tt.commands[0], tempDir) {
				os.Remove(file1)
				os.Remove(file2)
			}

			err := ExecuteCommandsParallel(tt.commands)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteCommandsParallel() error = %v, wantErr %v", err, tt.wantErr)
			}

			// 验证并行命令的执行结果
			if !tt.wantErr && len(tt.commands) > 0 && strings.Contains(tt.commands[0], tempDir) {
				// 给并行命令一些时间完成
				time.Sleep(200 * time.Millisecond)

				// 检查文件是否已创建
				_, err1 := os.Stat(file1)
				_, err2 := os.Stat(file2)
				if os.IsNotExist(err1) || os.IsNotExist(err2) {
					t.Errorf("并行命令未成功执行: file1存在=%v, file2存在=%v",
						!os.IsNotExist(err1), !os.IsNotExist(err2))
				}
			}
		})
	}
}
