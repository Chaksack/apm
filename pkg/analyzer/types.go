package analyzer

import (
	"time"
)

// SemgrepResult represents the overall result from a Semgrep scan
type SemgrepResult struct {
	Version      string         `json:"version"`
	Results      []Finding      `json:"results"`
	Errors       []SemgrepError `json:"errors"`
	Stats        *ScanStats     `json:"stats,omitempty"`
	ScanMetadata *ScanMetadata  `json:"scan_metadata,omitempty"`
}

// Finding represents a single security finding from Semgrep
type Finding struct {
	CheckID   string                 `json:"check_id"`
	Path      string                 `json:"path"`
	Line      int                    `json:"line"`
	Column    int                    `json:"column"`
	EndLine   int                    `json:"end_line"`
	EndColumn int                    `json:"end_column"`
	Message   string                 `json:"message"`
	Severity  string                 `json:"severity"`
	Extra     *FindingExtra          `json:"extra,omitempty"`
	FixRegex  *FixRegex              `json:"fix_regex,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// FindingExtra contains additional information about a finding
type FindingExtra struct {
	Lines       string                 `json:"lines"`
	Message     string                 `json:"message"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Metavars    map[string]Metavar     `json:"metavars,omitempty"`
	Fingerprint string                 `json:"fingerprint"`
	IsIgnored   bool                   `json:"is_ignored"`
}

// Metavar represents a metavariable in a Semgrep finding
type Metavar struct {
	Start           Position    `json:"start"`
	End             Position    `json:"end"`
	AbstractContent string      `json:"abstract_content"`
	PropagatedValue interface{} `json:"propagated_value,omitempty"`
}

// Position represents a position in source code
type Position struct {
	Line   int `json:"line"`
	Column int `json:"col"`
	Offset int `json:"offset"`
}

// FixRegex contains information for automated fixes
type FixRegex struct {
	Regex       string `json:"regex"`
	Replacement string `json:"replacement"`
	Count       int    `json:"count"`
}

// SemgrepError represents an error that occurred during scanning
type SemgrepError struct {
	Code    int                    `json:"code"`
	Level   string                 `json:"level"`
	Type    string                 `json:"type"`
	Message string                 `json:"message"`
	Path    string                 `json:"path,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ScanStats contains statistics about the scan
type ScanStats struct {
	TotalTime     float64 `json:"total_time"`
	RulesCount    int     `json:"rules_count"`
	FilesCount    int     `json:"files_count"`
	TargetCount   int     `json:"target_count"`
	FindingsCount int     `json:"findings_count"`
	LinesScanned  int     `json:"lines_scanned"`
}

// ScanMetadata contains metadata about the scan
type ScanMetadata struct {
	Timestamp   time.Time `json:"timestamp"`
	ProjectName string    `json:"project_name"`
	Repository  string    `json:"repository"`
	Branch      string    `json:"branch"`
	CommitHash  string    `json:"commit_hash"`
}

// SeverityLevel represents the severity level of a finding
type SeverityLevel string

const (
	SeverityError   SeverityLevel = "ERROR"
	SeverityWarning SeverityLevel = "WARNING"
	SeverityInfo    SeverityLevel = "INFO"
)

// AnalysisReport represents a structured report of the analysis
type AnalysisReport struct {
	Metadata   *ScanMetadata        `json:"metadata"`
	Summary    *AnalysisSummary     `json:"summary"`
	Findings   []Finding            `json:"findings"`
	ByCategory map[string][]Finding `json:"by_category"`
	BySeverity map[string][]Finding `json:"by_severity"`
	ByFile     map[string][]Finding `json:"by_file"`
	Errors     []SemgrepError       `json:"errors,omitempty"`
}

// AnalysisSummary provides a summary of the analysis results
type AnalysisSummary struct {
	TotalFindings int            `json:"total_findings"`
	BySeverity    map[string]int `json:"by_severity"`
	TopIssues     []string       `json:"top_issues"`
	FilesAnalyzed int            `json:"files_analyzed"`
	ExecutionTime float64        `json:"execution_time"`
	SecurityScore float64        `json:"security_score"`
}
