/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-15 21:31:25
* @Description: 安全功能命令 (已移除，保留接口兼容性)
 */

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewSecurityCmd() *cobra.Command {
	securityCmd := &cobra.Command{
		Use:   "security",
		Short: "安全功能 (已移除)",
		Long:  "安全功能已从核心模块中移除，专注于终端复用功能",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("安全功能已从核心模块中移除")
			fmt.Println("如需安全功能，请使用专门的安全工具")
		},
	}

	return securityCmd
}
