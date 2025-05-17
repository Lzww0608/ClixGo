package security

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 创建临时测试文件
func setupTestFiles(t *testing.T) (string, string, func()) {
	tmpDir, err := os.MkdirTemp("", "command-test")
	require.NoError(t, err, "创建临时目录失败")

	statsFile := filepath.Join(tmpDir, "stats.json")
	policiesFile := filepath.Join(tmpDir, "policies.json")

	// 返回清理函数
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return statsFile, policiesFile, cleanup
}

// TestNewCommandManager 测试创建命令管理器
func TestNewCommandManager(t *testing.T) {
	statsFile, policiesFile, cleanup := setupTestFiles(t)
	defer cleanup()

	cm, err := NewCommandManager(statsFile, policiesFile)
	assert.NoError(t, err, "创建命令管理器不应返回错误")
	assert.NotNil(t, cm, "命令管理器不应为nil")
	assert.Equal(t, statsFile, cm.statsFile, "statsFile应正确设置")
	assert.Equal(t, policiesFile, cm.policiesFile, "policiesFile应正确设置")
	assert.Empty(t, cm.stats, "初始stats应为空")
	assert.Empty(t, cm.policies, "初始policies应为空")
}

// TestRecordCommand 测试记录命令执行
func TestRecordCommand(t *testing.T) {
	statsFile, policiesFile, cleanup := setupTestFiles(t)
	defer cleanup()

	cm, err := NewCommandManager(statsFile, policiesFile)
	require.NoError(t, err)

	// 记录一个命令
	cmdStats := CommandStats{
		Command:     "test-cmd",
		User:        "test-user",
		Timestamp:   time.Now(),
		Duration:    100,
		Success:     true,
		Args:        []string{"arg1", "arg2"},
		Environment: "test",
	}

	err = cm.RecordCommand(cmdStats)
	assert.NoError(t, err, "记录命令不应返回错误")
	assert.Len(t, cm.stats, 1, "stats应包含1个记录")
	assert.Equal(t, cmdStats, cm.stats[0], "记录的命令统计应正确")

	// 验证文件是否正确写入
	fileData, err := os.ReadFile(statsFile)
	assert.NoError(t, err, "读取统计文件不应返回错误")

	var loadedStats []CommandStats
	err = json.Unmarshal(fileData, &loadedStats)
	assert.NoError(t, err, "解析统计数据不应返回错误")
	assert.Len(t, loadedStats, 1, "加载的stats应包含1个记录")
	assert.Equal(t, cmdStats.Command, loadedStats[0].Command, "命令名称应匹配")
	assert.Equal(t, cmdStats.User, loadedStats[0].User, "用户名应匹配")
}

// TestAddPolicy 测试添加策略
func TestAddPolicy(t *testing.T) {
	statsFile, policiesFile, cleanup := setupTestFiles(t)
	defer cleanup()

	cm, err := NewCommandManager(statsFile, policiesFile)
	require.NoError(t, err)

	// 添加策略
	policy := CommandPolicy{
		Command: "test-cmd",
		Allowed: true,
		Users:   []string{"user1", "user2"},
	}

	err = cm.AddPolicy(policy)
	assert.NoError(t, err, "添加策略不应返回错误")
	assert.Len(t, cm.policies, 1, "policies应包含1个记录")
	assert.Equal(t, policy, cm.policies[0], "策略应正确添加")

	// 验证文件是否正确写入
	fileData, err := os.ReadFile(policiesFile)
	assert.NoError(t, err, "读取策略文件不应返回错误")

	var loadedPolicies []CommandPolicy
	err = json.Unmarshal(fileData, &loadedPolicies)
	assert.NoError(t, err, "解析策略数据不应返回错误")
	assert.Len(t, loadedPolicies, 1, "加载的policies应包含1个记录")
	assert.Equal(t, policy.Command, loadedPolicies[0].Command, "命令名称应匹配")
	assert.Equal(t, policy.Allowed, loadedPolicies[0].Allowed, "allowed字段应匹配")
	assert.Equal(t, policy.Users, loadedPolicies[0].Users, "users字段应匹配")

	// 添加重复策略（更新）
	updatedPolicy := CommandPolicy{
		Command: "test-cmd",
		Allowed: false,
		Users:   []string{"user3"},
	}

	err = cm.AddPolicy(updatedPolicy)
	assert.NoError(t, err, "更新策略不应返回错误")
	assert.Len(t, cm.policies, 1, "policies长度应保持为1")
	assert.Equal(t, false, cm.policies[0].Allowed, "策略应被更新")
	assert.Equal(t, []string{"user3"}, cm.policies[0].Users, "用户列表应被更新")
}

