# Instrumentation Best Practices

This document outlines best practices for implementing effective instrumentation in production environments.

## Naming Conventions

### Metrics

#### Format
Use dot-separated names with clear hierarchical structure:
```
<component>.<operation>.<measurement>
```

#### Examples
```go
// Good metric names
"http.request.duration"
"http.request.size"
"db.query.duration"
"cache.hit.ratio"
"queue.message.processed"

// Bad metric names
"RequestDuration"      // Not hierarchical
"http-request-count"   // Use dots, not dashes
"users"               // Too generic
```

#### Guidelines
- Use lowercase letters
- Separate words with dots (.)
- Be descriptive but concise
- Include units in the name when not obvious
- Group related metrics with common prefixes

### Spans and Traces

#### Format
```
<component>.<operation>
```

#### Examples
```go
// Good span names
"http.server.request"
"db.query"
"cache.get"
"external.api.call"
"message.process"

// Bad span names
"GET /api/v1/users/:id"  // Too specific, includes parameters
"database"               // Too generic
"ProcessUserData"        // Use dot notation
```

### Attributes/Labels

#### Common Attributes
```go
// Standard HTTP attributes
"http.method"        // GET, POST, etc.
"http.route"         // /api/v1/users/:id
"http.status_code"   // 200, 404, 500
"http.url"           // Full URL (be careful with PII)

// Standard database attributes  
"db.system"          // postgresql, mysql, redis
"db.operation"       // select, insert, update
"db.table"           // users, orders

// Custom attributes
"user.type"          // free, premium, enterprise
"feature.flag"       // feature flag name
"error.type"         // validation, timeout, permission
```

## Sampling Strategies for Production

### Trace Sampling

#### Head-Based Sampling
Best for most applications:

```go
// Development/Staging - Higher sampling
instrumentation.WithSamplingRate(0.1), // 10%

// Production - Lower sampling
instrumentation.WithSamplingRate(0.001), // 0.1%

// High-traffic services
instrumentation.WithSamplingRate(0.0001), // 0.01%
```

#### Adaptive Sampling
Adjust based on traffic:

```go
func adaptiveSamplingRate() float64 {
    rps := getCurrentRequestsPerSecond()
    switch {
    case rps < 100:
        return 1.0    // 100% for low traffic
    case rps < 1000:
        return 0.1    // 10% for medium traffic
    case rps < 10000:
        return 0.01   // 1% for high traffic
    default:
        return 0.001  // 0.1% for very high traffic
    }
}
```

#### Priority Sampling
Sample important requests:

```go
func shouldSample(ctx context.Context) bool {
    // Always sample errors
    if hasError(ctx) {
        return true
    }
    
    // Always sample slow requests
    if duration > 5*time.Second {
        return true
    }
    
    // Sample premium users more
    if isPremiumUser(ctx) {
        return rand.Float64() < 0.1 // 10%
    }
    
    // Default sampling
    return rand.Float64() < 0.001 // 0.1%
}
```

### Metric Sampling

#### Histogram Buckets
Configure appropriate buckets:

```go
// HTTP request duration buckets (in seconds)
httpDurationBuckets := []float64{
    0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

// Database query duration buckets (in seconds)
dbDurationBuckets := []float64{
    0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5,
}
```

## Performance Considerations

### 1. Minimize Overhead

#### Use Async Export
```go
instrumentation.WithAsyncExport(true),
instrumentation.WithBatchSize(100),
instrumentation.WithQueueSize(10000),
```

#### Pre-compute Attributes
```go
// Bad - computed every time
span.SetAttributes(
    attribute.String("hostname", os.Hostname()),
    attribute.String("version", getVersion()),
)

// Good - computed once
var (
    hostname = mustGetHostname()
    version  = getVersion()
)
span.SetAttributes(
    attribute.String("hostname", hostname),
    attribute.String("version", version),
)
```

### 2. Control Cardinality

