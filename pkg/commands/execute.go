package commands

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/alias"
	"github.com/Lzww0608/ClixGo/pkg/history"
	"github.com/Lzww0608/ClixGo/pkg/logger"
	"go.uber.org/zap"
)

// 默认超时时间
const defaultCmdTimeout = 30 * time.Second

// ExecuteCommand 执行单个命令
func ExecuteCommand(command string) error {
	// 扩展别名
	expandedCommand := alias.ExpandCommand(command)
	if expandedCommand != command {
		logger.Info("扩展别名",
			zap.String("original", command),
			zap.String("expanded", expandedCommand))
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

	// 使用默认超时时间
	ctx, cancel := context.WithTimeout(context.Background(), defaultCmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()

	cmdHistory.EndTime = time.Now()
	cmdHistory.Duration = cmdHistory.EndTime.Sub(startTime).String()
	cmdHistory.Output = string(output)

	if err != nil {
		cmdHistory.Status = "failed"
		logger.Error("命令执行失败", zap.Error(err))
	} else {
		cmdHistory.Status = "success"
		logger.Info("命令执行成功", zap.String("output", string(output)))
	}

	if err := history.SaveHistory(cmdHistory); err != nil {
		logger.Error("保存命令历史失败", zap.Error(err))
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

	// 如果没有命令要执行，直接返回成功
	if len(commands) == 0 {
		return nil
	}

	for _, cmd := range commands {
		wg.Add(1)
		go func(command string) {
			defer wg.Done()
			if err := ExecuteCommand(command); err != nil {
				// 使用非阻塞发送避免死锁
				select {
				case errChan <- err:
					// 成功发送错误
				default:
					// 通道已满，记录日志但不阻塞
					logger.Error("无法将错误发送到错误通道，可能通道已满",
						zap.String("command", command),
						zap.Error(err))
				}
			}
		}(cmd)
	}

	// 使用额外的goroutine关闭通道，确保所有任务完成后才关闭
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// 收集第一个错误并返回
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}
