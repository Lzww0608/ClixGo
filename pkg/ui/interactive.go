package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/schollz/progressbar/v3"
)

// InteractiveUI 提供交互式命令行界面
type InteractiveUI struct {
	history     []string
	historyFile string
}

// NewInteractiveUI 创建新的交互式界面
func NewInteractiveUI(historyFile string) *InteractiveUI {
	return &InteractiveUI{
		history:     make([]string, 0),
		historyFile: historyFile,
	}
}

// Prompt 显示提示并获取用户输入
func (ui *InteractiveUI) Prompt(message string) (string, error) {
	var answer string
	prompt := &survey.Input{
		Message: message,
	}
	err := survey.AskOne(prompt, &answer)
	return answer, err
}

// Select 显示选项列表并获取用户选择
func (ui *InteractiveUI) Select(message string, options []string) (string, error) {
	var answer string
	prompt := &survey.Select{
		Message: message,
		Options: options,
	}
	err := survey.AskOne(prompt, &answer)
	return answer, err
}

// MultiSelect 显示多选列表并获取用户选择
func (ui *InteractiveUI) MultiSelect(message string, options []string) ([]string, error) {
	var answers []string
	prompt := &survey.MultiSelect{
		Message: message,
		Options: options,
	}
	err := survey.AskOne(prompt, &answers)
	return answers, err
}

// Confirm 显示确认提示
func (ui *InteractiveUI) Confirm(message string) (bool, error) {
	var answer bool
	prompt := &survey.Confirm{
		Message: message,
	}
	err := survey.AskOne(prompt, &answer)
	return answer, err
}

// ShowProgress 显示进度条
func (ui *InteractiveUI) ShowProgress(total int64, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions64(
		total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(10),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)
}

// ShowTable 显示表格
func (ui *InteractiveUI) ShowTable(headers []string, rows [][]string) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleLight)

	// 设置表头
	headerRow := make([]interface{}, len(headers))
	for i, h := range headers {
		headerRow[i] = h
	}
	t.AppendHeader(headerRow)

	// 添加数据行
	for _, row := range rows {
		rowData := make([]interface{}, len(row))
		for i, cell := range row {
			rowData[i] = cell
		}
		t.AppendRow(rowData)
	}

	t.Render()
}

// ShowSuccess 显示成功消息
func (ui *InteractiveUI) ShowSuccess(message string) {
	color.Green("✓ %s", message)
}

// ShowError 显示错误消息
func (ui *InteractiveUI) ShowError(message string) {
	color.Red("✗ %s", message)
}

// ShowWarning 显示警告消息
func (ui *InteractiveUI) ShowWarning(message string) {
	color.Yellow("⚠ %s", message)
}

// ShowInfo 显示信息消息
func (ui *InteractiveUI) ShowInfo(message string) {
	color.Blue("ℹ %s", message)
}

// ShowDebug 显示调试消息
func (ui *InteractiveUI) ShowDebug(message string) {
	color.Cyan("⚡ %s", message)
}

// AddToHistory 添加命令到历史记录
func (ui *InteractiveUI) AddToHistory(command string) {
	ui.history = append(ui.history, command)
	if err := ui.saveHistory(); err != nil {
		ui.ShowError(fmt.Sprintf("保存历史记录失败: %v", err))
	}
}

// GetHistory 获取历史记录
func (ui *InteractiveUI) GetHistory() []string {
	return ui.history
}

// ClearHistory 清除历史记录
func (ui *InteractiveUI) ClearHistory() {
	ui.history = make([]string, 0)
	if err := ui.saveHistory(); err != nil {
		ui.ShowError(fmt.Sprintf("清除历史记录失败: %v", err))
	}
}

// saveHistory 保存历史记录到文件
func (ui *InteractiveUI) saveHistory() error {
	if ui.historyFile == "" {
		return nil
	}

	data := strings.Join(ui.history, "\n")
	return os.WriteFile(ui.historyFile, []byte(data), 0644)
}

// loadHistory 从文件加载历史记录
func (ui *InteractiveUI) loadHistory() error {
	if ui.historyFile == "" {
		return nil
	}

	data, err := os.ReadFile(ui.historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			ui.history = make([]string, 0) // 文件不存在，历史记录视为空
			return nil
		}
		return err
	}

	if len(data) == 0 {
		ui.history = make([]string, 0) // 空文件，则历史记录为空
		return nil
	}

	// 移除末尾的换行符，然后按换行符分割
	// 这样可以避免因文件末尾换行符导致产生额外的空历史条目
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")

	// 如果分割后只有一个空字符串元素（例如，文件内容为 "\n"），则历史记录视为空
	if len(lines) == 1 && lines[0] == "" {
		ui.history = make([]string, 0)
	} else {
		ui.history = lines
	}
	return nil
}
