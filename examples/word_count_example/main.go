// 中文分词示例程序
package main

import (
	"fmt"
	"strings"

	"github.com/Lzww0608/ClixGo/pkg/text"
)

func main() {
	// 在程序结束时释放jieba资源
	defer text.FreeJieba()

	// 测试文本
	texts := []string{
		"这是一个中文分词的示例程序",
		"结巴分词是一个非常优秀的中文分词库",
		"我们可以使用不同的分词模式来处理文本",
		"第一行\n第二行\n第三行",
	}

	fmt.Println("======= 中文分词示例 =======")

	for _, t := range texts {
		fmt.Printf("\n原文: %s\n", t)

		// 统计单词数
		count, _ := text.CountWords(t)
		fmt.Printf("单词数: %d\n", count)

		// 获取默认模式（精确模式）分词结果
		words, _ := text.GetWords(t, text.ModeDefault, true)
		fmt.Printf("精确模式分词: %s\n", strings.Join(words, "/"))

		// 获取搜索引擎模式分词结果
		words, _ = text.GetWords(t, text.ModeSearch, true)
		fmt.Printf("搜索引擎模式分词: %s\n", strings.Join(words, "/"))

		// 获取全模式分词结果
		words, _ = text.GetWords(t, text.ModeAll, true)
		fmt.Printf("全模式分词: %s\n", strings.Join(words, "/"))
	}

	// 添加自定义词汇
	customWord := "中文分词示例"
	text.AddCustomWord(customWord)

	fmt.Println("\n\n======= 添加自定义词汇后 =======")

	// 再次分词测试
	testText := "这是一个中文分词示例程序"
	words, _ := text.GetWords(testText, text.ModeDefault, true)
	fmt.Printf("\n原文: %s\n", testText)
	fmt.Printf("精确模式分词: %s\n", strings.Join(words, "/"))
}
