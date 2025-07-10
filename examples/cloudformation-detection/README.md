# CloudFormation Stack Detection Example

This example demonstrates the comprehensive CloudFormation stack detection functionality for APM infrastructure monitoring.

## Features Demonstrated

### 1. Stack Discovery and Filtering
- List CloudFormation stacks across multiple regions
- Filter stacks by status, tags, and naming conventions
- Identify APM-related stacks automatically
- Parallel discovery across regions for performance

### 2. APM Resource Identification
- Automatic detection of APM-related resources:
  - Application Load Balancers (ALB/NLB)
  - ECS Services and Task Definitions
  - RDS Instances and Aurora Clusters
  - Lambda Functions
  - ElastiCache Clusters
  - S3 Buckets
  - VPC Resources
- Detailed resource mapping with connection details

### 3. Stack Health Validation
- Comprehensive health checking of stack resources
- Individual resource health assessment
- Health recommendations and issue identification
- Overall stack health scoring

### 4. Infrastructure Drift Detection
- Automated drift detection for stack resources
- Property-level difference analysis
- Drift recommendations and remediation actions
- Real-time drift status monitoring

### 5. Cross-Region Operations
- Multi-region stack discovery with parallel processing
- Region-specific filtering and optimization
- Performance optimizations with concurrent operations

## Prerequisites

1. **AWS CLI v2.x** installed and configured
2. **AWS credentials** properly set up (via `aws configure` or environment variables)
3. **IAM permissions** for CloudFormation operations:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": [
           "cloudformation:ListStacks",
           "cloudformation:DescribeStacks",
           "cloudformation:ListStackResources",
           "cloudformation:DetectStackDrift",
           "cloudformation:DescribeStackDriftDetectionStatus",
           "cloudformation:DescribeStackResourceDrifts",
           "elbv2:DescribeLoadBalancers",
           "ecs:DescribeServices",
           "rds:DescribeDBInstances",
           "lambda:GetFunction",
           "elasticache:DescribeCacheClusters",
           "s3:GetBucketLocation",
           "s3:ListBuckets",
           "s3:GetBucketVersioning",
           "s3:GetBucketEncryption",
           "ec2:DescribeVpcs",
           "ec2:DescribeSubnets",
           "ec2:DescribeRouteTables",
           "ec2:DescribeSecurityGroups"
         ],
         "Resource": "*"
       }
     ]
   }
   ```

## Usage

### Basic Execution
```bash
# Navigate to the example directory
cd examples/cloudformation-detection

# Run the example
go run main.go
```

### Expected Output

```
🔍 CloudFormation Stack Detection for APM Infrastructure
========================================================

1. Listing CloudFormation stacks...
Found 3 CloudFormation stacks
  - my-apm-stack (arn:aws:cloudformation:us-east-1:123456789012:stack/my-apm-stack/...) in us-east-1 - Status: CREATE_COMPLETE
  - web-app-infrastructure (arn:aws:cloudformation:us-west-2:123456789012:stack/web-app-infrastructure/...) in us-west-2 - Status: UPDATE_COMPLETE

2. Listing APM-specific CloudFormation stacks...
Found 1 APM-related stacks
  - my-apm-stack in us-east-1 - APM Resources: true
    Load Balancers: 1
      - my-alb-1234567890.us-east-1.elb.amazonaws.com (application): internet-facing
    ECS Services: 2
      - web-service in my-cluster: 3/3 tasks
      - api-service in my-cluster: 2/2 tasks
    RDS Instances: 1
      - my-database (postgres): available

3. Getting detailed information for first stack...
Stack: my-apm-stack
  Status: CREATE_COMPLETE
  Created: 2024-01-15T10:30:00Z
  Resources: 15
  Parameters: 5
  Outputs: 8
  Tags: 3
  Is APM Stack: true
  Top 5 Resources:
    - MyLoadBalancer (AWS::ElasticLoadBalancingV2::LoadBalancer): CREATE_COMPLETE
    - MyECSService (AWS::ECS::Service): CREATE_COMPLETE
    - MyDatabase (AWS::RDS::DBInstance): CREATE_COMPLETE
    - MyVPC (AWS::EC2::VPC): CREATE_COMPLETE
    - MyLambdaFunction (AWS::Lambda::Function): CREATE_COMPLETE
  Outputs:
    LoadBalancerDNS: my-alb-1234567890.us-east-1.elb.amazonaws.com
    DatabaseEndpoint: my-database.cluster-xyz.us-east-1.rds.amazonaws.com:5432
    APIEndpoint: https://api.example.com

4. Validating stack health...
Overall Health: healthy
Healthy Resources: 14
Unhealthy Resources: 1
Issues:
  - Lambda function MyLambdaFunction has errors in recent executions
Recommendations:
  - Stack is mostly healthy - investigate Lambda function issues
  - Review CloudWatch logs for application errors

5. Detecting stack drift (this may take a few minutes)...
Drift Status: DRIFTED
Total Resources: 15
Drifted Resources: 2
Detection Time: 2024-01-15T11:45:30Z
Drifted Resources:
  - MyLoadBalancer (AWS::ElasticLoadBalancingV2::LoadBalancer): MODIFIED
  - MyECSService (AWS::ECS::Service): MODIFIED
