# Loki User Guide

## Overview

Loki is a log aggregation system designed to store and query logs efficiently. Unlike traditional logging systems, Loki indexes only metadata (labels) and keeps the log content unindexed, making it cost-effective and performant.

## Getting Started

### Accessing Loki

1. **Web Interface**: Navigate to `http://localhost:3100` (default)
2. **API**: Access via REST API at `http://localhost:3100/loki/api/v1/`
3. **Grafana**: Use Loki as a data source in Grafana

### Basic Architecture

- **Promtail**: Log shipping agent
- **Loki**: Storage and query engine
- **Grafana**: Visualization and exploration

## LogQL Fundamentals

### Basic Syntax

LogQL is Loki's query language, similar to PromQL but for logs.

#### Log Stream Selectors
```logql
# Basic selector
{job="nginx"}

# Multiple labels
{job="nginx", env="production"}

# Regex matching
{job=~"nginx|apache"}

# Negative matching
{job!="nginx"}
```

#### Log Pipeline
```logql
# Basic pipeline
{job="nginx"} |= "error"

# Chain operations
{job="nginx"} |= "error" |~ "5[0-9][0-9]"

# JSON parsing
{job="app"} | json | level="error"
```

### Query Types

#### Log Range Queries
```logql
# Last 5 minutes
{job="nginx"}[5m]

# Specific time range
{job="nginx"}[2023-01-01T00:00:00Z:2023-01-01T23:59:59Z]
```

#### Metric Queries
```logql
# Count over time
count_over_time({job="nginx"}[5m])

# Rate of logs
rate({job="nginx"}[5m])

# Bytes over time
bytes_over_time({job="nginx"}[5m])
```

## Advanced LogQL Queries

### Filtering Operations

#### Line Filters
```logql
# Contains
{job="nginx"} |= "error"

# Does not contain
{job="nginx"} != "debug"

# Regex match
{job="nginx"} |~ "error|ERROR"

# Negative regex
{job="nginx"} !~ "health|ping"

# Case insensitive
{job="nginx"} |~ "(?i)error"
```

#### Label Filters
```logql
# After parsing
{job="nginx"} | json | level="error"

# Numeric comparison
{job="nginx"} | json | response_time > 100

# String comparison
{job="nginx"} | json | method="POST"
```

### Parsing Operations

#### JSON Parsing
```logql
# Parse JSON
{job="app"} | json

# Parse specific fields
{job="app"} | json level, message, timestamp

# Nested JSON
{job="app"} | json | json field="request"
```

#### Logfmt Parsing
```logql
# Parse logfmt
{job="app"} | logfmt

# Parse specific fields
{job="app"} | logfmt level, msg, ts
```

#### Regex Parsing
```logql
# Extract with regex
{job="nginx"} | regexp "(?P<method>\\w+) (?P<path>\\S+) (?P<status>\\d+)"

# Named groups
{job="nginx"} | regexp `(?P<ip>\S+) - - \[(?P<timestamp>[^\]]+)\] "(?P<method>\S+) (?P<path>\S+) (?P<protocol>\S+)" (?P<status>\d+) (?P<bytes>\d+)`
```

### Aggregation Functions

#### Count Functions
```logql
# Count logs
count_over_time({job="nginx"}[5m])

# Count by label
count_over_time({job="nginx"} | json | level="error"[5m])

# Count distinct
count_over_time({job="nginx"} | json | __error__=""[5m])
```

#### Rate Functions
```logql
# Log rate
rate({job="nginx"}[5m])

# Bytes rate
bytes_rate({job="nginx"}[5m])

# Error rate
rate({job="nginx"} |= "error"[5m])
```

#### Statistical Functions
```logql
# Average
avg_over_time({job="nginx"} | json | unwrap response_time[5m])

# Quantiles
quantile_over_time(0.95, {job="nginx"} | json | unwrap response_time[5m])

# Min/Max
min_over_time({job="nginx"} | json | unwrap response_time[5m])
max_over_time({job="nginx"} | json | unwrap response_time[5m])
```

## Log Exploration

### Search Strategies

#### Time-based Search
```logql
# Recent logs
{job="nginx"}

# Specific time range
{job="nginx"}[2023-01-01T10:00:00Z:2023-01-01T11:00:00Z]

# Relative time
{job="nginx"}[1h]
```

#### Service-based Search
```logql
# Specific service
{service="user-service"}

# Multiple services
{service=~"user-service|order-service"}

# Environment filtering
{service="user-service", env="production"}
```

#### Error Investigation
```logql
# All errors
{job="app"} |= "error"

# Specific error types
{job="app"} |~ "(?i)(error|exception|failed|panic)"

# Error levels
{job="app"} | json | level="error"

# HTTP errors
{job="nginx"} | regexp "(?P<status>5\\d\\d)"
```

### Correlation Techniques

