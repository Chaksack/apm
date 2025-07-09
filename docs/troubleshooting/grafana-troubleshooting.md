# Grafana Troubleshooting Guide

## Dashboard Issues

### 1. Dashboard Not Loading

**Symptoms:**
- Blank dashboard
- "Dashboard not found" error
- Slow dashboard rendering

**Diagnostic Commands:**
```bash
# Check Grafana pod status
kubectl get pods -n monitoring | grep grafana

# Check Grafana logs
kubectl logs -n monitoring deployment/grafana | tail -50

# Check dashboard JSON
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://grafana:3000/api/dashboards/uid/$DASHBOARD_UID" | jq .

# Test dashboard import
curl -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer $GRAFANA_TOKEN" \
  -d @dashboard.json \
  "http://grafana:3000/api/dashboards/db"
```

**Solutions:**
- Check dashboard permissions
- Verify data source configuration
- Validate dashboard JSON syntax
- Clear browser cache
- Restart Grafana pod

### 2. Panel Query Errors

**Symptoms:**
- "No data" in panels
- Query execution errors
- Timeout errors

**Diagnostic Commands:**
```bash
# Test query directly in Prometheus
curl -s "http://prometheus:9090/api/v1/query?query=up" | jq .

# Check query performance
curl -s "http://prometheus:9090/api/v1/query?query=prometheus_engine_query_duration_seconds" | jq .

# Validate PromQL syntax
curl -s "http://prometheus:9090/api/v1/query?query=invalid_query" | jq .
```

**Solutions:**
- Fix PromQL syntax errors
- Adjust time range
- Check data source connectivity
- Optimize query performance
- Increase query timeout

### 3. Templating Issues

**Symptoms:**
- Variables not populating
- Template queries failing
- Dynamic dashboards broken

**Diagnostic Commands:**
```bash
# Check variable queries
curl -s "http://prometheus:9090/api/v1/label/__name__/values" | jq .

# Test label queries
curl -s "http://prometheus:9090/api/v1/query?query=label_values(up, instance)" | jq .

# Verify template variable configuration
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://grafana:3000/api/dashboards/uid/$DASHBOARD_UID" | jq '.dashboard.templating'
```

**Solutions:**
- Fix template query syntax
- Check variable dependencies
- Validate regex patterns
- Update variable refresh settings

## Data Source Problems

### 1. Data Source Connection Issues

**Symptoms:**
- "Data source not found" errors
- Connection timeouts
- Authentication failures

**Diagnostic Commands:**
```bash
# Test data source connectivity
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://grafana:3000/api/datasources/$DATASOURCE_ID" | jq .

# Test Prometheus connectivity from Grafana pod
kubectl exec -n monitoring deployment/grafana -- curl -s http://prometheus:9090/api/v1/query?query=up

# Check network policies
kubectl get networkpolicies -n monitoring

# Test DNS resolution
kubectl exec -n monitoring deployment/grafana -- nslookup prometheus
```

**Solutions:**
- Verify service endpoints
- Check network policies
- Update data source URL
- Fix authentication credentials
- Restart Grafana pod

### 2. Prometheus Data Source Configuration

**Correct configuration:**
```json
{
  "name": "Prometheus",
  "type": "prometheus",
  "url": "http://prometheus:9090",
  "access": "proxy",
  "basicAuth": false,
  "isDefault": true,
  "jsonData": {
    "timeInterval": "15s",
    "queryTimeout": "60s",
    "httpMethod": "POST"
  }
}
```

### 3. Loki Data Source Issues

**Symptoms:**
- No log data in explore
- LogQL query failures
- Slow log queries

**Diagnostic Commands:**
```bash
# Test Loki connectivity
curl -s "http://loki:3100/ready"

# Test LogQL query
curl -s -G "http://loki:3100/loki/api/v1/query" \
  --data-urlencode 'query={job="app"}' | jq .

# Check Loki labels
curl -s "http://loki:3100/loki/api/v1/labels" | jq .
```

**Solutions:**
- Verify Loki endpoint
- Check LogQL syntax
- Adjust time range
- Optimize log queries

## Performance Optimization

### 1. Slow Dashboard Loading

**Optimization strategies:**
- Reduce query complexity
- Use recording rules
- Implement caching
- Optimize time ranges
- Use appropriate refresh intervals

**Dashboard optimization example:**
```json
{
  "refresh": "30s",
  "time": {
    "from": "now-1h",
    "to": "now"
  },
  "panels": [
    {
      "type": "graph",
      "targets": [
        {
          "expr": "instance:cpu_usage:rate5m",
          "intervalFactor": 2,
          "step": 60
        }
      ]
    }
  ]
}
```

### 2. Memory Usage Optimization

**Diagnostic Commands:**
```bash
# Check Grafana memory usage
kubectl top pods -n monitoring | grep grafana

# Check memory limits
kubectl describe pod -n monitoring -l app=grafana | grep -A 5 -B 5 memory
```

**Optimization:**
```yaml
spec:
  containers:
  - name: grafana
    resources:
      requests:
        memory: "512Mi"
        cpu: "250m"
      limits:
        memory: "1Gi"
        cpu: "500m"
    env:
    - name: GF_RENDERING_SERVER_URL
      value: "http://grafana-image-renderer:8081/render"
```

### 3. Query Performance Tuning

**Best practices:**
- Use appropriate aggregation functions
- Limit query range
- Use recording rules for complex queries
- Implement proper caching

