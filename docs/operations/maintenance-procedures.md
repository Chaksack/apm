# Maintenance Procedures Runbook

## Overview
This runbook provides comprehensive procedures for regular maintenance tasks, update procedures, downtime planning, and health check schedules for the APM system.

## Regular Maintenance Tasks

### Daily Maintenance

#### System Health Checks
```bash
#!/bin/bash
# daily-health-check.sh

LOG_FILE="/var/log/daily-maintenance.log"
DATE=$(date +%Y-%m-%d)

echo "=== Daily Health Check - $DATE ===" >> $LOG_FILE

# Check system resources
echo "System Resources:" >> $LOG_FILE
df -h >> $LOG_FILE
free -h >> $LOG_FILE
uptime >> $LOG_FILE

# Check service status
echo "Service Status:" >> $LOG_FILE
systemctl status apm-service >> $LOG_FILE
systemctl status postgresql >> $LOG_FILE
systemctl status nginx >> $LOG_FILE

# Check application health
echo "Application Health:" >> $LOG_FILE
curl -f http://localhost:8080/health >> $LOG_FILE 2>&1
if [ $? -eq 0 ]; then
    echo "Application health check: PASS" >> $LOG_FILE
else
    echo "Application health check: FAIL" >> $LOG_FILE
    # Send alert
    curl -X POST -H 'Content-type: application/json' \
        --data '{"text":"Daily health check failed!"}' \
        $SLACK_WEBHOOK_URL
fi

# Check database connectivity
echo "Database Connectivity:" >> $LOG_FILE
psql -h localhost -U apm_user -d apm_db -c "SELECT 1;" >> $LOG_FILE 2>&1
if [ $? -eq 0 ]; then
    echo "Database connectivity: PASS" >> $LOG_FILE
else
    echo "Database connectivity: FAIL" >> $LOG_FILE
fi

# Check backup status
echo "Backup Status:" >> $LOG_FILE
if [ -f "/var/backups/database/backup_$(date +%Y%m%d)_*.backup" ]; then
    echo "Daily backup: FOUND" >> $LOG_FILE
else
    echo "Daily backup: MISSING" >> $LOG_FILE
fi

# Check log rotation
echo "Log Rotation:" >> $LOG_FILE
ls -la /var/log/apm/*.log >> $LOG_FILE

echo "=== End Daily Health Check ===" >> $LOG_FILE
```

#### Log Analysis
```bash
#!/bin/bash
# daily-log-analysis.sh

LOG_FILE="/var/log/log-analysis.log"
DATE=$(date +%Y-%m-%d)
APP_LOG="/var/log/apm/application.log"

echo "=== Daily Log Analysis - $DATE ===" >> $LOG_FILE

# Error count analysis
ERROR_COUNT=$(grep -c "ERROR" $APP_LOG)
WARN_COUNT=$(grep -c "WARN" $APP_LOG)

echo "Error Count: $ERROR_COUNT" >> $LOG_FILE
echo "Warning Count: $WARN_COUNT" >> $LOG_FILE

# Top error messages
echo "Top Error Messages:" >> $LOG_FILE
grep "ERROR" $APP_LOG | awk '{print $4, $5, $6, $7, $8}' | sort | uniq -c | sort -nr | head -5 >> $LOG_FILE

# Performance metrics from logs
echo "Performance Metrics:" >> $LOG_FILE
grep "response_time" $APP_LOG | awk '{sum+=$NF; count++} END {print "Average Response Time:", sum/count, "ms"}' >> $LOG_FILE

# Database query analysis
echo "Database Query Analysis:" >> $LOG_FILE
grep "slow_query" $APP_LOG | wc -l >> $LOG_FILE

# Alert on high error rates
if [ "$ERROR_COUNT" -gt "100" ]; then
    curl -X POST -H 'Content-type: application/json' \
        --data "{\"text\":\"High error count detected: $ERROR_COUNT errors\"}" \
        $SLACK_WEBHOOK_URL
fi

echo "=== End Daily Log Analysis ===" >> $LOG_FILE
```

### Weekly Maintenance

