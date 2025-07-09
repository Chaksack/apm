# Jaeger User Guide

## Overview

Jaeger is a distributed tracing system that helps monitor and troubleshoot complex microservices architectures. This guide covers trace exploration, search and filtering, performance analysis, and integration setup.

## Getting Started

### Accessing Jaeger

1. **Web Interface**: Navigate to `http://localhost:16686` (default)
2. **API**: Access via REST API at `http://localhost:16686/api/`
3. **GRPC**: Query service at `localhost:16685`

### Basic Navigation

- **Search**: Find traces by service, operation, tags, or time
- **Trace Timeline**: Visualize individual traces
- **System Architecture**: View service dependencies
- **Service Performance**: Analyze service metrics

## Understanding Traces

### Trace Structure

```
Trace
├── Span A (root)
│   ├── Span B (child)
│   │   └── Span D (child)
│   └── Span C (child)
└── Span E (follows from A)
```

### Key Concepts

#### Trace
- Represents a single request through the system
- Contains multiple spans
- Has a unique trace ID

#### Span
- Represents a unit of work
- Has start time, end time, and duration
- Contains operation name and tags
- Can have parent-child relationships

#### Tags
- Key-value pairs that describe the span
- Used for filtering and analysis
- Examples: `http.method=GET`, `error=true`

#### Logs
- Timestamped events within a span
- Provide additional context
- Can include error messages or debug info

## Trace Exploration

### Search Interface

#### Basic Search
```
Service: user-service
Operation: GET /api/users
Tags: http.status_code=200
Lookback: 1h
Min Duration: 100ms
Max Duration: 5s
```

#### Advanced Search
```
Service: payment-service
Tags: error=true AND customer.id=12345
Duration: >1s
Time Range: Custom (2024-01-01 to 2024-01-02)
```

### Trace Timeline View

#### Understanding the Timeline
- **Gantt Chart**: Shows span hierarchy and timing
- **Span Details**: Click spans for detailed information
- **Critical Path**: Identifies bottlenecks
- **Service Calls**: Shows inter-service communication

#### Navigation Tips
1. **Zoom**: Use mouse wheel or controls
2. **Pan**: Click and drag to move
3. **Collapse**: Hide child spans
4. **Expand**: Show all spans
5. **Search**: Find specific spans

### Span Details

#### Span Information
```json
{
  "traceID": "1234567890abcdef",
  "spanID": "abcdef1234567890",
  "operationName": "GET /api/users",
  "startTime": "2024-01-01T10:00:00Z",
  "duration": "150ms",
  "tags": {
    "http.method": "GET",
    "http.url": "/api/users",
    "http.status_code": 200,
    "component": "http"
  },
  "logs": [
    {
      "timestamp": "2024-01-01T10:00:00.100Z",
      "fields": {
        "event": "request_start",
        "user_id": "12345"
      }
    }
  ]
}
```

#### Process Information
- **Service Name**: Which service created the span
- **Instance**: Specific instance information
- **Version**: Application version
- **Environment**: Deployment environment

## Search and Filtering

### Service-based Search

#### Find All Traces for a Service
```
Service: order-service
Operation: All
Lookback: 2h
```

#### Find Specific Operations
```
Service: order-service
Operation: POST /api/orders
Lookback: 1h
```

### Tag-based Filtering

#### HTTP Status Codes
```
Tags: http.status_code=500
```

#### Error Traces
```
Tags: error=true
```

#### Custom Tags
```
Tags: user.id=12345 AND region=us-west
```

#### Complex Queries
```
Tags: (http.status_code=500 OR http.status_code=502) AND service.name=api-gateway
```

### Time-based Search

#### Absolute Time Range
```
Start: 2024-01-01 10:00:00
End: 2024-01-01 11:00:00
```

#### Relative Time Range
```
Lookback: 1h
```

#### Duration Filters
```
Min Duration: 100ms
Max Duration: 5s
```

### Advanced Search Techniques

#### Multiple Services
```
Service: user-service
Tags: downstream.service=payment-service
```

#### Correlation IDs
```
Tags: correlation.id=abc123
```

#### Business Context
```
Tags: customer.tier=premium AND region=us-east
```

## Performance Analysis

### Identifying Bottlenecks

