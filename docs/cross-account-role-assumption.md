# Cross-Account Role Assumption Documentation

## Overview

The APM tool's cross-account role assumption functionality provides enterprise-grade support for managing AWS credentials across multiple accounts. This feature enables secure, automated access to resources in different AWS accounts while maintaining security best practices and providing comprehensive session management.

## Features

### 🔐 Enhanced Security
- **MFA-based role assumption** for sensitive operations
- **External ID validation** for partner integrations
- **Condition-based access control** with trust policy analysis
- **Secure credential storage** with encryption at rest
- **Audit logging** for all cross-account operations

### ⚡ Performance & Reliability
- **Automatic session refresh** before expiry
- **Intelligent credential caching** with TTL management
- **Background refresh workers** for seamless operation
- **Comprehensive error handling** with retry mechanisms
- **Connection pooling** for optimal performance

### 🎯 Advanced Capabilities
- **Role chaining** for complex multi-account scenarios
- **Multi-region support** with region switching
- **Session duration management** with configurable limits
- **Configuration persistence** to file or S3
- **Multi-account configuration management**

## Quick Start

### Basic Cross-Account Role Assumption

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/ybke/apm/pkg/cloud"
)

func main() {
    // Create AWS provider
    config := &cloud.ProviderConfig{
        Provider:      cloud.ProviderAWS,
        DefaultRegion: "us-east-1",
        EnableCache:   true,
        CacheDuration: 30 * time.Minute,
    }
    
    provider, err := cloud.NewAWSProvider(config)
    if err != nil {
        log.Fatalf("Failed to create AWS provider: %v", err)
    }
    
    ctx := context.Background()
    
    // Assume role in different account
    credentials, err := provider.AssumeRoleAcrossAccount(
        ctx,
        "123456789012", // source account
        "987654321098", // target account
        "APMCrossAccountRole", // role name
        &cloud.AssumeRoleOptions{
            SessionName:     "apm-cross-account",
            DurationSeconds: 3600,
            Region:          "us-east-1",
        },
    )
    
    if err != nil {
        log.Fatalf("Failed to assume role: %v", err)
    }
    
    log.Printf("Successfully assumed role: %s", credentials.Properties["assumed_role_arn"])
}
```

## API Reference

### Core Types

#### AssumeRoleOptions

```go
type AssumeRoleOptions struct {
    SessionName            string            `json:"sessionName"`
    DurationSeconds        int               `json:"durationSeconds"`
    ExternalID             string            `json:"externalId,omitempty"`
    MFASerialNumber        string            `json:"mfaSerialNumber,omitempty"`
    MFATokenCode           string            `json:"mfaTokenCode,omitempty"`
    PolicyArns             []string          `json:"policyArns,omitempty"`
    Policy                 string            `json:"policy,omitempty"`
    SourceIdentity         string            `json:"sourceIdentity,omitempty"`
    TransitiveTagKeys      []string          `json:"transitiveTagKeys,omitempty"`
    Tags                   map[string]string `json:"tags,omitempty"`
    Region                 string            `json:"region,omitempty"`
    EnableCredentialCache  bool              `json:"enableCredentialCache"`
    CredentialCacheTTL     time.Duration     `json:"credentialCacheTtl"`
    EnableAutoRefresh      bool              `json:"enableAutoRefresh"`
    AutoRefreshThreshold   time.Duration     `json:"autoRefreshThreshold"`
}
```

#### CrossAccountSession

```go
type CrossAccountSession struct {
    Credentials      *Credentials      `json:"credentials"`
    RoleArn          string            `json:"roleArn"`
    SourceArn        string            `json:"sourceArn"`
    SessionName      string            `json:"sessionName"`
    CreatedAt        time.Time         `json:"createdAt"`
    ExpiresAt        time.Time         `json:"expiresAt"`
    RefreshThreshold time.Duration     `json:"refreshThreshold"`
    Options          *AssumeRoleOptions `json:"options"`
}
```

### Core Methods

#### Basic Role Assumption

```go
// AssumeRoleWithOptions assumes a role with comprehensive options
func (p *AWSProvider) AssumeRoleWithOptions(ctx context.Context, roleArn string, options *AssumeRoleOptions) (*Credentials, error)

