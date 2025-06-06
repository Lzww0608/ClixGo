/*
* @Author: Lzww0608
* @Date: 2025-6-6 11:29:27
* @LastEditors: Lzww0608
* @LastEditTime: 2025-6-6 12:56:50
* @Description: 对象池
 */

package performance

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
)

// PoolStats 池统计信息
type PoolStats struct {
	Gets       int64 `json:"gets"`        // 从池中获取对象的次数
	Puts       int64 `json:"puts"`        // 归还对象到池的次数
	Hits       int64 `json:"hits"`        // 池命中次数（成功从池中获取）
	Misses     int64 `json:"misses"`      // 池失效次数（需要新建对象）
	Created    int64 `json:"created"`     // 创建新对象的次数
	Destroyed  int64 `json:"destroyed"`   // 销毁对象的次数
	MaxSize    int   `json:"max_size"`    // 池的最大容量
	CurrentLen int   `json:"current_len"` // 当前池中对象数量
}

// ObjectPoolManager 对象池管理器
type ObjectPoolManager struct {
	bufferPools  map[int]*sync.Pool // 不同大小的字节缓冲区池
	builderPool  *sync.Pool         // 字符串构建器池
	slicePools   map[int]*sync.Pool // 不同大小的切片池
	channelPools map[int]*sync.Pool // 不同缓冲区大小的通道池

	stats         map[string]*PoolStats // 池统计信息
	statsMutex    sync.RWMutex          // 统计数据锁
	config        PoolConfig            // 池配置
	cleanupTicker *time.Ticker          // 清理定时器
	stopCh        chan struct{}         // 停止信号
	logger        logger.Logger         // 日志记录器
}

// PoolConfig 池配置
type PoolConfig struct {
	MaxPoolSize     int           `json:"max_pool_size"`    // 每个池的最大容量
	CleanupInterval time.Duration `json:"cleanup_interval"` // 清理间隔
	MaxIdleTime     time.Duration `json:"max_idle_time"`    // 最大空闲时间
	EnableStats     bool          `json:"enable_stats"`     // 是否启用统计
	DefaultSizes    []int         `json:"default_sizes"`    // 预设的缓冲区大小
}

// DefaultPoolConfig 默认池配置
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxPoolSize:     100,
		CleanupInterval: 5 * time.Minute,
		MaxIdleTime:     10 * time.Minute,
		EnableStats:     true,
		DefaultSizes:    []int{64, 256, 1024, 4096, 16384, 65536}, // 预设的缓冲区大小
	}
}

// NewObjectPoolManager 创建对象池管理器
func NewObjectPoolManager(config PoolConfig) *ObjectPoolManager {
	opm := &ObjectPoolManager{
		bufferPools:  make(map[int]*sync.Pool),
		slicePools:   make(map[int]*sync.Pool),
		channelPools: make(map[int]*sync.Pool),
		stats:        make(map[string]*PoolStats),
		config:       config,
		stopCh:       make(chan struct{}),
		logger:       logger.GetDefaultLogger(),
	}

	// 初始化字符串构建器池
	opm.builderPool = &sync.Pool{
		New: func() interface{} {
			opm.recordCreated("string_builder")
			return &strings.Builder{}
		},
	}

	// 为默认大小创建池
	for _, size := range config.DefaultSizes {
		opm.initBufferPool(size)
		opm.initSlicePool(size)
		opm.initChannelPool(size)
	}

	// 启动定期清理
	if config.CleanupInterval > 0 {
		opm.cleanupTicker = time.NewTicker(config.CleanupInterval)
		go opm.cleanupRoutine()
	}

	return opm
}

// initBufferPool 初始化指定大小的缓冲区池
func (opm *ObjectPoolManager) initBufferPool(size int) {
	if _, exists := opm.bufferPools[size]; exists {
		return
	}

	opm.bufferPools[size] = &sync.Pool{
		New: func() interface{} {
			opm.recordCreated(opm.getBufferPoolKey(size))
			return make([]byte, 0, size)
		},
	}

	if opm.config.EnableStats {
		opm.statsMutex.Lock()
		opm.stats[opm.getBufferPoolKey(size)] = &PoolStats{
			MaxSize: opm.config.MaxPoolSize,
		}
		opm.statsMutex.Unlock()
	}
}

// initSlicePool 初始化指定大小的切片池
func (opm *ObjectPoolManager) initSlicePool(size int) {
	if _, exists := opm.slicePools[size]; exists {
		return
	}

	opm.slicePools[size] = &sync.Pool{
		New: func() interface{} {
			opm.recordCreated(opm.getSlicePoolKey(size))
			return make([]interface{}, 0, size)
		},
	}

	if opm.config.EnableStats {
		opm.statsMutex.Lock()
		opm.stats[opm.getSlicePoolKey(size)] = &PoolStats{
			MaxSize: opm.config.MaxPoolSize,
		}
		opm.statsMutex.Unlock()
	}
}

