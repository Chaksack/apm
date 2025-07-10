package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DockerDeployer handles Docker-based deployments
type DockerDeployer struct {
	config       DeploymentConfig
	dockerClient DockerClient
}

// DeploymentConfig holds deployment configuration
type DeploymentConfig struct {
	Dockerfile  string
	ImageName   string
	ImageTag    string
	Registry    string
	BuildArgs   map[string]string
	APMConfig   APMConfig
	Environment string
	ServiceName string
}

// APMConfig holds APM instrumentation settings
type APMConfig struct {
	InjectAgent bool
	AgentType   string // opentelemetry, datadog, newrelic
	ServiceName string
	Environment string
	Endpoint    string
}

// DockerClient interface for Docker operations
type DockerClient interface {
	Build(ctx context.Context, dockerfile, tag string, buildArgs map[string]string) error
	Push(ctx context.Context, image string) error
	Login(ctx context.Context, registry, username, password string) error
}

// NewDockerDeployer creates a new Docker deployer
func NewDockerDeployer(config DeploymentConfig) *DockerDeployer {
	return &DockerDeployer{
		config:       config,
		dockerClient: &CLIDockerClient{},
	}
}

// Deploy performs the Docker deployment
func (d *DockerDeployer) Deploy(ctx context.Context) error {
	// 1. Validate Dockerfile exists
	if err := d.validateDockerfile(); err != nil {
		return fmt.Errorf("dockerfile validation failed: %w", err)
	}

	// 2. Inject APM if needed
	if d.config.APMConfig.InjectAgent {
		if err := d.injectAPMAgent(); err != nil {
			return fmt.Errorf("APM injection failed: %w", err)
		}
	}

	// 3. Build image
	fullImageName := fmt.Sprintf("%s:%s", d.config.ImageName, d.config.ImageTag)
	if d.config.Registry != "" {
		fullImageName = fmt.Sprintf("%s/%s", d.config.Registry, fullImageName)
	}

	if err := d.dockerClient.Build(ctx, d.config.Dockerfile, fullImageName, d.config.BuildArgs); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	// 4. Push to registry if specified
	if d.config.Registry != "" {
		if err := d.dockerClient.Push(ctx, fullImageName); err != nil {
			return fmt.Errorf("docker push failed: %w", err)
		}
	}

	return nil
}

// validateDockerfile checks if the Dockerfile exists
func (d *DockerDeployer) validateDockerfile() error {
	if _, err := os.Stat(d.config.Dockerfile); os.IsNotExist(err) {
		return fmt.Errorf("dockerfile not found: %s", d.config.Dockerfile)
	}
	return nil
}

// injectAPMAgent modifies the Dockerfile to include APM instrumentation
func (d *DockerDeployer) injectAPMAgent() error {
	// Read original Dockerfile
	content, err := os.ReadFile(d.config.Dockerfile)
	if err != nil {
		return fmt.Errorf("failed to read dockerfile: %w", err)
	}

	// Detect base image and language
	language := detectLanguage(string(content))

	// Generate APM injection based on language
	apmInjection := generateAPMInjection(language, d.config.APMConfig)

	// Create temporary Dockerfile with APM
	tmpFile := d.config.Dockerfile + ".apm"
	modifiedContent := injectAPMIntoDockerfile(string(content), apmInjection)

	if err := os.WriteFile(tmpFile, []byte(modifiedContent), 0644); err != nil {
		return fmt.Errorf("failed to write modified dockerfile: %w", err)
	}

	// Update config to use modified Dockerfile
	d.config.Dockerfile = tmpFile

	return nil
}

// detectLanguage attempts to detect the programming language from Dockerfile
func detectLanguage(dockerfile string) string {
	lines := strings.Split(dockerfile, "\n")
	for _, line := range lines {
		line = strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(line, "from") {
			if strings.Contains(line, "golang") || strings.Contains(line, "go:") {
				return "go"
			} else if strings.Contains(line, "node") {
				return "nodejs"
			} else if strings.Contains(line, "python") {
				return "python"
			} else if strings.Contains(line, "java") || strings.Contains(line, "openjdk") {
				return "java"
			}
		}
	}
	return "unknown"
}

