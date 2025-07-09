# Logging Troubleshooting Guide

## Loki Issues

### 1. Loki Not Starting

**Symptoms:**
- Loki pods in CrashLoopBackOff
- "Failed to start" errors
- Configuration validation failures

**Diagnostic Commands:**
```bash
# Check Loki pod status
kubectl get pods -n monitoring | grep loki

# Check Loki logs
kubectl logs -n monitoring loki-0 --tail=50

# Check Loki configuration
kubectl get configmap -n monitoring loki-config -o yaml

# Validate Loki config
kubectl exec -n monitoring loki-0 -- loki -config.file=/etc/loki/loki.yaml -verify-config

# Check storage permissions
kubectl exec -n monitoring loki-0 -- ls -la /loki
```

**Solutions:**
- Fix configuration syntax errors
- Check storage permissions
- Verify resource limits
- Update storage configuration
- Check network connectivity

### 2. Log Ingestion Problems

**Symptoms:**
- Logs not appearing in Loki
- Ingestion rate limiting
- Out of order log entries

**Diagnostic Commands:**
```bash
# Check Loki ingestion rate
kubectl exec -n monitoring loki-0 -- curl -s localhost:3100/metrics | grep loki_ingester_ingested_samples_total

# Check ingestion errors
kubectl logs -n monitoring loki-0 | grep -i error

# Test log ingestion
curl -H "Content-Type: application/json" -XPOST -s "http://loki:3100/loki/api/v1/push" --data-raw \
  '{"streams": [{ "stream": { "job": "test" }, "values": [ [ "'$(date +%s%N)'", "test log message" ] ] }]}'

# Check ingestion limits
kubectl exec -n monitoring loki-0 -- curl -s localhost:3100/config | jq .limits_config
```

**Solutions:**
- Increase ingestion rate limits
- Fix log timestamp ordering
- Check log format
- Verify Promtail configuration
- Increase Loki resources

### 3. Storage Issues

**Symptoms:**
- "No space left on device" errors
- Slow query performance
- Index corruption

**Diagnostic Commands:**
```bash
# Check storage usage
kubectl exec -n monitoring loki-0 -- df -h /loki

# Check index health
kubectl exec -n monitoring loki-0 -- curl -s localhost:3100/ready

# Check storage metrics
kubectl exec -n monitoring loki-0 -- curl -s localhost:3100/metrics | grep loki_chunk_store

# Check retention configuration
kubectl exec -n monitoring loki-0 -- curl -s localhost:3100/config | jq .table_manager
```

**Loki storage configuration:**
```yaml
schema_config:
  configs:
  - from: 2020-05-15
    store: boltdb-shipper
    object_store: s3
    schema: v11
    index:
      prefix: loki_index_
      period: 24h

storage_config:
  boltdb_shipper:
    active_index_directory: /loki/index
    cache_location: /loki/index_cache
    shared_store: s3
    cache_ttl: 24h
  
  aws:
    s3: s3://loki-bucket
    region: us-east-1

table_manager:
  retention_deletes_enabled: true
  retention_period: 168h  # 7 days
```

**Solutions:**
- Increase storage capacity
- Configure retention policies
- Optimize index configuration
- Use object storage
- Implement compaction

## Log Parsing Problems

### 1. Promtail Not Collecting Logs

**Symptoms:**
- No logs in Loki from specific pods
- Promtail errors
- Missing log files

**Diagnostic Commands:**
```bash
# Check Promtail status
kubectl get pods -n monitoring | grep promtail

# Check Promtail logs
kubectl logs -n monitoring daemonset/promtail | grep -i error

# Check Promtail configuration
kubectl get configmap -n monitoring promtail-config -o yaml

# Test log file discovery
kubectl exec -n monitoring promtail-xxx -- ls -la /var/log/pods/

# Check Promtail targets
kubectl exec -n monitoring promtail-xxx -- curl -s localhost:9080/targets
```