#### Critical Path Analysis
1. **Longest Spans**: Identify slowest operations
2. **Sequential vs Parallel**: Analyze execution patterns
3. **Wait Time**: Find idle periods
4. **Resource Contention**: Identify blocking operations

#### Service Latency
```
Service: database-service
Operation: SELECT users
Duration: >500ms
```

#### Error Analysis
```
Tags: error=true
Service: payment-service
```

### Performance Metrics

#### Span Duration Analysis
- **P50, P90, P99**: Percentile analysis
- **Mean Duration**: Average response time
- **Max Duration**: Worst-case performance
- **Throughput**: Requests per second

#### Error Rate Analysis
- **Error Count**: Number of failed requests
- **Error Rate**: Percentage of failed requests
- **Error Types**: Classification of errors
- **Error Trends**: Error rate over time

### Comparison Analysis

#### Before/After Comparison
```
Time Range 1: 2024-01-01 (before deployment)
Time Range 2: 2024-01-02 (after deployment)
Service: user-service
```

#### A/B Testing
```
Tags: experiment.variant=A
Tags: experiment.variant=B
Service: recommendation-service
```

## System Architecture View

### Service Map
- **Nodes**: Services in your architecture
- **Edges**: Communication between services
- **Metrics**: Request rates and error rates
- **Dependencies**: Service dependencies

### Dependency Analysis

#### Service Dependencies
```
Root Service: api-gateway
Downstream Services: user-service, order-service, payment-service
```

#### Call Patterns
- **Synchronous**: Direct service calls
- **Asynchronous**: Message queue patterns
- **Fan-out**: One-to-many patterns
- **Fan-in**: Many-to-one patterns

### Architecture Insights

#### Service Utilization
- **Request Volume**: Calls per service
- **Error Rates**: Service reliability
- **Latency**: Service performance
- **Throughput**: Service capacity

#### Communication Patterns
- **Chatty Services**: High call frequency
- **Bottleneck Services**: High latency impact
- **Critical Services**: High failure impact
- **Leaf Services**: External dependencies

## Integration Setup

### Instrumentation

#### Go Application
```go
import (
    "github.com/opentracing/opentracing-go"
    "github.com/uber/jaeger-client-go"
    "github.com/uber/jaeger-client-go/config"
)

func initJaeger(serviceName string) (opentracing.Tracer, io.Closer) {
    cfg := config.Configuration{
        ServiceName: serviceName,
        Sampler: &config.SamplerConfig{
            Type:  "const",
            Param: 1,
        },
        Reporter: &config.ReporterConfig{
            LogSpans: true,
            LocalAgentHostPort: "localhost:6831",
        },
    }
    
    tracer, closer, err := cfg.NewTracer()
    if err != nil {
        log.Fatal(err)
    }
    
    return tracer, closer
}
```

#### Manual Instrumentation
```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    span := tracer.StartSpan("handle_request")
    defer span.Finish()
    
    span.SetTag("http.method", r.Method)
    span.SetTag("http.url", r.URL.String())
    
    // Add span to context
    ctx := opentracing.ContextWithSpan(r.Context(), span)
    
    // Call downstream service
    callDownstream(ctx)
    
    span.SetTag("http.status_code", 200)
}
```

### Sampling Configuration

#### Sampling Strategies
```json
{
  "default_strategy": {
    "type": "probabilistic",
    "param": 0.1
  },
  "per_service_strategies": [
    {
      "service": "high-volume-service",
      "type": "probabilistic",
      "param": 0.01
    },
    {
      "service": "critical-service",
      "type": "const",
      "param": 1
    }
  ]
}
```

#### Adaptive Sampling
```json
{
  "default_strategy": {
    "type": "adaptive",
    "max_traces_per_second": 100
  },
  "per_operation_strategies": [
    {
      "service": "user-service",
      "operation": "GET /health",
      "type": "probabilistic",
      "param": 0.001
    }
  ]
}
```

### Storage Configuration

#### Elasticsearch
```yaml
SPAN_STORAGE_TYPE: elasticsearch
ES_SERVER_URLS: http://localhost:9200
ES_INDEX_PREFIX: jaeger
```

#### Cassandra
```yaml
SPAN_STORAGE_TYPE: cassandra
CASSANDRA_SERVERS: localhost:9042
CASSANDRA_KEYSPACE: jaeger_v1_dc1
```

