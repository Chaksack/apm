# Prometheus Troubleshooting Guide

## Common Prometheus Issues

### 1. High Memory Usage

**Symptoms:**
- Prometheus pod being OOMKilled
- High memory consumption on monitoring nodes
- Slow query responses

**Diagnostic Commands:**
```bash
# Check Prometheus memory usage
kubectl top pods -n monitoring | grep prometheus

# Check memory limits and requests
kubectl describe pod prometheus-0 -n monitoring | grep -A 5 -B 5 memory

# Check current memory usage from Prometheus metrics
curl -s "http://prometheus:9090/api/v1/query?query=process_resident_memory_bytes" | jq .
```

**Solutions:**
- Increase memory limits in Prometheus configuration
- Reduce retention period
- Optimize scrape intervals
- Use recording rules for frequently queried metrics

### 2. Storage Issues

**Symptoms:**
- Prometheus unable to start
- "No space left on device" errors
- Missing historical data

**Diagnostic Commands:**
```bash
# Check storage usage
kubectl exec -n monitoring prometheus-0 -- df -h /prometheus

# Check PVC status
kubectl get pvc -n monitoring

# Check storage class
kubectl get storageclass

# Check Prometheus storage metrics
curl -s "http://prometheus:9090/api/v1/query?query=prometheus_tsdb_symbol_table_size_bytes" | jq .
```

**Solutions:**
- Increase PVC size
- Adjust retention policies
- Clean up old data manually
- Configure proper storage class with expansion capability

### 3. Scraping Failures

**Symptoms:**
- Missing metrics from targets
- "Context deadline exceeded" errors
- Intermittent data gaps

**Diagnostic Commands:**
```bash
# Check target status
curl -s "http://prometheus:9090/api/v1/targets" | jq '.data.activeTargets[] | select(.health != "up")'

# Check scrape duration
curl -s "http://prometheus:9090/api/v1/query?query=scrape_duration_seconds" | jq .

# Check failed scrapes
curl -s "http://prometheus:9090/api/v1/query?query=up == 0" | jq .

# Check Prometheus logs
kubectl logs -n monitoring prometheus-0 | grep -i error
```

**Solutions:**
- Increase scrape timeout
- Reduce scrape frequency for heavy targets
- Check network connectivity
- Verify target endpoint availability

### 4. Configuration Issues

**Symptoms:**
- Prometheus not starting
- Rules not loading
- Targets not discovered

**Diagnostic Commands:**
```bash
# Validate Prometheus config
kubectl exec -n monitoring prometheus-0 -- promtool check config /etc/prometheus/prometheus.yml

# Check config reload
kubectl exec -n monitoring prometheus-0 -- promtool query instant 'prometheus_config_last_reload_successful'

# Check rule files
kubectl exec -n monitoring prometheus-0 -- promtool check rules /etc/prometheus/rules/*.yml
```

**Solutions:**
- Fix YAML syntax errors
- Validate rule expressions
- Check file permissions
- Restart Prometheus after config changes

## Query Optimization

### 1. Slow Queries

**Diagnostic Commands:**
```bash
# Check query performance
curl -s "http://prometheus:9090/api/v1/query?query=prometheus_engine_query_duration_seconds" | jq .

# Find expensive queries
curl -s "http://prometheus:9090/api/v1/query?query=topk(10, prometheus_engine_query_duration_seconds)" | jq .

# Check concurrent queries
curl -s "http://prometheus:9090/api/v1/query?query=prometheus_engine_queries" | jq .
```

**Optimization Strategies:**
- Use recording rules for complex queries
- Limit query range and resolution
- Use appropriate aggregation functions
- Avoid high cardinality metrics

### 2. Recording Rules

**Example optimized recording rules:**
```yaml
groups:
  - name: cpu_usage
    interval: 30s
    rules:
    - record: instance:cpu_usage:rate5m
      expr: |
        100 * (
          1 - avg by (instance) (
            rate(node_cpu_seconds_total{mode="idle"}[5m])
          )
        )
    
    - record: cluster:cpu_usage:rate5m
      expr: |
        100 * (
          1 - avg(
            rate(node_cpu_seconds_total{mode="idle"}[5m])
          )
        )
```