**Promtail configuration example:**
```yaml
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push
    tenant_id: default

scrape_configs:
- job_name: kubernetes-pods
  kubernetes_sd_configs:
  - role: pod
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_node_name]
    target_label: __host__
  - action: labelmap
    regex: __meta_kubernetes_pod_label_(.+)
  - action: replace
    replacement: /var/log/pods/*$1/*.log
    separator: /
    source_labels: [__meta_kubernetes_pod_uid, __meta_kubernetes_pod_container_name]
    target_label: __path__
```

**Solutions:**
- Fix Promtail configuration
- Check file permissions
- Verify log file paths
- Update service discovery
- Restart Promtail pods

### 2. Log Format Issues

**Symptoms:**
- Unparsed log entries
- Missing log fields
- Incorrect log levels

**Diagnostic Commands:**
```bash
# Check log format
kubectl logs -n myapp deployment/myapp | head -5

# Test log parsing
kubectl exec -n monitoring promtail-xxx -- curl -s localhost:9080/debug/targets

# Check parsing errors
kubectl logs -n monitoring daemonset/promtail | grep -i parse

# Test regex patterns
echo "2023-01-01T10:00:00Z INFO [main] Application started" | grep -oP '\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z'
```

**Log parsing configuration:**
```yaml
scrape_configs:
- job_name: structured-logs
  static_configs:
  - targets: [localhost]
    labels:
      job: myapp
      __path__: /var/log/myapp/*.log
  
  pipeline_stages:
  - json:
      expressions:
        timestamp: timestamp
        level: level
        message: message
        service: service
  
  - timestamp:
      source: timestamp
      format: RFC3339
  
  - labels:
      level:
      service:
  
  - output:
      source: message
```

**Solutions:**
- Fix log format consistency
- Update parsing rules
- Use structured logging
- Implement proper timestamps
- Add log field validation

### 3. Performance Issues

**Symptoms:**
- Slow log shipping
- High CPU usage in Promtail
- Log backlogs

**Diagnostic Commands:**
```bash
# Check Promtail performance
kubectl top pods -n monitoring | grep promtail

# Check log shipping rate
kubectl exec -n monitoring promtail-xxx -- curl -s localhost:9080/metrics | grep promtail_sent_entries_total

# Check log file sizes
kubectl exec -n monitoring promtail-xxx -- du -sh /var/log/pods/*

# Monitor log processing
kubectl logs -n monitoring daemonset/promtail | grep -i "rate\|batch"
```

**Performance optimization:**
```yaml
server:
  http_listen_port: 9080
  grpc_listen_port: 0

clients:
  - url: http://loki:3100/loki/api/v1/push
    batchwait: 1s
    batchsize: 1048576
    timeout: 10s
    backoff_config:
      min_period: 500ms
      max_period: 5m
      max_retries: 10

limits_config:
  readline_rate: 10000
  readline_burst: 20000
```

**Solutions:**
- Optimize batch settings
- Increase resource limits
- Use log rotation
- Filter unnecessary logs
- Implement log compression

## Query Performance

### 1. Slow Log Queries

**Symptoms:**
- Query timeouts
- High query latency
- Grafana dashboard loading slowly

**Diagnostic Commands:**
```bash
# Check query performance
kubectl exec -n monitoring loki-0 -- curl -s localhost:3100/metrics | grep loki_query_duration_seconds

# Test specific queries
curl -G -s "http://loki:3100/loki/api/v1/query_range" \
  --data-urlencode 'query={job="myapp"}' \
  --data-urlencode 'start=1609459200' \
  --data-urlencode 'end=1609462800' | jq .

# Check query limits
kubectl exec -n monitoring loki-0 -- curl -s localhost:3100/config | jq .limits_config.max_query_length

# Monitor query cache
kubectl exec -n monitoring loki-0 -- curl -s localhost:3100/metrics | grep loki_cache
```

**Query optimization strategies:**
- Use specific time ranges
- Add label filters
- Limit result sets
- Use aggregation functions
- Implement query caching

**Example optimized queries:**
```logql
# Good: Specific time range and labels
{job="myapp", level="error"} |= "database" | json | line_format "{{.message}}"

# Better: With aggregation
sum(rate({job="myapp", level="error"}[5m])) by (service)

# Best: With filters and limited scope
{job="myapp", service="auth"} |= "login failed" | json | level="error" | line_format "{{.timestamp}} {{.message}}"
```

