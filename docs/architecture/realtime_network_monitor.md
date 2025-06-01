# ClixGo 实时网络资源监控

## 概述

ClixGo 实时网络资源监控是 Phase 2 开发计划的核心功能，提供了全面的网络性能监控和可视化界面。该功能具备超时机制和防死锁保护，确保系统稳定性。

## 主要特性

### 🚀 核心功能
- **实时网络接口监控**：带宽使用、错误统计、利用率分析
- **连接状态跟踪**：TCP/UDP连接统计、端口使用排行
- **延迟和丢包监控**：多目标ping测试、网络质量评估
- **系统资源监控**：文件描述符、内存使用、线程统计
- **智能告警系统**：可配置阈值、多级别告警

### 🛡️ 安全特性
- **超时机制**：防止网络操作死锁
- **非阻塞通道**：避免数据积压导致的阻塞
- **上下文取消**：优雅的资源清理
- **并发安全**：读写锁保护共享数据

### 🎨 用户界面
- **图形界面模式**：基于 tcell/tview 的实时仪表板
- **命令行模式**：适合脚本和自动化场景
- **交互式操作**：支持暂停/恢复、重启、告警确认

## 快速开始

### 安装

```bash
# 克隆项目
git clone https://github.com/Lzww0608/ClixGo.git
cd ClixGo

# 构建网络监控工具
go build -o netmonitor ./cmd/netmonitor
```

### 基本使用

#### 图形界面模式（默认）
```bash
# 启动图形界面监控
./netmonitor

# 自定义监控目标和间隔
./netmonitor -targets=8.8.8.8,1.1.1.1 -interval=1s
```

#### 命令行模式
```bash
# 启动命令行模式
./netmonitor -ui=false

# 监控特定接口
./netmonitor -ui=false -interfaces=eth0,wlan0
```

### 高级配置

```bash
# 完整配置示例
./netmonitor \
  -targets=8.8.8.8,1.1.1.1,google.com \
  -interfaces=eth0,wlan0 \
  -interval=2s \
  -timeout=5s \
  -history=200 \
  -alerts=true \
  -latency-threshold=50.0 \
  -packetloss-threshold=1.0 \
  -bandwidth-threshold=50.0 \
  -connection-threshold=500
```

## 配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-interval` | 2s | 监控更新间隔 |
| `-timeout` | 5s | 操作超时时间 |
| `-history` | 100 | 最大历史记录数 |
| `-alerts` | true | 启用告警 |
| `-targets` | 8.8.8.8,1.1.1.1 | 监控目标，逗号分隔 |
| `-interfaces` | "" | 监控接口，空表示所有 |
| `-ui` | true | 启用图形界面 |

### 告警阈值配置

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-latency-threshold` | 100.0 | 延迟告警阈值(毫秒) |
| `-packetloss-threshold` | 5.0 | 丢包率告警阈值(%) |
| `-bandwidth-threshold` | 100.0 | 带宽使用告警阈值(Mbps) |
| `-connection-threshold` | 1000 | 连接数告警阈值 |
| `-error-threshold` | 1.0 | 错误率告警阈值(%) |

## 图形界面操作

### 键盘快捷键
- `q` / `Q` / `Esc`：退出程序
- `r` / `R`：重启监控
- `p` / `P`：暂停/恢复监控
- `c` / `C`：清除告警
- `Tab`：切换焦点

### 界面布局
```
┌─────────────────────────────────────────────────────────────┐
│ 网络性能评分        │ 系统资源                              │
├─────────────────────────────────────────────────────────────┤
│ 网络接口统计                    │ 连接统计                  │
├─────────────────────────────────────────────────────────────┤
│ 目标延迟统计                    │ 实时告警                  │
├─────────────────────────────────────────────────────────────┤
│ 状态栏                                                      │
└─────────────────────────────────────────────────────────────┘
```

## API 使用

### 基本用法

```go
package main

import (
    "time"
    "github.com/Lzww0608/ClixGo/pkg/network"
)

