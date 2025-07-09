# Load Testing Suite for GoFiber Application

This directory contains comprehensive load testing scenarios designed specifically for GoFiber applications using k6.

## Test Files

### 1. `k6-basic-load.js`
- **Purpose**: Basic load testing with realistic user scenarios
- **Features**:
  - Gradual ramp-up and sustained load patterns
  - Health checks, user management, product catalog, and authentication scenarios
  - Performance thresholds (95% < 500ms, error rate < 10%)
  - Realistic test data with different payload sizes
  - Custom metrics for error tracking

### 2. `k6-stress-test.js`
- **Purpose**: Stress testing beyond normal capacity
- **Features**:
  - Progressive load increase up to 800 concurrent users
  - Memory stress testing with large payloads
  - CPU stress testing with complex queries
  - Database stress testing with concurrent operations
  - Resource exhaustion scenarios
  - Breaking point identification

### 3. `k6-spike-test.js`
- **Purpose**: Sudden traffic spike simulation
- **Features**:
  - Sudden load spikes (5 → 100 → 200 → 500 users)
  - Circuit breaker testing
  - Auto-scaling validation
  - Recovery time measurement
  - Rate limiting validation
  - Scenario-specific testing (registration, search, orders, auth)

### 4. `api-endpoints-test.js`
- **Purpose**: Comprehensive API endpoint testing
- **Features**:
  - All endpoint coverage (auth, users, products, orders, admin, system)
  - Variable payload sizes (small, medium, large)
  - Error scenario testing (404, 400, 500)
  - Authentication flow testing
  - Custom metrics for API calls and payload sizes

## Running the Tests

### Prerequisites
1. Install k6: https://k6.io/docs/getting-started/installation/
2. Ensure your GoFiber application is running
3. Optional: Install jq for enhanced reporting

### Individual Test Execution
```bash
# Basic load test
k6 run --env BASE_URL=http://localhost:3000 tests/load/k6-basic-load.js

# Stress test
k6 run --env BASE_URL=http://localhost:3000 tests/load/k6-stress-test.js

# Spike test
k6 run --env BASE_URL=http://localhost:3000 tests/load/k6-spike-test.js

# API endpoints test
k6 run --env BASE_URL=http://localhost:3000 tests/load/api-endpoints-test.js
```

### Automated Test Suite
Use the provided automation script:
```bash
# Run all tests
./scripts/run-load-tests.sh

# Run specific test
./scripts/run-load-tests.sh --basic-only

# Run with custom URL
./scripts/run-load-tests.sh -u http://localhost:8080

# Run tests concurrently
./scripts/run-load-tests.sh --concurrent

# Get help
./scripts/run-load-tests.sh --help
```

## Test Configuration

### Environment Variables
- `BASE_URL`: Target application URL (default: http://localhost:3000)
- `CONCURRENT_TESTS`: Run tests concurrently (true/false)
- `GENERATE_HTML_REPORT`: Generate HTML report (true/false)
- `SEND_SLACK_NOTIFICATION`: Send notifications (true/false)
- `SLACK_WEBHOOK_URL`: Slack webhook for notifications

### Performance Thresholds
- **Basic Load Test**: 95% < 500ms, error rate < 10%
- **Stress Test**: 95% < 2000ms, error rate < 50%
- **Spike Test**: 95% < 3000ms, error rate < 30%
- **API Endpoints Test**: 95% < 1000ms, error rate < 5%

## Expected Endpoints

The tests assume the following GoFiber endpoints exist:

### Authentication
- `POST /api/auth/login`
- `POST /api/auth/register`
- `POST /api/auth/logout`
- `POST /api/auth/refresh`

### Users
- `GET /api/users`
- `POST /api/users`
- `GET /api/users/:id`
- `PUT /api/users/:id`
- `DELETE /api/users/:id`
- `GET /api/users/search`

### Products
- `GET /api/products`
- `POST /api/products`
- `GET /api/products/:id`
- `GET /api/products/search`
- `GET /api/categories`

### Orders
- `GET /api/orders`
- `POST /api/orders`
- `GET /api/orders/:id`
- `GET /api/orders/history`

### System
- `GET /health`
- `GET /metrics`
- `GET /api/version`

## Customization

### Adding New Scenarios
1. Create test functions in the appropriate test file
2. Add them to the `group()` calls in the main function
3. Update thresholds if needed

### Modifying Test Data
Update the `testData` objects in each test file to match your application's data structure.

### Adjusting Load Patterns
Modify the `stages` array in the `options` object to change load patterns.

## Results and Reporting

### Output Files
- JSON results: `results/load-tests/[timestamp]/`
- HTML reports: `reports/load-tests/[timestamp]/`
- Summary metrics: `*_summary.json` files

### Metrics Collected
- HTTP request duration (avg, p95, p99)
- Error rates (HTTP failures, custom errors)
- Request rate (requests per second)
- Custom metrics (payload sizes, API calls, circuit breakers)

### Interpreting Results
1. **Response Times**: Check p95 and p99 percentiles
2. **Error Rates**: Monitor HTTP failures and custom errors
3. **Throughput**: Analyze requests per second
4. **Breaking Points**: Identify when error rates spike
5. **Recovery Time**: Monitor system recovery after spikes

## Troubleshooting

### Common Issues
1. **Service not responding**: Ensure your GoFiber app is running
2. **High error rates**: Check endpoint implementations
3. **Slow response times**: Review database queries and resource usage
4. **Memory issues**: Monitor application memory usage during tests

### Debugging Tips
1. Run tests with `--verbose` flag for detailed output
2. Use smaller user counts for initial testing
3. Check application logs during test execution
4. Monitor system resources (CPU, memory, disk)

## Best Practices

1. **Gradual Load Increase**: Start with basic load before stress testing
2. **Baseline Measurements**: Run tests before and after changes
3. **Regular Testing**: Include load tests in CI/CD pipeline
4. **Resource Monitoring**: Monitor system resources during tests
5. **Test Environment**: Use production-like environment for accurate results

## Integration with CI/CD

Add to your CI/CD pipeline:
```yaml
load-tests:
  stage: test
  script:
    - ./scripts/run-load-tests.sh --basic-only
  artifacts:
    reports:
      paths:
        - reports/load-tests/
```

This load testing suite provides comprehensive coverage for GoFiber applications, helping you identify performance bottlenecks, validate scalability, and ensure your application can handle production traffic patterns.