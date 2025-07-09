# APM Runbooks

This directory contains operational runbooks for handling alerts and incidents in the APM (Application Performance Monitoring) system.

## Quick Reference

### Alert to Runbook Mapping

| Alert Type | Severity | Runbook | Primary Owner |
|------------|----------|---------|---------------|
| Error Rate > 5% | Critical | [High Error Rate](./high-error-rate.md) | Platform Engineering |
| P95 Latency > 1000ms | Warning/Critical | [High Latency](./high-latency.md) | Platform Engineering |
| Service Health Check Failed | Critical | [Service Down](./service-down.md) | Platform Engineering + Service Owner |
| Node CPU > 85% | Warning | [Infrastructure - Node Issues](./infrastructure-alerts.md#high-cpu-usage) | SRE Team |
| Node Memory > 90% | Critical | [Infrastructure - Memory](./infrastructure-alerts.md#high-memory-usage) | SRE Team |
| Disk Usage > 85% | Warning | [Infrastructure - Disk](./infrastructure-alerts.md#node-disk-space-low) | SRE Team |
| etcd Unhealthy | Critical | [Infrastructure - etcd](./infrastructure-alerts.md#etcd-issues) | SRE Team |

## Runbook Index

### Application-Level Runbooks
1. **[High Error Rate Runbook](./high-error-rate.md)**
   - Handles error rate spikes across services
   - Includes diagnosis steps, common causes, and remediation
   - Escalation procedures for business impact

2. **[High Latency Runbook](./high-latency.md)**
   - Performance troubleshooting guide
   - Query analysis and optimization steps
   - Scaling procedures for immediate relief

3. **[Service Down Runbook](./service-down.md)**
   - Complete service recovery procedures
   - Dependency verification steps
   - Rollback procedures for quick recovery

### Infrastructure-Level Runbooks
4. **[Infrastructure Alerts Runbook](./infrastructure-alerts.md)**
   - Node-level issues (CPU, Memory, Disk, Network)
   - Cluster component problems (etcd, API server)
   - Storage and network troubleshooting

## On-Call Procedures

### Initial Response (0-5 minutes)
1. **Acknowledge Alert**
   - Respond in #incidents channel
   - Claim ownership of the incident

2. **Initial Assessment**
   - Determine severity and impact
   - Check monitoring dashboards
   - Review recent changes

3. **Communication**
   - Post initial status update
   - Create incident channel if needed
   - Notify stakeholders for P0/P1

### During Incident (5-30 minutes)
1. **Follow Runbook**
   - Use appropriate runbook based on alert
   - Execute diagnosis steps systematically
   - Try immediate remediation actions

2. **Escalate if Needed**
   - Follow escalation matrix below
   - Provide context to next level
   - Continue working the issue

3. **Regular Updates**
   - Post updates every 15 minutes
   - Document actions taken
   - Update status page if customer-facing

### Post-Incident (After resolution)
1. **Verification**
   - Confirm service is healthy
   - Monitor for 30 minutes
   - Check for secondary impacts

2. **Documentation**
   - Create incident report
   - Update runbook if needed
   - Schedule postmortem

## Escalation Matrix

### Severity Levels
- **P0**: Complete service outage, data loss risk
- **P1**: Major functionality broken, significant user impact
- **P2**: Degraded performance, partial feature impact
- **P3**: Minor issues, minimal user impact

### Escalation Path

| Time | P0 | P1 | P2 | P3 |
|------|----|----|----|----|
| 0-15 min | On-call Engineer | On-call Engineer | On-call Engineer | On-call Engineer |
| 15-30 min | + Team Lead + SRE Lead | + Team Lead | Continue solo | Continue solo |
| 30-60 min | + Engineering Manager + Director | + SRE Lead | + Team Lead | Continue solo |
| 60+ min | + VP Engineering + CTO | + Engineering Manager | + SRE Lead | + Team Lead |

### Contact Information
- **On-Call Phone**: +1-XXX-XXX-XXXX
- **Incidents Channel**: #incidents
- **War Room Template**: #incident-YYYY-MM-DD-description
- **Status Page**: https://status.company.com

## Best Practices

### Do's
- ✅ Follow runbooks systematically
- ✅ Communicate early and often
- ✅ Document all actions taken
- ✅ Ask for help when stuck
- ✅ Focus on recovery first, root cause second

### Don'ts
- ❌ Skip diagnosis steps
- ❌ Make multiple changes at once
- ❌ Work in isolation on P0/P1
- ❌ Blame individuals
- ❌ Hide mistakes or issues

## Runbook Maintenance

### Review Schedule
- Monthly: Review and update based on incidents
- Quarterly: Full review with team
- Annually: Major revision and restructure

### Update Process
1. Create PR with runbook changes
2. Get review from on-call engineer
3. Test procedures in staging
4. Merge and announce updates

### Contributing
- Add new scenarios as discovered
- Update commands when tools change
- Improve clarity based on feedback
- Add automation where possible

## Tools and Access

### Required Access
- Kubernetes cluster (kubectl)
- Monitoring systems (Prometheus, Grafana)
- Log aggregation (ELK, Loki)
- Cloud provider console
- Database access (read-only)

### Useful Commands
```bash
# Quick cluster health check
kubectl get nodes && kubectl top nodes && kubectl get pods -A | grep -v Running

# Recent events
kubectl get events -A --sort-by='.lastTimestamp' | tail -20

# Service status check
for svc in service1 service2 service3; do
  echo "Checking $svc..."
  curl -s http://$svc/health | jq .
done

# Database connection test
kubectl run -it --rm dbtest --image=postgres:13 --restart=Never -- psql -h dbhost -U user -c "SELECT 1"
```

## Training Resources
- [Incident Command Training](../training/incident-command.md)
- [Kubernetes Troubleshooting](../training/k8s-troubleshooting.md)
- [Monitoring and Alerting](../training/monitoring.md)
- [On-Call Best Practices](../training/on-call.md)

---
*Last Updated: 2025-01-09*
*Version: 1.0*