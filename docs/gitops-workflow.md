# GitOps Workflow with ArgoCD

This document outlines the GitOps workflow implementation for the APM stack using ArgoCD, including best practices, branch strategies, and promotion workflows.

## Overview

GitOps is a declarative approach to continuous deployment that uses Git as the single source of truth for infrastructure and application configuration. ArgoCD monitors Git repositories and automatically deploys changes to Kubernetes clusters.

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Development   │    │     Staging     │    │   Production    │
│   Environment   │    │   Environment   │    │   Environment   │
│                 │    │                 │    │                 │
│  ┌───────────┐  │    │  ┌───────────┐  │    │  ┌───────────┐  │
│  │  ArgoCD   │  │    │  │  ArgoCD   │  │    │  │  ArgoCD   │  │
│  │Application│  │    │  │Application│  │    │  │Application│  │
│  └───────────┘  │    │  └───────────┘  │    │  └───────────┘  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Git Repository                               │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │    main     │  │  staging    │  │   develop   │            │
│  │   branch    │  │   branch    │  │   branch    │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└─────────────────────────────────────────────────────────────────┘
```

## GitOps Best Practices

### 1. Repository Structure

```
apm/
├── deployments/
│   ├── k8s/                    # Kubernetes manifests
│   │   ├── base/               # Base configurations
│   │   └── overlays/           # Environment-specific overlays
│   ├── argocd/                 # ArgoCD configurations
│   │   ├── applications/       # Application definitions
│   │   ├── projects/           # Project definitions
│   │   └── install.yaml        # ArgoCD installation
│   └── monitoring/             # Monitoring stack
├── .argocd/
│   └── config.yaml            # ArgoCD configuration
└── docs/
    └── gitops-workflow.md     # This document
```

### 2. Configuration Management

- **Separation of Concerns**: Keep application code and deployment configurations in separate repositories or directories
- **Environment Isolation**: Use different branches or directories for different environments
- **Declarative Configuration**: All configurations should be declarative and version-controlled
- **Immutable Artifacts**: Use immutable container images with tags (avoid `latest`)

### 3. Security Best Practices

- **Least Privilege**: Grant minimal necessary permissions to ArgoCD
- **Secret Management**: Use external secret management systems (e.g., External Secrets Operator)
- **RBAC**: Implement proper role-based access control
- **Network Policies**: Restrict network access between components
- **Audit Logging**: Enable comprehensive audit logging

### 4. Monitoring and Observability

- **Application Health**: Monitor application health through ArgoCD
- **Sync Status**: Track sync status and failures
- **Drift Detection**: Monitor for configuration drift
- **Metrics**: Export ArgoCD metrics to Prometheus
- **Alerting**: Set up alerts for sync failures and health issues

## Branch Strategies

### 1. GitFlow Strategy

```
main branch (production)
├── staging branch
│   ├── feature/apm-monitoring
│   ├── feature/new-dashboard
│   └── hotfix/security-patch
└── develop branch
    ├── feature/metrics-collection
    └── feature/alerting-rules
```

**Workflow:**
1. Feature development on `develop` branch
2. Staging deployment from `staging` branch
3. Production deployment from `main` branch
4. Hotfixes directly to `main` with backport to `develop`

### 2. Environment Branches Strategy

```
environments/
├── development/
├── staging/
└── production/
```

**Workflow:**
1. Promote changes from `development` → `staging` → `production`
2. Each environment has its own branch
3. Pull requests for promotions between environments

### 3. Kustomize Overlays Strategy

```
deployments/k8s/
├── base/
│   ├── deployment.yaml
│   ├── service.yaml
│   └── kustomization.yaml
└── overlays/
    ├── development/
    ├── staging/
    └── production/
```

**Workflow:**
1. Base configurations in `base/`
2. Environment-specific overlays in `overlays/`
3. Single branch with environment-specific paths

## Promotion Workflows

### 1. Automated Promotion (Development)

```yaml
# ArgoCD Application with automated sync
syncPolicy:
  automated:
    prune: true
    selfHeal: true
    allowEmpty: false
```

**Process:**
1. Developer commits to `develop` branch
2. ArgoCD automatically syncs to development environment
3. Continuous integration tests run
4. Feedback provided to developer

### 2. Manual Promotion (Staging)

```yaml
# ArgoCD Application with manual sync
syncPolicy:
  automated:
    prune: true
    selfHeal: false  # Manual healing
  manual: true
```

**Process:**
1. Create pull request from `develop` to `staging`
2. Code review and approval
3. Merge to `staging` branch
4. Manual sync trigger in ArgoCD UI
5. Validation and testing

### 3. Controlled Promotion (Production)

```yaml
# ArgoCD Application with strict controls
syncPolicy:
  automated: false  # Fully manual
  syncOptions:
    - CreateNamespace=true
    - PrunePropagationPolicy=foreground
```

**Process:**
1. Create pull request from `staging` to `main`
2. Comprehensive review and approval
3. Merge to `main` branch
4. Manual sync with approval workflow
5. Rollback plan ready

## Deployment Workflows

### 1. Blue-Green Deployment

```yaml
# Blue-Green deployment strategy
spec:
  strategy:
    blueGreen:
      activeService: apm-active
      previewService: apm-preview
      autoPromotionEnabled: false
      scaleDownDelaySeconds: 30
      prePromotionAnalysis:
        templates:
          - templateName: success-rate
        args:
          - name: service-name
            value: apm-preview
