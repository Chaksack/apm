# GCP Comprehensive Integration Example

This example demonstrates the complete Google Cloud Platform (GCP) CLI integration capabilities of the APM tool.

## Features Demonstrated

- ✅ **CLI Detection & Validation** - Automatic detection of gcloud CLI installation
- ✅ **Authentication Management** - Multiple authentication methods (CLI, Service Account, OAuth2, ADC, Workload Identity)
- ✅ **Project Management** - List, get, and switch between GCP projects
- ✅ **Service Account Operations** - Create, manage, and authenticate with service accounts
- ✅ **Container Registry Support** - Both GCR and Artifact Registry authentication
- ✅ **GKE Cluster Management** - List clusters, get kubeconfig, and manage cluster credentials
- ✅ **Cloud Storage Integration** - Manage Cloud Storage buckets for APM data
- ✅ **Cloud Monitoring & Trace** - Enable and configure monitoring APIs
- ✅ **Complete APM Setup** - One-command APM infrastructure setup

## Prerequisites

1. **Google Cloud CLI** installed and configured:
   ```bash
   # Install gcloud (macOS)
   brew install google-cloud-sdk
   
   # Install gcloud (Linux)
   curl https://sdk.cloud.google.com | bash
   
   # Initialize
   gcloud init
   ```

2. **Authentication** - Choose one method:
   ```bash
   # Method 1: User authentication (recommended for development)
   gcloud auth login
   gcloud config set project YOUR_PROJECT_ID
   
   # Method 2: Service account (recommended for production)
   export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
   
   # Method 3: Application Default Credentials
   gcloud auth application-default login
   ```

3. **Required Permissions** - Your account/service account needs:
   - `roles/compute.viewer` - For listing resources
   - `roles/container.developer` - For GKE operations
   - `roles/storage.admin` - For Cloud Storage operations
   - `roles/monitoring.editor` - For Cloud Monitoring
   - `roles/iam.serviceAccountAdmin` - For service account operations

## Running the Example

1. **Clone and navigate to the project:**
   ```bash
   git clone <repository>
   cd apm/examples/gcp-comprehensive
   ```

2. **Run the example:**
   ```bash
   go run main.go
   ```

3. **Expected output:**
   ```
   === Validating gcloud CLI ===
   === Detecting CLI ===
   CLI Status: Installed=true, Version=400.0.0, Path=gcloud
   
   === Validating Authentication ===
   === Getting Credentials ===
   Provider: gcp, Method: cli, Account: user@example.com
   
   === Getting Advanced Operations ===
   === Resource Manager Operations ===
   Found 3 projects:
     - My Project (my-project-123)
     - Test Project (test-project-456)
     - Demo Project (demo-project-789)
   Current project: my-project-123
   
   === Service Account Operations ===
   Found 5 service accounts:
     - Compute Engine default service account (123-compute@developer.gserviceaccount.com)
     - App Engine default service account (my-project-123@appspot.gserviceaccount.com)
     - APM Service Account (apm-service@my-project-123.iam.gserviceaccount.com)
   
   === Authentication Manager Operations ===
   Active accounts: [user@example.com]
   Access token: ya29.a0ARrdaM1234...
   
   === Registry Operations ===
   Found 6 registries:
     - gcr.io (GCR) in global
     - us.gcr.io (GCR) in us
     - eu.gcr.io (GCR) in eu
     - asia.gcr.io (GCR) in asia
     - my-repo (Artifact Registry) in us-central1
     - docker-repo (Artifact Registry) in europe-west1
   
   === Cluster Operations ===
   Found 2 GKE clusters:
     - production-cluster (1.24.8-gke.2000) in us-central1-a, Status: RUNNING, Nodes: 3
     - staging-cluster (1.24.8-gke.2000) in us-west1-b, Status: RUNNING, Nodes: 1
   
   === Region Operations ===
   Found 35 regions (showing first 5):
     - asia-east1
     - asia-east2
     - asia-northeast1
     - asia-northeast2
     - asia-northeast3
     ... and 30 more
   Current region: us-central1
   
   === Storage Operations ===
   Found 4 storage buckets:
     - my-project-logs in us-central1 (STANDARD)
     - my-project-backups in us-central1 (NEARLINE)
     - my-project-artifacts in us-central1 (STANDARD)
   
   === Monitoring Operations ===
   Ensuring Cloud Monitoring API is enabled...
   Cloud Monitoring API is enabled
   Ensuring Cloud Trace API is enabled...
   Cloud Trace API is enabled
   
   === APM Integration Setup ===
   APM Integration configuration for project my-project-123:
     - Region: us-central1
     - Service Account ID: apm-service-account
     - Create Service Account: false
     - Setup Monitoring: true
     - Create Storage Bucket: false
   
   === GCP Integration Demo Completed ===
   ```

