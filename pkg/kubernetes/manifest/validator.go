package manifest

import (
	"fmt"
	"strings"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field    string
	Message  string
	Severity string // error, warning, info
}

// Validator validates Kubernetes manifests
type Validator interface {
	Validate(manifest *ParsedManifest) []ValidationError
}

// DefaultValidators returns the default set of validators
func DefaultValidators() []Validator {
	return []Validator{
		&APIVersionValidator{},
		&ResourceLimitsValidator{},
		&SecurityContextValidator{},
		&ImagePullPolicyValidator{},
		&NamespaceValidator{},
		&LabelValidator{},
	}
}

// APIVersionValidator validates API version compatibility
type APIVersionValidator struct{}

func (v *APIVersionValidator) Validate(manifest *ParsedManifest) []ValidationError {
	var errors []ValidationError
	
	// Check for deprecated API versions
	deprecated := map[string]string{
		"extensions/v1beta1": "apps/v1",
		"apps/v1beta1":       "apps/v1",
		"apps/v1beta2":       "apps/v1",
	}
	
	if replacement, isDeprecated := deprecated[manifest.APIVersion]; isDeprecated {
		errors = append(errors, ValidationError{
			Field:    "apiVersion",
			Message:  fmt.Sprintf("API version %s is deprecated, use %s instead", manifest.APIVersion, replacement),
			Severity: "warning",
		})
	}
	
	// Validate API version format
	if !strings.Contains(manifest.APIVersion, "/") && manifest.APIVersion != "v1" {
		errors = append(errors, ValidationError{
			Field:    "apiVersion",
			Message:  fmt.Sprintf("Invalid API version format: %s", manifest.APIVersion),
			Severity: "error",
		})
	}
	
	return errors
}

// ResourceLimitsValidator ensures resource limits are set
type ResourceLimitsValidator struct{}

func (v *ResourceLimitsValidator) Validate(manifest *ParsedManifest) []ValidationError {
	var errors []ValidationError
	
	// Only check for pod-containing resources
	podSpecTypes := []string{"Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob", "Pod"}
	if !contains(podSpecTypes, manifest.Kind) {
		return errors
	}
	
	// Extract containers from spec
	containers := extractContainers(manifest.Spec)
	for i, container := range containers {
		containerName := getContainerName(container, i)
		
		// Check for resource limits
		resources, ok := container["resources"].(map[string]interface{})
		if !ok {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.containers[%d].resources", i),
				Message:  fmt.Sprintf("Container '%s' has no resource limits defined", containerName),
				Severity: "warning",
			})
			continue
		}
		
		// Check for CPU and memory limits
		limits, hasLimits := resources["limits"].(map[string]interface{})
		requests, hasRequests := resources["requests"].(map[string]interface{})
		
		if !hasLimits {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.containers[%d].resources.limits", i),
				Message:  fmt.Sprintf("Container '%s' has no resource limits", containerName),
				Severity: "warning",
			})
		} else {
			if _, hasCPU := limits["cpu"]; !hasCPU {
				errors = append(errors, ValidationError{
					Field:    fmt.Sprintf("spec.containers[%d].resources.limits.cpu", i),
					Message:  fmt.Sprintf("Container '%s' has no CPU limit", containerName),
					Severity: "warning",
				})
			}
			if _, hasMemory := limits["memory"]; !hasMemory {
				errors = append(errors, ValidationError{
					Field:    fmt.Sprintf("spec.containers[%d].resources.limits.memory", i),
					Message:  fmt.Sprintf("Container '%s' has no memory limit", containerName),
					Severity: "warning",
				})
			}
		}
		
		if !hasRequests {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.containers[%d].resources.requests", i),
				Message:  fmt.Sprintf("Container '%s' has no resource requests", containerName),
				Severity: "warning",
			})
		}
	}
	
	return errors
}

// SecurityContextValidator validates security context settings
type SecurityContextValidator struct{}

