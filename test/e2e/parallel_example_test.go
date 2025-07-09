// Package e2e provides end-to-end tests for the APM stack
package e2e

import (
	"testing"
	"time"
)

// Example of how to use the parallel test runner in regular test files

func TestExampleParallelExecution(t *testing.T) {
	// Create a parallel test runner
	runner := NewParallelTestRunner(
		4,             // max workers
		2,             // retry count
		5*time.Minute, // timeout
	)

	// Define custom test functions
	tests := []TestFunc{
		{
			Name:     "Example_QuickTest",
			Category: "example",
			Priority: 10,
			Function: func(t *testing.T) error {
				// Quick test that should pass
				if err := WaitForService(nil, "http://localhost:9090/-/ready", 30*time.Second); err != nil {
					return err
				}
				return nil
			},
		},
		{
			Name:       "Example_CriticalTest",
			Category:   "example",
			Priority:   10,
			Required:   true, // If this fails, stop all other tests
			MaxRetries: 0,    // Don't retry critical tests
			Function: func(t *testing.T) error {
				// Critical test that must pass
				return TestPrometheusMetricsCollection(t)
			},
		},
		{
			Name:       "Example_FlakyTest",
			Category:   "example",
			Priority:   5,
			MaxRetries: 3, // This test is flaky, allow more retries
			Function: func(t *testing.T) error {
				// Test that might fail occasionally
				return TestLokiLogAggregation(t)
			},
		},
	}

	// Run the tests
	report, err := runner.Run(tests)
	if err != nil {
		t.Fatalf("Failed to run tests: %v", err)
	}

	// Save reports
	if err := runner.SaveReport(report, "test-results/example-report.json"); err != nil {
		t.Errorf("Failed to save JSON report: %v", err)
	}

	if err := runner.SaveHTMLReport(report, "test-results/example-report.html"); err != nil {
		t.Errorf("Failed to save HTML report: %v", err)
	}

	// Print summary
	runner.PrintSummary(report)

	// Check if any tests failed
	if report.FailedTests > 0 {
		t.Fatalf("%d tests failed", report.FailedTests)
	}
}

// Example of running a specific scenario programmatically
func TestExampleLoadScenario(t *testing.T) {
	runner := NewParallelTestRunner(8, 1, 10*time.Minute)

	// Get predefined load test scenario
	tests := GetLoadTestScenario()

	// Run the scenario
	report, err := runner.Run(tests)
	if err != nil {
		t.Fatalf("Failed to run load tests: %v", err)
	}

	// Analyze results
	if report.FailedTests > 0 {
		t.Errorf("Load tests failed: %d out of %d", report.FailedTests, report.TotalTests)
	}

	// Check performance metrics
	if report.Resources.MaxCPU > 80.0 {
		t.Logf("Warning: High CPU usage detected: %.2f%%", report.Resources.MaxCPU)
	}

	if report.Resources.MaxMemory > 1024.0 {
		t.Logf("Warning: High memory usage detected: %.2f MB", report.Resources.MaxMemory)
	}
}

// Example of custom test organization
func TestExampleCustomScenario(t *testing.T) {
	runner := NewParallelTestRunner(4, 2, 30*time.Minute)

	// Combine tests from different scenarios
	tests := make([]TestFunc, 0)

	// Add critical monitoring tests
	monitoringTests := GetMonitoringPipelineScenario()
	for i := 0; i < 3 && i < len(monitoringTests); i++ {
		tests = append(tests, monitoringTests[i])
	}

	// Add some security tests
	securityTests := GetSecurityScanScenario()
	for i := 0; i < 2 && i < len(securityTests); i++ {
		tests = append(tests, securityTests[i])
	}

	// Add custom test
	tests = append(tests, TestFunc{
		Name:     "Custom_ValidationTest",
		Category: "custom",
		Priority: 8,
		Function: func(t *testing.T) error {
			// Custom validation logic
			return nil
		},
	})

	// Run the custom scenario
	report, err := runner.Run(tests)
	if err != nil {
		t.Fatalf("Failed to run custom scenario: %v", err)
	}

	// Custom success criteria
	successRate := float64(report.PassedTests) / float64(report.TotalTests) * 100
	if successRate < 95.0 {
		t.Errorf("Success rate too low: %.2f%% (expected >= 95%%)", successRate)
	}
}

// Wrapper functions to convert existing tests for parallel execution
func TestPrometheusMetricsCollection(t *testing.T) error {
	// This is a wrapper that adapts the existing test for parallel execution
	// In a real implementation, you would refactor the original test
	// to return an error instead of using testing.T directly

	err := WaitForService(nil, "http://localhost:9090/-/ready", 30*time.Second)
	if err != nil {
		return err
	}

	// Run the actual test logic
	// For now, we'll just do a basic check
	return nil
}

func TestLokiLogAggregation(t *testing.T) error {
	// Another wrapper example
	err := WaitForService(nil, "http://localhost:3100/ready", 30*time.Second)
	if err != nil {
		return err
	}

	// Test log aggregation
	testLog := `{"level":"info","msg":"Parallel test log","source":"parallel-runner"}`
	return SendLogToLoki("http://localhost:3100", testLog)
}
