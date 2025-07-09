# Prometheus Recording and Alerting Rules

This directory contains Prometheus recording and alerting rules following SRE best practices for monitoring infrastructure, applications, service mesh, and SLOs.

## Files

- **recording-rules.yaml**: Contains recording rules for pre-calculating common metrics
- **alerting-rules.yaml**: Contains alerting rules for various scenarios
- **rules-configmap.yaml**: Kubernetes ConfigMap packaging all rules
- **prometheus-config-sample.yaml**: Sample Prometheus configuration and deployment

## Recording Rules

Recording rules pre-calculate frequently used expressions to improve query performance and simplify alert expressions.

### Categories:

1. **SLI Calculations** (`sli_calculations`)
   - Request rates (1m and 5m windows)
   - Error rates
   - Success rates
   - Request duration percentiles (p50, p90, p95, p99)

2. **Resource Utilization** (`resource_utilization`)
   - CPU usage by namespace, pod, and node
   - Memory usage aggregations
   - Network I/O rates
   - Disk I/O rates
   - Filesystem usage percentages

3. **Service Mesh Metrics** (`service_mesh_metrics`)
   - Mesh request rates
   - Success rates between services
   - Request duration percentiles
   - Istio-specific metrics

4. **SLO Metrics** (`slo_metrics`)
   - Error rates over multiple time windows (1h, 6h, 24h, 3d)
   - Availability calculations
   - Latency target compliance rates

## Alerting Rules

Alerts are organized by domain and follow a severity-based approach.

### Categories:

1. **Infrastructure Alerts** (`infrastructure_alerts`)
   - Node down/not ready
   - High CPU/memory usage
   - Low disk space
   - Kubernetes node issues

2. **Application Alerts** (`application_alerts`)
   - High error rates (warning at 5%, critical at 10%)
   - Slow response times (warning at 1s p95, critical at 5s)
   - Traffic anomalies (drops and spikes)

3. **Service Mesh Alerts** (`service_mesh_alerts`)
   - Circuit breaker status
   - High retry rates
   - Service mesh latency
   - Inter-service error rates

4. **SLO Alerts** (`slo_alerts`)
   - Multi-window multi-burn-rate alerts
   - Error budget burn rate monitoring
   - Latency SLO violations
   - Availability risk alerts

5. **Resource Alerts** (`resource_alerts`)
   - Container CPU/memory usage
   - OOM kill detection

## Alert Severity Levels

- **Critical**: Immediate action required, potential service impact
- **Warning**: Investigation needed, may escalate if not addressed

## Deployment

1. Create the monitoring namespace:
   ```bash
   kubectl create namespace monitoring
   ```

2. Apply the rules ConfigMap:
   ```bash
   kubectl apply -f rules-configmap.yaml
   ```

3. Deploy Prometheus with the sample configuration:
   ```bash
   kubectl apply -f prometheus-config-sample.yaml
   ```

## Customization

### Adjusting Thresholds

Edit the threshold values in the alerting rules based on your service requirements:

```yaml
- alert: HighErrorRate
  expr: service:error_rate > 0.05  # Adjust this threshold
```

### Adding New Rules

Add new recording or alerting rules to the appropriate group in the respective files.

### SLO Configuration

The current rules assume:
- 99.9% availability SLO (0.1% error budget)
- 1-second latency target for 95% of requests

Adjust these values in the `slo_alerts` group based on your actual SLOs.

## Integration with Alertmanager

Configure Alertmanager to route alerts based on labels:
- `team`: Routes to appropriate team (infrastructure, application)
- `severity`: Determines notification urgency
- `slo`: Identifies SLO-related alerts

## Best Practices

1. **Use Recording Rules**: Pre-calculate complex expressions
2. **Multi-Window Alerts**: Use multiple time windows for SLO alerts
3. **Actionable Alerts**: Include runbook URLs in annotations
4. **Proper Labels**: Use consistent labeling for routing and filtering
5. **Avoid Alert Fatigue**: Set appropriate thresholds and time windows

## Monitoring the Monitors

Remember to monitor Prometheus itself:
- Prometheus up/down status
- Rule evaluation failures
- Storage usage
- Query performance

## References

- [Google SRE Book - Alerting on SLOs](https://sre.google/workbook/alerting-on-slos/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/)
- [The USE Method](http://www.brendangregg.com/usemethod.html)
- [The RED Method](https://grafana.com/blog/2018/08/02/the-red-method-how-to-instrument-your-services/)