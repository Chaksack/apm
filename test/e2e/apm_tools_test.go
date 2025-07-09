// Package e2e provides end-to-end tests for the APM stack
package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrometheusMetricsCollection tests Prometheus metrics collection
func TestPrometheusMetricsCollection(t *testing.T) {
	ctx := context.Background()

	// Wait for Prometheus to be ready
	err := WaitForService(ctx, "http://localhost:9090/-/ready", 30*time.Second)
	require.NoError(t, err, "Prometheus should be ready")

	// Query Prometheus for up metrics
	resp, err := http.Get("http://localhost:9090/api/v1/query?query=up")
	require.NoError(t, err, "Should be able to query Prometheus")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Should be able to read response body")

	var result PrometheusQueryResponse
	err = json.Unmarshal(body, &result)
	require.NoError(t, err, "Should be able to parse Prometheus response")

	assert.Equal(t, "success", result.Status, "Query should be successful")
	assert.NotEmpty(t, result.Data.Result, "Should have some up metrics")

	// Test custom metrics
	t.Run("CustomMetrics", func(t *testing.T) {
		// Send test metrics
		err = SendTestMetrics("http://localhost:8080/metrics")
		require.NoError(t, err, "Should be able to send test metrics")

		// Wait for metrics to be scraped
		time.Sleep(15 * time.Second)

		// Query for custom metrics
		resp, err := http.Get("http://localhost:9090/api/v1/query?query=apm_test_counter")
		require.NoError(t, err, "Should be able to query custom metrics")
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var customResult PrometheusQueryResponse
		err = json.Unmarshal(body, &customResult)
		require.NoError(t, err)

		assert.Equal(t, "success", customResult.Status)
		assert.NotEmpty(t, customResult.Data.Result, "Should have custom metrics")
	})
}

// TestGrafanaDashboardConnectivity tests Grafana connectivity and datasources
func TestGrafanaDashboardConnectivity(t *testing.T) {
	ctx := context.Background()

	// Wait for Grafana to be ready
	err := WaitForService(ctx, "http://localhost:3000/api/health", 30*time.Second)
	require.NoError(t, err, "Grafana should be ready")

	// Test Grafana API
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:3000/api/datasources", nil)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "admin")

	resp, err := client.Do(req)
	require.NoError(t, err, "Should be able to access Grafana API")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var datasources []GrafanaDatasource
	err = json.Unmarshal(body, &datasources)
	require.NoError(t, err, "Should be able to parse datasources")

	// Verify expected datasources
	expectedDatasources := []string{"Prometheus", "Loki", "Jaeger", "AlertManager"}
	for _, expected := range expectedDatasources {
		found := false
		for _, ds := range datasources {
			if ds.Name == expected {
				found = true
				assert.Equal(t, "OK", ds.Status, fmt.Sprintf("%s datasource should be healthy", expected))
				break
			}
		}
		assert.True(t, found, fmt.Sprintf("Should have %s datasource", expected))
	}

	// Test dashboard provisioning
	t.Run("DashboardProvisioning", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://localhost:3000/api/search?type=dash-db", nil)
		require.NoError(t, err)
		req.SetBasicAuth("admin", "admin")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var dashboards []GrafanaDashboard
		err = json.Unmarshal(body, &dashboards)
		require.NoError(t, err)

		assert.NotEmpty(t, dashboards, "Should have provisioned dashboards")
	})
}

