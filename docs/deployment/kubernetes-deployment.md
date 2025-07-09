# Kubernetes Production Deployment

This guide covers deploying the APM system to production Kubernetes clusters with high availability, scalability, and reliability.

## Prerequisites

### Required Tools
- **kubectl** (version 1.25+) configured for your cluster
- **Helm** (version 3.8+) for package management
- **Docker** for building images
- **Kubernetes cluster** (version 1.25+) with:
  - At least 3 worker nodes
  - 16GB+ RAM per node
  - 100GB+ storage
  - LoadBalancer or Ingress controller

### Cluster Requirements
```bash
# Verify cluster access
kubectl cluster-info

# Check node resources
kubectl get nodes -o wide

# Verify storage classes
kubectl get storageclass
```

## 1. Namespace Setup

### Create Namespace
```bash
# Create APM namespace
kubectl create namespace apm-system

# Set default namespace
kubectl config set-context --current --namespace=apm-system

# Create service account
kubectl create serviceaccount apm-service-account
```

### RBAC Configuration
```yaml
# rbac.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: apm-cluster-role
rules:
- apiGroups: [""]
  resources: ["pods", "services", "endpoints", "secrets", "configmaps"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["monitoring.coreos.com"]
  resources: ["servicemonitors", "prometheusrules"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: apm-cluster-role-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: apm-cluster-role
subjects:
- kind: ServiceAccount
  name: apm-service-account
  namespace: apm-system
```

Apply RBAC configuration:
```bash
kubectl apply -f rbac.yaml
```

## 2. Helm Chart Configuration

### Install Helm Chart Repository
```bash
# Add and update Helm repositories
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo add jaegertracing https://jaegertracing.github.io/helm-charts
helm repo update
```

### Create Values File
Create `values-production.yaml`:

```yaml
# values-production.yaml
global:
  storageClass: "fast-ssd"
  imageRegistry: "your-registry.com"
  imagePullSecrets:
    - name: registry-secret

# Database Configuration
postgresql:
  enabled: true
  auth:
    username: apm_user
    password: "your-secure-password"
    database: apm_db
  primary:
    persistence:
      enabled: true
      size: 100Gi
      storageClass: "fast-ssd"
    resources:
      limits:
        memory: "4Gi"
        cpu: "2"
      requests:
        memory: "2Gi"
        cpu: "1"
  readReplicas:
    replicaCount: 2
    persistence:
      enabled: true
      size: 100Gi
    resources:
      limits:
        memory: "2Gi"
        cpu: "1"
      requests:
        memory: "1Gi"
        cpu: "500m"

# Redis Configuration
redis:
  enabled: true
  auth:
    enabled: true
    password: "your-redis-password"
  master:
    persistence:
      enabled: true
      size: 20Gi
    resources:
      limits:
        memory: "2Gi"
        cpu: "1"
      requests:
        memory: "1Gi"
        cpu: "500m"
  replica:
    replicaCount: 2
    persistence:
      enabled: true
      size: 20Gi
    resources:
      limits:
        memory: "1Gi"
        cpu: "500m"
      requests:
        memory: "512Mi"
        cpu: "250m"

# API Server Configuration
api:
  replicaCount: 3
  image:
    repository: your-registry.com/apm-api
    tag: "v1.0.0"
    pullPolicy: IfNotPresent
  
  resources:
    limits:
      memory: "2Gi"
      cpu: "1"
    requests:
      memory: "1Gi"
      cpu: "500m"
  
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
    targetMemoryUtilizationPercentage: 80
  
  service:
    type: ClusterIP
    port: 8080
  
  env:
    - name: DB_HOST
      valueFrom:
        secretKeyRef:
          name: apm-secrets
          key: db-host
    - name: REDIS_HOST
      valueFrom:
        secretKeyRef:
          name: apm-secrets
          key: redis-host
    - name: LOG_LEVEL
      value: "info"

# Frontend Configuration
frontend:
  replicaCount: 2
  image:
    repository: your-registry.com/apm-frontend
    tag: "v1.0.0"
    pullPolicy: IfNotPresent
  
  resources:
    limits:
      memory: "512Mi"
      cpu: "500m"
    requests:
      memory: "256Mi"
      cpu: "250m"
  
  service:
    type: ClusterIP
    port: 80
  
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 5
    targetCPUUtilizationPercentage: 70

# Ingress Configuration
ingress:
  enabled: true
  className: "nginx"
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
  hosts:
    - host: apm.yourdomain.com
      paths:
        - path: /
          pathType: Prefix
          service:
            name: apm-frontend
            port: 80
        - path: /api
          pathType: Prefix
          service:
            name: apm-api
            port: 8080
  tls:
    - secretName: apm-tls-secret
      hosts:
        - apm.yourdomain.com

# Monitoring Configuration
monitoring:
  prometheus:
    enabled: true
    persistence:
      enabled: true
      size: 50Gi
    resources:
      limits:
        memory: "4Gi"
        cpu: "2"
      requests:
        memory: "2Gi"
        cpu: "1"
  
  grafana:
    enabled: true
    persistence:
      enabled: true
      size: 10Gi
    resources:
      limits:
        memory: "1Gi"
        cpu: "500m"
      requests:
        memory: "512Mi"
        cpu: "250m"
  
  jaeger:
    enabled: true
    storage:
      type: elasticsearch
      elasticsearch:
        nodeCount: 3
        resources:
          limits:
            memory: "2Gi"
            cpu: "1"
          requests:
            memory: "1Gi"
            cpu: "500m"

# Security Configuration
security:
  podSecurityPolicy:
    enabled: true
  networkPolicy:
    enabled: true
  serviceMonitor:
    enabled: true
```

