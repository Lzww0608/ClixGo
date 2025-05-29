/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 终端状态栏的实现
 */

package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// UpdateStatus 更新状态栏内容
func (sb *StatusBar) UpdateStatus(left, center, right string) {
	sb.left = left
	sb.center = center
	sb.right = right
	sb.render()
}

// SetVisible 设置状态栏可见性
func (sb *StatusBar) SetVisible(visible bool) {
	sb.visible = visible
	if visible {
		sb.render()
	} else {
		sb.view.SetText("")
	}
}

// SetStyle 设置状态栏样式
func (sb *StatusBar) SetStyle(style tcell.Style) {
	sb.style = style
	fg, bg, _ := style.Decompose()
	sb.view.SetBackgroundColor(bg)
	sb.view.SetTextColor(fg)
}

// render 渲染状态栏内容
func (sb *StatusBar) render() {
	if !sb.visible {
		return
	}

	// 简化的状态栏渲染，直接拼接三部分内容
	statusText := fmt.Sprintf("%s | %s | %s", sb.left, sb.center, sb.right)

	// 设置文本
	sb.view.SetText(statusText)
}

// GetHeight 获取状态栏高度
func (sb *StatusBar) GetHeight() int {
	if sb.visible {
		return 1
	}
	return 0
}

// Clear 清空状态栏
func (sb *StatusBar) Clear() {
	sb.left = ""
	sb.center = ""
	sb.right = ""
	sb.view.SetText("")
}

// SetLeftText 设置左侧文本
func (sb *StatusBar) SetLeftText(text string) {
	sb.left = text
	sb.render()
}

// SetCenterText 设置中间文本
func (sb *StatusBar) SetCenterText(text string) {
	sb.center = text
	sb.render()
}

// SetRightText 设置右侧文本
func (sb *StatusBar) SetRightText(text string) {
	sb.right = text
	sb.render()
}

// GetView 获取底层的TextView
func (sb *StatusBar) GetView() *tview.TextView {
	return sb.view
}
