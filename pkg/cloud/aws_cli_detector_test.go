package cloud

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

// TestLogger implements CLILogger for testing
type TestLogger struct {
	logs []string
}

func (l *TestLogger) Debug(msg string, fields ...interface{}) {
	l.logs = append(l.logs, fmt.Sprintf("DEBUG: %s", msg))
}

func (l *TestLogger) Info(msg string, fields ...interface{}) {
	l.logs = append(l.logs, fmt.Sprintf("INFO: %s", msg))
}

func (l *TestLogger) Warn(msg string, fields ...interface{}) {
	l.logs = append(l.logs, fmt.Sprintf("WARN: %s", msg))
}

func (l *TestLogger) Error(msg string, fields ...interface{}) {
	l.logs = append(l.logs, fmt.Sprintf("ERROR: %s", msg))
}

func TestAWSCLIDetector_parseVersionOutput(t *testing.T) {
	detector := NewAWSCLIDetector()

	testCases := []struct {
		name            string
		output          string
		expectedVersion string
		expectedMajor   int
		expectedIsV1    bool
	}{
		{
			name:            "AWS CLI v2 standard",
			output:          "aws-cli/2.4.29 Python/3.8.8 Linux/5.4.0-72-generic exe/x86_64.ubuntu.20 prompt/off",
			expectedVersion: "2.4.29",
			expectedMajor:   2,
			expectedIsV1:    false,
		},
		{
			name:            "AWS CLI v2 with build info",
			output:          "aws-cli/2.4.29-1.el8 Python/3.8.8 Linux/5.4.0-72-generic exe/x86_64.centos.8 prompt/off",
			expectedVersion: "2.4.29",
			expectedMajor:   2,
			expectedIsV1:    false,
		},
		{
			name:            "AWS CLI v1 legacy",
			output:          "aws-cli/1.20.30 Python/3.8.8 Linux/5.4.0-72-generic botocore/1.21.30",
			expectedVersion: "1.20.30",
			expectedMajor:   1,
			expectedIsV1:    true,
		},
		{
			name:            "AWS CLI v2 pre-release",
			output:          "aws-cli/2.5.0.dev0 Python/3.9.7 macOS/11.6 source/x86_64 prompt/off",
			expectedVersion: "2.5.0.dev0",
			expectedMajor:   2,
			expectedIsV1:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			version, major, isV1 := detector.parseVersionOutput(tc.output)

			if version != tc.expectedVersion {
				t.Errorf("Expected version %s, got %s", tc.expectedVersion, version)
			}

			if major != tc.expectedMajor {
				t.Errorf("Expected major version %d, got %d", tc.expectedMajor, major)
			}

			if isV1 != tc.expectedIsV1 {
				t.Errorf("Expected isV1 %v, got %v", tc.expectedIsV1, isV1)
			}
		})
	}
}

func TestAWSCLIDetector_isVersionNewer(t *testing.T) {
	detector := NewAWSCLIDetector()

	testCases := []struct {
		name     string
		version1 string
		version2 string
		expected bool
	}{
		{
			name:     "v2.5.0 is newer than v2.4.29",
			version1: "2.5.0",
			version2: "2.4.29",
			expected: true,
		},
		{
			name:     "v2.4.29 is not newer than v2.5.0",
			version1: "2.4.29",
			version2: "2.5.0",
			expected: false,
		},
		{
			name:     "v2.4.30 is newer than v2.4.29",
			version1: "2.4.30",
			version2: "2.4.29",
			expected: true,
		},
		{
			name:     "v2.0.0 is not newer than v2.0.0",
			version1: "2.0.0",
			version2: "2.0.0",
			expected: false,
		},
		{
			name:     "v2.0.1 is newer than v2.0.0",
			version1: "2.0.1",
			version2: "2.0.0",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.isVersionNewer(tc.version1, tc.version2)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for versions %s vs %s", tc.expected, result, tc.version1, tc.version2)
			}
		})
	}
}

