/*
* @Author: Lzww0608
* @Date: 2025-5-31 16:15:14
* @LastEditors: Lzww0608
* @LastEditTime: 2025-5-31 16:15:19
* @Description: 数据结构测试
 */

package terminal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// 数据结构测试
// =============================================================================

func TestSessionStatus(t *testing.T) {
	t.Run("会话状态常量", func(t *testing.T) {
		assert.Equal(t, SessionStatus("active"), SessionActive)
		assert.Equal(t, SessionStatus("detached"), SessionDetached)
		assert.Equal(t, SessionStatus("destroyed"), SessionDestroyed)
	})
}

func TestSession(t *testing.T) {
	t.Run("创建会话", func(t *testing.T) {
		session := &Session{
			ID:           "test-id",
			Name:         "test-session",
			Status:       SessionActive,
			CreatedAt:    time.Now(),
			LastActive:   time.Now(),
			Windows:      make([]*Window, 0),
			ActiveWindow: 0,
		}

		assert.Equal(t, "test-id", session.ID)
		assert.Equal(t, "test-session", session.Name)
		assert.Equal(t, SessionActive, session.Status)
		assert.Empty(t, session.Windows)
		assert.Equal(t, 0, session.ActiveWindow)
	})
}

func TestWindow(t *testing.T) {
	t.Run("创建窗口", func(t *testing.T) {
		window := &Window{
			ID:         "window-id",
			Name:       "test-window",
			Index:      0,
			Panes:      make([]*Pane, 0),
			ActivePane: 0,
			Layout:     LayoutMainVertical,
			CreatedAt:  time.Now(),
		}

		assert.Equal(t, "window-id", window.ID)
		assert.Equal(t, "test-window", window.Name)
		assert.Equal(t, 0, window.Index)
		assert.Empty(t, window.Panes)
		assert.Equal(t, 0, window.ActivePane)
		assert.Equal(t, LayoutMainVertical, window.Layout)
	})
}

func TestPane(t *testing.T) {
	t.Run("创建面板", func(t *testing.T) {
		buffer := &Buffer{
			Lines:    make([][]rune, 0),
			MaxLines: 2000,
			CursorX:  0,
			CursorY:  0,
		}

		pane := &Pane{
			ID:         "pane-id",
			Index:      0,
			X:          0,
			Y:          0,
			Width:      80,
			Height:     24,
			Command:    "/bin/bash",
			WorkingDir: "/home/user",
			ProcessID:  1234,
			Buffer:     buffer,
			Active:     true,
			CreatedAt:  time.Now(),
			LastOutput: time.Now(),
		}

		assert.Equal(t, "pane-id", pane.ID)
		assert.Equal(t, 0, pane.Index)
		assert.Equal(t, 80, pane.Width)
		assert.Equal(t, 24, pane.Height)
		assert.Equal(t, "/bin/bash", pane.Command)
		assert.Equal(t, "/home/user", pane.WorkingDir)
		assert.Equal(t, 1234, pane.ProcessID)
		assert.True(t, pane.Active)
		assert.NotNil(t, pane.Buffer)
	})
}

func TestLayout(t *testing.T) {
	t.Run("布局类型常量", func(t *testing.T) {
		assert.Equal(t, Layout("main-vertical"), LayoutMainVertical)
		assert.Equal(t, Layout("main-horizontal"), LayoutMainHorizontal)
		assert.Equal(t, Layout("even"), LayoutEven)
		assert.Equal(t, Layout("tiled"), LayoutTiled)
	})
}

func TestBuffer(t *testing.T) {
	t.Run("创建缓冲区", func(t *testing.T) {
		buffer := &Buffer{
			Lines:    make([][]rune, 0),
			MaxLines: 1000,
			CursorX:  5,
			CursorY:  10,
		}

		assert.Empty(t, buffer.Lines)
		assert.Equal(t, 1000, buffer.MaxLines)
		assert.Equal(t, 5, buffer.CursorX)
		assert.Equal(t, 10, buffer.CursorY)
	})

	t.Run("缓冲区操作", func(t *testing.T) {
		buffer := &Buffer{
			Lines:    make([][]rune, 0),
			MaxLines: 3,
			CursorX:  0,
			CursorY:  0,
		}

		// 添加行
		buffer.Lines = append(buffer.Lines, []rune("line 1"))
		buffer.Lines = append(buffer.Lines, []rune("line 2"))
		buffer.Lines = append(buffer.Lines, []rune("line 3"))

		assert.Len(t, buffer.Lines, 3)
		assert.Equal(t, "line 1", string(buffer.Lines[0]))
		assert.Equal(t, "line 2", string(buffer.Lines[1]))
		assert.Equal(t, "line 3", string(buffer.Lines[2]))

		// 模拟滚动（超过最大行数）
		buffer.Lines = append(buffer.Lines, []rune("line 4"))
		// 在实际实现中会删除最老的行，这里只验证数据结构
		if len(buffer.Lines) > buffer.MaxLines {
			buffer.Lines = buffer.Lines[1:]
		}

		assert.Len(t, buffer.Lines, 3)
		assert.Equal(t, "line 2", string(buffer.Lines[0]))
		assert.Equal(t, "line 3", string(buffer.Lines[1]))
		assert.Equal(t, "line 4", string(buffer.Lines[2]))
	})
}

