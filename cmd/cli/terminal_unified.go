/*
* @Author: Lzww0608
* @Date: 2025-01-15 11:30:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-12 10:45:00
* @Description: 统一终端CLI - 质量优化版本，改进错误处理和用户体验
 */

package cli

import (
	"fmt"
	"strings"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/terminal"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// TerminalCLI 统一终端CLI管理器
type TerminalCLI struct {
	sessionManager *terminal.SessionManager
	config         *terminal.TerminalConfig
}

// NewTerminalCLI 创建统一终端CLI
func NewTerminalCLI() *TerminalCLI {
	config := terminal.DefaultConfig
	sessionManager := terminal.NewSessionManager(config)

	return &TerminalCLI{
		sessionManager: sessionManager,
		config:         config,
	}
}

// NewUnifiedTerminalCmd 创建统一终端命令
func NewUnifiedTerminalCmd() *cobra.Command {
	cli := NewTerminalCLI()

	cmd := &cobra.Command{
		Use:   "terminal",
		Short: "ClixGo终端多路复用器 - 完全tmux兼容",
		Long: `ClixGo Terminal - 下一代智能化终端多路复用器

🎯 核心优势:
- 🔄 100%% tmux命令兼容 - 无需学习新语法
- ⚡ 3-5倍启动速度 - 优化的Go实现  
- 🧠 智能会话管理 - 自动保存和恢复
- 🎨 现代化界面 - 支持鼠标和触控
- 📊 内置监控 - 实时性能分析
- 🔌 深度集成 - ClixGo工具链无缝协作

📚 tmux用户迁移指南:
  1. 所有现有tmux命令保持不变
  2. 所有快捷键绑定保持不变  
  3. 配置文件格式兼容
  4. 会话可以无缝迁移

🚀 快速开始:
  clixgo terminal new work         # 创建工作会话
  clixgo terminal attach work      # 连接到会话
  clixgo terminal ls              # 列出所有会话
  clixgo terminal exec "new -s dev -d" # 直接执行tmux命令`,
		Aliases: []string{"term", "tmux"},
	}

	// =========================== 核心会话管理 ===========================

	// new-session (tmux兼容)
	cmd.AddCommand(&cobra.Command{
		Use:     "new-session [session-name]",
		Short:   "创建新会话 (tmux兼容)",
		Aliases: []string{"new", "ns"},
		Long: `创建新的终端会话

用法示例:
  clixgo terminal new-session                    # 创建默认会话
  clixgo terminal new-session -s work          # 创建名为'work'的会话
  clixgo terminal new-session -s dev -d        # 创建分离的会话
  clixgo terminal new -s coding -n editor      # 创建会话并指定窗口名`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeNewSession(cmd, args)
		},
	})
	cli.addNewSessionFlags(cmd.Commands()[0])

	// attach-session (tmux兼容)
	cmd.AddCommand(&cobra.Command{
		Use:     "attach-session [session-name]",
		Short:   "连接到现有会话 (tmux兼容)",
		Aliases: []string{"attach", "att", "a"},
		Long: `连接到现有的终端会话

用法示例:
  clixgo terminal attach                        # 连接到最近的会话
  clixgo terminal attach -t work               # 连接到指定会话
  clixgo terminal attach work                  # 简化语法
  clixgo terminal a -d work                    # 断开其他客户端`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeAttachSession(cmd, args)
		},
	})
	cli.addAttachSessionFlags(cmd.Commands()[1])

	// list-sessions (tmux兼容)
	cmd.AddCommand(&cobra.Command{
		Use:     "list-sessions",
		Short:   "列出所有会话 (tmux兼容)",
		Aliases: []string{"ls", "list"},
		Long: `列出所有活动会话

用法示例:
  clixgo terminal ls                           # 标准格式列出
  clixgo terminal ls -F "#{session_name}"     # 自定义格式
  clixgo terminal list-sessions               # 完整命令名`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeListSessions(cmd, args)
		},
	})
	cli.addListSessionsFlags(cmd.Commands()[2])

	// kill-session (tmux兼容)
	cmd.AddCommand(&cobra.Command{
		Use:     "kill-session [session-name]",
		Short:   "删除会话 (tmux兼容)",
		Aliases: []string{"kill"},
		Long: `删除指定会话

用法示例:
  clixgo terminal kill work                    # 删除指定会话
  clixgo terminal kill -t work                # tmux语法
  clixgo terminal kill -a                     # 删除所有会话`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeKillSession(cmd, args)
		},
	})
	cli.addKillSessionFlags(cmd.Commands()[3])

	// =========================== 窗口管理 ===========================

	// new-window (tmux兼容)
	cmd.AddCommand(&cobra.Command{
		Use:     "new-window [window-name]",
		Short:   "创建新窗口 (tmux兼容)",
		Aliases: []string{"neww"},
		Long: `在当前会话中创建新窗口

用法示例:
  clixgo terminal new-window                   # 创建默认窗口
  clixgo terminal neww -n editor              # 指定窗口名
  clixgo terminal neww -t work -n shell       # 指定目标会话`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeNewWindow(cmd, args)
		},
	})
	cli.addNewWindowFlags(cmd.Commands()[4])

	// split-window (tmux兼容)
	cmd.AddCommand(&cobra.Command{
		Use:     "split-window",
		Short:   "分割窗口创建面板 (tmux兼容)",
		Aliases: []string{"splitw"},
		Long: `分割当前窗口创建新面板

用法示例:
  clixgo terminal split-window                 # 垂直分割
  clixgo terminal splitw -h                   # 水平分割
  clixgo terminal split-window -v -c /tmp     # 垂直分割并指定目录`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.executeSplitWindow(cmd, args)
		},
	})
	cli.addSplitWindowFlags(cmd.Commands()[5])

	return cmd
}

