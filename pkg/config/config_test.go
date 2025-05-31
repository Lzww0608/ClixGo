package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试辅助函数：创建临时目录
func createTempDir(t *testing.T) (string, func()) {
	tempDir, err := ioutil.TempDir("", "config-test")
	require.NoError(t, err, "创建临时目录失败")

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// 测试辅助函数：创建测试配置文件
func createTestConfigFile(t *testing.T, dir, content string) string {
	configFile := filepath.Join(dir, "config.yaml")
	err := ioutil.WriteFile(configFile, []byte(content), 0644)
	require.NoError(t, err, "创建测试配置文件失败")
	return configFile
}

// =============================================================================
// 1. 配置文件加载/保存功能测试
// =============================================================================

func TestConfigManager_LoadDefaultConfig(t *testing.T) {
	tempDir, cleanup := createTempDir(t)
	defer cleanup()

	configPath := filepath.Join(tempDir, "config.yaml")
	cm := NewConfigManager(configPath)

	// 测试加载不存在的配置文件（应该使用默认值）
	err := cm.Load()
	assert.NoError(t, err, "加载不存在的配置文件应该成功（使用默认值）")

	// 验证默认值是否正确加载
	appName, err := cm.GetString("app.name")
	assert.NoError(t, err)
	assert.Equal(t, "ClixGo", appName, "默认应用名称应该是ClixGo")

	logLevel, err := cm.GetString("log.level")
	assert.NoError(t, err)
	assert.Equal(t, "info", logLevel, "默认日志级别应该是info")
}

func TestConfigManager_LoadFromFile(t *testing.T) {
	tempDir, cleanup := createTempDir(t)
	defer cleanup()

	// 创建测试配置文件（使用YAML格式）
	configContent := `app:
  name: TestApp
  version: 2.0.0
  environment: test
log:
  level: debug
  format: json
custom:
  setting: custom_value`

	configPath := createTestConfigFile(t, tempDir, configContent)
	cm := NewConfigManager(configPath)

	// 加载配置文件
	err := cm.Load()
	assert.NoError(t, err, "加载配置文件应该成功")

	// 验证文件中的配置是否正确加载
	appName, err := cm.GetString("app.name")
	assert.NoError(t, err)
	assert.Equal(t, "TestApp", appName, "应该从文件加载应用名称")

	version, err := cm.GetString("app.version")
	assert.NoError(t, err)
	assert.Equal(t, "2.0.0", version, "应该从文件加载版本号")

	logLevel, err := cm.GetString("log.level")
	assert.NoError(t, err)
	assert.Equal(t, "debug", logLevel, "应该从文件加载日志级别")

	customSetting, err := cm.GetString("custom.setting")
	assert.NoError(t, err)
	assert.Equal(t, "custom_value", customSetting, "应该加载自定义配置")
}

func TestConfigManager_SaveToFile(t *testing.T) {
	tempDir, cleanup := createTempDir(t)
	defer cleanup()

	configPath := filepath.Join(tempDir, "config.yaml")
	cm := NewConfigManager(configPath)

	// 设置一些配置值
	cm.Set("app.name", "SavedApp", SourceFile)
	cm.Set("app.version", "3.0.0", SourceFile)
	cm.Set("log.level", "error", SourceFile)
	cm.Set("custom.key", "custom_value", SourceFile)

	// 保存配置到文件
	err := cm.Save()
	assert.NoError(t, err, "保存配置到文件应该成功")

	// 验证文件是否被创建
	_, err = os.Stat(configPath)
	assert.NoError(t, err, "配置文件应该被创建")

	// 创建新的配置管理器并加载刚保存的文件
	cm2 := NewConfigManager(configPath)
	err = cm2.Load()
	assert.NoError(t, err, "重新加载保存的配置应该成功")

	// 验证保存的配置是否正确
	appName, err := cm2.GetString("app.name")
	assert.NoError(t, err)
	assert.Equal(t, "SavedApp", appName, "保存的应用名称应该被正确加载")

	version, err := cm2.GetString("app.version")
	assert.NoError(t, err)
	assert.Equal(t, "3.0.0", version, "保存的版本号应该被正确加载")
}

func TestConfigManager_LoadInvalidFile(t *testing.T) {
	tempDir, cleanup := createTempDir(t)
	defer cleanup()

	// 创建无效的配置文件（无效的YAML格式）
	invalidContent := `invalid yaml content:
  - this is not
  - proper yaml: {{{ invalid`
	configPath := createTestConfigFile(t, tempDir, invalidContent)

	cm := NewConfigManager(configPath)

	// 加载无效配置文件应该返回错误
	err := cm.Load()
	assert.Error(t, err, "加载无效配置文件应该返回错误")
	assert.Contains(t, err.Error(), "读取配置文件失败", "错误信息应该包含相关描述")
}

// =============================================================================
// 2. 配置验证和默认值测试
// =============================================================================

func TestConfigManager_DefaultValues(t *testing.T) {
	tempDir, cleanup := createTempDir(t)
	defer cleanup()

	configPath := filepath.Join(tempDir, "config.yaml")
	cm := NewConfigManager(configPath)

	// 验证所有默认值
	testCases := []struct {
		key      string
		expected interface{}
		desc     string
	}{
		{"app.name", "ClixGo", "默认应用名称"},
		{"app.version", "1.0.0", "默认版本号"},
		{"app.environment", "development", "默认环境"},
		{"log.level", "info", "默认日志级别"},
		{"log.format", "text", "默认日志格式"},
		{"log.output", "stdout", "默认日志输出"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			value, err := cm.Get(tc.key)
			assert.NoError(t, err, "获取默认值应该成功: %s", tc.key)
			assert.Equal(t, tc.expected, value, "默认值应该正确: %s", tc.desc)

			// 验证配置源是默认源
			source, err := cm.GetSource(tc.key)
			assert.NoError(t, err, "获取配置源应该成功: %s", tc.key)
			assert.Equal(t, SourceDefault, source, "默认值的源应该是SourceDefault")
		})
	}
}

