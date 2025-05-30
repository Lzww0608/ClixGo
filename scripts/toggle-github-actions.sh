#!/bin/bash

# GitHub Actions 工作流切换脚本

WORKFLOW_FILE=".github/workflows/test.yml"
DISABLED_FILE=".github/workflows/test.yml.disabled"

# 检查当前状态
if [ -f "$WORKFLOW_FILE" ]; then
    echo "🟢 GitHub Actions 当前状态: 启用"
    echo "📋 工作流将在推送到 main/develop 分支时运行完整质量检查"
    echo ""
    read -p "是否要禁用 GitHub Actions? (y/N): " choice
    if [[ $choice =~ ^[Yy]$ ]]; then
        mv "$WORKFLOW_FILE" "$DISABLED_FILE"
        echo "🔴 已禁用 GitHub Actions 工作流"
        echo "💡 文件已重命名为: test.yml.disabled"
    else
        echo "✅ 保持 GitHub Actions 启用状态"
    fi
elif [ -f "$DISABLED_FILE" ]; then
    echo "🔴 GitHub Actions 当前状态: 禁用"
    echo "📋 不会执行自动化质量检查"
    echo ""
    read -p "是否要启用 GitHub Actions? (y/N): " choice
    if [[ $choice =~ ^[Yy]$ ]]; then
        mv "$DISABLED_FILE" "$WORKFLOW_FILE"
        echo "🟢 已启用 GitHub Actions 工作流"
        echo "💡 下次推送时将执行完整质量检查"
    else
        echo "✅ 保持 GitHub Actions 禁用状态"
    fi
else
    echo "❌ 错误: 找不到工作流文件"
    echo "🔍 请检查 .github/workflows/ 目录"
    exit 1
fi

echo ""
echo "📊 当前状态总结:"
if [ -f "$WORKFLOW_FILE" ]; then
    echo "  - GitHub Actions: 🟢 启用"
    echo "  - 本地预提交检查: 🟢 启用 (快速模式)"
    echo "  - 推送时触发: 🟢 是"
else
    echo "  - GitHub Actions: 🔴 禁用"  
    echo "  - 本地预提交检查: 🟢 启用 (快速模式)"
    echo "  - 推送时触发: 🔴 否"
fi 