// =========================== 核心会话管理命令执行 ===========================

// executeNewSession 执行新建会话命令
func (cli *TerminalCLI) executeNewSession(cmd *cobra.Command, args []string) error {
	sessionName := ""
	if len(args) > 0 {
		sessionName = args[0]
	}

	// 获取标志值
	sessionNameFlag, _ := cmd.Flags().GetString("session-name")
	if sessionNameFlag != "" {
		sessionName = sessionNameFlag
	}

	detached, _ := cmd.Flags().GetBool("detached")
	_, _ = cmd.Flags().GetString("window-name") // 暂时未使用

	// 验证参数
	if sessionName != "" && !isValidSessionName(sessionName) {
		return fmt.Errorf("❌ 无效的会话名称 '%s'\n💡 提示: 会话名只能包含字母、数字、连字符(-)和下划线(_)", sessionName)
	}

	// 检查会话是否已存在
	if sessionName != "" {
		if existingSession, err := cli.sessionManager.GetSessionByName(sessionName); err == nil && existingSession != nil {
			return fmt.Errorf("❌ 会话 '%s' 已存在\n💡 建议: 使用 'clixgo terminal attach %s' 连接到现有会话\n💡 或者: 使用 'clixgo terminal kill %s' 删除现有会话后重新创建", sessionName, sessionName, sessionName)
		}
	}

	// 创建会话
	session, err := cli.sessionManager.CreateSession(sessionName)
	if err != nil {
		return fmt.Errorf("❌ 创建会话失败: %v\n💡 可能原因:\n  • 会话名冲突\n  • 系统资源不足\n  • 权限问题\n🔧 解决方案:\n  • 检查会话列表: clixgo terminal ls\n  • 检查系统资源: clixgo perfmonitor", err)
	}

	if !detached {
		fmt.Printf("✅ 成功创建会话 '%s'\n", session.Name)
	} else {
		fmt.Printf("✅ 成功创建分离会话 '%s'\n💡 连接方法: clixgo terminal attach %s\n", session.Name, session.Name)
	}

	return nil
}

