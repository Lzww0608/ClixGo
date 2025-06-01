# 开发指南文档

本目录包含ClixGo项目的开发者参考指南、贡献指南和最佳实践文档。

## 📋 指南列表

### 🤝 [贡献指南](./CONTRIBUTING.md)
**描述**: 如何为ClixGo项目做贡献的详细指南  
**内容**:
- 项目贡献流程和规范
- 代码提交和PR要求
- 测试和文档标准
- 社区行为准则
- 问题报告和功能请求


### 📊 [技术债务处理指南](./TECH_DEBT_GUIDE.md)
**描述**: 技术债务识别、评估和处理的实施指南  
**内容**:
- 技术债务识别方法
- 优先级评估框架
- 具体处理步骤和工具
- 代码质量改进策略
- 持续改进流程

**适用人群**: 架构师、技术负责人、核心开发者  
**最后更新**: 2025-06-01

## 🎯 开发规范

### 代码规范
- **语言**: Go 1.21+
- **格式化**: 使用 `gofmt` 和 `goimports`
- **静态检查**: 通过 `go vet` 和 `golangci-lint`
- **测试覆盖**: 新代码要求 ≥ 80% 覆盖率
- **文档**: 公开API必须有完整注释

### 提交规范
```
<type>(<scope>): <subject>

<body>

<footer>
```

**类型 (type)**:
- `feat`: 新功能
- `fix`: Bug修复
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建/工具相关

### 分支策略
- `main`: 主分支，稳定版本
- `develop`: 开发分支，集成最新功能
- `feature/*`: 功能分支
- `hotfix/*`: 紧急修复分支
- `release/*`: 发布准备分支

## 🛠️ 开发工具

### 必需工具
- **Go**: 1.21+ 版本
- **Git**: 版本控制
- **Make**: 构建工具
- **Docker**: 容器化测试

### 推荐工具
- **VS Code**: 推荐IDE，配置文件已提供
- **GoLand**: JetBrains IDE
- **golangci-lint**: 代码检查
- **go-mod-outdated**: 依赖更新检查

### 开发环境设置
```bash
# 克隆项目
git clone https://github.com/Lzww0608/ClixGo.git
cd ClixGo

# 安装依赖
go mod download

# 运行测试
make test

# 构建项目
make build

# 运行linter
make lint
```

## 📚 学习资源

### 项目相关
- [项目README](../../README.md) - 项目概述
- [开发路线图](../../ROADMAP.md) - 发展规划
- [架构设计](../architecture/) - 系统架构
- [实现细节](../implementation/) - 技术实现

### Go语言
- [Go官方文档](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### 终端开发
- [tcell文档](https://github.com/gdamore/tcell)
- [tview文档](https://github.com/rivo/tview)
- [PTY编程指南](https://github.com/creack/pty)

## 🧪 测试指南

### 测试类型
1. **单元测试**: 测试单个函数/方法
2. **集成测试**: 测试模块间交互
3. **端到端测试**: 测试完整功能流程
4. **性能测试**: 基准测试和压力测试

### 测试规范
- 测试文件命名: `*_test.go`
- 测试函数命名: `TestXxx` 或 `BenchmarkXxx`
- 使用 `testify` 库进行断言
- 模拟外部依赖使用 `mock`
- 并发测试使用 `race` 检测

### 测试命令
```bash
# 运行所有测试
go test ./...

# 运行带覆盖率的测试
go test -cover ./...

# 运行基准测试
go test -bench=. ./...

# 运行竞态检测
go test -race ./...
```

## 📝 文档指南

### 文档类型
- **API文档**: 自动生成的接口文档
- **用户文档**: 使用说明和教程
- **开发文档**: 架构和实现细节
- **维护文档**: 运维和故障排除

### 文档规范
- 使用Markdown格式
- 包含清晰的目录结构
- 提供代码示例
- 保持链接有效性
- 定期更新内容

## 🚀 发布流程

### 版本规范
遵循 [语义化版本](https://semver.org/lang/zh-CN/) 规范:
- `MAJOR.MINOR.PATCH`
- 主版本号：不兼容的API修改
- 次版本号：向下兼容的功能性新增
- 修订号：向下兼容的问题修正

### 发布步骤
1. 创建release分支
2. 更新版本号和CHANGELOG
3. 运行完整测试套件
4. 创建PR到main分支
5. 合并后创建Git标签
6. 构建和发布二进制文件

## 🔗 相关资源

### 内部文档
- [架构设计](../architecture/) - 系统设计文档
- [实现细节](../implementation/) - 技术实现
- [项目报告](../reports/) - 开发记录
- [开发进度](../development/) - 进度跟踪

### 外部资源
- [Go语言规范](https://golang.org/ref/spec)
- [GitHub Flow](https://guides.github.com/introduction/flow/)
- [开源贡献指南](https://opensource.guide/zh-cn/)

---
