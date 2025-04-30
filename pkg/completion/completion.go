package completion

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Lzww0608/ClixGo/pkg/alias"
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
    local aliases
    for name := range alias.ListAliases() {
        aliases="$aliases $name"
    }

    COMPREPLY=($(compgen -W "${subcommands} ${aliases}" -- "${cur}"))
}

complete -F _gocli_completion gocli
`

	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// 创建补全脚本目录
	completionDir := filepath.Join(homeDir, ".bash_completion.d")
	if err := os.MkdirAll(completionDir, 0755); err != nil {
		return err
	}

	// 写入补全脚本
	completionFile := filepath.Join(completionDir, "gocli")
	if err := os.WriteFile(completionFile, []byte(bashScript), 0644); err != nil {
		return err
	}

	// 确保.bashrc中包含补全脚本
	bashrc := filepath.Join(homeDir, ".bashrc")
	completionLine := `source ~/.bash_completion.d/gocli`

	// 检查是否已经包含补全脚本
	data, err := os.ReadFile(bashrc)
	if err != nil {
		return err
	}

	if !strings.Contains(string(data), completionLine) {
		// 添加补全脚本到.bashrc
		f, err := os.OpenFile(bashrc, os.O_APPEND|os.O_WRONLY, 0644)
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