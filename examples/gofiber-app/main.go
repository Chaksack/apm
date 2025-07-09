package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/contrib/otelfiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ybke/apm/internal/middleware"
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
	// Global logger
	log *zap.Logger

	// Global tracer
	tracer trace.Tracer

	// Custom metrics
	businessMetrics = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "business_operations_total",
			Help: "Total number of business operations",
		},
		[]string{"operation", "status"},
	)

	operationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "business_operation_duration_seconds",
			Help:    "Duration of business operations in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"operation"},
	)

	// Service configuration
	appConfig *AppConfig
)

// AppConfig holds application configuration
type AppConfig struct {
	AppName         string
	Version         string
	Environment     string
	Port            string
	MetricsPort     string
	LogLevel        string
	TracingEnabled  bool
	TracingEndpoint string
	DBEnabled       bool
	CacheEnabled    bool
}

func init() {
	// Register custom metrics
	prometheus.MustRegister(businessMetrics)
	prometheus.MustRegister(operationDuration)
}

// initConfig initializes application configuration from environment variables
func initConfig() *AppConfig {
	return &AppConfig{
		AppName:         getEnv("APP_NAME", "gofiber-example-app"),
		Version:         getEnv("APP_VERSION", "1.0.0"),
		Environment:     getEnv("ENVIRONMENT", "development"),
		Port:            getEnv("APP_PORT", "8080"),
		MetricsPort:     getEnv("METRICS_PORT", "9091"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		TracingEnabled:  getEnv("TRACING_ENABLED", "true") == "true",
		TracingEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "jaeger:4317"),
		DBEnabled:       getEnv("DB_ENABLED", "true") == "true",
		CacheEnabled:    getEnv("CACHE_ENABLED", "true") == "true",
	}
}

// initLogger initializes the structured logger
func initLogger(cfg *AppConfig) error {
	config := zap.NewProductionConfig()

	level, err := zapcore.ParseLevel(cfg.LogLevel)
	if err != nil {
		return err
	}
	config.Level = zap.NewAtomicLevelAt(level)

	// Use JSON format for structured logging
	config.Encoding = "json"

	// Add service metadata to all logs
	config.InitialFields = map[string]interface{}{
		"service":     cfg.AppName,
		"version":     cfg.Version,
		"environment": cfg.Environment,
	}

	log, err = config.Build()
	return err
}

// initTracer initializes OpenTelemetry tracing
func initTracer(ctx context.Context, cfg *AppConfig) (*sdktrace.TracerProvider, error) {
	if !cfg.TracingEnabled {
		log.Info("Tracing is disabled")
		return nil, nil
	}

	// Create OTLP exporter
	exporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(cfg.TracingEndpoint),
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.AppName),
			semconv.ServiceVersion(cfg.Version),
			attribute.String("environment", cfg.Environment),
			attribute.String("deployment.environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider with batching
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	tracer = tp.Tracer(cfg.AppName)

	log.Info("Tracing initialized",
		zap.String("endpoint", cfg.TracingEndpoint),
		zap.String("service", cfg.AppName),
	)

	return tp, nil
}

// structuredLoggingMiddleware adds structured logging with trace correlation
func structuredLoggingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Get request ID
		requestID := c.Locals("requestid").(string)

		// Get trace ID from context
		span := trace.SpanFromContext(c.UserContext())
		traceID := span.SpanContext().TraceID().String()

		// Add trace context to logger
		ctxLogger := log.With(
			zap.String("request_id", requestID),
			zap.String("trace_id", traceID),
		)

		// Store logger in context for handlers to use
		c.Locals("logger", ctxLogger)

		// Process request
		err := c.Next()

		// Log request completion
		fields := []zapcore.Field{
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("ip", c.IP()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("latency", time.Since(start)),
			zap.String("user_agent", c.Get("User-Agent")),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			ctxLogger.Error("Request failed", fields...)
		} else {
			ctxLogger.Info("Request completed", fields...)
		}

		return err
	}
}

