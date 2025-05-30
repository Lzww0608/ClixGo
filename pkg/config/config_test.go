package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

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
