package alias

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// 用于测试的临时别名文件
var testAliasFile string

// 测试初始化函数
func TestMain(m *testing.M) {
	// 设置测试环境
	tempDir, err := os.MkdirTemp("", "alias_test")
	if err != nil {
		panic("无法创建临时目录: " + err.Error())
	}
	defer os.RemoveAll(tempDir)

	// 保存原始文件路径和别名映射
	originalAliasFile := aliasFile
	originalAliases := aliases

	// 设置测试文件路径
	testAliasFile = filepath.Join(tempDir, "test_aliases.json")
	aliasFile = testAliasFile
	aliases = make(map[string]string)

	// 运行测试
	exitCode := m.Run()

	// 恢复原始设置
	aliasFile = originalAliasFile
	aliases = originalAliases

	// 退出
	os.Exit(exitCode)
}

// 辅助函数：准备测试环境
func setupTest() {
	aliases = make(map[string]string)
	os.Remove(testAliasFile) // 确保文件不存在
}

// 测试InitAliases函数 - 文件不存在的情况
func TestInitAliasesFileNotExists(t *testing.T) {
	setupTest()

	err := InitAliases()
	if err != nil {
		t.Errorf("当别名文件不存在时，InitAliases应该返回nil，而不是错误: %v", err)
	}

	if len(aliases) != 0 {
		t.Errorf("当别名文件不存在时，别名映射应该为空，实际大小: %d", len(aliases))
	}
}

// 测试InitAliases函数 - 文件存在且有效的情况
func TestInitAliasesWithValidFile(t *testing.T) {
	setupTest()

	// 创建测试数据
	testAliases := []Alias{
		{Name: "test1", Command: "echo test1"},
		{Name: "test2", Command: "echo test2"},
	}

	// 写入测试文件
	data, _ := json.MarshalIndent(testAliases, "", "  ")
	err := os.WriteFile(testAliasFile, data, 0644)
	if err != nil {
		t.Fatalf("无法写入测试文件: %v", err)
	}

	// 测试初始化
	err = InitAliases()
	if err != nil {
		t.Errorf("InitAliases应该成功初始化，而不是返回错误: %v", err)
	}

	// 验证结果
	if len(aliases) != 2 {
		t.Errorf("别名映射应该有2个条目，实际有: %d", len(aliases))
	}

	if aliases["test1"] != "echo test1" {
		t.Errorf("别名test1的值不正确，期望: %s, 实际: %s", "echo test1", aliases["test1"])
	}

	if aliases["test2"] != "echo test2" {
		t.Errorf("别名test2的值不正确，期望: %s, 实际: %s", "echo test2", aliases["test2"])
	}
}

// 测试InitAliases函数 - 文件损坏的情况
func TestInitAliasesWithInvalidFile(t *testing.T) {
	setupTest()

	// 写入无效的JSON数据
	err := os.WriteFile(testAliasFile, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("无法写入测试文件: %v", err)
	}

	// 测试初始化
	err = InitAliases()
	if err == nil {
		t.Errorf("对于无效的JSON文件，InitAliases应该返回错误")
	}
}

// 测试SaveAliases函数
func TestSaveAliases(t *testing.T) {
	setupTest()

	// 添加测试数据
	aliases["test1"] = "echo test1"
	aliases["test2"] = "echo test2"

	// 保存别名
	err := SaveAliases()
	if err != nil {
		t.Errorf("SaveAliases应该成功保存，而不是返回错误: %v", err)
	}

	// 读取并验证保存的数据
	data, err := os.ReadFile(testAliasFile)
	if err != nil {
		t.Fatalf("无法读取保存的别名文件: %v", err)
	}

	var savedAliases []Alias
	err = json.Unmarshal(data, &savedAliases)
	if err != nil {
		t.Fatalf("无法解析保存的别名数据: %v", err)
	}

	if len(savedAliases) != 2 {
		t.Errorf("保存的别名数据应该有2个条目，实际有: %d", len(savedAliases))
	}

	// 检查是否所有别名都正确保存
	foundTest1 := false
	foundTest2 := false
	for _, alias := range savedAliases {
		if alias.Name == "test1" && alias.Command == "echo test1" {
			foundTest1 = true
		}
		if alias.Name == "test2" && alias.Command == "echo test2" {
			foundTest2 = true
		}
	}

	if !foundTest1 {
		t.Errorf("未找到保存的test1别名或其值不正确")
	}
	if !foundTest2 {
		t.Errorf("未找到保存的test2别名或其值不正确")
	}
}

