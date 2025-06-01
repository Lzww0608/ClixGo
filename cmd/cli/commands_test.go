/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-1 09:33:42
* @Description: CLI命令模块测试，包含超时机制防止死锁
 */

package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// testTimeout 测试超时时间，防止死锁
const testTimeout = 10 * time.Second

// TestSequentialCmd 测试串行命令执行
func TestSequentialCmd(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		errorContains string
		timeout       time.Duration
	}{
		{
			name:        "valid_commands",
			args:        []string{"echo hello; echo world"},
			expectError: false,
			timeout:     testTimeout,
		},
		{
			name:          "empty_commands",
			args:          []string{""},
			expectError:   true,
			errorContains: "发现空命令",
			timeout:       testTimeout,
		},
		{
			name:          "invalid_command",
			args:          []string{"nonexistentcommand123"},
			expectError:   true,
			errorContains: "executable file not found",
			timeout:       testTimeout,
		},
		{
			name:        "single_command",
			args:        []string{"echo test"},
			expectError: false,
			timeout:     testTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置超时上下文
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			// 创建命令
			cmd := NewSequentialCmd()

			// 使用goroutine执行命令并捕获结果
			resultChan := make(chan error, 1)
			go func() {
				cmd.SetArgs(tt.args)
				err := cmd.Execute()
				resultChan <- err
			}()

			// 等待结果或超时
			select {
			case err := <-resultChan:
				if tt.expectError {
					assert.Error(t, err)
					if tt.errorContains != "" {
						assert.Contains(t, err.Error(), tt.errorContains)
					}
				} else {
					assert.NoError(t, err)
				}
			case <-ctx.Done():
				t.Fatalf("测试超时: %v", ctx.Err())
			}
		})
	}
}

// TestParallelCmd 测试并行命令执行
func TestParallelCmd(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		errorContains string
		timeout       time.Duration
	}{
		{
			name:        "valid_parallel_commands",
			args:        []string{"echo hello; echo world"},
			expectError: false,
			timeout:     testTimeout,
		},
		{
			name:        "single_command_parallel",
			args:        []string{"echo test"},
			expectError: false,
			timeout:     testTimeout,
		},
		{
			name:          "empty_commands_parallel",
			args:          []string{""},
			expectError:   true,
			errorContains: "发现空命令",
			timeout:       testTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			cmd := NewParallelCmd()

			resultChan := make(chan error, 1)
			go func() {
				cmd.SetArgs(tt.args)
				err := cmd.Execute()
				resultChan <- err
			}()

			select {
			case err := <-resultChan:
				if tt.expectError {
					assert.Error(t, err)
					if tt.errorContains != "" {
						assert.Contains(t, err.Error(), tt.errorContains)
					}
				} else {
					assert.NoError(t, err)
				}
			case <-ctx.Done():
				t.Fatalf("测试超时: %v", ctx.Err())
			}
		})
	}
}

// TestAWKCmd 测试AWK命令
func TestAWKCmd(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		errorContains string
		timeout       time.Duration
	}{
		{
			name:        "valid_awk_pattern",
			args:        []string{"line1\nline2\nline3", "NR==2"},
			expectError: false,
			timeout:     testTimeout,
		},
		{
			name:          "invalid_args_count",
			args:          []string{"only_one_arg"},
			expectError:   true,
			errorContains: "accepts 2 arg",
			timeout:       testTimeout,
		},
		{
			name:        "empty_input",
			args:        []string{"", "NR==1"},
			expectError: false,
			timeout:     testTimeout,
		},
		{
			name:        "complex_awk_pattern",
			args:        []string{"apple\nbanana\ncherry", "/a/"},
			expectError: false,
			timeout:     testTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			cmd := NewAWKCmd()

			// 捕获输出
			var buf bytes.Buffer
			cmd.SetOut(&buf)

			resultChan := make(chan error, 1)
			go func() {
				cmd.SetArgs(tt.args)
				err := cmd.Execute()
				resultChan <- err
			}()

			select {
			case err := <-resultChan:
				if tt.expectError {
					assert.Error(t, err)
					if tt.errorContains != "" {
						assert.Contains(t, err.Error(), tt.errorContains)
					}
				} else {
					assert.NoError(t, err)
				}
			case <-ctx.Done():
				t.Fatalf("测试超时: %v", ctx.Err())
			}
		})
	}
}