#### Database Maintenance
```bash
#!/bin/bash
# weekly-database-maintenance.sh

LOG_FILE="/var/log/weekly-maintenance.log"
DATE=$(date +%Y-%m-%d)

echo "=== Weekly Database Maintenance - $DATE ===" >> $LOG_FILE

# Database vacuum and analyze
echo "Starting VACUUM ANALYZE..." >> $LOG_FILE
psql -h localhost -U apm_user -d apm_db -c "VACUUM ANALYZE;" >> $LOG_FILE 2>&1

# Update database statistics
echo "Updating database statistics..." >> $LOG_FILE
psql -h localhost -U apm_user -d apm_db -c "ANALYZE;" >> $LOG_FILE 2>&1

# Check for unused indexes
echo "Checking for unused indexes..." >> $LOG_FILE
psql -h localhost -U apm_user -d apm_db -c "
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes 
WHERE idx_tup_read = 0 
AND idx_tup_fetch = 0
ORDER BY schemaname, tablename, indexname;
" >> $LOG_FILE

# Check database size trends
echo "Database size analysis..." >> $LOG_FILE
psql -h localhost -U apm_user -d apm_db -c "
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE schemaname NOT IN ('information_schema', 'pg_catalog')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
LIMIT 10;
" >> $LOG_FILE

# Check for long-running queries
echo "Long-running queries check..." >> $LOG_FILE
psql -h localhost -U apm_user -d apm_db -c "
SELECT 
    pid,
    now() - pg_stat_activity.query_start AS duration,
    query 
FROM pg_stat_activity 
WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes'
AND state = 'active';
" >> $LOG_FILE

echo "=== End Weekly Database Maintenance ===" >> $LOG_FILE
```

#### System Cleanup
```bash
#!/bin/bash
# weekly-system-cleanup.sh

LOG_FILE="/var/log/weekly-cleanup.log"
DATE=$(date +%Y-%m-%d)

echo "=== Weekly System Cleanup - $DATE ===" >> $LOG_FILE

# Log rotation cleanup
echo "Cleaning up old logs..." >> $LOG_FILE
find /var/log -name "*.log" -mtime +30 -delete >> $LOG_FILE 2>&1
find /var/log -name "*.log.gz" -mtime +90 -delete >> $LOG_FILE 2>&1

# Temporary file cleanup
echo "Cleaning temporary files..." >> $LOG_FILE
find /tmp -mtime +7 -delete >> $LOG_FILE 2>&1

# Docker cleanup (if applicable)
echo "Docker cleanup..." >> $LOG_FILE
docker system prune -f >> $LOG_FILE 2>&1

# Package cache cleanup
echo "Package cache cleanup..." >> $LOG_FILE
apt-get clean >> $LOG_FILE 2>&1
apt-get autoclean >> $LOG_FILE 2>&1

# Check disk usage after cleanup
echo "Disk usage after cleanup:" >> $LOG_FILE
df -h >> $LOG_FILE

echo "=== End Weekly System Cleanup ===" >> $LOG_FILE
```

### Monthly Maintenance

#### Security Updates
```bash
#!/bin/bash
# monthly-security-updates.sh

LOG_FILE="/var/log/monthly-security-updates.log"
DATE=$(date +%Y-%m-%d)

echo "=== Monthly Security Updates - $DATE ===" >> $LOG_FILE

# Check for security updates
echo "Checking for security updates..." >> $LOG_FILE
apt list --upgradable 2>/dev/null | grep -i security >> $LOG_FILE

# Update package lists
echo "Updating package lists..." >> $LOG_FILE
apt-get update >> $LOG_FILE 2>&1

# Install security updates
echo "Installing security updates..." >> $LOG_FILE
DEBIAN_FRONTEND=noninteractive apt-get -y upgrade >> $LOG_FILE 2>&1

# Check for services that need restart
echo "Checking for services needing restart..." >> $LOG_FILE
needrestart -r l >> $LOG_FILE 2>&1

# Update container images
echo "Updating container images..." >> $LOG_FILE
docker pull postgres:13 >> $LOG_FILE 2>&1
docker pull nginx:latest >> $LOG_FILE 2>&1

# SSL certificate check
echo "Checking SSL certificates..." >> $LOG_FILE
openssl x509 -in /etc/ssl/certs/apm.crt -text -noout | grep "Not After" >> $LOG_FILE

echo "=== End Monthly Security Updates ===" >> $LOG_FILE
```

