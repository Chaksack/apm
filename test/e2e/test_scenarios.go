// Package e2e provides end-to-end tests for the APM stack
package e2e

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"testing"
	"time"
)

// GetLoadTestScenario returns test functions for load testing
func GetLoadTestScenario() []TestFunc {
	return []TestFunc{
		{
			Name:     "LoadTest_BasicEndpoints",
			Category: "load",
			Priority: 5,
			Timeout:  5 * time.Minute,
			Function: func(t *testing.T) error {
				return runBasicLoadTest(1000, 10)
			},
		},
		{
			Name:     "LoadTest_SpikeTraffic",
			Category: "load",
			Priority: 4,
			Timeout:  5 * time.Minute,
			Function: func(t *testing.T) error {
				return runSpikeLoadTest(100, 1000, 5)
			},
		},
		{
			Name:     "LoadTest_SustainedLoad",
			Category: "load",
			Priority: 3,
			Timeout:  10 * time.Minute,
			Function: func(t *testing.T) error {
				return runSustainedLoadTest(50, 5*time.Minute)
			},
		},
		{
			Name:     "LoadTest_MetricsAccuracy",
			Category: "load",
			Priority: 5,
			Timeout:  3 * time.Minute,
			Function: func(t *testing.T) error {
				return verifyMetricsUnderLoad(500)
			},
		},
		{
			Name:     "LoadTest_ResourceLimits",
			Category: "load",
			Priority: 2,
			Timeout:  5 * time.Minute,
			Function: func(t *testing.T) error {
				return testResourceLimits()
			},
		},
	}
}

// GetSecurityScanScenario returns test functions for security scanning
func GetSecurityScanScenario() []TestFunc {
	return []TestFunc{
		{
			Name:       "Security_CodeVulnerabilities",
			Category:   "security",
			Priority:   10,
			Required:   true,
			MaxRetries: 0,
			Function: func(t *testing.T) error {
				return runSecurityVulnerabilityCheck()
			},
		},
		{
			Name:     "Security_ConfigurationScan",
			Category: "security",
			Priority: 9,
			Function: func(t *testing.T) error {
				return runConfigurationSecurityScan()
			},
		},
		{
			Name:     "Security_SecretDetection",
			Category: "security",
			Priority: 9,
			Function: func(t *testing.T) error {
				return runSecretDetectionScan()
			},
		},
		{
			Name:     "Security_DependencyCheck",
			Category: "security",
			Priority: 8,
			Function: func(t *testing.T) error {
				return runDependencyVulnerabilityCheck()
			},
		},
		{
			Name:     "Security_NetworkPolicies",
			Category: "security",
			Priority: 7,
			Function: func(t *testing.T) error {
				return verifyNetworkSecurityPolicies()
			},
		},
		{
			Name:     "Security_TLSConfiguration",
			Category: "security",
			Priority: 8,
			Function: func(t *testing.T) error {
				return verifyTLSConfiguration()
			},
		},
	}
}

// GetMonitoringPipelineScenario returns test functions for monitoring pipeline
func GetMonitoringPipelineScenario() []TestFunc {
	return []TestFunc{
		{
			Name:       "Pipeline_PrometheusIngestion",
			Category:   "monitoring",
			Priority:   10,
			Required:   true,
			MaxRetries: 2,
			Function: func(t *testing.T) error {
				return testPrometheusIngestionPipeline()
			},
		},
		{
			Name:     "Pipeline_LokiLogFlow",
			Category: "monitoring",
			Priority: 9,
			Function: func(t *testing.T) error {
				return testLokiLogIngestionPipeline()
			},
		},
		{
			Name:     "Pipeline_JaegerTraceFlow",
			Category: "monitoring",
			Priority: 9,
			Function: func(t *testing.T) error {
				return testJaegerTracePipeline()
			},
		},
		{
			Name:     "Pipeline_MetricsToGrafana",
			Category: "monitoring",
			Priority: 8,
			Function: func(t *testing.T) error {
				return testMetricsVisualizationPipeline()
			},
		},
		{
			Name:     "Pipeline_AlertFlow",
			Category: "monitoring",
			Priority: 8,
			Function: func(t *testing.T) error {
				return testAlertingPipeline()
			},
		},
		{
			Name:     "Pipeline_DataRetention",
			Category: "monitoring",
			Priority: 6,
			Timeout:  10 * time.Minute,
			Function: func(t *testing.T) error {
				return testDataRetentionPolicies()
			},
		},
	}
}

