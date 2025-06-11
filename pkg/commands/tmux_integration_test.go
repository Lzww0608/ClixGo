/*
* @Author: Lzww0608
* @Date: 2025-6-11 11:16:14
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-11 11:16:18
* @Description: tmux集成测试 - 验证tmux兼容层与现有SessionManager的集成效果
 */

package commands

import (
	"fmt"
	"testing"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/terminal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTmuxSessionIntegration 测试tmux会话集成
func TestTmuxSessionIntegration(t *testing.T) {
	logger := &MockLogger{}
	sessionManager := terminal.NewSessionManager(nil)
	integrationLayer := NewSessionIntegrationLayer(sessionManager, logger)
	enhancedParser := NewEnhancedParser(logger)

	// 注册tmux命令
	err := RegisterTmuxCommands(enhancedParser, integrationLayer, logger)
	require.NoError(t, err)

	// 测试new-session命令
	cmdList, err := enhancedParser.Parse("new-session -s test-session")
	require.NoError(t, err)
	assert.Len(t, cmdList.Commands, 1)
	assert.Equal(t, "new-session", cmdList.Commands[0].Name())

	// 创建测试上下文
	ctx := &Context{
		Session:   nil,
		Window:    nil,
		Pane:      nil,
		Client:    nil,
		Variables: make(map[string]interface{}),
		Logger:    logger,
	}

	// 执行命令
	err = cmdList.Execute(ctx)
	require.NoError(t, err)

	// 验证会话是否创建
	sessions := sessionManager.ListSessions()
	assert.Len(t, sessions, 1)
	assert.Equal(t, "test-session", sessions[0].Name)
}

// TestTmuxCommandParsing 测试tmux命令解析
func TestTmuxCommandParsing(t *testing.T) {
	logger := &MockLogger{}
	parser := NewEnhancedParser(logger)

	tests := []struct {
		name        string
		input       string
		expectedCmd string
		expectError bool
	}{
		{
			name:        "new-session with name",
			input:       "new-session -s mysession",
			expectedCmd: "new-session",
			expectError: false,
		},
		{
			name:        "attach-session",
			input:       "attach-session -t mysession",
			expectedCmd: "attach-session",
			expectError: false,
		},
		{
			name:        "list-sessions",
			input:       "list-sessions",
			expectedCmd: "list-sessions",
			expectError: false,
		},
		{
			name:        "kill-session",
			input:       "kill-session -t mysession",
			expectedCmd: "kill-session",
			expectError: false,
		},
		{
			name:        "new-window",
			input:       "new-window -n mywindow",
			expectedCmd: "new-window",
			expectError: false,
		},
	}

	// 注册tmux命令
	sessionManager := terminal.NewSessionManager(nil)
	integrationLayer := NewSessionIntegrationLayer(sessionManager, logger)
	err := RegisterTmuxCommands(parser, integrationLayer, logger)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdList, err := parser.Parse(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cmdList)
				assert.Len(t, cmdList.Commands, 1)
				assert.Equal(t, tt.expectedCmd, cmdList.Commands[0].Name())
			}
		})
	}
}

// TestTmuxAliasExpansion 测试tmux别名展开
func TestTmuxAliasExpansion(t *testing.T) {
	logger := &MockLogger{}
	parser := NewEnhancedParser(logger)

	// 测试内置别名
	tests := []struct {
		alias    string
		expanded string
	}{
		{"new", "new-session"},
		{"attach", "attach-session"},
		{"ls", "list-sessions"},
		{"neww", "new-window"},
		{"killw", "kill-window"},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			// 验证别名存在
			supportedCommands := parser.GetSupportedTmuxCommands()
			found := false
			for _, cmd := range supportedCommands {
				if cmd == tt.alias+" -> "+tt.expanded {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected alias %s -> %s to be found", tt.alias, tt.expanded)
		})
	}
}

