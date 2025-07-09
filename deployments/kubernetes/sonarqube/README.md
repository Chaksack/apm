# SonarQube Kubernetes Deployment

This directory contains Kubernetes manifests for deploying SonarQube with PostgreSQL database for the GoFiber APM Solution.

## Components

### 1. SonarQube Server
- **Image**: `sonarqube:10.3-community`
- **Resources**: 3-6Gi memory, 1-2 CPU cores
- **Persistent Storage**: 20Gi total (data, logs, extensions)
- **Configuration**: Optimized for Go/GoFiber analysis

### 2. PostgreSQL Database
- **Image**: `postgres:14-alpine`
- **Resources**: 512Mi-1Gi memory, 250m-500m CPU
- **Persistent Storage**: 20Gi
- **Database**: `sonarqube`

### 3. Persistent Volumes
- **postgres-pvc**: 20Gi for PostgreSQL data
- **sonarqube-data-pvc**: 10Gi for SonarQube data
- **sonarqube-logs-pvc**: 5Gi for SonarQube logs
- **sonarqube-extensions-pvc**: 5Gi for SonarQube extensions

## Deployment Instructions

### Prerequisites

1. **Kubernetes cluster** with sufficient resources
2. **Storage class** named `standard` (adjust if different)
3. **Ingress controller** (optional, for external access)
4. **Prometheus Operator** (optional, for monitoring)

### Quick Deployment

```bash
# Deploy SonarQube with PostgreSQL
kubectl apply -f deployments/kubernetes/sonarqube/deployment.yaml

# Check deployment status
kubectl get pods -n sonarqube -w

# Check services
kubectl get svc -n sonarqube
```

### Access SonarQube

#### Port Forward (for local access)
```bash
kubectl port-forward -n sonarqube svc/sonarqube 9000:9000
```
Then access: http://localhost:9000

#### Ingress (for external access)
Update the ingress host in `deployment.yaml`:
```yaml
spec:
  rules:
  - host: sonarqube.yourdomain.com  # Change this
```

### Initial Setup

1. **Login**: Default credentials are `admin/admin`
2. **Change password**: You'll be prompted to change the default password
3. **Create project**: 
   - Project key: `apm-solution`
   - Project name: `APM Solution - GoFiber APM Stack`
4. **Generate token**: Create a token for CI/CD integration

## Configuration

### Environment Variables

The deployment includes several environment variables for optimal Go analysis:

```yaml
env:
- name: SONAR_JDBC_URL
  value: "jdbc:postgresql://postgres:5432/sonarqube"
- name: SONAR_WEB_JAVAOPTS
  value: "-Xmx2g -Xms1g"
- name: SONAR_CE_JAVAOPTS
  value: "-Xmx2g -Xms1g"
- name: SONAR_SEARCH_JAVAOPTS
  value: "-Xmx2g -Xms2g"
```

### Resource Requirements

**Minimum Requirements:**
- Memory: 4Gi total (2Gi for SonarQube, 512Mi for PostgreSQL)
- CPU: 1.5 cores total (1 core for SonarQube, 0.5 for PostgreSQL)
- Storage: 40Gi total

**Recommended for Production:**
- Memory: 8Gi total (6Gi for SonarQube, 2Gi for PostgreSQL)
- CPU: 3 cores total (2 cores for SonarQube, 1 for PostgreSQL)
- Storage: 100Gi total

## Monitoring and Observability

### Health Checks

SonarQube includes liveness and readiness probes:

```yaml
livenessProbe:
  httpGet:
    path: /api/system/status
    port: 9000
  initialDelaySeconds: 180
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /api/system/status
    port: 9000
  initialDelaySeconds: 60
  periodSeconds: 10
```

### Prometheus Monitoring

A ServiceMonitor is included for Prometheus scraping:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: sonarqube-metrics
  namespace: sonarqube
spec:
  endpoints:
  - port: http
    path: /api/monitoring/metrics
    interval: 30s
