/*
* @Author: Lzww0608
* @Date: 2025-6-1 21:10:02
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-1 21:10:05
* @Description: Logger接口定义
 */

package logger

import (
	"context"

	"go.uber.org/zap"
)

// Logger 标准日志接口
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)

	// 带上下文的日志记录
	WithContext(ctx context.Context) Logger
	WithFields(fields map[string]interface{}) Logger

	// 记录调用栈
	WithStack() Logger

	// 同步日志（确保写入）
	Sync() error
}

// Field 日志字段辅助函数
var (
	StringField    = zap.String
	IntField       = zap.Int
	Int64Field     = zap.Int64
	Float64Field   = zap.Float64
	BoolField      = zap.Bool
	AnyField       = zap.Any
	ErrorField     = zap.Error
	DurationField  = zap.Duration
	TimeField      = zap.Time
	NamespaceField = zap.Namespace
)

// zapLogger 基于zap的Logger实现
type zapLogger struct {
	logger *zap.Logger
}

// NewZapLogger 创建基于zap的Logger
func NewZapLogger(logger *zap.Logger) Logger {
	return &zapLogger{logger: logger}
}

func (l *zapLogger) Debug(msg string, fields ...zap.Field) {
	l.logger.Debug(msg, fields...)
}

func (l *zapLogger) Info(msg string, fields ...zap.Field) {
	l.logger.Info(msg, fields...)
}

func (l *zapLogger) Warn(msg string, fields ...zap.Field) {
	l.logger.Warn(msg, fields...)
}

func (l *zapLogger) Error(msg string, fields ...zap.Field) {
	l.logger.Error(msg, fields...)
}

func (l *zapLogger) Fatal(msg string, fields ...zap.Field) {
	l.logger.Fatal(msg, fields...)
}

func (l *zapLogger) WithContext(ctx context.Context) Logger {
	// 从context中提取追踪信息等
	return &zapLogger{logger: l.logger}
}

func (l *zapLogger) WithFields(fields map[string]interface{}) Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return &zapLogger{logger: l.logger.With(zapFields...)}
}

func (l *zapLogger) WithStack() Logger {
	return &zapLogger{logger: l.logger.WithOptions(zap.AddStacktrace(zap.InfoLevel))}
}

func (l *zapLogger) Sync() error {
	return l.logger.Sync()
}

// GetDefaultLogger 获取默认Logger
func GetDefaultLogger() Logger {
	if Log == nil {
		// 如果还没初始化，返回一个nop logger
		return NewZapLogger(zap.NewNop())
	}
	return NewZapLogger(Log)
}