#### Performance Optimization
```bash
#!/bin/bash
# monthly-performance-optimization.sh

LOG_FILE="/var/log/monthly-performance.log"
DATE=$(date +%Y-%m-%d)

echo "=== Monthly Performance Optimization - $DATE ===" >> $LOG_FILE

# Analyze slow queries
echo "Analyzing slow queries..." >> $LOG_FILE
psql -h localhost -U apm_user -d apm_db -c "
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    rows
FROM pg_stat_statements 
ORDER BY total_time DESC 
LIMIT 10;
" >> $LOG_FILE

# Check index usage
echo "Checking index usage..." >> $LOG_FILE
psql -h localhost -U apm_user -d apm_db -c "
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_tup_read,
    idx_tup_fetch,
    idx_tup_read + idx_tup_fetch as total_reads
FROM pg_stat_user_indexes 
ORDER BY total_reads DESC 
LIMIT 20;
" >> $LOG_FILE

# Application performance metrics
echo "Application performance analysis..." >> $LOG_FILE
curl -s http://localhost:8080/metrics | grep -E "(response_time|request_rate|error_rate)" >> $LOG_FILE

# Memory usage analysis
echo "Memory usage analysis..." >> $LOG_FILE
ps aux --sort=-%mem | head -10 >> $LOG_FILE

# I/O analysis
echo "I/O analysis..." >> $LOG_FILE
iostat -x 1 5 >> $LOG_FILE

echo "=== End Monthly Performance Optimization ===" >> $LOG_FILE
```

## Update Procedures

### Application Updates

#### Rolling Update Procedure
```bash
#!/bin/bash
# rolling-update.sh

# Configuration
NEW_VERSION=$1
DEPLOYMENT_NAME="apm-service"
HEALTH_CHECK_URL="http://localhost:8080/health"
LOG_FILE="/var/log/deployment.log"

if [ -z "$NEW_VERSION" ]; then
    echo "Usage: $0 <new_version>"
    exit 1
fi

echo "=== Rolling Update to $NEW_VERSION ===" >> $LOG_FILE

# Pre-update checks
echo "Pre-update health check..." >> $LOG_FILE
curl -f $HEALTH_CHECK_URL >> $LOG_FILE 2>&1
if [ $? -ne 0 ]; then
    echo "Pre-update health check failed. Aborting update." >> $LOG_FILE
    exit 1
fi

# Update deployment
echo "Updating deployment to $NEW_VERSION..." >> $LOG_FILE
kubectl set image deployment/$DEPLOYMENT_NAME apm-service=apm-service:$NEW_VERSION >> $LOG_FILE 2>&1

# Wait for rollout
echo "Waiting for rollout to complete..." >> $LOG_FILE
kubectl rollout status deployment/$DEPLOYMENT_NAME --timeout=600s >> $LOG_FILE 2>&1

if [ $? -eq 0 ]; then
    echo "Rollout completed successfully" >> $LOG_FILE
    
    # Post-update health check
    sleep 30
    curl -f $HEALTH_CHECK_URL >> $LOG_FILE 2>&1
    if [ $? -eq 0 ]; then
        echo "Post-update health check passed" >> $LOG_FILE
    else
        echo "Post-update health check failed. Consider rollback." >> $LOG_FILE
        exit 1
    fi
else
    echo "Rollout failed. Initiating rollback..." >> $LOG_FILE
    kubectl rollout undo deployment/$DEPLOYMENT_NAME >> $LOG_FILE 2>&1
    exit 1
fi

echo "=== Update Complete ===" >> $LOG_FILE
```

