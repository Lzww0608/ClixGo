package commands

import (
	"strings"
	"testing"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func init() {
	// 防止日志系统panic
	go func() {
		defer func() {
			recover()
		}()
		logger.SetLogPath("test_commands.log")
		logger.InitLogger()
	}()
}

// 测试AWK命令
func TestAWKCommand(t *testing.T) {
	input := "line1 abc\nline2 def\nline3 abc"
	pattern := "{print $2}"

	output, err := AWKCommand(input, pattern)
	assert.NoError(t, err, "AWK命令应该成功执行")
	assert.Equal(t, "abc\ndef\nabc\n", output, "AWK命令应该返回正确结果")

	// 测试错误情况
	_, err = AWKCommand(input, "{invalid")
	assert.Error(t, err, "AWK命令应该返回语法错误")
}

// 测试Grep命令
func TestGrepCommand(t *testing.T) {
	input := "line1 abc\nline2 def\nline3 abc"
	pattern := "abc"

	output, err := GrepCommand(input, pattern)
	assert.NoError(t, err, "Grep命令应该成功执行")
	assert.True(t, strings.Contains(output, "line1 abc"), "Grep命令应该包含匹配行")
	assert.True(t, strings.Contains(output, "line3 abc"), "Grep命令应该包含匹配行")
	assert.False(t, strings.Contains(output, "line2 def"), "Grep命令不应该包含不匹配行")

	// 测试没有匹配的情况
	_, err = GrepCommand(input, "xyz")
	assert.Error(t, err, "没有匹配应该返回错误")
}

// 测试Sed命令
func TestSedCommand(t *testing.T) {
	input := "line1 abc\nline2 def\nline3 abc"
	pattern := "s/abc/xyz/g"

	output, err := SedCommand(input, pattern)
	assert.NoError(t, err, "Sed命令应该成功执行")

	// 使用strings.Contains而不是完全相等，因为不同平台可能有不同的换行符
	assert.Contains(t, output, "line1 xyz", "Sed命令应该替换abc为xyz")
	assert.Contains(t, output, "line2 def", "Sed命令不应该修改不包含abc的行")
	assert.Contains(t, output, "line3 xyz", "Sed命令应该替换abc为xyz")

	// 测试错误情况
	_, err = SedCommand(input, "s/abc")
	assert.Error(t, err, "Sed命令应该返回语法错误")
}

// 测试管道命令
func TestPipeCommands(t *testing.T) {
	// 测试有效的管道命令
	commands := []string{
		"echo 'line1 abc'",
		"sed s/abc/xyz/g",
	}

	output, err := PipeCommands(commands)
	assert.NoError(t, err, "管道命令应该成功执行")
	assert.Contains(t, output, "xyz", "管道命令应该返回正确结果")
	assert.NotContains(t, output, "def", "管道命令不应包含不相关内容")

	// 测试空命令列表
	_, err = PipeCommands([]string{})
	assert.Error(t, err, "空命令列表应该返回错误")
	assert.Contains(t, err.Error(), "没有提供命令", "错误信息不正确")

	// 测试包含空命令
	_, err = PipeCommands([]string{"echo test", ""})
	assert.Error(t, err, "空命令应该返回错误")
	assert.Contains(t, err.Error(), "空命令", "错误信息不正确")

	// 测试无效命令
	_, err = PipeCommands([]string{"echo test", "invalidcmd"})
	assert.Error(t, err, "无效命令应该返回错误")
}
