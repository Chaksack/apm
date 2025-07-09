# Scripts Documentation

## Overview

This directory contains automation scripts for the APM stack deployment, configuration, and maintenance. The scripts are designed to be idempotent, well-documented, and follow best practices for shell scripting.

## Script Inventory

### Available Scripts

| Script | Purpose | Usage | Dependencies |
|--------|---------|-------|--------------|
| `setup-slack-integration.sh` | Configure Slack webhooks and channels | `./setup-slack-integration.sh [--dry-run]` | curl, jq, kubectl |
| `deploy-stack.sh` | Deploy the complete APM stack | `./deploy-stack.sh [environment]` | docker, kubectl, helm |
| `backup-monitoring-data.sh` | Backup Prometheus and Grafana data | `./backup-monitoring-data.sh` | kubectl, aws-cli |
| `health-check.sh` | Comprehensive health check | `./health-check.sh [--verbose]` | curl, jq |
| `log-collector.sh` | Collect logs for debugging | `./log-collector.sh [service]` | kubectl, docker |
| `performance-test.sh` | Run performance tests | `./performance-test.sh [--load-test]` | curl, ab |
| `cleanup-resources.sh` | Clean up unused resources | `./cleanup-resources.sh [--dry-run]` | kubectl, docker |
| `update-certificates.sh` | Update SSL certificates | `./update-certificates.sh` | openssl, kubectl |

## Script Usage Examples

### 1. setup-slack-integration.sh

**Purpose**: Automate Slack webhook configuration for AlertManager notifications

**Basic Usage**:
```bash
# Set up Slack integration with dry-run
export SLACK_TOKEN="xoxb-your-slack-token"
./setup-slack-integration.sh --dry-run

# Actual setup
./setup-slack-integration.sh --workspace "my-company"

# Test existing webhook
./setup-slack-integration.sh --test "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

**Parameters**:
- `--dry-run`: Preview changes without applying them
- `--workspace NAME`: Specify Slack workspace name
- `--test WEBHOOK`: Test specific webhook URL
- `--help`: Show help message

**Environment Variables**:
```bash
export SLACK_TOKEN="xoxb-your-slack-bot-token"
export SLACK_WORKSPACE="your-workspace"
export NAMESPACE="monitoring"  # Kubernetes namespace
```

**Example Output**:
```
[INFO] Starting Slack integration setup...
[INFO] Checking prerequisites...
[SUCCESS] Prerequisites check passed
[INFO] Creating Slack channels...
[SUCCESS] Created channel: #alerts
[INFO] Channel already exists: #ops-critical
[SUCCESS] Generated webhook configuration at: /path/to/webhook-urls.env
[WARN] Please update the webhook URLs in /path/to/webhook-urls.env
[SUCCESS] Slack integration setup completed!
```

### 2. deploy-stack.sh

**Purpose**: Deploy the complete APM stack to Kubernetes or Docker Compose

**Basic Usage**:
```bash
# Deploy to development environment
./deploy-stack.sh dev

# Deploy to production with specific image tag
./deploy-stack.sh prod --image-tag v1.2.3

# Deploy with custom configuration
./deploy-stack.sh staging --config-file custom-config.yaml
```

**Parameters**:
- `environment`: Target environment (dev, staging, prod)
- `--image-tag TAG`: Specify image tag (default: latest)
- `--config-file FILE`: Custom configuration file
- `--skip-tests`: Skip pre-deployment tests
- `--dry-run`: Preview deployment changes

**Example Script**:
```bash
#!/bin/bash
set -euo pipefail

# Deploy APM Stack
# Usage: ./deploy-stack.sh [environment] [options]

ENVIRONMENT=${1:-dev}
IMAGE_TAG=${2:-latest}
CONFIG_FILE=${3:-configs/default.yaml}

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Pre-deployment checks
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        error "kubectl is not installed"
        exit 1
    fi
    
    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check namespace
    if ! kubectl get namespace ${NAMESPACE} &> /dev/null; then
        log "Creating namespace ${NAMESPACE}"
        kubectl create namespace ${NAMESPACE}
    fi
    
    log "Prerequisites check passed"
}

# Deploy components
deploy_prometheus() {
    log "Deploying Prometheus..."
    
    helm upgrade --install prometheus prometheus/prometheus \
        --namespace ${NAMESPACE} \
        --values configs/prometheus/values-${ENVIRONMENT}.yaml \
        --set image.tag=${IMAGE_TAG} \
        --wait
    
    log "Prometheus deployed successfully"
}

