package docker

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// APMInjector handles APM agent injection into Dockerfiles
type APMInjector struct {
	language    Language
	strategy    InjectionStrategy
	agentConfig APMAgentConfig
}

// NewAPMInjector creates a new APM injector for a specific language
func NewAPMInjector(language Language) *APMInjector {
	return &APMInjector{
		language: language,
		strategy: InjectionStrategyBuildTime,
		agentConfig: APMAgentConfig{
			Strategy: InjectionStrategyBuildTime,
		},
	}
}

// WithStrategy sets the injection strategy
func (i *APMInjector) WithStrategy(strategy InjectionStrategy) *APMInjector {
	i.strategy = strategy
	i.agentConfig.Strategy = strategy
	return i
}

// WithAgentConfig sets the agent configuration
func (i *APMInjector) WithAgentConfig(config APMAgentConfig) *APMInjector {
	i.agentConfig = config
	return i
}

// InjectAgent modifies a Dockerfile to include APM instrumentation
func (i *APMInjector) InjectAgent(dockerfilePath string) (string, error) {
	// Read original Dockerfile
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read Dockerfile: %w", err)
	}

	// Parse Dockerfile
	instructions, err := i.parseDockerfile(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse Dockerfile: %w", err)
	}

	// Detect language if not specified
	if i.language == LanguageUnknown {
		i.language = i.detectLanguage(instructions)
	}

	// Inject APM based on strategy
	switch i.strategy {
	case InjectionStrategyBuildTime:
		return i.injectBuildTime(instructions)
	case InjectionStrategyRuntime:
		return i.injectRuntime(instructions)
	case InjectionStrategySidecar:
		return i.createSidecarConfig(instructions)
	case InjectionStrategyVolume:
		return i.injectVolumeMount(instructions)
	default:
		return "", fmt.Errorf("unsupported injection strategy: %s", i.strategy)
	}
}

// injectBuildTime adds APM agent during Docker build
func (i *APMInjector) injectBuildTime(instructions []DockerInstruction) (string, error) {
	var modifiedInstructions []string
	injected := false

	for _, inst := range instructions {
		// Add instruction to output
		modifiedInstructions = append(modifiedInstructions, inst.Original)

		// Inject APM after FROM instruction
		if inst.Command == "FROM" && !injected {
			apmInstructions := i.getAPMInstructions()
			modifiedInstructions = append(modifiedInstructions, apmInstructions...)
			injected = true
		}

		// Modify ENTRYPOINT or CMD to include agent
		if inst.Command == "ENTRYPOINT" || inst.Command == "CMD" {
			modifiedInstructions[len(modifiedInstructions)-1] = i.wrapCommand(inst)
		}
	}

	return strings.Join(modifiedInstructions, "\n"), nil
}

// getAPMInstructions returns language-specific APM installation instructions
func (i *APMInjector) getAPMInstructions() []string {
	switch i.language {
	case LanguageGo:
		return i.getGoAPMInstructions()
	case LanguageJava:
		return i.getJavaAPMInstructions()
	case LanguagePython:
		return i.getPythonAPMInstructions()
	case LanguageNodeJS:
		return i.getNodeJSAPMInstructions()
	case LanguageRuby:
		return i.getRubyAPMInstructions()
	case LanguagePHP:
		return i.getPHPAPMInstructions()
	case LanguageDotNet:
		return i.getDotNetAPMInstructions()
	default:
		return []string{"# APM injection not supported for this language"}
	}
}

