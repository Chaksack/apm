package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// KubernetesDeployer handles Kubernetes deployments
type KubernetesDeployer struct {
	config          KubernetesConfig
	kubectlClient   KubectlClient
	manifestManager *ManifestManager
}

// KubernetesConfig holds Kubernetes deployment configuration
type KubernetesConfig struct {
	ManifestPath   string
	Namespace      string
	Context        string
	APMConfig      APMConfig
	InjectSidecars bool
	Sidecars       []SidecarConfig
}

// SidecarConfig defines a sidecar container
type SidecarConfig struct {
	Name    string
	Image   string
	Command []string
	Env     map[string]string
	Ports   []int
}

// KubectlClient interface for kubectl operations
type KubectlClient interface {
	Apply(ctx context.Context, manifestPath, namespace, context string) error
	GetDeploymentStatus(ctx context.Context, name, namespace, context string) (string, error)
	CreateConfigMap(ctx context.Context, name, namespace string, data map[string]string) error
	CreateSecret(ctx context.Context, name, namespace string, data map[string]string) error
}

// ManifestManager handles Kubernetes manifest modifications
type ManifestManager struct {
	config KubernetesConfig
}

// NewKubernetesDeployer creates a new Kubernetes deployer
func NewKubernetesDeployer(config KubernetesConfig) *KubernetesDeployer {
	return &KubernetesDeployer{
		config:          config,
		kubectlClient:   &CLIKubectlClient{},
		manifestManager: &ManifestManager{config: config},
	}
}

// Deploy performs the Kubernetes deployment
func (d *KubernetesDeployer) Deploy(ctx context.Context) error {
	// 1. Validate manifests exist
	if err := d.validateManifests(); err != nil {
		return fmt.Errorf("manifest validation failed: %w", err)
	}

	// 2. Create namespace if needed
	if err := d.ensureNamespace(ctx); err != nil {
		return fmt.Errorf("namespace creation failed: %w", err)
	}

	// 3. Create APM ConfigMaps and Secrets
	if err := d.createAPMResources(ctx); err != nil {
		return fmt.Errorf("APM resource creation failed: %w", err)
	}

	// 4. Process manifests (inject sidecars, add annotations)
	processedManifests, err := d.manifestManager.ProcessManifests()
	if err != nil {
		return fmt.Errorf("manifest processing failed: %w", err)
	}

	// 5. Apply manifests
	for _, manifest := range processedManifests {
		if err := d.kubectlClient.Apply(ctx, manifest, d.config.Namespace, d.config.Context); err != nil {
			return fmt.Errorf("kubectl apply failed for %s: %w", manifest, err)
		}
	}

	return nil
}

// validateManifests checks if manifest files exist
func (d *KubernetesDeployer) validateManifests() error {
	info, err := os.Stat(d.config.ManifestPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("manifest path not found: %s", d.config.ManifestPath)
	}

	if info.IsDir() {
		// Check for YAML files in directory
		files, err := filepath.Glob(filepath.Join(d.config.ManifestPath, "*.yaml"))
		if err != nil {
			return fmt.Errorf("failed to list yaml files: %w", err)
		}

		yamlFiles, err := filepath.Glob(filepath.Join(d.config.ManifestPath, "*.yml"))
		if err != nil {
			return fmt.Errorf("failed to list yml files: %w", err)
		}

		files = append(files, yamlFiles...)

		if len(files) == 0 {
			return fmt.Errorf("no YAML files found in %s", d.config.ManifestPath)
		}
	}

	return nil
}

// ensureNamespace creates the namespace if it doesn't exist
func (d *KubernetesDeployer) ensureNamespace(ctx context.Context) error {
	// Check if namespace exists
	cmd := exec.CommandContext(ctx, "kubectl", "get", "namespace", d.config.Namespace)
	if d.config.Context != "" {
		cmd = exec.CommandContext(ctx, "kubectl", "--context", d.config.Context, "get", "namespace", d.config.Namespace)
	}

	if err := cmd.Run(); err != nil {
		// Namespace doesn't exist, create it
		createCmd := exec.CommandContext(ctx, "kubectl", "create", "namespace", d.config.Namespace)
		if d.config.Context != "" {
			createCmd = exec.CommandContext(ctx, "kubectl", "--context", d.config.Context, "create", "namespace", d.config.Namespace)
		}

		if err := createCmd.Run(); err != nil {
			return fmt.Errorf("failed to create namespace: %w", err)
		}
	}

	return nil
}

