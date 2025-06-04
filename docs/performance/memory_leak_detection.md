# 内存泄漏检测器

## 概述

内存泄漏检测器是ClixGo性能优化专项的核心组件，提供runtime级别的内存泄漏监控和检测功能。它能够实时监控goroutine、内存分配、定时器等资源的使用情况，并通过智能算法检测潜在的内存泄漏问题。

## 功能特性

### 🔍 多维度监控
- **Goroutine监控**: 追踪goroutine的创建、销毁和状态变化
- **内存监控**: 监控堆内存、栈内存的分配和使用情况
- **定时器监控**: 追踪Timer和Ticker的创建和销毁
- **GC监控**: 监控垃圾回收的频率和性能

### 🧠 智能检测算法
- **趋势分析**: 基于历史数据分析资源使用趋势
- **可疑模式识别**: 自动识别异常的资源增长模式
- **置信度评估**: 为每个检测结果提供置信度评分
- **多重验证**: 结合多个指标进行综合判断

### 🚨 实时告警系统
- **分级告警**: 支持critical、high、medium、low四个级别
- **详细证据**: 提供具体的证据和建议
- **通道通信**: 通过Go channel提供非阻塞的告警通知

### 📊 性能分析
- **资源基线**: 建立系统资源使用基线
- **历史快照**: 保存历史资源使用快照
- **Profile集成**: 集成runtime/pprof进行深度分析

## 快速开始

### 基本使用

```go
package main

import (
    "time"
    "github.com/Lzww0608/ClixGo/pkg/performance"
    "go.uber.org/zap"
)

func main() {
    // 创建日志记录器
    logger, _ := zap.NewDevelopment()
    defer logger.Sync()

    // 配置检测器
    config := performance.MemoryLeakDetectorConfig{
        CheckInterval:            30 * time.Second,
        BaselineWarmupTime:       60 * time.Second,
        MaxSnapshots:             100,
        GoroutineGrowthThreshold: 50,
        MemoryGrowthThresholdMB:  100.0,
        HeapGrowthThresholdMB:    50.0,
    }

    // 创建并启动检测器
    detector := performance.NewMemoryLeakDetector(config, logger)
    err := detector.Start()
    if err != nil {
        logger.Fatal("启动检测器失败", zap.Error(err))
    }
    defer detector.Stop()

    // 监控告警
    go func() {
        for alert := range detector.GetAlertChannel() {
            logger.Warn("内存泄漏告警",
                zap.String("type", alert.Type),
                zap.String("severity", alert.Severity),
                zap.String("description", alert.Description))
        }
    }()

    // 你的应用程序逻辑...
}
```

### 配置参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `CheckInterval` | `time.Duration` | 30s | 检查间隔 |
| `BaselineWarmupTime` | `time.Duration` | 60s | 基线预热时间 |
| `MaxSnapshots` | `int` | 100 | 最大快照数量 |
| `GoroutineGrowthThreshold` | `int` | 50 | Goroutine增长阈值 |
| `MemoryGrowthThresholdMB` | `float64` | 100.0 | 内存增长阈值(MB) |
| `HeapGrowthThresholdMB` | `float64` | 50.0 | 堆内存增长阈值(MB) |
| `GCFrequencyThreshold` | `int` | 0 | GC频率阈值 |
| `TimerGrowthThreshold` | `int` | 0 | 定时器增长阈值 |
| `EnablePprof` | `bool` | false | 启用pprof |
| `PprofPort` | `int` | 0 | pprof端口 |

## 检测类型

### 1. Goroutine泄漏检测

检测器会监控goroutine数量的变化，当发现以下情况时会触发告警：
- Goroutine数量持续增长
- 增长速度超过阈值
- 存在长时间运行的goroutine

**常见原因**:
- 未正确关闭的goroutine
- 缺少退出机制的无限循环
- 阻塞的channel操作

**建议修复**:
```go
// 错误示例
go func() {
    for {
        // 无退出条件的循环
        doSomething()
    }
}()

// 正确示例
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            doSomething()
        }
    }
}()
```

### 2. 内存泄漏检测

监控堆内存的使用情况，检测以下模式：
- 内存使用量持续增长
- GC后内存未释放
- 大对象长时间占用

**常见原因**:
- 循环引用
- 全局变量持有大对象
- 缓存未设置过期时间

**建议修复**:
```go
// 使用sync.Pool复用对象
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 1024)
    },
}

func processData() {
    buffer := bufferPool.Get().([]byte)
    defer bufferPool.Put(buffer)
    
    // 使用buffer处理数据
}
```

