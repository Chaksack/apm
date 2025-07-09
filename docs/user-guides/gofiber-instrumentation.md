# Go Fiber Instrumentation Guide

## Overview

This guide covers comprehensive instrumentation of Go Fiber applications with OpenTelemetry, Prometheus metrics, distributed tracing, and structured logging for effective application performance monitoring (APM).

## Prerequisites

### Required Dependencies

```bash
go get github.com/gofiber/fiber/v2
go get github.com/gofiber/fiber/v2/middleware/monitor
go get github.com/gofiber/fiber/v2/middleware/logger
go get github.com/gofiber/fiber/v2/middleware/recover
go get github.com/gofiber/fiber/v2/middleware/cors
go get github.com/gofiber/fiber/v2/middleware/requestid

# OpenTelemetry
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/trace
go get go.opentelemetry.io/otel/exporters/jaeger
go get go.opentelemetry.io/otel/exporters/prometheus
go get go.opentelemetry.io/otel/sdk/trace
go get go.opentelemetry.io/otel/sdk/metric
go get go.opentelemetry.io/otel/semconv/v1.4.0
go get go.opentelemetry.io/contrib/instrumentation/github.com/gofiber/fiber/otelfiber

# Prometheus
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp

# Logging
go get github.com/sirupsen/logrus
go get github.com/rs/zerolog
```

## Basic Instrumentation Setup

### Application Structure

```
project/
├── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── middleware/
│   │   ├── metrics.go
│   │   ├── tracing.go
│   │   └── logging.go
│   ├── handlers/
│   │   └── handlers.go
│   └── telemetry/
│       ├── telemetry.go
│       ├── metrics.go
│       └── tracing.go
└── docker-compose.yml
```

### Configuration Setup

```go
// internal/config/config.go
package config

import (
    "os"
    "strconv"
    "time"
)

type Config struct {
    Server   ServerConfig
    Telemetry TelemetryConfig
    Database DatabaseConfig
}

type ServerConfig struct {
    Host         string
    Port         int
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
}

type TelemetryConfig struct {
    ServiceName    string
    ServiceVersion string
    Environment    string
    JaegerEndpoint string
    PrometheusPort int
}

type DatabaseConfig struct {
    Host     string
    Port     int
    Database string
    Username string
    Password string
}

func Load() *Config {
    port, _ := strconv.Atoi(getEnv("PORT", "3000"))
    prometheusPort, _ := strconv.Atoi(getEnv("PROMETHEUS_PORT", "2112"))
    
    return &Config{
        Server: ServerConfig{
            Host:         getEnv("HOST", "localhost"),
            Port:         port,
            ReadTimeout:  30 * time.Second,
            WriteTimeout: 30 * time.Second,
        },
        Telemetry: TelemetryConfig{
            ServiceName:    getEnv("SERVICE_NAME", "fiber-app"),
            ServiceVersion: getEnv("SERVICE_VERSION", "1.0.0"),
            Environment:    getEnv("ENVIRONMENT", "development"),
            JaegerEndpoint: getEnv("JAEGER_ENDPOINT", "http://localhost:14268/api/traces"),
            PrometheusPort: prometheusPort,
        },
        Database: DatabaseConfig{
            Host:     getEnv("DB_HOST", "localhost"),
            Port:     5432,
            Database: getEnv("DB_NAME", "myapp"),
            Username: getEnv("DB_USER", "postgres"),
            Password: getEnv("DB_PASSWORD", ""),
        },
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

### Telemetry Initialization

```go
// internal/telemetry/telemetry.go
package telemetry

import (
    "context"
    "fmt"
    "log"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/exporters/prometheus"
    "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type Telemetry struct {
    TracerProvider *trace.TracerProvider
    MeterProvider  *metric.MeterProvider
    Exporter       *prometheus.Exporter
}

