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
RELEASE_NAME="apm-stack"
CHART_PATH=""
ENVIRONMENT="dev"
VALUES_FILE=""
TIMEOUT="600s"
DRY_RUN=""
FORCE_UPGRADE=""
ATOMIC="--atomic"
WAIT="--wait"
CREATE_NAMESPACE="--create-namespace"

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Usage function
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Deploy APM stack using Helm with value overrides for environments.

OPTIONS:
    -n, --namespace NAMESPACE      Kubernetes namespace (default: apm-system)
    -r, --release-name RELEASE     Helm release name (default: apm-stack)
    -c, --chart-path PATH          Path to Helm chart (default: ./deployments/helm/apm-stack)
    -e, --environment ENV          Environment (dev/staging/prod, default: dev)
    -f, --values-file FILE         Additional values file
    -t, --timeout TIMEOUT          Timeout for deployment (default: 600s)
    --dry-run                      Perform dry run
    --force-upgrade                Force upgrade even if no changes
    --no-atomic                    Don't use atomic upgrades
    --no-wait                      Don't wait for deployment to complete
    --no-create-namespace          Don't create namespace if it doesn't exist
    -h, --help                     Show this help message

Examples:
    $0 --environment prod --namespace apm-system-prod
    $0 --values-file custom-values.yaml --dry-run
    $0 --force-upgrade --timeout 900s
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
    
    if ! command_exists helm; then
        log_error "helm is required but not installed"
        exit 1
    fi
    
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

# Update Helm dependencies
update_dependencies() {
    log_info "Updating Helm dependencies..."
    
    if [ -f "${CHART_PATH}/Chart.yaml" ]; then
        if grep -q "dependencies:" "${CHART_PATH}/Chart.yaml"; then
            log_info "Found dependencies, updating..."
            helm dependency update "$CHART_PATH"
        else
            log_info "No dependencies found, skipping update"
        fi
    else
        log_error "Chart.yaml not found at $CHART_PATH"
        exit 1
    fi
    
    log_success "Dependencies updated successfully"
}

# Validate Helm chart
validate_chart() {
    log_info "Validating Helm chart..."
    
    if ! helm lint "$CHART_PATH"; then
        log_error "Helm chart validation failed"
        exit 1
    fi
    
    log_success "Helm chart validation passed"
}

# Create environment-specific values
create_environment_values() {
    local env_values_file="${PROJECT_ROOT}/deployments/helm/apm-stack/values-${ENVIRONMENT}.yaml"
    
    if [ ! -f "$env_values_file" ]; then
        log_info "Creating environment-specific values file: $env_values_file"
        
        case "$ENVIRONMENT" in
            dev)
                cat > "$env_values_file" << EOF
# Development environment values
global:
  namespace: apm-system-dev

# Reduced resources for development
prometheus:
  resources:
    requests:
      memory: "256Mi"
      cpu: "100m"
    limits:
      memory: "1Gi"
      cpu: "500m"
  storage:
    size: 5Gi
    retentionDays: 7

grafana:
  resources:
    requests:
      memory: "64Mi"
      cpu: "50m"
    limits:
      memory: "256Mi"
      cpu: "250m"
  storage:
    size: 500Mi

loki:
  resources:
    requests:
      memory: "128Mi"
      cpu: "50m"
    limits:
      memory: "512Mi"
      cpu: "250m"
  storage:
    size: 5Gi
    retentionDays: 3

jaeger:
  resources:
    requests:
      memory: "128Mi"
      cpu: "50m"
    limits:
      memory: "512Mi"
      cpu: "250m"
  storage:
    size: 2Gi

alertmanager:
  resources:
    requests:
      memory: "64Mi"
      cpu: "25m"
    limits:
      memory: "128Mi"
      cpu: "100m"
  storage:
    size: 500Mi
EOF
                ;;
            staging)
                cat > "$env_values_file" << EOF
# Staging environment values
global:
  namespace: apm-system-staging

# Moderate resources for staging
prometheus:
  resources:
    requests:
      memory: "512Mi"
      cpu: "250m"
    limits:
      memory: "2Gi"
      cpu: "1000m"
  storage:
    size: 10Gi
    retentionDays: 15

