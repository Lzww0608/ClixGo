# 翻译插件 (Translation Plugin)

一个功能强大的文本翻译插件，支持多语言翻译、文件翻译、并发处理和性能监控。

## 功能特点

- 支持文本、文件和目录的翻译
- 智能缓存系统
- 并发处理大文件
- 完整的监控指标
- 自动重试机制
- 流式处理大文件
- 可配置的限流策略

## 安装

```bash
# 设置 API 密钥
export TRANSLATE_API_KEY="your-api-key"

# 创建配置目录
mkdir -p ~/.gocli

# 创建配置文件
cat > ~/.gocli/translate.yaml << EOF
api_key: ${TRANSLATE_API_KEY}
default_source: auto
default_target: zh
cache_enabled: true
cache_duration: 24h
max_concurrency: 5
timeout: 30s
chunk_size: 1000
rate_limit: 10.0
max_retries: 3
EOF
```

## 使用方法

### 基本翻译

```bash
# 翻译文本
gocli translate text "Hello, World" --source en --target zh

# 检测语言
gocli translate detect "Bonjour le monde"
```

### 文件翻译

```bash
# 翻译单个文件
gocli translate file document.txt --source en --target zh

# 翻译整个目录
gocli translate dir ./documents --source en --target zh
```

### 配置管理

```bash
# 查看当前配置
gocli translate config show

# 修改配置
gocli translate config set --key default_target --value ja
```

## 配置选项

| 选项 | 说明 | 默认值 |
|------|------|--------|
| api_key | API 密钥 | 环境变量 TRANSLATE_API_KEY |
| default_source | 默认源语言 | auto |
| default_target | 默认目标语言 | zh |
| cache_enabled | 启用缓存 | true |
| cache_duration | 缓存有效期 | 24h |
| max_concurrency | 最大并发数 | CPU核心数 |
| timeout | 请求超时时间 | 30s |
| chunk_size | 分块大小 | 1000 |
| rate_limit | API限流速率 | 10.0 |
| max_retries | 最大重试次数 | 3 |

## 监控指标

### Prometheus 指标

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| translate_requests_total | Counter | 翻译请求总数 |
| translate_request_duration_seconds | Histogram | 请求耗时分布 |
| translate_cache_hits_total | Counter | 缓存命中次数 |
| translate_cache_misses_total | Counter | 缓存未命中次数 |
| translate_errors_total | Counter | 错误总数 |
| translate_active_requests | Gauge | 当前活跃请求数 |
| translate_bytes_processed_total | Counter | 处理字节总数 |
| translate_concurrent_workers | Gauge | 当前工作协程数 |

### Grafana 面板

可以使用以下查询创建监控面板：

```
# 请求成功率
rate(translate_requests_total{status="success"}[5m]) / 
rate(translate_requests_total[5m])

# 平均响应时间
rate(translate_request_duration_seconds_sum[5m]) /
rate(translate_request_duration_seconds_count[5m])

# 缓存命中率
rate(translate_cache_hits_total[5m]) /
(rate(translate_cache_hits_total[5m]) + rate(translate_cache_misses_total[5m]))
```

## 开发指南

### 运行测试

```bash
# 运行单元测试
go test -v ./...

# 运行基准测试
go test -bench=. -benchmem

# 生成测试覆盖报告
go test -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### 性能优化

1. 使用内存缓存减少 API 调用
2. 实现流式处理避免内存溢出
3. 使用工作池控制并发
4. 实现限流避免 API 限制

### 错误处理

1. 使用自定义错误类型
2. 实现重试机制
3. 详细的错误日志
4. 友好的错误提示

## 常见问题

1. API 密钥未设置
   ```bash
   export TRANSLATE_API_KEY="your-api-key"
   ```

2. 配置文件不存在
   ```bash
   mkdir -p ~/.gocli
   touch ~/.gocli/translate.yaml
   ```

3. 请求超时
   ```bash
   # 增加超时时间
   gocli translate config set --key timeout --value 60s
   ```

4. 内存使用过高
   ```bash
   # 减小分块大小
   gocli translate config set --key chunk_size --value 500
   ```

## 贡献指南

1. Fork 项目
2. 创建特性分支
3. 提交变更
4. 推送到分支
5. 创建 Pull Request

## 许可证

MIT License 