```

### 2. Canary Deployment

```yaml
# Canary deployment strategy
spec:
  strategy:
    canary:
      maxSurge: 25%
      maxUnavailable: 25%
      steps:
        - setWeight: 20
        - pause: {}
        - setWeight: 50
        - pause: {duration: 10m}
        - setWeight: 100
```

### 3. Rolling Update

```yaml
# Rolling update strategy
spec:
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
```

## Sync Waves and Dependencies

### Sync Wave Configuration

```yaml
# Infrastructure components (databases, storage)
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "-10"

# Monitoring stack
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "-5"

# Applications
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "0"

# Ingress and networking
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "5"
```

### Health Checks

```yaml
# Custom health check
spec:
  health:
    - group: apps
      kind: Deployment
      check: |
        hs = {}
        if obj.status ~= nil then
          if obj.status.readyReplicas == obj.spec.replicas then
            hs.status = "Healthy"
          else
            hs.status = "Progressing"
          end
        end
        return hs
```

## Troubleshooting

### Common Issues

1. **Sync Failures**
   - Check resource quotas
   - Verify RBAC permissions
   - Review application logs

2. **Health Check Failures**
   - Validate custom health checks
   - Check pod readiness probes
   - Review resource limits

3. **Configuration Drift**
   - Enable drift detection
   - Review manual changes
   - Use resource hooks

### Debugging Commands

```bash
# Check application status
argocd app get apm-stack

# View sync status
argocd app sync apm-stack --dry-run

# Check application logs
argocd app logs apm-stack

# View application resources
argocd app resources apm-stack
```

## Security Considerations

### 1. Access Control

```yaml
# RBAC configuration
policy.csv: |
  p, role:admin, applications, *, */*, allow
  p, role:developer, applications, get, */*, allow
  p, role:developer, applications, sync, */*, allow
  p, role:readonly, applications, get, */*, allow
  g, platform-team, role:admin
  g, dev-team, role:developer
```

### 2. Secret Management

```yaml
# External Secrets Operator
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: apm-secrets
spec:
  refreshInterval: 15s
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: apm-secrets
    creationPolicy: Owner
  data:
    - secretKey: database-password
      remoteRef:
        key: secret/apm
        property: db-password
```

### 3. Network Security

```yaml
# Network policy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: argocd-network-policy
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: argocd-server
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
      ports:
        - protocol: TCP
          port: 8080
```

## Performance Optimization

### 1. Resource Allocation

```yaml
# Resource limits and requests
resources:
  limits:
    cpu: 2000m
    memory: 4Gi
  requests:
    cpu: 500m
    memory: 1Gi
```

### 2. Concurrency Settings

```yaml
# Controller settings
env:
  - name: ARGOCD_APPLICATION_CONTROLLER_REPLICAS
    value: "2"
  - name: ARGOCD_SERVER_REPLICAS
    value: "2"
  - name: ARGOCD_REPO_SERVER_REPLICAS
    value: "2"
```

### 3. Caching

```yaml
# Repository caching
server:
  config:
    repositories.cache.expiration: 24h
```

## Monitoring and Alerting

### 1. ArgoCD Metrics

```yaml
# ServiceMonitor for Prometheus
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: argocd-metrics
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: argocd-server-metrics
  endpoints:
    - port: metrics
      interval: 30s
      path: /metrics
```

### 2. Alerting Rules

```yaml
# PrometheusRule for alerts
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: argocd-alerts
spec:
  groups:
    - name: argocd
      rules:
        - alert: ArgoCDSyncFailed
          expr: argocd_app_sync_total{sync_status="Failed"} > 0
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "ArgoCD sync failed"
            description: "Application {{ $labels.name }} sync failed"
```

## Backup and Disaster Recovery

### 1. Backup Strategy

```yaml
# Backup configuration
backup:
  enabled: true
  schedule: "0 2 * * *"
  retention: 30d
  storage:
    type: s3
    bucket: argocd-backups
```

### 2. Disaster Recovery

1. **Backup Requirements**
   - Git repository access
   - Kubernetes cluster access
   - ArgoCD configuration backup

2. **Recovery Process**
   - Restore ArgoCD installation
   - Restore application configurations
   - Trigger sync from Git repository

## Best Practices Summary

1. **Use Infrastructure as Code** for all configurations
2. **Implement proper RBAC** and security controls
3. **Monitor sync status** and application health
4. **Use sync waves** for dependency management
5. **Implement automated testing** in CI/CD pipeline
6. **Maintain separation** between environments
7. **Document procedures** and runbooks
8. **Regular backup** and disaster recovery testing
9. **Monitor resource usage** and optimize performance
10. **Keep configurations minimal** and secure

## Getting Started

1. Install ArgoCD using the provided manifests
2. Configure Git repository access
3. Create AppProject for your applications
4. Define Application manifests
5. Set up monitoring and alerting
6. Test promotion workflows
7. Implement backup procedures

For more information, refer to the official ArgoCD documentation and best practices guides.