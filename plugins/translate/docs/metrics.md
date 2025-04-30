# 监控模块说明

## 监控系统设计

### 架构设计
```
metrics/
├── collectors/    # 指标收集器
├── exporters/     # 指标导出器
└── dashboards/    # Grafana 面板
```

### 核心指标
```go
type Metrics struct {
    requestsTotal      *prometheus.CounterVec
    requestDuration    *prometheus.HistogramVec
    cacheHits         prometheus.Counter
    cacheMisses       prometheus.Counter
    errorsTotal       *prometheus.CounterVec
    activeRequests    prometheus.Gauge
    bytesProcessed    prometheus.Counter
    concurrentWorkers prometheus.Gauge
}
```

## 指标详解

### 1. 请求统计
```go
// 请求计数器
requestsTotal: promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "translate_requests_total",
        Help: "翻译请求总数",
    },
    []string{"type", "source", "target"},
)

// 使用示例
m.requestsTotal.WithLabelValues("text", "en", "zh").Inc()
```

#### 监控维度
- 请求类型 (text/file/batch)
- 源语言
- 目标语言
- 时间分布

### 2. 性能指标
```go
// 请求耗时
requestDuration: promauto.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "translate_request_duration_seconds",
        Help:    "翻译请求耗时",
        Buckets: prometheus.DefBuckets,
    },
    []string{"type"},
)

// 使用示例
duration := time.Since(start).Seconds()
m.requestDuration.WithLabelValues("text").Observe(duration)
```

#### 监控维度
- 请求耗时分布
- 平均响应时间
- 95/99 百分位
- 最大响应时间

### 3. 缓存指标
```go
// 缓存命中
cacheHits: promauto.NewCounter(
    prometheus.CounterOpts{
        Name: "translate_cache_hits_total",
        Help: "缓存命中次数",
    },
)

// 缓存未命中
cacheMisses: promauto.NewCounter(
    prometheus.CounterOpts{
        Name: "translate_cache_misses_total",
        Help: "缓存未命中次数",
    },
)
```

#### 监控维度
- 命中率
- 未命中率
- 缓存效率
- 内存使用

### 4. 错误统计
```go
// 错误计数器
errorsTotal: promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "translate_errors_total",
        Help: "错误总数",
    },
    []string{"type"},
)

// 使用示例
m.errorsTotal.WithLabelValues("api_error").Inc()
```

#### 监控维度
- 错误类型
- 错误率
- 错误分布
- 重试次数

### 5. 资源使用
```go
// 活跃请求
activeRequests: promauto.NewGauge(
    prometheus.GaugeOpts{
        Name: "translate_active_requests",
        Help: "当前活跃请求数",
    },
)

// 工作协程数
concurrentWorkers: promauto.NewGauge(
    prometheus.GaugeOpts{
        Name: "translate_concurrent_workers",
        Help: "当前并发工作协程数",
    },
)
```

#### 监控维度
- CPU 使用率
- 内存使用
- 协程数量
- I/O 使用率

## Grafana 面板

### 1. 概览面板
```
# 请求 QPS
rate(translate_requests_total[5m])

# 错误率
rate(translate_errors_total[5m]) / 
rate(translate_requests_total[5m])

# 平均响应时间
rate(translate_request_duration_seconds_sum[5m]) /
rate(translate_request_duration_seconds_count[5m])
```

### 2. 性能面板
```
# 响应时间分布
histogram_quantile(0.95, 
  rate(translate_request_duration_seconds_bucket[5m]))

# 并发请求数
translate_active_requests

# 工作协程数
translate_concurrent_workers
```

### 3. 缓存面板
```
# 缓存命中率
rate(translate_cache_hits_total[5m]) /
(rate(translate_cache_hits_total[5m]) + 
 rate(translate_cache_misses_total[5m]))

# 缓存效率
increase(translate_cache_hits_total[1h])
```

## 告警规则

### 1. 高错误率
```yaml
alert: HighErrorRate
expr: |
  rate(translate_errors_total[5m]) /
  rate(translate_requests_total[5m]) > 0.1
for: 5m
labels:
  severity: warning
annotations:
  summary: 错误率过高
  description: "错误率超过 10%，当前值: {{ $value }}"
```

### 2. 响应时间过长
```yaml
alert: HighLatency
expr: |
  histogram_quantile(0.95,
    rate(translate_request_duration_seconds_bucket[5m])) > 1
for: 5m
labels:
  severity: warning
annotations:
  summary: 响应时间过长
  description: "95% 响应时间超过 1s，当前值: {{ $value }}s"
```

### 3. 资源使用过高
```yaml
alert: HighConcurrency
expr: translate_active_requests > 100
for: 5m
labels:
  severity: warning
annotations:
  summary: 并发请求过多
  description: "当前并发请求数: {{ $value }}"
```

## 监控最佳实践

### 1. 指标收集
- 使用标签区分不同维度
- 合理设置 bucket 范围
- 避免高基数标签
- 定期清理过期指标

### 2. 告警配置
- 设置合理的阈值
- 添加告警延迟
- 分级告警策略
- 配置通知渠道

### 3. 面板配置
- 合理布局面板
- 设置刷新间隔
- 添加图例说明
- 配置变量模板

### 4. 性能优化
- 减少指标基数
- 优化查询语句
- 设置采样率
- 配置保留策略

## 使用示例

### 1. 查看当前指标
```bash
# 获取所有指标
curl http://localhost:9090/metrics | grep translate_

# 查看特定指标
curl http://localhost:9090/metrics | grep translate_requests_total
```

### 2. 监控查询
```bash
# 最近 5 分钟的 QPS
rate(translate_requests_total[5m])

# 按语言分组的请求数
sum by (source_lang, target_lang) (translate_requests_total)

# 错误分布
topk(10, sum by (type) (translate_errors_total))
```

### 3. 性能分析
```bash
# 响应时间热力图
histogram_quantile(0.95, 
  sum(rate(translate_request_duration_seconds_bucket[5m])) by (le))

# 资源使用趋势
rate(translate_bytes_processed_total[1h])
```

## 故障排查

### 1. 高延迟问题
1. 检查响应时间分布
2. 分析并发请求数
3. 查看资源使用情况
4. 检查外部依赖

### 2. 错误率升高
1. 查看错误类型分布
2. 分析错误日志
3. 检查系统资源
4. 排查外部服务

### 3. 缓存效率低
1. 分析命中率趋势
2. 检查内存使用
3. 优化缓存策略
4. 调整缓存容量 