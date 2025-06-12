/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-12 10:45:00
* @Description: 命令执行相关的CLI命令定义，包含串行、并行、AWK、grep、sed、pipe等命令
 */

package cli

import (
	"fmt"
	"strings"

	"github.com/Lzww0608/ClixGo/pkg/commands"
	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/utils"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func NewSequentialCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sequential <commands>",
		Short: "🔗 串行执行多个命令",
		Long: `按顺序执行多个命令，用分号(;)分隔

示例:
  clixgo sequential "echo hello; echo world; ls -la"
  clixgo sequential "mkdir test; cd test; touch file.txt"

注意: 如果任何一个命令失败，后续命令将不会执行`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(args[0]) == "" {
				return fmt.Errorf("❌ 错误: 命令不能为空\n\n💡 提示: 请提供要执行的命令，例如:\n  clixgo sequential \"echo hello; echo world\"")
			}

			commandList := utils.SplitCommands(args[0])
			if len(commandList) == 0 {
				return fmt.Errorf("❌ 错误: 没有找到有效的命令\n\n💡 提示: 请确保命令格式正确，用分号分隔多个命令")
			}

			if err := utils.ValidateCommands(commandList); err != nil {
				return fmt.Errorf("❌ 命令验证失败: %v\n\n💡 提示: 请检查命令语法是否正确", err)
			}

			logger.Info("🚀 开始串行执行命令",
				zap.String("commands", args[0]),
				zap.Int("command_count", len(commandList)))

			fmt.Printf("🔗 串行执行 %d 个命令...\n", len(commandList))
			for i, cmd := range commandList {
				fmt.Printf("  [%d/%d] %s\n", i+1, len(commandList), cmd)
			}
			fmt.Println()

			if err := commands.ExecuteCommandsSequentially(commandList); err != nil {
				return fmt.Errorf("❌ 串行执行失败: %v\n\n💡 建议:\n  • 检查命令是否存在\n  • 确认文件路径是否正确\n  • 验证权限是否足够", err)
			}

			fmt.Println("✅ 所有命令执行完成")
			return nil
		},
	}
}

func NewParallelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "parallel <commands>",
		Short: "⚡ 并行执行多个命令",
		Long: `同时执行多个命令，用分号(;)分隔

示例:
  clixgo parallel "ping google.com; ping baidu.com; ping github.com"
  clixgo parallel "ls /tmp; ps aux; df -h"

注意: 所有命令会同时启动，适合独立的操作`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(args[0]) == "" {
				return fmt.Errorf("❌ 错误: 命令不能为空\n\n💡 提示: 请提供要执行的命令，例如:\n  clixgo parallel \"ping google.com; ping baidu.com\"")
			}

			commandList := utils.SplitCommands(args[0])
			if len(commandList) == 0 {
				return fmt.Errorf("❌ 错误: 没有找到有效的命令\n\n💡 提示: 请确保命令格式正确，用分号分隔多个命令")
			}

			if len(commandList) > 10 {
				return fmt.Errorf("❌ 错误: 并行命令数量过多 (%d > 10)\n\n💡 提示: 为了系统稳定性，建议一次并行执行不超过10个命令", len(commandList))
			}

			if err := utils.ValidateCommands(commandList); err != nil {
				return fmt.Errorf("❌ 命令验证失败: %v\n\n💡 提示: 请检查命令语法是否正确", err)
			}

			logger.Info("⚡ 开始并行执行命令",
				zap.String("commands", args[0]),
				zap.Int("command_count", len(commandList)))

			fmt.Printf("⚡ 并行执行 %d 个命令...\n", len(commandList))
			for i, cmd := range commandList {
				fmt.Printf("  [%d] %s\n", i+1, cmd)
			}
			fmt.Println()

			if err := commands.ExecuteCommandsParallel(commandList); err != nil {
				return fmt.Errorf("❌ 并行执行失败: %v\n\n💡 建议:\n  • 检查命令是否存在\n  • 确认是否有资源冲突\n  • 减少并行命令数量", err)
			}

			fmt.Println("✅ 所有命令执行完成")
			return nil
		},
	}
}

func NewAWKCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "awk <input> <pattern>",
		Short: "🔍 执行AWK文本处理命令",
		Long: `使用AWK模式处理输入文本

示例:
  clixgo awk "hello world" '{print $1}'     # 输出第一个字段
  clixgo awk "1 2 3" '{sum+=$1} END{print sum}'  # 计算总和
  clixgo awk "line1\nline2" 'NR==1'         # 输出第一行

常用模式:
  {print $1}        - 打印第一个字段
  {print NF}        - 打印字段数量
  {print NR}        - 打印行号
  /pattern/         - 匹配包含pattern的行`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := args[0]
			pattern := args[1]

			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("❌ 错误: 输入文本不能为空\n\n💡 提示: 请提供要处理的文本")
			}

			if strings.TrimSpace(pattern) == "" {
				return fmt.Errorf("❌ 错误: AWK模式不能为空\n\n💡 提示: 请提供AWK处理模式，例如: '{print $1}'")
			}

			// 检查常见的AWK语法错误
			if strings.Count(pattern, "{") != strings.Count(pattern, "}") {
				return fmt.Errorf("❌ 错误: AWK模式语法错误 - 大括号不匹配\n\n💡 提示: 请检查 { 和 } 是否配对")
			}

			logger.Info("🔍 执行AWK命令",
				zap.String("pattern", pattern),
				zap.Int("input_length", len(input)))

			fmt.Printf("🔍 AWK处理: %s\n", pattern)
			result, err := commands.AWKCommand(input, pattern)
			if err != nil {
				// 提供更友好的错误信息
				if strings.Contains(err.Error(), "syntax error") {
					return fmt.Errorf("❌ AWK语法错误: %v\n\n💡 常见问题:\n  • 检查大括号是否配对\n  • 确认字段引用格式 ($1, $2, ...)\n  • 验证正则表达式语法", err)
				}
				return fmt.Errorf("❌ AWK执行失败: %v\n\n💡 建议:\n  • 检查输入文本格式\n  • 验证AWK模式语法\n  • 尝试简化处理逻辑", err)
			}

			if result == "" {
				fmt.Println("ℹ️  AWK处理完成，但没有输出结果")
			} else {
				fmt.Println(result)
			}
			return nil
		},
	}
}

func NewGrepCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "grep <input> <pattern>",
		Short: "🔎 执行grep文本搜索命令",
		Long: `在输入文本中搜索匹配的模式

示例:
  clixgo grep "hello world" "hello"        # 搜索包含hello的行
  clixgo grep "line1\nline2" "line"        # 搜索包含line的行
  clixgo grep "test123" "[0-9]+"           # 使用正则表达式搜索数字

支持的模式:
  • 普通文本匹配
  • 正则表达式 (使用 -E 标志)
  • 大小写不敏感 (使用 -i 标志)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := args[0]
			pattern := args[1]

			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("❌ 错误: 输入文本不能为空\n\n💡 提示: 请提供要搜索的文本")
			}

			if strings.TrimSpace(pattern) == "" {
				return fmt.Errorf("❌ 错误: 搜索模式不能为空\n\n💡 提示: 请提供要搜索的模式，例如: \"hello\"")
			}

			logger.Info("🔎 执行grep命令",
				zap.String("pattern", pattern),
				zap.Int("input_length", len(input)))

			fmt.Printf("🔎 搜索模式: %s\n", pattern)
			result, err := commands.GrepCommand(input, pattern)
			if err != nil {
				// 提供更友好的错误信息
				if strings.Contains(err.Error(), "invalid regular expression") {
					return fmt.Errorf("❌ 正则表达式错误: %v\n\n💡 建议:\n  • 检查正则表达式语法\n  • 尝试转义特殊字符\n  • 使用简单的文本匹配", err)
				}
				return fmt.Errorf("❌ grep执行失败: %v\n\n💡 建议:\n  • 检查搜索模式是否正确\n  • 确认输入文本格式\n  • 尝试简化搜索条件", err)
			}

			if result == "" {
				fmt.Printf("ℹ️  没有找到匹配 '%s' 的内容\n", pattern)
			} else {
				fmt.Println(result)
			}
			return nil
		},
	}
}

func NewSedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sed <input> <pattern>",
		Short: "✏️  执行sed文本编辑命令",
		Long: `使用sed模式编辑输入文本