grafana:
  resources:
    requests:
      memory: "128Mi"
      cpu: "100m"
    limits:
      memory: "512Mi"
      cpu: "500m"
  storage:
    size: 1Gi

loki:
  resources:
    requests:
      memory: "256Mi"
      cpu: "100m"
    limits:
      memory: "1Gi"
      cpu: "500m"
  storage:
    size: 10Gi
    retentionDays: 7

jaeger:
  resources:
    requests:
      memory: "256Mi"
      cpu: "100m"
    limits:
      memory: "1Gi"
      cpu: "500m"
  storage:
    size: 5Gi

alertmanager:
  resources:
    requests:
      memory: "128Mi"
      cpu: "50m"
    limits:
      memory: "256Mi"
      cpu: "200m"
  storage:
    size: 1Gi
EOF
                ;;
            prod)
                cat > "$env_values_file" << EOF
# Production environment values
global:
  namespace: apm-system-prod

# Full resources for production
prometheus:
  resources:
    requests:
      memory: "1Gi"
      cpu: "500m"
    limits:
      memory: "4Gi"
      cpu: "2000m"
  storage:
    size: 50Gi
    retentionDays: 30

grafana:
  resources:
    requests:
      memory: "256Mi"
      cpu: "200m"
    limits:
      memory: "1Gi"
      cpu: "1000m"
  storage:
    size: 5Gi

loki:
  resources:
    requests:
      memory: "512Mi"
      cpu: "200m"
    limits:
      memory: "2Gi"
      cpu: "1000m"
  storage:
    size: 50Gi
    retentionDays: 15

jaeger:
  resources:
    requests:
      memory: "512Mi"
      cpu: "200m"
    limits:
      memory: "2Gi"
      cpu: "1000m"
  storage:
    size: 20Gi

alertmanager:
  resources:
    requests:
      memory: "256Mi"
      cpu: "100m"
    limits:
      memory: "512Mi"
      cpu: "500m"
  storage:
    size: 5Gi

# Production-specific configurations
nodeSelector:
  node-type: monitoring

tolerations:
  - key: "monitoring"
    operator: "Equal"
    value: "true"
    effect: "NoSchedule"
EOF
                ;;
        esac
        
        log_success "Environment-specific values file created"
    else
        log_info "Using existing environment values file: $env_values_file"
    fi
    
    VALUES_FILE="$env_values_file"
}

# Build Helm command
build_helm_command() {
    local cmd="helm upgrade --install $RELEASE_NAME $CHART_PATH"
    
    # Add namespace
    cmd="$cmd --namespace $NAMESPACE"
    
    # Add values files
    cmd="$cmd --values ${CHART_PATH}/values.yaml"
    if [ -n "$VALUES_FILE" ]; then
        cmd="$cmd --values $VALUES_FILE"
    fi
    
    # Add flags
    cmd="$cmd --timeout $TIMEOUT"
    [ -n "$DRY_RUN" ] && cmd="$cmd --dry-run"
    [ -n "$FORCE_UPGRADE" ] && cmd="$cmd --force"
    [ -n "$ATOMIC" ] && cmd="$cmd $ATOMIC"
    [ -n "$WAIT" ] && cmd="$cmd $WAIT"
    [ -n "$CREATE_NAMESPACE" ] && cmd="$cmd $CREATE_NAMESPACE"
    
    echo "$cmd"
}

# Deploy with Helm
deploy_helm() {
    log_info "Deploying APM stack with Helm..."
    
    local helm_cmd
    helm_cmd=$(build_helm_command)
    
    log_info "Executing: $helm_cmd"
    
    if eval "$helm_cmd"; then
        log_success "Helm deployment completed successfully"
    else
        log_error "Helm deployment failed"
        exit 1
    fi
}

