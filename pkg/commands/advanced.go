package commands

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"go.uber.org/zap"
)

// AWKCommand 执行AWK命令
func AWKCommand(input string, pattern string) (string, error) {
	cmd := exec.Command("awk", pattern)
	cmd.Stdin = strings.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		if logger.Log != nil {
			logger.Error("AWK命令执行失败", zap.Error(err))
		}
		return "", err
	}

	return out.String(), nil
}

// GrepCommand 执行grep命令
func GrepCommand(input string, pattern string) (string, error) {
	cmd := exec.Command("grep", pattern)
	cmd.Stdin = strings.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		if logger.Log != nil {
			logger.Error("grep命令执行失败", zap.Error(err))
		}
		return "", err
	}

	return out.String(), nil
}

// SedCommand 执行sed命令
func SedCommand(input string, pattern string) (string, error) {
	cmd := exec.Command("sed", pattern)
	cmd.Stdin = strings.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		if logger.Log != nil {
			logger.Error("sed命令执行失败", zap.Error(err))
		}
		return "", err
	}

	return out.String(), nil
}

// PipeCommands 执行管道命令
func PipeCommands(commands []string) (string, error) {
	if len(commands) == 0 {
		return "", fmt.Errorf("没有提供命令")
	}

	var lastOutput bytes.Buffer
	var err error

	for i, command := range commands {
		parts := strings.Fields(command)
		if len(parts) == 0 {
			return "", fmt.Errorf("空命令")
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
			if logger.Log != nil {
				logger.Error("管道命令执行失败", zap.Error(err), zap.String("command", command))
			}
			return "", err
		}
	}

	return lastOutput.String(), nil
}
