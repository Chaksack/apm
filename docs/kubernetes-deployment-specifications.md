# Kubernetes Deployment Capabilities Specifications

## Overview

This document provides comprehensive specifications for Kubernetes deployment capabilities in the APM stack, including manifest manipulation, sidecar injection, configuration management, and multi-cloud support.

## 1. Manifest File Detection and Parsing

### Detection Patterns

The system will automatically detect Kubernetes manifests using the following patterns:

```yaml
file_patterns:
  - "**/*.yaml"
  - "**/*.yml"
  - "**/k8s/**/*.yaml"
  - "**/kubernetes/**/*.yaml"
  - "**/manifests/**/*.yaml"
  - "**/deploy/**/*.yaml"
  - "**/deployment/**/*.yaml"
  - "**/charts/**/templates/**/*.yaml"

exclude_patterns:
  - "**/node_modules/**"
  - "**/vendor/**"
  - "**/.git/**"
  - "**/test/**/*.yaml"
  - "**/tests/**/*.yaml"
```

### Parsing Capabilities

```go
package manifest

import (
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "sigs.k8s.io/yaml"
)

type ManifestParser struct {
    validators []Validator
    cache      *ManifestCache
}

type ParsedManifest struct {
    APIVersion string
    Kind       string
    Metadata   Metadata
    Spec       map[string]interface{}
    Raw        []byte
    Path       string
    Hash       string
}

type Metadata struct {
    Name        string
    Namespace   string
    Labels      map[string]string
    Annotations map[string]string
    UID         string
}

// Parse multiple documents from a single YAML file
func (p *ManifestParser) ParseMultiDocument(content []byte) ([]*ParsedManifest, error)

// Validate manifest against Kubernetes schemas
func (p *ManifestParser) Validate(manifest *ParsedManifest) []ValidationError

// Transform manifest with multiple transformers
func (p *ManifestParser) Transform(manifest *ParsedManifest, transformers ...Transformer) error
```

### Validation Rules

```yaml
validation_rules:
  - name: api_version_check
    description: Validate API version compatibility
    severity: error
    
  - name: resource_limits
    description: Ensure resource limits are set
    severity: warning
    
  - name: security_context
    description: Validate security context settings
    severity: error
    
  - name: image_pull_policy
    description: Check image pull policy
    severity: warning
    
  - name: namespace_existence
    description: Verify namespace exists
    severity: error
```

## 2. APM Sidecar Injection Patterns

### Sidecar Configuration

```go
package sidecar

type SidecarInjector struct {
    config     InjectorConfig
    registry   *SidecarRegistry
    mutator    *AdmissionMutator
}

type InjectorConfig struct {
    // Global injection settings
    AutoInject        bool
    NamespaceSelector labels.Selector
    PodSelector       labels.Selector
    
    // Sidecar defaults
    DefaultCPURequest    string
    DefaultMemoryRequest string
    DefaultCPULimit      string
    DefaultMemoryLimit   string
    
    // Security settings
    RunAsNonRoot     bool
    ReadOnlyRootFS   bool
    DropCapabilities []string
}

type SidecarSpec struct {
    Type        SidecarType
    Name        string
    Image       string
    Version     string
    Ports       []ContainerPort
    Environment []EnvVar
    Volumes     []VolumeMount
    Resources   ResourceRequirements
    Probes      ProbeConfig
}
```

### Injection Methods

#### 1. Annotation-Based Injection

```yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    # Global injection control
    apm.io/inject: "true"
    
    # Component-specific injection
    apm.io/inject-metrics: "true"
    apm.io/inject-logging: "true"
    apm.io/inject-tracing: "true"
    
    # Configuration overrides
    apm.io/metrics-port: "9090"
    apm.io/metrics-path: "/metrics"
    apm.io/log-level: "info"
    apm.io/trace-sampling-rate: "0.1"
    
    # Resource overrides
    apm.io/sidecar-cpu-request: "50m"
    apm.io/sidecar-memory-request: "64Mi"
```

#### 2. Webhook-Based Injection

