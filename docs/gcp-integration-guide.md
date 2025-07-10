# Google Cloud Platform (GCP) Integration Guide

## Overview

This guide provides comprehensive documentation for the GCP CLI integration in the APM tool. The integration provides full support for Google Cloud services including GCR/Artifact Registry, GKE clusters, service accounts, Cloud Monitoring, Cloud Trace, and Cloud Storage.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Authentication Methods](#authentication-methods)
4. [Core Features](#core-features)
5. [Advanced Operations](#advanced-operations)
6. [APM Integration](#apm-integration)
7. [Error Handling](#error-handling)
8. [Best Practices](#best-practices)
9. [Troubleshooting](#troubleshooting)
10. [Examples](#examples)

## Prerequisites

### Required Software

1. **Google Cloud CLI (gcloud)** - Version 400.0.0 or later
   ```bash
   # Install on macOS
   brew install google-cloud-sdk
   
   # Install on Linux
   curl https://sdk.cloud.google.com | bash
   
   # Install on Windows
   # Download from: https://cloud.google.com/sdk/docs/install#windows
   ```

2. **kubectl** (for GKE operations)
   ```bash
   # Install kubectl
   gcloud components install kubectl
   ```

3. **Docker** (for container registry operations)
   ```bash
   # Configure Docker to use gcloud as credential helper
   gcloud auth configure-docker
   ```

### Authentication Setup

Choose one of the following authentication methods:

#### 1. User Authentication (Recommended for development)
```bash
gcloud auth login
gcloud config set project YOUR_PROJECT_ID
```

#### 2. Service Account Authentication (Recommended for production)
```bash
# Create and download service account key
gcloud iam service-accounts create apm-service-account
gcloud iam service-accounts keys create key.json \
  --iam-account apm-service-account@YOUR_PROJECT_ID.iam.gserviceaccount.com

# Set environment variable
export GOOGLE_APPLICATION_CREDENTIALS=./key.json
```

#### 3. Application Default Credentials
```bash
gcloud auth application-default login
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/chaksack/apm/pkg/cloud"
)

func main() {
    ctx := context.Background()
    
    // Create GCP provider
    provider, err := cloud.NewGCPProvider(&cloud.ProviderConfig{
        Provider:      cloud.ProviderGCP,
        DefaultRegion: "us-central1",
        EnableCache:   true,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Validate setup
    if err := provider.ValidateAuth(ctx); err != nil {
        log.Fatal("Authentication failed:", err)
    }
    
    // List GKE clusters
    clusters, err := provider.ListClusters(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, cluster := range clusters {
        log.Printf("Cluster: %s in %s", cluster.Name, cluster.Region)
    }
}
```

## Authentication Methods

### 1. CLI Authentication

The most common method for interactive use:

```go
// This uses the current gcloud authentication
provider, _ := cloud.NewGCPProvider(config)
err := provider.ValidateAuth(ctx)
```

### 2. Service Account Key File

For automated/production environments:

```go
authManager := provider.GetAdvancedOperations().GetAuthenticationManager()
err := authManager.AuthenticateWithServiceAccount(ctx, "/path/to/key.json")
```

### 3. Application Default Credentials (ADC)

For applications running on GCP or with ADC configured:

```go
authManager := provider.GetAdvancedOperations().GetAuthenticationManager()
err := authManager.AuthenticateWithApplicationDefaultCredentials(ctx)
```

### 4. OAuth2 Flow

For applications requiring OAuth2:

```go
authManager := provider.GetAdvancedOperations().GetAuthenticationManager()
err := authManager.AuthenticateWithOAuth2(ctx)
```

### 5. Workload Identity (GKE)

For applications running in GKE:

```go
authManager := provider.GetAdvancedOperations().GetAuthenticationManager()
err := authManager.SetupWorkloadIdentity(ctx, 
    "project-id", "cluster-name", "location", 
    "namespace", "k8s-service-account", "gcp-service-account")
```

## Core Features

### Project Management

```go
resourceManager := provider.GetAdvancedOperations().GetResourceManager()

// List projects
projects, err := resourceManager.ListProjects(ctx)

// Get current project
currentProject, err := resourceManager.GetCurrentProject(ctx)

// Set current project
err = resourceManager.SetCurrentProject(ctx, "new-project-id")

// Get project details
project, err := resourceManager.GetProject(ctx, "project-id")
```

### Container Registry Operations

#### GCR (Google Container Registry)
```go
// List all registries (includes both GCR and Artifact Registry)
registries, err := provider.ListRegistries(ctx)

// Authenticate Docker with GCR
registry, err := provider.GetRegistry(ctx, "gcr.io")
err = provider.AuthenticateRegistry(ctx, registry)

// After authentication, you can use Docker commands:
// docker push gcr.io/PROJECT_ID/IMAGE_NAME
```

#### Artifact Registry
```go
// Artifact Registry repositories are included in ListRegistries()
for _, registry := range registries {
    if registry.Type == "Artifact Registry" {
        fmt.Printf("Artifact Registry: %s in %s\n", registry.Name, registry.Region)
    }
}

// Authenticate with Artifact Registry
err = provider.AuthenticateRegistry(ctx, artifactRegistry)
```

### GKE Cluster Management

```go
// List all GKE clusters
clusters, err := provider.ListClusters(ctx)

// Get specific cluster
cluster, err := provider.GetCluster(ctx, "cluster-name")

// Get kubeconfig for cluster
kubeconfig, err := provider.GetKubeconfig(ctx, cluster)

// Save kubeconfig to file
err = ioutil.WriteFile("kubeconfig.yaml", kubeconfig, 0644)
```

### Service Account Management

```go
serviceAccountManager := provider.GetAdvancedOperations().GetServiceAccountManager()

// List service accounts
serviceAccounts, err := serviceAccountManager.ListServiceAccounts(ctx)

// Create service account
sa, err := serviceAccountManager.CreateServiceAccount(ctx, 
    "account-id", "Display Name", "Description")

// Create service account key
key, err := serviceAccountManager.CreateServiceAccountKey(ctx, 
    "sa@project.iam.gserviceaccount.com", "key.json")

// List service account keys
keys, err := serviceAccountManager.ListServiceAccountKeys(ctx, 
    "sa@project.iam.gserviceaccount.com")

// Delete service account key
err = serviceAccountManager.DeleteServiceAccountKey(ctx, 
    "sa@project.iam.gserviceaccount.com", "key-id")
```

## Advanced Operations

### Cloud Monitoring Integration

```go
monitoringManager := provider.GetAdvancedOperations().GetMonitoringManager()

// Enable Cloud Monitoring API
err := monitoringManager.EnableMonitoringAPI(ctx)

// Enable Cloud Trace API
err = monitoringManager.EnableCloudTraceAPI(ctx)

// List monitoring workspaces
workspaces, err := monitoringManager.ListMonitoringWorkspaces(ctx)
```

### Cloud Storage Management

```go
storageManager := provider.GetAdvancedOperations().GetStorageManager()

// List storage buckets
buckets, err := storageManager.ListStorageBuckets(ctx)

// Create storage bucket
bucket, err := storageManager.CreateStorageBucket(ctx, 
    "bucket-name", "us-central1", "STANDARD")

// Get bucket details
bucket, err = storageManager.GetStorageBucket(ctx, "bucket-name")

// Delete bucket
err = storageManager.DeleteStorageBucket(ctx, "bucket-name", true)
```

### Token Management

```go
authManager := provider.GetAdvancedOperations().GetAuthenticationManager()

// Get access token
token, err := authManager.GetAccessToken(ctx)

// Get identity token for specific audience
identityToken, err := authManager.GetIdentityToken(ctx, "audience")

// List active accounts
accounts, err := authManager.ListActiveAccounts(ctx)

// Switch to different account
err = authManager.SwitchAccount(ctx, "account@example.com")

// Revoke authentication
err = authManager.RevokeAuthentication(ctx, "account@example.com")
```

## APM Integration

### Comprehensive APM Setup

The GCP provider includes a one-stop method for setting up complete APM integration:

```go
advOps := provider.GetAdvancedOperations()

config := cloud.APMIntegrationConfig{
    ProjectID:            "my-project",
    Region:               "us-central1",
    ServiceAccountID:     "apm-monitoring",
    CreateServiceAccount: true,
    SetupMonitoring:      true,
    CreateStorageBucket:  true,
    EnableWorkloadIdentity: true,
    ClusterName:          "my-gke-cluster",
    ClusterLocation:      "us-central1-a",
}

err := advOps.SetupAPMIntegration(ctx, config)
```

This will:
1. Enable all required APIs
2. Create a service account for APM
3. Generate service account keys
4. Set up Cloud Monitoring workspace
5. Enable Cloud Trace
6. Create storage bucket for logs/traces
7. Configure Workload Identity (if specified)

### Manual API Enablement

```go
// Enable all APIs required for APM
err := provider.EnableRequiredAPIs(ctx)

// Or enable specific APIs
apis := []string{
    "monitoring.googleapis.com",
    "cloudtrace.googleapis.com",
    "logging.googleapis.com",
}

for _, api := range apis {
    cmd := exec.Command("gcloud", "services", "enable", api)
    err := cmd.Run()
}
```

## Error Handling

### Common Error Scenarios

1. **Authentication Errors**
   ```go
   if err := provider.ValidateAuth(ctx); err != nil {
       if strings.Contains(err.Error(), "not authenticated") {
           fmt.Println("Please run: gcloud auth login")
           return
       }
       if strings.Contains(err.Error(), "no active account") {
           fmt.Println("Please set up authentication")
           return
       }
   }
   ```

2. **API Not Enabled**
   ```go
   if strings.Contains(err.Error(), "API not enabled") {
       fmt.Println("Enabling required API...")
       // Enable the API and retry
   }
   ```

3. **Permission Errors**
   ```go
   if strings.Contains(err.Error(), "permission denied") {
       fmt.Println("Insufficient permissions. Required roles:")
       fmt.Println("- Compute Viewer")
       fmt.Println("- Container Developer") 
       fmt.Println("- Storage Admin")
   }
   ```

### Error Logging

The provider includes comprehensive error logging:

```go
// All errors include context and suggestions
// Example error output:
// "failed to list clusters: API not enabled. Run: gcloud services enable container.googleapis.com"
```

## Best Practices

### 1. Authentication
- Use service accounts for production environments
- Use user authentication for development
- Enable Workload Identity for GKE applications
- Rotate service account keys regularly

### 2. Project Management
- Always validate project access before operations
- Use project-specific service accounts
- Tag resources appropriately

### 3. Resource Management
- Cache credentials when possible
- Use regional resources when appropriate
- Clean up unused resources

### 4. Security
- Use least-privilege access
- Enable audit logging
- Use VPC-native clusters
- Enable network policies

### 5. Performance
- Enable credential caching
- Use regional endpoints
- Batch operations when possible

## Troubleshooting

### Common Issues

1. **gcloud CLI not found**
   ```bash
   # Solution: Install gcloud CLI
   curl https://sdk.cloud.google.com | bash
   exec -l $SHELL
   gcloud init
   ```

2. **Authentication failed**
   ```bash
   # Solution: Authenticate with gcloud
   gcloud auth login
   gcloud config set project PROJECT_ID
   ```

3. **API not enabled**
   ```bash
   # Solution: Enable required APIs
   gcloud services enable container.googleapis.com
   gcloud services enable monitoring.googleapis.com
   ```

4. **Permission denied**
   ```bash
   # Solution: Grant required roles
   gcloud projects add-iam-policy-binding PROJECT_ID \
     --member="user:EMAIL" \
     --role="roles/container.developer"
   ```

5. **Region/Zone not set**
   ```bash
   # Solution: Set default region/zone
   gcloud config set compute/region us-central1
   gcloud config set compute/zone us-central1-a
   ```

### Debug Mode

Enable debug logging for troubleshooting:

```go
// Set environment variable for detailed gcloud output
os.Setenv("CLOUDSDK_CORE_VERBOSITY", "debug")

// Check gcloud configuration
cmd := exec.Command("gcloud", "config", "list")
output, _ := cmd.Output()
fmt.Println(string(output))
```

## Examples

### Complete Example: Setting up APM for GKE

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/chaksack/apm/pkg/cloud"
)

func main() {
    ctx := context.Background()
    
    // Initialize provider
    provider, err := cloud.NewGCPProvider(&cloud.ProviderConfig{
        Provider:      cloud.ProviderGCP,
        DefaultRegion: "us-central1",
        EnableCache:   true,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Validate authentication
    if err := provider.ValidateAuth(ctx); err != nil {
        log.Fatal("Please authenticate: gcloud auth login")
    }
    
    projectID := provider.GetProjectID()
    if projectID == "" {
        log.Fatal("No project set. Run: gcloud config set project PROJECT_ID")
    }
    
    fmt.Printf("Setting up APM for project: %s\n", projectID)
    
    // Enable required APIs
    fmt.Println("Enabling required APIs...")
    if err := provider.EnableRequiredAPIs(ctx); err != nil {
        log.Printf("Warning: Failed to enable some APIs: %v", err)
    }
    
    // Set up APM integration
    advOps := provider.GetAdvancedOperations()
    config := cloud.APMIntegrationConfig{
        ProjectID:            projectID,
        Region:               "us-central1",
        ServiceAccountID:     "apm-monitoring",
        CreateServiceAccount: true,
        SetupMonitoring:      true,
        CreateStorageBucket:  true,
    }
    
    fmt.Println("Setting up APM integration...")
    if err := advOps.SetupAPMIntegration(ctx, config); err != nil {
        log.Fatal("APM setup failed:", err)
    }
    
    // Configure Docker for registry access
    registries, _ := provider.ListRegistries(ctx)
    for _, registry := range registries {
        if registry.Type == "GCR" {
            provider.AuthenticateRegistry(ctx, registry)
        }
    }
    
    fmt.Println("APM setup completed successfully!")
    fmt.Println("\nNext steps:")
    fmt.Println("1. Deploy your applications to GKE")
    fmt.Println("2. Configure Prometheus to scrape metrics")
    fmt.Println("3. Set up Grafana dashboards")
    fmt.Println("4. Configure alerting rules")
}
```

### Example: Working with Service Accounts

```go
func manageServiceAccounts(ctx context.Context, provider *cloud.GCPProvider) {
    sam := provider.GetAdvancedOperations().GetServiceAccountManager()
    
    // Create service account for monitoring
    sa, err := sam.CreateServiceAccount(ctx, 
        "monitoring-sa", 
        "Monitoring Service Account", 
        "Used for APM monitoring and logging")
    if err != nil {
        log.Printf("Failed to create service account: %v", err)
        return
    }
    
    // Create key for the service account
    keyPath := "monitoring-sa-key.json"
    _, err = sam.CreateServiceAccountKey(ctx, sa.Email, keyPath)
    if err != nil {
        log.Printf("Failed to create service account key: %v", err)
        return
    }
    
    fmt.Printf("Service account created: %s\n", sa.Email)
    fmt.Printf("Key saved to: %s\n", keyPath)
    
    // Grant necessary roles (this would be done via gcloud or IAM API)
    fmt.Println("Don't forget to grant necessary roles:")
    fmt.Printf("gcloud projects add-iam-policy-binding %s \\\n", provider.GetProjectID())
    fmt.Printf("  --member='serviceAccount:%s' \\\n", sa.Email)
    fmt.Printf("  --role='roles/monitoring.writer'\n")
}
```

This comprehensive guide covers all aspects of the GCP integration. For additional examples and advanced usage patterns, see the `/examples/gcp-comprehensive/` directory.