#### Blue-Green Deployment
```bash
#!/bin/bash
# blue-green-deployment.sh

# Configuration
NEW_VERSION=$1
BLUE_SERVICE="apm-service-blue"
GREEN_SERVICE="apm-service-green"
ACTIVE_SERVICE_FILE="/var/lib/apm/active_service"
LOG_FILE="/var/log/blue-green-deployment.log"

if [ -z "$NEW_VERSION" ]; then
    echo "Usage: $0 <new_version>"
    exit 1
fi

# Determine current active service
if [ -f "$ACTIVE_SERVICE_FILE" ]; then
    CURRENT_SERVICE=$(cat $ACTIVE_SERVICE_FILE)
else
    CURRENT_SERVICE=$BLUE_SERVICE
fi

# Determine target service
if [ "$CURRENT_SERVICE" = "$BLUE_SERVICE" ]; then
    TARGET_SERVICE=$GREEN_SERVICE
else
    TARGET_SERVICE=$BLUE_SERVICE
fi

echo "=== Blue-Green Deployment to $NEW_VERSION ===" >> $LOG_FILE
echo "Current active: $CURRENT_SERVICE" >> $LOG_FILE
echo "Target service: $TARGET_SERVICE" >> $LOG_FILE

# Deploy to target service
echo "Deploying to $TARGET_SERVICE..." >> $LOG_FILE
kubectl set image deployment/$TARGET_SERVICE apm-service=apm-service:$NEW_VERSION >> $LOG_FILE 2>&1

# Wait for deployment
kubectl rollout status deployment/$TARGET_SERVICE --timeout=600s >> $LOG_FILE 2>&1

# Health check on target service
echo "Health checking $TARGET_SERVICE..." >> $LOG_FILE
TARGET_IP=$(kubectl get service $TARGET_SERVICE -o jsonpath='{.spec.clusterIP}')
curl -f "http://$TARGET_IP:8080/health" >> $LOG_FILE 2>&1

if [ $? -eq 0 ]; then
    # Switch traffic
    echo "Switching traffic to $TARGET_SERVICE..." >> $LOG_FILE
    kubectl patch service apm-service-main -p '{"spec":{"selector":{"app":"'$TARGET_SERVICE'"}}}' >> $LOG_FILE 2>&1
    
    # Update active service file
    echo $TARGET_SERVICE > $ACTIVE_SERVICE_FILE
    
    echo "Deployment successful. Traffic switched to $TARGET_SERVICE" >> $LOG_FILE
else
    echo "Health check failed. Deployment aborted." >> $LOG_FILE
    exit 1
fi

echo "=== Blue-Green Deployment Complete ===" >> $LOG_FILE
```

### Database Updates

#### Database Schema Migration
```bash
#!/bin/bash
# database-migration.sh

MIGRATION_FILE=$1
LOG_FILE="/var/log/database-migration.log"
BACKUP_DIR="/var/backups/pre-migration"

if [ -z "$MIGRATION_FILE" ]; then
    echo "Usage: $0 <migration_file>"
    exit 1
fi

echo "=== Database Migration ===" >> $LOG_FILE

# Create pre-migration backup
echo "Creating pre-migration backup..." >> $LOG_FILE
mkdir -p $BACKUP_DIR
pg_dump -h localhost -U apm_user -d apm_db > $BACKUP_DIR/pre-migration-$(date +%Y%m%d_%H%M%S).sql

# Validate migration file
echo "Validating migration file..." >> $LOG_FILE
if [ ! -f "$MIGRATION_FILE" ]; then
    echo "Migration file not found: $MIGRATION_FILE" >> $LOG_FILE
    exit 1
fi

# Run migration in transaction
echo "Running migration..." >> $LOG_FILE
psql -h localhost -U apm_user -d apm_db -f $MIGRATION_FILE >> $LOG_FILE 2>&1

if [ $? -eq 0 ]; then
    echo "Migration completed successfully" >> $LOG_FILE
    
    # Verify migration
    echo "Verifying migration..." >> $LOG_FILE
    psql -h localhost -U apm_user -d apm_db -c "SELECT version();" >> $LOG_FILE 2>&1
    
    # Update schema version
    MIGRATION_VERSION=$(basename $MIGRATION_FILE .sql)
    psql -h localhost -U apm_user -d apm_db -c "INSERT INTO schema_migrations (version) VALUES ('$MIGRATION_VERSION');" >> $LOG_FILE 2>&1
    
else
    echo "Migration failed. Check logs for details." >> $LOG_FILE
    exit 1
fi

echo "=== Migration Complete ===" >> $LOG_FILE
```

### System Updates

