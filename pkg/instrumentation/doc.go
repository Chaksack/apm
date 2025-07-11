// Package instrumentation provides production-ready OpenTelemetry SDK integration
// for Go applications, with special support for GoFiber web framework.
//
// Features:
//
// - Multiple Exporters: Support for OTLP (gRPC/HTTP), Jaeger, and stdout exporters
// - Multi-Exporter Setup: Export traces to multiple destinations simultaneously
// - GoFiber Integration: Middleware for automatic trace context propagation
// - Correlation ID Management: Built-in correlation ID generation and propagation
// - Baggage Support: Propagate contextual data across service boundaries
// - Configurable Sampling: Control trace sampling rates
// - Batch Processing: Efficient trace export with configurable batch processing
//
// Basic Setup:
//
//	import (
//	    "context"
//	    "github.com/yourusername/apm/pkg/instrumentation"
//	)
//
//	// Initialize tracer
//	ctx := context.Background()
//	config := instrumentation.TracerConfig{
//	    ServiceName:    "apm",
//	    ServiceVersion: "1.0.0",
//	    Environment:    "production",
//	    ExporterType:   "otlp",
//	    Endpoint:       "localhost:4317",
//	    SampleRate:     0.1, // Sample 10% of traces
//	}
//
//	tracerProvider, cleanup, err := instrumentation.InitTracer(ctx, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cleanup()
//
// GoFiber Middleware:
//
//	import (
//	    "github.com/gofiber/fiber/v2"
//	    "github.com/yourusername/apm/pkg/instrumentation"
//	)
//
//	app := fiber.New()
//
//	// Add OpenTelemetry middleware
//	app.Use(instrumentation.FiberOtelMiddleware("my-service"))
//
// For more examples and detailed documentation, see the package README.md
// or visit https://github.com/yourusername/apm
package instrumentation
