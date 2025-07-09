# Application Troubleshooting Guide

## GoFiber Application Issues

### 1. Application Startup Problems

**Symptoms:**
- Application fails to start
- Port binding errors
- Configuration loading failures

**Diagnostic Commands:**
```bash
# Check application logs
kubectl logs -n myapp deployment/myapp --tail=50

# Check application health endpoint
kubectl exec -n myapp deployment/myapp -- curl -s localhost:8080/health

# Check port binding
kubectl exec -n myapp deployment/myapp -- netstat -tlnp | grep 8080

# Check configuration
kubectl get configmap -n myapp myapp-config -o yaml

# Check environment variables
kubectl exec -n myapp deployment/myapp -- env | grep APP_
```

**Solutions:**
- Fix configuration syntax errors
- Check port conflicts
- Verify environment variables
- Update resource limits
- Check file permissions

### 2. HTTP Server Issues

**Symptoms:**
- HTTP 500 errors
- Connection timeouts
- Slow response times

**Diagnostic Commands:**
```bash
# Test HTTP endpoints
kubectl exec -n myapp debug-pod -- curl -v http://myapp:8080/api/health

# Check fiber middleware stack
kubectl logs -n myapp deployment/myapp | grep -i middleware

# Monitor HTTP metrics
kubectl exec -n myapp deployment/myapp -- curl -s localhost:8080/metrics | grep http

# Check concurrent connections
kubectl exec -n myapp deployment/myapp -- ss -s
```

**Example GoFiber health check:**
```go
func healthCheck(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{
        "status": "healthy",
        "timestamp": time.Now().Unix(),
        "version": version,
        "uptime": time.Since(startTime).String(),
    })
}
```

**Solutions:**
- Add proper error handling
- Configure timeout settings
- Optimize middleware stack
- Check database connections
- Monitor resource usage

### 3. Database Connection Issues

**Symptoms:**
- Database connection errors
- Query timeouts
- Connection pool exhaustion

**Diagnostic Commands:**
```bash
# Check database connectivity
kubectl exec -n myapp deployment/myapp -- nc -zv postgres 5432

# Check connection pool status
kubectl logs -n myapp deployment/myapp | grep -i "connection pool"

# Test database queries
kubectl exec -n myapp deployment/myapp -- psql -h postgres -U myuser -d mydb -c "SELECT 1"

# Check database metrics
kubectl exec -n myapp deployment/myapp -- curl -s localhost:8080/metrics | grep db
```

**Example database connection configuration:**
```go
type Database struct {
    *gorm.DB
}

func InitDatabase() (*Database, error) {
    dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
        os.Getenv("DB_PORT"),
    )
    
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, err
    }
    
    sqlDB, err := db.DB()
    if err != nil {
        return nil, err
    }
    
    sqlDB.SetMaxOpenConns(25)
    sqlDB.SetMaxIdleConns(25)
    sqlDB.SetConnMaxLifetime(5 * time.Minute)
    
    return &Database{db}, nil
}
```

**Solutions:**
- Configure connection pool settings
- Add connection retry logic
- Check database server status
- Monitor connection metrics
- Implement circuit breaker pattern

## Instrumentation Problems

### 1. Prometheus Metrics Issues

**Symptoms:**
- Missing metrics in Prometheus
- Incorrect metric values
- High cardinality metrics

**Diagnostic Commands:**
```bash
# Check metrics endpoint
kubectl exec -n myapp deployment/myapp -- curl -s localhost:8080/metrics

# Test specific metrics
kubectl exec -n myapp deployment/myapp -- curl -s localhost:8080/metrics | grep myapp_

# Check metric exposition format
kubectl exec -n myapp deployment/myapp -- curl -s localhost:8080/metrics | head -20

# Monitor metrics scraping
kubectl logs -n monitoring prometheus-0 | grep myapp
```

**Example Prometheus metrics setup:**
```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    httpRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "myapp_http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status_code"},
    )
    
    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "myapp_http_request_duration_seconds",
            Help: "Duration of HTTP requests",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )
    
    activeConnections = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "myapp_active_connections",
            Help: "Number of active connections",
        },
    )
)

func prometheusMiddleware(c *fiber.Ctx) error {
    start := time.Now()
    
    err := c.Next()
    
    duration := time.Since(start).Seconds()
    status := c.Response().StatusCode()
    
    httpRequestsTotal.WithLabelValues(
        c.Method(),
        c.Path(),
        strconv.Itoa(status),
    ).Inc()
    
    httpRequestDuration.WithLabelValues(
        c.Method(),
        c.Path(),
    ).Observe(duration)
    
    return err
}
```

**Solutions:**
- Fix metric naming conventions
- Reduce metric cardinality
- Check scrape configuration
- Verify metric exposition format
- Add proper labels

### 2. Distributed Tracing Issues

**Symptoms:**
- Missing traces in Jaeger
- Incomplete span information
- Trace correlation problems

