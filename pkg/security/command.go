package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CommandStats 表示命令执行统计信息
type CommandStats struct {
	Command     string    `json:"command"`
	User        string    `json:"user"`
	Timestamp   time.Time `json:"timestamp"`
	Duration    int64     `json:"duration_ms"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	Args        []string  `json:"args"`
	Environment string    `json:"environment"`
}

// CommandPolicy 表示命令执行策略
type CommandPolicy struct {
	Command     string   `json:"command"`
	Allowed     bool     `json:"allowed"`
	Users       []string `json:"users,omitempty"`
	Groups      []string `json:"groups,omitempty"`
	TimeRange   []string `json:"time_range,omitempty"` // ["09:00", "18:00"]
	MaxDuration int      `json:"max_duration_seconds,omitempty"`
	MaxCalls    int      `json:"max_calls_per_hour,omitempty"`
}

// CommandManager 管理命令执行统计和权限控制
type CommandManager struct {
	statsFile    string
	policiesFile string
	stats        []CommandStats
	policies     []CommandPolicy
	mu           sync.RWMutex
}

// NewCommandManager 创建新的命令管理器
func NewCommandManager(statsFile, policiesFile string) (*CommandManager, error) {
	cm := &CommandManager{
		statsFile:    statsFile,
		policiesFile: policiesFile,
	}

	// 加载现有统计信息
	if err := cm.loadStats(); err != nil {
		return nil, fmt.Errorf("加载统计信息失败: %v", err)
	}

	// 加载策略
	if err := cm.loadPolicies(); err != nil {
		return nil, fmt.Errorf("加载策略失败: %v", err)
	}

	return cm, nil
}

// RecordCommand 记录命令执行
func (cm *CommandManager) RecordCommand(stats CommandStats) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.stats = append(cm.stats, stats)
	return cm.saveStats()
}

// CheckPermission 检查命令执行权限
func (cm *CommandManager) CheckPermission(command string, user string, groups []string) (bool, string) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 查找匹配的策略
	for _, policy := range cm.policies {
		if policy.Command == command {
			// 检查基本权限
			if !policy.Allowed {
				return false, "命令被策略禁止执行"
			}

			// 检查用户权限
			if len(policy.Users) > 0 {
				userAllowed := false
				for _, allowedUser := range policy.Users {
					if allowedUser == user {
						userAllowed = true
						break
					}
				}
				if !userAllowed {
					return false, "用户无权执行此命令"
				}
			}

			// 检查组权限
			if len(policy.Groups) > 0 {
				groupAllowed := false
				for _, allowedGroup := range policy.Groups {
					for _, userGroup := range groups {
						if allowedGroup == userGroup {
							groupAllowed = true
							break
						}
					}
					if groupAllowed {
						break
					}
				}
				if !groupAllowed {
					return false, "用户组无权执行此命令"
				}
			}

			// 检查时间范围
			if len(policy.TimeRange) == 2 {
				now := time.Now()
				currentTime := now.Format("15:04")
				if currentTime < policy.TimeRange[0] || currentTime > policy.TimeRange[1] {
					return false, "当前时间不允许执行此命令"
				}
			}

			// 检查调用频率
			if policy.MaxCalls > 0 {
				hourAgo := time.Now().Add(-time.Hour)
				callCount := 0
				for _, stat := range cm.stats {
					if stat.Command == command && stat.Timestamp.After(hourAgo) {
						callCount++
					}
				}
				if callCount >= policy.MaxCalls {
					return false, "超出每小时最大调用次数限制"
				}
			}

			return true, ""
		}
	}

	// 如果没有找到匹配的策略，默认允许执行
	return true, ""
}

// GetCommandStats 获取命令执行统计信息
func (cm *CommandManager) GetCommandStats(command string, duration time.Duration) []CommandStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var filteredStats []CommandStats
	cutoff := time.Now().Add(-duration)

	for _, stat := range cm.stats {
		if stat.Command == command && stat.Timestamp.After(cutoff) {
			filteredStats = append(filteredStats, stat)
		}
	}

	return filteredStats
}

// AddPolicy 添加新的命令策略
func (cm *CommandManager) AddPolicy(policy CommandPolicy) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 检查策略是否已存在
	for i, p := range cm.policies {
		if p.Command == policy.Command {
			cm.policies[i] = policy
			return cm.savePolicies()
		}
	}

	cm.policies = append(cm.policies, policy)
	return cm.savePolicies()
}

// RemovePolicy 移除命令策略
func (cm *CommandManager) RemovePolicy(command string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i, policy := range cm.policies {
		if policy.Command == command {
			cm.policies = append(cm.policies[:i], cm.policies[i+1:]...)
			return cm.savePolicies()
		}
	}

	return nil
}

// 加载统计信息
func (cm *CommandManager) loadStats() error {
	if _, err := os.Stat(cm.statsFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(cm.statsFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &cm.stats)
}

// 保存统计信息
func (cm *CommandManager) saveStats() error {
	data, err := json.MarshalIndent(cm.stats, "", "  ")
	if err != nil {
		return err
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(cm.statsFile), 0755); err != nil {
		return err
	}

	return os.WriteFile(cm.statsFile, data, 0644)
}

// 加载策略
func (cm *CommandManager) loadPolicies() error {
	if _, err := os.Stat(cm.policiesFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(cm.policiesFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &cm.policies)
}

// 保存策略
func (cm *CommandManager) savePolicies() error {
	data, err := json.MarshalIndent(cm.policies, "", "  ")
	if err != nil {
		return err
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(cm.policiesFile), 0755); err != nil {
		return err
	}

	return os.WriteFile(cm.policiesFile, data, 0644)
} 