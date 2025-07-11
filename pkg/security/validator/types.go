package validator

import (
	"regexp"
	"strings"
)

// ValidationRule represents a validation rule
type ValidationRule struct {
	Field       string
	Required    bool
	MinLength   int
	MaxLength   int
	Pattern     string
	PatternDesc string
	MinValue    *float64
	MaxValue    *float64
	Enum        []string
	Custom      func(value interface{}) error
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

// Error implements error interface
func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return ""
	}

	var messages []string
	for _, err := range v {
		messages = append(messages, err.Field+": "+err.Message)
	}
	return "validation failed: " + strings.Join(messages, ", ")
}

// Common patterns for validation
var (
	// AlphanumericPattern allows letters, numbers, hyphens, and underscores
	AlphanumericPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	// NamePattern allows letters, numbers, spaces, hyphens, and underscores
	NamePattern = regexp.MustCompile(`^[a-zA-Z0-9 _-]+$`)

	// EmailPattern validates email addresses
	EmailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	// URLPattern validates URLs
	URLPattern = regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+(?:\.[a-zA-Z]{2,})+(?:/[^/]*)*$`)

	// UUIDPattern validates UUIDs
	UUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

	// VersionPattern validates semantic versions
	VersionPattern = regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:-[a-zA-Z0-9]+)?$`)

	// FilenamePattern validates safe filenames
	FilenamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

	// PathPattern validates safe paths (no .. or absolute paths)
	PathPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9./_-]*$`)
)

// SanitizationRule represents a sanitization rule
type SanitizationRule struct {
	Field           string
	TrimSpace       bool
	ToLower         bool
	ToUpper         bool
	RemoveHTML      bool
	RemoveScript    bool
	MaxLength       int
	AllowedChars    string
	ReplacePatterns map[string]string
	Custom          func(value string) string
}

// RequestValidationRules represents validation rules for different request parts
type RequestValidationRules struct {
	Body    map[string][]ValidationRule
	Query   map[string][]ValidationRule
	Params  map[string][]ValidationRule
	Headers map[string][]ValidationRule
}

// Common validation rules for reuse
var (
	// IDValidation validates IDs
	IDValidation = ValidationRule{
		Required:    true,
		Pattern:     UUIDPattern.String(),
		PatternDesc: "must be a valid UUID",
	}

	// NameValidation validates names
	NameValidation = ValidationRule{
		Required:    true,
		MinLength:   3,
		MaxLength:   100,
		Pattern:     NamePattern.String(),
		PatternDesc: "must contain only letters, numbers, spaces, hyphens, and underscores",
	}

	// EmailValidation validates email addresses
	EmailValidation = ValidationRule{
		Required:    true,
		Pattern:     EmailPattern.String(),
		PatternDesc: "must be a valid email address",
	}

	// URLValidation validates URLs
	URLValidation = ValidationRule{
		Required:    true,
		Pattern:     URLPattern.String(),
		PatternDesc: "must be a valid HTTP(S) URL",
	}

	// VersionValidation validates version strings
	VersionValidation = ValidationRule{
		Required:    true,
		Pattern:     VersionPattern.String(),
		PatternDesc: "must be a valid semantic version (e.g., 1.2.3)",
	}
)

// DeploymentValidationRules provides validation for deployment requests
var DeploymentValidationRules = RequestValidationRules{
	Body: map[string][]ValidationRule{
		"name":    {NameValidation},
		"version": {VersionValidation},
		"environment": {
			{
				Required: true,
				Enum:     []string{"development", "staging", "production"},
			},
		},
		"replicas": {
			{
				Required: true,
				MinValue: floatPtr(1),
				MaxValue: floatPtr(100),
			},
		},
	},
}

// ConfigValidationRules provides validation for configuration requests
var ConfigValidationRules = RequestValidationRules{
	Body: map[string][]ValidationRule{
		"name": {NameValidation},
		"type": {
			{
				Required: true,
				Enum:     []string{"prometheus", "grafana", "jaeger", "loki", "alertmanager"},
			},
		},
		"content": {
			{
				Required:  true,
				MinLength: 1,
				MaxLength: 1024 * 1024, // 1MB
			},
		},
	},
}

// APIKeyValidationRules provides validation for API key requests
var APIKeyValidationRules = RequestValidationRules{
	Body: map[string][]ValidationRule{
		"name": {NameValidation},
		"roles": {
			{
				Required: true,
				Custom: func(value interface{}) error {
					roles, ok := value.([]string)
					if !ok || len(roles) == 0 {
						return ValidationErrors{{Field: "roles", Message: "must be a non-empty array of strings"}}
					}
					return nil
				},
			},
		},
		"expires_in_days": {
			{
				MinValue: floatPtr(1),
				MaxValue: floatPtr(365),
			},
		},
	},
}

// floatPtr returns a pointer to a float64
func floatPtr(f float64) *float64 {
	return &f
}
