package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/Lzww0608/ClixGo/pkg/alias"
	"github.com/Lzww0608/ClixGo/pkg/logger"
)

func NewAliasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alias",
		Short: "管理命令别名",
		Long:  `添加、删除、列出命令别名`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "add",
		Short: "添加别名",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			command := args[1]
			if err := alias.AddAlias(name, command); err != nil {
				return err
			}
			logger.Info("别名添加成功", logger.Log.String("name", name), logger.Log.String("command", command))
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "remove",
		Short: "删除别名",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := alias.RemoveAlias(name); err != nil {
				return err
			}
			logger.Info("别名删除成功", logger.Log.String("name", name))
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "列出所有别名",
		RunE: func(cmd *cobra.Command, args []string) error {
			aliases := alias.ListAliases()
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "别名\t命令")
			fmt.Fprintln(w, "----\t----")

			for name, command := range aliases {
				fmt.Fprintf(w, "%s\t%s\n", name, command)
			}
			w.Flush()
			return nil
		},
	})

	return cmd
} 