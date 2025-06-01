/*
* @Author: Lzww0608
* @Date: 2025-6-1 21:09:25
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-1 21:26:15
* @Description: ClixGo 统一错误处理框架
 */

package errors

import (
	"fmt"
	"runtime"
	"time"
)

// ErrorCode 错误码类型
type ErrorCode string

// 预定义错误码
const (
	// 通用错误
	ErrCodeUnknown  ErrorCode = "UNKNOWN"
	ErrCodeInternal ErrorCode = "INTERNAL"
	ErrCodeInvalid  ErrorCode = "INVALID"
	ErrCodeNotFound ErrorCode = "NOT_FOUND"
	ErrCodeExists   ErrorCode = "EXISTS"
	ErrCodeTimeout  ErrorCode = "TIMEOUT"
	ErrCodeCanceled ErrorCode = "CANCELED"

	// 会话管理错误
	ErrCodeSessionNotFound ErrorCode = "SESSION_NOT_FOUND"
	ErrCodeSessionExists   ErrorCode = "SESSION_EXISTS"
	ErrCodeSessionInactive ErrorCode = "SESSION_INACTIVE"
	ErrCodeWindowNotFound  ErrorCode = "WINDOW_NOT_FOUND"
	ErrCodePaneNotFound    ErrorCode = "PANE_NOT_FOUND"

	// 文件系统错误
	ErrCodeFileNotFound     ErrorCode = "FILE_NOT_FOUND"
	ErrCodeFileExists       ErrorCode = "FILE_EXISTS"
	ErrCodePermissionDenied ErrorCode = "PERMISSION_DENIED"
	ErrCodeDiskFull         ErrorCode = "DISK_FULL"

	// 网络错误
	ErrCodeNetworkUnreachable ErrorCode = "NETWORK_UNREACHABLE"
	ErrCodeConnectionRefused  ErrorCode = "CONNECTION_REFUSED"
	ErrCodeConnectionTimeout  ErrorCode = "CONNECTION_TIMEOUT"

	// 配置错误
	ErrCodeConfigInvalid  ErrorCode = "CONFIG_INVALID"
	ErrCodeConfigNotFound ErrorCode = "CONFIG_NOT_FOUND"

	// 插件错误
	ErrCodePluginNotFound   ErrorCode = "PLUGIN_NOT_FOUND"
	ErrCodePluginLoadFailed ErrorCode = "PLUGIN_LOAD_FAILED"

	// 任务错误
	ErrCodeTaskNotFound ErrorCode = "TASK_NOT_FOUND"
	ErrCodeTaskCanceled ErrorCode = "TASK_CANCELED"
	ErrCodeTaskFailed   ErrorCode = "TASK_FAILED"
)

// ClixGoError ClixGo项目的标准错误类型
type ClixGoError struct {
	Code      ErrorCode              `json:"code"`
	Message   string                 `json:"message"`
	Details   string                 `json:"details,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Cause     error                  `json:"-"`
	Stack     string                 `json:"stack,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	File      string                 `json:"file,omitempty"`
	Line      int                    `json:"line,omitempty"`
}