## 3. Secrets Management

### Create Secrets
```bash
# Create database secret
kubectl create secret generic apm-db-secret \
  --from-literal=username=apm_user \
  --from-literal=password=your-secure-password \
  --from-literal=host=postgresql.apm-system.svc.cluster.local

# Create Redis secret
kubectl create secret generic apm-redis-secret \
  --from-literal=password=your-redis-password \
  --from-literal=host=redis.apm-system.svc.cluster.local

# Create JWT secret
kubectl create secret generic apm-jwt-secret \
  --from-literal=secret=your-jwt-secret-key

# Create Docker registry secret
kubectl create secret docker-registry registry-secret \
  --docker-server=your-registry.com \
  --docker-username=your-username \
  --docker-password=your-password \
  --docker-email=your-email@domain.com
```

### Seal Secrets (Recommended)
```bash
# Install sealed-secrets controller
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.18.0/controller.yaml

# Create sealed secret
echo -n your-secure-password | kubectl create secret generic apm-sealed-secret \
  --dry-run=client --from-file=password=/dev/stdin -o yaml | \
  kubeseal -o yaml > sealed-secret.yaml

kubectl apply -f sealed-secret.yaml
```

## 4. Environment-Specific Configurations

### Development Environment
```yaml
# values-dev.yaml
global:
  environment: development

api:
  replicaCount: 1
  resources:
    limits:
      memory: "1Gi"
      cpu: "500m"
    requests:
      memory: "512Mi"
      cpu: "250m"

postgresql:
  primary:
    persistence:
      size: 10Gi
  readReplicas:
    replicaCount: 0

redis:
  replica:
    replicaCount: 0

monitoring:
  prometheus:
    persistence:
      size: 10Gi
```

### Staging Environment
```yaml
# values-staging.yaml
global:
  environment: staging

api:
  replicaCount: 2
  resources:
    limits:
      memory: "1.5Gi"
      cpu: "750m"
    requests:
      memory: "750Mi"
      cpu: "375m"

postgresql:
  primary:
    persistence:
      size: 50Gi
  readReplicas:
    replicaCount: 1

redis:
  replica:
    replicaCount: 1
```

### Production Environment
Use the full `values-production.yaml` configuration above.

## 5. Deployment Commands

### Deploy with Helm
```bash
# Create namespace
kubectl create namespace apm-system

# Deploy to development
helm install apm-dev ./helm/apm-chart \
  --namespace apm-system \
  --values values-dev.yaml

# Deploy to staging
helm install apm-staging ./helm/apm-chart \
  --namespace apm-staging \
  --values values-staging.yaml

# Deploy to production
helm install apm-prod ./helm/apm-chart \
  --namespace apm-production \
  --values values-production.yaml
```

### Upgrade Deployment
```bash
# Upgrade with new values
helm upgrade apm-prod ./helm/apm-chart \
  --namespace apm-production \
  --values values-production.yaml

# Rollback if needed
helm rollback apm-prod 1
```

## 6. Scaling and Resource Management

### Horizontal Pod Autoscaler
```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: apm-api-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: apm-api
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
      - type: Percent
        value: 100
        periodSeconds: 15
```