func Initialize(serviceName, serviceVersion, environment string, jaegerEndpoint string) (*Telemetry, error) {
    // Create resource
    res, err := resource.New(
        context.Background(),
        resource.WithAttributes(
            semconv.ServiceNameKey.String(serviceName),
            semconv.ServiceVersionKey.String(serviceVersion),
            semconv.DeploymentEnvironmentKey.String(environment),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create resource: %w", err)
    }

    // Initialize tracing
    tracerProvider, err := initTracing(res, jaegerEndpoint)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize tracing: %w", err)
    }

    // Initialize metrics
    meterProvider, promExporter, err := initMetrics(res)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize metrics: %w", err)
    }

    // Set global providers
    otel.SetTracerProvider(tracerProvider)
    otel.SetMeterProvider(meterProvider)

    return &Telemetry{
        TracerProvider: tracerProvider,
        MeterProvider:  meterProvider,
        Exporter:       promExporter,
    }, nil
}

func initTracing(res *resource.Resource, jaegerEndpoint string) (*trace.TracerProvider, error) {
    exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
    if err != nil {
        return nil, err
    }

    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(res),
        trace.WithSampler(trace.AlwaysSample()),
    )

    return tp, nil
}

func initMetrics(res *resource.Resource) (*metric.MeterProvider, *prometheus.Exporter, error) {
    exporter, err := prometheus.New()
    if err != nil {
        return nil, nil, err
    }

    provider := metric.NewMeterProvider(
        metric.WithReader(exporter),
        metric.WithResource(res),
    )

    return provider, exporter, nil
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
    if err := t.TracerProvider.Shutdown(ctx); err != nil {
        return err
    }
    if err := t.MeterProvider.Shutdown(ctx); err != nil {
        return err
    }
    return nil
}
```

## Custom Metrics Implementation

### Metrics Collection

```go
// internal/telemetry/metrics.go
package telemetry

import (
    "context"
    "strconv"
    "time"

    "github.com/gofiber/fiber/v2"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

type Metrics struct {
    RequestDuration  metric.Float64Histogram
    RequestCount     metric.Int64Counter
    ActiveRequests   metric.Int64UpDownCounter
    ResponseSize     metric.Int64Histogram
    ErrorCount       metric.Int64Counter
    DatabaseQueries  metric.Int64Counter
    CacheHits        metric.Int64Counter
    CacheMisses      metric.Int64Counter
}

func NewMetrics() (*Metrics, error) {
    meter := otel.Meter("fiber-app")

    requestDuration, err := meter.Float64Histogram(
        "http_request_duration_seconds",
        metric.WithDescription("Duration of HTTP requests in seconds"),
        metric.WithUnit("s"),
    )
    if err != nil {
        return nil, err
    }

    requestCount, err := meter.Int64Counter(
        "http_requests_total",
        metric.WithDescription("Total number of HTTP requests"),
    )
    if err != nil {
        return nil, err
    }

    activeRequests, err := meter.Int64UpDownCounter(
        "http_requests_active",
        metric.WithDescription("Number of active HTTP requests"),
    )
    if err != nil {
        return nil, err
    }

    responseSize, err := meter.Int64Histogram(
        "http_response_size_bytes",
        metric.WithDescription("Size of HTTP responses in bytes"),
        metric.WithUnit("By"),
    )
    if err != nil {
        return nil, err
    }

    errorCount, err := meter.Int64Counter(
        "http_errors_total",
        metric.WithDescription("Total number of HTTP errors"),
    )
    if err != nil {
        return nil, err
    }

    databaseQueries, err := meter.Int64Counter(
        "database_queries_total",
        metric.WithDescription("Total number of database queries"),
    )
    if err != nil {
        return nil, err
    }

    cacheHits, err := meter.Int64Counter(
        "cache_hits_total",
        metric.WithDescription("Total number of cache hits"),
    )
    if err != nil {
        return nil, err
    }

    cacheMisses, err := meter.Int64Counter(
        "cache_misses_total",
        metric.WithDescription("Total number of cache misses"),
    )
    if err != nil {
        return nil, err
    }

    return &Metrics{
        RequestDuration:  requestDuration,
        RequestCount:     requestCount,
        ActiveRequests:   activeRequests,
        ResponseSize:     responseSize,
        ErrorCount:       errorCount,
        DatabaseQueries:  databaseQueries,
        CacheHits:        cacheHits,
        CacheMisses:      cacheMisses,
    }, nil
}

func (m *Metrics) RecordRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration, responseSize int64) {
    labels := []attribute.KeyValue{
        attribute.String("method", method),
        attribute.String("path", path),
        attribute.String("status_code", strconv.Itoa(statusCode)),
    }

    m.RequestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(labels...))
    m.RequestCount.Add(ctx, 1, metric.WithAttributes(labels...))
    m.ResponseSize.Record(ctx, responseSize, metric.WithAttributes(labels...))

    if statusCode >= 400 {
        m.ErrorCount.Add(ctx, 1, metric.WithAttributes(labels...))
    }
}

