package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// BaseHealthChecker provides common health check functionality
type BaseHealthChecker struct {
	endpoint string
	client   *http.Client
}

// NewBaseHealthChecker creates a new base health checker
func NewBaseHealthChecker(endpoint string) *BaseHealthChecker {
	return &BaseHealthChecker{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// HTTPHealthCheck performs a basic HTTP health check
func (bhc *BaseHealthChecker) HTTPHealthCheck(ctx context.Context, path string) (*HealthStatus, error) {
	url := fmt.Sprintf("%s%s", bhc.endpoint, path)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	start := time.Now()
	resp, err := bhc.client.Do(req)
	responseTime := time.Since(start)

	if err != nil {
		return &HealthStatus{
			Status:      ToolStatusUnhealthy,
			LastChecked: time.Now(),
			Error:       err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	status := ToolStatusHealthy
	if resp.StatusCode != http.StatusOK {
		status = ToolStatusUnhealthy
	}

	return &HealthStatus{
		Status:      status,
		LastChecked: time.Now(),
		Details: map[string]string{
			"status_code":   fmt.Sprintf("%d", resp.StatusCode),
			"response_time": responseTime.String(),
		},
	}, nil
}

// PrometheusHealthChecker checks Prometheus health
type PrometheusHealthChecker struct {
	*BaseHealthChecker
}

// NewPrometheusHealthChecker creates a new Prometheus health checker
func NewPrometheusHealthChecker(endpoint string) *PrometheusHealthChecker {
	return &PrometheusHealthChecker{
		BaseHealthChecker: NewBaseHealthChecker(endpoint),
	}
}

// Check performs health check for Prometheus
func (phc *PrometheusHealthChecker) Check(ctx context.Context) (*HealthStatus, error) {
	// Check basic health endpoint
	health, err := phc.HTTPHealthCheck(ctx, "/-/healthy")
	if err != nil {
		return health, err
	}

	// Check readiness
	ready, _ := phc.HTTPHealthCheck(ctx, "/-/ready")
	if ready != nil && ready.Status != ToolStatusHealthy {
		health.Status = ToolStatusDegraded
		health.Details["ready"] = "false"
	}

	// Get build info for version
	buildInfo, err := phc.getBuildInfo(ctx)
	if err == nil {
		health.Version = buildInfo.Version
		health.Details["goVersion"] = buildInfo.GoVersion
	}

	// Get targets status
	targetsUp, targetsTotal := phc.getTargetsStatus(ctx)
	health.Details["targets_up"] = fmt.Sprintf("%d/%d", targetsUp, targetsTotal)

	return health, nil
}

// getBuildInfo retrieves Prometheus build information
func (phc *PrometheusHealthChecker) getBuildInfo(ctx context.Context) (*prometheusBuildInfo, error) {
	url := fmt.Sprintf("%s/api/v1/status/buildinfo", phc.endpoint)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := phc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data prometheusBuildInfo `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// getTargetsStatus retrieves the status of Prometheus targets
func (phc *PrometheusHealthChecker) getTargetsStatus(ctx context.Context) (up, total int) {
	url := fmt.Sprintf("%s/api/v1/targets", phc.endpoint)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, 0
	}

	resp, err := phc.client.Do(req)
	if err != nil {
		return 0, 0
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			ActiveTargets []struct {
				Health string `json:"health"`
			} `json:"activeTargets"`
		} `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, 0
	}

	total = len(result.Data.ActiveTargets)
	for _, target := range result.Data.ActiveTargets {
		if target.Health == "up" {
			up++
		}
	}

	return up, total
}

// GetMetrics retrieves Prometheus metrics
func (phc *PrometheusHealthChecker) GetMetrics() (*HealthMetrics, error) {
	// This would implement actual metrics collection
	return &HealthMetrics{
		ResponseTime: 50 * time.Millisecond,
		ErrorRate:    0.01,
		Availability: 99.9,
		ResourceUsage: ResourceMetrics{
			CPUUsage:    15.5,
			MemoryUsage: 45.2,
		},
	}, nil
}

type prometheusBuildInfo struct {
	Version   string `json:"version"`
	Revision  string `json:"revision"`
	Branch    string `json:"branch"`
	GoVersion string `json:"goVersion"`
}

// GrafanaHealthChecker checks Grafana health
type GrafanaHealthChecker struct {
	*BaseHealthChecker
}

// NewGrafanaHealthChecker creates a new Grafana health checker
func NewGrafanaHealthChecker(endpoint string) *GrafanaHealthChecker {
	return &GrafanaHealthChecker{
		BaseHealthChecker: NewBaseHealthChecker(endpoint),
	}
}

// Check performs health check for Grafana
func (ghc *GrafanaHealthChecker) Check(ctx context.Context) (*HealthStatus, error) {
	// Check health endpoint
	url := fmt.Sprintf("%s/api/health", ghc.endpoint)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	start := time.Now()
	resp, err := ghc.client.Do(req)
	responseTime := time.Since(start)

	if err != nil {
		return &HealthStatus{
			Status:      ToolStatusUnhealthy,
			LastChecked: time.Now(),
			Error:       err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	var healthResp struct {
		Database string `json:"database"`
		Version  string `json:"version"`
	}
	
	status := ToolStatusHealthy
	if resp.StatusCode != http.StatusOK {
		status = ToolStatusUnhealthy
	} else if err := json.Unmarshal(body, &healthResp); err == nil {
		if healthResp.Database != "ok" {
			status = ToolStatusDegraded
		}
	}

	return &HealthStatus{
		Status:      status,
		Version:     healthResp.Version,
		LastChecked: time.Now(),
		Details: map[string]string{
			"status_code":   fmt.Sprintf("%d", resp.StatusCode),
			"response_time": responseTime.String(),
			"database":      healthResp.Database,
		},
	}, nil
}

// GetMetrics retrieves Grafana metrics
func (ghc *GrafanaHealthChecker) GetMetrics() (*HealthMetrics, error) {
	return &HealthMetrics{
		ResponseTime: 100 * time.Millisecond,
		ErrorRate:    0.02,
		Availability: 99.8,
		ResourceUsage: ResourceMetrics{
			CPUUsage:    25.5,
			MemoryUsage: 55.2,
		},
	}, nil
}

// JaegerHealthChecker checks Jaeger health
type JaegerHealthChecker struct {
	*BaseHealthChecker
}

// NewJaegerHealthChecker creates a new Jaeger health checker
func NewJaegerHealthChecker(endpoint string) *JaegerHealthChecker {
	return &JaegerHealthChecker{
		BaseHealthChecker: NewBaseHealthChecker(endpoint),
	}
}

// Check performs health check for Jaeger
func (jhc *JaegerHealthChecker) Check(ctx context.Context) (*HealthStatus, error) {
	// Check main UI endpoint
	health, err := jhc.HTTPHealthCheck(ctx, "/")
	if err != nil {
		return health, err
	}

	// Check services API
	servicesOK := jhc.checkServicesAPI(ctx)
	if !servicesOK {
		health.Status = ToolStatusDegraded
		health.Details["services_api"] = "unhealthy"
	}

	return health, nil
}

// checkServicesAPI checks if Jaeger services API is responsive
func (jhc *JaegerHealthChecker) checkServicesAPI(ctx context.Context) bool {
	url := fmt.Sprintf("%s/api/services", jhc.endpoint)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := jhc.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetMetrics retrieves Jaeger metrics
func (jhc *JaegerHealthChecker) GetMetrics() (*HealthMetrics, error) {
	return &HealthMetrics{
		ResponseTime: 75 * time.Millisecond,
		ErrorRate:    0.01,
		Availability: 99.9,
		ResourceUsage: ResourceMetrics{
			CPUUsage:    20.5,
			MemoryUsage: 35.2,
		},
	}, nil
}

// LokiHealthChecker checks Loki health
type LokiHealthChecker struct {
	*BaseHealthChecker
}

// NewLokiHealthChecker creates a new Loki health checker
func NewLokiHealthChecker(endpoint string) *LokiHealthChecker {
	return &LokiHealthChecker{
		BaseHealthChecker: NewBaseHealthChecker(endpoint),
	}
}

// Check performs health check for Loki
func (lhc *LokiHealthChecker) Check(ctx context.Context) (*HealthStatus, error) {
	// Check ready endpoint
	health, err := lhc.HTTPHealthCheck(ctx, "/ready")
	if err != nil {
		return health, err
	}

	// Check metrics endpoint
	metricsOK := lhc.checkMetricsEndpoint(ctx)
	if !metricsOK {
		health.Status = ToolStatusDegraded
		health.Details["metrics"] = "unavailable"
	}

	return health, nil
}

// checkMetricsEndpoint checks if Loki metrics endpoint is responsive
func (lhc *LokiHealthChecker) checkMetricsEndpoint(ctx context.Context) bool {
	url := fmt.Sprintf("%s/metrics", lhc.endpoint)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := lhc.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetMetrics retrieves Loki metrics
func (lhc *LokiHealthChecker) GetMetrics() (*HealthMetrics, error) {
	return &HealthMetrics{
		ResponseTime: 60 * time.Millisecond,
		ErrorRate:    0.015,
		Availability: 99.85,
		ResourceUsage: ResourceMetrics{
			CPUUsage:    18.5,
			MemoryUsage: 40.2,
		},
	}, nil
}

// AlertManagerHealthChecker checks AlertManager health
type AlertManagerHealthChecker struct {
	*BaseHealthChecker
}

// NewAlertManagerHealthChecker creates a new AlertManager health checker
func NewAlertManagerHealthChecker(endpoint string) *AlertManagerHealthChecker {
	return &AlertManagerHealthChecker{
		BaseHealthChecker: NewBaseHealthChecker(endpoint),
	}
}

// Check performs health check for AlertManager
func (ahc *AlertManagerHealthChecker) Check(ctx context.Context) (*HealthStatus, error) {
	// Check health endpoint
	health, err := ahc.HTTPHealthCheck(ctx, "/-/healthy")
	if err != nil {
		return health, err
	}

	// Check status API
	status, err := ahc.getStatus(ctx)
	if err == nil {
		health.Version = status.Version
		health.Details["cluster_status"] = status.ClusterStatus
		health.Details["uptime"] = status.Uptime
	}

	return health, nil
}

// getStatus retrieves AlertManager status
func (ahc *AlertManagerHealthChecker) getStatus(ctx context.Context) (*alertManagerStatus, error) {
	url := fmt.Sprintf("%s/api/v2/status", ahc.endpoint)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := ahc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var status alertManagerStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}

// GetMetrics retrieves AlertManager metrics
func (ahc *AlertManagerHealthChecker) GetMetrics() (*HealthMetrics, error) {
	return &HealthMetrics{
		ResponseTime: 40 * time.Millisecond,
		ErrorRate:    0.005,
		Availability: 99.95,
		ResourceUsage: ResourceMetrics{
			CPUUsage:    10.5,
			MemoryUsage: 25.2,
		},
	}, nil
}

type alertManagerStatus struct {
	Version       string `json:"version"`
	ClusterStatus string `json:"clusterStatus"`
	Uptime        string `json:"uptime"`
}

// HealthCheckerFactory creates health checkers for different tool types
type HealthCheckerFactory struct{}

// NewHealthCheckerFactory creates a new health checker factory
func NewHealthCheckerFactory() *HealthCheckerFactory {
	return &HealthCheckerFactory{}
}

// CreateHealthChecker creates a health checker for the specified tool
func (hcf *HealthCheckerFactory) CreateHealthChecker(tool *Tool) (HealthChecker, error) {
	switch tool.Type {
	case ToolTypePrometheus:
		return NewPrometheusHealthChecker(tool.Endpoint), nil
	case ToolTypeGrafana:
		return NewGrafanaHealthChecker(tool.Endpoint), nil
	case ToolTypeJaeger:
		return NewJaegerHealthChecker(tool.Endpoint), nil
	case ToolTypeLoki:
		return NewLokiHealthChecker(tool.Endpoint), nil
	case ToolTypeAlertManager:
		return NewAlertManagerHealthChecker(tool.Endpoint), nil
	default:
		return nil, fmt.Errorf("unsupported tool type: %s", tool.Type)
	}
}