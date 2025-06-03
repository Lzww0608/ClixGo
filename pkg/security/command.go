/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-3 18:51:34
* @Description: 安全管理功能的核心实现，提供命令权限控制和审计
 */

package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CommandStats 命令执行统计信息
// 记录命令执行的详细信息，用于审计和监控
type CommandStats struct {
	Command     string    `json:"command"`         // 执行的命令名称
	User        string    `json:"user"`            // 执行命令的用户
	Timestamp   time.Time `json:"timestamp"`       // 命令执行时间戳
	Duration    int64     `json:"duration_ms"`     // 命令执行耗时（毫秒）
	Success     bool      `json:"success"`         // 命令是否执行成功
	Error       string    `json:"error,omitempty"` // 错误信息（如有）
	Args        []string  `json:"args"`            // 命令参数列表
	Environment string    `json:"environment"`     // 执行环境标识
}

// CommandPolicy 命令执行策略配置
// 定义特定命令的访问控制和执行限制规则
type CommandPolicy struct {
	Command     string   `json:"command"`                        // 受策略控制的命令名称
	Allowed     bool     `json:"allowed"`                        // 是否允许执行该命令
	Users       []string `json:"users,omitempty"`                // 允许执行的用户列表
	Groups      []string `json:"groups,omitempty"`               // 允许执行的用户组列表
	TimeRange   []string `json:"time_range,omitempty"`           // 允许执行的时间范围 ["09:00", "18:00"]
	MaxDuration int      `json:"max_duration_seconds,omitempty"` // 最大执行时长限制（秒）
	MaxCalls    int      `json:"max_calls_per_hour,omitempty"`   // 每小时最大调用次数限制
}

// CommandManager 命令安全管理器
// 负责命令执行的权限控制、统计记录和策略管理
type CommandManager struct {
	statsFile    string          // 统计数据文件路径
	policiesFile string          // 策略配置文件路径
	stats        []CommandStats  // 命令执行统计记录
	policies     []CommandPolicy // 命令执行策略列表
	mu           sync.RWMutex    // 读写锁保护并发访问
}

// NewCommandManager 创建新的命令管理器实例
//
// 参数:
//   - statsFile: 命令统计数据存储文件路径
//   - policiesFile: 命令策略配置文件路径
//
// 返回:
//   - *CommandManager: 初始化完成的命令管理器实例
//   - error: 初始化过程中的错误
//
// 该函数会自动加载现有的统计数据和策略配置
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

// RecordCommand 记录命令执行统计信息
//
// 参数:
//   - stats: 包含命令执行详细信息的统计数据
//
// 返回:
//   - error: 记录过程中的错误，nil表示成功
//
// 该函数会将统计信息添加到内存缓存并立即持久化到磁盘
func (cm *CommandManager) RecordCommand(stats CommandStats) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.stats = append(cm.stats, stats)
	return cm.saveStats()
}

// CheckPermission 检查指定用户是否有权限执行特定命令
//
// 参数:
//   - command: 要检查的命令名称
//   - user: 执行命令的用户名
//   - groups: 用户所属的用户组列表
//
// 返回:
//   - bool: true表示有权限，false表示无权限
//   - string: 权限检查的详细说明，拒绝时包含具体原因
//
// 该函数会按照以下顺序进行权限检查：
// 1. 命令是否被策略禁止
// 2. 用户是否在允许列表中
// 3. 用户组是否在允许列表中
// 4. 当前时间是否在允许的时间范围内
// 5. 是否超出调用频率限制
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

// GetCommandStats 获取指定命令在特定时间段内的执行统计信息
//
// 参数:
//   - command: 要查询的命令名称
//   - duration: 查询的时间范围（从当前时间向前计算）
//
// 返回:
//   - []CommandStats: 符合条件的命令执行统计记录列表
//
// 该函数返回指定时间段内该命令的所有执行记录，可用于分析命令使用模式
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

// AddPolicy 添加或更新命令执行策略
//
// 参数:
//   - policy: 要添加的命令策略配置
//
// 返回:
//   - error: 操作过程中的错误，nil表示成功
//
// 如果指定命令的策略已存在，则会更新现有策略；否则添加新策略
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

// RemovePolicy 移除指定命令的执行策略
//
// 参数:
//   - command: 要移除策略的命令名称
//
// 返回:
//   - error: 操作过程中的错误，nil表示成功
//
// 移除策略后，该命令将使用默认的权限控制规则
// 如果策略不存在，函数会静默成功而不返回错误
func (cm *CommandManager) RemovePolicy(command string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i, policy := range cm.policies {
		if policy.Command == command {
			cm.policies = append(cm.policies[:i], cm.policies[i+1:]...)
			return cm.savePolicies()
		}
	}

	// 策略不存在时静默成功，不返回错误
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
