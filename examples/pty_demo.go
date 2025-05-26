package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

func main() {
	fmt.Println("=== ClixGo PTY Demo ===")
	fmt.Println("基于 creack/pty 库的真实PTY支持演示")
	fmt.Println()

	// 初始化logger
	err := logger.InitLogger()
	if err != nil {
		log.Fatalf("初始化logger失败: %v", err)
	}
	defer logger.Close()

	// 创建PTY管理器
	config := &terminal.TerminalConfig{
		PrefixKey:         "C-b",
		MouseEnabled:      true,
		StatusBar:         true,
		BufferSize:        2000,
		ScrollBack:        2000,
		ClixGoIntegration: true,
	}

	manager := terminal.NewCreackPTYManager(config)

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动交互式演示
	go func() {
		<-sigChan
		fmt.Println("\n正在清理资源...")
		// 清理所有PTY
		for _, id := range manager.ListCreackPTYs() {
			manager.DestroyCreackPTY(id)
		}
		os.Exit(0)
	}()

	// 演示菜单
	for {
		showMenu()
		choice := getUserInput("请选择操作: ")

		switch choice {
		case "1":
			demoCreatePTY(manager)
		case "2":
			demoListPTYs(manager)
		case "3":
			demoInteractWithPTY(manager)
		case "4":
			demoResizePTY(manager)
		case "5":
			demoDestroyPTY(manager)
		case "6":
			demoShellSession(manager)
		case "0":
			fmt.Println("退出演示程序")
			return
		default:
			fmt.Println("无效选择，请重试")
		}

		fmt.Println()
	}
}

func showMenu() {
	fmt.Println("=== PTY 操作菜单 ===")
	fmt.Println("1. 创建新的PTY")
	fmt.Println("2. 列出所有PTY")
	fmt.Println("3. 与PTY交互")
	fmt.Println("4. 调整PTY大小")
	fmt.Println("5. 销毁PTY")
	fmt.Println("6. 完整Shell会话演示")
	fmt.Println("0. 退出")
	fmt.Println()
}

func getUserInput(prompt string) string {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func demoCreatePTY(manager *terminal.CreackPTYManager) {
	fmt.Println("=== 创建新的PTY ===")

	id := getUserInput("输入PTY ID: ")
	if id == "" {
		id = fmt.Sprintf("pty_%d", time.Now().Unix())
	}

	command := getUserInput("输入要执行的命令 (留空使用默认shell): ")
	workingDir := getUserInput("输入工作目录 (留空使用当前目录): ")

	pty, err := manager.CreateCreackPTY(id, command, workingDir, 80, 24)
	if err != nil {
		fmt.Printf("创建PTY失败: %v\n", err)
		return
	}

	err = pty.Start()
	if err != nil {
		fmt.Printf("启动PTY失败: %v\n", err)
		manager.DestroyCreackPTY(id)
		return
	}

	fmt.Printf("PTY '%s' 创建成功，PID: %d\n", id, pty.GetPID())
}

func demoListPTYs(manager *terminal.CreackPTYManager) {
	fmt.Println("=== 当前PTY列表 ===")

	ptys := manager.ListCreackPTYs()
	if len(ptys) == 0 {
		fmt.Println("没有活动的PTY")
		return
	}

	for i, id := range ptys {
		pty, err := manager.GetCreackPTY(id)
		if err != nil {
			continue
		}

		width, height := pty.GetSize()
		fmt.Printf("%d. ID: %s, PID: %d, 大小: %dx%d, 运行中: %v\n",
			i+1, id, pty.GetPID(), width, height, pty.IsRunning())
	}
}

func demoInteractWithPTY(manager *terminal.CreackPTYManager) {
	fmt.Println("=== 与PTY交互 ===")

	ptys := manager.ListCreackPTYs()
	if len(ptys) == 0 {
		fmt.Println("没有可用的PTY")
		return
	}

	// 显示可用的PTY
	for i, id := range ptys {
		fmt.Printf("%d. %s\n", i+1, id)
	}

	choice := getUserInput("选择PTY编号: ")
	var selectedID string

	for i, id := range ptys {
		if choice == fmt.Sprintf("%d", i+1) {
			selectedID = id
			break
		}
	}

	if selectedID == "" {
		fmt.Println("无效选择")
		return
	}

	pty, err := manager.GetCreackPTY(selectedID)
	if err != nil {
		fmt.Printf("获取PTY失败: %v\n", err)
		return
	}

	command := getUserInput("输入要发送的命令: ")
	if command == "" {
		return
	}

	// 发送命令
	err = pty.Write([]byte(command + "\n"))
	if err != nil {
		fmt.Printf("发送命令失败: %v\n", err)
		return
	}

	fmt.Println("命令已发送，等待输出...")

	// 读取输出
	for i := 0; i < 5; i++ { // 最多读取5次
		data, err := pty.ReadWithTimeout(1 * time.Second)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				break
			}
			fmt.Printf("读取输出失败: %v\n", err)
			break
		}

		if len(data) > 0 {
			fmt.Printf("输出: %s", string(data))
		}
	}
}