// TestLokiLogAggregation tests Loki log aggregation
func TestLokiLogAggregation(t *testing.T) {
	ctx := context.Background()

	// Wait for Loki to be ready
	err := WaitForService(ctx, "http://localhost:3100/ready", 30*time.Second)
	require.NoError(t, err, "Loki should be ready")

	// Send test logs
	testLogs := []string{
		`{"level":"info","msg":"Test log 1","app":"apm-test"}`,
		`{"level":"error","msg":"Test error log","app":"apm-test"}`,
		`{"level":"debug","msg":"Test debug log","app":"apm-test"}`,
	}

	for _, log := range testLogs {
		err = SendLogToLoki("http://localhost:3100", log)
		require.NoError(t, err, "Should be able to send logs to Loki")
	}

	// Wait for logs to be ingested
	time.Sleep(5 * time.Second)

	// Query logs
	query := `{app="apm-test"}`
	resp, err := http.Get(fmt.Sprintf("http://localhost:3100/loki/api/v1/query_range?query=%s&limit=100", query))
	require.NoError(t, err, "Should be able to query Loki")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result LokiQueryResponse
	err = json.Unmarshal(body, &result)
	require.NoError(t, err, "Should be able to parse Loki response")

	assert.Equal(t, "success", result.Status)
	assert.NotEmpty(t, result.Data.Result, "Should have log entries")

	// Test log levels
	t.Run("LogLevels", func(t *testing.T) {
		errorQuery := `{app="apm-test",level="error"}`
		resp, err := http.Get(fmt.Sprintf("http://localhost:3100/loki/api/v1/query_range?query=%s", errorQuery))
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var errorResult LokiQueryResponse
		err = json.Unmarshal(body, &errorResult)
		require.NoError(t, err)

		assert.Equal(t, "success", errorResult.Status)
		// Verify we have error logs
		foundError := false
		for _, stream := range errorResult.Data.Result {
			for _, value := range stream.Values {
				if strings.Contains(value[1], "error") {
					foundError = true
					break
				}
			}
		}
		assert.True(t, foundError, "Should find error logs")
	})
}