```go
// Admission webhook for automatic injection
type SidecarWebhook struct {
    injector *SidecarInjector
    config   *WebhookConfig
}

func (w *SidecarWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
    pod := &v1.Pod{}
    if err := json.Unmarshal(req.Object.Raw, pod); err != nil {
        return admission.Errored(http.StatusBadRequest, err)
    }
    
    // Check if injection is needed
    if !w.shouldInject(pod) {
        return admission.Allowed("No injection required")
    }
    
    // Inject sidecars
    modifiedPod := w.injector.InjectSidecars(pod)
    
    // Create patch
    patch, err := createPatch(pod, modifiedPod)
    if err != nil {
        return admission.Errored(http.StatusInternalServerError, err)
    }
    
    return admission.Patched("Sidecars injected", patch)
}
```

### Sidecar Templates

#### Metrics Sidecar (Prometheus)

```yaml
containers:
- name: prometheus-exporter
  image: prom/node-exporter:v1.5.0
  args:
    - --path.procfs=/host/proc
    - --path.sysfs=/host/sys
    - --path.rootfs=/host/root
    - --collector.filesystem.mount-points-exclude=^/(dev|proc|sys|var/lib/docker/.+)($|/)
  ports:
  - containerPort: 9100
    name: metrics
    protocol: TCP
  resources:
    requests:
      cpu: 10m
      memory: 32Mi
    limits:
      cpu: 100m
      memory: 128Mi
  volumeMounts:
  - name: proc
    mountPath: /host/proc
    readOnly: true
  - name: sys
    mountPath: /host/sys
    readOnly: true
  - name: root
    mountPath: /host/root
    readOnly: true
  securityContext:
    runAsNonRoot: true
    runAsUser: 65534
    readOnlyRootFilesystem: true
    capabilities:
      drop: ["ALL"]
```

#### Logging Sidecar (Fluentbit)

```yaml
containers:
- name: fluent-bit
  image: fluent/fluent-bit:2.1.0
  env:
  - name: FLUENT_LOKI_URL
    value: "http://loki:3100/loki/api/v1/push"
  - name: K8S_NAMESPACE
    valueFrom:
      fieldRef:
        fieldPath: metadata.namespace
  - name: K8S_POD_NAME
    valueFrom:
      fieldRef:
        fieldPath: metadata.name
  volumeMounts:
  - name: config
    mountPath: /fluent-bit/etc/
  - name: varlog
    mountPath: /var/log
    readOnly: true
  - name: varlibdockercontainers
    mountPath: /var/lib/docker/containers
    readOnly: true
  resources:
    requests:
      cpu: 20m
      memory: 64Mi
    limits:
      cpu: 100m
      memory: 256Mi
```

#### Tracing Sidecar (OpenTelemetry)

```yaml
containers:
- name: otel-agent
  image: otel/opentelemetry-collector:0.88.0
  args:
    - "--config=/conf/otel-agent-config.yaml"
  env:
  - name: OTEL_RESOURCE_ATTRIBUTES
    value: "service.name=$(K8S_POD_NAME),k8s.namespace.name=$(K8S_NAMESPACE)"
  - name: K8S_NAMESPACE
    valueFrom:
      fieldRef:
        fieldPath: metadata.namespace
  - name: K8S_POD_NAME
    valueFrom:
      fieldRef:
        fieldPath: metadata.name
  ports:
  - containerPort: 4317  # OTLP gRPC
    name: otlp-grpc
  - containerPort: 4318  # OTLP HTTP
    name: otlp-http
  - containerPort: 6831  # Jaeger Thrift Compact
    protocol: UDP
    name: jaeger-compact
  - containerPort: 14250  # Jaeger gRPC
    name: jaeger-grpc
  resources:
    requests:
      cpu: 50m
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 512Mi
  volumeMounts:
  - name: otel-agent-config
    mountPath: /conf
```

## 3. ConfigMap/Secret Generation

### Configuration Templates