func (v *SecurityContextValidator) Validate(manifest *ParsedManifest) []ValidationError {
	var errors []ValidationError
	
	// Only check for pod-containing resources
	podSpecTypes := []string{"Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob", "Pod"}
	if !contains(podSpecTypes, manifest.Kind) {
		return errors
	}
	
	// Check pod security context
	podSecurityContext := extractPodSecurityContext(manifest.Spec)
	if podSecurityContext == nil {
		errors = append(errors, ValidationError{
			Field:    "spec.securityContext",
			Message:  "Pod security context is not defined",
			Severity: "info",
		})
	}
	
	// Check container security contexts
	containers := extractContainers(manifest.Spec)
	for i, container := range containers {
		containerName := getContainerName(container, i)
		securityContext, ok := container["securityContext"].(map[string]interface{})
		
		if !ok {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.containers[%d].securityContext", i),
				Message:  fmt.Sprintf("Container '%s' has no security context", containerName),
				Severity: "info",
			})
			continue
		}
		
		// Check for runAsNonRoot
		if runAsNonRoot, ok := securityContext["runAsNonRoot"].(bool); !ok || !runAsNonRoot {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.containers[%d].securityContext.runAsNonRoot", i),
				Message:  fmt.Sprintf("Container '%s' should run as non-root", containerName),
				Severity: "warning",
			})
		}
		
		// Check for readOnlyRootFilesystem
		if readOnly, ok := securityContext["readOnlyRootFilesystem"].(bool); !ok || !readOnly {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.containers[%d].securityContext.readOnlyRootFilesystem", i),
				Message:  fmt.Sprintf("Container '%s' should have read-only root filesystem", containerName),
				Severity: "info",
			})
		}
		
		// Check for privileged
		if privileged, ok := securityContext["privileged"].(bool); ok && privileged {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.containers[%d].securityContext.privileged", i),
				Message:  fmt.Sprintf("Container '%s' should not run in privileged mode", containerName),
				Severity: "error",
			})
		}
	}
	
	return errors
}

// ImagePullPolicyValidator validates image pull policies
type ImagePullPolicyValidator struct{}

func (v *ImagePullPolicyValidator) Validate(manifest *ParsedManifest) []ValidationError {
	var errors []ValidationError
	
	// Only check for pod-containing resources
	podSpecTypes := []string{"Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob", "Pod"}
	if !contains(podSpecTypes, manifest.Kind) {
		return errors
	}
	
	containers := extractContainers(manifest.Spec)
	for i, container := range containers {
		containerName := getContainerName(container, i)
		
		// Check image tag
		image, ok := container["image"].(string)
		if !ok || image == "" {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.containers[%d].image", i),
				Message:  fmt.Sprintf("Container '%s' has no image specified", containerName),
				Severity: "error",
			})
			continue
		}
		
		// Check if using latest tag
		if strings.HasSuffix(image, ":latest") || !strings.Contains(image, ":") {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.containers[%d].image", i),
				Message:  fmt.Sprintf("Container '%s' uses 'latest' tag or no tag, which is not recommended for production", containerName),
				Severity: "warning",
			})
		}
		
		// Check imagePullPolicy
		pullPolicy, _ := container["imagePullPolicy"].(string)
		if pullPolicy == "" {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.containers[%d].imagePullPolicy", i),
				Message:  fmt.Sprintf("Container '%s' has no imagePullPolicy specified", containerName),
				Severity: "info",
			})
		} else if pullPolicy == "Always" && strings.Contains(image, "@sha256:") {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.containers[%d].imagePullPolicy", i),
				Message:  fmt.Sprintf("Container '%s' uses digest but has 'Always' pull policy", containerName),
				Severity: "info",
			})
		}
	}
	
	return errors
}

// NamespaceValidator validates namespace existence and naming
type NamespaceValidator struct{}