func demoResizePTY(manager *terminal.CreackPTYManager) {
	fmt.Println("=== 调整PTY大小 ===")

	ptys := manager.ListCreackPTYs()
	if len(ptys) == 0 {
		fmt.Println("没有可用的PTY")
		return
	}

	// 显示可用的PTY
	for i, id := range ptys {
		pty, _ := manager.GetCreackPTY(id)
		width, height := pty.GetSize()
		fmt.Printf("%d. %s (当前大小: %dx%d)\n", i+1, id, width, height)
	}

	choice := getUserInput("选择PTY编号: ")
	var selectedID string

	for i, id := range ptys {
		if choice == fmt.Sprintf("%d", i+1) {
			selectedID = id
			break
		}
	}

	if selectedID == "" {
		fmt.Println("无效选择")
		return
	}

	pty, err := manager.GetCreackPTY(selectedID)
	if err != nil {
		fmt.Printf("获取PTY失败: %v\n", err)
		return
	}

	widthStr := getUserInput("输入新宽度: ")
	heightStr := getUserInput("输入新高度: ")

	var width, height int
	fmt.Sscanf(widthStr, "%d", &width)
	fmt.Sscanf(heightStr, "%d", &height)

	if width <= 0 || height <= 0 {
		fmt.Println("无效的尺寸")
		return
	}

	err = pty.Resize(width, height)
	if err != nil {
		fmt.Printf("调整大小失败: %v\n", err)
		return
	}

	fmt.Printf("PTY '%s' 大小已调整为 %dx%d\n", selectedID, width, height)
}

func demoDestroyPTY(manager *terminal.CreackPTYManager) {
	fmt.Println("=== 销毁PTY ===")

	ptys := manager.ListCreackPTYs()
	if len(ptys) == 0 {
		fmt.Println("没有可用的PTY")
		return
	}

	// 显示可用的PTY
	for i, id := range ptys {
		fmt.Printf("%d. %s\n", i+1, id)
	}

	choice := getUserInput("选择要销毁的PTY编号: ")
	var selectedID string

	for i, id := range ptys {
		if choice == fmt.Sprintf("%d", i+1) {
			selectedID = id
			break
		}
	}

	if selectedID == "" {
		fmt.Println("无效选择")
		return
	}

	err := manager.DestroyCreackPTY(selectedID)
	if err != nil {
		fmt.Printf("销毁PTY失败: %v\n", err)
		return
	}

	fmt.Printf("PTY '%s' 已销毁\n", selectedID)
}

func demoShellSession(manager *terminal.CreackPTYManager) {
	fmt.Println("=== 完整Shell会话演示 ===")
	fmt.Println("这将创建一个交互式shell会话")
	fmt.Println("输入 'exit' 或按 Ctrl+C 退出")
	fmt.Println()

	// 创建一个新的PTY用于shell会话
	sessionID := fmt.Sprintf("shell_session_%d", time.Now().Unix())

	pty, err := manager.CreateCreackPTY(sessionID, "", "", 80, 24)
	if err != nil {
		fmt.Printf("创建shell会话失败: %v\n", err)
		return
	}

	err = pty.Start()
	if err != nil {
		fmt.Printf("启动shell会话失败: %v\n", err)
		manager.DestroyCreackPTY(sessionID)
		return
	}

	fmt.Printf("Shell会话已启动 (PID: %d)\n", pty.GetPID())
	fmt.Println("开始交互模式...")

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	// 启动输出读取goroutine
	go func() {
		for {
			data, err := pty.ReadWithTimeout(100 * time.Millisecond)
			if err != nil {
				if !strings.Contains(err.Error(), "timeout") {
					fmt.Printf("\n读取错误: %v\n", err)
					break
				}
				continue
			}

			if len(data) > 0 {
				fmt.Print(string(data))
			}
		}
	}()

	// 主输入循环
	scanner := bufio.NewScanner(os.Stdin)
	for {
		select {
		case <-sigChan:
			fmt.Println("\n退出shell会话...")
			manager.DestroyCreackPTY(sessionID)
			return
		default:
			if scanner.Scan() {
				input := scanner.Text()
				if input == "exit" {
					fmt.Println("退出shell会话...")
					manager.DestroyCreackPTY(sessionID)
					return
				}

				err := pty.Write([]byte(input + "\n"))
				if err != nil {
					fmt.Printf("发送命令失败: %v\n", err)
					break
				}
			}
		}
	}
}