# Check deployment status
check_deployment_status() {
    if [ -n "$DRY_RUN" ]; then
        log_info "Skipping status check (dry run mode)"
        return 0
    fi
    
    log_info "Checking deployment status..."
    
    # Check Helm release status
    helm status "$RELEASE_NAME" --namespace "$NAMESPACE"
    
    # Check pod status
    log_info "Pod Status:"
    kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/managed-by=Helm
    
    # Check service status
    log_info "Service Status:"
    kubectl get services -n "$NAMESPACE"
    
    log_success "Deployment status check completed"
}

# Perform smoke tests
run_smoke_tests() {
    if [ -n "$DRY_RUN" ]; then
        log_info "Skipping smoke tests (dry run mode)"
        return 0
    fi
    
    log_info "Running smoke tests..."
    
    # Test Prometheus
    if kubectl get service prometheus -n "$NAMESPACE" >/dev/null 2>&1; then
        log_info "✓ Prometheus service is running"
    else
        log_warning "✗ Prometheus service not found"
    fi
    
    # Test Grafana
    if kubectl get service grafana -n "$NAMESPACE" >/dev/null 2>&1; then
        log_info "✓ Grafana service is running"
    else
        log_warning "✗ Grafana service not found"
    fi
    
    # Test Loki
    if kubectl get service loki -n "$NAMESPACE" >/dev/null 2>&1; then
        log_info "✓ Loki service is running"
    else
        log_warning "✗ Loki service not found"
    fi
    
    # Test Jaeger
    if kubectl get service jaeger -n "$NAMESPACE" >/dev/null 2>&1; then
        log_info "✓ Jaeger service is running"
    else
        log_warning "✗ Jaeger service not found"
    fi
    
    # Test Alertmanager
    if kubectl get service alertmanager -n "$NAMESPACE" >/dev/null 2>&1; then
        log_info "✓ Alertmanager service is running"
    else
        log_warning "✗ Alertmanager service not found"
    fi
    
    log_success "Smoke tests completed"
}

# Release management
manage_release() {
    log_info "Managing Helm release..."
    
    # Show release history
    log_info "Release History:"
    helm history "$RELEASE_NAME" --namespace "$NAMESPACE" --max 5
    
    # Show current values
    log_info "Current Release Values:"
    helm get values "$RELEASE_NAME" --namespace "$NAMESPACE"
    
    log_success "Release management completed"
}

# Main function
main() {
    log_info "Starting APM stack Helm deployment..."
    log_info "Environment: $ENVIRONMENT"
    log_info "Namespace: $NAMESPACE"
    log_info "Release Name: $RELEASE_NAME"
    log_info "Chart Path: $CHART_PATH"
    
    # Set default chart path if not provided
    if [ -z "$CHART_PATH" ]; then
        CHART_PATH="${PROJECT_ROOT}/deployments/helm/apm-stack"
    fi
    
    # Adjust namespace and release name for environment
    if [ "$ENVIRONMENT" != "dev" ]; then
        NAMESPACE="apm-system-${ENVIRONMENT}"
        RELEASE_NAME="apm-stack-${ENVIRONMENT}"
    fi
    
    # Check prerequisites
    check_prerequisites
    
    # Update dependencies
    update_dependencies
    
    # Validate chart
    validate_chart
    
    # Create environment-specific values
    create_environment_values
    
    # Deploy with Helm
    deploy_helm
    
    # Check deployment status
    check_deployment_status
    
    # Run smoke tests
    run_smoke_tests
    
    # Manage release
    manage_release
    
    log_success "APM stack Helm deployment completed successfully!"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -r|--release-name)
            RELEASE_NAME="$2"
            shift 2
            ;;
        -c|--chart-path)
            CHART_PATH="$2"
            shift 2
            ;;
        -e|--environment)
            ENVIRONMENT="$2"
            shift 2
            ;;
        -f|--values-file)
            VALUES_FILE="$2"
            shift 2
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN="true"
            shift
            ;;
        --force-upgrade)
            FORCE_UPGRADE="true"
            shift
            ;;
        --no-atomic)
            ATOMIC=""
            shift
            ;;
        --no-wait)
            WAIT=""
            shift
            ;;
        --no-create-namespace)
            CREATE_NAMESPACE=""
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

# Run main function
main