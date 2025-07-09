# Observability Fundamentals Training

## Learning Objectives
By the end of this training, you will be able to:
- Understand the three pillars of observability
- Distinguish between metrics, logs, and traces
- Define and implement SLI/SLO concepts
- Apply observability best practices in production systems

## The Three Pillars of Observability

### 1. Metrics
**Definition**: Numerical measurements aggregated over time intervals

**Characteristics**:
- Low storage overhead
- Efficient for alerting
- Good for dashboards and trends
- Limited context for debugging

**Examples**:
```yaml
# Application Metrics
http_requests_total{method="GET", status="200"} 1500
response_time_seconds{endpoint="/api/users"} 0.125
error_rate_percent{service="user-service"} 0.02

# Infrastructure Metrics
cpu_usage_percent{host="web-01"} 65.5
memory_usage_bytes{container="app"} 1073741824
disk_io_operations_per_second{device="/dev/sda1"} 150
```

**When to Use**:
- Real-time monitoring
- Alerting on thresholds
- Capacity planning
- Performance trending

### 2. Logs
**Definition**: Discrete events with timestamp and contextual information

**Characteristics**:
- High storage overhead
- Rich contextual information
- Good for debugging
- Difficult to aggregate

**Examples**:
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "ERROR",
  "service": "user-service",
  "message": "Failed to authenticate user",
  "user_id": "12345",
  "request_id": "abc-123-def",
  "error_code": "AUTH_FAILED",
  "stack_trace": "..."
}
```

**Log Levels**:
- **FATAL**: System unusable
- **ERROR**: Error conditions
- **WARN**: Warning conditions
- **INFO**: Informational messages
- **DEBUG**: Debug-level messages

**When to Use**:
- Root cause analysis
- Security auditing
- Compliance requirements
- Detailed debugging

### 3. Traces
**Definition**: Records of request paths through distributed systems

**Characteristics**:
- Shows request flow
- Identifies bottlenecks
- Correlates across services
- Higher overhead than metrics

**Example Trace Structure**:
```
Trace ID: abc-123-def-456
├── Span: API Gateway (200ms)
│   ├── Span: Authentication Service (50ms)
│   └── Span: User Service (150ms)
│       ├── Span: Database Query (100ms)
│       └── Span: Cache Lookup (20ms)
```

**When to Use**:
- Performance optimization
- Dependency mapping
- Error correlation
- Latency analysis

## Metrics vs Logs vs Traces Comparison

| Aspect | Metrics | Logs | Traces |
|--------|---------|------|---------|
| **Storage Cost** | Low | High | Medium |
| **Query Performance** | Fast | Slow | Medium |
| **Debugging Value** | Low | High | High |
| **Alerting** | Excellent | Good | Fair |
| **Retention** | Long-term | Short-term | Medium-term |
| **Aggregation** | Easy | Difficult | Medium |

## SLI/SLO Concepts

### Service Level Indicators (SLIs)
**Definition**: Quantitative measures of service level

**Common SLIs**:
```yaml
# Availability SLI
availability = successful_requests / total_requests

# Latency SLI
latency_p99 = 99th_percentile_response_time

# Quality SLI
quality = valid_responses / total_responses

# Throughput SLI
throughput = requests_per_second
```

### Service Level Objectives (SLOs)
**Definition**: Target reliability expressed as SLI ranges

**Example SLOs**:
```yaml
# Availability SLO
- sli: availability
  target: 99.9%
  window: 30d

# Latency SLO
- sli: latency_p99
  target: < 200ms
  window: 7d

# Error Rate SLO
- sli: error_rate
  target: < 0.1%
  window: 24h
```

### Error Budgets
**Definition**: Amount of unreliability you can tolerate

**Calculation**:
```
Error Budget = 1 - SLO
If SLO = 99.9%, Error Budget = 0.1%

Monthly Error Budget = 0.1% × 30 days × 24 hours × 60 minutes
                    = 43.2 minutes of downtime per month
```

## Best Practices Overview

### 1. Instrumentation Strategy
```yaml
# Metrics Strategy
- Instrument business-critical paths
- Use consistent naming conventions
- Include relevant labels/tags
- Monitor both technical and business metrics

