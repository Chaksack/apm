# APM Deploy Command Architecture

## Overview

The `apm deploy` command provides a comprehensive deployment solution for the APM stack across multiple platforms including Docker, Kubernetes, and major cloud providers (AWS, Azure, GCP). This document outlines the complete architecture, design decisions, and implementation details.

## Command Structure

### Main Command
```bash
apm deploy [flags]
```

### Subcommands
```bash
apm deploy docker      # Deploy to Docker/Docker Compose
apm deploy kubernetes  # Deploy to Kubernetes cluster
apm deploy cloud       # Deploy to cloud provider (AWS/Azure/GCP)
```

### Global Flags
```bash
--config string         # Path to deployment config file (default: "deploy.yaml")
--env string           # Environment (dev/staging/prod) (default: "dev")
--dry-run              # Validate and show deployment plan without executing
--force                # Force deployment without confirmation
--output string        # Output format (json/yaml/text) (default: "text")
--verbose              # Enable verbose logging
--timeout duration     # Deployment timeout (default: 30m)
```

### Provider-Specific Flags
```bash
# Kubernetes flags
--kubeconfig string    # Path to kubeconfig file
--context string       # Kubernetes context to use
--namespace string     # Target namespace (default: "apm-system")

# Cloud provider flags
--provider string      # Cloud provider (aws/azure/gcp)
--region string        # Cloud region
--cluster string       # Target cluster name
--project string       # GCP project ID
--subscription string  # Azure subscription ID

# Security flags
--credentials-file string  # Path to credentials file
--vault-path string       # HashiCorp Vault path for secrets
--kms-key string         # Cloud KMS key for encryption
```

## Interactive Wizard Flow

### Step 1: Deployment Target Selection
```
┌─────────────────────────────────────────┐
│ Select deployment target:               │
│                                         │
│ ▸ Docker (Local Development)            │
│   Kubernetes (Production Ready)         │
│   AWS (EKS/ECS)                        │
│   Azure (AKS/Container Instances)      │
│   GCP (GKE/Cloud Run)                  │
│                                         │
│ [Use arrows to navigate, Enter to select] │
└─────────────────────────────────────────┘
```

### Step 2: Environment Configuration
```
┌─────────────────────────────────────────┐
│ Select environment:                     │
│                                         │
│ ▸ Development                           │
│   Staging                               │
│   Production                            │
│                                         │
│ Configure:                              │
│ ☑ Enable debug logging                  │
│ ☐ Enable distributed tracing (100%)     │
│ ☑ Enable metrics collection             │
│                                         │
└─────────────────────────────────────────┘
```

### Step 3: Component Selection
```
┌─────────────────────────────────────────┐
│ Select APM components to deploy:        │
│                                         │
│ Core Components:                        │
│ ☑ Prometheus (Metrics)                  │
│ ☑ Grafana (Dashboards)                  │
│ ☑ Loki (Logs)                          │
│ ☑ Jaeger (Tracing)                     │
│ ☑ AlertManager (Alerts)                 │
│                                         │
│ Optional Components:                    │
│ ☐ SonarQube (Code Quality)             │
│ ☐ Istio (Service Mesh)                 │
│ ☐ ArgoCD (GitOps)                      │
│                                         │
└─────────────────────────────────────────┘
```

### Step 4: Resource Configuration
```
┌─────────────────────────────────────────┐
│ Configure resources:                    │
│                                         │
│ Resource Profile:                       │
│ ○ Small (2 CPU, 4GB RAM per component) │
│ ● Medium (4 CPU, 8GB RAM)              │
│ ○ Large (8 CPU, 16GB RAM)              │
│ ○ Custom                                │
│                                         │
│ Storage:                                │
│ Prometheus: [50GB   ] ☑ Persistent      │
│ Loki:       [100GB  ] ☑ Persistent      │
│ Grafana:    [10GB   ] ☑ Persistent      │
│                                         │
└─────────────────────────────────────────┘
```

