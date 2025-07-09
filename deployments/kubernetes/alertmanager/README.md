# AlertManager Configuration

This directory contains the complete AlertManager configuration for the APM system, including routing rules, notification templates, and secrets management.

## Files Overview

- **alertmanager-config.yaml**: Main configuration file with routing rules and receiver definitions
- **notification-templates.yaml**: Email and Slack notification templates
- **secrets.yaml**: Template for sensitive credentials (DO NOT commit actual values)
- **deployment-updated.yaml**: Updated deployment manifest that mounts all configurations
- **configmap.yaml**: Legacy configuration (kept for backward compatibility)
- **deployment.yaml**: Original deployment manifest
- **service.yaml**: Service definition for AlertManager

## Setup Instructions

### 1. Create Namespace (if not exists)
```bash
kubectl create namespace monitoring
```

### 2. Create Secrets
First, create actual secrets from the template:
```bash
# Copy the template
cp secrets.yaml secrets-actual.yaml

# Edit secrets-actual.yaml and replace all placeholder values
# DO NOT commit secrets-actual.yaml to version control

# Apply the secrets
kubectl apply -f secrets-actual.yaml
```

### 3. Apply Configurations
```bash
# Apply the main configuration
kubectl apply -f alertmanager-config.yaml

# Apply the notification templates
kubectl apply -f notification-templates.yaml

# Apply the updated deployment (includes PVC, Service, ServiceAccount)
kubectl apply -f deployment-updated.yaml
```

### 4. Verify Deployment
```bash
# Check if AlertManager is running
kubectl get pods -n monitoring -l app=alertmanager

# Check logs
kubectl logs -n monitoring -l app=alertmanager

# Access AlertManager UI (port-forward)
kubectl port-forward -n monitoring svc/alertmanager 9093:9093
# Then open http://localhost:9093
```

## Configuration Details

### Routing Tree Structure

The routing configuration follows this hierarchy:

1. **Root Route**: Groups by alertname, cluster, namespace, and service
2. **Severity-based Routes**:
   - Critical: Immediate notifications (10s group wait)
   - High: Quick notifications (30s group wait)
   - Warning: Grouped notifications (5m group wait)
   - Info: Low priority (10m group wait)
3. **Environment Routes**: Special handling for production namespace
4. **Service Routes**: Team-specific routing based on service labels
5. **Alert Type Routes**: Special handling for specific alert types

### Notification Channels

1. **Email**:
   - SMTP configuration with TLS
   - HTML and plain text templates
   - Different templates for each severity

2. **Slack**:
   - Channel-based routing
   - Rich formatting with colors and emojis
   - Severity-specific templates

3. **Webhooks**:
   - PagerDuty integration for critical alerts
   - Custom webhook support
   - Health check endpoints

### Inhibition Rules

- Critical alerts suppress warnings for the same service
- Service down alerts suppress related alerts
- Cluster-level issues suppress node-specific alerts
- Database down suppresses connection pool alerts

## Customization Guide

### Adding New Routes

Add new routes in the `route.routes` section of alertmanager-config.yaml:
```yaml
- match:
    team: your-team
  receiver: your-team-receiver
  group_wait: 1m
```

### Adding New Receivers

1. Add receiver configuration in alertmanager-config.yaml:
```yaml
- name: 'your-team-receiver'
  email_configs:
    - to: 'your-team@example.com'
  slack_configs:
    - channel: '#your-team-alerts'
```

2. Add necessary secrets to secrets.yaml

### Creating Custom Templates

Add new templates to notification-templates.yaml:
```yaml
{{ define "custom.template.name" }}
Your template content here
{{ end }}
```

## Best Practices

1. **Secrets Management**:
   - Never commit real credentials
   - Use different credentials per environment
   - Rotate credentials regularly
   - Consider using External Secrets Operator or similar

2. **Alert Routing**:
   - Keep routes simple and maintainable
   - Use labels effectively for routing
   - Test routing rules before production
   - Document team responsibilities

3. **Notification Templates**:
   - Include actionable information
   - Link to dashboards and runbooks
   - Keep messages concise
   - Use appropriate urgency indicators

4. **Testing**:
   - Test with amtool: `amtool config check alertmanager-config.yaml`
   - Send test alerts to verify routing
   - Validate webhook endpoints
   - Test during low-traffic periods

## Troubleshooting

### Common Issues

1. **Notifications not sending**:
   - Check secrets are properly mounted
   - Verify SMTP/Slack credentials
   - Check AlertManager logs
   - Test with amtool

2. **Wrong routing**:
   - Use AlertManager UI to inspect routing tree
   - Check label matching
   - Verify route order (first match wins)

3. **Template errors**:
   - Check template syntax
   - Verify all referenced fields exist
   - Test templates with sample data

### Debug Commands

```bash
# Check configuration syntax
kubectl exec -n monitoring alertmanager-0 -- amtool check-config /etc/alertmanager/alertmanager.yml

# View current alerts
kubectl exec -n monitoring alertmanager-0 -- amtool alert query

# Test routing
kubectl exec -n monitoring alertmanager-0 -- amtool config routes test --config.file=/etc/alertmanager/alertmanager.yml
```

## Integration with Prometheus

Ensure Prometheus is configured to send alerts to AlertManager:
```yaml
alerting:
  alertmanagers:
  - static_configs:
    - targets:
      - alertmanager.monitoring.svc.cluster.local:9093
```

## Monitoring AlertManager

Consider adding these Prometheus rules to monitor AlertManager itself:
- AlertManager configuration reload failures
- Notification failures
- Cluster status (if running multiple instances)
- Storage issues