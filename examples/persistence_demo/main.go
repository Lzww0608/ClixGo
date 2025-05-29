/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 持久化功能示例程序
 */

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

func main() {
	// 初始化日志系统
	err := logger.InitLogger()
	if err != nil {
		log.Fatal("初始化日志系统失败:", err)
	}
	defer logger.Close()

	fmt.Println("🔄 ClixGo 会话持久化演示程序")
	fmt.Println("================================")

	// 创建会话管理器
	config := terminal.DefaultConfig
	sessionManager := terminal.NewSessionManager(config)

	// 创建演示会话
	setupDemoSessions(sessionManager)

	// 交互式菜单
	scanner := bufio.NewScanner(os.Stdin)
	for {
		showMenu()
		fmt.Print("请选择操作: ")

		if !scanner.Scan() {
			break
		}

		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			listCurrentSessions(sessionManager)
		case "2":
			saveSession(sessionManager, scanner)
		case "3":
			loadSession(sessionManager, scanner)
		case "4":
			listSavedSessions(sessionManager)
		case "5":
			deleteSavedSession(sessionManager, scanner)
		case "6":
			showSessionDetails(sessionManager, scanner)
		case "7":
			demonstrateAutoSave(sessionManager)
		case "8":
			createNewSession(sessionManager, scanner)
		case "0":
			fmt.Println("👋 退出程序")
			return
		default:
			fmt.Println("❌ 无效选择，请重试")
		}

		fmt.Println()
		time.Sleep(500 * time.Millisecond) // 添加延迟防止死锁
	}
}

func showMenu() {
	fmt.Println("\n📋 操作菜单:")
	fmt.Println("1. 列出当前会话")
	fmt.Println("2. 保存会话")
	fmt.Println("3. 加载会话")
	fmt.Println("4. 列出已保存的会话")
	fmt.Println("5. 删除已保存的会话")
	fmt.Println("6. 查看会话详情")
	fmt.Println("7. 演示自动保存")
	fmt.Println("8. 创建新会话")
	fmt.Println("0. 退出")
	fmt.Println("--------------------------------")
}

func setupDemoSessions(sm *terminal.SessionManager) {
	fmt.Println("🚀 创建演示会话...")

	// 创建开发会话
	devSession, err := sm.CreateSession("development")
	if err != nil {
		fmt.Printf("❌ 创建开发会话失败: %v\n", err)
		return
	}

	// 为开发会话创建窗口
	_, err = sm.CreateWindow(devSession.ID, "editor")
	if err != nil {
		fmt.Printf("❌ 创建编辑器窗口失败: %v\n", err)
	}

	_, err = sm.CreateWindow(devSession.ID, "terminal")
	if err != nil {
		fmt.Printf("❌ 创建终端窗口失败: %v\n", err)
	}

	// 创建监控会话
	monitorSession, err := sm.CreateSession("monitoring")
	if err != nil {
		fmt.Printf("❌ 创建监控会话失败: %v\n", err)
		return
	}

	// 为监控会话创建窗口
	_, err = sm.CreateWindow(monitorSession.ID, "htop")
	if err != nil {
		fmt.Printf("❌ 创建htop窗口失败: %v\n", err)
	}

	_, err = sm.CreateWindow(monitorSession.ID, "logs")
	if err != nil {
		fmt.Printf("❌ 创建日志窗口失败: %v\n", err)
	}

	fmt.Println("✅ 演示会话创建完成")
	time.Sleep(1 * time.Second) // 添加延迟
}

func listCurrentSessions(sm *terminal.SessionManager) {
	fmt.Println("\n📝 当前活动会话:")
	sessions := sm.ListSessions()

	if len(sessions) == 0 {
		fmt.Println("  无活动会话")
		return
	}

	for i, session := range sessions {
		fmt.Printf("  %d. %s (ID: %s)\n", i+1, session.Name, session.ID[:8])
		fmt.Printf("     状态: %s, 窗口数: %d\n", session.Status, len(session.Windows))
		fmt.Printf("     创建时间: %s\n", session.CreatedAt.Format("2006-01-02 15:04:05"))
	}
}