#### Avoid Unbounded Labels
```go
// Bad - unlimited cardinality
meter.RecordDuration("api.request.duration",
    attribute.String("user_id", userID),     // Millions of values
    attribute.String("session_id", sessionID), // Unlimited values
)

// Good - bounded cardinality
meter.RecordDuration("api.request.duration",
    attribute.String("user_type", getUserType(userID)), // few values
    attribute.Bool("authenticated", isAuthenticated),
)
```

#### Use Buckets for Numeric Values
```go
// Bad
attribute.Int("response_size", size)

// Good
attribute.String("response_size_bucket", getSizeBucket(size))

func getSizeBucket(size int) string {
    switch {
    case size < 1024:
        return "small"
    case size < 1024*1024:
        return "medium"
    default:
        return "large"
    }
}
```

### 3. Efficient Context Propagation

```go
// Reuse contexts when possible
ctx := c.UserContext()
ctx = instr.InjectContext(ctx)

// Pass context through your application
result, err := processRequest(ctx, data)
```

### 4. Batch Operations

```go
// Bad - individual metrics
for _, item := range items {
    meter.RecordOne("items.processed")
}

// Good - batch recording
meter.RecordBatch("items.processed", len(items))
```

## Security Best Practices

### 1. Sanitize Sensitive Data

#### Never Log PII
```go
// Bad
span.SetAttributes(
    attribute.String("user.email", email),
    attribute.String("user.ssn", ssn),
    attribute.String("credit_card", ccNumber),
)

// Good
span.SetAttributes(
    attribute.String("user.id", userID),
    attribute.String("user.type", userType),
    attribute.Bool("payment.valid", isValidPayment),
)
```

#### Sanitize URLs
```go
func sanitizeURL(rawURL string) string {
    u, err := url.Parse(rawURL)
    if err != nil {
        return "invalid-url"
    }
    
    // Remove sensitive query parameters
    q := u.Query()
    for _, param := range []string{"token", "key", "secret", "password"} {
        q.Del(param)
    }
    u.RawQuery = q.Encode()
    
    return u.String()
}
```

### 2. Secure Communication

#### Use TLS for OTLP
```go
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS12,
    ServerName: "otel-collector.example.com",
}

instrumentation.WithTLS(tlsConfig),
instrumentation.WithEndpoint("https://otel-collector.example.com:4317"),
```

#### Authentication
```go
// API Key authentication
instrumentation.WithHeaders(map[string]string{
    "X-API-Key": os.Getenv("OTEL_API_KEY"),
}),

// mTLS
instrumentation.WithClientCertificate(certFile, keyFile),
```

### 3. Access Control

#### Separate Environments
```go
// Use different endpoints/credentials per environment
endpoint := map[string]string{
    "dev":     "dev-collector.internal:4317",
    "staging": "staging-collector.internal:4317", 
    "prod":    "prod-collector.internal:4317",
}[environment]
```

#### Limit Metric Access
```go
// Add team/service labels for access control
instrumentation.WithAttributes(map[string]string{
    "team": "backend",
    "service.owner": "platform-team",
    "cost.center": "engineering",
}),
```

### 4. Data Retention

#### Configure Appropriate Retention
- Metrics: 30-90 days for most use cases
- Traces: 7-30 days depending on volume
- Logs: 30-90 days for debugging

#### Implement Data Lifecycle
```go
// Add timestamp for automated cleanup
span.SetAttributes(
    attribute.Int64("retention.days", 30),
    attribute.String("data.classification", "internal"),
)
```

## Resource Management

### Memory Management
```go
// Limit queue sizes to prevent OOM
instrumentation.WithQueueSize(10000),
instrumentation.WithMaxExportBatchSize(512),

// Set timeouts to prevent resource leaks
instrumentation.WithExportTimeout(30 * time.Second),
instrumentation.WithShutdownTimeout(5 * time.Second),
```

### Connection Pooling
```go
// Reuse connections
instrumentation.WithMaxIdleConns(10),
instrumentation.WithMaxConnsPerHost(10),
instrumentation.WithKeepAlive(30 * time.Second),
```

### Graceful Shutdown
```go
// Ensure proper cleanup
defer func() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := instr.Shutdown(ctx); err != nil {
        log.Printf("Failed to shutdown instrumentation: %v", err)
    }
}()
```