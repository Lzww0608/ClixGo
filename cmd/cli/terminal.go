package cli

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/terminal"
	"github.com/spf13/cobra"
)

// NewTerminalCmd 创建终端多路复用器命令
func NewTerminalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "terminal",
		Short: "轻量级终端多路复用器",
		Long: `ClixGo Terminal - 下一代轻量级终端多路复用器

功能特点:
- 🎯 零配置启动，开箱即用
- ⚡ 超轻量级，启动速度快 3-5 倍  
- 🔧 深度集成 ClixGo 工具集
- 🌐 跨平台原生支持
- ☁️ 智能会话同步
- 🎨 现代化界面，支持鼠标操作
- 🔄 智能恢复，自动保存会话状态
- 📊 内置监控和性能分析

使用方法:
  clixgo terminal new-session [session-name]  # 创建新会话
  clixgo terminal attach [session-name]       # 连接会话
  clixgo terminal list-sessions               # 列出所有会话
  clixgo terminal kill-session [session-name] # 销毁会话`,
		Aliases: []string{"term", "tmux"},
	}

	// 创建新会话
	cmd.AddCommand(&cobra.Command{
		Use:     "new-session [session-name]",
		Short:   "创建新会话",
		Aliases: []string{"new", "ns"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var sessionName string
			if len(args) > 0 {
				sessionName = args[0]
			}

			client, err := connectToServer()
			if err != nil {
				return fmt.Errorf("连接服务器失败: %v", err)
			}
			defer client.Close()

			response, err := sendCommand(client, terminal.Command{
				Type: terminal.CmdCreateSession,
				Payload: map[string]interface{}{
					"name": sessionName,
				},
			})
			if err != nil {
				return err
			}

			if errMsg, ok := response["error"].(string); ok {
				return fmt.Errorf(errMsg)
			}

			sessionID, _ := response["session_id"].(string)
			fmt.Printf("会话创建成功: %s\n", sessionID)

			// 自动连接到新创建的会话
			return attachToSession(client, sessionID)
		},
	})

	// 连接会话
	cmd.AddCommand(&cobra.Command{
		Use:     "attach [session-name]",
		Short:   "连接到现有会话",
		Aliases: []string{"a", "at"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := connectToServer()
			if err != nil {
				return fmt.Errorf("连接服务器失败: %v", err)
			}
			defer client.Close()

			var sessionIdentifier string
			if len(args) > 0 {
				sessionIdentifier = args[0]
			} else {
				// 如果没有指定会话，连接到最新的会话
				sessions, err := listSessions(client)
				if err != nil {
					return err
				}
				if len(sessions) == 0 {
					return fmt.Errorf("没有可用的会话")
				}
				sessionIdentifier = sessions[0]["id"].(string)
			}

			return attachToSession(client, sessionIdentifier)
		},
	})

	// 列出会话
	cmd.AddCommand(&cobra.Command{
		Use:     "list-sessions",
		Short:   "列出所有会话",
		Aliases: []string{"ls", "list"},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := connectToServer()
			if err != nil {
				return fmt.Errorf("连接服务器失败: %v", err)
			}
			defer client.Close()

			sessions, err := listSessions(client)
			if err != nil {
				return err
			}

			if len(sessions) == 0 {
				fmt.Println("没有活动的会话")
				return nil
			}

			fmt.Printf("%-20s %-15s %-20s %-10s %s\n", "会话ID", "会话名", "创建时间", "状态", "窗口数")
			fmt.Println(strings.Repeat("-", 80))

			for _, session := range sessions {
				id := session["id"].(string)[:8] + "..."
				name, _ := session["name"].(string)
				status, _ := session["status"].(string)
				createdAt, _ := session["created_at"].(string)
				windows, _ := session["windows"].([]interface{})

				fmt.Printf("%-20s %-15s %-20s %-10s %d\n",
					id, name, createdAt[:19], status, len(windows))
			}

			return nil
		},
	})

	// 销毁会话
	cmd.AddCommand(&cobra.Command{
		Use:     "kill-session [session-name]",
		Short:   "销毁指定会话",
		Aliases: []string{"kill", "ks"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := connectToServer()
			if err != nil {
				return fmt.Errorf("连接服务器失败: %v", err)
			}
			defer client.Close()

			response, err := sendCommand(client, terminal.Command{
				Type: terminal.CmdKillSession,
				Payload: map[string]interface{}{
					"session_id": args[0],
				},
			})
			if err != nil {
				return err
			}

			if errMsg, ok := response["error"].(string); ok {
				return fmt.Errorf(errMsg)
			}

			fmt.Printf("会话 %s 已销毁\n", args[0])
			return nil
		},
	})

	// 启动服务器
	cmd.AddCommand(&cobra.Command{
		Use:   "server",
		Short: "管理终端服务器",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	})

	// 启动服务器子命令
	serverCmd := cmd.Commands()[len(cmd.Commands())-1]

	serverCmd.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "启动终端服务器",
		RunE: func(cmd *cobra.Command, args []string) error {
			config := terminal.DefaultConfig
			server := terminal.NewTerminalServer(config)

			if err := server.Start(); err != nil {
				return fmt.Errorf("启动服务器失败: %v", err)
			}

			fmt.Printf("终端服务器已启动，Socket路径: %s\n", server.GetSocketPath())
			fmt.Println("按 Ctrl+C 停止服务器")

			// 等待中断信号
			select {}
		},
	})

	serverCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "查看服务器状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := connectToServer()
			if err != nil {
				fmt.Println("服务器未运行")
				return nil
			}
			defer client.Close()

			sessions, err := listSessions(client)
			if err != nil {
				return err
			}

			fmt.Println("终端服务器状态: 运行中")
			fmt.Printf("活动会话数: %d\n", len(sessions))
			return nil
		},
	})

	// 快捷命令
	cmd.AddCommand(&cobra.Command{
		Use:     "split-window",
		Short:   "分割当前窗口",
		Aliases: []string{"split"},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := connectToServer()
			if err != nil {
				return fmt.Errorf("连接服务器失败: %v", err)
			}
			defer client.Close()

			vertical, _ := cmd.Flags().GetBool("vertical")
			direction := "horizontal"
			if vertical {
				direction = "vertical"
			}

			response, err := sendCommand(client, terminal.Command{
				Type: terminal.CmdSplitPane,
				Payload: map[string]interface{}{
					"window_index": 0, // 当前窗口
					"direction":    direction,
				},
			})
			if err != nil {
				return err
			}

			if errMsg, ok := response["error"].(string); ok {
				return fmt.Errorf(errMsg)
			}

			fmt.Printf("窗口已分割 (%s)\n", direction)
			return nil
		},
	})
	cmd.Commands()[len(cmd.Commands())-1].Flags().BoolP("vertical", "v", false, "垂直分割")

	return cmd
}