### 2. Index Performance Issues

**Symptoms:**
- Slow label queries
- High memory usage
- Index corruption

**Diagnostic Commands:**
```bash
# Check index size
kubectl exec -n monitoring loki-0 -- du -sh /loki/index

# Check index metrics
kubectl exec -n monitoring loki-0 -- curl -s localhost:3100/metrics | grep loki_index

# Test label queries
curl -s "http://loki:3100/loki/api/v1/labels" | jq .

# Check index health
kubectl exec -n monitoring loki-0 -- curl -s localhost:3100/ready
```

**Index optimization:**
```yaml
schema_config:
  configs:
  - from: 2020-05-15
    store: boltdb-shipper
    object_store: s3
    schema: v11
    index:
      prefix: loki_index_
      period: 24h
    
    chunk_store:
      chunk_cache_config:
        enable_fifocache: true
        fifocache:
          max_size_items: 1024
          validity: 24h

limits_config:
  max_query_series: 500
  max_query_parallelism: 32
  max_streams_per_user: 0
  max_line_size: 256000
```

**Solutions:**
- Optimize index configuration
- Reduce label cardinality
- Use appropriate retention
- Implement index caching
- Monitor index size

## Log Aggregation and Analysis

### 1. Log Aggregation Issues

**Symptoms:**
- Inconsistent log formats
- Missing log context
- Difficult log correlation

**Diagnostic Commands:**
```bash
# Check log formats across services
kubectl logs -n myapp deployment/service1 | head -5
kubectl logs -n myapp deployment/service2 | head -5

# Test log aggregation
curl -G -s "http://loki:3100/loki/api/v1/query" \
  --data-urlencode 'query=sum(count_over_time({job="myapp"}[1h])) by (service)'

# Check log correlation
curl -G -s "http://loki:3100/loki/api/v1/query" \
  --data-urlencode 'query={job="myapp"} | json | trace_id="12345"'
```

**Standardized log format:**
```json
{
  "timestamp": "2023-01-01T10:00:00Z",
  "level": "info",
  "service": "auth-service",
  "trace_id": "550e8400-e29b-41d4-a716-446655440000",
  "span_id": "6b221d5bc9e6496c",
  "message": "User authenticated successfully",
  "user_id": "user123",
  "ip": "192.168.1.100",
  "duration": 45
}
```

**Solutions:**
- Standardize log formats
- Add correlation IDs
- Use structured logging
- Implement log context
- Add service metadata

### 2. Log Analysis Problems

**Symptoms:**
- Difficult to find specific logs
- No log insights
- Poor log searchability

**Diagnostic Commands:**
```bash
# Test log search
curl -G -s "http://loki:3100/loki/api/v1/query" \
  --data-urlencode 'query={job="myapp"} |= "error" | json | level="error"'

# Check log patterns
curl -G -s "http://loki:3100/loki/api/v1/query" \
  --data-urlencode 'query=count_over_time({job="myapp"} |= "error" [1h])'

# Test log metrics
curl -G -s "http://loki:3100/loki/api/v1/query" \
  --data-urlencode 'query=sum(rate({job="myapp"} |= "error" [5m])) by (service)'
```

**Log analysis queries:**
```logql
# Error rate by service
sum(rate({job="myapp"} |= "error" [5m])) by (service)

# Top error messages
topk(10, sum(count_over_time({job="myapp"} |= "error" [1h])) by (message))

# Response time analysis
avg_over_time({job="myapp"} | json | unwrap duration [5m])

# User activity tracking
count_over_time({job="myapp"} |= "login" | json | user_id="user123" [1h])
```

**Solutions:**
- Implement log search indexes
- Create log dashboards
- Use log-based metrics
- Add log analysis tools
- Implement alerting on logs

## Monitoring Log Pipeline Health

### Key Metrics to Monitor

```bash
# Loki metrics
loki_ingester_ingested_samples_total
loki_ingester_ingested_samples_failures_total
loki_query_duration_seconds
loki_index_entries_total

# Promtail metrics
promtail_sent_entries_total
promtail_dropped_entries_total
promtail_files_active_total
promtail_read_bytes_total
```

