package main

import (
	"fmt"
	"strings"

	"github.com/yanyiwu/gojieba"
)

func main() {
	// 初始化结巴分词器
	seg := gojieba.NewJieba()
	defer seg.Free()

	// 测试字符串
	texts := []string{
		"这是一个测试",
		"  多  空格  测试  ",
		"第一行\n第二行\n第三行",
	}

	// 打印每个文本的分词结果
	for _, text := range texts {
		fmt.Printf("原文: %q\n", text)

		// 精确模式
		words := seg.Cut(text, true)
		fmt.Printf("精确模式分词结果(%d个): %s\n", len(words), strings.Join(words, "/"))

		// 搜索引擎模式
		words = seg.CutForSearch(text, true)
		fmt.Printf("搜索引擎模式分词结果(%d个): %s\n", len(words), strings.Join(words, "/"))

		// 全模式
		words = seg.CutAll(text)
		fmt.Printf("全模式分词结果(%d个): %s\n", len(words), strings.Join(words, "/"))

		fmt.Println("--------------------")
	}
}
