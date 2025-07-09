package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds the configuration for the Semgrep analyzer
type Config struct {
	// SemgrepPath is the path to the semgrep executable
	SemgrepPath string

	// ConfigPath is the path to custom Semgrep rules/config
	ConfigPath string

	// RuleSet defines which rule set to use (auto, security, etc.)
	RuleSet string

	// TargetPath is the path to scan
	TargetPath string

	// ExcludePaths are paths to exclude from scanning
	ExcludePaths []string

	// IncludePatterns are file patterns to include
	IncludePatterns []string

	// OutputFormat specifies the output format (json, sarif, text)
	OutputFormat string

	// Timeout for the scan operation
	Timeout time.Duration

	// MaxMemory limits memory usage (in MB)
	MaxMemory int

	// Jobs specifies number of parallel jobs
	Jobs int

	// Verbose enables verbose output
	Verbose bool

	// NoGitIgnore disables .gitignore handling
	NoGitIgnore bool

	// Metrics configuration
	MetricsEnabled bool
	MetricsPrefix  string

	// Reporting configuration
	ReportPath string
	ReportHTML bool

	// Severity threshold (findings below this are ignored)
	SeverityThreshold SeverityLevel

	// Custom rules paths
	CustomRules []string

	// Cache directory for Semgrep
	CacheDir string
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		SemgrepPath:       "semgrep",
		RuleSet:           "auto",
		OutputFormat:      "json",
		Timeout:           30 * time.Minute,
		MaxMemory:         2048, // 2GB
		Jobs:              0,    // auto-detect
		MetricsEnabled:    true,
		MetricsPrefix:     "apm_semgrep",
		SeverityThreshold: SeverityInfo,
		CacheDir:          filepath.Join(os.TempDir(), "semgrep-cache"),
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.TargetPath == "" {
		return fmt.Errorf("target path is required")
	}

	if _, err := os.Stat(c.TargetPath); err != nil {
		return fmt.Errorf("target path does not exist: %w", err)
	}

	if c.ConfigPath != "" {
		if _, err := os.Stat(c.ConfigPath); err != nil {
			return fmt.Errorf("config path does not exist: %w", err)
		}
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if c.MaxMemory < 0 {
		return fmt.Errorf("max memory must be non-negative")
	}

	validOutputFormats := []string{"json", "sarif", "text", "junit-xml", "emacs", "vim"}
	if !contains(validOutputFormats, c.OutputFormat) {
		return fmt.Errorf("invalid output format: %s", c.OutputFormat)
	}

	return nil
}

// BuildCommand builds the semgrep command with arguments
func (c *Config) BuildCommand() []string {
	args := []string{c.SemgrepPath}

	// Add config or ruleset
	if c.ConfigPath != "" {
		args = append(args, "--config", c.ConfigPath)
	} else {
		args = append(args, "--config", c.RuleSet)
	}

	// Add output format
	args = append(args, "--json")

	// Add target path
	args = append(args, c.TargetPath)

	// Add exclude paths
	for _, exclude := range c.ExcludePaths {
		args = append(args, "--exclude", exclude)
	}

	// Add include patterns
	for _, include := range c.IncludePatterns {
		args = append(args, "--include", include)
	}

	// Add timeout
	args = append(args, "--timeout", fmt.Sprintf("%d", int(c.Timeout.Seconds())))

	// Add max memory
	if c.MaxMemory > 0 {
		args = append(args, "--max-memory", fmt.Sprintf("%d", c.MaxMemory))
	}

	// Add jobs
	if c.Jobs > 0 {
		args = append(args, "--jobs", fmt.Sprintf("%d", c.Jobs))
	}

	// Add verbose flag
	if c.Verbose {
		args = append(args, "--verbose")
	}

	// Add no-git-ignore flag
	if c.NoGitIgnore {
		args = append(args, "--no-git-ignore")
	}

	// Add custom rules
	for _, rule := range c.CustomRules {
		args = append(args, "--config", rule)
	}

	// Add metrics flag
	if c.MetricsEnabled {
		args = append(args, "--metrics", "on")
	}

	return args
}

// GetExcludeArgs returns exclude arguments for semgrep
func (c *Config) GetExcludeArgs() []string {
	var args []string
	for _, exclude := range c.ExcludePaths {
		args = append(args, "--exclude", exclude)
	}
	return args
}

// ShouldReportFinding checks if a finding should be reported based on severity
func (c *Config) ShouldReportFinding(severity string) bool {
	findingSeverity := normalizeSeverity(severity)
	return compareSeverity(findingSeverity, c.SeverityThreshold) >= 0
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func normalizeSeverity(severity string) SeverityLevel {
	switch strings.ToUpper(severity) {
	case "ERROR", "HIGH":
		return SeverityError
	case "WARNING", "MEDIUM":
		return SeverityWarning
	case "INFO", "LOW":
		return SeverityInfo
	default:
		return SeverityInfo
	}
}

func compareSeverity(a, b SeverityLevel) int {
	severityOrder := map[SeverityLevel]int{
		SeverityError:   3,
		SeverityWarning: 2,
		SeverityInfo:    1,
	}

	aVal := severityOrder[a]
	bVal := severityOrder[b]

	if aVal > bVal {
		return 1
	} else if aVal < bVal {
		return -1
	}
	return 0
}

// ConfigBuilder provides a fluent API for building configurations
type ConfigBuilder struct {
	config *Config
}

// NewConfigBuilder creates a new config builder
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: DefaultConfig(),
	}
}

// WithTargetPath sets the target path
func (b *ConfigBuilder) WithTargetPath(path string) *ConfigBuilder {
	b.config.TargetPath = path
	return b
}

// WithRuleSet sets the rule set
func (b *ConfigBuilder) WithRuleSet(ruleSet string) *ConfigBuilder {
	b.config.RuleSet = ruleSet
	return b
}

// WithTimeout sets the timeout
func (b *ConfigBuilder) WithTimeout(timeout time.Duration) *ConfigBuilder {
	b.config.Timeout = timeout
	return b
}

// WithExcludePaths sets paths to exclude
func (b *ConfigBuilder) WithExcludePaths(paths ...string) *ConfigBuilder {
	b.config.ExcludePaths = append(b.config.ExcludePaths, paths...)
	return b
}

// WithMetrics enables metrics with the given prefix
func (b *ConfigBuilder) WithMetrics(prefix string) *ConfigBuilder {
	b.config.MetricsEnabled = true
	b.config.MetricsPrefix = prefix
	return b
}

// Build validates and returns the configuration
func (b *ConfigBuilder) Build() (*Config, error) {
	if err := b.config.Validate(); err != nil {
		return nil, err
	}
	return b.config, nil
}
