/*
* @Author: Lzww0608
* @Description: ClixGo Terminal 简单使用示例 - 基于当前实际架构
 */

package main

import (
	"fmt"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

func main() {
	fmt.Println("🚀 ClixGo Terminal 简单示例")
	fmt.Println("============================")

	// 初始化日志系统
	if err := logger.InitLogger(); err != nil {
		fmt.Printf("初始化日志系统失败: %v\n", err)
		return
	}
	defer logger.Close()

	// 使用默认配置
	config := terminal.DefaultConfig
	fmt.Printf("✅ 使用配置: 前缀键=%s, 缓冲区=%d\n", config.PrefixKey, config.BufferSize)

	// 创建会话管理器
	sessionManager := terminal.NewSessionManager(config)
	fmt.Println("✅ 会话管理器已创建")

	// 创建测试会话
	session, err := sessionManager.CreateSession("demo-session")
	if err != nil {
		fmt.Printf("❌ 创建会话失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 会话已创建: %s (ID: %s)\n", session.Name, session.ID[:8])

	// 创建窗口
	window, err := sessionManager.CreateWindow(session.ID, "main-window")
	if err != nil {
		fmt.Printf("❌ 创建窗口失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 窗口已创建: %s (索引: %d)\n", window.Name, window.Index)

	// 分割面板
	pane, err := sessionManager.SplitPane(session.ID, 0, "vertical")
	if err != nil {
		fmt.Printf("❌ 分割面板失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 面板已分割: ID=%s\n", pane.ID[:8])

	// 再次分割面板
	pane2, err := sessionManager.SplitPane(session.ID, 0, "horizontal")
	if err != nil {
		fmt.Printf("❌ 分割面板失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 面板已分割: ID=%s\n", pane2.ID[:8])

	// 列出所有会话
	sessions := sessionManager.ListSessions()
	fmt.Printf("\n📋 当前会话列表 (共%d个):\n", len(sessions))
	for i, s := range sessions {
		fmt.Printf("  %d. %s (ID: %s, 状态: %s, 窗口数: %d)\n",
			i+1, s.Name, s.ID[:8], s.Status, len(s.Windows))

		// 显示窗口信息
		for j, w := range s.Windows {
			fmt.Printf("     窗口 %d: %s (面板数: %d)\n", j+1, w.Name, len(w.Panes))
		}
	}

	// 测试UI渲染器
	fmt.Println("\n🎨 测试UI渲染:")
	ui := terminal.NewUIRenderer(80, 24, nil)
	if len(session.Windows) > 0 {
		output := ui.RenderWindow(session.Windows[0])
		lines := len(output)
		fmt.Printf("✅ UI渲染成功 (输出行数: %d)\n", lines)

		// 显示前几行作为预览
		if lines > 0 {
			fmt.Println("   预览:")
			outputStr := string(output)
			if len(outputStr) > 200 {
				fmt.Printf("   %s...\n", outputStr[:200])
			} else {
				fmt.Printf("   %s\n", outputStr)
			}
		}
	}

	// 测试PTY管理器 (简化版)
	fmt.Println("\n🔧 测试PTY管理器:")
	ptyManager := terminal.NewSimplePTYManager(config)
	fmt.Println("✅ PTY管理器已创建")

	// 创建简单PTY
	pty, err := ptyManager.CreateSimplePTY("test-pty", "echo 'Hello from PTY'", "/tmp", 80, 24)
	if err != nil {
		fmt.Printf("❌ 创建PTY失败: %v\n", err)
	} else {
		fmt.Printf("✅ PTY已创建: ID=%s\n", pty.ID)

		// 启动PTY
		if err := pty.Start(); err != nil {
			fmt.Printf("❌ 启动PTY失败: %v\n", err)
		} else {
			fmt.Printf("✅ PTY已启动 (PID: %d)\n", pty.GetPID())

			// 等待一下让命令执行
			time.Sleep(100 * time.Millisecond)

			// 尝试读取输出
			if data, err := pty.Read(); err == nil && len(data) > 0 {
				fmt.Printf("✅ PTY输出: %s\n", string(data))
			}
		}
	}

	// 清理资源
	fmt.Println("\n🧹 清理资源:")
	if err := sessionManager.KillSession(session.ID); err != nil {
		fmt.Printf("⚠️  清理会话失败: %v\n", err)
	} else {
		fmt.Println("✅ 会话已清理")
	}

	fmt.Println("\n🎉 示例完成!")
	fmt.Println("\n💡 下一步:")
	fmt.Println("  1. 查看配置文件: examples/terminal/config.yaml")
	fmt.Println("  2. 运行CLI命令: ./clixgo terminal session create test")
	fmt.Println("  3. 等待Phase 1.3实现完整的PTY功能")
}