## Code Structure

### Main Functions

- **CLI Validation**: Validates gcloud CLI installation and version
- **Authentication**: Demonstrates different authentication methods
- **Resource Discovery**: Lists projects, regions, clusters, registries
- **Service Operations**: Shows service account and storage management
- **APM Integration**: Demonstrates complete APM setup

### Helper Functions

The example includes several helper functions showing specific use cases:

1. **`authenticateWithServiceAccount`** - Service account authentication
2. **`setupWorkloadIdentity`** - GKE Workload Identity configuration
3. **`createAPMStorageBucket`** - Storage bucket creation for APM data
4. **`enableAPMAPIs`** - Enable all required APIs
5. **`fullAPMSetup`** - Complete APM infrastructure setup

## Key Integration Points

### 1. Container Registry Authentication

```go
// Authenticate Docker with both GCR and Artifact Registry
registries, _ := provider.ListRegistries(ctx)
for _, registry := range registries {
    err := provider.AuthenticateRegistry(ctx, registry)
    if err == nil {
        fmt.Printf("✓ Authenticated with %s\n", registry.URL)
    }
}
```

### 2. GKE Cluster Access

```go
// Get kubeconfig for all clusters
clusters, _ := provider.ListClusters(ctx)
for _, cluster := range clusters {
    kubeconfig, err := provider.GetKubeconfig(ctx, cluster)
    if err == nil {
        // Save kubeconfig or configure kubectl
        fmt.Printf("✓ Got kubeconfig for %s\n", cluster.Name)
    }
}
```

### 3. Service Account Management

```go
// Create service account for APM
serviceAccountManager := provider.GetAdvancedOperations().GetServiceAccountManager()
sa, err := serviceAccountManager.CreateServiceAccount(ctx, 
    "apm-monitoring", "APM Service Account", "For monitoring and logging")
```

### 4. Complete APM Setup

```go
// One-command APM setup
advOps := provider.GetAdvancedOperations()
config := cloud.APMIntegrationConfig{
    ProjectID:            "my-project",
    Region:               "us-central1",
    ServiceAccountID:     "apm-monitoring",
    CreateServiceAccount: true,
    SetupMonitoring:      true,
    CreateStorageBucket:  true,
}
err := advOps.SetupAPMIntegration(ctx, config)
```

## Advanced Usage

### Workload Identity Setup

For GKE applications, you can set up Workload Identity:

```go
authManager := provider.GetAdvancedOperations().GetAuthenticationManager()
err := authManager.SetupWorkloadIdentity(ctx,
    "my-project",           // GCP Project ID
    "my-cluster",           // GKE Cluster name
    "us-central1-a",        // Cluster location
    "default",              // Kubernetes namespace
    "my-app-sa",           // Kubernetes service account
    "my-gcp-sa@my-project.iam.gserviceaccount.com") // GCP service account
```

### Multiple Authentication Methods

```go
authManager := provider.GetAdvancedOperations().GetAuthenticationManager()

// Switch between different authentication methods
err := authManager.AuthenticateWithServiceAccount(ctx, "/path/to/key.json")
err = authManager.AuthenticateWithOAuth2(ctx)
err = authManager.AuthenticateWithApplicationDefaultCredentials(ctx)
```

## Troubleshooting

### Common Issues

1. **gcloud not found**: Install Google Cloud CLI
2. **Authentication failed**: Run `gcloud auth login`
3. **Project not set**: Run `gcloud config set project PROJECT_ID`
4. **API not enabled**: The example automatically enables required APIs
5. **Permission denied**: Ensure your account has required IAM roles

### Debug Mode

Set environment variable for verbose output:
```bash
export CLOUDSDK_CORE_VERBOSITY=debug
go run main.go
```

## Next Steps

After running this example:

1. **Deploy APM Stack**: Use the discovered clusters to deploy monitoring components
2. **Configure Monitoring**: Set up Prometheus, Grafana, and other APM tools
3. **Set Up Alerting**: Configure Cloud Monitoring alerts
4. **Implement Logging**: Use Cloud Logging for centralized log management
5. **Enable Tracing**: Configure Cloud Trace for distributed tracing

## Related Documentation

- [GCP Integration Guide](../../docs/gcp-integration-guide.md) - Complete integration documentation
- [Cloud Provider Types](../../pkg/cloud/types.go) - Interface definitions
- [GCP Provider Implementation](../../pkg/cloud/gcp.go) - Full implementation
- [Google Cloud Documentation](https://cloud.google.com/docs) - Official GCP documentation

## Support

For issues or questions:
1. Check the [troubleshooting guide](../../docs/gcp-integration-guide.md#troubleshooting)
2. Review the [error handling section](../../docs/gcp-integration-guide.md#error-handling)
3. Create an issue in the project repository