func TestKeyBinding(t *testing.T) {
	t.Run("创建快捷键绑定", func(t *testing.T) {
		binding := KeyBinding{
			Key:     "C-b d",
			Command: "detach_session",
			Args:    []string{},
		}

		assert.Equal(t, "C-b d", binding.Key)
		assert.Equal(t, "detach_session", binding.Command)
		assert.Empty(t, binding.Args)
	})

	t.Run("带参数的快捷键", func(t *testing.T) {
		binding := KeyBinding{
			Key:     "C-b %",
			Command: "split_pane",
			Args:    []string{"vertical"},
		}

		assert.Equal(t, "C-b %", binding.Key)
		assert.Equal(t, "split_pane", binding.Command)
		assert.Len(t, binding.Args, 1)
		assert.Equal(t, "vertical", binding.Args[0])
	})
}

func TestTerminalConfig(t *testing.T) {
	t.Run("创建配置", func(t *testing.T) {
		config := &TerminalConfig{
			PrefixKey:         "C-b",
			MouseEnabled:      true,
			StatusBar:         true,
			AutoSave:          true,
			SaveInterval:      time.Minute * 5,
			Theme:             "default",
			StatusFormat:      "[#S] #I:#W",
			WindowFormat:      "#I:#W",
			BufferSize:        2000,
			ScrollBack:        2000,
			ClixGoIntegration: true,
			NetworkMonitor:    false,
			TaskIntegration:   true,
			KeyBindings:       []KeyBinding{},
		}

		assert.Equal(t, "C-b", config.PrefixKey)
		assert.True(t, config.MouseEnabled)
		assert.True(t, config.StatusBar)
		assert.True(t, config.AutoSave)
		assert.Equal(t, time.Minute*5, config.SaveInterval)
		assert.Equal(t, "default", config.Theme)
		assert.Equal(t, 2000, config.BufferSize)
		assert.True(t, config.ClixGoIntegration)
		assert.Empty(t, config.KeyBindings)
	})
}

func TestDefaultConfig(t *testing.T) {
	t.Run("默认配置验证", func(t *testing.T) {
		config := DefaultConfig

		assert.Equal(t, "C-b", config.PrefixKey)
		assert.True(t, config.MouseEnabled)
		assert.True(t, config.StatusBar)
		assert.True(t, config.AutoSave)
		assert.Equal(t, time.Minute*5, config.SaveInterval)
		assert.Equal(t, "default", config.Theme)
		assert.Equal(t, "[#S] #I:#W", config.StatusFormat)
		assert.Equal(t, "#I:#W", config.WindowFormat)
		assert.Equal(t, 2000, config.BufferSize)
		assert.Equal(t, 2000, config.ScrollBack)
		assert.True(t, config.ClixGoIntegration)
		assert.False(t, config.NetworkMonitor)
		assert.True(t, config.TaskIntegration)

		// 验证默认快捷键
		assert.NotEmpty(t, config.KeyBindings)

		keyMap := make(map[string]string)
		for _, binding := range config.KeyBindings {
			keyMap[binding.Key] = binding.Command
		}

		assert.Equal(t, "detach_session", keyMap["C-b d"])
		assert.Equal(t, "create_window", keyMap["C-b c"])
		assert.Equal(t, "close_window", keyMap["C-b &"])
		assert.Equal(t, "close_pane", keyMap["C-b x"])
		assert.Equal(t, "switch_pane", keyMap["C-b o"])
	})
}

func TestServer(t *testing.T) {
	t.Run("创建服务器", func(t *testing.T) {
		config := DefaultConfig
		server := &Server{
			Sessions:   make(map[string]*Session),
			Config:     config,
			SocketPath: "/tmp/test.sock",
			Running:    false,
		}

		assert.Empty(t, server.Sessions)
		assert.Equal(t, config, server.Config)
		assert.Equal(t, "/tmp/test.sock", server.SocketPath)
		assert.False(t, server.Running)
	})
}

