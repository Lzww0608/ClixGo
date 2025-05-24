package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

func main() {
	fmt.Println("=== ClixGo Terminal 基本使用示例 ===")

	// 0. 初始化 logger
	logger.InitLogger()

	// 1. 创建配置
	config := &terminal.TerminalConfig{
		PrefixKey:    "C-b",
		MouseEnabled: true,
		StatusBar:    true,
		AutoSave:     true,
		SaveInterval: time.Minute * 2,
		Theme:        "default",
		BufferSize:   1000,
		ScrollBack:   1000,
		KeyBindings: []terminal.KeyBinding{
			{Key: "C-b d", Command: "detach_session"},
			{Key: "C-b c", Command: "create_window"},
			{Key: "C-b \"", Command: "split_pane", Args: []string{"horizontal"}},
			{Key: "C-b %", Command: "split_pane", Args: []string{"vertical"}},
		},
	}

	// 2. 启动服务器
	fmt.Println("启动终端服务器...")
	server := terminal.NewTerminalServer(config)

	if err := server.Start(); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
	defer server.Stop()

	fmt.Printf("服务器已启动，Socket路径: %s\n", server.GetSocketPath())

	// 等待一秒确保服务器完全启动
	time.Sleep(time.Second)

	// 3. 创建客户端
	fmt.Println("创建客户端...")
	client := terminal.NewTerminalClient(config)

	if err := client.Connect(); err != nil {
		log.Fatalf("连接服务器失败: %v", err)
	}
	defer client.Disconnect()

	// 4. 创建会话
	fmt.Println("创建新会话...")
	if err := client.CreateSession("example-session"); err != nil {
		log.Fatalf("创建会话失败: %v", err)
	}

	// 5. 显示服务器状态
	fmt.Printf("服务器运行状态: %v\n", server.IsRunning())
	fmt.Printf("客户端连接数: %d\n", server.GetClientCount())

	// 6. 列出会话
	sessions := server.GetSessionManager().ListSessions()
	fmt.Printf("活动会话数: %d\n", len(sessions))

	for _, session := range sessions {
		fmt.Printf("  会话: %s (状态: %s, 窗口数: %d)\n",
			session.Name, session.Status, len(session.Windows))
	}

	fmt.Println("\n示例完成！你可以使用以下命令与终端交互:")
	fmt.Println("  clixgo terminal new-session <name>    # 创建新会话")
	fmt.Println("  clixgo terminal attach <name>         # 连接会话")
	fmt.Println("  clixgo terminal list-sessions         # 列出会话")
	fmt.Println("  clixgo terminal server status         # 查看服务器状态")
}