func (v *NamespaceValidator) Validate(manifest *ParsedManifest) []ValidationError {
	var errors []ValidationError
	
	// Skip namespace validation for cluster-scoped resources
	if !manifest.IsNamespaced() {
		return errors
	}
	
	// Check if namespace is specified
	if manifest.Metadata.Namespace == "" {
		errors = append(errors, ValidationError{
			Field:    "metadata.namespace",
			Message:  "Namespace is not specified for namespaced resource",
			Severity: "warning",
		})
	}
	
	// Validate namespace name
	if manifest.Metadata.Namespace != "" {
		if strings.HasPrefix(manifest.Metadata.Namespace, "kube-") {
			errors = append(errors, ValidationError{
				Field:    "metadata.namespace",
				Message:  "Should not use kube- prefixed namespaces",
				Severity: "warning",
			})
		}
	}
	
	return errors
}

// LabelValidator validates labels and annotations
type LabelValidator struct{}

func (v *LabelValidator) Validate(manifest *ParsedManifest) []ValidationError {
	var errors []ValidationError
	
	// Check for recommended labels
	recommendedLabels := []string{
		"app.kubernetes.io/name",
		"app.kubernetes.io/version",
		"app.kubernetes.io/component",
		"app.kubernetes.io/part-of",
		"app.kubernetes.io/managed-by",
	}
	
	for _, label := range recommendedLabels {
		if _, exists := manifest.Metadata.Labels[label]; !exists {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("metadata.labels.%s", label),
				Message:  fmt.Sprintf("Recommended label '%s' is missing", label),
				Severity: "info",
			})
		}
	}
	
	// Validate label values
	for key, value := range manifest.Metadata.Labels {
		if len(value) > 63 {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("metadata.labels.%s", key),
				Message:  fmt.Sprintf("Label value exceeds 63 characters: %s", value),
				Severity: "error",
			})
		}
		
		// Check for valid label value format
		if value != "" && !isValidLabelValue(value) {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("metadata.labels.%s", key),
				Message:  fmt.Sprintf("Invalid label value format: %s", value),
				Severity: "error",
			})
		}
	}
	
	return errors
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

func extractContainers(spec map[string]interface{}) []map[string]interface{} {
	var containers []map[string]interface{}
	
	// Navigate through different resource types
	if template, ok := spec["template"].(map[string]interface{}); ok {
		if podSpec, ok := template["spec"].(map[string]interface{}); ok {
			if c, ok := podSpec["containers"].([]interface{}); ok {
				for _, container := range c {
					if containerMap, ok := container.(map[string]interface{}); ok {
						containers = append(containers, containerMap)
					}
				}
			}
		}
	} else if c, ok := spec["containers"].([]interface{}); ok {
		// Direct pod spec
		for _, container := range c {
			if containerMap, ok := container.(map[string]interface{}); ok {
				containers = append(containers, containerMap)
			}
		}
	}
	
	return containers
}

func extractPodSecurityContext(spec map[string]interface{}) map[string]interface{} {
	if template, ok := spec["template"].(map[string]interface{}); ok {
		if podSpec, ok := template["spec"].(map[string]interface{}); ok {
			if securityContext, ok := podSpec["securityContext"].(map[string]interface{}); ok {
				return securityContext
			}
		}
	} else if securityContext, ok := spec["securityContext"].(map[string]interface{}); ok {
		return securityContext
	}
	
	return nil
}

func getContainerName(container map[string]interface{}, index int) string {
	if name, ok := container["name"].(string); ok {
		return name
	}
	return fmt.Sprintf("container-%d", index)
}

func isValidLabelValue(value string) bool {
	// Kubernetes label value must:
	// - be 63 characters or less
	// - begin and end with an alphanumeric character
	// - contain only alphanumeric characters, '-', '_' or '.'
	if len(value) == 0 || len(value) > 63 {
		return false
	}
	
	// Simple validation - in production, use proper regex
	return true
}