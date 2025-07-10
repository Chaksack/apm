package manifest

import (
	"fmt"
	"time"
)

// Transformer modifies Kubernetes manifests
type Transformer interface {
	Transform(manifest *ParsedManifest) error
	CanTransform(manifest *ParsedManifest) bool
}

// NamespaceTransformer sets or updates the namespace
type NamespaceTransformer struct {
	Namespace string
}

func (t *NamespaceTransformer) CanTransform(manifest *ParsedManifest) bool {
	return manifest.IsNamespaced()
}

func (t *NamespaceTransformer) Transform(manifest *ParsedManifest) error {
	if !t.CanTransform(manifest) {
		return nil
	}
	manifest.SetNamespace(t.Namespace)
	return nil
}

// LabelTransformer adds or updates labels
type LabelTransformer struct {
	Labels map[string]string
}

func (t *LabelTransformer) CanTransform(manifest *ParsedManifest) bool {
	return true
}

func (t *LabelTransformer) Transform(manifest *ParsedManifest) error {
	for key, value := range t.Labels {
		manifest.AddLabel(key, value)
	}
	return nil
}

// AnnotationTransformer adds or updates annotations
type AnnotationTransformer struct {
	Annotations map[string]string
}

func (t *AnnotationTransformer) CanTransform(manifest *ParsedManifest) bool {
	return true
}

func (t *AnnotationTransformer) Transform(manifest *ParsedManifest) error {
	for key, value := range t.Annotations {
		manifest.AddAnnotation(key, value)
	}
	return nil
}

// APMTransformer adds APM-specific annotations and labels
type APMTransformer struct {
	EnableMetrics bool
	EnableLogging bool
	EnableTracing bool
	Environment   string
}

func (t *APMTransformer) CanTransform(manifest *ParsedManifest) bool {
	// Only transform pod-containing resources
	podSpecTypes := []string{"Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob", "Pod"}
	return contains(podSpecTypes, manifest.Kind)
}

func (t *APMTransformer) Transform(manifest *ParsedManifest) error {
	// Add APM annotations
	manifest.AddAnnotation("apm.io/inject", "true")
	manifest.AddAnnotation("apm.io/environment", t.Environment)
	manifest.AddAnnotation("apm.io/transformed-at", time.Now().Format(time.RFC3339))
	
	if t.EnableMetrics {
		manifest.AddAnnotation("apm.io/inject-metrics", "true")
		manifest.AddAnnotation("prometheus.io/scrape", "true")
		manifest.AddAnnotation("prometheus.io/port", "9090")
		manifest.AddAnnotation("prometheus.io/path", "/metrics")
	}
	
	if t.EnableLogging {
		manifest.AddAnnotation("apm.io/inject-logging", "true")
		manifest.AddAnnotation("apm.io/log-format", "json")
	}
	
	if t.EnableTracing {
		manifest.AddAnnotation("apm.io/inject-tracing", "true")
		manifest.AddAnnotation("apm.io/trace-sampling-rate", "0.1")
	}
	
	// Add APM labels
	manifest.AddLabel("apm.io/monitored", "true")
	manifest.AddLabel("apm.io/environment", t.Environment)
	
	return nil
}

// ImageTransformer updates container images
type ImageTransformer struct {
	ImageMappings map[string]string // old image -> new image
	Registry      string            // optional: prepend registry to all images
}

func (t *ImageTransformer) CanTransform(manifest *ParsedManifest) bool {
	podSpecTypes := []string{"Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob", "Pod"}
	return contains(podSpecTypes, manifest.Kind)
}

func (t *ImageTransformer) Transform(manifest *ParsedManifest) error {
	containers := extractContainers(manifest.Spec)
	
	for _, container := range containers {
		if image, ok := container["image"].(string); ok {
			newImage := image
			
			// Check for explicit mapping
			if mapped, exists := t.ImageMappings[image]; exists {
				newImage = mapped
			} else if t.Registry != "" {
				// Prepend registry if not already present
				if !hasRegistry(image) {
					newImage = fmt.Sprintf("%s/%s", t.Registry, image)
				}
			}
			
			container["image"] = newImage
		}
	}
	
	return nil
}

// ResourceTransformer updates resource requests and limits
type ResourceTransformer struct {
	DefaultRequests ResourceRequirements
	DefaultLimits   ResourceRequirements
	Multiplier      float64 // optional: multiply existing values
}

type ResourceRequirements struct {
	CPU    string
	Memory string
}

func (t *ResourceTransformer) CanTransform(manifest *ParsedManifest) bool {
	podSpecTypes := []string{"Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob", "Pod"}
	return contains(podSpecTypes, manifest.Kind)
}

