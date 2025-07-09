# On-Call Guide

## Overview
This guide provides comprehensive information for on-call engineers supporting the APM system, including responsibilities, common issues and solutions, emergency contacts, and handoff procedures.

## On-Call Responsibilities

### Primary Responsibilities
- **24/7 System Monitoring**: Monitor alerts and system health
- **Incident Response**: Respond to incidents within defined SLAs
- **Escalation Management**: Escalate issues when necessary
- **Communication**: Keep stakeholders informed during incidents
- **Documentation**: Document all incidents and resolutions
- **Preventive Actions**: Implement immediate fixes to prevent recurrence

### Response Time SLAs
- **P1 (Critical)**: 15 minutes
- **P2 (High)**: 30 minutes
- **P3 (Medium)**: 2 hours
- **P4 (Low)**: Next business day

### On-Call Schedule
- **Rotation**: Weekly rotation
- **Handoff**: Every Monday at 9:00 AM
- **Backup**: Secondary on-call for escalation
- **Coverage**: 24/7 coverage required

## Alert Management

### Alert Channels
- **Primary**: PagerDuty
- **Secondary**: Slack #alerts channel
- **Tertiary**: Email notifications
- **Dashboard**: Grafana monitoring dashboard

### Alert Acknowledgment
```bash
# Acknowledge alert in PagerDuty
pd incident acknowledge --id <incident_id>

# Update Slack thread
curl -X POST -H 'Content-type: application/json' \
    --data '{"text":"Alert acknowledged. Investigating..."}' \
    $SLACK_WEBHOOK_URL

# Log acknowledgment
echo "$(date): Alert acknowledged - Incident ID: <incident_id>" >> /var/log/on-call.log
```

### Alert Triage Process
1. **Immediate Assessment** (0-5 minutes)
   - Check alert severity and impact
   - Verify system status in dashboards
   - Confirm alert is not a false positive

2. **Initial Investigation** (5-15 minutes)
   - Review recent deployments
   - Check system logs
   - Identify affected components

3. **Impact Assessment** (15-30 minutes)
   - Determine user impact
   - Assess business impact
   - Decide on escalation needs

## Common Issues and Solutions

### Database Issues

#### High Database Connection Usage
**Symptoms:**
- Alert: "Database connection pool high"
- Slow application response
- Connection timeout errors

**Immediate Actions:**
```bash
# Check current connections
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "
SELECT 
    count(*) as active_connections,
    (SELECT setting FROM pg_settings WHERE name = 'max_connections') as max_connections
FROM pg_stat_activity 
WHERE state = 'active';"

# Identify long-running queries
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "
SELECT 
    pid,
    now() - pg_stat_activity.query_start AS duration,
    query,
    state
FROM pg_stat_activity 
WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes'
ORDER BY duration DESC;"

# Kill problematic queries if necessary
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT pg_terminate_backend(<pid>);"
```

**Resolution Steps:**
1. Restart application to reset connection pool
2. Check for connection leaks in application code
3. Scale read replicas if needed
4. Consider increasing max_connections temporarily

#### Database Slow Performance
**Symptoms:**
- Slow query alerts
- High database CPU usage
- Application timeouts

**Immediate Actions:**
```bash
# Check slow queries
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    rows
FROM pg_stat_statements 
ORDER BY total_time DESC 
LIMIT 10;"

# Check for lock contention
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "
SELECT 
    blocked_locks.pid AS blocked_pid,
    blocked_activity.usename AS blocked_user,
    blocking_locks.pid AS blocking_pid,
    blocking_activity.usename AS blocking_user,
    blocked_activity.query AS blocked_statement,
    blocking_activity.query AS current_statement_in_blocking_process
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks blocking_locks ON blocking_locks.locktype = blocked_locks.locktype
JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.granted;"
```

**Resolution Steps:**
1. Identify and optimize slow queries
2. Check for missing indexes
3. Run VACUUM ANALYZE if needed
4. Consider read replica scaling

### Application Issues

#### High CPU Usage
**Symptoms:**
- CPU usage above 80%
- Slow response times
- Application timeouts

