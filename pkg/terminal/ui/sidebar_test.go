/*
* @Author: Lzww0608
* @Date: 2025-01-08 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-01-08 10:00:00
* @Description: 工具侧边栏的单元测试 - Step 3 测试验证
 */

package ui

import (
	"sync"
	"testing"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
)

// MockSessionManager 模拟会话管理器，用于测试
type MockSessionManager struct {
	sessions []MockSessionInfo
	stats    map[string]interface{}
}

func (m *MockSessionManager) GetPerformanceStats() interface{} {
	if m.stats == nil {
		return map[string]interface{}{
			"sessions":    len(m.sessions),
			"memory_mb":   2.5,
			"create_time": "1ms",
			"windows":     1,
			"panes":       1,
			"buffer_hits": 100,
			"buffer_miss": 5,
		}
	}
	return m.stats
}

func (m *MockSessionManager) ListSessions() []SessionInfo {
	result := make([]SessionInfo, len(m.sessions))
	for i, session := range m.sessions {
		result[i] = &session
	}
	return result
}

func (m *MockSessionManager) GetGoroutinePool() GoroutinePoolInterface {
	return &MockGoroutinePool{}
}

// MockSessionInfo 模拟会话信息
type MockSessionInfo struct {
	id      string
	name    string
	windows []MockWindowInfo
}

func (m *MockSessionInfo) GetID() string {
	return m.id
}

func (m *MockSessionInfo) GetName() string {
	return m.name
}

func (m *MockSessionInfo) GetWindows() []WindowInfo {
	result := make([]WindowInfo, len(m.windows))
	for i, window := range m.windows {
		result[i] = &window
	}
	return result
}

// MockWindowInfo 模拟窗口信息
type MockWindowInfo struct {
	panes []MockPaneInfo
}

func (m *MockWindowInfo) GetPanes() []PaneInfo {
	result := make([]PaneInfo, len(m.panes))
	for i, pane := range m.panes {
		result[i] = &pane
	}
	return result
}

// MockPaneInfo 模拟面板信息
type MockPaneInfo struct {
	id string
}

func (m *MockPaneInfo) GetID() string {
	return m.id
}

// MockGoroutinePool 模拟协程池
type MockGoroutinePool struct{}

func (m *MockGoroutinePool) GetMetrics() PoolMetrics {
	return &MockPoolMetrics{}
}

// MockPoolMetrics 模拟池指标
type MockPoolMetrics struct{}

func (m *MockPoolMetrics) GetActiveWorkers() int64 {
	return 5
}

// createMockSessionManager 创建模拟会话管理器
func createMockSessionManager() *MockSessionManager {
	return &MockSessionManager{
		sessions: []MockSessionInfo{
			{
				id:   "session1",
				name: "Test Session 1",
				windows: []MockWindowInfo{
					{
						panes: []MockPaneInfo{
							{id: "pane1"},
							{id: "pane2"},
						},
					},
				},
			},
			{
				id:   "session2",
				name: "Test Session 2",
				windows: []MockWindowInfo{
					{
						panes: []MockPaneInfo{
							{id: "pane3"},
						},
					},
				},
			},
		},
	}
}

// ====== 阶段1: Sidebar组件基础测试 ======

func TestNewSidebar(t *testing.T) {
	logger.InitLogger()
	defer logger.Close()

	mockManager := createMockSessionManager()
	sidebar := NewSidebar(mockManager)

	if sidebar == nil {
		t.Fatal("NewSidebar返回nil")
	}

	if sidebar.List == nil {
		t.Fatal("Sidebar.List为nil")
	}

	if sidebar.sessionManager == nil {
		t.Fatal("Sidebar.sessionManager为nil")
	}

	if !sidebar.visible {
		t.Error("Sidebar默认应该可见")
	}

	if sidebar.width != 30 {
		t.Errorf("Sidebar默认宽度错误，期望: 30, 实际: %d", sidebar.width)
	}

	if sidebar.updateInterval != 2*time.Second {
		t.Errorf("Sidebar更新间隔错误，期望: 2s, 实际: %v", sidebar.updateInterval)
	}

	// 验证工具数量
	if len(sidebar.tools) == 0 {
		t.Error("Sidebar工具列表为空")
	}

	t.Logf("成功创建Sidebar，工具数量: %d", len(sidebar.tools))
}

