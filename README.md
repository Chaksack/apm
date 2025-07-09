# APM - Application Performance Monitoring for GoFiber

[![Go Reference](https://pkg.go.dev/badge/github.com/chaksack/apm.svg)](https://pkg.go.dev/github.com/chaksack/apm)
[![Go Report Card](https://goreportcard.com/badge/github.com/chaksack/apm)](https://goreportcard.com/report/github.com/chaksack/apm)
[![Semgrep](https://img.shields.io/badge/Semgrep-Enabled-green.svg)](https://semgrep.dev/)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

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

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

## Support

- [GitHub Issues](https://github.com/chaksack/apm/issues)
- [Documentation](./docs/)
- [Examples](./examples/)