```go
package config

type ConfigGenerator struct {
    templates  *template.Template
    validators []ConfigValidator
    encryptor  SecretEncryptor
}

type APMConfig struct {
    Global      GlobalConfig
    Prometheus  PrometheusConfig
    Grafana     GrafanaConfig
    Loki        LokiConfig
    Jaeger      JaegerConfig
    AlertManager AlertManagerConfig
}

func (g *ConfigGenerator) GenerateConfigs(config APMConfig) (*ConfigBundle, error) {
    bundle := &ConfigBundle{
        ConfigMaps: make(map[string]*v1.ConfigMap),
        Secrets:    make(map[string]*v1.Secret),
    }
    
    // Generate Prometheus config
    promConfig, err := g.generatePrometheusConfig(config.Prometheus)
    if err != nil {
        return nil, err
    }
    bundle.ConfigMaps["prometheus-config"] = promConfig
    
    // Generate secrets
    secrets, err := g.generateSecrets(config)
    if err != nil {
        return nil, err
    }
    bundle.Secrets = secrets
    
    return bundle, nil
}
```

### Dynamic Configuration Examples

#### Prometheus Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: apm-system
data:
  prometheus.yml: |
    global:
      scrape_interval: ${SCRAPE_INTERVAL}
      evaluation_interval: ${EVAL_INTERVAL}
      external_labels:
        cluster: ${CLUSTER_NAME}
        region: ${REGION}
        
    # Dynamic scrape configs based on service discovery
    scrape_configs:
    - job_name: 'kubernetes-apiservers'
      kubernetes_sd_configs:
      - role: endpoints
      scheme: https
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      
    - job_name: 'kubernetes-pods'
      kubernetes_sd_configs:
      - role: pod
      relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
```

#### Grafana Datasources

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-datasources
  namespace: apm-system
data:
  datasources.yaml: |
    apiVersion: 1
    datasources:
    - name: Prometheus
      type: prometheus
      access: proxy
      url: http://prometheus:9090
      isDefault: true
      jsonData:
        timeInterval: ${SCRAPE_INTERVAL}
        queryTimeout: ${QUERY_TIMEOUT}
        httpMethod: POST
        
    - name: Loki
      type: loki
      access: proxy
      url: http://loki:3100
      jsonData:
        maxLines: ${MAX_LOG_LINES}
        
    - name: Jaeger
      type: jaeger
      access: proxy
      url: http://jaeger-query:16686
      jsonData:
        tracesToMetrics:
          datasourceUid: prometheus
```

#### Secret Management

```go
// Secret generation with encryption
func (g *ConfigGenerator) GenerateSecret(name string, data map[string][]byte) (*v1.Secret, error) {
    // Encrypt sensitive data
    encryptedData := make(map[string][]byte)
    for key, value := range data {
        encrypted, err := g.encryptor.Encrypt(value)
        if err != nil {
            return nil, fmt.Errorf("failed to encrypt %s: %w", key, err)
        }
        encryptedData[key] = encrypted
    }
    
    return &v1.Secret{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: "apm-system",
            Labels: map[string]string{
                "app.kubernetes.io/managed-by": "apm-operator",
                "app.kubernetes.io/component":  "secrets",
            },
            Annotations: map[string]string{
                "apm.io/last-rotation": time.Now().Format(time.RFC3339),
                "apm.io/encryption-version": "v1",
            },
        },
        Type: v1.SecretTypeOpaque,
        Data: encryptedData,
    }, nil
}
```

## 4. Cloud Provider Deployment Support

### Multi-Cloud Abstraction Layer

```go
package cloud

type CloudDeployer interface {
    // Cluster operations
    CreateCluster(config ClusterConfig) (*Cluster, error)
    UpdateCluster(clusterID string, config ClusterConfig) error
    DeleteCluster(clusterID string) error
    GetCluster(clusterID string) (*Cluster, error)
    
    // Deployment operations
    Deploy(manifest *Manifest, options DeployOptions) error
    Rollback(deployment string, revision int) error
    GetDeploymentStatus(deployment string) (*DeploymentStatus, error)
    
    // Networking
    CreateLoadBalancer(config LoadBalancerConfig) (*LoadBalancer, error)
    ConfigureIngress(config IngressConfig) error
    
    // Storage
    CreatePersistentVolume(config PVConfig) (*PersistentVolume, error)
    CreateStorageClass(config StorageClassConfig) error
}
```

### EKS (AWS) Implementation