// Go APM instructions
func (i *APMInjector) getGoAPMInstructions() []string {
	return []string{
		"",
		"# Install APM instrumentation for Go",
		"RUN go install github.com/open-telemetry/opentelemetry-go-instrumentation/cmd/otel@latest",
		"",
		"# Copy APM configuration",
		"COPY --from=apm-config /opt/apm/config.yaml /opt/apm/config.yaml",
		"",
		"# Set APM environment variables",
		"ENV OTEL_SERVICE_NAME=${APM_SERVICE_NAME:-go-service}",
		"ENV OTEL_EXPORTER_OTLP_ENDPOINT=${APM_ENDPOINT:-http://localhost:4317}",
		"ENV OTEL_TRACES_EXPORTER=otlp",
		"ENV OTEL_METRICS_EXPORTER=otlp",
		"ENV OTEL_LOGS_EXPORTER=otlp",
		"",
	}
}

// Java APM instructions
func (i *APMInjector) getJavaAPMInstructions() []string {
	return []string{
		"",
		"# Install APM agent for Java",
		"RUN mkdir -p /opt/apm && \\",
		"    wget -O /opt/apm/opentelemetry-javaagent.jar \\",
		"    https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar",
		"",
		"# Set Java agent options",
		"ENV JAVA_TOOL_OPTIONS=\"-javaagent:/opt/apm/opentelemetry-javaagent.jar\"",
		"ENV OTEL_SERVICE_NAME=${APM_SERVICE_NAME:-java-service}",
		"ENV OTEL_EXPORTER_OTLP_ENDPOINT=${APM_ENDPOINT:-http://localhost:4317}",
		"ENV OTEL_TRACES_EXPORTER=otlp",
		"ENV OTEL_METRICS_EXPORTER=otlp",
		"ENV OTEL_LOGS_EXPORTER=otlp",
		"",
	}
}

// Python APM instructions
func (i *APMInjector) getPythonAPMInstructions() []string {
	return []string{
		"",
		"# Install APM instrumentation for Python",
		"RUN pip install opentelemetry-distro[otlp] && \\",
		"    opentelemetry-bootstrap --action=install",
		"",
		"# Set APM environment variables",
		"ENV OTEL_SERVICE_NAME=${APM_SERVICE_NAME:-python-service}",
		"ENV OTEL_EXPORTER_OTLP_ENDPOINT=${APM_ENDPOINT:-http://localhost:4317}",
		"ENV OTEL_TRACES_EXPORTER=otlp",
		"ENV OTEL_METRICS_EXPORTER=otlp",
		"ENV OTEL_LOGS_EXPORTER=otlp",
		"ENV OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED=true",
		"",
	}
}

// Node.js APM instructions
func (i *APMInjector) getNodeJSAPMInstructions() []string {
	return []string{
		"",
		"# Install APM instrumentation for Node.js",
		"RUN npm install --save @opentelemetry/api \\",
		"    @opentelemetry/auto-instrumentations-node \\",
		"    @opentelemetry/sdk-node",
		"",
		"# Copy APM initialization script",
		"COPY --from=apm-config /opt/apm/tracing.js /opt/apm/tracing.js",
		"",
		"# Set APM environment variables",
		"ENV NODE_OPTIONS=\"--require /opt/apm/tracing.js\"",
		"ENV OTEL_SERVICE_NAME=${APM_SERVICE_NAME:-nodejs-service}",
		"ENV OTEL_EXPORTER_OTLP_ENDPOINT=${APM_ENDPOINT:-http://localhost:4317}",
		"ENV OTEL_TRACES_EXPORTER=otlp",
		"ENV OTEL_METRICS_EXPORTER=otlp",
		"ENV OTEL_LOGS_EXPORTER=otlp",
		"",
	}
}

// Ruby APM instructions
func (i *APMInjector) getRubyAPMInstructions() []string {
	return []string{
		"",
		"# Install APM instrumentation for Ruby",
		"RUN gem install opentelemetry-sdk opentelemetry-exporter-otlp \\",
		"    opentelemetry-instrumentation-all",
		"",
		"# Set APM environment variables",
		"ENV OTEL_SERVICE_NAME=${APM_SERVICE_NAME:-ruby-service}",
		"ENV OTEL_EXPORTER_OTLP_ENDPOINT=${APM_ENDPOINT:-http://localhost:4317}",
		"ENV OTEL_TRACES_EXPORTER=otlp",
		"ENV OTEL_METRICS_EXPORTER=otlp",
		"",
	}
}

