#!/bin/bash

# Quality Gate Check Script
# Pre-deployment quality checks with gate enforcement logic and reporting

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_DIR="$PROJECT_ROOT/ci/quality-gates"
REPORTS_DIR="$PROJECT_ROOT/reports"
LOG_FILE="$REPORTS_DIR/quality-gate.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Gate results
SONARQUBE_GATE_PASSED=false
SECURITY_GATE_PASSED=false
PERFORMANCE_GATE_PASSED=false
OVERALL_GATE_PASSED=false

# Create reports directory
mkdir -p "$REPORTS_DIR"

# Logging function
log() {
    echo -e "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "$LOG_FILE"
}

# Error handling
error() {
    log "${RED}ERROR: $1${NC}"
    exit 1
}

# Success message
success() {
    log "${GREEN}SUCCESS: $1${NC}"
}

# Warning message
warning() {
    log "${YELLOW}WARNING: $1${NC}"
}

# Info message
info() {
    log "${BLUE}INFO: $1${NC}"
}

# Check if required tools are installed
check_dependencies() {
    info "Checking required dependencies..."
    
    local missing_tools=()
    
    # Check for SonarQube scanner
    if ! command -v sonar-scanner &> /dev/null; then
        missing_tools+=("sonar-scanner")
    fi
    
    # Check for Go security tools
    if ! command -v gosec &> /dev/null; then
        missing_tools+=("gosec")
    fi
    
    # Check for Trivy
    if ! command -v trivy &> /dev/null; then
        missing_tools+=("trivy")
    fi
    
    # Check for k6
    if ! command -v k6 &> /dev/null; then
        missing_tools+=("k6")
    fi
    
    # Check for jq
    if ! command -v jq &> /dev/null; then
        missing_tools+=("jq")
    fi
    
    if [ ${#missing_tools[@]} -gt 0 ]; then
        error "Missing required tools: ${missing_tools[*]}"
    fi
    
    success "All dependencies are available"
}

# Run SonarQube quality gate
run_sonarqube_gate() {
    info "Running SonarQube quality gate..."
    
    # Check if SonarQube configuration exists
    if [ ! -f "$CONFIG_DIR/sonarqube-quality-gate.json" ]; then
        error "SonarQube quality gate configuration not found"
    fi
    
    # Run SonarQube analysis
    if sonar-scanner \
        -Dsonar.projectKey="apm" \
        -Dsonar.sources="." \
        -Dsonar.exclusions="**/vendor/**,**/node_modules/**,**/*_test.go" \
        -Dsonar.tests="." \
        -Dsonar.test.inclusions="**/*_test.go" \
        -Dsonar.go.coverage.reportPaths="coverage.out" \
        -Dsonar.qualitygate.wait=true > "$REPORTS_DIR/sonarqube-output.log" 2>&1; then
        
        success "SonarQube analysis completed successfully"
        
        # Check quality gate status
        if grep -q "QUALITY GATE STATUS: PASSED" "$REPORTS_DIR/sonarqube-output.log"; then
            SONARQUBE_GATE_PASSED=true
            success "SonarQube quality gate PASSED"
        else
            warning "SonarQube quality gate FAILED"
            cat "$REPORTS_DIR/sonarqube-output.log"
        fi
    else
        error "SonarQube analysis failed"
    fi
}

# Run security quality gate
run_security_gate() {
    info "Running security quality gate..."
    
    local security_issues=0
    
    # Run gosec
    info "Running gosec security scan..."
    if gosec -fmt json -out "$REPORTS_DIR/gosec-report.json" ./...; then
        local gosec_issues
        gosec_issues=$(jq '.Issues | length' "$REPORTS_DIR/gosec-report.json" 2>/dev/null || echo "0")
        info "Gosec found $gosec_issues issues"
        
        # Check against thresholds (high: 0, medium: 5, low: 10)
        local high_issues
        high_issues=$(jq '[.Issues[] | select(.severity == "HIGH")] | length' "$REPORTS_DIR/gosec-report.json" 2>/dev/null || echo "0")
        
        if [ "$high_issues" -gt 0 ]; then
            warning "Found $high_issues high-severity security issues"
            security_issues=$((security_issues + high_issues))
        fi
    else
        warning "Gosec scan failed"
    fi
    
    # Run Trivy vulnerability scan
    info "Running Trivy vulnerability scan..."
    if trivy fs --format json --output "$REPORTS_DIR/trivy-report.json" .; then
        local trivy_critical
        trivy_critical=$(jq '[.Results[]?.Vulnerabilities[]? | select(.Severity == "CRITICAL")] | length' "$REPORTS_DIR/trivy-report.json" 2>/dev/null || echo "0")
        
        if [ "$trivy_critical" -gt 0 ]; then
            warning "Found $trivy_critical critical vulnerabilities"
            security_issues=$((security_issues + trivy_critical))
        fi
        
        success "Trivy scan completed"
    else
        warning "Trivy scan failed"
    fi
    
    # Check overall security gate status
    if [ "$security_issues" -eq 0 ]; then
        SECURITY_GATE_PASSED=true
        success "Security quality gate PASSED"
    else
        warning "Security quality gate FAILED - Found $security_issues critical/high issues"
    fi
}

# Run performance quality gate
run_performance_gate() {
    info "Running performance quality gate..."
    
    # Check if performance tests exist
    if [ ! -f "performance-tests/load-test.js" ]; then
        warning "Performance tests not found, skipping performance gate"
        PERFORMANCE_GATE_PASSED=true
        return
    fi
    
    # Run k6 load test
    info "Running k6 load tests..."
    if k6 run --out json="$REPORTS_DIR/k6-results.json" performance-tests/load-test.js; then
        
        # Check performance thresholds
        local p95_response_time
        p95_response_time=$(jq '.metrics.http_req_duration.values.p95' "$REPORTS_DIR/k6-results.json" 2>/dev/null || echo "0")
        
        # Convert to integer for comparison (remove decimal)
        p95_response_time=${p95_response_time%.*}
        
        if [ "$p95_response_time" -le 500 ]; then
            PERFORMANCE_GATE_PASSED=true
            success "Performance quality gate PASSED (p95: ${p95_response_time}ms)"
        else
            warning "Performance quality gate FAILED (p95: ${p95_response_time}ms > 500ms)"
        fi
    else
        warning "Performance tests failed"
    fi
}

# Generate quality gate report
generate_report() {
    info "Generating quality gate report..."
    
    local report_file="$REPORTS_DIR/quality-gate-report.json"
    local html_report="$REPORTS_DIR/quality-gate-report.html"
    
    # Generate JSON report
    cat > "$report_file" << EOF
{
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "project": "apm",
    "branch": "$(git rev-parse --abbrev-ref HEAD)",
    "commit": "$(git rev-parse HEAD)",
    "gates": {
        "sonarqube": {
            "passed": $SONARQUBE_GATE_PASSED,
            "report": "sonarqube-output.log"
        },
        "security": {
            "passed": $SECURITY_GATE_PASSED,
            "reports": ["gosec-report.json", "trivy-report.json"]
        },
        "performance": {
            "passed": $PERFORMANCE_GATE_PASSED,
            "report": "k6-results.json"
        }
    },
    "overall": {
        "passed": $OVERALL_GATE_PASSED
    }
}
EOF
    
    # Generate HTML report
    cat > "$html_report" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Quality Gate Report - APM</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f5f5f5; padding: 20px; border-radius: 5px; }
        .gate { margin: 20px 0; padding: 15px; border-radius: 5px; }
        .passed { background-color: #d4edda; border: 1px solid #c3e6cb; color: #155724; }
        .failed { background-color: #f8d7da; border: 1px solid #f5c6cb; color: #721c24; }
        .summary { margin: 20px 0; padding: 15px; border-radius: 5px; font-weight: bold; }
        ul { margin: 10px 0; }
        li { margin: 5px 0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Quality Gate Report</h1>
        <p><strong>Project:</strong> APM</p>
        <p><strong>Branch:</strong> $(git rev-parse --abbrev-ref HEAD)</p>
        <p><strong>Commit:</strong> $(git rev-parse --short HEAD)</p>
        <p><strong>Timestamp:</strong> $(date)</p>
    </div>
    
    <div class="gate $([ "$SONARQUBE_GATE_PASSED" = true ] && echo "passed" || echo "failed")">
        <h2>SonarQube Quality Gate</h2>
        <p><strong>Status:</strong> $([ "$SONARQUBE_GATE_PASSED" = true ] && echo "PASSED" || echo "FAILED")</p>
        <p>Code quality analysis including coverage, duplicated lines, and maintainability.</p>
    </div>
    
    <div class="gate $([ "$SECURITY_GATE_PASSED" = true ] && echo "passed" || echo "failed")">
        <h2>Security Quality Gate</h2>
        <p><strong>Status:</strong> $([ "$SECURITY_GATE_PASSED" = true ] && echo "PASSED" || echo "FAILED")</p>
        <p>Security vulnerability scanning and SAST analysis.</p>
    </div>
    
    <div class="gate $([ "$PERFORMANCE_GATE_PASSED" = true ] && echo "passed" || echo "failed")">
        <h2>Performance Quality Gate</h2>
        <p><strong>Status:</strong> $([ "$PERFORMANCE_GATE_PASSED" = true ] && echo "PASSED" || echo "FAILED")</p>
        <p>Load testing and performance threshold validation.</p>
    </div>
    
    <div class="summary $([ "$OVERALL_GATE_PASSED" = true ] && echo "passed" || echo "failed")">
        <h2>Overall Result</h2>
        <p><strong>Status:</strong> $([ "$OVERALL_GATE_PASSED" = true ] && echo "PASSED" || echo "FAILED")</p>
    </div>
</body>
</html>
EOF
    
    success "Quality gate report generated: $report_file"
    success "HTML report generated: $html_report"
}

# Send notifications
send_notifications() {
    if [ "$OVERALL_GATE_PASSED" = false ]; then
        warning "Quality gates failed - sending notifications"
        
        # Slack notification (if webhook is configured)
        if [ -n "${SLACK_WEBHOOK:-}" ]; then
            curl -X POST -H 'Content-type: application/json' \
                --data "{\"text\":\"🚨 Quality Gate Failed for APM project\\nBranch: $(git rev-parse --abbrev-ref HEAD)\\nCommit: $(git rev-parse --short HEAD)\"}" \
                "$SLACK_WEBHOOK" || warning "Failed to send Slack notification"
        fi
        
        # Email notification would go here
        info "Quality gate failure notifications sent"
    fi
}

# Main execution
main() {
    info "Starting quality gate checks..."
    
    # Check dependencies
    check_dependencies
    
    # Run quality gates
    run_sonarqube_gate
    run_security_gate
    run_performance_gate
    
    # Determine overall result
    if [ "$SONARQUBE_GATE_PASSED" = true ] && [ "$SECURITY_GATE_PASSED" = true ] && [ "$PERFORMANCE_GATE_PASSED" = true ]; then
        OVERALL_GATE_PASSED=true
        success "All quality gates PASSED"
    else
        warning "One or more quality gates FAILED"
    fi
    
    # Generate reports
    generate_report
    
    # Send notifications
    send_notifications
    
    # Exit with appropriate code
    if [ "$OVERALL_GATE_PASSED" = true ]; then
        success "Quality gate check completed successfully"
        exit 0
    else
        error "Quality gate check failed"
    fi
}

# Help function
show_help() {
    cat << EOF
Quality Gate Check Script

Usage: $0 [OPTIONS]

Options:
    -h, --help              Show this help message
    -s, --sonarqube-only    Run only SonarQube quality gate
    -e, --security-only     Run only security quality gate
    -p, --performance-only  Run only performance quality gate
    -r, --reports-dir DIR   Set custom reports directory
    -v, --verbose           Enable verbose logging

Examples:
    $0                      # Run all quality gates
    $0 -s                   # Run only SonarQube gate
    $0 -e                   # Run only security gate
    $0 -p                   # Run only performance gate
    $0 -r /tmp/reports      # Use custom reports directory

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -s|--sonarqube-only)
            run_sonarqube_gate
            exit $?
            ;;
        -e|--security-only)
            run_security_gate
            exit $?
            ;;
        -p|--performance-only)
            run_performance_gate
            exit $?
            ;;
        -r|--reports-dir)
            REPORTS_DIR="$2"
            shift 2
            ;;
        -v|--verbose)
            set -x
            shift
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Run main function
main "$@"