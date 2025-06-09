/*
* @Author: Lzww0608
* @Date: 2025-6-9 15:37:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-09 15:37:00
* @Description: 日志接口
 */

package interfaces

import "go.uber.org/zap"

// Logger 日志记录器接口
// 提供统一的日志记录API，支持结构化日志记录
type Logger interface {
	// Info 记录信息级别日志
	Info(msg string, fields ...zap.Field)

	// Error 记录错误级别日志
	Error(msg string, fields ...zap.Field)

	// Debug 记录调试级别日志
	Debug(msg string, fields ...zap.Field)

	// Warn 记录警告级别日志
	Warn(msg string, fields ...zap.Field)

	// Fatal 记录致命级别日志并退出程序
	Fatal(msg string, fields ...zap.Field)

	// With 创建带有预设字段的子logger
	With(fields ...zap.Field) Logger

	// WithOptions 创建带有配置选项的子logger
	WithOptions(opts ...zap.Option) Logger

	// Named 创建带有名称的子logger
	Named(name string) Logger

	// Sync 同步缓冲的日志条目
	Sync() error
}

// LoggerConfig 日志配置接口
type LoggerConfig interface {
	// GetLevel 获取日志级别
	GetLevel() string

	// GetOutputPath 获取输出路径
	GetOutputPath() []string

	// GetErrorOutputPath 获取错误输出路径
	GetErrorOutputPath() []string

	// IsDebugMode 是否为调试模式
	IsDebugMode() bool

	// GetEncoding 获取编码格式 (json/console)
	GetEncoding() string

	// GetMaxSize 获取日志文件最大大小 (MB)
	GetMaxSize() int

	// GetMaxBackups 获取保留的旧日志文件最大数量
	GetMaxBackups() int

	// GetMaxAge 获取旧日志文件保留天数
	GetMaxAge() int

	// IsCompress 是否压缩旧日志文件
	IsCompress() bool
}

// LoggerFactory 日志工厂接口
type LoggerFactory interface {
	// CreateLogger 创建新的日志器
	CreateLogger(config LoggerConfig) (Logger, error)

	// CreateNamedLogger 创建带名称的日志器
	CreateNamedLogger(name string, config LoggerConfig) (Logger, error)

	// CreateConsoleLogger 创建控制台日志器
	CreateConsoleLogger(level string) (Logger, error)

	// CreateFileLogger 创建文件日志器
	CreateFileLogger(filename string, level string) (Logger, error)
}
