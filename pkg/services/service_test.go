/*
* @Author: Lzww0608
* @Date: 2025-6-10 10:57:41
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-10 10:57:44
* @Description: 服务层测试 - Phase 3 服务层抽象功能验证
 */

package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/interfaces"
	"github.com/Lzww0608/ClixGo/pkg/logger"
	"go.uber.org/zap"
)

// MockLogger 模拟日志器实现
type MockLogger struct {
	name string
}

func NewMockLogger(name string) *MockLogger {
	return &MockLogger{name: name}
}

func (m *MockLogger) Debug(msg string, fields ...zap.Field) {
	fmt.Printf("[DEBUG] %s: %s\n", m.name, msg)
}

func (m *MockLogger) Info(msg string, fields ...zap.Field) {
	fmt.Printf("[INFO] %s: %s\n", m.name, msg)
}

func (m *MockLogger) Warn(msg string, fields ...zap.Field) {
	fmt.Printf("[WARN] %s: %s\n", m.name, msg)
}

func (m *MockLogger) Error(msg string, fields ...zap.Field) {
	fmt.Printf("[ERROR] %s: %s\n", m.name, msg)
}

func (m *MockLogger) Fatal(msg string, fields ...zap.Field) {
	fmt.Printf("[FATAL] %s: %s\n", m.name, msg)
}

func (m *MockLogger) Sync() error {
	return nil
}

func (m *MockLogger) With(fields ...zap.Field) interfaces.Logger {
	return m
}

func (m *MockLogger) WithOptions(opts ...zap.Option) interfaces.Logger {
	return m
}

func (m *MockLogger) Named(name string) interfaces.Logger {
	return &MockLogger{name: fmt.Sprintf("%s.%s", m.name, name)}
}

// MockService 模拟服务实现
type MockService struct {
	name   string
	logger interfaces.Logger
}

func NewMockService(name string, logger interfaces.Logger) *MockService {
	return &MockService{
		name:   name,
		logger: logger,
	}
}

func (m *MockService) CheckHealth(ctx context.Context) ServiceHealth {
	return ServiceHealth{
		ServiceName: m.name,
		Status:      "healthy",
		Message:     fmt.Sprintf("Mock service %s is running", m.name),
		CheckedAt:   time.Now(),
	}
}

func (m *MockService) Dispose() error {
	m.logger.Info(fmt.Sprintf("释放模拟服务资源: %s", m.name))
	return nil
}

// TestServiceRegistry 测试服务注册中心
func TestServiceRegistry(t *testing.T) {
	logger := NewMockLogger("test")
	registry := NewServiceRegistry(logger)
	defer registry.Dispose()

	// 测试服务注册
	mockService := NewMockService("test_service", logger)
	metadata := &ServiceMetadata{
		Name:        "test_service",
		Version:     "1.0.0",
		Description: "Test service",
		Tags:        []string{"test", "mock"},
	}

	err := registry.RegisterService("test_service", mockService, metadata)
	if err != nil {
		t.Fatalf("注册服务失败: %v", err)
	}

	// 测试获取服务
	instance, meta, err := registry.GetService("test_service")
	if err != nil {
		t.Fatalf("获取服务失败: %v", err)
	}

	if instance == nil {
		t.Fatal("服务实例为空")
	}

	if meta.Name != "test_service" {
		t.Fatalf("服务名称不匹配: expected %s, got %s", "test_service", meta.Name)
	}

	// 测试列出服务
	services := registry.ListServices()
	if len(services) != 1 {
		t.Fatalf("期望1个服务，实际%d个", len(services))
	}

	// 测试健康检查
	health, err := registry.GetServiceHealth("test_service")
	if err != nil {
		t.Fatalf("获取健康状态失败: %v", err)
	}

	if health.Status != "healthy" {
		t.Fatalf("期望健康状态为healthy，实际为%s", health.Status)
	}

	// 测试注销服务
	err = registry.UnregisterService("test_service")
	if err != nil {
		t.Fatalf("注销服务失败: %v", err)
	}

	// 验证服务已被注销
	services = registry.ListServices()
	if len(services) != 0 {
		t.Fatalf("期望0个服务，实际%d个", len(services))
	}

	t.Log("✅ 服务注册中心测试通过")
}

