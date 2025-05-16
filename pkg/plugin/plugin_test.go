package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPluginManager 测试插件管理器的创建
func TestNewPluginManager(t *testing.T) {
	pm := NewPluginManager()
	assert.NotNil(t, pm, "插件管理器不应为nil")
	assert.NotNil(t, pm.plugins, "plugins map不应为nil")
	assert.Equal(t, 1, len(pm.pluginPaths), "默认应有一个插件路径")
	assert.Equal(t, "./plugins", pm.pluginPaths[0], "默认插件路径应为./plugins")
}

// TestAddPluginPath 测试添加插件路径
func TestAddPluginPath(t *testing.T) {
	pm := NewPluginManager()
	initialCount := len(pm.pluginPaths)

	pm.AddPluginPath("/test/path")
	assert.Equal(t, initialCount+1, len(pm.pluginPaths), "应该增加一个路径")
	assert.Equal(t, "/test/path", pm.pluginPaths[initialCount], "应该添加正确的路径")
}

// TestGetPlugin 测试获取插件
func TestGetPlugin(t *testing.T) {
	pm := NewPluginManager()

	// 添加一个测试插件
	testPlugin := &Plugin{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Description: "测试插件",
		Author:      "测试作者",
	}
	pm.plugins["test-plugin"] = testPlugin

	// 测试获取存在的插件
	plugin, err := pm.GetPlugin("test-plugin")
	assert.NoError(t, err, "获取存在的插件不应返回错误")
	assert.Equal(t, testPlugin, plugin, "应返回正确的插件")

	// 测试获取不存在的插件
	_, err = pm.GetPlugin("non-existent")
	assert.Error(t, err, "获取不存在的插件应返回错误")
	assert.Contains(t, err.Error(), "不存在", "错误信息应包含'不存在'")
}

// TestGetPlugins 测试获取所有插件
func TestGetPlugins(t *testing.T) {
	pm := NewPluginManager()

	// 添加测试插件
	testPlugins := []*Plugin{
		{Name: "plugin1", Version: "1.0.0"},
		{Name: "plugin2", Version: "1.0.0"},
	}

	for _, p := range testPlugins {
		pm.plugins[p.Name] = p
	}

	// 获取所有插件
	plugins := pm.GetPlugins()
	assert.Equal(t, len(testPlugins), len(plugins), "应返回正确数量的插件")

	// 验证返回的插件集合是否包含所有添加的插件
	pluginMap := make(map[string]*Plugin)
	for _, p := range plugins {
		pluginMap[p.Name] = p
	}

	for _, p := range testPlugins {
		assert.Contains(t, pluginMap, p.Name, "返回的插件集合应包含所有添加的插件")
	}
}

// TestUnloadPlugin 测试卸载插件
func TestUnloadPlugin(t *testing.T) {
	pm := NewPluginManager()

	// 添加测试插件
	testPlugin := &Plugin{
		Name:    "test-plugin",
		Version: "1.0.0",
	}
	pm.plugins["test-plugin"] = testPlugin

	// 测试卸载存在的插件
	err := pm.UnloadPlugin("test-plugin")
	assert.NoError(t, err, "卸载存在的插件不应返回错误")
	assert.NotContains(t, pm.plugins, "test-plugin", "插件应该被卸载")

	// 测试卸载不存在的插件
	err = pm.UnloadPlugin("non-existent")
	assert.Error(t, err, "卸载不存在的插件应返回错误")
	assert.Contains(t, err.Error(), "不存在", "错误信息应包含'不存在'")
}

// TestRegisterPluginCommands 测试注册插件命令
func TestRegisterPluginCommands(t *testing.T) {
	pm := NewPluginManager()

	// 创建根命令
	rootCmd := &cobra.Command{
		Use: "root",
	}

	// 创建测试插件命令
	cmd1 := &cobra.Command{Use: "cmd1"}
	cmd2 := &cobra.Command{Use: "cmd2"}

	// 添加测试插件
	testPlugin := &Plugin{
		Name:     "test-plugin",
		Version:  "1.0.0",
		Commands: []*cobra.Command{cmd1, cmd2},
	}
	pm.plugins["test-plugin"] = testPlugin

	// 注册命令
	pm.RegisterPluginCommands(rootCmd)

	// 验证命令是否被添加到根命令
	found := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		found[cmd.Use] = true
	}

	assert.True(t, found["cmd1"], "cmd1应该被添加到根命令")
	assert.True(t, found["cmd2"], "cmd2应该被添加到根命令")
}

// 创建模拟插件目录和文件进行测试
func setupMockPlugin(t *testing.T) (string, func()) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "plugin-test")
	require.NoError(t, err, "创建临时目录失败")

	// 创建插件目录
	pluginDir := filepath.Join(tmpDir, "test-plugin")
	err = os.Mkdir(pluginDir, 0755)
	require.NoError(t, err, "创建插件目录失败")

	// 创建插件元数据文件
	metadata := map[string]interface{}{
		"Name":        "test-plugin",
		"Version":     "1.0.0",
		"Description": "测试插件",
		"Author":      "测试作者",
	}

	metadataBytes, err := json.Marshal(metadata)
	require.NoError(t, err, "序列化元数据失败")

	err = os.WriteFile(filepath.Join(pluginDir, "plugin.json"), metadataBytes, 0644)
	require.NoError(t, err, "写入元数据文件失败")

	// 返回清理函数
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// TestLoadPlugins_PathDoesNotExist 测试加载不存在的插件路径
func TestLoadPlugins_PathDoesNotExist(t *testing.T) {
	pm := NewPluginManager()

	// 清空默认路径，添加不存在的路径
	pm.pluginPaths = []string{"/path/does/not/exist"}

	// 加载插件（应该成功但没有插件被加载）
	err := pm.LoadPlugins()
	assert.NoError(t, err, "加载不存在的路径不应返回错误")
	assert.Empty(t, pm.plugins, "不应加载任何插件")
}

