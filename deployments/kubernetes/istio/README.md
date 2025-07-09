# Istio Sidecar Injection Configuration

This directory contains configuration files for enabling automatic Istio sidecar injection in the APM system.

## Prerequisites

- Kubernetes cluster (1.20+)
- Istio installed (1.16+)
- kubectl configured to access your cluster

## Installation Instructions

### 1. Enable Istio Sidecar Injection

There are two ways to enable automatic sidecar injection:

#### Option A: Create namespace with labels (for new namespaces)
```bash
kubectl apply -f namespace-labels.yaml
```

#### Option B: Label existing namespace
```bash
kubectl label namespace apm-system istio-injection=enabled --overwrite
```

### 2. Apply Sidecar Configuration

Apply the sidecar configuration to control traffic flow:

```bash
kubectl apply -f sidecar-config.yaml
```

### 3. Restart Existing Pods

For existing deployments, restart pods to inject sidecars:

```bash
kubectl rollout restart deployment -n apm-system
```

## Verification Steps

### 1. Verify Namespace Label

Check that the namespace has the injection label:

```bash
kubectl get namespace apm-system --show-labels
```

Expected output should include: `istio-injection=enabled`

### 2. Verify Sidecar Injection

Check if pods have sidecars injected:

```bash
kubectl get pods -n apm-system -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.containers[*].name}{"\n"}{end}'
```

Each pod should have an `istio-proxy` container alongside application containers.

### 3. Verify Sidecar Configuration

Check that the Sidecar resource is created:

```bash
kubectl get sidecar -n apm-system
kubectl describe sidecar default -n apm-system
```

### 4. Check Proxy Status

Verify proxy configuration and health:

```bash
# Check proxy configuration
kubectl exec -n apm-system deployment/[your-deployment] -c istio-proxy -- curl -s localhost:15000/config_dump

# Check proxy stats
kubectl exec -n apm-system deployment/[your-deployment] -c istio-proxy -- curl -s localhost:15000/stats/prometheus
```

## Troubleshooting Guide

### Sidecars Not Being Injected

1. **Check namespace label:**
   ```bash
   kubectl get ns apm-system -o jsonpath='{.metadata.labels.istio-injection}'
   ```
   Should return: `enabled`

2. **Check MutatingWebhookConfiguration:**
   ```bash
   kubectl get mutatingwebhookconfiguration istio-sidecar-injector -o yaml
   ```

3. **Check istio-system pods:**
   ```bash
   kubectl get pods -n istio-system
   ```
   Ensure all Istio components are running.

### High Resource Usage

1. **Check sidecar resource limits:**
   ```bash
   kubectl describe pod [pod-name] -n apm-system | grep -A 5 istio-proxy
   ```

2. **Adjust resource requests/limits in deployment:**
   ```yaml
   annotations:
     sidecar.istio.io/proxyCPULimit: "200m"
     sidecar.istio.io/proxyMemoryLimit: "128Mi"
     sidecar.istio.io/proxyCPU: "100m"
     sidecar.istio.io/proxyMemory: "64Mi"
   ```

### Traffic Not Flowing

1. **Check Sidecar configuration:**
   ```bash
   kubectl describe sidecar default -n apm-system
   ```

2. **Check DestinationRules and VirtualServices:**
   ```bash
   kubectl get destinationrule,virtualservice -n apm-system
   ```

3. **Enable debug logging:**
   ```bash
   kubectl exec -n apm-system [pod-name] -c istio-proxy -- curl -X POST "localhost:15000/logging?level=debug"
   ```

### Disable Sidecar for Specific Pods

Add annotation to pod template:
```yaml
metadata:
  annotations:
    sidecar.istio.io/inject: "false"
```

## Performance Optimization

### 1. Limit Telemetry

Reduce telemetry overhead by configuring sampling:
```yaml
metadata:
  annotations:
    sidecar.istio.io/statsInclusionRegexps: "cluster.outbound|http.*|grpc.*"
```

### 2. Disable Unused Features

```yaml
metadata:
  annotations:
    # Disable tracing
    sidecar.istio.io/inject: "true"
    sidecar.istio.io/traceSampling: "0.0"
```

### 3. Configure Concurrency

```yaml
metadata:
  annotations:
    # Set proxy concurrency based on CPU cores
    sidecar.istio.io/proxyConcurrency: "2"
```

## Additional Resources

- [Istio Sidecar Injection Documentation](https://istio.io/latest/docs/setup/additional-setup/sidecar-injection/)
- [Sidecar Configuration Reference](https://istio.io/latest/docs/reference/config/networking/sidecar/)
- [Performance Best Practices](https://istio.io/latest/docs/ops/best-practices/performance-and-scalability/)