// 测试AddAlias函数
func TestAddAlias(t *testing.T) {
	setupTest()

	// 测试添加有效别名
	err := AddAlias("test", "echo test")
	if err != nil {
		t.Errorf("添加有效别名应该成功，而不是返回错误: %v", err)
	}

	if aliases["test"] != "echo test" {
		t.Errorf("别名值不正确，期望: %s, 实际: %s", "echo test", aliases["test"])
	}

	// 测试添加包含空格的无效别名
	err = AddAlias("invalid alias", "echo invalid")
	if err == nil {
		t.Errorf("添加包含空格的别名应该返回错误")
	}
}

// 测试RemoveAlias函数
func TestRemoveAlias(t *testing.T) {
	setupTest()

	// 添加测试数据
	aliases["test"] = "echo test"

	// 测试移除存在的别名
	err := RemoveAlias("test")
	if err != nil {
		t.Errorf("移除存在的别名应该成功，而不是返回错误: %v", err)
	}

	if _, exists := aliases["test"]; exists {
		t.Errorf("别名应该已被移除")
	}

	// 测试移除不存在的别名
	err = RemoveAlias("nonexistent")
	if err == nil {
		t.Errorf("移除不存在的别名应该返回错误")
	}
}

// 测试GetAlias函数
func TestGetAlias(t *testing.T) {
	setupTest()

	// 添加测试数据
	aliases["test"] = "echo test"

	// 测试获取存在的别名
	command, exists := GetAlias("test")
	if !exists {
		t.Errorf("应该找到别名test")
	}
	if command != "echo test" {
		t.Errorf("别名值不正确，期望: %s, 实际: %s", "echo test", command)
	}

	// 测试获取不存在的别名
	_, exists = GetAlias("nonexistent")
	if exists {
		t.Errorf("不应该找到不存在的别名")
	}
}

// 测试ListAliases函数
func TestListAliases(t *testing.T) {
	setupTest()

	// 添加测试数据
	aliases["test1"] = "echo test1"
	aliases["test2"] = "echo test2"

	// 测试列出别名
	aliasList := ListAliases()
	if len(aliasList) != 2 {
		t.Errorf("别名列表应该有2个条目，实际有: %d", len(aliasList))
	}

	if aliasList["test1"] != "echo test1" {
		t.Errorf("别名test1的值不正确，期望: %s, 实际: %s", "echo test1", aliasList["test1"])
	}

	if aliasList["test2"] != "echo test2" {
		t.Errorf("别名test2的值不正确，期望: %s, 实际: %s", "echo test2", aliasList["test2"])
	}
}

// 测试ExpandCommand函数
func TestExpandCommand(t *testing.T) {
	setupTest()

	// 添加测试数据
	aliases["ll"] = "ls -la"

	// 测试展开存在的别名
	expanded := ExpandCommand("ll /tmp")
	expected := "ls -la /tmp"
	if expanded != expected {
		t.Errorf("命令展开不正确，期望: %s, 实际: %s", expected, expanded)
	}

	// 测试展开不存在的别名
	nonExistingCommand := "unknown arg1 arg2"
	expanded = ExpandCommand(nonExistingCommand)
	if expanded != nonExistingCommand {
		t.Errorf("不存在的别名展开不正确，期望: %s, 实际: %s", nonExistingCommand, expanded)
	}

	// 测试空命令
	expanded = ExpandCommand("")
	if expanded != "" {
		t.Errorf("空命令展开不正确，期望空字符串，实际: %s", expanded)
	}
}

// 测试别名目录创建功能
func TestAliasDirectoryCreation(t *testing.T) {
	setupTest()

	// 删除测试目录如果存在
	testDir := filepath.Dir(testAliasFile)
	os.RemoveAll(testDir)

	// 确保目录不存在
	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Fatalf("测试前未能成功删除测试目录")
	}

	// 先初始化别名系统，这会创建目录
	err := InitAliases()
	if err != nil {
		t.Errorf("初始化别名系统失败: %v", err)
	}

	// 测试添加别名
	err = AddAlias("test", "echo test")
	if err != nil {
		t.Errorf("添加别名失败: %v", err)
	}

	// 验证目录已创建
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Errorf("添加别名后，目录应该被创建，但未找到")
	}

	// 验证别名文件已创建
	if _, err := os.Stat(testAliasFile); os.IsNotExist(err) {
		t.Errorf("添加别名后，别名文件应该被创建，但未找到")
	}
}

// 测试别名配置持久化路径
func TestAliasFilePath(t *testing.T) {
	setupTest()

	// 测试路径中包含正确的目录结构
	if filepath.Base(filepath.Dir(testAliasFile)) != filepath.Base(filepath.Dir(aliasFile)) {
		t.Errorf("别名文件路径目录结构不正确，期望包含 %s 目录", filepath.Base(filepath.Dir(aliasFile)))
	}

	// 测试别名文件名称
	if filepath.Base(testAliasFile) != "test_aliases.json" {
		t.Errorf("测试别名文件名称不正确，期望 %s，实际 %s", "test_aliases.json", filepath.Base(testAliasFile))
	}
}
