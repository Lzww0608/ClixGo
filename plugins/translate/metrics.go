package translate

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics 定义监控指标
type Metrics struct {
	requestsTotal      *prometheus.CounterVec
	requestDuration    *prometheus.HistogramVec
	cacheHits         prometheus.Counter
	cacheMisses       prometheus.Counter
	errorsTotal       *prometheus.CounterVec
	activeRequests    prometheus.Gauge
	bytesProcessed    prometheus.Counter
	concurrentWorkers prometheus.Gauge
}

var (
	metrics *Metrics
	once    sync.Once
)

// initMetrics 初始化监控指标
func initMetrics() *Metrics {
	once.Do(func() {
		metrics = &Metrics{
			requestsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "translate_requests_total",
					Help: "翻译请求总数",
				},
				[]string{"type", "source", "target"},
			),
			
			requestDuration: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "translate_request_duration_seconds",
					Help:    "翻译请求耗时",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"type"},
			),
			
			cacheHits: promauto.NewCounter(
				prometheus.CounterOpts{
					Name: "translate_cache_hits_total",
					Help: "缓存命中次数",
				},
			),
			
			cacheMisses: promauto.NewCounter(
				prometheus.CounterOpts{
					Name: "translate_cache_misses_total",
					Help: "缓存未命中次数",
				},
			),
			
			errorsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "translate_errors_total",
					Help: "错误总数",
				},
				[]string{"type"},
			),
			
			activeRequests: promauto.NewGauge(
				prometheus.GaugeOpts{
					Name: "translate_active_requests",
					Help: "当前活跃请求数",
				},
			),
			
			bytesProcessed: promauto.NewCounter(
				prometheus.CounterOpts{
					Name: "translate_bytes_processed_total",
					Help: "处理的字节总数",
				},
			),
			
			concurrentWorkers: promauto.NewGauge(
				prometheus.GaugeOpts{
					Name: "translate_concurrent_workers",
					Help: "当前并发工作协程数",
				},
			),
		}
	})
	return metrics
}

// recordMetrics 记录指标
func (s *TranslationService) recordMetrics(start time.Time, requestType, source, target string, err error, bytesProcessed int) {
	m := initMetrics()
	
	// 记录请求总数
	m.requestsTotal.WithLabelValues(requestType, source, target).Inc()
	
	// 记录请求耗时
	duration := time.Since(start).Seconds()
	m.requestDuration.WithLabelValues(requestType).Observe(duration)
	
	// 记录错误
	if err != nil {
		m.errorsTotal.WithLabelValues(requestType).Inc()
	}
	
	// 记录处理的字节数
	if bytesProcessed > 0 {
		m.bytesProcessed.Add(float64(bytesProcessed))
	}
}

// recordCacheMetrics 记录缓存指标
func (s *TranslationService) recordCacheMetrics(hit bool) {
	m := initMetrics()
	if hit {
		m.cacheHits.Inc()
	} else {
		m.cacheMisses.Inc()
	}
}

// recordActiveRequest 记录活跃请求
func (s *TranslationService) recordActiveRequest(delta float64) {
	m := initMetrics()
	m.activeRequests.Add(delta)
}

// recordWorkerCount 记录工作协程数
func (s *TranslationService) recordWorkerCount(delta float64) {
	m := initMetrics()
	m.concurrentWorkers.Add(delta)
}

// GetMetrics 获取当前指标快照
func (s *TranslationService) GetMetrics() map[string]interface{} {
	m := initMetrics()
	
	// 收集当前指标值
	metrics := make(map[string]interface{})
	
	// 使用 prometheus.DefaultGatherer 收集指标
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return metrics
	}
	
	// 处理收集到的指标
	for _, mf := range metricFamilies {
		if strings.HasPrefix(*mf.Name, "translate_") {
			for _, m := range mf.Metric {
				switch *mf.Type {
				case prometheus.MetricType_COUNTER:
					metrics[*mf.Name] = *m.Counter.Value
				case prometheus.MetricType_GAUGE:
					metrics[*mf.Name] = *m.Gauge.Value
				case prometheus.MetricType_HISTOGRAM:
					metrics[*mf.Name+"_sum"] = *m.Histogram.SampleSum
					metrics[*mf.Name+"_count"] = *m.Histogram.SampleCount
				}
			}
		}
	}
	
	return metrics
} 