# OpenTelemetry Instrumentation Package

This package provides production-ready OpenTelemetry SDK integration for Go applications, with special support for GoFiber web framework.

## Features

- **Multiple Exporters**: Support for OTLP (gRPC/HTTP), Jaeger, and stdout exporters
- **Multi-Exporter Setup**: Export traces to multiple destinations simultaneously
- **GoFiber Integration**: Middleware for automatic trace context propagation
- **Correlation ID Management**: Built-in correlation ID generation and propagation
- **Baggage Support**: Propagate contextual data across service boundaries
- **Configurable Sampling**: Control trace sampling rates
- **Batch Processing**: Efficient trace export with configurable batch processing

## Usage

### Basic Setup

```go
import (
    "context"
    "github.com/chaksack/apm/pkg/instrumentation"
)

// Initialize tracer
ctx := context.Background()
config := instrumentation.TracerConfig{
    ServiceName:    "apm",
    ServiceVersion: "1.0.0",
    Environment:    "production",
    ExporterType:   "otlp",
    Endpoint:       "localhost:4317",
    SampleRate:     0.1, // Sample 10% of traces
}

tracerProvider, cleanup, err := instrumentation.InitTracer(ctx, config)
if err != nil {
    log.Fatal(err)
}
defer cleanup()
```

### GoFiber Middleware

```go
import (
    "github.com/gofiber/fiber/v2"
    "github.com/chaksack/apm/pkg/instrumentation"
)

app := fiber.New()

// Add OpenTelemetry middleware
app.Use(instrumentation.FiberOtelMiddleware("my-service"))
```

### Multi-Exporter Setup

```go
ctx := context.Background()
exporterConfig := instrumentation.ExporterConfig{
    Type: "multi",
    Exporters: []instrumentation.ExporterConfig{
        {
            Type:     "otlp-grpc",
            Endpoint: "localhost:4317",
            Insecure: true,
        },
        {
            Type:     "jaeger",
            Endpoint: "http://localhost:14268/api/traces",
        },
        {
            Type: "stdout",
        },
    },
}

exporter, err := instrumentation.CreateExporter(ctx, exporterConfig)
if err != nil {
    log.Fatal(err)
}

processor := instrumentation.CreateBatchProcessor(exporter, exporterConfig)
```

### Correlation ID Usage

```go
// In a Fiber handler
app.Get("/api/users", func(c *fiber.Ctx) error {
    // Get context with correlation ID
    ctx := instrumentation.FiberContextWithCorrelation(c)
    
    // Correlation ID is automatically added to response headers
    // and propagated in trace context
    
    // Get correlation ID if needed
    correlationID := instrumentation.GetCorrelationID(ctx)
    
    return c.JSON(fiber.Map{
        "correlation_id": correlationID,
    })
})
```

### Manual Span Creation

```go
// Get tracer
tracer := instrumentation.GetTracer("my-service")

// Start span with correlation ID
ctx, span := instrumentation.StartSpanWithCorrelation(ctx, tracer, "operation-name")
defer span.End()

// Add attributes to span
span.SetAttributes(
    attribute.String("user.id", "123"),
    attribute.Int("items.count", 5),
)
```

### Baggage Propagation

```go
// Set baggage value
ctx = instrumentation.SetBaggageValue(ctx, "tenant-id", "acme-corp")

// Get baggage value
tenantID := instrumentation.GetBaggageValue(ctx, "tenant-id")
```

## Configuration Options

### TracerConfig

- `ServiceName`: Name of your service
- `ServiceVersion`: Version of your service
- `Environment`: Deployment environment (e.g., "production", "staging")
- `ExporterType`: Type of exporter ("otlp", "jaeger", "stdout")
- `Endpoint`: Endpoint for the exporter
- `SampleRate`: Sampling rate (0.0 to 1.0)

### ExporterConfig

- `Type`: Exporter type ("otlp-grpc", "otlp-http", "jaeger", "stdout", "multi")
- `Endpoint`: Endpoint URL
- `Headers`: Additional headers for OTLP exporters
- `Insecure`: Use insecure connection
- `BatchTimeout`: Batch timeout in milliseconds
- `MaxExportBatch`: Maximum batch size
- `MaxQueueSize`: Maximum queue size

## Best Practices

1. **Initialize Once**: Initialize the tracer once at application startup
2. **Use Correlation IDs**: Always use correlation IDs for request tracking
3. **Set Appropriate Sampling**: Use lower sampling rates in production
4. **Handle Cleanup**: Always defer the cleanup function
5. **Add Context**: Enrich spans with relevant attributes
6. **Use Batch Processing**: Configure batch processing for better performance