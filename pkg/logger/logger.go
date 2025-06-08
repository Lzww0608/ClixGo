/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-8 18:18:01
* @Description: 日志系统的核心实现，提供统一的日志记录接口
 */

package logger

import (
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 全局变量定义
var (
	// Log 全局日志记录器实例
	// 提供结构化日志记录功能，支持多种输出格式和级别
	Log *zap.Logger

	// logFile 日志文件句柄
	// 用于将日志输出到文件系统
	logFile *os.File

	// logPath 日志文件路径
	// 默认为当前目录下的gocli.log文件
	logPath = "gocli.log"

	// loggerLock 日志系统互斥锁
	// 保护日志系统初始化和配置的并发安全
	loggerLock sync.Mutex

	// initialized 初始化状态标志
	// 标记日志系统是否已经完成初始化
	initialized bool
)

// LogLevel 日志级别类型定义
type LogLevel = zapcore.Level

// 预定义的日志级别常量
const (
	DebugLevel LogLevel = zapcore.DebugLevel // 调试级别，最详细的日志信息
	InfoLevel  LogLevel = zapcore.InfoLevel  // 信息级别，一般性信息记录
	WarnLevel  LogLevel = zapcore.WarnLevel  // 警告级别，需要注意的问题
	ErrorLevel LogLevel = zapcore.ErrorLevel // 错误级别，错误信息记录
	FatalLevel LogLevel = zapcore.FatalLevel // 致命级别，程序无法继续运行
)

// LoggerConfig 日志配置结构
//
// 提供日志系统的完整配置选项，包括输出格式、级别、文件路径等
type LoggerConfig struct {
	FilePath      string   `json:"file_path"`      // 日志文件路径
	Level         LogLevel `json:"level"`          // 日志级别
	EnableConsole bool     `json:"enable_console"` // 是否启用控制台输出
	EnableFile    bool     `json:"enable_file"`    // 是否启用文件输出
	MaxSize       int      `json:"max_size"`       // 日志文件最大大小(MB)
	MaxBackups    int      `json:"max_backups"`    // 保留的旧日志文件数量
	MaxAge        int      `json:"max_age"`        // 日志文件保留天数
}

// DefaultConfig 返回默认的日志配置
//
// 返回:
//   - *LoggerConfig: 包含合理默认值的日志配置
//
// 默认配置包括：
//   - 文件路径：gocli.log
//   - 日志级别：Debug
//   - 同时启用控制台和文件输出
//   - 文件大小限制：100MB
//   - 保留10个备份文件
//   - 保留30天的日志
func DefaultConfig() *LoggerConfig {
	return &LoggerConfig{
		FilePath:      "gocli.log",
		Level:         DebugLevel,
		EnableConsole: true,
		EnableFile:    true,
		MaxSize:       100,
		MaxBackups:    10,
		MaxAge:        30,
	}
}

// SetLogPath 设置日志文件路径
//
// 该函数以线程安全的方式更新日志文件路径。
// 如果日志系统已经初始化，需要重新初始化才能生效。
//
// 参数:
//   - path: 新的日志文件路径
//
// 注意：路径变更后需要调用InitLogger()重新初始化才能生效
func SetLogPath(path string) {
	loggerLock.Lock()
	defer loggerLock.Unlock()
	logPath = path
}

// GetLogPath 获取当前日志文件路径
//
// 该函数以线程安全的方式返回当前配置的日志文件路径
//
// 返回:
//   - string: 当前的日志文件路径
func GetLogPath() string {
	loggerLock.Lock()
	defer loggerLock.Unlock()
	return logPath
}

// IsInitialized 检查日志系统是否已初始化
//
// 返回:
//   - bool: true表示已初始化，false表示未初始化
func IsInitialized() bool {
	loggerLock.Lock()
	defer loggerLock.Unlock()
	return initialized
}

// InitLogger 使用默认配置初始化日志系统
//
// 该函数使用默认配置初始化日志系统，包括文件和控制台输出
//
// 返回:
//   - error: 初始化过程中的错误，nil表示成功
//
// 默认行为：
//   - 同时输出到文件和控制台
//   - 使用JSON格式记录到文件
//   - 使用人类可读格式输出到控制台
//   - 自动添加调用栈信息
func InitLogger() error {
	return InitLoggerWithConfig(DefaultConfig())
}

// InitLoggerWithConfig 使用指定配置初始化日志系统
//
// 该函数是日志系统的核心初始化方法，支持完全自定义的配置
//
// 参数:
//   - config: 日志配置对象，包含所有必要的配置选项
//
// 返回:
//   - error: 初始化过程中的错误，nil表示成功
//
// 初始化流程：
// 1. 检查并关闭已存在的日志系统
// 2. 根据配置创建编码器
// 3. 设置输出目标（文件/控制台）
// 4. 创建多路输出核心
// 5. 配置调用栈和错误级别
func InitLoggerWithConfig(config *LoggerConfig) error {
	loggerLock.Lock()
	defer loggerLock.Unlock()

	// 如果已经初始化，先进行清理
	if err := cleanupExistingLogger(); err != nil {
		return fmt.Errorf("清理现有日志系统失败: %w", err)
	}

	// 创建编码器配置
	encoderConfig := createEncoderConfig()

	// 创建输出核心
	cores, err := createLoggerCores(config, encoderConfig)
	if err != nil {
		return fmt.Errorf("创建日志核心失败: %w", err)
	}

	// 创建最终的日志记录器
	Log = zap.New(
		zapcore.NewTee(cores...),
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCallerSkip(1), // 跳过包装函数的调用栈
	)

	// 更新全局状态
	logPath = config.FilePath
	initialized = true

	return nil
}

// cleanupExistingLogger 清理现有的日志系统
//
// 该函数负责安全地关闭和清理已存在的日志系统资源
//
// 返回:
//   - error: 清理过程中的错误，nil表示成功
func cleanupExistingLogger() error {
	if !initialized {
		return nil
	}

	// 同步并关闭现有的日志记录器
	if Log != nil {
		if err := Log.Sync(); err != nil {
			// 忽略同步错误，但记录警告
			fmt.Printf("警告: 日志同步失败: %v\n", err)
		}
		Log = nil
	}

	// 关闭日志文件
	if logFile != nil {
		if err := logFile.Close(); err != nil {
			return fmt.Errorf("关闭日志文件失败: %w", err)
		}
		logFile = nil
	}

	initialized = false
	return nil
}

// createEncoderConfig 创建编码器配置
//
// 该函数创建统一的编码器配置，用于格式化日志输出
//
// 返回:
//   - zapcore.EncoderConfig: 配置好的编码器配置
func createEncoderConfig() zapcore.EncoderConfig {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncodeLevel = zapcore.CapitalLevelEncoder
	config.EncodeCaller = zapcore.ShortCallerEncoder
	return config
}

// createLoggerCores 创建日志输出核心
//
// 根据配置创建相应的日志输出核心，支持文件和控制台输出
//
// 参数:
//   - config: 日志配置
//   - encoderConfig: 编码器配置
//
// 返回:
//   - []zapcore.Core: 日志核心列表
//   - error: 创建过程中的错误，nil表示成功
func createLoggerCores(config *LoggerConfig, encoderConfig zapcore.EncoderConfig) ([]zapcore.Core, error) {
	var cores []zapcore.Core

	// 创建文件输出核心
	if config.EnableFile {
		fileCore, err := createFileCore(config, encoderConfig)
		if err != nil {
			return nil, fmt.Errorf("创建文件输出核心失败: %w", err)
		}
		cores = append(cores, fileCore)
	}

	// 创建控制台输出核心
	if config.EnableConsole {
		consoleCore := createConsoleCore(config, encoderConfig)
		cores = append(cores, consoleCore)
	}

	if len(cores) == 0 {
		return nil, fmt.Errorf("至少需要启用一种输出方式（文件或控制台）")
	}

	return cores, nil
}

// createFileCore 创建文件输出核心
//
// 该函数创建用于文件输出的日志核心
//
// 参数:
//   - config: 日志配置
//   - encoderConfig: 编码器配置
//
// 返回:
//   - zapcore.Core: 文件输出核心
//   - error: 创建过程中的错误，nil表示成功
func createFileCore(config *LoggerConfig, encoderConfig zapcore.EncoderConfig) (zapcore.Core, error) {
	var err error
	logFile, err = os.OpenFile(config.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("无法打开日志文件 %s: %w", config.FilePath, err)
	}

	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
	writer := zapcore.AddSync(logFile)

	return zapcore.NewCore(fileEncoder, writer, config.Level), nil
}

// createConsoleCore 创建控制台输出核心
//
// 该函数创建用于控制台输出的日志核心
//
// 参数:
//   - config: 日志配置
//   - encoderConfig: 编码器配置
//
// 返回:
//   - zapcore.Core: 控制台输出核心
func createConsoleCore(config *LoggerConfig, encoderConfig zapcore.EncoderConfig) zapcore.Core {
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	writer := zapcore.AddSync(os.Stdout)

	return zapcore.NewCore(consoleEncoder, writer, config.Level)
}

// Close 关闭日志系统
//
// 该函数安全地关闭日志系统，释放所有相关资源
//
// 返回:
//   - error: 关闭过程中的错误，nil表示成功
//
// 关闭流程：
// 1. 同步所有待写入的日志
// 2. 关闭日志文件句柄
// 3. 重置全局状态
func Close() error {
	loggerLock.Lock()
	defer loggerLock.Unlock()

	return cleanupExistingLogger()
}

// checkInitialized 检查日志系统是否已初始化
//
// 该函数检查日志系统的初始化状态，如果未初始化则触发panic
// 这是一个内部函数，用于确保所有日志操作都在正确初始化后进行
//
// Panic条件：
//   - 日志系统未初始化
//   - Log实例为nil
func checkInitialized() {
	if !initialized || Log == nil {
		panic("日志系统未初始化，请先调用InitLogger()或InitLoggerWithConfig()")
	}
}

// Info 记录信息级别日志
//
// 该函数记录一般性信息，用于跟踪程序的正常执行流程
//
// 参数:
//   - msg: 日志消息内容
//   - fields: 可选的结构化字段，用于添加上下文信息
//
// 示例:
//
//	logger.Info("用户登录成功", zap.String("user_id", "12345"), zap.String("ip", "192.168.1.1"))
func Info(msg string, fields ...zap.Field) {
	checkInitialized()
	Log.Info(msg, fields...)
}

// Error 记录错误级别日志
//
// 该函数记录错误信息，用于跟踪程序执行过程中的错误
//
// 参数:
//   - msg: 错误消息内容
//   - fields: 可选的结构化字段，用于添加错误上下文
//
// 示例:
//
//	logger.Error("数据库连接失败", zap.Error(err), zap.String("database", "mysql"))
func Error(msg string, fields ...zap.Field) {
	checkInitialized()
	Log.Error(msg, fields...)
}

// Debug 记录调试级别日志
//
// 该函数记录详细的调试信息，通常用于开发和故障排查
//
// 参数:
//   - msg: 调试消息内容
//   - fields: 可选的结构化字段，用于添加调试上下文
//
// 示例:
//
//	logger.Debug("处理请求", zap.String("method", "GET"), zap.String("path", "/api/users"))
func Debug(msg string, fields ...zap.Field) {
	checkInitialized()
	Log.Debug(msg, fields...)
}

// Warn 记录警告级别日志
//
// 该函数记录警告信息，用于标记需要注意但不影响程序继续运行的问题
//
// 参数:
//   - msg: 警告消息内容
//   - fields: 可选的结构化字段，用于添加警告上下文
//
// 示例:
//
//	logger.Warn("配置文件缺失，使用默认配置", zap.String("config_file", "app.yaml"))
func Warn(msg string, fields ...zap.Field) {
	checkInitialized()
	Log.Warn(msg, fields...)
}

// Fatal 记录致命级别日志并退出程序
//
// 该函数记录致命错误信息，并调用os.Exit(1)终止程序
//
// 参数:
//   - msg: 致命错误消息内容
//   - fields: 可选的结构化字段，用于添加错误上下文
//
// 警告：此函数会导致程序立即退出，请谨慎使用
//
// 示例:
//
//	logger.Fatal("无法启动服务器", zap.Error(err), zap.Int("port", 8080))
func Fatal(msg string, fields ...zap.Field) {
	checkInitialized()
	Log.Fatal(msg, fields...)
}

// With 创建带有预设字段的子日志记录器
//
// 该函数创建一个新的日志记录器，预设了指定的字段，
// 后续使用该记录器记录的所有日志都会包含这些字段
//
// 参数:
//   - fields: 要预设的结构化字段
//
// 返回:
//   - *zap.Logger: 带有预设字段的新日志记录器
//
// 示例:
//
//	userLogger := logger.With(zap.String("user_id", "12345"))
//	userLogger.Info("执行操作") // 自动包含user_id字段
func With(fields ...zap.Field) *zap.Logger {
	checkInitialized()
	return Log.With(fields...)
}

// Sync 同步所有缓冲的日志条目
//
// 该函数强制将所有缓冲的日志条目写入到输出目标
// 通常在程序退出前调用以确保所有日志都被正确写入
//
// 返回:
//   - error: 同步过程中的错误，nil表示成功
func Sync() error {
	if !initialized || Log == nil {
		return nil
	}
	return Log.Sync()
}