// createAPMResources creates ConfigMaps and Secrets for APM
func (d *KubernetesDeployer) createAPMResources(ctx context.Context) error {
	// Create APM ConfigMap
	apmConfig := map[string]string{
		"service.name":           d.config.APMConfig.ServiceName,
		"environment":            d.config.APMConfig.Environment,
		"otel.exporter.endpoint": d.config.APMConfig.Endpoint,
		"metrics.enabled":        "true",
		"traces.enabled":         "true",
		"logs.enabled":           "true",
	}

	if err := d.kubectlClient.CreateConfigMap(ctx, "apm-config", d.config.Namespace, apmConfig); err != nil {
		return fmt.Errorf("failed to create APM configmap: %w", err)
	}

	return nil
}

// ProcessManifests processes and modifies Kubernetes manifests
func (m *ManifestManager) ProcessManifests() ([]string, error) {
	var processedFiles []string

	// Get all manifest files
	files, err := m.getManifestFiles()
	if err != nil {
		return nil, err
	}

	// Process each file
	for _, file := range files {
		processed, err := m.processManifestFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to process %s: %w", file, err)
		}
		processedFiles = append(processedFiles, processed)
	}

	return processedFiles, nil
}

// getManifestFiles returns all manifest files to process
func (m *ManifestManager) getManifestFiles() ([]string, error) {
	info, err := os.Stat(m.config.ManifestPath)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		files, err := filepath.Glob(filepath.Join(m.config.ManifestPath, "*.yaml"))
		if err != nil {
			return nil, err
		}

		yamlFiles, err := filepath.Glob(filepath.Join(m.config.ManifestPath, "*.yml"))
		if err != nil {
			return nil, err
		}

		return append(files, yamlFiles...), nil
	}

	return []string{m.config.ManifestPath}, nil
}

