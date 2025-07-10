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

## Quick Start

### Installation

```bash
go get github.com/chaksack/apm
```

### Basic Usage

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

### Advanced Configuration

```go
config := &apm.Config{
    ServiceName: "my-service",
    Environment: "production",
    Version:     "1.0.0",
    Metrics: apm.MetricsConfig{
        Enabled:   true,
        Namespace: "myapp",
        Subsystem: "api",
    },
    Logging: apm.LoggingConfig{
        Level:       "info",
        Encoding:    "json",
        Development: false,
    },
}

instr, err := apm.New(config)
```

## Components

### Metrics Collection

Automatic HTTP metrics collection with Prometheus:

```go
// HTTP metrics are automatically collected
// Custom metrics can be added:
counter := instr.Metrics.NewCounter("operations_total", "Total operations", []string{"type"})
counter.WithLabelValues("user_signup").Inc()
```

### Distributed Tracing

OpenTelemetry integration with automatic trace propagation:

```go
app.Get("/users/:id", func(c *fiber.Ctx) error {
    // Traces are automatically created for each request
    // Add custom spans:
    ctx, span := tracer.Start(c.UserContext(), "get-user")
    defer span.End()

    // Your business logic here
    return c.JSON(user)
})
```

### Structured Logging

Request-scoped logging with trace correlation:

```go
app.Use(func(c *fiber.Ctx) error {
    logger := apm.GetLogger(c) // Request-scoped logger with trace ID
    logger.Info("Processing request")
    return c.Next()
})
```

### Health Checks

Kubernetes-ready health endpoints:

```go
// Add health check endpoints
app.Get("/health/live", apm.LivenessHandler())
app.Get("/health/ready", apm.ReadinessHandler(healthChecker))

// Configure health checks
healthChecker := apm.NewHealthChecker()
healthChecker.AddCheck("database", apm.DatabaseHealthCheck("postgres://..."))
healthChecker.AddCheck("redis", apm.HTTPHealthCheck("redis", "http://redis:6379/ping"))
```

## Configuration

Configuration can be provided via environment variables:

```bash
# Service configuration
SERVICE_NAME=my-service
ENVIRONMENT=production
VERSION=1.0.0

# Metrics configuration
METRICS_ENABLED=true
METRICS_NAMESPACE=myapp
METRICS_PATH=/metrics

# Logging configuration
LOG_LEVEL=info
LOG_ENCODING=json
LOG_DEVELOPMENT=false
```

## Complete Example

See the [examples/gofiber-app](./examples/gofiber-app/) directory for a complete example application demonstrating:

- Full APM integration
- Custom metrics and tracing
- Health checks
- Error handling
- Docker deployment

## Kubernetes Deployment

The package includes production-ready Kubernetes manifests and Helm charts:

```bash
# Deploy with kubectl
kubectl apply -f deployments/kubernetes/

# Deploy with Helm
helm install my-app deployments/helm/apm-stack/
```

## Monitoring Stack Integration

Works seamlessly with the complete observability stack:

- **Prometheus** - Metrics collection and alerting
- **Grafana** - Visualization and dashboards  
- **Loki** - Log aggregation and analysis
- **Jaeger** - Distributed tracing
- **AlertManager** - Alert routing and notifications
- **Istio** - Service mesh observability

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