// TestRemovePolicy 测试移除策略
func TestRemovePolicy(t *testing.T) {
	statsFile, policiesFile, cleanup := setupTestFiles(t)
	defer cleanup()

	cm, err := NewCommandManager(statsFile, policiesFile)
	require.NoError(t, err)

	// 添加策略
	policy := CommandPolicy{
		Command: "test-cmd",
		Allowed: true,
	}
	cm.policies = append(cm.policies, policy)

	// 移除已存在的策略
	err = cm.RemovePolicy("test-cmd")
	assert.NoError(t, err, "移除存在的策略不应返回错误")
	assert.Empty(t, cm.policies, "policies应为空")

	// 移除不存在的策略
	err = cm.RemovePolicy("non-existent")
	assert.Nil(t, err, "移除不存在的策略不应返回错误")
}

// TestCheckPermission 测试检查权限
func TestCheckPermission(t *testing.T) {
	statsFile, policiesFile, cleanup := setupTestFiles(t)
	defer cleanup()

	cm, err := NewCommandManager(statsFile, policiesFile)
	require.NoError(t, err)

	// 添加策略
	cm.policies = append(cm.policies, CommandPolicy{
		Command: "cmd1",
		Allowed: true,
	})

	cm.policies = append(cm.policies, CommandPolicy{
		Command: "cmd2",
		Allowed: false,
	})

	cm.policies = append(cm.policies, CommandPolicy{
		Command: "cmd3",
		Allowed: true,
		Users:   []string{"user1", "user2"},
	})

	cm.policies = append(cm.policies, CommandPolicy{
		Command: "cmd4",
		Allowed: true,
		Groups:  []string{"group1", "group2"},
	})

	// 设置当前时间区间的命令
	now := time.Now()
	startTime := now.Add(-time.Hour).Format("15:04")
	endTime := now.Add(time.Hour).Format("15:04")

	cm.policies = append(cm.policies, CommandPolicy{
		Command:   "cmd5",
		Allowed:   true,
		TimeRange: []string{startTime, endTime},
	})

	// 设置超出时间区间的命令
	pastTime := now.Add(-2 * time.Hour).Format("15:04")
	futureTime := now.Add(-1 * time.Hour).Format("15:04")

	cm.policies = append(cm.policies, CommandPolicy{
		Command:   "cmd6",
		Allowed:   true,
		TimeRange: []string{pastTime, futureTime},
	})

	// 测试基本权限
	allowed, msg := cm.CheckPermission("cmd1", "anyuser", []string{})
	assert.True(t, allowed, "默认允许的命令应通过权限检查")
	assert.Empty(t, msg, "成功检查应没有错误消息")

	allowed, msg = cm.CheckPermission("cmd2", "anyuser", []string{})
	assert.False(t, allowed, "被禁止的命令不应通过权限检查")
	assert.Contains(t, msg, "禁止", "错误消息应包含'禁止'")

	// 测试用户限制
	allowed, msg = cm.CheckPermission("cmd3", "user1", []string{})
	assert.True(t, allowed, "授权用户应通过权限检查")
	assert.Empty(t, msg, "成功检查应没有错误消息")

	allowed, msg = cm.CheckPermission("cmd3", "user3", []string{})
	assert.False(t, allowed, "未授权用户不应通过权限检查")
	assert.Contains(t, msg, "用户无权", "错误消息应包含'用户无权'")

	// 测试组限制
	allowed, msg = cm.CheckPermission("cmd4", "anyuser", []string{"group1"})
	assert.True(t, allowed, "授权组的用户应通过权限检查")
	assert.Empty(t, msg, "成功检查应没有错误消息")

	allowed, msg = cm.CheckPermission("cmd4", "anyuser", []string{"group3"})
	assert.False(t, allowed, "未授权组的用户不应通过权限检查")
	assert.Contains(t, msg, "用户组无权", "错误消息应包含'用户组无权'")

	// 测试时间限制
	allowed, msg = cm.CheckPermission("cmd5", "anyuser", []string{})
	assert.True(t, allowed, "在允许时间范围内的命令应通过权限检查")
	assert.Empty(t, msg, "成功检查应没有错误消息")

	allowed, msg = cm.CheckPermission("cmd6", "anyuser", []string{})
	assert.False(t, allowed, "不在允许时间范围内的命令不应通过权限检查")
	assert.Contains(t, msg, "当前时间不允许", "错误消息应包含'当前时间不允许'")
}

