# Grafana Kubernetes Deployment

This directory contains Kubernetes manifests for deploying Grafana with pre-configured datasources for Prometheus, Loki, and Jaeger.

## Prerequisites

1. Kubernetes cluster with `monitoring` namespace created
2. Prometheus, Loki, and Jaeger already deployed in the cluster

## Files

- `configmap.yaml` - Grafana configuration and datasource provisioning
- `deployment.yaml` - Grafana deployment with resource limits and health checks
- `service.yaml` - ClusterIP service (with optional LoadBalancer configuration)
- `secret.yaml` - Admin password secret
- `kustomization.yaml` - Kustomize configuration for easy deployment

## Deployment

1. **Create the monitoring namespace** (if not exists):
   ```bash
   kubectl create namespace monitoring
   ```

2. **Update the admin password** in `secret.yaml`:
   ```bash
   # Generate a strong password
   openssl rand -base64 32
   ```

3. **Deploy using kubectl**:
   ```bash
   kubectl apply -k deployments/kubernetes/grafana/
   ```

   Or deploy individual files:
   ```bash
   kubectl apply -f deployments/kubernetes/grafana/secret.yaml
   kubectl apply -f deployments/kubernetes/grafana/configmap.yaml
   kubectl apply -f deployments/kubernetes/grafana/deployment.yaml
   kubectl apply -f deployments/kubernetes/grafana/service.yaml
   ```

4. **Access Grafana**:
   
   For local access via port-forward:
   ```bash
   kubectl port-forward -n monitoring svc/grafana 3000:3000
   ```
   Then open http://localhost:3000

   For external access, uncomment the LoadBalancer service in `service.yaml` and apply it.

## Configuration Details

### Datasources

The following datasources are pre-configured:
- **Prometheus** (default) - http://prometheus:9090
- **Loki** - http://loki:3100
- **Jaeger** - http://jaeger-query:16686

### Security

- Non-root user (UID 472)
- Admin credentials stored in Kubernetes secret
- Anonymous access disabled
- Login form enabled

### Resource Limits

- Requests: 100m CPU, 128Mi memory
- Limits: 500m CPU, 512Mi memory

## Customization

To customize the deployment:
1. Modify `configmap.yaml` for Grafana settings or datasource URLs
2. Adjust resource limits in `deployment.yaml`
3. Change service type in `service.yaml` for different access methods
4. Add persistent volume for data persistence (currently using emptyDir)

## Monitoring

Check deployment status:
```bash
kubectl get all -n monitoring -l app=grafana
kubectl logs -n monitoring -l app=grafana
```