示例:
  clixgo sed "hello world" "s/hello/hi/"    # 替换hello为hi
  clixgo sed "line1\nline2" "s/line/LINE/"  # 替换line为LINE
  clixgo sed "test123" "s/[0-9]/X/g"        # 替换所有数字为X

常用模式:
  s/old/new/     - 替换第一个匹配
  s/old/new/g    - 替换所有匹配
  /pattern/d     - 删除匹配的行
  /pattern/p     - 打印匹配的行`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := args[0]
			pattern := args[1]

			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("❌ 错误: 输入文本不能为空\n\n💡 提示: 请提供要编辑的文本")
			}

			if strings.TrimSpace(pattern) == "" {
				return fmt.Errorf("❌ 错误: sed模式不能为空\n\n💡 提示: 请提供sed编辑模式，例如: \"s/old/new/\"")
			}

			// 检查常见的sed语法错误
			if strings.HasPrefix(pattern, "s/") && strings.Count(pattern, "/") < 3 {
				return fmt.Errorf("❌ 错误: sed替换模式语法错误\n\n💡 提示: 替换模式格式应为 s/old/new/ 或 s/old/new/g")
			}

			logger.Info("✏️  执行sed命令",
				zap.String("pattern", pattern),
				zap.Int("input_length", len(input)))

			fmt.Printf("✏️  sed编辑: %s\n", pattern)
			result, err := commands.SedCommand(input, pattern)
			if err != nil {
				// 提供更友好的错误信息
				if strings.Contains(err.Error(), "unterminated") {
					return fmt.Errorf("❌ sed语法错误: %v\n\n💡 常见问题:\n  • 检查替换模式格式: s/old/new/\n  • 确认分隔符是否正确\n  • 验证正则表达式语法", err)
				}
				return fmt.Errorf("❌ sed执行失败: %v\n\n💡 建议:\n  • 检查sed模式语法\n  • 确认输入文本格式\n  • 尝试简化编辑操作", err)
			}

			if result == input {
				fmt.Println("ℹ️  文本没有变化")
			}
			fmt.Println(result)
			return nil
		},
	}
}

func NewPipeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pipe <commands>",
		Short: "🔄 执行管道命令",
		Long: `执行多个命令的管道操作，前一个命令的输出作为后一个命令的输入

示例:
  clixgo pipe "echo hello; grep h; sed s/h/H/"
  clixgo pipe "ls -la; grep txt; wc -l"

注意: 命令之间通过管道连接，数据流式传递`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(args[0]) == "" {
				return fmt.Errorf("❌ 错误: 管道命令不能为空\n\n💡 提示: 请提供要执行的管道命令，例如:\n  clixgo pipe \"echo hello; grep h\"")
			}

			commandList := utils.SplitCommands(args[0])
			if len(commandList) == 0 {
				return fmt.Errorf("❌ 错误: 没有找到有效的命令\n\n💡 提示: 请确保命令格式正确，用分号分隔多个命令")
			}

			if len(commandList) < 2 {
				return fmt.Errorf("❌ 错误: 管道操作至少需要2个命令\n\n💡 提示: 管道用于连接多个命令，例如: \"echo hello; grep h\"")
			}

			if err := utils.ValidateCommands(commandList); err != nil {
				return fmt.Errorf("❌ 命令验证失败: %v\n\n💡 提示: 请检查命令语法是否正确", err)
			}

			logger.Info("🔄 开始执行管道命令",
				zap.String("commands", args[0]),
				zap.Int("command_count", len(commandList)))

			fmt.Printf("🔄 管道执行 %d 个命令...\n", len(commandList))
			for i, cmd := range commandList {
				if i == 0 {
					fmt.Printf("  📥 输入: %s\n", cmd)
				} else if i == len(commandList)-1 {
					fmt.Printf("  📤 输出: %s\n", cmd)
				} else {
					fmt.Printf("  🔄 处理: %s\n", cmd)
				}
			}
			fmt.Println()

			result, err := commands.PipeCommands(commandList)
			if err != nil {
				return fmt.Errorf("❌ 管道执行失败: %v\n\n💡 建议:\n  • 检查每个命令是否正确\n  • 确认命令间的数据兼容性\n  • 验证管道连接顺序", err)
			}

			if result == "" {
				fmt.Println("ℹ️  管道处理完成，但没有输出结果")
			} else {
				fmt.Println(result)
			}
			return nil
		},
	}
}
