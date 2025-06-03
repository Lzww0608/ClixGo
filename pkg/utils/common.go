/*
* @Author: Lzww0608
* @Date: 2025-6-1 21:09:49
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-03 19:30:00
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

// StringUtils 字符串工具函数集合
// 提供常用的字符串操作和验证功能
type StringUtils struct{}

// IsEmpty 检查字符串是否为空或仅包含空白字符
//
// 参数:
//   - str: 要检查的字符串
//
// 返回:
//   - bool: true表示字符串为空或仅包含空白字符
func (s StringUtils) IsEmpty(str string) bool {
	return strings.TrimSpace(str) == ""
}

// IsNotEmpty 检查字符串是否非空且包含非空白字符
//
// 参数:
//   - str: 要检查的字符串
//
// 返回:
//   - bool: true表示字符串非空且包含非空白字符
func (s StringUtils) IsNotEmpty(str string) bool {
	return !s.IsEmpty(str)
}

// DefaultIfEmpty 如果字符串为空则返回默认值
//
// 参数:
//   - str: 要检查的字符串
//   - defaultValue: 默认值
//
// 返回:
//   - string: 如果str为空则返回defaultValue，否则返回str
func (s StringUtils) DefaultIfEmpty(str, defaultValue string) string {
	if s.IsEmpty(str) {
		return defaultValue
	}
	return str
}

// IsNumeric 检查字符串是否为有效的整数
//
// 参数:
//   - str: 要检查的字符串
//
// 返回:
//   - bool: true表示字符串可以转换为整数
func (s StringUtils) IsNumeric(str string) bool {
	_, err := strconv.Atoi(str)
	return err == nil
}

// SplitAndTrim 分割字符串并去除每个部分的空白字符
//
// 参数:
//   - str: 要分割的字符串
//   - sep: 分隔符
//
// 返回:
//   - []string: 分割后的非空字符串切片
//
// 该函数会自动过滤掉空字符串
func (s StringUtils) SplitAndTrim(str, sep string) []string {
	stringParts := strings.Split(str, sep)
	resultParts := make([]string, 0, len(stringParts))
	for _, part := range stringParts {
		if trimmedPart := strings.TrimSpace(part); trimmedPart != "" {
			resultParts = append(resultParts, trimmedPart)
		}
	}
	return resultParts
}

// FileUtils 文件操作工具函数集合
// 提供常用的文件和目录操作功能
type FileUtils struct{}

// Exists 检查文件或目录是否存在
//
// 参数:
//   - path: 文件或目录路径
//
// 返回:
//   - bool: true表示文件或目录存在
func (f FileUtils) Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// IsFile 检查路径是否为文件（非目录）
//
// 参数:
//   - path: 要检查的路径
//
// 返回:
//   - bool: true表示路径是文件
func (f FileUtils) IsFile(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !fileInfo.IsDir()
}

// IsDir 检查路径是否为目录
//
// 参数:
//   - path: 要检查的路径
//
// 返回:
//   - bool: true表示路径是目录
func (f FileUtils) IsDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}

// EnsureDir 确保目录存在，如果不存在则创建
//
// 参数:
//   - dir: 目录路径
//
// 返回:
//   - error: 创建目录时的错误，nil表示成功
//
// 该函数会递归创建所有必要的父目录
func (f FileUtils) EnsureDir(dir string) error {
	if !f.Exists(dir) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.Wrap(err, errors.ErrCodeFileNotFound, "创建目录失败")
		}
	}
	return nil
}

// SafeWriteFile 安全写入文件（先写入临时文件再重命名）
//
// 参数:
//   - filename: 目标文件名
//   - data: 要写入的数据
//   - perm: 文件权限
//
// 返回:
//   - error: 写入错误，nil表示成功
//
// 该函数使用原子操作确保文件写入的安全性
func (f FileUtils) SafeWriteFile(filename string, data []byte, perm os.FileMode) error {
	targetDir := filepath.Dir(filename)
	if err := f.EnsureDir(targetDir); err != nil {
		return err
	}

	tempFileName := filename + ".tmp"
	if err := os.WriteFile(tempFileName, data, perm); err != nil {
		return errors.Wrap(err, errors.ErrCodeFileNotFound, "写入临时文件失败")
	}

	if err := os.Rename(tempFileName, filename); err != nil {
		os.Remove(tempFileName) // 清理临时文件
		return errors.Wrap(err, errors.ErrCodeFileNotFound, "重命名文件失败")
	}

	return nil
}

// GetFileSize 获取文件大小
//
// 参数:
//   - path: 文件路径
//
// 返回:
//   - int64: 文件大小（字节）
//   - error: 获取错误，nil表示成功
func (f FileUtils) GetFileSize(path string) (int64, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeFileNotFound, "获取文件信息失败")
	}
	return fileInfo.Size(), nil
}

// SliceUtils 切片操作工具函数集合
// 提供常用的切片操作和查找功能
type SliceUtils struct{}

// Contains 检查字符串切片是否包含指定元素
//
// 参数:
//   - slice: 字符串切片
//   - item: 要查找的元素
//
// 返回:
//   - bool: true表示切片包含该元素
func (s SliceUtils) Contains(slice []string, item string) bool {
	for _, element := range slice {
		if element == item {
			return true
		}
	}
	return false
}

// ContainsInt 检查整数切片是否包含指定元素
//
// 参数:
//   - slice: 整数切片
//   - item: 要查找的元素
//
// 返回:
//   - bool: true表示切片包含该元素
func (s SliceUtils) ContainsInt(slice []int, item int) bool {
	for _, element := range slice {
		if element == item {
			return true
		}
	}
	return false
}

// RemoveDuplicates 去除字符串切片中的重复元素
//
// 参数:
//   - slice: 原始字符串切片
//
// 返回:
//   - []string: 去重后的字符串切片，保持原有顺序
func (s SliceUtils) RemoveDuplicates(slice []string) []string {
	seenElements := make(map[string]bool)
	uniqueElements := make([]string, 0)

	for _, element := range slice {
		if !seenElements[element] {
			seenElements[element] = true
			uniqueElements = append(uniqueElements, element)
		}
	}

	return uniqueElements
}

// FilterEmpty 过滤掉空字符串
//
// 参数:
//   - slice: 原始字符串切片
//
// 返回:
//   - []string: 过滤后的非空字符串切片
func (s SliceUtils) FilterEmpty(slice []string) []string {
	nonEmptyElements := make([]string, 0)
	stringUtils := StringUtils{}
	for _, element := range slice {
		if stringUtils.IsNotEmpty(element) {
			nonEmptyElements = append(nonEmptyElements, element)
		}
	}
	return nonEmptyElements
}

// TimeUtils 时间操作工具函数集合
// 提供时间格式化、超时检测等功能
type TimeUtils struct{}

// Now 获取当前时间
//
// 返回:
//   - time.Time: 当前时间
func (t TimeUtils) Now() time.Time {
	return time.Now()
}

// FormatDuration 格式化时间间隔为人类可读的字符串
//
// 参数:
//   - d: 时间间隔
//
// 返回:
//   - string: 格式化后的时间间隔字符串
//
// 根据时间长度自动选择合适的单位（毫秒、秒、分钟、小时）
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
//
// 参数:
//   - err: 要检查的错误
//
// 返回:
//   - bool: true表示是超时错误
func (t TimeUtils) IsTimeout(err error) bool {
	return os.IsTimeout(err)
}

// WithTimeout 执行带超时的操作
//
// 参数:
//   - timeout: 超时时间
//   - fn: 要执行的函数
//
// 返回:
//   - error: 执行错误或超时错误，nil表示成功
//
// 该函数会在独立的goroutine中执行操作，如果超时则返回超时错误
func (t TimeUtils) WithTimeout(timeout time.Duration, fn func() error) error {
	resultChannel := make(chan error, 1)

	go func() {
		resultChannel <- fn()
	}()

	select {
	case err := <-resultChannel:
		return err
	case <-time.After(timeout):
		return errors.New(errors.ErrCodeTimeout, "操作超时")
	}
}

// ValidationUtils 验证工具函数集合
// 提供常用的参数验证和数据校验功能
type ValidationUtils struct{}

// RequireNonEmpty 要求字符串非空，如果为空则返回错误
//
// 参数:
//   - value: 要验证的字符串值
//   - fieldName: 字段名称，用于错误消息
//
// 返回:
//   - error: 验证错误，nil表示验证通过
//
// 这是ValidateNotEmpty的别名，提供更语义化的函数名
func (v ValidationUtils) RequireNonEmpty(value, fieldName string) error {
	return v.ValidateNotEmpty(value, fieldName)
}

// ValidateNotEmpty 验证字符串非空
//
// 参数:
//   - value: 要验证的字符串值
//   - fieldName: 字段名称，用于错误消息
//
// 返回:
//   - error: 验证错误，nil表示验证通过
func (v ValidationUtils) ValidateNotEmpty(value, fieldName string) error {
	stringUtils := StringUtils{}
	if stringUtils.IsEmpty(value) {
		return errors.InvalidParam(fieldName, "不能为空")
	}
	return nil
}

// ValidatePositive 验证数字为正数
//
// 参数:
//   - value: 要验证的整数值
//   - fieldName: 字段名称，用于错误消息
//
// 返回:
//   - error: 验证错误，nil表示验证通过
func (v ValidationUtils) ValidatePositive(value int, fieldName string) error {
	if value <= 0 {
		return errors.InvalidParam(fieldName, fmt.Sprintf("必须为正数，当前值: %d", value))
	}
	return nil
}

// ValidateInRange 验证数字在指定范围内
//
// 参数:
//   - value: 要验证的整数值
//   - min: 最小值（包含）
//   - max: 最大值（包含）
//   - fieldName: 字段名称，用于错误消息
//
// 返回:
//   - error: 验证错误，nil表示验证通过
func (v ValidationUtils) ValidateInRange(value, min, max int, fieldName string) error {
	if value < min || value > max {
		return errors.InvalidParam(fieldName, fmt.Sprintf("必须在%d-%d范围内，当前值: %d", min, max, value))
	}
	return nil
}

// RetryUtils 重试工具函数集合
// 提供重试机制和退避策略功能
type RetryUtils struct{}

// Retry 重试执行函数
//
// 参数:
//   - attempts: 最大重试次数
//   - delay: 初始延迟时间
//   - fn: 要重试执行的函数
//
// 返回:
//   - error: 最后一次执行的错误，nil表示成功
//
// 该函数使用指数退避策略，每次重试的延迟时间会翻倍
func (r RetryUtils) Retry(attempts int, delay time.Duration, fn func() error) error {
	var lastExecutionError error

	for attemptIndex := 0; attemptIndex < attempts; attemptIndex++ {
		if err := fn(); err != nil {
			lastExecutionError = err
			if attemptIndex < attempts-1 {
				time.Sleep(delay)
				delay *= 2 // 指数退避
			}
			continue
		}
		return nil
	}

	return errors.Wrapf(lastExecutionError, errors.ErrCodeInternal, "重试%d次后仍然失败", attempts)
}

// RetryWithBackoff 带退避策略的重试
//
// 参数:
//   - attempts: 最大重试次数
//   - initialDelay: 初始延迟时间
//   - maxDelay: 最大延迟时间
//   - backoffFactor: 退避因子（每次延迟时间的倍数）
//   - fn: 要重试执行的函数
//
// 返回:
//   - error: 最后一次执行的错误，nil表示成功
//
// 该函数提供更灵活的退避策略配置
func (r RetryUtils) RetryWithBackoff(attempts int, initialDelay, maxDelay time.Duration, backoffFactor float64, fn func() error) error {
	var lastExecutionError error
	currentDelay := initialDelay

	for attemptIndex := 0; attemptIndex < attempts; attemptIndex++ {
		if err := fn(); err != nil {
			lastExecutionError = err
			if attemptIndex < attempts-1 {
				time.Sleep(currentDelay)
				// 计算下次延迟时间，但不超过最大延迟
				nextDelay := time.Duration(float64(currentDelay) * backoffFactor)
				if nextDelay > maxDelay {
					currentDelay = maxDelay
				} else {
					currentDelay = nextDelay
				}
			}
			continue
		}
		return nil
	}

	return errors.Wrapf(lastExecutionError, errors.ErrCodeInternal, "重试%d次后仍然失败", attempts)
}

// ConversionUtils 类型转换工具函数集合
// 提供安全的类型转换和格式化功能
type ConversionUtils struct{}

// StringToInt 安全地将字符串转换为整数
//
// 参数:
//   - str: 要转换的字符串
//   - defaultValue: 转换失败时的默认值
//
// 返回:
//   - int: 转换结果或默认值
func (c ConversionUtils) StringToInt(str string, defaultValue int) int {
	if convertedValue, err := strconv.Atoi(str); err == nil {
		return convertedValue
	}
	return defaultValue
}

// StringToInt64 安全地将字符串转换为64位整数
//
// 参数:
//   - str: 要转换的字符串
//   - defaultValue: 转换失败时的默认值
//
// 返回:
//   - int64: 转换结果或默认值
func (c ConversionUtils) StringToInt64(str string, defaultValue int64) int64 {
	if convertedValue, err := strconv.ParseInt(str, 10, 64); err == nil {
		return convertedValue
	}
	return defaultValue
}

// StringToBool 安全地将字符串转换为布尔值
//
// 参数:
//   - str: 要转换的字符串（支持"true"、"false"、"1"、"0"等）
//   - defaultValue: 转换失败时的默认值
//
// 返回:
//   - bool: 转换结果或默认值
func (c ConversionUtils) StringToBool(str string, defaultValue bool) bool {
	if convertedValue, err := strconv.ParseBool(str); err == nil {
		return convertedValue
	}
	return defaultValue
}

// IntToString 将整数转换为字符串
//
// 参数:
//   - value: 要转换的整数
//
// 返回:
//   - string: 转换后的字符串
func (c ConversionUtils) IntToString(value int) string {
	return strconv.Itoa(value)
}

// Int64ToString 将64位整数转换为字符串
//
// 参数:
//   - value: 要转换的64位整数
//
// 返回:
//   - string: 转换后的字符串
func (c ConversionUtils) Int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}

// BoolToString 将布尔值转换为字符串
//
// 参数:
//   - value: 要转换的布尔值
//
// 返回:
//   - string: 转换后的字符串（"true"或"false"）
func (c ConversionUtils) BoolToString(value bool) string {
	return strconv.FormatBool(value)
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
