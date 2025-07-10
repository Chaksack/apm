package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/registry"
)

// CreateBuildContext creates a tar archive from a directory for Docker build
func CreateBuildContext(dockerfilePath, contextPath string) (io.ReadCloser, error) {
	// Create a buffer to write our tar archive to
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	// Walk the context directory
	err := filepath.Walk(contextPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .dockerignore entries
		if shouldIgnore(path, contextPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, path)
		if err != nil {
			return err
		}

		// Update header name to be relative to context
		relPath, err := filepath.Rel(contextPath, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// If it's a file, write its content
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Make sure to flush the tar writer
	if err := tw.Close(); err != nil {
		return nil, err
	}

	return io.NopCloser(buf), nil
}

// shouldIgnore checks if a path should be ignored based on .dockerignore
func shouldIgnore(path, contextPath string) bool {
	// Read .dockerignore file
	dockerignorePath := filepath.Join(contextPath, ".dockerignore")
	ignorePatterns, err := readDockerignore(dockerignorePath)
	if err != nil {
		return false
	}

	relPath, err := filepath.Rel(contextPath, path)
	if err != nil {
		return false
	}

	// Check against ignore patterns
	for _, pattern := range ignorePatterns {
		matched, err := filepath.Match(pattern, relPath)
		if err == nil && matched {
			return true
		}
	}

	// Always ignore .git directory
	if strings.Contains(relPath, ".git") {
		return true
	}

	return false
}

// readDockerignore reads patterns from .dockerignore file
func readDockerignore(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var patterns []string
	buf := make([]byte, 1024)
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			break
		}

		lines := strings.Split(string(buf[:n]), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				patterns = append(patterns, line)
			}
		}
	}

	return patterns, nil
}

// calculateCPUPercent calculates CPU usage percentage from container stats
func calculateCPUPercent(stats *types.StatsJSON) float64 {
	// Calculate CPU delta
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)

	// Calculate system CPU delta
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)

	// Calculate number of CPUs
	cpuCount := float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	if cpuCount == 0 {
		cpuCount = 1
	}

	// Calculate percentage
	if systemDelta > 0 && cpuDelta > 0 {
		return (cpuDelta / systemDelta) * cpuCount * 100.0
	}

	return 0
}

// calculateMemoryUsage calculates memory usage from container stats
func calculateMemoryUsage(stats *types.StatsJSON) MemoryMetrics {
	return MemoryMetrics{
		UsageBytes:   stats.MemoryStats.Usage,
		LimitBytes:   stats.MemoryStats.Limit,
		UsagePercent: float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0,
		Cache:        stats.MemoryStats.Stats["cache"],
		RSS:          stats.MemoryStats.Stats["rss"],
	}
}

// extractNetworkStats extracts network statistics from container stats
func extractNetworkStats(stats *types.StatsJSON) NetworkMetrics {
	metrics := NetworkMetrics{}

	for _, network := range stats.Networks {
		metrics.RxBytes += network.RxBytes
		metrics.TxBytes += network.TxBytes
		metrics.RxPackets += network.RxPackets
		metrics.TxPackets += network.TxPackets
		metrics.RxErrors += network.RxErrors
		metrics.TxErrors += network.TxErrors
	}

	return metrics
}

// extractDiskStats extracts disk I/O statistics from container stats
func extractDiskStats(stats *types.StatsJSON) DiskMetrics {
	metrics := DiskMetrics{}

	for _, ioStats := range stats.BlkioStats.IoServiceBytesRecursive {
		if ioStats.Op == "Read" {
			metrics.ReadBytes += ioStats.Value
		} else if ioStats.Op == "Write" {
			metrics.WriteBytes += ioStats.Value
		}
	}

	for _, ioStats := range stats.BlkioStats.IoServicedRecursive {
		if ioStats.Op == "Read" {
			metrics.ReadOps += ioStats.Value
		} else if ioStats.Op == "Write" {
			metrics.WriteOps += ioStats.Value
		}
	}

	return metrics
}

// getECRAuth gets ECR authentication token
func (c *Client) getECRAuth() (registry.AuthConfig, error) {
	// This is a simplified version - in production, use AWS SDK
	return registry.AuthConfig{
		Username:      "AWS",
		Password:      os.Getenv("ECR_AUTH_TOKEN"), // Should be obtained via AWS SDK
		ServerAddress: "https://123456789.dkr.ecr.us-east-1.amazonaws.com",
	}, nil
}

// getGCRAuth gets GCR authentication token
func (c *Client) getGCRAuth() (registry.AuthConfig, error) {
	// This is a simplified version - in production, use Google Cloud SDK
	return registry.AuthConfig{
		Username:      "_json_key",
		Password:      os.Getenv("GCP_SERVICE_ACCOUNT_KEY"), // JSON key content
		ServerAddress: "https://gcr.io",
	}, nil
}

// getACRAuth gets ACR authentication token
func (c *Client) getACRAuth() (registry.AuthConfig, error) {
	// This is a simplified version - in production, use Azure SDK
	return registry.AuthConfig{
		Username:      os.Getenv("ACR_USERNAME"),
		Password:      os.Getenv("ACR_PASSWORD"),
		ServerAddress: fmt.Sprintf("https://%s.azurecr.io", os.Getenv("ACR_REGISTRY_NAME")),
	}, nil
}

