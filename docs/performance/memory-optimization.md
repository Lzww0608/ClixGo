# ClixGo 内存使用优化功能

## 📋 概述

ClixGo 内存使用优化功能是一个完整的内存监控、分析和优化解决方案，提供实时内存监控、pprof集成、内存泄漏检测和自动优化建议。

## 🎯 功能特性

### 1. 实时内存监控
- **内存基线建立**: 自动建立应用启动时的内存基线
- **持续监控**: 定期收集内存快照和运行时指标
- **多维度指标**: 堆内存、栈内存、GC指标、goroutine数量等
- **内存热点识别**: 自动识别内存使用异常的代码位置

### 2. pprof 集成
- **HTTP服务器**: 内置pprof HTTP服务器
- **多种Profile**: 支持heap、goroutine、allocs等profile类型
- **自动收集**: 定期收集性能剖析数据
- **数据管理**: 智能管理profile数据，避免内存泄漏

### 3. 智能告警系统
- **多级告警**: 支持low、medium、high、critical四个级别
- **告警类型**: 内存增长、堆内存增长、GC压力、内存碎片化
- **证据收集**: 提供详细的告警证据和指标数据
- **建议生成**: 自动生成针对性的优化建议

### 4. 自动优化功能
- **优化建议**: 基于监控数据生成智能优化建议
- **自动执行**: 支持安全的自动优化操作
- **影响评估**: 预估优化效果和风险级别
- **统计跟踪**: 记录优化操作的执行情况和效果

## 🚀 快速开始

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

    // 配置内存监控器
    config := performance.MemoryMonitorConfig{
        MonitorInterval:         10 * time.Second,
        BaselineWarmupTime:      30 * time.Second,
        MaxSnapshots:            100,
        EnablePprof:             true,
        PprofAddress:            ":6060",
        EnableAutoOptimization:  true,
        OptimizationInterval:    2 * time.Minute,
        
        // 告警阈值
        MemoryGrowthThresholdMB: 100.0,
        HeapGrowthThresholdMB:   50.0,
        GCPressureThreshold:     0.1,
        FragmentationThreshold:  0.3,
        
        // 优化参数
        AutoGCTriggerThresholdMB: 200.0,
        MemoryReleaseThresholdMB: 150.0,
    }

    // 创建并启动监控器
    monitor := performance.NewMemoryMonitor(config, logger)
    
    err := monitor.Start()
    if err != nil {
        logger.Fatal("启动内存监控器失败", zap.Error(err))
    }
    defer monitor.Stop()

    // 监听告警
    go func() {
        for alert := range monitor.GetAlertChannel() {
            logger.Warn("内存告警", 
                zap.String("type", alert.Type),
                zap.String("severity", alert.Severity),
                zap.String("description", alert.Description))
        }
    }()

    // 监听优化建议
    go func() {
        for suggestion := range monitor.GetOptimizationChannel() {
            logger.Info("优化建议",
                zap.String("type", suggestion.Type),
                zap.String("priority", suggestion.Priority),
                zap.String("title", suggestion.Title))
        }
    }()

    // 你的应用逻辑
    // ...
}
```

### 配置参数说明

#### 基本配置
- `MonitorInterval`: 监控间隔，建议5-30秒
- `BaselineWarmupTime`: 基线预热时间，建议30-60秒
- `MaxSnapshots`: 最大快照数量，建议50-200
- `EnablePprof`: 是否启用pprof服务器
- `PprofAddress`: pprof服务器地址，默认":6060"

#### 告警阈值
- `MemoryGrowthThresholdMB`: 内存增长告警阈值(MB)
- `HeapGrowthThresholdMB`: 堆内存增长告警阈值(MB)
- `GCPressureThreshold`: GC压力告警阈值(0-1)
- `FragmentationThreshold`: 内存碎片化告警阈值(0-1)

#### 优化参数
- `EnableAutoOptimization`: 是否启用自动优化
- `OptimizationInterval`: 优化检查间隔
- `AutoGCTriggerThresholdMB`: 自动GC触发阈值(MB)
- `MemoryReleaseThresholdMB`: 内存释放阈值(MB)

## 📊 pprof 使用

### 启动pprof服务器

```go
config := performance.MemoryMonitorConfig{
    EnablePprof:  true,
    PprofAddress: ":6060",
}
```

### 访问pprof端点

```bash
# 查看所有可用的profile
curl http://localhost:6060/debug/pprof/

# 分析堆内存使用
go tool pprof http://localhost:6060/debug/pprof/heap

# 分析内存分配
go tool pprof http://localhost:6060/debug/pprof/allocs

