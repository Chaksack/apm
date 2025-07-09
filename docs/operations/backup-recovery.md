# Backup and Recovery Runbook

## Overview
This runbook provides comprehensive procedures for data backup, recovery operations, disaster recovery planning, and backup integrity testing for the APM system.

## Backup Strategy

### Backup Types
- **Full Backup**: Complete database and file system backup
- **Incremental Backup**: Changes since last backup
- **Differential Backup**: Changes since last full backup
- **Point-in-Time Recovery**: Transaction log backups

### Backup Schedule
- **Full Backup**: Daily at 2:00 AM
- **Incremental Backup**: Every 4 hours
- **Transaction Log Backup**: Every 15 minutes
- **Configuration Backup**: After any changes

### Retention Policy
- **Daily Backups**: 30 days
- **Weekly Backups**: 12 weeks
- **Monthly Backups**: 12 months
- **Yearly Backups**: 7 years

## Data Backup Procedures

### Database Backup

#### PostgreSQL Backup Procedure
```bash
# Full database backup
pg_dump -h $DB_HOST -U $DB_USER -W -F c -b -v -f backup_$(date +%Y%m%d_%H%M%S).backup $DB_NAME

# Backup with compression
pg_dump -h $DB_HOST -U $DB_USER -W -F c -Z 9 -b -v -f backup_$(date +%Y%m%d_%H%M%S).backup $DB_NAME

# Backup specific tables
pg_dump -h $DB_HOST -U $DB_USER -W -F c -t table1 -t table2 -f selective_backup.backup $DB_NAME
```

#### MySQL Backup Procedure
```bash
# Full database backup
mysqldump -h $DB_HOST -u $DB_USER -p$DB_PASSWORD --single-transaction --routines --triggers $DB_NAME > backup_$(date +%Y%m%d_%H%M%S).sql

# Backup with compression
mysqldump -h $DB_HOST -u $DB_USER -p$DB_PASSWORD --single-transaction --routines --triggers $DB_NAME | gzip > backup_$(date +%Y%m%d_%H%M%S).sql.gz
```

#### Automated Backup Script
```bash
#!/bin/bash
# backup-database.sh

# Configuration
DB_HOST="localhost"
DB_NAME="apm_db"
DB_USER="backup_user"
BACKUP_DIR="/var/backups/database"
RETENTION_DAYS=30

# Create backup directory
mkdir -p $BACKUP_DIR

# Generate backup filename
BACKUP_FILE="$BACKUP_DIR/backup_$(date +%Y%m%d_%H%M%S).backup"

# Perform backup
pg_dump -h $DB_HOST -U $DB_USER -W -F c -b -v -f $BACKUP_FILE $DB_NAME

# Verify backup
if [ $? -eq 0 ]; then
    echo "Backup completed successfully: $BACKUP_FILE"
    # Log success
    echo "$(date): Database backup completed successfully" >> /var/log/backup.log
else
    echo "Backup failed!"
    # Alert on failure
    echo "$(date): Database backup failed" >> /var/log/backup.log
    # Send alert notification
    curl -X POST -H 'Content-type: application/json' \
        --data '{"text":"Database backup failed!"}' \
        $SLACK_WEBHOOK_URL
fi

# Clean up old backups
find $BACKUP_DIR -name "backup_*.backup" -mtime +$RETENTION_DAYS -delete

# Verify cleanup
echo "$(date): Cleanup completed. Removed backups older than $RETENTION_DAYS days" >> /var/log/backup.log
```

### File System Backup

#### Application Code Backup
```bash
#!/bin/bash
# backup-application.sh

APP_DIR="/opt/apm"
BACKUP_DIR="/var/backups/application"
RETENTION_DAYS=7

# Create backup
tar -czf $BACKUP_DIR/app_backup_$(date +%Y%m%d_%H%M%S).tar.gz -C $APP_DIR .

# Log backup
echo "$(date): Application backup completed" >> /var/log/backup.log
```

