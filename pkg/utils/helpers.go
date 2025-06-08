/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-8 18:30:00
* @Description: 通用工具函数和辅助方法的实现
 */

package utils

import (
	"fmt"
	"strings"
)

// SplitCommands 将命令字符串分割成命令数组
//
// 该函数将以分号分隔的命令字符串解析为独立的命令列表，
// 并自动清理每个命令的首尾空白字符
//
// 参数:
//   - commandString: 包含多个命令的字符串，命令之间用分号分隔
//
// 返回:
//   - []string: 清理后的命令列表
//
// 示例:
//
//	input: "ls -la; pwd; echo hello"
//	output: ["ls -la", "pwd", "echo hello"]
func SplitCommands(commandString string) []string {
	commandList := strings.Split(commandString, ";")
	for commandIndex := range commandList {
		commandList[commandIndex] = strings.TrimSpace(commandList[commandIndex])
	}
	return commandList
}

// ValidateCommands 验证命令数组是否有效
//
// 该函数检查命令列表中是否存在空命令或无效命令，
// 确保所有命令都包含有效的可执行内容
//
// 参数:
//   - commands: 待验证的命令列表
//
// 返回:
//   - error: 验证错误，nil表示所有命令都有效
//
// 验证规则:
//   - 命令不能为空字符串
//   - 命令不能仅包含空白字符
func ValidateCommands(commands []string) error {
	for _, command := range commands {
		if strings.TrimSpace(command) == "" {
			return fmt.Errorf("发现空命令")
		}
	}
	return nil
}