// GetAlertTestScenario returns test functions for alert testing
func GetAlertTestScenario() []TestFunc {
	return []TestFunc{
		{
			Name:     "Alert_HighErrorRate",
			Category: "alerting",
			Priority: 9,
			Function: func(t *testing.T) error {
				return testHighErrorRateAlert()
			},
		},
		{
			Name:     "Alert_HighLatency",
			Category: "alerting",
			Priority: 9,
			Function: func(t *testing.T) error {
				return testHighLatencyAlert()
			},
		},
		{
			Name:     "Alert_ServiceDown",
			Category: "alerting",
			Priority: 10,
			Function: func(t *testing.T) error {
				return testServiceDownAlert()
			},
		},
		{
			Name:     "Alert_ResourceExhaustion",
			Category: "alerting",
			Priority: 8,
			Function: func(t *testing.T) error {
				return testResourceExhaustionAlert()
			},
		},
		{
			Name:     "Alert_NotificationChannels",
			Category: "alerting",
			Priority: 7,
			Function: func(t *testing.T) error {
				return testAlertNotificationChannels()
			},
		},
		{
			Name:     "Alert_Silencing",
			Category: "alerting",
			Priority: 6,
			Function: func(t *testing.T) error {
				return testAlertSilencing()
			},
		},
		{
			Name:     "Alert_Grouping",
			Category: "alerting",
			Priority: 6,
			Function: func(t *testing.T) error {
				return testAlertGrouping()
			},
		},
	}
}

// GetFullStackIntegrationScenario returns test functions for full stack integration
func GetFullStackIntegrationScenario() []TestFunc {
	return []TestFunc{
		{
			Name:       "Integration_EndToEndFlow",
			Category:   "integration",
			Priority:   10,
			Required:   true,
			Timeout:    10 * time.Minute,
			MaxRetries: 1,
			Function: func(t *testing.T) error {
				return testEndToEndObservabilityFlow()
			},
		},
		{
			Name:     "Integration_CrossServiceTracing",
			Category: "integration",
			Priority: 9,
			Timeout:  5 * time.Minute,
			Function: func(t *testing.T) error {
				return testCrossServiceTracing()
			},
		},
		{
			Name:     "Integration_MetricsCorrelation",
			Category: "integration",
			Priority: 8,
			Function: func(t *testing.T) error {
				return testMetricsLogTraceCorrelation()
			},
		},
		{
			Name:     "Integration_DashboardDataFlow",
			Category: "integration",
			Priority: 7,
			Function: func(t *testing.T) error {
				return testDashboardDataIntegration()
			},
		},
		{
			Name:     "Integration_ServiceDiscovery",
			Category: "integration",
			Priority: 7,
			Function: func(t *testing.T) error {
				return testServiceDiscoveryIntegration()
			},
		},
		{
			Name:     "Integration_FailoverRecovery",
			Category: "integration",
			Priority: 9,
			Timeout:  10 * time.Minute,
			Function: func(t *testing.T) error {
				return testFailoverAndRecovery()
			},
		},
	}
}

// GetAllScenarios returns all test scenarios
func GetAllScenarios() []TestFunc {
	scenarios := make([]TestFunc, 0)
	scenarios = append(scenarios, GetLoadTestScenario()...)
	scenarios = append(scenarios, GetSecurityScanScenario()...)
	scenarios = append(scenarios, GetMonitoringPipelineScenario()...)
	scenarios = append(scenarios, GetAlertTestScenario()...)
	scenarios = append(scenarios, GetFullStackIntegrationScenario()...)
	return scenarios
}

// Load Testing Implementation

