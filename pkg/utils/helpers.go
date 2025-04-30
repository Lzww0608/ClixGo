package utils

import (
	"fmt"
	"strings"
)

// SplitCommands 将命令字符串分割成命令数组
func SplitCommands(commandString string) []string {
	commands := strings.Split(commandString, ";")
	for i := range commands {
		commands[i] = strings.TrimSpace(commands[i])
	}
	return commands
}

// ValidateCommands 验证命令数组是否有效
func ValidateCommands(commands []string) error {
	for _, cmd := range commands {
		if strings.TrimSpace(cmd) == "" {
			return fmt.Errorf("发现空命令")
		}
	}
	return nil
}