**Immediate Actions:**
```bash
# Check CPU usage by process
top -p $(pgrep -f apm-service)

# Check application metrics
curl -s http://localhost:8080/metrics | grep -E "(cpu|memory|goroutines)"

# Check for memory leaks
ps aux | grep apm-service | awk '{print $4, $6, $11}'

# Scale horizontally if needed
kubectl scale deployment apm-service --replicas=5
```

**Resolution Steps:**
1. Scale application horizontally
2. Identify CPU-intensive operations
3. Check for infinite loops or deadlocks
4. Review recent code changes

#### Memory Leaks
**Symptoms:**
- Memory usage continuously increasing
- Out of memory errors
- Application crashes

**Immediate Actions:**
```bash
# Monitor memory usage
watch -n 5 'ps aux | grep apm-service | awk "{print \$4, \$6}"'

# Check for memory leaks in metrics
curl -s http://localhost:8080/metrics | grep -E "(memory|heap|gc)"

# Get heap dump (if applicable)
curl -s http://localhost:8080/debug/pprof/heap > heap_dump_$(date +%Y%m%d_%H%M%S).prof

# Restart affected instances
kubectl delete pod -l app=apm-service --force
```

**Resolution Steps:**
1. Restart affected instances
2. Analyze heap dumps
3. Review code for memory leaks
4. Implement temporary memory limits

### Infrastructure Issues

#### Disk Space Full
**Symptoms:**
- Disk usage above 90%
- Application unable to write logs
- Database write errors

**Immediate Actions:**
```bash
# Check disk usage
df -h

# Find large files
find / -type f -size +1G -exec ls -lh {} \; 2>/dev/null | head -20

# Clean up temporary files
find /tmp -type f -mtime +7 -delete
find /var/log -name "*.log" -mtime +30 -delete

# Rotate logs immediately
logrotate -f /etc/logrotate.conf

# Clean Docker if applicable
docker system prune -f
```

**Resolution Steps:**
1. Free up immediate disk space
2. Identify and remove large unnecessary files
3. Implement log rotation
4. Add disk space monitoring

#### Network Connectivity Issues
**Symptoms:**
- Service unavailable errors
- Timeout errors
- DNS resolution failures

**Immediate Actions:**
```bash
# Check network connectivity
ping -c 3 google.com
nslookup apm.company.com

# Check listening ports
netstat -tlnp | grep -E "(8080|5432|80|443)"

# Check firewall rules
iptables -L -n

# Test internal service connectivity
curl -f http://localhost:8080/health
```

**Resolution Steps:**
1. Verify network configuration
2. Check firewall rules
3. Restart networking services if needed
4. Contact network team for infrastructure issues

### Kubernetes Issues

#### Pod Crashes
**Symptoms:**
- Pod restart alerts
- CrashLoopBackOff status
- Application unavailable

**Immediate Actions:**
```bash
# Check pod status
kubectl get pods -l app=apm-service

# Check pod logs
kubectl logs -l app=apm-service --tail=100

# Check events
kubectl get events --sort-by=.metadata.creationTimestamp

# Check resource limits
kubectl describe pod <pod-name>

# Scale up if needed
kubectl scale deployment apm-service --replicas=3
```

**Resolution Steps:**
1. Check pod logs for errors
2. Verify resource limits
3. Check for configuration issues
4. Scale deployment if needed

#### Service Discovery Issues
**Symptoms:**
- Service not reachable
- Load balancer errors
- DNS resolution failures

**Immediate Actions:**
```bash
# Check service status
kubectl get services

# Check endpoints
kubectl get endpoints apm-service

# Check service configuration
kubectl describe service apm-service

# Test service connectivity
kubectl run debug --image=busybox --rm -it --restart=Never -- nslookup apm-service
```

**Resolution Steps:**
1. Verify service configuration
2. Check endpoint health
3. Restart service if needed
4. Review network policies

## Escalation Procedures

### When to Escalate

#### Immediate Escalation (P1)
- Complete system outage
- Data loss or corruption
- Security incidents
- Unable to resolve within 1 hour

#### Escalation After Investigation (P2)
- Multiple system failures
- Performance severely degraded
- Unable to resolve within 2 hours
- Requires specialized knowledge

### Escalation Contacts

#### Primary Escalation
- **Engineering Manager**: [phone] [email]
- **Senior Engineer**: [phone] [email]
- **Database Administrator**: [phone] [email]
- **Security Team**: [phone] [email]

