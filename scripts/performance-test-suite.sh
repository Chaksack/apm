#!/bin/bash

# Comprehensive Monitoring Stack Performance Test Suite
# This script orchestrates performance testing for Prometheus, Grafana, and Loki

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
RESULTS_DIR="${PROJECT_DIR}/performance-test-results"
LOG_FILE="${RESULTS_DIR}/performance-test-$(date +%Y%m%d_%H%M%S).log"

# Default URLs
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
LOKI_URL="${LOKI_URL:-http://localhost:3100}"

# Test configuration
RUN_PROMETHEUS_TESTS="${RUN_PROMETHEUS_TESTS:-true}"
RUN_GRAFANA_TESTS="${RUN_GRAFANA_TESTS:-true}"
RUN_LOKI_TESTS="${RUN_LOKI_TESTS:-true}"
RUN_LOAD_TESTS="${RUN_LOAD_TESTS:-true}"
GENERATE_REPORT="${GENERATE_REPORT:-true}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Utility functions
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}[ERROR] $1${NC}" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}" | tee -a "$LOG_FILE"
}

warning() {
    echo -e "${YELLOW}[WARNING] $1${NC}" | tee -a "$LOG_FILE"
}

# Setup functions
setup_environment() {
    log "Setting up performance test environment..."
    
    # Create results directory
    mkdir -p "$RESULTS_DIR"
    
    # Create log file
    touch "$LOG_FILE"
    
    # Check required tools
    check_dependencies
    
    # Verify services are running
    verify_services
    
    success "Environment setup complete"
}

check_dependencies() {
    log "Checking dependencies..."
    
    local missing_deps=()
    
    # Check Node.js for Prometheus tests
    if [[ "$RUN_PROMETHEUS_TESTS" == "true" ]] && ! command -v node &> /dev/null; then
        missing_deps+=("node")
    fi
    
    # Check Python for Grafana tests
    if [[ "$RUN_GRAFANA_TESTS" == "true" ]] && ! command -v python3 &> /dev/null; then
        missing_deps+=("python3")
    fi
    
    # Check Go for Loki tests
    if [[ "$RUN_LOKI_TESTS" == "true" ]] && ! command -v go &> /dev/null; then
        missing_deps+=("go")
    fi
    
    # Check kubectl for load tests
    if [[ "$RUN_LOAD_TESTS" == "true" ]] && ! command -v kubectl &> /dev/null; then
        missing_deps+=("kubectl")
    fi
    
    # Check jq for JSON processing
    if ! command -v jq &> /dev/null; then
        missing_deps+=("jq")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        error "Missing dependencies: ${missing_deps[*]}"
        exit 1
    fi
    
    success "All dependencies are available"
}

