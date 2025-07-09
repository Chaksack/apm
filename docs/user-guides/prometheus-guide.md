# Prometheus User Guide

## Overview

Prometheus is a monitoring and alerting toolkit that collects metrics from your applications and infrastructure. This guide covers essential concepts and practical workflows for using Prometheus effectively.

## Getting Started

### Accessing Prometheus

1. **Web Interface**: Navigate to `http://localhost:9090` (default)
2. **API**: Access metrics data via REST API at `http://localhost:9090/api/v1/`

### Basic Navigation

- **Graph**: Query and visualize metrics
- **Alerts**: View active alerts and their states
- **Status**: Check configuration, targets, and service discovery
- **Targets**: Monitor scrape endpoints health

## PromQL Query Guide

### Basic Syntax

```promql
# Instant vector - current value
http_requests_total

# Range vector - values over time
http_requests_total[5m]

# Scalar - single numeric value
rate(http_requests_total[5m])
```

### Common Query Patterns

#### Rate Calculations
```promql
# Request rate per second
rate(http_requests_total[5m])

# Error rate
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])

# 99th percentile response time
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))
```

#### Aggregation
```promql
# Sum across all instances
sum(rate(http_requests_total[5m]))

# Average by service
avg by (service) (rate(http_requests_total[5m]))

# Top 5 services by request rate
topk(5, sum by (service) (rate(http_requests_total[5m])))
```

#### Filtering
```promql
# Filter by label
http_requests_total{method="GET"}

# Regex matching
http_requests_total{status=~"2.."}

# Exclude labels
http_requests_total{method!="OPTIONS"}
```

### Advanced Queries

#### Increase and Delta
```promql
# Total increase over time
increase(http_requests_total[1h])

# Delta for gauges
delta(cpu_usage_percent[5m])
```

#### Joins and Math
```promql
# CPU usage percentage
100 * (1 - rate(cpu_idle_seconds_total[5m]))

# Memory usage ratio
memory_used_bytes / memory_total_bytes
```

#### Time-based Functions
```promql
# Predict linear trend
predict_linear(disk_usage_bytes[1h], 3600)

# Day-over-day comparison
increase(http_requests_total[1h]) / increase(http_requests_total[1h] offset 24h)
```

## Metrics Exploration

### Discovering Metrics

1. **Metrics Browser**: Use the dropdown in the query interface
2. **Label Browser**: Explore available labels for each metric
3. **Metric Metadata**: Check help text and type information

### Common Metric Types

#### Counters
- Always increasing values
- Use `rate()` or `increase()` for meaningful data
- Examples: `http_requests_total`, `errors_total`

#### Gauges
- Can go up and down
- Use directly or with `delta()`
- Examples: `cpu_usage_percent`, `memory_used_bytes`

#### Histograms
- Measure distributions
- Use `histogram_quantile()` for percentiles
- Examples: `http_request_duration_seconds`, `response_size_bytes`

#### Summaries
- Pre-calculated quantiles
- Use directly without functions
- Examples: `request_duration_summary`

### Exploration Workflow

1. **Start Broad**: Begin with high-level metrics
2. **Add Filters**: Narrow down by service, environment, etc.
3. **Aggregate**: Sum, average, or group by relevant dimensions
4. **Visualize**: Use graphs to understand trends
5. **Drill Down**: Investigate anomalies or patterns

## Alert Rule Creation

### Basic Alert Structure

```yaml
groups:
  - name: example_alerts
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} requests per second"
```

### Common Alert Patterns

#### Service Health
```yaml
- alert: ServiceDown
  expr: up == 0
  for: 1m
  labels:
    severity: critical
  annotations:
    summary: "Service {{ $labels.instance }} is down"
    description: "{{ $labels.job }} has been down for more than 1 minute"
```

#### Resource Usage
```yaml
- alert: HighMemoryUsage
  expr: (memory_used_bytes / memory_total_bytes) > 0.8
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High memory usage on {{ $labels.instance }}"
    description: "Memory usage is {{ $value | humanizePercentage }}"
```

#### Performance Degradation
```yaml
- alert: HighResponseTime
  expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 0.5
  for: 2m
  labels:
    severity: warning
  annotations:
    summary: "High response time for {{ $labels.service }}"
    description: "95th percentile response time is {{ $value }}s"
```

