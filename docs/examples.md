# Instrumentation Examples

This document provides practical code examples for common instrumentation scenarios.

## Code Snippets for Common Scenarios

### Basic HTTP Server Instrumentation

```go
package main

import (
    "context"
    "log"
    
    "github.com/gofiber/fiber/v2"
    "github.com/yourusername/apm/pkg/instrumentation"
    "go.opentelemetry.io/otel/attribute"
)

func main() {
    // Initialize instrumentation
    instr, err := instrumentation.New(
        instrumentation.WithServiceName("api-gateway"),
        instrumentation.WithEnvironment("production"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer instr.Shutdown()

    app := fiber.New()
    
    // Add instrumentation middleware
    app.Use(instr.FiberMiddleware())
    
    // Example endpoint with custom instrumentation
    app.Get("/api/users/:id", func(c *fiber.Ctx) error {
        ctx := c.UserContext()
        
        // Get current span
        span := trace.SpanFromContext(ctx)
        span.SetAttributes(
            attribute.String("user.id", c.Params("id")),
            attribute.String("user.ip", c.IP()),
        )
        
        // Business logic with instrumentation
        user, err := getUserWithInstrumentation(ctx, c.Params("id"))
        if err != nil {
            span.RecordError(err)
            return c.Status(500).JSON(fiber.Map{"error": "Internal server error"})
        }
        
        return c.JSON(user)
    })
    
    log.Fatal(app.Listen(":3000"))
}
```

### Database Operation Instrumentation

```go
package database

import (
    "context"
    "database/sql"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/metric"
)

type InstrumentedDB struct {
    db     *sql.DB
    tracer trace.Tracer
    meter  metric.Meter
}

func NewInstrumentedDB(db *sql.DB) *InstrumentedDB {
    return &InstrumentedDB{
        db:     db,
        tracer: otel.Tracer("database"),
        meter:  otel.Meter("database"),
    }
}

func (idb *InstrumentedDB) QueryUser(ctx context.Context, userID string) (*User, error) {
    // Start span
    ctx, span := idb.tracer.Start(ctx, "db.query",
        trace.WithAttributes(
            attribute.String("db.operation", "select"),
            attribute.String("db.table", "users"),
        ),
    )
    defer span.End()
    
    // Record metrics
    startTime := time.Now()
    queryCounter, _ := idb.meter.Int64Counter("db.queries.total")
    queryDuration, _ := idb.meter.Float64Histogram("db.query.duration")
    
    // Execute query
    var user User
    err := idb.db.QueryRowContext(ctx, 
        "SELECT id, name, email FROM users WHERE id = $1", 
        userID,
    ).Scan(&user.ID, &user.Name, &user.Email)
    
    // Record duration
    duration := time.Since(startTime).Seconds()
    labels := []attribute.KeyValue{
        attribute.String("operation", "select"),
        attribute.String("table", "users"),
        attribute.Bool("success", err == nil),
    }
    
    queryCounter.Add(ctx, 1, metric.WithAttributes(labels...))
    queryDuration.Record(ctx, duration, metric.WithAttributes(labels...))
    
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, "query failed")
        return nil, err
    }
    
    span.SetAttributes(attribute.Bool("user.found", true))
    return &user, nil
}

// Batch operation example
func (idb *InstrumentedDB) BatchInsertUsers(ctx context.Context, users []User) error {
    ctx, span := idb.tracer.Start(ctx, "db.batch_insert",
        trace.WithAttributes(
            attribute.Int("batch.size", len(users)),
            attribute.String("db.operation", "insert"),
            attribute.String("db.table", "users"),
        ),
    )
    defer span.End()
    
    tx, err := idb.db.BeginTx(ctx, nil)
    if err != nil {
        span.RecordError(err)
        return err
    }
    defer tx.Rollback()
    
    stmt, err := tx.PrepareContext(ctx, 
        "INSERT INTO users (id, name, email) VALUES ($1, $2, $3)",
    )
    if err != nil {
        span.RecordError(err)
        return err
    }
    defer stmt.Close()
    
    for i, user := range users {
        // Create child span for each insert
        _, insertSpan := idb.tracer.Start(ctx, "db.insert_single",
            trace.WithAttributes(
                attribute.Int("batch.index", i),
            ),
        )
        
        _, err := stmt.ExecContext(ctx, user.ID, user.Name, user.Email)
        if err != nil {
            insertSpan.RecordError(err)
            insertSpan.End()
            span.RecordError(err)
            return err
        }
        
        insertSpan.End()
    }
    
    return tx.Commit()
}
```

