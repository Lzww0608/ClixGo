/*
* @Author: Lzww0608
* @Date: 2025-6-1 21:09:49
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-1 21:09:52
* @Description: 通用工具函数
 */

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/errors"
)

// StringUtils 字符串工具函数
type StringUtils struct{}

// IsEmpty 检查字符串是否为空
func (s StringUtils) IsEmpty(str string) bool {
	return strings.TrimSpace(str) == ""
}

// IsNotEmpty 检查字符串是否非空
func (s StringUtils) IsNotEmpty(str string) bool {
	return !s.IsEmpty(str)
}

// DefaultIfEmpty 如果字符串为空则返回默认值
func (s StringUtils) DefaultIfEmpty(str, defaultValue string) string {
	if s.IsEmpty(str) {
		return defaultValue
	}
	return str
}

// IsNumeric 检查字符串是否为数字
func (s StringUtils) IsNumeric(str string) bool {
	_, err := strconv.Atoi(str)
	return err == nil
}

// SplitAndTrim 分割字符串并去除空白
func (s StringUtils) SplitAndTrim(str, sep string) []string {
	parts := strings.Split(str, sep)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// FileUtils 文件工具函数
type FileUtils struct{}

// Exists 检查文件或目录是否存在
func (f FileUtils) Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// IsFile 检查路径是否为文件
func (f FileUtils) IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// IsDir 检查路径是否为目录
func (f FileUtils) IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// EnsureDir 确保目录存在，如果不存在则创建
func (f FileUtils) EnsureDir(dir string) error {
	if !f.Exists(dir) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.Wrap(err, errors.ErrCodeFileNotFound, "创建目录失败")
		}
	}
	return nil
}

// SafeWriteFile 安全写入文件（先写入临时文件再重命名）
func (f FileUtils) SafeWriteFile(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	if err := f.EnsureDir(dir); err != nil {
		return err
	}

	tmpFile := filename + ".tmp"
	if err := os.WriteFile(tmpFile, data, perm); err != nil {
		return errors.Wrap(err, errors.ErrCodeFileNotFound, "写入临时文件失败")
	}

	if err := os.Rename(tmpFile, filename); err != nil {
		os.Remove(tmpFile) // 清理临时文件
		return errors.Wrap(err, errors.ErrCodeFileNotFound, "重命名文件失败")
	}

	return nil
}

// GetFileSize 获取文件大小
func (f FileUtils) GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeFileNotFound, "获取文件信息失败")
	}
	return info.Size(), nil
}

// SliceUtils 切片工具函数
type SliceUtils struct{}

// Contains 检查切片是否包含指定元素
func (s SliceUtils) Contains(slice []string, item string) bool {
	for _, str := range slice {
		if str == item {
			return true
		}
	}
	return false
}

// ContainsInt 检查int切片是否包含指定元素
func (s SliceUtils) ContainsInt(slice []int, item int) bool {
	for _, i := range slice {
		if i == item {
			return true
		}
	}
	return false
}

// RemoveDuplicates 去除字符串切片中的重复元素
func (s SliceUtils) RemoveDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// FilterEmpty 过滤空字符串
func (s SliceUtils) FilterEmpty(slice []string) []string {
	result := make([]string, 0)
	stringUtils := StringUtils{}
	for _, str := range slice {
		if stringUtils.IsNotEmpty(str) {
			result = append(result, str)
		}
	}
	return result
}

// TimeUtils 时间工具函数
type TimeUtils struct{}

// Now 获取当前时间
func (t TimeUtils) Now() time.Time {
	return time.Now()
}

// FormatDuration 格式化时间间隔
func (t TimeUtils) FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.2fm", d.Minutes())
	}
	return fmt.Sprintf("%.2fh", d.Hours())
}

// IsTimeout 检查错误是否为超时错误
func (t TimeUtils) IsTimeout(err error) bool {
	return os.IsTimeout(err)
}

// WithTimeout 执行带超时的操作
func (t TimeUtils) WithTimeout(timeout time.Duration, fn func() error) error {
	done := make(chan error, 1)

	go func() {
		done <- fn()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return errors.New(errors.ErrCodeTimeout, "操作超时")
	}
}

// ValidationUtils 验证工具函数
type ValidationUtils struct{}

// ValidateNotEmpty 验证字符串非空
func (v ValidationUtils) ValidateNotEmpty(value, fieldName string) error {
	stringUtils := StringUtils{}
	if stringUtils.IsEmpty(value) {
		return errors.InvalidParam(fieldName, "不能为空")
	}
	return nil
}

// ValidatePositive 验证数字为正数
func (v ValidationUtils) ValidatePositive(value int, fieldName string) error {
	if value <= 0 {
		return errors.InvalidParam(fieldName, fmt.Sprintf("必须为正数，当前值: %d", value))
	}
	return nil
}

// ValidateInRange 验证数字在指定范围内
func (v ValidationUtils) ValidateInRange(value, min, max int, fieldName string) error {
	if value < min || value > max {
		return errors.InvalidParam(fieldName, fmt.Sprintf("必须在%d-%d范围内，当前值: %d", min, max, value))
	}
	return nil
}

// RetryUtils 重试工具函数
type RetryUtils struct{}

// Retry 重试执行函数
func (r RetryUtils) Retry(attempts int, delay time.Duration, fn func() error) error {
	var lastErr error

	for i := 0; i < attempts; i++ {
		if err := fn(); err != nil {
			lastErr = err
			if i < attempts-1 {
				time.Sleep(delay)
				delay *= 2 // 指数退避
			}
			continue
		}
		return nil
	}

	return errors.Wrapf(lastErr, errors.ErrCodeInternal, "重试%d次后仍然失败", attempts)
}

// RetryWithBackoff 带退避策略的重试
func (r RetryUtils) RetryWithBackoff(attempts int, initialDelay, maxDelay time.Duration, fn func() error) error {
	var lastErr error
	delay := initialDelay

	for i := 0; i < attempts; i++ {
		if err := fn(); err != nil {
			lastErr = err
			if i < attempts-1 {
				time.Sleep(delay)
				delay *= 2
				if delay > maxDelay {
					delay = maxDelay
				}
			}
			continue
		}
		return nil
	}

	return errors.Wrapf(lastErr, errors.ErrCodeInternal, "重试%d次后仍然失败", attempts)
}

// 全局工具实例
var (
	Strings    = StringUtils{}
	Files      = FileUtils{}
	Slices     = SliceUtils{}
	Times      = TimeUtils{}
	Validation = ValidationUtils{}
	Retry      = RetryUtils{}
)
