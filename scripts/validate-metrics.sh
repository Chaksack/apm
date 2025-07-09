#!/bin/bash

# validate-metrics.sh
# End-to-end metric validation script for APM system
# Executes all validation tests and provides comprehensive health checks

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VALIDATION_DIR="$PROJECT_ROOT/tests/validation"
RESULTS_DIR="$PROJECT_ROOT/validation-results"
LOG_FILE="$RESULTS_DIR/validation.log"

# Default URLs and configuration
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
GRAFANA_USERNAME="${GRAFANA_USERNAME:-admin}"
GRAFANA_PASSWORD="${GRAFANA_PASSWORD:-admin}"
GRAFANA_API_KEY="${GRAFANA_API_KEY:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

# Help function
show_help() {
    cat << EOF
Usage: $0 [OPTIONS] [TESTS]

End-to-end metric validation for APM system

OPTIONS:
    -h, --help              Show this help message
    -v, --verbose           Enable verbose output
    -o, --output DIR        Output directory for results (default: $RESULTS_DIR)
    --prometheus-url URL    Prometheus server URL (default: $PROMETHEUS_URL)
    --grafana-url URL       Grafana server URL (default: $GRAFANA_URL)
    --grafana-username USER Grafana username (default: $GRAFANA_USERNAME)
    --grafana-password PASS Grafana password (default: $GRAFANA_PASSWORD)
    --grafana-api-key KEY   Grafana API key (overrides username/password)
    --skip-deps             Skip dependency checks
    --no-cleanup            Don't cleanup temporary files
    --timeout SECONDS       Test timeout in seconds (default: 300)

TESTS:
    all                     Run all validation tests (default)
    prometheus             Run Prometheus metrics validation
    grafana                Run Grafana validation
    istio                  Run Istio metrics validation
    connectivity           Run connectivity checks only
    health                 Run health checks only

EXAMPLES:
    $0                                          # Run all tests with defaults
    $0 prometheus grafana                       # Run specific tests
    $0 --prometheus-url http://prom:9090 all    # Custom Prometheus URL
    $0 --grafana-api-key abc123 grafana        # Use API key authentication
    $0 --verbose --output /tmp/results         # Verbose output to custom directory

ENVIRONMENT VARIABLES:
    PROMETHEUS_URL         Prometheus server URL
    GRAFANA_URL           Grafana server URL
    GRAFANA_USERNAME      Grafana username
    GRAFANA_PASSWORD      Grafana password
    GRAFANA_API_KEY       Grafana API key
    KUBECONFIG            Kubernetes config file path
EOF
}

