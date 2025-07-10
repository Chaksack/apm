package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/chaksack/apm/pkg/cloud"
)

func main() {
	fmt.Println("AWS CLI Detection and S3 Manager Demo")
	fmt.Println("======================================")

	// Create an AWS provider
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		log.Fatalf("Failed to create AWS provider: %v", err)
	}

	// Perform basic CLI detection
	fmt.Println("\n1. Basic CLI Detection:")
	status, err := provider.DetectCLI()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Installed: %v\n", status.Installed)
		if status.Installed {
			fmt.Printf("Version: %s\n", status.Version)
			fmt.Printf("Path: %s\n", status.Path)
			fmt.Printf("Supported: %v\n", status.IsSupported)
			fmt.Printf("Min Version Required: %s\n", status.MinVersion)
		}
	}

	// Perform detailed CLI validation
	fmt.Println("\n2. Detailed CLI Validation:")
	result, err := provider.DetectCLIWithDetails()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	printValidationResult(result)

	// Demonstrate raw detector usage
	fmt.Println("\n3. Raw Detector Usage:")
	detector := cloud.NewAWSCLIDetector()
	detailedResult, err := detector.GetDetailedValidationResult()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Output as JSON for easy inspection
	if jsonOutput, err := json.MarshalIndent(detailedResult, "", "  "); err == nil {
		fmt.Println("Detailed Result (JSON):")
		fmt.Println(string(jsonOutput))
	}

	// Show installation instructions if CLI is not properly installed
	if result.Status != cloud.CLIStatusOK {
		fmt.Println("\n4. Installation Instructions:")
		fmt.Println(detector.GetInstallInstructions())
	}

	// Demonstrate S3 functionality if CLI is working
	if result.Status == cloud.CLIStatusOK {
		fmt.Println("\n5. S3 Manager Demo:")
		demonstrateS3Functionality(provider)
	}

	fmt.Println("\nDemo completed!")
}