// AssumeRoleAcrossAccount assumes a role in a different AWS account
func (p *AWSProvider) AssumeRoleAcrossAccount(ctx context.Context, sourceAccount, targetAccount, roleName string, options *AssumeRoleOptions) (*Credentials, error)
```

#### Enhanced Security

```go
// AssumeRoleWithMFA assumes a role using MFA
func (p *AWSProvider) AssumeRoleWithMFA(ctx context.Context, roleArn, mfaDeviceArn, mfaToken string, options *AssumeRoleOptions) (*Credentials, error)

// AssumeRoleWithExternalID assumes a role with an external ID for partner access
func (p *AWSProvider) AssumeRoleWithExternalID(ctx context.Context, roleArn, externalID string, options *AssumeRoleOptions) (*Credentials, error)
```

#### Advanced Scenarios

```go
// AssumeRoleChain assumes a chain of roles for complex multi-account scenarios
func (p *AWSProvider) AssumeRoleChain(ctx context.Context, roleChain []*RoleChainStep) (*Credentials, error)

// SwitchRole switches to a different role, optionally in a different region
func (p *AWSProvider) SwitchRole(ctx context.Context, targetRoleArn, sessionName string, options *AssumeRoleOptions) (*Credentials, error)
```

#### Session Management

```go
// RefreshCredentials refreshes the given credentials if they are near expiry
func (p *AWSProvider) RefreshCredentials(ctx context.Context, credentials *Credentials) (*Credentials, error)

// ValidateRoleAssumption validates that a role can be assumed
func (p *AWSProvider) ValidateRoleAssumption(ctx context.Context, roleArn string, options *AssumeRoleOptions) (*RoleValidation, error)
```

## Usage Examples

### 1. MFA-Based Role Assumption

```go
// Assume role with MFA for enhanced security
mfaCredentials, err := provider.AssumeRoleWithMFA(
    ctx,
    "arn:aws:iam::987654321098:role/SecureRole",
    "arn:aws:iam::123456789012:mfa/user@example.com",
    "123456", // MFA token from device
    &cloud.AssumeRoleOptions{
        SessionName:     "apm-secure-session",
        DurationSeconds: 1800, // 30 minutes for security
    },
)
```

### 2. Role Chaining for Multi-Account Access

```go
// Complex role chain across multiple accounts
roleChain := []*cloud.RoleChainStep{
    {
        RoleArn:     "arn:aws:iam::111111111111:role/OrganizationRole",
        SessionName: "apm-org-step",
    },
    {
        RoleArn:     "arn:aws:iam::222222222222:role/DevelopmentRole",
        SessionName: "apm-dev-step",
    },
    {
        RoleArn:     "arn:aws:iam::333333333333:role/ProductionRole",
        SessionName: "apm-prod-step",
        ExternalID:  "unique-external-id-123",
    },
}

finalCredentials, err := provider.AssumeRoleChain(ctx, roleChain)
```

### 3. Automatic Session Management

```go
// Get cross-account role manager
roleManager := provider.GetCrossAccountRoleManager()
defer roleManager.Close() // Important: Clean up background workers

// Create session with auto-refresh
session, err := roleManager.GetSession(ctx, roleArn, &cloud.AssumeRoleOptions{
    SessionName:          "apm-auto-refresh",
    DurationSeconds:      3600,
    EnableAutoRefresh:    true,
    AutoRefreshThreshold: 5 * time.Minute, // Refresh when 5 minutes left
})

// Use session credentials
credentials := session.Credentials
```

### 4. Multi-Account Configuration

```go
// Get account manager
accountManager := provider.GetAccountManager()

// Add development account
devAccount := &cloud.AccountConfig{
    AccountID:       "111111111111",
    AccountName:     "APM Development",
    Environment:     "dev",
    DefaultRegion:   "us-west-2",
    SessionDuration: 3600,
    Roles: []*cloud.RoleConfig{
        {
            RoleName:        "APMDeveloperRole",
            Description:     "Role for APM development access",
            SessionDuration: 3600,
        },
    },
}

err := accountManager.AddAccount(devAccount)

// Save configuration
err = accountManager.SaveConfig(ctx, "/path/to/config.json")
// Or save to S3
err = accountManager.SaveConfig(ctx, "s3://bucket/config.json")
```

### 5. Role Validation

```go
// Validate role before assumption
validation, err := provider.ValidateRoleAssumption(ctx, roleArn, options)
if err != nil {
    log.Fatalf("Validation failed: %v", err)
}

if !validation.CanAssume {
    log.Fatalf("Cannot assume role: %s", validation.Error)
}

