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

	for _, path := range pm.pluginPaths {
		err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() && d.Name() != "." && d.Name() != ".." {
				if err := pm.LoadPlugin(path); err != nil {
					fmt.Printf("加载插件 %s 失败: %v\n", path, err)
				}
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("遍历插件目录失败: %v", err)
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