**Example optimized query:**
```promql
# Instead of:
sum(rate(http_requests_total[5m])) by (method, status)

# Use:
instance:http_requests:rate5m
```

## Plugin Troubleshooting

### 1. Plugin Installation Issues

**Symptoms:**
- Plugin not loading
- Installation failures
- Missing plugin features

**Diagnostic Commands:**
```bash
# List installed plugins
kubectl exec -n monitoring deployment/grafana -- grafana-cli plugins ls

# Check plugin status
kubectl logs -n monitoring deployment/grafana | grep -i plugin

# Install plugin manually
kubectl exec -n monitoring deployment/grafana -- grafana-cli plugins install grafana-piechart-panel
```

**Solutions:**
- Check plugin compatibility
- Verify internet connectivity
- Update plugin versions
- Restart Grafana after installation

### 2. Panel Plugin Problems

**Common issues:**
- Plugin not rendering
- Configuration errors
- Data format issues

**Solutions:**
- Check plugin documentation
- Verify data source compatibility
- Update plugin configuration
- Check browser console for errors

## Authentication and Authorization

### 1. Login Issues

**Symptoms:**
- Cannot login to Grafana
- Authentication failures
- Permission denied errors

**Diagnostic Commands:**
```bash
# Check Grafana authentication config
kubectl exec -n monitoring deployment/grafana -- cat /etc/grafana/grafana.ini | grep -A 10 auth

# Check user database
kubectl exec -n monitoring deployment/grafana -- sqlite3 /var/lib/grafana/grafana.db "SELECT * FROM user;"

# Reset admin password
kubectl exec -n monitoring deployment/grafana -- grafana-cli admin reset-admin-password newpassword
```

**Solutions:**
- Reset admin password
- Check authentication configuration
- Verify LDAP/OAuth settings
- Check user permissions

### 2. RBAC Issues

**Symptoms:**
- Users cannot access dashboards
- Permission errors
- Role assignment problems

**Diagnostic Commands:**
```bash
# Check user roles
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://grafana:3000/api/org/users" | jq .

# Check dashboard permissions
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://grafana:3000/api/dashboards/uid/$DASHBOARD_UID/permissions" | jq .
```

**Solutions:**
- Update user roles
- Fix dashboard permissions
- Check organization settings
- Verify team assignments

## Alerting Issues

### 1. Alerts Not Firing

**Symptoms:**
- No alert notifications
- Alerts stuck in pending state
- Notification channel failures

**Diagnostic Commands:**
```bash
# Check alert rules
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://grafana:3000/api/alerts" | jq .

# Check notification channels
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://grafana:3000/api/alert-notifications" | jq .

# Test notification channel
curl -X POST -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://grafana:3000/api/alert-notifications/test" \
  -d '{"id": 1}'
```

**Solutions:**
- Check alert conditions
- Verify notification channels
- Test webhook endpoints
- Check alert evaluation frequency

### 2. Alert Manager Integration

**Configuration example:**
```json
{
  "name": "AlertManager",
  "type": "alertmanager",
  "settings": {
    "url": "http://alertmanager:9093",
    "basicAuth": false
  }
}
```

## Emergency Procedures

### 1. Grafana Recovery

**Complete reset:**
```bash
# Backup current data
kubectl exec -n monitoring deployment/grafana -- tar -czf /tmp/grafana-backup.tar.gz /var/lib/grafana

# Delete and recreate pod
kubectl delete pod -n monitoring -l app=grafana
kubectl rollout restart deployment/grafana -n monitoring

# Verify recovery
kubectl get pods -n monitoring | grep grafana
```

### 2. Dashboard Recovery

**Backup and restore:**
```bash
# Export dashboard
curl -s -H "Authorization: Bearer $GRAFANA_TOKEN" \
  "http://grafana:3000/api/dashboards/uid/$DASHBOARD_UID" > dashboard-backup.json

# Import dashboard
curl -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer $GRAFANA_TOKEN" \
  -d @dashboard-backup.json \
  "http://grafana:3000/api/dashboards/db"
```

## Common Error Messages and Solutions

| Error Message | Cause | Solution |
|---------------|-------|----------|
| "Dashboard not found" | Missing dashboard | Check dashboard UID and permissions |
| "Data source not found" | Incorrect data source | Verify data source configuration |
| "Query timeout" | Slow query | Optimize query or increase timeout |
| "Template variable error" | Invalid template query | Fix template query syntax |
| "Panel plugin not found" | Missing plugin | Install required plugin |
| "Authentication failed" | Invalid credentials | Check authentication configuration |
| "Permission denied" | Insufficient permissions | Update user roles and permissions |

## Monitoring Grafana Health

### Key Metrics to Monitor

```bash
# Grafana uptime
up{job="grafana"}

# Dashboard render time
grafana_dashboard_loading_duration_seconds

# Query performance
grafana_datasource_request_duration_seconds

# Active users
grafana_stat_active_users

# Memory usage
process_resident_memory_bytes{job="grafana"}
```

### Health Check Alerts

```yaml
groups:
  - name: grafana-health
    rules:
    - alert: GrafanaDown
      expr: up{job="grafana"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Grafana is down"
        description: "Grafana has been down for more than 5 minutes"

    - alert: GrafanaHighMemory
      expr: process_resident_memory_bytes{job="grafana"} > 1e9
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "Grafana high memory usage"
        description: "Grafana is using more than 1GB of memory"
```