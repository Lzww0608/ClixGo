#!/bin/bash

if [ $# -eq 0 ]; then
    echo "用法: $0 <包路径>"
    echo "示例: $0 pkg/terminal"
    exit 1
fi

PACKAGE_PATH=$1
PACKAGE_NAME=$(basename "$PACKAGE_PATH")
TEST_FILE="$PACKAGE_PATH/${PACKAGE_NAME}_test.go"

if [ -f "$TEST_FILE" ]; then
    echo "测试文件已存在: $TEST_FILE"
    exit 1
fi

mkdir -p "$PACKAGE_PATH"

cat > "$TEST_FILE" << EOL
package $PACKAGE_NAME

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func Test${PACKAGE_NAME^}Basic(t *testing.T) {
    // TODO: 实现 $PACKAGE_NAME 模块的基础测试
    assert.True(t, true, "基础测试模板")
}

// TODO: 添加更多测试函数
// func Test${PACKAGE_NAME^}Function1(t *testing.T) { ... }
// func Test${PACKAGE_NAME^}Function2(t *testing.T) { ... }
EOL

echo "✅ 创建测试文件模板: $TEST_FILE"
