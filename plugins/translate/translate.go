package translate

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// 自定义错误类型
var (
	ErrInvalidConfig = errors.New("无效的配置")
	ErrAPILimit      = errors.New("API 调用限制")
	ErrTranslation   = errors.New("翻译错误")
	ErrFileIO        = errors.New("文件操作错误")
)

// Config 表示翻译插件配置
type Config struct {
	APIKey         string        `mapstructure:"api_key"`
	DefaultSource  string        `mapstructure:"default_source"`
	DefaultTarget  string        `mapstructure:"default_target"`
	CacheEnabled   bool          `mapstructure:"cache_enabled"`
	CacheDuration  time.Duration `mapstructure:"cache_duration"`
	MaxConcurrency int           `mapstructure:"max_concurrency"`
	Timeout        time.Duration `mapstructure:"timeout"`
	ChunkSize      int           `mapstructure:"chunk_size"`
	RateLimit      float64       `mapstructure:"rate_limit"`
	MaxRetries     int           `mapstructure:"max_retries"`
}

// TranslationResult 表示翻译结果
type TranslationResult struct {
	Text       string    `json:"text"`
	SourceLang string    `json:"source_lang"`
	TargetLang string    `json:"target_lang"`
	Timestamp  time.Time `json:"timestamp"`
	RetryCount int       `json:"-"`
}

// LanguageDetection 表示语言检测结果
type LanguageDetection struct {
	Language string  `json:"language"`
	Score    float64 `json:"score"`
}

// TranslationCache 表示翻译缓存
type TranslationCache struct {
	mu    sync.RWMutex
	items map[string]*TranslationResult
}

// TranslationService 表示翻译服务
type TranslationService struct {
	config     *Config
	cache      *TranslationCache
	memory     *TranslationMemoryDB
	limiter    *rate.Limiter
	httpClient *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
}

var (
	service *TranslationService
)

func init() {
	// 初始化配置
	viper.SetConfigName("translate")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.gocli")
	viper.AddConfigPath(".")

	// 设置默认值
	viper.SetDefault("api_key", os.Getenv("TRANSLATE_API_KEY"))
	viper.SetDefault("default_source", "auto")
	viper.SetDefault("default_target", "zh")
	viper.SetDefault("cache_enabled", true)
	viper.SetDefault("cache_duration", 24*time.Hour)
	viper.SetDefault("max_concurrency", runtime.NumCPU())
	viper.SetDefault("timeout", 30*time.Second)
	viper.SetDefault("chunk_size", 1000)
	viper.SetDefault("rate_limit", 10.0)
	viper.SetDefault("max_retries", 3)

	var config Config
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Printf("读取配置文件错误: %+v\n", errors.WithStack(err))
		}
	}

	if err := viper.Unmarshal(&config); err != nil {
		fmt.Printf("解析配置错误: %+v\n", errors.WithStack(err))
	}

	// 初始化翻译记忆
	memoryPath := filepath.Join(os.Getenv("HOME"), ".gocli", "translate_memory.json")
	memory, err := NewTranslationMemoryDB(memoryPath)
	if err != nil {
		fmt.Printf("初始化翻译记忆失败: %+v\n", errors.WithStack(err))
	}

	// 创建翻译服务
	ctx, cancel := context.WithCancel(context.Background())
	service = &TranslationService{
		config:     &config,
		cache:      &TranslationCache{items: make(map[string]*TranslationResult)},
		memory:     memory,
		limiter:    rate.NewLimiter(rate.Limit(config.RateLimit), 1),
		httpClient: &http.Client{Timeout: config.Timeout},
		ctx:        ctx,
		cancel:     cancel,
	}

	// 启动缓存清理
	if config.CacheEnabled {
		go service.cleanCache()
	}
}

// cleanCache 定期清理过期缓存
func (s *TranslationService) cleanCache() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			s.cache.mu.Lock()
			for key, item := range s.cache.items {
				if now.Sub(item.Timestamp) > s.config.CacheDuration {
					delete(s.cache.items, key)
				}
			}
			s.cache.mu.Unlock()
		case <-s.ctx.Done():
			return
		}
	}
}