if validation.MFARequired {
    log.Println("MFA is required for this role")
}

if validation.ExternalIDRequired {
    log.Println("External ID is required for this role")
}
```

## Configuration Examples

### Multi-Account Configuration File

```json
{
  "organization": "APM Organization",
  "masterAccount": "123456789012",
  "defaultRegion": "us-east-1",
  "accounts": [
    {
      "accountId": "111111111111",
      "accountName": "APM Development",
      "environment": "dev",
      "defaultRegion": "us-west-2",
      "sessionDuration": 3600,
      "roles": [
        {
          "roleName": "APMDeveloperRole",
          "roleArn": "arn:aws:iam::111111111111:role/APMDeveloperRole",
          "description": "Role for APM development access",
          "sessionDuration": 3600
        }
      ]
    },
    {
      "accountId": "333333333333",
      "accountName": "APM Production",
      "environment": "prod",
      "defaultRegion": "us-east-1",
      "mfaRequired": true,
      "sessionDuration": 1800,
      "externalId": "prod-external-id-789",
      "roles": [
        {
          "roleName": "APMProductionRole",
          "roleArn": "arn:aws:iam::333333333333:role/APMProductionRole",
          "description": "Production access for APM operations",
          "mfaRequired": true,
          "sessionDuration": 1800,
          "externalId": "prod-role-external-id"
        }
      ]
    }
  ],
  "globalTags": {
    "Application": "APM",
    "Team": "Platform"
  },
  "createdAt": "2025-07-10T10:00:00Z",
  "updatedAt": "2025-07-10T10:00:00Z"
}
```

### Environment-Specific Configurations

```go
// Load environment-specific accounts
devAccounts := accountManager.GetAccountsByEnvironment("dev")
prodAccounts := accountManager.GetAccountsByEnvironment("prod")

// Use different session durations for different environments
for _, account := range prodAccounts {
    // Production accounts use shorter sessions
    account.SessionDuration = 1800 // 30 minutes
}
```

## Security Best Practices

### 1. MFA Enforcement

- Always use MFA for production environments
- Implement time-based MFA tokens
- Validate MFA device ARNs before assumption

```go
options := &cloud.AssumeRoleOptions{
    MFASerialNumber: "arn:aws:iam::account:mfa/user",
    MFATokenCode:    getMFAToken(), // From secure source
    DurationSeconds: 1800, // Shorter duration for MFA
}
```

### 2. External ID Validation

- Use unique external IDs for each partner integration
- Regularly rotate external IDs
- Validate external ID requirements during role validation

```go
options := &cloud.AssumeRoleOptions{
    ExternalID: generateUniqueExternalID(), // Unique per integration
}
```

### 3. Session Duration Management

- Use shorter session durations for sensitive environments
- Implement automatic refresh for long-running operations
- Monitor session usage and expiration

```go
// Production: shorter sessions
prodOptions := &cloud.AssumeRoleOptions{
    DurationSeconds: 1800, // 30 minutes
}

// Development: longer sessions for convenience
devOptions := &cloud.AssumeRoleOptions{
    DurationSeconds: 3600, // 1 hour
}
```

### 4. Credential Security

- Never log credentials or tokens
- Use secure storage for cached credentials
- Implement credential rotation policies

```go
// Enable secure caching
options := &cloud.AssumeRoleOptions{
    EnableCredentialCache: true,
    CredentialCacheTTL:    15 * time.Minute,
}
```

## Error Handling

### Common Error Scenarios

1. **Role Not Found**: The specified role doesn't exist
2. **Access Denied**: Insufficient permissions to assume role
3. **MFA Required**: Role requires MFA but none provided
4. **External ID Mismatch**: Provided external ID doesn't match
5. **Session Expired**: Credentials have expired
6. **Invalid Duration**: Requested duration exceeds maximum

### Error Handling Patterns

```go
credentials, err := provider.AssumeRoleAcrossAccount(ctx, sourceAccount, targetAccount, roleName, options)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "MFA"):
        log.Println("MFA required - please provide MFA token")
        // Handle MFA requirement
    case strings.Contains(err.Error(), "ExternalId"):
        log.Println("External ID required or mismatch")
        // Handle external ID issue
    case strings.Contains(err.Error(), "AccessDenied"):
        log.Println("Access denied - check IAM permissions")
        // Handle permission issue
    default:
        log.Printf("Role assumption failed: %v", err)
    }
    return
}
```

## Performance Considerations

### 1. Credential Caching

- Enable caching for frequently used roles
- Set appropriate TTL based on usage patterns
- Monitor cache hit rates

### 2. Connection Pooling

- Use connection pooling for concurrent operations
- Limit concurrent role assumptions
- Implement rate limiting

### 3. Background Refresh

- Use automatic refresh for long-running sessions
- Set appropriate refresh thresholds
- Monitor refresh failures

## Monitoring and Observability

### Metrics to Monitor

1. **Role Assumption Success Rate**
2. **Session Duration Usage**
3. **Cache Hit Rates**
4. **Refresh Failure Rates**
5. **Error Rates by Type**

### Logging

- Log all role assumptions with context
- Include session metadata
- Monitor for security anomalies

```go
log.Printf("Assumed role: %s, session: %s, expires: %v", 
    credentials.Properties["role_arn"],
    credentials.Properties["session_name"],
    credentials.Expiry)