# 分析goroutine
go tool pprof http://localhost:6060/debug/pprof/goroutine

# 生成火焰图
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/heap
```

### pprof 分析命令

在pprof交互模式下：

```bash
# 查看top消耗
(pprof) top

# 查看调用图
(pprof) web

# 查看特定函数
(pprof) list functionName

# 查看源码
(pprof) peek functionName
```

## 🔧 API 参考

### MemoryMonitor 主要方法

```go
// 启动监控器
func (mm *MemoryMonitor) Start() error

// 停止监控器
func (mm *MemoryMonitor) Stop() error

// 获取当前快照
func (mm *MemoryMonitor) GetCurrentSnapshot() (*MemorySnapshot, error)

// 获取基线信息
func (mm *MemoryMonitor) GetBaseline() *MemoryBaseline

// 获取历史快照
func (mm *MemoryMonitor) GetSnapshots() []MemorySnapshot

// 强制执行优化
func (mm *MemoryMonitor) ForceOptimization() error

// 获取告警通道
func (mm *MemoryMonitor) GetAlertChannel() <-chan MemoryAlert

// 获取优化建议通道
func (mm *MemoryMonitor) GetOptimizationChannel() <-chan OptimizationSuggestion

// 获取错误通道
func (mm *MemoryMonitor) GetErrorChannel() <-chan error
```

### 数据结构

#### MemorySnapshot
```go
type MemorySnapshot struct {
    Timestamp           time.Time
    HeapAllocMB         float64  // 堆内存分配(MB)
    HeapSysMB           float64  // 堆系统内存(MB)
    HeapIdleMB          float64  // 堆空闲内存(MB)
    HeapInuseMB         float64  // 堆使用内存(MB)
    StackInuseMB        float64  // 栈使用内存(MB)
    GoroutineCount      int      // goroutine数量
    GCCount             uint32   // GC次数
    GCPauseTotalMs      float64  // GC总暂停时间(ms)
    FragmentationRatio  float64  // 内存碎片化比率
    TopAllocators       []MemoryAllocatorInfo
    MemoryHotspots      []MemoryHotspot
    GCPressureIndicator float64  // GC压力指示器
}
```

#### MemoryAlert
```go
type MemoryAlert struct {
    ID           string
    Type         string        // 告警类型
    Severity     string        // 严重程度
    Title        string        // 告警标题
    Description  string        // 详细描述
    Evidence     []string      // 证据列表
    Suggestions  []string      // 建议列表
    Timestamp    time.Time
    MetricValues map[string]float64 // 关键指标
}
```

## 📈 监控指标

### 内存指标
- **HeapAlloc**: 当前堆内存分配量
- **HeapSys**: 堆系统内存总量
- **HeapIdle**: 堆空闲内存量
- **HeapInuse**: 堆使用内存量
- **StackInuse**: 栈使用内存量
- **FragmentationRatio**: 内存碎片化比率

### GC指标
- **GCCount**: GC执行次数
- **GCPauseTotal**: GC总暂停时间
- **GCCPUFraction**: GC CPU占用比例
- **GCPressureIndicator**: GC压力指示器

### 运行时指标
- **GoroutineCount**: goroutine数量
- **TimerCount**: 定时器数量
- **ActiveCGoCalls**: 活跃CGO调用数

## 🎯 最佳实践

### 1. 配置建议

#### 生产环境
```go
config := performance.MemoryMonitorConfig{
    MonitorInterval:         30 * time.Second,  // 较长间隔减少开销
    BaselineWarmupTime:      60 * time.Second,  // 充分预热
    MaxSnapshots:            50,                // 适中的历史记录
    EnablePprof:             false,             // 生产环境关闭pprof
    EnableAutoOptimization:  true,              // 启用自动优化
    MemoryGrowthThresholdMB: 200.0,            // 较高阈值避免误报
    HeapGrowthThresholdMB:   100.0,
    GCPressureThreshold:     0.1,
    FragmentationThreshold:  0.4,
}
```

#### 开发环境
```go
config := performance.MemoryMonitorConfig{
    MonitorInterval:         5 * time.Second,   // 短间隔快速反馈
    BaselineWarmupTime:      10 * time.Second,  // 快速启动
    MaxSnapshots:            200,               // 更多历史记录
    EnablePprof:             true,              // 启用pprof分析
    EnableAutoOptimization:  false,             // 手动控制优化
    MemoryGrowthThresholdMB: 50.0,             // 较低阈值敏感检测
    HeapGrowthThresholdMB:   25.0,
    GCPressureThreshold:     0.05,
    FragmentationThreshold:  0.2,
}
```

### 2. 告警处理

```go
// 告警处理示例
go func() {
    for alert := range monitor.GetAlertChannel() {
        switch alert.Severity {
        case "critical":
            // 立即处理关键告警
            handleCriticalAlert(alert)
        case "high":
            // 高优先级告警
            handleHighAlert(alert)
        case "medium":
            // 记录中等告警
            logMediumAlert(alert)
        case "low":
            // 低优先级告警
            logLowAlert(alert)
        }
    }
}()

