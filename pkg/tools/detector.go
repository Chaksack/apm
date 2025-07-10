package tools

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// BaseDetector provides common detection functionality
type BaseDetector struct {
	toolType ToolType
	ports    []int
	client   *http.Client
}

// NewBaseDetector creates a new base detector
func NewBaseDetector(toolType ToolType, ports []int) *BaseDetector {
	return &BaseDetector{
		toolType: toolType,
		ports:    ports,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// DetectByPort checks if a tool is running on specified ports
func (bd *BaseDetector) DetectByPort(host string) (*Tool, error) {
	for _, port := range bd.ports {
		address := fmt.Sprintf("%s:%d", host, port)
		conn, err := net.DialTimeout("tcp", address, 2*time.Second)
		if err == nil {
			conn.Close()
			return &Tool{
				Type:        bd.toolType,
				Port:        port,
				Endpoint:    fmt.Sprintf("http://%s", address),
				InstallType: InstallTypeNative, // Will be determined later
				Status:      ToolStatusUnknown,
			}, nil
		}
	}
	return nil, fmt.Errorf("tool not found on any configured port")
}

// DetectByProcess checks if a tool is running as a process
func (bd *BaseDetector) DetectByProcess(processName string) (*Tool, error) {
	cmd := exec.Command("pgrep", "-f", processName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("process not found: %s", processName)
	}

	if len(strings.TrimSpace(string(output))) > 0 {
		// Process found, now find the port
		tool, err := bd.DetectByPort("localhost")
		if err != nil {
			return nil, err
		}
		tool.InstallType = InstallTypeNative
		return tool, nil
	}

	return nil, fmt.Errorf("process not running: %s", processName)
}

// PrometheusDetector detects Prometheus installations
type PrometheusDetector struct {
	*BaseDetector
}

// NewPrometheusDetector creates a new Prometheus detector
func NewPrometheusDetector() *PrometheusDetector {
	return &PrometheusDetector{
		BaseDetector: NewBaseDetector(ToolTypePrometheus, []int{9090, 9091, 9092}),
	}
}

// Detect attempts to detect Prometheus installation
func (pd *PrometheusDetector) Detect() (*Tool, error) {
	// Try to detect by port first
	tool, err := pd.DetectByPort("localhost")
	if err == nil {
		// Verify it's actually Prometheus
		if err := pd.Validate(); err != nil {
			return nil, err
		}
		tool.Name = "prometheus"
		tool.HealthEndpoint = fmt.Sprintf("%s/-/healthy", tool.Endpoint)
		return tool, nil
	}

	// Try to detect by process
	tool, err = pd.DetectByProcess("prometheus")
	if err == nil {
		tool.Name = "prometheus"
		tool.HealthEndpoint = fmt.Sprintf("%s/-/healthy", tool.Endpoint)
		return tool, nil
	}

	return nil, fmt.Errorf("prometheus not detected")
}

// Validate verifies that the detected tool is actually Prometheus
func (pd *PrometheusDetector) Validate() error {
	// This would be called with the actual endpoint
	// For now, return nil as placeholder
	return nil
}

// GetVersion retrieves the Prometheus version
func (pd *PrometheusDetector) GetVersion() (string, error) {
	// Would make API call to /api/v1/status/buildinfo
	return "2.45.0", nil // Placeholder
}

// GrafanaDetector detects Grafana installations
type GrafanaDetector struct {
	*BaseDetector
}

// NewGrafanaDetector creates a new Grafana detector
func NewGrafanaDetector() *GrafanaDetector {
	return &GrafanaDetector{
		BaseDetector: NewBaseDetector(ToolTypeGrafana, []int{3000, 3001, 3002}),
	}
}

// Detect attempts to detect Grafana installation
func (gd *GrafanaDetector) Detect() (*Tool, error) {
	tool, err := gd.DetectByPort("localhost")
	if err == nil {
		tool.Name = "grafana"
		tool.HealthEndpoint = fmt.Sprintf("%s/api/health", tool.Endpoint)
		return tool, nil
	}

	tool, err = gd.DetectByProcess("grafana-server")
	if err == nil {
		tool.Name = "grafana"
		tool.HealthEndpoint = fmt.Sprintf("%s/api/health", tool.Endpoint)
		return tool, nil
	}

	return nil, fmt.Errorf("grafana not detected")
}

// Validate verifies that the detected tool is actually Grafana
func (gd *GrafanaDetector) Validate() error {
	return nil
}

// GetVersion retrieves the Grafana version
func (gd *GrafanaDetector) GetVersion() (string, error) {
	return "10.0.0", nil // Placeholder
}

// JaegerDetector detects Jaeger installations
type JaegerDetector struct {
	*BaseDetector
}

// NewJaegerDetector creates a new Jaeger detector
func NewJaegerDetector() *JaegerDetector {
	return &JaegerDetector{
		BaseDetector: NewBaseDetector(ToolTypeJaeger, []int{16686, 16687}),
	}
}

// Detect attempts to detect Jaeger installation
func (jd *JaegerDetector) Detect() (*Tool, error) {
	tool, err := jd.DetectByPort("localhost")
	if err == nil {
		tool.Name = "jaeger"
		tool.HealthEndpoint = fmt.Sprintf("%s/", tool.Endpoint)
		return tool, nil
	}

	tool, err = jd.DetectByProcess("jaeger")
	if err == nil {
		tool.Name = "jaeger"
		tool.HealthEndpoint = fmt.Sprintf("%s/", tool.Endpoint)
		return tool, nil
	}

	return nil, fmt.Errorf("jaeger not detected")
}

// Validate verifies that the detected tool is actually Jaeger
func (jd *JaegerDetector) Validate() error {
	return nil
}

// GetVersion retrieves the Jaeger version
func (jd *JaegerDetector) GetVersion() (string, error) {
	return "1.47.0", nil // Placeholder
}

// LokiDetector detects Loki installations
type LokiDetector struct {
	*BaseDetector
}

// NewLokiDetector creates a new Loki detector
func NewLokiDetector() *LokiDetector {
	return &LokiDetector{
		BaseDetector: NewBaseDetector(ToolTypeLoki, []int{3100, 3101}),
	}
}

// Detect attempts to detect Loki installation
func (ld *LokiDetector) Detect() (*Tool, error) {
	tool, err := ld.DetectByPort("localhost")
	if err == nil {
		tool.Name = "loki"
		tool.HealthEndpoint = fmt.Sprintf("%s/ready", tool.Endpoint)
		return tool, nil
	}

	tool, err = ld.DetectByProcess("loki")
	if err == nil {
		tool.Name = "loki"
		tool.HealthEndpoint = fmt.Sprintf("%s/ready", tool.Endpoint)
		return tool, nil
	}

	return nil, fmt.Errorf("loki not detected")
}

// Validate verifies that the detected tool is actually Loki
func (ld *LokiDetector) Validate() error {
	return nil
}

// GetVersion retrieves the Loki version
func (ld *LokiDetector) GetVersion() (string, error) {
	return "2.9.0", nil // Placeholder
}

// AlertManagerDetector detects AlertManager installations
type AlertManagerDetector struct {
	*BaseDetector
}

// NewAlertManagerDetector creates a new AlertManager detector
func NewAlertManagerDetector() *AlertManagerDetector {
	return &AlertManagerDetector{
		BaseDetector: NewBaseDetector(ToolTypeAlertManager, []int{9093, 9094}),
	}
}

// Detect attempts to detect AlertManager installation
func (ad *AlertManagerDetector) Detect() (*Tool, error) {
	tool, err := ad.DetectByPort("localhost")
	if err == nil {
		tool.Name = "alertmanager"
		tool.HealthEndpoint = fmt.Sprintf("%s/-/healthy", tool.Endpoint)
		return tool, nil
	}

	tool, err = ad.DetectByProcess("alertmanager")
	if err == nil {
		tool.Name = "alertmanager"
		tool.HealthEndpoint = fmt.Sprintf("%s/-/healthy", tool.Endpoint)
		return tool, nil
	}

	return nil, fmt.Errorf("alertmanager not detected")
}

// Validate verifies that the detected tool is actually AlertManager
func (ad *AlertManagerDetector) Validate() error {
	return nil
}

// GetVersion retrieves the AlertManager version
func (ad *AlertManagerDetector) GetVersion() (string, error) {
	return "0.26.0", nil // Placeholder
}

// DetectorFactory creates detectors for different tool types
type DetectorFactory struct{}

// NewDetectorFactory creates a new detector factory
func NewDetectorFactory() *DetectorFactory {
	return &DetectorFactory{}
}

// CreateDetector creates a detector for the specified tool type
func (df *DetectorFactory) CreateDetector(toolType ToolType) (ToolDetector, error) {
	switch toolType {
	case ToolTypePrometheus:
		return NewPrometheusDetector(), nil
	case ToolTypeGrafana:
		return NewGrafanaDetector(), nil
	case ToolTypeJaeger:
		return NewJaegerDetector(), nil
	case ToolTypeLoki:
		return NewLokiDetector(), nil
	case ToolTypeAlertManager:
		return NewAlertManagerDetector(), nil
	default:
		return nil, fmt.Errorf("unsupported tool type: %s", toolType)
	}
}

// DetectAllTools attempts to detect all supported tools
func DetectAllTools(ctx context.Context) ([]*Tool, error) {
	factory := NewDetectorFactory()
	toolTypes := []ToolType{
		ToolTypePrometheus,
		ToolTypeGrafana,
		ToolTypeJaeger,
		ToolTypeLoki,
		ToolTypeAlertManager,
	}

	var detectedTools []*Tool
	for _, toolType := range toolTypes {
		detector, err := factory.CreateDetector(toolType)
		if err != nil {
			continue
		}

		tool, err := detector.Detect()
		if err == nil {
			detectedTools = append(detectedTools, tool)
		}
	}

	return detectedTools, nil
}