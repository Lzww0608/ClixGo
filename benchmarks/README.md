# ClixGo 性能基准测试框架

## 🎯 基准测试目标

根据ROADMAP中的性能指标，建立以下性能基线：

### 性能目标
- **启动时间**: < 50ms (相比 tmux 150ms)
- **内存占用**: < 8MB (相比 tmux 25MB)  
- **CPU使用率**: < 1% (空闲状态)
- **终端创建**: < 10ms
- **会话切换**: < 5ms
- **UI渲染**: > 60 FPS

## 📊 基准测试分类

### 1. 核心功能性能测试
- 终端创建和销毁性能
- 会话管理性能
- UI渲染性能
- PTY操作性能

### 2. 数据处理性能测试  
- 大量文本处理
- 网络数据处理
- 配置文件处理
- 任务队列处理

### 3. 并发性能测试
- 多会话并发管理
- 并发网络监控
- 并发任务执行
- 并发UI更新

### 4. 内存性能测试
- 内存分配模式
- 垃圾回收影响
- 内存泄漏检测
- 缓存效率

## 🚀 使用方法

### 运行所有基准测试
```bash
./scripts/benchmark.sh
```

### 运行特定模块基准测试
```bash
go test -bench=BenchmarkTerminal ./pkg/terminal/...
go test -bench=BenchmarkUI ./pkg/ui/...
go test -bench=BenchmarkNetwork ./pkg/network/...
```

### 生成性能报告
```bash
./scripts/benchmark-report.sh
```

### 性能回归检测
```bash
./scripts/benchmark-compare.sh baseline.txt current.txt
```

## 📈 基准测试结果

基准测试结果保存在 `benchmarks/results/` 目录下，包含：
- 时间序列性能数据
- 内存分配分析
- CPU使用率分析
- 性能回归报告

## 🔧 基准测试配置

基准测试配置位于 `benchmarks/config.yaml`，可以调整：
- 测试运行时间
- 测试重复次数
- 性能阈值
- 报告格式 