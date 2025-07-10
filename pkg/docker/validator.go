package docker

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// DockerfileValidator validates Dockerfiles for best practices and APM compatibility
type DockerfileValidator struct {
	rules []ValidationRule
}

// ValidationRule represents a validation rule for Dockerfiles
type ValidationRule struct {
	Name        string
	Description string
	Severity    string // "error" or "warning"
	Check       func([]DockerInstruction) []ValidationIssue
}

// ValidationIssue represents an issue found during validation
type ValidationIssue struct {
	Line     int
	Message  string
	Rule     string
	Severity string
}

// NewDockerfileValidator creates a new Dockerfile validator
func NewDockerfileValidator() *DockerfileValidator {
	v := &DockerfileValidator{}
	v.initializeRules()
	return v
}

// Validate validates a Dockerfile
func (v *DockerfileValidator) Validate(dockerfilePath string) error {
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return fmt.Errorf("failed to read Dockerfile: %w", err)
	}

	instructions, err := parseDockerfileForValidation(content)
	if err != nil {
		return fmt.Errorf("failed to parse Dockerfile: %w", err)
	}

	validation := &DockerfileValidation{
		Valid:    true,
		APMReady: true,
	}

	// Run all validation rules
	for _, rule := range v.rules {
		issues := rule.Check(instructions)
		for _, issue := range issues {
			if issue.Severity == "error" {
				validation.Errors = append(validation.Errors, ValidationError{
					Line:    issue.Line,
					Message: issue.Message,
					Rule:    issue.Rule,
				})
				validation.Valid = false
			} else {
				validation.Warnings = append(validation.Warnings, ValidationWarning{
					Line:    issue.Line,
					Message: issue.Message,
					Rule:    issue.Rule,
				})
			}
		}
	}

	// Check APM readiness
	validation.APMReady = v.checkAPMReadiness(instructions)

	// Extract base image info
	validation.BaseImage = v.extractBaseImageInfo(instructions)

	// Generate suggestions
	validation.Suggestions = v.generateSuggestions(instructions, validation)

	if !validation.Valid {
		return fmt.Errorf("Dockerfile validation failed with %d errors", len(validation.Errors))
	}

	return nil
}

// initializeRules sets up all validation rules
func (v *DockerfileValidator) initializeRules() {
	v.rules = []ValidationRule{
		{
			Name:        "base-image-tag",
			Description: "Base image should use specific tag instead of latest",
			Severity:    "warning",
			Check:       v.checkBaseImageTag,
		},
		{
			Name:        "user-permissions",
			Description: "Container should run as non-root user",
			Severity:    "warning",
			Check:       v.checkUserPermissions,
		},
		{
			Name:        "healthcheck",
			Description: "Container should define HEALTHCHECK",
			Severity:    "warning",
			Check:       v.checkHealthcheck,
		},
		{
			Name:        "expose-ports",
			Description: "Container should EXPOSE necessary ports",
			Severity:    "warning",
			Check:       v.checkExposedPorts,
		},
		{
			Name:        "apt-get-cleanup",
			Description: "apt-get should clean up after installation",
			Severity:    "warning",
			Check:       v.checkAptGetCleanup,
		},
		{
			Name:        "copy-vs-add",
			Description: "Use COPY instead of ADD when possible",
			Severity:    "warning",
			Check:       v.checkCopyVsAdd,
		},
		{
			Name:        "workdir-absolute",
			Description: "WORKDIR should use absolute paths",
			Severity:    "error",
			Check:       v.checkWorkdirAbsolute,
		},
		{
			Name:        "multiple-runs",
			Description: "Combine RUN commands to reduce layers",
			Severity:    "warning",
			Check:       v.checkMultipleRuns,
		},
		{
			Name:        "secret-exposure",
			Description: "Avoid exposing secrets in Dockerfile",
			Severity:    "error",
			Check:       v.checkSecretExposure,
		},
		{
			Name:        "label-metadata",
			Description: "Add metadata labels for better tracking",
			Severity:    "warning",
			Check:       v.checkLabelMetadata,
		},
	}
}

// Validation rule implementations

func (v *DockerfileValidator) checkBaseImageTag(instructions []DockerInstruction) []ValidationIssue {
	var issues []ValidationIssue

	for _, inst := range instructions {
		if inst.Command == "FROM" {
			if !strings.Contains(inst.Value, ":") || strings.HasSuffix(inst.Value, ":latest") {
				issues = append(issues, ValidationIssue{
					Line:     inst.Line,
					Message:  "Base image should use specific tag instead of 'latest'",
					Rule:     "base-image-tag",
					Severity: "warning",
				})
			}
		}
	}

	return issues
}

func (v *DockerfileValidator) checkUserPermissions(instructions []DockerInstruction) []ValidationIssue {
	hasUser := false

	for _, inst := range instructions {
		if inst.Command == "USER" {
			hasUser = true
			break
		}
	}

	if !hasUser {
		return []ValidationIssue{{
			Line:     0,
			Message:  "Container runs as root user. Consider adding USER instruction",
			Rule:     "user-permissions",
			Severity: "warning",
		}}
	}

	return nil
}

func (v *DockerfileValidator) checkHealthcheck(instructions []DockerInstruction) []ValidationIssue {
	hasHealthcheck := false

	for _, inst := range instructions {
		if inst.Command == "HEALTHCHECK" {
			hasHealthcheck = true
			break
		}
	}

	if !hasHealthcheck {
		return []ValidationIssue{{
			Line:     0,
			Message:  "No HEALTHCHECK defined. Consider adding health check for better container management",
			Rule:     "healthcheck",
			Severity: "warning",
		}}
	}

	return nil
}

func (v *DockerfileValidator) checkExposedPorts(instructions []DockerInstruction) []ValidationIssue {
	hasExpose := false

	for _, inst := range instructions {
		if inst.Command == "EXPOSE" {
			hasExpose = true
			break
		}
	}

	if !hasExpose {
		return []ValidationIssue{{
			Line:     0,
			Message:  "No ports exposed. Consider using EXPOSE for documentation",
			Rule:     "expose-ports",
			Severity: "warning",
		}}
	}

	return nil
}

func (v *DockerfileValidator) checkAptGetCleanup(instructions []DockerInstruction) []ValidationIssue {
	var issues []ValidationIssue

	for _, inst := range instructions {
		if inst.Command == "RUN" && strings.Contains(inst.Value, "apt-get install") {
			if !strings.Contains(inst.Value, "rm -rf /var/lib/apt/lists/*") {
				issues = append(issues, ValidationIssue{
					Line:     inst.Line,
					Message:  "apt-get install should clean up package lists to reduce image size",
					Rule:     "apt-get-cleanup",
					Severity: "warning",
				})
			}
		}
	}

	return issues
}

func (v *DockerfileValidator) checkCopyVsAdd(instructions []DockerInstruction) []ValidationIssue {
	var issues []ValidationIssue

	for _, inst := range instructions {
		if inst.Command == "ADD" {
			// Check if ADD is used for remote URLs or tar extraction
			if !strings.HasPrefix(inst.Value, "http") && !strings.Contains(inst.Value, ".tar") {
				issues = append(issues, ValidationIssue{
					Line:     inst.Line,
					Message:  "Use COPY instead of ADD for simple file copying",
					Rule:     "copy-vs-add",
					Severity: "warning",
				})
			}
		}
	}

	return issues
}

func (v *DockerfileValidator) checkWorkdirAbsolute(instructions []DockerInstruction) []ValidationIssue {
	var issues []ValidationIssue

	for _, inst := range instructions {
		if inst.Command == "WORKDIR" {
			path := strings.Fields(inst.Value)[0]
			if !strings.HasPrefix(path, "/") {
				issues = append(issues, ValidationIssue{
					Line:     inst.Line,
					Message:  fmt.Sprintf("WORKDIR should use absolute path, found: %s", path),
					Rule:     "workdir-absolute",
					Severity: "error",
				})
			}
		}
	}

	return issues
}

