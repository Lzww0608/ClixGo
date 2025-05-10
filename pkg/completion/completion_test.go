//go:build !integration
// +build !integration

package completion

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// 为了测试，将文件操作相关函数抽象为变量，便于mock
// 注意: 这些变量已在主文件中声明，此处不要重复声明

// 模拟文件结构，实现WriteString和Close方法
type mockFile struct {
	*bytes.Buffer
}

func (m *mockFile) Close() error {
	return nil
}

func (m *mockFile) WriteString(s string) (n int, err error) {
	return m.Buffer.WriteString(s)
}

// 测试中的mock辅助函数
func patchOSMock(
	home string, mkdirErr, writeErr, readErr error, fileContent string,
) (restore func(), called *bool) {
	calledWrite := false

	// 保存原始函数
	origUserHomeDirFunc := userHomeDirFunc
	origMkdirAllFunc := mkdirAllFunc
	origWriteFileFunc := writeFileFunc
	origReadFileFunc := readFileFunc
	origOpenFileFunc := openFileFunc

	// 替换为mock函数
	userHomeDirFunc = func() (string, error) { return home, nil }
	mkdirAllFunc = func(path string, perm os.FileMode) error { return mkdirErr }
	writeFileFunc = func(name string, data []byte, perm os.FileMode) error {
		calledWrite = true
		return writeErr
	}
	readFileFunc = func(name string) ([]byte, error) {
		if readErr != nil {
			return nil, readErr
		}
		return []byte(fileContent), nil
	}

	openFileFunc = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		// 成功测试中返回nil错误，不真正打开文件
		// 在测试中会返回模拟的文件接口
		return nil, nil
	}

	// 返回恢复函数
	return func() {
		userHomeDirFunc = origUserHomeDirFunc
		mkdirAllFunc = origMkdirAllFunc
		writeFileFunc = origWriteFileFunc
		readFileFunc = origReadFileFunc
		openFileFunc = origOpenFileFunc
	}, &calledWrite
}

// 重构completion.go以便注入mock（需同步修改production代码）

func TestGenerateCompletionScript_Success(t *testing.T) {
	// 在成功情况下，直接mock到不尝试写入bashrc
	restore, called := patchOSMock("/tmp/testhome", nil, nil, nil, "source ~/.bash_completion.d/gocli")
	defer restore()

	cmd := &cobra.Command{Use: "gocli"}
	err := GenerateCompletionScript(cmd)
	if err != nil {
		t.Fatalf("GenerateCompletionScript 应该成功，实际: %v", err)
	}
	if !*called {
		t.Errorf("应该调用WriteFile写入补全脚本")
	}
}

func TestGenerateCompletionScript_UserHomeDirError(t *testing.T) {
	// 保存原始函数
	origFunc := userHomeDirFunc
	userHomeDirFunc = func() (string, error) { return "", errors.New("fail") }
	defer func() { userHomeDirFunc = origFunc }()

	cmd := &cobra.Command{Use: "gocli"}
	err := GenerateCompletionScript(cmd)
	if err == nil || !strings.Contains(err.Error(), "fail") {
		t.Errorf("UserHomeDir失败时应返回错误")
	}
}

func TestGenerateCompletionScript_MkdirAllError(t *testing.T) {
	restore, _ := patchOSMock("/tmp/testhome", errors.New("mkdir fail"), nil, nil, "")
	defer restore()

	cmd := &cobra.Command{Use: "gocli"}
	err := GenerateCompletionScript(cmd)
	if err == nil || !strings.Contains(err.Error(), "mkdir fail") {
		t.Errorf("MkdirAll失败时应返回错误")
	}
}

func TestGenerateCompletionScript_WriteFileError(t *testing.T) {
	restore, _ := patchOSMock("/tmp/testhome", nil, errors.New("write fail"), nil, "")
	defer restore()

	cmd := &cobra.Command{Use: "gocli"}
	err := GenerateCompletionScript(cmd)
	if err == nil || !strings.Contains(err.Error(), "write fail") {
		t.Errorf("WriteFile失败时应返回错误")
	}
}

func TestGenerateCompletionScript_ReadFileError(t *testing.T) {
	restore, _ := patchOSMock("/tmp/testhome", nil, nil, errors.New("read fail"), "")
	defer restore()

	cmd := &cobra.Command{Use: "gocli"}
	err := GenerateCompletionScript(cmd)
	if err == nil || !strings.Contains(err.Error(), "read fail") {
		t.Errorf("ReadFile失败时应返回错误")
	}
}
