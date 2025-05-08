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

// ConfigSource 表示配置源
type ConfigSource string

const (
	SourceDefault   ConfigSource = "default"
	SourceFile      ConfigSource = "file"
	SourceEnv       ConfigSource = "env"
	SourceFlag      ConfigSource = "flag"
	SourceRemote    ConfigSource = "remote"
	SourceEncrypted ConfigSource = "encrypted"
)

// ConfigValue 表示配置值
type ConfigValue struct {
	Value       interface{}
	Source      ConfigSource
	IsEncrypted bool
	Metadata    map[string]interface{}
}

// ConfigManager 管理配置
type ConfigManager struct {
	viper      *viper.Viper
	values     map[string]ConfigValue
	mu         sync.RWMutex
	configPath string
	profiles   []string
}

// 全局配置管理器实例
var instance *ConfigManager
var once sync.Once

// InitConfig 初始化全局配置
func InitConfig(configPath string) error {
	var err error
	once.Do(func() {
		if configPath == "" {
			homeDir, e := os.UserHomeDir()
			if e != nil {
				err = fmt.Errorf("获取用户目录失败: %v", e)
				return
			}
			configPath = filepath.Join(homeDir, ".clixgo", "config.yaml")
		}

		instance = NewConfigManager(configPath)
		err = instance.Load()
	})
	return err
}

// GetInstance 获取全局配置实例
func GetInstance() *ConfigManager {
	if instance == nil {
		panic("配置未初始化，请先调用 InitConfig")
	}
	return instance
}

// NewConfigManager 创建新的配置管理器
func NewConfigManager(configPath string) *ConfigManager {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)

	cm := &ConfigManager{
		viper:      v,
		values:     make(map[string]ConfigValue),
		configPath: configPath,
		profiles:   []string{"default"},
	}

	// 设置默认值
	cm.setDefaults()

	return cm
}

// setDefaults 设置默认配置
func (cm *ConfigManager) setDefaults() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 设置默认值
	defaults := map[string]interface{}{
		"app.name":        "ClixGo",
		"app.version":     "1.0.0",
		"app.environment": "development",
		"log.level":       "info",
		"log.format":      "text",
		"log.output":      "stdout",
	}

	for key, value := range defaults {
		cm.values[key] = ConfigValue{
			Value:       value,
			Source:      SourceDefault,
			IsEncrypted: false,
			Metadata:    make(map[string]interface{}),
		}
	}
}

// Load 加载配置
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

	// 合并配置
	for _, key := range cm.viper.AllKeys() {
		value := cm.viper.Get(key)
		cm.values[key] = ConfigValue{
			Value:       value,
			Source:      SourceFile,
			IsEncrypted: false,
			Metadata:    make(map[string]interface{}),
		}
	}

	// 加载环境变量
	for _, key := range cm.viper.AllKeys() {
		envKey := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
		if envValue := os.Getenv(envKey); envValue != "" {
			cm.values[key] = ConfigValue{
				Value:       envValue,
				Source:      SourceEnv,
				IsEncrypted: false,
				Metadata:    make(map[string]interface{}),
			}
		}
	}

	return nil
}

// Save 保存配置
func (cm *ConfigManager) Save() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 创建配置目录
	if err := os.MkdirAll(filepath.Dir(cm.configPath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	// 准备配置数据
	configData := make(map[string]interface{})
	for key, value := range cm.values {
		if value.Source == SourceFile {
			configData[key] = value.Value
		}
	}

	// 保存到文件
	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

// Get 获取配置值
func (cm *ConfigManager) Get(key string) (interface{}, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	value, ok := cm.values[key]
	if !ok {
		return nil, fmt.Errorf("配置项 %s 不存在", key)
	}

	return value.Value, nil
}

// Set 设置配置值
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

// GetString 获取字符串配置值
func (cm *ConfigManager) GetString(key string) (string, error) {
	value, err := cm.Get(key)
	if err != nil {
		return "", err
	}

	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("配置项 %s 不是字符串类型", key)
	}

	return str, nil
}

// GetInt 获取整数配置值
func (cm *ConfigManager) GetInt(key string) (int, error) {
	value, err := cm.Get(key)
	if err != nil {
		return 0, err
	}

	num, ok := value.(int)
	if !ok {
		return 0, fmt.Errorf("配置项 %s 不是整数类型", key)
	}

	return num, nil
}

// GetBool 获取布尔配置值
func (cm *ConfigManager) GetBool(key string) (bool, error) {
	value, err := cm.Get(key)
	if err != nil {
		return false, err
	}

	b, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("配置项 %s 不是布尔类型", key)
	}

	return b, nil
}

// GetSource 获取配置源
func (cm *ConfigManager) GetSource(key string) (ConfigSource, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	value, ok := cm.values[key]
	if !ok {
		return "", fmt.Errorf("配置项 %s 不存在", key)
	}

	return value.Source, nil
}

// IsEncrypted 检查配置是否加密
func (cm *ConfigManager) IsEncrypted(key string) (bool, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	value, ok := cm.values[key]
	if !ok {
		return false, fmt.Errorf("配置项 %s 不存在", key)
	}

	return value.IsEncrypted, nil
}

// SetProfile 设置当前配置环境
func (cm *ConfigManager) SetProfile(profile string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.profiles = []string{profile, "default"}
}

// GetProfiles 获取所有配置环境
func (cm *ConfigManager) GetProfiles() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.profiles
}

// GetConfigPath 获取配置文件路径
func (cm *ConfigManager) GetConfigPath() string {
	return cm.configPath
}