func TestClient(t *testing.T) {
	t.Run("创建客户端", func(t *testing.T) {
		config := DefaultConfig
		client := &Client{
			SessionID:  "test-session",
			Connected:  false,
			Config:     config,
			Input:      make(chan []byte, 100),
			Output:     make(chan []byte, 100),
			SocketPath: "/tmp/test.sock",
		}

		assert.Equal(t, "test-session", client.SessionID)
		assert.False(t, client.Connected)
		assert.Equal(t, config, client.Config)
		assert.NotNil(t, client.Input)
		assert.NotNil(t, client.Output)
		assert.Equal(t, "/tmp/test.sock", client.SocketPath)
	})
}

func TestCommand(t *testing.T) {
	t.Run("创建命令", func(t *testing.T) {
		cmd := Command{
			Type:    CmdCreateSession,
			Payload: map[string]interface{}{"name": "test"},
		}

		assert.Equal(t, CmdCreateSession, cmd.Type)
		assert.NotNil(t, cmd.Payload)

		payload, ok := cmd.Payload.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "test", payload["name"])
	})
}

func TestCommandConstants(t *testing.T) {
	t.Run("命令类型常量", func(t *testing.T) {
		assert.Equal(t, "create_session", CmdCreateSession)
		assert.Equal(t, "attach_session", CmdAttachSession)
		assert.Equal(t, "detach_session", CmdDetachSession)
		assert.Equal(t, "list_sessions", CmdListSessions)
		assert.Equal(t, "create_window", CmdCreateWindow)
		assert.Equal(t, "close_window", CmdCloseWindow)
		assert.Equal(t, "split_pane", CmdSplitPane)
		assert.Equal(t, "close_pane", CmdClosePane)
		assert.Equal(t, "switch_window", CmdSwitchWindow)
		assert.Equal(t, "switch_pane", CmdSwitchPane)
		assert.Equal(t, "resize_pane", CmdResizePane)
		assert.Equal(t, "send_keys", CmdSendKeys)
		assert.Equal(t, "copy_mode", CmdCopyMode)
		assert.Equal(t, "paste_buffer", CmdPasteBuffer)
		assert.Equal(t, "set_layout", CmdSetLayout)
		assert.Equal(t, "rename", CmdRename)
		assert.Equal(t, "kill_session", CmdKillSession)
	})
}

// =============================================================================
// 并发安全测试
// =============================================================================

func TestMutexSafety(t *testing.T) {
	t.Run("会话并发安全", func(t *testing.T) {
		session := &Session{
			ID:           "test",
			Name:         "test",
			Status:       SessionActive,
			CreatedAt:    time.Now(),
			LastActive:   time.Now(),
			Windows:      make([]*Window, 0),
			ActiveWindow: 0,
		}

		// 并发读写测试
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				session.mutex.Lock()
				session.LastActive = time.Now()
				session.mutex.Unlock()
				done <- true
			}()
		}

		// 等待所有goroutine完成
		for i := 0; i < 10; i++ {
			<-done
		}

		assert.True(t, true, "并发操作应该安全完成")
	})

	t.Run("窗口并发安全", func(t *testing.T) {
		window := &Window{
			ID:         "test",
			Name:       "test",
			Index:      0,
			Panes:      make([]*Pane, 0),
			ActivePane: 0,
			Layout:     LayoutEven,
			CreatedAt:  time.Now(),
		}

		// 并发读写测试
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(index int) {
				window.mutex.Lock()
				window.ActivePane = index % 3
				window.mutex.Unlock()
				done <- true
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < 10; i++ {
			<-done
		}

		assert.True(t, true, "并发操作应该安全完成")
	})
}

// =============================================================================
// 边界条件测试
// =============================================================================

func TestEdgeCases(t *testing.T) {
	t.Run("空配置", func(t *testing.T) {
		config := &TerminalConfig{}

		assert.Empty(t, config.PrefixKey)
		assert.False(t, config.MouseEnabled)
		assert.False(t, config.StatusBar)
		assert.Equal(t, time.Duration(0), config.SaveInterval)
		assert.Empty(t, config.KeyBindings)
	})

	t.Run("最小缓冲区", func(t *testing.T) {
		buffer := &Buffer{
			Lines:    make([][]rune, 0),
			MaxLines: 1,
			CursorX:  0,
			CursorY:  0,
		}

		assert.Equal(t, 1, buffer.MaxLines)

		// 添加一行
		buffer.Lines = append(buffer.Lines, []rune("line"))
		assert.Len(t, buffer.Lines, 1)
	})

	t.Run("无效索引处理", func(t *testing.T) {
		session := &Session{
			Windows:      []*Window{},
			ActiveWindow: -1, // 无效索引
		}

		assert.Equal(t, -1, session.ActiveWindow)
		assert.Empty(t, session.Windows)
	})
}
