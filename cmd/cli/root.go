package cli

import (
	"fmt"
	"os"

	"github.com/Lzww0608/ClixGo/cmd/task"
	"github.com/Lzww0608/ClixGo/pkg/alias"
	"github.com/Lzww0608/ClixGo/pkg/completion"
	"github.com/Lzww0608/ClixGo/pkg/config"
	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ClixGo",
	Short: "一个强大的命令行工具",
	Long: `ClixGo是一个功能强大的命令行工具，支持：
- 串行执行多个命令
- 并行执行多个命令
- 自定义命令执行
- AWK命令处理
- grep命令处理
- sed命令处理
- 管道命令处理
- 命令历史记录
- 命令别名
- 命令补全
- 后台任务管理
`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		if err := config.InitConfig(configPath); err != nil {
			fmt.Printf("初始化配置失败: %v\n", err)
			os.Exit(1)
		}
		logger.InitLogger()
		if err := alias.InitAliases(); err != nil {
			fmt.Printf("初始化别名失败: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// 在这里添加全局标志
	rootCmd.PersistentFlags().StringP("config", "c", "", "配置文件路径")

	// 添加子命令
	rootCmd.AddCommand(NewSequentialCmd())
	rootCmd.AddCommand(NewParallelCmd())
	rootCmd.AddCommand(NewAWKCmd())
	rootCmd.AddCommand(NewGrepCmd())
	rootCmd.AddCommand(NewSedCmd())
	rootCmd.AddCommand(NewPipeCmd())
	rootCmd.AddCommand(NewHistoryCmd())
	rootCmd.AddCommand(NewAliasCmd())
	rootCmd.AddCommand(NewNetworkCmd())
	rootCmd.AddCommand(NewTextCmd())
	rootCmd.AddCommand(NewFilesystemCmd())
	rootCmd.AddCommand(NewSecurityCmd())
	rootCmd.AddCommand(NewTerminalCmd())

	// 添加任务管理命令
	rootCmd.AddCommand(task.Command())

	// 添加补全命令
	rootCmd.AddCommand(&cobra.Command{
		Use:   "completion",
		Short: "生成命令补全脚本",
		RunE: func(cmd *cobra.Command, args []string) error {
			return completion.GenerateCompletionScript(rootCmd)
		},
	})
}
