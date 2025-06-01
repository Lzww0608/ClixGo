/*
* @Author: Lzww0608
* @Date: 2025-6-1 20:50:44
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-1 20:50:44
* @Description: 终端性能基准测试
 */

package benchmarks

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/terminal"
)

// 终端创建基准测试
func BenchmarkTerminalCreation(b *testing.B) {
	tempDir := "/tmp/clixgo_bench_" + strconv.Itoa(int(time.Now().Unix()))
	defer os.RemoveAll(tempDir)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 创建会话管理器
		sessionDir := filepath.Join(tempDir, fmt.Sprintf("session_%d", i))
		os.MkdirAll(sessionDir, 0755)

		manager := terminal.NewSessionManager(sessionDir)

		// 创建会话
		sessionID := fmt.Sprintf("bench_session_%d", i)
		session, err := manager.CreateSession(sessionID, &terminal.SessionConfig{
			Command: []string{"/bin/echo", "benchmark"},
			WorkDir: "/tmp",
		})

		if err != nil {
			b.Fatalf("创建会话失败: %v", err)
		}

		// 立即关闭以释放资源
		session.Close()
		manager.DestroySession(sessionID)
	}
}

// 会话切换性能基准测试
func BenchmarkSessionSwitch(b *testing.B) {
	tempDir := "/tmp/clixgo_bench_switch_" + strconv.Itoa(int(time.Now().Unix()))
	defer os.RemoveAll(tempDir)

	manager := terminal.NewSessionManager(tempDir)

	// 预创建多个会话
	sessionCount := 10
	sessionIDs := make([]string, sessionCount)

	for i := 0; i < sessionCount; i++ {
		sessionID := fmt.Sprintf("switch_session_%d", i)
		sessionIDs[i] = sessionID

		_, err := manager.CreateSession(sessionID, &terminal.SessionConfig{
			Command: []string{"/bin/sleep", "60"},
			WorkDir: "/tmp",
		})
		if err != nil {
			b.Fatalf("创建会话失败: %v", err)
		}
	}

	defer func() {
		for _, sessionID := range sessionIDs {
			manager.DestroySession(sessionID)
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
		_ = session.IsActive()
	}
}

// PTY操作性能基准测试
func BenchmarkPTYOperations(b *testing.B) {
	b.Run("PTY创建", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			pty, err := terminal.NewPTY(&terminal.PTYConfig{
				Command: []string{"/bin/echo", "test"},
				WorkDir: "/tmp",
			})
			if err != nil {
				b.Fatalf("创建PTY失败: %v", err)
			}
			pty.Close()
		}
	})

	b.Run("PTY写入", func(b *testing.B) {
		pty, err := terminal.NewPTY(&terminal.PTYConfig{
			Command: []string{"/bin/cat"},
			WorkDir: "/tmp",
		})
		if err != nil {
			b.Fatalf("创建PTY失败: %v", err)
		}
		defer pty.Close()

		testData := []byte("benchmark test data\n")

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := pty.Write(testData)
			if err != nil {
				b.Fatalf("PTY写入失败: %v", err)
			}
		}
	})

	b.Run("PTY读取", func(b *testing.B) {
		pty, err := terminal.NewPTY(&terminal.PTYConfig{
			Command: []string{"/bin/sh", "-c", "while true; do echo 'test data'; sleep 0.001; done"},
			WorkDir: "/tmp",
		})
		if err != nil {
			b.Fatalf("创建PTY失败: %v", err)
		}
		defer pty.Close()

		buffer := make([]byte, 1024)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := pty.Read(buffer)
			if err != nil {
				b.Fatalf("PTY读取失败: %v", err)
			}
		}
	})
}

// 并发会话管理基准测试
func BenchmarkConcurrentSessionManagement(b *testing.B) {
	tempDir := "/tmp/clixgo_bench_concurrent_" + strconv.Itoa(int(time.Now().Unix()))
	defer os.RemoveAll(tempDir)

	manager := terminal.NewSessionManager(tempDir)

	// 测试不同的并发级别
	concurrencyLevels := []int{1, 5, 10, 20, 50}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("并发数_%d", concurrency), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				errors := make(chan error, concurrency)

				for j := 0; j < concurrency; j++ {
					wg.Add(1)
					go func(id int) {
						defer wg.Done()

						sessionID := fmt.Sprintf("concurrent_session_%d_%d", i, id)

						session, err := manager.CreateSession(sessionID, &terminal.SessionConfig{
							Command: []string{"/bin/echo", "concurrent_test"},
							WorkDir: "/tmp",
						})

						if err != nil {
							errors <- err
							return
						}

						// 短暂操作后销毁会话
						time.Sleep(1 * time.Millisecond)
						session.Close()
						manager.DestroySession(sessionID)

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
			b.Run(fmt.Sprintf("缓冲区_%dB", size), func(b *testing.B) {
				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					buffer := make([]byte, size)
					// 模拟使用缓冲区
					for j := 0; j < len(buffer); j += 64 {
						buffer[j] = byte(i % 256)
					}
					_ = buffer
				}
			})
		}
	})

	b.Run("会话状态分配", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// 模拟创建会话状态对象
			state := &terminal.SessionState{
				ID:        fmt.Sprintf("session_%d", i),
				Command:   []string{"/bin/bash"},
				WorkDir:   "/tmp",
				Env:       []string{"PATH=/bin:/usr/bin"},
				CreatedAt: time.Now(),
			}
			_ = state
		}
	})
}

// 会话持久化性能基准测试
func BenchmarkSessionPersistence(b *testing.B) {
	tempDir := "/tmp/clixgo_bench_persistence_" + strconv.Itoa(int(time.Now().Unix()))
	defer os.RemoveAll(tempDir)

	manager := terminal.NewSessionManager(tempDir)

	// 创建测试会话
	sessionID := "persistence_test_session"
	session, err := manager.CreateSession(sessionID, &terminal.SessionConfig{
		Command: []string{"/bin/bash"},
		WorkDir: "/tmp",
	})
	if err != nil {
		b.Fatalf("创建会话失败: %v", err)
	}
	defer session.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 保存会话状态
		err := session.SaveState()
		if err != nil {
			b.Fatalf("保存会话状态失败: %v", err)
		}

		// 模拟从磁盘恢复
		_, err = manager.RestoreSession(sessionID)
		if err != nil {
			b.Fatalf("恢复会话失败: %v", err)
		}
	}
}

// 启动时间基准测试
func BenchmarkStartupTime(b *testing.B) {
	tempDir := "/tmp/clixgo_bench_startup_" + strconv.Itoa(int(time.Now().Unix()))
	defer os.RemoveAll(tempDir)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		startTime := time.Now()

		// 模拟完整的启动过程
		manager := terminal.NewSessionManager(tempDir)

		// 创建默认会话
		sessionID := "default"
		session, err := manager.CreateSession(sessionID, &terminal.SessionConfig{
			Command: []string{"/bin/bash"},
			WorkDir: "/tmp",
		})
		if err != nil {
			b.Fatalf("启动过程失败: %v", err)
		}

		elapsedTime := time.Since(startTime)

		// 记录启动时间
		if i == 0 {
			b.Logf("首次启动时间: %v", elapsedTime)
		}

		// 清理
		session.Close()
		manager.DestroySession(sessionID)
	}
}