#### Secondary Escalation
- **CTO**: [phone] [email]
- **Product Manager**: [phone] [email]
- **Operations Manager**: [phone] [email]

#### External Escalation
- **Cloud Provider Support**: [phone] [support portal]
- **Vendor Support**: [phone] [support portal]
- **Network Provider**: [phone] [support portal]

### Escalation Process

```bash
#!/bin/bash
# escalate-incident.sh

INCIDENT_ID=$1
ESCALATION_LEVEL=$2
CONTACT_INFO=$3

# Log escalation
echo "$(date): Escalating incident $INCIDENT_ID to $ESCALATION_LEVEL" >> /var/log/escalations.log

# Update PagerDuty
pd incident escalate --id $INCIDENT_ID --escalation-level $ESCALATION_LEVEL

# Send escalation notification
curl -X POST -H 'Content-type: application/json' \
    --data "{\"text\":\"Escalating incident $INCIDENT_ID to $ESCALATION_LEVEL\"}" \
    $SLACK_WEBHOOK_URL

# Call escalation contact
echo "Contact: $CONTACT_INFO"
echo "Incident: $INCIDENT_ID"
echo "Escalation Level: $ESCALATION_LEVEL"
```

## Communication Templates

### Incident Notification
```
Subject: [P{severity}] INCIDENT: {brief description}

INCIDENT ALERT:
- Incident ID: {incident_id}
- Severity: P{severity}
- System: {affected_system}
- Started: {start_time}
- Impact: {impact_description}

CURRENT STATUS:
- {current_status}
- Investigation in progress
- On-call engineer: {engineer_name}

NEXT UPDATE: {next_update_time}

Status Page: https://status.company.com
```

### Status Update
```
Subject: [P{severity}] UPDATE: {brief description}

INCIDENT UPDATE:
- Incident ID: {incident_id}
- Duration: {elapsed_time}
- Status: {current_status}

PROGRESS:
- {actions_taken}
- {current_findings}
- {next_steps}

ESTIMATED RESOLUTION: {eta}
NEXT UPDATE: {next_update_time}
```

### Resolution Notification
```
Subject: [RESOLVED] P{severity}: {brief description}

INCIDENT RESOLVED:
- Incident ID: {incident_id}
- Total Duration: {total_duration}
- Resolution: {resolution_summary}

IMPACT SUMMARY:
- {affected_users}
- {business_impact}
- {any_data_impact}

POST-INCIDENT REVIEW:
- Scheduled for: {review_date}
- Root cause analysis in progress
- Full report available by: {report_date}
```

## Handoff Procedures

### Weekly Handoff Checklist

#### Outgoing On-Call
- [ ] Document all open incidents
- [ ] Update incident status
- [ ] Brief incoming engineer on ongoing issues
- [ ] Transfer any escalated incidents
- [ ] Update on-call schedule
- [ ] Share relevant context and history

#### Incoming On-Call
- [ ] Review open incidents
- [ ] Check system health dashboard
- [ ] Verify alert configurations
- [ ] Test emergency contacts
- [ ] Review recent changes
- [ ] Confirm escalation procedures

### Handoff Meeting Template

```markdown
# On-Call Handoff Meeting - [Date]

## Attendees
- Outgoing: [Name]
- Incoming: [Name]
- Manager: [Name] (if needed)

## Open Incidents
- [Incident ID]: [Description] - [Status]
- [Incident ID]: [Description] - [Status]

## Recent Issues
- [Issue]: [Resolution] - [Date]
- [Issue]: [Resolution] - [Date]

## System Status
- Overall health: [Status]
- Recent deployments: [List]
- Planned maintenance: [Schedule]

## Important Notes
- [Any special considerations]
- [Vendor maintenance windows]
- [Known issues to watch]

## Action Items
- [Follow-up tasks]
- [Monitoring adjustments]
- [Documentation updates]
```

### Handoff Documentation