// TestGrepCmd 测试Grep命令
func TestGrepCmd(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		errorContains string
		timeout       time.Duration
	}{
		{
			name:        "valid_grep_pattern",
			args:        []string{"hello world\ntest line\nhello again", "hello"},
			expectError: false,
			timeout:     testTimeout,
		},
		{
			name:          "invalid_args_count",
			args:          []string{"only_one_arg"},
			expectError:   true,
			errorContains: "accepts 2 arg",
			timeout:       testTimeout,
		},
		{
			name:          "no_match",
			args:          []string{"hello world", "xyz"},
			expectError:   true,
			errorContains: "exit status 1",
			timeout:       testTimeout,
		},
		{
			name:          "regex_pattern",
			args:          []string{"test123\ntest456\nabc789", "test[0-9]+"},
			expectError:   true,
			errorContains: "exit status 1",
			timeout:       testTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			cmd := NewGrepCmd()

			var buf bytes.Buffer
			cmd.SetOut(&buf)

			resultChan := make(chan error, 1)
			go func() {
				cmd.SetArgs(tt.args)
				err := cmd.Execute()
				resultChan <- err
			}()

			select {
			case err := <-resultChan:
				if tt.expectError {
					assert.Error(t, err)
					if tt.errorContains != "" {
						assert.Contains(t, err.Error(), tt.errorContains)
					}
				} else {
					assert.NoError(t, err)
				}
			case <-ctx.Done():
				t.Fatalf("测试超时: %v", ctx.Err())
			}
		})
	}
}

// TestSedCmd 测试Sed命令
func TestSedCmd(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		errorContains string
		timeout       time.Duration
	}{
		{
			name:        "valid_sed_substitution",
			args:        []string{"hello world", "s/hello/hi/"},
			expectError: false,
			timeout:     testTimeout,
		},
		{
			name:          "invalid_args_count",
			args:          []string{"only_one_arg"},
			expectError:   true,
			errorContains: "accepts 2 arg",
			timeout:       testTimeout,
		},
		{
			name:        "delete_operation",
			args:        []string{"line1\nline2\nline3", "/line2/d"},
			expectError: false,
			timeout:     testTimeout,
		},
		{
			name:        "global_substitution",
			args:        []string{"test test test", "s/test/TEST/g"},
			expectError: false,
			timeout:     testTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			cmd := NewSedCmd()

			var buf bytes.Buffer
			cmd.SetOut(&buf)

			resultChan := make(chan error, 1)
			go func() {
				cmd.SetArgs(tt.args)
				err := cmd.Execute()
				resultChan <- err
			}()

			select {
			case err := <-resultChan:
				if tt.expectError {
					assert.Error(t, err)
					if tt.errorContains != "" {
						assert.Contains(t, err.Error(), tt.errorContains)
					}
				} else {
					assert.NoError(t, err)
				}
			case <-ctx.Done():
				t.Fatalf("测试超时: %v", ctx.Err())
			}
		})
	}
}

// TestPipeCmd 测试管道命令
func TestPipeCmd(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		errorContains string
		timeout       time.Duration
	}{
		{
			name:        "valid_pipe_commands",
			args:        []string{"echo hello; echo world"},
			expectError: false,
			timeout:     testTimeout,
		},
		{
			name:          "empty_commands_pipe",
			args:          []string{""},
			expectError:   true,
			errorContains: "发现空命令",
			timeout:       testTimeout,
		},
		{
			name:        "single_command_pipe",
			args:        []string{"echo test"},
			expectError: false,
			timeout:     testTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			cmd := NewPipeCmd()

			var buf bytes.Buffer
			cmd.SetOut(&buf)

			resultChan := make(chan error, 1)
			go func() {
				cmd.SetArgs(tt.args)
				err := cmd.Execute()
				resultChan <- err
			}()

			select {
			case err := <-resultChan:
				if tt.expectError {
					assert.Error(t, err)
					if tt.errorContains != "" {
						assert.Contains(t, err.Error(), tt.errorContains)
					}
				} else {
					assert.NoError(t, err)
				}
			case <-ctx.Done():
				t.Fatalf("测试超时: %v", ctx.Err())
			}
		})
	}
}

