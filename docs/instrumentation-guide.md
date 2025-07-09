# Instrumentation Guide for GoFiber Applications

This guide provides a quick start for instrumenting GoFiber applications with APM (Application Performance Monitoring) capabilities including metrics, tracing, and logging.

## Quick Start Guide

### 1. Installation

```bash
go get github.com/yourusername/apm/pkg/instrumentation
```

### 2. Basic Setup

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/yourusername/apm/pkg/instrumentation"
)

func main() {
    // Initialize instrumentation
    instr, err := instrumentation.New(
        instrumentation.WithServiceName("my-service"),
        instrumentation.WithMetricsEndpoint("localhost:4318"),
        instrumentation.WithTracingEndpoint("localhost:4317"),
    )
    if err != nil {
        panic(err)
    }
    defer instr.Shutdown()

    // Create Fiber app
    app := fiber.New()

    // Add instrumentation middleware
    app.Use(instr.FiberMiddleware())

    // Your routes
    app.Get("/", func(c *fiber.Ctx) error {
        return c.SendString("Hello, World!")
    })

    app.Listen(":3000")
}
```

## Integration Steps for Existing Applications

### Step 1: Add Dependencies

Update your `go.mod`:

```go
require (
    github.com/yourusername/apm v1.0.0
    go.opentelemetry.io/otel v1.19.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.19.0
    go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.42.0
)
```

### Step 2: Initialize Instrumentation

Add to your main function or initialization code:

```go
func initInstrumentation() (*instrumentation.Instrumentation, error) {
    return instrumentation.New(
        instrumentation.WithServiceName(os.Getenv("SERVICE_NAME")),
        instrumentation.WithEnvironment(os.Getenv("ENVIRONMENT")),
        instrumentation.WithVersion(os.Getenv("SERVICE_VERSION")),
        instrumentation.WithMetricsEndpoint(os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT")),
        instrumentation.WithTracingEndpoint(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")),
        instrumentation.WithSamplingRate(0.1), // 10% sampling
    )
}
```

### Step 3: Add Middleware to Routes

For existing Fiber apps:

```go
// Global middleware
app.Use(instr.FiberMiddleware())

// Or for specific routes
api := app.Group("/api", instr.FiberMiddleware())
```

### Step 4: Instrument Database Calls

```go
// Example with database instrumentation
func getUser(ctx context.Context, id string) (*User, error) {
    ctx, span := instr.Tracer().Start(ctx, "database.getUser")
    defer span.End()

    // Record custom metrics
    timer := instr.Metrics().NewTimer("db.query.duration", 
        attribute.String("operation", "select"),
        attribute.String("table", "users"),
    )
    defer timer.Record()

    // Your database logic here
    user, err := db.QueryUser(ctx, id)
    if err != nil {
        span.RecordError(err)
        instr.Metrics().RecordError("db.query.error", err)
        return nil, err
    }

    return user, nil
}
```

## Configuration Options

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OTEL_SERVICE_NAME` | Service name for telemetry | Required |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP endpoint for both metrics and traces | `localhost:4317` |
| `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` | Specific endpoint for metrics | Uses `OTEL_EXPORTER_OTLP_ENDPOINT` |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | Specific endpoint for traces | Uses `OTEL_EXPORTER_OTLP_ENDPOINT` |
| `OTEL_TRACES_SAMPLER` | Sampling strategy | `parentbased_traceidratio` |
| `OTEL_TRACES_SAMPLER_ARG` | Sampling rate (0.0 to 1.0) | `1.0` |
| `OTEL_METRIC_EXPORT_INTERVAL` | Metric export interval | `60s` |
| `OTEL_LOG_LEVEL` | Log level for instrumentation | `info` |

### Programmatic Configuration

```go
instr, err := instrumentation.New(
    // Basic configuration
    instrumentation.WithServiceName("my-service"),
    instrumentation.WithEnvironment("production"),
    instrumentation.WithVersion("1.0.0"),
    
    // Endpoints
    instrumentation.WithMetricsEndpoint("https://otel-collector:4318"),
    instrumentation.WithTracingEndpoint("https://otel-collector:4317"),
    
    // Advanced options
    instrumentation.WithSamplingRate(0.1),
    instrumentation.WithMetricInterval(30 * time.Second),
    instrumentation.WithTimeout(5 * time.Second),
    instrumentation.WithHeaders(map[string]string{
        "X-API-Key": "your-api-key",
    }),
    
    // TLS Configuration
    instrumentation.WithTLS(tlsConfig),
    
    // Custom attributes
    instrumentation.WithAttributes(map[string]string{
        "team": "backend",
        "region": "us-east-1",
    }),
)
```

## Troubleshooting Common Issues

### 1. No Metrics or Traces Appearing

**Check connectivity:**
```bash
# Test OTLP endpoint
curl -v http://localhost:4318/v1/metrics
```

**Verify configuration:**
```go
// Enable debug logging
instr, err := instrumentation.New(
    instrumentation.WithDebug(true),
    // ... other options
)
```

### 2. High Memory Usage

**Reduce cardinality:**
- Avoid high-cardinality labels (user IDs, request IDs)
- Use bounded values for labels
- Implement metric aggregation

**Adjust batching:**
```go
instrumentation.WithBatchSize(100),
instrumentation.WithQueueSize(1000),
```

### 3. Missing Traces

**Check sampling:**
```go
// For debugging, use 100% sampling
instrumentation.WithSamplingRate(1.0),

// For production, use lower rates
instrumentation.WithSamplingRate(0.01), // 1%
```

**Verify propagation:**
```go
// Ensure context is passed correctly
ctx = instr.InjectContext(c.UserContext())
```

### 4. Middleware Not Working

**Order matters:**
```go
// Instrumentation should be early in the chain
app.Use(instr.FiberMiddleware()) // First
app.Use(logger.New())            // Then other middleware
app.Use(recover.New())
```

### 5. Export Failures

**Check logs:**
```go
instr.SetLogLevel("debug")
```

**Common causes:**
- Firewall blocking OTLP ports (4317, 4318)
- Incorrect endpoint URLs
- Authentication failures
- Certificate issues for HTTPS endpoints

### Debug Mode

Enable comprehensive debugging:

```go
instr, err := instrumentation.New(
    instrumentation.WithDebug(true),
    instrumentation.WithLogLevel("debug"),
    instrumentation.WithStdoutExporter(), // Export to console for testing
)
```

### Health Check Endpoint

Add a health check to verify instrumentation:

```go
app.Get("/health/instrumentation", func(c *fiber.Ctx) error {
    status := instr.HealthCheck()
    if status.Healthy {
        return c.JSON(status)
    }
    return c.Status(503).JSON(status)
})
```