# Parse command line arguments
parse_args() {
    VERBOSE=false
    SKIP_DEPS=false
    NO_CLEANUP=false
    TIMEOUT=300
    TESTS_TO_RUN=()
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -o|--output)
                RESULTS_DIR="$2"
                shift 2
                ;;
            --prometheus-url)
                PROMETHEUS_URL="$2"
                shift 2
                ;;
            --grafana-url)
                GRAFANA_URL="$2"
                shift 2
                ;;
            --grafana-username)
                GRAFANA_USERNAME="$2"
                shift 2
                ;;
            --grafana-password)
                GRAFANA_PASSWORD="$2"
                shift 2
                ;;
            --grafana-api-key)
                GRAFANA_API_KEY="$2"
                shift 2
                ;;
            --skip-deps)
                SKIP_DEPS=true
                shift
                ;;
            --no-cleanup)
                NO_CLEANUP=true
                shift
                ;;
            --timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            all|prometheus|grafana|istio|connectivity|health)
                TESTS_TO_RUN+=("$1")
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # Default to all tests if none specified
    if [ ${#TESTS_TO_RUN[@]} -eq 0 ]; then
        TESTS_TO_RUN=("all")
    fi
}

# Setup environment
setup_environment() {
    log "Setting up validation environment..."
    
    # Create results directory
    mkdir -p "$RESULTS_DIR"
    
    # Create log file
    touch "$LOG_FILE"
    
    # Set verbose mode
    if [ "$VERBOSE" = true ]; then
        set -x
    fi
    
    log "Results directory: $RESULTS_DIR"
    log "Log file: $LOG_FILE"
}

# Check dependencies
check_dependencies() {
    if [ "$SKIP_DEPS" = true ]; then
        log "Skipping dependency checks"
        return 0
    fi
    
    log "Checking dependencies..."
    
    local missing_deps=()
    
    # Check Python and required packages
    if ! command -v python3 &> /dev/null; then
        missing_deps+=("python3")
    else
        # Check Python packages
        if ! python3 -c "import requests" &> /dev/null; then
            missing_deps+=("python3-requests")
        fi
    fi
    
    # Check Node.js
    if ! command -v node &> /dev/null; then
        missing_deps+=("nodejs")
    fi
    
    # Check npm and axios
    if command -v npm &> /dev/null; then
        if ! npm list axios &> /dev/null; then
            log "Installing axios for Node.js..."
            npm install axios || missing_deps+=("axios")
        fi
    fi
    
    # Check Go
    if ! command -v go &> /dev/null; then
        missing_deps+=("golang")
    fi
    
    # Check curl
    if ! command -v curl &> /dev/null; then
        missing_deps+=("curl")
    fi
    
    # Check jq
    if ! command -v jq &> /dev/null; then
        missing_deps+=("jq")
    fi
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        log_error "Missing dependencies: ${missing_deps[*]}"
        log_error "Please install missing dependencies and try again"
        exit 1
    fi
    
    log_success "All dependencies satisfied"
}

# Check connectivity to services
check_connectivity() {
    log "Checking connectivity to services..."
    
    local failures=0
    
    # Check Prometheus
    if ! curl -s --connect-timeout 5 "$PROMETHEUS_URL/api/v1/status/config" > /dev/null; then
        log_error "Cannot connect to Prometheus at $PROMETHEUS_URL"
        failures=$((failures + 1))
    else
        log_success "Prometheus connectivity: OK"
    fi
    
    # Check Grafana
    if ! curl -s --connect-timeout 5 "$GRAFANA_URL/api/health" > /dev/null; then
        log_error "Cannot connect to Grafana at $GRAFANA_URL"
        failures=$((failures + 1))
    else
        log_success "Grafana connectivity: OK"
    fi
    
    # Check Kubernetes (if available)
    if command -v kubectl &> /dev/null; then
        if kubectl cluster-info &> /dev/null; then
            log_success "Kubernetes connectivity: OK"
        else
            log_warning "Kubernetes connectivity: FAILED (Istio tests may fail)"
        fi
    else
        log_warning "kubectl not found (Istio tests may fail)"
    fi
    
    return $failures
}

# Run health checks
run_health_checks() {
    log "Running component health checks..."
    
    local health_file="$RESULTS_DIR/health_check.json"
    local health_results=()
    
    # Prometheus health
    local prom_health=$(curl -s "$PROMETHEUS_URL/api/v1/status/config" | jq -r '.status // "error"')
    health_results+=("\"prometheus\": {\"status\": \"$prom_health\", \"url\": \"$PROMETHEUS_URL\"}")
    
    # Grafana health
    local grafana_health=$(curl -s "$GRAFANA_URL/api/health" | jq -r '.database // "error"')
    health_results+=("\"grafana\": {\"status\": \"$grafana_health\", \"url\": \"$GRAFANA_URL\"}")
    
    # Kubernetes health (if available)
    if command -v kubectl &> /dev/null && kubectl cluster-info &> /dev/null; then
        health_results+=("\"kubernetes\": {\"status\": \"ok\"}")
    else
        health_results+=("\"kubernetes\": {\"status\": \"unavailable\"}")
    fi
    
    # Save health results
    echo "{$(IFS=','; echo "${health_results[*]}")}" | jq '.' > "$health_file"
    
    log_success "Health check results saved to $health_file"
}

# Run Prometheus validation
run_prometheus_validation() {
    log "Running Prometheus metrics validation..."
    
    local output_file="$RESULTS_DIR/prometheus_validation.json"
    local exit_code=0
    
    cd "$VALIDATION_DIR"
    
    if timeout "$TIMEOUT" python3 metrics-validation.py \
        --prometheus-url "$PROMETHEUS_URL" \
        --output "$output_file" 2>&1 | tee -a "$LOG_FILE"; then
        log_success "Prometheus validation completed successfully"
    else
        exit_code=$?
        log_error "Prometheus validation failed with exit code $exit_code"
    fi
    
    return $exit_code
}

# Run Grafana validation
run_grafana_validation() {
    log "Running Grafana validation..."
    
    local output_file="$RESULTS_DIR/grafana_validation.json"
    local exit_code=0
    
    cd "$VALIDATION_DIR"
    
    local grafana_args=(
        "--grafana-url=$GRAFANA_URL"
        "--output=$output_file"
    )
    
    if [ -n "$GRAFANA_API_KEY" ]; then
        grafana_args+=("--api-key=$GRAFANA_API_KEY")
    else
        grafana_args+=("--username=$GRAFANA_USERNAME")
        grafana_args+=("--password=$GRAFANA_PASSWORD")
    fi
    
    if timeout "$TIMEOUT" node grafana-validation.js "${grafana_args[@]}" 2>&1 | tee -a "$LOG_FILE"; then
        log_success "Grafana validation completed successfully"
    else
        exit_code=$?
        log_error "Grafana validation failed with exit code $exit_code"
    fi
    
    return $exit_code
}

# Run Istio validation
run_istio_validation() {
    log "Running Istio metrics validation..."
    
    local exit_code=0
    
    cd "$VALIDATION_DIR"
    
    # Check if Go test files exist
    if [ ! -f "istio-metrics-test.go" ]; then
        log_error "Istio test file not found: istio-metrics-test.go"
        return 1
    fi
    
    # Initialize Go module if needed
    if [ ! -f "go.mod" ]; then
        log "Initializing Go module..."
        go mod init istio-metrics-test
        go mod tidy
    fi
    
    # Run Go tests
    if timeout "$TIMEOUT" go test -v -timeout="${TIMEOUT}s" ./... 2>&1 | tee -a "$LOG_FILE"; then
        log_success "Istio validation completed successfully"
    else
        exit_code=$?
        log_error "Istio validation failed with exit code $exit_code"
    fi
    
    return $exit_code
}

# Generate summary report
generate_summary() {
    log "Generating validation summary..."
    
    local summary_file="$RESULTS_DIR/validation_summary.json"
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    # Count test results
    local total_tests=0
    local passed_tests=0
    local failed_tests=0
    
    for result_file in "$RESULTS_DIR"/*.json; do
        if [ -f "$result_file" ] && [ "$(basename "$result_file")" != "validation_summary.json" ]; then
            if jq -e '.test_results // .testResults' "$result_file" > /dev/null 2>&1; then
                local file_total=$(jq '.validation_run.total_tests // .validationRun.totalTests // 0' "$result_file")
                local file_passed=$(jq '.validation_run.passed // .validationRun.passed // 0' "$result_file")
                local file_failed=$(jq '.validation_run.failed // .validationRun.failed // 0' "$result_file")
                
                total_tests=$((total_tests + file_total))
                passed_tests=$((passed_tests + file_passed))
                failed_tests=$((failed_tests + file_failed))
            fi
        fi
    done
    
    # Generate summary
    cat > "$summary_file" << EOF
{
  "validation_summary": {
    "timestamp": "$timestamp",
    "configuration": {
      "prometheus_url": "$PROMETHEUS_URL",
      "grafana_url": "$GRAFANA_URL",
      "timeout": $TIMEOUT,
      "tests_run": $(printf '%s\n' "${TESTS_TO_RUN[@]}" | jq -R . | jq -s .)
    },
    "results": {
      "total_tests": $total_tests,
      "passed": $passed_tests,
      "failed": $failed_tests,
      "success_rate": $(echo "scale=2; $passed_tests * 100 / $total_tests" | bc -l 2>/dev/null || echo "0")
    },
    "files": {
      "log_file": "$LOG_FILE",
      "results_directory": "$RESULTS_DIR"
    }
  }
}
EOF
    
    log_success "Summary report generated: $summary_file"
    
    # Print summary to console
    echo
    echo "=================================="
    echo "       VALIDATION SUMMARY"
    echo "=================================="
    echo "Total Tests: $total_tests"
    echo "Passed: $passed_tests"
    echo "Failed: $failed_tests"
    if [ $total_tests -gt 0 ]; then
        echo "Success Rate: $(echo "scale=1; $passed_tests * 100 / $total_tests" | bc -l 2>/dev/null || echo "0")%"
    fi
    echo "Results Directory: $RESULTS_DIR"
    echo "=================================="
}

# Cleanup function
cleanup() {
    if [ "$NO_CLEANUP" = true ]; then
        log "Skipping cleanup"
        return 0
    fi
    
    log "Cleaning up..."
    
    # Remove temporary files
    find "$RESULTS_DIR" -name "*.tmp" -delete 2>/dev/null || true
    
    # Compress log file if it's large
    if [ -f "$LOG_FILE" ] && [ $(stat -f%z "$LOG_FILE" 2>/dev/null || stat -c%s "$LOG_FILE" 2>/dev/null || echo 0) -gt 1048576 ]; then
        log "Compressing large log file..."
        gzip "$LOG_FILE" 2>/dev/null || true
    fi
}

# Main execution function
main() {
    local start_time=$(date +%s)
    local exit_code=0
    
    # Parse arguments
    parse_args "$@"
    
    # Setup
    setup_environment
    check_dependencies
    
    log "Starting metrics validation at $(date)"
    log "Tests to run: ${TESTS_TO_RUN[*]}"
    
    # Run tests based on selection
    for test in "${TESTS_TO_RUN[@]}"; do
        case $test in
            all)
                check_connectivity || exit_code=$?
                run_health_checks || exit_code=$?
                run_prometheus_validation || exit_code=$?
                run_grafana_validation || exit_code=$?
                run_istio_validation || exit_code=$?
                ;;
            connectivity)
                check_connectivity || exit_code=$?
                ;;
            health)
                run_health_checks || exit_code=$?
                ;;
            prometheus)
                run_prometheus_validation || exit_code=$?
                ;;
            grafana)
                run_grafana_validation || exit_code=$?
                ;;
            istio)
                run_istio_validation || exit_code=$?
                ;;
            *)
                log_error "Unknown test: $test"
                exit_code=1
                ;;
        esac
    done
    
    # Generate summary
    generate_summary
    
    # Cleanup
    cleanup
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    log "Validation completed in ${duration}s"
    
    if [ $exit_code -eq 0 ]; then
        log_success "All validations passed successfully"
    else
        log_error "Some validations failed (exit code: $exit_code)"
    fi
    
    exit $exit_code
}

# Handle script interruption
trap cleanup EXIT

# Run main function
main "$@"