```

## Troubleshooting

### Common Issues

1. **Role assumption fails with AccessDenied**
   - Check IAM trust policy
   - Verify principal is allowed
   - Check for condition requirements

2. **MFA token rejected**
   - Verify MFA device ARN
   - Check token synchronization
   - Ensure token is current

3. **Session expires quickly**
   - Check role's maximum session duration
   - Verify requested duration doesn't exceed maximum
   - Consider using shorter durations with auto-refresh

4. **External ID mismatch**
   - Verify external ID in trust policy
   - Check for typos or case sensitivity
   - Ensure external ID is properly configured

### Debug Mode

Enable debug logging for detailed troubleshooting:

```go
config := &cloud.ProviderConfig{
    Provider:    cloud.ProviderAWS,
    DebugMode:   true,
    Logger:      customLogger,
}
```

## Migration Guide

### From Basic Role Assumption

If you're currently using the basic `AssumeRole` method:

```go
// Old way
credentials, err := provider.AssumeRole(ctx, roleArn, sessionName, duration)

// New way
credentials, err := provider.AssumeRoleWithOptions(ctx, roleArn, &cloud.AssumeRoleOptions{
    SessionName:     sessionName,
    DurationSeconds: duration,
})
```

### Adding Cross-Account Support

```go
// Extract account from role ARN
roleArn := "arn:aws:iam::987654321098:role/MyRole"
parts := strings.Split(roleArn, ":")
targetAccount := parts[4]

// Use cross-account method
credentials, err := provider.AssumeRoleAcrossAccount(ctx, sourceAccount, targetAccount, "MyRole", options)
```

## Advanced Use Cases

### 1. Kubernetes Integration

Use cross-account roles for multi-cluster deployments:

```go
// Assume role for different cluster account
clusterCredentials, err := provider.AssumeRoleAcrossAccount(
    ctx, currentAccount, clusterAccount, "KubernetesAccessRole", options)

// Configure kubectl with new credentials
err = configureKubectl(clusterCredentials)
```

### 2. CI/CD Pipeline Integration

Implement cross-account deployments:

```go
// Development to Production deployment
deployCredentials, err := provider.AssumeRoleChain(ctx, []*cloud.RoleChainStep{
    {RoleArn: "arn:aws:iam::dev-account:role/DeployRole"},
    {RoleArn: "arn:aws:iam::prod-account:role/ProductionDeployRole"},
})
```

### 3. Audit and Compliance

Track cross-account access for compliance:

```go
// Get all active sessions
sessions := roleManager.ListSessions()
for _, session := range sessions {
    auditLog := map[string]interface{}{
        "role_arn":    session.RoleArn,
        "source_arn":  session.SourceArn,
        "created_at":  session.CreatedAt,
        "expires_at":  session.ExpiresAt,
    }
    // Send to audit system
}
```

## API Stability

This API is designed to be stable and backward-compatible. Any breaking changes will be:
1. Announced with deprecation warnings
2. Supported for at least 6 months
3. Documented with migration guides

## Support

For issues and questions:
1. Check the troubleshooting section
2. Review error messages for specific guidance
3. Enable debug logging for detailed information
4. Consult the example implementations

## Related Documentation

- [AWS IAM Role Documentation](docs/aws-iam-roles.md)
- [Security Best Practices](docs/security-best-practices.md)
- [Multi-Account Architecture](docs/multi-account-architecture.md)
- [Monitoring and Alerting](docs/monitoring-alerting.md)