// initChannelPool 初始化指定缓冲区大小的通道池
func (opm *ObjectPoolManager) initChannelPool(bufSize int) {
	if _, exists := opm.channelPools[bufSize]; exists {
		return
	}

	opm.channelPools[bufSize] = &sync.Pool{
		New: func() interface{} {
			opm.recordCreated(opm.getChannelPoolKey(bufSize))
			return make(chan []byte, bufSize)
		},
	}

	if opm.config.EnableStats {
		opm.statsMutex.Lock()
		opm.stats[opm.getChannelPoolKey(bufSize)] = &PoolStats{
			MaxSize: opm.config.MaxPoolSize,
		}
		opm.statsMutex.Unlock()
	}
}

// GetBuffer 获取指定容量的字节缓冲区
func (opm *ObjectPoolManager) GetBuffer(capacity int) []byte {
	size := opm.findBestSize(capacity)

	if pool, exists := opm.bufferPools[size]; exists {
		opm.recordGet(opm.getBufferPoolKey(size))

		if obj := pool.Get(); obj != nil {
			buf := obj.([]byte)
			opm.recordHit(opm.getBufferPoolKey(size))
			return buf[:0] // 重置长度但保持容量
		}
	} else {
		opm.initBufferPool(size)
		return opm.GetBuffer(capacity) // 递归调用
	}

	// 池中无可用对象，创建新的
	opm.recordMiss(opm.getBufferPoolKey(size))
	return make([]byte, 0, size)
}

// PutBuffer 归还字节缓冲区到池中
func (opm *ObjectPoolManager) PutBuffer(buf []byte) {
	if buf == nil {
		return
	}

	capacity := cap(buf)
	size := opm.findBestSize(capacity)

	if pool, exists := opm.bufferPools[size]; exists {
		// 检查容量是否匹配，防止污染池
		if cap(buf) == size {
			opm.recordPut(opm.getBufferPoolKey(size))
			pool.Put(buf[:0]) // 重置长度
		}
	}
}

// GetStringBuilder 获取字符串构建器
func (opm *ObjectPoolManager) GetStringBuilder() *strings.Builder {
	opm.recordGet("string_builder")

	if obj := opm.builderPool.Get(); obj != nil {
		builder := obj.(*strings.Builder)
		opm.recordHit("string_builder")
		builder.Reset() // 重置内容
		return builder
	}

	// 池中无可用对象，创建新的
	opm.recordMiss("string_builder")
	return &strings.Builder{}
}

// PutStringBuilder 归还字符串构建器到池中
func (opm *ObjectPoolManager) PutStringBuilder(builder *strings.Builder) {
	if builder == nil {
		return
	}

	opm.recordPut("string_builder")
	builder.Reset() // 清理内容
	opm.builderPool.Put(builder)
}

// GetSlice 获取指定容量的通用切片
func (opm *ObjectPoolManager) GetSlice(capacity int) []interface{} {
	size := opm.findBestSize(capacity)

	if pool, exists := opm.slicePools[size]; exists {
		opm.recordGet(opm.getSlicePoolKey(size))

		if obj := pool.Get(); obj != nil {
			slice := obj.([]interface{})
			opm.recordHit(opm.getSlicePoolKey(size))
			return slice[:0] // 重置长度但保持容量
		}
	} else {
		opm.initSlicePool(size)
		return opm.GetSlice(capacity) // 递归调用
	}

	// 池中无可用对象，创建新的
	opm.recordMiss(opm.getSlicePoolKey(size))
	return make([]interface{}, 0, size)
}

// PutSlice 归还切片到池中
func (opm *ObjectPoolManager) PutSlice(slice []interface{}) {
	if slice == nil {
		return
	}

	capacity := cap(slice)
	size := opm.findBestSize(capacity)

	if pool, exists := opm.slicePools[size]; exists {
		// 检查容量是否匹配
		if cap(slice) == size {
			opm.recordPut(opm.getSlicePoolKey(size))
			// 清空切片内容，防止内存泄漏
			for i := range slice {
				slice[i] = nil
			}
			pool.Put(slice[:0])
		}
	}
}

// GetByteChannel 获取字节通道
func (opm *ObjectPoolManager) GetByteChannel(bufferSize int) chan []byte {
	size := opm.findBestSize(bufferSize)

	if pool, exists := opm.channelPools[size]; exists {
		opm.recordGet(opm.getChannelPoolKey(size))

		if obj := pool.Get(); obj != nil {
			ch := obj.(chan []byte)
			opm.recordHit(opm.getChannelPoolKey(size))
			// 清空通道中的残留数据
			for len(ch) > 0 {
				<-ch
			}
			return ch
		}
	} else {
		opm.initChannelPool(size)
		return opm.GetByteChannel(bufferSize) // 递归调用
	}

	// 池中无可用对象，创建新的
	opm.recordMiss(opm.getChannelPoolKey(size))
	return make(chan []byte, size)
}

