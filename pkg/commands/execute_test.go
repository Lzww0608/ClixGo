package commands

import (
	"os"
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

	// 创建一个临时文件来测试并行执行
	tempFile, err := os.CreateTemp("", "parallel_test")
	if err != nil {
		t.Fatalf("无法创建临时文件: %v", err)
	}
	tempFilePath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempFilePath)

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
				"sh -c \"echo test1 > " + tempFilePath + "\"",
				"sh -c \"echo test2 >> " + tempFilePath + "\"",
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
			// 确保测试开始时文件是空的
			if len(tt.commands) > 0 && strings.Contains(tt.commands[0], tempFilePath) {
				err := os.WriteFile(tempFilePath, []byte{}, 0644)
				if err != nil {
					t.Fatalf("无法清空临时文件: %v", err)
				}
			}

			err := ExecuteCommandsParallel(tt.commands)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteCommandsParallel() error = %v, wantErr %v", err, tt.wantErr)
			}

			// 验证并行命令的执行结果（可选）
			if !tt.wantErr && len(tt.commands) > 0 && strings.Contains(tt.commands[0], tempFilePath) {
				// 给并行命令一些时间完成
				time.Sleep(200 * time.Millisecond)

				// 检查文件内容
				content, err := os.ReadFile(tempFilePath)
				if err != nil {
					t.Errorf("读取临时文件失败: %v", err)
				}
				if len(content) == 0 {
					t.Errorf("并行命令似乎未执行完成，文件内容为空")
				}
			}
		})
	}
}