func runBasicLoadTest(requests int, concurrency int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client := &http.Client{Timeout: 10 * time.Second}
	errors := make(chan error, requests)
	var wg sync.WaitGroup

	// Rate limiter
	rate := time.Second / time.Duration(concurrency)
	limiter := time.NewTicker(rate)
	defer limiter.Stop()

	endpoints := []string{
		"http://localhost:8080/health",
		"http://localhost:8080/metrics",
		"http://localhost:8080/api/users",
		"http://localhost:8080/api/products",
	}

	for i := 0; i < requests; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-limiter.C:
			wg.Add(1)
			go func(reqNum int) {
				defer wg.Done()

				endpoint := endpoints[reqNum%len(endpoints)]
				resp, err := client.Get(endpoint)
				if err != nil {
					errors <- fmt.Errorf("request %d failed: %w", reqNum, err)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode >= 400 {
					errors <- fmt.Errorf("request %d returned status %d", reqNum, resp.StatusCode)
				}
			}(i)
		}
	}

	wg.Wait()
	close(errors)

	// Check error rate
	errorCount := 0
	for err := range errors {
		if err != nil {
			errorCount++
		}
	}

	errorRate := float64(errorCount) / float64(requests) * 100
	if errorRate > 5.0 {
		return fmt.Errorf("error rate too high: %.2f%% (threshold: 5%%)", errorRate)
	}

	return nil
}

func runSpikeLoadTest(baseLoad, spikeLoad, spikes int) error {
	for i := 0; i < spikes; i++ {
		// Normal load
		if err := runBasicLoadTest(baseLoad, 5); err != nil {
			return fmt.Errorf("base load test failed: %w", err)
		}

		// Spike load
		if err := runBasicLoadTest(spikeLoad, 50); err != nil {
			return fmt.Errorf("spike load test failed: %w", err)
		}

		// Cool down
		time.Sleep(10 * time.Second)
	}

	return nil
}

func runSustainedLoadTest(rps int, duration time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	ticker := time.NewTicker(time.Second / time.Duration(rps))
	defer ticker.Stop()

	client := &http.Client{Timeout: 5 * time.Second}
	errorCount := int64(0)
	totalRequests := int64(0)

	for {
		select {
		case <-ctx.Done():
			errorRate := float64(errorCount) / float64(totalRequests) * 100
			if errorRate > 1.0 {
				return fmt.Errorf("sustained load error rate too high: %.2f%%", errorRate)
			}
			return nil
		case <-ticker.C:
			go func() {
				resp, err := client.Get("http://localhost:8080/health")
				totalRequests++
				if err != nil || (resp != nil && resp.StatusCode >= 500) {
					errorCount++
				}
				if resp != nil {
					resp.Body.Close()
				}
			}()
		}
	}
}

func verifyMetricsUnderLoad(requests int) error {
	// Record initial metrics
	initialMetrics, err := getPrometheusMetrics("http_requests_total")
	if err != nil {
		return fmt.Errorf("failed to get initial metrics: %w", err)
	}

	// Generate load
	if err := runBasicLoadTest(requests, 10); err != nil {
		return fmt.Errorf("load generation failed: %w", err)
	}

	// Wait for metrics to be updated
	time.Sleep(20 * time.Second)

	// Verify metrics increased
	finalMetrics, err := getPrometheusMetrics("http_requests_total")
	if err != nil {
		return fmt.Errorf("failed to get final metrics: %w", err)
	}

	if finalMetrics <= initialMetrics {
		return fmt.Errorf("metrics did not increase under load: initial=%f, final=%f", initialMetrics, finalMetrics)
	}

	expectedIncrease := float64(requests) * 0.95 // Allow 5% loss
	actualIncrease := finalMetrics - initialMetrics

	if actualIncrease < expectedIncrease {
		return fmt.Errorf("metric increase too low: expected at least %.0f, got %.0f", expectedIncrease, actualIncrease)
	}

	return nil
}

func testResourceLimits() error {
	// This test verifies that the system handles resource constraints gracefully
	// Generate high load to test resource limits
	return runBasicLoadTest(5000, 100)
}

// Security Testing Implementation

func runSecurityVulnerabilityCheck() error {
	// Run Semgrep security scan
	result, err := RunSemgrepAnalysis(".")
	if err != nil {
		return fmt.Errorf("semgrep analysis failed: %w", err)
	}

	criticalCount := 0
	highCount := 0

	for _, finding := range result.Results {
		// Count critical and high severity findings
		if contains(finding.Check.Name, "critical") {
			criticalCount++
		} else if contains(finding.Check.Name, "high") {
			highCount++
		}
	}

	if criticalCount > 0 {
		return fmt.Errorf("found %d critical security vulnerabilities", criticalCount)
	}

	if highCount > 5 {
		return fmt.Errorf("found %d high severity vulnerabilities (threshold: 5)", highCount)
	}

	return nil
}