func (v *DockerfileValidator) checkMultipleRuns(instructions []DockerInstruction) []ValidationIssue {
	consecutiveRuns := 0
	lastRunLine := 0

	for _, inst := range instructions {
		if inst.Command == "RUN" {
			if lastRunLine > 0 && inst.Line == lastRunLine+1 {
				consecutiveRuns++
			} else {
				consecutiveRuns = 1
			}
			lastRunLine = inst.Line
		} else {
			consecutiveRuns = 0
		}

		if consecutiveRuns >= 3 {
			return []ValidationIssue{{
				Line:     inst.Line,
				Message:  "Multiple consecutive RUN commands detected. Consider combining them to reduce layers",
				Rule:     "multiple-runs",
				Severity: "warning",
			}}
		}
	}

	return nil
}

func (v *DockerfileValidator) checkSecretExposure(instructions []DockerInstruction) []ValidationIssue {
	var issues []ValidationIssue

	// Patterns that might indicate secrets
	secretPatterns := []string{
		`password\s*=`,
		`api[_-]?key\s*=`,
		`secret\s*=`,
		`token\s*=`,
		`private[_-]?key`,
		`credentials`,
	}

	for _, inst := range instructions {
		if inst.Command == "ENV" || inst.Command == "ARG" {
			lowerValue := strings.ToLower(inst.Value)
			for _, pattern := range secretPatterns {
				if matched, _ := regexp.MatchString(pattern, lowerValue); matched {
					issues = append(issues, ValidationIssue{
						Line:     inst.Line,
						Message:  "Possible secret exposed in Dockerfile. Use build secrets or runtime environment variables",
						Rule:     "secret-exposure",
						Severity: "error",
					})
					break
				}
			}
		}
	}

	return issues
}

func (v *DockerfileValidator) checkLabelMetadata(instructions []DockerInstruction) []ValidationIssue {
	hasLabels := false
	recommendedLabels := map[string]bool{
		"maintainer":  false,
		"version":     false,
		"description": false,
	}

	for _, inst := range instructions {
		if inst.Command == "LABEL" {
			hasLabels = true
			for label := range recommendedLabels {
				if strings.Contains(inst.Value, label) {
					recommendedLabels[label] = true
				}
			}
		}
	}

	if !hasLabels {
		return []ValidationIssue{{
			Line:     0,
			Message:  "No LABEL instructions found. Consider adding metadata labels",
			Rule:     "label-metadata",
			Severity: "warning",
		}}
	}

	var missingLabels []string
	for label, found := range recommendedLabels {
		if !found {
			missingLabels = append(missingLabels, label)
		}
	}

	if len(missingLabels) > 0 {
		return []ValidationIssue{{
			Line:     0,
			Message:  fmt.Sprintf("Consider adding these metadata labels: %s", strings.Join(missingLabels, ", ")),
			Rule:     "label-metadata",
			Severity: "warning",
		}}
	}

	return nil
}

// Helper methods

func (v *DockerfileValidator) checkAPMReadiness(instructions []DockerInstruction) bool {
	// Check if Dockerfile has necessary components for APM
	hasExpose := false
	hasHealthcheck := false
	hasUser := false

	for _, inst := range instructions {
		switch inst.Command {
		case "EXPOSE":
			hasExpose = true
		case "HEALTHCHECK":
			hasHealthcheck = true
		case "USER":
			hasUser = true
		}
	}

	// Check for APM-related environment variables
	hasAPMEnv := false
	for _, inst := range instructions {
		if inst.Command == "ENV" {
			if strings.Contains(inst.Value, "OTEL_") || strings.Contains(inst.Value, "APM_") {
				hasAPMEnv = true
				break
			}
		}
	}

	return hasExpose && (hasHealthcheck || hasAPMEnv)
}

func (v *DockerfileValidator) extractBaseImageInfo(instructions []DockerInstruction) BaseImageInfo {
	info := BaseImageInfo{}

	for _, inst := range instructions {
		if inst.Command == "FROM" {
			parts := strings.Split(inst.Value, ":")
			info.Name = parts[0]
			if len(parts) > 1 {
				info.Tag = parts[1]
			} else {
				info.Tag = "latest"
			}

			// Detect OS and architecture from common base images
			lowerName := strings.ToLower(info.Name)
			if strings.Contains(lowerName, "alpine") {
				info.OS = "alpine"
			} else if strings.Contains(lowerName, "ubuntu") {
				info.OS = "ubuntu"
			} else if strings.Contains(lowerName, "debian") {
				info.OS = "debian"
			} else if strings.Contains(lowerName, "centos") || strings.Contains(lowerName, "rhel") {
				info.OS = "rhel"
			} else if strings.Contains(lowerName, "scratch") {
				info.OS = "scratch"
			} else if strings.Contains(lowerName, "distroless") {
				info.OS = "distroless"
			}

			break
		}
	}

	return info
}