#### Operating System Updates
```bash
#!/bin/bash
# system-updates.sh

LOG_FILE="/var/log/system-updates.log"
DATE=$(date +%Y-%m-%d)

echo "=== System Updates - $DATE ===" >> $LOG_FILE

# Pre-update system snapshot
echo "Creating system snapshot..." >> $LOG_FILE
systemctl list-units --state=active > /tmp/services-before-update.txt

# Update package lists
echo "Updating package lists..." >> $LOG_FILE
apt-get update >> $LOG_FILE 2>&1

# Show available updates
echo "Available updates:" >> $LOG_FILE
apt list --upgradable >> $LOG_FILE 2>&1

# Install updates
echo "Installing updates..." >> $LOG_FILE
DEBIAN_FRONTEND=noninteractive apt-get -y upgrade >> $LOG_FILE 2>&1

# Check for services that need restart
echo "Checking for services needing restart..." >> $LOG_FILE
needrestart -r l >> $LOG_FILE 2>&1

# Verify system health post-update
echo "Post-update health check..." >> $LOG_FILE
systemctl list-units --state=failed >> $LOG_FILE 2>&1

# Clean up
echo "Cleaning up..." >> $LOG_FILE
apt-get autoremove -y >> $LOG_FILE 2>&1
apt-get autoclean >> $LOG_FILE 2>&1

echo "=== System Updates Complete ===" >> $LOG_FILE
```

## Downtime Planning

### Planned Maintenance Windows

#### Maintenance Schedule
```yaml
# maintenance-schedule.yaml
maintenance_windows:
  regular:
    - name: "Weekly Maintenance"
      schedule: "Saturday 02:00-04:00 UTC"
      frequency: "weekly"
      impact: "minimal"
      services_affected: ["database", "application"]
      
    - name: "Monthly Security Updates"
      schedule: "First Sunday 01:00-05:00 UTC"
      frequency: "monthly"
      impact: "moderate"
      services_affected: ["all"]
      
    - name: "Quarterly Infrastructure Review"
      schedule: "TBD - Business hours coordination"
      frequency: "quarterly"
      impact: "high"
      services_affected: ["all"]

  emergency:
    - name: "Critical Security Patches"
      schedule: "ASAP"
      frequency: "as-needed"
      impact: "high"
      services_affected: ["all"]
      
    - name: "Hardware Failures"
      schedule: "ASAP"
      frequency: "as-needed"
      impact: "critical"
      services_affected: ["infrastructure"]
```

#### Maintenance Notification Template
```bash
#!/bin/bash
# notify-maintenance.sh

MAINTENANCE_TYPE=$1
START_TIME=$2
END_TIME=$3
SERVICES_AFFECTED=$4

# Email notification
cat > /tmp/maintenance-notification.txt << EOF
Subject: Scheduled Maintenance - APM System

Dear Team,

This is to inform you of scheduled maintenance for the APM system.

Maintenance Details:
- Type: $MAINTENANCE_TYPE
- Start Time: $START_TIME
- End Time: $END_TIME
- Services Affected: $SERVICES_AFFECTED
- Expected Impact: Service may be temporarily unavailable

Preparation Required:
- Save any unsaved work
- Expect potential service interruptions
- Monitor system status page for updates

Contact Information:
- On-call engineer: [phone number]
- Emergency contact: [phone number]

Thank you for your understanding.

Operations Team
EOF

# Send notifications
mail -s "Scheduled Maintenance - APM System" team@company.com < /tmp/maintenance-notification.txt

# Update status page
curl -X POST "https://status.company.com/api/incidents" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Scheduled Maintenance",
    "status": "scheduled",
    "message": "System maintenance scheduled from '$START_TIME' to '$END_TIME'",
    "components": ["'$SERVICES_AFFECTED'"]
  }'

# Slack notification
curl -X POST -H 'Content-type: application/json' \
    --data '{"text":"Scheduled maintenance: '$MAINTENANCE_TYPE' from '$START_TIME' to '$END_TIME'"}' \
    $SLACK_WEBHOOK_URL
```

### Maintenance Checklist

