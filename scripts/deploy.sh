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
TIMEOUT="600s"
REPLICAS=""
IMAGE_TAG=""
ENVIRONMENT="dev"
DRY_RUN=""
FORCE_RECREATE=""
ROLLING_UPDATE_STRATEGY="RollingUpdate"
MAX_SURGE="25%"
MAX_UNAVAILABLE="25%"

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Usage function
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Deploy APM stack to Kubernetes with rolling update support.

OPTIONS:
    -n, --namespace NAMESPACE      Kubernetes namespace (default: apm-system)
    -d, --deployment DEPLOYMENT    Deployment name (default: apm-stack)
    -t, --timeout TIMEOUT          Timeout for deployment (default: 600s)
    -r, --replicas REPLICAS        Number of replicas
    -i, --image-tag TAG            Image tag to deploy
    -e, --environment ENV          Environment (dev/staging/prod, default: dev)
    -f, --force-recreate           Force recreate deployment
    --dry-run                      Perform dry run
    --max-surge PERCENTAGE         Max surge during rolling update (default: 25%)
    --max-unavailable PERCENTAGE   Max unavailable during rolling update (default: 25%)
    -h, --help                     Show this help message

Examples:
    $0 --namespace apm-system --image-tag v1.2.3
    $0 --environment prod --replicas 3 --timeout 900s
    $0 --dry-run --namespace test-apm
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
    
    if ! kubectl cluster-info >/dev/null 2>&1; then
        log_error "Unable to connect to Kubernetes cluster"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Create namespace if it doesn't exist
ensure_namespace() {
    log_info "Ensuring namespace '$NAMESPACE' exists..."
    
    if ! kubectl get namespace "$NAMESPACE" >/dev/null 2>&1; then
        log_info "Creating namespace '$NAMESPACE'"
        kubectl create namespace "$NAMESPACE"
    fi
    
    log_success "Namespace '$NAMESPACE' is ready"
}

# Apply Kubernetes manifests
apply_manifests() {
    log_info "Applying Kubernetes manifests..."
    
    local manifests_dir="${PROJECT_ROOT}/deployments/kubernetes"
    
    if [ ! -d "$manifests_dir" ]; then
        log_error "Kubernetes manifests directory not found: $manifests_dir"
        exit 1
    fi
    
    # Apply base resources first
    if [ -d "${manifests_dir}/base" ]; then
        log_info "Applying base resources..."
        kubectl apply -k "${manifests_dir}/base" --namespace="$NAMESPACE" ${DRY_RUN:+--dry-run=client}
    fi
    
    # Apply component manifests
    for component in prometheus grafana loki jaeger alertmanager; do
        if [ -d "${manifests_dir}/${component}" ]; then
            log_info "Applying $component manifests..."
            kubectl apply -f "${manifests_dir}/${component}/" --namespace="$NAMESPACE" ${DRY_RUN:+--dry-run=client}
        fi
    done
    
    log_success "Manifests applied successfully"
}

# Update deployment with rolling update strategy
update_deployment() {
    log_info "Updating deployment with rolling update strategy..."
    
    # Configure rolling update strategy
    kubectl patch deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" -p '{
        "spec": {
            "strategy": {
                "type": "'"$ROLLING_UPDATE_STRATEGY"'",
                "rollingUpdate": {
                    "maxSurge": "'"$MAX_SURGE"'",
                    "maxUnavailable": "'"$MAX_UNAVAILABLE"'"
                }
            }
        }
    }' ${DRY_RUN:+--dry-run=client}
    
    # Update image tag if specified
    if [ -n "$IMAGE_TAG" ]; then
        log_info "Updating image tag to $IMAGE_TAG..."
        kubectl set image deployment/"$DEPLOYMENT_NAME" app="$DEPLOYMENT_NAME:$IMAGE_TAG" -n "$NAMESPACE" ${DRY_RUN:+--dry-run=client}
    fi
    
    # Update replicas if specified
    if [ -n "$REPLICAS" ]; then
        log_info "Scaling deployment to $REPLICAS replicas..."
        kubectl scale deployment/"$DEPLOYMENT_NAME" --replicas="$REPLICAS" -n "$NAMESPACE" ${DRY_RUN:+--dry-run=client}
    fi
    
    # Force recreate if requested
    if [ -n "$FORCE_RECREATE" ]; then
        log_info "Force recreating deployment..."
        kubectl rollout restart deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE" ${DRY_RUN:+--dry-run=client}
    fi
    
    log_success "Deployment updated successfully"
}

