package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Analyzer is the main interface for Semgrep analysis
type Analyzer interface {
	// Analyze runs Semgrep analysis on the configured target
	Analyze(ctx context.Context) (*AnalysisReport, error)

	// AnalyzeWithConfig runs analysis with a custom configuration
	AnalyzeWithConfig(ctx context.Context, config *Config) (*AnalysisReport, error)

	// ParseResults parses raw Semgrep output
	ParseResults(reader io.Reader) (*SemgrepResult, error)

	// GenerateReport generates a formatted report from results
	GenerateReport(result *SemgrepResult) (*AnalysisReport, error)
}

// SemgrepAnalyzer implements the Analyzer interface
type SemgrepAnalyzer struct {
	config   *Config
	metrics  MetricsCollector
	reporter Reporter
}

// Reporter interface for generating reports
type Reporter interface {
	GenerateReport(report *AnalysisReport) error
	GenerateHTMLReport(report *AnalysisReport, outputPath string) error
}

// NewAnalyzer creates a new Semgrep analyzer
func NewAnalyzer(config *Config, metrics MetricsCollector) (*SemgrepAnalyzer, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	if metrics == nil {
		metrics = &NopMetricsCollector{}
	}

	return &SemgrepAnalyzer{
		config:  config,
		metrics: metrics,
	}, nil
}

// Analyze runs Semgrep analysis with the default configuration
func (a *SemgrepAnalyzer) Analyze(ctx context.Context) (*AnalysisReport, error) {
	return a.AnalyzeWithConfig(ctx, a.config)
}

// AnalyzeWithConfig runs Semgrep analysis with a custom configuration
func (a *SemgrepAnalyzer) AnalyzeWithConfig(ctx context.Context, config *Config) (*AnalysisReport, error) {
	start := time.Now()

	// Ensure cache directory exists
	if err := os.MkdirAll(config.CacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Run Semgrep
	result, err := a.runSemgrep(ctx, config)
	duration := time.Since(start)

	// Record metrics
	a.metrics.RecordScan(result, duration, err)

	if err != nil {
		return nil, fmt.Errorf("semgrep execution failed: %w", err)
	}

	// Generate report
	report, err := a.GenerateReport(result)
	if err != nil {
		return nil, fmt.Errorf("failed to generate report: %w", err)
	}

	// Calculate and record security score
	score := a.calculateSecurityScore(report)
	a.metrics.RecordSecurityScore(score)
	report.Summary.SecurityScore = score

	// Save report if configured
	if config.ReportPath != "" {
		if err := a.saveReport(report, config.ReportPath); err != nil {
			return nil, fmt.Errorf("failed to save report: %w", err)
		}
	}

	return report, nil
}

// runSemgrep executes the Semgrep command
func (a *SemgrepAnalyzer) runSemgrep(ctx context.Context, config *Config) (*SemgrepResult, error) {
	args := config.BuildCommand()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SEMGREP_CACHE_DIR=%s", config.CacheDir),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Check for context cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Parse output even if there was an error (Semgrep returns non-zero for findings)
	result, parseErr := a.ParseResults(&stdout)
	if parseErr != nil {
		// If we can't parse the output and there was an execution error, return both
		if err != nil {
			return nil, fmt.Errorf("execution error: %v, stderr: %s", err, stderr.String())
		}
		return nil, parseErr
	}

	// Add any stderr content as errors
	if stderr.Len() > 0 && config.Verbose {
		fmt.Fprintf(os.Stderr, "Semgrep stderr: %s\n", stderr.String())
	}

	return result, nil
}

// ParseResults parses Semgrep JSON output
func (a *SemgrepAnalyzer) ParseResults(reader io.Reader) (*SemgrepResult, error) {
	var result SemgrepResult
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse Semgrep output: %w", err)
	}
	return &result, nil
}

