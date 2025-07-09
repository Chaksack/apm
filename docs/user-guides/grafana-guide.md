# Grafana User Guide

## Overview

Grafana is a visualization and monitoring platform that creates dashboards and graphs from various data sources. This guide covers dashboard creation, panel configuration, alerting, and data source management.

## Getting Started

### Accessing Grafana

1. **Web Interface**: Navigate to `http://localhost:3000` (default)
2. **Default Login**: admin/admin (change on first login)
3. **API**: Access via REST API at `http://localhost:3000/api/`

### Basic Navigation

- **Home**: Dashboard overview and shortcuts
- **Dashboards**: Browse and manage dashboards
- **Explore**: Ad-hoc data exploration
- **Alerting**: Alert rules and notifications
- **Configuration**: Data sources and settings

## Dashboard Creation

### Creating a New Dashboard

1. **Navigate**: Go to Dashboards → New Dashboard
2. **Add Panel**: Click "Add new panel"
3. **Configure**: Set up queries, visualization, and options
4. **Save**: Name your dashboard and add tags

### Dashboard Organization

```json
{
  "dashboard": {
    "title": "Service Monitoring",
    "tags": ["monitoring", "service", "production"],
    "time": {
      "from": "now-1h",
      "to": "now"
    },
    "refresh": "5s",
    "panels": []
  }
}
```

### Dashboard Best Practices

1. **Logical Grouping**: Group related metrics
2. **Consistent Timeframes**: Use standard time ranges
3. **Clear Titles**: Descriptive panel and dashboard names
4. **Templating**: Use variables for dynamic content
5. **Annotations**: Add context with annotations

### Row Organization

```json
{
  "type": "row",
  "title": "Application Metrics",
  "collapsed": false,
  "panels": [
    {
      "title": "Request Rate",
      "type": "graph"
    },
    {
      "title": "Error Rate",
      "type": "graph"
    }
  ]
}
```

## Panel Configuration

### Basic Panel Setup

1. **Query Tab**: Configure data source and queries
2. **Visualization**: Choose chart type and settings
3. **Field Options**: Format values and units
4. **Overrides**: Customize specific series
5. **Panel Options**: Set title, description, and links

### Common Panel Types

#### Time Series (Graph)
```json
{
  "type": "timeseries",
  "title": "HTTP Request Rate",
  "targets": [
    {
      "expr": "rate(http_requests_total[5m])",
      "legendFormat": "{{instance}}"
    }
  ],
  "fieldConfig": {
    "defaults": {
      "unit": "reqps",
      "min": 0
    }
  }
}
```

#### Stat Panel
```json
{
  "type": "stat",
  "title": "Total Requests",
  "targets": [
    {
      "expr": "sum(http_requests_total)"
    }
  ],
  "fieldConfig": {
    "defaults": {
      "unit": "short",
      "color": {
        "mode": "thresholds"
      },
      "thresholds": {
        "steps": [
          {"color": "green", "value": null},
          {"color": "yellow", "value": 1000},
          {"color": "red", "value": 5000}
        ]
      }
    }
  }
}
```

#### Gauge Panel
```json
{
  "type": "gauge",
  "title": "CPU Usage",
  "targets": [
    {
      "expr": "100 * (1 - rate(cpu_idle_seconds_total[5m]))"
    }
  ],
  "fieldConfig": {
    "defaults": {
      "unit": "percent",
      "min": 0,
      "max": 100,
      "thresholds": {
        "steps": [
          {"color": "green", "value": null},
          {"color": "yellow", "value": 60},
          {"color": "red", "value": 80}
        ]
      }
    }
  }
}
```

#### Table Panel
```json
{
  "type": "table",
  "title": "Service Overview",
  "targets": [
    {
      "expr": "sum by (service) (rate(http_requests_total[5m]))",
      "format": "table"
    }
  ],
  "fieldConfig": {
    "overrides": [
      {
        "matcher": {"id": "byName", "options": "Value"},
        "properties": [
          {"id": "unit", "value": "reqps"},
          {"id": "custom.displayMode", "value": "color-background"}
        ]
      }
    ]
  }
}
```

### Advanced Panel Features

#### Value Mappings
```json
{
  "fieldConfig": {
    "defaults": {
      "mappings": [
        {
          "type": "value",
          "value": "0",
          "text": "Down"
        },
        {
          "type": "value",
          "value": "1",
          "text": "Up"
        }
      ]
    }
  }
}
```

#### Field Overrides
```json
{
  "fieldConfig": {
    "overrides": [
      {
        "matcher": {"id": "byName", "options": "errors"},
        "properties": [
          {"id": "color", "value": {"mode": "fixed", "fixedColor": "red"}},
          {"id": "unit", "value": "short"}
        ]
      }
    ]
  }
}
```

