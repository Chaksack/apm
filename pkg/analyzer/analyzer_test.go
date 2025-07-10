package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAnalyzer tests the creation of a new analyzer
func TestNewAnalyzer(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				TargetPath:        "/tmp",
				SemgrepPath:       "semgrep",
				RuleSet:           "auto",
				OutputFormat:      "json",
				Timeout:           30 * time.Minute,
				SeverityThreshold: SeverityInfo,
			},
			wantErr: false,
		},
		{
			name: "missing target path",
			config: &Config{
				SemgrepPath:  "semgrep",
				RuleSet:      "auto",
				OutputFormat: "json",
				Timeout:      30 * time.Minute,
			},
			wantErr: true,
			errMsg:  "target path is required",
		},
		{
			name: "invalid timeout",
			config: &Config{
				TargetPath:   "/tmp",
				SemgrepPath:  "semgrep",
				RuleSet:      "auto",
				OutputFormat: "json",
				Timeout:      -1 * time.Minute,
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name: "invalid output format",
			config: &Config{
				TargetPath:   "/tmp",
				SemgrepPath:  "semgrep",
				RuleSet:      "auto",
				OutputFormat: "invalid",
				Timeout:      30 * time.Minute,
			},
			wantErr: true,
			errMsg:  "invalid output format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer, err := NewAnalyzer(tt.config, nil)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, analyzer)
				assert.Equal(t, tt.config, analyzer.config)
			}
		})
	}
}

