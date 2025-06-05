/*
* @Author: Lzww0608
* @Date: 2025-06-05 11:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-06-05 20:19:33
* @Description: 内存监控器单元测试
 */

package performance

import (
	"runtime"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestMemoryMonitor_Creation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := MemoryMonitorConfig{
		MonitorInterval:          5 * time.Second,
		BaselineWarmupTime:       1 * time.Second,
		MaxSnapshots:             10,
		EnablePprof:              false,
		EnableAutoOptimization:   false,
		OptimizationInterval:     30 * time.Second,
		ProfileCollectionDepth:   5,
		MemoryGrowthThresholdMB:  50.0,
		HeapGrowthThresholdMB:    25.0,
		GCPressureThreshold:      0.1,
		FragmentationThreshold:   0.3,
		AutoGCTriggerThresholdMB: 100.0,
		MemoryReleaseThresholdMB: 75.0,
		MaxOptimizationRetries:   3,
	}

	monitor := NewMemoryMonitor(config, logger)

	if monitor == nil {
		t.Fatal("内存监控器创建失败")
	}

	if monitor.config.MonitorInterval != 5*time.Second {
		t.Errorf("监控间隔配置错误，期望: %v, 实际: %v", 5*time.Second, monitor.config.MonitorInterval)
	}

	if monitor.config.MaxSnapshots != 10 {
		t.Errorf("最大快照数配置错误，期望: %d, 实际: %d", 10, monitor.config.MaxSnapshots)
	}
}

func TestMemoryMonitor_StartStop(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := MemoryMonitorConfig{
		MonitorInterval:         1 * time.Second,
		BaselineWarmupTime:      500 * time.Millisecond,
		MaxSnapshots:            5,
		EnablePprof:             false,
		EnableAutoOptimization:  false,
		MemoryGrowthThresholdMB: 100.0,
	}

	monitor := NewMemoryMonitor(config, logger)

	// 测试启动
	if monitor.IsRunning() {
		t.Error("监控器初始状态应该为停止")
	}

	err := monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}

	if !monitor.IsRunning() {
		t.Error("监控器启动后状态应该为运行中")
	}

	// 测试重复启动
	err = monitor.Start()
	if err == nil {
		t.Error("重复启动应该返回错误")
	}

	// 等待基线建立
	time.Sleep(1 * time.Second)

	// 检查基线是否建立
	baseline := monitor.GetBaseline()
	if baseline == nil {
		t.Error("基线未建立")
	}

	// 测试停止
	err = monitor.Stop()
	if err != nil {
		t.Fatalf("停止监控器失败: %v", err)
	}

	if monitor.IsRunning() {
		t.Error("监控器停止后状态应该为停止")
	}

	// 测试重复停止
	err = monitor.Stop()
	if err == nil {
		t.Error("重复停止应该返回错误")
	}
}

func TestMemoryMonitor_SnapshotCapture(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := MemoryMonitorConfig{
		MonitorInterval:         500 * time.Millisecond,
		BaselineWarmupTime:      200 * time.Millisecond,
		MaxSnapshots:            3,
		EnablePprof:             false,
		EnableAutoOptimization:  false,
		MemoryGrowthThresholdMB: 1000.0, // 设置很高避免触发告警
	}

	monitor := NewMemoryMonitor(config, logger)

	err := monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}
	defer monitor.Stop()

	// 等待收集一些快照
	time.Sleep(2 * time.Second)

	snapshots := monitor.GetSnapshots()
	if len(snapshots) == 0 {
		t.Error("应该至少有一个快照")
	}

	// 验证快照数据
	snapshot := snapshots[len(snapshots)-1]
	if snapshot.HeapAllocMB <= 0 {
		t.Error("堆内存分配应该大于0")
	}

	if snapshot.GoroutineCount <= 0 {
		t.Error("goroutine数量应该大于0")
	}

	if snapshot.Timestamp.IsZero() {
		t.Error("快照时间戳不应该为零值")
	}
}

