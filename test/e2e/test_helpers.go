// Package e2e provides end-to-end tests for the APM stack
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// WaitForService waits for a service to be ready by checking its health endpoint
func WaitForService(ctx context.Context, endpoint string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 5 * time.Second}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			resp, err := client.Get(endpoint)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode < 400 {
					return nil
				}
			}
			time.Sleep(2 * time.Second)
		}
	}
	return fmt.Errorf("service at %s did not become ready within %v", endpoint, timeout)
}

// StartServices starts all APM services using docker-compose
func StartServices(composeFile string) error {
	cmd := exec.Command("docker-compose", "-f", composeFile, "up", "-d")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start services: %v\nOutput: %s", err, output)
	}
	return nil
}

// StopServices stops all APM services
func StopServices(composeFile string) error {
	cmd := exec.Command("docker-compose", "-f", composeFile, "down", "-v")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop services: %v\nOutput: %s", err, output)
	}
	return nil
}

// SendTestMetrics sends test metrics to the application endpoint
func SendTestMetrics(endpoint string) error {
	// Generate some test metrics
	for i := 0; i < 10; i++ {
		resp, err := http.Get(endpoint)
		if err != nil {
			return fmt.Errorf("failed to send metrics: %v", err)
		}
		resp.Body.Close()
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

// SendLogToLoki sends a log entry to Loki
func SendLogToLoki(lokiURL, logEntry string) error {
	now := time.Now().UnixNano()
	payload := fmt.Sprintf(`{
		"streams": [
			{
				"stream": {
					"app": "apm-test",
					"job": "e2e-test"
				},
				"values": [
					["%d", %s]
				]
			}
		]
	}`, now, logEntry)

	resp, err := http.Post(lokiURL+"/loki/api/v1/push", "application/json", strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to send log to Loki: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}
	return nil
}

// GenerateTraceID generates a random trace ID
func GenerateTraceID() string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// SendTestTrace sends a test trace to Jaeger
func SendTestTrace(jaegerURL, serviceName, traceID string) error {
	spans := []map[string]interface{}{
		{
			"traceID":       traceID,
			"spanID":        GenerateTraceID()[:16],
			"operationName": "test-operation",
			"startTime":     time.Now().UnixMicro(),
			"duration":      rand.Intn(1000) + 100,
			"tags": []map[string]interface{}{
				{"key": "http.method", "type": "string", "value": "GET"},
				{"key": "http.status_code", "type": "int64", "value": 200},
				{"key": "span.kind", "type": "string", "value": "server"},
			},
			"process": map[string]interface{}{
				"serviceName": serviceName,
				"tags": []map[string]interface{}{
					{"key": "hostname", "type": "string", "value": "test-host"},
				},
			},
		},
	}

	payload, err := json.Marshal(map[string]interface{}{
		"data": []map[string]interface{}{
			{
				"traceID":   traceID,
				"spans":     spans,
				"processes": map[string]interface{}{},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal trace: %v", err)
	}

	resp, err := http.Post(jaegerURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to send trace to Jaeger: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}
	return nil
}

// SendTestAlert sends a test alert to AlertManager
func SendTestAlert(alertmanagerURL string, alerts []Alert) error {
	payload, err := json.Marshal(alerts)
	if err != nil {
		return fmt.Errorf("failed to marshal alerts: %v", err)
	}

	resp, err := http.Post(alertmanagerURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to send alert to AlertManager: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}
	return nil
}

// RunSemgrepAnalysis runs Semgrep analysis on a file
func RunSemgrepAnalysis(filePath string) (*SemgrepResult, error) {
	cmd := exec.Command("semgrep", "--config=auto", "--json", filePath)
	output, err := cmd.Output()
	if err != nil {
		// Semgrep returns non-zero exit code when it finds issues
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// This is expected when findings are present
		} else {
			return nil, fmt.Errorf("failed to run semgrep: %v", err)
		}
	}

	var result SemgrepResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse semgrep output: %v", err)
	}

	return &result, nil
}

// CreateTestFile creates a temporary test file
func CreateTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// RemoveTestFile removes a test file
func RemoveTestFile(path string) error {
	return os.Remove(path)
}

// GenerateApplicationLoad generates load on the application
func GenerateApplicationLoad(appURL string, requests int, withTracing bool) error {
	client := &http.Client{Timeout: 10 * time.Second}

	for i := 0; i < requests; i++ {
		// Create different types of requests
		endpoints := []string{
			"/health",
			"/metrics",
			"/api/users",
			"/api/products",
			"/api/orders",
		}

		endpoint := endpoints[i%len(endpoints)]
		req, err := http.NewRequest("GET", appURL+endpoint, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %v", err)
		}

		// Add tracing headers if enabled
		if withTracing {
			traceID := GenerateTraceID()
			req.Header.Set("X-Trace-Id", traceID)
			req.Header.Set("X-Parent-Span-Id", GenerateTraceID()[:16])
			req.Header.Set("X-Span-Id", GenerateTraceID()[:16])
		}

		resp, err := client.Do(req)
		if err != nil {
			// Don't fail on individual request errors
			continue
		}
		resp.Body.Close()

		// Add some delay between requests
		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

// CheckServiceLogs checks if a service has expected logs
func CheckServiceLogs(serviceName string, expectedPattern string) (bool, error) {
	cmd := exec.Command("docker-compose", "logs", serviceName)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to get logs for %s: %v", serviceName, err)
	}

	return strings.Contains(string(output), expectedPattern), nil
}

// RestartService restarts a specific service
func RestartService(composeFile, serviceName string) error {
	cmd := exec.Command("docker-compose", "-f", composeFile, "restart", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart %s: %v\nOutput: %s", serviceName, err, output)
	}
	return nil
}

// ScaleService scales a service to the specified number of instances
func ScaleService(composeFile, serviceName string, instances int) error {
	cmd := exec.Command("docker-compose", "-f", composeFile, "scale", fmt.Sprintf("%s=%d", serviceName, instances))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to scale %s: %v\nOutput: %s", serviceName, err, output)
	}
	return nil
}

// GetServicePort gets the exposed port for a service
func GetServicePort(serviceName, internalPort string) (string, error) {
	cmd := exec.Command("docker-compose", "port", serviceName, internalPort)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get port for %s: %v", serviceName, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// WaitForMetric waits for a specific metric to appear in Prometheus
func WaitForMetric(prometheusURL, metricName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 5 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/query?query=%s", prometheusURL, metricName))
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			var result PrometheusQueryResponse
			if json.Unmarshal(body, &result) == nil && len(result.Data.Result) > 0 {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("metric %s did not appear within %v", metricName, timeout)
}

// VerifyGrafanaDashboard verifies that a Grafana dashboard is accessible
func VerifyGrafanaDashboard(grafanaURL, dashboardUID, username, password string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/dashboards/uid/%s", grafanaURL, dashboardUID), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get dashboard: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	return nil
}

// CleanupTestData cleans up test data from all services
func CleanupTestData() error {
	// This is a placeholder for cleanup operations
	// In a real implementation, you might want to:
	// - Clear test metrics from Prometheus
	// - Delete test logs from Loki
	// - Remove test traces from Jaeger
	// - Clear test alerts from AlertManager
	return nil
}