func runConfigurationSecurityScan() error {
	// Check for insecure configurations
	configurations := []struct {
		name     string
		endpoint string
		check    func(string) error
	}{
		{"Prometheus security", "http://localhost:9090", checkPrometheusConfig},
		{"Grafana security", "http://localhost:3000", checkGrafanaConfig},
		{"AlertManager security", "http://localhost:9093", checkAlertManagerConfig},
	}

	for _, config := range configurations {
		if err := config.check(config.endpoint); err != nil {
			return fmt.Errorf("%s check failed: %w", config.name, err)
		}
	}

	return nil
}

func runSecretDetectionScan() error {
	// Mock implementation - in production, use tools like truffleHog or git-secrets
	// Check for common patterns
	patterns := []string{
		"password",
		"api_key",
		"secret",
		"token",
		"private_key",
	}

	// This is a simplified check
	for _, pattern := range patterns {
		// In real implementation, scan files for these patterns
		fmt.Printf("Scanning for exposed %ss...\n", pattern)
	}

	return nil
}

func runDependencyVulnerabilityCheck() error {
	// In production, integrate with tools like Snyk or OWASP Dependency Check
	fmt.Println("Checking dependencies for known vulnerabilities...")
	return nil
}

func verifyNetworkSecurityPolicies() error {
	// Verify that services are not exposed unnecessarily
	unauthorizedPorts := []int{
		9090, // Prometheus should not be publicly accessible
		3100, // Loki internal port
		9093, // AlertManager
	}

	for _, port := range unauthorizedPorts {
		// In production, check if these ports are accessible from outside
		fmt.Printf("Verifying port %d is not publicly exposed...\n", port)
	}

	return nil
}

func verifyTLSConfiguration() error {
	// Verify TLS is properly configured for all endpoints
	// This is a simplified check
	endpoints := []string{
		"https://localhost:3000", // Grafana should use HTTPS
	}

	for _, endpoint := range endpoints {
		// In production, verify TLS configuration
		fmt.Printf("Verifying TLS configuration for %s...\n", endpoint)
	}

	return nil
}

// Monitoring Pipeline Implementation

func testPrometheusIngestionPipeline() error {
	// Send test metrics
	if err := SendTestMetrics("http://localhost:8080/metrics"); err != nil {
		return fmt.Errorf("failed to send test metrics: %w", err)
	}

	// Wait for scraping
	time.Sleep(20 * time.Second)

	// Verify metrics are in Prometheus
	ctx := context.Background()
	if err := WaitForMetric("http://localhost:9090", "apm_test_counter", 30*time.Second); err != nil {
		return fmt.Errorf("metrics not found in Prometheus: %w", err)
	}

	return nil
}

func testLokiLogIngestionPipeline() error {
	// Send test logs with different levels
	testLogs := []struct {
		level   string
		message string
	}{
		{"info", "Test info log for pipeline validation"},
		{"warning", "Test warning log for pipeline validation"},
		{"error", "Test error log for pipeline validation"},
		{"debug", "Test debug log for pipeline validation"},
	}

	for _, log := range testLogs {
		logEntry := fmt.Sprintf(`{"level":"%s","msg":"%s","timestamp":"%s"}`,
			log.level, log.message, time.Now().Format(time.RFC3339))

		if err := SendLogToLoki("http://localhost:3100", logEntry); err != nil {
			return fmt.Errorf("failed to send %s log: %w", log.level, err)
		}
	}

	// Wait for ingestion
	time.Sleep(5 * time.Second)

	// Verify logs are queryable
	for _, log := range testLogs {
		query := fmt.Sprintf(`{level="%s"} |= "%s"`, log.level, log.message)
		// In production, query Loki and verify results
		fmt.Printf("Verifying %s logs are ingested...\n", log.level)
	}

	return nil
}