// TestJaegerTraceCollection tests Jaeger trace collection
func TestJaegerTraceCollection(t *testing.T) {
	ctx := context.Background()

	// Wait for Jaeger to be ready
	err := WaitForService(ctx, "http://localhost:16686", 30*time.Second)
	require.NoError(t, err, "Jaeger should be ready")

	// Send test traces
	testService := "apm-test-service"
	traceID := GenerateTraceID()

	err = SendTestTrace("http://localhost:14268/api/traces", testService, traceID)
	require.NoError(t, err, "Should be able to send traces to Jaeger")

	// Wait for traces to be processed
	time.Sleep(5 * time.Second)

	// Query traces
	resp, err := http.Get(fmt.Sprintf("http://localhost:16686/api/traces?service=%s", testService))
	require.NoError(t, err, "Should be able to query traces")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result JaegerTraceResponse
	err = json.Unmarshal(body, &result)
	require.NoError(t, err, "Should be able to parse Jaeger response")

	assert.NotEmpty(t, result.Data, "Should have trace data")

	// Test trace search
	t.Run("TraceSearch", func(t *testing.T) {
		// Search by operation name
		resp, err := http.Get(fmt.Sprintf("http://localhost:16686/api/traces?service=%s&operation=test-operation", testService))
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var searchResult JaegerTraceResponse
		err = json.Unmarshal(body, &searchResult)
		require.NoError(t, err)

		// Verify we can find our test traces
		found := false
		for _, trace := range searchResult.Data {
			if trace.TraceID == traceID {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find our test trace")
	})
}

// TestAlertManagerNotifications tests AlertManager notifications
func TestAlertManagerNotifications(t *testing.T) {
	ctx := context.Background()

	// Wait for AlertManager to be ready
	err := WaitForService(ctx, "http://localhost:9093/-/ready", 30*time.Second)
	require.NoError(t, err, "AlertManager should be ready")

	// Send test alert
	alert := Alert{
		Labels: map[string]string{
			"alertname": "TestAlert",
			"severity":  "warning",
			"service":   "apm-test",
		},
		Annotations: map[string]string{
			"summary":     "Test alert for e2e testing",
			"description": "This is a test alert to verify AlertManager functionality",
		},
		StartsAt: time.Now(),
		EndsAt:   time.Now().Add(1 * time.Hour),
	}

	err = SendTestAlert("http://localhost:9093/api/v1/alerts", []Alert{alert})
	require.NoError(t, err, "Should be able to send alerts to AlertManager")

	// Wait for alert to be processed
	time.Sleep(3 * time.Second)

	// Query alerts
	resp, err := http.Get("http://localhost:9093/api/v1/alerts")
	require.NoError(t, err, "Should be able to query alerts")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result AlertManagerResponse
	err = json.Unmarshal(body, &result)
	require.NoError(t, err, "Should be able to parse AlertManager response")

	// Verify our test alert exists
	found := false
	for _, a := range result.Data {
		if a.Labels["alertname"] == "TestAlert" {
			found = true
			assert.Equal(t, "warning", a.Labels["severity"])
			assert.Equal(t, "apm-test", a.Labels["service"])
			break
		}
	}
	assert.True(t, found, "Should find our test alert")

	// Test silences
	t.Run("Silences", func(t *testing.T) {
		// Create a silence
		silence := Silence{
			Matchers: []Matcher{
				{
					Name:    "alertname",
					Value:   "TestAlert",
					IsRegex: false,
				},
			},
			StartsAt:  time.Now(),
			EndsAt:    time.Now().Add(1 * time.Hour),
			Comment:   "Test silence for e2e testing",
			CreatedBy: "e2e-test",
		}

		silenceJSON, err := json.Marshal(silence)
		require.NoError(t, err)

		resp, err := http.Post("http://localhost:9093/api/v1/silences", "application/json", strings.NewReader(string(silenceJSON)))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should be able to create silence")
	})
}

// TestSemgrepSecurityAnalysis tests Semgrep security analysis
func TestSemgrepSecurityAnalysis(t *testing.T) {
	// Create a test file with security issues
	testFile := "/tmp/test-security.go"
	testCode := `package main

import (
	"database/sql"
	"fmt"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	// SQL injection vulnerability
	userInput := r.URL.Query().Get("id")
	query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", userInput)
	
	db, _ := sql.Open("mysql", "user:password@/dbname")
	rows, _ := db.Query(query)
	defer rows.Close()
	
	// Hardcoded credentials
	password := "admin123"
	fmt.Fprintf(w, "Password: %s", password)
}
`

	err := CreateTestFile(testFile, testCode)
	require.NoError(t, err, "Should be able to create test file")
	defer RemoveTestFile(testFile)

	// Run Semgrep analysis
	results, err := RunSemgrepAnalysis(testFile)
	require.NoError(t, err, "Should be able to run Semgrep analysis")

	// Verify security issues were found
	assert.NotEmpty(t, results.Results, "Should find security issues")

	// Check for specific vulnerabilities
	foundSQLInjection := false
	foundHardcodedCreds := false

	for _, result := range results.Results {
		if strings.Contains(result.Check.Name, "sql") || strings.Contains(result.Check.Message, "SQL") {
			foundSQLInjection = true
		}
		if strings.Contains(result.Check.Name, "hardcoded") || strings.Contains(result.Check.Message, "credential") {
			foundHardcodedCreds = true
		}
	}

	assert.True(t, foundSQLInjection, "Should detect SQL injection vulnerability")
	assert.True(t, foundHardcodedCreds, "Should detect hardcoded credentials")
}

// TestHealthCheckEndpoints tests health check endpoints for all services
func TestHealthCheckEndpoints(t *testing.T) {
	healthEndpoints := []struct {
		name     string
		endpoint string
		timeout  time.Duration
	}{
		{"Prometheus", "http://localhost:9090/-/ready", 30 * time.Second},
		{"Grafana", "http://localhost:3000/api/health", 30 * time.Second},
		{"Loki", "http://localhost:3100/ready", 30 * time.Second},
		{"Jaeger", "http://localhost:16686", 30 * time.Second},
		{"AlertManager", "http://localhost:9093/-/ready", 30 * time.Second},
		{"Application", "http://localhost:8080/health", 30 * time.Second},
	}

	ctx := context.Background()

	for _, endpoint := range healthEndpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			err := WaitForService(ctx, endpoint.endpoint, endpoint.timeout)
			assert.NoError(t, err, fmt.Sprintf("%s should be healthy", endpoint.name))

			// Additional health checks
			resp, err := http.Get(endpoint.endpoint)
			if err == nil {
				defer resp.Body.Close()
				assert.True(t, resp.StatusCode < 400, fmt.Sprintf("%s should return success status", endpoint.name))
			}
		})
	}
}