### Cache Instrumentation

```go
package cache

import (
    "context"
    "time"
    
    "github.com/go-redis/redis/v8"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

type InstrumentedCache struct {
    client    *redis.Client
    tracer    trace.Tracer
    hitRate   metric.Float64Counter
    missRate  metric.Float64Counter
    opDuration metric.Float64Histogram
}

func NewInstrumentedCache(client *redis.Client) *InstrumentedCache {
    meter := otel.Meter("cache")
    
    hitRate, _ := meter.Float64Counter("cache.hits")
    missRate, _ := meter.Float64Counter("cache.misses") 
    opDuration, _ := meter.Float64Histogram("cache.operation.duration")
    
    return &InstrumentedCache{
        client:     client,
        tracer:     otel.Tracer("cache"),
        hitRate:    hitRate,
        missRate:   missRate,
        opDuration: opDuration,
    }
}

func (c *InstrumentedCache) Get(ctx context.Context, key string) (string, error) {
    ctx, span := c.tracer.Start(ctx, "cache.get",
        trace.WithAttributes(
            attribute.String("cache.key", key),
        ),
    )
    defer span.End()
    
    start := time.Now()
    
    value, err := c.client.Get(ctx, key).Result()
    
    duration := time.Since(start).Seconds()
    c.opDuration.Record(ctx, duration,
        metric.WithAttributes(
            attribute.String("operation", "get"),
            attribute.Bool("hit", err != redis.Nil),
        ),
    )
    
    if err == redis.Nil {
        c.missRate.Add(ctx, 1)
        span.SetAttributes(attribute.Bool("cache.hit", false))
        return "", nil
    } else if err != nil {
        span.RecordError(err)
        return "", err
    }
    
    c.hitRate.Add(ctx, 1)
    span.SetAttributes(
        attribute.Bool("cache.hit", true),
        attribute.Int("cache.value_size", len(value)),
    )
    
    return value, nil
}

func (c *InstrumentedCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
    ctx, span := c.tracer.Start(ctx, "cache.set",
        trace.WithAttributes(
            attribute.String("cache.key", key),
            attribute.Float64("cache.ttl_seconds", ttl.Seconds()),
            attribute.Int("cache.value_size", len(value)),
        ),
    )
    defer span.End()
    
    start := time.Now()
    
    err := c.client.Set(ctx, key, value, ttl).Err()
    
    duration := time.Since(start).Seconds()
    c.opDuration.Record(ctx, duration,
        metric.WithAttributes(
            attribute.String("operation", "set"),
            attribute.Bool("success", err == nil),
        ),
    )
    
    if err != nil {
        span.RecordError(err)
        return err
    }
    
    return nil
}
```

## Custom Metric Examples

### Business Metrics