deploy_grafana() {
    log "Deploying Grafana..."
    
    helm upgrade --install grafana grafana/grafana \
        --namespace ${NAMESPACE} \
        --values configs/grafana/values-${ENVIRONMENT}.yaml \
        --wait
    
    log "Grafana deployed successfully"
}

# Main deployment function
main() {
    log "Starting APM stack deployment to ${ENVIRONMENT}"
    
    check_prerequisites
    deploy_prometheus
    deploy_grafana
    
    log "APM stack deployment completed successfully"
}

# Execute main function
main "$@"
```

### 3. backup-monitoring-data.sh

**Purpose**: Backup Prometheus metrics and Grafana dashboards

**Basic Usage**:
```bash
# Backup to default location
./backup-monitoring-data.sh

# Backup to S3
./backup-monitoring-data.sh --s3-bucket my-backups

# Backup specific components
./backup-monitoring-data.sh --components prometheus,grafana
```

**Parameters**:
- `--s3-bucket BUCKET`: Upload backup to S3 bucket
- `--components LIST`: Comma-separated list of components to backup
- `--retention-days DAYS`: Number of days to retain backups (default: 30)
- `--compress`: Compress backup files

**Example Usage**:
```bash
#!/bin/bash
# Backup script example

BACKUP_DIR="/tmp/apm-backup-$(date +%Y%m%d-%H%M%S)"
S3_BUCKET=""
COMPONENTS="prometheus,grafana,loki"
RETENTION_DAYS=30

# Create backup directory
mkdir -p ${BACKUP_DIR}

# Backup Prometheus data
backup_prometheus() {
    log "Backing up Prometheus data..."
    
    kubectl exec -n monitoring statefulset/prometheus-server -- \
        tar czf - /prometheus | \
        cat > ${BACKUP_DIR}/prometheus-data.tar.gz
    
    log "Prometheus backup completed"
}

# Backup Grafana dashboards
backup_grafana() {
    log "Backing up Grafana dashboards..."
    
    # Export dashboards via API
    GRAFANA_URL="http://$(kubectl get svc grafana -n monitoring -o jsonpath='{.spec.clusterIP}'):3000"
    
    curl -s -H "Authorization: Bearer ${GRAFANA_TOKEN}" \
        "${GRAFANA_URL}/api/search" | \
        jq -r '.[].uri' | \
        while read uri; do
            curl -s -H "Authorization: Bearer ${GRAFANA_TOKEN}" \
                "${GRAFANA_URL}/api/dashboards/${uri}" > \
                "${BACKUP_DIR}/dashboard-${uri//\//-}.json"
        done
    
    log "Grafana backup completed"
}

# Upload to S3 if specified
upload_to_s3() {
    if [[ -n "${S3_BUCKET}" ]]; then
        log "Uploading backup to S3..."
        
        tar czf ${BACKUP_DIR}.tar.gz -C $(dirname ${BACKUP_DIR}) $(basename ${BACKUP_DIR})
        aws s3 cp ${BACKUP_DIR}.tar.gz s3://${S3_BUCKET}/apm-backups/
        
        log "Backup uploaded to S3"
    fi
}

# Main backup function
main() {
    log "Starting monitoring data backup..."
    
    backup_prometheus
    backup_grafana
    upload_to_s3
    
    log "Backup completed successfully"
}

main "$@"
```

### 4. health-check.sh

**Purpose**: Comprehensive health check for all APM components

**Basic Usage**:
```bash
# Basic health check
./health-check.sh

# Verbose output with details
./health-check.sh --verbose

# Check specific service
./health-check.sh --service prometheus

# Output in JSON format
./health-check.sh --format json
```

**Parameters**:
- `--verbose`: Detailed output
- `--service SERVICE`: Check specific service
- `--format FORMAT`: Output format (text, json, xml)
- `--timeout SECONDS`: Request timeout (default: 10)

**Example Implementation**:
```bash
#!/bin/bash
# Health check script

VERBOSE=false
SERVICE=""
FORMAT="text"
TIMEOUT=10

# Health check functions
check_prometheus() {
    local url="http://prometheus:9090"
    local health_endpoint="${url}/-/healthy"
    local ready_endpoint="${url}/-/ready"
    
    if curl -s --max-time ${TIMEOUT} ${health_endpoint} &> /dev/null; then
        if curl -s --max-time ${TIMEOUT} ${ready_endpoint} &> /dev/null; then
            echo "✓ Prometheus: Healthy and Ready"
        else
            echo "⚠ Prometheus: Healthy but not Ready"
        fi
    else
        echo "✗ Prometheus: Unhealthy"
        return 1
    fi
}

