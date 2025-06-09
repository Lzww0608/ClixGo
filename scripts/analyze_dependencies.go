package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ModuleDependency 表示模块依赖关系
type ModuleDependency struct {
	Module       string
	Dependencies []string
	ExternalDeps []string
}

// DependencyAnalyzer 依赖关系分析器
type DependencyAnalyzer struct {
	ProjectPath   string
	ModuleName    string
	Dependencies  map[string]*ModuleDependency
	circularPaths [][]string
}

// NewDependencyAnalyzer 创建新的依赖分析器
func NewDependencyAnalyzer(projectPath, moduleName string) *DependencyAnalyzer {
	return &DependencyAnalyzer{
		ProjectPath:  projectPath,
		ModuleName:   moduleName,
		Dependencies: make(map[string]*ModuleDependency),
	}
}

// Analyze 分析依赖关系
func (da *DependencyAnalyzer) Analyze() error {
	pkgPath := filepath.Join(da.ProjectPath, "pkg")
	cmdPath := filepath.Join(da.ProjectPath, "cmd")

	// 分析pkg目录下的包
	if err := da.analyzeDirectory(pkgPath, "pkg"); err != nil {
		return fmt.Errorf("分析pkg目录失败: %v", err)
	}

	// 分析cmd目录下的包
	if err := da.analyzeDirectory(cmdPath, "cmd"); err != nil {
		return fmt.Errorf("分析cmd目录失败: %v", err)
	}

	// 检测循环依赖
	da.detectCircularDependencies()

	return nil
}

// analyzeDirectory 分析指定目录
func (da *DependencyAnalyzer) analyzeDirectory(dirPath, prefix string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// 获取包名
		relPath, _ := filepath.Rel(da.ProjectPath, path)
		packagePath := filepath.Dir(relPath)
		modulePath := strings.Replace(packagePath, string(filepath.Separator), "/", -1)

		if _, exists := da.Dependencies[modulePath]; !exists {
			da.Dependencies[modulePath] = &ModuleDependency{
				Module:       modulePath,
				Dependencies: []string{},
				ExternalDeps: []string{},
			}
		}

		// 解析文件的import语句
		deps, err := da.parseImports(path)
		if err != nil {
			return fmt.Errorf("解析文件 %s 失败: %v", path, err)
		}

		// 合并依赖
		da.mergeDependencies(modulePath, deps)

		return nil
	})
}

// parseImports 解析文件中的import语句
func (da *DependencyAnalyzer) parseImports(filePath string) ([]string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	var imports []string
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		imports = append(imports, importPath)
	}

	return imports, nil
}

// mergeDependencies 合并依赖关系
func (da *DependencyAnalyzer) mergeDependencies(modulePath string, imports []string) {
	dep := da.Dependencies[modulePath]

	for _, imp := range imports {
		if strings.HasPrefix(imp, da.ModuleName+"/pkg/") {
			// 内部依赖
			internalDep := strings.TrimPrefix(imp, da.ModuleName+"/")
			if !contains(dep.Dependencies, internalDep) {
				dep.Dependencies = append(dep.Dependencies, internalDep)
			}
		} else if strings.HasPrefix(imp, da.ModuleName+"/cmd/") {
			// 命令行依赖
			cmdDep := strings.TrimPrefix(imp, da.ModuleName+"/")
			if !contains(dep.Dependencies, cmdDep) {
				dep.Dependencies = append(dep.Dependencies, cmdDep)
			}
		} else if !isStandardLibrary(imp) {
			// 外部依赖
			if !contains(dep.ExternalDeps, imp) {
				dep.ExternalDeps = append(dep.ExternalDeps, imp)
			}
		}
	}
}

// detectCircularDependencies 检测循环依赖
func (da *DependencyAnalyzer) detectCircularDependencies() {
	visited := make(map[string]bool)
	stack := make(map[string]bool)
	path := []string{}

	for module := range da.Dependencies {
		if !visited[module] {
			da.dfsCircular(module, visited, stack, path)
		}
	}
}

// dfsCircular DFS检测循环依赖
func (da *DependencyAnalyzer) dfsCircular(module string, visited, stack map[string]bool, path []string) {
	visited[module] = true
	stack[module] = true
	path = append(path, module)

	if dep, exists := da.Dependencies[module]; exists {
		for _, dependency := range dep.Dependencies {
			if stack[dependency] {
				// 找到循环依赖
				circularPath := make([]string, 0)
				found := false
				for _, p := range path {
					if p == dependency {
						found = true
					}
					if found {
						circularPath = append(circularPath, p)
					}
				}
				circularPath = append(circularPath, dependency)
				da.circularPaths = append(da.circularPaths, circularPath)
			} else if !visited[dependency] {
				da.dfsCircular(dependency, visited, stack, path)
			}
		}
	}

	stack[module] = false
	path = path[:len(path)-1]
}