func saveSession(sm *terminal.SessionManager, scanner *bufio.Scanner) {
	sessions := sm.ListSessions()
	if len(sessions) == 0 {
		fmt.Println("❌ 没有可保存的会话")
		return
	}

	fmt.Println("\n💾 选择要保存的会话:")
	for i, session := range sessions {
		fmt.Printf("  %d. %s\n", i+1, session.Name)
	}

	fmt.Print("请输入会话编号: ")
	if !scanner.Scan() {
		return
	}

	choice, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || choice < 1 || choice > len(sessions) {
		fmt.Println("❌ 无效选择")
		return
	}

	session := sessions[choice-1]
	fmt.Printf("正在保存会话 '%s'...\n", session.Name)

	if err := sm.SaveSession(session.ID, ""); err != nil {
		fmt.Printf("❌ 保存失败: %v\n", err)
	} else {
		fmt.Printf("✅ 会话 '%s' 保存成功\n", session.Name)
	}

	time.Sleep(1 * time.Second) // 添加延迟
}

func loadSession(sm *terminal.SessionManager, scanner *bufio.Scanner) {
	savedSessions, err := sm.ListSavedSessions()
	if err != nil {
		fmt.Printf("❌ 获取已保存会话失败: %v\n", err)
		return
	}

	if len(savedSessions) == 0 {
		fmt.Println("❌ 没有已保存的会话")
		return
	}

	fmt.Println("\n📂 选择要加载的会话:")
	// 去重会话名称
	uniqueSessions := make(map[string]bool)
	var sessionList []string
	for _, sessionName := range savedSessions {
		if !uniqueSessions[sessionName] {
			uniqueSessions[sessionName] = true
			sessionList = append(sessionList, sessionName)
		}
	}

	for i, sessionName := range sessionList {
		fmt.Printf("  %d. %s\n", i+1, sessionName)
	}

	fmt.Print("请输入会话编号: ")
	if !scanner.Scan() {
		return
	}

	choice, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || choice < 1 || choice > len(sessionList) {
		fmt.Println("❌ 无效选择")
		return
	}

	sessionName := sessionList[choice-1]
	fmt.Printf("正在加载会话 '%s'...\n", sessionName)

	session, err := sm.LoadSessionByName(sessionName)
	if err != nil {
		fmt.Printf("❌ 加载失败: %v\n", err)
	} else {
		fmt.Printf("✅ 会话 '%s' 加载成功 (ID: %s)\n", session.Name, session.ID[:8])
		fmt.Printf("   窗口数: %d\n", len(session.Windows))
	}

	time.Sleep(1 * time.Second) // 添加延迟
}

func listSavedSessions(sm *terminal.SessionManager) {
	fmt.Println("\n💾 已保存的会话:")

	savedSessions, err := sm.ListSavedSessions()
	if err != nil {
		fmt.Printf("❌ 获取已保存会话失败: %v\n", err)
		return
	}

	if len(savedSessions) == 0 {
		fmt.Println("  无已保存的会话")
		return
	}

	// 统计每个会话的快照数量
	sessionCount := make(map[string]int)
	for _, sessionName := range savedSessions {
		sessionCount[sessionName]++
	}

	i := 1
	for sessionName, count := range sessionCount {
		fmt.Printf("  %d. %s (%d个快照)\n", i, sessionName, count)
		i++
	}
}

func deleteSavedSession(sm *terminal.SessionManager, scanner *bufio.Scanner) {
	savedSessions, err := sm.ListSavedSessions()
	if err != nil {
		fmt.Printf("❌ 获取已保存会话失败: %v\n", err)
		return
	}

	if len(savedSessions) == 0 {
		fmt.Println("❌ 没有已保存的会话")
		return
	}

	// 去重会话名称
	uniqueSessions := make(map[string]bool)
	var sessionList []string
	for _, sessionName := range savedSessions {
		if !uniqueSessions[sessionName] {
			uniqueSessions[sessionName] = true
			sessionList = append(sessionList, sessionName)
		}
	}

	fmt.Println("\n🗑️ 选择要删除的会话:")
	for i, sessionName := range sessionList {
		fmt.Printf("  %d. %s\n", i+1, sessionName)
	}

	fmt.Print("请输入会话编号: ")
	if !scanner.Scan() {
		return
	}

	choice, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || choice < 1 || choice > len(sessionList) {
		fmt.Println("❌ 无效选择")
		return
	}

	sessionName := sessionList[choice-1]
	fmt.Printf("确认删除会话 '%s' 的所有快照? (y/N): ", sessionName)

	if !scanner.Scan() {
		return
	}

	confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if confirm != "y" && confirm != "yes" {
		fmt.Println("❌ 取消删除")
		return
	}

	if err := sm.DeleteSavedSession(sessionName); err != nil {
		fmt.Printf("❌ 删除失败: %v\n", err)
	} else {
		fmt.Printf("✅ 会话 '%s' 的快照已删除\n", sessionName)
	}

	time.Sleep(1 * time.Second) // 添加延迟
}

