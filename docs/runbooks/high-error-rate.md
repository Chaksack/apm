# High Error Rate Runbook

## Alert Definition
- **Trigger**: Error rate > 5% for 5 minutes
- **Severity**: Critical
- **Team**: Platform Engineering

## Diagnosis Steps

### 1. Initial Assessment
```bash
# Check current error rate across services
curl -s http://localhost:9090/api/v1/query?query=rate(http_requests_total{status=~"5.."}[5m])

# Identify top error-producing services
curl -s http://localhost:9090/api/v1/query?query=topk(10,rate(http_requests_total{status=~"5.."}[5m]))
```

### 2. Error Pattern Analysis
- Check error distribution by status code (500, 502, 503, 504)
- Identify if errors are concentrated in specific endpoints
- Review recent deployments in the affected time window

### 3. Log Investigation
```bash
# Check application logs for stack traces
kubectl logs -n <namespace> <pod-name> --tail=100 | grep -E "ERROR|EXCEPTION|FATAL"

# Check for database connection errors
kubectl logs -n <namespace> <pod-name> | grep -i "connection refused\|timeout"
```

## Common Causes

### 1. Database Issues
- **Symptoms**: Connection timeouts, "too many connections" errors
- **Check**: Database CPU, connection pool exhaustion
- **Fix**: Scale database, optimize queries, increase connection pool

### 2. Memory Pressure
- **Symptoms**: OutOfMemoryError, pod restarts
- **Check**: Memory usage metrics, GC logs
- **Fix**: Increase memory limits, fix memory leaks

### 3. Dependency Failures
- **Symptoms**: 502/503 errors, timeout exceptions
- **Check**: Health status of downstream services
- **Fix**: Circuit breaker activation, fallback mechanisms

### 4. Bad Deployment
- **Symptoms**: Immediate spike after deployment
- **Check**: Deployment timeline, canary metrics
- **Fix**: Rollback deployment

## Remediation Actions

### Immediate Actions (0-5 minutes)
1. **Assess Impact**
   - Number of affected users
   - Critical business functions impacted
   - Revenue impact

2. **Quick Fixes**
   ```bash
   # Scale up affected service
   kubectl scale deployment <service> --replicas=<new-count>
   
   # Enable circuit breaker (if available)
   kubectl set env deployment/<service> CIRCUIT_BREAKER_ENABLED=true
   ```

3. **Communication**
   - Update status page
   - Notify stakeholders via Slack
   - Create incident channel

### Short-term Actions (5-30 minutes)
1. **Rollback if needed**
   ```bash
   # Rollback to previous version
   kubectl rollout undo deployment/<service>
   
   # Verify rollback
   kubectl rollout status deployment/<service>
   ```

2. **Resource Adjustment**
   ```bash
   # Increase resource limits
   kubectl patch deployment <service> -p '{"spec":{"template":{"spec":{"containers":[{"name":"<container>","resources":{"limits":{"memory":"2Gi","cpu":"2"}}}]}}}}'
   ```

3. **Traffic Management**
   - Enable rate limiting
   - Redirect traffic to healthy regions
   - Activate maintenance mode for non-critical features

### Long-term Actions (30+ minutes)
1. **Root Cause Analysis**
   - Collect all logs and metrics
   - Create timeline of events
   - Identify contributing factors

2. **Permanent Fixes**
   - Code fixes for identified bugs
   - Infrastructure improvements
   - Monitoring enhancements

## Escalation Procedures

### Level 1 (0-15 minutes)
- **Who**: On-call engineer
- **Actions**: Initial diagnosis, immediate remediation
- **Escalate if**: Error rate continues to climb or business impact is severe

### Level 2 (15-30 minutes)
- **Who**: Service owner + SRE lead
- **Actions**: Coordinate rollback, advanced troubleshooting
- **Escalate if**: Multiple services affected or data integrity concerns

### Level 3 (30+ minutes)
- **Who**: Engineering Manager + Director
- **Actions**: Major incident response, external communication
- **Criteria**: Revenue impact > $10k/hour or > 50% users affected

## Post-Incident
1. Create incident report within 24 hours
2. Schedule blameless postmortem within 48 hours
3. Track action items in JIRA
4. Update runbook with learnings

## Related Documents
- [Service Down Runbook](./service-down.md)
- [High Latency Runbook](./high-latency.md)
- [Incident Response Process](../incident-response.md)