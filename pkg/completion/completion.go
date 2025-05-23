package completion

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// 包级可替换的文件操作函数，便于测试mock
var (
	userHomeDirFunc = os.UserHomeDir
	mkdirAllFunc    = os.MkdirAll
	writeFileFunc   = os.WriteFile
	readFileFunc    = os.ReadFile
	openFileFunc    = os.OpenFile
)

func GenerateCompletionScript(cmd *cobra.Command) error {
	// 生成bash补全脚本
	bashScript := `#!/bin/bash

_gocli_completion() {
    local cur prev words cword
    _init_completion || return

    case "${prev}" in
        alias)
            COMPREPLY=($(compgen -W "add remove list" -- "${cur}"))
            return
            ;;
        history)
            COMPREPLY=($(compgen -W "list clear show" -- "${cur}"))
            return
            ;;
        show)
            # 这里可以添加历史记录索引的补全
            return
            ;;
    esac

    # 获取所有子命令
    local subcommands
    subcommands=$(gocli --help 2>&1 | awk '/^  [a-z]/ {print $1}')
    
    # 添加别名到补全列表
    local aliases=""
    for name in $(gocli alias list 2>/dev/null | awk '{print $1}'); do
        aliases="$aliases $name"
    done

    COMPREPLY=($(compgen -W "${subcommands} ${aliases}" -- "${cur}"))
}

complete -F _gocli_completion gocli
`

	// 获取用户主目录
	homeDir, err := userHomeDirFunc()
	if err != nil {
		return err
	}

	// 创建补全脚本目录
	completionDir := filepath.Join(homeDir, ".bash_completion.d")
	if err := mkdirAllFunc(completionDir, 0755); err != nil {
		return err
	}

	// 写入补全脚本
	completionFile := filepath.Join(completionDir, "gocli")
	if err := writeFileFunc(completionFile, []byte(bashScript), 0644); err != nil {
		return err
	}

	// 确保.bashrc中包含补全脚本
	bashrc := filepath.Join(homeDir, ".bashrc")
	completionLine := `source ~/.bash_completion.d/gocli`

	// 检查是否已经包含补全脚本
	data, err := readFileFunc(bashrc)
	if err != nil {
		return err
	}

	if !strings.Contains(string(data), completionLine) {
		// 添加补全脚本到.bashrc
		f, err := openFileFunc(bashrc, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := f.WriteString("\n" + completionLine + "\n"); err != nil {
			return err
		}
	}

	return nil
}