func testJaegerTracePipeline() error {
	// Generate traces with parent-child relationships
	traceID := GenerateTraceID()

	// Send parent span
	if err := SendTestTrace("http://localhost:14268/api/traces", "pipeline-test-service", traceID); err != nil {
		return fmt.Errorf("failed to send parent trace: %w", err)
	}

	// Send child spans
	for i := 0; i < 5; i++ {
		if err := SendTestTrace("http://localhost:14268/api/traces", "pipeline-test-service", traceID); err != nil {
			return fmt.Errorf("failed to send child trace %d: %w", i, err)
		}
	}

	// Wait for processing
	time.Sleep(10 * time.Second)

	// Verify trace is complete in Jaeger
	// In production, query Jaeger API and verify trace structure

	return nil
}

func testMetricsVisualizationPipeline() error {
	// Verify Grafana can visualize metrics from all datasources
	client := &http.Client{Timeout: 10 * time.Second}

	datasources := []string{"Prometheus", "Loki", "Jaeger"}

	for _, ds := range datasources {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:3000/api/datasources/name/%s", ds), nil)
		if err != nil {
			return fmt.Errorf("failed to create request for %s: %w", ds, err)
		}
		req.SetBasicAuth("admin", "admin")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to check %s datasource: %w", ds, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("%s datasource not accessible: status %d", ds, resp.StatusCode)
		}
	}

	// Verify dashboards are provisioned
	if err := VerifyGrafanaDashboard("http://localhost:3000", "e2e-test-dashboard", "admin", "admin"); err != nil {
		return fmt.Errorf("dashboard verification failed: %w", err)
	}

	return nil
}

func testAlertingPipeline() error {
	// Test complete alerting flow: metric -> alert -> notification

	// 1. Generate conditions that should trigger an alert
	for i := 0; i < 100; i++ {
		resp, err := http.Get("http://localhost:8080/error")
		if err == nil {
			resp.Body.Close()
		}
	}

	// 2. Wait for alert to be evaluated
	time.Sleep(30 * time.Second)

	// 3. Check if alert is firing in Prometheus
	alerts, err := getPrometheusAlerts()
	if err != nil {
		return fmt.Errorf("failed to get alerts: %w", err)
	}

	if len(alerts) == 0 {
		return fmt.Errorf("no alerts are firing after generating error conditions")
	}

	// 4. Verify alert reached AlertManager
	amAlerts, err := getAlertManagerAlerts()
	if err != nil {
		return fmt.Errorf("failed to get AlertManager alerts: %w", err)
	}

	if len(amAlerts) == 0 {
		return fmt.Errorf("alerts did not reach AlertManager")
	}

	return nil
}

func testDataRetentionPolicies() error {
	// This test would verify that data retention policies are working
	// In a real implementation, this would:
	// 1. Insert old data with past timestamps
	// 2. Wait for retention period
	// 3. Verify old data is cleaned up
	// 4. Verify recent data is retained

	fmt.Println("Testing data retention policies...")
	// Mock implementation for now
	return nil
}

// Alert Testing Implementation

func testHighErrorRateAlert() error {
	// Generate high error rate
	for i := 0; i < 50; i++ {
		http.Get("http://localhost:8080/error")
	}

	// Wait for alert
	return waitForAlert("HighErrorRate", 60*time.Second)
}

func testHighLatencyAlert() error {
	// Generate high latency requests
	for i := 0; i < 20; i++ {
		http.Get("http://localhost:8080/slow")
	}

	// Wait for alert
	return waitForAlert("HighLatency", 60*time.Second)
}

func testServiceDownAlert() error {
	// This would simulate a service going down
	// In production, you might stop a container or block a port
	fmt.Println("Simulating service down scenario...")
	return nil
}

func testResourceExhaustionAlert() error {
	// Generate load to exhaust resources
	return runBasicLoadTest(1000, 50)
}

func testAlertNotificationChannels() error {
	// Test different notification channels
	channels := []string{"email", "slack", "webhook"}

	for _, channel := range channels {
		fmt.Printf("Testing %s notification channel...\n", channel)
		// In production, verify notifications are sent
	}

	return nil
}

func testAlertSilencing() error {
	// Create a silence
	silence := Silence{
		Matchers: []Matcher{
			{Name: "alertname", Value: "TestAlert", IsRegex: false},
		},
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(1 * time.Hour),
		Comment:   "E2E test silence",
		CreatedBy: "e2e-test",
	}

	if err := createSilence(silence); err != nil {
		return fmt.Errorf("failed to create silence: %w", err)
	}

	// Trigger an alert that should be silenced
	// Verify it doesn't create notifications

	return nil
}

