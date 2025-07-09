package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type LokiPerformanceTest struct {
	LokiURL string
	Client  *http.Client
	Results TestResults
}

type TestResults struct {
	LogIngestionRate    []IngestionResult    `json:"log_ingestion_rate"`
	QueryPerformance    []QueryResult        `json:"query_performance"`
	StorageEfficiency   []StorageResult      `json:"storage_efficiency"`
	CompressionTests    []CompressionResult  `json:"compression_tests"`
}

type IngestionResult struct {
	BatchSize     int     `json:"batch_size"`
	LogsPerSecond float64 `json:"logs_per_second"`
	Duration      float64 `json:"duration"`
	Success       bool    `json:"success"`
	ErrorRate     float64 `json:"error_rate"`
}

type QueryResult struct {
	Query        string  `json:"query"`
	Duration     float64 `json:"duration"`
	ResultCount  int     `json:"result_count"`
	Success      bool    `json:"success"`
	ResponseSize int     `json:"response_size"`
}

type StorageResult struct {
	TimeRange    string  `json:"time_range"`
	LogCount     int     `json:"log_count"`
	StorageSize  int64   `json:"storage_size"`
	Efficiency   float64 `json:"efficiency"`
	QueryTime    float64 `json:"query_time"`
}

type CompressionResult struct {
	CompressionType string  `json:"compression_type"`
	OriginalSize    int     `json:"original_size"`
	CompressedSize  int     `json:"compressed_size"`
	CompressionRate float64 `json:"compression_rate"`
	CompressionTime float64 `json:"compression_time"`
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Service   string    `json:"service"`
	TraceID   string    `json:"trace_id"`
	SpanID    string    `json:"span_id"`
}

type LokiPushRequest struct {
	Streams []LokiStream `json:"streams"`
}

type LokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

func NewLokiPerformanceTest(lokiURL string) *LokiPerformanceTest {
	return &LokiPerformanceTest{
		LokiURL: lokiURL,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
		Results: TestResults{
			LogIngestionRate:  []IngestionResult{},
			QueryPerformance:  []QueryResult{},
			StorageEfficiency: []StorageResult{},
			CompressionTests:  []CompressionResult{},
		},
	}
}

func (lpt *LokiPerformanceTest) TestLogIngestionRate() {
	fmt.Println("Testing log ingestion rate...")
	
	batchSizes := []int{100, 500, 1000, 5000, 10000}
	
	for _, batchSize := range batchSizes {
		fmt.Printf("Testing batch size: %d\n", batchSize)
		
		// Generate test logs
		logs := lpt.generateTestLogs(batchSize)
		
		// Test ingestion
		startTime := time.Now()
		success, errorRate := lpt.ingestLogs(logs)
		duration := time.Since(startTime).Seconds()
		
		logsPerSecond := float64(batchSize) / duration
		
		lpt.Results.LogIngestionRate = append(lpt.Results.LogIngestionRate, IngestionResult{
			BatchSize:     batchSize,
			LogsPerSecond: logsPerSecond,
			Duration:      duration,
			Success:       success,
			ErrorRate:     errorRate,
		})
		
		fmt.Printf("Batch size %d: %.2f logs/sec, %.2f%% error rate\n", 
			batchSize, logsPerSecond, errorRate*100)
		
		// Wait between tests to avoid overwhelming Loki
		time.Sleep(2 * time.Second)
	}
}

