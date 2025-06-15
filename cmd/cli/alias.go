/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-15 21:28:37
* @Description: 别名管理命令的实现
 */

package cli

import (
	"fmt"

	"github.com/Lzww0608/ClixGo/pkg/commands"
	"github.com/spf13/cobra"
)

func NewAliasCmd() *cobra.Command {
	aliasCmd := &cobra.Command{
		Use:   "alias",
		Short: "管理命令别名",
		Long:  "创建、删除、列出命令别名",
	}

	// 添加别名
	addCmd := &cobra.Command{
		Use:   "add [name] [command]",
		Short: "添加新别名",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.AddAlias(args[0], args[1])
		},
	}

	// 删除别名
	removeCmd := &cobra.Command{
		Use:   "remove [name]",
		Short: "删除别名",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.RemoveAlias(args[0])
		},
	}

	// 列出别名
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出所有别名",
		RunE: func(cmd *cobra.Command, args []string) error {
			aliases := commands.ListAliases()
			if len(aliases) == 0 {
				fmt.Println("没有设置任何别名")
				return nil
			}

			fmt.Println("当前设置的别名:")
			for name, command := range aliases {
				fmt.Printf("  %s -> %s\n", name, command)
			}
			return nil
		},
	}

	aliasCmd.AddCommand(addCmd, removeCmd, listCmd)
	return aliasCmd
}