// PrintDependencyGraph 打印依赖关系图
func (da *DependencyAnalyzer) PrintDependencyGraph() {
	fmt.Println("# ClixGo 项目依赖关系分析")
	fmt.Println()

	// 按模块名排序
	var modules []string
	for module := range da.Dependencies {
		modules = append(modules, module)
	}
	sort.Strings(modules)

	fmt.Println("## 模块依赖关系详情")
	fmt.Println()

	for _, module := range modules {
		dep := da.Dependencies[module]
		fmt.Printf("### %s\n", module)

		if len(dep.Dependencies) > 0 {
			fmt.Println("**内部依赖:**")
			sort.Strings(dep.Dependencies)
			for _, d := range dep.Dependencies {
				fmt.Printf("- %s\n", d)
			}
		}

		if len(dep.ExternalDeps) > 0 {
			fmt.Println("**外部依赖:**")
			sort.Strings(dep.ExternalDeps)
			for _, d := range dep.ExternalDeps {
				fmt.Printf("- %s\n", d)
			}
		}
		fmt.Println()
	}
}

// PrintCircularDependencies 打印循环依赖
func (da *DependencyAnalyzer) PrintCircularDependencies() {
	if len(da.circularPaths) == 0 {
		fmt.Println("## ✅ 循环依赖检查")
		fmt.Println("未发现循环依赖问题")
		return
	}

	fmt.Println("## ⚠️ 发现循环依赖")
	fmt.Println()

	for i, path := range da.circularPaths {
		fmt.Printf("### 循环依赖 %d:\n", i+1)
		fmt.Printf("%s\n", strings.Join(path, " → "))
		fmt.Println()
	}
}

// GenerateMermaidDiagram 生成Mermaid依赖图
func (da *DependencyAnalyzer) GenerateMermaidDiagram() string {
	var builder strings.Builder

	builder.WriteString("graph TD\n")

	// 生成节点定义
	coreModules := []string{"config", "logger", "errors", "utils"}
	uiModules := []string{"terminal", "ui"}
	networkModules := []string{"network", "performance"}
	toolModules := []string{"commands", "text", "filesystem", "security"}

	// 为不同类型的模块使用不同的样式
	for _, module := range coreModules {
		if _, exists := da.Dependencies["pkg/"+module]; exists {
			builder.WriteString(fmt.Sprintf("    %s[%s - 核心模块]\n",
				strings.Replace(module, "/", "_", -1), module))
		}
	}

	for _, module := range uiModules {
		if _, exists := da.Dependencies["pkg/"+module]; exists {
			builder.WriteString(fmt.Sprintf("    %s[%s - UI模块]\n",
				strings.Replace(module, "/", "_", -1), module))
		}
	}

	for _, module := range networkModules {
		if _, exists := da.Dependencies["pkg/"+module]; exists {
			builder.WriteString(fmt.Sprintf("    %s[%s - 网络模块]\n",
				strings.Replace(module, "/", "_", -1), module))
		}
	}

	for _, module := range toolModules {
		if _, exists := da.Dependencies["pkg/"+module]; exists {
			builder.WriteString(fmt.Sprintf("    %s[%s - 工具模块]\n",
				strings.Replace(module, "/", "_", -1), module))
		}
	}

	// 生成依赖关系
	for module, dep := range da.Dependencies {
		if !strings.HasPrefix(module, "pkg/") {
			continue
		}

		moduleNode := strings.Replace(strings.TrimPrefix(module, "pkg/"), "/", "_", -1)

		for _, dependency := range dep.Dependencies {
			if strings.HasPrefix(dependency, "pkg/") {
				depNode := strings.Replace(strings.TrimPrefix(dependency, "pkg/"), "/", "_", -1)
				builder.WriteString(fmt.Sprintf("    %s --> %s\n", moduleNode, depNode))
			}
		}
	}

	return builder.String()
}

// PrintModuleDecouplingPlan 打印模块解耦计划
func (da *DependencyAnalyzer) PrintModuleDecouplingPlan() {
	fmt.Println("## 🔧 模块解耦优化建议")
	fmt.Println()

	// 分析高耦合模块
	dependencyCount := make(map[string]int)
	dependentCount := make(map[string]int)

	for module, dep := range da.Dependencies {
		dependencyCount[module] = len(dep.Dependencies)

		for _, dependency := range dep.Dependencies {
			dependentCount[dependency]++
		}
	}

	fmt.Println("### 1. 高依赖模块 (需要重构)")
	for module, count := range dependencyCount {
		if count > 5 {
			fmt.Printf("- **%s**: %d个依赖\n", module, count)
		}
	}
	fmt.Println()

	fmt.Println("### 2. 被高度依赖的模块 (需要稳定化)")
	for module, count := range dependentCount {
		if count > 3 {
			fmt.Printf("- **%s**: 被%d个模块依赖\n", module, count)
		}
	}
	fmt.Println()

	fmt.Println("### 3. 建议的解耦策略")
	fmt.Println("- **接口抽象**: 为被高度依赖的模块定义接口")
	fmt.Println("- **依赖注入**: 使用依赖注入减少直接依赖")
	fmt.Println("- **事件驱动**: 使用事件机制解耦模块间通信")
	fmt.Println("- **分层架构**: 建立清晰的分层结构")
	fmt.Println()
}

