package instrumentation

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsCollector holds all the Prometheus metrics
type MetricsCollector struct {
	namespace string
	subsystem string

	// HTTP metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize     *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec

	// Custom collectors
	customCollectors []prometheus.Collector
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(namespace, subsystem string) *MetricsCollector {
	mc := &MetricsCollector{
		namespace: namespace,
		subsystem: subsystem,
	}

	// Initialize HTTP metrics
	mc.httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	mc.httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	mc.httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "http_request_size_bytes",
			Help:      "HTTP request size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 7), // 100B to 100MB
		},
		[]string{"method", "path"},
	)

	mc.httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "http_response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 7), // 100B to 100MB
		},
		[]string{"method", "path"},
	)

	return mc
}

// RecordHTTPRequest records an HTTP request
func (mc *MetricsCollector) RecordHTTPRequest(method, path string, status int, duration time.Duration) {
	statusStr := statusCodeClass(status)

	mc.httpRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
	mc.httpRequestDuration.WithLabelValues(method, path, statusStr).Observe(duration.Seconds())
}

// RecordHTTPRequestSize records the size of an HTTP request
func (mc *MetricsCollector) RecordHTTPRequestSize(method, path string, size float64) {
	mc.httpRequestSize.WithLabelValues(method, path).Observe(size)
}

// RecordHTTPResponseSize records the size of an HTTP response
func (mc *MetricsCollector) RecordHTTPResponseSize(method, path string, size float64) {
	mc.httpResponseSize.WithLabelValues(method, path).Observe(size)
}

// NewCounter creates a new counter metric
func (mc *MetricsCollector) NewCounter(name, help string, labels []string) *prometheus.CounterVec {
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mc.namespace,
			Subsystem: mc.subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)

	mc.customCollectors = append(mc.customCollectors, counter)
	return counter
}

// NewGauge creates a new gauge metric
func (mc *MetricsCollector) NewGauge(name, help string, labels []string) *prometheus.GaugeVec {
	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: mc.namespace,
			Subsystem: mc.subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)

	mc.customCollectors = append(mc.customCollectors, gauge)
	return gauge
}

// NewHistogram creates a new histogram metric
func (mc *MetricsCollector) NewHistogram(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	if buckets == nil {
		buckets = prometheus.DefBuckets
	}

	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mc.namespace,
			Subsystem: mc.subsystem,
			Name:      name,
			Help:      help,
			Buckets:   buckets,
		},
		labels,
	)

	mc.customCollectors = append(mc.customCollectors, histogram)
	return histogram
}

// NewSummary creates a new summary metric
func (mc *MetricsCollector) NewSummary(name, help string, labels []string, objectives map[float64]float64) *prometheus.SummaryVec {
	if objectives == nil {
		objectives = map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}
	}

	summary := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  mc.namespace,
			Subsystem:  mc.subsystem,
			Name:       name,
			Help:       help,
			Objectives: objectives,
		},
		labels,
	)

	mc.customCollectors = append(mc.customCollectors, summary)
	return summary
}

// statusCodeClass returns the status code class (2xx, 3xx, 4xx, 5xx)
func statusCodeClass(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}