func printValidationResult(result *cloud.AWSCLIValidationResult) {
	fmt.Printf("Platform: %s\n", result.Platform)
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Total Installations: %d\n", result.TotalInstallations)

	if result.SelectedInstallation != nil {
		fmt.Printf("Selected Installation:\n")
		fmt.Printf("  Path: %s\n", result.SelectedInstallation.Path)
		fmt.Printf("  Version: %s\n", result.SelectedInstallation.Version)
		fmt.Printf("  Major Version: %d\n", result.SelectedInstallation.MajorVersion)
		fmt.Printf("  Is V1: %v\n", result.SelectedInstallation.IsV1)
		fmt.Printf("  Install Method: %s\n", result.SelectedInstallation.InstallMethod)
		fmt.Printf("  Execution Time: %v\n", result.SelectedInstallation.ExecutionTime)
	}

	if result.ErrorMessage != "" {
		fmt.Printf("Error: %s\n", result.ErrorMessage)
	}

	if result.SuccessMessage != "" {
		fmt.Printf("Success: %s\n", result.SuccessMessage)
	}

	if len(result.Warnings) > 0 {
		fmt.Println("Warnings:")
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	if len(result.Recommendations) > 0 {
		fmt.Println("Recommendations:")
		for _, rec := range result.Recommendations {
			fmt.Printf("  - %s\n", rec)
		}
	}

	if len(result.AllInstallations) > 1 {
		fmt.Printf("All %d installations found:\n", len(result.AllInstallations))
		for i, installation := range result.AllInstallations {
			fmt.Printf("  %d. %s (v%s, %s)\n", i+1, installation.Path, installation.Version, installation.InstallMethod)
		}
	}
}

func demonstrateS3Functionality(provider *cloud.AWSProvider) {
	ctx := context.Background()
	s3Manager := provider.GetS3Manager()
	
	// Initialize logger and metrics
	logger := cloud.NewS3Logger(provider, true, cloud.LogLevelInfo)
	metrics := cloud.NewS3Metrics()
	s3Manager.SetLogger(logger)
	s3Manager.SetMetrics(metrics)
	
	fmt.Println("Testing S3 Manager functionality...")
	
	// Test 1: List buckets
	fmt.Println("\n- Listing S3 buckets:")
	buckets, err := s3Manager.ListBuckets(ctx, "us-east-1")
	if err != nil {
		fmt.Printf("  Error listing buckets: %v\n", err)
	} else {
		fmt.Printf("  Found %d buckets\n", len(buckets))
		for i, bucket := range buckets {
			if i < 3 { // Show only first 3 buckets
				fmt.Printf("    - %s (created: %s)\n", bucket.Name, bucket.CreationDate.Format("2006-01-02"))
			}
		}
		if len(buckets) > 3 {
			fmt.Printf("    ... and %d more\n", len(buckets)-3)
		}
	}
	
	// Test 2: Health check
	fmt.Println("\n- Running S3 health check:")
	healthChecker := cloud.NewS3HealthChecker(s3Manager, logger)
	testBucket := "apm-test-" + fmt.Sprintf("%d", time.Now().Unix())
	
	healthResult := healthChecker.CheckS3Health(ctx, testBucket, "us-east-1")
	fmt.Printf("  Health Status: %s\n", healthResult.Status)
	fmt.Printf("  Response Time: %v\n", healthResult.ResponseTime)
	
	if len(healthResult.Errors) > 0 {
		fmt.Println("  Errors:")
		for _, err := range healthResult.Errors {
			fmt.Printf("    - %s\n", err)
		}
	}
	
	if len(healthResult.Details) > 0 {
		fmt.Println("  Details:")
		for key, value := range healthResult.Details {
			fmt.Printf("    %s: %s\n", key, value)
		}
	}
	
	// Test 3: APM Configuration Management (demonstration)
	fmt.Println("\n- Testing APM configuration management:")
	
	// Create a sample Prometheus configuration
	prometheusConfig := map[string]interface{}{
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
	
	// Validate the configuration
	if err := s3Manager.ValidateAPMConfig("prometheus", prometheusConfig); err != nil {
		fmt.Printf("  Prometheus config validation failed: %v\n", err)
	} else {
		fmt.Println("  Prometheus config validation: PASSED")
	}
	
	// Test with different APM tool configurations
	testConfigs := map[string]map[string]interface{}{
		"grafana": {
			"server": map[string]interface{}{
				"http_port": 3000,
			},
			"database": map[string]interface{}{
				"type": "sqlite3",
			},
		},
		"jaeger": {
			"storage": map[string]interface{}{
				"type": "memory",
			},
		},
		"loki": {
			"auth_enabled": false,
			"server": map[string]interface{}{
				"http_listen_port": 3100,
			},
		},
	}
	
	for configType, config := range testConfigs {
		if err := s3Manager.ValidateAPMConfig(configType, config); err != nil {
			fmt.Printf("  %s config validation failed: %v\n", configType, err)
		} else {
			fmt.Printf("  %s config validation: PASSED\n", configType)
		}
	}
	
	// Test 4: Show metrics
	fmt.Println("\n- S3 Operation Metrics:")
	currentMetrics := metrics.GetMetrics()
	fmt.Printf("  Total Operations: %d\n", currentMetrics.TotalOperations)
	fmt.Printf("  Successful Operations: %d\n", currentMetrics.SuccessfulOps)
	fmt.Printf("  Failed Operations: %d\n", currentMetrics.FailedOps)
	if currentMetrics.TotalOperations > 0 {
		successRate := float64(currentMetrics.SuccessfulOps) / float64(currentMetrics.TotalOperations) * 100
		fmt.Printf("  Success Rate: %.1f%%\n", successRate)
	}
	fmt.Printf("  Average Response Time: %v\n", currentMetrics.AverageResponseTime)
	
	if len(currentMetrics.OperationCounts) > 0 {
		fmt.Println("  Operation Counts:")
		for operation, count := range currentMetrics.OperationCounts {
			fmt.Printf("    %s: %d\n", operation, count)
		}
	}
	
	// Test 5: Error handling demonstration
	fmt.Println("\n- Testing error handling:")
	
	// Try to access a bucket that doesn't exist
	_, err = s3Manager.GetBucket(ctx, "non-existent-bucket-12345", "us-east-1")
	if err != nil {
		fmt.Printf("  Expected error for non-existent bucket: %v\n", err)
		
		// Demonstrate error classification
		if cloudErr, ok := err.(*cloud.CloudError); ok {
			fmt.Printf("  Error Code: %s\n", cloudErr.Code)
			fmt.Printf("  Retryable: %v\n", cloudErr.Retryable)
			fmt.Printf("  Provider: %s\n", cloudErr.Provider)
		}
	}
	
	// Test 6: Retry mechanism demonstration
	fmt.Println("\n- Testing retry mechanism:")
	
	retryErr := cloud.RetryS3Operation(func() error {
		// Simulate a retryable error on first attempt
		currentMetrics := metrics.GetMetrics()
		if currentMetrics.TotalOperations < 2 {
			return &cloud.CloudError{
				Provider:  cloud.ProviderAWS,
				Operation: "TestOperation",
				Code:      "THROTTLED",
				Message:   "Simulated throttling error",
				Retryable: true,
				Timestamp: time.Now(),
			}
		}
		return nil // Success on subsequent attempts
	}, 3, 100*time.Millisecond, "TestRetry")
	
	if retryErr != nil {
		fmt.Printf("  Retry failed: %v\n", retryErr)
	} else {
		fmt.Println("  Retry mechanism: SUCCESS")
	}
	
	fmt.Println("\nS3 Manager demo completed!")
}