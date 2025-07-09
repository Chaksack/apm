# Loki and Promtail Configuration for GoFiber Applications

This directory contains production-ready Loki and Promtail configurations optimized for GoFiber applications running in Kubernetes.

## Overview

- **Loki**: A horizontally-scalable, highly-available log aggregation system
- **Promtail**: An agent that ships logs to Loki

## Key Features

### Loki Configuration (`loki-config.yaml`)

1. **Storage Configuration**
   - BoltDB Shipper for index storage with 24-hour chunks
   - Filesystem storage for chunks with compaction enabled
   - 30-day retention policy (configurable)

2. **Performance Optimizations**
   - Query result caching with 24-hour TTL
   - Embedded cache for chunk store (100MB)
   - Parallel query execution (up to 32 concurrent)
   - WAL enabled for data durability

3. **Limits and Quotas**
   - 16MB/s ingestion rate with 32MB burst
   - 5MB/s per-stream rate limit
   - 5000 max entries per query
   - 30-day query lookback period

### Promtail Configuration (`promtail-config.yaml`)

1. **GoFiber-Specific Pipeline Stages**
   - JSON log parsing for structured logs
   - Multiline handling for stack traces
   - Label extraction (method, path, status, latency)
   - Timestamp parsing with multiple format support

2. **Log Filtering**
   - Drops health check endpoints (/health, /metrics, /ready)
   - Filters out common non-critical errors
   - Optional debug log dropping

3. **Metrics Generation**
   - HTTP request duration histogram from latency field

### Enhanced DaemonSet (`promtail-daemonset-enhanced.yaml`)

1. **Resource Management**
   - CPU: 100m request, 200m limit
   - Memory: 128Mi request, 256Mi limit
   - PodDisruptionBudget for high availability

2. **Security**
   - Non-root user execution
   - Read-only root filesystem
   - Minimal capabilities (only DAC_READ_SEARCH)

3. **Volume Mounts**
   - Container logs from multiple runtimes (Docker, containerd)
   - System logs and journal
   - Persistent position tracking

## Deployment

1. **Deploy Loki:**
```bash
kubectl apply -f loki-config.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

2. **Deploy Promtail:**
```bash
kubectl apply -f promtail-config.yaml
kubectl apply -f promtail-daemonset-enhanced.yaml
```

## GoFiber Application Configuration

Use the provided `gofiber-logging-example.go` as a reference for implementing structured logging in your GoFiber application. Key points:

1. Use JSON format for logs
2. Include standard fields: time, level, method, path, status, latency
3. Add request IDs for tracing
4. Log errors with appropriate context

## Monitoring and Troubleshooting

1. **Check Promtail status:**
```bash
kubectl logs -n default -l app=promtail --tail=50
```

2. **Check Loki status:**
```bash
kubectl logs -n default -l app=loki --tail=50
```

3. **Verify log ingestion:**
```bash
curl http://loki:3100/loki/api/v1/query?query='{job="kubernetes-pods"}'
```

## Configuration Tuning

### For High-Volume Applications

1. Increase Loki limits:
   - `ingestion_rate_mb`: Up to 50
   - `ingestion_burst_size_mb`: Up to 100
   - `max_streams_per_user`: Up to 10000

2. Scale Promtail resources:
   - CPU limit: Up to 500m
   - Memory limit: Up to 512Mi

### For Long-Term Storage

1. Adjust retention:
   - `retention_period`: Up to 2160h (90 days)
   - Consider using object storage (S3, GCS) for chunks

2. Configure compaction:
   - `compaction_interval`: Reduce to 5m for faster cleanup
   - `retention_delete_worker_count`: Increase for faster deletes

## Integration with Grafana

1. Add Loki as a data source in Grafana
2. Use LogQL queries to explore logs:
   ```
   {namespace="default", app="gofiber"} |= "error"
   ```
3. Create dashboards with log panels and metrics

## Best Practices

1. **Structured Logging**: Always use JSON format with consistent fields
2. **Label Cardinality**: Keep labels low-cardinality (< 100 unique values)
3. **Log Levels**: Use appropriate log levels (debug, info, warn, error)
4. **Sensitive Data**: Never log passwords, tokens, or PII
5. **Performance**: Use sampling for high-frequency logs