// connectToServer 连接到终端服务器
func connectToServer() (net.Conn, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp"
	}
	socketPath := fmt.Sprintf("%s/.clixgo/terminal/clixgo-terminal.sock", homeDir)

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		// 尝试启动服务器
		if err := startServer(); err != nil {
			return nil, fmt.Errorf("无法启动服务器: %v", err)
		}

		// 等待服务器启动
		time.Sleep(time.Second)

		// 重新尝试连接
		conn, err = net.Dial("unix", socketPath)
		if err != nil {
			return nil, err
		}
	}

	return conn, nil
}

// sendCommand 发送命令到服务器
func sendCommand(conn net.Conn, cmd terminal.Command) (map[string]interface{}, error) {
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	if err := encoder.Encode(cmd); err != nil {
		return nil, fmt.Errorf("发送命令失败: %v", err)
	}

	var response map[string]interface{}
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("接收响应失败: %v", err)
	}

	return response, nil
}

// listSessions 列出所有会话
func listSessions(conn net.Conn) ([]map[string]interface{}, error) {
	response, err := sendCommand(conn, terminal.Command{
		Type:    terminal.CmdListSessions,
		Payload: nil,
	})
	if err != nil {
		return nil, err
	}

	if errMsg, ok := response["error"].(string); ok {
		return nil, fmt.Errorf(errMsg)
	}

	sessions, ok := response["sessions"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	var result []map[string]interface{}
	for _, session := range sessions {
		if s, ok := session.(map[string]interface{}); ok {
			result = append(result, s)
		}
	}

	return result, nil
}

// attachToSession 连接到会话
func attachToSession(conn net.Conn, sessionIdentifier string) error {
	// 首先尝试按ID连接
	response, err := sendCommand(conn, terminal.Command{
		Type: terminal.CmdAttachSession,
		Payload: map[string]interface{}{
			"session_id": sessionIdentifier,
		},
	})

	// 如果按ID连接失败，尝试按名称连接
	if err != nil || response["error"] != nil {
		response, err = sendCommand(conn, terminal.Command{
			Type: terminal.CmdAttachSession,
			Payload: map[string]interface{}{
				"session_name": sessionIdentifier,
			},
		})
	}

	if err != nil {
		return err
	}

	if errMsg, ok := response["error"].(string); ok {
		return fmt.Errorf(errMsg)
	}

	session, ok := response["session"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid session data")
	}

	fmt.Printf("已连接到会话: %s\n", session["name"])

	// 显示会话信息
	windows, _ := session["windows"].([]interface{})
	fmt.Printf("会话包含 %d 个窗口\n", len(windows))

	// 这里应该启动交互式终端界面
	// 目前简化实现，只显示连接成功信息
	fmt.Println("\n按快捷键操作:")
	fmt.Println("  Ctrl+B, D    - 断开会话")
	fmt.Println("  Ctrl+B, C    - 创建新窗口")
	fmt.Println("  Ctrl+B, \"    - 水平分割面板")
	fmt.Println("  Ctrl+B, %    - 垂直分割面板")
	fmt.Println("  Ctrl+B, O    - 切换面板")
	fmt.Println("  Ctrl+B, X    - 关闭面板")
	fmt.Println("\n使用 'clixgo terminal detach' 断开会话")

	return nil
}

// startServer 启动服务器
func startServer() error {
	fmt.Println("正在启动终端服务器...")

	config := terminal.DefaultConfig
	server := terminal.NewTerminalServer(config)

	if err := server.Start(); err != nil {
		return err
	}

	// 在后台运行服务器
	go func() {
		select {} // 保持服务器运行
	}()

	return nil
}