#### Configuration Backup
```bash
#!/bin/bash
# backup-config.sh

CONFIG_DIRS="/etc/nginx /etc/ssl /opt/apm/config"
BACKUP_DIR="/var/backups/config"
RETENTION_DAYS=30

# Create backup
tar -czf $BACKUP_DIR/config_backup_$(date +%Y%m%d_%H%M%S).tar.gz $CONFIG_DIRS

# Log backup
echo "$(date): Configuration backup completed" >> /var/log/backup.log
```

### Cloud Storage Backup

#### AWS S3 Backup
```bash
#!/bin/bash
# backup-to-s3.sh

LOCAL_BACKUP_DIR="/var/backups"
S3_BUCKET="apm-backups"
S3_PREFIX="daily-backups/$(date +%Y/%m/%d)"

# Sync to S3
aws s3 sync $LOCAL_BACKUP_DIR s3://$S3_BUCKET/$S3_PREFIX/

# Verify sync
if [ $? -eq 0 ]; then
    echo "$(date): S3 backup sync completed" >> /var/log/backup.log
else
    echo "$(date): S3 backup sync failed" >> /var/log/backup.log
fi
```

## Recovery Procedures

### Database Recovery

#### PostgreSQL Recovery
```bash
# Stop application services
sudo systemctl stop apm-service

# Drop existing database (if needed)
dropdb -h $DB_HOST -U $DB_USER $DB_NAME

# Create new database
createdb -h $DB_HOST -U $DB_USER $DB_NAME

# Restore from backup
pg_restore -h $DB_HOST -U $DB_USER -d $DB_NAME -v backup_file.backup

# Verify restoration
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT COUNT(*) FROM users;"

# Start application services
sudo systemctl start apm-service
```

#### MySQL Recovery
```bash
# Stop application services
sudo systemctl stop apm-service

# Drop existing database (if needed)
mysql -h $DB_HOST -u $DB_USER -p$DB_PASSWORD -e "DROP DATABASE $DB_NAME;"

# Create new database
mysql -h $DB_HOST -u $DB_USER -p$DB_PASSWORD -e "CREATE DATABASE $DB_NAME;"

# Restore from backup
mysql -h $DB_HOST -u $DB_USER -p$DB_PASSWORD $DB_NAME < backup_file.sql

# Start application services
sudo systemctl start apm-service
```

### Point-in-Time Recovery

#### PostgreSQL Point-in-Time Recovery
```bash
# Stop PostgreSQL
sudo systemctl stop postgresql

# Restore base backup
tar -xzf base_backup.tar.gz -C /var/lib/postgresql/data/

# Create recovery configuration
cat > /var/lib/postgresql/data/recovery.conf << EOF
restore_command = 'cp /var/lib/postgresql/wal_archive/%f %p'
recovery_target_time = '2024-01-15 14:30:00'
EOF

# Start PostgreSQL
sudo systemctl start postgresql

# Monitor recovery
tail -f /var/log/postgresql/postgresql.log
```

### Application Recovery

#### Application Code Recovery
```bash
#!/bin/bash
# restore-application.sh

BACKUP_FILE="/var/backups/application/app_backup_20240115_140000.tar.gz"
APP_DIR="/opt/apm"

# Stop services
sudo systemctl stop apm-service

# Backup current version
sudo mv $APP_DIR $APP_DIR.old

# Create application directory
sudo mkdir -p $APP_DIR

# Extract backup
sudo tar -xzf $BACKUP_FILE -C $APP_DIR

# Set permissions
sudo chown -R apm:apm $APP_DIR

# Start services
sudo systemctl start apm-service

# Verify restoration
curl -f http://localhost:8080/health
```

