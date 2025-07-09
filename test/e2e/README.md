# APM E2E Test Suite

This directory contains comprehensive end-to-end tests for the APM (Application Performance Monitoring) stack. The tests verify that all components work correctly both individually and when integrated together.

## Overview

The E2E test suite validates the following APM components:

- **Prometheus**: Metrics collection and alerting
- **Grafana**: Visualization and dashboards
- **Loki**: Log aggregation and querying
- **Jaeger**: Distributed tracing
- **AlertManager**: Alert routing and notifications
- **Semgrep**: Security analysis
- **Application**: Health checks and instrumentation

## Prerequisites

Before running the tests, ensure you have the following installed:

- Docker and Docker Compose
- Go 1.19 or later
- Make
- Semgrep (for security tests)
- curl (for health checks)

## Quick Start

### Run All Tests

```bash
make test
```

This command will:
1. Start all APM services using Docker Compose
2. Wait for services to be ready
3. Run all E2E tests
4. Display test results

### Run Specific Test Suites

```bash
# Test individual components
make test-prometheus      # Test Prometheus metrics collection
make test-grafana        # Test Grafana connectivity and dashboards
make test-loki           # Test Loki log aggregation
make test-jaeger         # Test Jaeger trace collection
make test-alertmanager   # Test AlertManager notifications
make test-semgrep        # Test Semgrep security analysis
make test-health         # Test health check endpoints
make test-integration    # Test integration between services
```

## Test Structure

### Test Files