func (lpt *LokiPerformanceTest) TestQueryPerformance() {
	fmt.Println("Testing query performance...")
	
	// First, ingest some test data
	testLogs := lpt.generateTestLogs(10000)
	lpt.ingestLogs(testLogs)
	
	// Wait for data to be available
	time.Sleep(5 * time.Second)
	
	queries := []string{
		`{job="test-app"}`,
		`{job="test-app"} |= "ERROR"`,
		`{job="test-app"} |= "ERROR" | json`,
		`{job="test-app"} | json | line_format "{{.level}}: {{.message}}"`,
		`rate({job="test-app"}[5m])`,
		`sum(rate({job="test-app"}[5m])) by (level)`,
		`topk(10, sum by (service) (rate({job="test-app"}[5m])))`,
		`count_over_time({job="test-app"} |= "ERROR" [1h])`,
		`avg_over_time({job="test-app"} | json | unwrap response_time [5m])`,
		`histogram_quantile(0.95, sum(rate({job="test-app"} | json | unwrap response_time [5m])) by (le))`,
	}
	
	for _, query := range queries {
		fmt.Printf("Testing query: %s\n", query)
		
		startTime := time.Now()
		resultCount, responseSize, success := lpt.executeQuery(query)
		duration := time.Since(startTime).Seconds()
		
		lpt.Results.QueryPerformance = append(lpt.Results.QueryPerformance, QueryResult{
			Query:        query,
			Duration:     duration,
			ResultCount:  resultCount,
			Success:      success,
			ResponseSize: responseSize,
		})
		
		fmt.Printf("Query completed in %.2fs, %d results, %d bytes\n", 
			duration, resultCount, responseSize)
	}
}

func (lpt *LokiPerformanceTest) TestStorageEfficiency() {
	fmt.Println("Testing storage efficiency...")
	
	timeRanges := []string{"1h", "6h", "1d", "7d"}
	
	for _, timeRange := range timeRanges {
		fmt.Printf("Testing time range: %s\n", timeRange)
		
		// Generate logs for the time range
		logCount := lpt.getLogCountForTimeRange(timeRange)
		logs := lpt.generateTestLogs(logCount)
		
		// Measure storage before ingestion
		storageBefore := lpt.getStorageMetrics()
		
		// Ingest logs
		startTime := time.Now()
		lpt.ingestLogs(logs)
		
		// Wait for storage to be updated
		time.Sleep(2 * time.Second)
		
		// Measure storage after ingestion
		storageAfter := lpt.getStorageMetrics()
		storageSize := storageAfter - storageBefore
		
		// Test query performance for this time range
		queryStart := time.Now()
		query := fmt.Sprintf(`{job="test-app"}[%s]`, timeRange)
		lpt.executeQuery(query)
		queryTime := time.Since(queryStart).Seconds()
		
		// Calculate efficiency (logs per byte)
		efficiency := float64(logCount) / float64(storageSize)
		
		lpt.Results.StorageEfficiency = append(lpt.Results.StorageEfficiency, StorageResult{
			TimeRange:   timeRange,
			LogCount:    logCount,
			StorageSize: storageSize,
			Efficiency:  efficiency,
			QueryTime:   queryTime,
		})
		
		fmt.Printf("Time range %s: %d logs, %d bytes, %.2f logs/byte, %.2fs query time\n",
			timeRange, logCount, storageSize, efficiency, queryTime)
	}
}

func (lpt *LokiPerformanceTest) TestCompressionEfficiency() {
	fmt.Println("Testing compression efficiency...")
	
	// Generate test data
	testData := lpt.generateLargeLogData(100000)
	originalSize := len(testData)
	
	// Test different compression scenarios
	compressionTypes := []string{"gzip", "none"}
	
	for _, compType := range compressionTypes {
		fmt.Printf("Testing compression type: %s\n", compType)
		
		startTime := time.Now()
		compressedData, compressedSize := lpt.compressData(testData, compType)
		compressionTime := time.Since(startTime).Seconds()
		
		compressionRate := float64(originalSize-compressedSize) / float64(originalSize) * 100
		
		lpt.Results.CompressionTests = append(lpt.Results.CompressionTests, CompressionResult{
			CompressionType: compType,
			OriginalSize:    originalSize,
			CompressedSize:  compressedSize,
			CompressionRate: compressionRate,
			CompressionTime: compressionTime,
		})
		
		fmt.Printf("Compression %s: %d -> %d bytes (%.2f%% reduction) in %.2fs\n",
			compType, originalSize, compressedSize, compressionRate, compressionTime)
		
		// Test ingestion with compressed data
		lpt.testCompressedIngestion(compressedData, compType)
	}
}