// TestIntegrationBetweenServices tests integration between all APM services
func TestIntegrationBetweenServices(t *testing.T) {
	ctx := context.Background()

	// Ensure all services are ready
	services := []string{
		"http://localhost:9090/-/ready",    // Prometheus
		"http://localhost:3000/api/health", // Grafana
		"http://localhost:3100/ready",      // Loki
		"http://localhost:16686",           // Jaeger
		"http://localhost:9093/-/ready",    // AlertManager
	}

	for _, service := range services {
		err := WaitForService(ctx, service, 30*time.Second)
		require.NoError(t, err, fmt.Sprintf("Service %s should be ready", service))
	}

	// Test complete flow: Generate load -> Collect metrics -> Trigger alerts -> View in Grafana
	t.Run("CompleteObservabilityFlow", func(t *testing.T) {
		// 1. Generate application load with traces
		err := GenerateApplicationLoad("http://localhost:8080", 100, true)
		require.NoError(t, err, "Should be able to generate load")

		// 2. Wait for metrics to be collected
		time.Sleep(20 * time.Second)

		// 3. Verify metrics in Prometheus
		resp, err := http.Get("http://localhost:9090/api/v1/query?query=http_requests_total")
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var metricsResult PrometheusQueryResponse
		err = json.Unmarshal(body, &metricsResult)
		require.NoError(t, err)

		assert.Equal(t, "success", metricsResult.Status)
		assert.NotEmpty(t, metricsResult.Data.Result, "Should have HTTP request metrics")

		// 4. Verify logs in Loki
		logsResp, err := http.Get(`http://localhost:3100/loki/api/v1/query_range?query={job="apm-app"}`)
		require.NoError(t, err)
		defer logsResp.Body.Close()

		logsBody, err := io.ReadAll(logsResp.Body)
		require.NoError(t, err)

		var logsResult LokiQueryResponse
		err = json.Unmarshal(logsBody, &logsResult)
		require.NoError(t, err)

		assert.Equal(t, "success", logsResult.Status)

		// 5. Verify traces in Jaeger
		tracesResp, err := http.Get("http://localhost:16686/api/traces?service=apm-app&limit=10")
		require.NoError(t, err)
		defer tracesResp.Body.Close()

		tracesBody, err := io.ReadAll(tracesResp.Body)
		require.NoError(t, err)

		var tracesResult JaegerTraceResponse
		err = json.Unmarshal(tracesBody, &tracesResult)
		require.NoError(t, err)

		assert.NotEmpty(t, tracesResult.Data, "Should have traces")

		// 6. Verify Grafana can query all datasources
		client := &http.Client{}
		datasources := []string{"Prometheus", "Loki", "Jaeger"}

		for _, ds := range datasources {
			req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:3000/api/datasources/name/%s", ds), nil)
			require.NoError(t, err)
			req.SetBasicAuth("admin", "admin")

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode, fmt.Sprintf("Grafana should be able to access %s datasource", ds))
		}
	})
}

// Response types
type PrometheusQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type GrafanaDatasource struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	URL       string `json:"url"`
	Status    string `json:"status"`
	IsDefault bool   `json:"isDefault"`
}

type GrafanaDashboard struct {
	ID    int    `json:"id"`
	UID   string `json:"uid"`
	Title string `json:"title"`
	URI   string `json:"uri"`
	URL   string `json:"url"`
	Type  string `json:"type"`
}

type LokiQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Stream map[string]string `json:"stream"`
			Values [][]string        `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

type JaegerTraceResponse struct {
	Data []struct {
		TraceID   string `json:"traceID"`
		Processes map[string]struct {
			ServiceName string `json:"serviceName"`
		} `json:"processes"`
		Spans []struct {
			TraceID       string `json:"traceID"`
			SpanID        string `json:"spanID"`
			OperationName string `json:"operationName"`
		} `json:"spans"`
	} `json:"data"`
}

type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
}

type AlertManagerResponse struct {
	Status string  `json:"status"`
	Data   []Alert `json:"data"`
}

type Silence struct {
	Matchers  []Matcher `json:"matchers"`
	StartsAt  time.Time `json:"startsAt"`
	EndsAt    time.Time `json:"endsAt"`
	CreatedBy string    `json:"createdBy"`
	Comment   string    `json:"comment"`
}

type Matcher struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	IsRegex bool   `json:"isRegex"`
}

type SemgrepResult struct {
	Results []struct {
		Check struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Message string `json:"message"`
		} `json:"check_id"`
		Path  string `json:"path"`
		Start struct {
			Line   int `json:"line"`
			Column int `json:"col"`
		} `json:"start"`
	} `json:"results"`
}
