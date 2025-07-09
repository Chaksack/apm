// Package e2e provides end-to-end tests for the APM stack
package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestResult represents the result of a single test execution
type TestResult struct {
	Name      string        `json:"name"`
	Status    string        `json:"status"` // passed, failed, skipped
	Duration  time.Duration `json:"duration"`
	Error     error         `json:"error,omitempty"`
	Retries   int           `json:"retries"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Output    string        `json:"output,omitempty"`
}

// TestReport represents the complete test execution report
type TestReport struct {
	StartTime    time.Time           `json:"start_time"`
	EndTime      time.Time           `json:"end_time"`
	Duration     time.Duration       `json:"duration"`
	TotalTests   int                 `json:"total_tests"`
	PassedTests  int                 `json:"passed_tests"`
	FailedTests  int                 `json:"failed_tests"`
	SkippedTests int                 `json:"skipped_tests"`
	RetryCount   int                 `json:"retry_count"`
	Results      []TestResult        `json:"results"`
	Environment  map[string]string   `json:"environment"`
	Resources    ResourceUtilization `json:"resources"`
}

// ResourceUtilization tracks resource usage during tests
type ResourceUtilization struct {
	MaxCPU    float64 `json:"max_cpu_percent"`
	MaxMemory float64 `json:"max_memory_mb"`
	AvgCPU    float64 `json:"avg_cpu_percent"`
	AvgMemory float64 `json:"avg_memory_mb"`
}

// ParallelTestRunner manages parallel test execution
type ParallelTestRunner struct {
	maxWorkers   int
	retryCount   int
	timeout      time.Duration
	progressChan chan TestProgress
	resultsChan  chan TestResult
	results      []TestResult
	mutex        sync.Mutex
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc

	// Counters
	totalTests    int32
	passedTests   int32
	failedTests   int32
	skippedTests  int32
	retryAttempts int32

	// Resource tracking
	resourceMonitor *ResourceMonitor
}

// TestProgress represents real-time test progress
type TestProgress struct {
	TestName     string
	Status       string // running, completed, failed, retrying
	CurrentRetry int
	Message      string
	Timestamp    time.Time
}

// TestFunc represents a test function
type TestFunc struct {
	Name       string
	Function   func(*testing.T) error
	Category   string
	Priority   int
	MaxRetries int
	Timeout    time.Duration
	Required   bool // If true, failure stops all tests
}

// NewParallelTestRunner creates a new parallel test runner
func NewParallelTestRunner(maxWorkers int, retryCount int, timeout time.Duration) *ParallelTestRunner {
	ctx, cancel := context.WithCancel(context.Background())

	return &ParallelTestRunner{
		maxWorkers:      maxWorkers,
		retryCount:      retryCount,
		timeout:         timeout,
		progressChan:    make(chan TestProgress, 100),
		resultsChan:     make(chan TestResult, 100),
		results:         make([]TestResult, 0),
		ctx:             ctx,
		cancel:          cancel,
		resourceMonitor: NewResourceMonitor(),
	}
}

// Run executes tests in parallel
func (ptr *ParallelTestRunner) Run(tests []TestFunc) (*TestReport, error) {
	startTime := time.Now()
	atomic.StoreInt32(&ptr.totalTests, int32(len(tests)))

	// Start resource monitoring
	ptr.resourceMonitor.Start()
	defer ptr.resourceMonitor.Stop()

	// Start progress reporter
	go ptr.progressReporter()

	// Start result collector
	go ptr.resultCollector()

	// Create worker pool
	testQueue := make(chan TestFunc, len(tests))

	// Start workers
	for i := 0; i < ptr.maxWorkers; i++ {
		ptr.wg.Add(1)
		go ptr.worker(i, testQueue)
	}

	// Queue tests by priority
	prioritizedTests := ptr.prioritizeTests(tests)
	for _, test := range prioritizedTests {
		select {
		case testQueue <- test:
		case <-ptr.ctx.Done():
			close(testQueue)
			return nil, fmt.Errorf("test execution cancelled")
		}
	}
	close(testQueue)

	// Wait for all workers to complete
	ptr.wg.Wait()

	// Signal completion
	close(ptr.resultsChan)
	time.Sleep(100 * time.Millisecond) // Allow final results to be collected

	// Generate report
	endTime := time.Now()
	report := &TestReport{
		StartTime:    startTime,
		EndTime:      endTime,
		Duration:     endTime.Sub(startTime),
		TotalTests:   int(atomic.LoadInt32(&ptr.totalTests)),
		PassedTests:  int(atomic.LoadInt32(&ptr.passedTests)),
		FailedTests:  int(atomic.LoadInt32(&ptr.failedTests)),
		SkippedTests: int(atomic.LoadInt32(&ptr.skippedTests)),
		RetryCount:   int(atomic.LoadInt32(&ptr.retryAttempts)),
		Results:      ptr.results,
		Environment:  ptr.getEnvironmentInfo(),
		Resources:    ptr.resourceMonitor.GetSummary(),
	}

	return report, nil
}

// worker processes tests from the queue
func (ptr *ParallelTestRunner) worker(id int, testQueue <-chan TestFunc) {
	defer ptr.wg.Done()

	for test := range testQueue {
		select {
		case <-ptr.ctx.Done():
			return
		default:
			ptr.executeTest(test)
		}
	}
}

// executeTest runs a single test with retry logic
func (ptr *ParallelTestRunner) executeTest(test TestFunc) {
	maxRetries := test.MaxRetries
	if maxRetries == 0 {
		maxRetries = ptr.retryCount
	}

	timeout := test.Timeout
	if timeout == 0 {
		timeout = ptr.timeout
	}

	var lastError error
	var duration time.Duration
	startTime := time.Now()

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			atomic.AddInt32(&ptr.retryAttempts, 1)
			ptr.progressChan <- TestProgress{
				TestName:     test.Name,
				Status:       "retrying",
				CurrentRetry: attempt,
				Message:      fmt.Sprintf("Retrying test (attempt %d/%d)", attempt+1, maxRetries+1),
				Timestamp:    time.Now(),
			}
			time.Sleep(time.Second * time.Duration(attempt)) // Exponential backoff
		} else {
			ptr.progressChan <- TestProgress{
				TestName:  test.Name,
				Status:    "running",
				Message:   "Starting test execution",
				Timestamp: time.Now(),
			}
		}

		// Execute test with timeout
		testCtx, cancel := context.WithTimeout(ptr.ctx, timeout)
		t := &testing.T{}

		errChan := make(chan error, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					errChan <- fmt.Errorf("test panicked: %v", r)
				}
			}()
			errChan <- test.Function(t)
		}()

		select {
		case err := <-errChan:
			duration = time.Since(startTime)
			if err == nil {
				// Test passed
				atomic.AddInt32(&ptr.passedTests, 1)
				ptr.resultsChan <- TestResult{
					Name:      test.Name,
					Status:    "passed",
					Duration:  duration,
					Retries:   attempt,
					StartTime: startTime,
					EndTime:   time.Now(),
				}
				ptr.progressChan <- TestProgress{
					TestName:  test.Name,
					Status:    "completed",
					Message:   "Test passed successfully",
					Timestamp: time.Now(),
				}
				cancel()
				return
			}
			lastError = err

		case <-testCtx.Done():
			lastError = fmt.Errorf("test timed out after %v", timeout)
		}

		cancel()
	}

	// Test failed after all retries
	atomic.AddInt32(&ptr.failedTests, 1)
	ptr.resultsChan <- TestResult{
		Name:      test.Name,
		Status:    "failed",
		Duration:  duration,
		Error:     lastError,
		Retries:   maxRetries,
		StartTime: startTime,
		EndTime:   time.Now(),
	}

	ptr.progressChan <- TestProgress{
		TestName:  test.Name,
		Status:    "failed",
		Message:   fmt.Sprintf("Test failed: %v", lastError),
		Timestamp: time.Now(),
	}

	// If this was a required test, cancel all remaining tests
	if test.Required {
		ptr.cancel()
	}
}

// prioritizeTests sorts tests by priority (higher priority first)
func (ptr *ParallelTestRunner) prioritizeTests(tests []TestFunc) []TestFunc {
	// Simple priority sorting - in production, use a proper sorting algorithm
	prioritized := make([]TestFunc, len(tests))
	copy(prioritized, tests)

	// Sort by priority (descending) and required tests first
	for i := 0; i < len(prioritized)-1; i++ {
		for j := i + 1; j < len(prioritized); j++ {
			if prioritized[j].Required && !prioritized[i].Required {
				prioritized[i], prioritized[j] = prioritized[j], prioritized[i]
			} else if prioritized[j].Priority > prioritized[i].Priority {
				prioritized[i], prioritized[j] = prioritized[j], prioritized[i]
			}
		}
	}

	return prioritized
}

// progressReporter handles real-time progress updates
func (ptr *ParallelTestRunner) progressReporter() {
	for progress := range ptr.progressChan {
		// Format and print progress
		timestamp := progress.Timestamp.Format("15:04:05")
		status := ptr.formatStatus(progress.Status)

		fmt.Printf("[%s] %s %s: %s\n", timestamp, status, progress.TestName, progress.Message)

		// Also print overall progress
		if progress.Status == "completed" || progress.Status == "failed" {
			passed := atomic.LoadInt32(&ptr.passedTests)
			failed := atomic.LoadInt32(&ptr.failedTests)
			total := atomic.LoadInt32(&ptr.totalTests)
			completed := passed + failed

			percentage := float64(completed) / float64(total) * 100
			fmt.Printf("Progress: %d/%d (%.1f%%) - Passed: %d, Failed: %d\n\n",
				completed, total, percentage, passed, failed)
		}
	}
}

// resultCollector collects test results
func (ptr *ParallelTestRunner) resultCollector() {
	for result := range ptr.resultsChan {
		ptr.mutex.Lock()
		ptr.results = append(ptr.results, result)
		ptr.mutex.Unlock()
	}
}

// formatStatus formats status with color codes
func (ptr *ParallelTestRunner) formatStatus(status string) string {
	switch status {
	case "running":
		return "\033[34m[RUNNING]\033[0m"
	case "completed":
		return "\033[32m[PASSED]\033[0m"
	case "failed":
		return "\033[31m[FAILED]\033[0m"
	case "retrying":
		return "\033[33m[RETRY]\033[0m"
	default:
		return fmt.Sprintf("[%s]", status)
	}
}

// getEnvironmentInfo collects environment information
func (ptr *ParallelTestRunner) getEnvironmentInfo() map[string]string {
	return map[string]string{
		"go_version":  os.Getenv("GO_VERSION"),
		"os":          os.Getenv("GOOS"),
		"arch":        os.Getenv("GOARCH"),
		"max_workers": fmt.Sprintf("%d", ptr.maxWorkers),
		"retry_count": fmt.Sprintf("%d", ptr.retryCount),
		"timeout":     ptr.timeout.String(),
	}
}

// SaveReport saves the test report to a file
func (ptr *ParallelTestRunner) SaveReport(report *TestReport, filename string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// SaveHTMLReport generates and saves an HTML report
func (ptr *ParallelTestRunner) SaveHTMLReport(report *TestReport, filename string) error {
	htmlTemplate := `<!DOCTYPE html>
<html>
<head>
    <title>E2E Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .summary { margin: 20px 0; }
        .summary-item { display: inline-block; margin: 10px 20px 10px 0; }
        .passed { color: green; }
        .failed { color: red; }
        .skipped { color: orange; }
        table { border-collapse: collapse; width: 100%; margin-top: 20px; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        tr:nth-child(even) { background-color: #f9f9f9; }
        .error { background-color: #ffeeee; }
        .success { background-color: #eeffee; }
        .progress-bar { width: 100%; height: 20px; background-color: #f0f0f0; border-radius: 10px; margin: 10px 0; }
        .progress-fill { height: 100%; border-radius: 10px; background-color: #4CAF50; }
    </style>
</head>
<body>
    <div class="header">
        <h1>E2E Test Report</h1>
        <p>Generated: %s</p>
        <p>Duration: %s</p>
    </div>
    
    <div class="summary">
        <h2>Summary</h2>
        <div class="summary-item">Total Tests: <strong>%d</strong></div>
        <div class="summary-item passed">Passed: <strong>%d</strong></div>
        <div class="summary-item failed">Failed: <strong>%d</strong></div>
        <div class="summary-item skipped">Skipped: <strong>%d</strong></div>
        <div class="summary-item">Retries: <strong>%d</strong></div>
    </div>
    
    <div class="progress-bar">
        <div class="progress-fill" style="width: %.1f%%"></div>
    </div>
    <p>Success Rate: %.1f%%</p>
    
    <h2>Test Results</h2>
    <table>
        <tr>
            <th>Test Name</th>
            <th>Status</th>
            <th>Duration</th>
            <th>Retries</th>
            <th>Error</th>
        </tr>
        %s
    </table>
    
    <h2>Resource Utilization</h2>
    <table>
        <tr>
            <th>Metric</th>
            <th>Average</th>
            <th>Maximum</th>
        </tr>
        <tr>
            <td>CPU Usage</td>
            <td>%.2f%%</td>
            <td>%.2f%%</td>
        </tr>
        <tr>
            <td>Memory Usage</td>
            <td>%.2f MB</td>
            <td>%.2f MB</td>
        </tr>
    </table>
</body>
</html>`

	// Calculate success rate
	successRate := float64(report.PassedTests) / float64(report.TotalTests) * 100

	// Generate test rows
	var testRows string
	for _, result := range report.Results {
		rowClass := "success"
		if result.Status == "failed" {
			rowClass = "error"
		}

		errorMsg := ""
		if result.Error != nil {
			errorMsg = result.Error.Error()
		}

		testRows += fmt.Sprintf(`
        <tr class="%s">
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%d</td>
            <td>%s</td>
        </tr>`, rowClass, result.Name, result.Status, result.Duration, result.Retries, errorMsg)
	}

	// Generate HTML
	html := fmt.Sprintf(htmlTemplate,
		report.EndTime.Format("2006-01-02 15:04:05"),
		report.Duration,
		report.TotalTests,
		report.PassedTests,
		report.FailedTests,
		report.SkippedTests,
		report.RetryCount,
		successRate,
		successRate,
		testRows,
		report.Resources.AvgCPU,
		report.Resources.MaxCPU,
		report.Resources.AvgMemory,
		report.Resources.MaxMemory,
	)

	return os.WriteFile(filename, []byte(html), 0644)
}

// PrintSummary prints a summary of the test results
func (ptr *ParallelTestRunner) PrintSummary(report *TestReport) {
	fmt.Println("\n========================================")
	fmt.Println("Test Execution Summary")
	fmt.Println("========================================")
	fmt.Printf("Start Time: %s\n", report.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("End Time: %s\n", report.EndTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Duration: %s\n", report.Duration)
	fmt.Println("----------------------------------------")
	fmt.Printf("Total Tests: %d\n", report.TotalTests)
	fmt.Printf("Passed: %d (%.1f%%)\n", report.PassedTests, float64(report.PassedTests)/float64(report.TotalTests)*100)
	fmt.Printf("Failed: %d (%.1f%%)\n", report.FailedTests, float64(report.FailedTests)/float64(report.TotalTests)*100)
	fmt.Printf("Skipped: %d\n", report.SkippedTests)
	fmt.Printf("Total Retries: %d\n", report.RetryCount)
	fmt.Println("----------------------------------------")

	if report.FailedTests > 0 {
		fmt.Println("\nFailed Tests:")
		for _, result := range report.Results {
			if result.Status == "failed" {
				fmt.Printf("  - %s: %v\n", result.Name, result.Error)
			}
		}
	}

	fmt.Println("\nResource Usage:")
	fmt.Printf("  CPU: %.2f%% avg, %.2f%% max\n", report.Resources.AvgCPU, report.Resources.MaxCPU)
	fmt.Printf("  Memory: %.2f MB avg, %.2f MB max\n", report.Resources.AvgMemory, report.Resources.MaxMemory)
	fmt.Println("========================================")
}

// Stop gracefully stops the test runner
func (ptr *ParallelTestRunner) Stop() {
	ptr.cancel()
	close(ptr.progressChan)
}

// ResourceMonitor tracks system resource usage during tests
type ResourceMonitor struct {
	measurements []ResourceMeasurement
	ticker       *time.Ticker
	done         chan bool
	mutex        sync.Mutex
}

type ResourceMeasurement struct {
	Timestamp time.Time
	CPUUsage  float64
	MemoryMB  float64
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor() *ResourceMonitor {
	return &ResourceMonitor{
		measurements: make([]ResourceMeasurement, 0),
		done:         make(chan bool),
	}
}

// Start begins resource monitoring
func (rm *ResourceMonitor) Start() {
	rm.ticker = time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-rm.ticker.C:
				rm.recordMeasurement()
			case <-rm.done:
				return
			}
		}
	}()
}

// Stop stops resource monitoring
func (rm *ResourceMonitor) Stop() {
	rm.ticker.Stop()
	close(rm.done)
}

// recordMeasurement records current resource usage
func (rm *ResourceMonitor) recordMeasurement() {
	// This is a simplified implementation
	// In production, use proper system monitoring libraries
	measurement := ResourceMeasurement{
		Timestamp: time.Now(),
		CPUUsage:  10.0 + float64(time.Now().Unix()%20),  // Mock data
		MemoryMB:  100.0 + float64(time.Now().Unix()%50), // Mock data
	}

	rm.mutex.Lock()
	rm.measurements = append(rm.measurements, measurement)
	rm.mutex.Unlock()
}

// GetSummary returns resource utilization summary
func (rm *ResourceMonitor) GetSummary() ResourceUtilization {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if len(rm.measurements) == 0 {
		return ResourceUtilization{}
	}

	var totalCPU, totalMemory, maxCPU, maxMemory float64

	for _, m := range rm.measurements {
		totalCPU += m.CPUUsage
		totalMemory += m.MemoryMB

		if m.CPUUsage > maxCPU {
			maxCPU = m.CPUUsage
		}
		if m.MemoryMB > maxMemory {
			maxMemory = m.MemoryMB
		}
	}

	count := float64(len(rm.measurements))
	return ResourceUtilization{
		MaxCPU:    maxCPU,
		MaxMemory: maxMemory,
		AvgCPU:    totalCPU / count,
		AvgMemory: totalMemory / count,
	}
}

// ExportMetrics exports resource metrics to a file
func (rm *ResourceMonitor) ExportMetrics(filename string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := io.Writer(file)
	fmt.Fprintln(writer, "Timestamp,CPU_Usage,Memory_MB")

	for _, m := range rm.measurements {
		fmt.Fprintf(writer, "%s,%.2f,%.2f\n",
			m.Timestamp.Format("2006-01-02 15:04:05"),
			m.CPUUsage,
			m.MemoryMB,
		)
	}

	return nil
}
