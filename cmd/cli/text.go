package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"gocli/pkg/logger"
	"gocli/pkg/text"
)

func NewTextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "text",
		Short: "文本处理工具",
		Long:  `提供各种文本处理功能，如排序、统计、查找等`,
	}

	// 排序命令
	cmd.AddCommand(&cobra.Command{
		Use:   "sort",
		Short: "对文本行进行排序",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reverse, _ := cmd.Flags().GetBool("reverse")
			result, err := text.SortLines(args[0], reverse)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})
	cmd.Commands()[0].Flags().BoolP("reverse", "r", false, "反向排序")

	// 统计命令
	cmd.AddCommand(&cobra.Command{
		Use:   "count",
		Short: "统计文本信息",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			words, err := text.CountWords(args[0])
			if err != nil {
				return err
			}
			lines, err := text.CountLines(args[0])
			if err != nil {
				return err
			}
			chars, err := text.CountChars(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("单词数: %d\n行数: %d\n字符数: %d\n", words, lines, chars)
			return nil
		},
	})

	// 查找命令
	cmd.AddCommand(&cobra.Command{
		Use:   "find",
		Short: "查找匹配模式的行",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			matches, err := text.FindPattern(args[0], args[1])
			if err != nil {
				return err
			}
			for _, match := range matches {
				fmt.Println(match)
			}
			return nil
		},
	})

	// 替换命令
	cmd.AddCommand(&cobra.Command{
		Use:   "replace",
		Short: "替换匹配模式的内容",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := text.ReplacePattern(args[0], args[1], args[2])
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	// 去重命令
	cmd.AddCommand(&cobra.Command{
		Use:   "dedup",
		Short: "移除重复行",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := text.RemoveDuplicates(args[0])
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	})

	// 大小写转换命令
	cmd.AddCommand(&cobra.Command{
		Use:   "case",
		Short: "转换大小写",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			toUpper, _ := cmd.Flags().GetBool("upper")
			result := text.ConvertCase(args[0], toUpper)
			fmt.Println(result)
			return nil
		},
	})
	cmd.Commands()[5].Flags().BoolP("upper", "u", false, "转换为大写")

	// 提取命令
	cmd.AddCommand(&cobra.Command{
		Use:   "extract",
		Short: "提取特定内容",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var results []string
			var err error

			switch args[0] {
			case "numbers":
				results, err = text.ExtractNumbers(args[1])
			case "emails":
				results, err = text.ExtractEmails(args[1])
			case "urls":
				results, err = text.ExtractURLs(args[1])
			default:
				return fmt.Errorf("不支持的提取类型: %s", args[0])
			}

			if err != nil {
				return err
			}

			for _, result := range results {
				fmt.Println(result)
			}
			return nil
		},
	})

	// JSON格式化命令
	cmd.AddCommand(&cobra.Command{
		Use:   "json",
		Short: "JSON处理",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "format":
				result, err := text.FormatJSON(args[1])
				if err != nil {
					return err
				}
				fmt.Println(result)
			case "validate":
				err := text.ValidateJSON(args[1])
				if err != nil {
					return fmt.Errorf("无效的JSON: %v", err)
				}
				fmt.Println("JSON格式有效")
			default:
				return fmt.Errorf("不支持的JSON操作: %s", args[0])
			}
			return nil
		},
	})

	return cmd
} 