## Templating and Variables

### Creating Variables

1. **Dashboard Settings**: Click gear icon → Variables
2. **Add Variable**: Click "New variable"
3. **Configure**: Set name, type, and query
4. **Use**: Reference with `$variable_name`

### Variable Types

#### Query Variable
```json
{
  "name": "instance",
  "type": "query",
  "query": "label_values(up, instance)",
  "refresh": "on_dashboard_load",
  "multi": true,
  "includeAll": true
}
```

#### Custom Variable
```json
{
  "name": "environment",
  "type": "custom",
  "options": [
    {"text": "Production", "value": "prod"},
    {"text": "Staging", "value": "stage"},
    {"text": "Development", "value": "dev"}
  ]
}
```

#### Interval Variable
```json
{
  "name": "interval",
  "type": "interval",
  "options": ["1m", "5m", "10m", "30m", "1h"]
}
```

### Using Variables in Queries

```promql
# In Prometheus queries
rate(http_requests_total{instance=~"$instance"}[$interval])

# In panel titles
HTTP Requests - $instance

# In annotations
avg by (instance) (rate(http_requests_total{instance=~"$instance"}[5m]))
```

## Data Source Management

### Adding Data Sources

1. **Configuration**: Go to Configuration → Data Sources
2. **Add Data Source**: Click "Add data source"
3. **Configure**: Enter connection details
4. **Test**: Verify connection works
5. **Save**: Store configuration

### Prometheus Data Source

```json
{
  "name": "Prometheus",
  "type": "prometheus",
  "url": "http://localhost:9090",
  "access": "proxy",
  "basicAuth": false,
  "jsonData": {
    "httpMethod": "POST",
    "timeInterval": "15s"
  }
}
```

### Common Data Source Types

#### Loki (Logs)
```json
{
  "name": "Loki",
  "type": "loki",
  "url": "http://localhost:3100",
  "jsonData": {
    "maxLines": 1000,
    "derivedFields": [
      {
        "name": "TraceID",
        "matcherRegex": "trace_id=(\\w+)",
        "url": "http://localhost:16686/trace/${__value.raw}"
      }
    ]
  }
}
```

#### Jaeger (Tracing)
```json
{
  "name": "Jaeger",
  "type": "jaeger",
  "url": "http://localhost:16686",
  "jsonData": {
    "tracesToLogs": {
      "datasourceUid": "loki-uid",
      "tags": ["service", "instance"]
    }
  }
}
```

#### InfluxDB
```json
{
  "name": "InfluxDB",
  "type": "influxdb",
  "url": "http://localhost:8086",
  "database": "monitoring",
  "user": "admin",
  "secureJsonData": {
    "password": "password"
  }
}
```

## Alerting Setup

### Alert Rules

1. **Create Alert**: Go to Alerting → Alert Rules
2. **Configure Query**: Set up data source and query
3. **Set Conditions**: Define alert conditions
4. **Add Labels**: Tag alerts for routing
5. **Configure Notifications**: Set up contact points

### Alert Rule Configuration

```json
{
  "title": "High CPU Usage",
  "condition": "A",
  "data": [
    {
      "refId": "A",
      "queryType": "",
      "model": {
        "expr": "100 * (1 - rate(cpu_idle_seconds_total[5m]))",
        "intervalMs": 1000,
        "maxDataPoints": 43200
      }
    }
  ],
  "conditionsLogic": "A > 80",
  "executionErrorState": "alerting",
  "noDataState": "no_data",
  "for": "5m",
  "annotations": {
    "description": "CPU usage is above 80% for more than 5 minutes",
    "summary": "High CPU usage detected"
  },
  "labels": {
    "severity": "warning",
    "team": "infrastructure"
  }
}
```

### Contact Points

#### Email
```json
{
  "name": "email-alerts",
  "type": "email",
  "settings": {
    "addresses": ["admin@example.com"],
    "subject": "Grafana Alert: {{ .GroupLabels.alertname }}",
    "message": "{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}"
  }
}
```

#### Slack
```json
{
  "name": "slack-alerts",
  "type": "slack",
  "settings": {
    "url": "https://hooks.slack.com/services/...",
    "channel": "#alerts",
    "username": "Grafana",
    "title": "{{ .GroupLabels.alertname }}",
    "text": "{{ range .Alerts }}{{ .Annotations.description }}{{ end }}"
  }
}
```

#### Webhook
```json
{
  "name": "webhook-alerts",
  "type": "webhook",
  "settings": {
    "url": "https://api.example.com/alerts",
    "httpMethod": "POST",
    "httpHeaders": {
      "Content-Type": "application/json",
      "Authorization": "Bearer token"
    }
  }
}
```

### Notification Policies

