# Deployment Monitoring and Rollback

The APM platform provides comprehensive deployment monitoring and rollback capabilities for applications deployed across Kubernetes, Docker, and cloud platforms.

## Features

### 1. Real-time Deployment Progress Tracking

Monitor your deployments in real-time with:
- **WebSocket-based updates**: Get instant status changes without polling
- **Multi-stage tracking**: Monitor preparing, deploying, verifying, and completion stages
- **Component-level visibility**: Track individual component deployments
- **Progress indicators**: See percentage completion and estimated time remaining

### 2. Health Check Integration

Comprehensive health monitoring during and after deployment:
- **Kubernetes probes**: Readiness, liveness, and startup probe monitoring
- **Custom health endpoints**: Validate application-specific health checks
- **Service mesh integration**: Monitor Istio service health
- **Dependency verification**: Check database connections and external services

### 3. Automatic Rollback Generation

Platform-specific rollback strategies:
- **Kubernetes**: `kubectl rollout undo` with revision management
- **Docker**: Container state preservation and restoration
- **Cloud platforms**: Native rollback APIs for AWS, GCP, and Azure
- **Dependency-aware ordering**: Rollback components in the correct sequence

### 4. Deployment History Tracking

Complete audit trail of all deployments:
- **Configuration snapshots**: Track what was deployed
- **Timeline visualization**: See deployment duration and stages
- **Success/failure metrics**: Monitor deployment reliability
- **Rollback history**: Track why and when rollbacks occurred

## API Endpoints

### Start a Deployment

```bash
POST /api/v1/deployments
```

Request body:
```json
{
  "name": "my-app",
  "version": "v2.0.0",
  "platform": "kubernetes",
  "environment": "production",
  "configuration": {
    "deployment_name": "my-app-deployment",
    "service_name": "my-app-service",
    "labels": {
      "app": "my-app",
      "version": "v2.0.0"
    }
  }
}
```

### Get Deployment Status

```bash
GET /api/v1/deployments/{deployment-id}
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "my-app",
  "version": "v2.0.0",
  "status": "deploying",
  "progress": {
    "percentage": 66.7,
    "current_step": 2,
    "total_steps": 3,
    "current_stage": "Waiting for pods to be ready"
  },
  "health_checks": [
    {
      "name": "pod-abc123-readiness",
      "type": "readiness",
      "status": "healthy",
      "message": "Pod is ready"
    }
  ]
}
```

### WebSocket Real-time Updates

```javascript
const ws = new WebSocket('ws://localhost:8080/ws/deployments?deployment_id=550e8400-e29b-41d4-a716-446655440000');

ws.onmessage = (event) => {
  const update = JSON.parse(event.data);
  console.log(`${update.type}: ${JSON.stringify(update.data)}`);
};
```

### Initiate Rollback

```bash
POST /api/v1/deployments/{deployment-id}/rollback
```

Request body:
```json
{
  "reason": "Application errors detected",
  "target_version": "v1.9.0"
}
```

## CLI Commands

### Deploy Management

```bash
# Start a new deployment
apm deploy start --name my-app --version v2.0.0 --platform kubernetes --environment production

# Watch deployment progress
apm deploy start --name my-app --version v2.0.0 --watch

# Check deployment status
apm deploy status 550e8400-e29b-41d4-a716-446655440000

# List recent deployments
apm deploy list --platform kubernetes --status completed --limit 10

# Rollback a deployment
apm deploy rollback 550e8400-e29b-41d4-a716-446655440000 --reason "Performance degradation"

# Dry-run rollback (show commands without executing)
apm deploy rollback 550e8400-e29b-41d4-a716-446655440000 --dry-run
```

## Dashboard Integration

### Grafana Dashboard

The deployment monitoring system provides a pre-configured Grafana dashboard with:

1. **Deployment Status Overview**: Current status of all deployments
2. **Success Rate Gauge**: Percentage of successful deployments
3. **Average Duration Graph**: Deployment time by platform
4. **Platform Distribution**: Pie chart of deployments by platform
5. **Health Check Status**: Table of current health checks
6. **Rollback Trend**: Graph showing rollback frequency over time
7. **Active Deployments**: Real-time table of ongoing deployments

### Prometheus Metrics

The following metrics are exposed for Prometheus:

```
# Total deployments by platform, environment, and version
deployment_total{platform="kubernetes",environment="production",version="v2.0.0"} 1

# Deployment duration in seconds
deployment_duration_seconds{platform="kubernetes",environment="production",version="v2.0.0"} 180.5

# Current deployment status
deployment_status{platform="kubernetes",environment="production",version="v2.0.0",status="completed"} 1

# Health check status
deployment_health_check_status{platform="kubernetes",check_type="readiness",status="healthy"} 3

# Rollback count
deployment_rollback_total{platform="kubernetes",environment="production",version="v2.0.0"} 0
```

## Configuration

### Kubernetes Monitor Configuration

```yaml
deployment:
  kubernetes:
    kubeconfig: ~/.kube/config  # Optional, uses in-cluster config if not specified
    namespace: default
    monitor:
      interval: 10s
      timeout: 30m
```

### Database Configuration

```yaml
deployment:
  history:
    database:
      url: postgres://user:password@localhost:5432/apm_deployments
      max_connections: 10
      max_idle: 5
```

### Redis Configuration

```yaml
deployment:
  cache:
    redis:
      url: redis://localhost:6379
      max_idle: 3
      max_active: 10
```

## Platform-Specific Features

### Kubernetes

- Deployment resource monitoring with revision tracking
- Rolling update progress tracking
- Pod health aggregation
- Service endpoint verification
- Helm release management support
- kubectl command generation for manual intervention

### Docker/Docker Compose

- Container state monitoring
- Service dependency tracking
- Volume backup and restore capabilities
- Network configuration preservation
- docker-compose scale tracking

### Cloud Platforms

- **AWS**: ECS/EKS deployment monitoring, Auto Scaling Group tracking
- **GCP**: GKE deployment monitoring, Cloud Run revision tracking
- **Azure**: AKS deployment monitoring, Container Instances tracking

## Security Considerations

1. **Authentication**: All deployment operations require proper authentication
2. **RBAC**: Role-based access control for deployment management
3. **Audit Logging**: All deployment actions are logged with actor information
4. **Encrypted Storage**: Sensitive configuration data is encrypted at rest
5. **TLS**: All WebSocket connections should use WSS in production

## Best Practices

1. **Always specify a reason** when initiating rollbacks for audit trail
2. **Use dry-run** to preview rollback commands before execution
3. **Monitor health checks** continuously during deployment
4. **Set appropriate timeouts** for different deployment stages
5. **Configure alerts** for failed deployments and rollbacks
6. **Maintain deployment history** for compliance and troubleshooting

## Troubleshooting

### Common Issues

1. **Deployment stuck in "verifying" state**
   - Check pod logs for startup errors
   - Verify health check endpoints are responding
   - Review resource limits and requests

2. **WebSocket connection drops**
   - Check network connectivity
   - Verify WebSocket proxy configuration
   - Review connection timeout settings

3. **Rollback fails**
   - Ensure previous version resources still exist
   - Check RBAC permissions for rollback operations
   - Verify storage backend has previous configurations

## API Reference

For complete API documentation, see the [API Reference](./api-reference.md#deployment-monitoring).