#!/bin/bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
NAMESPACE="apm-system"
DEPLOYMENT_NAME="apm-stack"
RELEASE_NAME="apm-stack"
ENVIRONMENT="dev"
ROLLBACK_REVISION=""
TIMEOUT="600s"
DRY_RUN=""
METHOD="kubernetes"  # kubernetes or helm
PRESERVE_STATE="true"
NOTIFY_SLACK="true"
BACKUP_BEFORE_ROLLBACK="true"

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Usage function
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Quick rollback procedures for APM stack with state preservation and notifications.

OPTIONS:
    -n, --namespace NAMESPACE      Kubernetes namespace (default: apm-system)
    -d, --deployment DEPLOYMENT    Deployment name (default: apm-stack)
    -r, --release-name RELEASE     Helm release name (default: apm-stack)
    -e, --environment ENV          Environment (dev/staging/prod, default: dev)
    -v, --revision REVISION        Specific revision to rollback to
    -t, --timeout TIMEOUT          Timeout for rollback (default: 600s)
    -m, --method METHOD            Rollback method (kubernetes/helm, default: kubernetes)
    --dry-run                      Perform dry run
    --no-preserve-state            Don't preserve state during rollback
    --no-notify                    Don't send notifications
    --no-backup                    Don't backup before rollback
    -h, --help                     Show this help message

Examples:
    $0 --environment prod --method helm
    $0 --revision 3 --namespace apm-system-staging
    $0 --dry-run --no-preserve-state
EOF
}

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command_exists kubectl; then
        log_error "kubectl is required but not installed"
        exit 1
    fi
    
    if [ "$METHOD" = "helm" ] && ! command_exists helm; then
        log_error "helm is required but not installed"
        exit 1
    fi
    
    if ! kubectl cluster-info >/dev/null 2>&1; then
        log_error "Unable to connect to Kubernetes cluster"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Send notification
send_notification() {
    local message="$1"
    local status="$2"  # info, warning, error, success
    
    if [ "$NOTIFY_SLACK" = "false" ]; then
        return 0
    fi
    
    local color="good"
    local emoji="ℹ️"
    
    case "$status" in
        warning)
            color="warning"
            emoji="⚠️"
            ;;
        error)
            color="danger"
            emoji="❌"
            ;;
        success)
            color="good"
            emoji="✅"
            ;;
    esac
    
    log_info "Sending notification: $message"
    
    # Send to Slack if webhook URL is available
    if [ -n "${SLACK_WEBHOOK_URL:-}" ]; then
        curl -X POST -H 'Content-type: application/json' \
            --data '{
                "text": "'"$emoji"' APM Stack Rollback Notification",
                "attachments": [
                    {
                        "color": "'"$color"'",
                        "fields": [
                            {
                                "title": "Environment",
                                "value": "'"$ENVIRONMENT"'",
                                "short": true
                            },
                            {
                                "title": "Namespace",
                                "value": "'"$NAMESPACE"'",
                                "short": true
                            },
                            {
                                "title": "Message",
                                "value": "'"$message"'",
                                "short": false
                            }
                        ]
                    }
                ]
            }' \
            "$SLACK_WEBHOOK_URL" 2>/dev/null || log_warning "Failed to send Slack notification"
    fi
    
    # Send email notification if configured
    if [ -n "${EMAIL_WEBHOOK_URL:-}" ]; then
        curl -X POST -H 'Content-type: application/json' \
            --data '{
                "to": "devops@company.com",
                "subject": "APM Stack Rollback - '"$ENVIRONMENT"'",
                "body": "'"$message"'"
            }' \
            "$EMAIL_WEBHOOK_URL" 2>/dev/null || log_warning "Failed to send email notification"
    fi
}