```json
{
  "receiver": "default",
  "group_by": ["alertname", "cluster", "service"],
  "group_wait": "10s",
  "group_interval": "10s",
  "repeat_interval": "1h",
  "routes": [
    {
      "match": {
        "severity": "critical"
      },
      "receiver": "critical-alerts",
      "group_wait": "0s",
      "repeat_interval": "5m"
    }
  ]
}
```

## Advanced Features

### Annotations

#### Query-based Annotations
```json
{
  "name": "Deployments",
  "datasource": "Prometheus",
  "expr": "increase(deployments_total[1m])",
  "iconColor": "blue",
  "textFormat": "Deployment: {{service}}"
}
```

#### Manual Annotations
- Add context to specific time periods
- Mark incidents, deployments, or changes
- Correlate events with metrics

### Transformations

#### Join by Field
```json
{
  "id": "joinByField",
  "options": {
    "byField": "instance",
    "mode": "outer"
  }
}
```

#### Add Field from Calculation
```json
{
  "id": "calculateField",
  "options": {
    "alias": "error_rate",
    "mode": "binary",
    "binary": {
      "left": "errors",
      "operation": "/",
      "right": "requests"
    }
  }
}
```

### Provisioning

#### Dashboard Provisioning
```yaml
apiVersion: 1
providers:
  - name: 'default'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    options:
      path: /var/lib/grafana/dashboards
```

#### Data Source Provisioning
```yaml
apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    url: http://localhost:9090
    access: proxy
    isDefault: true
```

## Performance Optimization

### Query Optimization

1. **Limit Time Range**: Use appropriate time windows
2. **Reduce Resolution**: Use lower resolution for overview panels
3. **Query Caching**: Enable query result caching
4. **Panel Limits**: Set max data points and series limits

### Dashboard Performance

1. **Reduce Panels**: Limit panels per dashboard
2. **Optimize Queries**: Use efficient PromQL expressions
3. **Template Variables**: Use variables efficiently
4. **Refresh Rates**: Set appropriate refresh intervals

## Best Practices

### Dashboard Design

1. **User-Centric**: Design for your audience
2. **Hierarchy**: Use rows and panels logically
3. **Consistency**: Maintain consistent styling
4. **Context**: Provide sufficient context
5. **Actionability**: Make dashboards actionable

### Alerting Strategy

1. **Signal vs Noise**: Alert on actionable issues
2. **Escalation**: Implement proper escalation paths
3. **Documentation**: Include runbooks and context
4. **Testing**: Regularly test alert rules
5. **Maintenance**: Review and update alerts

### Organization

1. **Folder Structure**: Organize dashboards logically
2. **Naming Convention**: Use consistent naming
3. **Tags**: Tag dashboards for discovery
4. **Permissions**: Set appropriate access controls
5. **Version Control**: Use dashboard versioning

## Common Workflows

### Creating a Service Dashboard

1. **Plan Layout**: Sketch dashboard structure
2. **Identify Metrics**: Choose relevant metrics
3. **Create Panels**: Build visualization panels
4. **Add Variables**: Implement template variables
5. **Configure Alerts**: Set up alerting rules
6. **Test and Iterate**: Validate and improve

### Troubleshooting Issues

1. **Check Data Sources**: Verify connectivity
2. **Validate Queries**: Test queries in Explore
3. **Review Logs**: Check Grafana logs
4. **Performance**: Monitor query performance
5. **Permissions**: Verify user permissions

### Dashboard Maintenance

1. **Regular Reviews**: Periodically review dashboards
2. **Update Queries**: Adapt to metric changes
3. **Clean Up**: Remove unused dashboards
4. **Optimize**: Improve performance
5. **Document**: Maintain documentation

## Integration Examples

### With Prometheus
- Use Prometheus as primary data source
- Implement recording rules for performance
- Set up alert rules in Prometheus

### With Loki
- Correlate metrics with logs
- Use derived fields for trace correlation
- Implement log-based alerting

### With Jaeger
- Link traces to metrics
- Use trace IDs in annotations
- Implement distributed tracing dashboards

## Troubleshooting

### Common Issues

1. **No Data**: Check data source configuration
2. **Slow Loading**: Optimize queries and reduce time range
3. **Alert Not Firing**: Verify alert rule configuration
4. **Permission Errors**: Check user roles and permissions

### Debug Tools

1. **Explore**: Test queries interactively
2. **Query Inspector**: Analyze query performance
3. **Network Tab**: Debug API calls
4. **Grafana Logs**: Check server logs

## Resources

- [Grafana Documentation](https://grafana.com/docs/)
- [Dashboard Best Practices](https://grafana.com/docs/grafana/latest/best-practices/)
- [Alerting Guide](https://grafana.com/docs/grafana/latest/alerting/)
- [Provisioning](https://grafana.com/docs/grafana/latest/administration/provisioning/)