### Vertical Pod Autoscaler
```yaml
# vpa.yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: apm-api-vpa
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: apm-api
  updatePolicy:
    updateMode: "Auto"
  resourcePolicy:
    containerPolicies:
    - containerName: apm-api
      maxAllowed:
        cpu: 2
        memory: 4Gi
      minAllowed:
        cpu: 100m
        memory: 128Mi
```

### Cluster Autoscaler
```yaml
# cluster-autoscaler.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cluster-autoscaler
  namespace: kube-system
spec:
  template:
    spec:
      containers:
      - image: k8s.gcr.io/autoscaling/cluster-autoscaler:v1.21.0
        name: cluster-autoscaler
        resources:
          limits:
            cpu: 100m
            memory: 300Mi
          requests:
            cpu: 100m
            memory: 300Mi
        command:
        - ./cluster-autoscaler
        - --v=4
        - --stderrthreshold=info
        - --cloud-provider=aws
        - --skip-nodes-with-local-storage=false
        - --expander=least-waste
        - --node-group-auto-discovery=asg:tag=k8s.io/cluster-autoscaler/enabled,k8s.io/cluster-autoscaler/apm-cluster
```

## 7. Monitoring and Observability

### ServiceMonitor Configuration
```yaml
# servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: apm-metrics
  labels:
    app: apm
spec:
  selector:
    matchLabels:
      app: apm-api
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

### PrometheusRule Configuration
```yaml
# prometheus-rules.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: apm-rules
spec:
  groups:
  - name: apm.rules
    rules:
    - alert: APMHighCPUUsage
      expr: rate(container_cpu_usage_seconds_total{pod=~"apm-api-.*"}[5m]) > 0.8
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High CPU usage detected"
        description: "APM API pod {{ $labels.pod }} CPU usage is above 80%"
    
    - alert: APMHighMemoryUsage
      expr: container_memory_usage_bytes{pod=~"apm-api-.*"} / container_spec_memory_limit_bytes > 0.9
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "High memory usage detected"
        description: "APM API pod {{ $labels.pod }} memory usage is above 90%"
```

## 8. Health Checks and Readiness

### Liveness and Readiness Probes
```yaml
# deployment-with-probes.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: apm-api
spec:
  template:
    spec:
      containers:
      - name: apm-api
        image: your-registry.com/apm-api:v1.0.0
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
        startupProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 30
```

## 9. Backup and Disaster Recovery

### Database Backup
```bash
# Create backup job
kubectl create job --from=cronjob/postgresql-backup manual-backup-$(date +%Y%m%d%H%M%S)

# Backup script
#!/bin/bash
kubectl exec -it postgresql-0 -- pg_dump -U apm_user apm_db > backup-$(date +%Y%m%d).sql
```

### Disaster Recovery Plan
```yaml
# disaster-recovery-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: apm-backup
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: your-registry.com/backup-tool:latest
            env:
            - name: DB_HOST
              value: postgresql.apm-system.svc.cluster.local
            - name: BACKUP_STORAGE
              value: s3://your-backup-bucket/apm-backups
            command:
            - /bin/bash
            - -c
            - |
              pg_dump -h $DB_HOST -U apm_user apm_db | gzip > /tmp/backup-$(date +%Y%m%d).sql.gz
              aws s3 cp /tmp/backup-$(date +%Y%m%d).sql.gz $BACKUP_STORAGE/
          restartPolicy: OnFailure
```

## 10. Troubleshooting Commands

### Common Debugging Commands
```bash
# Check pod status
kubectl get pods -o wide

# View pod logs
kubectl logs -f deployment/apm-api

# Describe pod issues
kubectl describe pod apm-api-xxx

# Check events
kubectl get events --sort-by=.metadata.creationTimestamp

# Port forward for debugging
kubectl port-forward service/apm-api 8080:8080

# Execute commands in pod
kubectl exec -it apm-api-xxx -- /bin/bash

# Check resource usage
kubectl top pods
kubectl top nodes

# View service endpoints
kubectl get endpoints
```

### Performance Troubleshooting
```bash
# Check HPA status
kubectl get hpa

# View cluster autoscaler logs
kubectl logs -f deployment/cluster-autoscaler -n kube-system

# Check resource quotas
kubectl describe resourcequota

# Monitor real-time metrics
kubectl get --raw /metrics | grep apm
```

---

**Deployment Time**: 30-45 minutes  
**Complexity**: Advanced  
**Prerequisites**: Kubernetes cluster, Helm, Docker registry  
**Next Steps**: See `security-hardening.md` for security configuration