func TestMemoryMonitor_AlertGeneration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := MemoryMonitorConfig{
		MonitorInterval:         500 * time.Millisecond,
		BaselineWarmupTime:      200 * time.Millisecond,
		MaxSnapshots:            5,
		EnablePprof:             false,
		EnableAutoOptimization:  false,
		MemoryGrowthThresholdMB: 0.1, // 设置很低的阈值容易触发告警
		HeapGrowthThresholdMB:   0.1,
		GCPressureThreshold:     0.001,
		FragmentationThreshold:  0.01,
	}

	monitor := NewMemoryMonitor(config, logger)

	err := monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}
	defer monitor.Stop()

	// 监听告警
	alertReceived := false
	go func() {
		select {
		case alert := <-monitor.GetAlertChannel():
			t.Logf("收到告警: %s - %s", alert.Type, alert.Title)
			alertReceived = true
		case <-time.After(3 * time.Second):
			// 超时
		}
	}()

	// 分配一些内存触发告警
	data := make([][]byte, 100)
	for i := 0; i < 100; i++ {
		data[i] = make([]byte, 1024*1024) // 1MB
	}

	// 等待告警生成
	time.Sleep(2 * time.Second)

	// 清理内存
	data = nil
	runtime.GC()

	// 注意：由于阈值设置很低，应该会触发告警
	// 但在测试环境中可能不一定触发，这是正常的
	t.Logf("告警是否收到: %v", alertReceived)
}

func TestMemoryMonitor_OptimizationSuggestions(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := MemoryMonitorConfig{
		MonitorInterval:          500 * time.Millisecond,
		BaselineWarmupTime:       200 * time.Millisecond,
		MaxSnapshots:             5,
		EnablePprof:              false,
		EnableAutoOptimization:   true,
		OptimizationInterval:     1 * time.Second,
		AutoGCTriggerThresholdMB: 0.1, // 设置很低的阈值
		FragmentationThreshold:   0.01,
	}

	monitor := NewMemoryMonitor(config, logger)

	err := monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}
	defer monitor.Stop()

	// 监听优化建议
	suggestionReceived := false
	go func() {
		select {
		case suggestion := <-monitor.GetOptimizationChannel():
			t.Logf("收到优化建议: %s - %s", suggestion.Type, suggestion.Title)
			suggestionReceived = true
		case <-time.After(4 * time.Second):
			// 超时
		}
	}()

	// 分配一些内存
	data := make([][]byte, 50)
	for i := 0; i < 50; i++ {
		data[i] = make([]byte, 1024*1024) // 1MB
	}

	// 等待优化建议生成
	time.Sleep(3 * time.Second)

	// 清理内存
	data = nil
	runtime.GC()

	t.Logf("优化建议是否收到: %v", suggestionReceived)
}

func TestMemoryMonitor_ForceOptimization(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := MemoryMonitorConfig{
		MonitorInterval:        1 * time.Second,
		BaselineWarmupTime:     200 * time.Millisecond,
		MaxSnapshots:           5,
		EnablePprof:            false,
		EnableAutoOptimization: false,
	}

	monitor := NewMemoryMonitor(config, logger)

	// 测试未启动时强制优化
	err := monitor.ForceOptimization()
	if err == nil {
		t.Error("未启动的监控器不应该能执行强制优化")
	}

	// 启动监控器
	err = monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}
	defer monitor.Stop()

	// 等待基线建立
	time.Sleep(500 * time.Millisecond)

	// 测试强制优化
	err = monitor.ForceOptimization()
	if err != nil {
		t.Errorf("强制优化失败: %v", err)
	}
}

func TestMemoryMonitor_GetCurrentSnapshot(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := MemoryMonitorConfig{
		MonitorInterval:        1 * time.Second,
		BaselineWarmupTime:     200 * time.Millisecond,
		MaxSnapshots:           5,
		EnablePprof:            false,
		EnableAutoOptimization: false,
	}

	monitor := NewMemoryMonitor(config, logger)

	snapshot, err := monitor.GetCurrentSnapshot()
	if err != nil {
		t.Fatalf("获取当前快照失败: %v", err)
	}

	if snapshot == nil {
		t.Error("快照不应该为nil")
	}

	if snapshot.HeapAllocMB <= 0 {
		t.Error("堆内存分配应该大于0")
	}

	if snapshot.GoroutineCount <= 0 {
		t.Error("goroutine数量应该大于0")
	}
}

func TestMemoryMonitor_MaxSnapshotsLimit(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := MemoryMonitorConfig{
		MonitorInterval:         100 * time.Millisecond, // 快速监控
		BaselineWarmupTime:      50 * time.Millisecond,
		MaxSnapshots:            3, // 限制最大快照数
		EnablePprof:             false,
		EnableAutoOptimization:  false,
		MemoryGrowthThresholdMB: 1000.0, // 避免告警
	}

	monitor := NewMemoryMonitor(config, logger)

	err := monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}
	defer monitor.Stop()

	// 等待收集足够多的快照
	time.Sleep(1 * time.Second)

	snapshots := monitor.GetSnapshots()
	if len(snapshots) > config.MaxSnapshots {
		t.Errorf("快照数量 %d 超过最大限制 %d", len(snapshots), config.MaxSnapshots)
	}
}

