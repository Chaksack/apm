package analyzer

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"time"
)

// Metrics holds all Prometheus metrics for the Semgrep analyzer
type Metrics struct {
	// Scan metrics
	ScansTotal   prometheus.Counter
	ScanDuration prometheus.Histogram
	ScanErrors   prometheus.Counter
	FilesScanned prometheus.Histogram
	LinesScanned prometheus.Histogram

	// Finding metrics
	FindingsTotal      *prometheus.CounterVec
	FindingsBySeverity *prometheus.CounterVec
	FindingsByRule     *prometheus.CounterVec
	FindingsByFile     *prometheus.GaugeVec

	// Performance metrics
	MemoryUsage  prometheus.Gauge
	CPUUsage     prometheus.Gauge
	CacheHitRate prometheus.Gauge

	// Security score metrics
	SecurityScore      prometheus.Gauge
	VulnerabilityScore *prometheus.GaugeVec
}

// NewMetrics creates and registers all metrics
func NewMetrics(namespace, subsystem string) *Metrics {
	return &Metrics{
		ScansTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "scans_total",
			Help:      "Total number of Semgrep scans performed",
		}),

		ScanDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "scan_duration_seconds",
			Help:      "Duration of Semgrep scans in seconds",
			Buckets:   prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~17min
		}),

		ScanErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "scan_errors_total",
			Help:      "Total number of scan errors",
		}),

		FilesScanned: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "files_scanned",
			Help:      "Number of files scanned per scan",
			Buckets:   prometheus.ExponentialBuckets(1, 2, 15), // 1 to 32k files
		}),

		LinesScanned: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "lines_scanned",
			Help:      "Number of lines scanned per scan",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 7), // 100 to 100M lines
		}),

		FindingsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "findings_total",
			Help:      "Total number of findings by category",
		}, []string{"category"}),

		FindingsBySeverity: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "findings_by_severity_total",
			Help:      "Total number of findings by severity",
		}, []string{"severity"}),

		FindingsByRule: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "findings_by_rule_total",
			Help:      "Total number of findings by rule",
		}, []string{"rule_id", "severity"}),

		FindingsByFile: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "findings_by_file",
			Help:      "Current number of findings per file",
		}, []string{"file", "severity"}),

		MemoryUsage: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "memory_usage_bytes",
			Help:      "Memory usage of the Semgrep process in bytes",
		}),

		CPUUsage: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cpu_usage_percent",
			Help:      "CPU usage percentage of the Semgrep process",
		}),

		CacheHitRate: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cache_hit_rate",
			Help:      "Cache hit rate for Semgrep scans",
		}),

		SecurityScore: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "security_score",
			Help:      "Overall security score (0-100)",
		}),

		VulnerabilityScore: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "vulnerability_score",
			Help:      "Vulnerability score by category",
		}, []string{"category"}),
	}
}

// RecordScan records metrics for a completed scan
func (m *Metrics) RecordScan(result *SemgrepResult, duration time.Duration, err error) {
	m.ScansTotal.Inc()
	m.ScanDuration.Observe(duration.Seconds())

	if err != nil {
		m.ScanErrors.Inc()
		return
	}

	if result.Stats != nil {
		m.FilesScanned.Observe(float64(result.Stats.FilesCount))
		m.LinesScanned.Observe(float64(result.Stats.LinesScanned))
	}

	// Record findings by severity
	severityCounts := make(map[string]int)
	for _, finding := range result.Results {
		severity := string(normalizeSeverity(finding.Severity))
		severityCounts[severity]++
		m.FindingsBySeverity.WithLabelValues(severity).Inc()
		m.FindingsByRule.WithLabelValues(finding.CheckID, severity).Inc()
	}

	// Update current findings by file
	fileCounts := make(map[string]map[string]int)
	for _, finding := range result.Results {
		if fileCounts[finding.Path] == nil {
			fileCounts[finding.Path] = make(map[string]int)
		}
		severity := string(normalizeSeverity(finding.Severity))
		fileCounts[finding.Path][severity]++
	}

	// Update gauge metrics
	for file, severities := range fileCounts {
		for severity, count := range severities {
			m.FindingsByFile.WithLabelValues(file, severity).Set(float64(count))
		}
	}
}

// RecordSecurityScore records the calculated security score
func (m *Metrics) RecordSecurityScore(score float64) {
	m.SecurityScore.Set(score)
}

// RecordVulnerabilityScore records vulnerability scores by category
func (m *Metrics) RecordVulnerabilityScore(category string, score float64) {
	m.VulnerabilityScore.WithLabelValues(category).Set(score)
}

// RecordResourceUsage records resource usage metrics
func (m *Metrics) RecordResourceUsage(memoryBytes int64, cpuPercent float64) {
	m.MemoryUsage.Set(float64(memoryBytes))
	m.CPUUsage.Set(cpuPercent)
}

// RecordCacheHitRate records the cache hit rate
func (m *Metrics) RecordCacheHitRate(hitRate float64) {
	m.CacheHitRate.Set(hitRate)
}

// MetricsCollector provides an interface for collecting metrics
type MetricsCollector interface {
	RecordScan(result *SemgrepResult, duration time.Duration, err error)
	RecordSecurityScore(score float64)
	RecordVulnerabilityScore(category string, score float64)
	RecordResourceUsage(memoryBytes int64, cpuPercent float64)
	RecordCacheHitRate(hitRate float64)
}

// NopMetricsCollector is a no-op implementation of MetricsCollector
type NopMetricsCollector struct{}

func (n *NopMetricsCollector) RecordScan(*SemgrepResult, time.Duration, error) {}
func (n *NopMetricsCollector) RecordSecurityScore(float64)                     {}
func (n *NopMetricsCollector) RecordVulnerabilityScore(string, float64)        {}
func (n *NopMetricsCollector) RecordResourceUsage(int64, float64)              {}
func (n *NopMetricsCollector) RecordCacheHitRate(float64)                      {}

// DefaultMetrics returns default metrics or a no-op collector
func DefaultMetrics(enabled bool, namespace, subsystem string) MetricsCollector {
	if enabled {
		return NewMetrics(namespace, subsystem)
	}
	return &NopMetricsCollector{}
}