#### Configuration Recovery
```bash
#!/bin/bash
# restore-config.sh

BACKUP_FILE="/var/backups/config/config_backup_20240115_140000.tar.gz"

# Stop services
sudo systemctl stop nginx apm-service

# Extract configuration
sudo tar -xzf $BACKUP_FILE -C /

# Restart services
sudo systemctl start nginx apm-service

# Verify configuration
nginx -t
```

## Disaster Recovery Plans

### Recovery Time Objectives (RTO)
- **Critical Systems**: 1 hour
- **Important Systems**: 4 hours
- **Standard Systems**: 24 hours
- **Non-critical Systems**: 72 hours

### Recovery Point Objectives (RPO)
- **Critical Data**: 15 minutes
- **Important Data**: 1 hour
- **Standard Data**: 4 hours
- **Non-critical Data**: 24 hours

### Disaster Recovery Scenarios

#### Scenario 1: Database Server Failure
**Impact**: Complete application unavailability
**RTO**: 1 hour
**RPO**: 15 minutes

**Recovery Steps**:
1. **Immediate (0-5 minutes)**:
   - Activate incident response
   - Notify stakeholders
   - Assess damage extent

2. **Short-term (5-30 minutes)**:
   - Provision new database server
   - Restore latest full backup
   - Apply transaction logs

3. **Medium-term (30-60 minutes)**:
   - Update application configuration
   - Restart application services
   - Verify system functionality

#### Scenario 2: Complete Data Center Outage
**Impact**: All services unavailable
**RTO**: 4 hours
**RPO**: 1 hour

**Recovery Steps**:
1. **Immediate (0-15 minutes)**:
   - Activate disaster recovery team
   - Initiate failover procedures
   - Notify all stakeholders

2. **Short-term (15-60 minutes)**:
   - Provision infrastructure at secondary site
   - Restore from off-site backups
   - Configure network routing

3. **Medium-term (1-4 hours)**:
   - Restore all applications
   - Verify data integrity
   - Update DNS records
   - Test system functionality

#### Scenario 3: Data Corruption
**Impact**: Data integrity compromised
**RTO**: 2 hours
**RPO**: 4 hours

**Recovery Steps**:
1. **Immediate (0-10 minutes)**:
   - Stop all write operations
   - Isolate affected systems
   - Assess corruption scope

2. **Short-term (10-60 minutes)**:
   - Identify clean backup point
   - Restore from clean backup
   - Verify data integrity

3. **Medium-term (1-2 hours)**:
   - Replay transactions if possible
   - Validate application functionality
   - Resume normal operations

### Disaster Recovery Testing

#### Monthly Tests
- Backup restore verification
- Database recovery procedures
- Application failover testing
- Communication plan testing

#### Quarterly Tests
- Full disaster recovery drill
- Cross-site failover testing
- Recovery time measurement
- Process improvement review

#### Annual Tests
- Complete disaster simulation
- Third-party recovery testing
- Vendor failover validation
- Business continuity assessment

## Testing Backup Integrity

### Automated Integrity Checks

#### Database Backup Verification
```bash
#!/bin/bash
# verify-database-backup.sh

BACKUP_FILE="/var/backups/database/backup_$(date +%Y%m%d)_*.backup"
TEST_DB="apm_test_restore"

# Create test database
createdb -h localhost -U postgres $TEST_DB

# Restore backup to test database
pg_restore -h localhost -U postgres -d $TEST_DB -v $BACKUP_FILE

# Verify restoration
if [ $? -eq 0 ]; then
    # Run integrity checks
    psql -h localhost -U postgres -d $TEST_DB -c "
        SELECT 
            schemaname,
            tablename,
            n_tup_ins,
            n_tup_upd,
            n_tup_del
        FROM pg_stat_user_tables;
    "
    
    echo "$(date): Backup integrity verified" >> /var/log/backup.log
else
    echo "$(date): Backup integrity check failed" >> /var/log/backup.log
    # Send alert
    curl -X POST -H 'Content-type: application/json' \
        --data '{"text":"Backup integrity check failed!"}' \
        $SLACK_WEBHOOK_URL
fi

# Clean up test database
dropdb -h localhost -U postgres $TEST_DB
```

