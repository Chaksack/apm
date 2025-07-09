# Tracing Validation Suite

This directory contains comprehensive end-to-end tracing validation tests for the GoFiber application, focusing on distributed tracing, Jaeger integration, and performance impact assessment.

## Overview

The tracing validation suite consists of four main components:

1. **Python Trace Validation** (`trace-validation.py`) - Validates trace completeness, span relationships, timing, and annotations
2. **Go Distributed Tests** (`distributed-trace-test.go`) - Tests multi-service tracing, context propagation, and baggage
3. **JavaScript Jaeger Integration** (`jaeger-integration-test.js`) - Tests Jaeger API, UI functionality, and export/import
4. **Validation Script** (`../scripts/validate-tracing.sh`) - Automated orchestration and reporting

## Prerequisites

- **Jaeger**: Running on `http://localhost:16686`
- **Python 3.8+**: For trace validation tests
- **Node.js 16+**: For Jaeger integration tests
- **Go 1.21+**: For distributed tracing tests
- **GoFiber Application**: Running and instrumented with tracing

### Quick Start with Jaeger

```bash
# Start Jaeger all-in-one
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 6831:6831/udp \
  jaegertracing/all-in-one:latest
```

## Usage

### Automated Validation

Run the complete validation suite:

```bash
./scripts/validate-tracing.sh
```

### Individual Test Components

```bash
# Python trace validation only
./scripts/validate-tracing.sh python

# Go distributed tests only
./scripts/validate-tracing.sh go

# JavaScript Jaeger integration only
./scripts/validate-tracing.sh js

# Generate test load only
./scripts/validate-tracing.sh load

# Generate report only
./scripts/validate-tracing.sh report
```

### Manual Testing

#### Python Trace Validation

```bash
cd tests/tracing
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt

# Run validation
python3 trace-validation.py

# Run pytest tests
pytest trace-validation.py -v
```

#### Go Distributed Tests

```bash
cd tests/tracing
go mod tidy
go test -v
go test -bench=. -benchmem
```

#### JavaScript Jaeger Integration

```bash
cd tests/tracing
npm install

# Run standalone tests
node jaeger-integration-test.js

# Run with Mocha
npm test
```

## Test Categories

### 1. Trace Completeness Validation

**File**: `trace-validation.py`

- **Root Span Detection**: Ensures traces have root spans
- **Span Hierarchy**: Validates parent-child relationships
- **Required Tags**: Checks for GoFiber-specific tags (http.method, http.url, component)
- **Error Handling**: Validates error trace annotations

**Key Functions**:
- `validate_trace_completeness()`: Checks trace structure
- `validate_span_relationships()`: Validates parent-child timing
- `validate_timing_and_duration()`: Ensures timing consistency
- `validate_tags_and_annotations()`: Checks GoFiber-specific tags

### 2. Distributed Tracing Tests

**File**: `distributed-trace-test.go`

- **Multi-Service Traces**: Tests cross-service trace propagation
- **Context Propagation**: Validates OpenTracing context headers
- **Baggage Verification**: Tests baggage item propagation
- **Error Trace Validation**: Ensures error traces are properly marked

**Key Test Functions**:
- `TestMultiServiceTraceValidation()`: Multi-service trace flow
- `TestContextPropagation()`: Context header validation
- `TestBaggageVerification()`: Baggage propagation testing
- `TestErrorTraceValidation()`: Error trace validation

### 3. Jaeger Integration Tests

**File**: `jaeger-integration-test.js`

- **API Health**: Tests Jaeger API endpoints
- **Trace Queries**: Validates search and filtering
- **UI Functionality**: Tests UI-related API endpoints
- **Export/Import**: Validates trace export/import functionality

**Key Test Methods**:
- `testApiHealth()`: Jaeger API availability
- `testTraceQuery()`: Trace search functionality
- `testTraceSearch()`: Advanced filtering
- `testGoFiberTraces()`: GoFiber-specific validation

### 4. Performance Impact Assessment

**File**: `validate-tracing.sh` (performance functions)

- **Load Generation**: Creates realistic traffic patterns
- **Latency Measurement**: Measures request duration with tracing
- **Overhead Calculation**: Estimates tracing overhead
- **Performance Rating**: Provides performance assessment

## Configuration

### Environment Variables

```bash
# Jaeger configuration
export JAEGER_URL="http://localhost:16686"

# Service configuration
export SERVICE_NAME="apm-service"

# Test configuration
export TIMEOUT="300"
```

### Test Customization

#### Python Configuration

```python
# In trace-validation.py
JAEGER_CONFIG = {
    "jaeger_url": "http://localhost:16686",
    "service_name": "apm-service",
    "lookback": "1h",
    "max_traces": 100
}
```

#### Go Configuration

```go
// In distributed-trace-test.go
const (
    TestServiceA = "service-a"
    TestServiceB = "service-b"
    TestPortA    = 8081
    TestPortB    = 8082
)
```

#### JavaScript Configuration

```javascript
// In jaeger-integration-test.js
const JAEGER_CONFIG = {
    baseUrl: 'http://localhost:16686',
    timeout: 30000,
    maxRetries: 3
};
```

## Expected Outputs

### Validation Report

The validation script generates a comprehensive HTML report:

```
test-results/tracing/validation_report.html
```

### JSON Results

Individual test results are saved as JSON files:

```
test-results/tracing/
├── python_validation.json      # Python validation results
├── js_integration.json         # JavaScript integration results
├── js_tests.json              # Mocha test results
├── performance_analysis.json   # Performance assessment
├── load_test_results.json     # Load test data
└── validation_report.html     # Comprehensive report
```

### Log Files

Test execution logs:

```
test-results/tracing/
├── python_tests.log           # pytest output
├── go_tests.log              # Go test output
├── go_benchmarks.log         # Go benchmark results
└── load_test.log             # Load generation log
```

## Validation Criteria

### Trace Completeness (80% pass rate required)

- ✅ All traces have root spans
- ✅ Parent-child relationships are valid
- ✅ Timing is consistent across spans
- ✅ Required tags are present

### Performance Impact (acceptable thresholds)

- ✅ Average request duration < 500ms
- ✅ Tracing overhead < 10%
- ✅ Success rate > 95%
- ✅ Memory usage increase < 20%

### Jaeger Integration (all tests must pass)

- ✅ API endpoints respond correctly
- ✅ Trace queries return expected results
- ✅ Export/import functionality works
- ✅ UI components are functional

## Troubleshooting

### Common Issues

1. **Jaeger Not Running**
   ```bash
   curl http://localhost:16686/api/services
   # Should return JSON with services list
   ```

2. **No Traces Found**
   - Ensure application is generating traffic
   - Check Jaeger agent connectivity
   - Verify service name configuration

3. **Test Failures**
   - Check log files in `test-results/tracing/`
   - Verify dependencies are installed
   - Ensure sufficient test data exists

### Debug Mode

Enable verbose logging:

```bash
export DEBUG=1
./scripts/validate-tracing.sh
```

## Contributing

When adding new validation tests:

1. Follow the existing test structure
2. Update this README with new test descriptions
3. Ensure proper error handling and logging
4. Add performance impact assessment for new features

## Integration with CI/CD

The validation script is designed for CI/CD integration:

```yaml
# Example GitHub Actions step
- name: Validate Tracing
  run: |
    ./scripts/validate-tracing.sh
    # Upload results as artifacts
    tar -czf tracing-validation-results.tar.gz test-results/tracing/
```

## License

This testing suite is part of the APM project and follows the same licensing terms.