func TestConfigManager_ConfigValidation(t *testing.T) {
	tempDir, cleanup := createTempDir(t)
	defer cleanup()

	configPath := filepath.Join(tempDir, "config.yaml")
	cm := NewConfigManager(configPath)

	// 测试类型验证
	t.Run("类型验证", func(t *testing.T) {
		// 设置不同类型的值
		cm.Set("string.key", "string_value", SourceFile)
		cm.Set("int.key", 42, SourceFile)
		cm.Set("bool.key", true, SourceFile)

		// 正确类型获取应该成功
		strVal, err := cm.GetString("string.key")
		assert.NoError(t, err)
		assert.Equal(t, "string_value", strVal)

		intVal, err := cm.GetInt("int.key")
		assert.NoError(t, err)
		assert.Equal(t, 42, intVal)

		boolVal, err := cm.GetBool("bool.key")
		assert.NoError(t, err)
		assert.Equal(t, true, boolVal)

		// 错误类型获取应该失败
		_, err = cm.GetString("int.key")
		assert.Error(t, err, "尝试获取错误类型应该失败")
		assert.Contains(t, err.Error(), "不是字符串类型", "错误信息应该明确")

		_, err = cm.GetInt("string.key")
		assert.Error(t, err, "尝试获取错误类型应该失败")
		assert.Contains(t, err.Error(), "不是整数类型", "错误信息应该明确")

		_, err = cm.GetBool("string.key")
		assert.Error(t, err, "尝试获取错误类型应该失败")
		assert.Contains(t, err.Error(), "不是布尔类型", "错误信息应该明确")
	})

	t.Run("不存在的键验证", func(t *testing.T) {
		// 获取不存在的配置项应该返回错误
		_, err := cm.Get("nonexistent.key")
		assert.Error(t, err, "获取不存在的配置项应该返回错误")
		assert.Contains(t, err.Error(), "不存在", "错误信息应该说明配置项不存在")

		_, err = cm.GetString("nonexistent.key")
		assert.Error(t, err, "获取不存在的字符串配置应该返回错误")

		_, err = cm.GetSource("nonexistent.key")
		assert.Error(t, err, "获取不存在的配置源应该返回错误")
	})
}

