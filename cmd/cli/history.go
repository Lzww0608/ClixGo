/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-15 21:29:28
* @Description: 命令历史管理的CLI命令定义
 */

package cli

import (
	"fmt"
	"strconv"

	"github.com/Lzww0608/ClixGo/pkg/terminal"
	"github.com/spf13/cobra"
)

func NewHistoryCmd() *cobra.Command {
	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "管理命令历史",
		Long:  "查看、清除命令历史记录",
	}

	// 列出历史记录
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出命令历史",
		RunE: func(cmd *cobra.Command, args []string) error {
			history, err := terminal.GetHistory()
			if err != nil {
				return err
			}

			if len(history) == 0 {
				fmt.Println("没有历史记录")
				return nil
			}

			fmt.Println("命令历史记录:")
			for i, h := range history {
				fmt.Printf("%d. %s [%s] (%s)\n", i+1, h.Command, h.Status, h.Duration)
			}
			return nil
		},
	}

	// 显示特定历史记录
	showCmd := &cobra.Command{
		Use:   "show [index]",
		Short: "显示特定历史记录的详细信息",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			index, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("无效的索引: %s", args[0])
			}

			history, err := terminal.GetHistory()
			if err != nil {
				return err
			}

			if index < 1 || index > len(history) {
				return fmt.Errorf("索引超出范围: %d", index)
			}

			h := history[index-1]
			fmt.Printf("命令: %s\n", h.Command)
			fmt.Printf("状态: %s\n", h.Status)
			fmt.Printf("开始时间: %s\n", h.StartTime.Format("2006-01-02 15:04:05"))
			fmt.Printf("结束时间: %s\n", h.EndTime.Format("2006-01-02 15:04:05"))
			fmt.Printf("执行时长: %s\n", h.Duration)
			if h.Output != "" {
				fmt.Printf("输出:\n%s\n", h.Output)
			}
			return nil
		},
	}

	// 清除历史记录
	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "清除所有历史记录",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := terminal.ClearHistory(); err != nil {
				return err
			}
			fmt.Println("历史记录已清除")
			return nil
		},
	}

	historyCmd.AddCommand(listCmd, showCmd, clearCmd)
	return historyCmd
}
