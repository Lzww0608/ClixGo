package plugin

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/spf13/cobra"
)

// Plugin 表示一个插件
type Plugin struct {
	Name        string
	Version     string
	Description string
	Author      string
	Commands    []*cobra.Command
	Config      map[string]interface{}
}

// PluginManager 管理插件的加载和执行
type PluginManager struct {
	plugins     map[string]*Plugin
	pluginPaths []string
	mu          sync.RWMutex
}

// NewPluginManager 创建新的插件管理器
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins:     make(map[string]*Plugin),
		pluginPaths: []string{"./plugins"},
	}
}

// AddPluginPath 添加插件搜索路径
func (pm *PluginManager) AddPluginPath(path string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.pluginPaths = append(pm.pluginPaths, path)
}

// LoadPlugin 加载插件
func (pm *PluginManager) LoadPlugin(path string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	return pm.loadPluginInternal(path)
}

// loadPluginInternal 内部加载插件方法（无锁版本）
func (pm *PluginManager) loadPluginInternal(path string) error {
	// 读取插件元数据
	metadataPath := filepath.Join(path, "plugin.json")
	metadataData, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("读取插件元数据失败: %v", err)
	}

	var p Plugin
	if err := json.Unmarshal(metadataData, &p); err != nil {
		return fmt.Errorf("解析插件元数据失败: %v", err)
	}

	// 加载插件二进制文件
	pluginPath := filepath.Join(path, "plugin.so")
	plug, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("加载插件二进制文件失败: %v", err)
	}

	// 获取插件命令
	commandsSymbol, err := plug.Lookup("Commands")
	if err != nil {
		return fmt.Errorf("获取插件命令失败: %v", err)
	}

	commands, ok := commandsSymbol.(*[]*cobra.Command)
	if !ok {
		return fmt.Errorf("插件命令类型错误")
	}

	p.Commands = *commands
	pm.plugins[p.Name] = &p

	return nil
}

// LoadPlugins 加载所有插件
func (pm *PluginManager) LoadPlugins() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, basePath := range pm.pluginPaths {
		// 检查路径是否存在
		if _, err := os.Stat(basePath); os.IsNotExist(err) {
			// 如果路径不存在，跳过但不报错
			continue
		}

		err := filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// 只处理目录，跳过basePath本身、"."和".."
			if !d.IsDir() || path == basePath || d.Name() == "." || d.Name() == ".." {
				return nil
			}

			// 检查这个目录是否是一个有效的插件目录（含有plugin.json）
			metadataPath := filepath.Join(path, "plugin.json")
			if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
				return nil // 不是插件目录，继续遍历
			}

			// 看起来是一个插件目录，尝试加载
			if err := pm.loadPluginInternal(path); err != nil {
				fmt.Printf("加载插件 %s 失败: %v\n", path, err)
				// 继续尝试其他插件，不中断遍历
			} else {
				// 成功加载插件，跳过其子目录
				return filepath.SkipDir
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("遍历插件目录 %s 失败: %v", basePath, err)
		}
	}

	return nil
}

// GetPlugin 获取插件
func (pm *PluginManager) GetPlugin(name string) (*Plugin, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, ok := pm.plugins[name]
	if !ok {
		return nil, fmt.Errorf("插件 %s 不存在", name)
	}

	return plugin, nil
}

// GetPlugins 获取所有插件
func (pm *PluginManager) GetPlugins() []*Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins := make([]*Plugin, 0, len(pm.plugins))
	for _, plugin := range pm.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// UnloadPlugin 卸载插件
func (pm *PluginManager) UnloadPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, ok := pm.plugins[name]; !ok {
		return fmt.Errorf("插件 %s 不存在", name)
	}

	delete(pm.plugins, name)
	return nil
}

// RegisterPluginCommands 注册插件命令
func (pm *PluginManager) RegisterPluginCommands(rootCmd *cobra.Command) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, plugin := range pm.plugins {
		for _, cmd := range plugin.Commands {
			rootCmd.AddCommand(cmd)
		}
	}
}
