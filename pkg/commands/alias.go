/*
* @Author: Lzww0608
* @Date: 2025-06-14 11:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-14 11:00:00
* @Description: 命令别名管理功能 (从pkg/alias迁移)
 */

package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/Lzww0608/ClixGo/pkg/errors"
	"github.com/Lzww0608/ClixGo/pkg/utils"
)

type Alias struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

var aliasFile = filepath.Join(os.Getenv("HOME"), ".clixgo", "aliases.json")
var aliases = make(map[string]string)

func InitAliases() error {
	if err := utils.Files.EnsureDir(filepath.Dir(aliasFile)); err != nil {
		return errors.Wrap(err, errors.ErrCodeFileNotFound, "创建别名配置目录失败")
	}

	if !utils.Files.Exists(aliasFile) {
		return nil
	}

	data, err := os.ReadFile(aliasFile)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeFileNotFound, "读取别名文件失败")
	}

	var aliasList []Alias
	if err := json.Unmarshal(data, &aliasList); err != nil {
		return errors.Wrap(err, errors.ErrCodeConfigInvalid, "解析别名文件失败")
	}

	for _, a := range aliasList {
		aliases[a.Name] = a.Command
	}

	return nil
}

func SaveAliases() error {
	var aliasList []Alias
	for name, command := range aliases {
		aliasList = append(aliasList, Alias{
			Name:    name,
			Command: command,
		})
	}

	data, err := json.MarshalIndent(aliasList, "", "  ")
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "序列化别名失败")
	}

	if err := utils.Files.SafeWriteFile(aliasFile, data, 0644); err != nil {
		return errors.Wrap(err, errors.ErrCodeFileNotFound, "保存别名文件失败")
	}

	return nil
}

func AddAlias(name, command string) error {
	// 参数验证
	if err := utils.Validation.RequireNonEmpty(name, "name"); err != nil {
		return err
	}
	if err := utils.Validation.RequireNonEmpty(command, "command"); err != nil {
		return err
	}

	if strings.Contains(name, " ") {
		return errors.New(errors.ErrCodeInvalidParam, "别名不能包含空格").
			WithDetails("alias name: " + name)
	}

	aliases[name] = command
	return SaveAliases()
}

func RemoveAlias(name string) error {
	// 参数验证
	if err := utils.Validation.RequireNonEmpty(name, "name"); err != nil {
		return err
	}

	if _, exists := aliases[name]; !exists {
		return errors.New(errors.ErrCodeNotFound, "别名不存在").
			WithDetails("alias name: " + name)
	}

	delete(aliases, name)
	return SaveAliases()
}

func GetAlias(name string) (string, bool) {
	command, exists := aliases[name]
	return command, exists
}

func ListAliases() map[string]string {
	return aliases
}

func ExpandCommand(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return command
	}

	if expanded, exists := aliases[parts[0]]; exists {
		return expanded + " " + strings.Join(parts[1:], " ")
	}

	return command
}