```go
package eks

type EKSDeployer struct {
    eksClient *eks.Client
    ec2Client *ec2.Client
    iamClient *iam.Client
}

type EKSConfig struct {
    ClusterName      string
    Region           string
    Version          string
    NodeGroups       []NodeGroupConfig
    VPCConfig        VPCConfig
    IAMRoleARN       string
    SecurityGroups   []string
    LoggingConfig    LoggingConfig
    EncryptionConfig EncryptionConfig
    Tags             map[string]string
}

func (d *EKSDeployer) CreateAPMStack(config APMStackConfig) error {
    // Create IAM roles and policies
    roles, err := d.createIAMRoles(config)
    if err != nil {
        return fmt.Errorf("failed to create IAM roles: %w", err)
    }
    
    // Create security groups
    securityGroups, err := d.createSecurityGroups(config)
    if err != nil {
        return fmt.Errorf("failed to create security groups: %w", err)
    }
    
    // Deploy Helm chart with EKS-specific values
    helmValues := map[string]interface{}{
        "global": map[string]interface{}{
            "cloudProvider": "aws",
            "region":        config.Region,
            "eksConfig": map[string]interface{}{
                "iamRoleArn":     roles.ServiceAccountRole,
                "securityGroups": securityGroups,
            },
        },
        "prometheus": map[string]interface{}{
            "serviceAccount": map[string]interface{}{
                "annotations": map[string]string{
                    "eks.amazonaws.com/role-arn": roles.PrometheusRole,
                },
            },
        },
        "grafana": map[string]interface{}{
            "serviceAccount": map[string]interface{}{
                "annotations": map[string]string{
                    "eks.amazonaws.com/role-arn": roles.GrafanaRole,
                },
            },
        },
    }
    
    return d.deployHelmChart("apm-stack", helmValues)
}
```

### AKS (Azure) Implementation

```go
package aks

type AKSDeployer struct {
    containerClient *containerservice.Client
    storageClient   *storage.Client
    monitorClient   *monitor.Client
}

type AKSConfig struct {
    ResourceGroup      string
    ClusterName        string
    Location           string
    KubernetesVersion  string
    NodePools          []NodePoolConfig
    NetworkProfile     NetworkProfile
    IdentityProfile    IdentityProfile
    MonitoringConfig   MonitoringConfig
    Tags               map[string]string
}

func (d *AKSDeployer) CreateAPMStack(config APMStackConfig) error {
    // Configure managed identity
    identity, err := d.configureManagedIdentity(config)
    if err != nil {
        return fmt.Errorf("failed to configure managed identity: %w", err)
    }
    
    // Create Log Analytics workspace
    workspace, err := d.createLogAnalyticsWorkspace(config)
    if err != nil {
        return fmt.Errorf("failed to create workspace: %w", err)
    }
    
    // Deploy with AKS-specific configurations
    helmValues := map[string]interface{}{
        "global": map[string]interface{}{
            "cloudProvider": "azure",
            "location":      config.Location,
            "aksConfig": map[string]interface{}{
                "resourceGroup":   config.ResourceGroup,
                "managedIdentity": identity.ClientID,
                "workspaceId":     workspace.ID,
            },
        },
        "prometheus": map[string]interface{}{
            "podIdentity": map[string]interface{}{
                "enabled": true,
                "identityName": "prometheus-identity",
            },
        },
    }
    
    return d.deployHelmChart("apm-stack", helmValues)
}
```

### GKE (Google Cloud) Implementation

```go
package gke

type GKEDeployer struct {
    containerClient *container.Client
    storageClient   *storage.Client
    monitoringClient *monitoring.Client
}

type GKEConfig struct {
    ProjectID          string
    Zone               string
    ClusterName        string
    MachineType        string
    NodePools          []NodePoolConfig
    NetworkConfig      NetworkConfig
    WorkloadIdentity   WorkloadIdentityConfig
    MonitoringConfig   MonitoringConfig
    Labels             map[string]string
}

func (d *GKEDeployer) CreateAPMStack(config APMStackConfig) error {
    // Configure Workload Identity
    workloadIdentity, err := d.configureWorkloadIdentity(config)
    if err != nil {
        return fmt.Errorf("failed to configure workload identity: %w", err)
    }
    
    // Enable required APIs
    apis := []string{
        "container.googleapis.com",
        "monitoring.googleapis.com",
        "logging.googleapis.com",
    }
    if err := d.enableAPIs(config.ProjectID, apis); err != nil {
        return fmt.Errorf("failed to enable APIs: %w", err)
    }
    
    // Deploy with GKE-specific configurations
    helmValues := map[string]interface{}{
        "global": map[string]interface{}{
            "cloudProvider": "gcp",
            "projectId":     config.ProjectID,
            "gkeConfig": map[string]interface{}{
                "workloadIdentity": workloadIdentity,
                "enableStackdriver": true,
            },
        },
        "prometheus": map[string]interface{}{
            "serviceAccount": map[string]interface{}{
                "annotations": map[string]string{
                    "iam.gke.io/gcp-service-account": workloadIdentity.PrometheusAccount,
                },
            },
        },
    }
    
    return d.deployHelmChart("apm-stack", helmValues)
}
```