check_grafana() {
    local url="http://grafana:3000"
    local health_endpoint="${url}/api/health"
    
    local response=$(curl -s --max-time ${TIMEOUT} ${health_endpoint})
    local status=$(echo ${response} | jq -r '.database')
    
    if [[ "${status}" == "ok" ]]; then
        echo "✓ Grafana: Healthy"
    else
        echo "✗ Grafana: Unhealthy"
        return 1
    fi
}

check_jaeger() {
    local url="http://jaeger:16686"
    local health_endpoint="${url}/api/health"
    
    if curl -s --max-time ${TIMEOUT} ${health_endpoint} | grep -q "ok"; then
        echo "✓ Jaeger: Healthy"
    else
        echo "✗ Jaeger: Unhealthy"
        return 1
    fi
}

# Main health check
main() {
    echo "APM Stack Health Check"
    echo "====================="
    
    local overall_status=0
    
    check_prometheus || overall_status=1
    check_grafana || overall_status=1
    check_jaeger || overall_status=1
    
    echo "====================="
    if [[ ${overall_status} -eq 0 ]]; then
        echo "✓ Overall Status: Healthy"
    else
        echo "✗ Overall Status: Unhealthy"
    fi
    
    return ${overall_status}
}

main "$@"
```

### 5. log-collector.sh

**Purpose**: Collect logs from various components for debugging

**Basic Usage**:
```bash
# Collect all logs
./log-collector.sh

# Collect logs for specific service
./log-collector.sh prometheus

# Collect logs with specific time range
./log-collector.sh --since "2024-01-01 00:00:00" --until "2024-01-02 00:00:00"
```

**Parameters**:
- `service`: Specific service name (prometheus, grafana, jaeger, etc.)
- `--since TIMESTAMP`: Start time for log collection
- `--until TIMESTAMP`: End time for log collection
- `--lines N`: Number of lines to collect (default: 1000)
- `--output-dir DIR`: Output directory for collected logs

### 6. performance-test.sh

**Purpose**: Run performance tests against the APM stack

**Basic Usage**:
```bash
# Basic performance test
./performance-test.sh

# Load test with specific parameters
./performance-test.sh --load-test --concurrent 100 --duration 60s

# Stress test
./performance-test.sh --stress-test --rps 1000
```

**Parameters**:
- `--load-test`: Run load tests
- `--stress-test`: Run stress tests
- `--concurrent N`: Number of concurrent users
- `--duration TIME`: Test duration
- `--rps N`: Requests per second

## Parameter Documentation

### Common Parameters

#### Environment Variables
All scripts support these common environment variables:

```bash
# Kubernetes configuration
export KUBECONFIG=/path/to/kubeconfig
export NAMESPACE=monitoring

# Docker configuration
export DOCKER_REGISTRY=your-registry.com
export DOCKER_USERNAME=your-username
export DOCKER_PASSWORD=your-password

# Monitoring configuration
export PROMETHEUS_URL=http://prometheus:9090
export GRAFANA_URL=http://grafana:3000
export GRAFANA_TOKEN=your-grafana-token

# Notification configuration
export SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL
export EMAIL_RECIPIENTS=admin@company.com,ops@company.com
```

#### Command Line Flags
Standard flags supported by most scripts:

```bash
--help          # Show help message
--verbose       # Verbose output
--dry-run       # Preview changes without applying
--config FILE   # Custom configuration file
--timeout N     # Operation timeout in seconds
--force         # Force operation without confirmation
--quiet         # Suppress output
```

### Script-Specific Parameters

#### setup-slack-integration.sh
```bash
# Required environment variables
SLACK_TOKEN="xoxb-your-bot-token"
SLACK_WORKSPACE="your-workspace"

# Optional parameters
--dry-run                    # Preview changes
--workspace NAME             # Specify workspace
--test WEBHOOK_URL          # Test webhook
--channels "chan1,chan2"    # Specific channels to create
```

#### deploy-stack.sh
```bash
# Required parameters
ENVIRONMENT="dev|staging|prod"

# Optional parameters
--image-tag TAG             # Docker image tag
--config-file FILE          # Custom configuration
--skip-tests               # Skip pre-deployment tests
--rollback                 # Rollback to previous version
--wait-timeout SECONDS     # Deployment wait timeout
```

#### backup-monitoring-data.sh
```bash
# Optional parameters
--s3-bucket BUCKET         # S3 bucket for backup
--components LIST          # Components to backup
--retention-days N         # Backup retention period
--compress                 # Compress backup files
--encrypt                  # Encrypt backup files
```

## Dependencies

### System Dependencies

#### Required Tools
```bash
# Core tools
curl                # HTTP client
jq                  # JSON processor
kubectl             # Kubernetes CLI
docker              # Container runtime
helm                # Kubernetes package manager

