/*
* @Author: Lzww0608
* @Date: 2025-6-9 15:42:15
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-09 15:42:15
* @Description: 服务接口
 */

package interfaces

import (
	"context"
	"io"
	"time"
)

// ================== 终端服务接口 ==================

// Session 会话接口
type Session interface {
	// ID 获取会话ID
	ID() string

	// Name 获取会话名称
	Name() string

	// SetName 设置会话名称
	SetName(name string) error

	// CreatedAt 获取创建时间
	CreatedAt() time.Time

	// IsActive 检查会话是否激活
	IsActive() bool

	// Close 关闭会话
	Close() error

	// Resize 调整会话大小
	Resize(width, height int) error

	// Write 向会话写入数据
	Write(data []byte) (int, error)

	// Read 从会话读取数据
	Read(data []byte) (int, error)
}

// Window 窗口接口
type Window interface {
	// ID 获取窗口ID
	ID() string

	// SessionID 获取所属会话ID
	SessionID() string

	// Name 获取窗口名称
	Name() string

	// SetName 设置窗口名称
	SetName(name string) error

	// IsActive 检查窗口是否激活
	IsActive() bool

	// Panes 获取所有面板
	Panes() []Pane

	// CreatePane 创建新面板
	CreatePane() (Pane, error)

	// ClosePane 关闭面板
	ClosePane(paneID string) error

	// SwitchPane 切换面板
	SwitchPane(paneID string) error
}

// Pane 面板接口
type Pane interface {
	// ID 获取面板ID
	ID() string

	// WindowID 获取所属窗口ID
	WindowID() string

	// Command 获取执行的命令
	Command() string

	// IsActive 检查面板是否激活
	IsActive() bool

	// ProcessID 获取进程ID
	ProcessID() int

	// WorkingDirectory 获取工作目录
	WorkingDirectory() string

	// SetWorkingDirectory 设置工作目录
	SetWorkingDirectory(dir string) error

	// Write 写入数据
	Write(data []byte) (int, error)

	// Read 读取数据
	Read(data []byte) (int, error)

	// Kill 终止进程
	Kill() error
}

// TerminalService 终端服务接口
type TerminalService interface {
	// CreateSession 创建新会话
	CreateSession(name string) (Session, error)

	// GetSession 获取会话
	GetSession(id string) (Session, error)

	// ListSessions 列出所有会话
	ListSessions() ([]Session, error)

	// CloseSession 关闭会话
	CloseSession(id string) error

	// CreateWindow 创建新窗口
	CreateWindow(sessionID, name string) (Window, error)

	// GetWindow 获取窗口
	GetWindow(sessionID, windowID string) (Window, error)

	// AttachToSession 连接到会话
	AttachToSession(sessionID string) error

	// DetachFromSession 断开会话连接
	DetachFromSession(sessionID string) error
}

// ================== 网络服务接口 ==================

// NetworkInfo 网络信息
type NetworkInfo struct {
	Interface  string   `json:"interface"`
	IP         string   `json:"ip"`
	Subnet     string   `json:"subnet"`
	Gateway    string   `json:"gateway"`
	DNSServers []string `json:"dns_servers"`
}

// BandwidthInfo 带宽信息
type BandwidthInfo struct {
	Interface   string    `json:"interface"`
	BytesSent   uint64    `json:"bytes_sent"`
	BytesRecv   uint64    `json:"bytes_recv"`
	PacketsSent uint64    `json:"packets_sent"`
	PacketsRecv uint64    `json:"packets_recv"`
	Timestamp   time.Time `json:"timestamp"`
}

// PingResult 网络延迟测试结果
type PingResult struct {
	Host    string        `json:"host"`
	RTT     time.Duration `json:"rtt"`
	Success bool          `json:"success"`
	Error   string        `json:"error,omitempty"`
}

// NetworkService 网络服务接口
type NetworkService interface {
	// Ping 网络延迟测试
	Ping(host string, timeout time.Duration) (PingResult, error)

	// CheckPort 端口连通性检查
	CheckPort(host string, port int) (bool, error)

	// GetNetworkInfo 获取网络信息
	GetNetworkInfo() ([]NetworkInfo, error)

	// MonitorBandwidth 监控带宽使用
	MonitorBandwidth(ctx context.Context, interval time.Duration) (<-chan BandwidthInfo, error)

	// TracePath 路径追踪
	TracePath(host string, maxHops int) ([]string, error)

	// LookupDNS DNS解析
	LookupDNS(domain string) ([]string, error)

	// CheckSSLCert SSL证书检查
	CheckSSLCert(host string, port int) (map[string]interface{}, error)
}

