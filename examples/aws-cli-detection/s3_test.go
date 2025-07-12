package main

import (
	"context"
	"testing"
	"time"

	"github.com/chaksack/apm/pkg/cloud"
)

func TestS3ManagerInitialization(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	s3Manager := provider.GetS3Manager()
	if s3Manager == nil {
		t.Fatal("S3Manager should not be nil")
	}

	// Test logger and metrics initialization
	logger := cloud.NewS3Logger(provider, true, cloud.LogLevelInfo)
	metrics := cloud.NewS3Metrics()

	if logger == nil {
		t.Fatal("S3Logger should not be nil")
	}

	if metrics == nil {
		t.Fatal("S3Metrics should not be nil")
	}

	// Set logger and metrics
	s3Manager.SetLogger(logger)
	s3Manager.SetMetrics(metrics)

	// Verify they were set
	if s3Manager.GetLogger() != logger {
		t.Error("Logger was not set correctly")
	}

	if s3Manager.GetMetrics() != metrics {
		t.Error("Metrics was not set correctly")
	}
}

func TestS3Logger(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	logger := cloud.NewS3Logger(provider, true, cloud.LogLevelInfo)

	// Test logging with different levels
	ctx := &cloud.S3OperationContext{
		Operation: "TestOperation",
		Bucket:    "test-bucket",
		StartTime: time.Now(),
		Success:   true,
	}

	// This should not panic
	logger.Log(cloud.LogLevelInfo, "TestOperation", "Test message", ctx)
	logger.LogOperation(ctx)

	// Test with disabled logger
	disabledLogger := cloud.NewS3Logger(provider, false, cloud.LogLevelInfo)
	disabledLogger.Log(cloud.LogLevelInfo, "TestOperation", "Test message", ctx)
}

func TestS3Metrics(t *testing.T) {
	metrics := cloud.NewS3Metrics()

	// Test initial state
	currentMetrics := metrics.GetMetrics()
	if currentMetrics.TotalOperations != 0 {
		t.Error("Initial total operations should be 0")
	}

	// Record a successful operation
	ctx := &cloud.S3OperationContext{
		Operation: "TestUpload",
		Success:   true,
		Duration:  100 * time.Millisecond,
		Size:      1024,
	}

	metrics.RecordOperation(ctx)

	// Verify metrics were updated
	currentMetrics = metrics.GetMetrics()
	if currentMetrics.TotalOperations != 1 {
		t.Errorf("Expected 1 operation, got %d", currentMetrics.TotalOperations)
	}

	if currentMetrics.SuccessfulOps != 1 {
		t.Errorf("Expected 1 successful operation, got %d", currentMetrics.SuccessfulOps)
	}

	if currentMetrics.TotalBytesUploaded != 1024 {
		t.Errorf("Expected 1024 bytes uploaded, got %d", currentMetrics.TotalBytesUploaded)
	}

	// Record a failed operation
	failedCtx := &cloud.S3OperationContext{
		Operation: "TestDownload",
		Success:   false,
		ErrorCode: "ACCESS_DENIED",
		Duration:  50 * time.Millisecond,
	}

	metrics.RecordOperation(failedCtx)

	// Verify metrics were updated
	currentMetrics = metrics.GetMetrics()
	if currentMetrics.TotalOperations != 2 {
		t.Errorf("Expected 2 operations, got %d", currentMetrics.TotalOperations)
	}

	if currentMetrics.FailedOps != 1 {
		t.Errorf("Expected 1 failed operation, got %d", currentMetrics.FailedOps)
	}

	if currentMetrics.ErrorCounts["ACCESS_DENIED"] != 1 {
		t.Errorf("Expected 1 ACCESS_DENIED error, got %d", currentMetrics.ErrorCounts["ACCESS_DENIED"])
	}

	// Test metrics reset
	metrics.ResetMetrics()
	currentMetrics = metrics.GetMetrics()
	if currentMetrics.TotalOperations != 0 {
		t.Error("Metrics should be reset to 0")
	}
}