# Create backup before rollback
create_backup() {
    if [ "$BACKUP_BEFORE_ROLLBACK" = "false" ]; then
        log_info "Skipping backup creation"
        return 0
    fi
    
    if [ -n "$DRY_RUN" ]; then
        log_info "Skipping backup creation (dry run mode)"
        return 0
    fi
    
    log_info "Creating backup before rollback..."
    
    local backup_dir="/tmp/apm-rollback-backup-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$backup_dir"
    
    # Backup deployments
    log_info "Backing up deployments..."
    kubectl get deployments -n "$NAMESPACE" -o yaml > "$backup_dir/deployments.yaml"
    
    # Backup services
    log_info "Backing up services..."
    kubectl get services -n "$NAMESPACE" -o yaml > "$backup_dir/services.yaml"
    
    # Backup configmaps
    log_info "Backing up configmaps..."
    kubectl get configmaps -n "$NAMESPACE" -o yaml > "$backup_dir/configmaps.yaml"
    
    # Backup secrets (metadata only for security)
    log_info "Backing up secrets metadata..."
    kubectl get secrets -n "$NAMESPACE" -o yaml | grep -v "data:" > "$backup_dir/secrets-metadata.yaml"
    
    # Backup persistent volumes
    log_info "Backing up persistent volumes..."
    kubectl get pv -o yaml > "$backup_dir/persistentvolumes.yaml"
    
    # Backup persistent volume claims
    log_info "Backing up persistent volume claims..."
    kubectl get pvc -n "$NAMESPACE" -o yaml > "$backup_dir/persistentvolumeclaims.yaml"
    
    # Create backup summary
    cat > "$backup_dir/backup-info.txt" << EOF
APM Stack Rollback Backup
=========================
Backup Date: $(date)
Environment: $ENVIRONMENT
Namespace: $NAMESPACE
Deployment: $DEPLOYMENT_NAME
Method: $METHOD
Backup Directory: $backup_dir

Files included:
- deployments.yaml
- services.yaml
- configmaps.yaml
- secrets-metadata.yaml
- persistentvolumes.yaml
- persistentvolumeclaims.yaml
EOF
    
    log_success "Backup created at: $backup_dir"
    echo "export APM_ROLLBACK_BACKUP_DIR=\"$backup_dir\"" > /tmp/apm-rollback-backup-path
}

# Preserve state during rollback
preserve_state() {
    if [ "$PRESERVE_STATE" = "false" ]; then
        log_info "Skipping state preservation"
        return 0
    fi
    
    if [ -n "$DRY_RUN" ]; then
        log_info "Skipping state preservation (dry run mode)"
        return 0
    fi
    
    log_info "Preserving state during rollback..."
    
    # Scale down deployments gracefully
    log_info "Scaling down deployments gracefully..."
    kubectl scale deployment --all --replicas=0 -n "$NAMESPACE" --timeout="$TIMEOUT"
    
    # Wait for pods to terminate
    log_info "Waiting for pods to terminate..."
    kubectl wait --for=delete pods --all -n "$NAMESPACE" --timeout="$TIMEOUT" || true
    
    # Create state preservation annotations
    log_info "Creating state preservation annotations..."
    kubectl annotate namespace "$NAMESPACE" apm.rollback/preserved-at="$(date -Iseconds)" --overwrite
    
    log_success "State preservation completed"
}

# Restore state after rollback
restore_state() {
    if [ "$PRESERVE_STATE" = "false" ]; then
        log_info "Skipping state restoration"
        return 0
    fi
    
    if [ -n "$DRY_RUN" ]; then
        log_info "Skipping state restoration (dry run mode)"
        return 0
    fi
    
    log_info "Restoring state after rollback..."
    
    # Wait for new pods to be ready
    log_info "Waiting for new pods to be ready..."
    kubectl wait --for=condition=ready pods --all -n "$NAMESPACE" --timeout="$TIMEOUT" || true
    
    # Remove state preservation annotations
    log_info "Removing state preservation annotations..."
    kubectl annotate namespace "$NAMESPACE" apm.rollback/preserved-at- || true
    
    log_success "State restoration completed"
}

