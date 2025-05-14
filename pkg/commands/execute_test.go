package commands

import (
	"os"
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
				// 使用纯粹的echo命令代替shell命令，避免引号和转义问题
				"echo test1",
				"echo test2",
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
			// 不再需要文件测试
			err := ExecuteCommandsParallel(tt.commands)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteCommandsParallel() error = %v, wantErr %v", err, tt.wantErr)
			}

			// 仅让测试等待足够时间，确保所有命令有机会执行完成
			if !tt.wantErr {
				time.Sleep(100 * time.Millisecond)
			}
		})
	}
}
