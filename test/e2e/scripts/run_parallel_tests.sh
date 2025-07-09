#!/bin/bash

# E2E Parallel Test Runner Script
# This script runs E2E tests in parallel with proper resource management

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPORT_DIR="${PROJECT_ROOT}/test-results"
LOG_DIR="${PROJECT_ROOT}/logs"
PARALLEL_WORKERS=${PARALLEL_WORKERS:-4}
RETRY_COUNT=${RETRY_COUNT:-2}
TEST_TIMEOUT=${TEST_TIMEOUT:-30m}

# Create necessary directories
mkdir -p "${REPORT_DIR}"
mkdir -p "${LOG_DIR}"

# Function to print colored output
print_status() {
    local color=$1
    shift
    echo -e "${color}$@${NC}"
}

# Function to check prerequisites
check_prerequisites() {
    print_status $BLUE "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_status $RED "Error: Go is not installed"
        exit 1
    fi
    
    # Check if Docker is running
    if ! docker info &> /dev/null; then
        print_status $RED "Error: Docker is not running"
        exit 1
    fi
    
    # Check if docker-compose is installed
    if ! command -v docker-compose &> /dev/null; then
        print_status $RED "Error: docker-compose is not installed"
        exit 1
    fi
    
    print_status $GREEN "Prerequisites check passed"
}

# Function to start services
start_services() {
    print_status $BLUE "Starting APM services..."
    
    cd "${PROJECT_ROOT}"
    
    # Check if services are already running
    if docker-compose -f docker-compose.test.yml ps | grep -q "Up"; then
        print_status $YELLOW "Services are already running"
    else
        docker-compose -f docker-compose.test.yml up -d
        
        # Wait for services to be ready
        print_status $BLUE "Waiting for services to be ready..."
        sleep 30
        
        # Health check
        "${SCRIPT_DIR}/health_check.sh" || {
            print_status $RED "Services health check failed"
            docker-compose -f docker-compose.test.yml logs
            exit 1
        }
    fi
    
    print_status $GREEN "Services are ready"
}

# Function to run specific test scenario
run_scenario() {
    local scenario=$1
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local report_file="${REPORT_DIR}/test-report-${scenario}-${timestamp}.json"
    local html_report="${REPORT_DIR}/test-report-${scenario}-${timestamp}.html"
    local log_file="${LOG_DIR}/test-${scenario}-${timestamp}.log"
    
    print_status $BLUE "Running ${scenario} tests..."
    
    cd "${PROJECT_ROOT}"
    
    # Create a Go test file that uses the parallel runner
    cat > "${PROJECT_ROOT}/parallel_runner_test.go" << EOF
//go:build e2e
// +build e2e

package e2e

import (
    "fmt"
    "os"
    "testing"
    "time"
)

func TestParallelRunner_${scenario}(t *testing.T) {
    // Create parallel test runner
    runner := NewParallelTestRunner(${PARALLEL_WORKERS}, ${RETRY_COUNT}, ${TEST_TIMEOUT})
    
    // Get test scenario
    var tests []TestFunc
    switch "${scenario}" {
    case "all":
        tests = GetAllScenarios()
    case "load":
        tests = GetLoadTestScenario()
    case "security":
        tests = GetSecurityScanScenario()
    case "monitoring":
        tests = GetMonitoringPipelineScenario()
    case "alerts":
        tests = GetAlertTestScenario()
    case "integration":
        tests = GetFullStackIntegrationScenario()
    default:
        t.Fatalf("Unknown scenario: ${scenario}")
    }
    
    // Run tests
    report, err := runner.Run(tests)
    if err != nil {
        t.Fatalf("Failed to run tests: %v", err)
    }
    
    // Save reports
    if err := runner.SaveReport(report, "${report_file}"); err != nil {
        t.Errorf("Failed to save JSON report: %v", err)
    }
    
    if err := runner.SaveHTMLReport(report, "${html_report}"); err != nil {
        t.Errorf("Failed to save HTML report: %v", err)
    }
    
    // Print summary
    runner.PrintSummary(report)
    
    // Save resource metrics
    report.Resources.ExportMetrics("${REPORT_DIR}/resource-metrics-${scenario}-${timestamp}.csv")
    
    // Fail if there were failed tests
    if report.FailedTests > 0 {
        t.Fatalf("%d tests failed", report.FailedTests)
    }
}
EOF

    # Run the test with logging
    go test -v -tags=e2e -timeout="${TEST_TIMEOUT}" -run="TestParallelRunner_${scenario}" . 2>&1 | tee "${log_file}"
    
    local exit_code=${PIPESTATUS[0]}
    
    # Clean up test file
    rm -f "${PROJECT_ROOT}/parallel_runner_test.go"
    
    if [ $exit_code -eq 0 ]; then
        print_status $GREEN "✓ ${scenario} tests completed successfully"
        print_status $GREEN "Reports saved to:"
        print_status $GREEN "  - JSON: ${report_file}"
        print_status $GREEN "  - HTML: ${html_report}"
        print_status $GREEN "  - Log: ${log_file}"
    else
        print_status $RED "✗ ${scenario} tests failed"
        print_status $RED "Check logs at: ${log_file}"
        return $exit_code
    fi
}

# Function to cleanup
cleanup() {
    print_status $YELLOW "Cleaning up..."
    
    if [ "${KEEP_SERVICES_RUNNING}" != "true" ]; then
        cd "${PROJECT_ROOT}"
        docker-compose -f docker-compose.test.yml down -v
    fi
    
    # Archive old reports
    if [ -d "${REPORT_DIR}" ]; then
        find "${REPORT_DIR}" -name "*.json" -mtime +7 -delete
        find "${REPORT_DIR}" -name "*.html" -mtime +7 -delete
    fi
    
    # Archive old logs
    if [ -d "${LOG_DIR}" ]; then
        find "${LOG_DIR}" -name "*.log" -mtime +7 -delete
    fi
}