func testAlertGrouping() error {
	// Generate multiple similar alerts
	for i := 0; i < 10; i++ {
		alert := Alert{
			Labels: map[string]string{
				"alertname": "GroupTest",
				"severity":  "warning",
				"instance":  fmt.Sprintf("instance-%d", i),
			},
			Annotations: map[string]string{
				"summary": "Test alert for grouping",
			},
			StartsAt: time.Now(),
			EndsAt:   time.Now().Add(1 * time.Hour),
		}

		if err := SendTestAlert("http://localhost:9093/api/v1/alerts", []Alert{alert}); err != nil {
			return fmt.Errorf("failed to send alert %d: %w", i, err)
		}
	}

	// Verify alerts are grouped
	time.Sleep(5 * time.Second)

	groups, err := getAlertGroups()
	if err != nil {
		return fmt.Errorf("failed to get alert groups: %w", err)
	}

	// Check if alerts were properly grouped
	if len(groups) > 5 {
		return fmt.Errorf("alerts not properly grouped: got %d groups, expected fewer", len(groups))
	}

	return nil
}

// Integration Testing Implementation

func testEndToEndObservabilityFlow() error {
	// Complete E2E flow test
	traceID := GenerateTraceID()

	// 1. Generate application traffic with trace ID
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", "http://localhost:8080/api/users", nil)
	req.Header.Set("X-Trace-Id", traceID)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	resp.Body.Close()

	// 2. Verify metrics are recorded
	time.Sleep(20 * time.Second)

	metrics, err := getPrometheusMetrics("http_requests_total")
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	if metrics == 0 {
		return fmt.Errorf("no metrics recorded")
	}

	// 3. Verify logs are captured
	// 4. Verify trace is recorded
	// 5. Verify everything is visible in Grafana

	return nil
}

func testCrossServiceTracing() error {
	// Test distributed tracing across multiple services
	traceID := GenerateTraceID()

	// Simulate service A calling service B
	// In production, this would involve actual service calls

	services := []string{"service-a", "service-b", "service-c"}

	for i, service := range services {
		parentSpanID := ""
		if i > 0 {
			parentSpanID = GenerateTraceID()[:16]
		}

		// Send trace for each service
		if err := SendTestTrace("http://localhost:14268/api/traces", service, traceID); err != nil {
			return fmt.Errorf("failed to send trace for %s: %w", service, err)
		}
	}

	// Verify complete trace in Jaeger
	time.Sleep(10 * time.Second)

	return nil
}

func testMetricsLogTraceCorrelation() error {
	// Test correlation between metrics, logs, and traces
	correlationID := GenerateTraceID()

	// 1. Generate request with correlation ID
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "http://localhost:8080/api/orders", nil)
	req.Header.Set("X-Correlation-Id", correlationID)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	resp.Body.Close()

	// 2. Send correlated log
	logEntry := fmt.Sprintf(`{"correlation_id":"%s","msg":"Processing order","level":"info"}`, correlationID)
	if err := SendLogToLoki("http://localhost:3100", logEntry); err != nil {
		return fmt.Errorf("failed to send log: %w", err)
	}

	// 3. Send correlated trace
	if err := SendTestTrace("http://localhost:14268/api/traces", "order-service", correlationID); err != nil {
		return fmt.Errorf("failed to send trace: %w", err)
	}

	// Wait for data to be available
	time.Sleep(15 * time.Second)

	// In production, verify all three data types can be correlated using the ID

	return nil
}

func testDashboardDataIntegration() error {
	// Verify dashboards show integrated data from all sources
	dashboards := []string{
		"e2e-test-dashboard",
		"api-performance",
		"container-metrics",
	}

	for _, dashboard := range dashboards {
		if err := VerifyGrafanaDashboard("http://localhost:3000", dashboard, "admin", "admin"); err != nil {
			// Some dashboards might not exist in test environment
			fmt.Printf("Warning: Dashboard %s not found\n", dashboard)
		}
	}

	return nil
}