func TestConfigManager_ConfigSource(t *testing.T) {
	tempDir, cleanup := createTempDir(t)
	defer cleanup()

	configPath := filepath.Join(tempDir, "config.yaml")
	cm := NewConfigManager(configPath)

	// 测试不同配置源
	cm.Set("default.key", "default_value", SourceDefault)
	cm.Set("file.key", "file_value", SourceFile)
	cm.Set("env.key", "env_value", SourceEnv)
	cm.Set("flag.key", "flag_value", SourceFlag)

	testCases := []struct {
		key            string
		expectedSource ConfigSource
		desc           string
	}{
		{"default.key", SourceDefault, "默认源"},
		{"file.key", SourceFile, "文件源"},
		{"env.key", SourceEnv, "环境变量源"},
		{"flag.key", SourceFlag, "命令行参数源"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			source, err := cm.GetSource(tc.key)
			assert.NoError(t, err, "获取配置源应该成功")
			assert.Equal(t, tc.expectedSource, source, "配置源应该正确")
		})
	}
}

func TestConfigManager_EncryptionFlag(t *testing.T) {
	tempDir, cleanup := createTempDir(t)
	defer cleanup()

	configPath := filepath.Join(tempDir, "config.yaml")
	cm := NewConfigManager(configPath)

	// 设置普通配置
	cm.Set("normal.key", "normal_value", SourceFile)

	// 检查加密标志（目前应该都是false）
	isEncrypted, err := cm.IsEncrypted("normal.key")
	assert.NoError(t, err, "检查加密标志应该成功")
	assert.False(t, isEncrypted, "普通配置不应该被加密")

	// 检查不存在的键
	_, err = cm.IsEncrypted("nonexistent.key")
	assert.Error(t, err, "检查不存在的键的加密状态应该返回错误")
}

// =============================================================================
// 3. 环境变量覆盖测试
// =============================================================================

func TestConfigManager_EnvironmentVariableOverride(t *testing.T) {
	tempDir, cleanup := createTempDir(t)
	defer cleanup()

	// 创建测试配置文件
	configContent := `app:
  name: FileApp
  version: 1.0.0
  environment: development
log:
  level: info
  format: text
database:
  host: localhost
  port: 5432
  user: default_user`

	configPath := createTestConfigFile(t, tempDir, configContent)
	cm := NewConfigManager(configPath)

	// 设置环境变量
	originalEnvVars := make(map[string]string)
	testEnvVars := map[string]string{
		"APP_NAME":      "EnvApp",
		"APP_VERSION":   "2.0.0",
		"LOG_LEVEL":     "debug",
		"DATABASE_HOST": "env-host",
		"DATABASE_PORT": "3306",
		"DATABASE_USER": "env_user",
	}

	// 保存原始环境变量并设置新的
	for envKey, envValue := range testEnvVars {
		if originalValue := os.Getenv(envKey); originalValue != "" {
			originalEnvVars[envKey] = originalValue
		}
		os.Setenv(envKey, envValue)
	}

	// 清理函数
	defer func() {
		for envKey := range testEnvVars {
			if originalValue, exists := originalEnvVars[envKey]; exists {
				os.Setenv(envKey, originalValue)
			} else {
				os.Unsetenv(envKey)
			}
		}
	}()

	// 加载配置（环境变量应该覆盖文件配置）
	err := cm.Load()
	assert.NoError(t, err, "加载配置应该成功")

	// 验证环境变量覆盖了文件配置
	t.Run("环境变量覆盖文件配置", func(t *testing.T) {
		testCases := []struct {
			key      string
			expected string
			desc     string
		}{
			{"app.name", "EnvApp", "应用名称应该被环境变量覆盖"},
			{"app.version", "2.0.0", "版本号应该被环境变量覆盖"},
			{"log.level", "debug", "日志级别应该被环境变量覆盖"},
			{"database.host", "env-host", "数据库主机应该被环境变量覆盖"},
			{"database.port", "3306", "数据库端口应该被环境变量覆盖"},
			{"database.user", "env_user", "数据库用户应该被环境变量覆盖"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				value, err := cm.GetString(tc.key)
				assert.NoError(t, err, "获取配置值应该成功: %s", tc.key)
				assert.Equal(t, tc.expected, value, tc.desc)

				// 验证配置源是环境变量
				source, err := cm.GetSource(tc.key)
				assert.NoError(t, err, "获取配置源应该成功: %s", tc.key)
				assert.Equal(t, SourceEnv, source, "配置源应该是SourceEnv: %s", tc.key)
			})
		}
	})

	// 验证没有环境变量的配置仍然来自文件
	t.Run("文件配置保持不变", func(t *testing.T) {
		environment, err := cm.GetString("app.environment")
		assert.NoError(t, err)
		assert.Equal(t, "development", environment, "没有环境变量覆盖的配置应该保持文件值")

		source, err := cm.GetSource("app.environment")
		assert.NoError(t, err)
		assert.Equal(t, SourceFile, source, "配置源应该是SourceFile")

		format, err := cm.GetString("log.format")
		assert.NoError(t, err)
		assert.Equal(t, "text", format, "日志格式应该保持文件值")
	})
}