func showSessionDetails(sm *terminal.SessionManager, scanner *bufio.Scanner) {
	sessions := sm.ListSessions()
	if len(sessions) == 0 {
		fmt.Println("❌ 没有活动会话")
		return
	}

	fmt.Println("\n🔍 选择要查看详情的会话:")
	for i, session := range sessions {
		fmt.Printf("  %d. %s\n", i+1, session.Name)
	}

	fmt.Print("请输入会话编号: ")
	if !scanner.Scan() {
		return
	}

	choice, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || choice < 1 || choice > len(sessions) {
		fmt.Println("❌ 无效选择")
		return
	}

	session := sessions[choice-1]

	fmt.Printf("\n📊 会话详情: %s\n", session.Name)
	fmt.Printf("ID: %s\n", session.ID)
	fmt.Printf("状态: %s\n", session.Status)
	fmt.Printf("创建时间: %s\n", session.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("最后活动: %s\n", session.LastActive.Format("2006-01-02 15:04:05"))
	fmt.Printf("活动窗口: %d\n", session.ActiveWindow)

	fmt.Printf("\n窗口列表 (%d个):\n", len(session.Windows))
	for i, window := range session.Windows {
		fmt.Printf("  %d. %s (ID: %s)\n", i+1, window.Name, window.ID[:8])
		fmt.Printf("     面板数: %d, 布局: %s\n", len(window.Panes), window.Layout)
		fmt.Printf("     创建时间: %s\n", window.CreatedAt.Format("2006-01-02 15:04:05"))
	}
}

func demonstrateAutoSave(sm *terminal.SessionManager) {
	sessions := sm.ListSessions()
	if len(sessions) == 0 {
		fmt.Println("❌ 没有可演示自动保存的会话")
		return
	}

	session := sessions[0]
	fmt.Printf("\n⏰ 演示自动保存功能 (会话: %s)\n", session.Name)
	fmt.Println("将每5秒自动保存一次，持续20秒...")

	// 启动自动保存（在后台运行）
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		count := 0
		for {
			select {
			case <-ticker.C:
				count++
				fmt.Printf("🔄 第%d次自动保存...\n", count)
				if err := sm.SaveSession(session.ID, ""); err != nil {
					fmt.Printf("❌ 自动保存失败: %v\n", err)
				} else {
					fmt.Printf("✅ 自动保存成功\n")
				}

				if count >= 4 { // 保存4次后停止
					done <- true
					return
				}
			case <-done:
				return
			}
		}
	}()

	// 等待演示完成
	<-done
	fmt.Println("✅ 自动保存演示完成")
	time.Sleep(1 * time.Second) // 添加延迟
}

func createNewSession(sm *terminal.SessionManager, scanner *bufio.Scanner) {
	fmt.Print("\n📝 请输入新会话名称: ")
	if !scanner.Scan() {
		return
	}

	sessionName := strings.TrimSpace(scanner.Text())
	if sessionName == "" {
		fmt.Println("❌ 会话名称不能为空")
		return
	}

	fmt.Printf("正在创建会话 '%s'...\n", sessionName)

	session, err := sm.CreateSession(sessionName)
	if err != nil {
		fmt.Printf("❌ 创建失败: %v\n", err)
		return
	}

	// 创建一个默认窗口
	_, err = sm.CreateWindow(session.ID, "main")
	if err != nil {
		fmt.Printf("❌ 创建默认窗口失败: %v\n", err)
	}

	fmt.Printf("✅ 会话 '%s' 创建成功 (ID: %s)\n", session.Name, session.ID[:8])
	time.Sleep(1 * time.Second) // 添加延迟
}
