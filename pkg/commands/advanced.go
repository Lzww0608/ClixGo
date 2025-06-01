/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-1 22:07:14
* @Description: 高级命令处理功能的实现，包括AWK、grep、sed等文本处理命令
 */

package commands

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/Lzww0608/ClixGo/pkg/errors"
	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/utils"
	"go.uber.org/zap"
)

// AWKCommand 执行AWK命令
func AWKCommand(input string, pattern string) (string, error) {
	// 参数验证
	if err := utils.Validation.RequireNonEmpty(input, "input"); err != nil {
		return "", err
	}
	if err := utils.Validation.RequireNonEmpty(pattern, "pattern"); err != nil {
		return "", err
	}

	cmd := exec.Command("awk", pattern)
	cmd.Stdin = strings.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		logger.Error("AWK命令执行失败",
			zap.String("pattern", pattern),
			zap.Error(err))
		return "", errors.Wrap(err, errors.ErrCodeCommandExecution, "AWK命令执行失败").
			WithDetails("awk pattern: " + pattern)
	}

	return out.String(), nil
}

// GrepCommand 执行grep命令
func GrepCommand(input string, pattern string) (string, error) {
	// 参数验证
	if err := utils.Validation.RequireNonEmpty(input, "input"); err != nil {
		return "", err
	}
	if err := utils.Validation.RequireNonEmpty(pattern, "pattern"); err != nil {
		return "", err
	}

	cmd := exec.Command("grep", pattern)
	cmd.Stdin = strings.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		logger.Error("grep命令执行失败",
			zap.String("pattern", pattern),
			zap.Error(err))
		return "", errors.Wrap(err, errors.ErrCodeCommandExecution, "grep命令执行失败").
			WithDetails("grep pattern: " + pattern)
	}

	return out.String(), nil
}

// SedCommand 执行sed命令
func SedCommand(input string, pattern string) (string, error) {
	// 参数验证
	if err := utils.Validation.RequireNonEmpty(input, "input"); err != nil {
		return "", err
	}
	if err := utils.Validation.RequireNonEmpty(pattern, "pattern"); err != nil {
		return "", err
	}

	cmd := exec.Command("sed", pattern)
	cmd.Stdin = strings.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		logger.Error("sed命令执行失败",
			zap.String("pattern", pattern),
			zap.Error(err))
		return "", errors.Wrap(err, errors.ErrCodeCommandExecution, "sed命令执行失败").
			WithDetails("sed pattern: " + pattern)
	}

	return out.String(), nil
}

// PipeCommands 执行管道命令
func PipeCommands(commands []string) (string, error) {
	if len(commands) == 0 {
		return "", errors.New(errors.ErrCodeInvalidParam, "没有提供命令").
			WithDetails("commands slice is empty")
	}

	var lastOutput bytes.Buffer
	var err error

	for i, command := range commands {
		parts := strings.Fields(command)
		if len(parts) == 0 {
			return "", errors.New(errors.ErrCodeInvalidParam, "空命令").
				WithDetails("command at index " + utils.Strings.DefaultIfEmpty(string(rune(i)), "unknown") + " is empty")
		}

		cmd := exec.Command(parts[0], parts[1:]...)

		if i > 0 {
			// 只有从第二个命令开始才使用前一个命令的输出作为输入
			cmd.Stdin = strings.NewReader(lastOutput.String())
		}

		lastOutput.Reset()
		cmd.Stdout = &lastOutput

		err = cmd.Run()
		if err != nil {
			logger.Error("管道命令执行失败",
				zap.String("command", command),
				zap.Int("index", i),
				zap.Error(err))
			return "", errors.Wrap(err, errors.ErrCodeCommandExecution, "管道命令执行失败").
				WithDetails("失败的命令: " + command)
		}
	}

	return lastOutput.String(), nil
}
