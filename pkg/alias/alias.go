package alias

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Alias struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

var aliasFile = filepath.Join(os.Getenv("HOME"), ".clixgo_aliases.json")
var aliases = make(map[string]string)

func InitAliases() error {
	if _, err := os.Stat(aliasFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(aliasFile)
	if err != nil {
		return fmt.Errorf("读取别名文件失败: %v", err)
	}

	var aliasList []Alias
	if err := json.Unmarshal(data, &aliasList); err != nil {
		return fmt.Errorf("解析别名文件失败: %v", err)
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
		return fmt.Errorf("序列化别名失败: %v", err)
	}

	if err := os.WriteFile(aliasFile, data, 0644); err != nil {
		return fmt.Errorf("保存别名文件失败: %v", err)
	}

	return nil
}

func AddAlias(name, command string) error {
	if strings.Contains(name, " ") {
		return fmt.Errorf("别名不能包含空格")
	}
	aliases[name] = command
	return SaveAliases()
}

func RemoveAlias(name string) error {
	if _, exists := aliases[name]; !exists {
		return fmt.Errorf("别名不存在: %s", name)
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