func (m *Metrics) IncActiveRequests(ctx context.Context) {
    m.ActiveRequests.Add(ctx, 1)
}

func (m *Metrics) DecActiveRequests(ctx context.Context) {
    m.ActiveRequests.Add(ctx, -1)
}

func (m *Metrics) RecordDatabaseQuery(ctx context.Context, operation string, duration time.Duration) {
    labels := []attribute.KeyValue{
        attribute.String("operation", operation),
    }
    m.DatabaseQueries.Add(ctx, 1, metric.WithAttributes(labels...))
}

func (m *Metrics) RecordCacheHit(ctx context.Context, key string) {
    labels := []attribute.KeyValue{
        attribute.String("key", key),
    }
    m.CacheHits.Add(ctx, 1, metric.WithAttributes(labels...))
}

func (m *Metrics) RecordCacheMiss(ctx context.Context, key string) {
    labels := []attribute.KeyValue{
        attribute.String("key", key),
    }
    m.CacheMisses.Add(ctx, 1, metric.WithAttributes(labels...))
}
```

### Metrics Middleware

```go
// internal/middleware/metrics.go
package middleware

import (
    "context"
    "strconv"
    "time"

    "github.com/gofiber/fiber/v2"
    "your-app/internal/telemetry"
)

func MetricsMiddleware(metrics *telemetry.Metrics) fiber.Handler {
    return func(c *fiber.Ctx) error {
        start := time.Now()
        ctx := context.Background()

        // Increment active requests
        metrics.IncActiveRequests(ctx)
        defer metrics.DecActiveRequests(ctx)

        // Process request
        err := c.Next()

        // Record metrics
        duration := time.Since(start)
        statusCode := c.Response().StatusCode()
        responseSize := int64(len(c.Response().Body()))
        
        metrics.RecordRequest(
            ctx,
            c.Method(),
            c.Path(),
            statusCode,
            duration,
            responseSize,
        )

        return err
    }
}
```

## Distributed Tracing Integration

### Tracing Middleware

```go
// internal/middleware/tracing.go
package middleware

