/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-03 19:10:00
* @Description: 配置管理功能的核心实现
 */

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// ConfigSource 表示配置源枚举
// 用于标识配置值的来源，支持多种配置源的优先级管理
type ConfigSource string

const (
	SourceDefault   ConfigSource = "default"   // 默认配置源
	SourceFile      ConfigSource = "file"      // 配置文件源
	SourceEnv       ConfigSource = "env"       // 环境变量源
	SourceFlag      ConfigSource = "flag"      // 命令行标志源
	SourceRemote    ConfigSource = "remote"    // 远程配置源
	SourceEncrypted ConfigSource = "encrypted" // 加密配置源
)

// ConfigValue 表示配置值的完整信息
// 包含值本身、来源、加密状态和元数据
type ConfigValue struct {
	Value       interface{}            // 配置值
	Source      ConfigSource           // 配置来源
	IsEncrypted bool                   // 是否为加密配置
	Metadata    map[string]interface{} // 配置元数据
}

// ConfigManager 配置管理器
// 负责管理多源配置、支持配置文件、环境变量、命令行参数等
// 提供线程安全的配置读写操作和配置变更通知
type ConfigManager struct {
	viper      *viper.Viper           // Viper配置管理器
	values     map[string]ConfigValue // 配置值存储
	mu         sync.RWMutex           // 读写锁保护并发访问
	configPath string                 // 配置文件路径
	profiles   []string               // 激活的配置文件列表
}

// 全局配置管理器实例
var instance *ConfigManager
var once sync.Once

// InitConfig 初始化全局配置管理器
//
// 参数:
//   - configPath: 配置文件路径，如果为空则使用默认路径
//
// 返回:
//   - error: 初始化错误，nil表示成功
//
// 该函数使用单例模式确保全局只有一个配置管理器实例
func InitConfig(configPath string) error {
	var err error
	once.Do(func() {
		if configPath == "" {
			userHomeDir, e := os.UserHomeDir()
			if e != nil {
				err = fmt.Errorf("获取用户目录失败: %v", e)
				return
			}
			configPath = filepath.Join(userHomeDir, ".clixgo", "config.yaml")
		}

		instance = NewConfigManager(configPath)
		err = instance.Load()
	})
	return err
}

// GetInstance 获取全局配置管理器实例
//
// 返回:
//   - *ConfigManager: 全局配置管理器实例
//
// 注意: 必须先调用InitConfig进行初始化，否则会panic
func GetInstance() *ConfigManager {
	if instance == nil {
		panic("配置未初始化，请先调用 InitConfig")
	}
	return instance
}

// NewConfigManager 创建新的配置管理器
//
// 参数:
//   - configPath: 配置文件完整路径
//
// 返回:
//   - *ConfigManager: 新创建的配置管理器实例
//
// 该函数会自动解析配置文件路径，设置Viper配置，并初始化默认值
func NewConfigManager(configPath string) *ConfigManager {
	viperInstance := viper.New()

	// 从配置文件路径中提取目录和文件名
	configDirectory := filepath.Dir(configPath)
	configFilename := filepath.Base(configPath)
	configExtension := filepath.Ext(configFilename)
	configBaseName := strings.TrimSuffix(configFilename, configExtension)

	// 设置viper配置
	viperInstance.SetConfigName(configBaseName)
	if configExtension != "" {
		// 移除开头的点号
		configType := strings.TrimPrefix(configExtension, ".")
		viperInstance.SetConfigType(configType)
	} else {
		viperInstance.SetConfigType("yaml") // 默认类型
	}
	viperInstance.AddConfigPath(configDirectory)

	configManager := &ConfigManager{
		viper:      viperInstance,
		values:     make(map[string]ConfigValue),
		configPath: configPath,
		profiles:   []string{"default"},
	}

	// 设置默认值
	configManager.setDefaults()

	return configManager
}

// setDefaults 设置默认配置值
// 为核心应用程序配置项设置合理的默认值
func (cm *ConfigManager) setDefaults() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 设置默认值
	defaultValues := map[string]interface{}{
		"app.name":        "ClixGo",
		"app.version":     "1.0.0",
		"app.environment": "development",
		"log.level":       "info",
		"log.format":      "text",
		"log.output":      "stdout",
	}

	for configKey, defaultValue := range defaultValues {
		cm.values[configKey] = ConfigValue{
			Value:       defaultValue,
			Source:      SourceDefault,
			IsEncrypted: false,
			Metadata:    make(map[string]interface{}),
		}
	}
}

// Load 加载配置数据
//
// 返回:
//   - error: 加载过程中的错误，nil表示成功
//
// 该函数按优先级顺序加载配置：默认值 < 配置文件 < 环境变量
func (cm *ConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 加载配置文件
	if err := cm.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("读取配置文件失败: %v", err)
		}
	}

	// 加载环境变量
	cm.viper.AutomaticEnv()
	cm.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 合并配置文件中的配置
	for _, configKey := range cm.viper.AllKeys() {
		configValue := cm.viper.Get(configKey)
		cm.values[configKey] = ConfigValue{
			Value:       configValue,
			Source:      SourceFile,
			IsEncrypted: false,
			Metadata:    make(map[string]interface{}),
		}
	}

	// 加载环境变量并覆盖文件配置
	for _, configKey := range cm.viper.AllKeys() {
		envVariableName := strings.ToUpper(strings.ReplaceAll(configKey, ".", "_"))
		if envValue := os.Getenv(envVariableName); envValue != "" {
			cm.values[configKey] = ConfigValue{
				Value:       envValue,
				Source:      SourceEnv,
				IsEncrypted: false,
				Metadata:    make(map[string]interface{}),
			}
		}
	}

	return nil
}