# Wait for deployment to be ready
wait_for_deployment() {
    if [ -n "$DRY_RUN" ]; then
        log_info "Skipping deployment wait (dry run mode)"
        return 0
    fi
    
    log_info "Waiting for deployment to be ready (timeout: $TIMEOUT)..."
    
    if kubectl wait --for=condition=available deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE" --timeout="$TIMEOUT"; then
        log_success "Deployment is ready"
    else
        log_error "Deployment failed to become ready within timeout"
        show_deployment_status
        exit 1
    fi
}

# Show deployment status
show_deployment_status() {
    log_info "Deployment Status:"
    kubectl get deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" -o wide
    
    log_info "Pod Status:"
    kubectl get pods -n "$NAMESPACE" -l app="$DEPLOYMENT_NAME" -o wide
    
    log_info "Recent Events:"
    kubectl get events -n "$NAMESPACE" --sort-by=.metadata.creationTimestamp | tail -10
}

# Health check validation
health_check() {
    if [ -n "$DRY_RUN" ]; then
        log_info "Skipping health check (dry run mode)"
        return 0
    fi
    
    log_info "Performing health check..."
    
    # Check if all pods are ready
    local ready_pods
    ready_pods=$(kubectl get pods -n "$NAMESPACE" -l app="$DEPLOYMENT_NAME" -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}')
    
    if [[ "$ready_pods" == *"False"* ]]; then
        log_error "Some pods are not ready"
        show_deployment_status
        exit 1
    fi
    
    # Check if services are accessible
    log_info "Checking service endpoints..."
    kubectl get endpoints -n "$NAMESPACE" -l app="$DEPLOYMENT_NAME"
    
    log_success "Health check passed"
}

# Rollback function
rollback_deployment() {
    log_error "Deployment failed, initiating rollback..."
    
    if kubectl rollout undo deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE"; then
        log_info "Rollback initiated, waiting for completion..."
        kubectl rollout status deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE" --timeout="$TIMEOUT"
        log_success "Rollback completed successfully"
    else
        log_error "Rollback failed"
        exit 1
    fi
}

# Main deployment function
main() {
    log_info "Starting APM stack deployment..."
    log_info "Environment: $ENVIRONMENT"
    log_info "Namespace: $NAMESPACE"
    log_info "Deployment: $DEPLOYMENT_NAME"
    
    # Set deployment name based on environment
    if [ "$ENVIRONMENT" != "dev" ]; then
        DEPLOYMENT_NAME="apm-stack-$ENVIRONMENT"
        NAMESPACE="apm-system-$ENVIRONMENT"
    fi
    
    # Check prerequisites
    check_prerequisites
    
    # Ensure namespace exists
    ensure_namespace
    
    # Apply manifests
    apply_manifests
    
    # Update deployment
    if kubectl get deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" >/dev/null 2>&1; then
        update_deployment
    else
        log_info "Deployment '$DEPLOYMENT_NAME' does not exist, it will be created by manifests"
    fi
    
    # Wait for deployment to be ready
    wait_for_deployment
    
    # Perform health check
    health_check
    
    # Show final status
    show_deployment_status
    
    log_success "APM stack deployment completed successfully!"
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
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        -r|--replicas)
            REPLICAS="$2"
            shift 2
            ;;
        -i|--image-tag)
            IMAGE_TAG="$2"
            shift 2
            ;;
        -e|--environment)
            ENVIRONMENT="$2"
            shift 2
            ;;
        -f|--force-recreate)
            FORCE_RECREATE="true"
            shift
            ;;
        --dry-run)
            DRY_RUN="true"
            shift
            ;;
        --max-surge)
            MAX_SURGE="$2"
            shift 2
            ;;
        --max-unavailable)
            MAX_UNAVAILABLE="$2"
            shift 2
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

# Trap to handle errors and perform rollback
trap 'if [ $? -ne 0 ]; then rollback_deployment; fi' ERR

# Run main function
main