func TestAPMConfigValidation(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	s3Manager := provider.GetS3Manager()

	// Test Prometheus configuration validation
	validPrometheusConfig := map[string]interface{}{
		"global": map[string]interface{}{
			"scrape_interval": "15s",
		},
		"scrape_configs": []map[string]interface{}{
			{
				"job_name": "prometheus",
				"static_configs": []map[string]interface{}{
					{
						"targets": []string{"localhost:9090"},
					},
				},
			},
		},
	}

	err = s3Manager.ValidateAPMConfig("prometheus", validPrometheusConfig)
	if err != nil {
		t.Errorf("Valid Prometheus config should pass validation: %v", err)
	}

	// Test invalid Prometheus configuration
	invalidPrometheusConfig := map[string]interface{}{
		"invalid_field": "value",
	}

	err = s3Manager.ValidateAPMConfig("prometheus", invalidPrometheusConfig)
	if err == nil {
		t.Error("Invalid Prometheus config should fail validation")
	}

	// Test Grafana configuration validation
	validGrafanaConfig := map[string]interface{}{
		"server": map[string]interface{}{
			"http_port": 3000,
		},
		"security": map[string]interface{}{
			"admin_password": "secure_password",
		},
	}

	err = s3Manager.ValidateAPMConfig("grafana", validGrafanaConfig)
	if err != nil {
		t.Errorf("Valid Grafana config should pass validation: %v", err)
	}

	// Test Grafana configuration with insecure password
	insecureGrafanaConfig := map[string]interface{}{
		"security": map[string]interface{}{
			"admin_password": "admin", // This should trigger validation error
		},
	}

	err = s3Manager.ValidateAPMConfig("grafana", insecureGrafanaConfig)
	if err == nil {
		t.Error("Grafana config with insecure password should fail validation")
	}
}

func TestErrorWrapping(t *testing.T) {
	originalErr := cloud.NewErrorBuilder(cloud.ProviderAWS, "TestOperation").
		Build("TEST_ERROR", "Test error message")

	wrappedErr := cloud.WrapS3Error(originalErr, "TestS3Operation", "test-bucket", "test-key")

	if wrappedErr == nil {
		t.Fatal("Wrapped error should not be nil")
	}

	cloudErr, ok := wrappedErr.(*cloud.CloudError)
	if !ok {
		t.Fatal("Wrapped error should be a CloudError")
	}

	if cloudErr.Provider != cloud.ProviderAWS {
		t.Errorf("Expected provider AWS, got %s", cloudErr.Provider)
	}

	if cloudErr.Operation != "TestS3Operation" {
		t.Errorf("Expected operation TestS3Operation, got %s", cloudErr.Operation)
	}
}

func TestHealthChecker(t *testing.T) {
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	s3Manager := provider.GetS3Manager()
	logger := cloud.NewS3Logger(provider, false, cloud.LogLevelError) // Disabled for testing

	healthChecker := cloud.NewS3HealthChecker(s3Manager, logger)
	if healthChecker == nil {
		t.Fatal("HealthChecker should not be nil")
	}

	// Test metrics monitoring
	metrics := cloud.NewS3Metrics()

	// Add some test metrics
	metrics.RecordOperation(&cloud.S3OperationContext{
		Operation: "TestOp1",
		Success:   true,
		Duration:  100 * time.Millisecond,
	})

	metrics.RecordOperation(&cloud.S3OperationContext{
		Operation: "TestOp2",
		Success:   false,
		ErrorCode: "TEST_ERROR",
		Duration:  200 * time.Millisecond,
	})

	ctx := context.Background()
	result := healthChecker.MonitorS3Operations(ctx, metrics)

	if result == nil {
		t.Fatal("Health check result should not be nil")
	}

	if result.Service != "S3Operations" {
		t.Errorf("Expected service S3Operations, got %s", result.Service)
	}

	// Should have some details
	if len(result.Details) == 0 {
		t.Error("Health check result should have details")
	}
}

func TestRetryMechanism(t *testing.T) {
	// Test successful retry
	attemptCount := 0
	err := cloud.RetryS3Operation(func() error {
		attemptCount++
		if attemptCount < 3 {
			return &cloud.CloudError{
				Provider:  cloud.ProviderAWS,
				Operation: "TestRetry",
				Code:      "THROTTLED",
				Message:   "Throttled",
				Retryable: true,
				Timestamp: time.Now(),
			}
		}
		return nil // Success on third attempt
	}, 5, 10*time.Millisecond, "TestRetry")

	if err != nil {
		t.Errorf("Retry should succeed: %v", err)
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}

	// Test non-retryable error
	err = cloud.RetryS3Operation(func() error {
		return &cloud.CloudError{
			Provider:  cloud.ProviderAWS,
			Operation: "TestRetry",
			Code:      "ACCESS_DENIED",
			Message:   "Access denied",
			Retryable: false,
			Timestamp: time.Now(),
		}
	}, 5, 10*time.Millisecond, "TestRetry")

	if err == nil {
		t.Error("Non-retryable error should not be retried")
	}
}
