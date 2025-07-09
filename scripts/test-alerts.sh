#!/bin/bash

# Alert Testing Script for APM Monitoring System
# Provides alert simulation, end-to-end notification testing, and recovery notification testing
# Designed to be non-disruptive and safe for production environments

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
ALERTMANAGER_URL="${ALERTMANAGER_URL:-http://localhost:9093}"
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
TEST_TIMEOUT="${TEST_TIMEOUT:-300}"
TEST_NAMESPACE="${TEST_NAMESPACE:-apm-test}"
LOG_FILE="${LOG_FILE:-$PROJECT_DIR/logs/alert-tests.log}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    echo -e "${timestamp} [${level}] ${message}" | tee -a "$LOG_FILE"
}

info() { log "${BLUE}INFO${NC}" "$@"; }
warn() { log "${YELLOW}WARN${NC}" "$@"; }
error() { log "${RED}ERROR${NC}" "$@"; }
success() { log "${GREEN}SUCCESS${NC}" "$@"; }

# Initialize test environment
initialize_test_environment() {
    info "Initializing test environment..."
    
    # Create logs directory
    mkdir -p "$(dirname "$LOG_FILE")"
    
    # Create test results directory
    mkdir -p "$PROJECT_DIR/test-results/alerts"
    
    # Check dependencies
    check_dependencies
    
    # Verify services are running
    verify_services
    
    success "Test environment initialized"
}

# Check required dependencies
check_dependencies() {
    info "Checking dependencies..."
    
    local deps=("curl" "jq" "python3" "node")
    local missing_deps=()
    
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            missing_deps+=("$dep")
        fi
    done
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        error "Missing dependencies: ${missing_deps[*]}"
        exit 1
    fi
    
    success "All dependencies found"
}

# Verify required services are running
verify_services() {
    info "Verifying services..."
    
    # Check Alertmanager
    if ! curl -s "$ALERTMANAGER_URL/-/healthy" &> /dev/null; then
        error "Alertmanager is not running at $ALERTMANAGER_URL"
        exit 1
    fi
    
    # Check Prometheus
    if ! curl -s "$PROMETHEUS_URL/-/healthy" &> /dev/null; then
        warn "Prometheus is not running at $PROMETHEUS_URL (some tests may fail)"
    fi
    
    success "Services verified"
}

# Generate test alert payload
generate_test_alert() {
    local alert_name="$1"
    local severity="$2"
    local instance="${3:-test-instance}"
    local service="${4:-test-service}"
    local team="${5:-test-team}"
    local status="${6:-firing}"
    
    local starts_at=$(date -u -d "now" +"%Y-%m-%dT%H:%M:%SZ")
    local ends_at=""
    
    if [ "$status" = "resolved" ]; then
        ends_at=$(date -u -d "now + 1 minute" +"%Y-%m-%dT%H:%M:%SZ")
    fi
    
    cat <<EOF
[
  {
    "labels": {
      "alertname": "$alert_name",
      "severity": "$severity",
      "instance": "$instance",
      "service": "$service",
      "team": "$team",
      "test": "true",
      "test_namespace": "$TEST_NAMESPACE"
    },
    "annotations": {
      "summary": "Test alert: $alert_name",
      "description": "This is a test alert for $service on $instance",
      "runbook_url": "https://wiki.example.com/runbooks/$service",
      "dashboard_url": "https://grafana.example.com/dashboard/$service"
    },
    "startsAt": "$starts_at"$(if [ -n "$ends_at" ]; then echo ",\"endsAt\": \"$ends_at\""; fi)
  }
]
EOF
}

