# Service Down Runbook

## Alert Definition
- **Trigger**: Service health check failing for 2 minutes
- **Severity**: Critical
- **Team**: Platform Engineering + Service Owner

## Service Recovery Steps

### 1. Initial Verification
```bash
# Check service status
kubectl get pods -n <namespace> -l app=<service>
kubectl get deployment <service> -n <namespace>

# Check recent events
kubectl describe pods -n <namespace> -l app=<service>
kubectl events -n <namespace> --for pod/<pod-name>

# Verify endpoints
kubectl get endpoints <service> -n <namespace>

# Test service connectivity
curl -v http://<service>.<namespace>.svc.cluster.local/health
```

### 2. Quick Diagnostics
```bash
# Check if pods are crash-looping
kubectl get pods -n <namespace> -l app=<service> -w

# Get pod logs
kubectl logs -n <namespace> <pod-name> --previous
kubectl logs -n <namespace> <pod-name> --tail=100

# Check resource usage
kubectl top pods -n <namespace> -l app=<service>
kubectl describe nodes | grep -A 5 "Allocated resources"
```

## Recovery Actions

### Immediate Recovery (0-5 minutes)

#### 1. Restart Pods
```bash
# Delete pods to force restart
kubectl delete pods -n <namespace> -l app=<service>

# Or rollout restart
kubectl rollout restart deployment/<service> -n <namespace>

# Monitor restart
kubectl rollout status deployment/<service> -n <namespace>
```

#### 2. Scale Operations
```bash
# Scale up if partial failure
kubectl scale deployment <service> -n <namespace> --replicas=5

# Scale down and up to force recreation
kubectl scale deployment <service> -n <namespace> --replicas=0
kubectl scale deployment <service> -n <namespace> --replicas=3
```

#### 3. Emergency Traffic Redirect
```bash
# Update service selector to redirect traffic
kubectl patch service <service> -n <namespace> -p '{"spec":{"selector":{"version":"stable"}}}'

# Or route to backup service
kubectl patch ingress <ingress> -n <namespace> --type=json -p='[{"op": "replace", "path": "/spec/rules/0/http/paths/0/backend/service/name", "value":"<backup-service>"}]'
```

### Advanced Recovery (5-15 minutes)

#### 1. Configuration Issues
```bash
# Check ConfigMaps and Secrets
kubectl get configmap -n <namespace> -l app=<service>
kubectl describe configmap <config> -n <namespace>

# Rollback ConfigMap if recently changed
kubectl rollout history deployment/<service> -n <namespace>
kubectl rollout undo deployment/<service> -n <namespace>

# Verify environment variables
kubectl exec -n <namespace> <pod> -- env | grep -E "DB_|API_|SERVICE_"
```

#### 2. Resource Constraints
```bash
# Check for resource limits
kubectl describe deployment <service> -n <namespace> | grep -A 5 "Limits"

# Temporarily increase resources
kubectl patch deployment <service> -n <namespace> --type='json' -p='[
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/memory", "value":"4Gi"},
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/cpu", "value":"2"}
]'
```

#### 3. Node Issues
```bash
# Check node status
kubectl get nodes
kubectl describe node <node-name> | grep -E "Conditions|Allocated"

# Cordon problematic node
kubectl cordon <node-name>

# Evacuate pods from node
kubectl drain <node-name> --ignore-daemonsets --delete-emptydir-data
```

## Dependency Checks

### 1. Database Connectivity
```bash
# Test database connection
kubectl run -it --rm debug --image=postgres:13 --restart=Never -- psql -h <db-host> -U <user> -d <database> -c "SELECT 1"

# Check database pods (if in-cluster)
kubectl get pods -n <db-namespace> -l app=postgresql
kubectl logs -n <db-namespace> <postgres-pod> --tail=50
```

### 2. Message Queue
```bash
# Check RabbitMQ/Kafka status
kubectl exec -n <namespace> <pod> -- rabbitmqctl status
kubectl exec -n <namespace> <pod> -- kafka-topics --bootstrap-server localhost:9092 --list

# Check queue depth
kubectl exec -n <namespace> <pod> -- rabbitmqctl list_queues
```

### 3. External Services
```bash
# Test external API connectivity
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- curl -v https://api.external-service.com/health

# Check DNS resolution
kubectl run -it --rm debug --image=busybox --restart=Never -- nslookup api.external-service.com

# Verify service discovery
kubectl run -it --rm debug --image=tutum/dnsutils --restart=Never -- dig +short <service>.<namespace>.svc.cluster.local
```

## Rollback Procedures

### 1. Deployment Rollback
```bash
# View rollout history
kubectl rollout history deployment/<service> -n <namespace>

# Rollback to previous version
kubectl rollout undo deployment/<service> -n <namespace>

# Rollback to specific revision
kubectl rollout undo deployment/<service> -n <namespace> --to-revision=<number>

# Verify rollback
kubectl rollout status deployment/<service> -n <namespace>
kubectl get pods -n <namespace> -l app=<service> -o jsonpath='{.items[*].spec.containers[*].image}'
```

### 2. Configuration Rollback
```bash
# Backup current config
kubectl get configmap <config> -n <namespace> -o yaml > configmap-backup.yaml

# Apply previous version
kubectl apply -f configmap-previous.yaml

# Restart pods to pick up config
kubectl rollout restart deployment/<service> -n <namespace>
```

### 3. Database Rollback
```sql
-- Check for recent schema changes
SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 10;

-- Rollback migration (example)
BEGIN;
-- Rollback DDL statements here
DROP INDEX IF EXISTS idx_new_index;
ALTER TABLE users DROP COLUMN IF EXISTS new_column;
COMMIT;
```

## Validation Steps

### 1. Service Health
```bash
# Check endpoints are populated
kubectl get endpoints <service> -n <namespace>

# Verify health checks
for pod in $(kubectl get pods -n <namespace> -l app=<service> -o name); do
  echo "Checking $pod"
  kubectl exec -n <namespace> $pod -- curl -s localhost:8080/health
done

# Test through ingress
curl -v https://<public-endpoint>/health
```

### 2. Functionality Verification
```bash
# Run smoke tests
kubectl run -it --rm test --image=<test-image> --restart=Never -- /run-smoke-tests.sh

# Check key metrics
curl -s http://prometheus:9090/api/v1/query?query=up{job="<service>"}
curl -s http://prometheus:9090/api/v1/query?query=rate(http_requests_total{service="<service>"}[1m])
```

## Communication Template

### Status Update
```
SERVICE DOWN - <service-name>
Time Detected: <timestamp>
Impact: <user-facing impact>
Current Status: <investigating|restoring|monitoring>
Next Update: <time>

Actions Taken:
- <action 1>
- <action 2>

Next Steps:
- <planned action>
```

### Escalation Message
```
ESCALATION REQUIRED - <service-name>
Duration: <time> minutes
Attempted Actions:
- Restart: Failed/Succeeded
- Rollback: Failed/Succeeded
- Scale: Failed/Succeeded

Need assistance with:
- <specific help needed>

Joining: <incident-channel>
```

## Post-Recovery

1. **Monitoring Period**
   - Watch metrics for 30 minutes
   - Check for error rate changes
   - Monitor resource usage

2. **Documentation**
   - Update incident log
   - Record timeline
   - Note effective actions

3. **Follow-up Actions**
   - Schedule postmortem
   - Create JIRA tickets
   - Update monitoring

## Related Documents
- [High Error Rate Runbook](./high-error-rate.md)
- [Infrastructure Alerts Runbook](./infrastructure-alerts.md)
- [Disaster Recovery Plan](../disaster-recovery.md)