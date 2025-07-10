# Cloud Provider CLI Integration

This document describes the cloud provider CLI integration architecture for AWS, Azure, and Google Cloud Platform.

## Overview

The cloud provider integration provides a unified interface for interacting with AWS, Azure, and GCP cloud services, focusing on:
- Container registry management (ECR, ACR, GCR)
- Kubernetes cluster management (EKS, AKS, GKE)
- Authentication and credential management
- Cross-platform compatibility

## Architecture

### Core Components

1. **Types (`pkg/cloud/types.go`)**
   - Common data structures for all cloud providers
   - Provider-agnostic interfaces
   - Credential and authentication types

2. **CLI Detection (`pkg/cloud/detector.go`)**
   - Automatic detection of installed CLIs
   - Version validation
   - Installation instructions
   - Platform-specific compatibility

3. **Credential Management (`pkg/cloud/credentials.go`)**
   - Secure credential storage using OS keychain
   - Encryption at rest using AES-GCM
   - Support for multiple authentication methods
   - Credential rotation capabilities

4. **Provider Implementations**
   - AWS (`pkg/cloud/aws.go`)
   - Azure (`pkg/cloud/azure.go`)
   - GCP (`pkg/cloud/gcp.go`)

5. **Factory and Manager (`pkg/cloud/factory.go`)**
   - Provider factory pattern
   - Multi-cloud operations
   - Concurrent operations support

## CLI Detection and Validation

### Detection Methods

The system automatically detects installed cloud CLIs using:

```go
// Example usage
detector := NewAWSCLIDetector()
status, err := detector.Detect()
if err != nil {
    log.Fatal(err)
}

if !status.Installed {
    fmt.Println(detector.GetInstallInstructions())
} else {
    fmt.Printf("AWS CLI v%s installed at %s\n", status.Version, status.Path)
}
```

### Validation Process

1. **CLI Presence**: Check if CLI binary exists in PATH
2. **Version Check**: Ensure minimum version requirements
3. **Authentication**: Verify active authentication
4. **Configuration**: Check for valid configuration files

### Cross-Platform Support

The integration supports Windows, macOS, and Linux with platform-specific:
- Configuration paths
- Environment variables
- Installation methods

## Authentication and Credential Management

### Authentication Methods

1. **CLI Authentication** (Recommended)
   - Uses existing CLI credentials
   - No credential storage required
   - Automatic token refresh

2. **Access Key Authentication**
   - Direct credential storage
   - Encrypted at rest
   - Manual rotation required

3. **Service Principal/Key Authentication**
   - For service accounts
   - Long-lived credentials
   - Suitable for automation

4. **IAM Role Authentication**
   - For cloud-native workloads
   - No credential management
   - Automatic rotation

### Secure Credential Storage

Credentials are encrypted using:
- AES-256-GCM encryption
- PBKDF2 key derivation
- Machine-specific encryption keys
- File permissions (0600)

```go
// Example: Store credentials securely
credMgr, _ := NewSecureCredentialManager("/path/to/store")
creds := &Credentials{
    Provider:   ProviderAWS,
    AuthMethod: AuthMethodAccessKey,
    AccessKey:  "AKIAIOSFODNN7EXAMPLE",
    SecretKey:  "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
    Region:     "us-east-1",
}
err := credMgr.Store(creds)
```

### Credential Retrieval Hierarchy

1. Environment variables (highest priority)
2. CLI configuration files
3. Stored credentials
4. Default credentials

## Container Registry Integration

### Supported Registries

- **AWS**: Elastic Container Registry (ECR)
- **Azure**: Azure Container Registry (ACR)
- **GCP**: Google Container Registry (GCR) and Artifact Registry

### Registry Operations

```go
// List all registries
provider, _ := NewAWSProvider(config)
registries, err := provider.ListRegistries(ctx)

// Authenticate to registry
err = provider.AuthenticateRegistry(ctx, registry)

// Docker operations are then available
// docker push registry.url/image:tag
```

### Authentication Flow

1. Get authentication token from cloud provider
2. Configure Docker with credentials
3. Token refresh handled automatically by CLI

## Kubernetes Cluster Integration

### Supported Clusters

- **AWS**: Elastic Kubernetes Service (EKS)
- **Azure**: Azure Kubernetes Service (AKS)
- **GCP**: Google Kubernetes Engine (GKE)

### Cluster Operations

```go
// List clusters
clusters, err := provider.ListClusters(ctx)

// Get cluster details
cluster, err := provider.GetCluster(ctx, "my-cluster")

// Get kubeconfig
kubeconfig, err := provider.GetKubeconfig(ctx, cluster)
```

### Kubeconfig Management

- Temporary kubeconfig generation
- Automatic context switching
- Credential injection for kubectl

## API Fallback Options

When CLI is not available, the system provides API fallback:

