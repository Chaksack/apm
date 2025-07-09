# Monitoring Stack Performance Tests

This directory contains comprehensive performance tests for the monitoring stack components including Prometheus, Grafana, and Loki.

## Test Components

### 1. Prometheus Performance Tests (`prometheus-performance.js`)
Tests Prometheus query performance and resource utilization:
- **Query Performance**: Tests various query types and complexity levels
- **High Cardinality Metrics**: Validates handling of metrics with many labels
- **Storage Performance**: Tests range queries across different time periods
- **Memory Usage**: Monitors memory consumption during heavy query loads

### 2. Grafana Performance Tests (`grafana-performance.py`)
Tests Grafana dashboard and API performance:
- **Dashboard Rendering**: Measures dashboard load times and panel rendering
- **Query Responsiveness**: Tests data source query performance
- **Concurrent Users**: Simulates multiple users accessing dashboards
- **Data Source Performance**: Tests connection and query performance

### 3. Loki Performance Tests (`loki-performance.go`)
Tests Loki log ingestion and query performance:
- **Log Ingestion Rate**: Tests batch ingestion performance
- **Query Performance**: Validates LogQL query response times
- **Storage Efficiency**: Tests log compression and storage optimization
- **Compression Testing**: Compares different compression strategies

### 4. Full Stack Load Tests (`monitoring-stack-load.yaml`)
Kubernetes-based load testing for the entire monitoring stack:
- **Full Stack Load Testing**: Tests all components together
- **Resource Utilization**: Monitors CPU, memory, and storage usage
- **Scaling Behavior**: Validates auto-scaling and performance under load

### 5. Performance Test Suite (`../scripts/performance-test-suite.sh`)
Comprehensive test orchestration script:
- **Automated Test Execution**: Runs all performance tests
- **Resource Monitoring**: Collects system metrics during tests
- **Report Generation**: Creates detailed HTML performance reports

## Prerequisites

### System Requirements
- Node.js 16+ (for Prometheus tests)
- Python 3.8+ (for Grafana tests)
- Go 1.19+ (for Loki tests)
- kubectl (for Kubernetes load tests)
- curl, jq (for API testing)

### Service Requirements
- Prometheus running on http://localhost:9090
- Grafana running on http://localhost:3000
- Loki running on http://localhost:3100
- Kubernetes cluster (for load tests)

## Running Tests

### Individual Tests

#### Prometheus Tests
```bash
cd tests/performance
npm install
node prometheus-performance.js
```

#### Grafana Tests
```bash
cd tests/performance
pip3 install aiohttp psutil
python3 grafana-performance.py
```

#### Loki Tests
```bash
cd tests/performance
go run loki-performance.go
```

#### Kubernetes Load Tests
```bash
cd tests/performance
kubectl apply -f monitoring-stack-load.yaml
```

### Comprehensive Test Suite
```bash
# Run all tests with default settings
./scripts/performance-test-suite.sh

# Run with custom URLs
./scripts/performance-test-suite.sh \
  --prometheus-url http://prometheus.example.com:9090 \
  --grafana-url http://grafana.example.com:3000 \
  --loki-url http://loki.example.com:3100

# Skip specific tests
./scripts/performance-test-suite.sh \
  --skip-prometheus \
  --skip-load

# See all options
./scripts/performance-test-suite.sh --help
```

## Test Configurations

### Prometheus Tests
- Query types: Basic, aggregations, range queries, mathematical operations
- Batch sizes: 100, 500, 1000, 5000, 10000 metrics
- Time ranges: 1h, 6h, 1d, 7d, 30d
- Concurrent queries: Up to 50 simultaneous queries

### Grafana Tests
- Dashboard load testing: Top 5 dashboards
- User simulation: 5, 10, 20, 50 concurrent users
- API endpoints: Health, search, dashboard, query APIs
- Data source types: Prometheus, Loki

### Loki Tests
- Log ingestion rates: 100-10000 logs per batch
- Query types: Label queries, regex filters, aggregations
- Compression: gzip vs uncompressed
- Storage efficiency: Logs per byte metrics

### Load Tests
- Duration: 16 minutes with varying load
- Stages: Ramp up to 100 concurrent users
- Metrics: Response times, error rates, throughput
- Resource monitoring: CPU, memory, disk usage

## Performance Metrics

### Key Performance Indicators (KPIs)
- **Query Response Time**: Average, p95, p99 percentiles
- **Throughput**: Queries per second, logs per second
- **Error Rate**: Percentage of failed requests
- **Resource Utilization**: CPU, memory, disk usage
- **Concurrency**: Maximum concurrent users supported

### Benchmark Targets
- Prometheus query response: < 1s for simple queries
- Grafana dashboard load: < 5s for typical dashboards
- Loki log ingestion: > 1000 logs/second
- Error rate: < 1% under normal load
- Memory usage: Stable without leaks

## Output and Reports

### Test Results
Each test generates detailed JSON results:
- `prometheus-performance-results.json`
- `grafana-performance-results.json`
- `loki-performance-results.json`
- `load-test-results.log`
- `system-metrics.json`

### Performance Report
The test suite generates a comprehensive HTML report:
- Executive summary
- Test execution status
- Detailed metrics and charts
- Performance recommendations
- System information

### Log Files
Detailed execution logs are saved with timestamps:
- `performance-test-YYYYMMDD_HHMMSS.log`

## Troubleshooting

### Common Issues

#### Connection Errors
```bash
# Check service health
curl -s http://localhost:9090/api/v1/status/config
curl -s http://localhost:3000/api/health
curl -s http://localhost:3100/ready
```

#### Dependency Issues
```bash
# Install Node.js dependencies
npm install

# Install Python dependencies
pip3 install aiohttp psutil

# Check Go version
go version
```

#### Kubernetes Issues
```bash
# Check cluster connection
kubectl cluster-info

# Check namespace
kubectl get namespace performance-testing

# Check pod status
kubectl get pods -n performance-testing
```

### Performance Optimization

#### Prometheus Optimization
- Increase `--query.timeout` for complex queries
- Optimize recording rules for frequently used queries
- Monitor `prometheus_tsdb_*` metrics for storage health

#### Grafana Optimization
- Enable query result caching
- Optimize dashboard queries
- Use dashboard folders for organization

#### Loki Optimization
- Tune `ingester` configuration for throughput
- Optimize log parsing and labeling
- Monitor `loki_ingester_*` metrics

## Monitoring and Alerting

### During Tests
- Monitor system resources (CPU, memory, disk)
- Watch for error logs in service logs
- Track query performance metrics
- Monitor network I/O and latency

### After Tests
- Review performance trends over time
- Compare results with previous runs
- Identify performance regressions
- Plan capacity based on results

## Contributing

### Adding New Tests
1. Create test file in appropriate language
2. Follow existing naming conventions
3. Add configuration to test suite script
4. Update documentation

### Test Guidelines
- Use realistic data volumes
- Include error handling
- Provide detailed logging
- Generate structured output
- Include resource monitoring

## Security Considerations

### Test Data
- Use synthetic data only
- Avoid real production data
- Clean up test data after runs
- Secure API credentials

### Access Control
- Use minimal required permissions
- Rotate test credentials regularly
- Limit network access during tests
- Monitor for unauthorized access

## License

MIT License - see LICENSE file for details.