### Step 5: Security Configuration
```
┌─────────────────────────────────────────┐
│ Configure security:                     │
│                                         │
│ Authentication:                         │
│ ● OAuth2/OIDC                          │
│ ○ Basic Auth                           │
│ ○ No Auth (Dev Only)                   │
│                                         │
│ TLS Configuration:                      │
│ ☑ Enable TLS for all components        │
│ ☑ Auto-generate certificates           │
│ ○ Use existing certificates            │
│                                         │
│ Secrets Management:                     │
│ ● Kubernetes Secrets                    │
│ ○ HashiCorp Vault                      │
│ ○ Cloud KMS                            │
│                                         │
└─────────────────────────────────────────┘
```

### Step 6: Review and Confirmation
```
┌─────────────────────────────────────────┐
│ Deployment Summary:                     │
│                                         │
│ Target: Kubernetes (EKS)                │
│ Environment: Production                 │
│ Region: us-west-2                      │
│ Namespace: apm-system                   │
│                                         │
│ Components:                             │
│ ✓ Prometheus (2 replicas, 50GB)        │
│ ✓ Grafana (2 replicas, 10GB)          │
│ ✓ Loki (3 replicas, 100GB)            │
│ ✓ Jaeger (3 replicas, 50GB)           │
│ ✓ AlertManager (3 replicas)            │
│                                         │
│ Estimated Cost: $450/month              │
│                                         │
│ [Deploy] [Save Config] [Cancel]         │
└─────────────────────────────────────────┘
```

## Data Models

### Core Configuration Models

```go
// DeploymentConfig represents the complete deployment configuration
type DeploymentConfig struct {
    Version     string                `yaml:"version" json:"version"`
    Target      DeploymentTarget      `yaml:"target" json:"target"`
    Environment string                `yaml:"environment" json:"environment"`
    Components  []ComponentConfig     `yaml:"components" json:"components"`
    Resources   ResourceConfig        `yaml:"resources" json:"resources"`
    Security    SecurityConfig        `yaml:"security" json:"security"`
    CloudConfig *CloudProviderConfig  `yaml:"cloudConfig,omitempty" json:"cloudConfig,omitempty"`
    APMConfig   APMInjectionConfig    `yaml:"apmConfig" json:"apmConfig"`
    Metadata    DeploymentMetadata    `yaml:"metadata" json:"metadata"`
}

// DeploymentTarget specifies where to deploy
type DeploymentTarget struct {
    Type     string         `yaml:"type" json:"type"` // docker, kubernetes, aws, azure, gcp
    Platform PlatformConfig `yaml:"platform" json:"platform"`
}

// PlatformConfig contains platform-specific settings
type PlatformConfig struct {
    // Kubernetes specific
    Kubeconfig string `yaml:"kubeconfig,omitempty" json:"kubeconfig,omitempty"`
    Context    string `yaml:"context,omitempty" json:"context,omitempty"`
    Namespace  string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
    
    // Docker specific
    ComposeFile string `yaml:"composeFile,omitempty" json:"composeFile,omitempty"`
    Network     string `yaml:"network,omitempty" json:"network,omitempty"`
    
    // Cloud specific
    ClusterName string `yaml:"clusterName,omitempty" json:"clusterName,omitempty"`
    ServiceType string `yaml:"serviceType,omitempty" json:"serviceType,omitempty"`
}

// ComponentConfig defines a single APM component
type ComponentConfig struct {
    Name         string                 `yaml:"name" json:"name"`
    Enabled      bool                   `yaml:"enabled" json:"enabled"`
    Version      string                 `yaml:"version" json:"version"`
    Replicas     int                    `yaml:"replicas" json:"replicas"`
    Resources    ResourceRequirements   `yaml:"resources" json:"resources"`
    Storage      StorageConfig          `yaml:"storage,omitempty" json:"storage,omitempty"`
    Config       map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
    Dependencies []string               `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
    Ports        []PortConfig           `yaml:"ports,omitempty" json:"ports,omitempty"`
}