# Optional tools
aws-cli             # AWS CLI (for S3 backups)
gzip                # Compression
openssl             # SSL/TLS operations
apache2-utils       # Apache bench (ab) for load testing
```

#### Installation Commands
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y curl jq docker.io

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# macOS
brew install curl jq kubectl helm docker
```

### Go Dependencies
```bash
# For scripts that interact with Go applications
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Python Dependencies
```bash
# For advanced scripting (if needed)
pip install requests pyyaml kubernetes
```

## Best Practices

### Script Development Guidelines

1. **Error Handling**:
   ```bash
   # Always use strict mode
   set -euo pipefail
   
   # Handle errors gracefully
   trap 'echo "Error on line $LINENO"; exit 1' ERR
   ```

2. **Logging**:
   ```bash
   # Use consistent logging
   log() {
       echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
   }
   
   error() {
       echo "[ERROR] $1" >&2
   }
   ```

3. **Configuration**:
   ```bash
   # Use environment variables for configuration
   CONFIG_FILE="${CONFIG_FILE:-/etc/default/apm}"
   TIMEOUT="${TIMEOUT:-30}"
   ```

4. **Validation**:
   ```bash
   # Validate prerequisites
   check_prerequisites() {
       for tool in kubectl helm docker; do
           if ! command -v $tool &> /dev/null; then
               error "$tool is not installed"
               exit 1
           fi
       done
   }
   ```

### Security Best Practices

1. **Secret Management**:
   ```bash
   # Never hardcode secrets
   if [[ -z "${SLACK_TOKEN}" ]]; then
       error "SLACK_TOKEN environment variable is required"
       exit 1
   fi
   
   # Use secure temporary files
   TEMP_FILE=$(mktemp)
   trap "rm -f ${TEMP_FILE}" EXIT
   ```

2. **Input Validation**:
   ```bash
   # Validate input parameters
   validate_environment() {
       case "$1" in
           dev|staging|prod)
               echo "Valid environment: $1"
               ;;
           *)
               error "Invalid environment: $1"
               exit 1
               ;;
       esac
   }
   ```

3. **Privilege Escalation**:
   ```bash
   # Check if running as root when necessary
   if [[ $EUID -eq 0 ]]; then
       error "This script should not be run as root"
       exit 1
   fi
   ```

### Testing Scripts

#### Unit Testing
```bash
# test-setup-slack.sh
#!/bin/bash
source ./setup-slack-integration.sh

test_channel_creation() {
    export DRY_RUN=true
    export SLACK_TOKEN="test-token"
    
    result=$(create_slack_channels)
    
    if [[ $result == *"[DRY RUN]"* ]]; then
        echo "✓ Channel creation test passed"
    else
        echo "✗ Channel creation test failed"
        exit 1
    fi
}

test_channel_creation
```

#### Integration Testing
```bash
# integration-test.sh
#!/bin/bash

# Test complete deployment workflow
test_deployment() {
    ./deploy-stack.sh dev --dry-run
    
    if [[ $? -eq 0 ]]; then
        echo "✓ Deployment test passed"
    else
        echo "✗ Deployment test failed"
        exit 1
    fi
}

test_deployment
```

## Related Documentation

- [CI/CD Pipeline](../docs/ci-cd-pipeline.md)
- [Quality Gates](../docs/quality-gates.md)
- [CI Configuration](../ci/README.md)
- [Deployment Guide](../deployments/README.md)
- [Monitoring Setup](../deployments/kubernetes/prometheus/README.md)

## Contributing

### Adding New Scripts

1. **Create the script** following the naming convention
2. **Add documentation** to this README
3. **Include usage examples**
4. **Add parameter documentation**
5. **List dependencies**
6. **Test thoroughly**

### Script Template
```bash
#!/bin/bash
# Script: new-script.sh
# Purpose: Brief description of what the script does
# Usage: ./new-script.sh [options]
# Author: Your Name
# Version: 1.0.0

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Default values
VERBOSE=false
DRY_RUN=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Logging functions
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Help function
show_help() {
    cat << EOF
Usage: $0 [OPTIONS]

Description:
    Brief description of what the script does

Options:
    --verbose       Enable verbose output
    --dry-run       Preview changes without applying
    --help          Show this help message

Examples:
    $0 --verbose
    $0 --dry-run

EOF
}

# Main function
main() {
    log "Starting script execution..."
    
    # Add your script logic here
    
    log "Script execution completed"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --verbose)
            VERBOSE=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Execute main function
main "$@"
```

## Support

For issues and questions about scripts:
- Create an issue in the repository
- Contact the DevOps team
- Check the troubleshooting sections
- Review the monitoring dashboards for script execution metrics