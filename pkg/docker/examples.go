package docker

import (
	"context"
	"fmt"
	"log"
	"os"
)

// Example demonstrates various Docker deployment features
func Example() {
	ctx := context.Background()

	// 1. Create Docker client with registry configuration
	client, err := NewClient(
		WithRegistry(RegistryConfig{
			Username: os.Getenv("DOCKER_USERNAME"),
			Password: os.Getenv("DOCKER_PASSWORD"),
			AuthConfig: map[string]registry.AuthConfig{
				"ecr": {
					Username:      "AWS",
					Password:      os.Getenv("ECR_TOKEN"),
					ServerAddress: "123456789.dkr.ecr.us-east-1.amazonaws.com",
				},
			},
		}),
		WithBuildConfig(BuildConfig{
			BuildArgs: map[string]*string{
				"GO_VERSION":  stringPtr("1.21"),
				"APP_VERSION": stringPtr("v1.0.0"),
			},
			Labels: map[string]string{
				"app.name": "my-service",
				"app.team": "platform",
			},
			Platform: "linux/amd64,linux/arm64",
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// 2. Validate Dockerfile before building
	validator := NewDockerfileValidator()
	validation, err := validator.ValidateAndSuggest("./Dockerfile")
	if err != nil {
		log.Printf("Validation failed: %v", err)
	}

	if !validation.Valid {
		for _, e := range validation.Errors {
			log.Printf("ERROR [Line %d]: %s", e.Line, e.Message)
		}
		return
	}

	for _, w := range validation.Warnings {
		log.Printf("WARNING [Line %d]: %s", w.Line, w.Message)
	}

	// 3. Build image with APM integration
	buildOpts := BuildOptions{
		ContextPath: ".",
		Tags: []string{
			"myapp:latest",
			"myapp:v1.0.0",
			"123456789.dkr.ecr.us-east-1.amazonaws.com/myapp:latest",
		},
		ServiceName:  "my-service",
		Environment:  "production",
		Language:     LanguageGo, // Auto-detected if not specified
		ScanImage:    true,
		OutputStream: os.Stdout,
		APMConfig: APMConfig{
			Enabled:      true,
			AgentVersion: "latest",
			Endpoint:     "http://jaeger:4317",
			SamplingRate: 1.0,
			LogLevel:     "info",
			Features: APMFeatures{
				Metrics:       true,
				Tracing:       true,
				Logging:       true,
				Profiling:     false,
				ErrorTracking: true,
			},
		},
	}

	imageID, err := client.BuildWithAPM(ctx, "./Dockerfile", buildOpts)
	if err != nil {
		log.Fatalf("Build failed: %v", err)
	}
	log.Printf("Built image: %s", imageID)

	// 4. Push to multiple registries
	registries := []struct {
		Type RegistryType
		Tag  string
	}{
		{RegistryTypeDockerHub, "myapp:latest"},
		{RegistryTypeECR, "123456789.dkr.ecr.us-east-1.amazonaws.com/myapp:latest"},
	}

	for _, reg := range registries {
		log.Printf("Pushing to %s...", reg.Type)
		if err := client.PushToRegistry(ctx, reg.Tag, reg.Type); err != nil {
			log.Printf("Failed to push to %s: %v", reg.Type, err)
		}
	}

	// 5. List containers with APM
	containers, err := client.ListContainersWithAPM(ctx)
	if err != nil {
		log.Printf("Failed to list containers: %v", err)
	}

	for _, container := range containers {
		log.Printf("Container: %s (Image: %s)", container.Names[0], container.Image)

		// Get APM metrics
		metrics, err := client.GetContainerAPMMetrics(ctx, container.ID)
		if err != nil {
			log.Printf("Failed to get metrics: %v", err)
			continue
		}

		log.Printf("  CPU: %.2f%%", metrics.CPU.UsagePercent)
		log.Printf("  Memory: %.2f%% (%d MB)",
			metrics.Memory.UsagePercent,
			metrics.Memory.UsageBytes/1024/1024)
	}
}

// ExampleDockerCompose demonstrates Docker Compose with APM
func ExampleDockerCompose() {
	composeConfig := `
version: '3.8'

services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
      args:
        APM_ENABLED: "true"
        APM_SERVICE_NAME: "my-service"
    image: myapp:latest
    environment:
      # OpenTelemetry configuration
      OTEL_SERVICE_NAME: ${APM_SERVICE_NAME:-my-service}
      OTEL_EXPORTER_OTLP_ENDPOINT: http://jaeger:4317
      OTEL_TRACES_EXPORTER: otlp
      OTEL_METRICS_EXPORTER: prometheus
      OTEL_LOGS_EXPORTER: otlp
      # Application configuration
      APP_ENV: production
      LOG_LEVEL: info
    ports:
      - "8080:8080"
      - "9090:9090"  # Prometheus metrics
    labels:
      - "apm.enabled=true"
      - "prometheus.io/scrape=true"
      - "prometheus.io/port=9090"
    depends_on:
      - jaeger
      - prometheus
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  # APM Sidecar for legacy applications
  apm-agent:
    image: opentelemetry/opentelemetry-collector:latest
    command: ["--config=/etc/otel/config.yaml"]
    volumes:
      - ./otel-collector-config.yaml:/etc/otel/config.yaml:ro
      - apm-data:/var/lib/otel
    environment:
      - OTEL_RESOURCE_ATTRIBUTES=service.name=legacy-app
    networks:
      - apm-network

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP
    environment:
      - COLLECTOR_OTLP_ENABLED=true
    networks:
      - apm-network

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
    networks:
      - apm-network

networks:
  apm-network:
    driver: bridge

volumes:
  apm-data:
`
	fmt.Println(composeConfig)
}

// ExampleKubernetesInitContainer demonstrates init container pattern
func ExampleKubernetesInitContainer() {
	k8sManifest := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
  labels:
    app: myapp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
        apm.enabled: "true"
    spec:
      initContainers:
      - name: apm-agent-installer
        image: apm/agent-installer:latest
        command:
        - sh
        - -c
        - |
          cp -r /opt/apm/agent /shared/
          cp /opt/apm/config.yaml /shared/
        volumeMounts:
        - name: shared-data
          mountPath: /shared
      containers:
      - name: app
        image: myapp:latest
        env:
        - name: JAVA_TOOL_OPTIONS
          value: "-javaagent:/opt/apm/agent/opentelemetry-javaagent.jar"
        - name: OTEL_SERVICE_NAME
          value: "myapp"
        - name: OTEL_EXPORTER_OTLP_ENDPOINT
          value: "http://jaeger-collector:4317"
        volumeMounts:
        - name: shared-data
          mountPath: /opt/apm
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: shared-data
        emptyDir: {}
`
	fmt.Println(k8sManifest)
}

// ExampleLanguageSpecificInjection shows language-specific APM injection
func ExampleLanguageSpecificInjection() {
	examples := map[Language]string{
		LanguageGo: `
# Go with OpenTelemetry auto-instrumentation
FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o myapp .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/myapp /myapp
ENV OTEL_SERVICE_NAME=go-service
ENV OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
EXPOSE 8080
ENTRYPOINT ["/myapp"]
`,
		LanguageJava: `
# Java with OpenTelemetry Java Agent
FROM openjdk:17-jdk-slim AS builder
WORKDIR /app
COPY . .
RUN ./gradlew build

FROM openjdk:17-jre-slim
RUN mkdir -p /opt/apm
ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar /opt/apm/opentelemetry-javaagent.jar
COPY --from=builder /app/build/libs/myapp.jar /app.jar
ENV JAVA_TOOL_OPTIONS="-javaagent:/opt/apm/opentelemetry-javaagent.jar"
ENV OTEL_SERVICE_NAME=java-service
ENV OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
EXPOSE 8080
ENTRYPOINT ["java", "-jar", "/app.jar"]
`,
		LanguagePython: `
# Python with OpenTelemetry auto-instrumentation
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
RUN pip install opentelemetry-distro[otlp] opentelemetry-instrumentation
RUN opentelemetry-bootstrap --action=install
COPY . .
ENV OTEL_SERVICE_NAME=python-service
ENV OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
ENV OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED=true
EXPOSE 8080
CMD ["opentelemetry-instrument", "python", "app.py"]
`,
		LanguageNodeJS: `
# Node.js with OpenTelemetry auto-instrumentation
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

FROM node:18-alpine
WORKDIR /app
COPY --from=builder /app/node_modules ./node_modules
COPY . .
RUN npm install @opentelemetry/api @opentelemetry/auto-instrumentations-node
ENV NODE_OPTIONS="--require @opentelemetry/auto-instrumentations-node/register"
ENV OTEL_SERVICE_NAME=nodejs-service
ENV OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
EXPOSE 3000
CMD ["node", "server.js"]
`,
	}

	for lang, dockerfile := range examples {
		fmt.Printf("=== %s Dockerfile Example ===\n%s\n", lang, dockerfile)
	}
}

// ExampleSecurityBestPractices demonstrates security best practices
func ExampleSecurityBestPractices() {
	secureDockerfile := `
# Security-hardened Dockerfile with APM
FROM golang:1.21-alpine AS builder

# Install security updates
RUN apk update && apk upgrade && apk add --no-cache ca-certificates git

# Create non-root user for building
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /build

# Copy dependency files first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY --chown=appuser:appuser . .

# Build as non-root user
USER appuser
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o app ./cmd/main.go

# Final stage - distroless for minimal attack surface
FROM gcr.io/distroless/static:nonroot

# Copy the binary
COPY --from=builder /build/app /app

# Copy APM configuration
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Use non-root user
USER nonroot:nonroot

# APM environment variables
ENV OTEL_SERVICE_NAME=secure-service
ENV OTEL_EXPORTER_OTLP_ENDPOINT=https://apm-collector:4317
ENV OTEL_EXPORTER_OTLP_HEADERS="Authorization=Bearer ${APM_TOKEN}"

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app", "health"]

# Metadata labels
LABEL maintainer="platform-team@example.com"
LABEL version="1.0.0"
LABEL description="Secure Go application with APM instrumentation"
LABEL security.scan="enabled"

EXPOSE 8080
ENTRYPOINT ["/app"]
`
	fmt.Println(secureDockerfile)
}