// PHP APM instructions
func (i *APMInjector) getPHPAPMInstructions() []string {
	return []string{
		"",
		"# Install APM extension for PHP",
		"RUN pecl install opentelemetry && \\",
		"    docker-php-ext-enable opentelemetry",
		"",
		"# Configure PHP for APM",
		"RUN echo 'opentelemetry.enabled = On' >> /usr/local/etc/php/conf.d/opentelemetry.ini",
		"",
		"# Set APM environment variables",
		"ENV OTEL_SERVICE_NAME=${APM_SERVICE_NAME:-php-service}",
		"ENV OTEL_EXPORTER_OTLP_ENDPOINT=${APM_ENDPOINT:-http://localhost:4317}",
		"ENV OTEL_TRACES_EXPORTER=otlp",
		"ENV OTEL_PHP_AUTOLOAD_ENABLED=true",
		"",
	}
}

// .NET APM instructions
func (i *APMInjector) getDotNetAPMInstructions() []string {
	return []string{
		"",
		"# Install APM instrumentation for .NET",
		"RUN dotnet tool install --global dotnet-trace && \\",
		"    dotnet tool install --global dotnet-counters",
		"",
		"# Download and install OpenTelemetry .NET Auto-instrumentation",
		"RUN curl -L https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/latest/download/otel-dotnet-auto-install.sh -o /tmp/otel-dotnet-auto-install.sh && \\",
		"    chmod +x /tmp/otel-dotnet-auto-install.sh && \\",
		"    /tmp/otel-dotnet-auto-install.sh",
		"",
		"# Set APM environment variables",
		"ENV OTEL_SERVICE_NAME=${APM_SERVICE_NAME:-dotnet-service}",
		"ENV OTEL_EXPORTER_OTLP_ENDPOINT=${APM_ENDPOINT:-http://localhost:4317}",
		"ENV OTEL_TRACES_EXPORTER=otlp",
		"ENV OTEL_METRICS_EXPORTER=otlp",
		"ENV OTEL_LOGS_EXPORTER=otlp",
		"ENV CORECLR_ENABLE_PROFILING=1",
		"ENV CORECLR_PROFILER={918728DD-259F-4A6A-AC2B-B85E1B658318}",
		"ENV CORECLR_PROFILER_PATH=/opt/opentelemetry/OpenTelemetry.AutoInstrumentation.Native.so",
		"ENV DOTNET_ADDITIONAL_DEPS=/opt/opentelemetry/AdditionalDeps",
		"ENV DOTNET_SHARED_STORE=/opt/opentelemetry/store",
		"ENV DOTNET_STARTUP_HOOKS=/opt/opentelemetry/net/OpenTelemetry.AutoInstrumentation.StartupHook.dll",
		"",
	}
}

// wrapCommand modifies ENTRYPOINT/CMD to include APM agent
func (i *APMInjector) wrapCommand(inst DockerInstruction) string {
	switch i.language {
	case LanguageJava:
		// Java uses JAVA_TOOL_OPTIONS, no wrapping needed
		return inst.Original
	case LanguagePython:
		return i.wrapPythonCommand(inst)
	case LanguageNodeJS:
		// Node.js uses NODE_OPTIONS, no wrapping needed
		return inst.Original
	default:
		return inst.Original
	}
}

// wrapPythonCommand wraps Python commands with opentelemetry-instrument
func (i *APMInjector) wrapPythonCommand(inst DockerInstruction) string {
	if strings.Contains(inst.Value, "python") {
		// Replace python with opentelemetry-instrument python
		wrapped := strings.Replace(inst.Value, "python", "opentelemetry-instrument python", 1)
		return fmt.Sprintf("%s %s", inst.Command, wrapped)
	}
	return inst.Original
}

