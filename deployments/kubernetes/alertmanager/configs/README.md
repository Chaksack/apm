# AlertManager Configuration Guide

This directory contains advanced AlertManager routing configurations for comprehensive alert management.

## Configuration Files

### 1. routing-rules.yaml
Defines how alerts are routed to different teams and receivers based on:
- **Team-based routing**: Routes alerts to specific teams (platform, backend, frontend, security)
- **Severity-based routing**: Critical alerts get immediate attention with shorter intervals
- **Time-based routing**: Different handling for business hours vs. non-business hours
- **Service-based routing**: Specific routing for critical services (payment, authentication)
- **Environment-based routing**: Different handling for production, staging, and development

Key features:
- Business hours definitions for intelligent routing
- Cascading routes for complex decision trees
- Time intervals for maintenance windows
- Group-by configurations to reduce alert noise

### 2. inhibition-rules.yaml
Implements alert suppression to reduce noise and focus on root causes:
- **Dependency-based inhibition**: Suppresses downstream alerts when upstream failures occur
- **Infrastructure inhibition**: Node/cluster failures suppress service-level alerts
- **Maintenance window support**: Automatically suppresses non-critical alerts during maintenance
- **Cascading failure prevention**: Prevents alert storms during major incidents

Key patterns:
- Platform-wide issues suppress service-specific alerts
- Critical alerts suppress warnings for the same component
- Network issues suppress communication alerts
- Database master failures suppress replica alerts

### 3. silences.yaml
Pre-defined silence templates for common scenarios:
- **Planned maintenance windows**: Database, infrastructure, and application deployments
- **Development environment silences**: Reduces noise from non-production environments
- **Testing silences**: Load testing, chaos engineering, security testing
- **Known issues**: Temporary silences for acknowledged problems
- **Recurring silences**: Automated silencing for regular operations

Features:
- Recurring silence schedules for regular maintenance
- Auto-silence rules based on alert frequency
- Emergency silence templates for incident response

### 4. receivers.yaml
Comprehensive notification channel configurations:
- **PagerDuty integration**: Multiple services for different teams and severities
- **Webhook receivers**: Custom integrations with internal systems
- **Email lists by team**: Targeted email notifications
- **Slack channels**: Team-specific Slack notifications

Integration types:
- Critical alerts → PagerDuty
- Team notifications → Slack channels
- Aggregated warnings → Email digests
- Custom webhooks → Internal systems (SIEM, incident management, monitoring)

## Usage

### Merging Configurations
In production, merge these configurations into a single AlertManager config:

```yaml
global:
  # Global settings from receivers.yaml

route:
  # Content from routing-rules.yaml

receivers:
  # Content from receivers.yaml

inhibit_rules:
  # Content from inhibition-rules.yaml
```

### Applying Silences
Use the AlertManager API to apply silence templates:

```bash
# Apply a maintenance silence
curl -X POST http://alertmanager:9093/api/v1/silences \
  -H "Content-Type: application/json" \
  -d @silence-template.json
```

### Time Intervals
Configure your timezone in the time_intervals section:
```yaml
location: 'America/New_York'  # Change to your timezone
```

### Secret Management
Store sensitive data in Kubernetes secrets:
```bash
kubectl create secret generic alertmanager-secrets \
  --from-literal=smtp-password=YOUR_PASSWORD \
  --from-literal=pagerduty-key=YOUR_KEY \
  --from-literal=slack-webhook=YOUR_WEBHOOK_URL
```

## Best Practices

1. **Test routing rules** before deploying to production
2. **Keep inhibition rules simple** to avoid accidentally suppressing critical alerts
3. **Review silences regularly** to ensure they're still needed
4. **Monitor receiver failures** and have fallback channels
5. **Document on-call procedures** for each receiver type
6. **Use descriptive alert names** that work well with routing patterns
7. **Implement gradual rollout** when changing routing rules

## Validation

Validate configuration before applying:
```bash
amtool config check alertmanager.yaml
```

Test routing for specific alerts:
```bash
amtool config routes test alertmanager.yaml \
  --labels="team=backend,severity=critical,service=api"
```

## Monitoring AlertManager

Monitor AlertManager itself:
- Alert delivery success/failure rates
- Notification delays
- Silence and inhibition effectiveness
- Receiver health status