```go
package business

import (
    "context"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

type OrderService struct {
    meter          metric.Meter
    orderCounter   metric.Int64Counter
    orderValue     metric.Float64Histogram
    processingTime metric.Float64Histogram
}

func NewOrderService() *OrderService {
    meter := otel.Meter("business.orders")
    
    orderCounter, _ := meter.Int64Counter("orders.created",
        metric.WithDescription("Total number of orders created"),
        metric.WithUnit("1"),
    )
    
    orderValue, _ := meter.Float64Histogram("orders.value",
        metric.WithDescription("Order value distribution"),
        metric.WithUnit("USD"),
    )
    
    processingTime, _ := meter.Float64Histogram("orders.processing_time",
        metric.WithDescription("Time to process an order"),
        metric.WithUnit("s"),
    )
    
    return &OrderService{
        meter:          meter,
        orderCounter:   orderCounter,
        orderValue:     orderValue,
        processingTime: processingTime,
    }
}

func (s *OrderService) CreateOrder(ctx context.Context, order Order) error {
    start := time.Now()
    
    // Process order
    err := s.processOrder(ctx, order)
    
    // Record metrics
    labels := []attribute.KeyValue{
        attribute.String("order.type", order.Type),
        attribute.String("customer.tier", order.CustomerTier),
        attribute.String("payment.method", order.PaymentMethod),
        attribute.Bool("success", err == nil),
    }
    
    s.orderCounter.Add(ctx, 1, metric.WithAttributes(labels...))
    s.orderValue.Record(ctx, order.TotalValue, metric.WithAttributes(labels...))
    s.processingTime.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(labels...))
    
    if err != nil {
        // Record specific error metrics
        errorCounter, _ := s.meter.Int64Counter("orders.errors")
        errorCounter.Add(ctx, 1, metric.WithAttributes(
            attribute.String("error.type", categorizeError(err)),
        ))
    }
    
    return err
}

// Gauge example for current state
func (s *OrderService) RecordActiveOrders(ctx context.Context) {
    activeGauge, _ := s.meter.Int64ObservableGauge("orders.active",
        metric.WithDescription("Number of currently active orders"),
    )
    
    _, err := s.meter.RegisterCallback(
        func(_ context.Context, o metric.Observer) error {
            // Get current active orders count
            activeCount := s.getActiveOrderCount()
            
            o.ObserveInt64(activeGauge, activeCount,
                metric.WithAttributes(
                    attribute.String("status", "processing"),
                ),
            )
            
            return nil
        },
        activeGauge,
    )
    
    if err != nil {
        panic(err)
    }
}
```

### Performance Metrics

```go
package performance

import (
    "context"
    "runtime"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/metric"
)

func SetupRuntimeMetrics() {
    meter := otel.Meter("runtime")
    
    // Memory metrics
    memStats := &runtime.MemStats{}
    
    allocGauge, _ := meter.Float64ObservableGauge("runtime.memory.alloc",
        metric.WithDescription("Bytes of allocated heap objects"),
        metric.WithUnit("By"),
    )
    
    gcCount, _ := meter.Int64ObservableCounter("runtime.gc.count",
        metric.WithDescription("Number of completed GC cycles"),
    )
    
    goroutines, _ := meter.Int64ObservableGauge("runtime.goroutines",
        metric.WithDescription("Number of goroutines"),
    )
    
    _, err := meter.RegisterCallback(
        func(_ context.Context, o metric.Observer) error {
            runtime.ReadMemStats(memStats)
            
            o.ObserveFloat64(allocGauge, float64(memStats.Alloc))
            o.ObserveInt64(gcCount, int64(memStats.NumGC))
            o.ObserveInt64(goroutines, int64(runtime.NumGoroutine()))
            
            return nil
        },
        allocGauge, gcCount, goroutines,
    )
    
    if err != nil {
        panic(err)
    }
}

// Request rate limiter with metrics
type RateLimiter struct {
    meter    metric.Meter
    allowed  metric.Int64Counter
    rejected metric.Int64Counter
}

func NewRateLimiter() *RateLimiter {
    meter := otel.Meter("ratelimiter")
    
    allowed, _ := meter.Int64Counter("ratelimit.allowed")
    rejected, _ := meter.Int64Counter("ratelimit.rejected")
    
    return &RateLimiter{
        meter:    meter,
        allowed:  allowed,
        rejected: rejected,
    }
}

func (rl *RateLimiter) Allow(ctx context.Context, key string) bool {
    allowed := rl.checkLimit(key)
    
    labels := metric.WithAttributes(
        attribute.String("key", key),
    )
    
    if allowed {
        rl.allowed.Add(ctx, 1, labels)
    } else {
        rl.rejected.Add(ctx, 1, labels)
    }
    
    return allowed
}
```