// contains 检查切片是否包含指定元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// isStandardLibrary 判断是否为标准库
func isStandardLibrary(importPath string) bool {
	standardLibs := []string{
		"fmt", "os", "path", "strings", "time", "sync", "context",
		"net", "net/http", "encoding/json", "io", "bufio", "bytes",
		"errors", "log", "runtime", "syscall", "unsafe", "reflect",
		"go/ast", "go/parser", "go/token", "sort", "strconv",
	}

	for _, lib := range standardLibs {
		if strings.HasPrefix(importPath, lib) {
			return true
		}
	}

	return false
}

func main() {
	projectPath := "."
	moduleName := "github.com/Lzww0608/ClixGo"

	analyzer := NewDependencyAnalyzer(projectPath, moduleName)

	fmt.Println("正在分析项目依赖关系...")
	if err := analyzer.Analyze(); err != nil {
		fmt.Printf("分析失败: %v\n", err)
		os.Exit(1)
	}

	// 打印依赖关系
	analyzer.PrintDependencyGraph()

	// 检查循环依赖
	analyzer.PrintCircularDependencies()

	// 打印解耦建议
	analyzer.PrintModuleDecouplingPlan()

	// 生成Mermaid图表
	fmt.Println("## 📊 依赖关系图表 (Mermaid)")
	fmt.Println()
	fmt.Println("```mermaid")
	fmt.Println(analyzer.GenerateMermaidDiagram())
	fmt.Println("```")

	// 保存到文件
	output := "dependency_analysis.md"
	file, err := os.Create(output)
	if err != nil {
		fmt.Printf("创建输出文件失败: %v\n", err)
		return
	}
	defer file.Close()

	// 创建一个字符串构建器来收集所有输出
	var content strings.Builder

	// 手动生成内容
	content.WriteString("# ClixGo 项目依赖关系分析\n\n")
	content.WriteString("## 模块依赖关系详情\n\n")

	// 按模块名排序
	var modules []string
	for module := range analyzer.Dependencies {
		modules = append(modules, module)
	}
	sort.Strings(modules)

	for _, module := range modules {
		dep := analyzer.Dependencies[module]
		content.WriteString(fmt.Sprintf("### %s\n", module))

		if len(dep.Dependencies) > 0 {
			content.WriteString("**内部依赖:**\n")
			sort.Strings(dep.Dependencies)
			for _, d := range dep.Dependencies {
				content.WriteString(fmt.Sprintf("- %s\n", d))
			}
		}

		if len(dep.ExternalDeps) > 0 {
			content.WriteString("**外部依赖:**\n")
			sort.Strings(dep.ExternalDeps)
			for _, d := range dep.ExternalDeps {
				content.WriteString(fmt.Sprintf("- %s\n", d))
			}
		}
		content.WriteString("\n")
	}

	// 循环依赖检查
	if len(analyzer.circularPaths) == 0 {
		content.WriteString("## ✅ 循环依赖检查\n")
		content.WriteString("未发现循环依赖问题\n")
	} else {
		content.WriteString("## ⚠️ 发现循环依赖\n\n")
		for i, path := range analyzer.circularPaths {
			content.WriteString(fmt.Sprintf("### 循环依赖 %d:\n", i+1))
			content.WriteString(fmt.Sprintf("%s\n\n", strings.Join(path, " → ")))
		}
	}

	// 模块解耦建议
	content.WriteString("## 🔧 模块解耦优化建议\n\n")

	// 分析高耦合模块
	dependencyCount := make(map[string]int)
	dependentCount := make(map[string]int)

	for module, dep := range analyzer.Dependencies {
		dependencyCount[module] = len(dep.Dependencies)
		for _, dependency := range dep.Dependencies {
			dependentCount[dependency]++
		}
	}

	content.WriteString("### 1. 高依赖模块 (需要重构)\n")
	for module, count := range dependencyCount {
		if count > 5 {
			content.WriteString(fmt.Sprintf("- **%s**: %d个依赖\n", module, count))
		}
	}
	content.WriteString("\n")

	content.WriteString("### 2. 被高度依赖的模块 (需要稳定化)\n")
	for module, count := range dependentCount {
		if count > 3 {
			content.WriteString(fmt.Sprintf("- **%s**: 被%d个模块依赖\n", module, count))
		}
	}
	content.WriteString("\n")

	content.WriteString("### 3. 建议的解耦策略\n")
	content.WriteString("- **接口抽象**: 为被高度依赖的模块定义接口\n")
	content.WriteString("- **依赖注入**: 使用依赖注入减少直接依赖\n")
	content.WriteString("- **事件驱动**: 使用事件机制解耦模块间通信\n")
	content.WriteString("- **分层架构**: 建立清晰的分层结构\n\n")

	// 添加Mermaid图表
	content.WriteString("## 📊 依赖关系图表 (Mermaid)\n\n")
	content.WriteString("```mermaid\n")
	content.WriteString(analyzer.GenerateMermaidDiagram())
	content.WriteString("```\n")

	// 写入文件
	_, err = file.WriteString(content.String())
	if err != nil {
		fmt.Printf("写入文件失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 依赖关系分析完成！结果已保存到: %s\n", output)
}