# Function to generate consolidated report
generate_consolidated_report() {
    print_status $BLUE "Generating consolidated test report..."
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local consolidated_report="${REPORT_DIR}/consolidated-report-${timestamp}.html"
    
    # Create a simple consolidated HTML report
    cat > "${consolidated_report}" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>E2E Test Consolidated Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .scenario { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .passed { background-color: #eeffee; }
        .failed { background-color: #ffeeee; }
        .link { color: #0066cc; text-decoration: none; }
        .link:hover { text-decoration: underline; }
        table { border-collapse: collapse; width: 100%; margin-top: 10px; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h1>E2E Test Consolidated Report</h1>
        <p>Generated: $(date)</p>
        <p>Test Environment: ${TEST_ENV:-development}</p>
    </div>
    
    <h2>Test Scenarios</h2>
EOF

    # Add links to individual reports
    for report in "${REPORT_DIR}"/test-report-*.html; do
        if [ -f "$report" ]; then
            local filename=$(basename "$report")
            echo "<div class='scenario'>" >> "${consolidated_report}"
            echo "<a class='link' href='${filename}'>${filename}</a>" >> "${consolidated_report}"
            echo "</div>" >> "${consolidated_report}"
        fi
    done
    
    echo "</body></html>" >> "${consolidated_report}"
    
    print_status $GREEN "Consolidated report saved to: ${consolidated_report}"
}

# Function to send notifications
send_notification() {
    local status=$1
    local message=$2
    
    # Slack notification (if configured)
    if [ -n "${SLACK_WEBHOOK_URL}" ]; then
        local color="good"
        if [ "$status" = "failed" ]; then
            color="danger"
        fi
        
        curl -X POST "${SLACK_WEBHOOK_URL}" \
            -H 'Content-Type: application/json' \
            -d "{
                \"attachments\": [{
                    \"color\": \"${color}\",
                    \"title\": \"E2E Test Results\",
                    \"text\": \"${message}\",
                    \"timestamp\": \"$(date +%s)\"
                }]
            }" 2>/dev/null || true
    fi
    
    # Email notification (if configured)
    if [ -n "${EMAIL_RECIPIENT}" ]; then
        echo "${message}" | mail -s "E2E Test Results - ${status}" "${EMAIL_RECIPIENT}" 2>/dev/null || true
    fi
}

# Main execution
main() {
    local scenario=${1:-all}
    local start_time=$(date +%s)
    
    print_status $BLUE "========================================"
    print_status $BLUE "E2E Parallel Test Runner"
    print_status $BLUE "========================================"
    print_status $BLUE "Scenario: ${scenario}"
    print_status $BLUE "Workers: ${PARALLEL_WORKERS}"
    print_status $BLUE "Retry Count: ${RETRY_COUNT}"
    print_status $BLUE "Timeout: ${TEST_TIMEOUT}"
    print_status $BLUE "========================================"
    
    # Check prerequisites
    check_prerequisites
    
    # Start services if needed
    if [ "${SKIP_SERVICE_START}" != "true" ]; then
        start_services
    fi
    
    # Run tests
    local exit_code=0
    
    case "$scenario" in
        all)
            # Run all scenarios
            for s in load security monitoring alerts integration; do
                run_scenario "$s" || exit_code=$?
            done
            ;;
        load|security|monitoring|alerts|integration)
            run_scenario "$scenario" || exit_code=$?
            ;;
        *)
            print_status $RED "Unknown scenario: $scenario"
            print_status $YELLOW "Available scenarios: all, load, security, monitoring, alerts, integration"
            exit 1
            ;;
    esac
    
    # Generate consolidated report
    generate_consolidated_report
    
    # Calculate duration
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    local minutes=$((duration / 60))
    local seconds=$((duration % 60))
    
    # Send notifications
    if [ $exit_code -eq 0 ]; then
        send_notification "passed" "All E2E tests passed in ${minutes}m ${seconds}s"
    else
        send_notification "failed" "E2E tests failed after ${minutes}m ${seconds}s"
    fi
    
    # Cleanup
    cleanup
    
    print_status $BLUE "========================================"
    print_status $BLUE "Test execution completed in ${minutes}m ${seconds}s"
    
    if [ $exit_code -eq 0 ]; then
        print_status $GREEN "All tests passed!"
    else
        print_status $RED "Some tests failed!"
    fi
    
    print_status $BLUE "========================================"
    
    exit $exit_code
}

# Handle script arguments
case "${1}" in
    -h|--help)
        echo "Usage: $0 [scenario]"
        echo ""
        echo "Scenarios:"
        echo "  all         - Run all test scenarios"
        echo "  load        - Run load testing scenario"
        echo "  security    - Run security testing scenario"
        echo "  monitoring  - Run monitoring pipeline tests"
        echo "  alerts      - Run alert testing scenario"
        echo "  integration - Run full integration tests"
        echo ""
        echo "Environment Variables:"
        echo "  PARALLEL_WORKERS        - Number of parallel workers (default: 4)"
        echo "  RETRY_COUNT            - Number of retries for failed tests (default: 2)"
        echo "  TEST_TIMEOUT           - Test timeout (default: 30m)"
        echo "  SKIP_SERVICE_START     - Skip starting services (default: false)"
        echo "  KEEP_SERVICES_RUNNING  - Keep services running after tests (default: false)"
        echo "  SLACK_WEBHOOK_URL      - Slack webhook for notifications"
        echo "  EMAIL_RECIPIENT        - Email address for notifications"
        echo "  TEST_ENV               - Test environment name"
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac