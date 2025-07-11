package security

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidateContainerName validates Docker/Kubernetes container names
func ValidateContainerName(name string) error {
	if name == "" {
		return fmt.Errorf("container name cannot be empty")
	}

	// Container names must be lowercase alphanumeric with hyphens
	// Max length 253 characters (DNS-1123 subdomain)
	if len(name) > 253 {
		return fmt.Errorf("container name too long (max 253 characters)")
	}

	validName := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid container name: must be lowercase alphanumeric with hyphens")
	}

	return nil
}

// ValidateNamespace validates Kubernetes namespace
func ValidateNamespace(namespace string) error {
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	// Kubernetes namespace validation
	if len(namespace) > 63 {
		return fmt.Errorf("namespace too long (max 63 characters)")
	}

	validNamespace := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !validNamespace.MatchString(namespace) {
		return fmt.Errorf("invalid namespace: must be lowercase alphanumeric with hyphens")
	}

	return nil
}

// ValidateServiceName validates service names
func ValidateServiceName(name string) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	// Service names follow DNS-1035 label rules
	if len(name) > 63 {
		return fmt.Errorf("service name too long (max 63 characters)")
	}

	validService := regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)
	if !validService.MatchString(name) {
		return fmt.Errorf("invalid service name: must start with lowercase letter, contain only lowercase alphanumeric and hyphens")
	}

	return nil
}

// ValidateFilePath validates file paths to prevent traversal
func ValidateFilePath(path string, allowedDirs []string) error {
	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Clean and resolve the path
	cleanPath := filepath.Clean(path)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	// Check if path is within allowed directories
	allowed := false
	for _, dir := range allowedDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}

		// Check if path is within allowed directory
		relPath, err := filepath.Rel(absDir, absPath)
		if err == nil && !strings.HasPrefix(relPath, "..") {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("file path outside allowed directories")
	}

	return nil
}

// ValidateImageName validates Docker image names
func ValidateImageName(image string) error {
	if image == "" {
		return fmt.Errorf("image name cannot be empty")
	}

	// Basic Docker image name validation
	// Format: [registry/]namespace/name[:tag]
	// Max length for full reference is 255
	if len(image) > 255 {
		return fmt.Errorf("image name too long (max 255 characters)")
	}

	// Check for invalid characters
	validImage := regexp.MustCompile(`^[a-z0-9._\-/:]+$`)
	if !validImage.MatchString(image) {
		return fmt.Errorf("invalid image name: contains invalid characters")
	}

	// Prevent obvious command injection attempts
	if strings.Contains(image, ";") || strings.Contains(image, "&") ||
		strings.Contains(image, "|") || strings.Contains(image, "$") ||
		strings.Contains(image, "`") || strings.Contains(image, "\\") {
		return fmt.Errorf("invalid image name: contains shell metacharacters")
	}

	return nil
}

// ValidateRegistryURL validates container registry URLs
func ValidateRegistryURL(url string) error {
	if url == "" {
		return nil // Empty registry is valid (uses Docker Hub)
	}

	// Basic registry URL validation
	validRegistry := regexp.MustCompile(`^[a-z0-9.-]+(:[0-9]+)?(/[a-z0-9._-]+)*$`)
	if !validRegistry.MatchString(url) {
		return fmt.Errorf("invalid registry URL format")
	}

	return nil
}

// ValidateDuration validates time duration strings
func ValidateDuration(duration string) error {
	if duration == "" {
		return nil
	}

	// Validate duration format (e.g., 1h, 30m, 5s)
	validDuration := regexp.MustCompile(`^[0-9]+(ns|us|µs|ms|s|m|h)$`)
	if !validDuration.MatchString(duration) {
		return fmt.Errorf("invalid duration format")
	}

	return nil
}

// SanitizeLogFilter sanitizes log filter patterns
func SanitizeLogFilter(filter string) string {
	// Remove potentially dangerous characters while preserving search functionality
	// Allow alphanumeric, spaces, and basic punctuation
	sanitized := regexp.MustCompile(`[^a-zA-Z0-9\s\-_.,!?@#]`).ReplaceAllString(filter, "")

	// Limit length to prevent DOS
	if len(sanitized) > 256 {
		sanitized = sanitized[:256]
	}

	return sanitized
}

// ValidateTailLines validates the number of lines for tail operation
func ValidateTailLines(lines int) error {
	if lines < 0 {
		return fmt.Errorf("tail lines cannot be negative")
	}

	// Limit to prevent memory exhaustion
	if lines > 10000 {
		return fmt.Errorf("tail lines too large (max 10000)")
	}

	return nil
}
