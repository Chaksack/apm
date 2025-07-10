# CloudWatch Integration Examples

This directory contains comprehensive examples and documentation for the CloudWatch integration functionality in the APM tool.

## Contents

- `main.go` - Complete CloudWatch integration demonstration
- `cloudwatch_test.go` - Comprehensive test suite for CloudWatch functionality
- `CLOUDWATCH_DOCUMENTATION.md` - Complete documentation and usage guide
- `README.md` - This file

## Quick Start

### Prerequisites

1. **AWS CLI**: Install and configure AWS CLI v2.x
2. **AWS Credentials**: Configure with appropriate CloudWatch permissions
3. **Go Environment**: Go 1.23+ required

### Running the Examples

```bash
# Clone the repository
git clone https://github.com/chaksack/apm.git
cd apm/examples/cloudwatch-integration

# Run the main demo
go run main.go

# Run the test suite
go test -v ./...
```

### What the Demo Shows

The main demo (`main.go`) demonstrates:

1. **Dashboard Management**: Creating APM-specific dashboards
2. **Alarm Management**: Setting up intelligent alerting
3. **Log Management**: Centralized logging with retention policies
4. **CloudWatch Insights**: Advanced log analysis and querying
5. **Events and SNS**: Automated notifications and event handling
6. **APM Integration**: Native integration with Prometheus, Grafana, Jaeger, and Loki
7. **Health Checks**: Monitoring system health and performance

## CloudWatch Features Demonstrated

### Dashboard Types

- **Infrastructure**: CPU, memory, disk, network metrics
- **Application**: Request rates, response times, error rates
- **Service Mesh**: Istio/Envoy metrics, service topology
- **Logs**: Log volume, error rates, pattern analysis
- **Tracing**: Trace latency, service dependencies
- **Cost Optimization**: Resource utilization, cost trends

### Alarm Types

- **High CPU Utilization**: > 80%
- **High Memory Utilization**: > 85%
- **Low Disk Space**: < 20%
- **Service Down**: Health check failures
- **High Error Rate**: > 5%
- **Slow Response Time**: > 2 seconds
- **APM Tool Specific**: Prometheus, Grafana, Jaeger, Loki monitoring

### Log Management Features

- **Log Groups**: Organized by APM tool and environment
- **Metric Filters**: Convert log events to metrics
- **Retention Policies**: Automatic log cleanup
- **Real-time Streaming**: Live log monitoring
- **Encryption**: Secure log storage

### Insights Queries

- **Error Analysis**: Error patterns and trends
- **Performance Metrics**: Response time analysis
- **Request Patterns**: Traffic analysis by method/endpoint
- **Service Health**: Health check status monitoring

### Event Processing

- **EC2 State Changes**: Instance lifecycle monitoring
- **Auto Scaling Events**: Scaling activity tracking
- **Deployment Events**: Application deployment monitoring
- **Custom Events**: Application-specific event handling

### SNS Integration

- **Multi-channel Notifications**: Email, SMS, Slack, webhooks
- **Severity-based Routing**: Critical, warning, info channels
- **Filter Policies**: Selective notification delivery
- **Message Formatting**: Custom notification templates

## APM Tool Integration

### Prometheus Integration

- **Metrics Collection**: Custom APM metrics
- **Scrape Configuration**: Automated target discovery
- **Alert Rules**: Prometheus alerting integration
- **Federation**: Multi-cluster metrics aggregation

### Grafana Integration

- **Dashboard Management**: Automated dashboard deployment
- **Datasource Configuration**: CloudWatch, Prometheus, Loki integration
- **Alert Management**: Grafana alerting integration
- **User Management**: Automated user provisioning

### Jaeger Integration

- **Trace Collection**: Distributed tracing monitoring
- **Service Maps**: Service dependency visualization
- **Performance Analysis**: Trace latency analysis
- **Error Tracking**: Failed trace monitoring

### Loki Integration

- **Log Aggregation**: Centralized log collection
- **Query Interface**: LogQL query integration
- **Alert Rules**: Log-based alerting
- **Retention Management**: Log lifecycle management

## Performance Optimization

### Caching

- **Intelligent Caching**: Automatic cache warming
- **TTL Management**: Configurable cache expiration
- **Cache Statistics**: Hit rate monitoring
- **Cache Invalidation**: Selective cache clearing

### Batch Operations