#### Trace Correlation
```logql
# Find logs for specific trace
{job="app"} | json | trace_id="abc123"

# Find related spans
{job="app"} | json | span_id="def456"
```

#### User Journey Tracking
```logql
# Track user sessions
{job="app"} | json | user_id="12345"

# Track requests
{job="app"} | json | request_id="req-789"
```

#### Time-based Correlation
```logql
# Logs around specific time
{job="app"}[2023-01-01T10:00:00Z:2023-01-01T10:05:00Z]

# Before/after comparison
{job="app"} | json | timestamp > "2023-01-01T10:00:00Z"
```

## Label Management

### Label Best Practices

#### Label Cardinality
```logql
# Good: Low cardinality
{job="nginx", env="production"}

# Bad: High cardinality
{job="nginx", user_id="12345"}
```

#### Label Naming
```logql
# Consistent naming
{service="user-service", environment="prod"}

# Avoid dynamic labels
{job="nginx"} | json | level="error"  # Parse at query time
```

### Label Manipulation

#### Label Formatting
```logql
# Format labels
{job="nginx"} | json | line_format "{{.method}} {{.path}} {{.status}}"

# Custom formatting
{job="nginx"} | json | line_format "{{.timestamp}} [{{.level}}] {{.message}}"
```

#### Label Filtering
```logql
# Keep specific labels
{job="nginx"} | json | label_format level=level, message=msg

# Remove labels
{job="nginx"} | json | label_format timestamp=""
```

## Performance Optimization

### Query Optimization

#### Efficient Selectors
```logql
# Good: Specific labels
{job="nginx", env="production"}

# Bad: Too broad
{job=~".*"}
```

#### Time Range Optimization
```logql
# Good: Reasonable time range
{job="nginx"}[1h]

# Bad: Too wide
{job="nginx"}[30d]
```

#### Parsing Optimization
```logql
# Good: Parse after filtering
{job="nginx"} |= "error" | json

# Bad: Parse before filtering
{job="nginx"} | json | level="error"
```

### Index Optimization

#### Label Design
```yaml
# Good labels (low cardinality)
- job
- environment
- service
- level

# Bad labels (high cardinality)
- user_id
- request_id
- timestamp
- ip_address
```

#### Chunk Configuration
```yaml
# Optimize chunk size
chunk_idle_period: 30m
chunk_retain_period: 15m
max_chunk_age: 1h
```

### Storage Optimization

#### Retention Configuration
```yaml
# Configure retention
retention_deletes_enabled: true
retention_period: 168h  # 7 days

# Per-tenant retention
table_manager:
  retention_deletes_enabled: true
  retention_period: 168h
```

#### Compression
```yaml
# Enable compression
chunk_encoding: gzip
chunk_target_size: 1572864  # 1.5MB
```

## Integration Examples

### Promtail Configuration

#### Basic Configuration
```yaml
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://localhost:3100/loki/api/v1/push

scrape_configs:
  - job_name: system
    static_configs:
      - targets:
          - localhost
        labels:
          job: varlogs
          __path__: /var/log/*log
```

#### Advanced Configuration
```yaml
scrape_configs:
  - job_name: nginx
    static_configs:
      - targets:
          - localhost
        labels:
          job: nginx
          env: production
          __path__: /var/log/nginx/*.log
    pipeline_stages:
      - regex:
          expression: '^(?P<remote_addr>\S+) - - \[(?P<time>[^\]]+)\] "(?P<method>\S+) (?P<path>\S+) (?P<protocol>\S+)" (?P<status>\d+) (?P<bytes>\d+)'
      - timestamp:
          source: time
          format: 02/Jan/2006:15:04:05 -0700
      - labels:
          method:
          status:
```

### Docker Integration

#### Docker Driver
```yaml
version: '3.8'
services:
  app:
    image: nginx
    logging:
      driver: loki
      options:
        loki-url: http://localhost:3100/loki/api/v1/push
        loki-external-labels: job=nginx,env=production
```

#### Compose with Promtail
```yaml
version: '3.8'
services:
  app:
    image: nginx
    volumes:
      - ./logs:/var/log/nginx
    labels:
      - "promtail.enable=true"
      - "promtail.job=nginx"
      - "promtail.path=/var/log/nginx/*.log"
```

### Kubernetes Integration

#### DaemonSet Configuration
```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: promtail
spec:
  selector:
    matchLabels:
      app: promtail
  template:
    metadata:
      labels:
        app: promtail
    spec:
      containers:
        - name: promtail
          image: grafana/promtail:latest
          args:
            - -config.file=/etc/promtail/config.yml
          volumeMounts:
            - name: config
              mountPath: /etc/promtail
            - name: varlog
              mountPath: /var/log
              readOnly: true
```

#### Pod Annotations
```yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    promtail.io/collect: "true"
    promtail.io/logs-path: "/var/log/app.log"
spec:
  containers:
    - name: app
      image: myapp:latest
```