import (
    "github.com/gofiber/fiber/v2"
    "go.opentelemetry.io/contrib/instrumentation/github.com/gofiber/fiber/otelfiber"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func TracingMiddleware(serviceName string) fiber.Handler {
    return otelfiber.Middleware(
        otelfiber.WithTracerProvider(otel.GetTracerProvider()),
        otelfiber.WithSpanNameFormatter(func(ctx *fiber.Ctx) string {
            return ctx.Method() + " " + ctx.Path()
        }),
    )
}

func CustomTracingMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        tracer := otel.Tracer("fiber-app")
        
        ctx, span := tracer.Start(c.Context(), c.Method()+" "+c.Path())
        defer span.End()
        
        // Add custom attributes
        span.SetAttributes(
            attribute.String("http.method", c.Method()),
            attribute.String("http.url", c.OriginalURL()),
            attribute.String("http.scheme", c.Protocol()),
            attribute.String("user_agent", c.Get("User-Agent")),
            attribute.String("request_id", c.Get("X-Request-ID")),
        )
        
        // Set context for downstream handlers
        c.SetUserContext(ctx)
        
        // Process request
        err := c.Next()
        
        // Record response attributes
        span.SetAttributes(
            attribute.Int("http.status_code", c.Response().StatusCode()),
            attribute.Int64("http.response_size", int64(len(c.Response().Body()))),
        )
        
        // Record error if present
        if err != nil {
            span.RecordError(err)
            span.SetStatus(trace.StatusCodeError, err.Error())
        }
        
        return err
    }
}
```

### Manual Instrumentation

```go
// internal/handlers/handlers.go
package handlers

import (
    "context"
    "fmt"
    "time"

    "github.com/gofiber/fiber/v2"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
    "your-app/internal/telemetry"
)

type Handler struct {
    metrics *telemetry.Metrics
    tracer  trace.Tracer
}

func NewHandler(metrics *telemetry.Metrics) *Handler {
    return &Handler{
        metrics: metrics,
        tracer:  otel.Tracer("fiber-app-handlers"),
    }
}

func (h *Handler) GetUser(c *fiber.Ctx) error {
    ctx, span := h.tracer.Start(c.Context(), "get_user")
    defer span.End()

    userID := c.Params("id")
    span.SetAttributes(attribute.String("user.id", userID))

    // Database operation with tracing
    user, err := h.getUserFromDB(ctx, userID)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusCodeError, err.Error())
        return c.Status(500).JSON(fiber.Map{"error": "Internal server error"})
    }

    span.SetAttributes(
        attribute.String("user.name", user.Name),
        attribute.String("user.email", user.Email),
    )

    return c.JSON(user)
}

func (h *Handler) getUserFromDB(ctx context.Context, userID string) (*User, error) {
    ctx, span := h.tracer.Start(ctx, "db_get_user")
    defer span.End()

    span.SetAttributes(
        attribute.String("db.operation", "SELECT"),
        attribute.String("db.table", "users"),
        attribute.String("db.query", "SELECT * FROM users WHERE id = ?"),
    )

    start := time.Now()
    
    // Simulate database call
    time.Sleep(50 * time.Millisecond)
    
    // Record database metrics
    h.metrics.RecordDatabaseQuery(ctx, "SELECT", time.Since(start))

    // Return mock user
    return &User{
        ID:    userID,
        Name:  "John Doe",
        Email: "john@example.com",
    }, nil
}

func (h *Handler) CreateUser(c *fiber.Ctx) error {
    ctx, span := h.tracer.Start(c.Context(), "create_user")
    defer span.End()

    var user User
    if err := c.BodyParser(&user); err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusCodeError, "Failed to parse request body")
        return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
    }

    // Validate user data
    if err := h.validateUser(ctx, &user); err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusCodeError, "Validation failed")
        return c.Status(400).JSON(fiber.Map{"error": err.Error()})
    }

    // Save to database
    if err := h.saveUserToDB(ctx, &user); err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusCodeError, "Failed to save user")
        return c.Status(500).JSON(fiber.Map{"error": "Failed to create user"})
    }

    span.SetAttributes(
        attribute.String("user.id", user.ID),
        attribute.String("user.name", user.Name),
    )

    return c.Status(201).JSON(user)
}

func (h *Handler) validateUser(ctx context.Context, user *User) error {
    ctx, span := h.tracer.Start(ctx, "validate_user")
    defer span.End()

    if user.Name == "" {
        return fmt.Errorf("name is required")
    }
    if user.Email == "" {
        return fmt.Errorf("email is required")
    }

    span.SetAttributes(
        attribute.Bool("validation.passed", true),
    )

    return nil
}