// processManifestFile processes a single manifest file
func (m *ManifestManager) processManifestFile(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	// Parse YAML
	var manifest map[string]interface{}
	if err := yaml.Unmarshal(content, &manifest); err != nil {
		return "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Check if it's a Deployment or StatefulSet
	kind, ok := manifest["kind"].(string)
	if !ok {
		return filename, nil // Not a resource we need to modify
	}

	if kind == "Deployment" || kind == "StatefulSet" || kind == "DaemonSet" {
		// Inject sidecars and annotations
		if err := m.injectAPMConfiguration(manifest); err != nil {
			return "", fmt.Errorf("failed to inject APM config: %w", err)
		}

		// Write modified manifest to temp file
		modifiedContent, err := yaml.Marshal(manifest)
		if err != nil {
			return "", fmt.Errorf("failed to marshal modified manifest: %w", err)
		}

		tmpFile := filename + ".apm.yaml"
		if err := os.WriteFile(tmpFile, modifiedContent, 0644); err != nil {
			return "", fmt.Errorf("failed to write modified manifest: %w", err)
		}

		return tmpFile, nil
	}

	return filename, nil
}

// injectAPMConfiguration injects APM sidecars and annotations
func (m *ManifestManager) injectAPMConfiguration(manifest map[string]interface{}) error {
	// Navigate to spec.template.spec
	spec, ok := manifest["spec"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("spec not found")
	}

	template, ok := spec["template"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("spec.template not found")
	}

	templateSpec, ok := template["spec"].(map[string]interface{})
	if !ok {
		templateSpec = make(map[string]interface{})
		template["spec"] = templateSpec
	}

	// Add annotations
	metadata, ok := template["metadata"].(map[string]interface{})
	if !ok {
		metadata = make(map[string]interface{})
		template["metadata"] = metadata
	}

	annotations, ok := metadata["annotations"].(map[string]interface{})
	if !ok {
		annotations = make(map[string]interface{})
		metadata["annotations"] = annotations
	}

	// Add APM annotations
	annotations["apm.instrumentation/inject"] = "true"
	annotations["apm.instrumentation/service-name"] = m.config.APMConfig.ServiceName
	annotations["apm.instrumentation/environment"] = m.config.APMConfig.Environment

	// Inject sidecars if enabled
	if m.config.InjectSidecars {
		containers, ok := templateSpec["containers"].([]interface{})
		if !ok {
			containers = []interface{}{}
		}

		// Add default sidecars
		sidecars := m.getDefaultSidecars()
		for _, sidecar := range sidecars {
			containers = append(containers, sidecar)
		}

		templateSpec["containers"] = containers
	}

	// Add environment variables to main container
	if containers, ok := templateSpec["containers"].([]interface{}); ok && len(containers) > 0 {
		if mainContainer, ok := containers[0].(map[string]interface{}); ok {
			env, ok := mainContainer["env"].([]interface{})
			if !ok {
				env = []interface{}{}
			}

			// Add APM environment variables
			apmEnvVars := []map[string]interface{}{
				{
					"name":  "OTEL_SERVICE_NAME",
					"value": m.config.APMConfig.ServiceName,
				},
				{
					"name":  "OTEL_ENVIRONMENT",
					"value": m.config.APMConfig.Environment,
				},
				{
					"name":  "OTEL_EXPORTER_OTLP_ENDPOINT",
					"value": m.config.APMConfig.Endpoint,
				},
			}

			for _, envVar := range apmEnvVars {
				env = append(env, envVar)
			}

			mainContainer["env"] = env
		}
	}

	return nil
}

// getDefaultSidecars returns default sidecar containers
func (m *ManifestManager) getDefaultSidecars() []map[string]interface{} {
	sidecars := []map[string]interface{}{}

	// OpenTelemetry Collector sidecar
	otelCollector := map[string]interface{}{
		"name":  "otel-collector",
		"image": "otel/opentelemetry-collector:latest",
		"args": []string{
			"--config=/etc/otel-collector-config.yaml",
		},
		"env": []map[string]interface{}{
			{
				"name": "SERVICE_NAME",
				"valueFrom": map[string]interface{}{
					"fieldRef": map[string]interface{}{
						"fieldPath": "metadata.labels['app']",
					},
				},
			},
		},
		"ports": []map[string]interface{}{
			{
				"name":          "otlp-grpc",
				"containerPort": 4317,
			},
			{
				"name":          "otlp-http",
				"containerPort": 4318,
			},
			{
				"name":          "metrics",
				"containerPort": 8888,
			},
		},
		"volumeMounts": []map[string]interface{}{
			{
				"name":      "otel-collector-config",
				"mountPath": "/etc/otel-collector-config.yaml",
				"subPath":   "otel-collector-config.yaml",
			},
		},
	}

	sidecars = append(sidecars, otelCollector)

	// Fluent Bit sidecar for log forwarding
	if m.config.APMConfig.InjectAgent {
		fluentBit := map[string]interface{}{
			"name":  "fluent-bit",
			"image": "fluent/fluent-bit:latest",
			"env": []map[string]interface{}{
				{
					"name":  "FLUENT_ELASTICSEARCH_HOST",
					"value": "elasticsearch.logging.svc.cluster.local",
				},
			},
			"volumeMounts": []map[string]interface{}{
				{
					"name":      "varlog",
					"mountPath": "/var/log",
				},
			},
		}
		sidecars = append(sidecars, fluentBit)
	}

	return sidecars
}

// CLIKubectlClient implements KubectlClient using kubectl CLI
type CLIKubectlClient struct{}

// Apply applies a Kubernetes manifest
func (c *CLIKubectlClient) Apply(ctx context.Context, manifestPath, namespace, context string) error {
	args := []string{"apply", "-f", manifestPath}

	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	if context != "" {
		args = append(args, "--context", context)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// GetDeploymentStatus gets the status of a deployment
func (c *CLIKubectlClient) GetDeploymentStatus(ctx context.Context, name, namespace, context string) (string, error) {
	args := []string{"get", "deployment", name, "-o", "jsonpath={.status.conditions[?(@.type=='Progressing')].status}"}

	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	if context != "" {
		args = append(args, "--context", context)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// CreateConfigMap creates a Kubernetes ConfigMap
func (c *CLIKubectlClient) CreateConfigMap(ctx context.Context, name, namespace string, data map[string]string) error {
	args := []string{"create", "configmap", name}

	for k, v := range data {
		args = append(args, fmt.Sprintf("--from-literal=%s=%s", k, v))
	}

	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	// Add --dry-run and -o yaml to check if it exists first
	checkArgs := append(args, "--dry-run=client", "-o", "yaml")

	cmd := exec.CommandContext(ctx, "kubectl", checkArgs...)
	if err := cmd.Run(); err == nil {
		// ConfigMap can be created, so create it
		cmd = exec.CommandContext(ctx, "kubectl", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return nil
}

// CreateSecret creates a Kubernetes Secret
func (c *CLIKubectlClient) CreateSecret(ctx context.Context, name, namespace string, data map[string]string) error {
	args := []string{"create", "secret", "generic", name}

	for k, v := range data {
		args = append(args, fmt.Sprintf("--from-literal=%s=%s", k, v))
	}

	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