verify_services() {
    log "Verifying monitoring services..."
    
    local service_errors=()
    
    # Check Prometheus
    if [[ "$RUN_PROMETHEUS_TESTS" == "true" ]]; then
        if ! curl -s "${PROMETHEUS_URL}/api/v1/status/config" > /dev/null; then
            service_errors+=("Prometheus at ${PROMETHEUS_URL}")
        fi
    fi
    
    # Check Grafana
    if [[ "$RUN_GRAFANA_TESTS" == "true" ]]; then
        if ! curl -s "${GRAFANA_URL}/api/health" > /dev/null; then
            service_errors+=("Grafana at ${GRAFANA_URL}")
        fi
    fi
    
    # Check Loki
    if [[ "$RUN_LOKI_TESTS" == "true" ]]; then
        if ! curl -s "${LOKI_URL}/ready" > /dev/null; then
            service_errors+=("Loki at ${LOKI_URL}")
        fi
    fi
    
    if [ ${#service_errors[@]} -ne 0 ]; then
        error "Cannot connect to services: ${service_errors[*]}"
        exit 1
    fi
    
    success "All services are accessible"
}

# Test execution functions
run_prometheus_tests() {
    if [[ "$RUN_PROMETHEUS_TESTS" != "true" ]]; then
        return 0
    fi
    
    log "Running Prometheus performance tests..."
    
    cd "${PROJECT_DIR}/tests/performance"
    
    # Install Node.js dependencies if package.json exists
    if [[ -f "package.json" ]]; then
        npm install
    else
        # Install required packages
        npm init -y
        npm install axios
    fi
    
    # Run Prometheus tests
    if node prometheus-performance.js; then
        success "Prometheus tests completed successfully"
        
        # Move results to results directory
        if [[ -f "prometheus-performance-results.json" ]]; then
            mv prometheus-performance-results.json "$RESULTS_DIR/"
        fi
    else
        error "Prometheus tests failed"
        return 1
    fi
}

run_grafana_tests() {
    if [[ "$RUN_GRAFANA_TESTS" != "true" ]]; then
        return 0
    fi
    
    log "Running Grafana performance tests..."
    
    cd "${PROJECT_DIR}/tests/performance"
    
    # Install Python dependencies
    if ! python3 -c "import aiohttp, psutil" 2>/dev/null; then
        pip3 install aiohttp psutil
    fi
    
    # Run Grafana tests
    if python3 grafana-performance.py --url "$GRAFANA_URL"; then
        success "Grafana tests completed successfully"
        
        # Move results to results directory
        if [[ -f "grafana-performance-results.json" ]]; then
            mv grafana-performance-results.json "$RESULTS_DIR/"
        fi
    else
        error "Grafana tests failed"
        return 1
    fi
}

run_loki_tests() {
    if [[ "$RUN_LOKI_TESTS" != "true" ]]; then
        return 0
    fi
    
    log "Running Loki performance tests..."
    
    cd "${PROJECT_DIR}/tests/performance"
    
    # Initialize Go module if not exists
    if [[ ! -f "go.mod" ]]; then
        go mod init loki-performance-test
    fi
    
    # Run Loki tests
    if go run loki-performance.go "$LOKI_URL"; then
        success "Loki tests completed successfully"
        
        # Move results to results directory
        if [[ -f "loki-performance-results.json" ]]; then
            mv loki-performance-results.json "$RESULTS_DIR/"
        fi
    else
        error "Loki tests failed"
        return 1
    fi
}

run_load_tests() {
    if [[ "$RUN_LOAD_TESTS" != "true" ]]; then
        return 0
    fi
    
    log "Running monitoring stack load tests..."
    
    cd "${PROJECT_DIR}/tests/performance"
    
    # Apply Kubernetes manifests
    if kubectl apply -f monitoring-stack-load.yaml; then
        success "Load test resources deployed"
        
        # Wait for deployment to be ready
        log "Waiting for monitoring stack to be ready..."
        kubectl wait --for=condition=available --timeout=300s deployment/prometheus-load-test -n performance-testing
        kubectl wait --for=condition=available --timeout=300s deployment/grafana-load-test -n performance-testing
        kubectl wait --for=condition=available --timeout=300s deployment/loki-load-test -n performance-testing
        
        # Run load tests
        log "Starting load tests..."
        kubectl wait --for=condition=complete --timeout=1800s job/monitoring-stack-load-test -n performance-testing
        
        # Get load test results
        kubectl logs job/monitoring-stack-load-test -n performance-testing > "$RESULTS_DIR/load-test-results.log"
        
        # Get resource monitoring results
        kubectl logs job/resource-monitor -n performance-testing > "$RESULTS_DIR/resource-monitor-results.log"
        
        success "Load tests completed successfully"
        
        # Cleanup
        if [[ "${CLEANUP_AFTER_TESTS:-true}" == "true" ]]; then
            log "Cleaning up load test resources..."
            kubectl delete -f monitoring-stack-load.yaml
        fi
    else
        error "Load tests failed"
        return 1
    fi
}

collect_system_metrics() {
    log "Collecting system metrics..."
    
    local metrics_file="$RESULTS_DIR/system-metrics.json"
    
    # Collect system information
    {
        echo "{"
        echo "  \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\","
        echo "  \"hostname\": \"$(hostname)\","
        echo "  \"os\": \"$(uname -s)\","
        echo "  \"arch\": \"$(uname -m)\","
        echo "  \"kernel\": \"$(uname -r)\","
        echo "  \"cpu_info\": {"
        echo "    \"cores\": $(nproc),"
        echo "    \"model\": \"$(cat /proc/cpuinfo | grep 'model name' | head -1 | cut -d':' -f2 | xargs)\""
        echo "  },"
        echo "  \"memory\": {"
        echo "    \"total\": $(free -b | grep '^Mem:' | awk '{print $2}'),"
        echo "    \"available\": $(free -b | grep '^Mem:' | awk '{print $7}'),"
        echo "    \"used\": $(free -b | grep '^Mem:' | awk '{print $3}')"
        echo "  },"
        echo "  \"disk\": {"
        echo "    \"total\": $(df -B1 / | tail -1 | awk '{print $2}'),"
        echo "    \"used\": $(df -B1 / | tail -1 | awk '{print $3}'),"
        echo "    \"available\": $(df -B1 / | tail -1 | awk '{print $4}')"
        echo "  },"
        echo "  \"load_average\": {"
        echo "    \"1min\": $(uptime | awk -F'load average:' '{print $2}' | cut -d',' -f1 | xargs),"
        echo "    \"5min\": $(uptime | awk -F'load average:' '{print $2}' | cut -d',' -f2 | xargs),"
        echo "    \"15min\": $(uptime | awk -F'load average:' '{print $2}' | cut -d',' -f3 | xargs)"
        echo "  }"
        echo "}"
    } > "$metrics_file"
    
    success "System metrics collected"
}

generate_performance_report() {
    if [[ "$GENERATE_REPORT" != "true" ]]; then
        return 0
    fi
    
    log "Generating comprehensive performance report..."
    
    local report_file="$RESULTS_DIR/performance-report.html"
    
    # Generate HTML report
    cat > "$report_file" << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Monitoring Stack Performance Report</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            padding: 30px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            color: #2c3e50;
            text-align: center;
            margin-bottom: 30px;
        }
        h2 {
            color: #34495e;
            border-bottom: 2px solid #3498db;
            padding-bottom: 10px;
        }
        .metric-card {
            background: #f8f9fa;
            border: 1px solid #dee2e6;
            border-radius: 8px;
            padding: 20px;
            margin: 15px 0;
        }
        .metric-title {
            font-weight: bold;
            color: #495057;
            margin-bottom: 10px;
        }
        .metric-value {
            font-size: 24px;
            color: #28a745;
            font-weight: bold;
        }
        .error {
            color: #dc3545;
        }
        .warning {
            color: #ffc107;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        th, td {
            border: 1px solid #ddd;
            padding: 12px;
            text-align: left;
        }
        th {
            background-color: #f2f2f2;
            font-weight: bold;
        }
        .timestamp {
            color: #6c757d;
            font-size: 0.9em;
        }
        .summary {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 30px;
        }
        .chart-container {
            margin: 20px 0;
            padding: 20px;
            background: #f8f9fa;
            border-radius: 8px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Monitoring Stack Performance Report</h1>
        
        <div class="summary">
            <h2 style="color: white; border-bottom: 2px solid white;">Executive Summary</h2>
            <p>This report provides a comprehensive analysis of the monitoring stack performance including Prometheus, Grafana, and Loki components.</p>
            <p class="timestamp">Generated on: $(date)</p>
        </div>
        
        <div id="system-info">
            <h2>System Information</h2>
            <div class="metric-card">
                <div class="metric-title">Test Environment</div>
                <p>Hostname: $(hostname)</p>
                <p>OS: $(uname -s) $(uname -r)</p>
                <p>Architecture: $(uname -m)</p>
                <p>CPU Cores: $(nproc)</p>
                <p>Memory: $(free -h | grep '^Mem:' | awk '{print $2}') total</p>
            </div>
        </div>
        
        <div id="test-results">
            <h2>Test Results Summary</h2>
            <div class="metric-card">
                <div class="metric-title">Test Execution Status</div>
                <table>
                    <tr>
                        <th>Component</th>
                        <th>Status</th>
                        <th>Duration</th>
                        <th>Key Metrics</th>
                    </tr>
EOF

    # Add test results to report
    if [[ -f "$RESULTS_DIR/prometheus-performance-results.json" ]]; then
        echo "                    <tr><td>Prometheus</td><td class='metric-value'>✓ Completed</td><td>$(stat -c %Y "$RESULTS_DIR/prometheus-performance-results.json" | xargs -I {} date -d @{} '+%H:%M:%S')</td><td>Query Performance, High Cardinality</td></tr>" >> "$report_file"
    fi
    
    if [[ -f "$RESULTS_DIR/grafana-performance-results.json" ]]; then
        echo "                    <tr><td>Grafana</td><td class='metric-value'>✓ Completed</td><td>$(stat -c %Y "$RESULTS_DIR/grafana-performance-results.json" | xargs -I {} date -d @{} '+%H:%M:%S')</td><td>Dashboard Rendering, Concurrent Users</td></tr>" >> "$report_file"
    fi
    
    if [[ -f "$RESULTS_DIR/loki-performance-results.json" ]]; then
        echo "                    <tr><td>Loki</td><td class='metric-value'>✓ Completed</td><td>$(stat -c %Y "$RESULTS_DIR/loki-performance-results.json" | xargs -I {} date -d @{} '+%H:%M:%S')</td><td>Log Ingestion, Query Performance</td></tr>" >> "$report_file"
    fi
    
    if [[ -f "$RESULTS_DIR/load-test-results.log" ]]; then
        echo "                    <tr><td>Load Tests</td><td class='metric-value'>✓ Completed</td><td>$(stat -c %Y "$RESULTS_DIR/load-test-results.log" | xargs -I {} date -d @{} '+%H:%M:%S')</td><td>Full Stack Load Testing</td></tr>" >> "$report_file"
    fi
    
    # Close the report
    cat >> "$report_file" << 'EOF'
                </table>
            </div>
        </div>
        
        <div id="detailed-results">
            <h2>Detailed Results</h2>
            <p>Detailed test results are available in the following files:</p>
            <ul>
                <li><strong>prometheus-performance-results.json</strong> - Prometheus query performance and metrics</li>
                <li><strong>grafana-performance-results.json</strong> - Grafana dashboard and API performance</li>
                <li><strong>loki-performance-results.json</strong> - Loki log ingestion and query performance</li>
                <li><strong>load-test-results.log</strong> - Full stack load testing results</li>
                <li><strong>system-metrics.json</strong> - System resource utilization</li>
            </ul>
        </div>
        
        <div id="recommendations">
            <h2>Performance Recommendations</h2>
            <div class="metric-card">
                <div class="metric-title">Optimization Opportunities</div>
                <ul>
                    <li>Monitor query response times and optimize slow queries</li>
                    <li>Implement proper resource limits and requests</li>
                    <li>Consider horizontal scaling for high-load scenarios</li>
                    <li>Regular monitoring of storage efficiency and cleanup</li>
                    <li>Implement caching strategies for frequently accessed data</li>
                </ul>
            </div>
        </div>
    </div>
</body>
</html>
EOF

    success "Performance report generated: $report_file"
}

# Main execution
main() {
    log "Starting monitoring stack performance test suite..."
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --prometheus-url)
                PROMETHEUS_URL="$2"
                shift 2
                ;;
            --grafana-url)
                GRAFANA_URL="$2"
                shift 2
                ;;
            --loki-url)
                LOKI_URL="$2"
                shift 2
                ;;
            --skip-prometheus)
                RUN_PROMETHEUS_TESTS=false
                shift
                ;;
            --skip-grafana)
                RUN_GRAFANA_TESTS=false
                shift
                ;;
            --skip-loki)
                RUN_LOKI_TESTS=false
                shift
                ;;
            --skip-load)
                RUN_LOAD_TESTS=false
                shift
                ;;
            --no-report)
                GENERATE_REPORT=false
                shift
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  --prometheus-url URL    Prometheus URL (default: http://localhost:9090)"
                echo "  --grafana-url URL       Grafana URL (default: http://localhost:3000)"
                echo "  --loki-url URL          Loki URL (default: http://localhost:3100)"
                echo "  --skip-prometheus       Skip Prometheus tests"
                echo "  --skip-grafana          Skip Grafana tests"
                echo "  --skip-loki             Skip Loki tests"
                echo "  --skip-load             Skip load tests"
                echo "  --no-report             Skip report generation"
                echo "  --help                  Show this help message"
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Setup environment
    setup_environment
    
    # Collect system metrics
    collect_system_metrics
    
    # Run tests
    local test_failures=0
    
    if ! run_prometheus_tests; then
        ((test_failures++))
    fi
    
    if ! run_grafana_tests; then
        ((test_failures++))
    fi
    
    if ! run_loki_tests; then
        ((test_failures++))
    fi
    
    if ! run_load_tests; then
        ((test_failures++))
    fi
    
    # Generate report
    generate_performance_report
    
    # Summary
    log "Performance test suite completed"
    log "Results directory: $RESULTS_DIR"
    log "Log file: $LOG_FILE"
    
    if [[ $test_failures -eq 0 ]]; then
        success "All tests completed successfully"
        exit 0
    else
        error "$test_failures test(s) failed"
        exit 1
    fi
}

# Run main function
main "$@"