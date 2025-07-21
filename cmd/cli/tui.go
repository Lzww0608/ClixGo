/*
* @Author: Lzww0608
* @Date: 2025-06-18 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-18 10:00:00
* @Description: TUI命令入口 - 集成终端复用TUI界面
 */

package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/terminal"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// NewTUICmd 创建TUI命令
func NewTUICmd() *cobra.Command {
	var (
		sessionName string
		prefixKey   string
		configFile  string
	)

	tuiCmd := &cobra.Command{
		Use:   "tui",
		Short: "启动终端复用TUI界面",
		Long: `ClixGo TUI - 高性能终端多路复用器图形界面

🎯 特性：
- 🖥️  现代化TUI界面 - 基于tcell/tview的专业终端界面
- ⚡ 真实PTY支持 - 完整的终端仿真和ANSI颜色支持
- 🎮 tmux风格操作 - 熟悉的快捷键和会话管理
- 📊 实时性能监控 - 内置的性能统计和状态显示
- 🔄 会话恢复 - 自动保存和恢复会话状态

🎮 快捷键：
  Prefix Key (默认 Ctrl+b) 后按：
    c     - 创建新终端面板
    "     - 水平分割面板  
    %     - 垂直分割面板
    o     - 切换到下一个面板
    x     - 关闭当前面板
    d     - 分离会话
    ?     - 显示帮助

📋 使用示例：
  clixgo tui                              # 启动默认会话
  clixgo tui --session work               # 启动名为'work'的会话
  clixgo tui --prefix C-a                 # 使用Ctrl+a作为prefix键
  clixgo tui --config ~/.clixgo-tui.yaml # 使用指定配置文件`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 初始化logger
			if err := logger.InitLogger(); err != nil {
				return fmt.Errorf("初始化logger失败: %v", err)
			}

			logger.Info("启动ClixGo TUI界面",
				zap.String("session", sessionName),
				zap.String("prefix_key", prefixKey))

			// 创建TUI配置
			config := createTUIConfig(prefixKey, configFile)

			// 创建TUI管理器
			tui, err := terminal.NewTerminalTUI(config)
			if err != nil {
				return fmt.Errorf("创建TUI管理器失败: %v", err)
			}

			// 设置信号处理
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGWINCH)

			// 启动信号处理协程
			go func() {
				for sig := range sigChan {
					switch sig {
					case syscall.SIGINT, syscall.SIGTERM:
						logger.Info("收到退出信号，正在关闭TUI...")
						tui.Stop()
						return
					case syscall.SIGWINCH:
						// 窗口大小变化信号
						logger.Debug("窗口大小发生变化")
						// TODO: 实现窗口大小调整
					}
				}
			}()

			// 显示启动信息
			fmt.Printf("🚀 ClixGo TUI 正在启动...\n")
			fmt.Printf("   会话名称: %s\n", sessionName)
			fmt.Printf("   Prefix键: %s\n", prefixKey)
			fmt.Printf("   按 %s + ? 查看帮助\n", prefixKey)
			fmt.Printf("   按 %s + d 分离会话\n\n", prefixKey)

			// 启动TUI（阻塞直到退出）
			if err := tui.Start(); err != nil {
				return fmt.Errorf("TUI运行失败: %v", err)
			}

			fmt.Println("👋 ClixGo TUI 已退出")
			return nil
		},
	}

	// 命令选项
	tuiCmd.Flags().StringVarP(&sessionName, "session", "s", "main",
		"会话名称")
	tuiCmd.Flags().StringVarP(&prefixKey, "prefix", "p", "C-b",
		"Prefix键 (C-a, C-b, C-x)")
	tuiCmd.Flags().StringVarP(&configFile, "config", "c", "",
		"配置文件路径")

	return tuiCmd
}

// createTUIConfig 创建TUI配置
func createTUIConfig(prefixKey, configFile string) *terminal.TerminalTUIConfig {
	// 基础配置
	config := terminal.DefaultTerminalTUIConfig()
	config.PrefixKey = prefixKey

	// 如果指定了配置文件，加载配置
	if configFile != "" {
		if err := loadConfigFile(config, configFile); err != nil {
			logger.Warn("加载配置文件失败，使用默认配置",
				zap.String("config_file", configFile),
				zap.Error(err))
		}
	}

	return config
}

// loadConfigFile 加载配置文件
func loadConfigFile(_ *terminal.TerminalTUIConfig, configFile string) error {
	// TODO: 实现配置文件加载
	// 这里可以使用viper或其他配置库来加载YAML/JSON配置
	logger.Info("加载TUI配置文件", zap.String("file", configFile))
	return nil
}