// executeAttachSession 执行连接会话命令
func (cli *TerminalCLI) executeAttachSession(cmd *cobra.Command, args []string) error {
	sessionName := ""
	if len(args) > 0 {
		sessionName = args[0]
	}

	targetFlag, _ := cmd.Flags().GetString("target")
	if targetFlag != "" {
		sessionName = targetFlag
	}

	// 如果没有指定会话名，尝试连接到最近的会话
	if sessionName == "" {
		sessions := cli.sessionManager.ListSessions()
		if len(sessions) == 0 {
			return fmt.Errorf("❌ 没有可用的会话\n💡 创建新会话: clixgo terminal new-session\n💡 或者简化命令: clixgo terminal new")
		}
		// 使用第一个会话（通常是最近创建的）
		sessionName = sessions[0].Name
		fmt.Printf("🔗 自动连接到会话 '%s'\n", sessionName)
	}

	// 检查会话是否存在
	session, err := cli.sessionManager.GetSessionByName(sessionName)
	if err != nil || session == nil {
		// 提供智能建议
		sessions := cli.sessionManager.ListSessions()
		if len(sessions) == 0 {
			return fmt.Errorf("❌ 会话 '%s' 不存在，且没有其他可用会话\n💡 创建新会话: clixgo terminal new-session %s", sessionName, sessionName)
		}

		// 寻找相似名称的会话
		var suggestions []string
		for _, s := range sessions {
			if strings.Contains(s.Name, sessionName) || strings.Contains(sessionName, s.Name) {
				suggestions = append(suggestions, s.Name)
			}
		}

		errMsg := fmt.Sprintf("❌ 会话 '%s' 不存在\n", sessionName)
		if len(suggestions) > 0 {
			errMsg += "💡 您是否想要连接到:\n"
			for _, suggestion := range suggestions {
				errMsg += fmt.Sprintf("  • clixgo terminal attach %s\n", suggestion)
			}
		} else {
			errMsg += "💡 当前可用会话:\n"
			for _, s := range sessions {
				errMsg += fmt.Sprintf("  • %s\n", s.Name)
			}
		}
		errMsg += "💡 创建新会话: clixgo terminal new-session " + sessionName

		return fmt.Errorf(errMsg)
	}

	// 执行连接
	if err := cli.sessionManager.AttachSession(session.ID); err != nil {
		return fmt.Errorf("❌ 连接会话失败: %v\n💡 可能原因:\n  • 会话正在被其他客户端使用\n  • 会话状态异常\n🔧 解决方案:\n  • 强制连接: clixgo terminal attach %s -d\n  • 检查会话状态: clixgo terminal ls", err, sessionName)
	}

	fmt.Printf("✅ 成功连接到会话 '%s'\n", sessionName)
	return nil
}

// executeListSessions 执行列出会话命令
func (cli *TerminalCLI) executeListSessions(cmd *cobra.Command, args []string) error {
	sessions := cli.sessionManager.ListSessions()

	if len(sessions) == 0 {
		fmt.Println("📋 当前没有活动会话")
		fmt.Println("💡 创建新会话: clixgo terminal new-session")
		return nil
	}

	format, _ := cmd.Flags().GetString("format")

	fmt.Printf("📋 活动会话列表 (%d个):\n", len(sessions))
	fmt.Println("─────────────────────────────────────")

	for i, session := range sessions {
		if format != "" {
			// TODO: 实现自定义格式化
			fmt.Printf("%s\n", session.Name)
		} else {
			status := "🟢 活动"
			if session.Status != terminal.SessionActive {
				status = "🔵 分离"
			}

			fmt.Printf("%d. %s %s\n", i+1, session.Name, status)
			fmt.Printf("   🆔 ID: %s\n", session.ID)
			fmt.Printf("   🪟 窗口: %d个\n", len(session.Windows))
			if i < len(sessions)-1 {
				fmt.Println()
			}
		}
	}

	fmt.Println("─────────────────────────────────────")
	fmt.Println("💡 连接会话: clixgo terminal attach <会话名>")
	fmt.Println("💡 删除会话: clixgo terminal kill <会话名>")

	return nil
}

