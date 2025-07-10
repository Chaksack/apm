package tools

import (
	"context"
	"time"
)

// ToolType represents the type of APM tool
type ToolType string

const (
	ToolTypePrometheus   ToolType = "prometheus"
	ToolTypeGrafana      ToolType = "grafana"
	ToolTypeJaeger       ToolType = "jaeger"
	ToolTypeLoki         ToolType = "loki"
	ToolTypeAlertManager ToolType = "alertmanager"
	ToolTypeSonarQube    ToolType = "sonarqube"
)

// InstallType represents how a tool is installed
type InstallType string

const (
	InstallTypeDocker     InstallType = "docker"
	InstallTypeNative     InstallType = "native"
	InstallTypeKubernetes InstallType = "kubernetes"
)

// ToolStatus represents the current status of a tool
type ToolStatus string

const (
	ToolStatusHealthy   ToolStatus = "healthy"
	ToolStatusDegraded  ToolStatus = "degraded"
	ToolStatusUnhealthy ToolStatus = "unhealthy"
	ToolStatusUnknown   ToolStatus = "unknown"
	ToolStatusStopped   ToolStatus = "stopped"
)

// Tool represents an APM tool with its configuration
type Tool struct {
	Name            string            `json:"name"`
	Type            ToolType          `json:"type"`
	Version         string            `json:"version"`
	InstallType     InstallType       `json:"install_type"`
	Endpoint        string            `json:"endpoint"`
	HealthEndpoint  string            `json:"health_endpoint"`
	Port            int               `json:"port"`
	Status          ToolStatus        `json:"status"`
	Labels          map[string]string `json:"labels,omitempty"`
	LastHealthCheck time.Time         `json:"last_health_check"`
}

// ToolConfig holds configuration for a tool
type ToolConfig struct {
	Name                string                 `json:"name"`
	Type                ToolType               `json:"type"`
	Port                int                    `json:"port"`
	Host                string                 `json:"host"`
	InstallType         InstallType            `json:"install_type"`
	DockerImage         string                 `json:"docker_image,omitempty"`
	DockerTag           string                 `json:"docker_tag,omitempty"`
	ConfigPath          string                 `json:"config_path,omitempty"`
	DataPath            string                 `json:"data_path,omitempty"`
	CustomScrapeConfigs []interface{}          `json:"custom_scrape_configs,omitempty"`
	ExtraConfig         map[string]interface{} `json:"extra_config,omitempty"`
}

// HealthStatus represents the health status of a tool
type HealthStatus struct {
	Status      ToolStatus        `json:"status"`
	Version     string            `json:"version"`
	Uptime      time.Duration     `json:"uptime"`
	LastChecked time.Time         `json:"last_checked"`
	Details     map[string]string `json:"details,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// HealthMetrics provides detailed metrics about tool health
type HealthMetrics struct {
	ResponseTime  time.Duration   `json:"response_time"`
	ErrorRate     float64         `json:"error_rate"`
	ResourceUsage ResourceMetrics `json:"resource_usage"`
	Availability  float64         `json:"availability"`
}

// ResourceMetrics contains resource usage information
type ResourceMetrics struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage,omitempty"`
}

// ToolMetrics contains operational metrics for a tool
type ToolMetrics struct {
	Uptime       time.Duration          `json:"uptime"`
	RequestCount int64                  `json:"request_count"`
	ErrorCount   int64                  `json:"error_count"`
	Custom       map[string]interface{} `json:"custom,omitempty"`
}

// PortConfig defines port configuration for a tool
type PortConfig struct {
	Default      int      `json:"default"`
	Alternatives []int    `json:"alternatives"`
	Protocol     string   `json:"protocol"`
	Description  string   `json:"description"`
}

// APMTool interface for all monitoring tools
type APMTool interface {
	// Lifecycle management
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error

	// Configuration
	GetConfig() ToolConfig
	UpdateConfig(config ToolConfig) error
	ValidateConfig(config ToolConfig) error

	// Health and status
	HealthCheck(ctx context.Context) (*HealthStatus, error)
	GetMetrics() (*ToolMetrics, error)
	GetStatus() ToolStatus

	// Integration
	GetEndpoints() map[string]string
	GetAPIClient() interface{}
}

// ToolDetector interface for detecting tool installations
type ToolDetector interface {
	Detect() (*Tool, error)
	Validate() error
	GetVersion() (string, error)
}

// HealthChecker interface for tool health monitoring
type HealthChecker interface {
	Check(ctx context.Context) (*HealthStatus, error)
	GetMetrics() (*HealthMetrics, error)
}

// Installer interface for different installation types
type Installer interface {
	Install(config ToolConfig) error
	Uninstall() error
	Upgrade(version string) error
	GetInstallationType() InstallType
}

// ToolPlugin interface for extending with new tools
type ToolPlugin interface {
	// Metadata
	Name() string
	Version() string
	Description() string

	// Factory method
	CreateTool(config map[string]interface{}) (APMTool, error)

	// Configuration schema
	GetConfigSchema() []byte

	// Default configuration
	GetDefaultConfig() map[string]interface{}
}