## 5. Helm Chart Integration

### Helm Management Interface

```go
package helm

import (
    "helm.sh/helm/v3/pkg/action"
    "helm.sh/helm/v3/pkg/chart"
    "helm.sh/helm/v3/pkg/release"
)

type HelmManager struct {
    actionConfig *action.Configuration
    settings     *cli.EnvSettings
}

type ChartDeployment struct {
    Name         string
    Namespace    string
    Chart        string
    Version      string
    Repository   string
    Values       map[string]interface{}
    Wait         bool
    Timeout      time.Duration
    Atomic       bool
    CreateNS     bool
}

func (m *HelmManager) DeployAPMStack(deployment ChartDeployment) (*release.Release, error) {
    // Add repository if needed
    if deployment.Repository != "" {
        if err := m.addRepository("apm-repo", deployment.Repository); err != nil {
            return nil, fmt.Errorf("failed to add repository: %w", err)
        }
    }
    
    // Load chart
    chart, err := m.loadChart(deployment.Chart, deployment.Version)
    if err != nil {
        return nil, fmt.Errorf("failed to load chart: %w", err)
    }
    
    // Validate values
    if err := m.validateValues(chart, deployment.Values); err != nil {
        return nil, fmt.Errorf("invalid values: %w", err)
    }
    
    // Install or upgrade
    client := action.NewUpgrade(m.actionConfig)
    client.Namespace = deployment.Namespace
    client.Install = true
    client.Wait = deployment.Wait
    client.Timeout = deployment.Timeout
    client.Atomic = deployment.Atomic
    client.CreateNamespace = deployment.CreateNS
    
    return client.Run(deployment.Name, chart, deployment.Values)
}
```

### Dynamic Values Generation

```go
func (m *HelmManager) GenerateValues(config APMConfig) (map[string]interface{}, error) {
    values := map[string]interface{}{
        "global": map[string]interface{}{
            "environment":   config.Environment,
            "cloudProvider": config.CloudProvider,
            "region":        config.Region,
            "clusterName":   config.ClusterName,
            "domain":        config.Domain,
            "storageClass":  config.StorageClass,
        },
    }
    
    // Component-specific values
    if config.Prometheus.Enabled {
        values["prometheus"] = m.generatePrometheusValues(config.Prometheus)
    }
    
    if config.Grafana.Enabled {
        values["grafana"] = m.generateGrafanaValues(config.Grafana)
    }
    
    if config.Loki.Enabled {
        values["loki"] = m.generateLokiValues(config.Loki)
    }
    
    if config.Jaeger.Enabled {
        values["jaeger"] = m.generateJaegerValues(config.Jaeger)
    }
    
    return values, nil
}
```

### Helm Hooks and Tests