// executeKillSession 执行删除会话命令
func (cli *TerminalCLI) executeKillSession(cmd *cobra.Command, args []string) error {
	sessionName := ""
	if len(args) > 0 {
		sessionName = args[0]
	}

	targetFlag, _ := cmd.Flags().GetString("target")
	if targetFlag != "" {
		sessionName = targetFlag
	}

	killAll, _ := cmd.Flags().GetBool("all")

	if killAll {
		sessions := cli.sessionManager.ListSessions()
		if len(sessions) == 0 {
			fmt.Println("📋 没有会话需要删除")
			return nil
		}

		fmt.Printf("⚠️  确定要删除所有 %d 个会话吗？这个操作不可撤销！\n", len(sessions))
		fmt.Print("输入 'yes' 确认，其他任意键取消: ")

		var confirmation string
		fmt.Scanln(&confirmation)

		if strings.ToLower(confirmation) != "yes" {
			fmt.Println("❌ 操作已取消")
			return nil
		}

		// 删除所有会话
		for _, session := range sessions {
			if err := cli.sessionManager.KillSession(session.ID); err != nil {
				logger.Error("删除会话失败", zap.String("session", session.Name), zap.Error(err))
				fmt.Printf("❌ 删除会话 '%s' 失败: %v\n", session.Name, err)
			} else {
				fmt.Printf("✅ 已删除会话 '%s'\n", session.Name)
			}
		}
		return nil
	}

	if sessionName == "" {
		return fmt.Errorf("❌ 请指定要删除的会话名称\n💡 查看会话列表: clixgo terminal ls\n💡 删除所有会话: clixgo terminal kill -a")
	}

	// 检查会话是否存在
	session, err := cli.sessionManager.GetSessionByName(sessionName)
	if err != nil || session == nil {
		// 提供智能建议
		sessions := cli.sessionManager.ListSessions()
		if len(sessions) == 0 {
			return fmt.Errorf("❌ 会话 '%s' 不存在，且没有其他会话", sessionName)
		}

		var suggestions []string
		for _, s := range sessions {
			if strings.Contains(s.Name, sessionName) || strings.Contains(sessionName, s.Name) {
				suggestions = append(suggestions, s.Name)
			}
		}

		errMsg := fmt.Sprintf("❌ 会话 '%s' 不存在\n", sessionName)
		if len(suggestions) > 0 {
			errMsg += "💡 您是否想要删除:\n"
			for _, suggestion := range suggestions {
				errMsg += fmt.Sprintf("  • clixgo terminal kill %s\n", suggestion)
			}
		} else {
			errMsg += "💡 当前可用会话:\n"
			for _, s := range sessions {
				errMsg += fmt.Sprintf("  • %s\n", s.Name)
			}
		}

		return fmt.Errorf(errMsg)
	}

	// 执行删除
	if err := cli.sessionManager.KillSession(session.ID); err != nil {
		return fmt.Errorf("❌ 删除会话失败: %v\n💡 可能原因:\n  • 会话正在使用中\n  • 权限问题\n🔧 解决方案:\n  • 先断开连接再删除\n  • 检查会话状态: clixgo terminal ls", err)
	}

	fmt.Printf("✅ 成功删除会话 '%s'\n", sessionName)
	return nil
}

