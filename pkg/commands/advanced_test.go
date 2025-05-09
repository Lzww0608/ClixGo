package commands

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// 检查命令是否可用
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func TestAWKCommand(t *testing.T) {
	if !commandExists("awk") {
		t.Skip("awk 命令不可用，跳过测试")
	}

	tests := []struct {
		name     string
		input    string
		pattern  string
		expected string
		wantErr  bool
	}{
		{
			name:     "基本过滤",
			input:    "hello\nworld\n123",
			pattern:  "/^[a-z]/",
			expected: "hello\nworld\n",
			wantErr:  false,
		},
		{
			name:     "打印特定列",
			input:    "name:john age:25\nname:alice age:30",
			pattern:  "{print $2}",
			expected: "age:25\nage:30\n",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AWKCommand(tt.input, tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("AWKCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(result, tt.expected) {
				t.Errorf("AWKCommand() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGrepCommand(t *testing.T) {
	if !commandExists("grep") {
		t.Skip("grep 命令不可用，跳过测试")
	}

	tests := []struct {
		name     string
		input    string
		pattern  string
		expected string
		wantErr  bool
	}{
		{
			name:     "基本匹配",
			input:    "hello\nworld\n123",
			pattern:  "world",
			expected: "world\n",
			wantErr:  false,
		},
		{
			name:     "正则匹配",
			input:    "hello\nworld\n123",
			pattern:  "^[0-9]",
			expected: "123\n",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GrepCommand(tt.input, tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("GrepCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(result, tt.expected) {
				t.Errorf("GrepCommand() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSedCommand(t *testing.T) {
	if !commandExists("sed") {
		t.Skip("sed 命令不可用，跳过测试")
	}

	tests := []struct {
		name     string
		input    string
		pattern  string
		expected string
		wantErr  bool
	}{
		{
			name:     "基本替换",
			input:    "hello world",
			pattern:  "s/world/golang/",
			expected: "hello golang",
			wantErr:  false,
		},
		{
			name:     "删除特定行",
			input:    "line1\nline2\nline3",
			pattern:  "2d",
			expected: "line1\nline3",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SedCommand(tt.input, tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("SedCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(result, tt.expected) {
				t.Errorf("SedCommand() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPipeCommands(t *testing.T) {
	// 检查操作系统，在Windows上使用不同的命令
	isWindows := runtime.GOOS == "windows"

	var echoCmd string
	if isWindows {
		echoCmd = "cmd"

		// 如果是Windows且这些命令不可用，则跳过测试
		if !commandExists("cmd") || !commandExists("findstr") {
			t.Skip("Windows 所需命令不可用，跳过测试")
		}
	} else {
		echoCmd = "echo"

		// 如果是非Windows且这些命令不可用，则跳过测试
		if !commandExists("echo") || !commandExists("grep") {
			t.Skip("Unix 所需命令不可用，跳过测试")
		}
	}

	tests := []struct {
		name     string
		commands []string
		expected string
		wantErr  bool
	}{
		{
			name:     "空命令列表",
			commands: []string{},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "包含空命令",
			commands: []string{echoCmd + " test", ""},
			expected: "",
			wantErr:  true,
		},
	}

	// 根据操作系统添加不同的测试用例
	if isWindows {
		tests = append(tests, struct {
			name     string
			commands []string
			expected string
			wantErr  bool
		}{
			name:     "基本命令",
			commands: []string{"cmd /c echo hello"},
			expected: "hello",
			wantErr:  false,
		})
	} else {
		tests = append(tests, []struct {
			name     string
			commands []string
			expected string
			wantErr  bool
		}{
			{
				name:     "基本命令",
				commands: []string{"echo hello"},
				expected: "hello",
				wantErr:  false,
			},
			{
				name:     "简单管道",
				commands: []string{"echo -e 'hello\nworld'", "grep hello"},
				expected: "hello",
				wantErr:  false,
			},
		}...)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PipeCommands(tt.commands)
			if (err != nil) != tt.wantErr {
				t.Errorf("PipeCommands() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != "" && !strings.Contains(result, tt.expected) {
				t.Errorf("PipeCommands() = %v, want %v", result, tt.expected)
			}
		})
	}
}
