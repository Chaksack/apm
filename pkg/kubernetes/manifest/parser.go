package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

// Parser handles Kubernetes manifest parsing and manipulation
type Parser struct {
	scheme     *runtime.Scheme
	validators []Validator
}

// NewParser creates a new manifest parser
func NewParser() *Parser {
	return &Parser{
		scheme:     scheme.Scheme,
		validators: DefaultValidators(),
	}
}

// ParsedManifest represents a parsed Kubernetes manifest
type ParsedManifest struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   Metadata               `json:"metadata"`
	Spec       map[string]interface{} `json:"spec,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Raw        []byte                 `json:"-"`
	Path       string                 `json:"-"`
	Hash       string                 `json:"-"`
}

// Metadata represents Kubernetes object metadata
type Metadata struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	UID         string            `json:"uid,omitempty"`
}

// Parse parses a single Kubernetes manifest
func (p *Parser) Parse(content []byte) (*ParsedManifest, error) {
	// Decode YAML to unstructured
	obj := &unstructured.Unstructured{}
	dec := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 4096)
	
	if err := dec.Decode(obj); err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
	}

	// Extract metadata
	metadata := Metadata{
		Name:        obj.GetName(),
		Namespace:   obj.GetNamespace(),
		Labels:      obj.GetLabels(),
		Annotations: obj.GetAnnotations(),
		UID:         string(obj.GetUID()),
	}

	// Calculate hash
	h := sha256.New()
	h.Write(content)
	hash := hex.EncodeToString(h.Sum(nil))

	manifest := &ParsedManifest{
		APIVersion: obj.GetAPIVersion(),
		Kind:       obj.GetKind(),
		Metadata:   metadata,
		Raw:        content,
		Hash:       hash,
	}

	// Extract spec or data based on kind
	if spec, found, err := unstructured.NestedMap(obj.Object, "spec"); found && err == nil {
		manifest.Spec = spec
	}
	if data, found, err := unstructured.NestedMap(obj.Object, "data"); found && err == nil {
		manifest.Data = data
	}

	return manifest, nil
}

// ParseMultiDocument parses multiple YAML documents from a single file
func (p *Parser) ParseMultiDocument(content []byte) ([]*ParsedManifest, error) {
	var manifests []*ParsedManifest
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 4096)
	
	for {
		manifest := &ParsedManifest{}
		raw := &unstructured.Unstructured{}
		
		if err := decoder.Decode(raw); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to decode document: %w", err)
		}

		// Skip empty documents
		if len(raw.Object) == 0 {
			continue
		}

		// Convert back to YAML for raw storage
		yamlBytes, err := yaml.Marshal(raw.Object)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal to YAML: %w", err)
		}

		// Parse the individual manifest
		manifest, err = p.Parse(yamlBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse manifest: %w", err)
		}

		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

// ParseFile parses a Kubernetes manifest file
func (p *Parser) ParseFile(path string) ([]*ParsedManifest, error) {
	content, err := readFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	manifests, err := p.ParseMultiDocument(content)
	if err != nil {
		return nil, err
	}

	// Set file path for all manifests
	for _, m := range manifests {
		m.Path = path
	}

	return manifests, nil
}

// Validate validates a parsed manifest
func (p *Parser) Validate(manifest *ParsedManifest) []ValidationError {
	var errors []ValidationError
	
	for _, validator := range p.validators {
		if errs := validator.Validate(manifest); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}
	
	return errors
}

// Transform applies transformations to a manifest
func (p *Parser) Transform(manifest *ParsedManifest, transformers ...Transformer) error {
	for _, t := range transformers {
		if !t.CanTransform(manifest) {
			continue
		}
		
		if err := t.Transform(manifest); err != nil {
			return fmt.Errorf("transformation failed: %w", err)
		}
	}
	
	return nil
}

// Serialize converts a parsed manifest back to YAML
func (p *Parser) Serialize(manifest *ParsedManifest) ([]byte, error) {
	// Create unstructured object
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": manifest.APIVersion,
			"kind":       manifest.Kind,
			"metadata": map[string]interface{}{
				"name":        manifest.Metadata.Name,
				"namespace":   manifest.Metadata.Namespace,
				"labels":      manifest.Metadata.Labels,
				"annotations": manifest.Metadata.Annotations,
			},
		},
	}

	// Add spec if present
	if len(manifest.Spec) > 0 {
		obj.Object["spec"] = manifest.Spec
	}

	// Add data if present (for ConfigMaps/Secrets)
	if len(manifest.Data) > 0 {
		obj.Object["data"] = manifest.Data
	}

	// Convert to YAML
	return yaml.Marshal(obj.Object)
}

// GetGVK returns the GroupVersionKind of a manifest
func (m *ParsedManifest) GetGVK() schema.GroupVersionKind {
	gv, _ := schema.ParseGroupVersion(m.APIVersion)
	return schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    m.Kind,
	}
}

// IsNamespaced returns true if the resource is namespaced
func (m *ParsedManifest) IsNamespaced() bool {
	// Cluster-scoped resources
	clusterScoped := map[string]bool{
		"Namespace":               true,
		"Node":                    true,
		"PersistentVolume":        true,
		"ClusterRole":             true,
		"ClusterRoleBinding":      true,
		"StorageClass":            true,
		"CustomResourceDefinition": true,
	}
	
	return !clusterScoped[m.Kind]
}

// SetNamespace sets the namespace for a manifest
func (m *ParsedManifest) SetNamespace(namespace string) {
	if m.IsNamespaced() {
		m.Metadata.Namespace = namespace
	}
}

// AddLabel adds a label to the manifest
func (m *ParsedManifest) AddLabel(key, value string) {
	if m.Metadata.Labels == nil {
		m.Metadata.Labels = make(map[string]string)
	}
	m.Metadata.Labels[key] = value
}

// AddAnnotation adds an annotation to the manifest
func (m *ParsedManifest) AddAnnotation(key, value string) {
	if m.Metadata.Annotations == nil {
		m.Metadata.Annotations = make(map[string]string)
	}
	m.Metadata.Annotations[key] = value
}

// DetectionPatterns returns file patterns for manifest detection
func DetectionPatterns() []string {
	return []string{
		"*.yaml",
		"*.yml",
		"k8s/*.yaml",
		"kubernetes/*.yaml",
		"manifests/*.yaml",
		"deploy/*.yaml",
		"deployment/*.yaml",
		"charts/*/templates/*.yaml",
	}
}

// IsManifestFile checks if a file path matches manifest patterns
func IsManifestFile(path string) bool {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".yaml" && ext != ".yml" {
		return false
	}

	// Check against common patterns
	patterns := []string{
		"k8s", "kubernetes", "manifests", "deploy", 
		"deployment", "charts", "templates",
	}
	
	dir := filepath.Dir(path)
	for _, pattern := range patterns {
		if strings.Contains(dir, pattern) {
			return true
		}
	}
	
	// Check filename patterns
	base := filepath.Base(path)
	manifestNames := []string{
		"deployment", "service", "configmap", "secret",
		"ingress", "pod", "statefulset", "daemonset",
		"job", "cronjob", "namespace", "rbac",
	}
	
	baseLower := strings.ToLower(base)
	for _, name := range manifestNames {
		if strings.Contains(baseLower, name) {
			return true
		}
	}
	
	return false
}

// Helper function to read file (can be mocked for testing)
var readFile = func(path string) ([]byte, error) {
	// This would normally use os.ReadFile
	// Placeholder for actual implementation
	return nil, fmt.Errorf("not implemented")
}