package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/ybke/apm/pkg/instrumentation"
)

func main() {
	// Initialize tracer
	ctx := context.Background()
	tracerConfig := instrumentation.TracerConfig{
		ServiceName:    "example-service",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		ExporterType:   "otlp",
		Endpoint:       "localhost:4317",
		SampleRate:     1.0,
	}

	_, cleanup, err := instrumentation.InitTracer(ctx, tracerConfig)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer cleanup()

	// Create Fiber app
	app := fiber.New()

	// Add OpenTelemetry middleware
	app.Use(instrumentation.FiberOtelMiddleware("example-service"))

	// Example route
	app.Get("/", func(c *fiber.Ctx) error {
		// Get context with correlation ID
		ctx := instrumentation.FiberContextWithCorrelation(c)

		// Create a span for some operation
		tracer := instrumentation.GetTracer("example-service")
		ctx, span := instrumentation.StartSpanWithCorrelation(ctx, tracer, "process-request")
		defer span.End()

		// Simulate some work
		time.Sleep(100 * time.Millisecond)

		// Get correlation ID for response
		correlationID := instrumentation.GetCorrelationID(ctx)

		return c.JSON(fiber.Map{
			"message":        "Hello, World!",
			"correlation_id": correlationID,
		})
	})

	// Start server
	log.Fatal(app.Listen(":3000"))
}