// detectLanguage tries to detect the programming language from Dockerfile
func (i *APMInjector) detectLanguage(instructions []DockerInstruction) Language {
	for _, inst := range instructions {
		value := strings.ToLower(inst.Value)

		// Check FROM instruction
		if inst.Command == "FROM" {
			if strings.Contains(value, "golang") || strings.Contains(value, "go:") {
				return LanguageGo
			}
			if strings.Contains(value, "openjdk") || strings.Contains(value, "java") || strings.Contains(value, "maven") || strings.Contains(value, "gradle") {
				return LanguageJava
			}
			if strings.Contains(value, "python") {
				return LanguagePython
			}
			if strings.Contains(value, "node") {
				return LanguageNodeJS
			}
			if strings.Contains(value, "ruby") {
				return LanguageRuby
			}
			if strings.Contains(value, "php") {
				return LanguagePHP
			}
			if strings.Contains(value, "dotnet") || strings.Contains(value, "aspnet") {
				return LanguageDotNet
			}
		}

		// Check RUN instructions for package managers
		if inst.Command == "RUN" {
			if strings.Contains(value, "go mod") || strings.Contains(value, "go build") {
				return LanguageGo
			}
			if strings.Contains(value, "mvn") || strings.Contains(value, "gradle") || strings.Contains(value, "jar") {
				return LanguageJava
			}
			if strings.Contains(value, "pip") || strings.Contains(value, "python") {
				return LanguagePython
			}
			if strings.Contains(value, "npm") || strings.Contains(value, "yarn") || strings.Contains(value, "node") {
				return LanguageNodeJS
			}
			if strings.Contains(value, "gem") || strings.Contains(value, "bundle") {
				return LanguageRuby
			}
			if strings.Contains(value, "composer") || strings.Contains(value, "php") {
				return LanguagePHP
			}
			if strings.Contains(value, "dotnet") {
				return LanguageDotNet
			}
		}
	}

	return LanguageUnknown
}

// DockerInstruction represents a single Dockerfile instruction
type DockerInstruction struct {
	Command  string
	Value    string
	Original string
	Line     int
}

// parseDockerfile parses Dockerfile content into instructions
func (i *APMInjector) parseDockerfile(content []byte) ([]DockerInstruction, error) {
	var instructions []DockerInstruction
	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0

	var currentInstruction *DockerInstruction

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			if currentInstruction == nil {
				instructions = append(instructions, DockerInstruction{
					Original: line,
					Line:     lineNum,
				})
			}
			continue
		}

		// Handle line continuation
		if strings.HasSuffix(trimmed, "\\") {
			if currentInstruction == nil {
				parts := strings.SplitN(trimmed, " ", 2)
				currentInstruction = &DockerInstruction{
					Command:  strings.ToUpper(parts[0]),
					Value:    strings.TrimSuffix(parts[1], "\\"),
					Original: line,
					Line:     lineNum,
				}
			} else {
				currentInstruction.Value += " " + strings.TrimSuffix(trimmed, "\\")
				currentInstruction.Original += "\n" + line
			}
			continue
		}

		// Complete instruction
		if currentInstruction != nil {
			currentInstruction.Value += " " + trimmed
			currentInstruction.Original += "\n" + line
			instructions = append(instructions, *currentInstruction)
			currentInstruction = nil
		} else {
			parts := strings.SplitN(trimmed, " ", 2)
			inst := DockerInstruction{
				Command:  strings.ToUpper(parts[0]),
				Original: line,
				Line:     lineNum,
			}
			if len(parts) > 1 {
				inst.Value = parts[1]
			}
			instructions = append(instructions, inst)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning Dockerfile: %w", err)
	}

	return instructions, nil
}