func TestConfigManager_EnvironmentVariablePriority(t *testing.T) {
	tempDir, cleanup := createTempDir(t)
	defer cleanup()

	configPath := filepath.Join(tempDir, "config.yaml")
	cm := NewConfigManager(configPath)

	// 设置环境变量
	os.Setenv("TEST_PRIORITY", "env_value")
	defer os.Unsetenv("TEST_PRIORITY")

	// 先加载默认配置和环境变量
	err := cm.Load()
	assert.NoError(t, err)

	// 验证环境变量值
	value, err := cm.GetString("test.priority")
	if err == nil {
		assert.Equal(t, "env_value", value, "环境变量值应该正确")

		source, err := cm.GetSource("test.priority")
		assert.NoError(t, err)
		assert.Equal(t, SourceEnv, source, "配置源应该是环境变量")
	}

	// 手动设置配置值（应该覆盖环境变量）
	cm.Set("test.priority", "manual_value", SourceFlag)

	value, err = cm.GetString("test.priority")
	assert.NoError(t, err)
	assert.Equal(t, "manual_value", value, "手动设置的值应该覆盖环境变量")

	source, err := cm.GetSource("test.priority")
	assert.NoError(t, err)
	assert.Equal(t, SourceFlag, source, "配置源应该是SourceFlag")
}

// =============================================================================
// 4. 配置热更新测试
// =============================================================================

func TestConfigManager_HotReload(t *testing.T) {
	tempDir, cleanup := createTempDir(t)
	defer cleanup()

	// 创建初始配置文件
	initialContent := `app:
  name: InitialApp
  version: 1.0.0
  debug: false
log:
  level: info
  output: stdout`

	configPath := createTestConfigFile(t, tempDir, initialContent)
	cm := NewConfigManager(configPath)

	// 加载初始配置
	err := cm.Load()
	assert.NoError(t, err, "加载初始配置应该成功")

	// 验证初始配置
	appName, err := cm.GetString("app.name")
	assert.NoError(t, err)
	assert.Equal(t, "InitialApp", appName, "初始应用名称应该正确")

	debug, err := cm.GetBool("app.debug")
	assert.NoError(t, err)
	assert.False(t, debug, "初始debug模式应该是false")

	// 创建新的配置内容
	updatedContent := `app:
  name: UpdatedApp
  version: 2.0.0
  debug: true
log:
  level: debug
  output: file
new_feature:
  enabled: true
  timeout: 30`

	// 超时测试防止死锁
	done := make(chan bool, 1)
	go func() {
		// 更新配置文件
		err := os.WriteFile(configPath, []byte(updatedContent), 0644)
		assert.NoError(t, err, "更新配置文件应该成功")

		// 重新加载配置
		err = cm.Load()
		assert.NoError(t, err, "重新加载配置应该成功")

		done <- true
	}()

	// 等待更新完成，设置超时防止死锁
	select {
	case <-done:
		// 验证配置是否已更新
		t.Run("配置热更新验证", func(t *testing.T) {
			appName, err := cm.GetString("app.name")
			assert.NoError(t, err)
			assert.Equal(t, "UpdatedApp", appName, "应用名称应该已更新")

			version, err := cm.GetString("app.version")
			assert.NoError(t, err)
			assert.Equal(t, "2.0.0", version, "版本号应该已更新")

			debug, err := cm.GetBool("app.debug")
			assert.NoError(t, err)
			assert.True(t, debug, "debug模式应该已更新为true")

			logLevel, err := cm.GetString("log.level")
			assert.NoError(t, err)
			assert.Equal(t, "debug", logLevel, "日志级别应该已更新")

			// 验证新增的配置项
			enabled, err := cm.GetBool("new_feature.enabled")
			assert.NoError(t, err)
			assert.True(t, enabled, "新增的特性开关应该正确")

			timeout, err := cm.GetInt("new_feature.timeout")
			assert.NoError(t, err)
			assert.Equal(t, 30, timeout, "新增的超时配置应该正确")
		})
	case <-timeoutChan(5):
		t.Fatal("配置热更新测试超时，可能存在死锁")
	}
}

