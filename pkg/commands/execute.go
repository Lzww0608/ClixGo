package commands

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"gocli/pkg/alias"
	"gocli/pkg/config"
	"gocli/pkg/history"
	"gocli/pkg/logger"
)

// ExecuteCommand 执行单个命令
func ExecuteCommand(command string) error {
	// 扩展别名
	expandedCommand := alias.ExpandCommand(command)
	if expandedCommand != command {
		logger.Info("扩展别名", logger.Log.String("original", command), logger.Log.String("expanded", expandedCommand))
		command = expandedCommand
	}

	startTime := time.Now()
	cmdHistory := &history.CommandHistory{
		Command:   command,
		StartTime: startTime,
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		cmdHistory.Status = "failed"
		cmdHistory.EndTime = time.Now()
		cmdHistory.Duration = cmdHistory.EndTime.Sub(startTime).String()
		history.SaveHistory(cmdHistory)
		return fmt.Errorf("空命令")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.AppConfig.Commands.Timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()

	cmdHistory.EndTime = time.Now()
	cmdHistory.Duration = cmdHistory.EndTime.Sub(startTime).String()
	cmdHistory.Output = string(output)

	if err != nil {
		cmdHistory.Status = "failed"
		logger.Error("命令执行失败", logger.Log.Error(err))
	} else {
		cmdHistory.Status = "success"
		logger.Info("命令执行成功", logger.Log.String("output", string(output)))
	}

	if err := history.SaveHistory(cmdHistory); err != nil {
		logger.Error("保存命令历史失败", logger.Log.Error(err))
	}

	if err != nil {
		return fmt.Errorf("执行命令失败: %v\n输出: %s", err, string(output))
	}
	fmt.Printf("命令输出: %s\n", string(output))
	return nil
}

// ExecuteCommandsSequentially 串行执行多个命令
func ExecuteCommandsSequentially(commands []string) error {
	for _, cmd := range commands {
		if err := ExecuteCommand(cmd); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteCommandsParallel 并行执行多个命令
func ExecuteCommandsParallel(commands []string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(commands))

	for _, cmd := range commands {
		wg.Add(1)
		go func(command string) {
			defer wg.Done()
			if err := ExecuteCommand(command); err != nil {
				errChan <- err
			}
		}(cmd)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}