func main() {
    // 创建监控配置
    config := network.RealtimeMonitorConfig{
        UpdateInterval:   2 * time.Second,
        Timeout:         5 * time.Second,
        MaxHistory:      100,
        EnableAlerts:    true,
        MonitoredTargets: []string{"8.8.8.8", "1.1.1.1"},
        AlertThresholds: network.AlertThresholds{
            LatencyMs:         100.0,
            PacketLossPercent: 5.0,
            BandwidthMbps:     100.0,
            ConnectionCount:   1000,
            ErrorRate:         1.0,
        },
    }
    
    // 创建监控器
    monitor := network.NewRealtimeNetworkMonitor(config)
    
    // 启动监控
    if err := monitor.Start(); err != nil {
        panic(err)
    }
    defer monitor.Stop()
    
    // 监听更新
    for {
        select {
        case snapshot := <-monitor.GetUpdateChannel():
            // 处理网络快照数据
            processSnapshot(snapshot)
        case err := <-monitor.GetErrorChannel():
            // 处理错误
            handleError(err)
        }
    }
}
```

### 图形界面集成

```go
// 创建UI
ui := network.NewRealtimeNetworkUI(monitor)

// 启动UI（阻塞）
if err := ui.Start(); err != nil {
    log.Fatal(err)
}
```

## 数据结构

### NetworkResourceSnapshot
```go
type NetworkResourceSnapshot struct {
    Timestamp        time.Time                    // 时间戳
    Interfaces       map[string]InterfaceStats    // 接口统计
    Connections      ConnectionSummary            // 连接汇总
    TargetLatencies  map[string]LatencyStats      // 延迟统计
    SystemResources  SystemNetworkResources       // 系统资源
    Alerts          []Alert                      // 告警列表
    PerformanceScore float64                      // 性能评分
}
```

### InterfaceStats
```go
type InterfaceStats struct {
    Name           string    // 接口名称
    IsUp           bool      // 是否启用
    Speed          uint64    // 接口速度(bps)
    MTU            int       // 最大传输单元
    BytesIn        uint64    // 接收字节数
    BytesOut       uint64    // 发送字节数
    PacketsIn      uint64    // 接收包数
    PacketsOut     uint64    // 发送包数
    ErrorsIn       uint64    // 接收错误数
    ErrorsOut      uint64    // 发送错误数
    BandwidthInMbps  float64 // 接收带宽(Mbps)
    BandwidthOutMbps float64 // 发送带宽(Mbps)
    Utilization    float64   // 使用率(%)
}
```

## 性能特性

### 超时机制
- **数据收集超时**：防止网络操作阻塞
- **Ping测试超时**：避免不可达目标影响整体性能
- **上下文取消**：支持优雅的中断和清理

### 防死锁保护
- **非阻塞通道发送**：通道满时丢弃旧数据而不阻塞
- **并发安全**：使用读写锁保护共享状态
- **资源清理**：确保协程和资源的正确释放

### 性能优化
- **并发数据收集**：多个数据源并行收集
- **历史记录限制**：自动清理过期数据
- **内存管理**：避免内存泄漏

## 测试

### 运行测试
```bash
# 运行所有实时监控测试
go test ./pkg/network -v -run TestRealtimeNetworkMonitor

# 运行特定测试
go test ./pkg/network -v -run TestRealtimeNetworkMonitor_TimeoutProtection

# 运行基准测试
go test ./pkg/network -bench=BenchmarkRealtimeNetworkMonitor
```

### 测试覆盖
- ✅ 基本功能测试
- ✅ 超时保护测试
- ✅ 通道非阻塞测试
- ✅ 告警阈值测试
- ✅ 上下文取消测试
- ✅ 统计信息测试
- ✅ 历史记录管理测试

## 故障排除

### 常见问题

#### 1. Ping测试失败
```
目标延迟: 8.8.8.8: 不可达, 延迟=0.00ms, 丢包=0.00%
```
**解决方案**：
- 检查网络连接
- 确认防火墙设置
- 尝试其他测试目标

#### 2. 权限不足
```
Error: 收集接口统计失败: permission denied
```
**解决方案**：
```bash
# 使用sudo运行
sudo ./netmonitor

# 或者设置capabilities
sudo setcap cap_net_raw+ep ./netmonitor
```

#### 3. 接口统计为空
**解决方案**：
- 确认运行在Linux系统上
- 检查/proc/net/dev文件权限
- 指定具体的网络接口

### 调试模式
```bash
# 启用详细日志
./netmonitor -ui=false -targets=localhost -interval=1s
```

## 贡献

欢迎提交Issue和Pull Request来改进实时网络监控功能。

### 开发环境
- Go 1.23+
- Linux系统（推荐）
- 网络访问权限

### 代码规范
- 遵循Go代码规范
- 添加适当的测试
- 包含超时机制
- 确保并发安全

## 许可证

本项目采用MIT许可证。详见LICENSE文件。 