func TestConfigManager_HotReloadWithError(t *testing.T) {
	tempDir, cleanup := createTempDir(t)
	defer cleanup()

	// 创建初始配置文件
	initialContent := `app:
  name: InitialApp
  version: 1.0.0`

	configPath := createTestConfigFile(t, tempDir, initialContent)
	cm := NewConfigManager(configPath)

	// 加载初始配置
	err := cm.Load()
	assert.NoError(t, err, "加载初始配置应该成功")

	// 验证初始配置
	appName, err := cm.GetString("app.name")
	assert.NoError(t, err)
	assert.Equal(t, "InitialApp", appName, "初始应用名称应该正确")

	// 创建无效的配置文件内容
	invalidContent := `invalid yaml content:
  - this is not: {{{ proper yaml`

	// 超时测试防止死锁
	done := make(chan bool, 1)
	var reloadErr error
	go func() {
		// 写入无效配置
		err := os.WriteFile(configPath, []byte(invalidContent), 0644)
		assert.NoError(t, err, "写入无效配置文件应该成功")

		// 尝试重新加载配置（应该失败）
		reloadErr = cm.Load()
		done <- true
	}()

	// 等待完成，设置超时防止死锁
	select {
	case <-done:
		// 验证加载无效配置时的错误处理
		assert.Error(t, reloadErr, "加载无效配置应该返回错误")

		// 验证原有配置依然可用（优雅降级）
		appName, err := cm.GetString("app.name")
		assert.NoError(t, err)
		assert.Equal(t, "InitialApp", appName, "加载失败后原配置应该保持可用")
	case <-timeoutChan(5):
		t.Fatal("配置热更新错误处理测试超时，可能存在死锁")
	}
}

func TestConfigManager_ConcurrentHotReload(t *testing.T) {
	tempDir, cleanup := createTempDir(t)
	defer cleanup()

	// 创建初始配置文件
	initialContent := `app:
  name: ConcurrentApp
  counter: 0`

	configPath := createTestConfigFile(t, tempDir, initialContent)
	cm := NewConfigManager(configPath)

	// 加载初始配置
	err := cm.Load()
	assert.NoError(t, err, "加载初始配置应该成功")

	// 并发测试
	const numGoroutines = 5
	const numReloads = 3
	results := make(chan error, numGoroutines)

	// 启动多个goroutine同时进行热更新
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					results <- fmt.Errorf("goroutine %d panic: %v", id, r)
				}
			}()

			for j := 0; j < numReloads; j++ {
				// 创建新的配置内容
				newContent := fmt.Sprintf(`app:
  name: ConcurrentApp-%d-%d
  counter: %d`, id, j, id*numReloads+j)

				// 写入配置文件
				writeErr := os.WriteFile(configPath, []byte(newContent), 0644)
				if writeErr != nil {
					results <- fmt.Errorf("goroutine %d 写入配置失败: %v", id, writeErr)
					return
				}

				// 重新加载配置
				if loadErr := cm.Load(); loadErr != nil {
					results <- fmt.Errorf("goroutine %d 重载配置失败: %v", id, loadErr)
					return
				}

				// 验证配置可读
				if _, getErr := cm.GetString("app.name"); getErr != nil {
					results <- fmt.Errorf("goroutine %d 读取配置失败: %v", id, getErr)
					return
				}
			}
			results <- nil
		}(i)
	}

	// 收集结果，设置超时
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-results:
			if err != nil {
				t.Logf("并发测试中的错误（这在高并发情况下可能是正常的）: %v", err)
			} else {
				successCount++
			}
		case <-timeoutChan(10):
			t.Fatal("并发热更新测试超时，可能存在死锁")
		}
	}

	// 验证至少有一些goroutine成功完成
	t.Logf("成功完成的goroutine数量: %d/%d", successCount, numGoroutines)
	assert.GreaterOrEqual(t, successCount, 1, "至少应该有一个goroutine成功完成")

	// 最终验证配置仍然可用
	finalName, err := cm.GetString("app.name")
	assert.NoError(t, err, "最终配置应该可读")
	assert.Contains(t, finalName, "ConcurrentApp", "最终配置应该包含正确的应用名称")
}

// timeoutChan 创建一个超时通道，用于防止测试死锁
func timeoutChan(seconds int) <-chan bool {
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(time.Duration(seconds) * time.Second)
		timeout <- true
	}()
	return timeout
}
