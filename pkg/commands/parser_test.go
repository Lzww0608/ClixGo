package commands

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	messages []string
}

func (m *MockLogger) Debug(msg string, args ...interface{}) {
	m.messages = append(m.messages, fmt.Sprintf("DEBUG: "+msg, args...))
}

func (m *MockLogger) Info(msg string, args ...interface{}) {
	m.messages = append(m.messages, fmt.Sprintf("INFO: "+msg, args...))
}

func (m *MockLogger) Error(msg string, args ...interface{}) {
	m.messages = append(m.messages, fmt.Sprintf("ERROR: "+msg, args...))
}

func (m *MockLogger) GetMessages() []string {
	return m.messages
}

func (m *MockLogger) Clear() {
	m.messages = nil
}

// TestModernParser tests the modern command parser
func TestModernParser(t *testing.T) {
	logger := &MockLogger{}
	parser := NewModernParser(logger)

	// Test command registration
	newCmd := NewNewSessionCommand(logger)
	err := parser.RegisterCommand(newCmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	// Test command retrieval
	cmd, exists := parser.GetCommand("session new")
	if !exists {
		t.Fatal("Command not found after registration")
	}

	if cmd.Name() != "session new" {
		t.Errorf("Expected command name 'session new', got '%s'", cmd.Name())
	}

	// Test command listing
	commands := parser.ListCommands()
	if len(commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(commands))
	}
}

// TestTmuxCompatibility tests tmux command compatibility
func TestTmuxCompatibility(t *testing.T) {
	logger := &MockLogger{}
	parser := NewModernParser(logger)
	compat := NewTmuxCompatLayer(parser, logger)

	// Register session commands
	parser.RegisterCommand(NewNewSessionCommand(logger))
	parser.RegisterCommand(NewAttachSessionCommand(logger))
	parser.RegisterCommand(NewListSessionsCommand(logger))

	// Test tmux command mapping
	testCases := []struct {
		tmuxCmd    string
		shouldWork bool
	}{
		{"new-session -s test", true},
		{"new -s test -d", true},
		{"attach-session -t test", true},
		{"attach -t test", true},
		{"list-sessions", true},
		{"ls", true},
		{"unknown-command", false},
	}

	for _, tc := range testCases {
		t.Run(tc.tmuxCmd, func(t *testing.T) {
			cmdList, err := compat.ParseTmuxCommand(tc.tmuxCmd)

			if tc.shouldWork {
				if err != nil {
					t.Errorf("Expected command to work, got error: %v", err)
				}
				if cmdList == nil {
					t.Error("Expected command list, got nil")
				}
			} else {
				if err == nil {
					t.Error("Expected error for invalid command")
				}
			}
		})
	}
}

