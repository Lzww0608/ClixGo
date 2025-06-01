/*
* @Author: Lzww0608
* @Date: 2025-6-1 20:50:44
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-1 21:21:40
* @Description: 终端性能基准测试
 */

package benchmarks

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

// TestMain 在测试开始前初始化logger
func TestMain(m *testing.M) {
	// 初始化日志系统
	logger.InitLogger()
	defer logger.Close()

	// 运行测试
	code := m.Run()
	os.Exit(code)
}

// 终端创建性能基准测试
func BenchmarkTerminalCreation(b *testing.B) {
	tempDir := "/tmp/clixgo_bench_" + strconv.Itoa(int(time.Now().Unix()))
	defer os.RemoveAll(tempDir)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 创建会话管理器
		config := &terminal.TerminalConfig{
			BufferSize: 2000,
			ScrollBack: 2000,
		}
		manager := terminal.NewSessionManager(config)

		// 创建会话
		sessionID := fmt.Sprintf("bench_session_%d", i)
		session, err := manager.CreateSession(sessionID)

		if err != nil {
			b.Fatalf("创建会话失败: %v", err)
		}

		// 立即销毁以释放资源
		manager.KillSession(session.ID)
	}
}

// 会话切换性能基准测试
func BenchmarkSessionSwitch(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}
	manager := terminal.NewSessionManager(config)

	// 预创建多个会话
	sessionCount := 10
	sessionIDs := make([]string, sessionCount)

	for i := 0; i < sessionCount; i++ {
		sessionName := fmt.Sprintf("switch_session_%d", i)
		session, err := manager.CreateSession(sessionName)
		if err != nil {
			b.Fatalf("创建会话失败: %v", err)
		}
		sessionIDs[i] = session.ID
	}

	defer func() {
		for _, sessionID := range sessionIDs {
			manager.KillSession(sessionID)
		}
	}()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sessionID := sessionIDs[i%sessionCount]

		// 模拟会话切换
		session, err := manager.GetSession(sessionID)
		if err != nil {
			b.Fatalf("获取会话失败: %v", err)
		}

		// 模拟激活会话
		_ = (session.Status == terminal.SessionActive)
	}
}

// PTY操作性能基准测试
func BenchmarkPTYOperations(b *testing.B) {
	b.Run("会话创建", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		config := &terminal.TerminalConfig{
			BufferSize: 2000,
			ScrollBack: 2000,
		}

		for i := 0; i < b.N; i++ {
			manager := terminal.NewSessionManager(config)
			sessionName := fmt.Sprintf("pty_session_%d", i)
			session, err := manager.CreateSession(sessionName)
			if err != nil {
				b.Fatalf("创建会话失败: %v", err)
			}
			manager.KillSession(session.ID)
		}
	})

	b.Run("窗口操作", func(b *testing.B) {
		config := &terminal.TerminalConfig{
			BufferSize: 2000,
			ScrollBack: 2000,
		}
		manager := terminal.NewSessionManager(config)
		session, err := manager.CreateSession("window_test_session")
		if err != nil {
			b.Fatalf("创建会话失败: %v", err)
		}
		defer manager.KillSession(session.ID)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			windowName := fmt.Sprintf("window_%d", i)
			window, err := manager.CreateWindow(session.ID, windowName)
			if err != nil {
				b.Fatalf("创建窗口失败: %v", err)
			}

			// 立即关闭窗口
			err = manager.CloseWindow(session.ID, window.Index)
			if err != nil {
				// 忽略关闭错误，可能是最后一个窗口
			}
		}
	})
}

// 并发会话管理基准测试
func BenchmarkConcurrentSessionManagement(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	// 测试不同的并发级别
	concurrencyLevels := []int{1, 5, 10, 20}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("并发数_%d", concurrency), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				manager := terminal.NewSessionManager(config)
				var wg sync.WaitGroup
				errors := make(chan error, concurrency)

				for j := 0; j < concurrency; j++ {
					wg.Add(1)
					go func(id int) {
						defer wg.Done()

						sessionName := fmt.Sprintf("concurrent_session_%d_%d", i, id)

						session, err := manager.CreateSession(sessionName)
						if err != nil {
							errors <- err
							return
						}

						// 短暂操作后销毁会话
						time.Sleep(1 * time.Millisecond)
						manager.KillSession(session.ID)

						errors <- nil
					}(j)
				}

				wg.Wait()

				// 检查错误
				for j := 0; j < concurrency; j++ {
					if err := <-errors; err != nil {
						b.Fatalf("并发操作失败: %v", err)
					}
				}
			}
		})
	}
}

// 内存分配模式基准测试
func BenchmarkTerminalMemoryAllocation(b *testing.B) {
	b.Run("缓冲区分配", func(b *testing.B) {
		bufferSizes := []int{1024, 4096, 16384, 65536}

		for _, size := range bufferSizes {
			b.Run(fmt.Sprintf("缓冲区大小_%d", size), func(b *testing.B) {
				config := &terminal.TerminalConfig{
					BufferSize: size,
					ScrollBack: size,
				}

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					manager := terminal.NewSessionManager(config)
					sessionName := fmt.Sprintf("mem_session_%d", i)
					session, err := manager.CreateSession(sessionName)
					if err != nil {
						b.Fatalf("创建会话失败: %v", err)
					}
					manager.KillSession(session.ID)
				}
			})
		}
	})
}

// 会话持久化基准测试
func BenchmarkSessionPersistence(b *testing.B) {
	config := &terminal.TerminalConfig{
		BufferSize: 2000,
		ScrollBack: 2000,
	}

	b.Run("会话保存", func(b *testing.B) {
		manager := terminal.NewSessionManager(config)

		// 创建测试会话
		session, err := manager.CreateSession("persistence_test_session")
		if err != nil {
			b.Fatalf("创建会话失败: %v", err)
		}
		defer manager.KillSession(session.ID)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// 模拟会话状态保存
			err := manager.SaveSession(session.ID, fmt.Sprintf("/tmp/session_save_%d.json", i))
			if err != nil {
				b.Fatalf("保存会话失败: %v", err)
			}
		}
	})

	b.Run("会话加载", func(b *testing.B) {
		manager := terminal.NewSessionManager(config)

		// 预先保存一个会话
		session, err := manager.CreateSession("load_test_session")
		if err != nil {
			b.Fatalf("创建会话失败: %v", err)
		}

		saveFile := "/tmp/session_load_test.json"
		err = manager.SaveSession(session.ID, saveFile)
		if err != nil {
			b.Fatalf("保存会话失败: %v", err)
		}
		defer os.Remove(saveFile)

		manager.KillSession(session.ID)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := manager.LoadSession(saveFile)
			if err != nil {
				b.Fatalf("加载会话失败: %v", err)
			}
		}
	})
}

// 启动时间基准测试
func BenchmarkStartupTime(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		config := &terminal.TerminalConfig{
			BufferSize: 2000,
			ScrollBack: 2000,
		}

		// 测量从创建管理器到第一个会话可用的时间
		start := time.Now()

		manager := terminal.NewSessionManager(config)
		session, err := manager.CreateSession("startup_test_session")
		if err != nil {
			b.Fatalf("创建会话失败: %v", err)
		}

		elapsed := time.Since(start)

		// 记录启动时间（可选）
		if elapsed > 100*time.Millisecond {
			b.Logf("启动时间较长: %v", elapsed)
		}

		manager.KillSession(session.ID)
	}
}