Recommended Actions:
  - Review drifted resources and determine if changes were intentional
  - Consider updating CloudFormation template to match current state
  - 2 Load Balancer(s) drifted - check security groups and listeners

6. Getting APM infrastructure summary...
APM Infrastructure Summary:
  Total Stacks: 1
  Healthy: 1, Degraded: 0, Unhealthy: 0
  Last Updated: 2024-01-15T12:00:00Z
  Resource Summary:
    Load Balancers: 1
    ECS Services: 2
    RDS Instances: 1
    Lambda Functions: 3
    ElastiCache Clusters: 0
    S3 Buckets: 2
    VPCs: 1
  By Region:
    us-east-1: 1 stacks (1 healthy)

7. Searching for APM resources...

Searching for loadbalancer resources...
Found 1 loadbalancer resources
  - my-alb-1234567890.us-east-1.elb.amazonaws.com (LoadBalancer) in my-apm-stack: active
    Endpoint: my-alb-1234567890.us-east-1.elb.amazonaws.com

Searching for ecs resources...
Found 2 ecs resources
  - web-service (ECSService) in my-apm-stack: ACTIVE
    Endpoint: my-cluster/web-service
  - api-service (ECSService) in my-apm-stack: ACTIVE
    Endpoint: my-cluster/api-service

✅ CloudFormation stack detection demonstration completed!

8. Exporting summary to JSON...
Summary exported to apm-infrastructure-summary.json
```

## Generated Files

The example creates an `apm-infrastructure-summary.json` file containing detailed infrastructure information:

```json
{
  "totalStacks": 1,
  "healthyStacks": 1,
  "degradedStacks": 0,
  "unhealthyStacks": 0,
  "regionSummary": {
    "us-east-1": {
      "region": "us-east-1",
      "stackCount": 1,
      "healthyStacks": 1,
      "issues": []
    }
  },
  "resourceSummary": {
    "loadBalancers": 1,
    "ecsServices": 2,
    "rdsInstances": 1,
    "lambdaFunctions": 3,
    "elastiCacheClusters": 0,
    "s3Buckets": 2,
    "vpcs": 1
  },
  "lastUpdated": "2024-01-15T12:00:00Z"
}
```

## Key CloudFormation Features

### Stack Filtering
```go
filters := &cloud.StackFilters{
    Regions: []string{"us-east-1", "us-west-2"},
    StackStatus: []string{"CREATE_COMPLETE", "UPDATE_COMPLETE"},
    APMOnly: true,
    Tags: map[string]string{
        "Environment": "production",
        "Application": "apm",
    },
}
```

### Resource Detection
The system automatically identifies APM resources by:
- **Resource Types**: ALB, ECS, RDS, Lambda, ElastiCache, S3, VPC
- **Tags**: Monitoring, observability, APM-related tags
- **Naming Conventions**: Common APM naming patterns
- **Resource Relationships**: Connected infrastructure components

### Health Validation
Each resource type has specific health checks:
- **Load Balancers**: State and target health
- **ECS Services**: Task count and service status
- **RDS Instances**: Database availability
- **Lambda Functions**: Function state and execution status

### Drift Detection
Comprehensive drift analysis including:
- **Resource-level drift**: Individual resource changes
- **Property differences**: Detailed change analysis
- **Recommendations**: Actionable remediation steps

## Error Handling

The implementation includes comprehensive error handling for:
- **AWS CLI authentication issues**
- **Region accessibility problems**
- **Resource permission limitations**
- **Network connectivity issues**
- **CloudFormation service limits**

## Performance Considerations

- **Parallel Processing**: Multi-region operations run concurrently
- **Intelligent Filtering**: Reduces API calls with targeted queries
- **Caching**: Resource information cached for efficiency
- **Timeout Handling**: Appropriate timeouts for long-running operations

## Integration Examples

### Using in Production Code
```go
// Initialize provider
provider, err := cloud.NewAWSProvider(&cloud.ProviderConfig{
    DefaultRegion: "us-east-1",
    EnableCache:   true,
})

// Get APM infrastructure summary
summary, err := provider.GetAPMStackSummary(ctx, []string{
    "us-east-1", "us-west-2", "eu-west-1",
})

// Search for specific resources
loadBalancers, err := provider.SearchAPMResources(ctx, "loadbalancer", regions)

// Monitor stack health
health, err := provider.ValidateCloudFormationStackHealth(ctx, stackName, region)
```

## Troubleshooting

### Common Issues

1. **AWS CLI Not Found**
   - Ensure AWS CLI v2.x is installed
   - Verify PATH contains AWS CLI location

2. **Authentication Errors**
   - Run `aws sts get-caller-identity` to test credentials
   - Check IAM permissions for CloudFormation operations

3. **Permission Denied**
   - Verify IAM policy includes all required permissions
   - Check resource-level permissions for specific services

4. **No Stacks Found**
   - Verify stacks exist in specified regions
   - Check stack status filters
   - Ensure stacks contain APM-related resources

5. **Drift Detection Timeout**
   - Large stacks may take several minutes for drift detection
   - Check CloudFormation service limits
   - Verify network connectivity

### Debug Mode

Enable debug logging by setting environment variables:
```bash
export AWS_DEBUG=true
export APM_LOG_LEVEL=debug
go run main.go
```

This comprehensive example demonstrates all aspects of CloudFormation stack detection for APM infrastructure, providing a solid foundation for production monitoring and management systems.