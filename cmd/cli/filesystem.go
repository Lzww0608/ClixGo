package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/Lzww0608/ClixGo/pkg/filesystem"
)

func NewFilesystemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fs",
		Short: "文件系统操作",
		Long:  `提供各种文件系统操作功能，包括文件管理、权限控制、归档等`,
	}

	// 列出文件命令
	cmd.AddCommand(&cobra.Command{
		Use:   "ls",
		Short: "列出目录内容",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			recursive, _ := cmd.Flags().GetBool("recursive")
			showHidden, _ := cmd.Flags().GetBool("all")
			long, _ := cmd.Flags().GetBool("long")

			files, err := filesystem.ListFiles(args[0], recursive, showHidden)
			if err != nil {
				return err
			}

			if long {
				fmt.Printf("总计 %d 个文件\n", len(files))
				for _, file := range files {
					fmt.Printf("%s %s %s %8d %s %s\n",
						file.Permissions,
						file.Owner,
						file.Group,
						file.Size,
						file.ModTime.Format("2006-01-02 15:04:05"),
						file.Path)
				}
			} else {
				for _, file := range files {
					fmt.Println(file.Path)
				}
			}

			return nil
		},
	})
	cmd.Commands()[0].Flags().BoolP("recursive", "r", false, "递归列出子目录")
	cmd.Commands()[0].Flags().BoolP("all", "a", false, "显示隐藏文件")
	cmd.Commands()[0].Flags().BoolP("long", "l", false, "显示详细信息")

	// 复制文件命令
	cmd.AddCommand(&cobra.Command{
		Use:   "cp",
		Short: "复制文件或目录",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			result := filesystem.CopyFile(args[0], args[1])
			if !result.Success {
				return result.Error
			}
			fmt.Println(result.Message)
			return nil
		},
	})

	// 移动文件命令
	cmd.AddCommand(&cobra.Command{
		Use:   "mv",
		Short: "移动文件或目录",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			result := filesystem.MoveFile(args[0], args[1])
			if !result.Success {
				return result.Error
			}
			fmt.Println(result.Message)
			return nil
		},
	})

	// 删除文件命令
	cmd.AddCommand(&cobra.Command{
		Use:   "rm",
		Short: "删除文件或目录",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			recursive, _ := cmd.Flags().GetBool("recursive")
			force, _ := cmd.Flags().GetBool("force")

			if !force {
				fmt.Printf("确定要删除 %s 吗？(y/N): ", args[0])
				var confirm string
				fmt.Scanln(&confirm)
				if strings.ToLower(confirm) != "y" {
					return nil
				}
			}

			result := filesystem.DeleteFile(args[0], recursive)
			if !result.Success {
				return result.Error
			}
			fmt.Println(result.Message)
			return nil
		},
	})
	cmd.Commands()[3].Flags().BoolP("recursive", "r", false, "递归删除目录")
	cmd.Commands()[3].Flags().BoolP("force", "f", false, "强制删除，不提示确认")

	// 创建目录命令
	cmd.AddCommand(&cobra.Command{
		Use:   "mkdir",
		Short: "创建目录",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parents, _ := cmd.Flags().GetBool("parents")
			result := filesystem.CreateDirectory(args[0], parents)
			if !result.Success {
				return result.Error
			}
			fmt.Println(result.Message)
			return nil
		},
	})
	cmd.Commands()[4].Flags().BoolP("parents", "p", false, "创建父目录")

	// 修改权限命令
	cmd.AddCommand(&cobra.Command{
		Use:   "chmod",
		Short: "修改文件权限",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := strconv.ParseUint(args[0], 8, 32)
			if err != nil {
				return fmt.Errorf("无效的权限模式: %v", err)
			}

			recursive, _ := cmd.Flags().GetBool("recursive")
			result := filesystem.ChangePermissions(args[1], os.FileMode(mode), recursive)
			if !result.Success {
				return result.Error
			}
			fmt.Println(result.Message)
			return nil
		},
	})
	cmd.Commands()[5].Flags().BoolP("recursive", "R", false, "递归修改权限")

	// 修改所有者命令
	cmd.AddCommand(&cobra.Command{
		Use:   "chown",
		Short: "修改文件所有者",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			uid, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("无效的用户ID: %v", err)
			}

			gid, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("无效的组ID: %v", err)
			}

			recursive, _ := cmd.Flags().GetBool("recursive")
			result := filesystem.ChangeOwner(args[2], uid, gid, recursive)
			if !result.Success {
				return result.Error
			}
			fmt.Println(result.Message)
			return nil
		},
	})
	cmd.Commands()[6].Flags().BoolP("recursive", "R", false, "递归修改所有者")

	// 创建符号链接命令
	cmd.AddCommand(&cobra.Command{
		Use:   "ln",
		Short: "创建符号链接",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			result := filesystem.CreateSymlink(args[0], args[1])
			if !result.Success {
				return result.Error
			}
			fmt.Println(result.Message)
			return nil
		},
	})

	// 归档命令
	cmd.AddCommand(&cobra.Command{
		Use:   "archive",
		Short: "创建归档文件",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			result := filesystem.Archive(args[:len(args)-1], args[len(args)-1], format)
			if !result.Success {
				return result.Error
			}
			fmt.Println(result.Message)
			return nil
		},
	})
	cmd.Commands()[8].Flags().StringP("format", "f", "zip", "归档格式 (zip, tar, tar.gz)")

	// 解压命令
	cmd.AddCommand(&cobra.Command{
		Use:   "extract",
		Short: "解压归档文件",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			result := filesystem.Extract(args[0], args[1])
			if !result.Success {
				return result.Error
			}
			fmt.Println(result.Message)
			return nil
		},
	})

	// 搜索文件命令
	cmd.AddCommand(&cobra.Command{
		Use:   "find",
		Short: "搜索文件",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			caseSensitive, _ := cmd.Flags().GetBool("case-sensitive")
			files, err := filesystem.SearchFiles(args[0], args[1], caseSensitive)
			if err != nil {
				return err
			}

			for _, file := range files {
				fmt.Println(file.Path)
			}
			return nil
		},
	})
	cmd.Commands()[10].Flags().BoolP("case-sensitive", "c", false, "区分大小写")

	return cmd
} 