func TestAWSCLIDetector_ValidateVersionSemantic(t *testing.T) {
	detector := NewAWSCLIDetector()

	testCases := []struct {
		name     string
		version  string
		expected bool
	}{
		{
			name:     "v2.0.0 meets minimum requirement",
			version:  "2.0.0",
			expected: true,
		},
		{
			name:     "v2.4.29 meets minimum requirement",
			version:  "2.4.29",
			expected: true,
		},
		{
			name:     "v1.20.30 does not meet minimum requirement",
			version:  "1.20.30",
			expected: false,
		},
		{
			name:     "v1.99.99 does not meet minimum requirement",
			version:  "1.99.99",
			expected: false,
		},
		{
			name:     "v3.0.0 meets minimum requirement",
			version:  "3.0.0",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.ValidateVersionSemantic(tc.version)
			if result != tc.expected {
				t.Errorf("Expected %v for version %s", tc.expected, tc.version)
			}
		})
	}
}

func TestAWSCLIDetector_detectInstallMethod(t *testing.T) {
	detector := NewAWSCLIDetector()

	testCases := []struct {
		name     string
		path     string
		expected string
	}{}

	// Platform-specific test cases
	switch runtime.GOOS {
	case "darwin":
		testCases = []struct {
			name     string
			path     string
			expected string
		}{
			{
				name:     "Homebrew installation",
				path:     "/opt/homebrew/bin/aws",
				expected: "homebrew",
			},
			{
				name:     "Official installer",
				path:     "/usr/local/aws-cli/bin/aws",
				expected: "installer",
			},
			{
				name:     "User local installation",
				path:     "/usr/local/bin/aws",
				expected: "homebrew",
			},
		}
	case "linux":
		testCases = []struct {
			name     string
			path     string
			expected string
		}{
			{
				name:     "Snap installation",
				path:     "/snap/bin/aws",
				expected: "snap",
			},
			{
				name:     "Official installer",
				path:     "/opt/aws-cli/bin/aws",
				expected: "installer",
			},
			{
				name:     "Package manager",
				path:     "/usr/bin/aws",
				expected: "package-manager",
			},
		}
	case "windows":
		testCases = []struct {
			name     string
			path     string
			expected string
		}{
			{
				name:     "Official installer",
				path:     "C:\\Program Files\\Amazon\\AWSCLIV2\\aws.exe",
				expected: "installer",
			},
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.detectInstallMethod(tc.path)
			if result != tc.expected {
				t.Errorf("Expected %s for path %s, got %s", tc.expected, tc.path, result)
			}
		})
	}
}

func TestAWSCLIDetector_selectBestInstallation(t *testing.T) {
	detector := NewAWSCLIDetector()

	installations := []AWSCLIInstallation{
		{
			Path:          "/usr/bin/aws",
			Version:       "1.20.30",
			MajorVersion:  1,
			IsV1:          true,
			ExecutionTime: 100 * time.Millisecond,
		},
		{
			Path:          "/usr/local/bin/aws",
			Version:       "2.4.29",
			MajorVersion:  2,
			IsV1:          false,
			ExecutionTime: 50 * time.Millisecond,
		},
		{
			Path:          "/opt/aws-cli/bin/aws",
			Version:       "2.5.0",
			MajorVersion:  2,
			IsV1:          false,
			ExecutionTime: 75 * time.Millisecond,
		},
	}

	best := detector.selectBestInstallation(installations)

	// Should select the newest version (2.5.0)
	expectedPath := "/opt/aws-cli/bin/aws"
	if best.Path != expectedPath {
		t.Errorf("Expected best installation path %s, got %s", expectedPath, best.Path)
	}

	expectedVersion := "2.5.0"
	if best.Version != expectedVersion {
		t.Errorf("Expected best installation version %s, got %s", expectedVersion, best.Version)
	}
}

