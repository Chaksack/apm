package deployment

import (
	"encoding/json"
	"fmt"
	"time"
)

// DashboardMetrics provides deployment metrics for dashboard integration
type DashboardMetrics struct {
	TotalDeployments       int                       `json:"total_deployments"`
	ActiveDeployments      int                       `json:"active_deployments"`
	SuccessfulDeployments  int                       `json:"successful_deployments"`
	FailedDeployments      int                       `json:"failed_deployments"`
	RolledBackDeployments  int                       `json:"rolled_back_deployments"`
	AverageDeploymentTime  time.Duration             `json:"average_deployment_time"`
	DeploymentsByPlatform  map[string]int            `json:"deployments_by_platform"`
	DeploymentsByEnvironment map[string]int          `json:"deployments_by_environment"`
	RecentDeployments      []DeploymentSummary       `json:"recent_deployments"`
	HealthSummary          map[string]HealthSummary  `json:"health_summary"`
}

// DeploymentSummary provides a summary of a deployment for dashboard display
type DeploymentSummary struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Version      string             `json:"version"`
	Platform     DeploymentPlatform `json:"platform"`
	Environment  string             `json:"environment"`
	Status       DeploymentStatus   `json:"status"`
	StartTime    time.Time          `json:"start_time"`
	Duration     *time.Duration     `json:"duration,omitempty"`
	Progress     float64            `json:"progress"`
	HealthStatus HealthStatus       `json:"health_status"`
}

// HealthSummary provides health check summary for a deployment
type HealthSummary struct {
	TotalChecks    int          `json:"total_checks"`
	HealthyChecks  int          `json:"healthy_checks"`
	UnhealthyChecks int         `json:"unhealthy_checks"`
	OverallStatus  HealthStatus `json:"overall_status"`
	LastChecked    time.Time    `json:"last_checked"`
}

// GrafanaDashboard represents a Grafana dashboard configuration
type GrafanaDashboard struct {
	Dashboard json.RawMessage `json:"dashboard"`
	Overwrite bool            `json:"overwrite"`
}

// PrometheusMetrics generates Prometheus metrics for deployments
type PrometheusMetrics struct {
	DeploymentTotal         map[string]int
	DeploymentDuration      map[string]float64
	DeploymentStatus        map[string]int
	HealthCheckStatus       map[string]int
	RollbackTotal           map[string]int
}

// GenerateDashboardMetrics generates metrics for dashboard display
func (s *Service) GenerateDashboardMetrics(timeRange time.Duration) (*DashboardMetrics, error) {
	endTime := time.Now()
	startTime := endTime.Add(-timeRange)

	// Get deployments in time range
	deployments, err := s.GetDeployments(DeploymentFilters{
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     1000,
	})
	if err != nil {
		return nil, err
	}

	metrics := &DashboardMetrics{
		TotalDeployments:         len(deployments),
		DeploymentsByPlatform:    make(map[string]int),
		DeploymentsByEnvironment: make(map[string]int),
		HealthSummary:            make(map[string]HealthSummary),
	}

	var totalDuration time.Duration
	completedCount := 0

	for _, deployment := range deployments {
		// Count by status
		switch deployment.Status {
		case StatusCompleted:
			metrics.SuccessfulDeployments++
		case StatusFailed:
			metrics.FailedDeployments++
		case StatusRolledBack:
			metrics.RolledBackDeployments++
		case StatusDeploying, StatusVerifying, StatusPreparing:
			metrics.ActiveDeployments++
		}

		// Count by platform
		metrics.DeploymentsByPlatform[string(deployment.Platform)]++

		// Count by environment
		metrics.DeploymentsByEnvironment[deployment.Environment]++

		// Calculate duration for completed deployments
		if deployment.EndTime != nil && deployment.Status == StatusCompleted {
			duration := deployment.EndTime.Sub(deployment.StartTime)
			totalDuration += duration
			completedCount++
		}

		// Add to recent deployments (limit to 10)
		if len(metrics.RecentDeployments) < 10 {
			summary := DeploymentSummary{
				ID:          deployment.ID,
				Name:        deployment.Name,
				Version:     deployment.Version,
				Platform:    deployment.Platform,
				Environment: deployment.Environment,
				Status:      deployment.Status,
				StartTime:   deployment.StartTime,
			}

			if deployment.Progress != nil {
				summary.Progress = deployment.Progress.Percentage
			}

			if deployment.EndTime != nil {
				duration := deployment.EndTime.Sub(deployment.StartTime)
				summary.Duration = &duration
			}

			// Calculate health status
			summary.HealthStatus = s.calculateOverallHealthStatus(deployment.HealthChecks)

			metrics.RecentDeployments = append(metrics.RecentDeployments, summary)
		}

		// Generate health summary
		if len(deployment.HealthChecks) > 0 {
			healthSummary := HealthSummary{
				TotalChecks: len(deployment.HealthChecks),
				LastChecked: time.Now(),
			}

			for _, check := range deployment.HealthChecks {
				switch check.Status {
				case HealthStatusHealthy:
					healthSummary.HealthyChecks++
				case HealthStatusUnhealthy:
					healthSummary.UnhealthyChecks++
				}
			}

			healthSummary.OverallStatus = s.calculateOverallHealthStatus(deployment.HealthChecks)
			metrics.HealthSummary[deployment.ID] = healthSummary
		}
	}

	// Calculate average deployment time
	if completedCount > 0 {
		metrics.AverageDeploymentTime = totalDuration / time.Duration(completedCount)
	}

	return metrics, nil
}