// PutByteChannel 归还字节通道到池中
func (opm *ObjectPoolManager) PutByteChannel(ch chan []byte) {
	if ch == nil {
		return
	}

	bufferSize := cap(ch)
	size := opm.findBestSize(bufferSize)

	if pool, exists := opm.channelPools[size]; exists {
		// 检查容量是否匹配
		if cap(ch) == size {
			opm.recordPut(opm.getChannelPoolKey(size))
			// 尝试清空通道中的数据，但不阻塞
			for {
				select {
				case <-ch:
					// 继续清空
				default:
					// 通道已空或已关闭，退出循环
					goto cleanup_done
				}
			}
		cleanup_done:
			pool.Put(ch)
		}
	}
}

// findBestSize 找到最适合的池大小
func (opm *ObjectPoolManager) findBestSize(requested int) int {
	for _, size := range opm.config.DefaultSizes {
		if size >= requested {
			return size
		}
	}

	// 如果请求的大小超过所有预设大小，返回最大的预设大小
	if len(opm.config.DefaultSizes) > 0 {
		return opm.config.DefaultSizes[len(opm.config.DefaultSizes)-1]
	}

	// 如果没有预设大小，使用2的幂次方向上取整
	size := 1
	for size < requested {
		size <<= 1
	}
	return size
}

// GetStats 获取池统计信息
func (opm *ObjectPoolManager) GetStats() map[string]*PoolStats {
	if !opm.config.EnableStats {
		return nil
	}

	opm.statsMutex.RLock()
	defer opm.statsMutex.RUnlock()

	result := make(map[string]*PoolStats)
	for key, stats := range opm.stats {
		result[key] = &PoolStats{
			Gets:       atomic.LoadInt64(&stats.Gets),
			Puts:       atomic.LoadInt64(&stats.Puts),
			Hits:       atomic.LoadInt64(&stats.Hits),
			Misses:     atomic.LoadInt64(&stats.Misses),
			Created:    atomic.LoadInt64(&stats.Created),
			Destroyed:  atomic.LoadInt64(&stats.Destroyed),
			MaxSize:    stats.MaxSize,
			CurrentLen: stats.CurrentLen,
		}
	}

	return result
}

// GetEfficiency 获取池效率统计
func (opm *ObjectPoolManager) GetEfficiency() map[string]float64 {
	stats := opm.GetStats()
	if stats == nil {
		return nil
	}

	efficiency := make(map[string]float64)
	for key, stat := range stats {
		total := stat.Gets
		if total > 0 {
			efficiency[key] = float64(stat.Hits) / float64(total) * 100
		}
	}

	return efficiency
}

// Reset 重置所有池和统计信息
func (opm *ObjectPoolManager) Reset() {
	// 重新创建所有池以清空它们
	for size := range opm.bufferPools {
		opm.initBufferPool(size)
	}

	// 重新创建字符串构建器池
	opm.builderPool = &sync.Pool{
		New: func() interface{} {
			opm.recordCreated("string_builder")
			return &strings.Builder{}
		},
	}

	for size := range opm.slicePools {
		opm.initSlicePool(size)
	}

	for size := range opm.channelPools {
		opm.initChannelPool(size)
	}

	// 重置统计信息
	if opm.config.EnableStats {
		opm.statsMutex.Lock()
		for _, stats := range opm.stats {
			atomic.StoreInt64(&stats.Gets, 0)
			atomic.StoreInt64(&stats.Puts, 0)
			atomic.StoreInt64(&stats.Hits, 0)
			atomic.StoreInt64(&stats.Misses, 0)
			atomic.StoreInt64(&stats.Created, 0)
			atomic.StoreInt64(&stats.Destroyed, 0)
			stats.CurrentLen = 0
		}
		opm.statsMutex.Unlock()
	}
}

// Stop 停止对象池管理器
func (opm *ObjectPoolManager) Stop() {
	if opm.cleanupTicker != nil {
		opm.cleanupTicker.Stop()
	}

	// 使用 select 来避免重复关闭 channel
	select {
	case <-opm.stopCh:
		// 已经关闭，什么都不做
	default:
		close(opm.stopCh)
	}

	if opm.logger != nil {
		opm.logger.Info("对象池管理器已停止")
	}
}

// cleanupRoutine 定期清理例程
func (opm *ObjectPoolManager) cleanupRoutine() {
	for {
		select {
		case <-opm.cleanupTicker.C:
			opm.performCleanup()
		case <-opm.stopCh:
			return
		}
	}
}

