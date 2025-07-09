# Semgrep Analyzer Package

The `analyzer` package provides a Go interface for running Semgrep security analysis programmatically and integrating it with the APM's reporting and metrics system.

## Features

- **Programmatic Semgrep Execution**: Run Semgrep scans from Go code with full configuration control
- **Result Parsing**: Parse and structure Semgrep JSON output into Go types
- **Metrics Integration**: Export Prometheus metrics for security findings and scan performance
- **Report Generation**: Generate structured reports with findings grouped by severity, file, and category
- **Security Scoring**: Calculate security scores based on findings
- **Configurable**: Flexible configuration for rules, paths, timeouts, and more

## Installation

First, ensure Semgrep is installed:

```bash
# Using pip
pip install semgrep

# Using brew (macOS)
brew install semgrep

# Using Docker
docker pull returntocorp/semgrep
```

Then import the package:

```go
import "github.com/chaksack/apm/pkg/analyzer"
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    "github.com/chaksack/apm/pkg/analyzer"
)

func main() {
    // Create configuration
    config := analyzer.DefaultConfig()
    config.TargetPath = "./src"
    config.RuleSet = "auto"  // Use Semgrep's auto configuration
    
    // Create metrics collector (optional)
    metrics := analyzer.NewMetrics("apm", "semgrep")
    
    // Create analyzer
    semgrep, err := analyzer.NewAnalyzer(config, metrics)
    if err != nil {
        log.Fatal(err)
    }
    
    // Run analysis
    report, err := semgrep.Analyze(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    
    // Process results
    log.Printf("Found %d security issues", report.Summary.TotalFindings)
    log.Printf("Security score: %.2f/100", report.Summary.SecurityScore)
}
```

## Configuration

### Using Config Builder

```go
config, err := analyzer.NewConfigBuilder().
    WithTargetPath("./src").
    WithRuleSet("security").
    WithTimeout(15 * time.Minute).
    WithExcludePaths("vendor", "node_modules").
    WithMetrics("apm_security").
    Build()
```

### Configuration Options

```go
type Config struct {
    // Path to semgrep executable
    SemgrepPath string
    
    // Custom rules configuration file
    ConfigPath string
    
    // Built-in rule set (auto, security, etc.)
    RuleSet string
    
    // Target directory or file to scan
    TargetPath string
    
    // Paths to exclude from scanning
    ExcludePaths []string
    
    // File patterns to include
    IncludePatterns []string
    
    // Scan timeout
    Timeout time.Duration
    
    // Maximum memory usage (MB)
    MaxMemory int
    
    // Number of parallel jobs
    Jobs int
    
    // Enable metrics collection
    MetricsEnabled bool
    
    // Metrics namespace prefix
    MetricsPrefix string
    
    // Report output path
    ReportPath string
    
    // Minimum severity to report
    SeverityThreshold analyzer.SeverityLevel
}
```

## Metrics

The analyzer exports the following Prometheus metrics:

### Scan Metrics
- `apm_semgrep_scans_total`: Total number of scans performed
- `apm_semgrep_scan_duration_seconds`: Histogram of scan durations
- `apm_semgrep_scan_errors_total`: Total number of scan errors
- `apm_semgrep_files_scanned`: Histogram of files scanned per scan
- `apm_semgrep_lines_scanned`: Histogram of lines scanned per scan

### Finding Metrics
- `apm_semgrep_findings_total`: Total findings by category
- `apm_semgrep_findings_by_severity_total`: Total findings by severity
- `apm_semgrep_findings_by_rule_total`: Total findings by rule ID
- `apm_semgrep_findings_by_file`: Current findings per file

### Performance Metrics
- `apm_semgrep_memory_usage_bytes`: Memory usage during scan
- `apm_semgrep_cpu_usage_percent`: CPU usage percentage
- `apm_semgrep_cache_hit_rate`: Semgrep cache hit rate

### Security Metrics
- `apm_semgrep_security_score`: Overall security score (0-100)
- `apm_semgrep_vulnerability_score`: Vulnerability score by category

## Advanced Usage

### Custom Rules

```go
config := analyzer.DefaultConfig()
config.TargetPath = "./src"
config.CustomRules = []string{
    "./rules/custom-security.yaml",
    "./rules/company-standards.yaml",
}
```

### Filtering Results

```go
// Only report findings with severity ERROR or WARNING
config.SeverityThreshold = analyzer.SeverityWarning

// Custom filtering in report processing
report, _ := semgrep.Analyze(ctx)
criticalFindings := []analyzer.Finding{}
for _, finding := range report.Findings {
    if finding.Severity == "ERROR" {
        criticalFindings = append(criticalFindings, finding)
    }
}
```

### Integration with CI/CD

```go
// Example GitHub Actions integration
func runSecurityCheck() error {
    config := analyzer.DefaultConfig()
    config.TargetPath = "."
    config.ReportPath = "security-report.json"
    
    analyzer, _ := analyzer.NewAnalyzer(config, nil)
    report, err := analyzer.Analyze(context.Background())
    if err != nil {
        return err
    }
    
    // Fail if security score is too low
    if report.Summary.SecurityScore < 80.0 {
        return fmt.Errorf("security score too low: %.2f", report.Summary.SecurityScore)
    }
    
    // Fail if critical findings exist
    if report.Summary.BySeverity["ERROR"] > 0 {
        return fmt.Errorf("found %d critical security issues", report.Summary.BySeverity["ERROR"])
    }
    
    return nil
}
```

### Parsing External Results

```go
// Parse existing Semgrep JSON output
file, _ := os.Open("semgrep-results.json")
defer file.Close()

analyzer := &analyzer.SemgrepAnalyzer{}
result, err := analyzer.ParseResults(file)
if err != nil {
    log.Fatal(err)
}

report, _ := analyzer.GenerateReport(result)
```

## Report Structure

The analyzer generates structured reports with the following information:

```go
type AnalysisReport struct {
    // Scan metadata (timestamp, project, commit, etc.)
    Metadata *ScanMetadata
    
    // Summary statistics
    Summary *AnalysisSummary
    
    // All findings
    Findings []Finding
    
    // Findings grouped by category
    ByCategory map[string][]Finding
    
    // Findings grouped by severity
    BySeverity map[string][]Finding
    
    // Findings grouped by file
    ByFile map[string][]Finding
    
    // Any errors during scanning
    Errors []SemgrepError
}
```

## Error Handling

The analyzer provides detailed error information:

```go
report, err := analyzer.Analyze(ctx)
if err != nil {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        log.Printf("Scan timed out")
    case errors.Is(err, context.Canceled):
        log.Printf("Scan was canceled")
    default:
        log.Printf("Scan failed: %v", err)
    }
}

// Check for partial results (some files may have errors)
if report != nil && len(report.Errors) > 0 {
    for _, e := range report.Errors {
        log.Printf("Error in %s: %s", e.Path, e.Message)
    }
}
```

## Best Practices

1. **Configure Appropriate Timeouts**: Set timeouts based on codebase size
2. **Use Caching**: Semgrep caches results; reuse the cache directory for faster scans
3. **Exclude Unnecessary Paths**: Exclude vendor directories, build artifacts, etc.
4. **Monitor Metrics**: Use Prometheus metrics to track scan performance and trends
5. **Integrate with CI/CD**: Run scans on pull requests and fail on critical findings
6. **Custom Rules**: Write custom rules for company-specific security policies
7. **Regular Updates**: Keep Semgrep and rules updated for latest security checks

## Examples

See the `examples/` directory for complete examples:
- Basic security scanning
- CI/CD integration
- Custom rule implementation
- Metrics dashboard setup
- Report generation