// createSidecarConfig creates a sidecar container configuration
func (i *APMInjector) createSidecarConfig(instructions []DockerInstruction) (string, error) {
	// Generate docker-compose snippet for sidecar
	tmpl := `
# APM Sidecar Configuration
# Add this to your docker-compose.yml or Kubernetes manifest

apm-agent:
  image: {{ .AgentImage }}
  environment:
    - OTEL_SERVICE_NAME={{ .ServiceName }}
    - OTEL_EXPORTER_OTLP_ENDPOINT={{ .Endpoint }}
  volumes:
    - apm-shared:/var/run/apm
  networks:
    - app-network

# Modify your application container:
# volumes:
#   - apm-shared:/var/run/apm
# environment:
#   - APM_AGENT_PATH=/var/run/apm/agent
`

	t, err := template.New("sidecar").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	data := map[string]string{
		"AgentImage":  i.agentConfig.AgentImage,
		"ServiceName": "${APM_SERVICE_NAME}",
		"Endpoint":    "${APM_ENDPOINT}",
	}

	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// injectRuntime creates runtime injection configuration
func (i *APMInjector) injectRuntime(instructions []DockerInstruction) (string, error) {
	// This would generate Kubernetes init container configuration
	// or Docker Compose depends_on configuration
	return "", fmt.Errorf("runtime injection not yet implemented")
}

// injectVolumeMount adds volume mount for shared APM libraries
func (i *APMInjector) injectVolumeMount(instructions []DockerInstruction) (string, error) {
	var modifiedInstructions []string

	for _, inst := range instructions {
		modifiedInstructions = append(modifiedInstructions, inst.Original)

		// Add volume after FROM
		if inst.Command == "FROM" {
			modifiedInstructions = append(modifiedInstructions,
				"",
				"# Mount APM agent libraries",
				"VOLUME [\"/opt/apm\"]",
				"",
			)
		}
	}

	return strings.Join(modifiedInstructions, "\n"), nil
}

// GenerateInitScript generates initialization script for APM
func (i *APMInjector) GenerateInitScript(language Language, outputPath string) error {
	var script string

	switch language {
	case LanguageNodeJS:
		script = i.generateNodeJSInitScript()
	case LanguagePython:
		script = i.generatePythonInitScript()
	default:
		return fmt.Errorf("init script not supported for %s", language)
	}

	return os.WriteFile(filepath.Join(outputPath, "apm-init.sh"), []byte(script), 0755)
}

// generateNodeJSInitScript creates Node.js tracing initialization
func (i *APMInjector) generateNodeJSInitScript() string {
	return `// APM tracing initialization for Node.js
const { NodeSDK } = require('@opentelemetry/sdk-node');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');
const { Resource } = require('@opentelemetry/resources');
const { SemanticResourceAttributes } = require('@opentelemetry/semantic-conventions');

const sdk = new NodeSDK({
  resource: new Resource({
    [SemanticResourceAttributes.SERVICE_NAME]: process.env.OTEL_SERVICE_NAME || 'nodejs-app',
  }),
  instrumentations: [getNodeAutoInstrumentations()]
});

sdk.start()
  .then(() => console.log('APM tracing initialized successfully'))
  .catch((error) => console.log('Error initializing APM tracing', error));

process.on('SIGTERM', () => {
  sdk.shutdown()
    .then(() => console.log('APM tracing terminated'))
    .catch((error) => console.log('Error terminating APM tracing', error))
    .finally(() => process.exit(0));
});
`
}

// generatePythonInitScript creates Python tracing initialization
func (i *APMInjector) generatePythonInitScript() string {
	return `# APM tracing initialization for Python
import os
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

# Configure the tracer
resource = Resource.create({
    "service.name": os.environ.get("OTEL_SERVICE_NAME", "python-app")
})

provider = TracerProvider(resource=resource)
processor = BatchSpanProcessor(OTLPSpanExporter())
provider.add_span_processor(processor)

# Set the global tracer provider
trace.set_tracer_provider(provider)

print("APM tracing initialized successfully")
`
}
