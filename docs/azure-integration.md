# Azure CLI Integration Guide

This guide provides comprehensive documentation for the Azure CLI integration features in the APM tool.

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Authentication Methods](#authentication-methods)
4. [Core Features](#core-features)
5. [Usage Examples](#usage-examples)
6. [Security Best Practices](#security-best-practices)
7. [Troubleshooting](#troubleshooting)
8. [API Reference](#api-reference)

## Overview

The APM tool provides comprehensive Azure CLI integration that enables:

- Multiple authentication methods (Interactive, Device Code, Service Principal, Managed Identity)
- Subscription and resource group management
- Container registry (ACR) operations
- Kubernetes cluster (AKS) management
- Azure Monitor and Application Insights integration
- Storage account operations
- Key Vault secret management
- ARM template validation and deployment
- Secure credential management with encryption

## Prerequisites

### Required Software

1. **Azure CLI**: Version 2.30.0 or later
   ```bash
   # Install Azure CLI
   curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
   
   # Verify installation
   az --version
   ```

2. **Go**: Version 1.19 or later (for building from source)

3. **APM Tool**: Latest version with Azure integration

### Azure Account Requirements

- Active Azure subscription
- Appropriate permissions for the operations you plan to perform
- For service principal authentication: permission to create/manage service principals

## Authentication Methods

### 1. Interactive Browser Authentication

Best for development environments where browser access is available.

```go
err := azureProvider.AuthenticateInteractive(ctx)
```

**CLI Equivalent:**
```bash
az login
```

### 2. Device Code Authentication

Ideal for headless environments or when browser access is restricted.

```go
deviceAuth, err := azureProvider.AuthenticateDeviceCode(ctx)
if err == nil {
    fmt.Printf("Go to %s and enter code: %s\n", 
        deviceAuth.VerificationURL, deviceAuth.UserCode)
}
```

**CLI Equivalent:**
```bash
az login --use-device-code
```

### 3. Service Principal Authentication

Recommended for production environments and CI/CD pipelines.

```go
err := azureProvider.AuthenticateServicePrincipal(ctx, 
    "client-id", "client-secret", "tenant-id")
```

**CLI Equivalent:**
```bash
az login --service-principal \
    --username <client-id> \
    --password <client-secret> \
    --tenant <tenant-id>
```

### 4. Managed Identity Authentication

Perfect for applications running on Azure infrastructure.

```go
err := azureProvider.AuthenticateManagedIdentity(ctx)
```

**Requirements:**
- Application must be running on Azure (VM, App Service, etc.)
- Managed identity must be enabled and configured

## Core Features

### Subscription Management

```go
// List all subscriptions
subscriptions, err := azureProvider.ListSubscriptions(ctx)

// Get specific subscription
subscription, err := azureProvider.GetSubscription(ctx, "subscription-id")

// Set default subscription
err := azureProvider.SetDefaultSubscription(ctx, "subscription-id")
```

### Resource Group Management

```go
// List resource groups
resourceGroups, err := azureProvider.ListResourceGroups(ctx)

// Create resource group
tags := map[string]string{
    "Environment": "Production",
    "Project":     "APM",
}
rg, err := azureProvider.CreateResourceGroup(ctx, "my-rg", "eastus", tags)

// Delete resource group
err := azureProvider.DeleteResourceGroup(ctx, "my-rg")
```

### Container Registry (ACR) Operations

```go
// List ACR registries
registries, err := azureProvider.ListRegistries(ctx)

// Get specific registry
registry, err := azureProvider.GetRegistry(ctx, "myregistry")

// Authenticate to registry (for Docker operations)
err := azureProvider.AuthenticateRegistry(ctx, registry)
```

### Kubernetes (AKS) Operations

```go
// List AKS clusters
clusters, err := azureProvider.ListClusters(ctx)

// Get specific cluster
cluster, err := azureProvider.GetCluster(ctx, "my-aks-cluster")

// Get kubeconfig
kubeconfig, err := azureProvider.GetKubeconfig(ctx, cluster)
```

### Service Principal Management

```go
// Create service principal
sp, err := azureProvider.CreateServicePrincipal(ctx, "my-app-sp")

// List service principals
servicePrincipals, err := azureProvider.ListServicePrincipals(ctx)

// Rotate service principal secret
rotatedSP, err := azureProvider.RotateServicePrincipalSecret(ctx, sp.AppID)

// Delete service principal
err := azureProvider.DeleteServicePrincipal(ctx, sp.AppID)
```

### Azure Monitor Integration

```go
// Get metrics from Azure Monitor
metrics, err := azureProvider.GetMonitorMetrics(ctx, 
    "/subscriptions/sub-id/resourceGroups/rg/providers/Microsoft.Web/sites/myapp",
    []string{"CpuPercentage", "MemoryPercentage"},
    "PT1H") // Last hour

// Create alert rule
alertConfig := map[string]interface{}{
    "condition": map[string]interface{}{
        "allOf": []map[string]interface{}{
            {
                "metricName": "CpuPercentage",
                "operator":   "GreaterThan",
                "threshold":  80,
            },
        },
    },
}
err := azureProvider.CreateAlertRule(ctx, "high-cpu-alert", "my-rg", alertConfig)

// List action groups
actionGroups, err := azureProvider.ListActionGroups(ctx, "my-rg")
```

### Application Insights Integration

```go
// Create Application Insights
appInsights, err := azureProvider.CreateApplicationInsights(ctx, 
    "my-app-insights", "my-rg", "eastus")

// List Application Insights resources
appInsightsList, err := azureProvider.ListApplicationInsights(ctx)

// Get specific Application Insights
appInsights, err := azureProvider.GetApplicationInsights(ctx, 
    "my-app-insights", "my-rg")
```

### Storage Account Operations

```go
// List storage accounts
storageAccounts, err := azureProvider.ListStorageAccounts(ctx)

// Create storage account
storageAccount, err := azureProvider.CreateStorageAccount(ctx, 
    "mystorageaccount", "my-rg", "eastus")

// Get storage account keys
keys, err := azureProvider.GetStorageAccountKeys(ctx, 
    "mystorageaccount", "my-rg")
```

### Key Vault Integration

```go
// List key vaults
keyVaults, err := azureProvider.ListKeyVaults(ctx)

// Get secret from Key Vault
secret, err := azureProvider.GetSecret(ctx, "my-keyvault", "my-secret")

// Set secret in Key Vault
err := azureProvider.SetSecret(ctx, "my-keyvault", "my-secret", "secret-value")

// Delete secret
err := azureProvider.DeleteSecret(ctx, "my-keyvault", "my-secret")
```

### ARM Template Operations

```go
// Create ARM template
template := &cloud.AzureARMTemplate{
    Name:          "my-template",
    ResourceGroup: "my-rg",
    Template: map[string]interface{}{
        "$schema":        "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
        "contentVersion": "1.0.0.0",
        "resources": []interface{}{
            // ARM template resources
        },
    },
    Parameters: map[string]interface{}{
        "location": map[string]interface{}{
            "type":         "string",
            "defaultValue": "eastus",
        },
    },
    Mode: "Incremental",
}

// Validate template
validationResult, err := azureProvider.ValidateARMTemplate(ctx, template)

// Deploy template
deploymentName, err := azureProvider.DeployARMTemplate(ctx, template)

// Check deployment status
status, err := azureProvider.GetDeploymentStatus(ctx, "my-rg", deploymentName)
```

## Secure Credential Management

The APM tool includes a secure credential management system for Azure:

```go
// Create credential manager
credManager, err := cloud.NewAzureCredentialManager()

// Store credentials securely (encrypted)
creds := &cloud.Credentials{
    Provider:   cloud.ProviderAzure,
    AuthMethod: cloud.AuthMethodServicePrincipal,
    Profile:    "production",
    AccessKey:  "client-id",
    SecretKey:  "client-secret",
    Properties: map[string]string{
        "tenant_id": "tenant-id",
    },
}
err := credManager.Store(creds)

// Retrieve credentials
creds, err := credManager.Retrieve(cloud.ProviderAzure, "production")

// List all stored credentials
allCreds, err := credManager.List(cloud.ProviderAzure)

// Delete credentials
err := credManager.Delete(cloud.ProviderAzure, "production")

// Validate credentials
err := credManager.ValidateCredentials(ctx, creds)

// Refresh token (for applicable auth methods)
newCreds, err := credManager.RefreshToken(ctx, creds)
```

### Credential Storage Security

- Credentials are encrypted using AES-256-GCM encryption
- Encryption keys are derived using PBKDF2 with machine-specific data
- Stored in user's home directory with restricted permissions (0600)
- Sensitive data is masked when listing credentials

## Usage Examples

### Complete APM Setup for Azure

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/ybke/apm/pkg/cloud"
)

func main() {
    ctx := context.Background()

    // 1. Create Azure provider
    config := &cloud.ProviderConfig{
        Provider:      cloud.ProviderAzure,
        DefaultRegion: "eastus",
        EnableCache:   true,
        CacheDuration: 5 * time.Minute,
    }

    azureProvider, err := cloud.NewAzureProvider(config)
    if err != nil {
        log.Fatalf("Failed to create Azure provider: %v", err)
    }

    // 2. Validate authentication
    if err := azureProvider.ValidateAuth(ctx); err != nil {
        log.Fatalf("Authentication failed: %v", err)
    }

    // 3. Set up resources for APM
    
    // Create resource group
    rg, err := azureProvider.CreateResourceGroup(ctx, "apm-monitoring", "eastus", map[string]string{
        "Purpose": "APM-Monitoring",
    })
    if err != nil {
        log.Fatalf("Failed to create resource group: %v", err)
    }

    // Create Application Insights
    appInsights, err := azureProvider.CreateApplicationInsights(ctx, 
        "apm-app-insights", rg.Name, rg.Location)
    if err != nil {
        log.Fatalf("Failed to create Application Insights: %v", err)
    }

    // Create storage account for logs
    storage, err := azureProvider.CreateStorageAccount(ctx, 
        "apmstorageaccount", rg.Name, rg.Location)
    if err != nil {
        log.Fatalf("Failed to create storage account: %v", err)
    }

    log.Printf("APM infrastructure created successfully!")
    log.Printf("Resource Group: %s", rg.Name)
    log.Printf("Application Insights: %s", appInsights.Name)
    log.Printf("Storage Account: %s", storage.Name)
}
```

### CI/CD Pipeline Integration

```go
// In your CI/CD pipeline
func setupAzureAuth() error {
    ctx := context.Background()
    
    // Use service principal for CI/CD
    azureProvider, err := cloud.NewAzureProvider(nil)
    if err != nil {
        return err
    }

    // Authenticate using environment variables
    clientID := os.Getenv("AZURE_CLIENT_ID")
    clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
    tenantID := os.Getenv("AZURE_TENANT_ID")

    err = azureProvider.AuthenticateServicePrincipal(ctx, clientID, clientSecret, tenantID)
    if err != nil {
        return fmt.Errorf("CI/CD authentication failed: %w", err)
    }

    // Validate authentication
    return azureProvider.ValidateAuth(ctx)
}
```

### Multi-Environment Configuration

```go
// Production environment
prodCreds := &cloud.Credentials{
    Provider:   cloud.ProviderAzure,
    AuthMethod: cloud.AuthMethodServicePrincipal,
    Profile:    "production",
    Region:     "eastus",
    // ... other fields
}

// Development environment  
devCreds := &cloud.Credentials{
    Provider:   cloud.ProviderAzure,
    AuthMethod: cloud.AuthMethodCLI,
    Profile:    "development", 
    Region:     "eastus2",
    // ... other fields
}

// Store both configurations
credManager.Store(prodCreds)
credManager.Store(devCreds)

// Use appropriate credentials based on environment
env := os.Getenv("ENVIRONMENT")
creds, err := credManager.Retrieve(cloud.ProviderAzure, env)
```

## Security Best Practices

### 1. Authentication

- **Production**: Always use Service Principal or Managed Identity
- **Development**: Use Interactive or Device Code authentication
- **CI/CD**: Use Service Principal with minimal required permissions

### 2. Credential Management

- Store credentials securely using the built-in credential manager
- Rotate service principal secrets regularly
- Use different service principals for different environments
- Never store credentials in source code

### 3. Permissions

Follow the principle of least privilege:

```bash
# Create custom role with minimal permissions
az role definition create --role-definition '{
  "Name": "APM Monitoring Operator",
  "Description": "Custom role for APM monitoring operations",
  "Actions": [
    "Microsoft.Resources/subscriptions/resourceGroups/read",
    "Microsoft.ContainerRegistry/registries/read",
    "Microsoft.ContainerService/managedClusters/read",
    "Microsoft.Storage/storageAccounts/read",
    "Microsoft.KeyVault/vaults/secrets/read",
    "Microsoft.Insights/components/read",
    "Microsoft.Insights/metrics/read"
  ],
  "AssignableScopes": ["/subscriptions/your-subscription-id"]
}'
```

### 4. Network Security

- Use private endpoints where possible
- Implement network security groups (NSGs)
- Enable Azure Firewall for additional protection

### 5. Monitoring and Auditing

- Enable Azure Activity Log
- Set up security alerts for service principal activities
- Monitor credential usage patterns

## Troubleshooting

### Common Issues

#### 1. Azure CLI Not Found

**Error:** `Azure CLI not installed`

**Solution:**
```bash
# Install Azure CLI
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
```

#### 2. Authentication Failed

**Error:** `Azure CLI not authenticated: run 'az login'`

**Solution:**
```bash
# Interactive login
az login

# Or device code login
az login --use-device-code

# Or service principal login
az login --service-principal \
    --username $AZURE_CLIENT_ID \
    --password $AZURE_CLIENT_SECRET \
    --tenant $AZURE_TENANT_ID
```

#### 3. Insufficient Permissions

**Error:** `Authorization failed` or `Forbidden`

**Solution:**
- Check Azure RBAC permissions
- Ensure service principal has required roles
- Verify subscription and resource group access

#### 4. Resource Not Found

**Error:** `Resource group 'xyz' could not be found`

**Solution:**
- Verify resource group name and subscription
- Check if resource group exists in the current subscription
- Ensure proper subscription context

#### 5. Network Connectivity

**Error:** `Failed to connect to Azure services`

**Solution:**
- Check internet connectivity
- Verify firewall and proxy settings
- Ensure Azure endpoints are accessible

### Debugging

Enable debug logging:

```go
// Enable debug logging
azureProvider.SetLogLevel("debug")

// Or set environment variable
os.Setenv("AZURE_DEBUG", "true")
```

### Support Commands

```bash
# Check Azure CLI version
az --version

# Check current authentication
az account show

# List available subscriptions
az account list

# Test connectivity
az resource list --output table
```

## API Reference

### AzureProvider Interface

The main interface for Azure operations:

```go
type AzureProvider interface {
    CloudProvider
    
    // Authentication
    AuthenticateInteractive(ctx context.Context) error
    AuthenticateDeviceCode(ctx context.Context) (*DeviceCodeAuth, error)
    AuthenticateServicePrincipal(ctx context.Context, clientID, clientSecret, tenantID string) error
    AuthenticateManagedIdentity(ctx context.Context) error
    
    // Subscription management
    ListSubscriptions(ctx context.Context) ([]*AzureSubscription, error)
    GetSubscription(ctx context.Context, subscriptionID string) (*AzureSubscription, error)
    SetDefaultSubscription(ctx context.Context, subscriptionID string) error
    
    // Resource group management
    ListResourceGroups(ctx context.Context) ([]*AzureResourceGroup, error)
    CreateResourceGroup(ctx context.Context, name, location string, tags map[string]string) (*AzureResourceGroup, error)
    DeleteResourceGroup(ctx context.Context, name string) error
    
    // Service principal management
    CreateServicePrincipal(ctx context.Context, name string) (*AzureServicePrincipal, error)
    ListServicePrincipals(ctx context.Context) ([]*AzureServicePrincipal, error)
    DeleteServicePrincipal(ctx context.Context, appID string) error
    RotateServicePrincipalSecret(ctx context.Context, appID string) (*AzureServicePrincipal, error)
    
    // Azure Monitor
    GetMonitorMetrics(ctx context.Context, resourceID string, metricNames []string, timespan string) ([]*AzureMonitorMetric, error)
    CreateAlertRule(ctx context.Context, name, resourceGroup string, config map[string]interface{}) error
    ListActionGroups(ctx context.Context, resourceGroup string) ([]map[string]interface{}, error)
    
    // Application Insights
    CreateApplicationInsights(ctx context.Context, name, resourceGroup, location string) (*AzureApplicationInsights, error)
    ListApplicationInsights(ctx context.Context) ([]*AzureApplicationInsights, error)
    GetApplicationInsights(ctx context.Context, name, resourceGroup string) (*AzureApplicationInsights, error)
    
    // Storage Account
    ListStorageAccounts(ctx context.Context) ([]*AzureStorageAccount, error)
    CreateStorageAccount(ctx context.Context, name, resourceGroup, location string) (*AzureStorageAccount, error)
    GetStorageAccountKeys(ctx context.Context, name, resourceGroup string) ([]string, error)
    
    // Key Vault
    ListKeyVaults(ctx context.Context) ([]string, error)
    GetSecret(ctx context.Context, vaultName, secretName string) (*AzureKeyVaultSecret, error)
    SetSecret(ctx context.Context, vaultName, secretName, value string) error
    DeleteSecret(ctx context.Context, vaultName, secretName string) error
    
    // ARM Templates
    ValidateARMTemplate(ctx context.Context, template *AzureARMTemplate) (*ValidationResult, error)
    DeployARMTemplate(ctx context.Context, template *AzureARMTemplate) (string, error)
    GetDeploymentStatus(ctx context.Context, resourceGroup, deploymentName string) (string, error)
}
```

### AzureCredentialManager Interface

For secure credential management:

```go
type AzureCredentialManager interface {
    CredentialManager
    StoreServicePrincipal(sp *AzureServicePrincipal) error
    RetrieveServicePrincipal(appID string) (*AzureServicePrincipal, error)
    ListServicePrincipals() ([]*AzureServicePrincipal, error)
    DeleteServicePrincipal(appID string) error
    ValidateCredentials(ctx context.Context, creds *Credentials) error
    RefreshToken(ctx context.Context, creds *Credentials) (*Credentials, error)
}
```

### Data Types

Key data structures used in the Azure integration:

```go
type AzureSubscription struct {
    ID           string            `json:"id"`
    Name         string            `json:"name"`
    State        string            `json:"state"`
    TenantID     string            `json:"tenant_id"`
    IsDefault    bool              `json:"is_default"`
    CloudName    string            `json:"cloud_name"`
    HomeTenantID string            `json:"home_tenant_id"`
    Tags         map[string]string `json:"tags,omitempty"`
}

type AzureResourceGroup struct {
    ID                string            `json:"id"`
    Name              string            `json:"name"`
    Location          string            `json:"location"`
    SubscriptionID    string            `json:"subscription_id"`
    Tags              map[string]string `json:"tags,omitempty"`
    ProvisioningState string            `json:"provisioning_state"`
}

type AzureServicePrincipal struct {
    AppID       string     `json:"app_id"`
    DisplayName string     `json:"display_name"`
    Password    string     `json:"password,omitempty"`
    Tenant      string     `json:"tenant"`
    CreatedAt   time.Time  `json:"created_at"`
    ExpiresAt   *time.Time `json:"expires_at,omitempty"`
    KeyID       string     `json:"key_id,omitempty"`
    Certificate string     `json:"certificate,omitempty"`
}

// ... additional types
```

## Next Steps

1. **Try the Example**: Run the example application to see all features in action
2. **Integration**: Integrate Azure operations into your APM workflows
3. **Customization**: Extend the provider for your specific use cases
4. **Monitoring**: Set up comprehensive monitoring for your Azure resources

## Resources

- [Azure CLI Documentation](https://docs.microsoft.com/en-us/cli/azure/)
- [Azure Resource Manager Templates](https://docs.microsoft.com/en-us/azure/azure-resource-manager/templates/)
- [Azure Monitor Documentation](https://docs.microsoft.com/en-us/azure/azure-monitor/)
- [Application Insights Documentation](https://docs.microsoft.com/en-us/azure/azure-monitor/app/app-insights-overview)
- [Azure Service Principal Documentation](https://docs.microsoft.com/en-us/azure/active-directory/develop/app-objects-and-service-principals)

---

For more information or support, please visit the [APM project repository](https://github.com/ybke/apm) or create an issue.