```yaml
# Pre-install hook for validation
apiVersion: batch/v1
kind: Job
metadata:
  name: "{{ .Release.Name }}-preinstall-check"
  annotations:
    "helm.sh/hook": pre-install
    "helm.sh/hook-weight": "-5"
    "helm.sh/hook-delete-policy": before-hook-creation,hook-succeeded
spec:
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: pre-install-check
        image: {{ .Values.hooks.image }}
        command:
        - /bin/sh
        - -c
        - |
          echo "Checking cluster requirements..."
          kubectl version --short
          kubectl get nodes
          kubectl get storageclass
          echo "Validating permissions..."
          kubectl auth can-i create deployments --namespace={{ .Release.Namespace }}
          kubectl auth can-i create services --namespace={{ .Release.Namespace }}
          kubectl auth can-i create configmaps --namespace={{ .Release.Namespace }}

# Post-install hook for verification
apiVersion: batch/v1
kind: Job
metadata:
  name: "{{ .Release.Name }}-postinstall-verify"
  annotations:
    "helm.sh/hook": post-install
    "helm.sh/hook-weight": "5"
    "helm.sh/hook-delete-policy": before-hook-creation,hook-succeeded
spec:
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: post-install-verify
        image: {{ .Values.hooks.image }}
        command:
        - /bin/sh
        - -c
        - |
          echo "Verifying APM stack deployment..."
          kubectl wait --for=condition=ready pod -l app.kubernetes.io/instance={{ .Release.Name }} --timeout=300s
          echo "Testing connectivity..."
          curl -f http://prometheus:9090/-/healthy || exit 1
          curl -f http://grafana:3000/api/health || exit 1
          echo "APM stack deployed successfully!"
```

## 6. Service Mesh Integration

### Istio Integration

```yaml
# Istio sidecar configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: istio-sidecar-injector
  namespace: istio-system
data:
  config: |
    policy: enabled
    alwaysInjectSelector:
      - matchLabels:
          apm.io/mesh-enabled: "true"
    neverInjectSelector:
      - matchLabels:
          apm.io/mesh-enabled: "false"
    injectedAnnotations:
      apm.io/injected: "true"
      apm.io/injection-time: "{{ .InjectionTime }}"
    template: |
      containers:
      - name: istio-proxy
        image: docker.io/istio/proxyv2:{{ .ProxyVersion }}
        args:
        - proxy
        - sidecar
        - --domain
        - $(POD_NAMESPACE).svc.cluster.local
        - --proxyLogLevel={{ .ProxyLogLevel }}
        - --proxyComponentLogLevel={{ .ProxyComponentLogLevel }}
        - --log_output_level={{ .LogLevel }}
        env:
        - name: PILOT_ENABLE_WORKLOAD_ENTRY_AUTOREGISTRATION
          value: "true"
        - name: ISTIO_META_INTERCEPTION_MODE
          value: REDIRECT
        resources:
          requests:
            cpu: 10m
            memory: 40Mi
          limits:
            cpu: 2000m
            memory: 1024Mi
```

### Linkerd Integration

```go
package linkerd

type LinkerdIntegration struct {
    client *linkerd.Client
    config LinkerdConfig
}

func (l *LinkerdIntegration) InjectAPMAnnotations(pod *v1.Pod) {
    if pod.Annotations == nil {
        pod.Annotations = make(map[string]string)
    }
    
    // Enable Linkerd injection
    pod.Annotations["linkerd.io/inject"] = "enabled"
    
    // Configure observability
    pod.Annotations["config.linkerd.io/trace-collector"] = "jaeger-collector.apm-system:14268"
    pod.Annotations["config.linkerd.io/trace-collector-service-account"] = "jaeger"
    
    // Add custom labels for metrics
    pod.Labels["linkerd.io/app"] = pod.Name
    pod.Labels["linkerd.io/workload"] = pod.Name
}
```

## 7. Rollback Mechanisms

### Rollback Manager