**Diagnostic Commands:**
```bash
# Check Jaeger agent connectivity
kubectl exec -n myapp deployment/myapp -- nc -zv jaeger-agent 6831

# Check trace export
kubectl logs -n myapp deployment/myapp | grep -i trace

# Test tracing endpoint
kubectl exec -n myapp deployment/myapp -- curl -s localhost:8080/trace-test

# Check Jaeger UI
kubectl port-forward -n monitoring svc/jaeger-query 16686:16686
```

**Example OpenTelemetry tracing setup:**
```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func initTracing() error {
    exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(
        jaeger.WithEndpoint("http://jaeger-collector:14268/api/traces"),
    ))
    if err != nil {
        return err
    }
    
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("myapp"),
            semconv.ServiceVersionKey.String("v1.0.0"),
        )),
    )
    
    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))
    
    return nil
}

func tracingMiddleware(c *fiber.Ctx) error {
    tracer := otel.Tracer("myapp")
    
    ctx, span := tracer.Start(c.Context(), c.Path())
    defer span.End()
    
    c.SetUserContext(ctx)
    
    span.SetAttributes(
        semconv.HTTPMethodKey.String(c.Method()),
        semconv.HTTPURLKey.String(c.OriginalURL()),
    )
    
    err := c.Next()
    
    span.SetAttributes(
        semconv.HTTPStatusCodeKey.Int(c.Response().StatusCode()),
    )
    
    return err
}
```

**Solutions:**
- Configure proper trace sampling
- Fix trace propagation headers
- Check exporter configuration
- Verify service mesh tracing
- Add custom span attributes

### 3. Logging Issues

**Symptoms:**
- Logs not appearing in Loki
- Incorrect log format
- Missing log context

**Diagnostic Commands:**
```bash
# Check application logs
kubectl logs -n myapp deployment/myapp --tail=50

# Check log format
kubectl logs -n myapp deployment/myapp | head -5

# Test structured logging
kubectl exec -n myapp deployment/myapp -- curl -s localhost:8080/log-test

# Check Loki ingestion
kubectl logs -n monitoring loki-0 | grep myapp
```

**Example structured logging setup:**
```go
import (
    "github.com/sirupsen/logrus"
    "github.com/gofiber/fiber/v2/middleware/logger"
)

func initLogging() {
    logrus.SetFormatter(&logrus.JSONFormatter{
        TimestampFormat: time.RFC3339,
    })
    
    logrus.SetLevel(logrus.InfoLevel)
    
    if os.Getenv("LOG_LEVEL") == "debug" {
        logrus.SetLevel(logrus.DebugLevel)
    }
}

func loggingMiddleware(c *fiber.Ctx) error {
    start := time.Now()
    
    err := c.Next()
    
    logrus.WithFields(logrus.Fields{
        "method":      c.Method(),
        "path":        c.Path(),
        "status_code": c.Response().StatusCode(),
        "duration":    time.Since(start).Milliseconds(),
        "user_agent":  c.Get("User-Agent"),
        "ip":          c.IP(),
    }).Info("HTTP request")
    
    return err
}
```

**Solutions:**
- Use structured logging format
- Configure proper log levels
- Add request correlation IDs
- Check log shipping configuration
- Implement log rotation

## Performance Bottlenecks

### 1. High CPU Usage

**Symptoms:**
- CPU throttling
- Slow response times
- High CPU utilization

**Diagnostic Commands:**
```bash
# Check CPU usage
kubectl top pods -n myapp

# Check CPU limits
kubectl describe pod -n myapp myapp-xxx | grep -A 5 -B 5 cpu

# Profile CPU usage
kubectl exec -n myapp deployment/myapp -- curl -s localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof

# Check goroutine count
kubectl exec -n myapp deployment/myapp -- curl -s localhost:8080/debug/pprof/goroutine?debug=1
```

**Example CPU profiling endpoint:**
```go
import (
    _ "net/http/pprof"
    "net/http"
)

func setupProfiling() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
}
```

**Solutions:**
- Optimize expensive operations
- Use goroutine pools
- Implement caching
- Reduce algorithm complexity
- Profile and optimize hot paths

### 2. Memory Leaks

**Symptoms:**
- Increasing memory usage
- OutOfMemory errors
- Pod restarts

**Diagnostic Commands:**
```bash
# Check memory usage
kubectl top pods -n myapp

# Check memory limits
kubectl describe pod -n myapp myapp-xxx | grep -A 5 -B 5 memory

# Memory profiling
kubectl exec -n myapp deployment/myapp -- curl -s localhost:8080/debug/pprof/heap > heap.prof

# Check garbage collection
kubectl exec -n myapp deployment/myapp -- curl -s localhost:8080/debug/pprof/allocs
```

**Example memory optimization:**
```go
import (
    "runtime"
    "time"
)

func memoryMonitor() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            var m runtime.MemStats
            runtime.ReadMemStats(&m)
            
            logrus.WithFields(logrus.Fields{
                "alloc":      m.Alloc,
                "total_alloc": m.TotalAlloc,
                "sys":        m.Sys,
                "num_gc":     m.NumGC,
            }).Info("Memory stats")
            
            // Force GC if memory usage is high
            if m.Alloc > 100*1024*1024 { // 100MB
                runtime.GC()
            }
        }
    }
}
```