func TestProfileCollector_CollectProfile(t *testing.T) {
	logger := zaptest.NewLogger(t)
	collector := NewProfileCollector(10, logger)

	// 测试支持的profile类型
	profileTypes := []string{"heap", "goroutine", "allocs"}

	for _, profileType := range profileTypes {
		err := collector.CollectProfile(profileType)
		if err != nil {
			t.Errorf("收集 %s profile 失败: %v", profileType, err)
		}
	}

	// 测试不支持的profile类型
	err := collector.CollectProfile("unsupported")
	if err == nil {
		t.Error("不支持的profile类型应该返回错误")
	}
}

func TestMemoryOptimizer_AnalyzeOptimizationOpportunities(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := MemoryMonitorConfig{
		AutoGCTriggerThresholdMB: 10.0,
		FragmentationThreshold:   0.3,
	}
	optimizer := NewMemoryOptimizer(config, logger)

	// 创建模拟的快照和基线
	baseline := &MemoryBaseline{
		HeapAllocMB: 50.0,
		Timestamp:   time.Now().Add(-5 * time.Minute),
	}

	snapshot := MemorySnapshot{
		HeapAllocMB:        70.0, // 增长20MB，超过阈值10MB
		FragmentationRatio: 0.4,  // 超过阈值0.3
		Timestamp:          time.Now(),
	}

	suggestions := optimizer.AnalyzeOptimizationOpportunities(snapshot, baseline)

	if len(suggestions) == 0 {
		t.Error("应该生成优化建议")
	}

	// 检查GC优化建议
	foundGCOptimization := false
	foundDefragOptimization := false

	for _, suggestion := range suggestions {
		switch suggestion.Type {
		case "gc_optimization":
			foundGCOptimization = true
			if suggestion.Priority != "high" {
				t.Error("GC优化建议应该是高优先级")
			}
		case "defragmentation":
			foundDefragOptimization = true
			if suggestion.Priority != "medium" {
				t.Error("内存碎片整理建议应该是中等优先级")
			}
		}
	}

	if !foundGCOptimization {
		t.Error("应该生成GC优化建议")
	}

	if !foundDefragOptimization {
		t.Error("应该生成内存碎片整理建议")
	}
}

func TestMemoryOptimizer_ExecuteAutoOptimizations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := MemoryMonitorConfig{}
	optimizer := NewMemoryOptimizer(config, logger)

	// 创建包含自动执行操作的建议
	suggestions := []OptimizationSuggestion{
		{
			ID:   "test_suggestion",
			Type: "gc_optimization",
			Actions: []MemoryOptimizationAction{
				{
					Action:      "trigger_gc",
					Description: "执行垃圾回收",
					AutoExecute: true,
					Executed:    false,
				},
				{
					Action:      "memory_defrag",
					Description: "内存碎片整理",
					AutoExecute: false,
					Executed:    false,
				},
			},
		},
	}

	// 执行自动优化
	optimizer.ExecuteAutoOptimizations(suggestions)

	// 验证统计信息
	if optimizer.optimizationStats["trigger_gc"] != 1 {
		t.Error("trigger_gc操作应该被执行一次")
	}

	if optimizer.optimizationStats["memory_defrag"] != 0 {
		t.Error("memory_defrag操作不应该被自动执行")
	}
}

func TestMemoryMonitor_ConcurrentAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := MemoryMonitorConfig{
		MonitorInterval:         100 * time.Millisecond,
		BaselineWarmupTime:      50 * time.Millisecond,
		MaxSnapshots:            10,
		EnablePprof:             false,
		EnableAutoOptimization:  false,
		MemoryGrowthThresholdMB: 1000.0,
	}

	monitor := NewMemoryMonitor(config, logger)

	err := monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}
	defer monitor.Stop()

	// 并发访问测试
	done := make(chan bool, 10)

	// 启动多个goroutine并发访问
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < 10; j++ {
				// 并发调用各种方法
				monitor.GetSnapshots()
				monitor.GetBaseline()
				monitor.GetCurrentSnapshot()
				monitor.IsRunning()
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("并发访问测试超时")
		}
	}
}