#### Pre-Maintenance Checklist
```markdown
# Pre-Maintenance Checklist

## 24 Hours Before
- [ ] Notify all stakeholders
- [ ] Update status page
- [ ] Create system backup
- [ ] Verify rollback procedures
- [ ] Test maintenance procedures in staging
- [ ] Prepare emergency contacts list

## 2 Hours Before
- [ ] Send reminder notification
- [ ] Verify backup completion
- [ ] Check system health
- [ ] Confirm maintenance window
- [ ] Set up monitoring alerts

## 30 Minutes Before
- [ ] Final system health check
- [ ] Enable maintenance mode
- [ ] Notify start of maintenance
- [ ] Take final backup
- [ ] Begin pre-maintenance procedures
```

#### During Maintenance Checklist
```markdown
# During Maintenance Checklist

## Start of Maintenance
- [ ] Confirm maintenance start time
- [ ] Stop non-essential services
- [ ] Put system in maintenance mode
- [ ] Begin maintenance procedures
- [ ] Monitor system resources

## During Maintenance
- [ ] Follow maintenance procedures exactly
- [ ] Document any deviations
- [ ] Monitor for issues
- [ ] Maintain communication log
- [ ] Check time remaining regularly

## End of Maintenance
- [ ] Complete all maintenance tasks
- [ ] Restart services in correct order
- [ ] Verify system functionality
- [ ] Remove maintenance mode
- [ ] Notify completion
```

#### Post-Maintenance Checklist
```markdown
# Post-Maintenance Checklist

## Immediately After
- [ ] Verify all services running
- [ ] Check system health
- [ ] Test critical functionality
- [ ] Monitor error logs
- [ ] Confirm user access

## 30 Minutes After
- [ ] Monitor system performance
- [ ] Check for any issues
- [ ] Verify backup systems
- [ ] Review maintenance logs
- [ ] Update documentation

## 24 Hours After
- [ ] Generate maintenance report
- [ ] Review any issues encountered
- [ ] Update procedures if needed
- [ ] Schedule follow-up if required
- [ ] Archive maintenance logs
```

## Health Check Schedules

### Automated Health Checks

#### Application Health Check
```bash
#!/bin/bash
# application-health-check.sh

# Configuration
APP_URL="http://localhost:8080"
HEALTH_ENDPOINT="/health"
LOG_FILE="/var/log/health-checks.log"
ALERT_THRESHOLD=3

# Function to check health
check_health() {
    local response=$(curl -s -o /dev/null -w "%{http_code}" $APP_URL$HEALTH_ENDPOINT)
    local response_time=$(curl -s -o /dev/null -w "%{time_total}" $APP_URL$HEALTH_ENDPOINT)
    
    echo "$(date): Health check - Status: $response, Response time: ${response_time}s" >> $LOG_FILE
    
    if [ "$response" != "200" ]; then
        return 1
    fi
    
    if (( $(echo "$response_time > 5.0" | bc -l) )); then
        echo "$(date): WARNING - Slow response time: ${response_time}s" >> $LOG_FILE
        return 1
    fi
    
    return 0
}

# Perform health check
if ! check_health; then
    # Increment failure counter
    FAILURE_COUNT=$(cat /tmp/health_failures 2>/dev/null || echo "0")
    FAILURE_COUNT=$((FAILURE_COUNT + 1))
    echo $FAILURE_COUNT > /tmp/health_failures
    
    if [ "$FAILURE_COUNT" -ge "$ALERT_THRESHOLD" ]; then
        # Send alert
        curl -X POST -H 'Content-type: application/json' \
            --data '{"text":"Application health check failed '$FAILURE_COUNT' times"}' \
            $SLACK_WEBHOOK_URL
    fi
else
    # Reset failure counter
    echo "0" > /tmp/health_failures
fi
```

