package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Lzww0608/ClixGo/pkg/commands"
	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/utils"
)

func NewSequentialCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sequential",
		Short: "串行执行多个命令",
		Long:  `按顺序执行多个命令，用分号(;)分隔`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			commandList := utils.SplitCommands(args[0])
			if err := utils.ValidateCommands(commandList); err != nil {
				return err
			}
			logger.Info("开始串行执行命令", logger.Log.String("commands", args[0]))
			return commands.ExecuteCommandsSequentially(commandList)
		},
	}
}

func NewParallelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "parallel",
		Short: "并行执行多个命令",
		Long:  `同时执行多个命令，用分号(;)分隔`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			commandList := utils.SplitCommands(args[0])
			if err := utils.ValidateCommands(commandList); err != nil {
				return err
			}
			logger.Info("开始并行执行命令", logger.Log.String("commands", args[0]))
			return commands.ExecuteCommandsParallel(commandList)
		},
	}
}

func NewAWKCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "awk",
		Short: "执行AWK命令",
		Long:  `执行AWK命令处理输入文本`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := args[0]
			pattern := args[1]
			logger.Info("执行AWK命令", logger.Log.String("pattern", pattern))
			result, err := commands.AWKCommand(input, pattern)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
}

func NewGrepCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "grep",
		Short: "执行grep命令",
		Long:  `执行grep命令搜索文本`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := args[0]
			pattern := args[1]
			logger.Info("执行grep命令", logger.Log.String("pattern", pattern))
			result, err := commands.GrepCommand(input, pattern)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
}

func NewSedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sed",
		Short: "执行sed命令",
		Long:  `执行sed命令处理文本`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := args[0]
			pattern := args[1]
			logger.Info("执行sed命令", logger.Log.String("pattern", pattern))
			result, err := commands.SedCommand(input, pattern)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
}

func NewPipeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pipe",
		Short: "执行管道命令",
		Long:  `执行多个命令的管道操作，用分号(;)分隔`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			commandList := utils.SplitCommands(args[0])
			if err := utils.ValidateCommands(commandList); err != nil {
				return err
			}
			logger.Info("开始执行管道命令", logger.Log.String("commands", args[0]))
			result, err := commands.PipeCommands(commandList)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
} 