// GenerateReport generates a structured report from Semgrep results
func (a *SemgrepAnalyzer) GenerateReport(result *SemgrepResult) (*AnalysisReport, error) {
	report := &AnalysisReport{
		Metadata: &ScanMetadata{
			Timestamp:   time.Now(),
			ProjectName: filepath.Base(a.config.TargetPath),
			Repository:  a.config.TargetPath,
		},
		Summary: &AnalysisSummary{
			TotalFindings: len(result.Results),
			BySeverity:    make(map[string]int),
			TopIssues:     []string{},
		},
		Findings:   result.Results,
		ByCategory: make(map[string][]Finding),
		BySeverity: make(map[string][]Finding),
		ByFile:     make(map[string][]Finding),
		Errors:     result.Errors,
	}

	// Populate statistics
	if result.Stats != nil {
		report.Summary.FilesAnalyzed = result.Stats.FilesCount
		report.Summary.ExecutionTime = result.Stats.TotalTime
	}

	// Group findings
	issueCounts := make(map[string]int)
	for _, finding := range result.Results {
		// Filter by severity threshold
		if !a.config.ShouldReportFinding(finding.Severity) {
			continue
		}

		// By severity
		severity := string(normalizeSeverity(finding.Severity))
		report.BySeverity[severity] = append(report.BySeverity[severity], finding)
		report.Summary.BySeverity[severity]++

		// By category (extract from rule ID or metadata)
		category := extractCategory(finding)
		report.ByCategory[category] = append(report.ByCategory[category], finding)

		// By file
		report.ByFile[finding.Path] = append(report.ByFile[finding.Path], finding)

		// Count issues
		issueCounts[finding.CheckID]++
	}

	// Find top issues
	report.Summary.TopIssues = getTopIssues(issueCounts, 5)

	return report, nil
}

// calculateSecurityScore calculates a security score based on findings
func (a *SemgrepAnalyzer) calculateSecurityScore(report *AnalysisReport) float64 {
	if report.Summary.TotalFindings == 0 {
		return 100.0
	}

	// Weight findings by severity
	weights := map[string]float64{
		"ERROR":   10.0,
		"WARNING": 5.0,
		"INFO":    1.0,
	}

	totalWeight := 0.0
	for severity, count := range report.Summary.BySeverity {
		weight := weights[severity]
		totalWeight += weight * float64(count)
	}

	// Calculate score (0-100)
	// Assuming 1 error = -10 points, 1 warning = -5 points, 1 info = -1 point
	score := 100.0 - totalWeight
	if score < 0 {
		score = 0
	}

	return score
}

// saveReport saves the report to a file
func (a *SemgrepAnalyzer) saveReport(report *AnalysisReport, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	// Marshal report to JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	return nil
}

// Helper functions

func extractCategory(finding Finding) string {
	// Try to extract category from metadata
	if finding.Metadata != nil {
		if category, ok := finding.Metadata["category"].(string); ok {
			return category
		}
	}

	// Extract from rule ID (e.g., "security.injection.sql" -> "injection")
	parts := strings.Split(finding.CheckID, ".")
	if len(parts) > 1 {
		return parts[1]
	}

	return "uncategorized"
}

func getTopIssues(issueCounts map[string]int, limit int) []string {
	// Create a slice of issue-count pairs
	type issueCount struct {
		issue string
		count int
	}

	issues := make([]issueCount, 0, len(issueCounts))
	for issue, count := range issueCounts {
		issues = append(issues, issueCount{issue, count})
	}

	// Sort by count (descending)
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].count > issues[j].count
	})

	// Get top N
	result := make([]string, 0, limit)
	for i := 0; i < limit && i < len(issues); i++ {
		result = append(result, fmt.Sprintf("%s (%d)", issues[i].issue, issues[i].count))
	}

	return result
}

// CheckSemgrepInstalled verifies that Semgrep is installed and accessible
func CheckSemgrepInstalled(semgrepPath string) error {
	cmd := exec.Command(semgrepPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("semgrep not found or not executable: %w", err)
	}

	if !strings.Contains(string(output), "semgrep") {
		return fmt.Errorf("unexpected semgrep version output: %s", output)
	}

	return nil
}