# Logging Strategy
- Use structured logging (JSON)
- Include correlation IDs
- Log at appropriate levels
- Implement log sampling for high-volume services

# Tracing Strategy
- Trace critical user journeys
- Use consistent span naming
- Include relevant attributes
- Implement sampling strategies
```

### 2. Alerting Best Practices
```yaml
# Alert Design
- Alert on symptoms, not causes
- Use multiple severity levels
- Include actionable context
- Implement alert fatigue prevention

# SLO-Based Alerting
- Alert on error budget burn rate
- Use multi-window alerting
- Implement escalation policies
- Regular alert review and tuning
```

### 3. Dashboard Design
```yaml
# Dashboard Principles
- Start with user journey
- Use consistent time ranges
- Include SLI/SLO tracking
- Implement drill-down capabilities

# Dashboard Hierarchy
- Executive: High-level business metrics
- Service: Service-specific SLIs
- Infrastructure: System health metrics
- Debug: Detailed troubleshooting views
```

## Practical Exercises

### Exercise 1: SLI Definition
**Scenario**: Define SLIs for an e-commerce checkout service

**Task**: Create SLI definitions for:
1. Availability
2. Latency
3. Quality

**Solution Template**:
```yaml
# Your SLI definitions here
checkout_availability_sli:
  definition: "Percentage of successful checkout requests"
  measurement: "COUNT(successful_checkouts) / COUNT(total_checkouts)"
  
checkout_latency_sli:
  definition: "99th percentile checkout response time"
  measurement: "PERCENTILE(checkout_response_time, 99)"
```

### Exercise 2: Log Analysis
**Scenario**: Analyze application logs to identify issues

**Sample Log Data**:
```json
[
  {"timestamp": "2024-01-15T10:30:00Z", "level": "ERROR", "service": "checkout", "message": "Payment failed", "user_id": "12345", "error_code": "PAYMENT_DECLINED"},
  {"timestamp": "2024-01-15T10:30:01Z", "level": "INFO", "service": "checkout", "message": "Checkout completed", "user_id": "12346", "order_id": "67890"},
  {"timestamp": "2024-01-15T10:30:02Z", "level": "WARN", "service": "checkout", "message": "High response time", "user_id": "12347", "response_time": 2.5}
]
```

**Questions**:
1. What patterns do you observe?
2. How would you create alerts for these scenarios?
3. What additional information would be helpful?

### Exercise 3: Trace Analysis
**Scenario**: Optimize service performance using trace data

**Sample Trace**:
```
Trace ID: trace-123
├── API Gateway (500ms)
│   ├── Authentication (50ms)
│   ├── Order Service (400ms)
│   │   ├── Database Query (350ms) ← Bottleneck
│   │   └── Cache Check (10ms)
│   └── Notification Service (20ms)
```

**Questions**:
1. Where is the performance bottleneck?
2. What optimization strategies would you recommend?
3. How would you validate improvements?

## Knowledge Check Questions

1. **What are the three pillars of observability?**
   - Answer: Metrics, Logs, and Traces

2. **When would you use logs instead of metrics?**
   - Answer: When you need detailed contextual information for debugging specific issues

3. **What is an Error Budget?**
   - Answer: The amount of unreliability you can tolerate, calculated as 1 - SLO

4. **Why is distributed tracing important?**
   - Answer: It shows request flow through distributed systems, helping identify bottlenecks and dependencies

5. **What makes a good SLI?**
   - Answer: It should be user-centric, measurable, and aligned with business outcomes

## Additional Resources

### Books
- "Site Reliability Engineering" by Google
- "Building Secure and Reliable Systems" by Google
- "Observability Engineering" by Charity Majors

### Online Resources
- OpenTelemetry Documentation
- Prometheus Best Practices
- Jaeger Tracing Guides

### Tools to Explore
- Prometheus (Metrics)
- Grafana (Visualization)
- Jaeger (Tracing)
- OpenTelemetry (Instrumentation)

## Next Steps
1. Complete hands-on workshops
2. Practice with real monitoring scenarios
3. Learn advanced topics
4. Pursue certification