### Alert Best Practices

1. **Meaningful Thresholds**: Base on business impact, not arbitrary numbers
2. **Appropriate Duration**: Use `for` clause to avoid flapping
3. **Clear Annotations**: Include context and actionable information
4. **Severity Levels**: Use consistent labeling (critical, warning, info)
5. **Runbook Links**: Include troubleshooting documentation

## Recording Rules

### Purpose
- Pre-calculate expensive queries
- Improve dashboard performance
- Simplify complex expressions

### Example Recording Rules

```yaml
groups:
  - name: recording_rules
    interval: 30s
    rules:
      - record: job:http_requests:rate5m
        expr: sum(rate(http_requests_total[5m])) by (job)
      
      - record: job:http_errors:rate5m
        expr: sum(rate(http_requests_total{status=~"5.."}[5m])) by (job)
      
      - record: job:http_error_rate
        expr: job:http_errors:rate5m / job:http_requests:rate5m
```

## Best Practices

### Query Optimization

1. **Use Recording Rules**: For frequently used complex queries
2. **Limit Time Ranges**: Avoid unnecessarily long ranges
3. **Efficient Aggregation**: Aggregate before applying functions
4. **Avoid Regex**: Use exact matches when possible

### Metric Design

1. **Consistent Naming**: Follow naming conventions
2. **Appropriate Labels**: Use labels for dimensions, not values
3. **Label Cardinality**: Keep label combinations reasonable
4. **Metric Types**: Choose appropriate metric types

### Monitoring Strategy

1. **RED Method**: Rate, Errors, Duration for services
2. **USE Method**: Utilization, Saturation, Errors for resources
3. **Golden Signals**: Latency, traffic, errors, saturation
4. **SLI/SLO**: Define service level indicators and objectives

### Performance Tips

1. **Retention Policy**: Configure appropriate retention periods
2. **Scrape Intervals**: Balance freshness vs. performance
3. **Query Limits**: Set reasonable query timeouts
4. **Storage**: Monitor disk usage and optimize storage

## Common Workflows

### Troubleshooting Service Issues

1. **Check Service Health**:
   ```promql
   up{job="my-service"}
   ```

2. **Examine Error Rates**:
   ```promql
   rate(http_requests_total{status=~"5.."}[5m])
   ```

3. **Analyze Response Times**:
   ```promql
   histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
   ```

4. **Compare with Baselines**:
   ```promql
   rate(http_requests_total[5m]) / rate(http_requests_total[5m] offset 24h)
   ```

### Capacity Planning

1. **Resource Trends**:
   ```promql
   predict_linear(disk_usage_bytes[1h], 3600 * 24 * 7)
   ```

2. **Growth Rates**:
   ```promql
   rate(http_requests_total[1h]) - rate(http_requests_total[1h] offset 24h)
   ```

### Performance Analysis

1. **Identify Bottlenecks**:
   ```promql
   topk(5, rate(http_requests_total[5m]))
   ```

2. **Resource Correlation**:
   ```promql
   cpu_usage_percent and on(instance) memory_usage_percent > 80
   ```

## Integration Examples

### With Grafana
- Use Prometheus as data source
- Create dashboards with PromQL queries
- Set up alerting with Prometheus rules

### With Alertmanager
- Configure notification channels
- Set up alert routing and grouping
- Implement alert silencing and inhibition

### With Service Discovery
- Configure automatic target discovery
- Use file-based, DNS, or cloud provider discovery
- Implement dynamic relabeling

## Troubleshooting

### Common Issues

1. **No Data**: Check target health and scrape configuration
2. **High Cardinality**: Review label usage and reduce dimensions
3. **Slow Queries**: Optimize PromQL and use recording rules
4. **Memory Usage**: Adjust retention and scrape intervals

### Query Debugging

1. **Use Explain**: Check query execution plan
2. **Test Incrementally**: Build queries step by step
3. **Check Metrics**: Verify metric existence and labels
4. **Monitor Performance**: Use query duration metrics

## Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [PromQL Tutorial](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Best Practices](https://prometheus.io/docs/practices/naming/)
- [Alerting Rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)