# Get rollback revision
get_rollback_revision() {
    if [ -n "$ROLLBACK_REVISION" ]; then
        log_info "Using specified revision: $ROLLBACK_REVISION"
        return 0
    fi
    
    log_info "Determining rollback revision..."
    
    if [ "$METHOD" = "helm" ]; then
        # Get Helm release history
        local history
        history=$(helm history "$RELEASE_NAME" -n "$NAMESPACE" --output json 2>/dev/null || echo "[]")
        
        if [ "$history" = "[]" ]; then
            log_error "No Helm release history found for $RELEASE_NAME"
            exit 1
        fi
        
        # Get the previous successful revision
        ROLLBACK_REVISION=$(echo "$history" | jq -r '.[] | select(.status == "deployed") | .revision' | sort -n | tail -2 | head -1)
        
        if [ -z "$ROLLBACK_REVISION" ]; then
            log_error "No suitable revision found for rollback"
            exit 1
        fi
        
        log_info "Found suitable Helm revision for rollback: $ROLLBACK_REVISION"
    else
        # Get Kubernetes deployment rollout history
        local history
        history=$(kubectl rollout history deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE" 2>/dev/null || echo "")
        
        if [ -z "$history" ]; then
            log_error "No deployment history found for $DEPLOYMENT_NAME"
            exit 1
        fi
        
        # Use the previous revision (current - 1)
        ROLLBACK_REVISION=$(kubectl rollout history deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE" | tail -2 | head -1 | awk '{print $1}')
        
        if [ -z "$ROLLBACK_REVISION" ]; then
            log_error "No suitable revision found for rollback"
            exit 1
        fi
        
        log_info "Found suitable Kubernetes revision for rollback: $ROLLBACK_REVISION"
    fi
}

# Perform Kubernetes rollback
rollback_kubernetes() {
    log_info "Performing Kubernetes rollback..."
    
    if [ -n "$DRY_RUN" ]; then
        log_info "Would rollback deployment/$DEPLOYMENT_NAME to revision $ROLLBACK_REVISION"
        return 0
    fi
    
    # Perform the rollback
    if [ -n "$ROLLBACK_REVISION" ]; then
        log_info "Rolling back to revision $ROLLBACK_REVISION..."
        kubectl rollout undo deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE" --to-revision="$ROLLBACK_REVISION"
    else
        log_info "Rolling back to previous revision..."
        kubectl rollout undo deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE"
    fi
    
    # Wait for rollback to complete
    log_info "Waiting for rollback to complete..."
    kubectl rollout status deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE" --timeout="$TIMEOUT"
    
    log_success "Kubernetes rollback completed"
}

# Perform Helm rollback
rollback_helm() {
    log_info "Performing Helm rollback..."
    
    local rollback_cmd="helm rollback $RELEASE_NAME"
    
    if [ -n "$ROLLBACK_REVISION" ]; then
        rollback_cmd="$rollback_cmd $ROLLBACK_REVISION"
    fi
    
    rollback_cmd="$rollback_cmd --namespace $NAMESPACE --timeout $TIMEOUT --wait"
    
    if [ -n "$DRY_RUN" ]; then
        rollback_cmd="$rollback_cmd --dry-run"
    fi
    
    log_info "Executing: $rollback_cmd"
    
    if eval "$rollback_cmd"; then
        log_success "Helm rollback completed"
    else
        log_error "Helm rollback failed"
        exit 1
    fi
}

# Verify rollback
verify_rollback() {
    if [ -n "$DRY_RUN" ]; then
        log_info "Skipping rollback verification (dry run mode)"
        return 0
    fi
    
    log_info "Verifying rollback..."
    
    # Check deployment status
    local deployment_status
    deployment_status=$(kubectl get deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' 2>/dev/null || echo "Unknown")
    
    if [ "$deployment_status" = "True" ]; then
        log_success "Deployment is available after rollback"
    else
        log_error "Deployment is not available after rollback"
        kubectl get deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" -o wide
        exit 1
    fi
    
    # Check pod status
    local ready_pods
    ready_pods=$(kubectl get pods -n "$NAMESPACE" -l app="$DEPLOYMENT_NAME" -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "")
    
    if [[ "$ready_pods" == *"False"* ]]; then
        log_error "Some pods are not ready after rollback"
        kubectl get pods -n "$NAMESPACE" -l app="$DEPLOYMENT_NAME" -o wide
        exit 1
    fi
    
    # Basic health check
    log_info "Performing basic health check..."
    sleep 30  # Wait for services to stabilize
    
    # Check if services are responding
    local services
    services=$(kubectl get services -n "$NAMESPACE" -o name 2>/dev/null || echo "")
    
    if [ -n "$services" ]; then
        log_info "Services are available after rollback"
    else
        log_warning "No services found, skipping service check"
    fi
    
    log_success "Rollback verification completed"
}

# Show rollback summary
show_rollback_summary() {
    log_info "Rollback Summary:"
    echo "===================="
    echo "Environment: $ENVIRONMENT"
    echo "Namespace: $NAMESPACE"
    echo "Method: $METHOD"
    echo "Revision: $ROLLBACK_REVISION"
    echo "Timestamp: $(date)"
    echo "===================="
    
    # Show current deployment status
    if [ "$METHOD" = "helm" ]; then
        log_info "Current Helm release status:"
        helm status "$RELEASE_NAME" -n "$NAMESPACE" 2>/dev/null || echo "Unable to get Helm status"
    else
        log_info "Current deployment status:"
        kubectl get deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" -o wide 2>/dev/null || echo "Unable to get deployment status"
    fi
    
    # Show pod status
    log_info "Current pod status:"
    kubectl get pods -n "$NAMESPACE" -l app="$DEPLOYMENT_NAME" -o wide 2>/dev/null || echo "Unable to get pod status"
}

# Main function
main() {
    log_info "Starting APM stack rollback..."
    log_info "Environment: $ENVIRONMENT"
    log_info "Namespace: $NAMESPACE"
    log_info "Method: $METHOD"
    
    # Adjust namespace and deployment names for environment
    if [ "$ENVIRONMENT" != "dev" ]; then
        NAMESPACE="apm-system-${ENVIRONMENT}"
        DEPLOYMENT_NAME="apm-stack-${ENVIRONMENT}"
        RELEASE_NAME="apm-stack-${ENVIRONMENT}"
    fi
    
    # Send start notification
    send_notification "Starting rollback for $ENVIRONMENT environment" "info"
    
    # Check prerequisites
    check_prerequisites
    
    # Create backup
    create_backup
    
    # Get rollback revision
    get_rollback_revision
    
    # Preserve state
    preserve_state
    
    # Perform rollback based on method
    if [ "$METHOD" = "helm" ]; then
        rollback_helm
    else
        rollback_kubernetes
    fi
    
    # Restore state
    restore_state
    
    # Verify rollback
    verify_rollback
    
    # Show summary
    show_rollback_summary
    
    # Send success notification
    send_notification "Rollback completed successfully for $ENVIRONMENT environment" "success"
    
    log_success "APM stack rollback completed successfully!"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -d|--deployment)
            DEPLOYMENT_NAME="$2"
            shift 2
            ;;
        -r|--release-name)
            RELEASE_NAME="$2"
            shift 2
            ;;
        -e|--environment)
            ENVIRONMENT="$2"
            shift 2
            ;;
        -v|--revision)
            ROLLBACK_REVISION="$2"
            shift 2
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        -m|--method)
            METHOD="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN="true"
            shift
            ;;
        --no-preserve-state)
            PRESERVE_STATE="false"
            shift
            ;;
        --no-notify)
            NOTIFY_SLACK="false"
            shift
            ;;
        --no-backup)
            BACKUP_BEFORE_ROLLBACK="false"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Trap to handle errors and send error notification
trap 'if [ $? -ne 0 ]; then send_notification "Rollback failed for $ENVIRONMENT environment" "error"; fi' ERR

# Run main function
main