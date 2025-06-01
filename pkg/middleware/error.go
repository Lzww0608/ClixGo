/*
* @Author: Lzww0608
* @Date: 2025-6-1 21:10:10
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-1 21:10:13
* @Description: 错误处理中间件
 */

package middleware

import (
	"context"
	"runtime/debug"

	"github.com/Lzww0608/ClixGo/pkg/errors"
	"github.com/Lzww0608/ClixGo/pkg/logger"
	"go.uber.org/zap"
)

// ErrorHandler 错误处理中间件
type ErrorHandler struct {
	logger logger.Logger
}

// NewErrorHandler 创建错误处理中间件
func NewErrorHandler(log logger.Logger) *ErrorHandler {
	return &ErrorHandler{logger: log}
}

// Recover 恢复panic并转换为标准错误
func (h *ErrorHandler) Recover() {
	if r := recover(); r != nil {
		stack := string(debug.Stack())

		var err *errors.ClixGoError
		switch v := r.(type) {
		case error:
			err = errors.Wrap(v, errors.ErrCodeInternal, "系统内部错误")
		case string:
			err = errors.New(errors.ErrCodeInternal, v)
		default:
			err = errors.New(errors.ErrCodeInternal, "未知错误")
		}

		err.Stack = stack
		h.logger.Error("panic recovered",
			zap.String("error", err.Error()),
			zap.String("stack", stack),
		)
	}
}

// HandleError 统一错误处理
func (h *ErrorHandler) HandleError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	var clixErr *errors.ClixGoError
	switch v := err.(type) {
	case *errors.ClixGoError:
		clixErr = v
	default:
		clixErr = errors.Wrap(err, errors.ErrCodeInternal, "内部处理错误")
	}

	// 记录错误日志
	h.logger.Error("operation failed",
		zap.String("code", string(clixErr.Code)),
		zap.String("message", clixErr.Message),
		zap.Any("context", clixErr.Context),
		zap.Error(clixErr.Cause),
	)

	return clixErr
}

// WrapOperation 包装操作并处理错误
func (h *ErrorHandler) WrapOperation(ctx context.Context, operation func() error) error {
	defer h.Recover()

	err := operation()
	return h.HandleError(ctx, err)
}

// SafeExecute 安全执行函数，自动处理panic和错误
func (h *ErrorHandler) SafeExecute(ctx context.Context, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())

			switch v := r.(type) {
			case error:
				err = errors.Wrap(v, errors.ErrCodeInternal, "执行过程中发生panic").WithStack()
			case string:
				err = errors.New(errors.ErrCodeInternal, v).WithStack()
			default:
				err = errors.New(errors.ErrCodeInternal, "未知panic").WithStack()
			}

			h.logger.Error("panic in safe execution",
				zap.String("error", err.Error()),
				zap.String("stack", stack),
			)
		}
	}()

	return h.HandleError(ctx, fn())
}

// LogAndReturn 记录错误并返回
func (h *ErrorHandler) LogAndReturn(err error, message string, fields ...zap.Field) error {
	if err == nil {
		return nil
	}

	allFields := append([]zap.Field{zap.Error(err)}, fields...)
	h.logger.Error(message, allFields...)

	if clixErr, ok := errors.IsClixGoError(err); ok {
		return clixErr
	}

	return errors.Wrap(err, errors.ErrCodeInternal, message)
}

// CheckAndWrap 检查错误并包装为ClixGo错误
func CheckAndWrap(err error, code errors.ErrorCode, message string) error {
	if err == nil {
		return nil
	}

	if _, ok := errors.IsClixGoError(err); ok {
		return err
	}

	return errors.Wrap(err, code, message)
}

// MustSucceed 确保操作成功，否则panic
func MustSucceed(err error) {
	if err != nil {
		panic(err)
	}
}