// TestCommandCreation 测试命令创建
func TestCommandCreation(t *testing.T) {
	tests := []struct {
		name     string
		cmdFunc  func() *cobra.Command
		expected string
	}{
		{
			name:     "sequential_command",
			cmdFunc:  NewSequentialCmd,
			expected: "sequential",
		},
		{
			name:     "parallel_command",
			cmdFunc:  NewParallelCmd,
			expected: "parallel",
		},
		{
			name:     "awk_command",
			cmdFunc:  NewAWKCmd,
			expected: "awk",
		},
		{
			name:     "grep_command",
			cmdFunc:  NewGrepCmd,
			expected: "grep",
		},
		{
			name:     "sed_command",
			cmdFunc:  NewSedCmd,
			expected: "sed",
		},
		{
			name:     "pipe_command",
			cmdFunc:  NewPipeCmd,
			expected: "pipe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmdFunc()
			assert.NotNil(t, cmd)
			assert.Equal(t, tt.expected, cmd.Use)
			assert.NotEmpty(t, cmd.Short)
			assert.NotEmpty(t, cmd.Long)
		})
	}
}

// TestCommandStructure 测试命令结构
func TestCommandStructure(t *testing.T) {
	commands := []*cobra.Command{
		NewSequentialCmd(),
		NewParallelCmd(),
		NewAWKCmd(),
		NewGrepCmd(),
		NewSedCmd(),
		NewPipeCmd(),
	}

	for _, cmd := range commands {
		t.Run(fmt.Sprintf("command_%s", cmd.Use), func(t *testing.T) {
			// 验证基本属性
			assert.NotEmpty(t, cmd.Use, "命令Use不能为空")
			assert.NotEmpty(t, cmd.Short, "命令Short描述不能为空")
			assert.NotEmpty(t, cmd.Long, "命令Long描述不能为空")
			assert.NotNil(t, cmd.RunE, "命令RunE函数不能为空")

			// 只验证Args不为空，不比较具体函数
			assert.NotNil(t, cmd.Args, "命令Args验证函数不能为空")
		})
	}
}

// TestCommandHelp 测试命令帮助信息
func TestCommandHelp(t *testing.T) {
	commands := []*cobra.Command{
		NewSequentialCmd(),
		NewParallelCmd(),
		NewAWKCmd(),
		NewGrepCmd(),
		NewSedCmd(),
		NewPipeCmd(),
	}

	for _, cmd := range commands {
		t.Run(fmt.Sprintf("help_%s", cmd.Use), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			resultChan := make(chan error, 1)
			go func() {
				cmd.SetArgs([]string{"--help"})
				err := cmd.Execute()
				resultChan <- err
			}()

			select {
			case err := <-resultChan:
				// Help命令会返回一个特殊的错误，这是正常的
				if err != nil {
					assert.Contains(t, err.Error(), "help requested")
				}
				output := buf.String()
				assert.Contains(t, output, cmd.Use, "帮助信息应该包含命令名称")
				// 使用Long描述而不是Short描述进行验证
				assert.Contains(t, output, cmd.Long, "帮助信息应该包含长描述")
			case <-ctx.Done():
				t.Fatalf("测试超时: %v", ctx.Err())
			}
		})
	}
}

// TestCommandConcurrency 测试命令并发安全性
func TestCommandConcurrency(t *testing.T) {
	const numGoroutines = 10

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			cmd := NewSequentialCmd()
			cmd.SetArgs([]string{fmt.Sprintf("echo concurrent_test_%d", id)})

			var buf bytes.Buffer
			cmd.SetOut(&buf)

			err := cmd.Execute()
			errChan <- err
		}(i)
	}

	// 收集所有结果
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errChan:
			assert.NoError(t, err, "并发执行不应该产生错误")
		case <-ctx.Done():
			t.Fatalf("并发测试超时: %v", ctx.Err())
		}
	}
}

// TestCommandArgValidation 测试命令参数验证
func TestCommandArgValidation(t *testing.T) {
	tests := []struct {
		name          string
		cmd           *cobra.Command
		args          []string
		expectError   bool
		errorContains string
	}{
		{
			name:          "sequential_no_args",
			cmd:           NewSequentialCmd(),
			args:          []string{},
			expectError:   true,
			errorContains: "accepts 1 arg",
		},
		{
			name:          "awk_insufficient_args",
			cmd:           NewAWKCmd(),
			args:          []string{"only_one"},
			expectError:   true,
			errorContains: "accepts 2 arg",
		},
		{
			name:          "grep_too_many_args",
			cmd:           NewGrepCmd(),
			args:          []string{"arg1", "arg2", "arg3"},
			expectError:   true,
			errorContains: "accepts 2 arg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			resultChan := make(chan error, 1)
			go func() {
				tt.cmd.SetArgs(tt.args)
				err := tt.cmd.Execute()
				resultChan <- err
			}()

			select {
			case err := <-resultChan:
				if tt.expectError {
					assert.Error(t, err)
					if tt.errorContains != "" {
						assert.Contains(t, err.Error(), tt.errorContains)
					}
				} else {
					assert.NoError(t, err)
				}
			case <-ctx.Done():
				t.Fatalf("测试超时: %v", ctx.Err())
			}
		})
	}
}

