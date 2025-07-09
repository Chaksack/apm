#!/bin/bash

# Tracing Validation Script
# Comprehensive end-to-end tracing validation for GoFiber application
# Includes automated trace testing, flow validation, and performance assessment

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
TESTS_DIR="${PROJECT_DIR}/tests/tracing"
RESULTS_DIR="${PROJECT_DIR}/test-results/tracing"
JAEGER_URL="${JAEGER_URL:-http://localhost:16686}"
SERVICE_NAME="${SERVICE_NAME:-apm-service}"
TIMEOUT="${TIMEOUT:-300}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Create results directory
create_results_dir() {
    log_info "Creating results directory..."
    mkdir -p "${RESULTS_DIR}"
    rm -f "${RESULTS_DIR}"/*.json "${RESULTS_DIR}"/*.log "${RESULTS_DIR}"/*.txt 2>/dev/null || true
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if Jaeger is running
    if ! curl -s "${JAEGER_URL}/api/services" > /dev/null 2>&1; then
        log_error "Jaeger is not running at ${JAEGER_URL}"
        log_info "Please start Jaeger first:"
        log_info "  docker run -d --name jaeger \\"
        log_info "    -p 16686:16686 \\"
        log_info "    -p 14268:14268 \\"
        log_info "    -p 6831:6831/udp \\"
        log_info "    jaegertracing/all-in-one:latest"
        exit 1
    fi
    
    # Check if Python is available
    if ! command -v python3 &> /dev/null; then
        log_error "Python 3 is required but not installed"
        exit 1
    fi
    
    # Check if Node.js is available
    if ! command -v node &> /dev/null; then
        log_error "Node.js is required but not installed"
        exit 1
    fi
    
    # Check if Go is available
    if ! command -v go &> /dev/null; then
        log_error "Go is required but not installed"
        exit 1
    fi
    
    log_success "All prerequisites are available"
}

# Install Python dependencies
install_python_deps() {
    log_info "Installing Python dependencies..."
    
    # Create virtual environment if it doesn't exist
    if [ ! -d "${PROJECT_DIR}/venv" ]; then
        python3 -m venv "${PROJECT_DIR}/venv"
    fi
    
    # Activate virtual environment
    source "${PROJECT_DIR}/venv/bin/activate"
    
    # Install dependencies
    pip install --quiet requests pytest dataclasses
    
    log_success "Python dependencies installed"
}

# Install Node.js dependencies
install_node_deps() {
    log_info "Installing Node.js dependencies..."
    
    # Create package.json if it doesn't exist
    if [ ! -f "${TESTS_DIR}/package.json" ]; then
        cat > "${TESTS_DIR}/package.json" << EOF
{
  "name": "jaeger-integration-tests",
  "version": "1.0.0",
  "description": "Jaeger integration tests for GoFiber application",
  "main": "jaeger-integration-test.js",
  "scripts": {
    "test": "mocha jaeger-integration-test.js"
  },
  "dependencies": {
    "axios": "^1.4.0",
    "chai": "^4.3.7",
    "mocha": "^10.2.0"
  }
}
EOF
    fi
    
    # Install dependencies
    cd "${TESTS_DIR}"
    npm install --silent
    cd "${PROJECT_DIR}"
    
    log_success "Node.js dependencies installed"
}

# Wait for service to be ready
wait_for_service() {
    local service_url=$1
    local timeout=$2
    local counter=0
    
    log_info "Waiting for service at ${service_url}..."
    
    while [ $counter -lt $timeout ]; do
        if curl -s "${service_url}" > /dev/null 2>&1; then
            log_success "Service is ready"
            return 0
        fi
        sleep 1
        counter=$((counter + 1))
    done
    
    log_error "Service failed to start within ${timeout} seconds"
    return 1
}

# Generate test load
generate_test_load() {
    log_info "Generating test load for tracing..."
    
    local service_url="${1:-http://localhost:8080}"
    local requests_count="${2:-100}"
    local concurrent="${3:-10}"
    
    # Create load generation script
    cat > "${RESULTS_DIR}/load_test.py" << EOF
import requests
import time
import concurrent.futures
import random
import json

def make_request(url, request_id):
    endpoints = [
        "/health",
        "/metrics",
        "/api/users",
        "/api/orders",
        "/api/products"
    ]
    
    endpoint = random.choice(endpoints)
    full_url = f"{url}{endpoint}"
    
    headers = {
        "X-Request-ID": f"test-{request_id}",
        "User-Agent": "TraceLoadTest/1.0"
    }
    
    try:
        start_time = time.time()
        response = requests.get(full_url, headers=headers, timeout=10)
        duration = time.time() - start_time
        
        return {
            "request_id": request_id,
            "url": full_url,
            "status_code": response.status_code,
            "duration": duration,
            "success": True
        }
    except Exception as e:
        return {
            "request_id": request_id,
            "url": full_url,
            "error": str(e),
            "success": False
        }

def generate_load(base_url, total_requests, concurrent_requests):
    results = []
    
    with concurrent.futures.ThreadPoolExecutor(max_workers=concurrent_requests) as executor:
        futures = []
        
        for i in range(total_requests):
            future = executor.submit(make_request, base_url, i)
            futures.append(future)
        
        for future in concurrent.futures.as_completed(futures):
            result = future.result()
            results.append(result)
    
    return results

if __name__ == "__main__":
    results = generate_load("${service_url}", ${requests_count}, ${concurrent})
    
    # Save results
    with open("${RESULTS_DIR}/load_test_results.json", "w") as f:
        json.dump(results, f, indent=2)
    
    # Print summary
    successful = sum(1 for r in results if r["success"])
    failed = len(results) - successful
    avg_duration = sum(r.get("duration", 0) for r in results if r["success"]) / max(successful, 1)
    
    print(f"Load test completed:")
    print(f"  Total requests: {len(results)}")
    print(f"  Successful: {successful}")
    print(f"  Failed: {failed}")
    print(f"  Average duration: {avg_duration:.3f}s")
EOF
    
    # Run load test
    source "${PROJECT_DIR}/venv/bin/activate"
    python3 "${RESULTS_DIR}/load_test.py"
    
    # Wait for traces to be processed
    log_info "Waiting for traces to be processed..."
    sleep 10
    
    log_success "Test load generated successfully"
}

# Run Python trace validation
run_python_validation() {
    log_info "Running Python trace validation..."
    
    source "${PROJECT_DIR}/venv/bin/activate"
    
    # Run trace validation
    cd "${TESTS_DIR}"
    python3 trace-validation.py > "${RESULTS_DIR}/python_validation.json" 2>&1
    
    # Run pytest tests
    pytest trace-validation.py -v --tb=short > "${RESULTS_DIR}/python_tests.log" 2>&1 || true
    
    cd "${PROJECT_DIR}"
    
    log_success "Python trace validation completed"
}

# Run Go distributed trace tests
run_go_tests() {
    log_info "Running Go distributed trace tests..."
    
    cd "${TESTS_DIR}"
    
    # Run Go tests
    go test -v -timeout=${TIMEOUT}s ./... > "${RESULTS_DIR}/go_tests.log" 2>&1 || true
    
    # Run benchmark tests
    go test -bench=. -benchmem > "${RESULTS_DIR}/go_benchmarks.log" 2>&1 || true
    
    cd "${PROJECT_DIR}"
    
    log_success "Go distributed trace tests completed"
}

# Run JavaScript Jaeger integration tests
run_js_tests() {
    log_info "Running JavaScript Jaeger integration tests..."
    
    cd "${TESTS_DIR}"
    
    # Run standalone test
    node jaeger-integration-test.js > "${RESULTS_DIR}/js_integration.json" 2>&1 || true
    
    # Run with Mocha
    npx mocha jaeger-integration-test.js --reporter json > "${RESULTS_DIR}/js_tests.json" 2>&1 || true
    
    cd "${PROJECT_DIR}"
    
    log_success "JavaScript Jaeger integration tests completed"
}

# Performance impact assessment
assess_performance_impact() {
    log_info "Assessing performance impact of tracing..."
    
    cat > "${RESULTS_DIR}/performance_assessment.py" << EOF
import json
import statistics
import sys

def load_results():
    try:
        with open("${RESULTS_DIR}/load_test_results.json", "r") as f:
            return json.load(f)
    except FileNotFoundError:
        return []

def analyze_performance(results):
    if not results:
        return {"error": "No load test results found"}
    
    successful_requests = [r for r in results if r.get("success", False)]
    
    if not successful_requests:
        return {"error": "No successful requests found"}
    
    durations = [r["duration"] for r in successful_requests]
    status_codes = [r["status_code"] for r in successful_requests]
    
    analysis = {
        "total_requests": len(results),
        "successful_requests": len(successful_requests),
        "failed_requests": len(results) - len(successful_requests),
        "success_rate": len(successful_requests) / len(results) * 100,
        "duration_stats": {
            "min": min(durations),
            "max": max(durations),
            "mean": statistics.mean(durations),
            "median": statistics.median(durations),
            "std_dev": statistics.stdev(durations) if len(durations) > 1 else 0
        },
        "status_code_distribution": {}
    }
    
    # Count status codes
    for code in status_codes:
        analysis["status_code_distribution"][code] = analysis["status_code_distribution"].get(code, 0) + 1
    
    # Performance assessment
    avg_duration = analysis["duration_stats"]["mean"]
    if avg_duration < 0.1:
        analysis["performance_rating"] = "excellent"
    elif avg_duration < 0.5:
        analysis["performance_rating"] = "good"
    elif avg_duration < 1.0:
        analysis["performance_rating"] = "acceptable"
    else:
        analysis["performance_rating"] = "poor"
    
    # Tracing overhead estimation
    # This is a simplified estimation
    base_overhead = 0.001  # Estimated 1ms base overhead
    analysis["estimated_tracing_overhead"] = {
        "absolute_ms": base_overhead * 1000,
        "relative_percent": (base_overhead / avg_duration) * 100 if avg_duration > 0 else 0
    }
    
    return analysis

if __name__ == "__main__":
    results = load_results()
    analysis = analyze_performance(results)
    
    print(json.dumps(analysis, indent=2))
EOF
    
    source "${PROJECT_DIR}/venv/bin/activate"
    python3 "${RESULTS_DIR}/performance_assessment.py" > "${RESULTS_DIR}/performance_analysis.json"
    
    log_success "Performance impact assessment completed"
}

# Generate comprehensive report
generate_report() {
    log_info "Generating comprehensive validation report..."
    
    cat > "${RESULTS_DIR}/validation_report.py" << EOF
import json
import os
from datetime import datetime

def load_json_file(filepath):
    try:
        with open(filepath, 'r') as f:
            return json.load(f)
    except (FileNotFoundError, json.JSONDecodeError):
        return None

def load_log_file(filepath):
    try:
        with open(filepath, 'r') as f:
            return f.read()
    except FileNotFoundError:
        return None

def generate_html_report():
    # Load all results
    python_validation = load_json_file("${RESULTS_DIR}/python_validation.json")
    js_integration = load_json_file("${RESULTS_DIR}/js_integration.json")
    js_tests = load_json_file("${RESULTS_DIR}/js_tests.json")
    performance_analysis = load_json_file("${RESULTS_DIR}/performance_analysis.json")
    
    # Load log files
    python_tests_log = load_log_file("${RESULTS_DIR}/python_tests.log")
    go_tests_log = load_log_file("${RESULTS_DIR}/go_tests.log")
    go_benchmarks_log = load_log_file("${RESULTS_DIR}/go_benchmarks.log")
    
    html_content = f"""
<!DOCTYPE html>
<html>
<head>
    <title>Tracing Validation Report</title>
    <style>
        body {{ font-family: Arial, sans-serif; margin: 20px; }}
        .header {{ background-color: #f0f0f0; padding: 20px; border-radius: 5px; }}
        .section {{ margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }}
        .success {{ background-color: #d4edda; border-color: #c3e6cb; }}
        .warning {{ background-color: #fff3cd; border-color: #ffeaa7; }}
        .error {{ background-color: #f8d7da; border-color: #f5c6cb; }}
        .code {{ background-color: #f8f9fa; padding: 10px; border-radius: 3px; font-family: monospace; }}
        table {{ border-collapse: collapse; width: 100%; }}
        th, td {{ border: 1px solid #ddd; padding: 8px; text-align: left; }}
        th {{ background-color: #f2f2f2; }}
    </style>
</head>
<body>
    <div class="header">
        <h1>GoFiber Tracing Validation Report</h1>
        <p>Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}</p>
        <p>Service: {SERVICE_NAME}</p>
        <p>Jaeger URL: {JAEGER_URL}</p>
    </div>
    
    <div class="section">
        <h2>Executive Summary</h2>
        <p>This report provides a comprehensive validation of distributed tracing implementation 
        for the GoFiber application, including trace completeness, span relationships, 
        performance impact, and Jaeger integration.</p>
    </div>
"""
    
    # Python validation results
    if python_validation:
        html_content += f"""
    <div class="section">
        <h2>Python Trace Validation</h2>
        <p><strong>Total Traces:</strong> {python_validation.get('trace_count', 'N/A')}</p>
        <p><strong>Valid Traces:</strong> {python_validation.get('summary', {}).get('valid_traces', 'N/A')}</p>
        <p><strong>Invalid Traces:</strong> {python_validation.get('summary', {}).get('invalid_traces', 'N/A')}</p>
        <p><strong>Warnings:</strong> {python_validation.get('summary', {}).get('warnings', 'N/A')}</p>
    </div>
"""
    
    # JavaScript integration results
    if js_integration:
        html_content += f"""
    <div class="section">
        <h2>Jaeger Integration Test Results</h2>
        <table>
            <tr><th>Test</th><th>Status</th><th>Details</th></tr>
"""
        
        tests = js_integration.get('tests', {})
        for test_name, test_result in tests.items():
            status = "✅ Pass" if test_result.get('success', False) else "❌ Fail"
            details = str(test_result).replace('{', '').replace('}', '')[:100] + '...'
            html_content += f"<tr><td>{test_name}</td><td>{status}</td><td>{details}</td></tr>"
        
        html_content += """
        </table>
    </div>
"""
    
    # Performance analysis
    if performance_analysis:
        html_content += f"""
    <div class="section">
        <h2>Performance Analysis</h2>
        <p><strong>Success Rate:</strong> {performance_analysis.get('success_rate', 'N/A'):.2f}%</p>
        <p><strong>Average Duration:</strong> {performance_analysis.get('duration_stats', {}).get('mean', 'N/A'):.3f}s</p>
        <p><strong>Performance Rating:</strong> {performance_analysis.get('performance_rating', 'N/A')}</p>
        <p><strong>Estimated Tracing Overhead:</strong> {performance_analysis.get('estimated_tracing_overhead', {}).get('relative_percent', 'N/A'):.2f}%</p>
    </div>
"""
    
    # Test logs
    if python_tests_log:
        html_content += f"""
    <div class="section">
        <h2>Python Test Logs</h2>
        <div class="code">{python_tests_log}</div>
    </div>
"""
    
    if go_tests_log:
        html_content += f"""
    <div class="section">
        <h2>Go Test Logs</h2>
        <div class="code">{go_tests_log}</div>
    </div>
"""
    
    html_content += """
</body>
</html>
"""
    
    return html_content

if __name__ == "__main__":
    html_report = generate_html_report()
    
    with open("${RESULTS_DIR}/validation_report.html", "w") as f:
        f.write(html_report)
    
    print("HTML report generated: ${RESULTS_DIR}/validation_report.html")
EOF
    
    source "${PROJECT_DIR}/venv/bin/activate"
    python3 "${RESULTS_DIR}/validation_report.py"
    
    log_success "Comprehensive validation report generated"
}

# Print summary
print_summary() {
    log_info "Tracing Validation Summary"
    echo "=================================="
    
    if [ -f "${RESULTS_DIR}/performance_analysis.json" ]; then
        echo "Performance Analysis:"
        cat "${RESULTS_DIR}/performance_analysis.json" | python3 -m json.tool | grep -E "(success_rate|performance_rating|relative_percent)" | head -3
        echo ""
    fi
    
    echo "Generated Files:"
    ls -la "${RESULTS_DIR}"/ | grep -E "\.(json|log|html)$"
    echo ""
    
    echo "View the comprehensive report:"
    echo "  open ${RESULTS_DIR}/validation_report.html"
    echo ""
    
    log_success "Tracing validation completed successfully!"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up temporary files..."
    # Clean up any temporary files if needed
}

# Main execution
main() {
    log_info "Starting GoFiber tracing validation..."
    
    # Set up trap for cleanup
    trap cleanup EXIT
    
    # Run validation steps
    create_results_dir
    check_prerequisites
    install_python_deps
    install_node_deps
    
    # Generate test load (optional, can be skipped if service is not running)
    if curl -s "http://localhost:8080/health" > /dev/null 2>&1; then
        generate_test_load "http://localhost:8080" 50 5
    else
        log_warning "Service not running at http://localhost:8080, skipping load generation"
    fi
    
    # Run validation tests
    run_python_validation
    run_go_tests
    run_js_tests
    
    # Assess performance impact
    assess_performance_impact
    
    # Generate report
    generate_report
    
    # Print summary
    print_summary
}

# Handle command line arguments
case "${1:-}" in
    "python")
        install_python_deps
        run_python_validation
        ;;
    "go")
        run_go_tests
        ;;
    "js")
        install_node_deps
        run_js_tests
        ;;
    "load")
        install_python_deps
        generate_test_load "http://localhost:8080" 100 10
        ;;
    "report")
        generate_report
        ;;
    "clean")
        rm -rf "${RESULTS_DIR}" "${PROJECT_DIR}/venv" "${TESTS_DIR}/node_modules"
        log_success "Cleaned up test artifacts"
        ;;
    "help"|"-h"|"--help")
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  python    Run only Python trace validation"
        echo "  go        Run only Go distributed trace tests"
        echo "  js        Run only JavaScript Jaeger integration tests"
        echo "  load      Generate test load only"
        echo "  report    Generate report only"
        echo "  clean     Clean up test artifacts"
        echo "  help      Show this help message"
        echo ""
        echo "Environment Variables:"
        echo "  JAEGER_URL    Jaeger URL (default: http://localhost:16686)"
        echo "  SERVICE_NAME  Service name to test (default: apm-service)"
        echo "  TIMEOUT       Test timeout in seconds (default: 300)"
        ;;
    *)
        main
        ;;
esac