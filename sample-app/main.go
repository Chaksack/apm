package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gofiber/contrib/otelfiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	// Metrics
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route", "status_code"})

	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "route", "status_code"})

	// Logger
	log *zap.Logger

	// Tracer
	tracer trace.Tracer
)

func initLogger() error {
	config := zap.NewProductionConfig()
	
	// Configure based on environment
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	
	level, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	config.Level = zap.NewAtomicLevelAt(level)
	
	// Use JSON format for structured logging
	config.Encoding = "json"
	
	// Add service info to all logs
	config.InitialFields = map[string]interface{}{
		"service": getEnv("APP_NAME", "sample-gofiber-app"),
		"version": "1.0.0",
	}
	
	log, err = config.Build()
	return err
}

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	// Get OTLP endpoint from environment
	endpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "jaeger:4317")
	
	// Create OTLP exporter
	exporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(getEnv("OTEL_SERVICE_NAME", "sample-gofiber-app")),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("environment", getEnv("ENVIRONMENT", "development")),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	tracer = tp.Tracer("sample-gofiber-app")

	return tp, nil
}

func prometheusMiddleware(c *fiber.Ctx) error {
	start := time.Now()

	// Process request
	err := c.Next()

	// Record metrics
	duration := time.Since(start).Seconds()
	statusCode := strconv.Itoa(c.Response().StatusCode())
	route := c.Route().Path
	method := c.Method()

	httpDuration.WithLabelValues(method, route, statusCode).Observe(duration)
	httpRequests.WithLabelValues(method, route, statusCode).Inc()

	return err
}

func structuredLogger(log *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Get trace ID from context
		span := trace.SpanFromContext(c.UserContext())
		traceID := span.SpanContext().TraceID().String()

		// Process request
		err := c.Next()

		// Log request details
		fields := []zapcore.Field{
			zap.String("request_id", c.Locals("requestid").(string)),
			zap.String("trace_id", traceID),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("ip", c.IP()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("latency", time.Since(start)),
			zap.String("user_agent", c.Get("User-Agent")),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			log.Error("Request failed", fields...)
		} else {
			log.Info("Request completed", fields...)
		}

		return err
	}
}

// Handlers

func healthHandler(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "health-check")
	defer span.End()

	log.Info("Health check",
		zap.String("trace_id", span.SpanContext().TraceID().String()),
	)

	return c.JSON(fiber.Map{
		"status": "healthy",
		"time":   time.Now().UTC(),
	})
}

func rootHandler(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "root-handler")
	defer span.End()

	// Simulate some work
	workDuration := time.Duration(rand.Intn(100)) * time.Millisecond
	time.Sleep(workDuration)

	span.SetAttributes(
		attribute.String("handler", "root"),
		attribute.Int64("work_duration_ms", workDuration.Milliseconds()),
	)

	log.Info("Root handler",
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.Duration("work_duration", workDuration),
	)

	return c.JSON(fiber.Map{
		"message":  "Welcome to GoFiber APM Sample App",
		"version":  "1.0.0",
		"trace_id": span.SpanContext().TraceID().String(),
		"features": []string{
			"OpenTelemetry Tracing",
			"Prometheus Metrics",
			"Structured Logging",
			"Health Checks",
		},
	})
}

func slowHandler(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "slow-handler")
	defer span.End()

	// Simulate slow operation
	sleepDuration := time.Duration(rand.Intn(3000)+1000) * time.Millisecond

	log.Info("Slow operation started",
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.Duration("expected_duration", sleepDuration),
	)

	time.Sleep(sleepDuration)

	span.SetAttributes(
		attribute.String("handler", "slow"),
		attribute.Int64("sleep_duration_ms", sleepDuration.Milliseconds()),
	)

	return c.JSON(fiber.Map{
		"message":     "Slow operation completed",
		"duration_ms": sleepDuration.Milliseconds(),
		"trace_id":    span.SpanContext().TraceID().String(),
	})
}

func errorHandler(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "error-handler")
	defer span.End()

	// Randomly return different error codes
	errorCodes := []int{fiber.StatusBadRequest, fiber.StatusNotFound, fiber.StatusInternalServerError}
	statusCode := errorCodes[rand.Intn(len(errorCodes))]

	span.SetAttributes(
		attribute.String("handler", "error"),
		attribute.Int("status_code", statusCode),
	)

	log.Error("Simulated error",
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.Int("status_code", statusCode),
	)

	return c.Status(statusCode).JSON(fiber.Map{
		"error":     "Simulated error",
		"code":      statusCode,
		"trace_id":  span.SpanContext().TraceID().String(),
		"timestamp": time.Now().UTC(),
	})
}

func dataHandler(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "data-handler")
	defer span.End()

	// Simulate data processing
	itemCount := rand.Intn(100) + 1
	items := make([]map[string]interface{}, itemCount)

	for i := 0; i < itemCount; i++ {
		items[i] = map[string]interface{}{
			"id":        i + 1,
			"value":     rand.Float64() * 100,
			"timestamp": time.Now().UTC(),
		}
	}

	span.SetAttributes(
		attribute.String("handler", "data"),
		attribute.Int("item_count", itemCount),
	)

	log.Info("Data generated",
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.Int("item_count", itemCount),
	)

	return c.JSON(fiber.Map{
		"items":     items,
		"count":     itemCount,
		"trace_id":  span.SpanContext().TraceID().String(),
		"timestamp": time.Now().UTC(),
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Initialize logger
	if err := initLogger(); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer log.Sync()

	// Initialize tracer
	ctx := context.Background()
	tp, err := initTracer(ctx)
	if err != nil {
		log.Fatal("Failed to initialize tracer", zap.Error(err))
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Error("Error shutting down tracer provider", zap.Error(err))
		}
	}()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "GoFiber APM Sample App",
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			log.Error("Request error",
				zap.Error(err),
				zap.String("path", c.Path()),
				zap.Int("status", code),
			)

			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
				"code":  code,
			})
		},
	})

	// Middleware
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(otelfiber.Middleware())
	app.Use(prometheusMiddleware)
	app.Use(structuredLogger(log))

	// Routes
	app.Get("/", rootHandler)
	app.Get("/health", healthHandler)
	app.Get("/slow", slowHandler)
	app.Get("/error", errorHandler)
	app.Get("/data", dataHandler)

	// Start metrics server
	metricsPort := getEnv("METRICS_PORT", "9091")
	go func() {
		log.Info("Starting metrics server", zap.String("port", metricsPort))
		
		// Create a new Fiber app for metrics
		metricsApp := fiber.New(fiber.Config{
			DisableStartupMessage: true,
		})
		
		// Prometheus metrics endpoint
		metricsApp.Get("/metrics", func(c *fiber.Ctx) error {
			c.Set("Content-Type", "text/plain")
			handler := promhttp.Handler()
			handler.ServeHTTP(c.Context(), c.Request())
			return nil
		})
		
		if err := metricsApp.Listen(":" + metricsPort); err != nil {
			log.Fatal("Failed to start metrics server", zap.Error(err))
		}
	}()

	// Start main server
	port := getEnv("APP_PORT", "8080")
	
	// Setup graceful shutdown
	go func() {
		log.Info("Starting HTTP server",
			zap.String("port", port),
			zap.String("name", "GoFiber APM Sample App"),
		)
		
		if err := app.Listen(":" + port); err != nil {
			log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Shutdown with timeout
	if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server shutdown complete")
}