#### Database Health Check
```bash
#!/bin/bash
# database-health-check.sh

LOG_FILE="/var/log/database-health.log"
DB_HOST="localhost"
DB_NAME="apm_db"
DB_USER="apm_user"

# Function to check database health
check_database_health() {
    # Basic connectivity test
    psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT 1;" > /dev/null 2>&1
    if [ $? -ne 0 ]; then
        echo "$(date): Database connectivity failed" >> $LOG_FILE
        return 1
    fi
    
    # Check active connections
    ACTIVE_CONNECTIONS=$(psql -h $DB_HOST -U $DB_USER -d $DB_NAME -t -c "SELECT count(*) FROM pg_stat_activity WHERE state = 'active';")
    MAX_CONNECTIONS=$(psql -h $DB_HOST -U $DB_USER -d $DB_NAME -t -c "SELECT setting FROM pg_settings WHERE name = 'max_connections';")
    
    CONNECTION_USAGE=$(echo "scale=2; $ACTIVE_CONNECTIONS * 100 / $MAX_CONNECTIONS" | bc)
    
    echo "$(date): Active connections: $ACTIVE_CONNECTIONS/$MAX_CONNECTIONS (${CONNECTION_USAGE}%)" >> $LOG_FILE
    
    if (( $(echo "$CONNECTION_USAGE > 80" | bc -l) )); then
        echo "$(date): WARNING - High connection usage: ${CONNECTION_USAGE}%" >> $LOG_FILE
        return 1
    fi
    
    # Check for long-running queries
    LONG_QUERIES=$(psql -h $DB_HOST -U $DB_USER -d $DB_NAME -t -c "SELECT count(*) FROM pg_stat_activity WHERE state = 'active' AND now() - query_start > interval '5 minutes';")
    
    if [ "$LONG_QUERIES" -gt "0" ]; then
        echo "$(date): WARNING - $LONG_QUERIES long-running queries detected" >> $LOG_FILE
        return 1
    fi
    
    return 0
}

# Perform database health check
if ! check_database_health; then
    curl -X POST -H 'Content-type: application/json' \
        --data '{"text":"Database health check failed"}' \
        $SLACK_WEBHOOK_URL
fi
```

#### Infrastructure Health Check
```bash
#!/bin/bash
# infrastructure-health-check.sh

LOG_FILE="/var/log/infrastructure-health.log"

# Function to check infrastructure health
check_infrastructure_health() {
    local issues=0
    
    # CPU usage check
    CPU_USAGE=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d% -f1)
    if (( $(echo "$CPU_USAGE > 80" | bc -l) )); then
        echo "$(date): WARNING - High CPU usage: ${CPU_USAGE}%" >> $LOG_FILE
        issues=$((issues + 1))
    fi
    
    # Memory usage check
    MEM_USAGE=$(free | grep Mem | awk '{printf "%.1f", $3/$2 * 100.0}')
    if (( $(echo "$MEM_USAGE > 85" | bc -l) )); then
        echo "$(date): WARNING - High memory usage: ${MEM_USAGE}%" >> $LOG_FILE
        issues=$((issues + 1))
    fi
    
    # Disk usage check
    DISK_USAGE=$(df -h / | awk 'NR==2{print $5}' | cut -d% -f1)
    if [ "$DISK_USAGE" -gt "90" ]; then
        echo "$(date): WARNING - High disk usage: ${DISK_USAGE}%" >> $LOG_FILE
        issues=$((issues + 1))
    fi
    
    # Service status check
    SERVICES=("apm-service" "postgresql" "nginx")
    for service in "${SERVICES[@]}"; do
        if ! systemctl is-active --quiet $service; then
            echo "$(date): ERROR - Service $service is not running" >> $LOG_FILE
            issues=$((issues + 1))
        fi
    done
    
    echo "$(date): Infrastructure health check - Issues found: $issues" >> $LOG_FILE
    return $issues
}

# Perform infrastructure health check
if ! check_infrastructure_health; then
    curl -X POST -H 'Content-type: application/json' \
        --data '{"text":"Infrastructure health check detected issues"}' \
        $SLACK_WEBHOOK_URL
fi
```

### Health Check Scheduling

#### Cron Configuration
```bash
# /etc/cron.d/apm-health-checks

# Application health check every 5 minutes
*/5 * * * * root /usr/local/bin/application-health-check.sh

# Database health check every 10 minutes
*/10 * * * * root /usr/local/bin/database-health-check.sh

# Infrastructure health check every 15 minutes
*/15 * * * * root /usr/local/bin/infrastructure-health-check.sh

# Daily maintenance tasks
0 2 * * * root /usr/local/bin/daily-health-check.sh

# Weekly maintenance tasks
0 3 * * 0 root /usr/local/bin/weekly-database-maintenance.sh
0 4 * * 0 root /usr/local/bin/weekly-system-cleanup.sh

# Monthly maintenance tasks
0 1 1 * * root /usr/local/bin/monthly-security-updates.sh
0 2 1 * * root /usr/local/bin/monthly-performance-optimization.sh
```

