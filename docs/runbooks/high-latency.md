# High Latency Runbook

## Alert Definition
- **Trigger**: P95 latency > 1000ms for 5 minutes
- **Severity**: Warning/Critical (depends on SLA)
- **Team**: Platform Engineering

## Performance Troubleshooting

### 1. Latency Assessment
```bash
# Check current P50, P95, P99 latencies
curl -s http://localhost:9090/api/v1/query?query=histogram_quantile(0.95,rate(http_request_duration_seconds_bucket[5m]))

# Identify slowest endpoints
curl -s http://localhost:9090/api/v1/query?query=topk(10,histogram_quantile(0.95,rate(http_request_duration_seconds_bucket[5m])))

# Check latency trend
curl -s http://localhost:9090/api/v1/query_range?query=histogram_quantile(0.95,rate(http_request_duration_seconds_bucket[5m]))&start=$(date -u -d '1 hour ago' +%s)&end=$(date +%s)&step=60
```

### 2. Component Analysis
- **Database**: Query execution time, lock waits
- **Cache**: Hit rates, eviction rates
- **Network**: Packet loss, retransmissions
- **Application**: GC pauses, thread pool saturation

### 3. Distributed Tracing
```bash
# Sample slow requests
curl -s http://jaeger:16686/api/traces?service=<service>&minDuration=1000ms&limit=20

# Analyze trace breakdowns
# Look for: slow DB queries, sequential calls that could be parallelized, unnecessary calls
```

## Query Analysis

### Database Performance
```sql
-- Find slow queries (PostgreSQL)
SELECT query, mean_exec_time, calls, total_exec_time
FROM pg_stat_statements
WHERE mean_exec_time > 100
ORDER BY mean_exec_time DESC
LIMIT 20;

-- Check for lock contention
SELECT blocked_locks.pid AS blocked_pid,
       blocked_activity.usename AS blocked_user,
       blocking_locks.pid AS blocking_pid,
       blocking_activity.usename AS blocking_user,
       blocked_activity.query AS blocked_statement,
       blocking_activity.query AS blocking_statement
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks blocking_locks ON blocking_locks.locktype = blocked_locks.locktype
JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.granted;
```

### Application Profiling
```bash
# Enable CPU profiling (Go example)
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof -http=:8080 cpu.prof

# Java thread dump
kubectl exec <pod> -- jstack <pid> > thread_dump.txt

# Check for GC issues (Java)
kubectl exec <pod> -- jstat -gcutil <pid> 1000 10
```

## Common Causes

### 1. Database Issues
- **Slow queries**: Missing indexes, full table scans
- **Lock contention**: Long-running transactions
- **Connection pool**: Exhaustion, too many idle connections

### 2. Cache Problems
- **Cold cache**: After restart or cache flush
- **Cache stampede**: Multiple requests for same expired key
- **Inefficient caching**: Too short TTL, wrong granularity

### 3. Resource Saturation
- **CPU**: Inefficient algorithms, busy wait loops
- **Memory**: Excessive GC, swapping
- **I/O**: Disk saturation, network congestion

### 4. External Dependencies
- **Third-party APIs**: Rate limiting, degraded performance
- **CDN**: Origin overload, purge storms
- **Message queues**: Backlog processing

## Scaling Procedures

### Horizontal Scaling
```bash
# Auto-scale based on CPU
kubectl autoscale deployment <service> --cpu-percent=70 --min=3 --max=10

# Manual scale for immediate relief
kubectl scale deployment <service> --replicas=<count>

# Check scaling status
kubectl get hpa
```

### Vertical Scaling
```bash
# Update resource requests/limits
kubectl patch deployment <service> -p '
{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "<container>",
          "resources": {
            "requests": {"memory": "1Gi", "cpu": "500m"},
            "limits": {"memory": "2Gi", "cpu": "1000m"}
          }
        }]
      }
    }
  }
}'
```

### Database Scaling
```sql
-- Add read replicas
ALTER DATABASE <db> SET synchronous_commit = 'local';

-- Increase connection limits
ALTER SYSTEM SET max_connections = 500;
SELECT pg_reload_conf();

-- Enable query parallelization
SET max_parallel_workers_per_gather = 4;
```

## Remediation Steps

### Immediate Actions (0-5 minutes)
1. **Increase capacity**
   - Scale out affected services
   - Increase cache size/TTL
   - Enable query caching

2. **Reduce load**
   - Enable rate limiting
   - Activate circuit breakers
   - Defer non-critical background jobs

3. **Optimize critical path**
   - Disable expensive features temporarily
   - Switch to degraded mode
   - Serve cached/stale content

### Short-term Actions (5-30 minutes)
1. **Query optimization**
   ```sql
   -- Add missing indexes
   CREATE INDEX CONCURRENTLY idx_<table>_<columns> ON <table>(<columns>);
   
   -- Update statistics
   ANALYZE <table>;
   ```

2. **Cache warming**
   ```bash
   # Preload frequently accessed data
   for key in $(cat critical_keys.txt); do
     curl -X GET "http://service/cache/warm?key=$key"
   done
   ```

3. **Connection tuning**
   - Increase connection pool size
   - Adjust timeout values
   - Enable connection multiplexing

### Long-term Actions (30+ minutes)
1. **Architecture improvements**
   - Implement caching layers
   - Add read replicas
   - Optimize data models

2. **Code optimization**
   - Profile and fix hot paths
   - Batch operations
   - Implement pagination

## Monitoring Queries

```promql
# Latency by percentile
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# Requests per second
rate(http_requests_total[5m])

# Error rate correlation
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])

# Database query time
rate(mysql_query_duration_seconds_sum[5m]) / rate(mysql_query_duration_seconds_count[5m])

# Cache hit rate
rate(cache_hits_total[5m]) / (rate(cache_hits_total[5m]) + rate(cache_misses_total[5m]))
```

## Related Documents
- [High Error Rate Runbook](./high-error-rate.md)
- [Infrastructure Alerts Runbook](./infrastructure-alerts.md)
- [Performance Tuning Guide](../performance-tuning.md)