## Distributed Tracing Patterns

### Cross-Service Tracing

```go
package services

import (
    "context"
    "encoding/json"
    "net/http"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/trace"
)

// Client side - injecting trace context
type APIClient struct {
    httpClient *http.Client
    tracer     trace.Tracer
    propagator propagation.TextMapPropagator
}

func NewAPIClient() *APIClient {
    return &APIClient{
        httpClient: &http.Client{},
        tracer:     otel.Tracer("api-client"),
        propagator: otel.GetTextMapPropagator(),
    }
}

func (c *APIClient) GetUser(ctx context.Context, userID string) (*User, error) {
    // Start client span
    ctx, span := c.tracer.Start(ctx, "api.get_user",
        trace.WithSpanKind(trace.SpanKindClient),
    )
    defer span.End()
    
    // Create request
    req, err := http.NewRequestWithContext(ctx, 
        "GET", 
        "https://api.example.com/users/"+userID,
        nil,
    )
    if err != nil {
        span.RecordError(err)
        return nil, err
    }
    
    // Inject trace context into headers
    c.propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))
    
    // Make request
    resp, err := c.httpClient.Do(req)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }
    defer resp.Body.Close()
    
    span.SetAttributes(
        attribute.Int("http.status_code", resp.StatusCode),
    )
    
    var user User
    if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
        span.RecordError(err)
        return nil, err
    }
    
    return &user, nil
}

// Server side - extracting trace context
func TracingMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        propagator := otel.GetTextMapPropagator()
        tracer := otel.Tracer("api-server")
        
        // Extract trace context from headers
        ctx := propagator.Extract(r.Context(), 
            propagation.HeaderCarrier(r.Header),
        )
        
        // Start server span
        ctx, span := tracer.Start(ctx, r.Method+" "+r.URL.Path,
            trace.WithSpanKind(trace.SpanKindServer),
        )
        defer span.End()
        
        // Add request attributes
        span.SetAttributes(
            attribute.String("http.method", r.Method),
            attribute.String("http.url", r.URL.String()),
            attribute.String("http.user_agent", r.UserAgent()),
            attribute.String("http.remote_addr", r.RemoteAddr),
        )
        
        // Wrap response writer to capture status code
        wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
        
        // Call next handler with traced context
        next.ServeHTTP(wrapped, r.WithContext(ctx))
        
        // Record response attributes
        span.SetAttributes(
            attribute.Int("http.status_code", wrapped.statusCode),
        )
    }
}
```

### Async Processing Tracing

```go
package async

import (
    "context"
    "encoding/json"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/trace"
)

// Message with trace context
type TracedMessage struct {
    Data         json.RawMessage        `json:"data"`
    TraceContext map[string]string      `json:"trace_context"`
}

// Producer - injecting trace context
func ProduceMessage(ctx context.Context, data interface{}) error {
    tracer := otel.Tracer("producer")
    
    ctx, span := tracer.Start(ctx, "message.produce")
    defer span.End()
    
    // Serialize data
    dataBytes, err := json.Marshal(data)
    if err != nil {
        span.RecordError(err)
        return err
    }
    
    // Extract trace context
    carrier := propagation.MapCarrier{}
    otel.GetTextMapPropagator().Inject(ctx, carrier)
    
    // Create traced message
    msg := TracedMessage{
        Data:         dataBytes,
        TraceContext: carrier,
    }
    
    // Send to queue
    return sendToQueue(msg)
}

// Consumer - extracting trace context
func ConsumeMessage(msg TracedMessage) error {
    tracer := otel.Tracer("consumer")
    propagator := otel.GetTextMapPropagator()
    
    // Extract parent context
    ctx := propagator.Extract(context.Background(), 
        propagation.MapCarrier(msg.TraceContext),
    )
    
    // Start consumer span linked to producer
    ctx, span := tracer.Start(ctx, "message.consume",
        trace.WithSpanKind(trace.SpanKindConsumer),
    )
    defer span.End()
    
    // Process message
    return processMessage(ctx, msg.Data)
}
```

