# Istio Troubleshooting Guide

## Service Mesh Issues

### 1. Sidecar Injection Problems

**Symptoms:**
- Pods not getting sidecar containers
- Application containers failing to start
- Network connectivity issues

**Diagnostic Commands:**
```bash
# Check if namespace has sidecar injection enabled
kubectl get namespace -L istio-injection

# Check sidecar injection configuration
kubectl get mutatingwebhookconfiguration istio-sidecar-injector -o yaml

# Verify pod has sidecar
kubectl get pods -o jsonpath='{.items[*].spec.containers[*].name}' | grep istio-proxy

# Check sidecar injection logs
kubectl logs -n istio-system deployment/istiod | grep injection

# Force sidecar injection
kubectl patch deployment myapp -p '{"spec":{"template":{"metadata":{"annotations":{"sidecar.istio.io/inject":"true"}}}}}'
```

**Solutions:**
- Enable sidecar injection on namespace
- Check webhook configuration
- Verify pod annotations
- Restart deployment after enabling injection

### 2. Envoy Proxy Issues

**Symptoms:**
- HTTP 503 errors
- Connection refused errors
- Slow response times

**Diagnostic Commands:**
```bash
# Check Envoy configuration
kubectl exec -n myapp mypod-xxx -c istio-proxy -- curl -s localhost:15000/config_dump

# Check Envoy clusters
kubectl exec -n myapp mypod-xxx -c istio-proxy -- curl -s localhost:15000/clusters

# Check Envoy listeners
kubectl exec -n myapp mypod-xxx -c istio-proxy -- curl -s localhost:15000/listeners

# Check Envoy stats
kubectl exec -n myapp mypod-xxx -c istio-proxy -- curl -s localhost:15000/stats

# Check Envoy logs
kubectl logs -n myapp mypod-xxx -c istio-proxy | tail -50
```

**Solutions:**
- Check service configuration
- Verify DestinationRule settings
- Update Envoy configuration
- Restart affected pods

### 3. Service Discovery Issues

**Symptoms:**
- Services not discovered
- DNS resolution failures
- Load balancing not working

**Diagnostic Commands:**
```bash
# Check service endpoints
kubectl get endpoints -n myapp

# Check service registry
kubectl exec -n istio-system deployment/istiod -- pilot-discovery request GET /debug/registryz

# Check pilot configuration
kubectl logs -n istio-system deployment/istiod | grep -i discovery

# Test service resolution
kubectl exec -n myapp mypod-xxx -c istio-proxy -- nslookup myservice.myapp.svc.cluster.local
```

**Solutions:**
- Verify service labels and selectors
- Check service port configuration
- Update service discovery settings
- Restart istiod if needed

## mTLS Problems

### 1. Certificate Issues

**Symptoms:**
- TLS handshake failures
- Certificate validation errors
- Connection timeouts

**Diagnostic Commands:**
```bash
# Check certificate status
kubectl exec -n myapp mypod-xxx -c istio-proxy -- openssl s_client -connect myservice:443 -servername myservice

# Check certificate details
kubectl exec -n myapp mypod-xxx -c istio-proxy -- curl -s localhost:15000/certs

# Check mTLS configuration
kubectl get peerauthentication -A

# Check destination rule TLS settings
kubectl get destinationrule -A -o yaml | grep -A 10 tls

# Verify certificate rotation
kubectl logs -n istio-system deployment/istiod | grep cert
```

**Solutions:**
- Check certificate expiration
- Verify CA configuration
- Update PeerAuthentication policies
- Restart pods to refresh certificates

### 2. mTLS Policy Configuration

**Symptoms:**
- RBAC denies
- Authentication failures
- Mixed TLS/plaintext traffic issues

**Diagnostic Commands:**
```bash
# Check authentication policies
kubectl get peerauthentication -A

# Check authorization policies
kubectl get authorizationpolicy -A

# Test mTLS connectivity
kubectl exec -n myapp client-pod -- curl -v https://myservice.myapp.svc.cluster.local

# Check mTLS status
istioctl authn tls-check mypod-xxx.myapp.svc.cluster.local
```

**Example mTLS configuration:**
```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: myapp
spec:
  mtls:
    mode: STRICT
---
apiVersion: security.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: myservice
  namespace: myapp
spec:
  host: myservice.myapp.svc.cluster.local
  trafficPolicy:
    tls:
      mode: ISTIO_MUTUAL
```

**Solutions:**
- Configure proper authentication policies
- Update destination rules
- Check certificate chain
- Verify namespace configuration

## Traffic Routing Issues

### 1. Virtual Service Problems

**Symptoms:**
- Traffic not routing correctly
- 404 errors
- Load balancing issues