#### File System Backup Verification
```bash
#!/bin/bash
# verify-file-backup.sh

BACKUP_FILE="/var/backups/application/app_backup_$(date +%Y%m%d)_*.tar.gz"
TEST_DIR="/tmp/backup_test"

# Create test directory
mkdir -p $TEST_DIR

# Extract backup
tar -xzf $BACKUP_FILE -C $TEST_DIR

# Verify extraction
if [ $? -eq 0 ]; then
    # Check file integrity
    find $TEST_DIR -type f -exec md5sum {} + > /tmp/backup_checksums.txt
    
    echo "$(date): File backup integrity verified" >> /var/log/backup.log
else
    echo "$(date): File backup integrity check failed" >> /var/log/backup.log
fi

# Clean up
rm -rf $TEST_DIR
```

### Manual Verification Procedures

#### Weekly Verification Checklist
- [ ] Verify latest backup exists
- [ ] Check backup file sizes
- [ ] Validate backup timestamps
- [ ] Test random backup restoration
- [ ] Verify backup encryption
- [ ] Check off-site backup sync
- [ ] Validate backup metadata

#### Monthly Verification Checklist
- [ ] Full backup restoration test
- [ ] Cross-platform recovery test
- [ ] Performance impact assessment
- [ ] Backup storage utilization review
- [ ] Recovery procedure validation
- [ ] Documentation updates
- [ ] Team training verification

### Backup Monitoring and Alerting

#### Key Metrics to Monitor
- Backup completion time
- Backup file size trends
- Backup success/failure rates
- Storage utilization
- Recovery time metrics

#### Alert Conditions
- Backup failure
- Backup time exceeded threshold
- Storage space critical
- Integrity check failure
- Off-site sync failure

#### Monitoring Dashboard
```yaml
# Grafana Dashboard Configuration
dashboard:
  title: "Backup Monitoring"
  panels:
    - title: "Backup Success Rate"
      type: "stat"
      targets:
        - expr: "backup_success_rate"
    
    - title: "Backup Duration"
      type: "graph"
      targets:
        - expr: "backup_duration_seconds"
    
    - title: "Storage Utilization"
      type: "gauge"
      targets:
        - expr: "backup_storage_usage_percent"
```

## Recovery Documentation

### Recovery Runbook Template
```markdown
# Recovery Runbook: [System Name]

## Incident Information
- Date: [YYYY-MM-DD]
- Time: [HH:MM UTC]
- Incident ID: [ID]
- Recovery Type: [Full/Partial/Point-in-Time]

## Pre-Recovery Assessment
- [ ] Identify recovery point
- [ ] Validate backup integrity
- [ ] Assess system dependencies
- [ ] Notify stakeholders

## Recovery Steps
1. [Step 1]
2. [Step 2]
3. [Step 3]

## Post-Recovery Verification
- [ ] System functionality test
- [ ] Data integrity check
- [ ] Performance validation
- [ ] User access verification

## Lessons Learned
- What worked well
- What could be improved
- Action items for future
```

## Training and Documentation

### Team Training Schedule
- **Monthly**: Backup procedures review
- **Quarterly**: Recovery drill participation
- **Annually**: Disaster recovery certification

### Documentation Maintenance
- **Weekly**: Update backup logs
- **Monthly**: Review procedures
- **Quarterly**: Update contact information
- **Annually**: Complete procedure review

## Compliance and Auditing

### Compliance Requirements
- SOC 2 Type II backup requirements
- GDPR data protection regulations
- Industry-specific requirements
- Internal audit requirements

### Audit Trail
- Backup execution logs
- Recovery procedure logs
- Access logs
- Change management logs

### Reporting
- **Daily**: Backup status report
- **Weekly**: Recovery capability summary
- **Monthly**: Compliance status report
- **Quarterly**: Audit preparation report