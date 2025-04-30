package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/Lzww0608/ClixGo/pkg/security"
)

var (
	commandManager *security.CommandManager
)

func init() {
	var err error
	commandManager, err = security.NewCommandManager(
		"~/.gocli/stats.json",
		"~/.gocli/policies.json",
	)
	if err != nil {
		fmt.Printf("初始化命令管理器失败: %v\n", err)
		os.Exit(1)
	}
}

func NewSecurityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "security",
		Short: "安全管理",
		Long:  `提供命令执行统计和权限控制功能`,
	}

	// 查看统计信息命令
	cmd.AddCommand(&cobra.Command{
		Use:   "stats",
		Short: "查看命令执行统计",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			duration, _ := cmd.Flags().GetDuration("duration")
			format, _ := cmd.Flags().GetString("format")

			var command string
			if len(args) > 0 {
				command = args[0]
			}

			stats := commandManager.GetCommandStats(command, duration)
			if len(stats) == 0 {
				fmt.Println("没有找到统计信息")
				return nil
			}

			switch format {
			case "json":
				data, err := json.MarshalIndent(stats, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
			default:
				fmt.Printf("总计 %d 条记录\n", len(stats))
				for _, stat := range stats {
					fmt.Printf("命令: %s\n", stat.Command)
					fmt.Printf("用户: %s\n", stat.User)
					fmt.Printf("时间: %s\n", stat.Timestamp.Format("2006-01-02 15:04:05"))
					fmt.Printf("耗时: %dms\n", stat.Duration)
					fmt.Printf("状态: %v\n", stat.Success)
					if stat.Error != "" {
						fmt.Printf("错误: %s\n", stat.Error)
					}
					fmt.Printf("参数: %v\n", stat.Args)
					fmt.Println("---")
				}
			}

			return nil
		},
	})
	cmd.Commands()[0].Flags().DurationP("duration", "d", time.Hour*24, "统计时间范围")
	cmd.Commands()[0].Flags().StringP("format", "f", "text", "输出格式 (text, json)")

	// 添加策略命令
	cmd.AddCommand(&cobra.Command{
		Use:   "policy",
		Short: "管理命令执行策略",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			command := args[0]
			allowed, _ := cmd.Flags().GetBool("allowed")
			users, _ := cmd.Flags().GetStringSlice("users")
			groups, _ := cmd.Flags().GetStringSlice("groups")
			timeRange, _ := cmd.Flags().GetStringSlice("time-range")
			maxDuration, _ := cmd.Flags().GetInt("max-duration")
			maxCalls, _ := cmd.Flags().GetInt("max-calls")

			policy := security.CommandPolicy{
				Command:     command,
				Allowed:     allowed,
				Users:       users,
				Groups:      groups,
				TimeRange:   timeRange,
				MaxDuration: maxDuration,
				MaxCalls:    maxCalls,
			}

			return commandManager.AddPolicy(policy)
		},
	})
	cmd.Commands()[1].Flags().BoolP("allowed", "a", true, "是否允许执行")
	cmd.Commands()[1].Flags().StringSliceP("users", "u", nil, "允许的用户列表")
	cmd.Commands()[1].Flags().StringSliceP("groups", "g", nil, "允许的用户组列表")
	cmd.Commands()[1].Flags().StringSliceP("time-range", "t", nil, "允许执行的时间范围")
	cmd.Commands()[1].Flags().IntP("max-duration", "d", 0, "最大执行时间（秒）")
	cmd.Commands()[1].Flags().IntP("max-calls", "c", 0, "每小时最大调用次数")

	// 移除策略命令
	cmd.AddCommand(&cobra.Command{
		Use:   "remove-policy",
		Short: "移除命令执行策略",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return commandManager.RemovePolicy(args[0])
		},
	})

	// 检查权限命令
	cmd.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "检查命令执行权限",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			currentUser, err := user.Current()
			if err != nil {
				return err
			}

			// 获取用户组（简化实现）
			groups := []string{currentUser.Username}

			allowed, reason := commandManager.CheckPermission(args[0], currentUser.Username, groups)
			if allowed {
				fmt.Println("允许执行")
			} else {
				fmt.Printf("禁止执行: %s\n", reason)
			}
			return nil
		},
	})

	return cmd
}

// WrapCommand 包装命令以添加统计和权限控制
func WrapCommand(cmd *cobra.Command) *cobra.Command {
	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		startTime := time.Now()

		// 获取当前用户信息
		currentUser, err := user.Current()
		if err != nil {
			return err
		}

		// 获取用户组（简化实现）
		groups := []string{currentUser.Username}

		// 检查权限
		allowed, reason := commandManager.CheckPermission(cmd.Use, currentUser.Username, groups)
		if !allowed {
			return fmt.Errorf("权限不足: %s", reason)
		}

		// 执行原始命令
		var cmdErr error
		if originalRunE != nil {
			cmdErr = originalRunE(cmd, args)
		}

		// 记录统计信息
		stats := security.CommandStats{
			Command:     cmd.Use,
			User:        currentUser.Username,
			Timestamp:   startTime,
			Duration:    time.Since(startTime).Milliseconds(),
			Success:     cmdErr == nil,
			Error:       "",
			Args:        args,
			Environment: strings.Join(os.Environ(), " "),
		}
		if cmdErr != nil {
			stats.Error = cmdErr.Error()
		}

		if err := commandManager.RecordCommand(stats); err != nil {
			fmt.Printf("记录统计信息失败: %v\n", err)
		}

		return cmdErr
	}

	return cmd
} 