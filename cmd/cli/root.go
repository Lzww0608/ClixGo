/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-15 21:28:31
* @Description: CLI命令行工具的根命令和全局配置
 */

package cli

import (
	"os"

	"github.com/Lzww0608/ClixGo/pkg/commands"
	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use:   "clixgo",
	Short: "ClixGo - 增强型CLI工具套件",
	Long: `ClixGo 是一个高性能的增强型CLI工具套件，专为现代开发者设计。

特性:
  • 极速终端复用 - 比tmux快5倍的启动速度
  • 统一工具集成 - 网络诊断、文本处理、性能监控
  • 现代化界面 - 基于TUI的图形化界面
  • 智能补全 - 上下文感知的命令补全
  • 别名管理 - 自定义命令别名系统

使用示例:
  clixgo session new mywork    # 创建新会话
  clixgo window split -h       # 水平分割窗口
  clixgo tool network ping    # 网络诊断工具
  clixgo ui                    # 启动TUI界面`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// 初始化日志
		if err := logger.InitLogger(); err != nil {
			logger.Error("日志初始化失败", zap.Error(err))
		}

		// 初始化别名系统
		if err := commands.InitAliases(); err != nil {
			logger.Error("别名系统初始化失败", zap.Error(err))
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// 添加全局配置选项
	rootCmd.PersistentFlags().String("config", "", "配置文件路径")
	rootCmd.PersistentFlags().Bool("debug", false, "启用调试模式")
	rootCmd.PersistentFlags().Bool("verbose", false, "启用详细输出")

	// 添加子命令
	rootCmd.AddCommand(NewTerminalCmd())
	rootCmd.AddCommand(NewTUICmd()) // 新增TUI命令
	rootCmd.AddCommand(NewAliasCmd())
	rootCmd.AddCommand(NewHistoryCmd())
	rootCmd.AddCommand(NewFilesystemCmd())
	rootCmd.AddCommand(NewSecurityCmd())

	// 生成补全脚本命令
	rootCmd.AddCommand(&cobra.Command{
		Use:   "completion",
		Short: "生成补全脚本",
		Long:  "为bash生成命令补全脚本",
		Run: func(cmd *cobra.Command, args []string) {
			if err := commands.GenerateCompletionScript(rootCmd); err != nil {
				logger.Error("生成补全脚本失败", zap.Error(err))
				os.Exit(1)
			}
		},
	})
}
