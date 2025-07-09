#!/bin/bash

# Load Testing Automation Script for GoFiber Application
# This script runs comprehensive load tests and generates detailed reports

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TESTS_DIR="$PROJECT_ROOT/tests/load"
RESULTS_DIR="$PROJECT_ROOT/results/load-tests"
REPORTS_DIR="$PROJECT_ROOT/reports/load-tests"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Default configuration
BASE_URL="${BASE_URL:-http://localhost:3000}"
K6_VERSION="${K6_VERSION:-latest}"
CONCURRENT_TESTS="${CONCURRENT_TESTS:-false}"
GENERATE_HTML_REPORT="${GENERATE_HTML_REPORT:-true}"
SEND_SLACK_NOTIFICATION="${SEND_SLACK_NOTIFICATION:-false}"
SLACK_WEBHOOK_URL="${SLACK_WEBHOOK_URL:-}"

# Test configuration
BASIC_LOAD_TEST="$TESTS_DIR/k6-basic-load.js"
STRESS_TEST="$TESTS_DIR/k6-stress-test.js"
SPIKE_TEST="$TESTS_DIR/k6-spike-test.js"
API_ENDPOINTS_TEST="$TESTS_DIR/api-endpoints-test.js"

# Functions
print_header() {
    echo -e "${BLUE}=================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}=================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

check_dependencies() {
    print_header "Checking Dependencies"
    
    # Check if k6 is installed
    if ! command -v k6 &> /dev/null; then
        print_error "k6 is not installed. Please install k6 first."
        print_info "Visit: https://k6.io/docs/getting-started/installation/"
        exit 1
    fi
    
    k6_version=$(k6 version | head -n 1)
    print_success "k6 is installed: $k6_version"
    
    # Check if jq is installed (for JSON processing)
    if ! command -v jq &> /dev/null; then
        print_warning "jq is not installed. Some features may be limited."
        print_info "Install jq with: brew install jq (macOS) or apt-get install jq (Ubuntu)"
    else
        print_success "jq is available"
    fi
    
    # Check if curl is available
    if ! command -v curl &> /dev/null; then
        print_warning "curl is not installed. Service health checks may fail."
    else
        print_success "curl is available"
    fi
}

create_directories() {
    print_header "Creating Result Directories"
    
    mkdir -p "$RESULTS_DIR/$TIMESTAMP"
    mkdir -p "$REPORTS_DIR/$TIMESTAMP"
    
    print_success "Created results directory: $RESULTS_DIR/$TIMESTAMP"
    print_success "Created reports directory: $REPORTS_DIR/$TIMESTAMP"
}

check_service_health() {
    print_header "Checking Service Health"
    
    print_info "Testing service availability at: $BASE_URL"
    
    # Test health endpoint
    if curl -s -f "$BASE_URL/health" > /dev/null 2>&1; then
        print_success "Service is healthy and responding"
    else
        print_error "Service is not responding at $BASE_URL"
        print_info "Please ensure your GoFiber application is running"
        exit 1
    fi
    
    # Test basic API endpoints
    endpoints=("/api/users" "/api/products" "/api/orders")
    for endpoint in "${endpoints[@]}"; do
        if curl -s -f "$BASE_URL$endpoint" > /dev/null 2>&1; then
            print_success "Endpoint $endpoint is accessible"
        else
            print_warning "Endpoint $endpoint returned an error (this may be expected)"
        fi
    done
}

run_test() {
    local test_name=$1
    local test_file=$2
    local test_description=$3
    
    print_header "Running $test_name"
    print_info "$test_description"
    
    local output_file="$RESULTS_DIR/$TIMESTAMP/${test_name,,}.json"
    local summary_file="$RESULTS_DIR/$TIMESTAMP/${test_name,,}_summary.json"
    
    # Run the test
    local start_time=$(date +%s)
    
    if k6 run \
        --env BASE_URL="$BASE_URL" \
        --out json="$output_file" \
        --summary-export="$summary_file" \
        "$test_file"; then
        
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        print_success "$test_name completed successfully in ${duration}s"
        
        # Extract key metrics if jq is available
        if command -v jq &> /dev/null && [ -f "$summary_file" ]; then
            local avg_duration=$(jq -r '.metrics.http_req_duration.avg // "N/A"' "$summary_file")
            local p95_duration=$(jq -r '.metrics.http_req_duration.p95 // "N/A"' "$summary_file")
            local error_rate=$(jq -r '.metrics.http_req_failed.rate // "N/A"' "$summary_file")
            local total_requests=$(jq -r '.metrics.http_reqs.count // "N/A"' "$summary_file")
            
            print_info "Average response time: ${avg_duration}ms"
            print_info "95th percentile: ${p95_duration}ms"
            print_info "Error rate: ${error_rate}"
            print_info "Total requests: $total_requests"
        fi
        
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        print_error "$test_name failed after ${duration}s"
        return 1
    fi
}

run_basic_load_test() {
    run_test "Basic Load Test" "$BASIC_LOAD_TEST" "Testing normal load patterns with realistic user scenarios"
}

run_stress_test() {
    run_test "Stress Test" "$STRESS_TEST" "Testing system beyond normal capacity to identify breaking points"
}

run_spike_test() {
    run_test "Spike Test" "$SPIKE_TEST" "Testing sudden traffic spikes and auto-scaling capabilities"
}

run_api_endpoints_test() {
    run_test "API Endpoints Test" "$API_ENDPOINTS_TEST" "Comprehensive testing of all API endpoints with various payloads"
}

run_all_tests() {
    print_header "Running All Load Tests"
    
    local failed_tests=()
    
    # Run tests sequentially or concurrently based on configuration
    if [ "$CONCURRENT_TESTS" = "true" ]; then
        print_info "Running tests concurrently..."
        
        run_basic_load_test &
        pid1=$!
        
        run_api_endpoints_test &
        pid2=$!
        
        # Wait for basic tests to complete before running stress tests
        wait $pid1 || failed_tests+=("Basic Load Test")
        wait $pid2 || failed_tests+=("API Endpoints Test")
        
        # Run stress tests after basic tests
        run_stress_test &
        pid3=$!
        
        run_spike_test &
        pid4=$!
        
        wait $pid3 || failed_tests+=("Stress Test")
        wait $pid4 || failed_tests+=("Spike Test")
        
    else
        print_info "Running tests sequentially..."
        
        run_basic_load_test || failed_tests+=("Basic Load Test")
        run_api_endpoints_test || failed_tests+=("API Endpoints Test")
        run_stress_test || failed_tests+=("Stress Test")
        run_spike_test || failed_tests+=("Spike Test")
    fi
    
    # Report results
    if [ ${#failed_tests[@]} -eq 0 ]; then
        print_success "All load tests completed successfully!"
    else
        print_error "The following tests failed:"
        for test in "${failed_tests[@]}"; do
            print_error "  - $test"
        done
    fi
    
    return ${#failed_tests[@]}
}

generate_html_report() {
    print_header "Generating HTML Report"
    
    local report_file="$REPORTS_DIR/$TIMESTAMP/load-test-report.html"
    
    cat > "$report_file" << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Load Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #2c3e50; color: white; padding: 20px; border-radius: 5px; }
        .test-section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 10px; margin: 10px 0; }
        .metric { background: #f8f9fa; padding: 10px; border-radius: 3px; text-align: center; }
        .metric-value { font-size: 24px; font-weight: bold; color: #2c3e50; }
        .metric-label { font-size: 12px; color: #6c757d; }
        .success { color: #28a745; }
        .warning { color: #ffc107; }
        .error { color: #dc3545; }
        .chart-placeholder { height: 300px; background: #f8f9fa; border: 1px solid #ddd; border-radius: 3px; display: flex; align-items: center; justify-content: center; color: #6c757d; }
    </style>
</head>
<body>
    <div class="header">
        <h1>GoFiber Load Test Report</h1>
        <p>Generated on: $(date)</p>
        <p>Base URL: $BASE_URL</p>
    </div>
EOF

    # Add test results for each test
    for test_type in "basic_load_test" "stress_test" "spike_test" "api_endpoints_test"; do
        local summary_file="$RESULTS_DIR/$TIMESTAMP/${test_type}_summary.json"
        
        if [ -f "$summary_file" ]; then
            cat >> "$report_file" << EOF
    <div class="test-section">
        <h2>$(echo "$test_type" | tr '_' ' ' | sed 's/\b\w/\U&/g')</h2>
        <div class="metrics">
EOF
            
            # Add metrics if jq is available
            if command -v jq &> /dev/null; then
                local avg_duration=$(jq -r '.metrics.http_req_duration.avg // "N/A"' "$summary_file")
                local p95_duration=$(jq -r '.metrics.http_req_duration.p95 // "N/A"' "$summary_file")
                local error_rate=$(jq -r '.metrics.http_req_failed.rate // "N/A"' "$summary_file")
                local total_requests=$(jq -r '.metrics.http_reqs.count // "N/A"' "$summary_file")
                
                cat >> "$report_file" << EOF
            <div class="metric">
                <div class="metric-value">$(printf "%.2f" "$avg_duration" 2>/dev/null || echo "$avg_duration")ms</div>
                <div class="metric-label">Average Response Time</div>
            </div>
            <div class="metric">
                <div class="metric-value">$(printf "%.2f" "$p95_duration" 2>/dev/null || echo "$p95_duration")ms</div>
                <div class="metric-label">95th Percentile</div>
            </div>
            <div class="metric">
                <div class="metric-value">$(printf "%.2f%%" "$(echo "$error_rate * 100" | bc -l 2>/dev/null || echo "0")" 2>/dev/null || echo "$error_rate")</div>
                <div class="metric-label">Error Rate</div>
            </div>
            <div class="metric">
                <div class="metric-value">$total_requests</div>
                <div class="metric-label">Total Requests</div>
            </div>
EOF
            fi
            
            cat >> "$report_file" << EOF
        </div>
        <div class="chart-placeholder">
            Chart visualization would go here<br>
            (Raw data available in JSON format)
        </div>
    </div>
EOF
        fi
    done
    
    cat >> "$report_file" << EOF
    <div class="test-section">
        <h2>Test Files</h2>
        <ul>
            <li><strong>Basic Load Test:</strong> Realistic user scenarios with ramp-up patterns</li>
            <li><strong>Stress Test:</strong> Beyond normal capacity to identify breaking points</li>
            <li><strong>Spike Test:</strong> Sudden traffic spikes and auto-scaling validation</li>
            <li><strong>API Endpoints Test:</strong> Comprehensive endpoint testing with various payloads</li>
        </ul>
    </div>
    
    <div class="test-section">
        <h2>Raw Data Files</h2>
        <p>Detailed JSON results are available in: <code>$RESULTS_DIR/$TIMESTAMP/</code></p>
        <ul>
            <li>basic_load_test.json - Detailed test results</li>
            <li>basic_load_test_summary.json - Summary metrics</li>
            <li>stress_test.json - Stress test results</li>
            <li>spike_test.json - Spike test results</li>
            <li>api_endpoints_test.json - API endpoint results</li>
        </ul>
    </div>
</body>
</html>
EOF
    
    print_success "HTML report generated: $report_file"
}

send_slack_notification() {
    if [ "$SEND_SLACK_NOTIFICATION" = "true" ] && [ -n "$SLACK_WEBHOOK_URL" ]; then
        print_header "Sending Slack Notification"
        
        local message="Load tests completed for $BASE_URL at $(date)"
        
        curl -X POST -H 'Content-type: application/json' \
            --data "{\"text\":\"$message\"}" \
            "$SLACK_WEBHOOK_URL" || print_warning "Failed to send Slack notification"
    fi
}

cleanup() {
    print_header "Cleanup"
    
    # Clean up any temporary files
    find "$RESULTS_DIR" -name "*.tmp" -delete 2>/dev/null || true
    
    # Compress old results (older than 7 days)
    find "$RESULTS_DIR" -type d -name "202*" -mtime +7 -exec tar -czf {}.tar.gz {} \; -exec rm -rf {} \; 2>/dev/null || true
    
    print_success "Cleanup completed"
}

show_help() {
    cat << EOF
Load Testing Script for GoFiber Application

Usage: $0 [OPTIONS]

OPTIONS:
    -h, --help              Show this help message
    -u, --url URL           Base URL for testing (default: http://localhost:3000)
    -c, --concurrent        Run tests concurrently
    -s, --sequential        Run tests sequentially (default)
    --basic-only            Run only basic load test
    --stress-only           Run only stress test
    --spike-only            Run only spike test
    --api-only              Run only API endpoints test
    --no-html               Skip HTML report generation
    --slack-webhook URL     Slack webhook URL for notifications
    --cleanup-only          Only run cleanup
    --check-health          Only check service health

ENVIRONMENT VARIABLES:
    BASE_URL                Base URL for testing
    CONCURRENT_TESTS        Run tests concurrently (true/false)
    GENERATE_HTML_REPORT    Generate HTML report (true/false)
    SEND_SLACK_NOTIFICATION Send Slack notification (true/false)
    SLACK_WEBHOOK_URL       Slack webhook URL

EXAMPLES:
    $0                      Run all tests with default settings
    $0 -u http://localhost:8080 -c
                           Run all tests concurrently against localhost:8080
    $0 --basic-only        Run only basic load test
    $0 --cleanup-only      Clean up old test results
EOF
}

main() {
    local run_basic=true
    local run_stress=true
    local run_spike=true
    local run_api=true
    local cleanup_only=false
    local check_health_only=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -u|--url)
                BASE_URL="$2"
                shift 2
                ;;
            -c|--concurrent)
                CONCURRENT_TESTS="true"
                shift
                ;;
            -s|--sequential)
                CONCURRENT_TESTS="false"
                shift
                ;;
            --basic-only)
                run_basic=true
                run_stress=false
                run_spike=false
                run_api=false
                shift
                ;;
            --stress-only)
                run_basic=false
                run_stress=true
                run_spike=false
                run_api=false
                shift
                ;;
            --spike-only)
                run_basic=false
                run_stress=false
                run_spike=true
                run_api=false
                shift
                ;;
            --api-only)
                run_basic=false
                run_stress=false
                run_spike=false
                run_api=true
                shift
                ;;
            --no-html)
                GENERATE_HTML_REPORT="false"
                shift
                ;;
            --slack-webhook)
                SLACK_WEBHOOK_URL="$2"
                SEND_SLACK_NOTIFICATION="true"
                shift 2
                ;;
            --cleanup-only)
                cleanup_only=true
                shift
                ;;
            --check-health)
                check_health_only=true
                shift
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # Banner
    print_header "GoFiber Load Testing Suite"
    print_info "Base URL: $BASE_URL"
    print_info "Timestamp: $TIMESTAMP"
    print_info "Results Dir: $RESULTS_DIR/$TIMESTAMP"
    
    if [ "$cleanup_only" = true ]; then
        cleanup
        exit 0
    fi
    
    if [ "$check_health_only" = true ]; then
        check_service_health
        exit 0
    fi
    
    # Main execution
    check_dependencies
    create_directories
    check_service_health
    
    local failed_tests=0
    
    # Run selected tests
    if [ "$run_basic" = true ]; then
        run_basic_load_test || ((failed_tests++))
    fi
    
    if [ "$run_api" = true ]; then
        run_api_endpoints_test || ((failed_tests++))
    fi
    
    if [ "$run_stress" = true ]; then
        run_stress_test || ((failed_tests++))
    fi
    
    if [ "$run_spike" = true ]; then
        run_spike_test || ((failed_tests++))
    fi
    
    # Generate reports
    if [ "$GENERATE_HTML_REPORT" = "true" ]; then
        generate_html_report
    fi
    
    # Send notifications
    send_slack_notification
    
    # Cleanup
    cleanup
    
    # Final summary
    print_header "Test Suite Summary"
    if [ $failed_tests -eq 0 ]; then
        print_success "All tests completed successfully!"
        print_info "Results available in: $RESULTS_DIR/$TIMESTAMP"
        if [ "$GENERATE_HTML_REPORT" = "true" ]; then
            print_info "HTML report: $REPORTS_DIR/$TIMESTAMP/load-test-report.html"
        fi
    else
        print_error "$failed_tests test(s) failed"
        exit 1
    fi
}

# Handle script interruption
trap cleanup EXIT

# Run main function
main "$@"