func TestSidebarToolInitialization(t *testing.T) {
	logger.InitLogger()
	defer logger.Close()

	mockManager := createMockSessionManager()
	sidebar := NewSidebar(mockManager)

	expectedTools := []string{
		"performance_stats",
		"session_info",
		"system_info",
		"quick_actions",
		"network_monitor",
		"help_info",
	}

	if len(sidebar.tools) != len(expectedTools) {
		t.Errorf("工具数量错误，期望: %d, 实际: %d", len(expectedTools), len(sidebar.tools))
	}

	// 验证每个工具的基本属性
	for i, expectedID := range expectedTools {
		if i >= len(sidebar.tools) {
			t.Errorf("缺少工具: %s", expectedID)
			continue
		}

		tool := sidebar.tools[i]
		if tool.ID != expectedID {
			t.Errorf("工具ID错误，期望: %s, 实际: %s", expectedID, tool.ID)
		}

		if tool.Name == "" {
			t.Errorf("工具名称为空: %s", tool.ID)
		}

		if tool.Icon == "" {
			t.Errorf("工具图标为空: %s", tool.ID)
		}

		if tool.Category == "" {
			t.Errorf("工具分类为空: %s", tool.ID)
		}

		if tool.DataSource == nil {
			t.Errorf("工具数据源为nil: %s", tool.ID)
		}

		if tool.Formatter == nil {
			t.Errorf("工具格式化函数为nil: %s", tool.ID)
		}
	}

	// 验证分类索引
	if len(sidebar.categories) == 0 {
		t.Error("工具分类索引为空")
	}

	t.Logf("工具初始化验证完成，分类数量: %d", len(sidebar.categories))
}

func TestSidebarVisibilityControl(t *testing.T) {
	logger.InitLogger()
	defer logger.Close()

	mockManager := createMockSessionManager()
	sidebar := NewSidebar(mockManager)

	// 测试默认可见性
	if !sidebar.IsVisible() {
		t.Error("Sidebar默认应该可见")
	}

	// 测试切换可见性
	sidebar.Toggle()
	if sidebar.IsVisible() {
		t.Error("Toggle后Sidebar应该不可见")
	}

	sidebar.Toggle()
	if !sidebar.IsVisible() {
		t.Error("再次Toggle后Sidebar应该可见")
	}

	// 测试直接设置可见性
	sidebar.SetVisible(false)
	if sidebar.IsVisible() {
		t.Error("SetVisible(false)后Sidebar应该不可见")
	}

	sidebar.SetVisible(true)
	if !sidebar.IsVisible() {
		t.Error("SetVisible(true)后Sidebar应该可见")
	}
}

func TestSidebarWidthControl(t *testing.T) {
	logger.InitLogger()
	defer logger.Close()

	mockManager := createMockSessionManager()
	sidebar := NewSidebar(mockManager)

	// 测试默认宽度
	if sidebar.GetWidth() != 30 {
		t.Errorf("默认宽度错误，期望: 30, 实际: %d", sidebar.GetWidth())
	}

	// 测试设置正常宽度
	sidebar.SetWidth(50)
	if sidebar.GetWidth() != 50 {
		t.Errorf("设置宽度失败，期望: 50, 实际: %d", sidebar.GetWidth())
	}

	// 测试宽度边界限制
	sidebar.SetWidth(10) // 小于最小值20
	if sidebar.GetWidth() != 20 {
		t.Errorf("宽度下限限制失败，期望: 20, 实际: %d", sidebar.GetWidth())
	}

	sidebar.SetWidth(100) // 大于最大值80
	if sidebar.GetWidth() != 80 {
		t.Errorf("宽度上限限制失败，期望: 80, 实际: %d", sidebar.GetWidth())
	}
}

// ====== 阶段2: 数据源和格式化测试 ======

func TestSidebarDataSources(t *testing.T) {
	logger.InitLogger()
	defer logger.Close()

	mockManager := createMockSessionManager()
	sidebar := NewSidebar(mockManager)

	// 测试每个工具的数据源
	for _, tool := range sidebar.tools {
		if !tool.Enabled {
			continue
		}

		t.Run(tool.ID, func(t *testing.T) {
			// 获取数据
			data := tool.DataSource()
			if data == nil {
				t.Errorf("工具 %s 数据源返回nil", tool.ID)
				return
			}

			// 格式化数据
			formatted := tool.Formatter(data)
			if formatted == "" {
				t.Errorf("工具 %s 格式化结果为空", tool.ID)
				return
			}

			t.Logf("工具 %s 数据: %s", tool.ID, formatted)
		})
	}
}

func TestPerformanceStatsDataSource(t *testing.T) {
	logger.InitLogger()
	defer logger.Close()

	mockManager := createMockSessionManager()
	sidebar := NewSidebar(mockManager)

	// 获取性能统计数据
	data := sidebar.getPerformanceStats()
	if data == nil {
		t.Fatal("性能统计数据为nil")
	}

	statsMap, ok := data.(map[string]interface{})
	if !ok {
		t.Fatal("性能统计数据类型错误")
	}

	expectedKeys := []string{"sessions", "memory_mb", "create_time", "windows", "panes"}
	for _, key := range expectedKeys {
		if _, exists := statsMap[key]; !exists {
			t.Errorf("缺少性能统计字段: %s", key)
		}
	}

	// 测试格式化
	formatted := sidebar.formatPerformanceStats(data)
	if formatted == "" {
		t.Error("性能统计格式化结果为空")
	}

	t.Logf("性能统计格式化结果: %s", formatted)
}

func TestSessionInfoDataSource(t *testing.T) {
	logger.InitLogger()
	defer logger.Close()

	mockManager := createMockSessionManager()
	sidebar := NewSidebar(mockManager)

	// 获取会话信息数据
	data := sidebar.getSessionInfo()
	if data == nil {
		t.Fatal("会话信息数据为nil")
	}

	infoMap, ok := data.(map[string]interface{})
	if !ok {
		t.Fatal("会话信息数据类型错误")
	}

	// 验证会话统计
	totalSessions, ok := infoMap["total_sessions"].(int)
	if !ok || totalSessions != 2 {
		t.Errorf("会话数量错误，期望: 2, 实际: %v", infoMap["total_sessions"])
	}

	totalWindows, ok := infoMap["total_windows"].(int)
	if !ok || totalWindows != 2 {
		t.Errorf("窗口数量错误，期望: 2, 实际: %v", infoMap["total_windows"])
	}

	totalPanes, ok := infoMap["total_panes"].(int)
	if !ok || totalPanes != 3 {
		t.Errorf("面板数量错误，期望: 3, 实际: %v", infoMap["total_panes"])
	}

	// 测试格式化
	formatted := sidebar.formatSessionInfo(data)
	if formatted == "" {
		t.Error("会话信息格式化结果为空")
	}

	t.Logf("会话信息格式化结果: %s", formatted)
}

func TestSystemInfoDataSource(t *testing.T) {
	logger.InitLogger()
	defer logger.Close()

	mockManager := createMockSessionManager()
	sidebar := NewSidebar(mockManager)

	// 获取系统信息数据
	data := sidebar.getSystemInfo()
	if data == nil {
		t.Fatal("系统信息数据为nil")
	}

	infoMap, ok := data.(map[string]interface{})
	if !ok {
		t.Fatal("系统信息数据类型错误")
	}

	expectedKeys := []string{"time", "date", "uptime", "goroutines"}
	for _, key := range expectedKeys {
		if _, exists := infoMap[key]; !exists {
			t.Errorf("缺少系统信息字段: %s", key)
		}
	}

	// 测试格式化
	formatted := sidebar.formatSystemInfo(data)
	if formatted == "" {
		t.Error("系统信息格式化结果为空")
	}

	t.Logf("系统信息格式化结果: %s", formatted)
}

// ====== 阶段3: 生命周期测试 ======

func TestSidebarLifecycle(t *testing.T) {
	logger.InitLogger()
	defer logger.Close()

	mockManager := createMockSessionManager()
	sidebar := NewSidebar(mockManager)

	// 测试启动
	err := sidebar.Start()
	if err != nil {
		t.Fatalf("启动Sidebar失败: %v", err)
	}

	// 验证运行状态
	sidebar.mutex.RLock()
	isRunning := sidebar.isRunning
	sidebar.mutex.RUnlock()

	if !isRunning {
		t.Error("Sidebar启动后应该处于运行状态")
	}

	// 测试重复启动
	err = sidebar.Start()
	if err == nil {
		t.Error("重复启动应该返回错误")
	}

	// 等待一段时间，确保更新循环正常工作
	time.Sleep(100 * time.Millisecond)

	// 测试停止
	sidebar.Stop()

	// 验证停止状态
	sidebar.mutex.RLock()
	isRunning = sidebar.isRunning
	sidebar.mutex.RUnlock()

	if isRunning {
		t.Error("Sidebar停止后应该不处于运行状态")
	}

	// 测试重复停止
	sidebar.Stop() // 应该不会panic
}

// ====== 阶段4: 并发安全测试 ======