- **Bulk Creation**: Multiple resources in single call
- **Parallel Processing**: Concurrent operation execution
- **Error Handling**: Individual operation error tracking
- **Progress Monitoring**: Operation status tracking

### Connection Pooling

- **Connection Management**: Optimized AWS API connections
- **Timeout Configuration**: Configurable operation timeouts
- **Retry Logic**: Exponential backoff with jitter
- **Circuit Breaker**: Failure protection and recovery

## Error Handling

### Error Classification

- **Retryable Errors**: Automatic retry with backoff
- **Permanent Errors**: Immediate failure reporting
- **Rate Limiting**: Throttling detection and handling
- **Permission Issues**: Clear error messages and solutions

### Health Monitoring

- **Service Health**: Individual service status monitoring
- **Overall Health**: Aggregate system health status
- **Performance Metrics**: Response time and success rate tracking
- **Alert Integration**: Health-based alerting

## Testing

The test suite (`cloudwatch_test.go`) includes:

### Unit Tests

- **Manager Initialization**: Component creation and configuration
- **Dashboard Management**: CRUD operations testing
- **Alarm Management**: Alarm lifecycle testing
- **Log Management**: Log group and stream testing
- **Insights Queries**: Query execution and result handling
- **SNS Operations**: Topic and subscription management
- **Event Rules**: Event processing and routing

### Integration Tests

- **APM Integration**: End-to-end APM tool integration
- **Multi-region Operations**: Cross-region functionality
- **Concurrent Operations**: Parallel operation testing
- **Error Scenarios**: Error handling and recovery
- **Performance Testing**: Load and stress testing

### Test Coverage

- **Functionality**: All core features tested
- **Error Paths**: Error conditions and edge cases
- **Performance**: Response time and throughput testing
- **Concurrency**: Thread safety and race condition testing

## Configuration Examples

### Environment Variables

```bash
# AWS Configuration
export AWS_REGION=us-east-1
export AWS_PROFILE=apm-production

# APM Configuration
export APM_ENVIRONMENT=production
export APM_TOOLS=prometheus,grafana,jaeger,loki

# CloudWatch Configuration
export CLOUDWATCH_LOG_RETENTION_DAYS=30
export CLOUDWATCH_DASHBOARD_REFRESH_INTERVAL=300
export CLOUDWATCH_ALARM_EVALUATION_PERIODS=2
```

### Configuration Files

See `CLOUDWATCH_DOCUMENTATION.md` for detailed configuration examples including:

- Dashboard configuration JSON
- Alarm configuration YAML
- Log group configuration
- SNS topic and subscription setup
- Event rule patterns
- APM tool integration configs

## Best Practices

### Resource Naming

- **Consistent Prefixes**: Use `APM-` prefix for all resources
- **Environment Suffixes**: Include environment in resource names
- **Descriptive Names**: Clear, self-documenting resource names

### Tag Strategy

- **Standard Tags**: Environment, Service, Team, Purpose
- **Cost Allocation**: Use tags for cost tracking and optimization
- **Automation**: Tags for automated resource management

### Security

- **IAM Policies**: Least privilege access controls
- **Encryption**: Enable encryption for sensitive data
- **Access Logging**: Monitor all access and changes
- **Regular Audits**: Periodic security reviews

### Cost Optimization

- **Retention Policies**: Appropriate log retention periods
- **Resource Cleanup**: Regular removal of unused resources
- **Usage Monitoring**: Track CloudWatch costs and usage
- **Efficient Queries**: Optimize Insights queries for performance

## Troubleshooting

### Common Issues

1. **Permission Errors**: Check IAM policies and resource permissions
2. **Resource Limits**: Verify account limits and quotas
3. **Network Issues**: Check VPC configuration and security groups
4. **Rate Limiting**: Implement proper retry logic and backoff

### Debug Mode

Enable debug logging for detailed troubleshooting:

```go
manager.SetLogLevel(cloud.LogLevelDebug)
```

### Health Checks

Use built-in health checks to monitor system status:

```go
healthResult := manager.HealthCheck(ctx, "us-east-1")
```

## Support

For questions or issues:

1. Check the documentation in `CLOUDWATCH_DOCUMENTATION.md`
2. Review the test cases in `cloudwatch_test.go`
3. Examine the example code in `main.go`
4. Refer to AWS CloudWatch documentation
5. Check the main project repository for updates

## Contributing

To contribute to the CloudWatch integration:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Update documentation
5. Submit a pull request

## License

This project is licensed under the same license as the main APM project.