## Common Use Cases

### Application Logging

#### Error Tracking
```logql
# All application errors
{job="app"} | json | level="error"

# Error rate over time
rate({job="app"} | json | level="error"[5m])

# Top error messages
topk(10, count by (message) (count_over_time({job="app"} | json | level="error"[1h])))
```

#### Performance Analysis
```logql
# Slow requests
{job="nginx"} | json | response_time > 1000

# Response time distribution
histogram_quantile(0.95, sum(rate({job="nginx"} | json | unwrap response_time[5m])) by (le))
```

### Infrastructure Monitoring

#### System Logs
```logql
# System errors
{job="system"} |~ "(?i)(error|fail|panic)"

# Authentication failures
{job="system"} |~ "authentication failure"

# Disk space warnings
{job="system"} |~ "disk.*space"
```

#### Network Monitoring
```logql
# Network errors
{job="network"} |~ "(?i)(timeout|connection.*failed)"

# Traffic analysis
sum(rate({job="nginx"}[5m])) by (method)
```

### Security Monitoring

#### Access Logs
```logql
# Failed logins
{job="auth"} | json | event="login_failed"

# Suspicious IP addresses
{job="nginx"} | json | status="403"

# Unusual user agents
{job="nginx"} | json | user_agent=~"(?i)(bot|crawler|spider)"
```

#### Audit Logs
```logql
# Admin actions
{job="audit"} | json | user_role="admin"

# Permission changes
{job="audit"} | json | action="permission_change"
```

## Best Practices

### Query Best Practices

1. **Use Specific Labels**: Start with specific label selectors
2. **Filter Early**: Apply filters before parsing
3. **Reasonable Time Ranges**: Avoid overly broad time ranges
4. **Efficient Parsing**: Parse only necessary fields
5. **Regex Optimization**: Use efficient regex patterns

### Label Strategy

1. **Low Cardinality**: Keep label values limited
2. **Consistent Naming**: Use standard label names
3. **Avoid Dynamic Labels**: Don't use high-cardinality values
4. **Meaningful Labels**: Use descriptive label names
5. **Environment Separation**: Use environment labels

### Performance Guidelines

1. **Index Awareness**: Understand label indexing
2. **Query Caching**: Leverage query result caching
3. **Batch Operations**: Use batch APIs when possible
4. **Monitor Resources**: Track Loki resource usage
5. **Optimize Retention**: Set appropriate retention policies

## Troubleshooting

### Common Issues

#### No Data Found
```logql
# Check label existence
{job="nginx"}  # Verify job label exists

# Check time range
{job="nginx"}[24h]  # Expand time range

# Verify label values
{job=~".*"}  # List all jobs
```

#### High Cardinality
```logql
# Identify high cardinality labels
{__name__=~".+"} | json | line_format "{{.level}}"

# Check label distribution
count by (level) ({job="app"} | json)
```

#### Slow Queries
```logql
# Optimize with early filtering
{job="nginx"} |= "error" | json | level="error"

# Use efficient time ranges
{job="nginx"}[1h]  # Instead of [24h]
```

### Debug Techniques

#### Query Analysis
1. **Start Simple**: Begin with basic selectors
2. **Add Filters**: Gradually add complexity
3. **Check Results**: Verify each step
4. **Optimize**: Remove unnecessary operations

#### Label Debugging
```logql
# List all labels
{__name__=~".+"}

# Check label values
{job=~".*"}

# Verify parsing
{job="app"} | json | line_format "{{.level}}"
```

## Advanced Features

### Alerting with Loki

#### LogQL Alerts
```yaml
groups:
  - name: loki_alerts
    rules:
      - alert: HighErrorRate
        expr: |
          sum(rate({job="app"} | json | level="error"[5m])) > 10
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
```

### Log Patterns

#### Pattern Detection
```logql
# Detect patterns
{job="nginx"} | pattern "<ip> - - [<timestamp>] \"<method> <path> <protocol>\" <status> <bytes>"

# Count patterns
count by (pattern) ({job="nginx"} | pattern "<ip> - - [<timestamp>] \"<method> <path> <protocol>\" <status> <bytes>")
```

### Multi-tenancy

#### Tenant Configuration
```yaml
auth_enabled: true
server:
  http_listen_port: 3100
  grpc_listen_port: 9096
```

#### Tenant Queries
```bash
# Query with tenant header
curl -H "X-Scope-OrgID: tenant1" "http://localhost:3100/loki/api/v1/query?query={job=\"nginx\"}"
```

## Resources

- [Loki Documentation](https://grafana.com/docs/loki/latest/)
- [LogQL Reference](https://grafana.com/docs/loki/latest/logql/)
- [Promtail Configuration](https://grafana.com/docs/loki/latest/clients/promtail/)
- [Best Practices](https://grafana.com/docs/loki/latest/best-practices/)