func TestSidebarConcurrentAccess(t *testing.T) {
	logger.InitLogger()
	defer logger.Close()

	mockManager := createMockSessionManager()
	sidebar := NewSidebar(mockManager)

	// 启动sidebar
	err := sidebar.Start()
	if err != nil {
		t.Fatalf("启动Sidebar失败: %v", err)
	}
	defer sidebar.Stop()

	// 并发访问测试
	var wg sync.WaitGroup
	concurrency := 10
	iterations := 100

	// 并发读取测试
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// 并发读取操作
				_ = sidebar.IsVisible()
				_ = sidebar.GetWidth()

				// 并发数据更新
				for _, tool := range sidebar.tools {
					if tool.Enabled && tool.DataSource != nil {
						data := tool.DataSource()
						if tool.Formatter != nil && data != nil {
							_ = tool.Formatter(data)
						}
					}
				}

				// 短暂休眠，增加并发冲突概率
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	// 并发写入测试
	for i := 0; i < concurrency/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations/2; j++ {
				// 并发写入操作
				sidebar.Toggle()
				sidebar.SetWidth(30 + (id % 10))
				sidebar.SetVisible(j%2 == 0)

				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	// 等待所有协程完成
	wg.Wait()

	t.Log("并发访问测试完成")
}

func TestSidebarConcurrentToolSelection(t *testing.T) {
	logger.InitLogger()
	defer logger.Close()

	mockManager := createMockSessionManager()
	sidebar := NewSidebar(mockManager)

	// 启动sidebar
	err := sidebar.Start()
	if err != nil {
		t.Fatalf("启动Sidebar失败: %v", err)
	}
	defer sidebar.Stop()

	// 并发工具选择测试
	var wg sync.WaitGroup
	concurrency := 5
	iterations := 50

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// 模拟工具选择
				toolIndex := j % len(sidebar.tools)
				sidebar.handleToolSelection(toolIndex)
				sidebar.updateToolPreview(toolIndex)
				sidebar.refreshToolData(toolIndex)

				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	wg.Wait()
	t.Log("并发工具选择测试完成")
}

// ====== 阶段5: 错误处理和边界测试 ======

func TestSidebarErrorHandling(t *testing.T) {
	logger.InitLogger()
	defer logger.Close()

	// 测试nil sessionManager
	sidebar := NewSidebar(nil)
	if sidebar == nil {
		t.Fatal("NewSidebar(nil)应该返回有效的sidebar")
	}

	// 测试nil数据源的处理
	data := sidebar.getPerformanceStats()
	if data == nil {
		t.Log("nil sessionManager时性能统计返回nil，这是预期的")
	}

	// 测试工具选择边界
	sidebar.handleToolSelection(-1)  // 负数索引
	sidebar.handleToolSelection(100) // 超出范围的索引

	// 测试更新工具预览边界
	sidebar.updateToolPreview(-1)
	sidebar.updateToolPreview(100)

	// 测试刷新工具数据边界
	sidebar.refreshToolData(-1)
	sidebar.refreshToolData(100)
}

func TestSidebarMockDataHandling(t *testing.T) {
	logger.InitLogger()
	defer logger.Close()

	mockManager := createMockSessionManager()

	// 设置自定义统计数据
	mockManager.stats = map[string]interface{}{
		"sessions":    5,
		"memory_mb":   10.5,
		"create_time": "2ms",
		"windows":     8,
		"panes":       15,
		"buffer_hits": 1000,
		"buffer_miss": 50,
	}

	sidebar := NewSidebar(mockManager)

	// 测试自定义数据
	data := sidebar.getPerformanceStats()
	statsMap, ok := data.(map[string]interface{})
	if !ok {
		t.Fatal("性能统计数据类型错误")
	}

	if statsMap["sessions"] != 5 {
		t.Errorf("会话数量错误，期望: 5, 实际: %v", statsMap["sessions"])
	}

	if statsMap["memory_mb"] != 10.5 {
		t.Errorf("内存使用错误，期望: 10.5, 实际: %v", statsMap["memory_mb"])
	}

	// 测试格式化
	formatted := sidebar.formatPerformanceStats(data)
	if formatted == "" {
		t.Error("自定义数据格式化结果为空")
	}

	t.Logf("自定义数据格式化结果: %s", formatted)
}

// ====== 性能和内存测试 ======

func TestSidebarMemoryUsage(t *testing.T) {
	logger.InitLogger()
	defer logger.Close()

	// 创建多个sidebar实例，测试内存使用
	sidebars := make([]*Sidebar, 100)
	mockManager := createMockSessionManager()

	for i := 0; i < 100; i++ {
		sidebars[i] = NewSidebar(mockManager)
		if err := sidebars[i].Start(); err != nil {
			t.Errorf("启动Sidebar %d失败: %v", i, err)
		}
	}

	// 运行一段时间
	time.Sleep(500 * time.Millisecond)

	// 清理
	for i := 0; i < 100; i++ {
		sidebars[i].Stop()
	}

	t.Log("内存使用测试完成")
}

func BenchmarkSidebarDataUpdate(b *testing.B) {
	logger.InitLogger()
	defer logger.Close()

	mockManager := createMockSessionManager()
	sidebar := NewSidebar(mockManager)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sidebar.refreshData()
	}
}

func BenchmarkSidebarToolSelection(b *testing.B) {
	logger.InitLogger()
	defer logger.Close()

	mockManager := createMockSessionManager()
	sidebar := NewSidebar(mockManager)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		toolIndex := i % len(sidebar.tools)
		sidebar.handleToolSelection(toolIndex)
	}
}