func TestAWSCLIDetector_GetDetailedValidationResult(t *testing.T) {
	logger := &TestLogger{}
	detector := NewAWSCLIDetectorWithLogger(logger)

	// This test will run against the actual system
	// We can't control what's installed, so we test the structure
	result, err := detector.GetDetailedValidationResult()
	if err != nil {
		t.Fatalf("GetDetailedValidationResult failed: %v", err)
	}

	// Validate result structure
	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Platform == "" {
		t.Error("Platform should be set")
	}

	if result.Platform != runtime.GOOS {
		t.Errorf("Expected platform %s, got %s", runtime.GOOS, result.Platform)
	}

	if result.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}

	// If AWS CLI is installed, we should have installation details
	if result.TotalInstallations > 0 {
		if result.SelectedInstallation == nil {
			t.Error("SelectedInstallation should not be nil when installations found")
		}

		if len(result.AllInstallations) != result.TotalInstallations {
			t.Errorf("AllInstallations length (%d) should match TotalInstallations (%d)",
				len(result.AllInstallations), result.TotalInstallations)
		}
	}

	// Validate status is one of the expected values
	validStatuses := map[CLIStatusType]bool{
		CLIStatusOK:            true,
		CLIStatusNotFound:      true,
		CLIStatusVersionTooOld: true,
		CLIStatusDeprecated:    true,
		CLIStatusCorrupted:     true,
		CLIStatusMultiple:      true,
	}

	if !validStatuses[result.Status] {
		t.Errorf("Invalid status: %s", result.Status)
	}

	// If status is not OK, should have recommendations
	if result.Status != CLIStatusOK && len(result.Recommendations) == 0 {
		t.Error("Should have recommendations when status is not OK")
	}
}

func TestAWSProvider_DetectCLIWithDetails(t *testing.T) {
	provider, err := NewAWSProvider(nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	result, err := provider.DetectCLIWithDetails()
	if err != nil {
		t.Fatalf("DetectCLIWithDetails failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	// Check that provider status was updated
	if provider.cliStatus == nil {
		t.Error("Provider CLI status should be updated after detection")
	}

	// Validate consistency between detailed result and provider status
	if result.TotalInstallations > 0 {
		if !provider.cliStatus.Installed {
			t.Error("Provider status should show CLI as installed")
		}

		if result.SelectedInstallation != nil {
			if provider.cliStatus.Version != result.SelectedInstallation.Version {
				t.Errorf("Version mismatch: provider has %s, result has %s",
					provider.cliStatus.Version, result.SelectedInstallation.Version)
			}

			if provider.cliStatus.Path != result.SelectedInstallation.Path {
				t.Errorf("Path mismatch: provider has %s, result has %s",
					provider.cliStatus.Path, result.SelectedInstallation.Path)
			}
		}
	} else {
		if provider.cliStatus.Installed {
			t.Error("Provider status should show CLI as not installed")
		}
	}
}

// Benchmark tests for performance validation
func BenchmarkAWSCLIDetector_parseVersionOutput(b *testing.B) {
	detector := NewAWSCLIDetector()
	output := "aws-cli/2.4.29 Python/3.8.8 Linux/5.4.0-72-generic exe/x86_64.ubuntu.20 prompt/off"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.parseVersionOutput(output)
	}
}

func BenchmarkAWSCLIDetector_isVersionNewer(b *testing.B) {
	detector := NewAWSCLIDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.isVersionNewer("2.5.0", "2.4.29")
	}
}

// Example test showing how to use the enhanced detector
func ExampleAWSCLIDetector_usage() {
	// Create detector with custom logger
	logger := &TestLogger{}
	detector := NewAWSCLIDetectorWithLogger(logger)

	// Get comprehensive validation results
	result, err := detector.GetDetailedValidationResult()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print status
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Platform: %s\n", result.Platform)
	fmt.Printf("Installations found: %d\n", result.TotalInstallations)

	if result.SelectedInstallation != nil {
		fmt.Printf("Selected version: %s\n", result.SelectedInstallation.Version)
		fmt.Printf("Installation method: %s\n", result.SelectedInstallation.InstallMethod)
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
}
