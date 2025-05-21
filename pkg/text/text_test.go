package text

import (
	"testing"
)

func TestSortLines(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		reverse   bool
		expected  string
		expectErr bool
	}{
		{
			name:      "正常排序",
			input:     "c\nb\na",
			reverse:   false,
			expected:  "a\nb\nc",
			expectErr: false,
		},
		{
			name:      "反向排序",
			input:     "a\nb\nc",
			reverse:   true,
			expected:  "c\nb\na",
			expectErr: false,
		},
		{
			name:      "空字符串",
			input:     "",
			reverse:   false,
			expected:  "",
			expectErr: false,
		},
		{
			name:      "单行",
			input:     "single line",
			reverse:   false,
			expected:  "single line",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SortLines(tt.input, tt.reverse)
			if (err != nil) != tt.expectErr {
				t.Errorf("SortLines() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if result != tt.expected {
				t.Errorf("SortLines() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  int
		expectErr bool
	}{
		{
			name:      "正常的文本",
			input:     "这是一个测试",
			expected:  3, // jieba分词结果：这是/一个/测试
			expectErr: false,
		},
		{
			name:      "空的字符串",
			input:     "",
			expected:  0,
			expectErr: false,
		},
		{
			name:      "多空格",
			input:     "  多  空格  测试  ",
			expected:  3, // jieba分词结果：多/空格/测试
			expectErr: false,
		},
		{
			name:      "多行文本",
			input:     "第一行\n第二行\n第三行",
			expected:  3, // jieba分词结果：第一行/第二行/第三行
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := CountWords(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("CountWords() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if count != tt.expected {
				t.Errorf("CountWords() = %d, want %d", count, tt.expected)
			}
		})
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  int
		expectErr bool
	}{
		{
			name:      "多行文本",
			input:     "第一行\n第二行\n第三行",
			expected:  3,
			expectErr: false,
		},
		{
			name:      "空字符串",
			input:     "",
			expected:  0,
			expectErr: false,
		},
		{
			name:      "单行",
			input:     "单行文本",
			expected:  1,
			expectErr: false,
		},
		{
			name:      "末尾有换行符",
			input:     "第一行\n第二行\n",
			expected:  2,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := CountLines(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("CountLines() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if count != tt.expected {
				t.Errorf("CountLines() = %d, want %d", count, tt.expected)
			}
		})
	}
}

func TestCountChars(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  int
		expectErr bool
	}{
		{
			name:      "ASCII字符",
			input:     "abc123",
			expected:  6,
			expectErr: false,
		},
		{
			name:      "中文字符",
			input:     "中文测试",
			expected:  4,
			expectErr: false,
		},
		{
			name:      "混合字符",
			input:     "中文test123",
			expected:  9,
			expectErr: false,
		},
		{
			name:      "空字符串",
			input:     "",
			expected:  0,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := CountChars(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("CountChars() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if count != tt.expected {
				t.Errorf("CountChars() = %d, want %d", count, tt.expected)
			}
		})
	}
}

func TestFindPattern(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		pattern   string
		expected  []string
		expectErr bool
	}{
		{
			name:      "简单匹配",
			input:     "line1\nline2\nline3",
			pattern:   "line2",
			expected:  []string{"line2"},
			expectErr: false,
		},
		{
			name:      "多行匹配",
			input:     "abc123\ndef456\nghi789\nabc789",
			pattern:   "abc",
			expected:  []string{"abc123", "abc789"},
			expectErr: false,
		},
		{
			name:      "正则表达式",
			input:     "test1\ntest2\nfest3",
			pattern:   "[tf]est\\d",
			expected:  []string{"test1", "test2", "fest3"},
			expectErr: false,
		},
		{
			name:      "无匹配",
			input:     "line1\nline2\nline3",
			pattern:   "line4",
			expected:  []string{},
			expectErr: false,
		},
		{
			name:      "无效正则表达式",
			input:     "line1\nline2\nline3",
			pattern:   "[未闭合的括号",
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := FindPattern(tt.input, tt.pattern)
			if (err != nil) != tt.expectErr {
				t.Errorf("FindPattern() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && !compareStringSlices(matches, tt.expected) {
				t.Errorf("FindPattern() = %v, want %v", matches, tt.expected)
			}
		})
	}
}

func TestReplacePattern(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		pattern     string
		replacement string
		expected    string
		expectErr   bool
	}{
		{
			name:        "简单替换",
			input:       "hello world",
			pattern:     "world",
			replacement: "golang",
			expected:    "hello golang",
			expectErr:   false,
		},
		{
			name:        "正则替换",
			input:       "test1 test2 test3",
			pattern:     "test\\d",
			replacement: "replaced",
			expected:    "replaced replaced replaced",
			expectErr:   false,
		},
		{
			name:        "无匹配",
			input:       "hello world",
			pattern:     "notfound",
			replacement: "replaced",
			expected:    "hello world",
			expectErr:   false,
		},
		{
			name:        "无效正则表达式",
			input:       "hello world",
			pattern:     "[未闭合的括号",
			replacement: "replaced",
			expected:    "",
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ReplacePattern(tt.input, tt.pattern, tt.replacement)
			if (err != nil) != tt.expectErr {
				t.Errorf("ReplacePattern() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && result != tt.expected {
				t.Errorf("ReplacePattern() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{
			name:      "有重复",
			input:     "line1\nline2\nline1\nline3\nline2",
			expected:  "line1\nline2\nline3",
			expectErr: false,
		},
		{
			name:      "无重复",
			input:     "line1\nline2\nline3",
			expected:  "line1\nline2\nline3",
			expectErr: false,
		},
		{
			name:      "空字符串",
			input:     "",
			expected:  "",
			expectErr: false,
		},
		{
			name:      "全部重复",
			input:     "line1\nline1\nline1",
			expected:  "line1",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RemoveDuplicates(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("RemoveDuplicates() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && result != tt.expected {
				t.Errorf("RemoveDuplicates() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		toUpper  bool
		expected string
	}{
		{
			name:     "转大写",
			input:    "hello world",
			toUpper:  true,
			expected: "HELLO WORLD",
		},
		{
			name:     "转小写",
			input:    "HELLO WORLD",
			toUpper:  false,
			expected: "hello world",
		},
		{
			name:     "混合大小写转大写",
			input:    "Hello World",
			toUpper:  true,
			expected: "HELLO WORLD",
		},
		{
			name:     "混合大小写转小写",
			input:    "Hello World",
			toUpper:  false,
			expected: "hello world",
		},
		{
			name:     "空字符串",
			input:    "",
			toUpper:  true,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertCase(tt.input, tt.toUpper)
			if result != tt.expected {
				t.Errorf("ConvertCase() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTrimWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "前后有空格",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "只有前导空格",
			input:    "  hello world",
			expected: "hello world",
		},
		{
			name:     "只有后导空格",
			input:    "hello world  ",
			expected: "hello world",
		},
		{
			name:     "无空格",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "只有空格",
			input:    "   ",
			expected: "",
		},
		{
			name:     "空字符串",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TrimWhitespace(tt.input)
			if result != tt.expected {
				t.Errorf("TrimWhitespace() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractNumbers(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  []string
		expectErr bool
	}{
		{
			name:      "包含数字",
			input:     "abc123def456ghi789",
			expected:  []string{"123", "456", "789"},
			expectErr: false,
		},
		{
			name:      "无数字",
			input:     "abcdefghi",
			expected:  []string{},
			expectErr: false,
		},
		{
			name:      "只有数字",
			input:     "123456789",
			expected:  []string{"123456789"},
			expectErr: false,
		},
		{
			name:      "空字符串",
			input:     "",
			expected:  []string{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractNumbers(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("ExtractNumbers() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && !compareStringSlices(result, tt.expected) {
				t.Errorf("ExtractNumbers() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractEmails(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  []string
		expectErr bool
	}{
		{
			name:      "有效邮箱",
			input:     "联系我：user@example.com 或者 admin@test.cn",
			expected:  []string{"user@example.com", "admin@test.cn"},
			expectErr: false,
		},
		{
			name:      "无邮箱",
			input:     "这里没有邮箱地址",
			expected:  []string{},
			expectErr: false,
		},
		{
			name:      "无效邮箱",
			input:     "invalid@email 或 @domain.com",
			expected:  []string{},
			expectErr: false,
		},
		{
			name:      "空字符串",
			input:     "",
			expected:  []string{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractEmails(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("ExtractEmails() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && !compareStringSlices(result, tt.expected) {
				t.Errorf("ExtractEmails() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractURLs(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  []string
		expectErr bool
	}{
		{
			name:      "有效URL",
			input:     "访问 https://example.com 或 http://test.cn/page",
			expected:  []string{"https://example.com", "http://test.cn/page"},
			expectErr: false,
		},
		{
			name:      "无URL",
			input:     "这里没有URL",
			expected:  []string{},
			expectErr: false,
		},
		{
			name:      "URL中有特殊字符",
			input:     "网址 https://example.com/search?q=test&lang=zh",
			expected:  []string{"https://example.com/search?q=test&lang=zh"},
			expectErr: false,
		},
		{
			name:      "空字符串",
			input:     "",
			expected:  []string{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractURLs(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("ExtractURLs() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && !compareStringSlices(result, tt.expected) {
				t.Errorf("ExtractURLs() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{
			name:      "有效JSON",
			input:     `{"name":"测试","age":30}`,
			expected:  "{\n  \"name\": \"测试\",\n  \"age\": 30\n}",
			expectErr: false,
		},
		{
			name:      "已格式化JSON",
			input:     "{\n  \"name\": \"测试\"\n}",
			expected:  "{\n  \"name\": \"测试\"\n}",
			expectErr: false,
		},
		{
			name:      "无效JSON",
			input:     `{"name":"测试",}`,
			expected:  "",
			expectErr: true,
		},
		{
			name:      "空字符串",
			input:     "",
			expected:  "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatJSON(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("FormatJSON() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && result != tt.expected {
				t.Errorf("FormatJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{
			name:      "有效JSON",
			input:     `{"name":"测试","age":30}`,
			expectErr: false,
		},
		{
			name:      "有效JSON数组",
			input:     `[1,2,3,4]`,
			expectErr: false,
		},
		{
			name:      "无效JSON",
			input:     `{"name":"测试",}`,
			expectErr: true,
		},
		{
			name:      "非JSON字符串",
			input:     "hello world",
			expectErr: true,
		},
		{
			name:      "空字符串",
			input:     "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSON(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateJSON() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestGetWords(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		mode             int
		useHMM           bool
		minExpectedCount int
		expectErr        bool
	}{
		{
			name:             "精确模式分词",
			input:            "我来到北京清华大学",
			mode:             ModeDefault,
			useHMM:           true,
			minExpectedCount: 3, // 至少应该分出"我"、"来到"、"北京清华大学"
			expectErr:        false,
		},
		{
			name:             "全模式分词",
			input:            "我来到北京清华大学",
			mode:             ModeAll,
			useHMM:           true,
			minExpectedCount: 5, // 全模式会分出更多词语
			expectErr:        false,
		},
		{
			name:             "搜索引擎模式分词",
			input:            "小明硕士毕业于中国科学院计算所",
			mode:             ModeSearch,
			useHMM:           true,
			minExpectedCount: 8, // 搜索引擎模式会进一步切分
			expectErr:        false,
		},
		{
			name:             "空字符串",
			input:            "",
			mode:             ModeDefault,
			useHMM:           true,
			minExpectedCount: 0,
			expectErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			words, err := GetWords(tt.input, tt.mode, tt.useHMM)
			if (err != nil) != tt.expectErr {
				t.Errorf("GetWords() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if len(words) < tt.minExpectedCount {
				t.Errorf("GetWords() returned %d words, expected at least %d", len(words), tt.minExpectedCount)
				t.Logf("Words: %v", words)
			}
		})
	}

	// 测试自定义词语添加
	customWord := "结巴分词测试"
	AddCustomWord(customWord)
	words, _ := GetWords("这是结巴分词测试示例", ModeDefault, true)
	foundCustomWord := false
	for _, word := range words {
		if word == customWord {
			foundCustomWord = true
			break
		}
	}
	if !foundCustomWord {
		t.Errorf("Custom word '%s' was not found in segmentation result", customWord)
	}
}

// 辅助函数，比较两个字符串切片是否相等
func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