// errorHandler provides centralized error handling
func errorHandler(c *fiber.Ctx, err error) error {
	// Get logger from context
	logger := c.Locals("logger").(*zap.Logger)

	// Default to 500
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	// Check if it's a Fiber error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Log the error
	logger.Error("Request error",
		zap.Error(err),
		zap.Int("status", code),
		zap.String("path", c.Path()),
		zap.String("method", c.Method()),
	)

	// Record error metric
	businessMetrics.WithLabelValues("request", "error").Inc()

	// Return error response
	return c.Status(code).JSON(fiber.Map{
		"error": fiber.Map{
			"message":    message,
			"code":       code,
			"request_id": c.Locals("requestid"),
			"timestamp":  time.Now().UTC(),
		},
	})
}

// setupRoutes configures all application routes
func setupRoutes(app *fiber.App, services *Services) {
	// Health check endpoint
	app.Get("/health", NewHealthHandler(services))

	// API routes
	api := app.Group("/api/v1")

	// User endpoints
	api.Get("/users", NewListUsersHandler(services))
	api.Get("/users/:id", NewGetUserHandler(services))
	api.Post("/users", NewCreateUserHandler(services))
	api.Put("/users/:id", NewUpdateUserHandler(services))
	api.Delete("/users/:id", NewDeleteUserHandler(services))

	// Product endpoints
	api.Get("/products", NewListProductsHandler(services))
	api.Get("/products/:id", NewGetProductHandler(services))
	api.Post("/products", NewCreateProductHandler(services))

	// Order endpoints
	api.Post("/orders", NewCreateOrderHandler(services))
	api.Get("/orders/:id", NewGetOrderHandler(services))

	// Analytics endpoint
	api.Get("/analytics/dashboard", NewAnalyticsDashboardHandler(services))

	// Test endpoints
	api.Get("/test/slow", NewSlowHandler())
	api.Get("/test/error", NewErrorHandler())
	api.Get("/test/panic", NewPanicHandler())
}

// startMetricsServer starts the Prometheus metrics server
func startMetricsServer(port string) {
	metricsApp := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Prometheus metrics endpoint
	metricsApp.Get("/metrics", func(c *fiber.Ctx) error {
		handler := promhttp.Handler()
		handler.ServeHTTP(c.Context(), c.Request())
		return nil
	})

	// Health check for metrics server
	metricsApp.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	log.Info("Starting metrics server", zap.String("port", port))

	if err := metricsApp.Listen(":" + port); err != nil {
		log.Fatal("Failed to start metrics server", zap.Error(err))
	}
}

// getEnv retrieves environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Initialize configuration
	appConfig = initConfig()

	// Initialize logger
	if err := initLogger(appConfig); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer log.Sync()

	log.Info("Starting application",
		zap.String("name", appConfig.AppName),
		zap.String("version", appConfig.Version),
		zap.String("environment", appConfig.Environment),
	)

	// Initialize tracer
	ctx := context.Background()
	tp, err := initTracer(ctx, appConfig)
	if err != nil {
		log.Fatal("Failed to initialize tracer", zap.Error(err))
	}
	if tp != nil {
		defer func() {
			if err := tp.Shutdown(ctx); err != nil {
				log.Error("Error shutting down tracer provider", zap.Error(err))
			}
		}()
	}

	// Initialize services
	services, err := NewServices(appConfig)
	if err != nil {
		log.Fatal("Failed to initialize services", zap.Error(err))
	}
	defer services.Close()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               appConfig.AppName,
		DisableStartupMessage: true,
		ErrorHandler:          errorHandler,
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           120 * time.Second,
	})

	// Global middleware
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
	app.Use(requestid.New())
	app.Use(cors.New())

	// OpenTelemetry middleware (if tracing is enabled)
	if appConfig.TracingEnabled {
		app.Use(otelfiber.Middleware())
	}

	// Prometheus metrics middleware
	app.Use(middleware.PrometheusMetrics())

	// Structured logging middleware
	app.Use(structuredLoggingMiddleware())

	// Setup routes
	setupRoutes(app, services)

	// Start metrics server in a goroutine
	go startMetricsServer(appConfig.MetricsPort)

	// Start main server in a goroutine
	go func() {
		log.Info("Starting HTTP server",
			zap.String("port", appConfig.Port),
			zap.String("name", appConfig.AppName),
		)

		if err := app.Listen(":" + appConfig.Port); err != nil {
			log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server shutdown complete")
}
