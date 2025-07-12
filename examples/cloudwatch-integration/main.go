package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chaksack/apm/pkg/cloud"
)

func main() {
	fmt.Println("CloudWatch Integration Demo")
	fmt.Println("===========================")

	// Create an AWS provider
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		log.Fatalf("Failed to create AWS provider: %v", err)
	}

	// Get CloudWatch manager
	cloudWatchManager := provider.GetCloudWatchManager()
	if cloudWatchManager == nil {
		log.Fatal("CloudWatch manager is not available")
	}

	ctx := context.Background()

	// Demo 1: Dashboard Management
	fmt.Println("\n1. Dashboard Management:")
	demonstrateDashboardManagement(ctx, cloudWatchManager)

	// Demo 2: Alarm Management
	fmt.Println("\n2. Alarm Management:")
	demonstrateAlarmManagement(ctx, cloudWatchManager)

	// Demo 3: Log Management
	fmt.Println("\n3. Log Management:")
	demonstrateLogManagement(ctx, cloudWatchManager)

	// Demo 4: Insights Queries
	fmt.Println("\n4. CloudWatch Insights:")
	demonstrateInsightsQueries(ctx, cloudWatchManager)

	// Demo 5: Events and SNS
	fmt.Println("\n5. Events and SNS:")
	demonstrateEventsAndSNS(ctx, cloudWatchManager)

	// Demo 6: APM Integration
	fmt.Println("\n6. APM Integration:")
	demonstrateAPMIntegration(ctx, cloudWatchManager)

	// Demo 7: Health Check
	fmt.Println("\n7. Health Check:")
	demonstrateHealthCheck(ctx, cloudWatchManager)

	fmt.Println("\nCloudWatch Integration Demo completed!")
}

func demonstrateDashboardManagement(ctx context.Context, manager *cloud.CloudWatchManager) {
	fmt.Println("Creating APM dashboard...")

	// Create an APM infrastructure dashboard
	dashboardConfig := &cloud.DashboardConfig{
		Name:        "APM-Infrastructure-Dashboard",
		Description: "APM infrastructure monitoring dashboard",
		Region:      "us-east-1",
		Type:        "infrastructure",
		Environment: "production",
		Tags: map[string]string{
			"Environment": "production",
			"Purpose":     "apm-monitoring",
			"Team":        "platform",
		},
		APMTools: []string{"prometheus", "grafana", "jaeger", "loki"},
		Widgets: []*cloud.DashboardWidget{
			{
				Type:   "metric",
				Title:  "CPU Utilization",
				Width:  12,
				Height: 6,
				Properties: map[string]interface{}{
					"metrics": [][]interface{}{
						{"AWS/EC2", "CPUUtilization", "InstanceId", "i-1234567890abcdef0"},
					},
					"period": 300,
					"stat":   "Average",
					"region": "us-east-1",
				},
			},
			{
				Type:   "metric",
				Title:  "Memory Utilization",
				Width:  12,
				Height: 6,
				Properties: map[string]interface{}{
					"metrics": [][]interface{}{
						{"CWAgent", "mem_used_percent", "InstanceId", "i-1234567890abcdef0"},
					},
					"period": 300,
					"stat":   "Average",
					"region": "us-east-1",
				},
			},
		},
	}

	dashboard, err := manager.CreateDashboard(ctx, dashboardConfig)
	if err != nil {
		fmt.Printf("  Error creating dashboard: %v\n", err)
	} else {
		fmt.Printf("  ✓ Created dashboard: %s\n", dashboard.DashboardName)
		fmt.Printf("  ✓ Dashboard ARN: %s\n", dashboard.DashboardArn)
	}

	// List dashboards
	fmt.Println("Listing dashboards...")
	dashboards, err := manager.ListDashboards(ctx, "us-east-1", "APM-")
	if err != nil {
		fmt.Printf("  Error listing dashboards: %v\n", err)
	} else {
		fmt.Printf("  ✓ Found %d dashboards with APM prefix\n", len(dashboards))
		for _, dash := range dashboards {
			fmt.Printf("    - %s (modified: %s)\n", dash.DashboardName, dash.LastModified.Format("2006-01-02 15:04:05"))
		}
	}

	// Create APM-specific dashboard templates
	fmt.Println("Creating APM-specific dashboard templates...")
	templates := []string{"infrastructure", "application", "service-mesh", "logs", "tracing"}
	for _, template := range templates {
		templateDashboard, err := manager.CreateAPMDashboard(ctx, fmt.Sprintf("APM-%s-Template", template), "us-east-1", template, "production")
		if err != nil {
			fmt.Printf("  Error creating %s template: %v\n", template, err)
		} else {
			fmt.Printf("  ✓ Created %s template: %s\n", template, templateDashboard.DashboardName)
		}
	}
}

