package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/history"
)

func main() {
	// 设置自定义历史文件路径
	customPath := "./test_history.json"
	history.SetHistoryFilePath(customPath)
	fmt.Printf("历史记录将保存到: %s\n", history.GetHistoryFilePath())

	// 创建并保存一个命令历史
	now := time.Now()
	cmd := &history.CommandHistory{
		Command:   "echo 'Hello, ClixGo!'",
		Status:    "success",
		Output:    "Hello, ClixGo!",
		StartTime: now.Add(-time.Second),
		EndTime:   now,
		Duration:  "1s",
	}

	if err := history.SaveHistory(cmd); err != nil {
		fmt.Printf("保存历史记录失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("成功保存命令历史")

	// 获取所有历史记录
	records, err := history.GetHistory()
	if err != nil {
		fmt.Printf("获取历史记录失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("找到 %d 条历史记录:\n", len(records))
	for i, record := range records {
		fmt.Printf("%d. 命令: %s, 状态: %s, 用时: %s\n",
			i+1, record.Command, record.Status, record.Duration)
	}

	// 获取最后一条历史记录
	lastCmd, err := history.GetLastHistory()
	if err != nil {
		fmt.Printf("获取最后一条历史记录失败: %v\n", err)
		os.Exit(1)
	}

	if lastCmd != nil {
		fmt.Printf("\n最后执行的命令: %s\n", lastCmd.Command)
		fmt.Printf("执行结果: %s\n", lastCmd.Output)
	}

	// 清除历史记录
	if err := history.ClearHistory(); err != nil {
		fmt.Printf("清除历史记录失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("已清除所有历史记录")

	// 确认清除成功
	records, _ = history.GetHistory()
	fmt.Printf("清除后的记录数: %d\n", len(records))

	// 清理
	os.Remove(customPath)
}