## Error Handling Patterns

### Structured Error Tracking

```go
package errors

import (
    "context"
    "errors"
    "fmt"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

// Custom error types with instrumentation
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}

type BusinessError struct {
    Code    string
    Message string
    Details map[string]interface{}
}

func (e BusinessError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Error handler with rich instrumentation
func HandleError(ctx context.Context, err error) {
    span := trace.SpanFromContext(ctx)
    
    // Record error on span
    span.RecordError(err)
    
    // Categorize and add attributes based on error type
    switch e := err.(type) {
    case ValidationError:
        span.SetStatus(codes.Error, "validation failed")
        span.SetAttributes(
            attribute.String("error.type", "validation"),
            attribute.String("error.field", e.Field),
        )
        
    case BusinessError:
        span.SetStatus(codes.Error, "business rule violation")
        span.SetAttributes(
            attribute.String("error.type", "business"),
            attribute.String("error.code", e.Code),
        )
        for k, v := range e.Details {
            span.SetAttributes(attribute.String("error.detail."+k, fmt.Sprint(v)))
        }
        
    default:
        span.SetStatus(codes.Error, "internal error")
        span.SetAttributes(
            attribute.String("error.type", "internal"),
        )
    }
}

// Retry with instrumentation
func RetryWithInstrumentation(ctx context.Context, operation string, fn func() error) error {
    tracer := otel.Tracer("retry")
    
    ctx, span := tracer.Start(ctx, "retry."+operation)
    defer span.End()
    
    maxRetries := 3
    var lastErr error
    
    for i := 0; i < maxRetries; i++ {
        // Create span for each attempt
        _, attemptSpan := tracer.Start(ctx, fmt.Sprintf("attempt.%d", i+1))
        
        err := fn()
        
        attemptSpan.SetAttributes(
            attribute.Int("retry.attempt", i+1),
            attribute.Bool("retry.success", err == nil),
        )
        
        if err == nil {
            attemptSpan.End()
            return nil
        }
        
        lastErr = err
        attemptSpan.RecordError(err)
        attemptSpan.End()
        
        // Exponential backoff
        time.Sleep(time.Duration(i*i) * time.Second)
    }
    
    span.SetStatus(codes.Error, "max retries exceeded")
    span.SetAttributes(
        attribute.Int("retry.max_attempts", maxRetries),
    )
    
    return lastErr
}

// Circuit breaker with metrics
type CircuitBreaker struct {
    tracer       trace.Tracer
    meter        metric.Meter
    stateGauge   metric.Int64ObservableGauge
    successCount metric.Int64Counter
    failureCount metric.Int64Counter
}

func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
    ctx, span := cb.tracer.Start(ctx, "circuit_breaker.call")
    defer span.End()
    
    state := cb.getState()
    span.SetAttributes(
        attribute.String("circuit_breaker.state", state),
    )
    
    if state == "open" {
        cb.failureCount.Add(ctx, 1, 
            metric.WithAttributes(attribute.String("reason", "circuit_open")),
        )
        return errors.New("circuit breaker is open")
    }
    
    err := fn()
    
    if err != nil {
        cb.failureCount.Add(ctx, 1,
            metric.WithAttributes(attribute.String("reason", "operation_failed")),
        )
        cb.recordFailure()
    } else {
        cb.successCount.Add(ctx, 1)
        cb.recordSuccess()
    }
    
    return err
}
```