func (h *Handler) saveUserToDB(ctx context.Context, user *User) error {
    ctx, span := h.tracer.Start(ctx, "db_save_user")
    defer span.End()

    span.SetAttributes(
        attribute.String("db.operation", "INSERT"),
        attribute.String("db.table", "users"),
    )

    start := time.Now()
    
    // Simulate database save
    time.Sleep(100 * time.Millisecond)
    
    // Record database metrics
    h.metrics.RecordDatabaseQuery(ctx, "INSERT", time.Since(start))

    // Generate ID
    user.ID = "user-123"

    return nil
}

type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

## Structured Logging

### Logging Middleware

```go
// internal/middleware/logging.go
package middleware

import (
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/sirupsen/logrus"
    "go.opentelemetry.io/otel/trace"
)

func LoggingMiddleware(logger *logrus.Logger) fiber.Handler {
    return func(c *fiber.Ctx) error {
        start := time.Now()
        
        // Process request
        err := c.Next()
        
        // Extract trace information
        span := trace.SpanFromContext(c.Context())
        traceID := span.SpanContext().TraceID().String()
        spanID := span.SpanContext().SpanID().String()
        
        // Log request details
        logger.WithFields(logrus.Fields{
            "timestamp":     start.Format(time.RFC3339),
            "method":        c.Method(),
            "path":          c.Path(),
            "status_code":   c.Response().StatusCode(),
            "duration_ms":   time.Since(start).Milliseconds(),
            "user_agent":    c.Get("User-Agent"),
            "remote_addr":   c.IP(),
            "request_id":    c.Get("X-Request-ID"),
            "trace_id":      traceID,
            "span_id":       spanID,
            "response_size": len(c.Response().Body()),
        }).Info("HTTP Request")
        
        if err != nil {
            logger.WithFields(logrus.Fields{
                "error":    err.Error(),
                "trace_id": traceID,
                "span_id":  spanID,
            }).Error("Request failed")
        }
        
        return err
    }
}

func StructuredLogger() *logrus.Logger {
    logger := logrus.New()
    logger.SetFormatter(&logrus.JSONFormatter{
        TimestampFormat: time.RFC3339,
    })
    logger.SetLevel(logrus.InfoLevel)
    
    return logger
}
```

## Performance Monitoring

### Health Check Endpoint

```go
// internal/handlers/health.go
package handlers

import (
    "context"
    "time"

    "github.com/gofiber/fiber/v2"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

type HealthHandler struct {
    tracer trace.Tracer
}

func NewHealthHandler() *HealthHandler {
    return &HealthHandler{
        tracer: otel.Tracer("health-check"),
    }
}

func (h *HealthHandler) HealthCheck(c *fiber.Ctx) error {
    ctx, span := h.tracer.Start(c.Context(), "health_check")
    defer span.End()

    // Check database connectivity
    dbStatus := h.checkDatabase(ctx)
    
    // Check external services
    externalStatus := h.checkExternalServices(ctx)
    
    status := "healthy"
    statusCode := 200
    
    if !dbStatus || !externalStatus {
        status = "unhealthy"
        statusCode = 503
    }

    response := fiber.Map{
        "status":    status,
        "timestamp": time.Now().Format(time.RFC3339),
        "version":   "1.0.0",
        "checks": fiber.Map{
            "database": dbStatus,
            "external": externalStatus,
        },
    }

    span.SetAttributes(
        attribute.String("health.status", status),
        attribute.Bool("health.database", dbStatus),
        attribute.Bool("health.external", externalStatus),
    )

    return c.Status(statusCode).JSON(response)
}

func (h *HealthHandler) checkDatabase(ctx context.Context) bool {
    ctx, span := h.tracer.Start(ctx, "health_check_database")
    defer span.End()

    // Simulate database check
    time.Sleep(10 * time.Millisecond)
    
    return true
}

func (h *HealthHandler) checkExternalServices(ctx context.Context) bool {
    ctx, span := h.tracer.Start(ctx, "health_check_external")
    defer span.End()

    // Simulate external service check
    time.Sleep(20 * time.Millisecond)
    
    return true
}
```

