package security

import (
	"fmt"
	"regexp"
	"strings"
)

// SanitizeError removes sensitive information from error messages
func SanitizeError(err error) error {
	if err == nil {
		return nil
	}

	msg := err.Error()

	// Remove file paths that might expose system structure
	msg = sanitizeFilePaths(msg)

	// Remove potential credentials
	msg = sanitizeCredentials(msg)

	// Remove URLs with potential credentials
	msg = sanitizeURLs(msg)

	// Remove IP addresses
	msg = sanitizeIPs(msg)

	return fmt.Errorf("%s", msg)
}

// SanitizeLogMessage removes sensitive information from log messages
func SanitizeLogMessage(msg string) string {
	msg = sanitizeFilePaths(msg)
	msg = sanitizeCredentials(msg)
	msg = sanitizeURLs(msg)
	msg = sanitizeIPs(msg)
	msg = sanitizeSecrets(msg)

	return msg
}

// sanitizeFilePaths removes absolute file paths
func sanitizeFilePaths(msg string) string {
	// Replace absolute paths with relative ones
	patterns := []string{
		`/home/[^/\s]+`,
		`/Users/[^/\s]+`,
		`C:\\Users\\[^\\s]+`,
		`/var/[^/\s]+/[^/\s]+`,
		`/etc/[^/\s]+`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		msg = re.ReplaceAllString(msg, "<path>")
	}

	return msg
}

// sanitizeCredentials removes potential credentials
func sanitizeCredentials(msg string) string {
	// Patterns for various credential formats
	patterns := map[string]string{
		// AWS patterns
		`AKIA[0-9A-Z]{16}`:                  "<AWS_ACCESS_KEY>",
		`[0-9a-zA-Z/+=]{40}`:                "<AWS_SECRET_KEY>",
		`arn:aws:[^:]+:[^:]+:[^:]+:[^:\s]+`: "<AWS_ARN>",

		// API keys and tokens
		`[a-zA-Z0-9]{32,}`:             "<API_KEY>",
		`Bearer\s+[a-zA-Z0-9\-._~+/]+`: "Bearer <TOKEN>",

		// Generic patterns
		`password\s*[:=]\s*[^\s]+`:    "password=<REDACTED>",
		`token\s*[:=]\s*[^\s]+`:       "token=<REDACTED>",
		`api[_-]?key\s*[:=]\s*[^\s]+`: "api_key=<REDACTED>",
		`secret\s*[:=]\s*[^\s]+`:      "secret=<REDACTED>",

		// Email addresses
		`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`: "<EMAIL>",
	}

	for pattern, replacement := range patterns {
		re := regexp.MustCompile(pattern)
		msg = re.ReplaceAllString(msg, replacement)
	}

	return msg
}

// sanitizeURLs removes credentials from URLs
func sanitizeURLs(msg string) string {
	// Pattern for URLs with credentials
	urlPattern := regexp.MustCompile(`(https?://)([^:]+):([^@]+)@([^\s]+)`)
	msg = urlPattern.ReplaceAllString(msg, "$1<user>:<pass>@$4")

	// Pattern for connection strings
	connPattern := regexp.MustCompile(`(mongodb|postgresql|mysql|redis)://([^:]+):([^@]+)@([^\s]+)`)
	msg = connPattern.ReplaceAllString(msg, "$1://<user>:<pass>@$4")

	return msg
}

// sanitizeIPs removes IP addresses
func sanitizeIPs(msg string) string {
	// IPv4 pattern
	ipv4Pattern := regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`)
	msg = ipv4Pattern.ReplaceAllString(msg, "<IP>")

	// IPv6 pattern (simplified)
	ipv6Pattern := regexp.MustCompile(`\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`)
	msg = ipv6Pattern.ReplaceAllString(msg, "<IPv6>")

	return msg
}

// sanitizeSecrets removes various secret patterns
func sanitizeSecrets(msg string) string {
	// JWT tokens
	jwtPattern := regexp.MustCompile(`eyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`)
	msg = jwtPattern.ReplaceAllString(msg, "<JWT_TOKEN>")

	// Credit card numbers
	ccPattern := regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)
	msg = ccPattern.ReplaceAllString(msg, "<CREDIT_CARD>")

	// Social Security Numbers
	ssnPattern := regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	msg = ssnPattern.ReplaceAllString(msg, "<SSN>")

	return msg
}

// MaskString masks part of a string for display
func MaskString(s string, showFirst, showLast int) string {
	if len(s) <= showFirst+showLast {
		return strings.Repeat("*", len(s))
	}

	masked := s[:showFirst] + strings.Repeat("*", len(s)-showFirst-showLast) + s[len(s)-showLast:]
	return masked
}