// TestParseResults tests parsing of Semgrep JSON output
func TestParseResults(t *testing.T) {
	analyzer := &SemgrepAnalyzer{}

	tests := []struct {
		name    string
		input   string
		wantErr bool
		verify  func(t *testing.T, result *SemgrepResult)
	}{
		{
			name: "valid semgrep output",
			input: `{
				"version": "1.0.0",
				"results": [
					{
						"check_id": "security.sql-injection",
						"path": "main.go",
						"line": 10,
						"column": 5,
						"end_line": 10,
						"end_column": 20,
						"message": "Potential SQL injection",
						"severity": "ERROR",
						"extra": {
							"lines": "query := fmt.Sprintf(\"SELECT * FROM users WHERE id = %s\", userInput)",
							"message": "User input is directly interpolated into SQL query",
							"fingerprint": "abc123"
						}
					}
				],
				"errors": [],
				"stats": {
					"total_time": 2.5,
					"files_count": 10,
					"rules_count": 50,
					"findings_count": 1,
					"lines_scanned": 1000
				}
			}`,
			wantErr: false,
			verify: func(t *testing.T, result *SemgrepResult) {
				assert.Equal(t, "1.0.0", result.Version)
				assert.Len(t, result.Results, 1)
				assert.Equal(t, "security.sql-injection", result.Results[0].CheckID)
				assert.Equal(t, "main.go", result.Results[0].Path)
				assert.Equal(t, 10, result.Results[0].Line)
				assert.Equal(t, "ERROR", result.Results[0].Severity)
				assert.NotNil(t, result.Stats)
				assert.Equal(t, 2.5, result.Stats.TotalTime)
			},
		},
		{
			name: "empty results",
			input: `{
				"version": "1.0.0",
				"results": [],
				"errors": []
			}`,
			wantErr: false,
			verify: func(t *testing.T, result *SemgrepResult) {
				assert.Equal(t, "1.0.0", result.Version)
				assert.Empty(t, result.Results)
				assert.Empty(t, result.Errors)
			},
		},
		{
			name: "with errors",
			input: `{
				"version": "1.0.0",
				"results": [],
				"errors": [
					{
						"code": 1,
						"level": "error",
						"type": "parse_error",
						"message": "Failed to parse file",
						"path": "broken.go"
					}
				]
			}`,
			wantErr: false,
			verify: func(t *testing.T, result *SemgrepResult) {
				assert.Len(t, result.Errors, 1)
				assert.Equal(t, "parse_error", result.Errors[0].Type)
				assert.Equal(t, "broken.go", result.Errors[0].Path)
			},
		},
		{
			name:    "invalid json",
			input:   `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := analyzer.ParseResults(reader)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.verify != nil {
					tt.verify(t, result)
				}
			}
		})
	}
}

// TestGenerateReport tests report generation
func TestGenerateReport(t *testing.T) {
	config := &Config{
		TargetPath:        "/test/project",
		SeverityThreshold: SeverityInfo,
	}
	analyzer := &SemgrepAnalyzer{config: config}

	input := &SemgrepResult{
		Version: "1.0.0",
		Results: []Finding{
			{
				CheckID:  "security.sql-injection",
				Path:     "main.go",
				Line:     10,
				Message:  "SQL injection vulnerability",
				Severity: "ERROR",
				Metadata: map[string]interface{}{
					"category": "injection",
				},
			},
			{
				CheckID:  "security.hardcoded-secret",
				Path:     "config.go",
				Line:     20,
				Message:  "Hardcoded secret found",
				Severity: "WARNING",
				Metadata: map[string]interface{}{
					"category": "secrets",
				},
			},
			{
				CheckID:  "security.weak-crypto",
				Path:     "crypto.go",
				Line:     30,
				Message:  "Weak cryptographic algorithm",
				Severity: "WARNING",
				Metadata: map[string]interface{}{
					"category": "crypto",
				},
			},
			{
				CheckID:  "best-practice.error-handling",
				Path:     "main.go",
				Line:     40,
				Message:  "Error not handled",
				Severity: "INFO",
			},
		},
		Stats: &ScanStats{
			TotalTime:     5.5,
			FilesCount:    3,
			RulesCount:    100,
			FindingsCount: 4,
			LinesScanned:  500,
		},
		Errors: []SemgrepError{},
	}

	report, err := analyzer.GenerateReport(input)
	require.NoError(t, err)
	assert.NotNil(t, report)

	// Verify metadata
	assert.Equal(t, "project", report.Metadata.ProjectName)
	assert.Equal(t, "/test/project", report.Metadata.Repository)

	// Verify summary
	assert.Equal(t, 4, report.Summary.TotalFindings)
	assert.Equal(t, 3, report.Summary.FilesAnalyzed)
	assert.Equal(t, 5.5, report.Summary.ExecutionTime)
	assert.Equal(t, 1, report.Summary.BySeverity["ERROR"])
	assert.Equal(t, 2, report.Summary.BySeverity["WARNING"])
	assert.Equal(t, 1, report.Summary.BySeverity["INFO"])

	// Verify groupings
	assert.Len(t, report.BySeverity["ERROR"], 1)
	assert.Len(t, report.BySeverity["WARNING"], 2)
	assert.Len(t, report.BySeverity["INFO"], 1)

	assert.Len(t, report.ByCategory["injection"], 1)
	assert.Len(t, report.ByCategory["secrets"], 1)
	assert.Len(t, report.ByCategory["crypto"], 1)
	assert.Len(t, report.ByCategory["uncategorized"], 1)

	assert.Len(t, report.ByFile["main.go"], 2)
	assert.Len(t, report.ByFile["config.go"], 1)
	assert.Len(t, report.ByFile["crypto.go"], 1)

	// Verify top issues
	assert.NotEmpty(t, report.Summary.TopIssues)
}

// TestCalculateSecurityScore tests security score calculation
func TestCalculateSecurityScore(t *testing.T) {
	analyzer := &SemgrepAnalyzer{}

	tests := []struct {
		name          string
		report        *AnalysisReport
		expectedScore float64
	}{
		{
			name: "no findings",
			report: &AnalysisReport{
				Summary: &AnalysisSummary{
					TotalFindings: 0,
					BySeverity:    map[string]int{},
				},
			},
			expectedScore: 100.0,
		},
		{
			name: "only info findings",
			report: &AnalysisReport{
				Summary: &AnalysisSummary{
					TotalFindings: 5,
					BySeverity: map[string]int{
						"INFO": 5,
					},
				},
			},
			expectedScore: 95.0, // 100 - (5 * 1)
		},
		{
			name: "mixed severity findings",
			report: &AnalysisReport{
				Summary: &AnalysisSummary{
					TotalFindings: 8,
					BySeverity: map[string]int{
						"ERROR":   2, // 2 * 10 = 20
						"WARNING": 3, // 3 * 5 = 15
						"INFO":    3, // 3 * 1 = 3
					},
				},
			},
			expectedScore: 62.0, // 100 - 38
		},
		{
			name: "many errors",
			report: &AnalysisReport{
				Summary: &AnalysisSummary{
					TotalFindings: 15,
					BySeverity: map[string]int{
						"ERROR": 15,
					},
				},
			},
			expectedScore: 0.0, // 100 - 150 = -50, but clamped to 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzer.calculateSecurityScore(tt.report)
			assert.Equal(t, tt.expectedScore, score)
		})
	}
}

// TestBuildCommand tests command building
func TestBuildCommand(t *testing.T) {
	config := &Config{
		SemgrepPath:     "semgrep",
		ConfigPath:      "/path/to/config.yml",
		TargetPath:      "/project",
		ExcludePaths:    []string{"vendor", "node_modules"},
		IncludePatterns: []string{"*.go", "*.js"},
		Timeout:         10 * time.Minute,
		MaxMemory:       1024,
		Jobs:            4,
		Verbose:         true,
		NoGitIgnore:     true,
		CustomRules:     []string{"rule1.yml", "rule2.yml"},
		MetricsEnabled:  true,
	}

	cmd := config.BuildCommand()

	assert.Contains(t, cmd, "semgrep")
	assert.Contains(t, cmd, "--config")
	assert.Contains(t, cmd, "/path/to/config.yml")
	assert.Contains(t, cmd, "--json")
	assert.Contains(t, cmd, "/project")
	assert.Contains(t, cmd, "--exclude")
	assert.Contains(t, cmd, "vendor")
	assert.Contains(t, cmd, "node_modules")
	assert.Contains(t, cmd, "--include")
	assert.Contains(t, cmd, "*.go")
	assert.Contains(t, cmd, "*.js")
	assert.Contains(t, cmd, "--timeout")
	assert.Contains(t, cmd, "600") // 10 minutes in seconds
	assert.Contains(t, cmd, "--max-memory")
	assert.Contains(t, cmd, "1024")
	assert.Contains(t, cmd, "--jobs")
	assert.Contains(t, cmd, "4")
	assert.Contains(t, cmd, "--verbose")
	assert.Contains(t, cmd, "--no-git-ignore")
	assert.Contains(t, cmd, "rule1.yml")
	assert.Contains(t, cmd, "rule2.yml")
	assert.Contains(t, cmd, "--metrics")
	assert.Contains(t, cmd, "on")
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				TargetPath:   "/tmp",
				OutputFormat: "json",
				Timeout:      1 * time.Minute,
				MaxMemory:    512,
			},
			wantErr: false,
		},
		{
			name: "missing target path",
			config: &Config{
				OutputFormat: "json",
				Timeout:      1 * time.Minute,
			},
			wantErr: true,
			errMsg:  "target path is required",
		},
		{
			name: "non-existent target path",
			config: &Config{
				TargetPath:   "/non/existent/path",
				OutputFormat: "json",
				Timeout:      1 * time.Minute,
			},
			wantErr: true,
			errMsg:  "target path does not exist",
		},
		{
			name: "invalid output format",
			config: &Config{
				TargetPath:   "/tmp",
				OutputFormat: "invalid",
				Timeout:      1 * time.Minute,
			},
			wantErr: true,
			errMsg:  "invalid output format",
		},
		{
			name: "negative timeout",
			config: &Config{
				TargetPath:   "/tmp",
				OutputFormat: "json",
				Timeout:      -1 * time.Minute,
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name: "negative max memory",
			config: &Config{
				TargetPath:   "/tmp",
				OutputFormat: "json",
				Timeout:      1 * time.Minute,
				MaxMemory:    -100,
			},
			wantErr: true,
			errMsg:  "max memory must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSeverityHandling tests severity normalization and comparison
func TestSeverityHandling(t *testing.T) {
	tests := []struct {
		input    string
		expected SeverityLevel
	}{
		{"ERROR", SeverityError},
		{"error", SeverityError},
		{"HIGH", SeverityError},
		{"WARNING", SeverityWarning},
		{"warning", SeverityWarning},
		{"MEDIUM", SeverityWarning},
		{"INFO", SeverityInfo},
		{"info", SeverityInfo},
		{"LOW", SeverityInfo},
		{"unknown", SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeSeverity(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}

	// Test severity comparison
	assert.Equal(t, 1, compareSeverity(SeverityError, SeverityWarning))
	assert.Equal(t, 1, compareSeverity(SeverityWarning, SeverityInfo))
	assert.Equal(t, 0, compareSeverity(SeverityError, SeverityError))
	assert.Equal(t, -1, compareSeverity(SeverityInfo, SeverityWarning))
}

// TestConfigBuilder tests the config builder
func TestConfigBuilder(t *testing.T) {
	config, err := NewConfigBuilder().
		WithTargetPath("/tmp").
		WithRuleSet("security").
		WithTimeout(5*time.Minute).
		WithExcludePaths("vendor", "node_modules").
		WithMetrics("test_prefix").
		Build()

	require.NoError(t, err)
	assert.Equal(t, "/tmp", config.TargetPath)
	assert.Equal(t, "security", config.RuleSet)
	assert.Equal(t, 5*time.Minute, config.Timeout)
	assert.Contains(t, config.ExcludePaths, "vendor")
	assert.Contains(t, config.ExcludePaths, "node_modules")
	assert.True(t, config.MetricsEnabled)
	assert.Equal(t, "test_prefix", config.MetricsPrefix)
}

// TestSaveReport tests saving reports to file
func TestSaveReport(t *testing.T) {
	tempDir := t.TempDir()
	reportPath := filepath.Join(tempDir, "report.json")

	analyzer := &SemgrepAnalyzer{}
	report := &AnalysisReport{
		Metadata: &ScanMetadata{
			Timestamp:   time.Now(),
			ProjectName: "test-project",
			Repository:  "/test/repo",
		},
		Summary: &AnalysisSummary{
			TotalFindings: 5,
			BySeverity: map[string]int{
				"ERROR":   2,
				"WARNING": 3,
			},
			SecurityScore: 75.0,
		},
	}

	err := analyzer.saveReport(report, reportPath)
	require.NoError(t, err)

	// Verify file exists and contains valid JSON
	data, err := os.ReadFile(reportPath)
	require.NoError(t, err)

	var savedReport AnalysisReport
	err = json.Unmarshal(data, &savedReport)
	require.NoError(t, err)

	assert.Equal(t, report.Summary.TotalFindings, savedReport.Summary.TotalFindings)
	assert.Equal(t, report.Summary.SecurityScore, savedReport.Summary.SecurityScore)
}

// TestAnalyzeWithContext tests context cancellation
func TestAnalyzeWithContext(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	config := &Config{
		TargetPath:   tempDir,
		SemgrepPath:  "sleep", // Use sleep command to simulate long-running process
		RuleSet:      "auto",
		OutputFormat: "json",
		Timeout:      30 * time.Second,
	}

	analyzer := &SemgrepAnalyzer{
		config:  config,
		metrics: &NopMetricsCollector{},
	}

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := analyzer.Analyze(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// MockMetricsCollector for testing
type MockMetricsCollector struct {
	scanCalled          bool
	scoreCalled         bool
	vulnerabilityCalled bool
	resourceCalled      bool
	cacheCalled         bool
	lastScore           float64
	lastDuration        time.Duration
	lastError           error
}

func (m *MockMetricsCollector) RecordScan(result *SemgrepResult, duration time.Duration, err error) {
	m.scanCalled = true
	m.lastDuration = duration
	m.lastError = err
}

func (m *MockMetricsCollector) RecordSecurityScore(score float64) {
	m.scoreCalled = true
	m.lastScore = score
}

func (m *MockMetricsCollector) RecordVulnerabilityScore(category string, score float64) {
	m.vulnerabilityCalled = true
}

func (m *MockMetricsCollector) RecordResourceUsage(memoryBytes int64, cpuPercent float64) {
	m.resourceCalled = true
}

func (m *MockMetricsCollector) RecordCacheHitRate(hitRate float64) {
	m.cacheCalled = true
}

// TestMetricsRecording tests that metrics are properly recorded
func TestMetricsRecording(t *testing.T) {
	mockMetrics := &MockMetricsCollector{}
	analyzer := &SemgrepAnalyzer{
		config: &Config{
			TargetPath:        "/tmp",
			SeverityThreshold: SeverityInfo,
		},
		metrics: mockMetrics,
	}

	// Test successful scan metrics
	result := &SemgrepResult{
		Results: []Finding{
			{CheckID: "test", Severity: "ERROR"},
		},
		Stats: &ScanStats{
			FilesCount:   10,
			LinesScanned: 1000,
		},
	}

	duration := 5 * time.Second
	mockMetrics.RecordScan(result, duration, nil)

	assert.True(t, mockMetrics.scanCalled)
	assert.Equal(t, duration, mockMetrics.lastDuration)
	assert.Nil(t, mockMetrics.lastError)

	// Test security score recording
	report := &AnalysisReport{
		Summary: &AnalysisSummary{
			TotalFindings: 1,
			BySeverity: map[string]int{
				"ERROR": 1,
			},
		},
	}

	score := analyzer.calculateSecurityScore(report)
	mockMetrics.RecordSecurityScore(score)

	assert.True(t, mockMetrics.scoreCalled)
	assert.Equal(t, 90.0, mockMetrics.lastScore) // 100 - 10 for 1 ERROR
}

// TestExtractCategory tests category extraction logic
func TestExtractCategory(t *testing.T) {
	tests := []struct {
		name     string
		finding  Finding
		expected string
	}{
		{
			name: "category from metadata",
			finding: Finding{
				CheckID: "test.rule",
				Metadata: map[string]interface{}{
					"category": "injection",
				},
			},
			expected: "injection",
		},
		{
			name: "category from rule ID",
			finding: Finding{
				CheckID: "security.injection.sql",
			},
			expected: "injection",
		},
		{
			name: "no category",
			finding: Finding{
				CheckID: "simple-rule",
			},
			expected: "uncategorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCategory(tt.finding)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCheckSemgrepInstalled tests the Semgrep installation check
func TestCheckSemgrepInstalled(t *testing.T) {
	// Test with a command that exists (echo)
	err := CheckSemgrepInstalled("echo")
	assert.Error(t, err) // Echo won't output "semgrep"

	// Test with non-existent command
	err = CheckSemgrepInstalled("/non/existent/command")
	assert.Error(t, err)
}

// BenchmarkParseResults benchmarks parsing performance
func BenchmarkParseResults(b *testing.B) {
	analyzer := &SemgrepAnalyzer{}

	// Create a large sample result
	findings := make([]interface{}, 100)
	for i := range findings {
		findings[i] = map[string]interface{}{
			"check_id":   fmt.Sprintf("rule-%d", i),
			"path":       fmt.Sprintf("file%d.go", i),
			"line":       i + 1,
			"column":     1,
			"end_line":   i + 1,
			"end_column": 80,
			"message":    "Test finding",
			"severity":   "WARNING",
		}
	}

	data := map[string]interface{}{
		"version": "1.0.0",
		"results": findings,
		"errors":  []interface{}{},
		"stats": map[string]interface{}{
			"total_time":    10.5,
			"files_count":   100,
			"lines_scanned": 10000,
		},
	}

	jsonData, _ := json.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(jsonData)
		_, _ = analyzer.ParseResults(reader)
	}
}

// BenchmarkGenerateReport benchmarks report generation performance
func BenchmarkGenerateReport(b *testing.B) {
	analyzer := &SemgrepAnalyzer{
		config: &Config{
			TargetPath:        "/test",
			SeverityThreshold: SeverityInfo,
		},
	}

	// Create a large result set
	findings := make([]Finding, 1000)
	for i := range findings {
		findings[i] = Finding{
			CheckID:  fmt.Sprintf("rule-%d", i),
			Path:     fmt.Sprintf("file%d.go", i%10),
			Line:     i + 1,
			Message:  "Test finding",
			Severity: []string{"ERROR", "WARNING", "INFO"}[i%3],
		}
	}

	result := &SemgrepResult{
		Results: findings,
		Stats: &ScanStats{
			FilesCount: 10,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = analyzer.GenerateReport(result)
	}
}