// translateWithRetry 带重试的翻译
func (s *TranslationService) translateWithRetry(ctx context.Context, text, sourceLang, targetLang string) (*TranslationResult, error) {
	start := time.Now()
	s.recordActiveRequest(1)
	defer s.recordActiveRequest(-1)

	var lastErr error
	for i := 0; i < s.config.MaxRetries; i++ {
		select {
		case <-ctx.Done():
			s.recordMetrics(start, "text", sourceLang, targetLang, ctx.Err(), 0)
			return nil, ctx.Err()
		default:
		}

		if err := s.limiter.Wait(ctx); err != nil {
			s.recordMetrics(start, "text", sourceLang, targetLang, err, 0)
			return nil, errors.Wrap(ErrAPILimit, err.Error())
		}

		result, err := s.translate(ctx, text, sourceLang, targetLang)
		if err == nil {
			s.recordMetrics(start, "text", sourceLang, targetLang, nil, len(text))
			return result, nil
		}

		lastErr = err
		backoff := time.Duration(1<<uint(i)) * time.Second
		time.Sleep(backoff)
	}

	s.recordMetrics(start, "text", sourceLang, targetLang, lastErr, 0)
	return nil, errors.Wrap(ErrTranslation, lastErr.Error())
}

// translate 执行翻译
func (s *TranslationService) translate(ctx context.Context, text, sourceLang, targetLang string) (*TranslationResult, error) {
	baseURL := "https://translation.googleapis.com/language/translate/v2"

	params := url.Values{}
	params.Add("key", s.config.APIKey)
	params.Add("q", text)
	params.Add("source", sourceLang)
	params.Add("target", targetLang)

	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "创建请求失败")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "发送请求失败")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("API请求失败: %s", resp.Status)
	}

	var result struct {
		Data struct {
			Translations []struct {
				TranslatedText         string `json:"translatedText"`
				DetectedSourceLanguage string `json:"detectedSourceLanguage"`
			} `json:"translations"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "解析响应失败")
	}

	if len(result.Data.Translations) == 0 {
		return nil, errors.New("未找到翻译结果")
	}

	return &TranslationResult{
		Text:       result.Data.Translations[0].TranslatedText,
		SourceLang: result.Data.Translations[0].DetectedSourceLanguage,
		TargetLang: targetLang,
		Timestamp:  time.Now(),
	}, nil
}

// translateLargeFile 处理大文件翻译
func (s *TranslationService) translateLargeFile(ctx context.Context, filePath, sourceLang, targetLang string) error {
	start := time.Now()
	s.recordActiveRequest(1)
	defer s.recordActiveRequest(-1)

	file, err := os.Open(filePath)
	if err != nil {
		s.recordMetrics(start, "file", sourceLang, targetLang, err, 0)
		return errors.Wrap(ErrFileIO, fmt.Sprintf("打开文件失败: %v", err))
	}
	defer file.Close()

	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		s.recordMetrics(start, "file", sourceLang, targetLang, err, 0)
		return errors.Wrap(ErrFileIO, fmt.Sprintf("获取文件信息失败: %v", err))
	}

	outputPath := fmt.Sprintf("%s_translated.txt", filePath)
	output, err := os.Create(outputPath)
	if err != nil {
		s.recordMetrics(start, "file", sourceLang, targetLang, err, 0)
		return errors.Wrap(ErrFileIO, fmt.Sprintf("创建输出文件失败: %v", err))
	}
	defer output.Close()

	scanner := bufio.NewScanner(file)
	writer := bufio.NewWriter(output)

	var buffer strings.Builder
	lineCount := 0
	bytesProcessed := 0

	for scanner.Scan() {
		line := scanner.Text()
		buffer.WriteString(line)
		buffer.WriteString("\n")
		lineCount++

		if lineCount >= s.config.ChunkSize {
			if err := s.translateAndWrite(ctx, buffer.String(), sourceLang, targetLang, writer); err != nil {
				s.recordMetrics(start, "file", sourceLang, targetLang, err, bytesProcessed)
				return err
			}
			bytesProcessed += buffer.Len()
			buffer.Reset()
			lineCount = 0
		}
	}

	if buffer.Len() > 0 {
		if err := s.translateAndWrite(ctx, buffer.String(), sourceLang, targetLang, writer); err != nil {
			s.recordMetrics(start, "file", sourceLang, targetLang, err, bytesProcessed)
			return err
		}
		bytesProcessed += buffer.Len()
	}

	if err := writer.Flush(); err != nil {
		s.recordMetrics(start, "file", sourceLang, targetLang, err, bytesProcessed)
		return errors.Wrap(ErrFileIO, fmt.Sprintf("刷新输出缓冲失败: %v", err))
	}

	s.recordMetrics(start, "file", sourceLang, targetLang, nil, bytesProcessed)
	return nil
}

// translateAndWrite 翻译并写入结果
func (s *TranslationService) translateAndWrite(ctx context.Context, text, sourceLang, targetLang string, writer *bufio.Writer) error {
	result, err := s.translateWithRetry(ctx, text, sourceLang, targetLang)
	if err != nil {
		return err
	}

	if _, err := writer.WriteString(result.Text + "\n"); err != nil {
		return errors.Wrap(ErrFileIO, fmt.Sprintf("写入翻译结果失败: %v", err))
	}

	return nil
}

// translateDirectory 翻译目录中的所有文件
func (s *TranslationService) translateDirectory(ctx context.Context, dirPath, sourceLang, targetLang string) error {
	start := time.Now()
	s.recordActiveRequest(1)
	defer s.recordActiveRequest(-1)

	g, ctx := errgroup.WithContext(ctx)
	tasks := make(chan string, s.config.MaxConcurrency)

	// 启动工作协程
	for i := 0; i < s.config.MaxConcurrency; i++ {
		s.recordWorkerCount(1)
		g.Go(func() error {
			defer s.recordWorkerCount(-1)
			for path := range tasks {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					if err := s.translateLargeFile(ctx, path, sourceLang, targetLang); err != nil {
						return errors.Wrapf(err, "处理文件 %s 失败", path)
					}
				}
			}
			return nil
		})
	}

	// 遍历目录并分发任务
	var bytesProcessed int64
	g.Go(func() error {
		defer close(tasks)
		return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && strings.HasSuffix(info.Name(), ".txt") {
				bytesProcessed += info.Size()
				select {
				case <-ctx.Done():
					return ctx.Err()
				case tasks <- path:
				}
			}
			return nil
		})
	})

	err := g.Wait()
	s.recordMetrics(start, "directory", sourceLang, targetLang, err, int(bytesProcessed))
	return err
}

// translateText 翻译文本
func translateText(text, sourceLang, targetLang string) (*TranslationResult, error) {
	// 检查缓存
	cacheKey := getCacheKey(text, sourceLang, targetLang)
	if result, ok := getFromCache(cacheKey); ok {
		return result, nil
	}

	baseURL := "https://translation.googleapis.com/language/translate/v2"

	params := url.Values{}
	params.Add("key", config.APIKey)
	params.Add("q", text)
	params.Add("source", sourceLang)
	params.Add("target", targetLang)

	url := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	client := &http.Client{
		Timeout: config.Timeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var result struct {
		Data struct {
			Translations []struct {
				TranslatedText         string `json:"translatedText"`
				DetectedSourceLanguage string `json:"detectedSourceLanguage"`
			} `json:"translations"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if len(result.Data.Translations) == 0 {
		return nil, fmt.Errorf("未找到翻译结果")
	}

	translation := TranslationResult{
		Text:       result.Data.Translations[0].TranslatedText,
		SourceLang: result.Data.Translations[0].DetectedSourceLanguage,
		TargetLang: targetLang,
		Timestamp:  time.Now(),
	}

	// 保存到缓存
	saveToCache(cacheKey, translation)

	return &translation, nil
}

// detectLanguage 检测文本语言
func (s *TranslationService) detectLanguage(text string) (*LanguageDetection, error) {
	baseURL := "https://translation.googleapis.com/language/translate/v2/detect"

	params := url.Values{}
	params.Add("key", s.config.APIKey)
	params.Add("q", text)

	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	client := &http.Client{
		Timeout: s.config.Timeout,
	}

	resp, err := client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var result struct {
		Data struct {
			Detections [][]struct {
				Language string  `json:"language"`
				Score    float64 `json:"confidence"`
			} `json:"detections"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if len(result.Data.Detections) == 0 || len(result.Data.Detections[0]) == 0 {
		return nil, fmt.Errorf("未检测到语言")
	}

	return &LanguageDetection{
		Language: result.Data.Detections[0][0].Language,
		Score:    result.Data.Detections[0][0].Score,
	}, nil
}

// translateFile 翻译文件内容
func translateFile(filePath, sourceLang, targetLang string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %v", err)
	}

	result, err := translateText(string(content), sourceLang, targetLang)
	if err != nil {
		return err
	}

	outputPath := fmt.Sprintf("%s_translated.txt", filePath)
	if err := os.WriteFile(outputPath, []byte(result.Text), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	fmt.Printf("翻译完成，结果已保存到: %s\n", outputPath)
	return nil
}

// translateDirectory 翻译目录中的所有文件
func translateDirectory(dirPath, sourceLang, targetLang string) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.MaxConcurrency)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".txt") {
			wg.Add(1)
			go func() {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				if err := translateFile(path, sourceLang, targetLang); err != nil {
					fmt.Printf("翻译文件 %s 失败: %v\n", path, err)
				}
			}()
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("遍历目录失败: %v", err)
	}

	wg.Wait()
	return nil
}

// InitCommands 初始化翻译命令
func InitCommands() *cobra.Command {
	translateCmd := &cobra.Command{
		Use:   "translate",
		Short: "文本翻译工具",
		Long:  "提供文本翻译、文件翻译和语言检测功能",
	}

	// 翻译文本命令
	textCmd := &cobra.Command{
		Use:   "text [text]",
		Short: "翻译指定文本",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sourceLang, _ := cmd.Flags().GetString("source")
			targetLang, _ := cmd.Flags().GetString("target")

			result, err := translateText(args[0], sourceLang, targetLang)
			if err != nil {
				fmt.Printf("翻译失败: %v\n", err)
				return
			}

			fmt.Printf("原文 (%s): %s\n", result.SourceLang, args[0])
			fmt.Printf("译文 (%s): %s\n", result.TargetLang, result.Text)
		},
	}
	textCmd.Flags().StringP("source", "s", config.DefaultSource, "源语言代码")
	textCmd.Flags().StringP("target", "t", config.DefaultTarget, "目标语言代码")

	// 翻译文件命令
	fileCmd := &cobra.Command{
		Use:   "file [file]",
		Short: "翻译文件内容",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sourceLang, _ := cmd.Flags().GetString("source")
			targetLang, _ := cmd.Flags().GetString("target")

			if err := translateFile(args[0], sourceLang, targetLang); err != nil {
				fmt.Printf("翻译失败: %v\n", err)
			}
		},
	}
	fileCmd.Flags().StringP("source", "s", config.DefaultSource, "源语言代码")
	fileCmd.Flags().StringP("target", "t", config.DefaultTarget, "目标语言代码")

	// 翻译目录命令
	dirCmd := &cobra.Command{
		Use:   "dir [directory]",
		Short: "翻译目录中的所有文件",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sourceLang, _ := cmd.Flags().GetString("source")
			targetLang, _ := cmd.Flags().GetString("target")

			if err := translateDirectory(args[0], sourceLang, targetLang); err != nil {
				fmt.Printf("翻译失败: %v\n", err)
			}
		},
	}
	dirCmd.Flags().StringP("source", "s", config.DefaultSource, "源语言代码")
	dirCmd.Flags().StringP("target", "t", config.DefaultTarget, "目标语言代码")

	// 语言检测命令
	detectCmd := &cobra.Command{
		Use:   "detect [text]",
		Short: "检测文本语言",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			result, err := detectLanguage(args[0])
			if err != nil {
				fmt.Printf("检测失败: %v\n", err)
				return
			}

			fmt.Printf("检测到语言: %s (置信度: %.2f)\n", result.Language, result.Score)
		},
	}

	// 批量翻译命令
	batchCmd := &cobra.Command{
		Use:   "batch [file]",
		Short: "批量翻译文本文件",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sourceLang, _ := cmd.Flags().GetString("source")
			targetLang, _ := cmd.Flags().GetString("target")
			useMemory, _ := cmd.Flags().GetBool("use-memory")
			concurrency, _ := cmd.Flags().GetInt("concurrency")

			// 读取输入文件
			content, err := os.ReadFile(args[0])
			if err != nil {
				fmt.Printf("读取文件失败: %v\n", err)
				return
			}

			texts := strings.Split(string(content), "\n")

			// 设置进度回调
			progressChan := make(chan int, len(texts))
			go func() {
				completed := 0
				for range progressChan {
					completed++
					fmt.Printf("\r翻译进度: %d/%d", completed, len(texts))
				}
				fmt.Println()
			}()

			// 执行批量翻译
			results := service.BatchTranslate(cmd.Context(), texts, BatchTranslateOptions{
				SourceLang:     sourceLang,
				TargetLang:     targetLang,
				UseMemory:      useMemory,
				Concurrency:    concurrency,
				CollectMetrics: true,
				ProgressFunc: func(completed, total int) {
					progressChan <- completed
				},
			})

			close(progressChan)

			// 写入结果
			outputPath := args[0] + "_translated.txt"
			var output strings.Builder
			errorCount := 0

			for _, result := range results {
				if result.Error != nil {
					errorCount++
					output.WriteString(fmt.Sprintf("ERROR: %v\n", result.Error))
				} else {
					output.WriteString(result.Translated + "\n")
				}
			}

			if err := os.WriteFile(outputPath, []byte(output.String()), 0644); err != nil {
				fmt.Printf("写入结果失败: %v\n", err)
				return
			}

			fmt.Printf("翻译完成，成功: %d，失败: %d，结果已保存到: %s\n",
				len(texts)-errorCount, errorCount, outputPath)
		},
	}

	batchCmd.Flags().StringP("source", "s", config.DefaultSource, "源语言代码")
	batchCmd.Flags().StringP("target", "t", config.DefaultTarget, "目标语言代码")
	batchCmd.Flags().Bool("use-memory", true, "使用翻译记忆")
	batchCmd.Flags().IntP("concurrency", "c", config.MaxConcurrency, "并发数")

	// 流式翻译命令
	streamCmd := &cobra.Command{
		Use:   "stream [file]",
		Short: "流式翻译大文件",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sourceLang, _ := cmd.Flags().GetString("source")
			targetLang, _ := cmd.Flags().GetString("target")
			bufferSize, _ := cmd.Flags().GetInt("buffer-size")
			chunkSize, _ := cmd.Flags().GetInt("chunk-size")
			preserveHTML, _ := cmd.Flags().GetBool("preserve-html")
			preserveMarkdown, _ := cmd.Flags().GetBool("preserve-markdown")

			// 打开输入文件
			input, err := os.Open(args[0])
			if err != nil {
				fmt.Printf("打开文件失败: %v\n", err)
				return
			}
			defer input.Close()

			// 创建输出文件
			outputPath := args[0] + "_translated.txt"
			output, err := os.Create(outputPath)
			if err != nil {
				fmt.Printf("创建输出文件失败: %v\n", err)
				return
			}
			defer output.Close()

			// 执行流式翻译
			err = service.TranslateStream(cmd.Context(), input, output, TranslateStreamOptions{
				SourceLang: sourceLang,
				TargetLang: targetLang,
				BufferSize: bufferSize,
				ChunkSize:  chunkSize,
				Format: FormatOptions{
					PreserveHTML:     preserveHTML,
					PreserveMarkdown: preserveMarkdown,
				},
			})

			if err != nil {
				fmt.Printf("翻译失败: %v\n", err)
				return
			}

			fmt.Printf("翻译完成，结果已保存到: %s\n", outputPath)
		},
	}

	streamCmd.Flags().StringP("source", "s", config.DefaultSource, "源语言代码")
	streamCmd.Flags().StringP("target", "t", config.DefaultTarget, "目标语言代码")
	streamCmd.Flags().Int("buffer-size", 4096, "缓冲区大小")
	streamCmd.Flags().Int("chunk-size", config.ChunkSize, "分块大小")
	streamCmd.Flags().Bool("preserve-html", false, "保留HTML标签")
	streamCmd.Flags().Bool("preserve-markdown", false, "保留Markdown语法")

	// 高级语言检测命令
	detectAdvancedCmd := &cobra.Command{
		Use:   "detect-advanced [text]",
		Short: "高级语言检测",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			minConfidence, _ := cmd.Flags().GetFloat64("min-confidence")
			fastMode, _ := cmd.Flags().GetBool("fast")

			results, err := service.detectLanguageAdvanced(cmd.Context(), args[0], LanguageDetectionOptions{
				MinConfidence: minConfidence,
				FastMode:      fastMode,
			})

			if err != nil {
				fmt.Printf("检测失败: %v\n", err)
				return
			}

			for i, result := range results {
				fmt.Printf("检测结果 #%d:\n", i+1)
				fmt.Printf("  语言: %s\n", result.Language)
				fmt.Printf("  置信度: %.2f\n", result.Score)
			}
		},
	}

	detectAdvancedCmd.Flags().Float64("min-confidence", 0.5, "最小置信度")
	detectAdvancedCmd.Flags().Bool("fast", false, "快速模式")

	translateCmd.AddCommand(textCmd, fileCmd, dirCmd, detectCmd, batchCmd, streamCmd, detectAdvancedCmd)
	return translateCmd
}
