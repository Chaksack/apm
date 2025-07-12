# APM - Application Performance Monitoring for GoFiber

[![Go Reference](https://pkg.go.dev/badge/github.com/chaksack/apm.svg)](https://pkg.go.dev/github.com/chaksack/apm)
[![Go Report Card](https://goreportcard.com/badge/github.com/chaksack/apm)](https://goreportcard.com/report/github.com/chaksack/apm)
[![Semgrep](https://img.shields.io/badge/Semgrep-Enabled-green.svg)](https://semgrep.dev/)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![GoFiber](https://img.shields.io/badge/GoFiber-v2.52.0-00ACD7.svg)](https://github.com/gofiber/fiber)
[![Prometheus](https://img.shields.io/badge/Prometheus-v1.18.0-E6522C.svg)](https://github.com/prometheus/client_golang)
[![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-v1.37.0-425CC1.svg)](https://github.com/open-telemetry/opentelemetry-go)
[![Zap](https://img.shields.io/badge/Zap-v1.26.0-2088FF.svg)](https://github.com/uber-go/zap)
[![Viper](https://img.shields.io/badge/Viper-v1.20.1-5C7CFA.svg)](https://github.com/spf13/viper)
[![Testify](https://img.shields.io/badge/Testify-v1.10.0-9A76C9.svg)](https://github.com/stretchr/testify)
[![UUID](https://img.shields.io/badge/UUID-v1.6.0-FFC107.svg)](https://github.com/google/uuid)
[![FastHTTP](https://img.shields.io/badge/FastHTTP-v1.51.0-00B0D8.svg)](https://github.com/valyala/fasthttp)
[![Grafana](https://img.shields.io/badge/Grafana-Supported-F46800.svg)](https://grafana.com/)
[![Jaeger](https://img.shields.io/badge/Jaeger-Supported-66CFE3.svg)](https://www.jaegertracing.io/)
[![Loki](https://img.shields.io/badge/Loki-Supported-FF5A00.svg)](https://grafana.com/oss/loki/)
[![AlertManager](https://img.shields.io/badge/AlertManager-Supported-E6522C.svg)](https://prometheus.io/docs/alerting/latest/alertmanager/)
[![Istio](https://img.shields.io/badge/Istio-Supported-466BB0.svg)](https://istio.io/)

A comprehensive Application Performance Monitoring (APM) solution specifically designed for [GoFiber](https://gofiber.io/) applications. This package provides out-of-the-box observability including metrics collection, distributed tracing, structured logging, and health checks.

## Features

🚀 **Easy Integration** - Add comprehensive observability to your GoFiber app with just a few lines of code  
📊 **Prometheus Metrics** - Pre-configured HTTP metrics and custom metric builders  
🔍 **Distributed Tracing** - OpenTelemetry integration with Jaeger support  
📝 **Structured Logging** - Zap-based logging with request correlation  
💊 **Health Checks** - Kubernetes-ready health endpoints with dependency monitoring  
🧪 **Testing Utilities** - Mock collectors and test helpers for instrumented code  
⚙️ **Configuration** - Environment-based configuration with sensible defaults  
🚢 **Multi-Platform Deployment** - Deploy to Docker, Kubernetes, AWS, Azure, or GCP with interactive wizard  
🔐 **Security First** - Built-in authentication, TLS, and secrets management  
☁️ **Cloud-Native Ready** - Native integration with cloud provider services  
📚 **Comprehensive Documentation** - Extensive guides, examples, and API references

## Quick Start

### Installation

#### Install as a Go Package
```bash
go get github.com/chaksack/apm/pkg/instrumentation
```

#### Install the APM CLI Tool
```bash
go install github.com/chaksack/apm/cmd/apm@latest
```

Or build from source:
```bash
git clone https://github.com/chaksack/apm.git
cd apm
go build -o apm ./cmd/apm/
sudo mv apm /usr/local/bin/  # Optional: install globally
```

## 🎮 APM CLI Tool

The APM CLI is a comprehensive command-line interface that streamlines the setup, execution, and monitoring of GoFiber applications with integrated APM tools.

### CLI Commands

#### `apm init` - Interactive Configuration Setup

Initialize your APM configuration with an interactive wizard:

```bash
apm init
```

Features:
- 🧾 Step-by-step interactive wizard
- 🔧 Select and configure APM tools (Prometheus, Grafana, Jaeger, Loki)
- 💾 Saves configuration to `apm.yaml`
- 🔄 Update existing configurations
- 💬 Slack webhook integration for alerts
- 🔔 AlertManager auto-configuration

#### `apm run` - Run with Hot Reload

Run your application with automatic APM instrumentation and hot reload:

```bash
# Run using configuration from apm.yaml
apm run

# Run a specific command
apm run "go run main.go"

# Disable hot reload
apm run --no-reload
```

Features:
- 🔥 Hot reload on file changes
- 📊 Automatic APM agent injection
- 🌐 Environment variable configuration
- 📋 Real-time log output

#### `apm test` - Validate Configuration

Validate your APM configuration and check tool connectivity:

```bash
apm test
```

Performs:
- ✅ Configuration file validation
- 🔍 Required field checks
- 🏯 Tool connectivity tests
- 📊 Health status for each component

#### `apm dashboard` - Access Monitoring UIs

Interactive dashboard to access all monitoring interfaces:

```bash
apm dashboard
```

Features:
- 📊 List all configured APM tools
- 🔍 Real-time availability check
- 🌐 One-click browser access
- ⌨️ Keyboard navigation

#### `apm deploy` - Cloud Deployment with APM

Deploy your APM-instrumented application to cloud environments:

```bash
apm deploy
```

Features:
- 🚀 Interactive deployment wizard
- 🐳 Docker container deployment with APM agent injection
- ☸️ Kubernetes deployment with sidecar injection
- ☁️ Cloud provider support (AWS ECS/EKS, Azure ACI/AKS, Google Cloud Run/GKE)
- 🔐 Secure credential management using cloud CLIs
- 📊 Automatic APM instrumentation
- 🔄 Deployment status tracking
- ↩️ Rollback command generation

Deployment targets:
- **Docker**: Build, inject APM agents, and push to registries
- **Kubernetes**: Deploy with automatic sidecar injection
- **AWS**: ECS (Fargate) and EKS deployments
- **Azure**: Container Instances and AKS deployments
- **Google Cloud**: Cloud Run and GKE deployments


```bash
# Interactive deployment wizard
apm deploy

# Deploy to specific platform
apm deploy kubernetes
apm deploy docker
apm deploy cloud --provider aws

# Deploy with configuration file
apm deploy --config production.yaml

# Dry run to preview changes
apm deploy --dry-run
```

Features:
- 🚀 **Multi-Platform Support**: Docker, Kubernetes, AWS, Azure, GCP
- 🧙 **Interactive Wizard**: Step-by-step deployment configuration
- 📦 **Component Selection**: Choose which APM tools to deploy
- 🔧 **Resource Sizing**: Automatic resource allocation based on environment
- 🔐 **Security Configuration**: Built-in auth, TLS, and secrets management
- 🔄 **Rollback Support**: Automatic rollback on deployment failure
- 📊 **Progress Tracking**: Real-time deployment status updates

Deployment Examples:
```bash
# Deploy to local Docker environment
apm deploy docker

# Deploy to Kubernetes with custom namespace
apm deploy kubernetes --namespace monitoring

# Deploy to AWS EKS
apm deploy cloud --provider aws --region us-west-2

# Deploy to Azure AKS
apm deploy cloud --provider azure --region eastus

# Deploy to GCP GKE
apm deploy cloud --provider gcp --region us-central1
```

### Cloud Provider CLI Integration

The APM tool includes comprehensive cloud provider CLI integration to streamline multi-cloud deployments with automatic APM instrumentation.

#### Supported Cloud Providers

**🚀 AWS (Amazon Web Services)**
- **CLI Integration**: Automatic detection and validation of AWS CLI (v2.0.0+)
- **ECR Registry**: Push Docker images with APM agents to Amazon ECR
- **EKS Clusters**: Deploy to Elastic Kubernetes Service with sidecar injection
- **Authentication**: Support for IAM roles, access keys, and CLI profiles
- **Regions**: Multi-region support with automatic detection
- **Cross-Account Support**: Enterprise-grade cross-account role assumption
- **S3 Management**: APM configuration storage with lifecycle policies
- **CloudWatch Integration**: Metrics and monitoring integration

**🔷 Azure (Microsoft Azure)**
- **CLI Integration**: Automatic detection and validation of Azure CLI
- **ACR Registry**: Push Docker images with APM agents to Azure Container Registry
- **AKS Clusters**: Deploy to Azure Kubernetes Service with monitoring integration
- **Authentication**: Support for service principals, managed identity, and browser auth
- **Resource Groups**: Automatic resource group management and organization

**🟡 Google Cloud Platform (GCP)**
- **CLI Integration**: Automatic detection and validation of gcloud CLI
- **GCR/Artifact Registry**: Push Docker images to Google Container Registry
- **GKE Clusters**: Deploy to Google Kubernetes Engine with Cloud Monitoring
- **Authentication**: Support for service accounts, OAuth2, and Application Default Credentials
- **Projects**: Multi-project support with automatic project switching

#### Key Features

🔐 **Secure Credential Management**
- AES-256-GCM encryption for stored credentials
- Machine-specific encryption keys with PBKDF2 derivation
- Support for multiple authentication methods per provider
- Credential caching with TTL and automatic refresh

🔧 **CLI Detection & Validation**
- Automatic detection of installed cloud CLIs
- Version compatibility checking with minimum requirements
- Cross-platform support (Windows, macOS, Linux)
- Installation guidance for missing or outdated CLIs

🎯 **Multi-Cloud Operations**
- Unified interface for all cloud providers
- Concurrent operations for improved performance
- Cross-provider resource discovery and management
- Provider-agnostic deployment workflows

📊 **APM Integration**
- Automatic APM agent injection for all supported languages
- Cloud-specific monitoring service integration
- Registry authentication with temporary credentials
- Kubeconfig generation for all cluster types

#### Usage Examples

```bash
# Deploy with automatic provider detection
apm deploy

# Deploy to specific cloud provider
apm deploy --provider aws --region us-west-2
apm deploy --provider azure --resource-group mygroup
apm deploy --provider gcp --project myproject

# List available clusters across all providers
apm deploy --list-clusters

# Authenticate with multiple providers
apm deploy --setup-auth aws,azure,gcp
```

#### Prerequisites

Before using cloud provider integration, ensure you have the required CLIs installed:

```bash
# AWS CLI (minimum version 2.0.0)
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip && sudo ./aws/install

# Azure CLI (minimum version 2.30.0)
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

# Google Cloud CLI (minimum version 400.0.0)
curl https://sdk.cloud.google.com | bash
```

For detailed cloud provider setup instructions, see:
- [AWS Integration Guide](docs/cloud-provider-integration.md#aws)
- [Azure Integration Guide](docs/azure-integration.md)
- [GCP Integration Guide](docs/gcp-integration-guide.md)

### AWS Cross-Account Role Assumption

The APM tool provides enterprise-grade support for AWS multi-account environments with comprehensive cross-account role assumption capabilities:

#### Key Features

🔐 **Security-First Design**
- MFA (Multi-Factor Authentication) support for enhanced security
- External ID validation for partner access control
- Trust policy validation before role assumption
- Secure credential storage with AES-256-GCM encryption

🔄 **Advanced Role Management**
- Role chaining for complex multi-hop scenarios (up to 5 steps)
- Automatic session refresh before expiry
- Cross-region role switching support
- Concurrent session management with background workers

📋 **Configuration Management**
- YAML-based multi-account configuration
- Environment-specific settings (dev, staging, production)
- Role hierarchy and inheritance support
- Template-based configuration with validation

#### Usage Examples

```bash
# Deploy to production account with MFA
apm deploy --provider aws --account production --mfa

# Deploy using role chain across multiple accounts
apm deploy --provider aws --role-chain "dev-role,staging-role,prod-role"

# Use external ID for partner account access
apm deploy --provider aws --role arn:aws:iam::123456789012:role/PartnerRole --external-id "unique-id"
```

#### Configuration Example

```yaml
# apm.yaml with multi-account setup
aws:
  accounts:
    dev:
      account_id: "111111111111"
      region: "us-east-1"
      roles:
        - name: "dev-deploy-role"
          arn: "arn:aws:iam::111111111111:role/DevDeployRole"
    production:
      account_id: "222222222222"
      region: "us-west-2"
      roles:
        - name: "prod-deploy-role"
          arn: "arn:aws:iam::222222222222:role/ProdDeployRole"
          mfa_required: true
          external_id: "prod-external-id"

  role_chains:
    - name: "dev-to-prod"
      description: "Deploy from dev to production"
      steps:
        - role: "arn:aws:iam::111111111111:role/DevRole"
        - role: "arn:aws:iam::222222222222:role/ProdRole"
          external_id: "chain-external-id"
```

For detailed cross-account setup and best practices, see:
- [Cross-Account Role Assumption Guide](docs/cross-account-role-assumption.md)
- [AWS Role Chaining Documentation](docs/aws-role-chaining.md)
- [Multi-Account Examples](examples/cross-account-roles/)

### Configuration File (apm.yaml)

The CLI uses a YAML configuration file created by `apm init`:

```yaml
version: "1.0"
project:
  name: "my-app"
  environment: "development"

apm:
  prometheus:
    enabled: true
    port: 9090
    config:
      scrape_interval: "15s"

  grafana:
    enabled: true
    port: 3000
    config:
      datasources:
        - name: "Prometheus"
          type: "prometheus"
          url: "http://localhost:9090"

  jaeger:
    enabled: false
    agent_port: 6831
    ui_port: 16686

  loki:
    enabled: false
    port: 3100
    retention: "7d"

  alertmanager:
    enabled: false
    port: 9093
    config:
      receivers:
        - name: "default"
          slack_configs:
            - api_url: "YOUR_SLACK_WEBHOOK_URL"
              channel: "#alerts"
              title: "APM Alert"

notifications:
  slack:
    enabled: false
    webhook_url: "YOUR_SLACK_WEBHOOK_URL"
    channel: "#alerts"

application:
  entry_point: "./cmd/app/main.go"
  build_command: "go build"
  run_command: "./app"
  hot_reload:
    enabled: true
    paths: ["."]
    exclude: ["vendor", "node_modules", ".git"]
    extensions: [".go", ".mod"]

deployment:
  docker:
    dockerfile: "./Dockerfile"
    registry: "docker.io/myorg"
    build_args:
      VERSION: "1.0.0"
  kubernetes:
    manifests: "./k8s/"
    namespace: "production"
  cloud:
    provider: "aws"  # aws, azure, gcp
    region: "us-east-1"
```

### Basic Go Usage

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    apm "github.com/chaksack/apm/pkg/instrumentation"
)

func main() {
    // Initialize instrumentation
    instr, err := apm.New(apm.DefaultConfig())
    if err != nil {
        panic(err)
    }
    defer instr.Shutdown(context.Background())

    // Create Fiber app
    app := fiber.New()

    // Add APM middleware
    app.Use(instr.FiberMiddleware())

    // Add routes
    app.Get("/", func(c *fiber.Ctx) error {
        return c.SendString("Hello, World!")
    })

    // Start server
    app.Listen(":3000")
}
```

## 📦 Package Usage Documentation

### Installation

```bash
go get github.com/chaksack/apm/pkg/instrumentation
```

### Core Components

#### 1. Initialization and Configuration

```go
import "github.com/chaksack/apm/pkg/instrumentation"

// Option 1: Load from environment variables
config := instrumentation.LoadFromEnv()

// Option 2: Custom configuration
config := &instrumentation.Config{
    ServiceName: "my-service",
    Environment: "production",
    Version:     "1.0.0",
    Metrics: instrumentation.MetricsConfig{
        Enabled:   true,
        Namespace: "myapp",
        Subsystem: "api",
        Path:      "/metrics",
    },
    Logging: instrumentation.LoggingConfig{
        Level:       "info",
        Encoding:    "json",
        Development: false,
    },
}

// Initialize instrumentation
instr, err := instrumentation.New(config)
if err != nil {
    log.Fatal(err)
}
defer instr.Shutdown(context.Background())
```

#### 2. Middleware Integration

```go
// Create Fiber app
app := fiber.New()

// Add request ID middleware (recommended first)
app.Use(requestid.New())

// Add instrumentation middleware
app.Use(instr.FiberMiddleware())

// Add OpenTelemetry middleware for tracing
app.Use(instrumentation.FiberOtelMiddleware("my-service"))

// Add logging middleware
app.Use(instrumentation.LoggerMiddleware(instr.Logger))

// Prometheus metrics endpoint
app.Get("/metrics", func(c *fiber.Ctx) error {
    fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())(c.Context())
    return nil
})
```

#### 3. Metrics Collection

```go
// Access metrics collector
metrics := instr.Metrics

// Create custom metrics
counter := metrics.NewCounter(
    "operations_total",
    "Total operations performed",
    []string{"operation", "status"},
)

gauge := metrics.NewGauge(
    "active_connections",
    "Number of active connections",
    []string{"protocol"},
)

histogram := metrics.NewHistogram(
    "request_duration_seconds",
    "Request duration in seconds",
    []string{"endpoint"},
    []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
)

// Use metrics in handlers
app.Post("/orders", func(c *fiber.Ctx) error {
    start := time.Now()

    // Track metrics
    counter.WithLabelValues("create_order", "initiated").Inc()
    gauge.WithLabelValues("http").Inc()
    defer gauge.WithLabelValues("http").Dec()

    // Process order...

    histogram.WithLabelValues("/orders").Observe(time.Since(start).Seconds())
    counter.WithLabelValues("create_order", "completed").Inc()

    return c.JSON(order)
})
```

#### 4. Distributed Tracing

```go
// Initialize tracer
tracerConfig := instrumentation.TracerConfig{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    Environment:    "production",
    ExporterType:   "otlp",
    Endpoint:       "localhost:4317",
    SampleRate:     0.1, // Sample 10% of traces
}

tracerProvider, cleanup, err := instrumentation.InitTracer(ctx, tracerConfig)
defer cleanup()

// Get tracer
tracer := instrumentation.GetTracer("my-service")

// Use in handlers
app.Get("/users/:id", func(c *fiber.Ctx) error {
    // Get context with correlation ID
    ctx := instrumentation.FiberContextWithCorrelation(c)

    // Create span
    ctx, span := instrumentation.StartSpanWithCorrelation(ctx, tracer, "get-user")
    defer span.End()

    // Add span attributes
    span.SetAttributes(
        attribute.String("user.id", c.Params("id")),
        attribute.String("correlation.id", instrumentation.GetCorrelationID(ctx)),
    )

    // Propagate context for outgoing HTTP requests
    headers := make(map[string]string)
    instrumentation.PropagateContext(c, headers)

    return c.JSON(user)
})
```

#### 5. Structured Logging

```go
// Get request-scoped logger in handlers
app.Use(func(c *fiber.Ctx) error {
    logger := instrumentation.GetLogger(c)

    logger.Info("Request received",
        zap.String("method", c.Method()),
        zap.String("path", c.Path()),
        zap.String("request_id", c.Locals("requestid").(string)),
    )

    return c.Next()
})

// Error logging
app.Use(func(c *fiber.Ctx) error {
    err := c.Next()

    if err != nil {
        logger := instrumentation.GetLogger(c)
        logger.Error("Request failed",
            zap.Error(err),
            zap.Int("status", c.Response().StatusCode()),
        )
    }

    return err
})
```

#### 6. Health Checks

```go
// Basic health endpoints
app.Get("/health/live", func(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{
        "status": "alive",
        "timestamp": time.Now().Unix(),
    })
})

app.Get("/health/ready", func(c *fiber.Ctx) error {
    // Check dependencies
    checks := fiber.Map{
        "database": checkDatabase(),
        "cache": checkRedis(),
        "external_api": checkExternalAPI(),
    }

    allHealthy := true
    for _, status := range checks {
        if status != "healthy" {
            allHealthy = false
            break
        }
    }

    status := fiber.StatusOK
    if !allHealthy {
        status = fiber.StatusServiceUnavailable
    }

    return c.Status(status).JSON(fiber.Map{
        "status": allHealthy,
        "checks": checks,
    })
})
```

## 🛠️ Configuration Options

### Environment Variables

The package supports comprehensive configuration through environment variables:

#### Service Configuration
```bash
# Core service settings
SERVICE_NAME=my-service          # Service identifier
ENVIRONMENT=production          # Environment (dev/staging/production)
VERSION=1.0.0                   # Service version

# APM service configuration (when using internal/config)
APM_SERVICE_NAME=apm            # APM service name
APM_SERVER_PORT=:8080           # Server port
APM_SERVER_TIMEOUT=120          # Server timeout in seconds
APM_PREFORK_MODE=false          # Enable prefork mode for performance
```

#### Metrics Configuration
```bash
METRICS_ENABLED=true            # Enable metrics collection
METRICS_NAMESPACE=myapp         # Prometheus namespace
METRICS_SUBSYSTEM=api           # Prometheus subsystem
METRICS_PATH=/metrics           # Metrics endpoint path
```

#### Logging Configuration
```bash
LOG_LEVEL=info                  # Log level (debug/info/warn/error)
LOG_ENCODING=json               # Log encoding (json/console)
LOG_DEVELOPMENT=false           # Development mode
LOG_OUTPUT_PATHS=stdout         # Output paths (comma-separated)
LOG_ERROR_OUTPUT_PATHS=stderr   # Error output paths
LOG_ENABLE_CALLER=false         # Enable caller information
LOG_ENABLE_STACKTRACE=false     # Enable stack traces for errors
```

#### Tracing Configuration
```bash
# OpenTelemetry configuration
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
OTEL_EXPORTER_OTLP_INSECURE=true
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1

# Jaeger configuration
JAEGER_AGENT_HOST=localhost
JAEGER_AGENT_PORT=6831
JAEGER_COLLECTOR_ENDPOINT=http://localhost:14268/api/traces
JAEGER_SAMPLER_TYPE=probabilistic
JAEGER_SAMPLER_PARAM=0.1
```

#### APM Component Endpoints
```bash
APM_PROMETHEUS_ENDPOINT=http://localhost:9090
APM_GRAFANA_ENDPOINT=http://localhost:3000
APM_LOKI_ENDPOINT=http://localhost:3100
APM_JAEGER_ENDPOINT=http://localhost:16686
APM_ALERTMANAGER_ENDPOINT=http://localhost:9093
```

### Configuration File (config.yaml)

```yaml
service:
  name: "my-service"
  environment: "production"
  version: "1.0.0"
  server:
    port: ":8080"
    timeout: 120
    prefork: false

metrics:
  enabled: true
  namespace: "myapp"
  subsystem: "api"
  path: "/metrics"

logging:
  level: "info"
  encoding: "json"
  development: false
  outputPaths:
    - "stdout"
  errorOutputPaths:
    - "stderr"
  enableCaller: false
  enableStacktrace: false

tracing:
  enabled: true
  serviceName: "my-service"
  exporterType: "otlp"
  endpoint: "localhost:4317"
  sampleRate: 0.1

# Notification settings
notifications:
  smtp:
    enabled: false
    host: "smtp.gmail.com"
    port: 587
    from: "alerts@example.com"
  slack:
    enabled: false
    webhookURL: "https://hooks.slack.com/services/..."
    channel: "#alerts"
```

## 🔧 Integration Examples

### Basic Integration Pattern

```go
package main

import (
    "context"
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/requestid"
    "github.com/chaksack/apm/pkg/instrumentation"
)

func main() {
    // Load configuration
    cfg := instrumentation.LoadFromEnv()

    // Initialize instrumentation
    inst, err := instrumentation.New(cfg)
    if err != nil {
        panic(err)
    }
    defer inst.Shutdown(context.Background())

    // Create Fiber app with middleware stack
    app := fiber.New()
    app.Use(requestid.New())
    app.Use(inst.FiberMiddleware())
    app.Use(instrumentation.LoggerMiddleware(inst.Logger))

    // Your routes here
    setupRoutes(app, inst)

    app.Listen(":8080")
}
```

### Advanced Integration Examples

#### 1. **E-commerce Application**
```go
// Custom metrics for business operations
var (
    orderCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "orders_total",
            Help: "Total number of orders",
        },
        []string{"status", "payment_method"},
    )

    cartSize = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "cart_size",
            Help: "Number of items in cart",
            Buckets: []float64{1, 5, 10, 20, 50, 100},
        },
        []string{"user_type"},
    )
)

app.Post("/checkout", func(c *fiber.Ctx) error {
    ctx := instrumentation.FiberContextWithCorrelation(c)
    tracer := instrumentation.GetTracer("checkout-service")

    // Start main span
    ctx, span := tracer.Start(ctx, "process-checkout")
    defer span.End()

    // Process payment
    ctx, paymentSpan := tracer.Start(ctx, "process-payment")
    // ... payment logic
    paymentSpan.End()

    // Update inventory
    ctx, inventorySpan := tracer.Start(ctx, "update-inventory")
    // ... inventory logic
    inventorySpan.End()

    // Record metrics
    orderCounter.WithLabelValues("completed", "credit_card").Inc()
    cartSize.WithLabelValues("registered").Observe(float64(len(cart.Items)))

    logger := instrumentation.GetLogger(c)
    logger.Info("Order completed",
        zap.String("order_id", orderID),
        zap.Float64("total", orderTotal),
    )

    return c.JSON(order)
})
```

#### 2. **Microservices with Circuit Breaker**
```go
import "github.com/sony/gobreaker"

type PaymentService struct {
    breaker *gobreaker.CircuitBreaker
    tracer  trace.Tracer
}

func NewPaymentService() *PaymentService {
    settings := gobreaker.Settings{
        Name: "payment-api",
        MaxRequests: 3,
        Interval: 60 * time.Second,
        Timeout: 30 * time.Second,
        OnStateChange: func(name string, from, to gobreaker.State) {
            logger.Warn("Circuit breaker state change",
                zap.String("name", name),
                zap.String("from", from.String()),
                zap.String("to", to.String()),
            )
        },
    }

    return &PaymentService{
        breaker: gobreaker.NewCircuitBreaker(settings),
        tracer:  instrumentation.GetTracer("payment-service"),
    }
}

func (s *PaymentService) ProcessPayment(ctx context.Context, amount float64) error {
    _, span := s.tracer.Start(ctx, "payment.process")
    defer span.End()

    result, err := s.breaker.Execute(func() (interface{}, error) {
        // Make external API call
        return s.callPaymentAPI(ctx, amount)
    })

    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, "Payment processing failed")
        return err
    }

    span.SetAttributes(
        attribute.Float64("payment.amount", amount),
        attribute.String("payment.status", "success"),
    )

    return nil
}
```

#### 3. **Database Operations with Tracing**
```go
type UserRepository struct {
    db     *sql.DB
    tracer trace.Tracer
}

func (r *UserRepository) GetUser(ctx context.Context, id string) (*User, error) {
    ctx, span := r.tracer.Start(ctx, "repository.get_user")
    defer span.End()

    span.SetAttributes(
        attribute.String("db.operation", "SELECT"),
        attribute.String("db.table", "users"),
        attribute.String("user.id", id),
    )

    // Measure query duration
    start := time.Now()

    var user User
    err := r.db.QueryRowContext(ctx,
        "SELECT id, name, email FROM users WHERE id = $1", id,
    ).Scan(&user.ID, &user.Name, &user.Email)

    queryDuration.WithLabelValues("select_user").Observe(time.Since(start).Seconds())

    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, "Failed to fetch user")
        return nil, err
    }

    return &user, nil
}
```

#### 4. **Caching with Monitoring**
```go
type CacheService struct {
    client      *redis.Client
    tracer      trace.Tracer
    hitCounter  prometheus.Counter
    missCounter prometheus.Counter
}

func (c *CacheService) Get(ctx context.Context, key string) (string, error) {
    ctx, span := c.tracer.Start(ctx, "cache.get")
    defer span.End()

    span.SetAttributes(attribute.String("cache.key", key))

    value, err := c.client.Get(ctx, key).Result()

    if err == redis.Nil {
        c.missCounter.Inc()
        span.SetAttributes(attribute.Bool("cache.hit", false))
        return "", nil
    } else if err != nil {
        span.RecordError(err)
        return "", err
    }

    c.hitCounter.Inc()
    span.SetAttributes(attribute.Bool("cache.hit", true))

    return value, nil
}
```

### Complete Example Applications

1. **[examples/gofiber-app](./examples/gofiber-app/)** - Full-featured application with:
   - Complete APM integration
   - Custom business metrics
   - Circuit breaker pattern
   - Database and cache simulation
   - Health checks
   - Docker deployment

2. **[example/main.go](./example/main.go)** - Simple integration example

3. **[sample-app/](./sample-app/)** - Production-like application with:
   - Multi-tier architecture
   - External service integration
   - Advanced error handling
   - Performance optimization

### Testing Your Integration

```go
func TestMetricsCollection(t *testing.T) {
    // Create test configuration
    cfg := &instrumentation.Config{
        ServiceName: "test-service",
        Metrics: instrumentation.MetricsConfig{
            Enabled: true,
        },
    }

    // Initialize instrumentation
    inst, err := instrumentation.New(cfg)
    require.NoError(t, err)
    defer inst.Shutdown(context.Background())

    // Create test app
    app := fiber.New()
    app.Use(inst.FiberMiddleware())

    // Test endpoint
    app.Get("/test", func(c *fiber.Ctx) error {
        return c.SendString("OK")
    })

    // Make request
    req := httptest.NewRequest("GET", "/test", nil)
    resp, err := app.Test(req)

    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)

    // Verify metrics were collected
    // ... metric assertions
}
```

## 👥 Authors

- **Andrew Chakdahah** - [chakdahah@gmail.com](mailto:chakdahah@gmail.com)
- **Yaw Boateng Kessie** - [ybkess@gmail.com](mailto:ybkess@gmail.com)

## 📄 License

This project is licensed under the Apache License, Version 2.0 - see the [LICENSE](LICENSE) file for details.

```
Copyright (c) 2024 APM Solution Contributors
Authors: Andrew Chakdahah (chakdahah@gmail.com) and Yaw Boateng Kessie (ybkess@gmail.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

## 🚀 Deployment Options

### 1. Docker Deployment

#### Single Container
```bash
# Build the image
docker build -t my-app:latest .

# Run the container
docker run -d -p 8080:8080 \
  -e SERVICE_NAME=my-service \
  -e ENVIRONMENT=production \
  my-app:latest
```

#### Docker Compose (Full Stack)
```bash
# Start the complete APM stack
docker-compose up -d

# This includes:
# - Your application
# - Prometheus (metrics)
# - Grafana (visualization)
# - Loki (logs)
# - Jaeger (tracing)
# - AlertManager (alerts)
# - Node Exporter & cAdvisor (system metrics)
```

### 2. Kubernetes Deployment

#### Using kubectl
```bash
# Deploy base components
kubectl apply -f deployments/kubernetes/base/

# Deploy monitoring stack
kubectl apply -f deployments/kubernetes/prometheus/
kubectl apply -f deployments/kubernetes/grafana/
kubectl apply -f deployments/kubernetes/loki/
kubectl apply -f deployments/kubernetes/jaeger/
kubectl apply -f deployments/kubernetes/alertmanager/

# Deploy with Istio service mesh (optional)
kubectl apply -f deployments/kubernetes/istio/
```

#### Using Helm
```bash
# Add the Helm repository (if published)
helm repo add apm https://your-repo/apm

# Install the APM stack
helm install my-apm deployments/helm/apm-stack/ \
  --set app.name=my-service \
  --set app.environment=production \
  --set prometheus.enabled=true \
  --set grafana.enabled=true \
  --set loki.enabled=true \
  --set jaeger.enabled=true

# Or with custom values
helm install my-apm deployments/helm/apm-stack/ -f my-values.yaml
```

#### Using the deployment script
```bash
# Deploy to different environments
./scripts/deploy.sh dev
./scripts/deploy.sh staging
./scripts/deploy.sh production

# With rollback capability
./scripts/deploy.sh production --rollback
```

### 3. ArgoCD GitOps Deployment

```bash
# Install ArgoCD
kubectl apply -f deployments/argocd/install.yaml

# Create APM project
kubectl apply -f deployments/argocd/projects/apm-project.yaml

# Deploy APM application
kubectl apply -f deployments/argocd/applications/apm-app.yaml
```

ArgoCD will automatically:
- Monitor the Git repository for changes
- Sync deployments across environments
- Handle rollbacks and updates
- Send notifications via Slack/Email

### 4. Cloud Platform Deployments

#### AWS EKS
```bash
# Create EKS cluster
eksctl create cluster --name my-apm-cluster --region us-west-2

# Deploy APM stack
kubectl apply -f deployments/kubernetes/

# Configure AWS-specific integrations
kubectl apply -f deployments/cloud/aws/
```

#### Google GKE
```bash
# Create GKE cluster
gcloud container clusters create my-apm-cluster \
  --zone us-central1-a \
  --num-nodes 3

# Deploy APM stack
kubectl apply -f deployments/kubernetes/

# Configure GCP-specific integrations
kubectl apply -f deployments/cloud/gcp/
```

#### Azure AKS
```bash
# Create AKS cluster
az aks create \
  --resource-group myResourceGroup \
  --name my-apm-cluster \
  --node-count 3

# Get credentials
az aks get-credentials \
  --resource-group myResourceGroup \
  --name my-apm-cluster

# Deploy APM stack
kubectl apply -f deployments/kubernetes/
```

### 5. Local Development

```bash
# Using Make
make build          # Build the application
make run           # Run locally
make dev           # Run with hot reload
make up            # Start Docker Compose stack
make test          # Run tests
make test-e2e      # Run end-to-end tests

# Direct execution
go run cmd/apm/main.go

# With air for hot reload
air -c .air.toml
```

### 6. CI/CD Integration

#### GitHub Actions
```yaml
# The repository includes complete CI/CD pipelines
# .github/workflows/ci.yml provides:
# - Multi-platform builds (Linux, macOS, Windows)
# - Docker image building and pushing
# - Security scanning (Trivy, gosec, Semgrep)
# - Automated testing
# - Helm chart packaging
# - SonarQube analysis
```

#### Example deployment in CI/CD
```bash
# Build and push Docker image
docker build -t ghcr.io/your-org/apm:$VERSION .
docker push ghcr.io/your-org/apm:$VERSION

# Deploy to Kubernetes
kubectl set image deployment/apm-app \
  apm=ghcr.io/your-org/apm:$VERSION \
  -n production
```

## 📊 Monitoring Features

### Complete Observability Stack

The APM package provides comprehensive monitoring capabilities through integration with industry-standard tools:

#### 1. **Prometheus Metrics**
- **Automatic HTTP Metrics**: Request count, duration, size, and status codes
- **Custom Business Metrics**: Counters, gauges, histograms, and summaries
- **System Metrics**: CPU, memory, disk, and network usage via Node Exporter
- **Container Metrics**: Resource usage via cAdvisor
- **Pre-configured Alerts**: Response time, error rate, and availability alerts

```go
// Automatic HTTP metrics are collected for:
// - http_request_duration_seconds
// - http_request_size_bytes
// - http_response_size_bytes
// - http_requests_total
```

#### 2. **Grafana Dashboards**
- **Pre-built Dashboards**: Application overview, performance, and resource usage
- **Custom Dashboard Support**: Create your own visualizations
- **Alert Integration**: Visual alerts with Prometheus and AlertManager
- **Multi-datasource**: Combine metrics, logs, and traces in one view

#### 3. **Distributed Tracing with Jaeger**
- **Automatic Trace Collection**: Every HTTP request is traced
- **Service Dependency Mapping**: Visualize service interactions
- **Performance Analysis**: Identify bottlenecks and slow operations
- **Error Tracking**: Trace failed requests across services
- **Sampling Control**: Configurable sampling rates for production

#### 4. **Log Aggregation with Loki**
- **Structured Logging**: JSON format with automatic field extraction
- **Correlation**: Link logs to traces and metrics
- **LogQL Queries**: Powerful query language for log analysis
- **Promtail Integration**: Automatic log shipping and labeling
- **Retention Policies**: Configurable log retention

#### 5. **AlertManager Integration**
- **Alert Routing**: Route alerts to different teams/channels
- **Notification Channels**: Email, Slack, PagerDuty, webhooks
- **Alert Grouping**: Reduce noise by grouping related alerts
- **Silencing**: Maintenance windows and alert suppression
- **Alert Templates**: Customizable notification formats

#### 6. **Service Mesh Observability (Istio)**
- **Traffic Management**: Load balancing and routing metrics
- **Security Metrics**: mTLS and authorization statistics
- **Telemetry v2**: Enhanced metrics collection
- **Distributed Tracing**: Automatic trace propagation

### Monitoring Configuration

```yaml
# Prometheus scrape configuration
scrape_configs:
  - job_name: 'my-app'
    static_configs:
      - targets: ['my-app:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s

# Loki configuration for logs
clients:
  - url: http://loki:3100/loki/api/v1/push
    external_labels:
      job: my-app
      environment: production

# Jaeger configuration
sampling:
  default_strategy:
    type: probabilistic
    param: 0.1  # Sample 10% of traces
```

### Key Metrics Collected

1. **Application Metrics**
   - Request rate (req/sec)
   - Error rate (4xx, 5xx responses)
   - Response time (p50, p95, p99)
   - Active connections
   - Request/response sizes

2. **Business Metrics**
   - Custom operation counts
   - Processing durations
   - Queue lengths
   - Cache hit rates

3. **System Metrics**
   - CPU usage
   - Memory consumption
   - Disk I/O
   - Network traffic
   - Container resources

### Accessing Monitoring Tools

The APM service provides convenient routes to access all monitoring tools:

```
# List all available monitoring tools
GET /tools/

# Access specific monitoring tools
GET /tools/prometheus    # Prometheus UI
GET /tools/grafana       # Grafana dashboards
GET /tools/jaeger        # Jaeger UI for distributed tracing
GET /tools/loki          # Loki logs
GET /tools/alertmanager  # AlertManager UI
GET /tools/cadvisor      # cAdvisor for container metrics
GET /tools/node-exporter # Node Exporter metrics
```

These routes provide redirects to the appropriate tool interfaces, making it easy to navigate between different components of your observability stack.

## Dependencies

This package uses the following open-source libraries:

- [GoFiber](https://github.com/gofiber/fiber) - Fast HTTP web framework
- [GoFiber Adaptor](https://github.com/gofiber/adaptor) - HTTP handler adapters for Fiber
- [Prometheus Client](https://github.com/prometheus/client_golang) - Prometheus instrumentation library
- [OpenTelemetry](https://github.com/open-telemetry/opentelemetry-go) - Distributed tracing framework
- [Zap](https://github.com/uber-go/zap) - Structured logging library
- [Viper](https://github.com/spf13/viper) - Configuration management
- [Testify](https://github.com/stretchr/testify) - Testing toolkit
- [UUID](https://github.com/google/uuid) - UUID generation
- [FastHTTP](https://github.com/valyala/fasthttp) - Fast HTTP package for Go

## 🚀 Getting Started with APM CLI

### 1. Initialize Your Project

```bash
# Create a new APM configuration
apm init
```

Follow the interactive wizard to:
- Configure your project settings
- Select APM tools to integrate
- Set up monitoring parameters

### 2. Validate Your Setup

```bash
# Check configuration and tool connectivity
apm test
```

This ensures:
- Configuration file is valid
- All APM tools are accessible
- Application entry point exists

### 3. Run Your Application

```bash
# Start with hot reload and APM instrumentation
apm run
```

Your application will:
- Start with APM agent injection
- Reload automatically on code changes
- Send telemetry to configured tools

### 4. Monitor Your Application

```bash
# Access monitoring dashboards
apm dashboard
```

Quickly access:
- Prometheus metrics
- Grafana dashboards
- Jaeger traces
- Loki logs

## Testing

The package includes comprehensive testing utilities:

```go
func TestMyHandler(t *testing.T) {
    // Use test collector for metrics
    collector := apm.NewTestCollector()

    // Use test tracer for tracing
    tracer := apm.NewTestTracer()

    // Test your handlers
    app := apm.NewTestApp()
    // ... test assertions
}
```

### CLI Testing

The APM CLI commands can be tested using the provided test script:

```bash
# Navigate to the test directory
cd test

# Run the CLI test script
./cli_test.sh
```

The test script verifies that all CLI commands (init, run, test, dashboard) execute without errors. For more details, see [CLI Testing Documentation](test/CLI_TESTING.md).

## Documentation

- [API Reference](https://pkg.go.dev/github.com/chaksack/apm)
- [User Guides](./docs/user-guides/)
- [Deployment Guide](./docs/deployment/)
- [Troubleshooting](./docs/troubleshooting/)

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

## Authors

- **Andrew Chakdahah** - [chakdahah@gmail.com](mailto:chakdahah@gmail.com)
- **Yaw Boateng Kessie** - [ybkess@gmail.com](mailto:ybkess@gmail.com)

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

## Support

- [GitHub Issues](https://github.com/chaksack/apm/issues)
- [Documentation](./docs/)
- [Examples](./examples/)
