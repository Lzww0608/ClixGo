/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 主入口
 */

package main

import (
	"fmt"
	"os"

	"github.com/Lzww0608/ClixGo/cmd/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Printf("执行命令失败: %v\n", err)
		os.Exit(1)
	}
}
