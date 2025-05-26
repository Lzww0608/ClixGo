package main

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/creack/pty"
)

func main() {
	fmt.Println("=== 测试creack/pty库 ===")

	// 测试1: 执行简单命令
	fmt.Println("测试1: 执行 'echo hello world'")
	cmd1 := exec.Command("bash", "-c", "echo hello world")
	ptmx1, err := pty.Start(cmd1)
	if err != nil {
		fmt.Printf("启动命令失败: %v\n", err)
		return
	}
	defer ptmx1.Close()

	// 读取输出
	buffer := make([]byte, 1024)
	n, err := ptmx1.Read(buffer)
	if err != nil {
		fmt.Printf("读取输出失败: %v\n", err)
	} else {
		fmt.Printf("输出: %s", string(buffer[:n]))
	}

	// 测试2: 启动交互式shell
	fmt.Println("\n测试2: 启动交互式shell")
	cmd2 := exec.Command("bash")
	ptmx2, err := pty.Start(cmd2)
	if err != nil {
		fmt.Printf("启动shell失败: %v\n", err)
		return
	}
	defer ptmx2.Close()

	// 发送命令
	_, err = ptmx2.Write([]byte("echo 'shell test'\n"))
	if err != nil {
		fmt.Printf("发送命令失败: %v\n", err)
		return
	}

	// 等待并读取输出
	time.Sleep(100 * time.Millisecond)
	n, err = ptmx2.Read(buffer)
	if err != nil {
		fmt.Printf("读取shell输出失败: %v\n", err)
	} else {
		fmt.Printf("Shell输出: %s", string(buffer[:n]))
	}

	// 发送exit
	ptmx2.Write([]byte("exit\n"))

	fmt.Println("测试完成")
}