// ResourceRequirements defines CPU and memory requirements
type ResourceRequirements struct {
    Requests ResourceSpec `yaml:"requests" json:"requests"`
    Limits   ResourceSpec `yaml:"limits" json:"limits"`
}

// ResourceSpec defines specific resource values
type ResourceSpec struct {
    CPU    string `yaml:"cpu" json:"cpu"`       // e.g., "2", "500m"
    Memory string `yaml:"memory" json:"memory"` // e.g., "4Gi", "2048Mi"
}

// StorageConfig defines storage requirements
type StorageConfig struct {
    Size         string `yaml:"size" json:"size"`                                   // e.g., "50Gi"
    StorageClass string `yaml:"storageClass,omitempty" json:"storageClass,omitempty"`
    Persistent   bool   `yaml:"persistent" json:"persistent"`
}

// SecurityConfig defines security settings
type SecurityConfig struct {
    Authentication AuthConfig       `yaml:"authentication" json:"authentication"`
    TLS            TLSConfig        `yaml:"tls" json:"tls"`
    Secrets        SecretsConfig    `yaml:"secrets" json:"secrets"`
    RBAC           RBACConfig       `yaml:"rbac" json:"rbac"`
    NetworkPolicy  NetworkPolicy    `yaml:"networkPolicy" json:"networkPolicy"`
}

// CloudProviderConfig contains cloud-specific configuration
type CloudProviderConfig struct {
    Provider    string           `yaml:"provider" json:"provider"` // aws, azure, gcp
    Region      string           `yaml:"region" json:"region"`
    Credentials CredentialSource `yaml:"credentials" json:"credentials"`
    Networking  NetworkConfig    `yaml:"networking" json:"networking"`
    Storage     CloudStorage     `yaml:"storage" json:"storage"`
    
    // AWS specific
    AWS *AWSConfig `yaml:"aws,omitempty" json:"aws,omitempty"`
    
    // Azure specific
    Azure *AzureConfig `yaml:"azure,omitempty" json:"azure,omitempty"`
    
    // GCP specific
    GCP *GCPConfig `yaml:"gcp,omitempty" json:"gcp,omitempty"`
}

// APMInjectionConfig defines how APM is injected into applications
type APMInjectionConfig struct {
    AutoInject    bool                   `yaml:"autoInject" json:"autoInject"`
    Sidecars      []SidecarConfig        `yaml:"sidecars,omitempty" json:"sidecars,omitempty"`
    EnvVars       map[string]string      `yaml:"envVars,omitempty" json:"envVars,omitempty"`
    Annotations   map[string]string      `yaml:"annotations,omitempty" json:"annotations,omitempty"`
    ConfigMounts  []ConfigMount          `yaml:"configMounts,omitempty" json:"configMounts,omitempty"`
}
```

### Deployment State Management

```go
// DeploymentState tracks deployment progress and history
type DeploymentState struct {
    ID          string             `json:"id"`
    Config      DeploymentConfig   `json:"config"`
    Status      DeploymentStatus   `json:"status"`
    StartTime   time.Time          `json:"startTime"`
    EndTime     *time.Time         `json:"endTime,omitempty"`
    Components  []ComponentState   `json:"components"`
    Error       *DeploymentError   `json:"error,omitempty"`
    Rollback    *RollbackInfo      `json:"rollback,omitempty"`
}

// ComponentState tracks individual component deployment
type ComponentState struct {
    Name      string           `json:"name"`
    Status    ComponentStatus  `json:"status"`
    Message   string           `json:"message"`
    Resources []ResourceInfo   `json:"resources"`
    StartTime time.Time        `json:"startTime"`
    EndTime   *time.Time       `json:"endTime,omitempty"`
}

