/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-12 10:50:00
* @Description: 文件系统操作的CLI命令定义
 */

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Lzww0608/ClixGo/pkg/filesystem"
	"github.com/spf13/cobra"
)

func NewFilesystemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fs",
		Short: "📁 文件系统操作",
		Long:  `提供各种文件系统操作功能，包括文件管理、权限控制、归档等`,
	}

	// 列出文件命令
	lsCmd := &cobra.Command{
		Use:   "ls <directory>",
		Short: "📋 列出目录内容",
		Long: `列出指定目录的文件和子目录

示例:
  clixgo fs ls /home/user          # 列出目录内容
  clixgo fs ls -l /var/log         # 显示详细信息
  clixgo fs ls -a /tmp             # 显示隐藏文件
  clixgo fs ls -r /etc             # 递归列出子目录`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dirPath := args[0]

			// 验证目录路径
			if strings.TrimSpace(dirPath) == "" {
				return fmt.Errorf("❌ 错误: 目录路径不能为空\n\n💡 提示: 请提供要列出的目录路径")
			}

			// 检查目录是否存在
			if _, err := os.Stat(dirPath); os.IsNotExist(err) {
				return fmt.Errorf("❌ 错误: 目录不存在: %s\n\n💡 建议:\n  • 检查路径是否正确\n  • 确认目录是否已创建\n  • 验证权限是否足够", dirPath)
			}

			// 检查是否为目录
			if info, err := os.Stat(dirPath); err == nil && !info.IsDir() {
				return fmt.Errorf("❌ 错误: '%s' 不是目录\n\n💡 提示: 请提供目录路径，而不是文件路径", dirPath)
			}

			recursive, _ := cmd.Flags().GetBool("recursive")
			showHidden, _ := cmd.Flags().GetBool("all")
			long, _ := cmd.Flags().GetBool("long")

			fmt.Printf("📋 列出目录: %s\n", dirPath)
			if recursive {
				fmt.Println("🔄 递归模式已启用")
			}
			if showHidden {
				fmt.Println("👁️  显示隐藏文件")
			}
			fmt.Println()

			files, err := filesystem.ListFiles(dirPath, recursive, showHidden)
			if err != nil {
				return fmt.Errorf("❌ 列出文件失败: %v\n\n💡 建议:\n  • 检查目录权限\n  • 确认路径是否正确\n  • 尝试使用绝对路径", err)
			}

			if len(files) == 0 {
				fmt.Println("ℹ️  目录为空")
				return nil
			}

			if long {
				fmt.Printf("📊 总计 %d 个文件/目录\n\n", len(files))
				fmt.Printf("%-12s %-8s %-8s %10s %-20s %s\n", "权限", "所有者", "组", "大小", "修改时间", "名称")
				fmt.Println(strings.Repeat("-", 80))
				for _, file := range files {
					fmt.Printf("%-12s %-8s %-8s %10d %-20s %s\n",
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
	}
	lsCmd.Flags().BoolP("recursive", "r", false, "递归列出子目录")
	lsCmd.Flags().BoolP("all", "a", false, "显示隐藏文件")
	lsCmd.Flags().BoolP("long", "l", false, "显示详细信息")
	cmd.AddCommand(lsCmd)

	// 复制文件命令
	cpCmd := &cobra.Command{
		Use:   "cp <source> <destination>",
		Short: "📄 复制文件或目录",
		Long: `复制文件或目录到指定位置

示例:
  clixgo fs cp file.txt backup.txt     # 复制文件
  clixgo fs cp /etc/hosts /tmp/        # 复制到目录
  clixgo fs cp -r /home/user /backup/  # 递归复制目录`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			source := args[0]
			dest := args[1]

			// 验证参数
			if strings.TrimSpace(source) == "" {
				return fmt.Errorf("❌ 错误: 源路径不能为空")
			}
			if strings.TrimSpace(dest) == "" {
				return fmt.Errorf("❌ 错误: 目标路径不能为空")
			}

			// 检查源文件是否存在
			if _, err := os.Stat(source); os.IsNotExist(err) {
				return fmt.Errorf("❌ 错误: 源文件不存在: %s\n\n💡 建议:\n  • 检查文件路径是否正确\n  • 确认文件是否已创建", source)
			}

			// 检查目标路径
			if destInfo, err := os.Stat(dest); err == nil {
				if destInfo.IsDir() {
					// 目标是目录，构建完整路径
					dest = filepath.Join(dest, filepath.Base(source))
				} else {
					// 目标文件已存在，询问是否覆盖
					fmt.Printf("⚠️  目标文件已存在: %s\n", dest)
					fmt.Print("是否覆盖？(y/N): ")
					var confirm string
					fmt.Scanln(&confirm)
					if strings.ToLower(confirm) != "y" {
						fmt.Println("❌ 操作已取消")
						return nil
					}
				}
			}

			fmt.Printf("📄 复制: %s → %s\n", source, dest)
			result := filesystem.CopyFile(source, dest)
			if !result.Success {
				return fmt.Errorf("❌ 复制失败: %v\n\n💡 建议:\n  • 检查源文件权限\n  • 确认目标目录权限\n  • 检查磁盘空间是否足够", result.Error)
			}

			fmt.Printf("✅ %s\n", result.Message)
			return nil
		},
	}
	cmd.AddCommand(cpCmd)

	// 移动文件命令
	mvCmd := &cobra.Command{
		Use:   "mv <source> <destination>",
		Short: "🚚 移动文件或目录",
		Long: `移动文件或目录到指定位置（重命名）

示例:
  clixgo fs mv old.txt new.txt         # 重命名文件
  clixgo fs mv file.txt /tmp/          # 移动到目录
  clixgo fs mv /old/dir /new/location  # 移动目录`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			source := args[0]
			dest := args[1]

			// 验证参数
			if strings.TrimSpace(source) == "" {
				return fmt.Errorf("❌ 错误: 源路径不能为空")
			}
			if strings.TrimSpace(dest) == "" {
				return fmt.Errorf("❌ 错误: 目标路径不能为空")
			}

			// 检查源文件是否存在
			if _, err := os.Stat(source); os.IsNotExist(err) {
				return fmt.Errorf("❌ 错误: 源文件不存在: %s\n\n💡 建议:\n  • 检查文件路径是否正确\n  • 确认文件是否已创建", source)
			}

			// 检查目标路径
			if destInfo, err := os.Stat(dest); err == nil {
				if destInfo.IsDir() {
					// 目标是目录，构建完整路径
					dest = filepath.Join(dest, filepath.Base(source))
				} else {
					// 目标文件已存在，询问是否覆盖
					fmt.Printf("⚠️  目标文件已存在: %s\n", dest)
					fmt.Print("是否覆盖？(y/N): ")
					var confirm string
					fmt.Scanln(&confirm)
					if strings.ToLower(confirm) != "y" {
						fmt.Println("❌ 操作已取消")
						return nil
					}
				}
			}

			fmt.Printf("🚚 移动: %s → %s\n", source, dest)
			result := filesystem.MoveFile(source, dest)
			if !result.Success {
				return fmt.Errorf("❌ 移动失败: %v\n\n💡 建议:\n  • 检查源文件权限\n  • 确认目标目录权限\n  • 检查是否跨文件系统移动", result.Error)
			}

			fmt.Printf("✅ %s\n", result.Message)
			return nil
		},
	}
	cmd.AddCommand(mvCmd)

	// 删除文件命令
	rmCmd := &cobra.Command{
		Use:   "rm <path>",
		Short: "🗑️  删除文件或目录",
		Long: `删除指定的文件或目录

示例:
  clixgo fs rm file.txt            # 删除文件
  clixgo fs rm -r directory        # 递归删除目录
  clixgo fs rm -f file.txt         # 强制删除，不提示

⚠️  警告: 删除操作不可恢复，请谨慎使用`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			// 验证路径
			if strings.TrimSpace(path) == "" {
				return fmt.Errorf("❌ 错误: 路径不能为空")
			}

			// 检查文件是否存在
			info, err := os.Stat(path)
			if os.IsNotExist(err) {
				return fmt.Errorf("❌ 错误: 文件或目录不存在: %s", path)
			}

			recursive, _ := cmd.Flags().GetBool("recursive")
			force, _ := cmd.Flags().GetBool("force")

			// 检查是否为目录且未指定递归
			if info.IsDir() && !recursive {
				return fmt.Errorf("❌ 错误: '%s' 是目录\n\n💡 提示: 删除目录请使用 -r 标志:\n  clixgo fs rm -r %s", path, path)
			}

			// 安全检查 - 防止删除重要目录
			absPath, _ := filepath.Abs(path)
			dangerousPaths := []string{"/", "/bin", "/usr", "/etc", "/var", "/home"}
			for _, dangerous := range dangerousPaths {
				if absPath == dangerous {
					return fmt.Errorf("❌ 错误: 拒绝删除系统重要目录: %s\n\n💡 安全提示: 此操作可能损坏系统", dangerous)
				}
			}

			if !force {
				if info.IsDir() {
					fmt.Printf("⚠️  确定要删除目录 '%s' 及其所有内容吗？(y/N): ", path)
				} else {
					fmt.Printf("⚠️  确定要删除文件 '%s' 吗？(y/N): ", path)
				}
				var confirm string
				fmt.Scanln(&confirm)
				if strings.ToLower(confirm) != "y" {
					fmt.Println("❌ 操作已取消")
					return nil
				}
			}

			fmt.Printf("🗑️  删除: %s\n", path)
			result := filesystem.DeleteFile(path, recursive)
			if !result.Success {
				return fmt.Errorf("❌ 删除失败: %v\n\n💡 建议:\n  • 检查文件权限\n  • 确认文件未被占用\n  • 尝试使用管理员权限", result.Error)
			}

			fmt.Printf("✅ %s\n", result.Message)
			return nil
		},
	}
	rmCmd.Flags().BoolP("recursive", "r", false, "递归删除目录")
	rmCmd.Flags().BoolP("force", "f", false, "强制删除，不提示确认")
	cmd.AddCommand(rmCmd)

	// 创建目录命令
	mkdirCmd := &cobra.Command{
		Use:   "mkdir <directory>",
		Short: "📁 创建目录",
		Long: `创建新目录

示例:
  clixgo fs mkdir newdir           # 创建目录
  clixgo fs mkdir -p path/to/dir   # 创建父目录`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dirPath := args[0]

			// 验证路径
			if strings.TrimSpace(dirPath) == "" {
				return fmt.Errorf("❌ 错误: 目录路径不能为空")
			}

			// 检查目录是否已存在
			if _, err := os.Stat(dirPath); err == nil {
				return fmt.Errorf("❌ 错误: 目录已存在: %s", dirPath)
			}

			parents, _ := cmd.Flags().GetBool("parents")

			fmt.Printf("📁 创建目录: %s\n", dirPath)
			if parents {
				fmt.Println("🔄 创建父目录模式已启用")
			}

			result := filesystem.CreateDirectory(dirPath, parents)
			if !result.Success {
				return fmt.Errorf("❌ 创建目录失败: %v\n\n💡 建议:\n  • 检查父目录权限\n  • 确认路径格式正确\n  • 使用 -p 标志创建父目录", result.Error)
			}

			fmt.Printf("✅ %s\n", result.Message)
			return nil
		},
	}
	mkdirCmd.Flags().BoolP("parents", "p", false, "创建父目录")
	cmd.AddCommand(mkdirCmd)

	// 修改权限命令
	chmodCmd := &cobra.Command{
		Use:   "chmod <mode> <path>",
		Short: "🔐 修改文件权限",
		Long: `修改文件或目录的权限

示例:
  clixgo fs chmod 755 file.txt     # 设置权限为755
  clixgo fs chmod 644 *.txt        # 设置所有txt文件权限
  clixgo fs chmod -r 755 /path/    # 递归修改目录权限

权限说明:
  7 = 读+写+执行 (rwx)
  6 = 读+写 (rw-)
  5 = 读+执行 (r-x)
  4 = 只读 (r--)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			modeStr := args[0]
			path := args[1]

			// 验证参数
			if strings.TrimSpace(modeStr) == "" {
				return fmt.Errorf("❌ 错误: 权限模式不能为空\n\n💡 提示: 请提供八进制权限，例如: 755, 644")
			}
			if strings.TrimSpace(path) == "" {
				return fmt.Errorf("❌ 错误: 文件路径不能为空")
			}

			// 验证权限模式
			mode, err := strconv.ParseUint(modeStr, 8, 32)
			if err != nil {
				return fmt.Errorf("❌ 错误: 无效的权限模式: %s\n\n💡 提示:\n  • 使用八进制数字 (0-7)\n  • 常用权限: 755, 644, 600\n  • 格式: 用户权限+组权限+其他权限", modeStr)
			}

			// 验证权限范围
			if mode > 0777 {
				return fmt.Errorf("❌ 错误: 权限模式超出范围: %s\n\n💡 提示: 权限模式应在 000-777 之间", modeStr)
			}

			// 检查文件是否存在
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("❌ 错误: 文件不存在: %s", path)
			}

			recursive, _ := cmd.Flags().GetBool("recursive-chmod")

			fmt.Printf("🔐 修改权限: %s → %s\n", path, modeStr)
			if recursive {
				fmt.Println("🔄 递归模式已启用")
			}

			result := filesystem.ChangePermissions(path, os.FileMode(mode), recursive)
			if !result.Success {
				return fmt.Errorf("❌ 修改权限失败: %v\n\n💡 建议:\n  • 检查文件所有权\n  • 确认当前用户权限\n  • 尝试使用管理员权限", result.Error)
			}

			fmt.Printf("✅ %s\n", result.Message)
			return nil
		},
	}
	chmodCmd.Flags().Bool("recursive-chmod", false, "递归修改权限")
	cmd.AddCommand(chmodCmd)

	// 修改所有者命令
	chownCmd := &cobra.Command{
		Use:   "chown <uid> <gid> <path>",
		Short: "👤 修改文件所有者",
		Long: `修改文件或目录的所有者

示例:
  clixgo fs chown 1000 1000 file.txt     # 设置文件所有者为uid 1000, gid 1000
  clixgo fs chown -r 1000 1000 /path/    # 递归修改目录所有者`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			uid, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("❌ 错误: 无效的用户ID: %v", err)
			}

			gid, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("❌ 错误: 无效的组ID: %v", err)
			}

			path := args[2]

			// 验证路径
			if strings.TrimSpace(path) == "" {
				return fmt.Errorf("❌ 错误: 文件路径不能为空")
			}

			// 检查文件是否存在
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("❌ 错误: 文件不存在: %s", path)
			}

			recursive, _ := cmd.Flags().GetBool("recursive-chown")

			fmt.Printf("👤 修改所有者: %s → uid %d, gid %d\n", path, uid, gid)
			if recursive {
				fmt.Println("🔄 递归模式已启用")
			}

			result := filesystem.ChangeOwner(path, uid, gid, recursive)
			if !result.Success {
				return fmt.Errorf("❌ 修改所有者失败: %v\n\n💡 建议:\n  • 检查文件权限\n  • 确认当前用户权限\n  • 尝试使用管理员权限", result.Error)
			}

			fmt.Printf("✅ %s\n", result.Message)
			return nil
		},
	}
	chownCmd.Flags().Bool("recursive-chown", false, "递归修改所有者")
	cmd.AddCommand(chownCmd)

	// 创建符号链接命令
	lnCmd := &cobra.Command{
		Use:   "ln <target> <link>",
		Short: "🔗 创建符号链接",
		Long: `创建符号链接到指定目标

示例:
  clixgo fs ln target.txt link.txt     # 创建符号链接
  clixgo fs ln /old/dir /new/location  # 创建目录符号链接`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			link := args[1]

			// 验证参数
			if strings.TrimSpace(target) == "" {
				return fmt.Errorf("❌ 错误: 目标路径不能为空")
			}
			if strings.TrimSpace(link) == "" {
				return fmt.Errorf("❌ 错误: 链接路径不能为空")
			}

			// 检查目标是否存在
			if _, err := os.Stat(target); os.IsNotExist(err) {
				return fmt.Errorf("❌ 错误: 目标不存在: %s", target)
			}

			fmt.Printf("🔗 创建符号链接: %s → %s\n", target, link)
			result := filesystem.CreateSymlink(target, link)
			if !result.Success {
				return fmt.Errorf("❌ 创建符号链接失败: %v\n\n💡 建议:\n  • 检查目标权限\n  • 确认链接路径是否正确", result.Error)
			}

			fmt.Printf("✅ %s\n", result.Message)
			return nil
		},
	}
	cmd.AddCommand(lnCmd)

	// 归档命令
	archiveCmd := &cobra.Command{
		Use:   "archive <source>... <destination> <format>",
		Short: "📦 创建归档文件",
		Long: `创建归档文件到指定位置

示例:
  clixgo fs archive /home/user /backup/user.tar.gz    # 创建归档文件
  clixgo fs archive /home/user /backup/user.zip         # 创建zip归档文件`,
		Args: cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			source := args[:len(args)-2]
			dest := args[len(args)-2]
			format := args[len(args)-1]

			// 验证参数
			if len(source) == 0 {
				return fmt.Errorf("❌ 错误: 源路径不能为空")
			}
			if strings.TrimSpace(dest) == "" {
				return fmt.Errorf("❌ 错误: 目标路径不能为空")
			}
			if strings.TrimSpace(format) == "" {
				return fmt.Errorf("❌ 错误: 归档格式不能为空")
			}

			// 检查源文件是否存在
			for _, path := range source {
				if _, err := os.Stat(path); os.IsNotExist(err) {
					return fmt.Errorf("❌ 错误: 源文件不存在: %s", path)
				}
			}

			fmt.Printf("📦 创建归档文件: %s → %s (%s)\n", strings.Join(source, " "), dest, format)
			result := filesystem.Archive(source, dest, format)
			if !result.Success {
				return fmt.Errorf("❌ 创建归档文件失败: %v\n\n💡 建议:\n  • 检查源文件权限\n  • 确认目标目录权限\n  • 检查磁盘空间是否足够", result.Error)
			}

			fmt.Printf("✅ %s\n", result.Message)
			return nil
		},
	}
	cmd.AddCommand(archiveCmd)

	// 解压命令
	extractCmd := &cobra.Command{
		Use:   "extract <archive> <destination>",
		Short: "🗜️  解压归档文件",
		Long: `解压归档文件到指定位置

示例:
  clixgo fs extract user.tar.gz /home/user    # 解压tar.gz归档文件
  clixgo fs extract user.zip /home/user        # 解压zip归档文件`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			archive := args[0]
			dest := args[1]

			// 验证参数
			if strings.TrimSpace(archive) == "" {
				return fmt.Errorf("❌ 错误: 归档文件路径不能为空")
			}
			if strings.TrimSpace(dest) == "" {
				return fmt.Errorf("❌ 错误: 目标路径不能为空")
			}

			// 检查归档文件是否存在
			if _, err := os.Stat(archive); os.IsNotExist(err) {
				return fmt.Errorf("❌ 错误: 归档文件不存在: %s", archive)
			}

			fmt.Printf("🗜️  解压归档文件: %s → %s\n", archive, dest)
			result := filesystem.Extract(archive, dest)
			if !result.Success {
				return fmt.Errorf("❌ 解压归档文件失败: %v\n\n💡 建议:\n  • 检查归档文件完整性\n  • 确认目标目录权限\n  • 检查磁盘空间是否足够", result.Error)
			}

			fmt.Printf("✅ %s\n", result.Message)
			return nil
		},
	}
	cmd.AddCommand(extractCmd)

	// 搜索文件命令
	findCmd := &cobra.Command{
		Use:   "find <directory> <pattern>",
		Short: "🔍 搜索文件",
		Long: `搜索指定目录下的文件和目录

示例:
  clixgo fs find /home/user txt          # 搜索txt文件
  clixgo fs find /home/user -c            # 区分大小写搜索
  clixgo fs find /home/user -r            # 递归搜索`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			pattern := args[1]

			// 验证参数
			if strings.TrimSpace(dir) == "" {
				return fmt.Errorf("❌ 错误: 目录路径不能为空")
			}
			if strings.TrimSpace(pattern) == "" {
				return fmt.Errorf("❌ 错误: 搜索模式不能为空")
			}

			// 检查目录是否存在
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				return fmt.Errorf("❌ 错误: 目录不存在: %s", dir)
			}

			caseSensitive, _ := cmd.Flags().GetBool("case-sensitive")
			recursive, _ := cmd.Flags().GetBool("recursive")

			fmt.Printf("🔍 搜索文件: %s → %s\n", dir, pattern)
			if caseSensitive {
				fmt.Println("🔄 区分大小写搜索已启用")
			}
			if recursive {
				fmt.Println("🔄 递归搜索已启用")
			}

			files, err := filesystem.SearchFiles(dir, pattern, caseSensitive)
			if err != nil {
				return fmt.Errorf("❌ 搜索文件失败: %v\n\n💡 建议:\n  • 检查搜索模式是否正确\n  • 确认目录权限\n  • 尝试使用绝对路径", err)
			}

			if len(files) == 0 {
				fmt.Println("ℹ️  没有找到匹配的文件")
				return nil
			}

			fmt.Printf("📊 总计 %d 个文件/目录\n\n", len(files))
			fmt.Printf("%-12s %-8s %-8s %10s %-20s %s\n", "权限", "所有者", "组", "大小", "修改时间", "名称")
			fmt.Println(strings.Repeat("-", 80))
			for _, file := range files {
				fmt.Printf("%-12s %-8s %-8s %10d %-20s %s\n",
					file.Permissions,
					file.Owner,
					file.Group,
					file.Size,
					file.ModTime.Format("2006-01-02 15:04:05"),
					file.Path)
			}

			return nil
		},
	}
	findCmd.Flags().BoolP("case-sensitive", "c", false, "区分大小写")
	findCmd.Flags().BoolP("recursive", "r", false, "递归搜索")
	cmd.AddCommand(findCmd)

	return cmd
}