func demonstrateAlarmManagement(ctx context.Context, manager *cloud.CloudWatchManager) {
	fmt.Println("Creating APM alarms...")

	// Create CPU utilization alarm
	alarmConfig := &cloud.AlarmConfig{
		Name:               "APM-High-CPU-Utilization",
		Description:        "Alert when CPU utilization is high",
		Region:             "us-east-1",
		MetricName:         "CPUUtilization",
		Namespace:          "AWS/EC2",
		Statistic:          "Average",
		Period:             300,
		EvaluationPeriods:  2,
		Threshold:          80.0,
		ComparisonOperator: "GreaterThanThreshold",
		TreatMissingData:   "notBreaching",
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
	if err != nil {
		fmt.Printf("  Error creating alarm: %v\n", err)
	} else {
		fmt.Printf("  ✓ Created alarm: %s\n", alarm.AlarmName)
		fmt.Printf("  ✓ Alarm ARN: %s\n", alarm.AlarmArn)
	}

	// List alarms
	fmt.Println("Listing alarms...")
	alarms, err := manager.ListAlarms(ctx, "us-east-1", "APM-")
	if err != nil {
		fmt.Printf("  Error listing alarms: %v\n", err)
	} else {
		fmt.Printf("  ✓ Found %d alarms with APM prefix\n", len(alarms))
		for _, alarm := range alarms {
			fmt.Printf("    - %s (state: %s)\n", alarm.AlarmName, alarm.StateValue)
		}
	}

	// Create APM-specific alarms
	fmt.Println("Creating APM-specific alarms...")
	apmAlarms := []string{"prometheus-down", "grafana-high-response-time", "jaeger-storage-full", "loki-log-ingestion-rate"}
	for _, alarmType := range apmAlarms {
		apmAlarm, err := manager.CreateAPMAlarm(ctx, fmt.Sprintf("APM-%s", alarmType), "us-east-1", alarmType, "production")
		if err != nil {
			fmt.Printf("  Error creating %s alarm: %v\n", alarmType, err)
		} else {
			fmt.Printf("  ✓ Created %s alarm: %s\n", alarmType, apmAlarm.AlarmName)
		}
	}
}

func demonstrateLogManagement(ctx context.Context, manager *cloud.CloudWatchManager) {
	fmt.Println("Managing CloudWatch Logs...")

	// Create log group
	logGroupConfig := &cloud.LogGroupConfig{
		Name:            "/aws/apm/application-logs",
		Region:          "us-east-1",
		RetentionInDays: 30,
		KmsKeyId:        "", // Use default encryption
		Tags: map[string]string{
			"Environment": "production",
			"Service":     "apm",
		},
	}

	logGroup, err := manager.CreateLogGroup(ctx, logGroupConfig)
	if err != nil {
		fmt.Printf("  Error creating log group: %v\n", err)
	} else {
		fmt.Printf("  ✓ Created log group: %s\n", logGroup.LogGroupName)
		fmt.Printf("  ✓ Retention: %d days\n", logGroup.RetentionInDays)
	}

	// List log groups
	fmt.Println("Listing log groups...")
	logGroups, err := manager.ListLogGroups(ctx, "us-east-1", "/aws/apm/")
	if err != nil {
		fmt.Printf("  Error listing log groups: %v\n", err)
	} else {
		fmt.Printf("  ✓ Found %d log groups with APM prefix\n", len(logGroups))
		for _, lg := range logGroups {
			fmt.Printf("    - %s (size: %d bytes)\n", lg.LogGroupName, lg.StoredBytes)
		}
	}

	// Create metric filter
	metricFilterConfig := &cloud.MetricFilterConfig{
		FilterName:    "APM-Error-Count",
		LogGroupName:  "/aws/apm/application-logs",
		FilterPattern: "[timestamp, request_id, level=\"ERROR\", ...]",
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
	if err != nil {
		fmt.Printf("  Error creating metric filter: %v\n", err)
	} else {
		fmt.Printf("  ✓ Created metric filter: %s\n", metricFilter.FilterName)
	}

	// Publish log events (example)
	fmt.Println("Publishing sample log events...")
	logEvents := []*cloud.LogEvent{
		{
			Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			Message:   "[INFO] APM application started successfully",
		},
		{
			Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			Message:   "[ERROR] Failed to connect to database",
		},
	}

	err = manager.PutLogEvents(ctx, "/aws/apm/application-logs", "application-stream", logEvents)
	if err != nil {
		fmt.Printf("  Error publishing log events: %v\n", err)
	} else {
		fmt.Printf("  ✓ Published %d log events\n", len(logEvents))
	}
}

func demonstrateInsightsQueries(ctx context.Context, manager *cloud.CloudWatchManager) {
	fmt.Println("Executing CloudWatch Insights queries...")

	// Query for error analysis
	queryConfig := &cloud.InsightsQueryConfig{
		LogGroupNames: []string{"/aws/apm/application-logs"},
		StartTime:     time.Now().Add(-24 * time.Hour),
		EndTime:       time.Now(),
		QueryString:   "fields @timestamp, @message | filter @message like /ERROR/ | sort @timestamp desc | limit 10",
		Region:        "us-east-1",
	}

	query, err := manager.StartInsightsQuery(ctx, queryConfig)
	if err != nil {
		fmt.Printf("  Error starting insights query: %v\n", err)
	} else {
		fmt.Printf("  ✓ Started insights query: %s\n", query.QueryId)
		fmt.Printf("  ✓ Query status: %s\n", query.Status)
	}

	// Execute APM-specific queries
	fmt.Println("Executing APM-specific queries...")
	apmQueries := map[string]string{
		"error-analysis":      "fields @timestamp, @message | filter @message like /ERROR/ | stats count() by bin(5m)",
		"performance-metrics": "fields @timestamp, @message | filter @message like /METRIC/ | parse @message /latency=(?<latency>\\d+)/",
		"request-patterns":    "fields @timestamp, @message | filter @message like /REQUEST/ | stats count() by bin(1h)",
	}

	for queryName, queryString := range apmQueries {
		result, err := manager.ExecuteAPMInsightsQuery(ctx, queryName, []string{"/aws/apm/application-logs"}, queryString, "us-east-1")
		if err != nil {
			fmt.Printf("  Error executing %s query: %v\n", queryName, err)
		} else {
			fmt.Printf("  ✓ Executed %s query: %s\n", queryName, result.QueryId)
		}
	}
}

func demonstrateEventsAndSNS(ctx context.Context, manager *cloud.CloudWatchManager) {
	fmt.Println("Managing CloudWatch Events and SNS...")

	// Create SNS topic
	topicConfig := &cloud.SNSTopicConfig{
		Name:        "apm-notifications",
		DisplayName: "APM Notifications",
		Region:      "us-east-1",
		Tags: map[string]string{
			"Environment": "production",
			"Service":     "apm",
		},
	}

	topic, err := manager.CreateSNSTopic(ctx, topicConfig)
	if err != nil {
		fmt.Printf("  Error creating SNS topic: %v\n", err)
	} else {
		fmt.Printf("  ✓ Created SNS topic: %s\n", topic.TopicArn)
	}

	// Create event rule
	eventRuleConfig := &cloud.EventRuleConfig{
		Name:        "apm-instance-state-change",
		Description: "Monitor EC2 instance state changes for APM",
		EventPattern: map[string]interface{}{
			"source":      []string{"aws.ec2"},
			"detail-type": []string{"EC2 Instance State-change Notification"},
			"detail": map[string]interface{}{
				"state": []string{"running", "stopped", "terminated"},
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
			},
		},
	}

	eventRule, err := manager.CreateEventRule(ctx, eventRuleConfig)
	if err != nil {
		fmt.Printf("  Error creating event rule: %v\n", err)
	} else {
		fmt.Printf("  ✓ Created event rule: %s\n", eventRule.Name)
		fmt.Printf("  ✓ Rule ARN: %s\n", eventRule.Arn)
	}

	// List event rules
	fmt.Println("Listing event rules...")
	eventRules, err := manager.ListEventRules(ctx, "us-east-1", "apm-")
	if err != nil {
		fmt.Printf("  Error listing event rules: %v\n", err)
	} else {
		fmt.Printf("  ✓ Found %d event rules with APM prefix\n", len(eventRules))
		for _, rule := range eventRules {
			fmt.Printf("    - %s (state: %s)\n", rule.Name, rule.State)
		}
	}

	// Subscribe to SNS topic
	subscriptionConfig := &cloud.SNSSubscriptionConfig{
		TopicArn: topic.TopicArn,
		Protocol: "email",
		Endpoint: "apm-admin@example.com",
		Attributes: map[string]string{
			"FilterPolicy": `{"source": ["aws.ec2"]}`,
		},
	}

	subscription, err := manager.CreateSNSSubscription(ctx, subscriptionConfig)
	if err != nil {
		fmt.Printf("  Error creating SNS subscription: %v\n", err)
	} else {
		fmt.Printf("  ✓ Created SNS subscription: %s\n", subscription.SubscriptionArn)
	}
}

func demonstrateAPMIntegration(ctx context.Context, manager *cloud.CloudWatchManager) {
	fmt.Println("Demonstrating APM tool integration...")

	// Create comprehensive APM monitoring setup
	apmConfig := &cloud.APMMonitoringConfig{
		Environment: "production",
		Region:      "us-east-1",
		Tools:       []string{"prometheus", "grafana", "jaeger", "loki"},
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
	if err != nil {
		fmt.Printf("  Error creating APM monitoring setup: %v\n", err)
	} else {
		fmt.Printf("  ✓ Created APM monitoring setup\n")
		fmt.Printf("  ✓ Dashboards: %d\n", len(setup.Dashboards))
		fmt.Printf("  ✓ Alarms: %d\n", len(setup.Alarms))
		fmt.Printf("  ✓ Log Groups: %d\n", len(setup.LogGroups))
		fmt.Printf("  ✓ SNS Topics: %d\n", len(setup.SNSTopics))
		fmt.Printf("  ✓ Event Rules: %d\n", len(setup.EventRules))
	}

	// Configure APM tool specific integrations
	fmt.Println("Configuring APM tool integrations...")

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
		fmt.Printf("  Error configuring Prometheus integration: %v\n", err)
	} else {
		fmt.Printf("  ✓ Configured Prometheus integration\n")
	}

	// Grafana integration
	grafanaConfig := &cloud.GrafanaIntegrationConfig{
		DashboardURL: "http://grafana:3000",
		APIKey:       "grafana-api-key",
		DashboardIDs: []string{
			"infrastructure-dashboard",
			"application-dashboard",
			"service-mesh-dashboard",
		},
	}

	err = manager.ConfigureGrafanaIntegration(ctx, "us-east-1", grafanaConfig)
	if err != nil {
		fmt.Printf("  Error configuring Grafana integration: %v\n", err)
	} else {
		fmt.Printf("  ✓ Configured Grafana integration\n")
	}

	// Jaeger integration
	jaegerConfig := &cloud.JaegerIntegrationConfig{
		CollectorEndpoint: "http://jaeger-collector:14268/api/traces",
		QueryEndpoint:     "http://jaeger-query:16686",
		SamplingRate:      0.1,
		Services: []string{
			"apm-frontend",
			"apm-backend",
			"apm-database",
		},
	}

	err = manager.ConfigureJaegerIntegration(ctx, "us-east-1", jaegerConfig)
	if err != nil {
		fmt.Printf("  Error configuring Jaeger integration: %v\n", err)
	} else {
		fmt.Printf("  ✓ Configured Jaeger integration\n")
	}

	// Loki integration
	lokiConfig := &cloud.LokiIntegrationConfig{
		PushURL:  "http://loki:3100/loki/api/v1/push",
		QueryURL: "http://loki:3100/loki/api/v1/query",
		TenantID: "apm-tenant",
		LogLabels: map[string]string{
			"environment": "production",
			"service":     "apm",
		},
	}

	err = manager.ConfigureLokiIntegration(ctx, "us-east-1", lokiConfig)
	if err != nil {
		fmt.Printf("  Error configuring Loki integration: %v\n", err)
	} else {
		fmt.Printf("  ✓ Configured Loki integration\n")
	}
}

func demonstrateHealthCheck(ctx context.Context, manager *cloud.CloudWatchManager) {
	fmt.Println("Performing CloudWatch health check...")

	// Basic health check
	healthResult := manager.HealthCheck(ctx, "us-east-1")
	fmt.Printf("  Overall Health: %s\n", healthResult.Status)
	fmt.Printf("  Response Time: %v\n", healthResult.ResponseTime)
	fmt.Printf("  Timestamp: %s\n", healthResult.Timestamp.Format("2006-01-02 15:04:05"))

	if len(healthResult.Details) > 0 {
		fmt.Println("  Health Details:")
		for service, status := range healthResult.Details {
			fmt.Printf("    %s: %s\n", service, status)
		}
	}

	if len(healthResult.Errors) > 0 {
		fmt.Println("  Health Errors:")
		for _, err := range healthResult.Errors {
			fmt.Printf("    - %s\n", err)
		}
	}

	// Service-specific health checks
	services := []string{"dashboards", "alarms", "logs", "insights", "events", "sns"}
	for _, service := range services {
		serviceHealth := manager.CheckServiceHealth(ctx, service, "us-east-1")
		fmt.Printf("  %s health: %s\n", service, serviceHealth.Status)
	}

	// Get current metrics
	fmt.Println("CloudWatch operation metrics:")
	metrics := manager.GetMetrics()
	fmt.Printf("  Total Operations: %d\n", metrics.TotalOperations)
	fmt.Printf("  Success Rate: %.2f%%\n", metrics.SuccessRate)
	fmt.Printf("  Average Response Time: %v\n", metrics.AverageResponseTime)
	fmt.Printf("  Cache Hit Rate: %.2f%%\n", metrics.CacheHitRate)
}