```go
fallback := NewAWSAPIFallback(provider)
if fallback.IsAvailable() {
    clusters, err := fallback.ListClustersViaAPI(ctx)
}
```

### Fallback Requirements

- Direct API credentials (not CLI-based)
- Cloud provider SDK installation
- Network connectivity to cloud APIs

## Multi-Cloud Operations

The CloudManager provides unified operations across providers:

```go
// Initialize manager
manager, _ := NewCloudManager("/path/to/credentials")

// Register providers
manager.RegisterProvider(ProviderAWS, awsConfig)
manager.RegisterProvider(ProviderAzure, azureConfig)
manager.RegisterProvider(ProviderGCP, gcpConfig)

// List all clusters across providers
allClusters, err := manager.ListAllClusters(ctx)

// Find cluster by name
cluster, provider, err := multiCloudOps.FindCluster(ctx, "my-cluster")
```

## Security Best Practices

### Minimal Permissions

Required permissions for each provider:

**AWS**:
- `ecr:GetAuthorizationToken`
- `ecr:DescribeRepositories`
- `eks:ListClusters`
- `eks:DescribeCluster`

**Azure**:
- `Microsoft.ContainerRegistry/registries/read`
- `Microsoft.ContainerService/managedClusters/read`
- `Microsoft.ContainerService/managedClusters/listClusterUserCredential/action`

**GCP**:
- `container.clusters.list`
- `container.clusters.get`
- `artifactregistry.repositories.list`

### Temporary Credentials

- Use temporary tokens when possible
- Set expiration times
- Automatic cleanup of expired credentials

### Audit Logging

All operations are logged with:
- Timestamp
- Provider and operation
- Success/failure status
- Error details if applicable

## Usage Examples

### Basic Setup

```go
import "github.com/yourusername/apm/pkg/cloud"

// Create cloud manager
manager, err := cloud.NewCloudManager("~/.apm/credentials")
if err != nil {
    log.Fatal(err)
}

// Validate environment
results := manager.ValidateEnvironment(context.Background())
for provider, result := range results {
    if !result.Valid {
        fmt.Printf("%s: %v\n", provider, result.Errors)
    }
}
```

### Working with Registries

```go
// Get AWS provider
aws, _ := manager.GetProvider(cloud.ProviderAWS)

// List ECR registries
registries, err := aws.ListRegistries(ctx)
for _, reg := range registries {
    fmt.Printf("Registry: %s (%s)\n", reg.Name, reg.URL)
}

// Authenticate for Docker operations
if err := aws.AuthenticateRegistry(ctx, registries[0]); err != nil {
    log.Fatal(err)
}
```

### Working with Clusters

```go
// Get all clusters across providers
allClusters, _ := manager.ListAllClusters(ctx)
for provider, clusters := range allClusters {
    fmt.Printf("\n%s Clusters:\n", provider)
    for _, cluster := range clusters {
        fmt.Printf("  - %s (%s) - %d nodes\n", 
            cluster.Name, cluster.Status, cluster.NodeCount)
    }
}

// Get kubeconfig for specific cluster
azure, _ := manager.GetProvider(cloud.ProviderAzure)
cluster, _ := azure.GetCluster(ctx, "my-aks-cluster")
kubeconfig, _ := azure.GetKubeconfig(ctx, cluster)

// Write kubeconfig to file
os.WriteFile("kubeconfig.yaml", kubeconfig, 0600)
```

## Error Handling

The integration provides detailed error information:

```go
if err != nil {
    switch e := err.(type) {
    case *cloud.AuthenticationError:
        // Handle auth errors
    case *cloud.CLINotFoundError:
        // Show installation instructions
    case *cloud.VersionError:
        // Handle version mismatch
    default:
        // Generic error handling
    }
}
```

## Testing

### Unit Tests

- Mock CLI commands
- Test credential encryption/decryption
- Validate detection logic

### Integration Tests

- Real CLI interaction (requires CLIs installed)
- Live cloud provider APIs
- End-to-end workflows

## Future Enhancements

1. **Additional Providers**
   - Oracle Cloud Infrastructure (OCI)
   - IBM Cloud
   - Alibaba Cloud

2. **Enhanced Features**
   - Automated credential rotation
   - Multi-factor authentication support
   - Cloud resource tagging and management

3. **Performance Improvements**
   - Parallel operations
   - Response caching
   - Connection pooling

## Troubleshooting

### Common Issues

1. **CLI Not Detected**
   - Ensure CLI is in PATH
   - Check version compatibility
   - Verify installation location

2. **Authentication Failures**
   - Check credential expiration
   - Verify permissions
   - Ensure network connectivity

3. **Registry Access Issues**
   - Confirm Docker daemon is running
   - Check registry permissions
   - Verify network/firewall settings

### Debug Mode

Enable debug logging:
```go
os.Setenv("APM_CLOUD_DEBUG", "true")
```

This will output:
- CLI command execution
- API calls
- Credential operations (sanitized)
- Error stack traces