```go
package rollback

import (
    "k8s.io/client-go/kubernetes"
    "helm.sh/helm/v3/pkg/action"
)

type RollbackManager struct {
    k8sClient    kubernetes.Interface
    helmClient   *action.Configuration
    stateStore   StateStore
}

type RollbackStrategy struct {
    MaxRetries       int
    RetryDelay       time.Duration
    HealthCheckFunc  HealthChecker
    NotificationFunc NotificationSender
}

func (r *RollbackManager) CreateCheckpoint(name string) (*Checkpoint, error) {
    checkpoint := &Checkpoint{
        Name:      name,
        Timestamp: time.Now(),
        Resources: make([]ResourceSnapshot, 0),
    }
    
    // Capture current state
    deployments, err := r.captureDeployments()
    if err != nil {
        return nil, err
    }
    checkpoint.Resources = append(checkpoint.Resources, deployments...)
    
    configMaps, err := r.captureConfigMaps()
    if err != nil {
        return nil, err
    }
    checkpoint.Resources = append(checkpoint.Resources, configMaps...)
    
    // Capture Helm releases
    releases, err := r.captureHelmReleases()
    if err != nil {
        return nil, err
    }
    checkpoint.HelmReleases = releases
    
    // Store checkpoint
    if err := r.stateStore.SaveCheckpoint(checkpoint); err != nil {
        return nil, err
    }
    
    return checkpoint, nil
}

func (r *RollbackManager) Rollback(checkpointName string, strategy RollbackStrategy) error {
    checkpoint, err := r.stateStore.GetCheckpoint(checkpointName)
    if err != nil {
        return fmt.Errorf("failed to get checkpoint: %w", err)
    }
    
    // Validate rollback is safe
    if err := r.validateRollback(checkpoint); err != nil {
        return fmt.Errorf("rollback validation failed: %w", err)
    }
    
    // Create pre-rollback checkpoint
    preRollback, err := r.CreateCheckpoint(fmt.Sprintf("pre-rollback-%s", time.Now().Format("20060102-150405")))
    if err != nil {
        return fmt.Errorf("failed to create pre-rollback checkpoint: %w", err)
    }
    
    // Execute rollback with retries
    var lastErr error
    for i := 0; i < strategy.MaxRetries; i++ {
        if err := r.executeRollback(checkpoint); err != nil {
            lastErr = err
            time.Sleep(strategy.RetryDelay)
            continue
        }
        
        // Verify health
        if err := strategy.HealthCheckFunc(); err != nil {
            lastErr = err
            time.Sleep(strategy.RetryDelay)
            continue
        }
        
        // Success
        strategy.NotificationFunc("Rollback completed successfully", checkpoint)
        return nil
    }
    
    // Rollback failed, attempt to restore pre-rollback state
    if err := r.executeRollback(preRollback); err != nil {
        return fmt.Errorf("critical: failed to restore pre-rollback state: %w", err)
    }
    
    return fmt.Errorf("rollback failed after %d attempts: %w", strategy.MaxRetries, lastErr)
}
```

### Progressive Rollback

```yaml
# Flagger configuration for progressive rollback
apiVersion: flagger.app/v1beta1
kind: Canary
metadata:
  name: apm-prometheus
  namespace: apm-system
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: prometheus
  progressDeadlineSeconds: 600
  service:
    port: 9090
    targetPort: 9090
  analysis:
    interval: 1m
    threshold: 5
    maxWeight: 50
    stepWeight: 10
    metrics:
    - name: prometheus-health
      templateRef:
        name: prometheus-health-check
        namespace: apm-system
      thresholdRange:
        min: 99
      interval: 30s
    webhooks:
    - name: rollback-hook
      type: rollback
      url: http://apm-operator:8080/rollback
      metadata:
        deployment: prometheus
        checkpoint: ${CHECKPOINT_NAME}
```

## 8. kubectl Context Management

### Context Manager Implementation

```go
package context

import (
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/tools/clientcmd/api"
)

type ContextManager struct {
    configPath string
    config     *api.Config
    validator  ContextValidator
}

type ManagedContext struct {
    Name          string
    Cluster       string
    User          string
    Namespace     string
    CloudProvider string
    Region        string
    Environment   string
    Labels        map[string]string
}

func (m *ContextManager) CreateAPMContext(ctx ManagedContext) error {
    // Validate context
    if err := m.validator.Validate(ctx); err != nil {
        return fmt.Errorf("invalid context: %w", err)
    }
    
    // Create context in kubeconfig
    context := &api.Context{
        Cluster:   ctx.Cluster,
        AuthInfo:  ctx.User,
        Namespace: ctx.Namespace,
        Extensions: map[string]runtime.Object{
            "apm.io/cloud-provider": &runtime.Unknown{Raw: []byte(ctx.CloudProvider)},
            "apm.io/region":         &runtime.Unknown{Raw: []byte(ctx.Region)},
            "apm.io/environment":    &runtime.Unknown{Raw: []byte(ctx.Environment)},
        },
    }
    
    m.config.Contexts[ctx.Name] = context
    
    // Save config
    return clientcmd.WriteToFile(*m.config, m.configPath)
}

func (m *ContextManager) SwitchContext(name string, options SwitchOptions) error {
    // Validate context exists
    if _, exists := m.config.Contexts[name]; !exists {
        return fmt.Errorf("context %s not found", name)
    }
    
    // Save current context state if requested
    if options.SaveCurrentState {
        if err := m.saveContextState(m.config.CurrentContext); err != nil {
            return fmt.Errorf("failed to save current state: %w", err)
        }
    }
    
    // Switch context
    m.config.CurrentContext = name
    
    // Apply post-switch actions
    if options.SetupAPMNamespace {
        if err := m.setupAPMNamespace(name); err != nil {
            return fmt.Errorf("failed to setup namespace: %w", err)
        }
    }
    
    return clientcmd.WriteToFile(*m.config, m.configPath)
}
```