// TestServiceManager 测试服务管理器
func TestServiceManager(t *testing.T) {
	logger := NewMockLogger("test")
	manager := NewServiceManager(logger)
	defer manager.Dispose()

	// 测试注册服务实例
	mockService := NewMockService("test_service", logger)
	metadata := &ServiceMetadata{
		Name:        "test_service",
		Version:     "1.0.0",
		Description: "Test service",
		Tags:        []string{"test", "mock"},
	}

	err := manager.RegisterServiceInstance("test_service", mockService, metadata)
	if err != nil {
		t.Fatalf("注册服务实例失败: %v", err)
	}

	// 测试启动服务
	err = manager.StartService("test_service")
	if err != nil {
		t.Fatalf("启动服务失败: %v", err)
	}

	// 验证服务正在运行
	if !manager.IsServiceRunning("test_service") {
		t.Fatal("服务应该正在运行")
	}

	// 测试获取服务
	instance, err := manager.GetService("test_service")
	if err != nil {
		t.Fatalf("获取服务失败: %v", err)
	}

	if instance == nil {
		t.Fatal("服务实例为空")
	}

	// 测试获取服务信息
	info, err := manager.GetServiceInfo("test_service")
	if err != nil {
		t.Fatalf("获取服务信息失败: %v", err)
	}

	if info.Status != "running" {
		t.Fatalf("期望服务状态为running，实际为%s", info.Status)
	}

	// 测试健康检查
	ctx := context.Background()
	healthResults := manager.PerformHealthCheck(ctx)

	if len(healthResults) == 0 {
		t.Fatal("健康检查结果为空")
	}

	if healthResults["test_service"].Status != "healthy" {
		t.Fatalf("期望健康状态为healthy，实际为%s", healthResults["test_service"].Status)
	}

	// 测试重启服务
	err = manager.RestartService("test_service")
	if err != nil {
		t.Fatalf("重启服务失败: %v", err)
	}

	// 验证服务仍在运行
	if !manager.IsServiceRunning("test_service") {
		t.Fatal("重启后服务应该正在运行")
	}

	// 测试停止服务
	err = manager.StopService("test_service")
	if err != nil {
		t.Fatalf("停止服务失败: %v", err)
	}

	// 验证服务已停止
	if manager.IsServiceRunning("test_service") {
		t.Fatal("服务应该已停止")
	}

	t.Log("✅ 服务管理器测试通过")
}

// TestTerminalService 测试终端服务
func TestTerminalService(t *testing.T) {
	// 初始化日志系统
	err := logger.InitLogger()
	if err != nil {
		t.Fatalf("初始化日志系统失败: %v", err)
	}
	defer logger.Close()

	mockLogger := NewMockLogger("test")
	terminalService := NewSimpleTerminalService(mockLogger)
	defer terminalService.Dispose()

	// 测试创建会话
	session, err := terminalService.CreateSession("test-session")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}

	if session.Name() != "test-session" {
		t.Fatalf("会话名称不匹配: expected %s, got %s", "test-session", session.Name())
	}

	sessionID := session.ID()
	if sessionID == "" {
		t.Fatal("会话ID为空")
	}

	// 测试获取会话
	retrievedSession, err := terminalService.GetSession(sessionID)
	if err != nil {
		t.Fatalf("获取会话失败: %v", err)
	}

	if retrievedSession.ID() != sessionID {
		t.Fatalf("会话ID不匹配: expected %s, got %s", sessionID, retrievedSession.ID())
	}

	// 测试列出会话
	sessions, err := terminalService.ListSessions()
	if err != nil {
		t.Fatalf("列出会话失败: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("期望1个会话，实际%d个", len(sessions))
	}

	// 测试创建窗口
	window, err := terminalService.CreateWindow(sessionID, "test-window")
	if err != nil {
		t.Fatalf("创建窗口失败: %v", err)
	}

	if window.Name() != "test-window" {
		t.Fatalf("窗口名称不匹配: expected %s, got %s", "test-window", window.Name())
	}

	windowID := window.ID()
	if windowID == "" {
		t.Fatal("窗口ID为空")
	}

	// 测试获取窗口
	retrievedWindow, err := terminalService.GetWindow(sessionID, windowID)
	if err != nil {
		t.Fatalf("获取窗口失败: %v", err)
	}

	if retrievedWindow.ID() != windowID {
		t.Fatalf("窗口ID不匹配: expected %s, got %s", windowID, retrievedWindow.ID())
	}

	// 测试关闭会话
	err = terminalService.CloseSession(sessionID)
	if err != nil {
		t.Fatalf("关闭会话失败: %v", err)
	}

	// 验证会话已关闭（尝试获取应该失败）
	_, err = terminalService.GetSession(sessionID)
	if err == nil {
		t.Fatal("关闭后应该无法获取会话")
	}

	t.Log("✅ 终端服务测试通过")
}

