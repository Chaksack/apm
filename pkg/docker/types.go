package docker

import (
	"io"
	"time"
)

// Language represents a programming language for APM instrumentation
type Language string

const (
	LanguageGo      Language = "go"
	LanguageJava    Language = "java"
	LanguagePython  Language = "python"
	LanguageNodeJS  Language = "nodejs"
	LanguageRuby    Language = "ruby"
	LanguagePHP     Language = "php"
	LanguageDotNet  Language = "dotnet"
	LanguageUnknown Language = "unknown"
)

// RegistryType represents a container registry type
type RegistryType string

const (
	RegistryTypeDockerHub RegistryType = "dockerhub"
	RegistryTypeECR       RegistryType = "ecr"
	RegistryTypeGCR       RegistryType = "gcr"
	RegistryTypeACR       RegistryType = "acr"
	RegistryTypeCustom    RegistryType = "custom"
)

// BuildOptions contains options for building Docker images with APM
type BuildOptions struct {
	ContextPath  string
	Tags         []string
	ServiceName  string
	Environment  string
	Language     Language
	ScanImage    bool
	OutputStream io.Writer
	APMConfig    APMConfig
}

// APMConfig contains APM-specific configuration for instrumentation
type APMConfig struct {
	Enabled          bool
	AgentVersion     string
	Endpoint         string
	APIKey           string
	SamplingRate     float64
	LogLevel         string
	CustomAttributes map[string]string
	Features         APMFeatures
}

// APMFeatures represents enabled APM features
type APMFeatures struct {
	Metrics       bool
	Tracing       bool
	Logging       bool
	Profiling     bool
	ErrorTracking bool
}

// ContainerMetrics represents metrics collected from a container
type ContainerMetrics struct {
	ContainerID string
	Timestamp   time.Time
	CPU         CPUMetrics
	Memory      MemoryMetrics
	Network     NetworkMetrics
	Disk        DiskMetrics
}

// CPUMetrics represents CPU usage metrics
type CPUMetrics struct {
	UsagePercent  float64
	ThrottledTime uint64
	SystemCPU     uint64
	UserCPU       uint64
}

// MemoryMetrics represents memory usage metrics
type MemoryMetrics struct {
	UsageBytes   uint64
	LimitBytes   uint64
	UsagePercent float64
	Cache        uint64
	RSS          uint64
}

// NetworkMetrics represents network I/O metrics
type NetworkMetrics struct {
	RxBytes   uint64
	TxBytes   uint64
	RxPackets uint64
	TxPackets uint64
	RxErrors  uint64
	TxErrors  uint64
}

// DiskMetrics represents disk I/O metrics
type DiskMetrics struct {
	ReadBytes  uint64
	WriteBytes uint64
	ReadOps    uint64
	WriteOps   uint64
}

// DockerfileValidation represents the result of Dockerfile validation
type DockerfileValidation struct {
	Valid       bool
	Errors      []ValidationError
	Warnings    []ValidationWarning
	APMReady    bool
	BaseImage   BaseImageInfo
	Suggestions []string
}

// ValidationError represents a Dockerfile validation error
type ValidationError struct {
	Line    int
	Message string
	Rule    string
}

// ValidationWarning represents a Dockerfile validation warning
type ValidationWarning struct {
	Line    int
	Message string
	Rule    string
}

// BaseImageInfo contains information about the base Docker image
type BaseImageInfo struct {
	Name         string
	Tag          string
	OS           string
	Architecture string
	Size         int64
	Layers       int
}

// ScanReport represents the result of an image vulnerability scan
type ScanReport struct {
	ImageID         string
	ScanTime        time.Time
	Critical        int
	High            int
	Medium          int
	Low             int
	Unknown         int
	Vulnerabilities []Vulnerability
}

// Vulnerability represents a security vulnerability
type Vulnerability struct {
	ID          string
	Severity    string
	Package     string
	Version     string
	FixedIn     string
	Description string
	CVE         string
	CVSS        float64
}

// InjectionStrategy represents how APM agents are injected
type InjectionStrategy string

const (
	InjectionStrategyBuildTime InjectionStrategy = "build-time"
	InjectionStrategyRuntime   InjectionStrategy = "runtime"
	InjectionStrategySidecar   InjectionStrategy = "sidecar"
	InjectionStrategyVolume    InjectionStrategy = "volume"
)

// APMAgentConfig represents configuration for APM agent injection
type APMAgentConfig struct {
	Strategy     InjectionStrategy
	AgentImage   string
	AgentVersion string
	ConfigPath   string
	LibraryPath  string
	Environment  map[string]string
}

// RegistryCredentials represents registry authentication credentials
type RegistryCredentials struct {
	Registry string
	Username string
	Password string
	Token    string
	Email    string
	Auth     string // Base64 encoded username:password
}

// DockerComposeConfig represents Docker Compose configuration for APM
type DockerComposeConfig struct {
	Version  string
	Services map[string]ServiceConfig
	Networks map[string]NetworkConfig
	Volumes  map[string]VolumeConfig
}

// ServiceConfig represents a service in Docker Compose
type ServiceConfig struct {
	Image       string
	Build       BuildContext
	Environment map[string]string
	Ports       []string
	Volumes     []string
	Labels      map[string]string
	DependsOn   []string
	HealthCheck HealthCheckConfig
}

// BuildContext represents build configuration in Docker Compose
type BuildContext struct {
	Context    string
	Dockerfile string
	Args       map[string]string
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Test        []string
	Interval    string
	Timeout     string
	Retries     int
	StartPeriod string
}

// NetworkConfig represents network configuration in Docker Compose
type NetworkConfig struct {
	Driver string
	IPAM   IPAMConfig
}

// IPAMConfig represents IP address management configuration
type IPAMConfig struct {
	Driver string
	Config []IPAMPoolConfig
}

// IPAMPoolConfig represents an IP pool configuration
type IPAMPoolConfig struct {
	Subnet string
}

// VolumeConfig represents volume configuration in Docker Compose
type VolumeConfig struct {
	Driver     string
	DriverOpts map[string]string
}