// TestTmuxKeyBindings 测试tmux快捷键绑定
func TestTmuxKeyBindings(t *testing.T) {
	logger := &MockLogger{}
	parser := NewEnhancedParser(logger)

	// 测试基础快捷键
	keyBindings := parser.GetKeyBindings()
	assert.NotEmpty(t, keyBindings)

	// 测试特定按键
	expectedBindings := map[string]string{
		"d": "detach-client",
		"c": "new-window",
		"&": "confirm-before",
		"s": "choose-tree",
		"0": "select-window",
		"1": "select-window",
	}

	for key, expectedCmd := range expectedBindings {
		binding, exists := keyBindings[key]
		assert.True(t, exists, "Expected key binding for '%s' to exist", key)
		if exists {
			assert.Equal(t, expectedCmd, binding.Command, "Expected command for key '%s'", key)
		}
	}
}

// TestSessionDataConversion 测试会话数据转换
func TestSessionDataConversion(t *testing.T) {
	sessionManager := terminal.NewSessionManager(nil)

	// 创建测试会话
	session, err := sessionManager.CreateSession("test-conversion")
	require.NoError(t, err)

	// 测试转换
	tmuxSession := ConvertTerminalSessionToTmux(session)
	assert.NotNil(t, tmuxSession)
	assert.Equal(t, session.ID, tmuxSession.ID)
	assert.Equal(t, session.Name, tmuxSession.Name)
	assert.Equal(t, len(session.Windows), len(tmuxSession.Windows))

	// 测试反向转换
	backSession := ConvertTmuxSessionToTerminal(tmuxSession)
	assert.NotNil(t, backSession)
	assert.Equal(t, tmuxSession.ID, backSession.ID)
	assert.Equal(t, tmuxSession.Name, backSession.Name)
	assert.Equal(t, len(tmuxSession.Windows), len(backSession.Windows))
}

// TestTmuxCommandExecution 测试tmux命令执行
func TestTmuxCommandExecution(t *testing.T) {
	logger := &MockLogger{}
	sessionManager := terminal.NewSessionManager(nil)
	integrationLayer := NewSessionIntegrationLayer(sessionManager, logger)
	adapter := NewTmuxCommandAdapter(integrationLayer, logger)

	ctx := &Context{
		Session:   nil,
		Window:    nil,
		Pane:      nil,
		Client:    nil,
		Variables: make(map[string]interface{}),
		Logger:    logger,
	}

	// 测试new-session命令执行
	args := &Arguments{
		Command: "new-session",
		Flags: map[string]interface{}{
			"session-name": "test-exec-session",
			"detached":     false,
		},
		Positional: []string{},
		Raw:        "new-session -s test-exec-session",
	}

	err := adapter.ExecuteNewSession(ctx, args)
	require.NoError(t, err)

	// 验证会话创建
	sessions := sessionManager.ListSessions()
	assert.Len(t, sessions, 1)
	assert.Equal(t, "test-exec-session", sessions[0].Name)

	// 测试list-sessions命令
	listArgs := &Arguments{
		Command:    "list-sessions",
		Flags:      map[string]interface{}{},
		Positional: []string{},
		Raw:        "list-sessions",
	}

	err = adapter.ExecuteListSessions(ctx, listArgs)
	assert.NoError(t, err)

	// 测试kill-session命令
	killArgs := &Arguments{
		Command: "kill-session",
		Flags: map[string]interface{}{
			"target-session": "test-exec-session",
		},
		Positional: []string{},
		Raw:        "kill-session -t test-exec-session",
	}

	err = adapter.ExecuteKillSession(ctx, killArgs)
	assert.NoError(t, err)

	// 验证会话被删除
	sessions = sessionManager.ListSessions()
	assert.Len(t, sessions, 0)
}