#### Incident Handoff Form
```yaml
# incident-handoff.yaml
incident_id: "INC-2024-001"
severity: "P2"
title: "Database performance degradation"
status: "investigating"
assigned_to: "incoming-engineer@company.com"

timeline:
  - time: "2024-01-15 14:30"
    action: "Incident started"
    details: "Database slow query alerts triggered"
  - time: "2024-01-15 14:45"
    action: "Investigation begun"
    details: "Checking query performance and connection usage"
  - time: "2024-01-15 15:00"
    action: "Root cause identified"
    details: "Missing index on frequently queried table"

current_status: "Working on index creation in staging"
next_steps: "Deploy index fix to production"
estimated_resolution: "2024-01-15 16:00"

contacts_notified:
  - "engineering-manager@company.com"
  - "product-owner@company.com"

notes: "Index creation tested in staging, ready for production deployment"
```

## Tools and Resources

### Monitoring Tools
- **Grafana**: http://monitoring.company.com
- **PagerDuty**: https://company.pagerduty.com
- **Slack**: #alerts, #incidents, #on-call
- **Status Page**: https://status.company.com

### Access Requirements
- VPN access for remote troubleshooting
- Database read access for investigation
- Kubernetes cluster access
- SSH access to production servers
- PagerDuty admin access

### Reference Documentation
- System architecture diagrams
- Network topology
- Database schema documentation
- API documentation
- Troubleshooting playbooks

### Emergency Procedures

#### System Recovery Commands
```bash
# Restart all services
systemctl restart apm-service postgresql nginx

# Scale Kubernetes deployment
kubectl scale deployment apm-service --replicas=3

# Database emergency restart
systemctl stop postgresql
systemctl start postgresql

# Clear application cache
curl -X POST http://localhost:8080/admin/cache/clear

# Emergency maintenance mode
curl -X POST http://localhost:8080/admin/maintenance/enable
```

#### Emergency Contacts Script
```bash
#!/bin/bash
# emergency-contacts.sh

echo "=== EMERGENCY CONTACTS ==="
echo "Engineering Manager: +1-555-0001"
echo "Senior Engineer: +1-555-0002"
echo "Database Admin: +1-555-0003"
echo "Security Team: +1-555-0004"
echo "CTO: +1-555-0005"
echo ""
echo "=== VENDOR SUPPORT ==="
echo "AWS Support: +1-555-0010"
echo "Database Vendor: +1-555-0011"
echo "Monitoring Vendor: +1-555-0012"
echo ""
echo "=== ESCALATION GROUPS ==="
echo "Slack: #emergency-escalation"
echo "PagerDuty: emergency-escalation"
```

## Training and Certification

### Required Training
- Incident response procedures
- System architecture overview
- Database administration basics
- Kubernetes fundamentals
- Security incident handling

### Certification Requirements
- Complete on-call training program
- Pass incident response simulation
- Demonstrate tool proficiency
- Review system documentation
- Shadow experienced engineer

### Ongoing Education
- Monthly incident review sessions
- Quarterly system updates training
- Annual security awareness training
- Tool-specific training as needed

## Performance Metrics

### On-Call Performance Metrics
- Response time to alerts
- Time to resolution
- Escalation rate
- False positive rate
- Customer satisfaction

### Reporting
- Weekly on-call summary
- Monthly incident trends
- Quarterly process improvements
- Annual training effectiveness

## Best Practices

### Incident Management
- Acknowledge alerts quickly
- Communicate frequently
- Document everything
- Follow escalation procedures
- Learn from incidents

### Communication
- Be clear and concise
- Use appropriate channels
- Keep stakeholders informed
- Document decisions
- Follow up on resolutions

### Personal Care
- Take breaks when possible
- Ask for help when needed
- Maintain work-life balance
- Stay current with training
- Participate in team improvements

## Emergency Scenarios

### Complete System Outage
1. Acknowledge all alerts
2. Assess scope of outage
3. Notify stakeholders immediately
4. Escalate to engineering manager
5. Implement emergency procedures
6. Communicate regularly

### Data Loss Event
1. Stop all write operations
2. Assess extent of data loss
3. Notify security team
4. Escalate to CTO
5. Implement recovery procedures
6. Document forensic evidence

### Security Incident
1. Isolate affected systems
2. Notify security team
3. Preserve evidence
4. Follow security playbook
5. Communicate with legal team
6. Document all actions

This on-call guide provides comprehensive coverage of responsibilities, procedures, and resources needed for effective incident response. Regular updates and training ensure the guide remains current and effective.