**Diagnostic Commands:**
```bash
# Check virtual service configuration
kubectl get virtualservice -A -o yaml

# Check gateway configuration
kubectl get gateway -A -o yaml

# Test routing
kubectl exec -n myapp client-pod -- curl -v -H "Host: myservice.example.com" http://istio-ingressgateway.istio-system.svc.cluster.local

# Check Envoy route configuration
kubectl exec -n myapp mypod-xxx -c istio-proxy -- curl -s localhost:15000/config_dump | jq '.configs[].dynamicRouteConfigs'
```

**Example virtual service:**
```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: myservice
  namespace: myapp
spec:
  hosts:
  - myservice.example.com
  gateways:
  - mygateway
  http:
  - match:
    - uri:
        prefix: /api
    route:
    - destination:
        host: myservice.myapp.svc.cluster.local
        port:
          number: 8080
```

**Solutions:**
- Fix virtual service configuration
- Check gateway settings
- Verify host matching
- Update routing rules

### 2. Destination Rule Issues

**Symptoms:**
- Load balancing not working
- Circuit breaker not triggering
- Connection pool exhaustion

**Diagnostic Commands:**
```bash
# Check destination rule configuration
kubectl get destinationrule -A -o yaml

# Check connection pool settings
kubectl exec -n myapp mypod-xxx -c istio-proxy -- curl -s localhost:15000/stats | grep upstream

# Test circuit breaker
kubectl exec -n myapp client-pod -- for i in {1..100}; do curl -s myservice.myapp.svc.cluster.local; done

# Check outlier detection
kubectl exec -n myapp mypod-xxx -c istio-proxy -- curl -s localhost:15000/stats | grep outlier
```

**Example destination rule:**
```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: myservice
  namespace: myapp
spec:
  host: myservice.myapp.svc.cluster.local
  trafficPolicy:
    loadBalancer:
      simple: LEAST_CONN
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http1MaxPendingRequests: 50
        maxRequestsPerConnection: 2
    circuitBreaker:
      consecutiveErrors: 5
      interval: 30s
      baseEjectionTime: 30s
```

**Solutions:**
- Update destination rule settings
- Check connection pool configuration
- Verify circuit breaker settings
- Monitor traffic patterns

## Sidecar Problems

### 1. Sidecar Container Issues

**Symptoms:**
- Sidecar container not starting
- Resource constraints
- Network policy conflicts

**Diagnostic Commands:**
```bash
# Check sidecar container status
kubectl get pods -n myapp -o jsonpath='{.items[*].status.containerStatuses[?(@.name=="istio-proxy")].ready}'

# Check sidecar logs
kubectl logs -n myapp mypod-xxx -c istio-proxy

# Check sidecar resource usage
kubectl top pods -n myapp --containers | grep istio-proxy

# Check sidecar configuration
kubectl get pod mypod-xxx -n myapp -o yaml | grep -A 20 istio-proxy
```

**Solutions:**
- Increase resource limits
- Check init container logs
- Verify network policies
- Update sidecar image

### 2. Sidecar Configuration Issues

**Symptoms:**
- Incorrect proxy configuration
- Missing routes
- Wrong upstream clusters

**Diagnostic Commands:**
```bash
# Check sidecar resource configuration
kubectl get sidecar -A -o yaml

# Check proxy configuration
kubectl exec -n myapp mypod-xxx -c istio-proxy -- pilot-agent request GET /config_dump

# Check proxy version
kubectl exec -n myapp mypod-xxx -c istio-proxy -- pilot-agent request GET /version

# Force configuration sync
kubectl exec -n myapp mypod-xxx -c istio-proxy -- pilot-agent request POST /quitquitquit
```

**Example sidecar configuration:**
```yaml
apiVersion: networking.istio.io/v1beta1
kind: Sidecar
metadata:
  name: myapp-sidecar
  namespace: myapp
spec:
  workloadSelector:
    labels:
      app: myapp
  egress:
  - hosts:
    - "./myservice.myapp.svc.cluster.local"
    - "istio-system/*"
  - hosts:
    - "./*.local"
    port:
      number: 443
      protocol: HTTPS
      name: https
```

**Solutions:**
- Update sidecar configuration
- Check workload selector
- Verify egress configuration
- Restart proxy container

## Gateway and Ingress Issues

### 1. Gateway Configuration Problems

**Symptoms:**
- External traffic not reaching services
- TLS termination issues
- Port binding problems

**Diagnostic Commands:**
```bash
# Check gateway status
kubectl get gateway -A

# Check ingress gateway logs
kubectl logs -n istio-system deployment/istio-ingressgateway

# Test gateway connectivity
kubectl exec -n istio-system deployment/istio-ingressgateway -- curl -v localhost:8080

# Check gateway configuration
kubectl get gateway mygateway -o yaml
```

**Example gateway configuration:**
```yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: mygateway
  namespace: myapp
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - myservice.example.com
  - port:
      number: 443
      name: https
      protocol: HTTPS
    tls:
      mode: SIMPLE
      credentialName: myservice-creds
    hosts:
    - myservice.example.com
```