### Monitoring Dashboard

```go
// internal/handlers/monitoring.go
package handlers

import (
    "runtime"
    "time"

    "github.com/gofiber/fiber/v2"
)

type MonitoringHandler struct{}

func NewMonitoringHandler() *MonitoringHandler {
    return &MonitoringHandler{}
}

func (h *MonitoringHandler) GetMetrics(c *fiber.Ctx) error {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)

    metrics := fiber.Map{
        "timestamp": time.Now().Format(time.RFC3339),
        "memory": fiber.Map{
            "alloc":         m.Alloc,
            "total_alloc":   m.TotalAlloc,
            "sys":           m.Sys,
            "num_gc":        m.NumGC,
            "gc_cpu_fraction": m.GCCPUFraction,
        },
        "runtime": fiber.Map{
            "goroutines": runtime.NumGoroutine(),
            "gomaxprocs": runtime.GOMAXPROCS(0),
        },
    }

    return c.JSON(metrics)
}
```

## Main Application Setup

### Application Bootstrap

```go
// main.go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "github.com/gofiber/fiber/v2/middleware/recover"
    "github.com/gofiber/fiber/v2/middleware/requestid"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "your-app/internal/config"
    "your-app/internal/handlers"
    "your-app/internal/middleware"
    "your-app/internal/telemetry"
)

func main() {
    // Load configuration
    cfg := config.Load()

    // Initialize telemetry
    tel, err := telemetry.Initialize(
        cfg.Telemetry.ServiceName,
        cfg.Telemetry.ServiceVersion,
        cfg.Telemetry.Environment,
        cfg.Telemetry.JaegerEndpoint,
    )
    if err != nil {
        log.Fatal("Failed to initialize telemetry:", err)
    }
    defer tel.Shutdown(context.Background())

    // Initialize metrics
    metrics, err := telemetry.NewMetrics()
    if err != nil {
        log.Fatal("Failed to initialize metrics:", err)
    }

    // Initialize logger
    logger := middleware.StructuredLogger()

    // Create Fiber app
    app := fiber.New(fiber.Config{
        ReadTimeout:  cfg.Server.ReadTimeout,
        WriteTimeout: cfg.Server.WriteTimeout,
        ErrorHandler: func(c *fiber.Ctx, err error) error {
            code := fiber.StatusInternalServerError
            if e, ok := err.(*fiber.Error); ok {
                code = e.Code
            }
            
            logger.WithFields(logrus.Fields{
                "error":      err.Error(),
                "status":     code,
                "method":     c.Method(),
                "path":       c.Path(),
                "request_id": c.Get("X-Request-ID"),
            }).Error("Request error")
            
            return c.Status(code).JSON(fiber.Map{
                "error": err.Error(),
            })
        },
    })

    // Add middleware
    app.Use(recover.New())
    app.Use(requestid.New())
    app.Use(cors.New(cors.Config{
        AllowOrigins: "*",
        AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
        AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Request-ID",
    }))
    app.Use(middleware.LoggingMiddleware(logger))
    app.Use(middleware.TracingMiddleware(cfg.Telemetry.ServiceName))
    app.Use(middleware.MetricsMiddleware(metrics))

    // Initialize handlers
    userHandler := handlers.NewHandler(metrics)
    healthHandler := handlers.NewHealthHandler()
    monitoringHandler := handlers.NewMonitoringHandler()

    // Routes
    api := app.Group("/api/v1")
    {
        api.Get("/users/:id", userHandler.GetUser)
        api.Post("/users", userHandler.CreateUser)
        api.Get("/health", healthHandler.HealthCheck)
        api.Get("/metrics", monitoringHandler.GetMetrics)
    }

    // Start Prometheus metrics server
    go func() {
        http.Handle("/metrics", promhttp.Handler())
        log.Printf("Prometheus metrics server listening on :%d", cfg.Telemetry.PrometheusPort)
        if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Telemetry.PrometheusPort), nil); err != nil {
            log.Fatal("Failed to start metrics server:", err)
        }
    }()

    // Start main server
    go func() {
        addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
        log.Printf("Server starting on %s", addr)
        if err := app.Listen(addr); err != nil {
            log.Fatal("Failed to start server:", err)
        }
    }()

    // Wait for interrupt signal
    c := make(chan os.Signal, 1)
    signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
    <-c

    log.Println("Shutting down server...")
    
    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := app.ShutdownWithContext(ctx); err != nil {
        log.Printf("Server shutdown error: %v", err)
    }
    
    log.Println("Server stopped")
}
```

