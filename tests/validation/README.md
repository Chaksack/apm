# APM Metrics Validation Suite

This directory contains comprehensive validation tests for the APM (Application Performance Monitoring) system's metric collection components.

## Overview

The validation suite provides automated testing for:
- **Prometheus metrics collection** - Validates metrics presence, labels, and data integrity
- **Grafana dashboard functionality** - Tests dashboards, queries, and data sources
- **Istio service mesh metrics** - Validates service mesh, security, and mTLS metrics
- **End-to-end validation** - Comprehensive system health checks

## Components

### 1. Prometheus Metrics Validation (`metrics-validation.py`)
Validates Prometheus metrics collection and data integrity.

**Features:**
- Connectivity checks to Prometheus
- Core metric presence validation
- Application and infrastructure metric validation
- Label validation for required fields
- Value range validation
- Time series data availability checks

**Usage:**
```bash
python3 metrics-validation.py --prometheus-url http://localhost:9090
```

### 2. Grafana Validation (`grafana-validation.js`)
Validates Grafana dashboards, data sources, and alert configurations.

**Features:**
- Grafana connectivity and health checks
- Data source validation and testing
- Dashboard query validation
- Panel data verification
- Alert rule and instance validation

**Usage:**
```bash
node grafana-validation.js --grafana-url http://localhost:3000
```

### 3. Istio Metrics Validation (`istio-metrics-test.go`)
Go-based tests for Istio service mesh metrics validation.

**Features:**
- Control plane metrics validation
- Data plane (Envoy) metrics validation
- Security and mTLS metrics verification
- Service mesh connectivity validation
- Component health checks

**Usage:**
```bash
go test -v ./...
```

### 4. Validation Script (`validate-metrics.sh`)
Comprehensive shell script that orchestrates all validation tests.

**Features:**
- Dependency checking
- Service connectivity validation
- Automated test execution
- Result aggregation and reporting
- Configurable test selection

**Usage:**
```bash
./scripts/validate-metrics.sh [OPTIONS] [TESTS]
```

## Installation

### Prerequisites
- Python 3.7+
- Node.js 14+
- Go 1.19+
- curl, jq, bc (for shell script)

### Python Dependencies
```bash
pip3 install requests
```

### Node.js Dependencies
```bash
cd tests/validation
npm install
```

### Go Dependencies
```bash
cd tests/validation
go mod tidy
```

## Configuration

### Environment Variables
```bash
export PROMETHEUS_URL="http://localhost:9090"
export GRAFANA_URL="http://localhost:3000"
export GRAFANA_USERNAME="admin"
export GRAFANA_PASSWORD="admin"
export GRAFANA_API_KEY="your-api-key"
export KUBECONFIG="$HOME/.kube/config"
```

### Service URLs
- **Prometheus**: Default `http://localhost:9090`
- **Grafana**: Default `http://localhost:3000`
- **Kubernetes**: Uses default kubeconfig

## Usage Examples

### Run All Validations
```bash
./scripts/validate-metrics.sh
```

### Run Specific Tests
```bash
./scripts/validate-metrics.sh prometheus grafana
```

### Custom Configuration
```bash
./scripts/validate-metrics.sh \
  --prometheus-url http://prometheus:9090 \
  --grafana-url http://grafana:3000 \
  --grafana-api-key abc123 \
  --verbose
```

### Individual Test Execution

#### Prometheus Validation
```bash
cd tests/validation
python3 metrics-validation.py \
  --prometheus-url http://localhost:9090 \
  --output prometheus_results.json
```

#### Grafana Validation
```bash
cd tests/validation
node grafana-validation.js \
  --grafana-url http://localhost:3000 \
  --username admin \
  --password admin \
  --output grafana_results.json
```

#### Istio Validation
```bash
cd tests/validation
go test -v -timeout=300s ./...
```

## Test Categories

### Prometheus Tests
- **Core Metrics**: Basic Prometheus operational metrics
- **Application Metrics**: HTTP requests, response times, process metrics
- **Infrastructure Metrics**: CPU, memory, disk, network metrics
- **Value Validation**: Metric values within expected ranges
- **Time Series**: Historical data availability

### Grafana Tests
- **Connectivity**: Grafana API accessibility
- **Data Sources**: Prometheus data source health
- **Dashboards**: Dashboard loading and structure
- **Queries**: Query execution and results
- **Panels**: Panel configuration and data
- **Alerts**: Alert rules and instances

### Istio Tests
- **Control Plane**: Pilot, Citadel, Galley metrics
- **Data Plane**: Envoy proxy metrics
- **Security**: mTLS and security policy metrics
- **Traffic**: HTTP and TCP traffic metrics
- **Components**: Service mesh component health
- **Configuration**: Istio configuration metrics

## Output and Results

### Result Files
All validation results are saved to the `validation-results/` directory:
- `prometheus_validation.json` - Prometheus test results
- `grafana_validation.json` - Grafana test results
- `health_check.json` - Component health status
- `validation_summary.json` - Overall summary
- `validation.log` - Detailed execution log

### Result Format
```json
{
  "validation_run": {
    "timestamp": "2024-01-01T12:00:00Z",
    "total_tests": 25,
    "passed": 23,
    "failed": 2,
    "success_rate": 92.0
  },
  "test_results": [
    {
      "test_name": "Prometheus connectivity",
      "result": true,
      "duration": 0.15,
      "timestamp": "2024-01-01T12:00:00Z"
    }
  ]
}
```

## Troubleshooting

### Common Issues

#### Prometheus Connection Failed
```bash
# Check Prometheus status
curl http://localhost:9090/api/v1/status/config

# Verify port forwarding (if using Kubernetes)
kubectl port-forward svc/prometheus 9090:9090 -n monitoring
```

#### Grafana Authentication Failed
```bash
# Test basic auth
curl -u admin:admin http://localhost:3000/api/health

# Test with API key
curl -H "Authorization: Bearer YOUR_API_KEY" http://localhost:3000/api/health
```

#### Istio Metrics Missing
```bash
# Check Istio installation
kubectl get pods -n istio-system

# Verify service mesh injection
kubectl get pods -o wide --show-labels
```

### Debugging

Enable verbose output:
```bash
./scripts/validate-metrics.sh --verbose
```

Check individual components:
```bash
./scripts/validate-metrics.sh connectivity health
```

## Integration

### CI/CD Pipeline
```yaml
# Example GitHub Actions workflow
- name: Validate Metrics
  run: |
    ./scripts/validate-metrics.sh --timeout 600
    if [ $? -ne 0 ]; then
      echo "Metric validation failed"
      exit 1
    fi
```

### Monitoring Integration
The validation results can be integrated with monitoring systems to track validation success rates over time.

## Best Practices

1. **Regular Validation**: Run validations after deployments
2. **Environment-Specific**: Use different configurations for dev/staging/prod
3. **Baseline Establishment**: Establish baseline metrics for comparison
4. **Alert Integration**: Set up alerts for validation failures
5. **Documentation**: Keep validation criteria updated with system changes

## Contributing

When adding new validation tests:
1. Follow the existing test structure
2. Include error handling and timeouts
3. Add appropriate logging
4. Update documentation
5. Test in multiple environments