### Log Pipeline Alerts

```yaml
groups:
  - name: logging-alerts
    rules:
    - alert: LokiDown
      expr: up{job="loki"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Loki is down"
        description: "Loki has been down for more than 5 minutes"

    - alert: PromtailDown
      expr: up{job="promtail"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Promtail is down"
        description: "Promtail has been down for more than 5 minutes"

    - alert: HighLogIngestionErrors
      expr: rate(loki_ingester_ingested_samples_failures_total[5m]) > 0.01
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High log ingestion error rate"
        description: "Log ingestion error rate is above 1% for 5 minutes"

    - alert: LogVolumeHigh
      expr: rate(loki_ingester_ingested_samples_total[5m]) > 10000
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "High log volume"
        description: "Log ingestion rate is above 10k samples/sec for 10 minutes"

    - alert: PromtailLogLag
      expr: time() - promtail_file_bytes_total > 300
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Promtail log lag"
        description: "Promtail is lagging behind in log processing"
```

## Emergency Procedures

### 1. Loki Recovery

**Complete recovery:**
```bash
# Backup current data
kubectl exec -n monitoring loki-0 -- tar -czf /tmp/loki-backup.tar.gz /loki

# Stop Loki
kubectl scale statefulset loki -n monitoring --replicas=0

# Clear corrupted data (if needed)
kubectl exec -n monitoring loki-0 -- rm -rf /loki/index/*

# Restart Loki
kubectl scale statefulset loki -n monitoring --replicas=1

# Verify recovery
kubectl get pods -n monitoring | grep loki
```

### 2. Log Data Recovery

**Restore from backup:**
```bash
# Copy backup to Loki pod
kubectl cp loki-backup.tar.gz monitoring/loki-0:/tmp/

# Extract backup
kubectl exec -n monitoring loki-0 -- tar -xzf /tmp/loki-backup.tar.gz -C /

# Restart Loki
kubectl rollout restart statefulset/loki -n monitoring
```

### 3. Promtail Recovery

**Reset Promtail:**
```bash
# Clear position files
kubectl exec -n monitoring promtail-xxx -- rm -f /tmp/positions.yaml

# Restart Promtail
kubectl rollout restart daemonset/promtail -n monitoring

# Verify log collection
kubectl logs -n monitoring daemonset/promtail | grep -i "starting\|ready"
```

## Common Error Messages and Solutions

| Error Message | Cause | Solution |
|---------------|-------|----------|
| "out of order entry" | Timestamp ordering | Fix log timestamps or enable out-of-order ingestion |
| "too many outstanding requests" | Rate limiting | Increase ingestion limits or reduce log volume |
| "entry too far behind" | Old log entries | Adjust ingestion time window |
| "line too long" | Large log lines | Increase max line size limit |
| "failed to parse" | Invalid log format | Fix log format or parsing configuration |
| "context deadline exceeded" | Query timeout | Optimize queries or increase timeout |
| "no such host" | DNS resolution | Check service names and network connectivity |

## Best Practices

### 1. Log Format Standardization

```go
// Standardized log entry
type LogEntry struct {
    Timestamp time.Time `json:"timestamp"`
    Level     string    `json:"level"`
    Service   string    `json:"service"`
    TraceID   string    `json:"trace_id,omitempty"`
    SpanID    string    `json:"span_id,omitempty"`
    Message   string    `json:"message"`
    Fields    map[string]interface{} `json:"fields,omitempty"`
}
```

### 2. Log Retention Strategy

```yaml
table_manager:
  retention_deletes_enabled: true
  retention_period: 168h  # 7 days for regular logs
  
# For different log types
limits_config:
  per_stream_rate_limit: 3MB
  per_stream_rate_limit_burst: 15MB
  max_streams_per_user: 10000
```

### 3. Query Optimization

```logql
# Use specific time ranges
{job="myapp"} |= "error" | json | timestamp > "2023-01-01T00:00:00Z"

# Filter early in the query
{job="myapp", service="auth"} |= "login" | json | level="info"

# Use aggregation for metrics
sum(rate({job="myapp"} |= "error" [5m])) by (service)
```