```

### Logging

SonarQube logs are stored in the persistent volume and can be accessed via:

```bash
kubectl logs -n sonarqube deployment/sonarqube -f
```

## Security

### Network Policies

Network policies are configured to:
- Allow ingress on port 9000 from specific namespaces
- Allow egress to PostgreSQL on port 5432
- Allow egress to internet on ports 80/443

### Secrets Management

Sensitive data is stored in Kubernetes secrets:
- **postgres-secret**: Database credentials
- **sonarqube-secret**: SonarQube admin credentials

### Pod Security

The deployment includes security contexts:
```yaml
securityContext:
  runAsUser: 1000
  runAsGroup: 1000
  fsGroup: 1000
```

## Troubleshooting

### Common Issues

1. **Pod stuck in Pending state**
   - Check PVC status: `kubectl get pvc -n sonarqube`
   - Verify storage class exists: `kubectl get storageclass`

2. **SonarQube not starting**
   - Check system requirements (vm.max_map_count)
   - Verify database connection: `kubectl logs -n sonarqube deployment/postgres`

3. **Out of memory errors**
   - Increase memory limits in deployment
   - Check actual resource usage: `kubectl top pods -n sonarqube`

### Debugging Commands

```bash
# Check pod status
kubectl get pods -n sonarqube -o wide

# Check logs
kubectl logs -n sonarqube deployment/sonarqube --tail=100

# Check database
kubectl exec -it -n sonarqube deployment/postgres -- psql -U sonarqube -d sonarqube

# Check persistent volumes
kubectl get pv,pvc -n sonarqube

# Check services and endpoints
kubectl get svc,ep -n sonarqube
```

## Scaling and Performance

### Horizontal Scaling

SonarQube Community Edition does not support horizontal scaling. For high availability, consider:
- **PostgreSQL**: Use managed database service or PostgreSQL cluster
- **Storage**: Use high-performance storage classes
- **Resources**: Vertical scaling with more CPU/memory

### Performance Tuning

1. **Java Heap Size**: Adjust based on project size
2. **Database**: Use SSD storage for PostgreSQL
3. **Elasticsearch**: Tune based on project count
4. **Network**: Use cluster networking for better performance

## Backup and Recovery

### Database Backup

```bash
# Create backup
kubectl exec -n sonarqube deployment/postgres -- pg_dump -U sonarqube sonarqube > sonarqube-backup.sql

# Restore backup
kubectl exec -i -n sonarqube deployment/postgres -- psql -U sonarqube sonarqube < sonarqube-backup.sql
```

### Volume Backup

Use your cloud provider's volume snapshot feature or backup tools like Velero.

## Integration with CI/CD

### GitHub Actions

The deployment works with the included GitHub Actions workflow:

```yaml
env:
  SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}
  SONAR_HOST_URL: ${{ secrets.SONAR_HOST_URL }}
```

### Local Development

Use the included `scripts/sonar-scan.sh` script:

```bash
# Set environment variables
export SONAR_HOST_URL="http://localhost:9000"
export SONAR_LOGIN="admin"
export SONAR_PASSWORD="your-password"

# Run scan
./scripts/sonar-scan.sh
```

## Maintenance

### Updates

1. **SonarQube**: Update image tag in deployment
2. **PostgreSQL**: Follow PostgreSQL upgrade procedures
3. **Plugins**: Update through SonarQube admin interface

### Cleanup

```bash
# Remove deployment
kubectl delete -f deployments/kubernetes/sonarqube/deployment.yaml

# Remove persistent volumes (optional)
kubectl delete pvc -n sonarqube --all
```

## Support

For issues related to:
- **SonarQube**: Check [SonarQube documentation](https://docs.sonarqube.org/)
- **Kubernetes**: Check [Kubernetes documentation](https://kubernetes.io/docs/)
- **PostgreSQL**: Check [PostgreSQL documentation](https://www.postgresql.org/docs/)

## Contributing

When contributing to this deployment:
1. Test changes in a development environment
2. Update documentation
3. Ensure security best practices
4. Test with actual Go/GoFiber projects