**Solutions:**
- Fix memory leaks
- Implement object pooling
- Optimize data structures
- Use memory profiling tools
- Configure garbage collection

### 3. Database Performance Issues

**Symptoms:**
- Slow database queries
- Connection pool exhaustion
- Query timeouts

**Diagnostic Commands:**
```bash
# Check database connection pool
kubectl exec -n myapp deployment/myapp -- curl -s localhost:8080/debug/db/stats

# Check slow queries
kubectl exec -n postgres postgres-0 -- psql -c "SELECT query, mean_time, calls FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"

# Check database connections
kubectl exec -n postgres postgres-0 -- psql -c "SELECT count(*) FROM pg_stat_activity;"
```

**Example database optimization:**
```go
type DatabaseStats struct {
    OpenConnections int `json:"open_connections"`
    InUse          int `json:"in_use"`
    Idle           int `json:"idle"`
}

func (db *Database) GetStats() DatabaseStats {
    sqlDB, _ := db.DB.DB()
    stats := sqlDB.Stats()
    
    return DatabaseStats{
        OpenConnections: stats.OpenConnections,
        InUse:          stats.InUse,
        Idle:           stats.Idle,
    }
}

func optimizedQuery(db *Database) error {
    // Use connection context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // Use prepared statements
    stmt, err := db.PrepareContext(ctx, "SELECT * FROM users WHERE id = ?")
    if err != nil {
        return err
    }
    defer stmt.Close()
    
    // Execute with context
    rows, err := stmt.QueryContext(ctx, userID)
    if err != nil {
        return err
    }
    defer rows.Close()
    
    return nil
}
```

**Solutions:**
- Optimize database queries
- Use connection pooling
- Implement query caching
- Add database indexes
- Use prepared statements

## Application Monitoring

### Key Metrics to Monitor

```go
// Custom application metrics
var (
    requestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "myapp_requests_total",
            Help: "Total number of requests",
        },
        []string{"method", "endpoint", "status"},
    )
    
    requestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "myapp_request_duration_seconds",
            Help: "Request duration in seconds",
            Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
        },
        []string{"method", "endpoint"},
    )
    
    activeConnections = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "myapp_active_connections",
            Help: "Number of active connections",
        },
    )
    
    dbConnections = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "myapp_db_connections",
            Help: "Database connection pool stats",
        },
        []string{"state"}, // open, in_use, idle
    )
)
```

### Health Check Implementation

```go
type HealthStatus struct {
    Status    string            `json:"status"`
    Timestamp int64             `json:"timestamp"`
    Checks    map[string]string `json:"checks"`
}

func healthHandler(c *fiber.Ctx) error {
    status := HealthStatus{
        Status:    "healthy",
        Timestamp: time.Now().Unix(),
        Checks:    make(map[string]string),
    }
    
    // Check database connection
    if err := db.DB.Ping(); err != nil {
        status.Status = "unhealthy"
        status.Checks["database"] = "failed"
    } else {
        status.Checks["database"] = "ok"
    }
    
    // Check Redis connection
    if _, err := redis.Ping().Result(); err != nil {
        status.Status = "unhealthy"
        status.Checks["redis"] = "failed"
    } else {
        status.Checks["redis"] = "ok"
    }
    
    if status.Status == "unhealthy" {
        return c.Status(503).JSON(status)
    }
    
    return c.JSON(status)
}
```

### Alerting Rules

```yaml
groups:
  - name: myapp-alerts
    rules:
    - alert: ApplicationDown
      expr: up{job="myapp"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Application is down"
        description: "Application has been down for more than 5 minutes"

    - alert: HighErrorRate
      expr: rate(myapp_requests_total{status!~"2.."}[5m]) > 0.1
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High error rate"
        description: "Error rate is above 10% for 5 minutes"

    - alert: HighResponseTime
      expr: histogram_quantile(0.95, rate(myapp_request_duration_seconds_bucket[5m])) > 2
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High response time"
        description: "95th percentile response time is above 2 seconds"

    - alert: HighMemoryUsage
      expr: process_resident_memory_bytes{job="myapp"} > 1e9
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "High memory usage"
        description: "Application is using more than 1GB of memory"
```

## Common Error Patterns and Solutions

| Error Pattern | Cause | Solution |
|---------------|-------|----------|
| "connection refused" | Service not running | Check pod status and restart if needed |
| "context deadline exceeded" | Timeout | Increase timeout settings |
| "too many open files" | File descriptor limit | Increase ulimit |
| "connection pool exhausted" | High database load | Optimize queries and increase pool size |
| "out of memory" | Memory leak | Fix memory leaks and increase limits |
| "panic: runtime error" | Application bug | Fix code and add error handling |
| "database lock timeout" | Long-running transactions | Optimize transactions and add retry logic |