// RollbackInfo contains rollback details
type RollbackInfo struct {
    PreviousID   string    `json:"previousId"`
    Reason       string    `json:"reason"`
    StartTime    time.Time `json:"startTime"`
    Success      bool      `json:"success"`
}
```

## Cloud Provider Integration

### AWS Integration

```go
// AWS deployment flow
1. Validate AWS credentials (AWS CLI/SDK)
2. Check/Create EKS cluster or ECS service
3. Configure IAM roles and policies
4. Set up VPC and security groups
5. Deploy Load Balancers
6. Configure Route53 for DNS
7. Set up CloudWatch integration
8. Deploy APM components via Helm/ECS tasks
```

### Azure Integration

```go
// Azure deployment flow
1. Validate Azure credentials (Azure CLI)
2. Check/Create AKS cluster or Container Instances
3. Configure Azure AD integration
4. Set up Virtual Network and NSGs
5. Deploy Application Gateway
6. Configure Azure DNS
7. Set up Azure Monitor integration
8. Deploy APM components via Helm/ACI
```

### GCP Integration

```go
// GCP deployment flow
1. Validate GCP credentials (gcloud)
2. Check/Create GKE cluster or Cloud Run services
3. Configure IAM and service accounts
4. Set up VPC and firewall rules
5. Deploy Load Balancers
6. Configure Cloud DNS
7. Set up Cloud Monitoring integration
8. Deploy APM components via Helm/Cloud Run
```

## APM Configuration Injection

### Automatic Instrumentation
```yaml
# Injected into application pods
env:
  - name: OTEL_EXPORTER_OTLP_ENDPOINT
    value: "http://jaeger-collector:4317"
  - name: PROMETHEUS_ENDPOINT
    value: "http://prometheus:9090"
  - name: LOG_LEVEL
    value: "info"
  - name: APM_ENABLED
    value: "true"
```

### Sidecar Injection
```yaml
# Prometheus exporter sidecar
- name: prometheus-exporter
  image: prom/node-exporter:latest
  ports:
    - containerPort: 9100
  resources:
    limits:
      memory: "128Mi"
      cpu: "100m"
```

### ConfigMap Mounting
```yaml
# APM configuration mounted to pods
volumeMounts:
  - name: apm-config
    mountPath: /etc/apm
    readOnly: true
volumes:
  - name: apm-config
    configMap:
      name: apm-configuration
```

## Security Implementation

### Credential Management
```go
type CredentialManager interface {
    GetCredentials(source CredentialSource) (Credentials, error)
    RotateCredentials() error
    ValidateCredentials() error
}

// Implementation for different sources
- Environment variables
- Local files (encrypted)
- HashiCorp Vault
- AWS Secrets Manager
- Azure Key Vault
- GCP Secret Manager
```

### RBAC Configuration
```yaml
# Generated RBAC for APM components
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: apm-prometheus
rules:
  - apiGroups: [""]
    resources: ["nodes", "pods", "services"]
    verbs: ["get", "list", "watch"]
```

### Network Policies
```yaml
# Restrict traffic between components
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: apm-network-policy
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/part-of: apm
  policyTypes:
    - Ingress
    - Egress
```

## Error Handling and Recovery

### Pre-deployment Validation
```go
func ValidateDeployment(config DeploymentConfig) []ValidationError {
    var errors []ValidationError
    
    // Check connectivity
    if err := checkClusterConnectivity(); err != nil {
        errors = append(errors, ValidationError{
            Type: "connectivity",
            Message: "Cannot connect to cluster",
        })
    }
    
    // Validate resources
    if err := validateResourceAvailability(); err != nil {
        errors = append(errors, ValidationError{
            Type: "resources",
            Message: "Insufficient resources",
        })
    }
    
    // Check dependencies
    if err := validateDependencies(); err != nil {
        errors = append(errors, ValidationError{
            Type: "dependencies",
            Message: "Missing dependencies",
        })
    }
    
    return errors
}
```

### Rollback Strategy
```go
type RollbackStrategy interface {
    CreateSnapshot() (string, error)
    Rollback(snapshotID string) error
    ValidateRollback() error
}