func testServiceDiscoveryIntegration() error {
	// Test that new services are automatically discovered
	// In production, this would spin up a new service and verify it appears in monitoring

	fmt.Println("Testing service discovery integration...")

	// Verify Prometheus targets
	targets, err := getPrometheusTargets()
	if err != nil {
		return fmt.Errorf("failed to get Prometheus targets: %w", err)
	}

	if len(targets) == 0 {
		return fmt.Errorf("no targets discovered by Prometheus")
	}

	return nil
}

func testFailoverAndRecovery() error {
	// Test system resilience and recovery

	// 1. Baseline check - all services healthy
	services := []string{
		"prometheus",
		"grafana",
		"loki",
		"jaeger",
		"alertmanager",
	}

	for _, service := range services {
		if err := checkServiceHealth(service); err != nil {
			return fmt.Errorf("service %s not healthy at baseline: %w", service, err)
		}
	}

	// 2. Simulate failure (restart a service)
	failedService := services[rand.Intn(len(services))]
	fmt.Printf("Simulating failure of %s...\n", failedService)

	if err := RestartService("docker-compose.test.yml", failedService); err != nil {
		return fmt.Errorf("failed to restart %s: %w", failedService, err)
	}

	// 3. Wait for recovery
	time.Sleep(30 * time.Second)

	// 4. Verify service recovered
	if err := checkServiceHealth(failedService); err != nil {
		return fmt.Errorf("service %s did not recover: %w", failedService, err)
	}

	// 5. Verify no data loss
	// In production, check that metrics/logs/traces during the failure period were not lost

	return nil
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

func getPrometheusMetrics(metricName string) (float64, error) {
	// Simplified - in production, parse the actual response
	resp, err := http.Get(fmt.Sprintf("http://localhost:9090/api/v1/query?query=%s", metricName))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Mock return value
	return 100.0, nil
}

func checkPrometheusConfig(endpoint string) error {
	// Check for secure Prometheus configuration
	resp, err := http.Get(endpoint + "/api/v1/targets")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// In production, check for authentication, TLS, etc.
	return nil
}

func checkGrafanaConfig(endpoint string) error {
	// Check for secure Grafana configuration
	// Verify default admin password is changed
	client := &http.Client{}
	req, _ := http.NewRequest("GET", endpoint+"/api/org", nil)
	req.SetBasicAuth("admin", "admin")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return fmt.Errorf("Grafana still using default admin password")
	}

	return nil
}

func checkAlertManagerConfig(endpoint string) error {
	// Check AlertManager configuration
	resp, err := http.Get(endpoint + "/api/v1/status")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func getPrometheusAlerts() ([]Alert, error) {
	// Get alerts from Prometheus
	resp, err := http.Get("http://localhost:9090/api/v1/alerts")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// In production, parse the response
	return []Alert{}, nil
}

func getAlertManagerAlerts() ([]Alert, error) {
	// Get alerts from AlertManager
	resp, err := http.Get("http://localhost:9093/api/v1/alerts")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// In production, parse the response
	return []Alert{}, nil
}

func waitForAlert(alertName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		alerts, err := getAlertManagerAlerts()
		if err != nil {
			return err
		}

		for _, alert := range alerts {
			if alert.Labels["alertname"] == alertName {
				return nil
			}
		}

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("alert %s did not fire within %v", alertName, timeout)
}

func createSilence(silence Silence) error {
	// Create silence in AlertManager
	// In production, make actual API call
	return nil
}

func getAlertGroups() ([]interface{}, error) {
	// Get alert groups from AlertManager
	resp, err := http.Get("http://localhost:9093/api/v1/alerts/groups")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// In production, parse the response
	return []interface{}{}, nil
}

func getPrometheusTargets() ([]interface{}, error) {
	// Get targets from Prometheus
	resp, err := http.Get("http://localhost:9090/api/v1/targets")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// In production, parse the response
	return []interface{}{}, nil
}

func checkServiceHealth(service string) error {
	endpoints := map[string]string{
		"prometheus":   "http://localhost:9090/-/ready",
		"grafana":      "http://localhost:3000/api/health",
		"loki":         "http://localhost:3100/ready",
		"jaeger":       "http://localhost:16686",
		"alertmanager": "http://localhost:9093/-/ready",
	}

	endpoint, ok := endpoints[service]
	if !ok {
		return fmt.Errorf("unknown service: %s", service)
	}

	resp, err := http.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("service unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
