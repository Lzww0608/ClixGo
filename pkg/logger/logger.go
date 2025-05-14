package logger

import (
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Log         *zap.Logger
	logFile     *os.File
	logPath     = "gocli.log"
	loggerLock  sync.Mutex
	initialized bool
)

// SetLogPath 设置日志文件路径
func SetLogPath(path string) {
	loggerLock.Lock()
	defer loggerLock.Unlock()
	logPath = path
}

// GetLogPath 获取当前日志文件路径
func GetLogPath() string {
	loggerLock.Lock()
	defer loggerLock.Unlock()
	return logPath
}

// InitLogger 初始化日志系统
func InitLogger() error {
	loggerLock.Lock()
	defer loggerLock.Unlock()

	// 如果已经初始化，先关闭旧的
	if initialized && logFile != nil {
		logFile.Close()
	}

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncodeLevel = zapcore.CapitalLevelEncoder

	fileEncoder := zapcore.NewJSONEncoder(config)
	consoleEncoder := zapcore.NewConsoleEncoder(config)

	var err error
	logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("无法打开日志文件: %w", err)
	}

	writer := zapcore.AddSync(logFile)
	defaultLogLevel := zapcore.DebugLevel
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, writer, defaultLogLevel),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), defaultLogLevel),
	)

	Log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	initialized = true
	return nil
}

// Close 关闭日志系统
func Close() error {
	loggerLock.Lock()
	defer loggerLock.Unlock()

	if !initialized {
		return nil
	}

	if Log != nil {
		Log.Sync()
	}

	if logFile != nil {
		err := logFile.Close()
		logFile = nil
		initialized = false
		return err
	}

	return nil
}

// 检查logger是否已初始化
func checkInitialized() {
	if Log == nil {
		panic("logger未初始化，请先调用InitLogger()")
	}
}

func Info(msg string, fields ...zap.Field) {
	checkInitialized()
	Log.Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	checkInitialized()
	Log.Error(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	checkInitialized()
	Log.Debug(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	checkInitialized()
	Log.Warn(msg, fields...)
}
