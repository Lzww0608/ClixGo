/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-03 19:20:00
* @Description: 交互式用户界面的核心实现
 */

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

// InteractiveUI 交互式命令行界面管理器
// 提供用户输入、选择、确认、进度显示、表格展示和历史记录功能
type InteractiveUI struct {
	history     []string // 命令历史记录
	historyFile string   // 历史记录文件路径
}

// NewInteractiveUI 创建新的交互式界面实例
//
// 参数:
//   - historyFile: 历史记录文件路径，为空则不保存历史记录
//
// 返回:
//   - *InteractiveUI: 新创建的交互式界面实例
func NewInteractiveUI(historyFile string) *InteractiveUI {
	return &InteractiveUI{
		history:     make([]string, 0),
		historyFile: historyFile,
	}
}

// Prompt 显示提示信息并获取用户输入
//
// 参数:
//   - message: 提示信息文本
//
// 返回:
//   - string: 用户输入的文本
//   - error: 输入过程中的错误，nil表示成功
func (ui *InteractiveUI) Prompt(message string) (string, error) {
	var userInput string
	inputPrompt := &survey.Input{
		Message: message,
	}
	err := survey.AskOne(inputPrompt, &userInput)
	return userInput, err
}

// Select 显示单选列表并获取用户选择
//
// 参数:
//   - message: 提示信息文本
//   - options: 可选项列表
//
// 返回:
//   - string: 用户选择的选项
//   - error: 选择过程中的错误，nil表示成功
func (ui *InteractiveUI) Select(message string, options []string) (string, error) {
	var selectedOption string
	selectPrompt := &survey.Select{
		Message: message,
		Options: options,
	}
	err := survey.AskOne(selectPrompt, &selectedOption)
	return selectedOption, err
}

// MultiSelect 显示多选列表并获取用户选择
//
// 参数:
//   - message: 提示信息文本
//   - options: 可选项列表
//
// 返回:
//   - []string: 用户选择的选项列表
//   - error: 选择过程中的错误，nil表示成功
func (ui *InteractiveUI) MultiSelect(message string, options []string) ([]string, error) {
	var selectedOptions []string
	multiSelectPrompt := &survey.MultiSelect{
		Message: message,
		Options: options,
	}
	err := survey.AskOne(multiSelectPrompt, &selectedOptions)
	return selectedOptions, err
}

// Confirm 显示确认提示并获取用户确认
//
// 参数:
//   - message: 确认提示信息
//
// 返回:
//   - bool: true表示确认，false表示取消
//   - error: 确认过程中的错误，nil表示成功
func (ui *InteractiveUI) Confirm(message string) (bool, error) {
	var confirmed bool
	confirmPrompt := &survey.Confirm{
		Message: message,
	}
	err := survey.AskOne(confirmPrompt, &confirmed)
	return confirmed, err
}

// ShowProgress 创建并显示进度条
//
// 参数:
//   - total: 总进度值
//   - description: 进度条描述文本
//
// 返回:
//   - *progressbar.ProgressBar: 进度条实例，可用于更新进度
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

// ShowTable 格式化显示表格数据
//
// 参数:
//   - headers: 表头列表
//   - rows: 表格数据行，每行是字符串数组
//
// 该函数会在标准输出显示美化的表格
func (ui *InteractiveUI) ShowTable(headers []string, rows [][]string) {
	tableRenderer := table.NewWriter()
	tableRenderer.SetOutputMirror(os.Stdout)
	tableRenderer.SetStyle(table.StyleLight)

	// 设置表头
	headerRowData := make([]interface{}, len(headers))
	for index, headerText := range headers {
		headerRowData[index] = headerText
	}
	tableRenderer.AppendHeader(headerRowData)

	// 添加数据行
	for _, dataRow := range rows {
		rowData := make([]interface{}, len(dataRow))
		for index, cellContent := range dataRow {
			rowData[index] = cellContent
		}
		tableRenderer.AppendRow(rowData)
	}

	tableRenderer.Render()
}

// ShowSuccess 显示成功消息（绿色）
//
// 参数:
//   - message: 成功消息文本
func (ui *InteractiveUI) ShowSuccess(message string) {
	color.Green("✓ %s", message)
}

// ShowError 显示错误消息（红色）
//
// 参数:
//   - message: 错误消息文本
func (ui *InteractiveUI) ShowError(message string) {
	color.Red("✗ %s", message)
}

// ShowWarning 显示警告消息（黄色）
//
// 参数:
//   - message: 警告消息文本
func (ui *InteractiveUI) ShowWarning(message string) {
	color.Yellow("⚠ %s", message)
}

// ShowInfo 显示信息消息（蓝色）
//
// 参数:
//   - message: 信息消息文本
func (ui *InteractiveUI) ShowInfo(message string) {
	color.Blue("ℹ %s", message)
}

// ShowDebug 显示调试消息（青色）
//
// 参数:
//   - message: 调试消息文本
func (ui *InteractiveUI) ShowDebug(message string) {
	color.Cyan("⚡ %s", message)
}

// AddToHistory 添加命令到历史记录
//
// 参数:
//   - command: 要添加的命令文本
//
// 命令会被添加到内存历史记录并保存到文件（如果配置了历史文件）
func (ui *InteractiveUI) AddToHistory(command string) {
	ui.history = append(ui.history, command)
	if err := ui.saveHistory(); err != nil {
		ui.ShowError(fmt.Sprintf("保存历史记录失败: %v", err))
	}
}

// GetHistory 获取完整的历史记录列表
//
// 返回:
//   - []string: 历史记录命令列表
func (ui *InteractiveUI) GetHistory() []string {
	return ui.history
}

// ClearHistory 清除所有历史记录
// 会清除内存中的历史记录并更新历史文件
func (ui *InteractiveUI) ClearHistory() {
	ui.history = make([]string, 0)
	if err := ui.saveHistory(); err != nil {
		ui.ShowError(fmt.Sprintf("清除历史记录失败: %v", err))
	}
}

// saveHistory 将历史记录保存到文件
//
// 返回:
//   - error: 保存错误，nil表示成功
//
// 如果未配置历史文件路径则直接返回成功
func (ui *InteractiveUI) saveHistory() error {
	if ui.historyFile == "" {
		return nil
	}

	historyData := strings.Join(ui.history, "\n")
	return os.WriteFile(ui.historyFile, []byte(historyData), 0644)
}

// loadHistory 从文件加载历史记录
//
// 返回:
//   - error: 加载错误，nil表示成功
//
// 如果文件不存在或为空，历史记录将被初始化为空列表
func (ui *InteractiveUI) loadHistory() error {
	if ui.historyFile == "" {
		return nil
	}

	fileData, err := os.ReadFile(ui.historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			ui.history = make([]string, 0) // 文件不存在，历史记录视为空
			return nil
		}
		return err
	}

	if len(fileData) == 0 {
		ui.history = make([]string, 0) // 空文件，则历史记录为空
		return nil
	}

	// 移除末尾的换行符，然后按换行符分割
	// 这样可以避免因文件末尾换行符导致产生额外的空历史条目
	historyContent := strings.TrimSuffix(string(fileData), "\n")

	// 如果去除换行符后内容为空，则历史记录为空
	if historyContent == "" {
		ui.history = make([]string, 0)
		return nil
	}

	ui.history = strings.Split(historyContent, "\n")

	return nil
}