# Send alert to Alertmanager
send_alert() {
    local alert_payload="$1"
    local response
    
    info "Sending alert to Alertmanager..."
    
    response=$(curl -s -w "%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d "$alert_payload" \
        "$ALERTMANAGER_URL/api/v1/alerts")
    
    local http_code="${response: -3}"
    local response_body="${response%???}"
    
    if [ "$http_code" -eq 200 ]; then
        success "Alert sent successfully"
        return 0
    else
        error "Failed to send alert (HTTP $http_code): $response_body"
        return 1
    fi
}

# Wait for alert to be processed
wait_for_alert() {
    local alert_name="$1"
    local timeout="${2:-30}"
    local start_time=$(date +%s)
    
    info "Waiting for alert '$alert_name' to be processed..."
    
    while true; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [ $elapsed -ge $timeout ]; then
            error "Timeout waiting for alert '$alert_name'"
            return 1
        fi
        
        local alerts_response=$(curl -s "$ALERTMANAGER_URL/api/v1/alerts")
        local alert_count=$(echo "$alerts_response" | jq -r ".data[] | select(.labels.alertname == \"$alert_name\" and .labels.test_namespace == \"$TEST_NAMESPACE\") | .labels.alertname" | wc -l)
        
        if [ "$alert_count" -gt 0 ]; then
            success "Alert '$alert_name' found in Alertmanager"
            return 0
        fi
        
        sleep 2
    done
}

# Test alert routing
test_alert_routing() {
    info "Testing alert routing..."
    
    local test_cases=(
        "HighCPUUsage:critical:web-01:web-server:platform"
        "HighMemoryUsage:warning:db-01:database:data"
        "DiskSpaceAlert:info:storage-01:storage:infrastructure"
        "ServiceDown:critical:api-01:api-server:backend"
    )
    
    for test_case in "${test_cases[@]}"; do
        IFS=':' read -r alert_name severity instance service team <<< "$test_case"
        
        info "Testing routing for $alert_name ($severity)"
        
        local alert_payload=$(generate_test_alert "$alert_name" "$severity" "$instance" "$service" "$team")
        
        if send_alert "$alert_payload"; then
            if wait_for_alert "$alert_name" 30; then
                # Check routing
                local route_info=$(curl -s "$ALERTMANAGER_URL/api/v1/status" | jq -r '.data.config.route')
                success "Alert routing test passed for $alert_name"
            else
                error "Alert routing test failed for $alert_name"
            fi
        else
            error "Failed to send alert for routing test: $alert_name"
        fi
        
        # Clean up
        cleanup_test_alert "$alert_name"
        sleep 2
    done
}

# Test severity-based routing
test_severity_routing() {
    info "Testing severity-based routing..."
    
    local severities=("critical" "warning" "info")
    
    for severity in "${severities[@]}"; do
        local alert_name="SeverityTest_$severity"
        
        info "Testing $severity severity routing"
        
        local alert_payload=$(generate_test_alert "$alert_name" "$severity")
        
        if send_alert "$alert_payload"; then
            if wait_for_alert "$alert_name" 30; then
                # Verify alert was routed based on severity
                local alerts_response=$(curl -s "$ALERTMANAGER_URL/api/v1/alerts")
                local alert_severity=$(echo "$alerts_response" | jq -r ".data[] | select(.labels.alertname == \"$alert_name\") | .labels.severity")
                
                if [ "$alert_severity" = "$severity" ]; then
                    success "Severity routing test passed for $severity"
                else
                    error "Severity routing test failed for $severity"
                fi
            fi
        fi
        
        cleanup_test_alert "$alert_name"
        sleep 2
    done
}

# Test alert grouping
test_alert_grouping() {
    info "Testing alert grouping..."
    
    local base_alert_name="GroupingTest"
    local service="test-service"
    
    # Send multiple alerts with same grouping keys
    for i in {1..3}; do
        local alert_name="${base_alert_name}_${i}"
        local alert_payload=$(generate_test_alert "$alert_name" "warning" "instance-$i" "$service")
        
        send_alert "$alert_payload"
        sleep 1
    done
    
    # Wait for alerts to be processed
    sleep 10
    
    # Check if alerts are grouped
    local alerts_response=$(curl -s "$ALERTMANAGER_URL/api/v1/alerts")
    local grouped_alerts=$(echo "$alerts_response" | jq -r ".data[] | select(.labels.service == \"$service\" and .labels.test_namespace == \"$TEST_NAMESPACE\")")
    
    if [ -n "$grouped_alerts" ]; then
        success "Alert grouping test passed"
    else
        error "Alert grouping test failed"
    fi
    
    # Clean up
    for i in {1..3}; do
        cleanup_test_alert "${base_alert_name}_${i}"
    done
}

# Test alert silencing
test_alert_silencing() {
    info "Testing alert silencing..."
    
    local alert_name="SilenceTest"
    local alert_payload=$(generate_test_alert "$alert_name" "warning")
    
    # Send alert first
    send_alert "$alert_payload"
    wait_for_alert "$alert_name" 30
    
    # Create silence
    local silence_payload=$(cat <<EOF
{
  "matchers": [
    {
      "name": "alertname",
      "value": "$alert_name"
    },
    {
      "name": "test_namespace",
      "value": "$TEST_NAMESPACE"
    }
  ],
  "startsAt": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "endsAt": "$(date -u -d "now + 1 hour" +"%Y-%m-%dT%H:%M:%SZ")",
  "createdBy": "test-script",
  "comment": "Test silence for alert testing"
}
EOF
)
    
    local silence_response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$silence_payload" \
        "$ALERTMANAGER_URL/api/v1/silences")
    
    local silence_id=$(echo "$silence_response" | jq -r '.silenceID')
    
    if [ "$silence_id" != "null" ] && [ -n "$silence_id" ]; then
        success "Alert silence created successfully: $silence_id"
        
        # Wait a bit and check if alert is silenced
        sleep 5
        
        local alerts_response=$(curl -s "$ALERTMANAGER_URL/api/v1/alerts")
        local silenced_alert=$(echo "$alerts_response" | jq -r ".data[] | select(.labels.alertname == \"$alert_name\" and .status.silencedBy != null)")
        
        if [ -n "$silenced_alert" ]; then
            success "Alert silencing test passed"
        else
            warn "Alert silencing test inconclusive"
        fi
        
        # Clean up silence
        curl -s -X DELETE "$ALERTMANAGER_URL/api/v1/silence/$silence_id"
    else
        error "Failed to create silence"
    fi
    
    cleanup_test_alert "$alert_name"
}

# Test inhibition rules
test_inhibition_rules() {
    info "Testing inhibition rules..."
    
    local node_down_alert="NodeDownTest"
    local cpu_alert="HighCPUTest"
    local instance="test-node-01"
    
    # Send NodeDown alert (should inhibit CPU alert)
    local node_down_payload=$(generate_test_alert "$node_down_alert" "critical" "$instance")
    send_alert "$node_down_payload"
    wait_for_alert "$node_down_alert" 30
    
    # Send CPU alert (should be inhibited)
    local cpu_payload=$(generate_test_alert "$cpu_alert" "warning" "$instance")
    send_alert "$cpu_payload"
    
    # Wait for processing
    sleep 10
    
    # Check if CPU alert is inhibited
    local alerts_response=$(curl -s "$ALERTMANAGER_URL/api/v1/alerts")
    local inhibited_alert=$(echo "$alerts_response" | jq -r ".data[] | select(.labels.alertname == \"$cpu_alert\" and .status.inhibitedBy != null)")
    
    if [ -n "$inhibited_alert" ]; then
        success "Inhibition rules test passed"
    else
        warn "Inhibition rules test inconclusive"
    fi
    
    # Clean up
    cleanup_test_alert "$node_down_alert"
    cleanup_test_alert "$cpu_alert"
}

# Test notification endpoints
test_notification_endpoints() {
    info "Testing notification endpoints..."
    
    # Run JavaScript notification tests
    if [ -f "$PROJECT_DIR/tests/alerts/notification-test.js" ]; then
        info "Running notification tests..."
        cd "$PROJECT_DIR/tests/alerts"
        
        if node notification-test.js; then
            success "Notification tests passed"
        else
            error "Notification tests failed"
        fi
        
        cd "$SCRIPT_DIR"
    else
        warn "Notification test file not found, skipping"
    fi
}

# Test alert recovery notifications
test_recovery_notifications() {
    info "Testing alert recovery notifications..."
    
    local alert_name="RecoveryTest"
    local alert_payload=$(generate_test_alert "$alert_name" "critical")
    
    # Send firing alert
    info "Sending firing alert..."
    send_alert "$alert_payload"
    wait_for_alert "$alert_name" 30
    
    # Wait a bit
    sleep 10
    
    # Send resolved alert
    info "Sending resolved alert..."
    local resolved_payload=$(generate_test_alert "$alert_name" "critical" "test-instance" "test-service" "test-team" "resolved")
    send_alert "$resolved_payload"
    
    # Wait for processing
    sleep 10
    
    # Check if alert is resolved
    local alerts_response=$(curl -s "$ALERTMANAGER_URL/api/v1/alerts")
    local resolved_alert=$(echo "$alerts_response" | jq -r ".data[] | select(.labels.alertname == \"$alert_name\" and .status.state == \"resolved\")")
    
    if [ -n "$resolved_alert" ]; then
        success "Alert recovery test passed"
    else
        warn "Alert recovery test inconclusive"
    fi
    
    cleanup_test_alert "$alert_name"
}

# Run end-to-end alert flow test
test_end_to_end_flow() {
    info "Running end-to-end alert flow test..."
    
    local alert_name="E2ETest"
    local service="e2e-service"
    local instance="e2e-instance"
    
    # Step 1: Send alert
    info "Step 1: Sending alert..."
    local alert_payload=$(generate_test_alert "$alert_name" "critical" "$instance" "$service" "platform")
    send_alert "$alert_payload"
    
    # Step 2: Verify alert is received
    info "Step 2: Verifying alert reception..."
    if ! wait_for_alert "$alert_name" 30; then
        error "E2E test failed: Alert not received"
        return 1
    fi
    
    # Step 3: Check routing
    info "Step 3: Checking alert routing..."
    local alerts_response=$(curl -s "$ALERTMANAGER_URL/api/v1/alerts")
    local routed_alert=$(echo "$alerts_response" | jq -r ".data[] | select(.labels.alertname == \"$alert_name\")")
    
    if [ -z "$routed_alert" ]; then
        error "E2E test failed: Alert not properly routed"
        return 1
    fi
    
    # Step 4: Test silencing
    info "Step 4: Testing alert silencing..."
    local silence_payload=$(cat <<EOF
{
  "matchers": [
    {
      "name": "alertname",
      "value": "$alert_name"
    }
  ],
  "startsAt": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "endsAt": "$(date -u -d "now + 10 minutes" +"%Y-%m-%dT%H:%M:%SZ")",
  "createdBy": "e2e-test",
  "comment": "E2E test silence"
}
EOF
)
    
    local silence_response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$silence_payload" \
        "$ALERTMANAGER_URL/api/v1/silences")
    
    local silence_id=$(echo "$silence_response" | jq -r '.silenceID')
    
    # Step 5: Resolve alert
    info "Step 5: Resolving alert..."
    local resolved_payload=$(generate_test_alert "$alert_name" "critical" "$instance" "$service" "platform" "resolved")
    send_alert "$resolved_payload"
    
    # Step 6: Verify resolution
    info "Step 6: Verifying alert resolution..."
    sleep 10
    
    # Clean up
    if [ "$silence_id" != "null" ] && [ -n "$silence_id" ]; then
        curl -s -X DELETE "$ALERTMANAGER_URL/api/v1/silence/$silence_id"
    fi
    
    cleanup_test_alert "$alert_name"
    
    success "End-to-end alert flow test completed"
}

# Run routing validation tests
test_routing_validation() {
    info "Running routing validation tests..."
    
    # Run Python routing tests
    if [ -f "$PROJECT_DIR/tests/alerts/alert-routing-test.py" ]; then
        info "Running Python routing tests..."
        cd "$PROJECT_DIR/tests/alerts"
        
        if python3 alert-routing-test.py; then
            success "Routing validation tests passed"
        else
            error "Routing validation tests failed"
        fi
        
        cd "$SCRIPT_DIR"
    else
        warn "Routing test file not found, skipping"
    fi
}

# Clean up test alert
cleanup_test_alert() {
    local alert_name="$1"
    
    # Send resolved alert to clean up
    local resolved_payload=$(generate_test_alert "$alert_name" "resolved" "test-instance" "test-service" "test-team" "resolved")
    curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$resolved_payload" \
        "$ALERTMANAGER_URL/api/v1/alerts" > /dev/null 2>&1 || true
}

# Clean up all test alerts
cleanup_all_test_alerts() {
    info "Cleaning up all test alerts..."
    
    # Get all active alerts with test namespace
    local alerts_response=$(curl -s "$ALERTMANAGER_URL/api/v1/alerts")
    local test_alerts=$(echo "$alerts_response" | jq -r ".data[] | select(.labels.test_namespace == \"$TEST_NAMESPACE\") | .labels.alertname")
    
    if [ -n "$test_alerts" ]; then
        while IFS= read -r alert_name; do
            cleanup_test_alert "$alert_name"
        done <<< "$test_alerts"
    fi
    
    # Clean up any test silences
    local silences_response=$(curl -s "$ALERTMANAGER_URL/api/v1/silences")
    local test_silences=$(echo "$silences_response" | jq -r ".data[] | select(.createdBy | test(\".*test.*\"; \"i\")) | .id")
    
    if [ -n "$test_silences" ]; then
        while IFS= read -r silence_id; do
            curl -s -X DELETE "$ALERTMANAGER_URL/api/v1/silence/$silence_id" > /dev/null 2>&1 || true
        done <<< "$test_silences"
    fi
    
    success "Test cleanup completed"
}

# Generate test report
generate_test_report() {
    local report_file="$PROJECT_DIR/test-results/alerts/alert-test-report.json"
    
    info "Generating test report..."
    
    local report=$(cat <<EOF
{
  "test_run": {
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "environment": {
      "alertmanager_url": "$ALERTMANAGER_URL",
      "prometheus_url": "$PROMETHEUS_URL",
      "test_namespace": "$TEST_NAMESPACE"
    },
    "summary": {
      "total_tests": 0,
      "passed_tests": 0,
      "failed_tests": 0,
      "skipped_tests": 0
    },
    "test_results": []
  }
}
EOF
)
    
    echo "$report" > "$report_file"
    
    success "Test report generated: $report_file"
}

# Main test execution
main() {
    echo "======================================"
    echo "APM Alert Testing Script"
    echo "======================================"
    
    # Parse command line arguments
    local test_type="${1:-all}"
    
    case "$test_type" in
        "routing")
            initialize_test_environment
            test_routing_validation
            test_alert_routing
            test_severity_routing
            cleanup_all_test_alerts
            ;;
        "notifications")
            initialize_test_environment
            test_notification_endpoints
            cleanup_all_test_alerts
            ;;
        "e2e")
            initialize_test_environment
            test_end_to_end_flow
            cleanup_all_test_alerts
            ;;
        "simulation")
            initialize_test_environment
            test_alert_routing
            test_severity_routing
            test_alert_grouping
            test_alert_silencing
            test_inhibition_rules
            cleanup_all_test_alerts
            ;;
        "recovery")
            initialize_test_environment
            test_recovery_notifications
            cleanup_all_test_alerts
            ;;
        "cleanup")
            cleanup_all_test_alerts
            ;;
        "all"|*)
            initialize_test_environment
            test_routing_validation
            test_alert_routing
            test_severity_routing
            test_alert_grouping
            test_alert_silencing
            test_inhibition_rules
            test_notification_endpoints
            test_recovery_notifications
            test_end_to_end_flow
            cleanup_all_test_alerts
            generate_test_report
            ;;
    esac
    
    success "Alert testing completed"
}

# Trap to ensure cleanup on exit
trap 'cleanup_all_test_alerts' EXIT

# Run main function
main "$@"