# GoCLI 开发者指南

本文档旨在帮助开发者快速上手 GoCLI 项目的开发工作。

## 开发环境设置

### 1. 必要条件
- Go 1.18 或更高版本
- Git
- Make
- Docker (可选，用于容器化部署)
- VSCode 或其他 IDE

### 2. 开发工具
```bash
# 安装必要的 Go 工具
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/golang/mock/mockgen@latest
```

### 3. 项目设置
```bash
# 克隆项目
git clone https://github.com/yourusername/gocli.git
cd gocli

# 安装依赖
go mod download

# 编译项目
make build
```

## 代码规范

### 1. 代码风格
- 使用 `goimports` 格式化代码
- 遵循 [Effective Go](https://golang.org/doc/effective_go) 规范
- 使用 `golangci-lint` 进行代码检查
- 保持函数简短清晰

### 2. 命名规范
```go
// 包名使用小写
package translate

// 接口名使用 er 结尾
type Translator interface {
    Translate(text string) string
}

// 结构体使用驼峰命名
type TranslationService struct {
    config *Config
    cache  *Cache
}

// 公开方法使用驼峰命名
func (s *TranslationService) TranslateText(text string) string {
    // ...
}

// 私有方法使用小驼峰
func (s *TranslationService) validateInput(text string) error {
    // ...
}
```

### 3. 注释规范
```go
// Package translate 提供文本翻译功能
package translate

// TranslationService 实现了翻译服务的核心功能
type TranslationService struct {
    // ...
}

// TranslateText 将文本从源语言翻译为目标语言
// 参数:
//   - text: 要翻译的文本
//   - sourceLang: 源语言代码
//   - targetLang: 目标语言代码
// 返回:
//   - 翻译后的文本
//   - 错误信息
func (s *TranslationService) TranslateText(text, sourceLang, targetLang string) (string, error) {
    // ...
}
```

## 测试规范

### 1. 单元测试
```go
func TestTranslateText(t *testing.T) {
    tests := []struct {
        name       string
        input      string
        want       string
        wantErr    bool
    }{
        {
            name:    "正常翻译",
            input:   "Hello",
            want:    "你好",
            wantErr: false,
        },
        // 更多测试用例...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试代码...
        })
    }
}
```

### 2. 基准测试
```go
func BenchmarkTranslateText(b *testing.B) {
    service := NewTranslationService()
    text := "Hello, World"
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        service.TranslateText(text, "en", "zh")
    }
}
```

### 3. 集成测试
```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("跳过集成测试")
    }
    // 集成测试代码...
}
```

## 错误处理

### 1. 错误定义
```go
var (
    ErrInvalidInput = errors.New("无效的输入")
    ErrAPILimit     = errors.New("API 调用限制")
)

// 自定义错误类型
type TranslationError struct {
    Code    int
    Message string
    Err     error
}

func (e *TranslationError) Error() string {
    return fmt.Sprintf("翻译错误 [%d]: %s: %v", e.Code, e.Message, e.Err)
}
```

### 2. 错误处理
```go
func (s *TranslationService) TranslateText(text string) (string, error) {
    if text == "" {
        return "", ErrInvalidInput
    }

    result, err := s.translate(text)
    if err != nil {
        return "", &TranslationError{
            Code:    500,
            Message: "翻译服务错误",
            Err:     err,
        }
    }

    return result, nil
}
```

## 性能优化

### 1. 并发处理
```go
func (s *TranslationService) TranslateBatch(texts []string) []string {
    results := make([]string, len(texts))
    var wg sync.WaitGroup
    
    // 使用工作池控制并发
    semaphore := make(chan struct{}, s.config.MaxConcurrency)
    
    for i, text := range texts {
        wg.Add(1)
        go func(i int, text string) {
            defer wg.Done()
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            
            result, _ := s.TranslateText(text)
            results[i] = result
        }(i, text)
    }
    
    wg.Wait()
    return results
}
```

### 2. 内存优化
```go
// 使用对象池
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func (s *TranslationService) processText(text string) string {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()
    
    // 使用 buffer 处理文本...
    return buf.String()
}
```

## 日志规范

### 1. 日志级别
```go
// 初始化日志
logger := zap.NewProduction()
defer logger.Sync()

// 使用不同级别的日志
logger.Info("正常信息",
    zap.String("module", "translate"),
    zap.String("action", "translate_text"),
)

logger.Error("错误信息",
    zap.Error(err),
    zap.String("text", text),
)
```

### 2. 结构化日志
```go
// 定义日志字段
type LogFields struct {
    Module     string    `json:"module"`
    Action     string    `json:"action"`
    Duration   float64   `json:"duration"`
    StatusCode int       `json:"status_code"`
    Error      string    `json:"error,omitempty"`
}

// 记录日志
func logRequest(logger *zap.Logger, fields LogFields) {
    logger.Info("请求完成",
        zap.Object("fields", fields),
    )
}
```

## 文档规范

### 1. 代码文档
```go
// TranslationService 提供文本翻译服务
//
// Example:
//
//     service := NewTranslationService()
//     result, err := service.TranslateText("Hello", "en", "zh")
//     if err != nil {
//         log.Fatal(err)
//     }
//     fmt.Println(result)
type TranslationService struct {
    // ...
}
```

### 2. API 文档
```go
// @Summary 翻译文本
// @Description 将文本从源语言翻译为目标语言
// @Tags translate
// @Accept json
// @Produce json
// @Param text body string true "要翻译的文本"
// @Success 200 {object} TranslationResponse
// @Failure 400 {object} ErrorResponse
// @Router /translate [post]
func (s *Server) handleTranslate(w http.ResponseWriter, r *http.Request) {
    // ...
}
```

## 版本控制

### 1. 分支管理
```bash
# 创建特性分支
git checkout -b feature/new-translation-api

# 创建修复分支
git checkout -b fix/memory-leak
```

### 2. 提交规范
```bash
# 特性提交
git commit -m "feat: 添加批量翻译功能

- 支持多文件并发翻译
- 添加进度显示
- 优化内存使用"

# 修复提交
git commit -m "fix: 修复内存泄漏问题

- 修复 TranslationService 中的资源释放问题
- 添加内存使用监控
- 更新相关测试用例"
```

## CI/CD 流程

### 1. 持续集成
```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    - name: Set up Go
      uses: actions/setup-go@v2
      with:
        go-version: 1.18
    - name: Run tests
      run: make test
    - name: Run linter
      run: make lint
```

### 2. 持续部署
```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    - name: Build
      run: make build
    - name: Create Release
      uses: actions/create-release@v1
      with:
        tag_name: ${{ github.ref }}
        release_name: Release ${{ github.ref }}
```

## 问题反馈

1. 使用 GitHub Issues 报告问题
2. 提供详细的复现步骤
3. 附上相关的日志和错误信息
4. 标注影响范围和严重程度

## 代码审查

### 1. 审查清单
- 代码风格是否符合规范
- 是否包含充分的测试
- 是否有性能问题
- 是否有安全隐患
- 文档是否完整

### 2. 审查流程
1. 创建 Pull Request
2. 等待 CI 检查通过
3. 请求代码审查
4. 根据反馈修改
5. 合并到主分支

## 发布流程

### 1. 版本管理
```bash
# 创建新版本
make version VERSION=1.2.0

# 生成更新日志
make changelog

# 创建发布标签
git tag -a v1.2.0 -m "Release version 1.2.0"
```

### 2. 发布检查
- 更新版本号
- 更新文档
- 运行完整测试
- 检查依赖更新
- 生成更新日志 