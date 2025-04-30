package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gocli/pkg/history"
	"gocli/pkg/logger"
)

func NewHistoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "管理命令历史记录",
		Long:  `查看、清除命令历史记录`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "列出历史命令",
		RunE: func(cmd *cobra.Command, args []string) error {
			history, err := history.GetHistory()
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "时间\t命令\t状态\t耗时")
			fmt.Fprintln(w, "----\t----\t----\t----")

			for _, h := range history {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					h.StartTime.Format("2006-01-02 15:04:05"),
					h.Command,
					h.Status,
					h.Duration,
				)
			}
			w.Flush()
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "clear",
		Short: "清除历史记录",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := history.ClearHistory(); err != nil {
				return err
			}
			logger.Info("历史记录已清除")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "显示命令详情",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			history, err := history.GetHistory()
			if err != nil {
				return err
			}

			index := 0
			fmt.Sscanf(args[0], "%d", &index)
			if index < 0 || index >= len(history) {
				return fmt.Errorf("无效的历史记录索引")
			}

			h := history[index]
			fmt.Printf("命令: %s\n", h.Command)
			fmt.Printf("状态: %s\n", h.Status)
			fmt.Printf("开始时间: %s\n", h.StartTime.Format("2006-01-02 15:04:05"))
			fmt.Printf("结束时间: %s\n", h.EndTime.Format("2006-01-02 15:04:05"))
			fmt.Printf("耗时: %s\n", h.Duration)
			fmt.Printf("输出:\n%s\n", h.Output)

			return nil
		},
	})

	return cmd
} 