// performCleanup 执行清理操作
func (opm *ObjectPoolManager) performCleanup() {
	// 强制GC以清理未使用的对象
	runtime.GC()

	opm.logger.Debug("执行了对象池清理操作")
}

// Helper methods for statistics recording
func (opm *ObjectPoolManager) recordGet(poolKey string) {
	if !opm.config.EnableStats {
		return
	}

	opm.statsMutex.RLock()
	stats, exists := opm.stats[poolKey]
	opm.statsMutex.RUnlock()

	if exists {
		atomic.AddInt64(&stats.Gets, 1)
	}
}

func (opm *ObjectPoolManager) recordPut(poolKey string) {
	if !opm.config.EnableStats {
		return
	}

	opm.statsMutex.RLock()
	stats, exists := opm.stats[poolKey]
	opm.statsMutex.RUnlock()

	if exists {
		atomic.AddInt64(&stats.Puts, 1)
	}
}

func (opm *ObjectPoolManager) recordHit(poolKey string) {
	if !opm.config.EnableStats {
		return
	}

	opm.statsMutex.RLock()
	stats, exists := opm.stats[poolKey]
	opm.statsMutex.RUnlock()

	if exists {
		atomic.AddInt64(&stats.Hits, 1)
	}
}

func (opm *ObjectPoolManager) recordMiss(poolKey string) {
	if !opm.config.EnableStats {
		return
	}

	opm.statsMutex.RLock()
	stats, exists := opm.stats[poolKey]
	opm.statsMutex.RUnlock()

	if exists {
		atomic.AddInt64(&stats.Misses, 1)
	}
}

func (opm *ObjectPoolManager) recordCreated(poolKey string) {
	if !opm.config.EnableStats {
		return
	}

	opm.statsMutex.RLock()
	stats, exists := opm.stats[poolKey]
	opm.statsMutex.RUnlock()

	if exists {
		atomic.AddInt64(&stats.Created, 1)
	}
}

func (opm *ObjectPoolManager) recordDestroyed(poolKey string) {
	if !opm.config.EnableStats {
		return
	}

	opm.statsMutex.RLock()
	stats, exists := opm.stats[poolKey]
	opm.statsMutex.RUnlock()

	if exists {
		atomic.AddInt64(&stats.Destroyed, 1)
	}
}

// Helper methods for pool key generation
func (opm *ObjectPoolManager) getBufferPoolKey(size int) string {
	return fmt.Sprintf("buffer_%d", size)
}

func (opm *ObjectPoolManager) getSlicePoolKey(size int) string {
	return fmt.Sprintf("slice_%d", size)
}

func (opm *ObjectPoolManager) getChannelPoolKey(size int) string {
	return fmt.Sprintf("channel_%d", size)
}

// 全局对象池管理器实例
var (
	globalPoolManager *ObjectPoolManager
	poolOnce          sync.Once
)

// GetGlobalPoolManager 获取全局对象池管理器
func GetGlobalPoolManager() *ObjectPoolManager {
	poolOnce.Do(func() {
		globalPoolManager = NewObjectPoolManager(DefaultPoolConfig())
	})
	return globalPoolManager
}

// 便捷方法，使用全局池管理器

// GetPooledBuffer 从全局池获取缓冲区
func GetPooledBuffer(capacity int) []byte {
	return GetGlobalPoolManager().GetBuffer(capacity)
}

// PutPooledBuffer 归还缓冲区到全局池
func PutPooledBuffer(buf []byte) {
	GetGlobalPoolManager().PutBuffer(buf)
}

// GetPooledStringBuilder 从全局池获取字符串构建器
func GetPooledStringBuilder() *strings.Builder {
	return GetGlobalPoolManager().GetStringBuilder()
}

// PutPooledStringBuilder 归还字符串构建器到全局池
func PutPooledStringBuilder(builder *strings.Builder) {
	GetGlobalPoolManager().PutStringBuilder(builder)
}

// GetPooledSlice 从全局池获取切片
func GetPooledSlice(capacity int) []interface{} {
	return GetGlobalPoolManager().GetSlice(capacity)
}

// PutPooledSlice 归还切片到全局池
func PutPooledSlice(slice []interface{}) {
	GetGlobalPoolManager().PutSlice(slice)
}

// GetPooledByteChannel 从全局池获取字节通道
func GetPooledByteChannel(bufferSize int) chan []byte {
	return GetGlobalPoolManager().GetByteChannel(bufferSize)
}

// PutPooledByteChannel 归还字节通道到全局池
func PutPooledByteChannel(ch chan []byte) {
	GetGlobalPoolManager().PutByteChannel(ch)
}