// TestGetCommandStats 测试获取命令统计
func TestGetCommandStats(t *testing.T) {
	statsFile, policiesFile, cleanup := setupTestFiles(t)
	defer cleanup()

	cm, err := NewCommandManager(statsFile, policiesFile)
	require.NoError(t, err)

	// 添加测试统计数据
	now := time.Now()

	cm.stats = append(cm.stats, CommandStats{
		Command:   "cmd1",
		Timestamp: now.Add(-30 * time.Minute),
	})

	cm.stats = append(cm.stats, CommandStats{
		Command:   "cmd1",
		Timestamp: now.Add(-90 * time.Minute),
	})

	cm.stats = append(cm.stats, CommandStats{
		Command:   "cmd2",
		Timestamp: now.Add(-30 * time.Minute),
	})

	// 测试过滤
	stats := cm.GetCommandStats("cmd1", time.Hour)
	assert.Len(t, stats, 1, "应只返回最近一小时内的cmd1统计")
	assert.Equal(t, "cmd1", stats[0].Command, "返回的统计应匹配请求的命令")

	stats = cm.GetCommandStats("cmd1", 2*time.Hour)
	assert.Len(t, stats, 2, "应返回最近两小时内的所有cmd1统计")

	stats = cm.GetCommandStats("cmd2", time.Hour)
	assert.Len(t, stats, 1, "应返回最近一小时内的cmd2统计")

	stats = cm.GetCommandStats("cmd3", time.Hour)
	assert.Empty(t, stats, "不存在的命令应返回空统计")
}

// TestLoadExistingData 测试加载已存在的数据
func TestLoadExistingData(t *testing.T) {
	statsFile, policiesFile, cleanup := setupTestFiles(t)
	defer cleanup()

	// 预先写入一些测试数据
	stats := []CommandStats{
		{
			Command:   "test-cmd",
			User:      "test-user",
			Timestamp: time.Now(),
			Success:   true,
		},
	}

	statsData, err := json.MarshalIndent(stats, "", "  ")
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Dir(statsFile), 0755)
	require.NoError(t, err)

	err = os.WriteFile(statsFile, statsData, 0644)
	require.NoError(t, err)

	policies := []CommandPolicy{
		{
			Command: "test-cmd",
			Allowed: true,
			Users:   []string{"test-user"},
		},
	}

	policiesData, err := json.MarshalIndent(policies, "", "  ")
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Dir(policiesFile), 0755)
	require.NoError(t, err)

	err = os.WriteFile(policiesFile, policiesData, 0644)
	require.NoError(t, err)

	// 创建新的管理器，应加载现有数据
	cm, err := NewCommandManager(statsFile, policiesFile)
	assert.NoError(t, err, "创建命令管理器不应返回错误")
	assert.Len(t, cm.stats, 1, "应加载1个统计记录")
	assert.Len(t, cm.policies, 1, "应加载1个策略记录")
	assert.Equal(t, "test-cmd", cm.stats[0].Command, "应正确加载命令名称")
	assert.Equal(t, "test-user", cm.stats[0].User, "应正确加载用户名")
	assert.Equal(t, "test-cmd", cm.policies[0].Command, "应正确加载策略命令名称")
	assert.True(t, cm.policies[0].Allowed, "应正确加载policy allowed字段")
}

// TestMaxCallsLimit 测试最大调用次数限制
func TestMaxCallsLimit(t *testing.T) {
	statsFile, policiesFile, cleanup := setupTestFiles(t)
	defer cleanup()

	cm, err := NewCommandManager(statsFile, policiesFile)
	require.NoError(t, err)

	// 添加有最大调用次数限制的策略
	cm.policies = append(cm.policies, CommandPolicy{
		Command:  "limited-cmd",
		Allowed:  true,
		MaxCalls: 2,
	})

	// 添加一些历史调用记录
	now := time.Now()

	cm.stats = append(cm.stats, CommandStats{
		Command:   "limited-cmd",
		Timestamp: now.Add(-30 * time.Minute),
	})

	cm.stats = append(cm.stats, CommandStats{
		Command:   "limited-cmd",
		Timestamp: now.Add(-20 * time.Minute),
	})

	// 第一次检查应该到达限制
	allowed, msg := cm.CheckPermission("limited-cmd", "user", []string{})
	assert.False(t, allowed, "达到最大调用次数限制的命令不应通过权限检查")
	assert.Contains(t, msg, "超出每小时最大调用次数", "错误消息应包含'超出每小时最大调用次数'")

	// 添加一个超过一小时前的记录，不应计入限制
	cm.stats = append(cm.stats, CommandStats{
		Command:   "limited-cmd",
		Timestamp: now.Add(-61 * time.Minute),
	})

	// 现在应该仍有2条在一小时内的记录
	allowed, msg = cm.CheckPermission("limited-cmd", "user", []string{})
	assert.False(t, allowed, "达到最大调用次数限制的命令不应通过权限检查")

	// 移除一条记录使其低于限制
	cm.stats = cm.stats[:1]

	// 现在应该通过检查
	allowed, msg = cm.CheckPermission("limited-cmd", "user", []string{})
	assert.True(t, allowed, "未达到最大调用次数限制的命令应通过权限检查")
	assert.Empty(t, msg, "成功检查应没有错误消息")
}