// PushWithAuth pushes an image with authentication
func (c *Client) PushWithAuth(ctx context.Context, image string, auth registry.AuthConfig) error {
	// This method would be implemented using the Docker client
	// For now, it's a placeholder
	return fmt.Errorf("push not implemented")
}

// PullWithAuth pulls an image with authentication
func (c *Client) PullWithAuth(ctx context.Context, image string, auth registry.AuthConfig) error {
	// This method would be implemented using the Docker client
	// For now, it's a placeholder
	return fmt.Errorf("pull not implemented")
}

// ImageScanner scans Docker images for vulnerabilities
type ImageScanner struct {
	client *types.Client
}

// NewImageScanner creates a new image scanner
func NewImageScanner(client interface{}) *ImageScanner {
	return &ImageScanner{}
}

// Scan scans an image for vulnerabilities
func (s *ImageScanner) Scan(ctx context.Context, imageID string) (*ScanReport, error) {
	// This would integrate with a vulnerability scanning tool like Trivy
	// For now, return a mock report
	return &ScanReport{
		ImageID:  imageID,
		Critical: 0,
		High:     0,
		Medium:   2,
		Low:      5,
		Unknown:  1,
	}, nil
}

// convertECRScanReport converts ECR scan results to our format
func convertECRScanReport(findings interface{}) *ScanReport {
	// This would convert ECR-specific scan results
	// For now, return a mock report
	return &ScanReport{
		Critical: 0,
		High:     1,
		Medium:   3,
		Low:      10,
		Unknown:  2,
	}
}

// DockerComposeBuilder builds Docker Compose configurations
type DockerComposeBuilder struct {
	config DockerComposeConfig
}

// NewDockerComposeBuilder creates a new Docker Compose builder
func NewDockerComposeBuilder(version string) *DockerComposeBuilder {
	return &DockerComposeBuilder{
		config: DockerComposeConfig{
			Version:  version,
			Services: make(map[string]ServiceConfig),
			Networks: make(map[string]NetworkConfig),
			Volumes:  make(map[string]VolumeConfig),
		},
	}
}

// AddService adds a service to the Docker Compose configuration
func (b *DockerComposeBuilder) AddService(name string, config ServiceConfig) *DockerComposeBuilder {
	b.config.Services[name] = config
	return b
}

// AddAPMService adds standard APM services to Docker Compose
func (b *DockerComposeBuilder) AddAPMService(serviceName string) *DockerComposeBuilder {
	// Add Jaeger
	b.config.Services["jaeger"] = ServiceConfig{
		Image: "jaegertracing/all-in-one:latest",
		Ports: []string{
			"16686:16686",
			"4317:4317",
			"4318:4318",
		},
		Environment: map[string]string{
			"COLLECTOR_OTLP_ENABLED": "true",
		},
	}

	// Add Prometheus
	b.config.Services["prometheus"] = ServiceConfig{
		Image: "prom/prometheus:latest",
		Ports: []string{"9090:9090"},
		Volumes: []string{
			"./prometheus.yml:/etc/prometheus/prometheus.yml:ro",
		},
	}

	// Update the main service with APM configuration
	if service, ok := b.config.Services[serviceName]; ok {
		if service.Environment == nil {
			service.Environment = make(map[string]string)
		}
		service.Environment["OTEL_SERVICE_NAME"] = serviceName
		service.Environment["OTEL_EXPORTER_OTLP_ENDPOINT"] = "http://jaeger:4317"
		service.Environment["OTEL_TRACES_EXPORTER"] = "otlp"
		service.Environment["OTEL_METRICS_EXPORTER"] = "prometheus"

		if service.Labels == nil {
			service.Labels = make(map[string]string)
		}
		service.Labels["apm.enabled"] = "true"
		service.Labels["prometheus.io/scrape"] = "true"

		if service.DependsOn == nil {
			service.DependsOn = []string{}
		}
		service.DependsOn = append(service.DependsOn, "jaeger", "prometheus")

		b.config.Services[serviceName] = service
	}

	return b
}

// Build returns the Docker Compose configuration
func (b *DockerComposeBuilder) Build() DockerComposeConfig {
	return b.config
}

// GenerateDockerfile generates a Dockerfile with APM instrumentation
func GenerateDockerfile(language Language, appName string) (string, error) {
	injector := NewAPMInjector(language)

	// Create a basic Dockerfile template based on language
	var dockerfileTemplate string

	switch language {
	case LanguageGo:
		dockerfileTemplate = `FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o %s .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/%s .
EXPOSE 8080
CMD ["./%s"]`
		dockerfileTemplate = fmt.Sprintf(dockerfileTemplate, appName, appName, appName)

	case LanguageJava:
		dockerfileTemplate = `FROM maven:3.8-openjdk-17 AS builder
WORKDIR /app
COPY pom.xml .
RUN mvn dependency:go-offline
COPY src ./src
RUN mvn package

FROM openjdk:17-jre-slim
COPY --from=builder /app/target/%s.jar app.jar
EXPOSE 8080
ENTRYPOINT ["java", "-jar", "/app.jar"]`
		dockerfileTemplate = fmt.Sprintf(dockerfileTemplate, appName)

	default:
		return "", fmt.Errorf("unsupported language: %s", language)
	}

	// Write to temporary file
	tmpFile, err := os.CreateTemp("", "Dockerfile")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(dockerfileTemplate); err != nil {
		return "", err
	}
	tmpFile.Close()

	// Inject APM
	return injector.InjectAgent(tmpFile.Name())
}
