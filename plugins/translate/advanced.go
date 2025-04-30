package translate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/text/language"
)

// TranslationMemory 表示翻译记忆
type TranslationMemory struct {
	Source     string    `json:"source"`
	Target     string    `json:"target"`
	SourceLang string    `json:"source_lang"`
	TargetLang string    `json:"target_lang"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	UseCount   int       `json:"use_count"`
}

// TranslationMemoryDB 表示翻译记忆数据库
type TranslationMemoryDB struct {
	sync.RWMutex
	Entries map[string]*TranslationMemory
	Path    string
}

// NewTranslationMemoryDB 创建翻译记忆数据库
func NewTranslationMemoryDB(path string) (*TranslationMemoryDB, error) {
	db := &TranslationMemoryDB{
		Entries: make(map[string]*TranslationMemory),
		Path:    path,
	}
	
	if err := db.load(); err != nil {
		return nil, err
	}
	
	return db, nil
}

// load 加载翻译记忆
func (db *TranslationMemoryDB) load() error {
	if _, err := os.Stat(db.Path); os.IsNotExist(err) {
		return nil
	}
	
	data, err := os.ReadFile(db.Path)
	if err != nil {
		return errors.Wrap(err, "读取翻译记忆文件失败")
	}
	
	if err := json.Unmarshal(data, &db.Entries); err != nil {
		return errors.Wrap(err, "解析翻译记忆数据失败")
	}
	
	return nil
}

// save 保存翻译记忆
func (db *TranslationMemoryDB) save() error {
	data, err := json.MarshalIndent(db.Entries, "", "  ")
	if err != nil {
		return errors.Wrap(err, "序列化翻译记忆数据失败")
	}
	
	if err := os.MkdirAll(filepath.Dir(db.Path), 0755); err != nil {
		return errors.Wrap(err, "创建翻译记忆目录失败")
	}
	
	if err := os.WriteFile(db.Path, data, 0644); err != nil {
		return errors.Wrap(err, "写入翻译记忆文件失败")
	}
	
	return nil
}

// add 添加翻译记忆
func (db *TranslationMemoryDB) add(source, target, sourceLang, targetLang string) error {
	db.Lock()
	defer db.Unlock()
	
	key := fmt.Sprintf("%s|%s|%s", source, sourceLang, targetLang)
	now := time.Now()
	
	if entry, ok := db.Entries[key]; ok {
		entry.Target = target
		entry.UpdatedAt = now
		entry.UseCount++
	} else {
		db.Entries[key] = &TranslationMemory{
			Source:     source,
			Target:     target,
			SourceLang: sourceLang,
			TargetLang: targetLang,
			CreatedAt:  now,
			UpdatedAt:  now,
			UseCount:   1,
		}
	}
	
	return db.save()
}

// get 获取翻译记忆
func (db *TranslationMemoryDB) get(source, sourceLang, targetLang string) *TranslationMemory {
	db.RLock()
	defer db.RUnlock()
	
	key := fmt.Sprintf("%s|%s|%s", source, sourceLang, targetLang)
	if entry, ok := db.Entries[key]; ok {
		return entry
	}
	return nil
}

// BatchTranslationResult 表示批量翻译结果
type BatchTranslationResult struct {
	Original  string
	Translated string
	Error     error
}

// BatchTranslateOptions 表示批量翻译选项
type BatchTranslateOptions struct {
	SourceLang      string
	TargetLang      string
	Concurrency     int
	UseMemory       bool
	CollectMetrics  bool
	ProgressFunc    func(completed, total int)
}

// BatchTranslate 批量翻译文本
func (s *TranslationService) BatchTranslate(ctx context.Context, texts []string, opts BatchTranslateOptions) []BatchTranslationResult {
	results := make([]BatchTranslationResult, len(texts))
	
	if opts.Concurrency <= 0 {
		opts.Concurrency = s.config.MaxConcurrency
	}
	
	var wg sync.WaitGroup
	sem := make(chan struct{}, opts.Concurrency)
	
	for i, text := range texts {
		wg.Add(1)
		go func(i int, text string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			
			var result BatchTranslationResult
			result.Original = text
			
			// 检查翻译记忆
			if opts.UseMemory {
				if entry := s.memory.get(text, opts.SourceLang, opts.TargetLang); entry != nil {
					result.Translated = entry.Target
					results[i] = result
					if opts.ProgressFunc != nil {
						opts.ProgressFunc(i+1, len(texts))
					}
					return
				}
			}
			
			// 执行翻译
			translated, err := s.translateWithRetry(ctx, text, opts.SourceLang, opts.TargetLang)
			if err != nil {
				result.Error = err
			} else {
				result.Translated = translated.Text
				// 保存到翻译记忆
				if opts.UseMemory {
					_ = s.memory.add(text, translated.Text, opts.SourceLang, opts.TargetLang)
				}
			}
			
			results[i] = result
			if opts.ProgressFunc != nil {
				opts.ProgressFunc(i+1, len(texts))
			}
		}(i, text)
	}
	
	wg.Wait()
	return results
}

// FormatOptions 表示格式化选项
type FormatOptions struct {
	PreserveHTML     bool
	PreserveMarkdown bool
	PreserveTags     []string
}

// formatText 格式化文本
func formatText(text string, opts FormatOptions) string {
	if opts.PreserveHTML {
		// 保护 HTML 标签
		text = protectHTMLTags(text)
	}
	
	if opts.PreserveMarkdown {
		// 保护 Markdown 语法
		text = protectMarkdownSyntax(text)
	}
	
	if len(opts.PreserveTags) > 0 {
		// 保护自定义标签
		text = protectCustomTags(text, opts.PreserveTags)
	}
	
	return text
}

// protectHTMLTags 保护 HTML 标签
func protectHTMLTags(text string) string {
	// 实现 HTML 标签保护逻辑
	return text
}

// protectMarkdownSyntax 保护 Markdown 语法
func protectMarkdownSyntax(text string) string {
	// 实现 Markdown 语法保护逻辑
	return text
}

// protectCustomTags 保护自定义标签
func protectCustomTags(text string, tags []string) string {
	// 实现自定义标签保护逻辑
	return text
}

// LanguageDetectionOptions 表示语言检测选项
type LanguageDetectionOptions struct {
	MinConfidence float64
	FastMode     bool
}

// detectLanguageAdvanced 高级语言检测
func (s *TranslationService) detectLanguageAdvanced(ctx context.Context, text string, opts LanguageDetectionOptions) ([]LanguageDetection, error) {
	if opts.FastMode {
		// 使用本地语言检测
		return detectLanguageLocally(text)
	}
	
	// 使用 API 检测
	result, err := s.detectLanguage(text)
	if err != nil {
		return nil, err
	}
	
	if result.Score < opts.MinConfidence {
		// 置信度不足，尝试其他方法
		localResults, _ := detectLanguageLocally(text)
		return combineDetectionResults([]LanguageDetection{*result}, localResults), nil
	}
	
	return []LanguageDetection{*result}, nil
}

// detectLanguageLocally 本地语言检测
func detectLanguageLocally(text string) ([]LanguageDetection, error) {
	var results []LanguageDetection
	
	// 使用语言标签匹配
	tags := language.NewMatcher([]language.Tag{
		language.English,
		language.Chinese,
		language.Japanese,
		language.Korean,
		language.French,
		language.German,
		language.Spanish,
	})
	
	t, _, confidence := tags.Match(language.Make(text))
	results = append(results, LanguageDetection{
		Language: t.String(),
		Score:    float64(confidence) / 100.0,
	})
	
	return results, nil
}

// combineDetectionResults 合并检测结果
func combineDetectionResults(results1, results2 []LanguageDetection) []LanguageDetection {
	// 实现结果合并逻辑
	return append(results1, results2...)
}

// TranslateStreamOptions 表示流式翻译选项
type TranslateStreamOptions struct {
	SourceLang  string
	TargetLang  string
	BufferSize  int
	ChunkSize   int
	Format      FormatOptions
}

// TranslateStream 流式翻译
func (s *TranslationService) TranslateStream(ctx context.Context, r io.Reader, w io.Writer, opts TranslateStreamOptions) error {
	if opts.BufferSize <= 0 {
		opts.BufferSize = 4096
	}
	if opts.ChunkSize <= 0 {
		opts.ChunkSize = s.config.ChunkSize
	}
	
	buffer := make([]byte, opts.BufferSize)
	var text strings.Builder
	
	for {
		n, err := r.Read(buffer)
		if err != nil && err != io.EOF {
			return errors.Wrap(err, "读取输入失败")
		}
		
		if n > 0 {
			text.Write(buffer[:n])
			
			// 检查是否达到分块大小
			if text.Len() >= opts.ChunkSize {
				if err := s.translateAndWriteChunk(ctx, &text, w, opts); err != nil {
					return err
				}
			}
		}
		
		if err == io.EOF {
			break
		}
	}
	
	// 处理剩余文本
	if text.Len() > 0 {
		if err := s.translateAndWriteChunk(ctx, &text, w, opts); err != nil {
			return err
		}
	}
	
	return nil
}

// translateAndWriteChunk 翻译并写入数据块
func (s *TranslationService) translateAndWriteChunk(ctx context.Context, text *strings.Builder, w io.Writer, opts TranslateStreamOptions) error {
	// 格式化文本
	formattedText := formatText(text.String(), opts.Format)
	
	// 翻译文本
	result, err := s.translateWithRetry(ctx, formattedText, opts.SourceLang, opts.TargetLang)
	if err != nil {
		return err
	}
	
	// 写入结果
	if _, err := w.Write([]byte(result.Text)); err != nil {
		return errors.Wrap(err, "写入翻译结果失败")
	}
	
	// 清空缓冲区
	text.Reset()
	return nil
} 