#### Kafka
```yaml
SPAN_STORAGE_TYPE: kafka
KAFKA_PRODUCER_BROKERS: localhost:9092
KAFKA_TOPIC: jaeger-spans
```

## Best Practices

### Instrumentation Best Practices

1. **Meaningful Operation Names**: Use descriptive names
2. **Consistent Tagging**: Standardize tag names
3. **Error Handling**: Always tag errors
4. **Sampling**: Use appropriate sampling rates
5. **Context Propagation**: Pass context correctly

### Performance Considerations

1. **Sampling Rate**: Balance detail vs. performance
2. **Batch Size**: Optimize span batching
3. **Buffer Size**: Configure appropriate buffers
4. **Flush Interval**: Set reasonable flush intervals
5. **Resource Limits**: Monitor resource usage

### Operational Best Practices

1. **Monitoring**: Monitor Jaeger itself
2. **Alerting**: Set up Jaeger alerts
3. **Retention**: Configure appropriate retention
4. **Backup**: Backup trace data
5. **Scaling**: Plan for growth

## Common Use Cases

### Debugging Production Issues

1. **Find Error Traces**:
   ```
   Tags: error=true
   Service: affected-service
   Time: Last 1 hour
   ```

2. **Analyze Failed Requests**:
   ```
   Tags: http.status_code=500
   Duration: >1s
   ```

3. **Trace User Journey**:
   ```
   Tags: user.id=12345
   Service: api-gateway
   ```

### Performance Optimization

1. **Identify Slow Operations**:
   ```
   Service: database-service
   Duration: >500ms
   Limit: 100
   ```

2. **Compare Service Versions**:
   ```
   Tags: version=v1.0
   Tags: version=v2.0
   Service: recommendation-service
   ```

3. **Analyze Dependencies**:
   ```
   Service: order-service
   Operation: POST /api/orders
   Duration: >2s
   ```

### Capacity Planning

1. **Analyze Traffic Patterns**:
   ```
   Service: api-gateway
   Time Range: Last 24 hours
   ```

2. **Identify Bottlenecks**:
   ```
   Service: payment-service
   Duration: >1s
   ```

3. **Service Utilization**:
   ```
   Service: user-service
   Tags: instance=*
   ```

## Advanced Features

### Trace Comparison
- Compare traces with similar patterns
- Identify performance regressions
- Analyze A/B test results

### Deep Linking
- Link from alerts to specific traces
- Share trace URLs with team members
- Embed traces in documentation

### Custom Dashboards
- Create service-specific views
- Build operational dashboards
- Integrate with monitoring tools

## Integration with Other Tools

### Prometheus Integration
```yaml
# Scrape Jaeger metrics
- job_name: 'jaeger'
  static_configs:
    - targets: ['localhost:14269']
```

### Grafana Integration
```json
{
  "datasource": {
    "type": "jaeger",
    "url": "http://localhost:16686",
    "name": "Jaeger"
  }
}
```

### Loki Integration
```yaml
# Correlate logs with traces
derivedFields:
  - name: "TraceID"
    matcherRegex: "trace_id=(\\w+)"
    url: "http://localhost:16686/trace/${__value.raw}"
```

## Troubleshooting

### Common Issues

1. **Missing Traces**: Check sampling configuration
2. **Incomplete Traces**: Verify context propagation
3. **High Latency**: Optimize sampling and batching
4. **Storage Issues**: Monitor storage backend
5. **Network Problems**: Check agent connectivity

### Debug Tools

1. **Jaeger Agent Logs**: Check agent status
2. **Collector Logs**: Verify data ingestion
3. **Query Logs**: Debug search issues
4. **Client Logs**: Check instrumentation

### Performance Monitoring

1. **Span Ingestion Rate**: Monitor spans/second
2. **Storage Usage**: Track storage growth
3. **Query Performance**: Monitor query latency
4. **Error Rates**: Track system errors

## Resources

- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [OpenTracing Specification](https://opentracing.io/specification/)
- [Instrumentation Libraries](https://www.jaegertracing.io/docs/1.35/client-libraries/)
- [Deployment Guide](https://www.jaegertracing.io/docs/1.35/deployment/)