// TestArgumentParsing tests argument parsing functionality
func TestArgumentParsing(t *testing.T) {
	logger := &MockLogger{}
	parser := NewModernParser(logger)

	// Register test command
	newCmd := NewNewSessionCommand(logger)
	parser.RegisterCommand(newCmd)

	// Test cases for argument parsing
	testCases := []struct {
		input    string
		expected map[string]interface{}
	}{
		{
			"session new -s test",
			map[string]interface{}{"session-name": "test"},
		},
		{
			"session new -d -s test -n window1",
			map[string]interface{}{
				"detached":     true,
				"session-name": "test",
				"window-name":  "window1",
			},
		},
		{
			"session new -c /home/user",
			map[string]interface{}{"start-directory": "/home/user"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			cmdList, err := parser.Parse(tc.input)
			if err != nil {
				t.Fatalf("Failed to parse command: %v", err)
			}

			if len(cmdList.Args) == 0 {
				t.Fatal("No arguments parsed")
			}

			args := cmdList.Args[0]
			for key, expectedValue := range tc.expected {
				actualValue, exists := args.Flags[key]
				if !exists {
					t.Errorf("Expected flag '%s' not found", key)
					continue
				}

				if actualValue != expectedValue {
					t.Errorf("Flag '%s': expected %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

// TestKeyBindings tests key binding functionality
func TestKeyBindings(t *testing.T) {
	logger := &MockLogger{}
	parser := NewModernParser(logger)
	compat := NewTmuxCompatLayer(parser, logger)

	// Test key binding lookup
	testKeys := []string{
		"c",     // new window
		"d",     // detach
		"s",     // choose session
		"C-b",   // send prefix
		"Space", // next layout
	}

	for _, key := range testKeys {
		t.Run(key, func(t *testing.T) {
			cmdList, err := compat.HandleKeyBinding(key)

			// Most key bindings should work, but some might need registered commands
			if err != nil && !strings.Contains(err.Error(), "command not found") {
				t.Errorf("Unexpected error for key '%s': %v", key, err)
			}

			if err == nil && cmdList == nil {
				t.Errorf("Expected command list for key '%s'", key)
			}
		})
	}
}

// TestCommandExecution tests command execution
func TestCommandExecution(t *testing.T) {
	logger := &MockLogger{}
	parser := NewModernParser(logger)

	// Register test command
	newCmd := NewNewSessionCommand(logger)
	parser.RegisterCommand(newCmd)

	// Parse command
	cmdList, err := parser.Parse("session new -s test")
	if err != nil {
		t.Fatalf("Failed to parse command: %v", err)
	}

	// Create execution context
	ctx := &Context{
		Variables: make(map[string]interface{}),
		Logger:    logger,
	}

	// Execute command
	err = cmdList.Execute(ctx)
	if err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	// Verify session was created
	if ctx.Session == nil {
		t.Error("Session was not created")
	} else if ctx.Session.Name != "test" {
		t.Errorf("Expected session name 'test', got '%s'", ctx.Session.Name)
	}

	// Verify window was created
	if ctx.Window == nil {
		t.Error("Window was not created")
	}

	// Verify pane was created
	if ctx.Pane == nil {
		t.Error("Pane was not created")
	}
}

// TestTokenization tests input tokenization
func TestTokenization(t *testing.T) {
	logger := &MockLogger{}
	parser := NewModernParser(logger)

	testCases := []struct {
		input    string
		expected []string
	}{
		{
			`session new -s "test session"`,
			[]string{"session", "new", "-s", "test session"},
		},
		{
			`session new -s 'test session'`,
			[]string{"session", "new", "-s", "test session"},
		},
		{
			`session new -c "/home/user with spaces"`,
			[]string{"session", "new", "-c", "/home/user with spaces"},
		},
		{
			`session new command\ with\ escapes`,
			[]string{"session", "new", "command with escapes"},
		},
		{
			`session new -s test   -d`,
			[]string{"session", "new", "-s", "test", "-d"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			tokens := parser.tokenize(tc.input)

			if len(tokens) != len(tc.expected) {
				t.Errorf("Expected %d tokens, got %d", len(tc.expected), len(tokens))
				t.Errorf("Expected: %v", tc.expected)
				t.Errorf("Got: %v", tokens)
				return
			}

			for i, expected := range tc.expected {
				if tokens[i] != expected {
					t.Errorf("Token %d: expected '%s', got '%s'", i, expected, tokens[i])
				}
			}
		})
	}
}

// BenchmarkParseCommand benchmarks command parsing performance
func BenchmarkParseCommand(b *testing.B) {
	logger := &MockLogger{}
	parser := NewModernParser(logger)

	// Register test command
	newCmd := NewNewSessionCommand(logger)
	parser.RegisterCommand(newCmd)

	input := "session new -s test -d -n window1 -c /home/user"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

// BenchmarkTmuxCompat benchmarks tmux compatibility parsing
func BenchmarkTmuxCompat(b *testing.B) {
	logger := &MockLogger{}
	parser := NewModernParser(logger)
	compat := NewTmuxCompatLayer(parser, logger)

	// Register test command
	parser.RegisterCommand(NewNewSessionCommand(logger))

	input := "new-session -s test -d"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := compat.ParseTmuxCommand(input)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

// BenchmarkCommandExecution benchmarks command execution
func BenchmarkCommandExecution(b *testing.B) {
	logger := &MockLogger{}
	parser := NewModernParser(logger)

	// Register test command
	newCmd := NewNewSessionCommand(logger)
	parser.RegisterCommand(newCmd)

	// Pre-parse command
	cmdList, err := parser.Parse("session new -s test")
	if err != nil {
		b.Fatalf("Parse failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := &Context{
			Variables: make(map[string]interface{}),
			Logger:    logger,
		}

		err := cmdList.Execute(ctx)
		if err != nil {
			b.Fatalf("Execute failed: %v", err)
		}
	}
}

// TestComplexCommands tests complex command scenarios
func TestComplexCommands(t *testing.T) {
	logger := &MockLogger{}
	parser := NewModernParser(logger)
	compat := NewTmuxCompatLayer(parser, logger)

	// Register commands
	parser.RegisterCommand(NewNewSessionCommand(logger))
	parser.RegisterCommand(NewRenameSessionCommand(logger))

	// Test command chaining (semicolon separated)
	input := `new-session -s test -d; rename-session -t test "new name"`

	cmdList, err := compat.ParseTmuxCommand(input)
	if err != nil {
		t.Fatalf("Failed to parse complex command: %v", err)
	}

	if len(cmdList.Commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(cmdList.Commands))
	}
}

// TestErrorHandling tests error handling scenarios
func TestErrorHandling(t *testing.T) {
	logger := &MockLogger{}
	parser := NewModernParser(logger)

	// Test empty input
	_, err := parser.Parse("")
	if err == nil {
		t.Error("Expected error for empty input")
	}

	// Test unknown command
	_, err = parser.Parse("unknown-command")
	if err == nil {
		t.Error("Expected error for unknown command")
	}

	// Test duplicate command registration
	newCmd := NewNewSessionCommand(logger)
	parser.RegisterCommand(newCmd)
	err = parser.RegisterCommand(newCmd)
	if err == nil {
		t.Error("Expected error for duplicate command registration")
	}
}

// TestPerformanceComparison compares performance with tmux
func TestPerformanceComparison(t *testing.T) {
	logger := &MockLogger{}
	parser := NewModernParser(logger)
	compat := NewTmuxCompatLayer(parser, logger)

	// Register commands
	parser.RegisterCommand(NewNewSessionCommand(logger))
	parser.RegisterCommand(NewAttachSessionCommand(logger))
	parser.RegisterCommand(NewListSessionsCommand(logger))

	// Test commands that would be common in tmux usage
	commands := []string{
		"new-session -s work",
		"new-session -s personal -d",
		"attach-session -t work",
		"list-sessions",
		"new-session -s temp -d -c /tmp",
	}

	start := time.Now()
	iterations := 1000

	for i := 0; i < iterations; i++ {
		for _, cmd := range commands {
			_, err := compat.ParseTmuxCommand(cmd)
			if err != nil {
				t.Fatalf("Command failed: %s - %v", cmd, err)
			}
		}
	}

	elapsed := time.Since(start)
	avgPerCommand := elapsed / time.Duration(iterations*len(commands))

	t.Logf("Performance: %d commands in %v", iterations*len(commands), elapsed)
	t.Logf("Average per command: %v", avgPerCommand)

	// Performance target: should be faster than 1ms per command
	if avgPerCommand > time.Millisecond {
		t.Errorf("Performance too slow: %v per command (target: < 1ms)", avgPerCommand)
	}
}