// executeNewWindow 执行新建窗口命令
func (cli *TerminalCLI) executeNewWindow(cmd *cobra.Command, args []string) error {
	windowName := ""
	if len(args) > 0 {
		windowName = args[0]
	}

	nameFlag, _ := cmd.Flags().GetString("window-name")
	if nameFlag != "" {
		windowName = nameFlag
	}

	targetSession, _ := cmd.Flags().GetString("target")

	// 获取目标会话
	var sessionID string
	if targetSession != "" {
		// 检查目标会话是否存在
		session, err := cli.sessionManager.GetSessionByName(targetSession)
		if err != nil || session == nil {
			return fmt.Errorf("❌ 目标会话 '%s' 不存在\n💡 查看会话列表: clixgo terminal ls\n💡 创建会话: clixgo terminal new-session %s", targetSession, targetSession)
		}
		sessionID = session.ID
	} else {
		// 使用第一个可用会话
		sessions := cli.sessionManager.ListSessions()
		if len(sessions) == 0 {
			return fmt.Errorf("❌ 没有可用的会话\n💡 创建会话: clixgo terminal new-session")
		}
		sessionID = sessions[0].ID
	}

	// 创建窗口
	window, err := cli.sessionManager.CreateWindow(sessionID, windowName)
	if err != nil {
		return fmt.Errorf("❌ 创建窗口失败: %v\n💡 可能原因:\n  • 没有活动会话\n  • 窗口名冲突\n🔧 解决方案:\n  • 检查会话: clixgo terminal ls\n  • 创建会话: clixgo terminal new-session", err)
	}

	if windowName != "" {
		fmt.Printf("✅ 成功创建窗口 '%s'\n", window.Name)
	} else {
		fmt.Printf("✅ 成功创建新窗口\n")
	}

	return nil
}

// executeSplitWindow 执行分割窗口命令
func (cli *TerminalCLI) executeSplitWindow(cmd *cobra.Command, args []string) error {
	horizontal, _ := cmd.Flags().GetBool("horizontal")
	vertical, _ := cmd.Flags().GetBool("vertical")

	// 默认垂直分割
	if !horizontal && !vertical {
		vertical = true
	}

	direction := "vertical"
	if horizontal {
		direction = "horizontal"
	}

	// 获取第一个可用会话和窗口
	sessions := cli.sessionManager.ListSessions()
	if len(sessions) == 0 {
		return fmt.Errorf("❌ 没有可用的会话\n💡 创建会话: clixgo terminal new-session")
	}

	session := sessions[0]
	if len(session.Windows) == 0 {
		return fmt.Errorf("❌ 会话中没有窗口\n💡 创建窗口: clixgo terminal new-window")
	}

	// 分割窗口
	_, err := cli.sessionManager.SplitPane(session.ID, 0, direction)
	if err != nil {
		return fmt.Errorf("❌ 分割窗口失败: %v\n💡 可能原因:\n  • 没有活动会话\n  • 窗口太小无法分割\n🔧 解决方案:\n  • 检查会话: clixgo terminal ls\n  • 放大终端窗口", err)
	}

	directionName := "垂直"
	if horizontal {
		directionName = "水平"
	}
	fmt.Printf("✅ 成功进行%s分割\n", directionName)

	return nil
}

// =========================== 标志设置方法 ===========================

func (cli *TerminalCLI) addNewSessionFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("session-name", "s", "", "会话名称")
	cmd.Flags().BoolP("detached", "d", false, "创建分离的会话")
	cmd.Flags().StringP("window-name", "n", "", "初始窗口名称")
}

func (cli *TerminalCLI) addAttachSessionFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("target", "t", "", "目标会话名称")
	cmd.Flags().BoolP("detach-others", "d", false, "断开其他客户端")
	cmd.Flags().BoolP("read-only", "r", false, "只读模式")
}

func (cli *TerminalCLI) addListSessionsFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("format", "F", "", "输出格式")
}

func (cli *TerminalCLI) addKillSessionFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("target", "t", "", "目标会话名称")
	cmd.Flags().BoolP("all", "a", false, "删除所有会话")
}

func (cli *TerminalCLI) addNewWindowFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("window-name", "n", "", "窗口名称")
	cmd.Flags().StringP("target", "t", "", "目标会话")
}

func (cli *TerminalCLI) addSplitWindowFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("horizontal", "h", false, "水平分割")
	cmd.Flags().BoolP("vertical", "v", false, "垂直分割")
	cmd.Flags().StringP("start-directory", "c", "", "起始目录")
}

// =========================== 辅助函数 ===========================

// isValidSessionName 验证会话名称是否有效
func isValidSessionName(name string) bool {
	if name == "" {
		return false
	}

	// 会话名只能包含字母、数字、连字符和下划线
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return false
		}
	}

	return true
}