// BenchmarkTmuxCommandExecution 基准测试tmux命令执行性能
func BenchmarkTmuxCommandExecution(b *testing.B) {
	logger := &MockLogger{}
	sessionManager := terminal.NewSessionManager(nil)
	integrationLayer := NewSessionIntegrationLayer(sessionManager, logger)
	enhancedParser := NewEnhancedParser(logger)

	// 注册tmux命令
	err := RegisterTmuxCommands(enhancedParser, integrationLayer, logger)
	if err != nil {
		b.Fatal(err)
	}

	// 测试命令解析性能，而不是实际执行
	testCommands := []string{
		"new-session -s test-session -d",
		"list-sessions",
		"attach-session -t test-session",
		"new-window -t test-session -n test-window",
		"kill-session -t test-session",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 循环测试不同的命令解析
		cmdStr := testCommands[i%len(testCommands)]

		// 测试命令解析性能
		commands, err := enhancedParser.Parse(cmdStr)
		if err != nil {
			b.Errorf("Parse failed for %s: %v", cmdStr, err)
			continue
		}

		if commands == nil || len(commands.Commands) == 0 {
			b.Errorf("No commands parsed for: %s", cmdStr)
			continue
		}

		// 测试命令验证
		for i, cmd := range commands.Commands {
			if cmd == nil {
				b.Errorf("Command is nil for: %s", cmdStr)
				continue
			}

			// 获取对应的参数
			if i >= len(commands.Args) {
				b.Errorf("Missing arguments for command %d in: %s", i, cmdStr)
				continue
			}

			args := commands.Args[i]
			if args == nil {
				b.Errorf("Arguments is nil for command %d in: %s", i, cmdStr)
				continue
			}

			// 验证参数（不执行实际操作）
			if err := cmd.Validate(args); err != nil {
				// 验证失败不算错误，某些命令可能需要特定上下文
				continue
			}
		}
	}
}

// BenchmarkTmuxCommandParsing 基准测试纯命令解析性能
func BenchmarkTmuxCommandParsing(b *testing.B) {
	logger := &MockLogger{}
	enhancedParser := NewEnhancedParser(logger)

	testCommands := []string{
		"new-session -s test-session -d",
		"attach-session -t test-session",
		"list-sessions",
		"new-window -t test-session -n test-window",
		"kill-session -t test-session",
		"split-window -h -t test-session",
		"select-pane -t test-session:0.1",
		"resize-pane -t test-session:0.1 -R 10",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmdStr := testCommands[i%len(testCommands)]

		// 测试命令解析
		_, err := enhancedParser.Parse(cmdStr)
		if err != nil {
			// 解析错误在基准测试中不致命，继续测试
			continue
		}
	}
}

// BenchmarkTmuxDataConversion 基准测试数据转换性能
func BenchmarkTmuxDataConversion(b *testing.B) {
	// 创建测试用的session数据
	testSession := &terminal.Session{
		ID:           "test-session-id",
		Name:         "test-session",
		Windows:      []*terminal.Window{},
		ActiveWindow: 0,
		CreatedAt:    time.Now(),
		Status:       terminal.SessionActive,
	}

	// 添加一些测试窗口
	for i := 0; i < 3; i++ {
		window := &terminal.Window{
			ID:         fmt.Sprintf("window-%d", i),
			Name:       fmt.Sprintf("window-%d", i),
			Index:      i,
			ActivePane: 0,
			Panes:      []*terminal.Pane{},
		}

		// 添加一些测试面板
		for j := 0; j < 2; j++ {
			pane := &terminal.Pane{
				ID:     fmt.Sprintf("pane-%d-%d", i, j),
				Index:  j,
				Active: j == 0,
			}
			window.Panes = append(window.Panes, pane)
		}

		testSession.Windows = append(testSession.Windows, window)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 测试转换性能
		tmuxSession := ConvertTerminalSessionToTmux(testSession)
		if tmuxSession == nil {
			b.Error("Conversion failed")
			continue
		}

		// 测试反向转换
		backSession := ConvertTmuxSessionToTerminal(tmuxSession)
		if backSession == nil {
			b.Error("Reverse conversion failed")
		}
	}
}