// Automatic rollback on failure
func DeployWithRollback(config DeploymentConfig) error {
    snapshot, err := strategy.CreateSnapshot()
    if err != nil {
        return err
    }
    
    if err := deploy(config); err != nil {
        log.Error("Deployment failed, initiating rollback")
        if rbErr := strategy.Rollback(snapshot); rbErr != nil {
            return fmt.Errorf("deployment failed and rollback failed: %v, %v", err, rbErr)
        }
        return err
    }
    
    return nil
}
```

### State Recovery
```go
// Deployment state tracking
type StateManager struct {
    store StateStore
}

func (sm *StateManager) SaveState(state DeploymentState) error {
    return sm.store.Save(state)
}

func (sm *StateManager) RecoverState(deploymentID string) (*DeploymentState, error) {
    return sm.store.Get(deploymentID)
}

func (sm *StateManager) ListDeployments() ([]DeploymentState, error) {
    return sm.store.List()
}
```

## Progress Tracking

### Real-time Updates
```go
type ProgressReporter interface {
    Start(total int)
    Update(component string, status ComponentStatus, message string)
    Complete()
    Error(err error)
}

// Terminal UI progress
func (r *TerminalReporter) Update(component string, status ComponentStatus, message string) {
    icon := getStatusIcon(status)
    color := getStatusColor(status)
    fmt.Printf("%s %s %s: %s\n", icon, color(component), status, message)
}
```

### Deployment Logs
```go
// Structured logging for deployments
logger.Info("Starting deployment",
    zap.String("deploymentID", deployment.ID),
    zap.String("target", deployment.Target.Type),
    zap.String("environment", deployment.Environment),
    zap.Int("components", len(deployment.Components)),
)
```

## Example Usage

### Interactive Mode
```bash
$ apm deploy
? Select deployment target: Kubernetes
? Select environment: Production
? Select components: [Prometheus, Grafana, Loki, Jaeger, AlertManager]
? Configure resources: Medium (4 CPU, 8GB RAM)
? Configure security: OAuth2/OIDC
? Review and deploy? Yes

Deploying APM stack to Kubernetes...
✓ Prometheus:     Deployed (2/2 replicas ready)
✓ Grafana:        Deployed (2/2 replicas ready)
✓ Loki:           Deployed (3/3 replicas ready)
✓ Jaeger:         Deployed (3/3 replicas ready)
✓ AlertManager:   Deployed (3/3 replicas ready)

Deployment successful! Access your APM stack:
- Grafana:     https://grafana.example.com
- Prometheus:  https://prometheus.example.com
- Jaeger:      https://jaeger.example.com
```

### Configuration File Mode
```bash
$ apm deploy --config production-deploy.yaml --dry-run

Deployment Plan:
Target: AWS EKS (us-west-2)
Environment: Production
Components:
  - Prometheus (2 replicas, 50GB storage)
  - Grafana (2 replicas, 10GB storage)
  - Loki (3 replicas, 100GB storage)
  - Jaeger (3 replicas, 50GB storage)
  - AlertManager (3 replicas)

Resources Required:
  - Total CPU: 24 cores
  - Total Memory: 48GB
  - Total Storage: 210GB

Estimated Cost: $450/month

Run without --dry-run to execute deployment.
```

### Rollback Example
```bash
$ apm deploy rollback --deployment-id abc123

Rolling back deployment abc123...
✓ Snapshot found: snapshot-xyz789
✓ Restoring Prometheus
✓ Restoring Grafana
✓ Restoring Loki
✓ Restoring Jaeger
✓ Restoring AlertManager

Rollback completed successfully.
```

## Implementation Phases

### Phase 1: Core Framework
- Command structure and CLI integration
- Configuration models and validation
- Basic Docker deployment

### Phase 2: Kubernetes Support
- Kubernetes client integration
- Manifest generation and templating
- Helm chart deployment

### Phase 3: Cloud Provider Integration
- AWS SDK integration (EKS, ECS)
- Azure CLI integration (AKS)
- GCP client integration (GKE)

### Phase 4: Advanced Features
- Sidecar injection
- APM auto-instrumentation
- State management and rollback

### Phase 5: Security and Polish
- Credential management
- RBAC and network policies
- Progress tracking and UI