func handleCriticalAlert(alert performance.MemoryAlert) {
    // 发送紧急通知
    sendUrgentNotification(alert)
    
    // 执行紧急措施
    if alert.Type == "memory_growth" {
        runtime.GC() // 强制GC
    }
    
    // 记录详细日志
    logger.Error("关键内存告警", 
        zap.String("id", alert.ID),
        zap.String("description", alert.Description),
        zap.Any("metrics", alert.MetricValues))
}
```

### 3. 优化建议处理

```go
// 优化建议处理示例
go func() {
    for suggestion := range monitor.GetOptimizationChannel() {
        switch suggestion.Priority {
        case "high":
            // 高优先级建议立即执行
            executeOptimization(suggestion)
        case "medium":
            // 中等优先级建议评估后执行
            evaluateAndExecute(suggestion)
        case "low":
            // 低优先级建议记录备用
            logOptimizationSuggestion(suggestion)
        }
    }
}()

func executeOptimization(suggestion performance.OptimizationSuggestion) {
    logger.Info("执行优化建议",
        zap.String("id", suggestion.ID),
        zap.String("type", suggestion.Type),
        zap.Float64("estimated_saving", suggestion.Impact.EstimatedMemorySavingMB))
    
    for _, action := range suggestion.Actions {
        if action.AutoExecute {
            // 执行自动优化操作
            switch action.Action {
            case "trigger_gc":
                runtime.GC()
            case "memory_defrag":
                runtime.GC()
                runtime.GC() // 连续GC有助于内存整理
            }
        }
    }
}
```

### 4. 性能考虑

#### 监控开销最小化
- 合理设置监控间隔，避免过于频繁
- 限制快照历史数量，防止内存泄漏
- 生产环境关闭详细的profile收集

#### 内存使用优化
- 及时响应告警，避免内存问题恶化
- 定期分析pprof数据，识别内存热点
- 合理配置GC参数，平衡性能和内存使用

## 🔍 故障排除

### 常见问题

#### 1. 监控器启动失败
```bash
错误: 启动内存监控器失败: pprof服务器启动失败: listen tcp :6060: bind: address already in use
解决: 更改pprof端口或停止占用端口的进程
```

#### 2. 告警过于频繁
```bash
问题: 收到大量低级别告警
解决: 调整告警阈值，提高阈值或增加检查间隔
```

#### 3. pprof无法访问
```bash
问题: 无法访问pprof端点
解决: 检查防火墙设置，确保端口开放
```

### 调试技巧

#### 1. 启用详细日志
```go
logger, _ := zap.NewDevelopment()
// 或者
logger := zap.NewExample()
```

#### 2. 检查监控状态
```go
if monitor.IsRunning() {
    logger.Info("监控器正在运行")
    baseline := monitor.GetBaseline()
    snapshots := monitor.GetSnapshots()
    logger.Info("监控状态", 
        zap.Int("snapshots_count", len(snapshots)),
        zap.Any("baseline", baseline))
}
```

#### 3. 手动触发检查
```go
// 强制执行优化检查
err := monitor.ForceOptimization()
if err != nil {
    logger.Error("强制优化失败", zap.Error(err))
}

// 获取当前快照
snapshot, err := monitor.GetCurrentSnapshot()
if err != nil {
    logger.Error("获取快照失败", zap.Error(err))
} else {
    logger.Info("当前内存状态",
        zap.Float64("heap_alloc_mb", snapshot.HeapAllocMB),
        zap.Float64("heap_sys_mb", snapshot.HeapSysMB),
        zap.Int("goroutine_count", snapshot.GoroutineCount))
}
```

## 📚 相关文档

- [Go pprof 官方文档](https://golang.org/pkg/net/http/pprof/)
- [Go 内存管理](https://golang.org/doc/gc-guide)
- [性能分析最佳实践](https://golang.org/doc/diagnostics.html)

## 🤝 贡献

欢迎提交Issue和Pull Request来改进内存优化功能。

## 📄 许可证

本项目采用MIT许可证。 