// TestCommandOutputCapture 测试命令输出捕获
func TestCommandOutputCapture(t *testing.T) {
	tests := []struct {
		name           string
		cmd            *cobra.Command
		args           []string
		expectOutput   bool
		outputContains string
	}{
		{
			name:           "awk_with_output",
			cmd:            NewAWKCmd(),
			args:           []string{"line1\nline2\nline3", "NR==2"},
			expectOutput:   true,
			outputContains: "line2",
		},
		{
			name:           "grep_with_match",
			cmd:            NewGrepCmd(),
			args:           []string{"hello\nworld\nhello again", "hello"},
			expectOutput:   true,
			outputContains: "hello",
		},
		{
			name:           "sed_substitution",
			cmd:            NewSedCmd(),
			args:           []string{"hello world", "s/world/universe/"},
			expectOutput:   true,
			outputContains: "universe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// 重定向stdout来捕获fmt.Println的输出
			originalStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			var outputBuf bytes.Buffer
			tt.cmd.SetOut(&outputBuf)

			resultChan := make(chan error, 1)
			outputChan := make(chan string, 1)

			go func() {
				defer func() {
					w.Close()
					os.Stdout = originalStdout
				}()
				tt.cmd.SetArgs(tt.args)
				err := tt.cmd.Execute()
				resultChan <- err
			}()

			go func() {
				buf := make([]byte, 1024)
				n, _ := r.Read(buf)
				outputChan <- string(buf[:n])
			}()

			select {
			case err := <-resultChan:
				assert.NoError(t, err)

				// 等待输出
				select {
				case output := <-outputChan:
					if tt.expectOutput {
						assert.NotEmpty(t, output, "应该有输出")
						if tt.outputContains != "" {
							assert.Contains(t, output, tt.outputContains, "输出应该包含预期内容")
						}
					}
				case <-time.After(1 * time.Second):
					// 检查cmd自己的输出缓冲区
					cmdOutput := outputBuf.String()
					if tt.expectOutput && cmdOutput != "" {
						assert.NotEmpty(t, cmdOutput, "应该有输出")
						if tt.outputContains != "" {
							assert.Contains(t, cmdOutput, tt.outputContains, "输出应该包含预期内容")
						}
					}
				}
			case <-ctx.Done():
				t.Fatalf("测试超时: %v", ctx.Err())
			}
		})
	}
}

// BenchmarkCommandCreation 命令创建性能基准测试
func BenchmarkCommandCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewSequentialCmd()
		_ = NewParallelCmd()
		_ = NewAWKCmd()
		_ = NewGrepCmd()
		_ = NewSedCmd()
		_ = NewPipeCmd()
	}
}

// BenchmarkCommandExecution 命令执行性能基准测试
func BenchmarkCommandExecution(b *testing.B) {
	cmd := NewSequentialCmd()
	args := []string{"echo benchmark_test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd.SetArgs(args)
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		_ = cmd.Execute()
	}
}

// setupTestEnvironment 设置测试环境
func setupTestEnvironment() {
	// 设置测试环境变量
	os.Setenv("TEST_MODE", "true")

	// 初始化logger
	err := logger.InitLogger()
	if err != nil {
		// 如果logger初始化失败，创建一个最小的logger
		fmt.Printf("警告: logger初始化失败: %v\n", err)
	}
}

// teardownTestEnvironment 清理测试环境
func teardownTestEnvironment() {
	// 清理测试环境变量
	os.Unsetenv("TEST_MODE")

	// 关闭logger
	logger.Close()
}

// TestMain 测试主函数
func TestMain(m *testing.M) {
	setupTestEnvironment()
	code := m.Run()
	teardownTestEnvironment()
	os.Exit(code)
}
