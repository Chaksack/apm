# AWS Role Chaining Implementation

## Overview

The AWS Role Chaining functionality in the APM tool provides a robust solution for assuming multiple IAM roles sequentially across different AWS accounts. This is essential for complex enterprise environments where access to resources requires traversing multiple account boundaries with different trust relationships.

## Features

### Core Capabilities

1. **Sequential Role Assumption**: Assume multiple roles in sequence, where each subsequent role is assumed using the credentials from the previous role.

2. **Credential Passing**: Secure passing of temporary credentials between chain steps without using environment variables.

3. **Error Handling and Rollback**: Comprehensive error handling with automatic rollback on failure at any step in the chain.

4. **Validation**: Pre-validation of role chain configurations to detect issues before execution.

5. **Session Management**: Track and manage multiple role chain sessions with automatic refresh capabilities.

## Architecture

### Components

1. **RoleChainManager**: Core component that manages the execution of role chains
   - Handles credential propagation between steps
   - Manages session lifecycle
   - Provides automatic refresh for expiring credentials

2. **ChainedSession**: Represents an active role chain session
   - Tracks all credentials in the chain
   - Maintains session metadata
   - Supports refresh operations

3. **RoleChainConfig**: Configuration for chain behavior
   - Maximum chain length
   - Retry policies
   - Auto-refresh settings
   - Session duration defaults

## Usage

### Basic Role Chain

```go
// Define a simple two-hop chain
roleChain := []*cloud.RoleChainStep{
    {
        RoleArn:     "arn:aws:iam::123456789012:role/FirstRole",
        SessionName: "first-hop",
    },
    {
        RoleArn:     "arn:aws:iam::987654321098:role/SecondRole",
        SessionName: "second-hop",
    },
}

// Execute the chain
credentials, err := awsProvider.AssumeRoleChain(ctx, roleChain)
if err != nil {
    log.Fatalf("Failed to assume role chain: %v", err)
}
```

### Complex Chain with External IDs

```go
roleChain := []*cloud.RoleChainStep{
    {
        RoleArn:     "arn:aws:iam::111111111111:role/OrganizationRole",
        SessionName: "org-access",
        Options: &cloud.AssumeRoleOptions{
            DurationSeconds: 3600,
            Tags: map[string]string{
                "Purpose": "CrossAccountAccess",
            },
        },
    },
    {
        RoleArn:     "arn:aws:iam::222222222222:role/PartnerRole",
        ExternalID:  "unique-partner-id-12345",
        SessionName: "partner-access",
        Options: &cloud.AssumeRoleOptions{
            Policy: `{...}`, // Scoped-down policy
        },
    },
}
```

### Using the Enhanced Chain Manager

```go
// Create chain manager with configuration
chainManager := cloud.NewRoleChainManager(awsProvider)
defer chainManager.Close()

config := &cloud.RoleChainConfig{
    MaxSteps:              5,
    DefaultDuration:       3600,
    RefreshBeforeExpiry:   5 * time.Minute,
    EnableAutoRefresh:     true,
    RetryAttempts:         3,
    RetryDelay:            time.Second,
}

// Execute chain with session management
session, err := chainManager.AssumeRoleChain(ctx, roleChain, config)
if err != nil {
    log.Fatalf("Chain execution failed: %v", err)
}

// Access final credentials
finalCreds := session.FinalCreds
```

## Configuration Options

### RoleChainStep

| Field | Type | Description |
|-------|------|-------------|
| RoleArn | string | The ARN of the role to assume |
| ExternalID | string | External ID for third-party access |
| SessionName | string | Name for the assumed role session |
| Options | *AssumeRoleOptions | Additional options for role assumption |

### RoleChainConfig

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| MaxSteps | int | 5 | Maximum number of roles in a chain |
| DefaultDuration | int | 3600 | Default session duration in seconds |
| RefreshBeforeExpiry | time.Duration | 5 minutes | Time before expiry to trigger refresh |
| EnableAutoRefresh | bool | true | Enable automatic credential refresh |
| RetryAttempts | int | 3 | Number of retry attempts on failure |
| RetryDelay | time.Duration | 1 second | Delay between retry attempts |

## Security Considerations

### Credential Isolation

- Credentials are passed between chain steps using isolated command execution
- No global environment variable pollution
- Each step uses only the credentials from the previous step

### Validation

- Pre-execution validation of role ARNs
- Circular dependency detection
- External ID validation
- Trust policy compatibility checking

### Error Handling

- Automatic rollback on failure
- Detailed error messages with context
- Retry logic with exponential backoff
- Graceful degradation options

## Best Practices

### 1. Minimize Chain Length

Keep role chains as short as possible to reduce complexity and latency:

```go
// Good: Direct path
chain := []*RoleChainStep{
    {RoleArn: "arn:aws:iam::123:role/Gateway"},
    {RoleArn: "arn:aws:iam::456:role/Target"},
}

// Avoid: Unnecessary hops
chain := []*RoleChainStep{
    {RoleArn: "arn:aws:iam::123:role/Role1"},
    {RoleArn: "arn:aws:iam::123:role/Role2"}, // Same account
    {RoleArn: "arn:aws:iam::456:role/Target"},
}
```

### 2. Use Appropriate Session Durations

Configure session durations based on your use case:

```go
config := &RoleChainConfig{
    DefaultDuration: 900,  // 15 minutes for short tasks
    // or
    DefaultDuration: 3600, // 1 hour for longer operations
}
```

### 3. Implement Proper Error Handling

Always handle errors and implement fallback strategies:

```go
session, err := chainManager.AssumeRoleChain(ctx, roleChain, config)
if err != nil {
    // Check if it's a validation error
    if strings.Contains(err.Error(), "validation") {
        // Handle validation errors
    }
    // Implement fallback strategy
    return useAlternativeAccess()
}
```

### 4. Monitor Chain Performance

Track chain execution times and success rates:

```go
start := time.Now()
session, err := chainManager.AssumeRoleChain(ctx, roleChain, config)
duration := time.Since(start)

// Log metrics
log.Printf("Chain execution took %v", duration)
```

## Troubleshooting

### Common Issues

1. **"Failed at step X"**: Check trust relationships for the role at step X
2. **"External ID mismatch"**: Verify the external ID matches the trust policy
3. **"Circular dependency detected"**: Ensure no role appears twice in the chain
4. **"Maximum chain length exceeded"**: Reduce the number of steps or increase MaxSteps

### Debug Mode

Enable debug mode for detailed logging:

```go
provider := &cloud.AWSProvider{
    config: &cloud.ProviderConfig{
        DebugMode: true,
    },
}
```

## Examples

See the `/examples/aws-role-chain/` directory for complete working examples including:

- Simple two-hop chains
- Complex multi-account scenarios
- MFA-protected role chains
- Chains with external IDs
- Session management examples

## Limitations

1. Maximum chain length is configurable but defaults to 5 steps
2. Each role assumption adds latency (typically 1-2 seconds)
3. Credentials at each step have independent expiration times
4. MFA can only be used at the first step of a chain

## Future Enhancements

1. Parallel chain execution for independent paths
2. Credential caching across chain executions
3. Integration with AWS SSO
4. Support for assuming roles in AWS GovCloud and China regions
5. Chain visualization and debugging tools