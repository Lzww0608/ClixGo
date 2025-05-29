/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 用户界面功能示例程序
 */

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/terminal/ui"
)

func main() {
	// 初始化日志系统
	err := logger.InitLogger()
	if err != nil {
		log.Fatal("初始化日志系统失败:", err)
	}
	defer logger.Close()

	// 创建UI配置
	config := ui.DefaultUIConfig

	// 创建UI管理器
	uiManager, err := ui.NewUIManager(config)
	if err != nil {
		log.Fatal("创建UI管理器失败:", err)
	}

	// 创建演示面板
	setupDemoPanels(uiManager)

	// 启动UI
	if err := uiManager.Start(); err != nil {
		log.Fatal("启动UI失败:", err)
	}
}

func setupDemoPanels(uiManager *ui.UIManager) {
	// 创建主面板
	uiManager.CreatePanel("main", "Main Panel")

	// 向主面板写入欢迎信息
	welcomeText := `欢迎使用 ClixGo 终端UI系统！

这是一个功能强大的终端多路复用器，具有以下特性：

✨ 特性：
• 基于 tcell/tview 的高性能终端渲染
• 支持多面板分割显示
• 实时状态栏显示
• 完整的鼠标和键盘支持
• 智能布局管理

🎮 快捷键：
• Ctrl+N  - 创建新面板
• Ctrl+W  - 关闭当前面板
• Tab     - 切换面板
• Ctrl+H - 水平分割
• Ctrl+V - 垂直分割
• F1     - 显示帮助
• Ctrl+C - 退出程序

🖱️ 鼠标操作：
• 点击面板 - 切换焦点
• 滚轮     - 滚动内容

试试按 Ctrl+N 创建新面板！
`

	uiManager.WriteToPanel("main", welcomeText)

	// 启动演示内容更新
	go func() {
		time.Sleep(2 * time.Second)

		// 创建日志面板
		uiManager.CreatePanel("logs", "System Logs")

		// 模拟日志输出
		for i := 1; i <= 20; i++ {
			logMsg := fmt.Sprintf("[%s] 系统日志 #%d - 模拟日志消息\n",
				time.Now().Format("15:04:05"), i)
			uiManager.WriteToPanel("logs", logMsg)
			time.Sleep(500 * time.Millisecond)
		}

		// 创建监控面板
		time.Sleep(1 * time.Second)
		uiManager.CreatePanel("monitor", "System Monitor")

		// 模拟系统监控数据
		for i := 1; i <= 30; i++ {
			monitorData := fmt.Sprintf(`[%s] 系统监控数据 #%d
CPU使用率: %d%%
内存使用率: %d%%
网络流量: %d KB/s
磁盘I/O: %d MB/s

`,
				time.Now().Format("15:04:05"), i,
				30+i%40,   // CPU
				50+i%30,   // Memory
				100+i%200, // Network
				10+i%50,   // Disk
			)
			uiManager.WriteToPanel("monitor", monitorData)
			time.Sleep(1 * time.Second)
		}
	}()
}
