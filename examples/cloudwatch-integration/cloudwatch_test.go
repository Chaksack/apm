package main

import (
	"context"
	"testing"
	"time"

	"github.com/yourusername/apm/pkg/cloud"
)

func TestCloudWatchManagerInitialization(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	manager := provider.GetCloudWatchManager()
	if manager == nil {
		t.Fatal("CloudWatch manager should not be nil")
	}

	// Test manager components
	if manager.GetDashboardManager() == nil {
		t.Error("Dashboard manager should not be nil")
	}

	if manager.GetAlarmManager() == nil {
		t.Error("Alarm manager should not be nil")
	}

	if manager.GetLogsManager() == nil {
		t.Error("Logs manager should not be nil")
	}

	if manager.GetInsightsManager() == nil {
		t.Error("Insights manager should not be nil")
	}

	if manager.GetEventsManager() == nil {
		t.Error("Events manager should not be nil")
	}

	if manager.GetSNSManager() == nil {
		t.Error("SNS manager should not be nil")
	}

	if manager.GetAPMIntegrationManager() == nil {
		t.Error("APM integration manager should not be nil")
	}
}

func TestDashboardManagement(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	manager := provider.GetCloudWatchManager()
	ctx := context.Background()

	// Test dashboard creation
	dashboardConfig := &cloud.DashboardConfig{
		Name:        "Test-Dashboard",
		Description: "Test dashboard for unit testing",
		Region:      "us-east-1",
		Type:        "infrastructure",
		Environment: "test",
		Tags: map[string]string{
			"Environment": "test",
			"Purpose":     "unit-testing",
		},
		Widgets: []*cloud.DashboardWidget{
			{
				Type:   "metric",
				Title:  "Test Metric",
				Width:  12,
				Height: 6,
				Properties: map[string]interface{}{
					"metrics": [][]interface{}{
						{"AWS/EC2", "CPUUtilization", "InstanceId", "i-test"},
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
		t.Errorf("Failed to create dashboard: %v", err)
	} else {
		if dashboard.DashboardName != "Test-Dashboard" {
			t.Errorf("Expected dashboard name 'Test-Dashboard', got '%s'", dashboard.DashboardName)
		}
		if dashboard.Region != "us-east-1" {
			t.Errorf("Expected region 'us-east-1', got '%s'", dashboard.Region)
		}
	}

	// Test dashboard listing
	dashboards, err := manager.ListDashboards(ctx, "us-east-1", "Test-")
	if err != nil {
		t.Errorf("Failed to list dashboards: %v", err)
	} else {
		if len(dashboards) == 0 {
			t.Error("Expected at least one dashboard in the list")
		}
	}

	// Test APM dashboard creation
	apmDashboard, err := manager.CreateAPMDashboard(ctx, "APM-Test-Dashboard", "us-east-1", "infrastructure", "test")
	if err != nil {
		t.Errorf("Failed to create APM dashboard: %v", err)
	} else {
		if apmDashboard.DashboardName != "APM-Test-Dashboard" {
			t.Errorf("Expected dashboard name 'APM-Test-Dashboard', got '%s'", apmDashboard.DashboardName)
		}
	}
}

func TestAlarmManagement(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	manager := provider.GetCloudWatchManager()
	ctx := context.Background()

	// Test alarm creation
	alarmConfig := &cloud.AlarmConfig{
		Name:               "Test-Alarm",
		Description:        "Test alarm for unit testing",
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
			"InstanceId": "i-test",
		},
		Tags: map[string]string{
			"Environment": "test",
			"Purpose":     "unit-testing",
		},
	}

	alarm, err := manager.CreateAlarm(ctx, alarmConfig)
	if err != nil {
		t.Errorf("Failed to create alarm: %v", err)
	} else {
		if alarm.AlarmName != "Test-Alarm" {
			t.Errorf("Expected alarm name 'Test-Alarm', got '%s'", alarm.AlarmName)
		}
		if alarm.Threshold != 80.0 {
			t.Errorf("Expected threshold 80.0, got %f", alarm.Threshold)
		}
	}

	// Test alarm listing
	alarms, err := manager.ListAlarms(ctx, "us-east-1", "Test-")
	if err != nil {
		t.Errorf("Failed to list alarms: %v", err)
	} else {
		if len(alarms) == 0 {
			t.Error("Expected at least one alarm in the list")
		}
	}

	// Test APM alarm creation
	apmAlarm, err := manager.CreateAPMAlarm(ctx, "APM-Test-Alarm", "us-east-1", "high-cpu", "test")
	if err != nil {
		t.Errorf("Failed to create APM alarm: %v", err)
	} else {
		if apmAlarm.AlarmName != "APM-Test-Alarm" {
			t.Errorf("Expected alarm name 'APM-Test-Alarm', got '%s'", apmAlarm.AlarmName)
		}
	}
}

func TestLogManagement(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	manager := provider.GetCloudWatchManager()
	ctx := context.Background()

	// Test log group creation
	logGroupConfig := &cloud.LogGroupConfig{
		Name:            "/aws/test/unit-test-logs",
		Region:          "us-east-1",
		RetentionInDays: 7,
		Tags: map[string]string{
			"Environment": "test",
			"Purpose":     "unit-testing",
		},
	}

	logGroup, err := manager.CreateLogGroup(ctx, logGroupConfig)
	if err != nil {
		t.Errorf("Failed to create log group: %v", err)
	} else {
		if logGroup.LogGroupName != "/aws/test/unit-test-logs" {
			t.Errorf("Expected log group name '/aws/test/unit-test-logs', got '%s'", logGroup.LogGroupName)
		}
		if logGroup.RetentionInDays != 7 {
			t.Errorf("Expected retention 7 days, got %d", logGroup.RetentionInDays)
		}
	}

	// Test log group listing
	logGroups, err := manager.ListLogGroups(ctx, "us-east-1", "/aws/test/")
	if err != nil {
		t.Errorf("Failed to list log groups: %v", err)
	} else {
		if len(logGroups) == 0 {
			t.Error("Expected at least one log group in the list")
		}
	}

	// Test metric filter creation
	metricFilterConfig := &cloud.MetricFilterConfig{
		FilterName:    "Test-Error-Filter",
		LogGroupName:  "/aws/test/unit-test-logs",
		FilterPattern: "[timestamp, request_id, level=\"ERROR\", ...]",
		MetricTransformations: []*cloud.MetricTransformation{
			{
				MetricName:      "TestErrorCount",
				MetricNamespace: "Test/Application",
				MetricValue:     "1",
				DefaultValue:    0,
			},
		},
	}

	metricFilter, err := manager.CreateMetricFilter(ctx, metricFilterConfig)
	if err != nil {
		t.Errorf("Failed to create metric filter: %v", err)
	} else {
		if metricFilter.FilterName != "Test-Error-Filter" {
			t.Errorf("Expected filter name 'Test-Error-Filter', got '%s'", metricFilter.FilterName)
		}
	}

	// Test log event publishing
	logEvents := []*cloud.LogEvent{
		{
			Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			Message:   "[INFO] Test log message",
		},
		{
			Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			Message:   "[ERROR] Test error message",
		},
	}

	err = manager.PutLogEvents(ctx, "/aws/test/unit-test-logs", "test-stream", logEvents)
	if err != nil {
		t.Errorf("Failed to put log events: %v", err)
	}
}

func TestInsightsQueries(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	manager := provider.GetCloudWatchManager()
	ctx := context.Background()

	// Test insights query
	queryConfig := &cloud.InsightsQueryConfig{
		LogGroupNames: []string{"/aws/test/unit-test-logs"},
		StartTime:     time.Now().Add(-1 * time.Hour),
		EndTime:       time.Now(),
		QueryString:   "fields @timestamp, @message | filter @message like /ERROR/ | limit 10",
		Region:        "us-east-1",
	}

	query, err := manager.StartInsightsQuery(ctx, queryConfig)
	if err != nil {
		t.Errorf("Failed to start insights query: %v", err)
	} else {
		if query.QueryId == "" {
			t.Error("Expected query ID to be non-empty")
		}
		if query.Status == "" {
			t.Error("Expected query status to be non-empty")
		}
	}

	// Test APM insights query
	apmQuery, err := manager.ExecuteAPMInsightsQuery(ctx, "test-query", []string{"/aws/test/unit-test-logs"}, "fields @timestamp, @message | limit 5", "us-east-1")
	if err != nil {
		t.Errorf("Failed to execute APM insights query: %v", err)
	} else {
		if apmQuery.QueryId == "" {
			t.Error("Expected APM query ID to be non-empty")
		}
	}
}

func TestSNSManagement(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	manager := provider.GetCloudWatchManager()
	ctx := context.Background()

	// Test SNS topic creation
	topicConfig := &cloud.SNSTopicConfig{
		Name:        "test-notifications",
		DisplayName: "Test Notifications",
		Region:      "us-east-1",
		Tags: map[string]string{
			"Environment": "test",
			"Purpose":     "unit-testing",
		},
	}

	topic, err := manager.CreateSNSTopic(ctx, topicConfig)
	if err != nil {
		t.Errorf("Failed to create SNS topic: %v", err)
	} else {
		if topic.TopicArn == "" {
			t.Error("Expected topic ARN to be non-empty")
		}
		if topic.DisplayName != "Test Notifications" {
			t.Errorf("Expected display name 'Test Notifications', got '%s'", topic.DisplayName)
		}
	}

	// Test SNS topic listing
	topics, err := manager.ListSNSTopics(ctx, "us-east-1")
	if err != nil {
		t.Errorf("Failed to list SNS topics: %v", err)
	} else {
		if len(topics) == 0 {
			t.Error("Expected at least one SNS topic in the list")
		}
	}

	// Test subscription creation
	subscriptionConfig := &cloud.SNSSubscriptionConfig{
		TopicArn: topic.TopicArn,
		Protocol: "email",
		Endpoint: "test@example.com",
		Attributes: map[string]string{
			"FilterPolicy": `{"source": ["test"]}`,
		},
	}

	subscription, err := manager.CreateSNSSubscription(ctx, subscriptionConfig)
	if err != nil {
		t.Errorf("Failed to create SNS subscription: %v", err)
	} else {
		if subscription.SubscriptionArn == "" {
			t.Error("Expected subscription ARN to be non-empty")
		}
		if subscription.Protocol != "email" {
			t.Errorf("Expected protocol 'email', got '%s'", subscription.Protocol)
		}
	}
}

func TestEventRuleManagement(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	manager := provider.GetCloudWatchManager()
	ctx := context.Background()

	// Test event rule creation
	eventRuleConfig := &cloud.EventRuleConfig{
		Name:        "test-event-rule",
		Description: "Test event rule for unit testing",
		EventPattern: map[string]interface{}{
			"source":      []string{"aws.ec2"},
			"detail-type": []string{"EC2 Instance State-change Notification"},
		},
		State:  "ENABLED",
		Region: "us-east-1",
		Tags: map[string]string{
			"Environment": "test",
			"Purpose":     "unit-testing",
		},
	}

	eventRule, err := manager.CreateEventRule(ctx, eventRuleConfig)
	if err != nil {
		t.Errorf("Failed to create event rule: %v", err)
	} else {
		if eventRule.Name != "test-event-rule" {
			t.Errorf("Expected rule name 'test-event-rule', got '%s'", eventRule.Name)
		}
		if eventRule.State != "ENABLED" {
			t.Errorf("Expected state 'ENABLED', got '%s'", eventRule.State)
		}
	}

	// Test event rule listing
	eventRules, err := manager.ListEventRules(ctx, "us-east-1", "test-")
	if err != nil {
		t.Errorf("Failed to list event rules: %v", err)
	} else {
		if len(eventRules) == 0 {
			t.Error("Expected at least one event rule in the list")
		}
	}
}

func TestAPMIntegration(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	manager := provider.GetCloudWatchManager()
	ctx := context.Background()

	// Test APM monitoring setup
	apmConfig := &cloud.APMMonitoringConfig{
		Environment: "test",
		Region:      "us-east-1",
		Tools:       []string{"prometheus", "grafana"},
		Dashboards: map[string]string{
			"infrastructure": "Test-Infrastructure-Dashboard",
			"application":    "Test-Application-Dashboard",
		},
		Alarms: map[string]string{
			"high-cpu":     "Test-High-CPU-Alarm",
			"service-down": "Test-Service-Down-Alarm",
		},
		LogGroups: []string{
			"/aws/test/prometheus",
			"/aws/test/grafana",
		},
		SNSTopics: []string{
			"test-critical-alerts",
			"test-info-notifications",
		},
		Tags: map[string]string{
			"Environment": "test",
			"Purpose":     "unit-testing",
		},
	}

	setup, err := manager.CreateAPMMonitoringSetup(ctx, apmConfig)
	if err != nil {
		t.Errorf("Failed to create APM monitoring setup: %v", err)
	} else {
		if setup == nil {
			t.Error("Expected APM monitoring setup to be non-nil")
		}
	}

	// Test Prometheus integration
	prometheusConfig := &cloud.PrometheusIntegrationConfig{
		MetricsEndpoint: "http://localhost:9090/metrics",
		ScrapeInterval:  "15s",
		CustomMetrics: []string{
			"test_request_duration_seconds",
			"test_request_total",
		},
	}

	err = manager.ConfigurePrometheusIntegration(ctx, "us-east-1", prometheusConfig)
	if err != nil {
		t.Errorf("Failed to configure Prometheus integration: %v", err)
	}

	// Test Grafana integration
	grafanaConfig := &cloud.GrafanaIntegrationConfig{
		DashboardURL: "http://localhost:3000",
		APIKey:       "test-api-key",
		DashboardIDs: []string{
			"test-dashboard-1",
			"test-dashboard-2",
		},
	}

	err = manager.ConfigureGrafanaIntegration(ctx, "us-east-1", grafanaConfig)
	if err != nil {
		t.Errorf("Failed to configure Grafana integration: %v", err)
	}
}

func TestHealthCheck(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	manager := provider.GetCloudWatchManager()
	ctx := context.Background()

	// Test overall health check
	healthResult := manager.HealthCheck(ctx, "us-east-1")
	if healthResult == nil {
		t.Error("Expected health result to be non-nil")
	} else {
		if healthResult.Status == "" {
			t.Error("Expected health status to be non-empty")
		}
		if healthResult.Service != "CloudWatch" {
			t.Errorf("Expected service 'CloudWatch', got '%s'", healthResult.Service)
		}
	}

	// Test service-specific health checks
	services := []string{"dashboards", "alarms", "logs", "insights", "events", "sns"}
	for _, service := range services {
		serviceHealth := manager.CheckServiceHealth(ctx, service, "us-east-1")
		if serviceHealth == nil {
			t.Errorf("Expected health result for %s to be non-nil", service)
		} else {
			if serviceHealth.Service != service {
				t.Errorf("Expected service '%s', got '%s'", service, serviceHealth.Service)
			}
		}
	}
}

func TestErrorHandling(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	manager := provider.GetCloudWatchManager()
	ctx := context.Background()

	// Test invalid dashboard creation
	invalidConfig := &cloud.DashboardConfig{
		Name:        "", // Invalid empty name
		Region:      "us-east-1",
		Type:        "infrastructure",
		Environment: "test",
	}

	_, err = manager.CreateDashboard(ctx, invalidConfig)
	if err == nil {
		t.Error("Expected error for invalid dashboard config")
	}

	// Test invalid alarm creation
	invalidAlarmConfig := &cloud.AlarmConfig{
		Name:       "", // Invalid empty name
		Region:     "us-east-1",
		MetricName: "CPUUtilization",
		Namespace:  "AWS/EC2",
		Threshold:  -1, // Invalid threshold
	}

	_, err = manager.CreateAlarm(ctx, invalidAlarmConfig)
	if err == nil {
		t.Error("Expected error for invalid alarm config")
	}

	// Test invalid log group creation
	invalidLogConfig := &cloud.LogGroupConfig{
		Name:            "", // Invalid empty name
		Region:          "us-east-1",
		RetentionInDays: -1, // Invalid retention
	}

	_, err = manager.CreateLogGroup(ctx, invalidLogConfig)
	if err == nil {
		t.Error("Expected error for invalid log group config")
	}
}

func TestMetricsCollection(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	manager := provider.GetCloudWatchManager()

	// Test initial metrics
	metrics := manager.GetMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be non-nil")
	} else {
		if metrics.TotalOperations < 0 {
			t.Error("Expected total operations to be non-negative")
		}
		if metrics.SuccessRate < 0 || metrics.SuccessRate > 100 {
			t.Error("Expected success rate to be between 0 and 100")
		}
	}

	// Test metrics reset
	manager.ResetMetrics()
	resetMetrics := manager.GetMetrics()
	if resetMetrics == nil {
		t.Error("Expected reset metrics to be non-nil")
	} else {
		if resetMetrics.TotalOperations != 0 {
			t.Error("Expected total operations to be 0 after reset")
		}
	}
}

func TestCacheOperations(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	manager := provider.GetCloudWatchManager()

	// Test cache warming
	warmupConfig := &cloud.CloudWatchCacheWarmupConfig{
		Region: "us-east-1",
		Dashboards: []string{
			"test-dashboard-1",
			"test-dashboard-2",
		},
		Alarms: []string{
			"test-alarm-1",
			"test-alarm-2",
		},
		LogGroups: []string{
			"/aws/test/log-group-1",
			"/aws/test/log-group-2",
		},
	}

	err = manager.WarmupCache(context.Background(), warmupConfig)
	if err != nil {
		t.Errorf("Failed to warmup cache: %v", err)
	}

	// Test cache statistics
	stats := manager.GetCacheStats()
	if stats == nil {
		t.Error("Expected cache stats to be non-nil")
	}
}

func TestConcurrentOperations(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	manager := provider.GetCloudWatchManager()
	ctx := context.Background()

	// Test concurrent dashboard creation
	concurrentCount := 5
	done := make(chan bool, concurrentCount)
	errors := make(chan error, concurrentCount)

	for i := 0; i < concurrentCount; i++ {
		go func(index int) {
			defer func() { done <- true }()

			dashboardConfig := &cloud.DashboardConfig{
				Name:        fmt.Sprintf("Concurrent-Dashboard-%d", index),
				Description: fmt.Sprintf("Concurrent test dashboard %d", index),
				Region:      "us-east-1",
				Type:        "infrastructure",
				Environment: "test",
				Tags: map[string]string{
					"Environment": "test",
					"Index":       fmt.Sprintf("%d", index),
				},
			}

			_, err := manager.CreateDashboard(ctx, dashboardConfig)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrentCount; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent operation failed: %v", err)
		}
	}
}

func TestMultiRegionOperations(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	manager := provider.GetCloudWatchManager()
	ctx := context.Background()

	// Test multi-region dashboard creation
	regions := []string{"us-east-1", "us-west-2", "eu-west-1"}

	for _, region := range regions {
		dashboardConfig := &cloud.DashboardConfig{
			Name:        fmt.Sprintf("Multi-Region-Dashboard-%s", region),
			Description: fmt.Sprintf("Multi-region test dashboard for %s", region),
			Region:      region,
			Type:        "infrastructure",
			Environment: "test",
			Tags: map[string]string{
				"Environment": "test",
				"Region":      region,
			},
		}

		_, err := manager.CreateDashboard(ctx, dashboardConfig)
		if err != nil {
			t.Errorf("Failed to create dashboard in region %s: %v", region, err)
		}
	}

	// Test multi-region alarm creation
	for _, region := range regions {
		alarmConfig := &cloud.AlarmConfig{
			Name:               fmt.Sprintf("Multi-Region-Alarm-%s", region),
			Description:        fmt.Sprintf("Multi-region test alarm for %s", region),
			Region:             region,
			MetricName:         "CPUUtilization",
			Namespace:          "AWS/EC2",
			Statistic:          "Average",
			Period:             300,
			EvaluationPeriods:  2,
			Threshold:          80.0,
			ComparisonOperator: "GreaterThanThreshold",
			Tags: map[string]string{
				"Environment": "test",
				"Region":      region,
			},
		}

		_, err := manager.CreateAlarm(ctx, alarmConfig)
		if err != nil {
			t.Errorf("Failed to create alarm in region %s: %v", region, err)
		}
	}
}
