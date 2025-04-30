package translate

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 模拟翻译 API 服务器
func setupMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"data": {
				"translations": [
					{
						"translatedText": "你好，世界",
						"detectedSourceLanguage": "en"
					}
				]
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
}

// 创建测试服务实例
func setupTestService() (*TranslationService, func()) {
	server := setupMockServer()
	
	config := &Config{
		APIKey:         "test-key",
		DefaultSource:  "auto",
		DefaultTarget:  "zh",
		CacheEnabled:   true,
		CacheDuration:  time.Hour,
		MaxConcurrency: 2,
		Timeout:        time.Second * 5,
		ChunkSize:      100,
		RateLimit:      10.0,
		MaxRetries:     3,
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	service := &TranslationService{
		config:     config,
		cache:      &TranslationCache{items: make(map[string]*TranslationResult)},
		ctx:        ctx,
		cancel:     cancel,
		httpClient: server.Client(),
	}
	
	cleanup := func() {
		server.Close()
		cancel()
	}
	
	return service, cleanup
}

// TestTranslateText 测试文本翻译功能
func TestTranslateText(t *testing.T) {
	service, cleanup := setupTestService()
	defer cleanup()
	
	tests := []struct {
		name       string
		text       string
		source     string
		target     string
		wantErr    bool
		wantResult string
	}{
		{
			name:       "正常翻译",
			text:       "Hello, World",
			source:     "en",
			target:     "zh",
			wantErr:    false,
			wantResult: "你好，世界",
		},
		{
			name:       "空文本",
			text:       "",
			source:     "en",
			target:     "zh",
			wantErr:    true,
			wantResult: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.translateWithRetry(context.Background(), tt.text, tt.source, tt.target)
			
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			assert.Equal(t, tt.wantResult, result.Text)
		})
	}
}

// TestTranslationCache 测试缓存功能
func TestTranslationCache(t *testing.T) {
	service, cleanup := setupTestService()
	defer cleanup()
	
	// 首次翻译
	result1, err := service.translateWithRetry(context.Background(), "Hello", "en", "zh")
	require.NoError(t, err)
	
	// 从缓存获取
	result2, err := service.translateWithRetry(context.Background(), "Hello", "en", "zh")
	require.NoError(t, err)
	
	assert.Equal(t, result1.Text, result2.Text)
	assert.Equal(t, result1.SourceLang, result2.SourceLang)
}

// TestConcurrentTranslation 测试并发翻译
func TestConcurrentTranslation(t *testing.T) {
	service, cleanup := setupTestService()
	defer cleanup()
	
	const concurrency = 10
	done := make(chan bool, concurrency)
	
	for i := 0; i < concurrency; i++ {
		go func(i int) {
			text := fmt.Sprintf("Hello %d", i)
			_, err := service.translateWithRetry(context.Background(), text, "en", "zh")
			assert.NoError(t, err)
			done <- true
		}(i)
	}
	
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

// TestLargeFileTranslation 测试大文件翻译
func TestLargeFileTranslation(t *testing.T) {
	service, cleanup := setupTestService()
	defer cleanup()
	
	// 创建临时测试文件
	tmpfile, err := os.CreateTemp("", "test*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	
	// 写入测试数据
	content := "Hello\nWorld\n" + strings.Repeat("Test\n", 100)
	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	tmpfile.Close()
	
	err = service.translateLargeFile(context.Background(), tmpfile.Name(), "en", "zh")
	assert.NoError(t, err)
	
	// 检查输出文件是否存在
	outputPath := tmpfile.Name() + "_translated.txt"
	defer os.Remove(outputPath)
	
	_, err = os.Stat(outputPath)
	assert.NoError(t, err)
}

// BenchmarkTranslateText 基准测试：文本翻译
func BenchmarkTranslateText(b *testing.B) {
	service, cleanup := setupTestService()
	defer cleanup()
	
	ctx := context.Background()
	text := "Hello, World"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.translateWithRetry(ctx, text, "en", "zh")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTranslateWithCache 基准测试：带缓存的翻译
func BenchmarkTranslateWithCache(b *testing.B) {
	service, cleanup := setupTestService()
	defer cleanup()
	
	ctx := context.Background()
	text := "Hello, World"
	
	// 预热缓存
	_, err := service.translateWithRetry(ctx, text, "en", "zh")
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.translateWithRetry(ctx, text, "en", "zh")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConcurrentTranslation 基准测试：并发翻译
func BenchmarkConcurrentTranslation(b *testing.B) {
	service, cleanup := setupTestService()
	defer cleanup()
	
	ctx := context.Background()
	text := "Hello, World"
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := service.translateWithRetry(ctx, text, "en", "zh")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
} 