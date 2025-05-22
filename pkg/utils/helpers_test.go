package utils

import (
	"reflect"
	"testing"
)

func TestSplitCommands(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "空字符串",
			input:    "",
			expected: []string{""},
		},
		{
			name:     "单个命令",
			input:    "ls",
			expected: []string{"ls"},
		},
		{
			name:     "多个命令",
			input:    "ls;cd;pwd",
			expected: []string{"ls", "cd", "pwd"},
		},
		{
			name:     "命令前后有空格",
			input:    " ls ; cd ; pwd ",
			expected: []string{"ls", "cd", "pwd"},
		},
		{
			name:     "命令中间有多个空格",
			input:    "ls   -la;cd   /home",
			expected: []string{"ls   -la", "cd   /home"},
		},
		{
			name:     "包含空命令",
			input:    "ls;;pwd",
			expected: []string{"ls", "", "pwd"},
		},
		{
			name:     "以分号结尾",
			input:    "ls;cd;",
			expected: []string{"ls", "cd", ""},
		},
		{
			name:     "以分号开头",
			input:    ";ls;cd",
			expected: []string{"", "ls", "cd"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := SplitCommands(tc.input)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("期望 %v，得到 %v", tc.expected, result)
			}
		})
	}
}

func TestValidateCommands(t *testing.T) {
	testCases := []struct {
		name        string
		commands    []string
		expectError bool
	}{
		{
			name:        "有效命令",
			commands:    []string{"ls", "cd", "pwd"},
			expectError: false,
		},
		{
			name:        "空数组",
			commands:    []string{},
			expectError: false,
		},
		{
			name:        "包含空命令",
			commands:    []string{"ls", "", "pwd"},
			expectError: true,
		},
		{
			name:        "包含只有空格的命令",
			commands:    []string{"ls", "  ", "pwd"},
			expectError: true,
		},
		{
			name:        "只有空命令",
			commands:    []string{""},
			expectError: true,
		},
		{
			name:        "只有空格命令",
			commands:    []string{"  "},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCommands(tc.commands)
			if tc.expectError && err == nil {
				t.Errorf("期望出现错误，但没有得到错误")
			}
			if !tc.expectError && err != nil {
				t.Errorf("期望不出现错误，但得到错误: %v", err)
			}
		})
	}
}