## Performance Tuning

### 1. Scrape Configuration

**Optimized scrape config:**
```yaml
scrape_configs:
  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
    - role: pod
    relabel_configs:
    - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
      action: keep
      regex: true
    - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
      action: replace
      target_label: __metrics_path__
      regex: (.+)
    scrape_interval: 15s
    scrape_timeout: 10s
    metrics_path: /metrics
```

### 2. Storage Configuration

**Optimized storage settings:**
```yaml
storage:
  tsdb:
    retention.time: "15d"
    retention.size: "10GB"
    min-block-duration: "2h"
    max-block-duration: "25h"
    wal-compression: true
```

### 3. Memory Optimization

**Memory-efficient configuration:**
```yaml
spec:
  resources:
    requests:
      memory: "2Gi"
      cpu: "500m"
    limits:
      memory: "4Gi"
      cpu: "2000m"
  storage:
    volumeClaimTemplate:
      spec:
        resources:
          requests:
            storage: "50Gi"
```

## Monitoring Prometheus Health

### Key Metrics to Monitor

```bash
# Prometheus uptime
prometheus_build_info

# TSDB status
prometheus_tsdb_head_samples_appended_total
prometheus_tsdb_head_series

# Query performance
prometheus_engine_query_duration_seconds
prometheus_engine_queries_concurrent_max

# Storage metrics
prometheus_tsdb_symbol_table_size_bytes
prometheus_tsdb_head_chunks
prometheus_tsdb_wal_fsync_duration_seconds

# Scrape metrics
prometheus_target_scrapes_exceeded_sample_limit_total
prometheus_target_scrape_pool_exceeded_label_limits_total
```

## Alerting Rules for Prometheus Health

```yaml
groups:
  - name: prometheus-health
    rules:
    - alert: PrometheusDown
      expr: up{job="prometheus"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Prometheus is down"
        description: "Prometheus has been down for more than 5 minutes"

    - alert: PrometheusHighMemory
      expr: process_resident_memory_bytes{job="prometheus"} > 2e9
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "Prometheus high memory usage"
        description: "Prometheus is using more than 2GB of memory"

    - alert: PrometheusSlowQueries
      expr: prometheus_engine_query_duration_seconds{quantile="0.9"} > 10
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Prometheus slow queries"
        description: "90th percentile query duration is over 10 seconds"
```

## Emergency Procedures

### 1. Prometheus Recovery

```bash
# Emergency restart
kubectl delete pod prometheus-0 -n monitoring

# Check pod status
kubectl get pods -n monitoring -w

# Verify metrics collection
kubectl port-forward -n monitoring svc/prometheus 9090:9090
```

### 2. Data Recovery

```bash
# Backup current data
kubectl exec -n monitoring prometheus-0 -- tar -czf /tmp/prometheus-backup.tar.gz /prometheus

# Copy backup
kubectl cp monitoring/prometheus-0:/tmp/prometheus-backup.tar.gz ./prometheus-backup.tar.gz

# Restore from backup (if needed)
kubectl cp ./prometheus-backup.tar.gz monitoring/prometheus-0:/tmp/
kubectl exec -n monitoring prometheus-0 -- tar -xzf /tmp/prometheus-backup.tar.gz -C /
```

## Common Error Messages and Solutions

| Error Message | Cause | Solution |
|---------------|-------|----------|
| "opening storage failed" | Corrupted storage | Delete data directory and restart |
| "context deadline exceeded" | Slow scrape targets | Increase scrape timeout |
| "sample limit exceeded" | Too many metrics | Increase sample_limit or filter metrics |
| "out of memory" | Insufficient memory | Increase memory limits |
| "no space left on device" | Full storage | Increase PVC size or clean up data |