## Testing and Validation

### Load Testing

```bash
# Install hey load testing tool
go install github.com/rakyll/hey@latest

# Run load test
hey -n 1000 -c 10 -m GET http://localhost:3000/api/v1/users/123

# Test with different endpoints
hey -n 500 -c 5 -m POST -d '{"name":"John","email":"john@example.com"}' \
    -H "Content-Type: application/json" \
    http://localhost:3000/api/v1/users
```

### Metrics Validation

```bash
# Check Prometheus metrics
curl http://localhost:2112/metrics

# Check specific metrics
curl http://localhost:2112/metrics | grep http_requests_total
curl http://localhost:2112/metrics | grep http_request_duration_seconds
```

### Trace Validation

```bash
# Check Jaeger traces
curl "http://localhost:16686/api/traces?service=fiber-app&start=$(date -d '1 hour ago' +%s)000000&end=$(date +%s)000000"
```

## Best Practices

### Performance Optimization

1. **Sampling Strategy**: Use appropriate sampling rates
2. **Batch Processing**: Configure batch sizes for exporters
3. **Resource Limits**: Set reasonable resource limits
4. **Context Propagation**: Ensure proper context passing
5. **Metric Cardinality**: Keep label cardinality reasonable

### Security Considerations

1. **Sensitive Data**: Never log sensitive information
2. **Trace Filtering**: Filter out sensitive trace data
3. **Access Control**: Secure monitoring endpoints
4. **Data Retention**: Configure appropriate retention policies
5. **Transport Security**: Use TLS for data transmission

### Operational Guidelines

1. **Monitoring**: Monitor the monitoring system itself
2. **Alerting**: Set up alerts for critical metrics
3. **Documentation**: Document custom metrics and traces
4. **Testing**: Test instrumentation changes thoroughly
5. **Gradual Rollout**: Deploy instrumentation changes gradually

## Troubleshooting

### Common Issues

1. **Missing Traces**: Check sampling configuration
2. **High Memory Usage**: Optimize batch sizes and retention
3. **Slow Performance**: Review instrumentation overhead
4. **Missing Metrics**: Verify metric registration
5. **Context Loss**: Ensure proper context propagation

### Debug Tools

1. **Prometheus Metrics**: Check metric collection
2. **Jaeger UI**: Verify trace collection
3. **Application Logs**: Review structured logs
4. **Health Checks**: Monitor service health
5. **Performance Profiling**: Use Go pprof for analysis

## Resources

- [OpenTelemetry Go Documentation](https://opentelemetry.io/docs/instrumentation/go/)
- [Prometheus Go Client](https://github.com/prometheus/client_golang)
- [Jaeger Go Client](https://github.com/jaegertracing/jaeger-client-go)
- [Fiber Documentation](https://docs.gofiber.io/)
- [Go Fiber Middleware](https://github.com/gofiber/fiber/tree/master/middleware)