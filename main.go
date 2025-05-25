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