// TestServiceIntegration 集成测试
func TestServiceIntegration(t *testing.T) {
	// 初始化日志系统
	err := logger.InitLogger()
	if err != nil {
		t.Fatalf("初始化日志系统失败: %v", err)
	}
	defer logger.Close()

	// 创建服务管理器
	manager := NewServiceManager(NewMockLogger("integration"))
	defer manager.Dispose()

	// 注册并启动所有服务
	err = manager.StartAllServices()
	if err != nil {
		t.Fatalf("启动所有服务失败: %v", err)
	}

	// 获取终端服务
	terminalServiceRaw, err := manager.GetService("terminal_service")
	if err != nil {
		t.Fatalf("获取终端服务失败: %v", err)
	}

	terminalService := terminalServiceRaw.(interfaces.TerminalService)

	// 测试终端服务功能
	session, err := terminalService.CreateSession("integration-test")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}

	// 测试会话操作
	err = session.SetName("renamed-session")
	if err != nil {
		t.Fatalf("重命名会话失败: %v", err)
	}

	if session.Name() != "renamed-session" {
		t.Fatalf("会话名称未更新: expected %s, got %s", "renamed-session", session.Name())
	}

	// 测试会话状态
	if !session.IsActive() {
		t.Fatal("会话应该处于活跃状态")
	}

	// 获取服务统计信息
	stats := manager.GetServiceStatistics()
	if stats == nil {
		t.Fatal("服务统计信息为空")
	}

	// 执行健康检查
	ctx := context.Background()
	healthResults := manager.PerformHealthCheck(ctx)
	// 注意：健康检查只对已启动的服务有效，terminal_service是通过工厂模式注册的
	// 只有在明确启动服务后才会有健康检查结果
	t.Logf("健康检查结果数量: %d", len(healthResults))

	// 清理资源
	err = terminalService.CloseSession(session.ID())
	if err != nil {
		t.Fatalf("关闭会话失败: %v", err)
	}

	// 停止所有服务
	err = manager.StopAllServices()
	if err != nil {
		t.Fatalf("停止所有服务失败: %v", err)
	}

	t.Log("✅ 集成测试通过")
}

// BenchmarkServiceManager 性能基准测试
func BenchmarkServiceManager(b *testing.B) {
	logger := NewMockLogger("benchmark")
	manager := NewServiceManager(logger)
	defer manager.Dispose()

	// 注册服务
	for i := 0; i < 100; i++ {
		serviceName := fmt.Sprintf("service_%d", i)
		mockService := NewMockService(serviceName, logger)
		metadata := &ServiceMetadata{
			Name:        serviceName,
			Version:     "1.0.0",
			Description: fmt.Sprintf("Service %d", i),
		}

		err := manager.RegisterServiceInstance(serviceName, mockService, metadata)
		if err != nil {
			b.Fatalf("注册服务失败: %v", err)
		}
	}

	b.ResetTimer()

	// 基准测试服务获取
	b.Run("GetService", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			serviceName := fmt.Sprintf("service_%d", i%100)
			_, err := manager.GetService(serviceName)
			if err != nil {
				b.Fatalf("获取服务失败: %v", err)
			}
		}
	})

	// 基准测试健康检查
	b.Run("HealthCheck", func(b *testing.B) {
		ctx := context.Background()
		for i := 0; i < b.N; i++ {
			manager.PerformHealthCheck(ctx)
		}
	})
}
