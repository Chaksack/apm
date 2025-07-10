# Cross-Account Role Assumption Example

This example demonstrates the comprehensive cross-account role assumption functionality implemented in the APM tool.

## Features Demonstrated

1. **Basic Cross-Account Role Assumption**: Assume roles across different AWS accounts
2. **MFA-Based Role Assumption**: Enhanced security with multi-factor authentication
3. **Role Chaining**: Complex multi-account scenarios with role chains
4. **External ID Support**: Partner integrations with external ID validation
5. **Automatic Session Management**: Background refresh and credential caching
6. **Role Validation**: Pre-flight validation of role assumption capabilities
7. **Multi-Account Configuration**: Centralized configuration management
8. **Configuration Persistence**: Save/load configurations from file or S3

## Prerequisites

- AWS CLI installed and configured
- Go 1.21 or later
- Appropriate AWS IAM permissions for role assumption

## Running the Example

```bash
# Navigate to the example directory
cd examples/cross-account-roles

# Run the example
go run main.go

# Run the tests
go test -v

# Run benchmarks
go test -bench=.
```

## Configuration

The example uses mock account IDs and role ARNs. In a real environment, you would:

1. Replace the account IDs with your actual AWS account IDs
2. Ensure the roles exist and have proper trust policies
3. Configure appropriate IAM permissions
4. Set up MFA devices if using MFA features

## Example Trust Policy

For cross-account role assumption, the target role needs a trust policy like:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::SOURCE-ACCOUNT-ID:root"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "sts:ExternalId": "unique-external-id"
        },
        "Bool": {
          "aws:MultiFactorAuthPresent": "true"
        }
      }
    }
  ]
}
```

## Key Files

- `main.go`: Comprehensive example demonstrating all features
- `cross_account_test.go`: Unit tests and benchmarks
- `README.md`: This documentation
- `go.mod`: Go module configuration

## Documentation

For complete documentation, see:
- [Cross-Account Role Assumption Documentation](../../docs/cross-account-role-assumption.md)
- [AWS Integration Guide](../../docs/aws-integration-guide.md)

## Security Notes

- Never commit actual AWS credentials to version control
- Use IAM roles with least privilege principles
- Enable MFA for sensitive operations
- Monitor and audit all cross-account access
- Implement appropriate session duration limits