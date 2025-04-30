package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gocli/pkg/logger"
)

type CommandHistory struct {
	Command   string    `json:"command"`
	Status    string    `json:"status"`
	Output    string    `json:"output"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  string    `json:"duration"`
}

var historyFile = filepath.Join(os.Getenv("HOME"), ".gocli_history.json")

func SaveHistory(cmd *CommandHistory) error {
	// 读取现有历史记录
	var history []CommandHistory
	if _, err := os.Stat(historyFile); err == nil {
		data, err := os.ReadFile(historyFile)
		if err != nil {
			return fmt.Errorf("读取历史记录失败: %v", err)
		}
		if err := json.Unmarshal(data, &history); err != nil {
			return fmt.Errorf("解析历史记录失败: %v", err)
		}
	}

	// 添加新记录
	history = append(history, *cmd)

	// 只保留最近的100条记录
	if len(history) > 100 {
		history = history[len(history)-100:]
	}

	// 保存历史记录
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化历史记录失败: %v", err)
	}

	if err := os.WriteFile(historyFile, data, 0644); err != nil {
		return fmt.Errorf("保存历史记录失败: %v", err)
	}

	return nil
}

func GetHistory() ([]CommandHistory, error) {
	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		return []CommandHistory{}, nil
	}

	data, err := os.ReadFile(historyFile)
	if err != nil {
		return nil, fmt.Errorf("读取历史记录失败: %v", err)
	}

	var history []CommandHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("解析历史记录失败: %v", err)
	}

	return history, nil
}

func ClearHistory() error {
	if err := os.Remove(historyFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清除历史记录失败: %v", err)
	}
	return nil
} 