**Solutions:**
- Check gateway selector
- Verify TLS certificates
- Update port configuration
- Check DNS resolution

### 2. Load Balancer Issues

**Symptoms:**
- LoadBalancer service not getting external IP
- Traffic not distributed evenly
- Health check failures

**Diagnostic Commands:**
```bash
# Check LoadBalancer service
kubectl get svc -n istio-system istio-ingressgateway

# Check endpoints
kubectl get endpoints -n istio-system istio-ingressgateway

# Test load balancing
for i in {1..10}; do curl -s http://myservice.example.com/health; done
```

**Solutions:**
- Check cloud provider configuration
- Verify service annotations
- Update load balancer settings
- Check health check configuration

## Troubleshooting Tools

### 1. Istioctl Commands

```bash
# Check configuration
istioctl analyze -n myapp

# Check proxy configuration
istioctl proxy-config cluster mypod-xxx.myapp

# Check routes
istioctl proxy-config route mypod-xxx.myapp

# Check listeners
istioctl proxy-config listener mypod-xxx.myapp

# Check endpoints
istioctl proxy-config endpoint mypod-xxx.myapp

# Check authentication
istioctl authn tls-check mypod-xxx.myapp.svc.cluster.local

# Generate configuration dump
istioctl bug-report --include-logs=false
```

### 2. Envoy Admin Interface

```bash
# Access Envoy admin interface
kubectl port-forward -n myapp mypod-xxx 15000:15000

# Then access http://localhost:15000 in browser or:
curl localhost:15000/help
curl localhost:15000/stats
curl localhost:15000/clusters
curl localhost:15000/listeners
curl localhost:15000/config_dump
```

### 3. Debugging with Pilot

```bash
# Check pilot debug endpoints
kubectl port-forward -n istio-system deployment/istiod 15010:15010

# Then access:
curl localhost:15010/debug/registryz
curl localhost:15010/debug/endpointz
curl localhost:15010/debug/configz
```

## Monitoring Istio Health

### Key Metrics to Monitor

```bash
# Control plane metrics
istio_requests_total
istio_request_duration_milliseconds
istio_request_bytes
istio_response_bytes

# Data plane metrics
envoy_cluster_upstream_rq_total
envoy_cluster_upstream_rq_time
envoy_http_downstream_rq_total
envoy_http_downstream_rq_time

# mTLS metrics
istio_requests_total{security_policy="mutual_tls"}
```

### Health Check Alerts

```yaml
groups:
  - name: istio-health
    rules:
    - alert: IstiodDown
      expr: up{job="istiod"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Istiod is down"
        description: "Istiod has been down for more than 5 minutes"

    - alert: HighErrorRate
      expr: rate(istio_requests_total{response_code!~"2.."}[5m]) > 0.1
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High error rate in service mesh"
        description: "Error rate is above 10% for 5 minutes"

    - alert: mTLSConnectionFailures
      expr: rate(istio_requests_total{security_policy="none"}[5m]) > 0
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "mTLS connection failures detected"
        description: "Non-mTLS traffic detected in strict mode"
```

## Common Error Messages and Solutions

| Error Message | Cause | Solution |
|---------------|-------|----------|
| "503 Service Unavailable" | No healthy upstream | Check service endpoints and health checks |
| "Connection refused" | Service not running | Verify service is running and reachable |
| "TLS handshake failed" | Certificate issues | Check certificates and mTLS configuration |
| "RBAC: access denied" | Authorization policy | Update authorization policies |
| "upstream connect error" | Network connectivity | Check network policies and routing |
| "no healthy upstream" | All backends failing | Check backend health and load balancer |
| "circuit breaker open" | Circuit breaker triggered | Check error rates and circuit breaker config |

## Emergency Procedures

### 1. Disable Istio Temporarily

```bash
# Disable sidecar injection
kubectl label namespace myapp istio-injection-

# Remove existing sidecars
kubectl rollout restart deployment/myapp -n myapp

# Verify pods are running without sidecars
kubectl get pods -n myapp -o jsonpath='{.items[*].spec.containers[*].name}' | grep -v istio-proxy
```

### 2. Istio Control Plane Recovery

```bash
# Check control plane status
kubectl get pods -n istio-system

# Restart control plane
kubectl rollout restart deployment/istiod -n istio-system

# Verify recovery
kubectl get pods -n istio-system -w
```

### 3. Complete Istio Reinstall

```bash
# Backup current configuration
kubectl get virtualservice,destinationrule,gateway,peerauthentication,authorizationpolicy -A -o yaml > istio-backup.yaml

# Uninstall Istio
istioctl uninstall --purge

# Reinstall Istio
istioctl install --set values.pilot.env.EXTERNAL_ISTIOD=false

# Restore configuration
kubectl apply -f istio-backup.yaml
```