// NewAttachCmd 创建attach命令 - 附加到现有TUI会话
func NewAttachCmd() *cobra.Command {
	var sessionID string

	attachCmd := &cobra.Command{
		Use:   "attach [session-id]",
		Short: "附加到现有TUI会话",
		Long: `附加到现有的ClixGo TUI会话

📋 使用方式：
  clixgo attach                    # 附加到最近的会话
  clixgo attach main               # 附加到名为'main'的会话
  clixgo attach abc123             # 附加到ID为'abc123'的会话

💡 提示：
- 如果会话不存在，将显示可用会话列表
- 使用 'clixgo terminal session list' 查看所有会话`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 获取会话ID
			if len(args) > 0 {
				sessionID = args[0]
			}

			// 初始化logger
			if err := logger.InitLogger(); err != nil {
				return fmt.Errorf("初始化logger失败: %v", err)
			}

			logger.Info("尝试附加到TUI会话", zap.String("session_id", sessionID))

			// TODO: 实现会话附加逻辑
			// 1. 查找现有会话
			// 2. 如果找到，启动TUI并连接
			// 3. 如果没找到，显示可用会话列表

			fmt.Printf("🔍 正在查找会话: %s\n", sessionID)
			fmt.Printf("❌ 会话附加功能正在开发中...\n")
			fmt.Printf("💡 当前可以使用: clixgo tui --session %s\n", sessionID)

			return nil
		},
	}

	return attachCmd
}

// NewDetachCmd 创建detach命令 - 分离当前TUI会话
func NewDetachCmd() *cobra.Command {
	detachCmd := &cobra.Command{
		Use:   "detach",
		Short: "分离当前TUI会话",
		Long: `分离当前的ClixGo TUI会话，保持会话在后台运行

📋 功能：
- 🔌 分离当前TUI会话
- 💾 保持会话状态
- 🔄 稍后可以重新附加

💡 提示：
- 分离后可以使用 'clixgo attach' 重新连接
- 会话将继续在后台运行
- 所有面板和进程保持活跃状态`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 初始化logger
			if err := logger.InitLogger(); err != nil {
				return fmt.Errorf("初始化logger失败: %v", err)
			}

			logger.Info("分离TUI会话")

			// TODO: 实现会话分离逻辑
			// 1. 检查是否有活跃的TUI会话
			// 2. 发送分离信号
			// 3. 保存会话状态

			fmt.Printf("🔌 正在分离TUI会话...\n")
			fmt.Printf("❌ 会话分离功能正在开发中...\n")
			fmt.Printf("💡 当前可以在TUI中按 Prefix + d 分离会话\n")

			return nil
		},
	}

	return detachCmd
}

// NewListSessionsCmd 创建list-sessions命令 - 列出所有TUI会话
func NewListSessionsCmd() *cobra.Command {
	var (
		verbose bool
		format  string
	)

	listCmd := &cobra.Command{
		Use:     "list-sessions",
		Aliases: []string{"ls-sessions", "sessions"},
		Short:   "列出所有TUI会话",
		Long: `列出所有ClixGo TUI会话的状态信息

📋 显示信息：
- 📊 会话ID和名称
- ⏰ 创建时间和最后活跃时间  
- 🖥️  面板数量和状态
- 💾 会话大小和内存使用

📝 输出格式：
  default  - 默认格式
  json     - JSON格式
  yaml     - YAML格式
  table    - 表格格式`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 初始化logger
			if err := logger.InitLogger(); err != nil {
				return fmt.Errorf("初始化logger失败: %v", err)
			}

			logger.Info("列出TUI会话",
				zap.Bool("verbose", verbose),
				zap.String("format", format))

			// TODO: 实现会话列表逻辑
			// 1. 连接到会话管理器
			// 2. 获取所有会话信息
			// 3. 按指定格式输出

			fmt.Printf("📋 正在获取TUI会话列表...\n")
			fmt.Printf("❌ 会话列表功能正在开发中...\n")
			fmt.Printf("💡 当前可以使用: clixgo terminal session list\n")

			return nil
		},
	}

	// 命令选项
	listCmd.Flags().BoolVarP(&verbose, "verbose", "v", false,
		"显示详细信息")
	listCmd.Flags().StringVarP(&format, "format", "f", "default",
		"输出格式 (default|json|yaml|table)")

	return listCmd
}