func (t *ResourceTransformer) Transform(manifest *ParsedManifest) error {
	containers := extractContainers(manifest.Spec)
	
	for _, container := range containers {
		resources, ok := container["resources"].(map[string]interface{})
		if !ok {
			resources = make(map[string]interface{})
			container["resources"] = resources
		}
		
		// Set default requests
		if t.DefaultRequests.CPU != "" || t.DefaultRequests.Memory != "" {
			requests, ok := resources["requests"].(map[string]interface{})
			if !ok {
				requests = make(map[string]interface{})
				resources["requests"] = requests
			}
			
			if t.DefaultRequests.CPU != "" && requests["cpu"] == nil {
				requests["cpu"] = t.DefaultRequests.CPU
			}
			if t.DefaultRequests.Memory != "" && requests["memory"] == nil {
				requests["memory"] = t.DefaultRequests.Memory
			}
		}
		
		// Set default limits
		if t.DefaultLimits.CPU != "" || t.DefaultLimits.Memory != "" {
			limits, ok := resources["limits"].(map[string]interface{})
			if !ok {
				limits = make(map[string]interface{})
				resources["limits"] = limits
			}
			
			if t.DefaultLimits.CPU != "" && limits["cpu"] == nil {
				limits["cpu"] = t.DefaultLimits.CPU
			}
			if t.DefaultLimits.Memory != "" && limits["memory"] == nil {
				limits["memory"] = t.DefaultLimits.Memory
			}
		}
	}
	
	return nil
}

// SecurityTransformer applies security settings
type SecurityTransformer struct {
	RunAsNonRoot           bool
	ReadOnlyRootFilesystem bool
	DropCapabilities       []string
	RunAsUser              *int64
	RunAsGroup             *int64
}

func (t *SecurityTransformer) CanTransform(manifest *ParsedManifest) bool {
	podSpecTypes := []string{"Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob", "Pod"}
	return contains(podSpecTypes, manifest.Kind)
}

func (t *SecurityTransformer) Transform(manifest *ParsedManifest) error {
	// Apply pod security context
	podSpec := extractPodSpec(manifest.Spec)
	if podSpec != nil {
		securityContext, ok := podSpec["securityContext"].(map[string]interface{})
		if !ok {
			securityContext = make(map[string]interface{})
			podSpec["securityContext"] = securityContext
		}
		
		if t.RunAsNonRoot {
			securityContext["runAsNonRoot"] = true
		}
		if t.RunAsUser != nil {
			securityContext["runAsUser"] = *t.RunAsUser
		}
		if t.RunAsGroup != nil {
			securityContext["runAsGroup"] = *t.RunAsGroup
		}
	}
	
	// Apply container security contexts
	containers := extractContainers(manifest.Spec)
	for _, container := range containers {
		securityContext, ok := container["securityContext"].(map[string]interface{})
		if !ok {
			securityContext = make(map[string]interface{})
			container["securityContext"] = securityContext
		}
		
		if t.RunAsNonRoot {
			securityContext["runAsNonRoot"] = true
		}
		if t.ReadOnlyRootFilesystem {
			securityContext["readOnlyRootFilesystem"] = true
		}
		
		if len(t.DropCapabilities) > 0 {
			capabilities, ok := securityContext["capabilities"].(map[string]interface{})
			if !ok {
				capabilities = make(map[string]interface{})
				securityContext["capabilities"] = capabilities
			}
			
			drop := make([]interface{}, len(t.DropCapabilities))
			for i, cap := range t.DropCapabilities {
				drop[i] = cap
			}
			capabilities["drop"] = drop
		}
	}
	
	return nil
}

// NetworkPolicyTransformer adds network policy annotations
type NetworkPolicyTransformer struct {
	DefaultPolicy string // "deny-all", "allow-same-namespace", etc.
	AllowedCIDRs  []string
}

func (t *NetworkPolicyTransformer) CanTransform(manifest *ParsedManifest) bool {
	return manifest.Kind == "Namespace" || manifest.IsNamespaced()
}

func (t *NetworkPolicyTransformer) Transform(manifest *ParsedManifest) error {
	manifest.AddAnnotation("apm.io/network-policy", t.DefaultPolicy)
	
	if len(t.AllowedCIDRs) > 0 {
		cidrList := ""
		for i, cidr := range t.AllowedCIDRs {
			if i > 0 {
				cidrList += ","
			}
			cidrList += cidr
		}
		manifest.AddAnnotation("apm.io/allowed-cidrs", cidrList)
	}
	
	return nil
}

// Helper functions

func hasRegistry(image string) bool {
	// Simple check - in production, use more sophisticated logic
	return len(image) > 0 && (image[0] == '/' || 
		len(image) > 4 && image[0:4] == "http" ||
		len(image) > 0 && image[0] != ':')
}

func extractPodSpec(spec map[string]interface{}) map[string]interface{} {
	if template, ok := spec["template"].(map[string]interface{}); ok {
		if podSpec, ok := template["spec"].(map[string]interface{}); ok {
			return podSpec
		}
	} else if _, ok := spec["containers"]; ok {
		// Direct pod spec
		return spec
	}
	
	return nil
}