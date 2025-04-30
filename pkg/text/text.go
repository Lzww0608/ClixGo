package text

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"gocli/pkg/logger"
)

// SortLines 对文本行进行排序
func SortLines(input string, reverse bool) (string, error) {
	lines := strings.Split(input, "\n")
	if reverse {
		sort.Sort(sort.Reverse(sort.StringSlice(lines)))
	} else {
		sort.Strings(lines)
	}
	return strings.Join(lines, "\n"), nil
}

// CountWords 统计单词数量
func CountWords(input string) (int, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(bufio.ScanWords)
	count := 0
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

// CountLines 统计行数
func CountLines(input string) (int, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))
	count := 0
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

// CountChars 统计字符数
func CountChars(input string) (int, error) {
	return len([]rune(input)), nil
}

// FindPattern 查找匹配模式的行
func FindPattern(input string, pattern string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("无效的正则表达式: %v", err)
	}

	var matches []string
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		if re.MatchString(line) {
			matches = append(matches, line)
		}
	}
	return matches, scanner.Err()
}

// ReplacePattern 替换匹配模式的内容
func ReplacePattern(input string, pattern string, replacement string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("无效的正则表达式: %v", err)
	}
	return re.ReplaceAllString(input, replacement), nil
}

// RemoveDuplicates 移除重复行
func RemoveDuplicates(input string) (string, error) {
	seen := make(map[string]bool)
	var result []string
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		if !seen[line] {
			seen[line] = true
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n"), scanner.Err()
}

// ConvertCase 转换大小写
func ConvertCase(input string, toUpper bool) string {
	if toUpper {
		return strings.ToUpper(input)
	}
	return strings.ToLower(input)
}

// TrimWhitespace 去除空白字符
func TrimWhitespace(input string) string {
	return strings.TrimSpace(input)
}

// ExtractNumbers 提取数字
func ExtractNumbers(input string) ([]string, error) {
	re := regexp.MustCompile(`\d+`)
	return re.FindAllString(input, -1), nil
}

// ExtractEmails 提取电子邮件地址
func ExtractEmails(input string) ([]string, error) {
	re := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	return re.FindAllString(input, -1), nil
}

// ExtractURLs 提取URL
func ExtractURLs(input string) ([]string, error) {
	re := regexp.MustCompile(`https?://[^\s]+`)
	return re.FindAllString(input, -1), nil
}

// FormatJSON 格式化JSON
func FormatJSON(input string) (string, error) {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(input), "", "  ")
	if err != nil {
		return "", fmt.Errorf("格式化JSON失败: %v", err)
	}
	return out.String(), nil
}

// ValidateJSON 验证JSON
func ValidateJSON(input string) error {
	var js interface{}
	return json.Unmarshal([]byte(input), &js)
} 