- **apm_tools_test.go**: Main test file containing all test cases
- **test_helpers.go**: Helper functions for test setup, data generation, and verification
- **docker-compose.test.yml**: Docker Compose configuration for test environment
- **test-configs/**: Configuration files for each service

### Test Categories

1. **Component Tests**: Verify individual service functionality
   - Prometheus metrics collection and querying
   - Grafana dashboard provisioning and datasource connectivity
   - Loki log ingestion and querying
   - Jaeger trace collection and search
   - AlertManager alert routing and silencing

2. **Integration Tests**: Verify service interactions
   - Metrics flow from application → Prometheus → Grafana
   - Logs flow from application → Loki → Grafana
   - Traces flow from application → Jaeger → Grafana
   - Alerts flow from Prometheus → AlertManager

3. **Security Tests**: Verify security scanning
   - Semgrep analysis for code vulnerabilities
   - Detection of SQL injection
   - Detection of hardcoded credentials

## Makefile Targets

### Setup and Teardown

```bash
make setup      # Start all services
make teardown   # Stop and remove all services
make clean      # Clean up test artifacts and volumes
```

### Development Commands

```bash
make dev-setup     # Quick setup for development
make dev-test      # Run tests without setup/teardown
make dev-teardown  # Stop services without removing volumes
```

### Monitoring Commands

```bash
make logs              # Show logs from all services
make logs-prometheus   # Show logs from Prometheus
make logs-grafana      # Show logs from Grafana
make logs-loki         # Show logs from Loki
make logs-jaeger       # Show logs from Jaeger
make logs-alertmanager # Show logs from AlertManager
make logs-app          # Show logs from test application
```

### Health Checks

```bash
make check-all         # Check all services
make check-prometheus  # Check Prometheus health
make check-grafana     # Check Grafana health
make check-loki        # Check Loki health
make check-jaeger      # Check Jaeger health
make check-alertmanager # Check AlertManager health
```

### Advanced Commands

```bash
make test-coverage  # Run tests with coverage report
make benchmark      # Run benchmark tests
make watch          # Run tests continuously (watches for changes)
```

## Configuration

### Environment Variables

The test suite uses the following default endpoints:

- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)
- Loki: http://localhost:3100
- Jaeger: http://localhost:16686
- AlertManager: http://localhost:9093
- Application: http://localhost:8080

### Custom Configuration

To use custom configurations, modify the files in `test-configs/`:

- `prometheus.yml`: Prometheus scrape configs and rules
- `grafana/provisioning/`: Grafana datasources and dashboards
- `loki/loki-config.yaml`: Loki storage and schema configuration
- `promtail/promtail-config.yaml`: Log collection configuration
- `alertmanager/alertmanager.yml`: Alert routing configuration

## Writing New Tests

### Test Template

```go
func TestNewFeature(t *testing.T) {
    ctx := context.Background()
    
    // Wait for service to be ready
    err := WaitForService(ctx, "http://service:port/health", 30*time.Second)
    require.NoError(t, err, "Service should be ready")
    
    // Test implementation
    t.Run("SubTest", func(t *testing.T) {
        // Test specific functionality
        result, err := YourTestFunction()
        require.NoError(t, err)
        assert.Equal(t, expected, result)
    })
}
```

### Helper Functions

Use the provided helper functions in `test_helpers.go`:

- `WaitForService()`: Wait for a service to be ready
- `SendTestMetrics()`: Send test metrics to Prometheus
- `SendLogToLoki()`: Send logs to Loki
- `SendTestTrace()`: Send traces to Jaeger
- `SendTestAlert()`: Send alerts to AlertManager
- `GenerateApplicationLoad()`: Generate realistic application load

## Troubleshooting

### Common Issues

1. **Services not starting**: Check Docker logs
   ```bash
   make logs
   docker-compose -f docker-compose.test.yml ps
   ```

2. **Tests timing out**: Increase timeout in test files or wait longer for services
   ```go
   err := WaitForService(ctx, endpoint, 60*time.Second) // Increase timeout
   ```

3. **Port conflicts**: Ensure ports are not in use
   ```bash
   lsof -i :9090  # Check if Prometheus port is in use
   ```

4. **Semgrep not found**: Install Semgrep
   ```bash
   pip install semgrep
   # or
   brew install semgrep
   ```

### Debug Mode

Run tests with verbose output:
```bash
go test -v -run TestName ./...
```

Enable debug logging in services by modifying docker-compose.test.yml:
```yaml
environment:
  - LOG_LEVEL=debug
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: E2E Tests
on: [push, pull_request]

jobs:
  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.20'
      
      - name: Install Semgrep
        run: pip install semgrep
      
      - name: Run E2E Tests
        run: |
          cd test/e2e
          make test
      
      - name: Upload Coverage
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: coverage-report
          path: test/e2e/coverage/
```

### Jenkins Pipeline Example

```groovy
pipeline {
    agent any
    
    stages {
        stage('Setup') {
            steps {
                sh 'cd test/e2e && make setup'
            }
        }
        
        stage('Test') {
            steps {
                sh 'cd test/e2e && make test-coverage'
            }
        }
        
        stage('Cleanup') {
            always {
                sh 'cd test/e2e && make teardown'
            }
        }
    }
    
    post {
        always {
            publishHTML([
                reportDir: 'test/e2e/coverage',
                reportFiles: 'coverage.html',
                reportName: 'Coverage Report'
            ])
        }
    }
}
```

## Best Practices

1. **Isolation**: Each test should be independent and not rely on state from other tests
2. **Cleanup**: Always clean up test data after tests complete
3. **Timeouts**: Use appropriate timeouts for service readiness checks
4. **Parallel Testing**: Tests can be run in parallel using `go test -parallel`
5. **Resource Management**: Monitor resource usage during tests

## Contributing

When adding new tests:

1. Follow the existing test structure
2. Add appropriate helper functions to `test_helpers.go`
3. Update this README with new test documentation
4. Ensure tests are idempotent and can be run multiple times
5. Add cleanup logic for any test data created

## Performance Considerations

The full test suite may take 10-15 minutes to run. For faster feedback:

1. Run specific test suites during development
2. Use `dev-setup` to keep services running between test runs
3. Consider running heavy tests (like load tests) separately
4. Use test caching with `go test -count=1`

## Metrics and Monitoring

During test execution, you can monitor:

- Test execution time: Check test output
- Service health: Use `make check-all`
- Resource usage: `docker stats`
- Test coverage: `make test-coverage`

## License

This test suite is part of the APM project and follows the same license.