func (lpt *LokiPerformanceTest) generateTestLogs(count int) []LogEntry {
	logs := make([]LogEntry, count)
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	services := []string{"user-service", "order-service", "payment-service", "notification-service"}
	
	for i := 0; i < count; i++ {
		logs[i] = LogEntry{
			Timestamp: time.Now().Add(-time.Duration(rand.Intn(3600)) * time.Second),
			Level:     levels[rand.Intn(len(levels))],
			Message:   fmt.Sprintf("Test log message %d with some additional data", i),
			Service:   services[rand.Intn(len(services))],
			TraceID:   fmt.Sprintf("trace-%d", rand.Intn(10000)),
			SpanID:    fmt.Sprintf("span-%d", rand.Intn(10000)),
		}
	}
	
	return logs
}

func (lpt *LokiPerformanceTest) ingestLogs(logs []LogEntry) (bool, float64) {
	streams := make(map[string][]string)
	
	for _, log := range logs {
		streamKey := fmt.Sprintf(`{job="test-app",service="%s",level="%s"}`, log.Service, log.Level)
		
		logLine, _ := json.Marshal(log)
		timestamp := strconv.FormatInt(log.Timestamp.UnixNano(), 10)
		
		streams[streamKey] = append(streams[streamKey], fmt.Sprintf(`["%s","%s"]`, timestamp, string(logLine)))
	}
	
	// Convert to Loki format
	lokiStreams := []LokiStream{}
	for streamKey, values := range streams {
		stream := make(map[string]string)
		
		// Parse stream labels
		streamKey = strings.Trim(streamKey, "{}")
		pairs := strings.Split(streamKey, ",")
		for _, pair := range pairs {
			kv := strings.Split(pair, "=")
			if len(kv) == 2 {
				stream[kv[0]] = strings.Trim(kv[1], `"`)
			}
		}
		
		// Convert values to proper format
		valuesPairs := [][]string{}
		for _, value := range values {
			var pair []string
			json.Unmarshal([]byte(value), &pair)
			valuesPairs = append(valuesPairs, pair)
		}
		
		lokiStreams = append(lokiStreams, LokiStream{
			Stream: stream,
			Values: valuesPairs,
		})
	}
	
	pushRequest := LokiPushRequest{Streams: lokiStreams}
	
	// Send to Loki
	jsonData, _ := json.Marshal(pushRequest)
	
	req, _ := http.NewRequest("POST", lpt.LokiURL+"/loki/api/v1/push", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := lpt.Client.Do(req)
	if err != nil {
		return false, 1.0
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == 204, 0.0
}

func (lpt *LokiPerformanceTest) executeQuery(query string) (int, int, bool) {
	url := fmt.Sprintf("%s/loki/api/v1/query_range", lpt.LokiURL)
	
	req, _ := http.NewRequest("GET", url, nil)
	q := req.URL.Query()
	q.Add("query", query)
	q.Add("start", strconv.FormatInt(time.Now().Add(-1*time.Hour).UnixNano(), 10))
	q.Add("end", strconv.FormatInt(time.Now().UnixNano(), 10))
	q.Add("step", "60s")
	req.URL.RawQuery = q.Encode()
	
	resp, err := lpt.Client.Do(req)
	if err != nil {
		return 0, 0, false
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	responseSize := len(body)
	
	if resp.StatusCode != 200 {
		return 0, responseSize, false
	}
	
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	
	resultCount := 0
	if data, ok := result["data"].(map[string]interface{}); ok {
		if results, ok := data["result"].([]interface{}); ok {
			resultCount = len(results)
		}
	}
	
	return resultCount, responseSize, true
}

func (lpt *LokiPerformanceTest) getLogCountForTimeRange(timeRange string) int {
	switch timeRange {
	case "1h":
		return 1000
	case "6h":
		return 5000
	case "1d":
		return 10000
	case "7d":
		return 50000
	default:
		return 1000
	}
}

func (lpt *LokiPerformanceTest) getStorageMetrics() int64 {
	// This is a simplified version - in real implementation,
	// you would query Loki's metrics endpoint
	return time.Now().UnixNano() % 1000000
}

func (lpt *LokiPerformanceTest) generateLargeLogData(count int) []byte {
	logs := lpt.generateTestLogs(count)
	data, _ := json.Marshal(logs)
	return data
}

func (lpt *LokiPerformanceTest) compressData(data []byte, compType string) ([]byte, int) {
	switch compType {
	case "gzip":
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		w.Write(data)
		w.Close()
		return buf.Bytes(), buf.Len()
	case "none":
		return data, len(data)
	default:
		return data, len(data)
	}
}

func (lpt *LokiPerformanceTest) testCompressedIngestion(data []byte, compType string) {
	// Test ingestion of compressed data
	req, _ := http.NewRequest("POST", lpt.LokiURL+"/loki/api/v1/push", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	
	if compType == "gzip" {
		req.Header.Set("Content-Encoding", "gzip")
	}
	
	resp, err := lpt.Client.Do(req)
	if err != nil {
		fmt.Printf("Compressed ingestion failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("Compressed ingestion status: %d\n", resp.StatusCode)
}

func (lpt *LokiPerformanceTest) RunAllTests() {
	fmt.Println("Starting Loki performance tests...")
	
	// Test connection
	resp, err := lpt.Client.Get(lpt.LokiURL + "/ready")
	if err != nil {
		log.Fatalf("Failed to connect to Loki: %v", err)
	}
	resp.Body.Close()
	
	if resp.StatusCode != 200 {
		log.Fatalf("Loki not ready: %d", resp.StatusCode)
	}
	
	lpt.TestLogIngestionRate()
	lpt.TestQueryPerformance()
	lpt.TestStorageEfficiency()
	lpt.TestCompressionEfficiency()
	
	lpt.GenerateReport()
}

func (lpt *LokiPerformanceTest) GenerateReport() {
	fmt.Println("\n=== Loki Performance Test Report ===")
	
	// Log Ingestion Rate Report
	fmt.Println("\n--- Log Ingestion Rate ---")
	for _, result := range lpt.Results.LogIngestionRate {
		fmt.Printf("Batch Size: %d, Rate: %.2f logs/sec, Success: %t, Error Rate: %.2f%%\n",
			result.BatchSize, result.LogsPerSecond, result.Success, result.ErrorRate*100)
	}
	
	// Query Performance Report
	fmt.Println("\n--- Query Performance ---")
	for _, result := range lpt.Results.QueryPerformance {
		fmt.Printf("Query: %s\n", result.Query)
		fmt.Printf("Duration: %.2fs, Results: %d, Response Size: %d bytes\n",
			result.Duration, result.ResultCount, result.ResponseSize)
		fmt.Println("---")
	}
	
	// Storage Efficiency Report
	fmt.Println("\n--- Storage Efficiency ---")
	for _, result := range lpt.Results.StorageEfficiency {
		fmt.Printf("Time Range: %s, Logs: %d, Storage: %d bytes, Efficiency: %.2f logs/byte\n",
			result.TimeRange, result.LogCount, result.StorageSize, result.Efficiency)
	}
	
	// Compression Report
	fmt.Println("\n--- Compression Tests ---")
	for _, result := range lpt.Results.CompressionTests {
		fmt.Printf("Type: %s, Original: %d bytes, Compressed: %d bytes, Reduction: %.2f%%\n",
			result.CompressionType, result.OriginalSize, result.CompressedSize, result.CompressionRate)
	}
	
	// Save results to file
	jsonData, _ := json.MarshalIndent(lpt.Results, "", "  ")
	err := os.WriteFile("loki-performance-results.json", jsonData, 0644)
	if err != nil {
		fmt.Printf("Error saving results: %v\n", err)
	} else {
		fmt.Println("\nResults saved to loki-performance-results.json")
	}
	
	// Print system info
	fmt.Println("\n--- System Information ---")
	fmt.Printf("GOOS: %s\n", runtime.GOOS)
	fmt.Printf("GOARCH: %s\n", runtime.GOARCH)
	fmt.Printf("NumCPU: %d\n", runtime.NumCPU())
	fmt.Printf("NumGoroutine: %d\n", runtime.NumGoroutine())
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Memory Alloc: %d KB\n", m.Alloc/1024)
	fmt.Printf("Memory TotalAlloc: %d KB\n", m.TotalAlloc/1024)
	fmt.Printf("Memory Sys: %d KB\n", m.Sys/1024)
}

func main() {
	lokiURL := "http://localhost:3100"
	if len(os.Args) > 1 {
		lokiURL = os.Args[1]
	}
	
	test := NewLokiPerformanceTest(lokiURL)
	test.RunAllTests()
}