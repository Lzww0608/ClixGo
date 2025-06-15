/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-15 21:29:23
* @Description: 文件系统操作的CLI命令定义
 */

package cli

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/utils"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func NewFilesystemCmd() *cobra.Command {
	filesystemCmd := &cobra.Command{
		Use:   "fs",
		Short: "文件系统操作",
		Long:  "文件和目录的管理操作",
	}

	// 列出文件
	listCmd := &cobra.Command{
		Use:   "list [path]",
		Short: "列出文件和目录",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			recursive, _ := cmd.Flags().GetBool("recursive")
			showHidden, _ := cmd.Flags().GetBool("hidden")

			files, err := utils.ListFiles(path, recursive, showHidden)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "名称\t大小\t权限\t修改时间\t类型")
			fmt.Fprintln(w, "----\t----\t----\t--------\t----")

			for _, file := range files {
				sizeStr := "-"
				if !file.IsDir {
					sizeStr = strconv.FormatInt(file.Size, 10)
				}

				typeStr := "文件"
				if file.IsDir {
					typeStr = "目录"
				}
				if file.IsSymlink {
					typeStr = "符号链接"
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					file.Name,
					sizeStr,
					file.Permissions,
					file.ModTime.Format("2006-01-02 15:04:05"),
					typeStr,
				)
			}
			w.Flush()
			return nil
		},
	}
	listCmd.Flags().BoolP("recursive", "r", false, "递归列出子目录")
	listCmd.Flags().BoolP("hidden", "a", false, "显示隐藏文件")

	// 复制文件
	copyCmd := &cobra.Command{
		Use:   "copy [source] [destination]",
		Short: "复制文件或目录",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			result := utils.CopyFile(args[0], args[1])
			if !result.Success {
				logger.Error("文件复制失败", zap.Error(result.Error))
				return result.Error
			}
			fmt.Println(result.Message)
			logger.Info("文件复制成功", zap.String("source", args[0]), zap.String("destination", args[1]))
			return nil
		},
	}

	// 移动文件
	moveCmd := &cobra.Command{
		Use:   "move [source] [destination]",
		Short: "移动或重命名文件",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			result := utils.MoveFile(args[0], args[1])
			if !result.Success {
				logger.Error("文件移动失败", zap.Error(result.Error))
				return result.Error
			}
			fmt.Println(result.Message)
			logger.Info("文件移动成功", zap.String("source", args[0]), zap.String("destination", args[1]))
			return nil
		},
	}

	// 删除文件
	deleteCmd := &cobra.Command{
		Use:   "delete [path]",
		Short: "删除文件或目录",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			recursive, _ := cmd.Flags().GetBool("recursive")
			result := utils.DeleteFile(args[0], recursive)
			if !result.Success {
				logger.Error("文件删除失败", zap.Error(result.Error))
				return result.Error
			}
			fmt.Println(result.Message)
			logger.Info("文件删除成功", zap.String("path", args[0]))
			return nil
		},
	}
	deleteCmd.Flags().BoolP("recursive", "r", false, "递归删除目录")

	// 创建目录
	mkdirCmd := &cobra.Command{
		Use:   "mkdir [path]",
		Short: "创建目录",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parents, _ := cmd.Flags().GetBool("parents")
			result := utils.CreateDirectory(args[0], parents)
			if !result.Success {
				logger.Error("目录创建失败", zap.Error(result.Error))
				return result.Error
			}
			fmt.Println(result.Message)
			logger.Info("目录创建成功", zap.String("path", args[0]))
			return nil
		},
	}
	mkdirCmd.Flags().BoolP("parents", "p", false, "创建父目录")

	filesystemCmd.AddCommand(listCmd, copyCmd, moveCmd, deleteCmd, mkdirCmd)
	return filesystemCmd
}
