# CloudWatch Integration for APM Monitoring

This document provides comprehensive documentation for the CloudWatch integration functionality implemented in the APM tool. The CloudWatch integration provides monitoring, alerting, logging, and analytics capabilities for APM tools including Prometheus, Grafana, Jaeger, and Loki.

## Table of Contents

1. [Overview](#overview)
2. [Features](#features)
3. [Getting Started](#getting-started)
4. [Dashboard Management](#dashboard-management)
5. [Alarm Management](#alarm-management)
6. [Log Management](#log-management)
7. [CloudWatch Insights](#cloudwatch-insights)
8. [Events and SNS](#events-and-sns)
9. [APM Tool Integration](#apm-tool-integration)
10. [Performance Optimization](#performance-optimization)
11. [Error Handling](#error-handling)
12. [Examples](#examples)
13. [Best Practices](#best-practices)
14. [Troubleshooting](#troubleshooting)

## Overview

The CloudWatch integration is a comprehensive solution for monitoring APM (Application Performance Monitoring) infrastructure and applications. It provides:

- **Dashboard Management**: Create and manage APM-specific dashboards
- **Alarm Management**: Set up intelligent alerting for APM metrics
- **Log Management**: Centralized logging with retention policies
- **Insights Analytics**: Advanced log analysis and querying
- **Event Management**: Automated response to infrastructure events
- **SNS Integration**: Multi-channel notification system
- **APM Tool Integration**: Native integration with Prometheus, Grafana, Jaeger, and Loki

## Features

### Dashboard Management
- ✅ Create APM-specific dashboards with pre-built templates
- ✅ Infrastructure, application, service mesh, logs, and tracing dashboards
- ✅ Multi-region dashboard deployment
- ✅ Custom widget creation and management
- ✅ Dashboard sharing and access control

### Alarm Management
- ✅ APM-specific alarm templates (CPU, memory, disk, service health)
- ✅ Multi-threshold and composite alarms
- ✅ Alarm state management and history
- ✅ Integration with SNS for notifications
- ✅ Alarm suppression and escalation policies

### Log Management
- ✅ Centralized log collection from APM tools
- ✅ Log group creation with retention policies
- ✅ Metric filters for log-based alerting
- ✅ Real-time log streaming
- ✅ Log encryption and access control

### CloudWatch Insights
- ✅ Advanced log analysis and querying
- ✅ Pre-built APM queries (error analysis, performance metrics)
- ✅ Real-time and historical log analysis
- ✅ Custom query creation and management
- ✅ Query result visualization

### Events and SNS
- ✅ Automated event processing and routing
- ✅ Multi-channel notifications (email, SMS, webhook)
- ✅ Event pattern matching and filtering
- ✅ Integration with external systems
- ✅ Event history and audit trail

### Performance Features
- ✅ Intelligent caching with TTL
- ✅ Batch processing for bulk operations
- ✅ Connection pooling and optimization
- ✅ Multi-region operation support
- ✅ Health checks and monitoring

## Getting Started

### Prerequisites

1. **AWS CLI Installation**: Ensure AWS CLI v2.x is installed and configured
2. **AWS Credentials**: Configure AWS credentials with appropriate CloudWatch permissions
3. **Go Environment**: Go 1.23+ required

### Installation

```go
import "github.com/chaksack/apm/pkg/cloud"
```

### Basic Setup

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/chaksack/apm/pkg/cloud"
)

func main() {
    // Create AWS provider
    provider, err := cloud.NewAWSProvider(nil)
    if err != nil {
        log.Fatalf("Failed to create AWS provider: %v", err)
    }

    // Get CloudWatch manager
    manager := provider.GetCloudWatchManager()
    if manager == nil {
        log.Fatal("CloudWatch manager not available")
    }

    // Initialize context
    ctx := context.Background()

    // Example: Create a dashboard
    dashboardConfig := &cloud.DashboardConfig{
        Name:        "APM-Infrastructure-Dashboard",
        Description: "APM infrastructure monitoring",
        Region:      "us-east-1",
        Type:        "infrastructure",
        Environment: "production",
        Tags: map[string]string{
            "Environment": "production",
            "Purpose":     "apm-monitoring",
        },
    }

    dashboard, err := manager.CreateDashboard(ctx, dashboardConfig)
    if err != nil {
        log.Printf("Error creating dashboard: %v", err)
    } else {
        log.Printf("Created dashboard: %s", dashboard.DashboardName)
    }
}
```

## Dashboard Management

### Create APM Dashboard

```go
// Create infrastructure monitoring dashboard
dashboard, err := manager.CreateAPMDashboard(ctx, "APM-Infrastructure", "us-east-1", "infrastructure", "production")
if err != nil {
    log.Printf("Error: %v", err)
} else {
    log.Printf("Dashboard created: %s", dashboard.DashboardName)
}
```

### Available Dashboard Types

1. **Infrastructure**: CPU, memory, disk, network metrics
2. **Application**: Request rates, response times, error rates
3. **Service Mesh**: Istio/Envoy metrics, service topology
4. **Logs**: Log volume, error rates, pattern analysis
5. **Tracing**: Trace latency, service dependencies, error traces
6. **Cost Optimization**: Resource utilization, cost trends

### Custom Dashboard Creation

```go
dashboardConfig := &cloud.DashboardConfig{
    Name:        "Custom-APM-Dashboard",
    Description: "Custom APM monitoring dashboard",
    Region:      "us-east-1",
    Type:        "custom",
    Environment: "production",
    Widgets: []*cloud.DashboardWidget{
        {
            Type:   "metric",
            Title:  "Application Response Time",
            Width:  12,
            Height: 6,
            Properties: map[string]interface{}{
                "metrics": [][]interface{}{
                    {"APM/Application", "ResponseTime", "Service", "api-service"},
                },
                "period": 300,
                "stat":   "Average",
                "region": "us-east-1",
            },
        },
        {
            Type:   "log",
            Title:  "Error Log Analysis",
            Width:  12,
            Height: 6,
            Properties: map[string]interface{}{
                "query": "SOURCE '/aws/apm/logs' | fields @timestamp, @message | filter @message like /ERROR/",
                "region": "us-east-1",
            },
        },
    },
    Tags: map[string]string{
        "Environment": "production",
        "Custom":      "true",
    },
}

dashboard, err := manager.CreateDashboard(ctx, dashboardConfig)
```

### List and Manage Dashboards

```go
// List all APM dashboards
dashboards, err := manager.ListDashboards(ctx, "us-east-1", "APM-")
if err != nil {
    log.Printf("Error listing dashboards: %v", err)
} else {
    for _, dashboard := range dashboards {
        log.Printf("Dashboard: %s, Modified: %s", 
            dashboard.DashboardName, 
            dashboard.LastModified.Format("2006-01-02 15:04:05"))
    }
}

// Get specific dashboard
dashboard, err := manager.GetDashboard(ctx, "APM-Infrastructure", "us-east-1")
if err != nil {
    log.Printf("Error getting dashboard: %v", err)
} else {
    log.Printf("Dashboard widgets: %d", len(dashboard.Widgets))
}

// Update dashboard
dashboard.Description = "Updated APM infrastructure monitoring"
updatedDashboard, err := manager.UpdateDashboard(ctx, dashboard)

// Delete dashboard
err = manager.DeleteDashboard(ctx, "APM-Infrastructure", "us-east-1")
```

## Alarm Management

### Create APM Alarms

```go
// Create high CPU alarm
alarmConfig := &cloud.AlarmConfig{
    Name:        "APM-High-CPU-Utilization",
    Description: "Alert when CPU utilization exceeds 80%",
    Region:      "us-east-1",
    MetricName:  "CPUUtilization",
    Namespace:   "AWS/EC2",
    Statistic:   "Average",
    Period:      300,
    EvaluationPeriods: 2,
    Threshold:   80.0,
    ComparisonOperator: "GreaterThanThreshold",
    TreatMissingData: "notBreaching",
    Dimensions: map[string]string{
        "InstanceId": "i-1234567890abcdef0",
    },
    Tags: map[string]string{
        "Environment": "production",
        "Service":     "apm",
    },
    Actions: &cloud.AlarmActions{
        OKActions: []string{
            "arn:aws:sns:us-east-1:123456789012:apm-ok-notifications",
        },
        AlarmActions: []string{
            "arn:aws:sns:us-east-1:123456789012:apm-alarm-notifications",
        },
    },
}

alarm, err := manager.CreateAlarm(ctx, alarmConfig)
```

### APM-Specific Alarm Templates

```go
// Available alarm types
alarmTypes := []string{
    "high-cpu",           // CPU utilization > 80%
    "high-memory",        // Memory utilization > 85%
    "low-disk-space",     // Disk space < 20%
    "service-down",       // Service health check failure
    "high-error-rate",    // Error rate > 5%
    "slow-response-time", // Response time > 2s
    "prometheus-down",    // Prometheus service unavailable
    "grafana-down",       // Grafana service unavailable
    "jaeger-down",        // Jaeger service unavailable
    "loki-down",          // Loki service unavailable
}

// Create all APM alarms
for _, alarmType := range alarmTypes {
    alarm, err := manager.CreateAPMAlarm(ctx, 
        fmt.Sprintf("APM-%s", alarmType), 
        "us-east-1", 
        alarmType, 
        "production")
    if err != nil {
        log.Printf("Error creating %s alarm: %v", alarmType, err)
    } else {
        log.Printf("Created %s alarm: %s", alarmType, alarm.AlarmName)
    }
}
```

### Composite Alarms

```go
compositeConfig := &cloud.CompositeAlarmConfig{
    Name:        "APM-Service-Health",
    Description: "Overall APM service health",
    Region:      "us-east-1",
    AlarmRule:   "ALARM(APM-prometheus-down) OR ALARM(APM-grafana-down) OR ALARM(APM-jaeger-down) OR ALARM(APM-loki-down)",
    Actions: &cloud.AlarmActions{
        AlarmActions: []string{
            "arn:aws:sns:us-east-1:123456789012:apm-critical-alerts",
        },
    },
    Tags: map[string]string{
        "Environment": "production",
        "Type":        "composite",
    },
}

compositeAlarm, err := manager.CreateCompositeAlarm(ctx, compositeConfig)
```

### Alarm State Management

```go
// Get alarm state
alarmState, err := manager.GetAlarmState(ctx, "APM-High-CPU-Utilization", "us-east-1")
if err != nil {
    log.Printf("Error getting alarm state: %v", err)
} else {
    log.Printf("Alarm state: %s, Reason: %s", alarmState.StateValue, alarmState.StateReason)
}

// Get alarm history
history, err := manager.GetAlarmHistory(ctx, "APM-High-CPU-Utilization", "us-east-1")
if err != nil {
    log.Printf("Error getting alarm history: %v", err)
} else {
    for _, event := range history.AlarmHistoryItems {
        log.Printf("Event: %s, Time: %s", event.HistorySummary, event.Timestamp)
    }
}
```

## Log Management

### Create Log Groups

```go
// Create APM log group
logGroupConfig := &cloud.LogGroupConfig{
    Name:              "/aws/apm/application-logs",
    Region:            "us-east-1",
    RetentionInDays:   30,
    KmsKeyId:          "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
    Tags: map[string]string{
        "Environment": "production",
        "Service":     "apm",
    },
}

logGroup, err := manager.CreateLogGroup(ctx, logGroupConfig)
```

### Create APM Log Groups

```go
// Create all APM tool log groups
apmTools := []string{"prometheus", "grafana", "jaeger", "loki"}
for _, tool := range apmTools {
    logGroup, err := manager.CreateAPMLogGroup(ctx, 
        fmt.Sprintf("/aws/apm/%s", tool), 
        "us-east-1", 
        30, // 30 days retention
        "production")
    if err != nil {
        log.Printf("Error creating %s log group: %v", tool, err)
    } else {
        log.Printf("Created %s log group: %s", tool, logGroup.LogGroupName)
    }
}
```

### Metric Filters

```go
// Create error count metric filter
metricFilterConfig := &cloud.MetricFilterConfig{
    FilterName:         "APM-Error-Count",
    LogGroupName:       "/aws/apm/application-logs",
    FilterPattern:      "[timestamp, request_id, level=\"ERROR\", ...]",
    MetricTransformations: []*cloud.MetricTransformation{
        {
            MetricName:      "ErrorCount",
            MetricNamespace: "APM/Application",
            MetricValue:     "1",
            DefaultValue:    0,
        },
    },
}

metricFilter, err := manager.CreateMetricFilter(ctx, metricFilterConfig)
```

### Log Event Publishing

```go
// Publish log events
logEvents := []*cloud.LogEvent{
    {
        Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
        Message:   fmt.Sprintf("[INFO] %s APM service started", time.Now().Format("2006-01-02 15:04:05")),
    },
    {
        Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
        Message:   fmt.Sprintf("[ERROR] %s Database connection failed", time.Now().Format("2006-01-02 15:04:05")),
    },
}

err = manager.PutLogEvents(ctx, "/aws/apm/application-logs", "application-stream", logEvents)
```

### Log Stream Management

```go
// Create log stream
logStream, err := manager.CreateLogStream(ctx, "/aws/apm/application-logs", "new-application-stream")

// List log streams
streams, err := manager.ListLogStreams(ctx, "/aws/apm/application-logs", "us-east-1")

// Get log events
events, err := manager.GetLogEvents(ctx, "/aws/apm/application-logs", "application-stream", 
    &cloud.GetLogEventsInput{
        StartTime: time.Now().Add(-1 * time.Hour),
        EndTime:   time.Now(),
        Limit:     100,
    })
```

## CloudWatch Insights

### Execute Insights Queries

```go
// Error analysis query
queryConfig := &cloud.InsightsQueryConfig{
    LogGroupNames: []string{"/aws/apm/application-logs"},
    StartTime:     time.Now().Add(-24 * time.Hour),
    EndTime:       time.Now(),
    QueryString:   "fields @timestamp, @message | filter @message like /ERROR/ | stats count() by bin(5m)",
    Region:        "us-east-1",
}

query, err := manager.StartInsightsQuery(ctx, queryConfig)
if err != nil {
    log.Printf("Error starting query: %v", err)
} else {
    log.Printf("Query started: %s", query.QueryId)
}

// Get query results
results, err := manager.GetInsightsQueryResults(ctx, query.QueryId)
if err != nil {
    log.Printf("Error getting results: %v", err)
} else {
    for _, result := range results.Results {
        log.Printf("Result: %+v", result)
    }
}
```

### Pre-built APM Queries

```go
// Execute APM-specific queries
queries := map[string]string{
    "error-analysis": `
        fields @timestamp, @message 
        | filter @message like /ERROR/ 
        | stats count() by bin(5m)
        | sort @timestamp desc
    `,
    "performance-metrics": `
        fields @timestamp, @message 
        | filter @message like /METRIC/ 
        | parse @message /latency=(?<latency>\\d+)/
        | stats avg(latency), max(latency), min(latency) by bin(5m)
    `,
    "request-patterns": `
        fields @timestamp, @message 
        | filter @message like /REQUEST/ 
        | parse @message /method=(?<method>\\w+)/
        | stats count() by method, bin(1h)
    `,
    "service-health": `
        fields @timestamp, @message 
        | filter @message like /HEALTH/ 
        | parse @message /status=(?<status>\\w+)/
        | stats count() by status, bin(5m)
    `,
}

for queryName, queryString := range queries {
    result, err := manager.ExecuteAPMInsightsQuery(ctx, queryName, 
        []string{"/aws/apm/application-logs"}, 
        queryString, 
        "us-east-1")
    if err != nil {
        log.Printf("Error executing %s query: %v", queryName, err)
    } else {
        log.Printf("Executed %s query: %s", queryName, result.QueryId)
    }
}
```

### Query Management

```go
// List running queries
queries, err := manager.ListInsightsQueries(ctx, "us-east-1")

// Stop query
err = manager.StopInsightsQuery(ctx, "query-id")

// Get query statistics
stats, err := manager.GetInsightsQueryStats(ctx, "query-id")
```

## Events and SNS

### SNS Topic Management

```go
// Create SNS topic
topicConfig := &cloud.SNSTopicConfig{
    Name:        "apm-notifications",
    DisplayName: "APM Notifications",
    Region:      "us-east-1",
    Policy: `{
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Principal": {"Service": "cloudwatch.amazonaws.com"},
                "Action": "SNS:Publish",
                "Resource": "*"
            }
        ]
    }`,
    Tags: map[string]string{
        "Environment": "production",
        "Service":     "apm",
    },
}

topic, err := manager.CreateSNSTopic(ctx, topicConfig)
```

### SNS Subscriptions

```go
// Email subscription
emailConfig := &cloud.SNSSubscriptionConfig{
    TopicArn: topic.TopicArn,
    Protocol: "email",
    Endpoint: "apm-admin@example.com",
    Attributes: map[string]string{
        "FilterPolicy": `{"severity": ["HIGH", "CRITICAL"]}`,
    },
}

emailSub, err := manager.CreateSNSSubscription(ctx, emailConfig)

// Slack webhook subscription
slackConfig := &cloud.SNSSubscriptionConfig{
    TopicArn: topic.TopicArn,
    Protocol: "https",
    Endpoint: "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK",
    Attributes: map[string]string{
        "FilterPolicy": `{"source": ["apm"]}`,
    },
}

slackSub, err := manager.CreateSNSSubscription(ctx, slackConfig)
```

### Event Rules

```go
// Create event rule for EC2 state changes
eventRuleConfig := &cloud.EventRuleConfig{
    Name:        "apm-instance-state-change",
    Description: "Monitor EC2 instance state changes for APM",
    EventPattern: map[string]interface{}{
        "source":        []string{"aws.ec2"},
        "detail-type":   []string{"EC2 Instance State-change Notification"},
        "detail": map[string]interface{}{
            "state": []string{"running", "stopped", "terminated"},
            "instance-id": []string{"i-1234567890abcdef0"},
        },
    },
    State:  "ENABLED",
    Region: "us-east-1",
    Tags: map[string]string{
        "Environment": "production",
        "Service":     "apm",
    },
    Targets: []*cloud.EventTarget{
        {
            Id:      "1",
            Arn:     topic.TopicArn,
            RoleArn: "arn:aws:iam::123456789012:role/CloudWatchEventsRole",
            InputTransformer: &cloud.InputTransformer{
                InputPathsMap: map[string]string{
                    "instance": "$.detail.instance-id",
                    "state":    "$.detail.state",
                },
                InputTemplate: `{"instance": "<instance>", "state": "<state>", "source": "apm"}`,
            },
        },
    },
}

eventRule, err := manager.CreateEventRule(ctx, eventRuleConfig)
```

## APM Tool Integration

### Comprehensive APM Setup

```go
// Create complete APM monitoring setup
apmConfig := &cloud.APMMonitoringConfig{
    Environment: "production",
    Region:      "us-east-1",
    Tools: []string{"prometheus", "grafana", "jaeger", "loki"},
    Dashboards: map[string]string{
        "infrastructure": "APM-Infrastructure-Dashboard",
        "application":    "APM-Application-Dashboard",
        "service-mesh":   "APM-ServiceMesh-Dashboard",
        "logs":           "APM-Logs-Dashboard",
        "tracing":        "APM-Tracing-Dashboard",
    },
    Alarms: map[string]string{
        "high-cpu":           "APM-High-CPU-Utilization",
        "high-memory":        "APM-High-Memory-Utilization",
        "disk-space":         "APM-Low-Disk-Space",
        "service-down":       "APM-Service-Down",
        "high-error-rate":    "APM-High-Error-Rate",
        "slow-response-time": "APM-Slow-Response-Time",
    },
    LogGroups: []string{
        "/aws/apm/prometheus",
        "/aws/apm/grafana",
        "/aws/apm/jaeger",
        "/aws/apm/loki",
    },
    MetricFilters: map[string]string{
        "error-filter":       "APM-Error-Count",
        "performance-filter": "APM-Performance-Metrics",
        "security-filter":    "APM-Security-Events",
    },
    SNSTopics: []string{
        "apm-critical-alerts",
        "apm-warning-alerts",
        "apm-info-notifications",
    },
    EventRules: []string{
        "apm-instance-state-change",
        "apm-autoscaling-events",
        "apm-deployment-events",
    },
    Tags: map[string]string{
        "Environment": "production",
        "Service":     "apm",
        "Team":        "platform",
    },
}

setup, err := manager.CreateAPMMonitoringSetup(ctx, apmConfig)
```

### Individual Tool Integration

#### Prometheus Integration

```go
prometheusConfig := &cloud.PrometheusIntegrationConfig{
    MetricsEndpoint: "http://prometheus:9090/metrics",
    ScrapeInterval:  "15s",
    CustomMetrics: []string{
        "apm_request_duration_seconds",
        "apm_request_total",
        "apm_error_rate",
        "apm_cpu_usage_percent",
        "apm_memory_usage_bytes",
    },
    AlertmanagerURL: "http://alertmanager:9093",
    Rules: []string{
        "apm-high-error-rate",
        "apm-slow-response-time",
        "apm-high-cpu-usage",
    },
    Targets: []string{
        "api-service:8080",
        "frontend-service:3000",
        "database-service:5432",
    },
}

err = manager.ConfigurePrometheusIntegration(ctx, "us-east-1", prometheusConfig)
```

#### Grafana Integration

```go
grafanaConfig := &cloud.GrafanaIntegrationConfig{
    DashboardURL: "http://grafana:3000",
    APIKey:       "grafana-api-key",
    DashboardIDs: []string{
        "infrastructure-dashboard",
        "application-dashboard",
        "service-mesh-dashboard",
    },
    Datasources: []string{
        "prometheus",
        "loki",
        "jaeger",
    },
    Folders: []string{
        "APM Monitoring",
        "Infrastructure",
        "Applications",
    },
    Alerts: []string{
        "high-error-rate",
        "slow-response-time",
        "service-down",
    },
}

err = manager.ConfigureGrafanaIntegration(ctx, "us-east-1", grafanaConfig)
```

#### Jaeger Integration

```go
jaegerConfig := &cloud.JaegerIntegrationConfig{
    CollectorEndpoint: "http://jaeger-collector:14268/api/traces",
    QueryEndpoint:     "http://jaeger-query:16686",
    SamplingRate:      0.1,
    Services: []string{
        "apm-frontend",
        "apm-backend",
        "apm-database",
        "apm-cache",
    },
    Operations: []string{
        "http-request",
        "database-query",
        "cache-lookup",
    },
    Tags: map[string]string{
        "environment": "production",
        "version":     "1.0.0",
    },
}

err = manager.ConfigureJaegerIntegration(ctx, "us-east-1", jaegerConfig)
```

#### Loki Integration

```go
lokiConfig := &cloud.LokiIntegrationConfig{
    PushURL:     "http://loki:3100/loki/api/v1/push",
    QueryURL:    "http://loki:3100/loki/api/v1/query",
    TenantID:    "apm-tenant",
    LogLabels: map[string]string{
        "environment": "production",
        "service":     "apm",
        "team":        "platform",
    },
    RetentionPeriod: "30d",
    Streams: []string{
        "application-logs",
        "access-logs",
        "error-logs",
    },
    Alerting: map[string]string{
        "high-error-rate": "rate(({service=\"apm\"} |= \"ERROR\")[5m]) > 0.1",
        "no-logs":         "absent_over_time(({service=\"apm\"})[5m])",
    },
}

err = manager.ConfigureLokiIntegration(ctx, "us-east-1", lokiConfig)
```

## Performance Optimization

### Caching

```go
// Configure cache warming
warmupConfig := &cloud.CloudWatchCacheWarmupConfig{
    Region: "us-east-1",
    Dashboards: []string{
        "APM-Infrastructure-Dashboard",
        "APM-Application-Dashboard",
    },
    Alarms: []string{
        "APM-High-CPU-Utilization",
        "APM-Service-Down",
    },
    LogGroups: []string{
        "/aws/apm/prometheus",
        "/aws/apm/grafana",
    },
    MetricFilters: []string{
        "APM-Error-Count",
        "APM-Performance-Metrics",
    },
    SNSTopics: []string{
        "apm-critical-alerts",
        "apm-warning-alerts",
    },
    EventRules: []string{
        "apm-instance-state-change",
    },
}

err = manager.WarmupCache(ctx, warmupConfig)

// Get cache statistics
stats := manager.GetCacheStats()
log.Printf("Cache hit rate: %.2f%%", stats.HitRate)
log.Printf("Cache size: %d entries", stats.Size)
```

### Batch Operations

```go
// Batch dashboard creation
dashboardConfigs := []*cloud.DashboardConfig{
    {
        Name:        "APM-Infrastructure-Dashboard",
        Region:      "us-east-1",
        Type:        "infrastructure",
        Environment: "production",
    },
    {
        Name:        "APM-Application-Dashboard",
        Region:      "us-east-1",
        Type:        "application",
        Environment: "production",
    },
}

dashboards, err := manager.CreateDashboardsBatch(ctx, dashboardConfigs)

// Batch alarm creation
alarmConfigs := []*cloud.AlarmConfig{
    {
        Name:        "APM-High-CPU-Utilization",
        Region:      "us-east-1",
        MetricName:  "CPUUtilization",
        Threshold:   80.0,
    },
    {
        Name:        "APM-High-Memory-Utilization",
        Region:      "us-east-1",
        MetricName:  "MemoryUtilization",
        Threshold:   85.0,
    },
}

alarms, err := manager.CreateAlarmsBatch(ctx, alarmConfigs)
```

### Connection Pooling

```go
// Configure connection pool
poolConfig := &cloud.CloudWatchConnectionPoolConfig{
    MaxConnections:     10,
    IdleTimeout:        30 * time.Second,
    ConnectionTimeout:  5 * time.Second,
    RetryAttempts:      3,
    RetryDelay:         100 * time.Millisecond,
}

manager.ConfigureConnectionPool(poolConfig)
```

## Error Handling

### Error Classification

```go
// Handle CloudWatch errors
dashboard, err := manager.CreateDashboard(ctx, dashboardConfig)
if err != nil {
    if cloudErr, ok := err.(*cloud.CloudError); ok {
        switch cloudErr.Code {
        case "DASHBOARD_ALREADY_EXISTS":
            log.Printf("Dashboard already exists: %s", cloudErr.Message)
        case "INSUFFICIENT_PERMISSIONS":
            log.Printf("Insufficient permissions: %s", cloudErr.Message)
        case "RATE_LIMIT_EXCEEDED":
            log.Printf("Rate limit exceeded, retrying...")
            // Implement retry logic
        case "INVALID_PARAMETER":
            log.Printf("Invalid parameter: %s", cloudErr.Message)
        default:
            log.Printf("Unknown error: %s", cloudErr.Message)
        }
    } else {
        log.Printf("Non-CloudWatch error: %v", err)
    }
}
```

### Retry Logic

```go
// Automatic retry with exponential backoff
retryConfig := &cloud.RetryConfig{
    MaxAttempts:   3,
    InitialDelay:  100 * time.Millisecond,
    MaxDelay:      5 * time.Second,
    BackoffFactor: 2.0,
}

dashboard, err := manager.CreateDashboardWithRetry(ctx, dashboardConfig, retryConfig)
```

### Circuit Breaker

```go
// Configure circuit breaker
circuitConfig := &cloud.CircuitBreakerConfig{
    FailureThreshold:  5,
    RecoveryTimeout:   30 * time.Second,
    MonitoringPeriod:  1 * time.Minute,
}

manager.ConfigureCircuitBreaker(circuitConfig)
```

## Examples

### Complete APM Monitoring Setup

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/chaksack/apm/pkg/cloud"
)

func main() {
    // Initialize AWS provider
    provider, err := cloud.NewAWSProvider(nil)
    if err != nil {
        log.Fatalf("Failed to create AWS provider: %v", err)
    }

    manager := provider.GetCloudWatchManager()
    ctx := context.Background()

    // 1. Create SNS topics for notifications
    topics := []string{"critical", "warning", "info"}
    topicArns := make(map[string]string)
    
    for _, severity := range topics {
        topicConfig := &cloud.SNSTopicConfig{
            Name:        fmt.Sprintf("apm-%s-alerts", severity),
            DisplayName: fmt.Sprintf("APM %s Alerts", strings.Title(severity)),
            Region:      "us-east-1",
            Tags: map[string]string{
                "Environment": "production",
                "Severity":    severity,
            },
        }
        
        topic, err := manager.CreateSNSTopic(ctx, topicConfig)
        if err != nil {
            log.Printf("Failed to create %s topic: %v", severity, err)
            continue
        }
        
        topicArns[severity] = topic.TopicArn
        log.Printf("Created %s topic: %s", severity, topic.TopicArn)
    }

    // 2. Create APM dashboards
    dashboardTypes := []string{"infrastructure", "application", "service-mesh", "logs", "tracing"}
    
    for _, dashboardType := range dashboardTypes {
        dashboard, err := manager.CreateAPMDashboard(ctx, 
            fmt.Sprintf("APM-%s-Dashboard", strings.Title(dashboardType)), 
            "us-east-1", 
            dashboardType, 
            "production")
        if err != nil {
            log.Printf("Failed to create %s dashboard: %v", dashboardType, err)
            continue
        }
        
        log.Printf("Created %s dashboard: %s", dashboardType, dashboard.DashboardName)
    }

    // 3. Create APM alarms
    alarmConfigs := []struct {
        name       string
        alarmType  string
        topicArn   string
    }{
        {"APM-High-CPU-Utilization", "high-cpu", topicArns["warning"]},
        {"APM-High-Memory-Utilization", "high-memory", topicArns["warning"]},
        {"APM-Low-Disk-Space", "low-disk-space", topicArns["critical"]},
        {"APM-Service-Down", "service-down", topicArns["critical"]},
        {"APM-High-Error-Rate", "high-error-rate", topicArns["warning"]},
        {"APM-Slow-Response-Time", "slow-response-time", topicArns["warning"]},
    }

    for _, config := range alarmConfigs {
        alarm, err := manager.CreateAPMAlarm(ctx, config.name, "us-east-1", config.alarmType, "production")
        if err != nil {
            log.Printf("Failed to create %s alarm: %v", config.name, err)
            continue
        }
        
        // Add SNS notification to alarm
        err = manager.AddAlarmAction(ctx, alarm.AlarmName, "us-east-1", "ALARM", config.topicArn)
        if err != nil {
            log.Printf("Failed to add notification to %s: %v", config.name, err)
        }
        
        log.Printf("Created %s alarm: %s", config.alarmType, alarm.AlarmName)
    }

    // 4. Create log groups and metric filters
    apmTools := []string{"prometheus", "grafana", "jaeger", "loki"}
    
    for _, tool := range apmTools {
        logGroup, err := manager.CreateAPMLogGroup(ctx, 
            fmt.Sprintf("/aws/apm/%s", tool), 
            "us-east-1", 
            30, 
            "production")
        if err != nil {
            log.Printf("Failed to create %s log group: %v", tool, err)
            continue
        }
        
        log.Printf("Created %s log group: %s", tool, logGroup.LogGroupName)
        
        // Create error metric filter
        metricFilterConfig := &cloud.MetricFilterConfig{
            FilterName:         fmt.Sprintf("APM-%s-Error-Count", strings.Title(tool)),
            LogGroupName:       logGroup.LogGroupName,
            FilterPattern:      "[timestamp, request_id, level=\"ERROR\", ...]",
            MetricTransformations: []*cloud.MetricTransformation{
                {
                    MetricName:      fmt.Sprintf("%sErrorCount", strings.Title(tool)),
                    MetricNamespace: "APM/Application",
                    MetricValue:     "1",
                    DefaultValue:    0,
                },
            },
        }
        
        metricFilter, err := manager.CreateMetricFilter(ctx, metricFilterConfig)
        if err != nil {
            log.Printf("Failed to create %s metric filter: %v", tool, err)
            continue
        }
        
        log.Printf("Created %s metric filter: %s", tool, metricFilter.FilterName)
    }

    // 5. Create event rules
    eventRuleConfig := &cloud.EventRuleConfig{
        Name:        "apm-instance-state-change",
        Description: "Monitor EC2 instance state changes for APM",
        EventPattern: map[string]interface{}{
            "source":        []string{"aws.ec2"},
            "detail-type":   []string{"EC2 Instance State-change Notification"},
            "detail": map[string]interface{}{
                "state": []string{"running", "stopped", "terminated"},
            },
        },
        State:  "ENABLED",
        Region: "us-east-1",
        Targets: []*cloud.EventTarget{
            {
                Id:      "1",
                Arn:     topicArns["info"],
                RoleArn: "arn:aws:iam::123456789012:role/CloudWatchEventsRole",
            },
        },
    }

    eventRule, err := manager.CreateEventRule(ctx, eventRuleConfig)
    if err != nil {
        log.Printf("Failed to create event rule: %v", err)
    } else {
        log.Printf("Created event rule: %s", eventRule.Name)
    }

    // 6. Configure APM tool integrations
    log.Println("Configuring APM tool integrations...")
    
    // Prometheus integration
    prometheusConfig := &cloud.PrometheusIntegrationConfig{
        MetricsEndpoint: "http://prometheus:9090/metrics",
        ScrapeInterval:  "15s",
        CustomMetrics: []string{
            "apm_request_duration_seconds",
            "apm_request_total",
            "apm_error_rate",
        },
    }
    
    err = manager.ConfigurePrometheusIntegration(ctx, "us-east-1", prometheusConfig)
    if err != nil {
        log.Printf("Failed to configure Prometheus integration: %v", err)
    } else {
        log.Println("Configured Prometheus integration")
    }
    
    // Grafana integration
    grafanaConfig := &cloud.GrafanaIntegrationConfig{
        DashboardURL: "http://grafana:3000",
        APIKey:       "grafana-api-key",
        DashboardIDs: []string{
            "infrastructure-dashboard",
            "application-dashboard",
        },
    }
    
    err = manager.ConfigureGrafanaIntegration(ctx, "us-east-1", grafanaConfig)
    if err != nil {
        log.Printf("Failed to configure Grafana integration: %v", err)
    } else {
        log.Println("Configured Grafana integration")
    }

    // 7. Health check
    healthResult := manager.HealthCheck(ctx, "us-east-1")
    log.Printf("CloudWatch health status: %s", healthResult.Status)
    
    // 8. Show metrics
    metrics := manager.GetMetrics()
    log.Printf("Total operations: %d", metrics.TotalOperations)
    log.Printf("Success rate: %.2f%%", metrics.SuccessRate)
    log.Printf("Average response time: %v", metrics.AverageResponseTime)

    log.Println("APM monitoring setup completed!")
}
```

## Best Practices

### 1. Dashboard Organization
- Use consistent naming conventions: `APM-{Type}-{Environment}`
- Group related dashboards in folders
- Use dashboard tags for organization and filtering
- Create environment-specific dashboards

### 2. Alarm Configuration
- Set appropriate thresholds based on baseline metrics
- Use composite alarms for complex conditions
- Implement alarm escalation paths
- Regular review and adjustment of alarm thresholds

### 3. Log Management
- Use structured logging with consistent formats
- Set appropriate retention policies based on compliance requirements
- Create metric filters for important log patterns
- Use log encryption for sensitive data

### 4. Performance Optimization
- Enable caching for frequently accessed resources
- Use batch operations for bulk actions
- Implement connection pooling for concurrent operations
- Monitor and optimize query performance

### 5. Security
- Use IAM roles with least privilege access
- Enable encryption for sensitive data
- Implement access logging and monitoring
- Regular security audits and reviews

### 6. Cost Optimization
- Monitor CloudWatch costs and usage
- Use appropriate log retention policies
- Optimize dashboard and alarm configurations
- Regular cleanup of unused resources

## Troubleshooting

### Common Issues

#### 1. Dashboard Creation Failures
```
Error: Dashboard already exists (CODE: DASHBOARD_ALREADY_EXISTS)
```
**Solution**: 
- Check if dashboard already exists
- Use `GetDashboard` before creating
- Consider using `UpdateDashboard` instead

#### 2. Alarm Permission Issues
```
Error: Insufficient permissions (CODE: INSUFFICIENT_PERMISSIONS)
```
**Solution**:
- Check IAM permissions for CloudWatch actions
- Verify SNS topic permissions
- Ensure correct resource ARNs

#### 3. Log Group Access Denied
```
Error: Access denied to log group (CODE: ACCESS_DENIED)
```
**Solution**:
- Check CloudWatch Logs IAM permissions
- Verify log group exists
- Check encryption key permissions

#### 4. Insights Query Failures
```
Error: Query syntax error (CODE: INVALID_QUERY)
```
**Solution**:
- Validate query syntax
- Check log group names
- Verify time range parameters

### Debug Mode

Enable debug logging for troubleshooting:

```go
// Enable debug logging
manager.SetLogLevel(cloud.LogLevelDebug)

// Create dashboard with debug output
dashboard, err := manager.CreateDashboard(ctx, dashboardConfig)
if err != nil {
    // Debug information will be logged
    log.Printf("Dashboard creation failed: %v", err)
}
```

### Health Checks

Use health checks to monitor system status:

```go
// Comprehensive health check
healthResult := manager.HealthCheck(ctx, "us-east-1")
if healthResult.Status != "healthy" {
    log.Printf("CloudWatch health issues detected: %+v", healthResult.Details)
}

// Service-specific health checks
services := []string{"dashboards", "alarms", "logs", "insights", "events", "sns"}
for _, service := range services {
    serviceHealth := manager.CheckServiceHealth(ctx, service, "us-east-1")
    if serviceHealth.Status != "healthy" {
        log.Printf("%s service health issues: %+v", service, serviceHealth.Details)
    }
}
```

### Performance Monitoring

Monitor CloudWatch operation performance:

```go
// Get operation metrics
metrics := manager.GetMetrics()
log.Printf("Total operations: %d", metrics.TotalOperations)
log.Printf("Success rate: %.2f%%", metrics.SuccessRate)
log.Printf("Average response time: %v", metrics.AverageResponseTime)
log.Printf("Cache hit rate: %.2f%%", metrics.CacheHitRate)

// Monitor specific operations
operationMetrics := manager.GetOperationMetrics("CreateDashboard")
log.Printf("CreateDashboard operations: %d", operationMetrics.Count)
log.Printf("CreateDashboard success rate: %.2f%%", operationMetrics.SuccessRate)
```

## Support and Resources

- **AWS CloudWatch Documentation**: https://docs.aws.amazon.com/cloudwatch/
- **AWS CLI Documentation**: https://docs.aws.amazon.com/cli/
- **CloudWatch Logs Insights**: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/AnalyzingLogData.html
- **CloudWatch Events**: https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/
- **SNS Documentation**: https://docs.aws.amazon.com/sns/

For issues specific to this implementation, check the source code and tests in:
- `/pkg/cloud/aws.go` - Main CloudWatch implementation
- `/examples/cloudwatch-integration/` - Examples and tests

---

This documentation covers the comprehensive CloudWatch integration functionality for APM monitoring. The implementation provides enterprise-grade features including dashboard management, intelligent alerting, centralized logging, advanced analytics, and seamless integration with popular APM tools.