// Error 实现error接口
func (e *ClixGoError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 实现errors.Unwrap接口
func (e *ClixGoError) Unwrap() error {
	return e.Cause
}

// WithContext 添加上下文信息
func (e *ClixGoError) WithContext(key string, value interface{}) *ClixGoError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithStack 添加调用栈信息
func (e *ClixGoError) WithStack() *ClixGoError {
	if e.Stack == "" {
		stack := make([]byte, 4096)
		stack = stack[:runtime.Stack(stack, false)]
		e.Stack = string(stack)
	}
	return e
}

// New 创建新的错误
func New(code ErrorCode, message string) *ClixGoError {
	_, file, line, _ := runtime.Caller(1)
	return &ClixGoError{
		Code:      code,
		Message:   message,
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
		File:      file,
		Line:      line,
	}
}

// Newf 创建带格式化消息的错误
func Newf(code ErrorCode, format string, args ...interface{}) *ClixGoError {
	return New(code, fmt.Sprintf(format, args...))
}

// Wrap 包装现有错误
func Wrap(err error, code ErrorCode, message string) *ClixGoError {
	_, file, line, _ := runtime.Caller(1)
	return &ClixGoError{
		Code:      code,
		Message:   message,
		Cause:     err,
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
		File:      file,
		Line:      line,
	}
}

// Wrapf 包装现有错误并格式化消息
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *ClixGoError {
	return Wrap(err, code, fmt.Sprintf(format, args...))
}

// WithDetails 添加详细信息
func (e *ClixGoError) WithDetails(details string) *ClixGoError {
	e.Details = details
	return e
}

// HasErrorCode 检查错误码是否匹配
func (e *ClixGoError) HasErrorCode(code ErrorCode) bool {
	return e.Code == code
}

// IsClixGoError 检查是否为ClixGo错误
func IsClixGoError(err error) (*ClixGoError, bool) {
	if clixErr, ok := err.(*ClixGoError); ok {
		return clixErr, true
	}
	return nil, false
}

// HasCode 检查错误是否有指定代码
func HasCode(err error, code ErrorCode) bool {
	if clixErr, ok := IsClixGoError(err); ok {
		return clixErr.HasErrorCode(code)
	}
	return false
}

// 错误码对应的默认消息
var defaultMessages = map[ErrorCode]string{
	ErrCodeUnknown:            "未知错误",
	ErrCodeInternal:           "内部错误",
	ErrCodeInvalid:            "无效参数",
	ErrCodeNotFound:           "资源未找到",
	ErrCodeExists:             "资源已存在",
	ErrCodeTimeout:            "操作超时",
	ErrCodeCanceled:           "操作已取消",
	ErrCodeSessionNotFound:    "会话未找到",
	ErrCodeSessionExists:      "会话已存在",
	ErrCodeSessionInactive:    "会话未激活",
	ErrCodeWindowNotFound:     "窗口未找到",
	ErrCodePaneNotFound:       "面板未找到",
	ErrCodeFileNotFound:       "文件未找到",
	ErrCodeFileExists:         "文件已存在",
	ErrCodePermissionDenied:   "权限被拒绝",
	ErrCodeDiskFull:           "磁盘空间不足",
	ErrCodeNetworkUnreachable: "网络不可达",
	ErrCodeConnectionRefused:  "连接被拒绝",
	ErrCodeConnectionTimeout:  "连接超时",
	ErrCodeConfigInvalid:      "配置无效",
	ErrCodeConfigNotFound:     "配置未找到",
	ErrCodePluginNotFound:     "插件未找到",
	ErrCodePluginLoadFailed:   "插件加载失败",
	ErrCodeTaskNotFound:       "任务未找到",
	ErrCodeTaskCanceled:       "任务已取消",
	ErrCodeTaskFailed:         "任务执行失败",
}

// GetDefaultMessage 获取错误码的默认消息
func GetDefaultMessage(code ErrorCode) string {
	if msg, exists := defaultMessages[code]; exists {
		return msg
	}
	return string(code)
}

// NewWithDefault 使用默认消息创建错误
func NewWithDefault(code ErrorCode) *ClixGoError {
	return New(code, GetDefaultMessage(code))
}

// 常用错误构造函数
func NotFound(resource string) *ClixGoError {
	return New(ErrCodeNotFound, fmt.Sprintf("%s未找到", resource))
}

func AlreadyExists(resource string) *ClixGoError {
	return New(ErrCodeExists, fmt.Sprintf("%s已存在", resource))
}

func InvalidParam(param string, value interface{}) *ClixGoError {
	return New(ErrCodeInvalid, fmt.Sprintf("参数%s无效: %v", param, value))
}

func Internal(message string) *ClixGoError {
	return New(ErrCodeInternal, message).WithStack()
}

func SessionNotFound(sessionID string) *ClixGoError {
	return New(ErrCodeSessionNotFound, fmt.Sprintf("会话未找到: %s", sessionID))
}

func SessionExists(sessionName string) *ClixGoError {
	return New(ErrCodeSessionExists, fmt.Sprintf("会话已存在: %s", sessionName))
}

func WindowNotFound(windowIndex int) *ClixGoError {
	return New(ErrCodeWindowNotFound, fmt.Sprintf("窗口未找到: %d", windowIndex))
}

func PaneNotFound(paneIndex int) *ClixGoError {
	return New(ErrCodePaneNotFound, fmt.Sprintf("面板未找到: %d", paneIndex))
}
