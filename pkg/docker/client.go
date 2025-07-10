package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
)

// Client wraps the Docker client with APM-specific functionality
type Client struct {
	cli         *client.Client
	registry    RegistryConfig
	buildConfig BuildConfig
}

// RegistryConfig holds registry authentication details
type RegistryConfig struct {
	ServerAddress string
	Username      string
	Password      string
	Email         string
	AuthConfig    map[string]registry.AuthConfig // Multiple registry support
}

// BuildConfig holds build-time configuration
type BuildConfig struct {
	BuildArgs   map[string]*string
	Labels      map[string]string
	CacheFrom   []string
	NetworkMode string
	Platform    string
	Target      string
	NoCache     bool
	ForceRemove bool
	PullParent  bool
	Squash      bool
	CPUShares   int64
	Memory      int64
	ShmSize     int64
}

// NewClient creates a new Docker client with APM integration
func NewClient(opts ...ClientOption) (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	c := &Client{
		cli: cli,
		buildConfig: BuildConfig{
			BuildArgs: make(map[string]*string),
			Labels:    make(map[string]string),
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	// Add default APM labels
	c.buildConfig.Labels["apm.enabled"] = "true"
	c.buildConfig.Labels["apm.version"] = "1.0.0"

	return c, nil
}

// ClientOption is a functional option for configuring the client
type ClientOption func(*Client)

// WithRegistry configures registry authentication
func WithRegistry(config RegistryConfig) ClientOption {
	return func(c *Client) {
		c.registry = config
	}
}

// WithBuildConfig sets build configuration
func WithBuildConfig(config BuildConfig) ClientOption {
	return func(c *Client) {
		c.buildConfig = config
	}
}

// BuildWithAPM builds a Docker image with APM instrumentation
func (c *Client) BuildWithAPM(ctx context.Context, dockerfilePath string, options BuildOptions) (string, error) {
	// Validate Dockerfile for APM compatibility
	if err := c.ValidateDockerfile(dockerfilePath); err != nil {
		return "", fmt.Errorf("dockerfile validation failed: %w", err)
	}

	// Inject APM agent based on detected language
	modifiedDockerfile, err := c.InjectAPMAgent(dockerfilePath, options.Language)
	if err != nil {
		return "", fmt.Errorf("failed to inject APM agent: %w", err)
	}

	// Prepare build context
	buildContext, err := CreateBuildContext(modifiedDockerfile, options.ContextPath)
	if err != nil {
		return "", fmt.Errorf("failed to create build context: %w", err)
	}
	defer buildContext.Close()

	// Configure build options
	buildOpts := types.ImageBuildOptions{
		Dockerfile:  modifiedDockerfile,
		Tags:        options.Tags,
		BuildArgs:   c.buildConfig.BuildArgs,
		Labels:      c.buildConfig.Labels,
		CacheFrom:   c.buildConfig.CacheFrom,
		NetworkMode: c.buildConfig.NetworkMode,
		Platform:    c.buildConfig.Platform,
		Target:      c.buildConfig.Target,
		NoCache:     c.buildConfig.NoCache,
		Remove:      true,
		ForceRemove: c.buildConfig.ForceRemove,
		PullParent:  c.buildConfig.PullParent,
		Squash:      c.buildConfig.Squash,
		CPUShares:   c.buildConfig.CPUShares,
		Memory:      c.buildConfig.Memory,
		ShmSize:     c.buildConfig.ShmSize,
	}

	// Add APM-specific build args
	buildOpts.BuildArgs["APM_ENABLED"] = stringPtr("true")
	buildOpts.BuildArgs["APM_SERVICE_NAME"] = stringPtr(options.ServiceName)
	buildOpts.BuildArgs["APM_ENVIRONMENT"] = stringPtr(options.Environment)

	// Build the image
	resp, err := c.cli.ImageBuild(ctx, buildContext, buildOpts)
	if err != nil {
		return "", fmt.Errorf("failed to build image: %w", err)
	}
	defer resp.Body.Close()

	// Stream build output
	if err := c.streamBuildOutput(resp.Body, options.OutputStream); err != nil {
		return "", fmt.Errorf("build failed: %w", err)
	}

	// Get image ID
	imageID := c.getImageID(options.Tags[0])

	// Scan image for vulnerabilities if enabled
	if options.ScanImage {
		if err := c.ScanImage(ctx, imageID); err != nil {
			return "", fmt.Errorf("image scan failed: %w", err)
		}
	}

	return imageID, nil
}

// PushToRegistry pushes an image to a container registry
func (c *Client) PushToRegistry(ctx context.Context, imageTag string, registryType RegistryType) error {
	// Get registry-specific auth
	authConfig, err := c.getRegistryAuth(registryType)
	if err != nil {
		return fmt.Errorf("failed to get registry auth: %w", err)
	}

	encodedAuth, err := encodeAuthConfig(authConfig)
	if err != nil {
		return fmt.Errorf("failed to encode auth: %w", err)
	}

	// Push the image
	pushOpts := types.ImagePushOptions{
		RegistryAuth: encodedAuth,
	}

	reader, err := c.cli.ImagePush(ctx, imageTag, pushOpts)
	if err != nil {
		return fmt.Errorf("failed to push image: %w", err)
	}
	defer reader.Close()

	// Stream push output
	return c.streamPushOutput(reader)
}

// InjectAPMAgent modifies a Dockerfile to include APM instrumentation
func (c *Client) InjectAPMAgent(dockerfilePath string, language Language) (string, error) {
	injector := NewAPMInjector(language)
	return injector.InjectAgent(dockerfilePath)
}

// ValidateDockerfile checks if a Dockerfile is compatible with APM
func (c *Client) ValidateDockerfile(dockerfilePath string) error {
	validator := NewDockerfileValidator()
	return validator.Validate(dockerfilePath)
}

// ScanImage scans a Docker image for vulnerabilities
func (c *Client) ScanImage(ctx context.Context, imageID string) error {
	scanner := NewImageScanner(c.cli)
	report, err := scanner.Scan(ctx, imageID)
	if err != nil {
		return err
	}

	// Check vulnerability thresholds
	if report.Critical > 0 {
		return fmt.Errorf("image has %d critical vulnerabilities", report.Critical)
	}

	return nil
}

// ListContainersWithAPM lists all containers with APM instrumentation
func (c *Client) ListContainersWithAPM(ctx context.Context) ([]types.Container, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", "apm.enabled=true")

	return c.cli.ContainerList(ctx, container.ListOptions{
		Filters: filterArgs,
		All:     true,
	})
}

// GetContainerAPMMetrics retrieves APM metrics from a container
func (c *Client) GetContainerAPMMetrics(ctx context.Context, containerID string) (*ContainerMetrics, error) {
	// Get container stats
	stats, err := c.cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer stats.Body.Close()

	// Parse stats
	var statsJSON types.StatsJSON
	if err := json.NewDecoder(stats.Body).Decode(&statsJSON); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	// Extract APM-relevant metrics
	metrics := &ContainerMetrics{
		ContainerID: containerID,
		Timestamp:   time.Now(),
		CPU:         calculateCPUPercent(&statsJSON),
		Memory:      calculateMemoryUsage(&statsJSON),
		Network:     extractNetworkStats(&statsJSON),
		Disk:        extractDiskStats(&statsJSON),
	}

	return metrics, nil
}

// Helper functions

func (c *Client) streamBuildOutput(reader io.Reader, writer io.Writer) error {
	decoder := json.NewDecoder(reader)
	for {
		var msg jsonmessage.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if msg.Error != nil {
			return msg.Error
		}

		if writer != nil && msg.Stream != "" {
			fmt.Fprint(writer, msg.Stream)
		}
	}
	return nil
}

func (c *Client) streamPushOutput(reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	for {
		var msg jsonmessage.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if msg.Error != nil {
			return msg.Error
		}
	}
	return nil
}

func (c *Client) getImageID(tag string) string {
	// Extract image ID from tag or use inspection
	parts := strings.Split(tag, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return tag
}

func (c *Client) getRegistryAuth(registryType RegistryType) (registry.AuthConfig, error) {
	switch registryType {
	case RegistryTypeDockerHub:
		return registry.AuthConfig{
			Username:      c.registry.Username,
			Password:      c.registry.Password,
			ServerAddress: "https://index.docker.io/v1/",
		}, nil
	case RegistryTypeECR:
		return c.getECRAuth()
	case RegistryTypeGCR:
		return c.getGCRAuth()
	case RegistryTypeACR:
		return c.getACRAuth()
	default:
		return registry.AuthConfig{}, fmt.Errorf("unsupported registry type: %s", registryType)
	}
}

func encodeAuthConfig(authConfig registry.AuthConfig) (string, error) {
	authJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(authJSON), nil
}

func stringPtr(s string) *string {
	return &s
}

// Close closes the Docker client connection
func (c *Client) Close() error {
	return c.cli.Close()
}
