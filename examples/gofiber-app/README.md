# GoFiber APM Example Application

This is a fully instrumented example GoFiber application demonstrating best practices for Application Performance Monitoring (APM) including distributed tracing, metrics collection, and structured logging.

## Features

- **OpenTelemetry Tracing**: Distributed tracing with Jaeger
- **Prometheus Metrics**: Custom business metrics and HTTP metrics
- **Structured Logging**: JSON logging with trace correlation
- **Health Checks**: Database and cache health monitoring
- **Circuit Breaker**: Resilient external API calls
- **Error Handling**: Centralized error handling with proper instrumentation
- **Service Architecture**: Clean separation of handlers, services, and data layers

## Architecture

The application consists of several layers:

1. **Handlers** (`handlers.go`): HTTP request handlers with proper instrumentation
2. **Services** (`services.go`): Business logic with distributed tracing
3. **Middleware**: Custom middleware for logging, metrics, and tracing
4. **APM Stack**: Complete observability stack with Prometheus, Grafana, Jaeger, and Loki

## Endpoints

### Core API Endpoints

- `GET /health` - Health check endpoint
- `GET /api/v1/users` - List users
- `GET /api/v1/users/:id` - Get specific user
- `POST /api/v1/users` - Create new user
- `PUT /api/v1/users/:id` - Update user
- `DELETE /api/v1/users/:id` - Delete user
- `GET /api/v1/products` - List products
- `GET /api/v1/products/:id` - Get specific product
- `POST /api/v1/products` - Create new product
- `POST /api/v1/orders` - Create new order
- `GET /api/v1/orders/:id` - Get specific order
- `GET /api/v1/analytics/dashboard` - Analytics dashboard data

### Test Endpoints

- `GET /api/v1/test/slow` - Simulates slow operations
- `GET /api/v1/test/error` - Simulates various error conditions
- `GET /api/v1/test/panic` - Tests panic recovery

### Observability Endpoints

- `GET :9091/metrics` - Prometheus metrics endpoint

## Running the Example

### Using Docker Compose

1. Start all services:
```bash
docker-compose up -d
```

2. Access the services:
- Application: http://localhost:8080
- Grafana: http://localhost:3000 (admin/admin)
- Prometheus: http://localhost:9090
- Jaeger UI: http://localhost:16686
- Alertmanager: http://localhost:9093

### Local Development

1. Install dependencies:
```bash
go mod download
```

2. Set environment variables:
```bash
export APP_NAME=gofiber-example-app
export APP_PORT=8080
export METRICS_PORT=9091
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
export TRACING_ENABLED=true
export DB_ENABLED=true
export CACHE_ENABLED=true
```

3. Run the application:
```bash
go run .
```

## Instrumentation Examples

### 1. Distributed Tracing

```go
func (s *UserService) GetUser(ctx context.Context, userID string) (*User, error) {
    ctx, span := tracer.Start(ctx, "service.get-user")
    defer span.End()
    
    span.SetAttributes(attribute.String("user_id", userID))
    
    // Implementation...
}
```

### 2. Custom Metrics

```go
// Define metrics
var businessMetrics = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "business_operations_total",
        Help: "Total number of business operations",
    },
    []string{"operation", "status"},
)

// Record metrics
businessMetrics.WithLabelValues("create_user", "success").Inc()
```

### 3. Structured Logging with Trace Correlation

```go
logger.Info("User created successfully",
    zap.String("user_id", user.ID),
    zap.String("trace_id", span.SpanContext().TraceID().String()),
)
```

### 4. Circuit Breaker Pattern

```go
result, err := s.breaker.Execute(func() (interface{}, error) {
    // Call external API
    return callExternalAPI()
})
```

## Testing the Application

### Generate Load

```bash
# Test normal operations
for i in {1..100}; do
    curl http://localhost:8080/api/v1/users
    curl http://localhost:8080/api/v1/products
done

# Test error scenarios
curl http://localhost:8080/api/v1/test/error
curl http://localhost:8080/api/v1/test/slow
```

### View Metrics in Grafana

1. Open Grafana at http://localhost:3000
2. Navigate to Dashboards → GoFiber Example
3. View real-time metrics and traces

### Explore Traces in Jaeger

1. Open Jaeger UI at http://localhost:16686
2. Select "gofiber-example-app" service
3. Explore distributed traces

### Query Logs in Grafana

1. Open Grafana at http://localhost:3000
2. Go to Explore → Select Loki datasource
3. Query logs with: `{job="gofiber-app"}`
4. Correlate logs with traces using trace_id

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| APP_NAME | Application name | gofiber-example-app |
| APP_PORT | Main application port | 8080 |
| METRICS_PORT | Prometheus metrics port | 9091 |
| LOG_LEVEL | Logging level | info |
| TRACING_ENABLED | Enable OpenTelemetry tracing | true |
| OTEL_EXPORTER_OTLP_ENDPOINT | OTLP endpoint for traces | jaeger:4317 |
| DB_ENABLED | Enable database simulation | true |
| CACHE_ENABLED | Enable cache simulation | true |

## Best Practices Demonstrated

1. **Trace Context Propagation**: Traces flow through all layers of the application
2. **Error Handling**: Proper error recording in traces and metrics
3. **Resource Management**: Graceful shutdown and cleanup
4. **Security**: Non-root container user, minimal base image
5. **Performance**: Efficient metrics collection and batched trace export
6. **Resilience**: Circuit breaker for external dependencies
7. **Observability**: Complete visibility into application behavior

## Troubleshooting

### No traces appearing in Jaeger
- Check OTEL_EXPORTER_OTLP_ENDPOINT is correct
- Verify Jaeger is running: `docker-compose ps jaeger`
- Check application logs for OTLP errors

### Metrics not showing in Prometheus
- Verify metrics endpoint: `curl http://localhost:9091/metrics`
- Check Prometheus targets: http://localhost:9090/targets
- Ensure prometheus.yml is correctly mounted

### Logs not appearing in Loki
- Check Promtail is running: `docker-compose ps promtail`
- Verify log format matches pipeline stages
- Check Loki datasource in Grafana

## License

This example is part of the APM project and follows the same license.