// GeneratePrometheusMetrics generates Prometheus-compatible metrics
func (s *Service) GeneratePrometheusMetrics() (*PrometheusMetrics, error) {
	// Get recent deployments (last 24 hours)
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	deployments, err := s.GetDeployments(DeploymentFilters{
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     1000,
	})
	if err != nil {
		return nil, err
	}

	metrics := &PrometheusMetrics{
		DeploymentTotal:    make(map[string]int),
		DeploymentDuration: make(map[string]float64),
		DeploymentStatus:   make(map[string]int),
		HealthCheckStatus:  make(map[string]int),
		RollbackTotal:      make(map[string]int),
	}

	for _, deployment := range deployments {
		// Labels for metrics
		labels := fmt.Sprintf(`platform="%s",environment="%s",version="%s"`,
			deployment.Platform, deployment.Environment, deployment.Version)

		// Deployment total
		metrics.DeploymentTotal[labels]++

		// Deployment status
		statusLabel := fmt.Sprintf(`%s,status="%s"`, labels, deployment.Status)
		metrics.DeploymentStatus[statusLabel]++

		// Deployment duration (for completed deployments)
		if deployment.EndTime != nil && deployment.Status == StatusCompleted {
			duration := deployment.EndTime.Sub(deployment.StartTime).Seconds()
			metrics.DeploymentDuration[labels] = duration
		}

		// Health check status
		for _, check := range deployment.HealthChecks {
			healthLabel := fmt.Sprintf(`%s,check_type="%s",status="%s"`,
				labels, check.Type, check.Status)
			metrics.HealthCheckStatus[healthLabel]++
		}

		// Rollback count
		if deployment.Status == StatusRolledBack {
			metrics.RollbackTotal[labels]++
		}
	}

	return metrics, nil
}

