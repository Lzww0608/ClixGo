/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-12 10:30:00
* @Description: 高级命令处理功能的实现，包括AWK、grep、sed等文本处理命令 - 质量优化版
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
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// AWK语法错误通常是退出代码2
			if exitError.ExitCode() == 2 {
				errorMsg := strings.TrimSpace(stderr.String())
				if errorMsg == "" {
					errorMsg = "AWK语法错误"
				}
				logger.Error("AWK命令语法错误",
					zap.String("pattern", pattern),
					zap.String("stderr", errorMsg),
					zap.Int("exit_code", exitError.ExitCode()))
				return "", errors.New(errors.ErrCodeInvalidParam, "AWK语法错误: "+errorMsg).
					WithDetails("awk pattern: " + pattern)
			}
		}
		// 其他错误情况
		logger.Error("AWK命令执行失败",
			zap.String("pattern", pattern),
			zap.String("stderr", stderr.String()),
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
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// grep退出代码1表示没有找到匹配，这是正常情况
			if exitError.ExitCode() == 1 {
				// 没有匹配不是错误，返回空结果
				return "", nil
			}
			// grep退出代码2通常表示语法错误或其他错误
			if exitError.ExitCode() == 2 {
				errorMsg := strings.TrimSpace(stderr.String())
				if errorMsg == "" {
					errorMsg = "grep语法错误或文件访问错误"
				}
				logger.Error("grep命令语法错误",
					zap.String("pattern", pattern),
					zap.String("stderr", errorMsg),
					zap.Int("exit_code", exitError.ExitCode()))
				return "", errors.New(errors.ErrCodeInvalidParam, "grep语法错误: "+errorMsg).
					WithDetails("grep pattern: " + pattern)
			}
		}
		// 其他错误情况
		logger.Error("grep命令执行失败",
			zap.String("pattern", pattern),
			zap.String("stderr", stderr.String()),
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
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// sed退出代码1通常表示语法错误
			if exitError.ExitCode() == 1 {
				errorMsg := strings.TrimSpace(stderr.String())
				if errorMsg == "" {
					errorMsg = "sed语法错误"
				}
				logger.Error("sed命令语法错误",
					zap.String("pattern", pattern),
					zap.String("stderr", errorMsg),
					zap.Int("exit_code", exitError.ExitCode()))
				return "", errors.New(errors.ErrCodeInvalidParam, "sed语法错误: "+errorMsg).
					WithDetails("sed pattern: " + pattern)
			}
		}
		// 其他错误情况
		logger.Error("sed命令执行失败",
			zap.String("pattern", pattern),
			zap.String("stderr", stderr.String()),
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

		// 设置stderr捕获，提供更好的错误信息
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if i > 0 {
			// 只有从第二个命令开始才使用前一个命令的输出作为输入
			cmd.Stdin = strings.NewReader(lastOutput.String())
		}

		lastOutput.Reset()
		cmd.Stdout = &lastOutput

		err = cmd.Run()
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				errorMsg := strings.TrimSpace(stderr.String())
				if errorMsg == "" {
					errorMsg = "命令执行失败"
				}
				logger.Error("管道命令执行失败",
					zap.String("command", command),
					zap.Int("index", i),
					zap.Int("exit_code", exitError.ExitCode()),
					zap.String("stderr", errorMsg),
					zap.Error(err))
				return "", errors.Wrap(err, errors.ErrCodeCommandExecution, "管道命令执行失败: "+errorMsg).
					WithDetails("失败的命令: " + command + " (第" + string(rune(i+1)) + "个)")
			} else {
				logger.Error("管道命令执行失败",
					zap.String("command", command),
					zap.Int("index", i),
					zap.Error(err))
				return "", errors.Wrap(err, errors.ErrCodeCommandExecution, "管道命令执行失败").
					WithDetails("失败的命令: " + command)
			}
		}
	}

	return lastOutput.String(), nil
}
