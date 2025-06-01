/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-1 22:07:40
* @Description: 终端基础使用示例
 */

package main

import (
	"fmt"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/terminal"
	"go.uber.org/zap"
)

func main() {
	// 初始化日志系统
	if err := logger.InitLogger(); err != nil {
		fmt.Printf("初始化日志系统失败: %v\n", err)
		return
	}
	defer logger.Close()

	fmt.Println("🚀 ClixGo 终端基础使用示例")
	fmt.Println("============================")

	// 创建终端配置
	config := terminal.DefaultConfig
	config.AutoSave = true
	config.SaveInterval = time.Minute * 2

	// 创建并启动终端服务器
	server := terminal.NewTerminalServer(config)
	if err := server.Start(); err != nil {
		logger.Error("启动服务器失败", zap.Error(err))
		return
	}
	defer server.Stop()

	fmt.Printf("✅ 终端服务器已启动，Socket路径: %s\n", server.GetSocketPath())

	// 等待服务器完全启动
	time.Sleep(time.Second)

	// 创建客户端连接
	client := terminal.NewTerminalClient(config)
	if err := client.Connect(); err != nil {
		logger.Error("连接服务器失败", zap.Error(err))
		return
	}
	defer client.Disconnect()

	fmt.Println("✅ 客户端已连接到服务器")

	// 创建会话
	if err := client.CreateSession("demo-session"); err != nil {
		logger.Error("创建会话失败", zap.Error(err))
		return
	}

	fmt.Println("✅ 会话已创建: demo-session")

	// 显示服务器状态
	fmt.Printf("📊 服务器运行状态: %v\n", server.IsRunning())
	fmt.Printf("📊 客户端连接数: %d\n", server.GetClientCount())

	// 通过SessionManager获取会话信息
	sessionManager := server.GetSessionManager()
	sessions := sessionManager.ListSessions()
	fmt.Printf("📋 当前会话数量: %d\n", len(sessions))

	for i, s := range sessions {
		fmt.Printf("  %d. %s (ID: %s, 状态: %s, 窗口数: %d)\n",
			i+1, s.Name, s.ID, s.Status, len(s.Windows))
	}

	// 等待一段时间
	fmt.Println("⏳ 等待3秒...")
	time.Sleep(3 * time.Second)

	// 断开会话
	if err := client.DetachSession(); err != nil {
		logger.Error("断开会话失败", zap.Error(err))
		return
	}

	fmt.Println("✅ 会话已断开")

	// 重新连接会话
	if err := client.AttachSession("demo-session"); err != nil {
		logger.Error("重新连接会话失败", zap.Error(err))
		return
	}

	fmt.Println("✅ 会话已重新连接")

	fmt.Println("🎉 基础示例完成!")
	fmt.Println("\n💡 提示: 你可以使用以下命令与终端交互:")
	fmt.Println("  clixgo terminal new-session <name>    # 创建新会话")
	fmt.Println("  clixgo terminal attach <name>         # 连接会话")
	fmt.Println("  clixgo terminal list-sessions         # 列出会话")
	fmt.Println("  clixgo terminal server status         # 查看服务器状态")
}