#### Health Check Dashboard
```yaml
# health-dashboard.yaml
dashboard:
  title: "APM System Health Dashboard"
  refresh: "30s"
  
  panels:
    - title: "Application Health"
      type: "stat"
      datasource: "prometheus"
      targets:
        - expr: "up{job='apm-application'}"
        - expr: "http_request_duration_seconds{quantile='0.95'}"
    
    - title: "Database Health"
      type: "stat"
      datasource: "prometheus"
      targets:
        - expr: "pg_up"
        - expr: "pg_stat_database_numbackends / pg_settings_max_connections"
    
    - title: "Infrastructure Health"
      type: "stat"
      datasource: "prometheus"
      targets:
        - expr: "100 - (avg by (instance) (rate(node_cpu_seconds_total{mode='idle'}[5m])) * 100)"
        - expr: "(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100"
    
    - title: "Service Status"
      type: "table"
      datasource: "prometheus"
      targets:
        - expr: "up{job=~'apm-.*'}"
```

### Emergency Procedures

#### Service Recovery
```bash
#!/bin/bash
# emergency-service-recovery.sh

SERVICE_NAME=$1
LOG_FILE="/var/log/emergency-recovery.log"

if [ -z "$SERVICE_NAME" ]; then
    echo "Usage: $0 <service_name>"
    exit 1
fi

echo "=== Emergency Recovery for $SERVICE_NAME ===" >> $LOG_FILE

# Stop service
echo "Stopping $SERVICE_NAME..." >> $LOG_FILE
systemctl stop $SERVICE_NAME >> $LOG_FILE 2>&1

# Check for process cleanup
sleep 10
pkill -f $SERVICE_NAME >> $LOG_FILE 2>&1

# Start service
echo "Starting $SERVICE_NAME..." >> $LOG_FILE
systemctl start $SERVICE_NAME >> $LOG_FILE 2>&1

# Verify service status
if systemctl is-active --quiet $SERVICE_NAME; then
    echo "Service $SERVICE_NAME recovered successfully" >> $LOG_FILE
    
    # Notify team
    curl -X POST -H 'Content-type: application/json' \
        --data "{\"text\":\"Service $SERVICE_NAME recovered successfully\"}" \
        $SLACK_WEBHOOK_URL
else
    echo "Service $SERVICE_NAME failed to recover" >> $LOG_FILE
    
    # Escalate
    curl -X POST -H 'Content-type: application/json' \
        --data "{\"text\":\"CRITICAL: Service $SERVICE_NAME failed to recover\"}" \
        $SLACK_WEBHOOK_URL
fi

echo "=== Emergency Recovery Complete ===" >> $LOG_FILE
```

## Documentation and Reporting

### Maintenance Reports

#### Monthly Maintenance Report Template
```markdown
# Monthly Maintenance Report - [Month Year]

## Executive Summary
- Total maintenance windows: [count]
- Unplanned downtime: [duration]
- System availability: [percentage]
- Critical issues resolved: [count]

## Maintenance Activities
### Scheduled Maintenance
- [List of scheduled maintenance activities]
- [Outcomes and issues]

### Emergency Maintenance
- [List of emergency maintenance activities]
- [Root causes and resolutions]

## Performance Metrics
- Average response time: [ms]
- System uptime: [percentage]
- Error rate: [percentage]
- Resource utilization trends

## Issues and Resolutions
- [Major issues encountered]
- [Resolutions implemented]
- [Preventive measures taken]

## Recommendations
- [Process improvements]
- [Infrastructure upgrades]
- [Training needs]

## Next Month's Schedule
- [Planned maintenance activities]
- [Anticipated challenges]
- [Resource requirements]
```

### Maintenance Best Practices

#### Change Management
- Document all changes
- Test in staging environment
- Have rollback procedures ready
- Communicate changes to stakeholders
- Monitor post-change performance

#### Risk Management
- Assess impact before changes
- Plan for contingencies
- Have emergency contacts ready
- Monitor systems closely
- Document lessons learned

#### Team Coordination
- Clear role assignments
- Regular status updates
- Escalation procedures
- Knowledge sharing
- Training and development