// ================== 性能服务接口 ==================

// SystemInfo 系统信息
type SystemInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	CPUCores     int    `json:"cpu_cores"`
	TotalMemory  uint64 `json:"total_memory"`
	Hostname     string `json:"hostname"`
	Uptime       uint64 `json:"uptime"`
}

// ResourceInfo 资源使用信息
type ResourceInfo struct {
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float64   `json:"memory_percent"`
	MemoryUsed    uint64    `json:"memory_used"`
	MemoryTotal   uint64    `json:"memory_total"`
	DiskPercent   float64   `json:"disk_percent"`
	DiskUsed      uint64    `json:"disk_used"`
	DiskTotal     uint64    `json:"disk_total"`
	LoadAverage   []float64 `json:"load_average"`
	Timestamp     time.Time `json:"timestamp"`
}

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID           int32   `json:"pid"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryRSS     uint64  `json:"memory_rss"`
	CreateTime    int64   `json:"create_time"`
	Status        string  `json:"status"`
}

// PerformanceService 性能监控服务接口
type PerformanceService interface {
	// GetSystemInfo 获取系统信息
	GetSystemInfo() (SystemInfo, error)

	// GetResourceInfo 获取资源使用信息
	GetResourceInfo() (ResourceInfo, error)

	// MonitorResources 监控资源使用
	MonitorResources(ctx context.Context, interval time.Duration) (<-chan ResourceInfo, error)

	// GetProcessList 获取进程列表
	GetProcessList() ([]ProcessInfo, error)

	// GetProcessInfo 获取指定进程信息
	GetProcessInfo(pid int32) (ProcessInfo, error)

	// OptimizeMemory 内存优化
	OptimizeMemory() error

	// SetCPULimit 设置CPU限制
	SetCPULimit(percent float64) error

	// GetPerformanceProfile 获取性能分析数据
	GetPerformanceProfile(duration time.Duration) (io.Reader, error)
}

// ================== 任务服务接口 ==================

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCanceled  TaskStatus = "canceled"
)

// Task 任务接口
type Task interface {
	// ID 获取任务ID
	ID() string

	// Name 获取任务名称
	Name() string

	// Description 获取任务描述
	Description() string

	// Status 获取任务状态
	Status() TaskStatus

	// CreatedAt 获取创建时间
	CreatedAt() time.Time

	// StartedAt 获取开始时间
	StartedAt() *time.Time

	// CompletedAt 获取完成时间
	CompletedAt() *time.Time

	// Progress 获取任务进度 (0-100)
	Progress() int

	// Result 获取任务结果
	Result() interface{}

	// Error 获取任务错误
	Error() error

	// Cancel 取消任务
	Cancel() error
}

// TaskFilter 任务过滤器
type TaskFilter struct {
	Status        *TaskStatus `json:"status,omitempty"`
	Name          *string     `json:"name,omitempty"`
	CreatedAfter  *time.Time  `json:"created_after,omitempty"`
	CreatedBefore *time.Time  `json:"created_before,omitempty"`
	Limit         *int        `json:"limit,omitempty"`
	Offset        *int        `json:"offset,omitempty"`
}

// TaskService 任务服务接口
type TaskService interface {
	// CreateTask 创建任务
	CreateTask(name, description string, fn func() (interface{}, error)) (Task, error)

	// GetTask 获取任务
	GetTask(id string) (Task, error)

	// ListTasks 列出任务
	ListTasks(filter TaskFilter) ([]Task, error)

	// CancelTask 取消任务
	CancelTask(id string) error

	// WaitForTask 等待任务完成
	WaitForTask(id string, timeout time.Duration) (Task, error)

	// SubscribeToTask 订阅任务状态变化
	SubscribeToTask(id string) (<-chan Task, error)

	// CleanupCompletedTasks 清理已完成的任务
	CleanupCompletedTasks(olderThan time.Duration) error
}