func (v *DockerfileValidator) generateSuggestions(instructions []DockerInstruction, validation *DockerfileValidation) []string {
	var suggestions []string

	// Multi-stage build suggestion
	fromCount := 0
	for _, inst := range instructions {
		if inst.Command == "FROM" {
			fromCount++
		}
	}
	if fromCount == 1 {
		suggestions = append(suggestions, "Consider using multi-stage builds to reduce final image size")
	}

	// APM suggestions
	if !validation.APMReady {
		suggestions = append(suggestions, "Add HEALTHCHECK instruction for better container monitoring")
		suggestions = append(suggestions, "Consider adding APM environment variables (OTEL_SERVICE_NAME, etc.)")
	}

	// Security suggestions
	if validation.BaseImage.OS != "distroless" && validation.BaseImage.OS != "scratch" {
		suggestions = append(suggestions, "Consider using distroless or minimal base images for better security")
	}

	// Layer optimization
	runCount := 0
	for _, inst := range instructions {
		if inst.Command == "RUN" {
			runCount++
		}
	}
	if runCount > 5 {
		suggestions = append(suggestions, fmt.Sprintf("Dockerfile has %d RUN instructions. Consider combining them to reduce layers", runCount))
	}

	return suggestions
}

// parseDockerfileForValidation parses Dockerfile content for validation
func parseDockerfileForValidation(content []byte) ([]DockerInstruction, error) {
	var instructions []DockerInstruction
	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0

	var currentInstruction *DockerInstruction

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Handle line continuation
		if strings.HasSuffix(trimmed, "\\") {
			if currentInstruction == nil {
				parts := strings.SplitN(trimmed, " ", 2)
				currentInstruction = &DockerInstruction{
					Command:  strings.ToUpper(parts[0]),
					Value:    strings.TrimSuffix(parts[1], "\\"),
					Original: line,
					Line:     lineNum,
				}
			} else {
				currentInstruction.Value += " " + strings.TrimSuffix(trimmed, "\\")
				currentInstruction.Original += "\n" + line
			}
			continue
		}

		// Complete instruction
		if currentInstruction != nil {
			currentInstruction.Value += " " + trimmed
			currentInstruction.Original += "\n" + line
			instructions = append(instructions, *currentInstruction)
			currentInstruction = nil
		} else {
			parts := strings.SplitN(trimmed, " ", 2)
			inst := DockerInstruction{
				Command:  strings.ToUpper(parts[0]),
				Original: line,
				Line:     lineNum,
			}
			if len(parts) > 1 {
				inst.Value = parts[1]
			}
			instructions = append(instructions, inst)
		}
	}

	return instructions, scanner.Err()
}

// ValidateAndSuggest validates a Dockerfile and returns detailed results
func (v *DockerfileValidator) ValidateAndSuggest(dockerfilePath string) (*DockerfileValidation, error) {
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Dockerfile: %w", err)
	}

	instructions, err := parseDockerfileForValidation(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Dockerfile: %w", err)
	}

	validation := &DockerfileValidation{
		Valid:    true,
		APMReady: true,
	}

	// Run all validation rules
	for _, rule := range v.rules {
		issues := rule.Check(instructions)
		for _, issue := range issues {
			if issue.Severity == "error" {
				validation.Errors = append(validation.Errors, ValidationError{
					Line:    issue.Line,
					Message: issue.Message,
					Rule:    issue.Rule,
				})
				validation.Valid = false
			} else {
				validation.Warnings = append(validation.Warnings, ValidationWarning{
					Line:    issue.Line,
					Message: issue.Message,
					Rule:    issue.Rule,
				})
			}
		}
	}

	// Check APM readiness
	validation.APMReady = v.checkAPMReadiness(instructions)

	// Extract base image info
	validation.BaseImage = v.extractBaseImageInfo(instructions)

	// Generate suggestions
	validation.Suggestions = v.generateSuggestions(instructions, validation)

	return validation, nil
}
