/*
* @Author: Lzww0608
* @Date: 2025-06-09 15:36:52
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-09 15:36:52
* @Description: 错误处理接口
 */

package interfaces

import (
	"context"
	"time"
)

// ErrorCode 错误码类型
type ErrorCode string

const (
	// 通用错误码
	ErrCodeUnknown          ErrorCode = "UNKNOWN"
	ErrCodeInvalidParam     ErrorCode = "INVALID_PARAM"
	ErrCodeNotFound         ErrorCode = "NOT_FOUND"
	ErrCodePermissionDenied ErrorCode = "PERMISSION_DENIED"
	ErrCodeTimeout          ErrorCode = "TIMEOUT"
	ErrCodeCanceled         ErrorCode = "CANCELED"

	// 业务错误码
	ErrCodeSessionNotFound ErrorCode = "SESSION_NOT_FOUND"
	ErrCodeCommandFailed   ErrorCode = "COMMAND_FAILED"
	ErrCodeNetworkError    ErrorCode = "NETWORK_ERROR"
	ErrCodeFileSystemError ErrorCode = "FILESYSTEM_ERROR"
	ErrCodeConfigError     ErrorCode = "CONFIG_ERROR"
)

// ClixGoError 标准化错误接口
type ClixGoError interface {
	error

	// Code 获取错误码
	Code() ErrorCode

	// Message 获取错误消息
	Message() string

	// Cause 获取原始错误
	Cause() error

	// Context 获取错误上下文
	Context() map[string]interface{}

	// Timestamp 获取错误发生时间
	Timestamp() time.Time

	// Stack 获取错误堆栈信息
	Stack() string

	// WithContext 添加上下文信息
	WithContext(key string, value interface{}) ClixGoError

	// WithCause 添加原因错误
	WithCause(err error) ClixGoError
}

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	// Handle 处理错误
	Handle(err error) error

	// Wrap 包装错误并添加消息
	Wrap(err error, msg string) error

	// WrapWithCode 包装错误并添加错误码和消息
	WrapWithCode(err error, code ErrorCode, msg string) error

	// New 创建新错误
	New(msg string) error

	// NewWithCode 创建带错误码的新错误
	NewWithCode(code ErrorCode, msg string) error

	// IsErrorCode 检查错误是否包含指定错误码
	IsErrorCode(err error, code ErrorCode) bool

	// GetErrorCode 获取错误码
	GetErrorCode(err error) ErrorCode

	// WithContext 为错误添加上下文
	WithContext(err error, ctx context.Context) error

	// WithMetadata 为错误添加元数据
	WithMetadata(err error, key string, value interface{}) error
}

// ErrorRecovery 错误恢复接口
type ErrorRecovery interface {
	// Recover 从panic中恢复
	Recover() error

	// RecoverWithHandler 使用指定处理器从panic中恢复
	RecoverWithHandler(handler func(interface{})) error

	// SafeGo 安全执行goroutine
	SafeGo(fn func() error) <-chan error

	// SafeGoWithTimeout 带超时的安全执行goroutine
	SafeGoWithTimeout(fn func() error, timeout time.Duration) <-chan error
}

// ErrorReporter 错误报告接口
type ErrorReporter interface {
	// Report 报告错误
	Report(err error) error

	// ReportAsync 异步报告错误
	ReportAsync(err error)

	// ReportWithLevel 按级别报告错误
	ReportWithLevel(err error, level string) error

	// SetReportHandler 设置报告处理器
	SetReportHandler(handler func(error))
}

// ErrorValidator 错误验证接口
type ErrorValidator interface {
	// IsRetryable 检查错误是否可重试
	IsRetryable(err error) bool

	// IsTemporary 检查错误是否是临时的
	IsTemporary(err error) bool

	// IsFatal 检查错误是否是致命的
	IsFatal(err error) bool

	// GetSeverity 获取错误严重程度
	GetSeverity(err error) string
}
