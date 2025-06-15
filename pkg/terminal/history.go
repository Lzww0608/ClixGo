/*
* @Author: Lzww0608
* @Date: 2025-06-14 11:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-14 11:00:00
* @Description: 命令历史记录管理功能 (从pkg/history迁移到terminal)
 */

package terminal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type CommandHistory struct {
	Command   string    `json:"command"`
	Status    string    `json:"status"`
	Output    string    `json:"output"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  string    `json:"duration"`
}

var (
	historyFile  = filepath.Join(os.Getenv("HOME"), ".clixgo", "history.json")
	historyMutex = &sync.Mutex{} // 添加互斥锁以保护文件访问
)

// SetHistoryFilePath 允许自定义历史文件的路径
func SetHistoryFilePath(path string) {
	historyMutex.Lock()
	defer historyMutex.Unlock()
	historyFile = path
}

// GetHistoryFilePath 返回当前历史文件的路径
func GetHistoryFilePath() string {
	historyMutex.Lock()
	defer historyMutex.Unlock()
	return historyFile
}

func SaveHistory(cmd *CommandHistory) error {
	historyMutex.Lock()
	defer historyMutex.Unlock()

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(historyFile), 0755); err != nil {
		return fmt.Errorf("创建历史记录目录失败: %v", err)
	}

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
	historyMutex.Lock()
	defer historyMutex.Unlock()

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

// GetLastHistory 返回最近的一条历史记录
func GetLastHistory() (*CommandHistory, error) {
	history, err := GetHistory()
	if err != nil {
		return nil, err
	}

	if len(history) == 0 {
		return nil, nil
	}

	return &history[len(history)-1], nil
}

func ClearHistory() error {
	historyMutex.Lock()
	defer historyMutex.Unlock()

	if err := os.Remove(historyFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清除历史记录失败: %v", err)
	}
	return nil
}