// TestLoadPlugin_DirectoryStructure 测试加载具有正确目录结构的插件
func TestLoadPlugin_DirectoryStructure(t *testing.T) {
	// 创建模拟插件目录
	tmpDir, cleanup := setupMockPlugin(t)
	defer cleanup()

	pluginDir := filepath.Join(tmpDir, "test-plugin")

	// 创建假的.so文件，这只是为了测试目的
	// 注意：在实际运行时这会失败，因为它不是有效的Go插件
	soPath := filepath.Join(pluginDir, "plugin.so")
	err := os.WriteFile(soPath, []byte("mock plugin binary"), 0644)
	require.NoError(t, err, "创建模拟插件二进制文件失败")

	// 创建Plugin实例，它将用于测试目的
	pm := NewPluginManager()

	// 尝试加载插件（将失败，因为不是真正的插件，但会测试到元数据解析部分）
	err = pm.LoadPlugin(pluginDir)
	assert.Error(t, err, "加载非有效插件应该返回错误")
	assert.Contains(t, err.Error(), "插件二进制文件", "错误应该提及插件二进制问题")
}

// TestLoadPlugins_MultipleDirectories 测试从多个目录加载插件
func TestLoadPlugins_MultipleDirectories(t *testing.T) {
	// 创建带有多个插件目录的测试环境
	tmpDir, cleanup := setupMockPluginEnvironment(t)
	defer cleanup()

	pm := NewPluginManager()

	// 添加测试目录
	pm.AddPluginPath(tmpDir)

	// 尝试加载插件
	err := pm.LoadPlugins()
	assert.NoError(t, err, "LoadPlugins不应返回错误")

	// 验证扫描行为（不会加载任何插件，因为没有有效的.so文件）
	assert.Empty(t, pm.plugins, "不应加载任何插件，因为没有有效的.so文件")
}

// TestLoadPlugins_NestedDirectories 测试处理嵌套目录情况
func TestLoadPlugins_NestedDirectories(t *testing.T) {
	// 创建带有嵌套插件目录的测试环境
	tmpDir, cleanup := setupNestedPluginEnvironment(t)
	defer cleanup()

	pm := NewPluginManager()

	// 添加测试目录
	pm.AddPluginPath(tmpDir)

	// 尝试加载插件
	err := pm.LoadPlugins()
	assert.NoError(t, err, "LoadPlugins不应返回错误")
}

// setupMockPluginEnvironment 创建带有多个插件目录的测试环境
func setupMockPluginEnvironment(t *testing.T) (string, func()) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "plugins-test")
	require.NoError(t, err, "创建临时目录失败")

	// 创建多个插件目录
	for i := 1; i <= 3; i++ {
		pluginDir := filepath.Join(tmpDir, fmt.Sprintf("plugin-%d", i))
		err := os.Mkdir(pluginDir, 0755)
		require.NoError(t, err, "创建插件目录失败")

		// 创建插件元数据文件
		metadata := map[string]interface{}{
			"Name":        fmt.Sprintf("plugin-%d", i),
			"Version":     "1.0.0",
			"Description": fmt.Sprintf("测试插件 %d", i),
			"Author":      "测试作者",
		}

		metadataBytes, err := json.Marshal(metadata)
		require.NoError(t, err, "序列化元数据失败")

		err = os.WriteFile(filepath.Join(pluginDir, "plugin.json"), metadataBytes, 0644)
		require.NoError(t, err, "写入元数据文件失败")
	}

	// 创建一个没有plugin.json的目录，这个应该被跳过
	nonPluginDir := filepath.Join(tmpDir, "not-a-plugin")
	err = os.Mkdir(nonPluginDir, 0755)
	require.NoError(t, err, "创建非插件目录失败")

	// 返回清理函数
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// setupNestedPluginEnvironment 创建带有嵌套目录的测试环境
func setupNestedPluginEnvironment(t *testing.T) (string, func()) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "nested-plugins-test")
	require.NoError(t, err, "创建临时目录失败")

	// 创建父插件目录
	parentDir := filepath.Join(tmpDir, "parent-plugin")
	err = os.Mkdir(parentDir, 0755)
	require.NoError(t, err, "创建父插件目录失败")

	// 创建父目录的plugin.json
	parentMetadata := map[string]interface{}{
		"Name":        "parent-plugin",
		"Version":     "1.0.0",
		"Description": "父插件",
		"Author":      "测试作者",
	}

	parentMetadataBytes, err := json.Marshal(parentMetadata)
	require.NoError(t, err, "序列化父插件元数据失败")

	err = os.WriteFile(filepath.Join(parentDir, "plugin.json"), parentMetadataBytes, 0644)
	require.NoError(t, err, "写入父插件元数据文件失败")

	// 创建嵌套的子插件目录
	childDir := filepath.Join(parentDir, "child-plugin")
	err = os.Mkdir(childDir, 0755)
	require.NoError(t, err, "创建子插件目录失败")

	// 创建子目录的plugin.json
	childMetadata := map[string]interface{}{
		"Name":        "child-plugin",
		"Version":     "1.0.0",
		"Description": "子插件",
		"Author":      "测试作者",
	}

	childMetadataBytes, err := json.Marshal(childMetadata)
	require.NoError(t, err, "序列化子插件元数据失败")

	err = os.WriteFile(filepath.Join(childDir, "plugin.json"), childMetadataBytes, 0644)
	require.NoError(t, err, "写入子插件元数据文件失败")

	// 返回清理函数
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}