### Multi-Cluster Federation

```go
type ClusterFederation struct {
    contexts map[string]*ContextManager
    router   TrafficRouter
}

func (f *ClusterFederation) DeployToMultipleClusters(deployment MultiClusterDeployment) error {
    results := make(chan DeploymentResult, len(deployment.Clusters))
    
    // Deploy to each cluster in parallel
    for _, cluster := range deployment.Clusters {
        go func(c ClusterTarget) {
            ctx := f.contexts[c.Context]
            if err := ctx.SwitchContext(c.Context, SwitchOptions{}); err != nil {
                results <- DeploymentResult{Cluster: c.Name, Error: err}
                return
            }
            
            // Deploy APM stack
            deployer := NewDeployer(ctx)
            if err := deployer.Deploy(deployment.Manifest, c.Options); err != nil {
                results <- DeploymentResult{Cluster: c.Name, Error: err}
                return
            }
            
            results <- DeploymentResult{Cluster: c.Name, Success: true}
        }(cluster)
    }
    
    // Collect results
    var errors []error
    for i := 0; i < len(deployment.Clusters); i++ {
        result := <-results
        if result.Error != nil {
            errors = append(errors, fmt.Errorf("cluster %s: %w", result.Cluster, result.Error))
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("multi-cluster deployment failed: %v", errors)
    }
    
    // Configure cross-cluster networking if needed
    if deployment.EnableFederation {
        return f.router.ConfigureFederation(deployment.Clusters)
    }
    
    return nil
}
```

## Best Practices and Guidelines

### Security Considerations

1. **RBAC Configuration**
   - Use least privilege principle
   - Create service accounts per component
   - Implement pod security policies
   - Enable audit logging

2. **Secret Management**
   - Use external secret managers (Vault, AWS Secrets Manager)
   - Enable encryption at rest
   - Implement secret rotation
   - Use sealed secrets for GitOps

3. **Network Security**
   - Implement network policies
   - Use service mesh for mTLS
   - Configure ingress with TLS
   - Implement pod-to-pod encryption

### Resource Management

1. **Resource Quotas**
   ```yaml
   apiVersion: v1
   kind: ResourceQuota
   metadata:
     name: apm-quota
     namespace: apm-system
   spec:
     hard:
       requests.cpu: "10"
       requests.memory: 20Gi
       limits.cpu: "20"
       limits.memory: 40Gi
       persistentvolumeclaims: "10"
   ```

2. **Horizontal Pod Autoscaling**
   ```yaml
   apiVersion: autoscaling/v2
   kind: HorizontalPodAutoscaler
   metadata:
     name: prometheus-hpa
   spec:
     scaleTargetRef:
       apiVersion: apps/v1
       kind: Deployment
       name: prometheus
     minReplicas: 2
     maxReplicas: 10
     metrics:
     - type: Resource
       resource:
         name: cpu
         target:
           type: Utilization
           averageUtilization: 70
     - type: Resource
       resource:
         name: memory
         target:
           type: Utilization
           averageUtilization: 80
   ```

### Monitoring and Alerting

1. **Deployment Health Checks**
   - Readiness probes for all containers
   - Liveness probes with appropriate thresholds
   - Startup probes for slow-starting containers

2. **Deployment Metrics**
   - Track deployment success/failure rates
   - Monitor resource utilization
   - Alert on configuration drift
   - Track sidecar injection rates

This comprehensive specification provides a complete foundation for implementing Kubernetes deployment capabilities in the APM stack, with support for advanced features like multi-cloud deployments, automated sidecar injection, and intelligent rollback mechanisms.