// CreateGrafanaDashboard creates a Grafana dashboard configuration for deployments
func CreateGrafanaDashboard() *GrafanaDashboard {
	dashboardJSON := `{
		"title": "Deployment Monitoring",
		"panels": [
			{
				"title": "Deployment Status Overview",
				"type": "stat",
				"targets": [{
					"expr": "sum(deployment_status) by (status)"
				}],
				"gridPos": {"h": 8, "w": 6, "x": 0, "y": 0}
			},
			{
				"title": "Deployment Success Rate",
				"type": "gauge",
				"targets": [{
					"expr": "sum(deployment_status{status=\"completed\"}) / sum(deployment_status) * 100"
				}],
				"gridPos": {"h": 8, "w": 6, "x": 6, "y": 0}
			},
			{
				"title": "Average Deployment Duration",
				"type": "graph",
				"targets": [{
					"expr": "avg(deployment_duration) by (platform)"
				}],
				"gridPos": {"h": 8, "w": 12, "x": 12, "y": 0}
			},
			{
				"title": "Deployments by Platform",
				"type": "piechart",
				"targets": [{
					"expr": "sum(deployment_total) by (platform)"
				}],
				"gridPos": {"h": 8, "w": 8, "x": 0, "y": 8}
			},
			{
				"title": "Health Check Status",
				"type": "table",
				"targets": [{
					"expr": "deployment_health_check_status"
				}],
				"gridPos": {"h": 8, "w": 8, "x": 8, "y": 8}
			},
			{
				"title": "Rollback Trend",
				"type": "graph",
				"targets": [{
					"expr": "increase(deployment_rollback_total[1h])"
				}],
				"gridPos": {"h": 8, "w": 8, "x": 16, "y": 8}
			},
			{
				"title": "Active Deployments",
				"type": "table",
				"targets": [{
					"expr": "deployment_status{status=~\"deploying|verifying\"}"
				}],
				"gridPos": {"h": 8, "w": 24, "x": 0, "y": 16}
			}
		],
		"refresh": "5s",
		"time": {"from": "now-1h", "to": "now"},
		"timezone": "browser",
		"uid": "deployment-monitoring",
		"version": 1
	}`

	return &GrafanaDashboard{
		Dashboard: json.RawMessage(dashboardJSON),
		Overwrite: true,
	}
}

// calculateOverallHealthStatus calculates the overall health status from health checks
func (s *Service) calculateOverallHealthStatus(healthChecks []HealthCheck) HealthStatus {
	if len(healthChecks) == 0 {
		return HealthStatusUnknown
	}

	unhealthyCount := 0
	degradedCount := 0

	for _, check := range healthChecks {
		switch check.Status {
		case HealthStatusUnhealthy:
			unhealthyCount++
		case HealthStatusDegraded:
			degradedCount++
		}
	}

	if unhealthyCount > 0 {
		return HealthStatusUnhealthy
	}
	if degradedCount > 0 {
		return HealthStatusDegraded
	}
	return HealthStatusHealthy
}

// MetricsExporter formats metrics for Prometheus exposition
func FormatPrometheusMetrics(metrics *PrometheusMetrics) string {
	var output string

	// Deployment total
	output += "# HELP deployment_total Total number of deployments\n"
	output += "# TYPE deployment_total counter\n"
	for labels, value := range metrics.DeploymentTotal {
		output += fmt.Sprintf("deployment_total{%s} %d\n", labels, value)
	}

	// Deployment duration
	output += "\n# HELP deployment_duration_seconds Duration of deployments in seconds\n"
	output += "# TYPE deployment_duration_seconds gauge\n"
	for labels, value := range metrics.DeploymentDuration {
		output += fmt.Sprintf("deployment_duration_seconds{%s} %.2f\n", labels, value)
	}

	// Deployment status
	output += "\n# HELP deployment_status Current deployment status\n"
	output += "# TYPE deployment_status gauge\n"
	for labels, value := range metrics.DeploymentStatus {
		output += fmt.Sprintf("deployment_status{%s} %d\n", labels, value)
	}

	// Health check status
	output += "\n# HELP deployment_health_check_status Health check status\n"
	output += "# TYPE deployment_health_check_status gauge\n"
	for labels, value := range metrics.HealthCheckStatus {
		output += fmt.Sprintf("deployment_health_check_status{%s} %d\n", labels, value)
	}

	// Rollback total
	output += "\n# HELP deployment_rollback_total Total number of rollbacks\n"
	output += "# TYPE deployment_rollback_total counter\n"
	for labels, value := range metrics.RollbackTotal {
		output += fmt.Sprintf("deployment_rollback_total{%s} %d\n", labels, value)
	}

	return output
}