### 3. 定时器泄漏检测

监控Timer和Ticker的创建和销毁：
- 未调用Stop()的定时器
- 定时器数量异常增长

**建议修复**:
```go
// 确保定时器正确停止
timer := time.NewTimer(5 * time.Second)
defer timer.Stop()

ticker := time.NewTicker(1 * time.Second)
defer ticker.Stop()
```

## API参考

### 核心方法

#### `NewMemoryLeakDetector(config, logger) *MemoryLeakDetector`
创建新的内存泄漏检测器实例。

#### `Start() error`
启动检测器，开始监控资源使用情况。

#### `Stop() error`
停止检测器，清理资源。

#### `ForceCheck() (*LeakDetectionResult, error)`
强制执行一次检测检查。

#### `GetBaseline() *ResourceBaseline`
获取资源使用基线。

#### `GetSnapshots() []ResourceSnapshot`
获取历史资源快照。

#### `IsRunning() bool`
检查检测器是否正在运行。

### 通道接口

#### `GetAlertChannel() <-chan MemoryLeakAlert`
获取告警通道，用于接收内存泄漏告警。

#### `GetLeakDetectedChannel() <-chan LeakDetectionResult`
获取泄漏检测结果通道。

#### `GetErrorChannel() <-chan error`
获取错误通道。

## 数据结构

### ResourceBaseline
```go
type ResourceBaseline struct {
    Timestamp      time.Time
    GoroutineCount int
    HeapAllocMB    float64
    HeapSysMB      float64
    StackSysMB     float64
    GCCount        uint32
    TimerCount     int
    ActiveCGoCalls int64
}
```

### ResourceSnapshot
```go
type ResourceSnapshot struct {
    Timestamp           time.Time
    GoroutineCount      int
    GoroutineProfiles   []GoroutineProfile
    HeapAllocMB         float64
    HeapSysMB           float64
    StackSysMB          float64
    GCCount             uint32
    GCPauseTotalMs      float64
    TimerCount          int
    ActiveCGoCalls      int64
    TopMemoryAllocators []MemoryAllocatorProfile
    SuspiciousPatterns  []SuspiciousPattern
}
```

### MemoryLeakAlert
```go
type MemoryLeakAlert struct {
    ID          string
    Type        string
    Severity    string
    Title       string
    Description string
    Evidence    []string
    Snapshot    ResourceSnapshot
    Baseline    ResourceBaseline
    Timestamp   time.Time
    Suggestions []string
}
```

## 最佳实践

### 1. 合理设置阈值
根据应用的特点调整检测阈值：
- 对于高并发应用，适当提高goroutine阈值
- 对于内存敏感应用，降低内存增长阈值
- 根据业务周期调整检查间隔

### 2. 监控告警处理
```go
go func() {
    for alert := range detector.GetAlertChannel() {
        switch alert.Severity {
        case "critical":
            // 立即处理
            handleCriticalAlert(alert)
        case "high":
            // 高优先级处理
            handleHighAlert(alert)
        default:
            // 记录日志
            logger.Info("内存泄漏告警", zap.Any("alert", alert))
        }
    }
}()
```

### 3. 集成到CI/CD
在测试环境中集成内存泄漏检测：
```go
func TestMemoryLeak(t *testing.T) {
    detector := performance.NewMemoryLeakDetector(config, logger)
    detector.Start()
    defer detector.Stop()
    
    // 运行测试
    runTests()
    
    // 检查是否有泄漏
    result, _ := detector.ForceCheck()
    if result.HasLeak {
        t.Errorf("检测到内存泄漏: %s", result.Description)
    }
}
```

### 4. 性能考虑
- 检测器本身的性能开销很小（< 1% CPU）
- 可以通过调整检查间隔来平衡检测精度和性能
- 在生产环境中建议禁用详细的profile收集

## 故障排除

### 常见问题

**Q: 检测器报告误报怎么办？**
A: 调整相应的阈值参数，或者检查应用是否确实存在资源使用异常。

**Q: 检测器本身会造成内存泄漏吗？**
A: 不会。检测器使用了严格的资源管理和并发安全设计，包括：
- 限制快照数量
- 安全的通道操作
- 优雅的关闭机制

**Q: 如何在生产环境中使用？**
A: 建议：
- 设置合理的阈值
- 监控告警通道
- 定期检查检测结果
- 集成到监控系统

## 示例程序

完整的示例程序请参考：`examples/memory_leak_detection/main.go`

该示例演示了：
- 基本配置和启动
- 各种泄漏场景的模拟
- 告警监控和处理
- 检测结果分析