// Save 保存配置到文件
//
// 返回:
//   - error: 保存过程中的错误，nil表示成功
//
// 只保存来源为文件的配置项，确保不会保存环境变量或默认值
func (cm *ConfigManager) Save() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 创建配置目录
	if err := os.MkdirAll(filepath.Dir(cm.configPath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	// 准备配置数据
	configDataToSave := make(map[string]interface{})
	for configKey, configValue := range cm.values {
		if configValue.Source == SourceFile {
			configDataToSave[configKey] = configValue.Value
		}
	}

	// 保存到文件
	data, err := json.MarshalIndent(configDataToSave, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

// Get 获取配置值
//
// 参数:
//   - key: 配置项键名，支持点分隔符格式（如："app.name"）
//
// 返回:
//   - interface{}: 配置值
//   - error: 获取错误，nil表示成功
func (cm *ConfigManager) Get(key string) (interface{}, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configValue, exists := cm.values[key]
	if !exists {
		return nil, fmt.Errorf("配置项 %s 不存在", key)
	}

	return configValue.Value, nil
}

// Set 设置配置值
//
// 参数:
//   - key: 配置项键名
//   - value: 配置值
//   - source: 配置来源
//
// 该函数会立即更新内存中的配置值，但不会自动保存到文件
func (cm *ConfigManager) Set(key string, value interface{}, source ConfigSource) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.values[key] = ConfigValue{
		Value:       value,
		Source:      source,
		IsEncrypted: false,
		Metadata:    make(map[string]interface{}),
	}
}

// GetString 获取字符串类型的配置值
//
// 参数:
//   - key: 配置项键名
//
// 返回:
//   - string: 字符串值
//   - error: 获取错误或类型转换错误
func (cm *ConfigManager) GetString(key string) (string, error) {
	configValue, err := cm.Get(key)
	if err != nil {
		return "", err
	}

	stringValue, isString := configValue.(string)
	if !isString {
		return "", fmt.Errorf("配置项 %s 不是字符串类型", key)
	}

	return stringValue, nil
}

// GetInt 获取整数类型的配置值
//
// 参数:
//   - key: 配置项键名
//
// 返回:
//   - int: 整数值
//   - error: 获取错误或类型转换错误
func (cm *ConfigManager) GetInt(key string) (int, error) {
	configValue, err := cm.Get(key)
	if err != nil {
		return 0, err
	}

	intValue, isInt := configValue.(int)
	if !isInt {
		return 0, fmt.Errorf("配置项 %s 不是整数类型", key)
	}

	return intValue, nil
}

// GetBool 获取布尔类型的配置值
//
// 参数:
//   - key: 配置项键名
//
// 返回:
//   - bool: 布尔值
//   - error: 获取错误或类型转换错误
func (cm *ConfigManager) GetBool(key string) (bool, error) {
	configValue, err := cm.Get(key)
	if err != nil {
		return false, err
	}

	boolValue, isBool := configValue.(bool)
	if !isBool {
		return false, fmt.Errorf("配置项 %s 不是布尔类型", key)
	}

	return boolValue, nil
}

// GetSource 获取配置项的来源
//
// 参数:
//   - key: 配置项键名
//
// 返回:
//   - ConfigSource: 配置来源枚举值
//   - error: 获取错误，nil表示成功
func (cm *ConfigManager) GetSource(key string) (ConfigSource, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configValue, exists := cm.values[key]
	if !exists {
		return "", fmt.Errorf("配置项 %s 不存在", key)
	}

	return configValue.Source, nil
}

// IsEncrypted 检查配置项是否为加密配置
//
// 参数:
//   - key: 配置项键名
//
// 返回:
//   - bool: true表示加密，false表示未加密
//   - error: 获取错误，nil表示成功
func (cm *ConfigManager) IsEncrypted(key string) (bool, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configValue, exists := cm.values[key]
	if !exists {
		return false, fmt.Errorf("配置项 %s 不存在", key)
	}

	return configValue.IsEncrypted, nil
}

// SetProfile 设置当前激活的配置环境
//
// 参数:
//   - profile: 配置环境名称（如："development", "production"）
//
// 设置后会按优先级使用：指定环境 > 默认环境
func (cm *ConfigManager) SetProfile(profile string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.profiles = []string{profile, "default"}
}

// GetProfiles 获取当前激活的所有配置环境
//
// 返回:
//   - []string: 配置环境列表，按优先级排序
func (cm *ConfigManager) GetProfiles() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.profiles
}

// GetConfigPath 获取配置文件路径
//
// 返回:
//   - string: 配置文件的完整路径
func (cm *ConfigManager) GetConfigPath() string {
	return cm.configPath
}