// generateAPMInjection creates APM-specific Dockerfile instructions
func generateAPMInjection(language string, config APMConfig) string {
	switch config.AgentType {
	case "opentelemetry":
		return generateOpenTelemetryInjection(language, config)
	default:
		return generateDefaultAPMInjection(language, config)
	}
}

// generateOpenTelemetryInjection creates OpenTelemetry-specific injection
func generateOpenTelemetryInjection(language string, config APMConfig) string {
	envVars := fmt.Sprintf(`
ENV OTEL_SERVICE_NAME=%s
ENV OTEL_ENVIRONMENT=%s
ENV OTEL_EXPORTER_OTLP_ENDPOINT=%s
ENV OTEL_TRACES_EXPORTER=otlp
ENV OTEL_METRICS_EXPORTER=otlp
ENV OTEL_LOGS_EXPORTER=otlp
`, config.ServiceName, config.Environment, config.Endpoint)

	switch language {
	case "go":
		return fmt.Sprintf(`
# APM Agent Installation
RUN go install go.opentelemetry.io/auto/go-auto-instrumentation@latest
%s
`, envVars)

	case "nodejs":
		return fmt.Sprintf(`
# APM Agent Installation
RUN npm install --save @opentelemetry/api @opentelemetry/auto-instrumentations-node
%s
ENV NODE_OPTIONS="--require @opentelemetry/auto-instrumentations-node/register"
`, envVars)

	case "python":
		return fmt.Sprintf(`
# APM Agent Installation
RUN pip install opentelemetry-distro opentelemetry-exporter-otlp
RUN opentelemetry-bootstrap --action=install
%s
ENV PYTHONPATH="${PYTHONPATH}:/usr/local/lib/python3.9/site-packages"
`, envVars)

	case "java":
		return fmt.Sprintf(`
# APM Agent Installation
RUN wget https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar
%s
ENV JAVA_TOOL_OPTIONS="-javaagent:/opentelemetry-javaagent.jar"
`, envVars)

	default:
		return envVars
	}
}

// generateDefaultAPMInjection creates generic APM injection
func generateDefaultAPMInjection(language string, config APMConfig) string {
	return fmt.Sprintf(`
# APM Configuration
ENV SERVICE_NAME=%s
ENV ENVIRONMENT=%s
ENV APM_ENABLED=true
`, config.ServiceName, config.Environment)
}

// injectAPMIntoDockerfile injects APM instructions into the Dockerfile
func injectAPMIntoDockerfile(dockerfile, injection string) string {
	lines := strings.Split(dockerfile, "\n")
	result := []string{}
	injected := false

	for _, line := range lines {
		result = append(result, line)

		// Inject after the last FROM statement (for multi-stage builds)
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(line)), "FROM") {
			// Mark position for injection
			injected = false
		} else if !injected && !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(line)), "FROM") {
			// Inject after FROM but before other instructions
			result = append(result, injection)
			injected = true
		}
	}

	return strings.Join(result, "\n")
}

// CLIDockerClient implements DockerClient using Docker CLI
type CLIDockerClient struct{}

// Build builds a Docker image using CLI
func (c *CLIDockerClient) Build(ctx context.Context, dockerfile, tag string, buildArgs map[string]string) error {
	args := []string{"build", "-f", dockerfile, "-t", tag}

	// Add build arguments
	for k, v := range buildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
	}

	// Add context directory (parent of Dockerfile)
	args = append(args, filepath.Dir(dockerfile))

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Push pushes a Docker image to registry
func (c *CLIDockerClient) Push(ctx context.Context, image string) error {
	cmd := exec.CommandContext(ctx, "docker", "push", image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Login logs into a Docker registry
func (c *CLIDockerClient) Login(ctx context.Context, registry, username, password string) error {
	cmd := exec.CommandContext(ctx, "docker", "login", registry, "-u", username, "--password-stdin")
	cmd.Stdin = strings.NewReader(password)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
