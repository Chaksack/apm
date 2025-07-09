# Quality Gates Documentation

## Overview

Quality gates are automated checkpoints in the CI/CD pipeline that ensure code meets predefined quality standards before proceeding to the next stage. This document defines the quality gates, their thresholds, override procedures, and tracked metrics for the APM stack.

## Quality Gate Definitions

### 1. Code Coverage Gate
**Purpose**: Ensure adequate test coverage for code changes

**Criteria**:
- **Minimum Coverage**: 80% overall
- **New Code Coverage**: 85% minimum
- **Critical Files Coverage**: 90% minimum (main.go, handlers, middleware)
- **Uncovered Lines**: Maximum 50 new uncovered lines

**Measurement**:
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Check coverage percentage
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
if [ ${COVERAGE%.*} -lt 80 ]; then
    echo "Coverage $COVERAGE% is below threshold"
    exit 1
fi
```

**Failure Actions**:
- Block deployment
- Notify development team
- Generate detailed coverage report
- Suggest specific files to test

### 2. Unit Test Gate
**Purpose**: Validate code functionality through automated tests

**Criteria**:
- **Test Pass Rate**: 100% (all tests must pass)
- **Test Execution Time**: < 5 minutes
- **Flaky Test Tolerance**: 0% (no flaky tests allowed)
- **Minimum Test Count**: 1 test per public function

**Measurement**:
```bash
# Run tests with verbose output
go test -v -timeout=5m ./...

# Run tests with race detection
go test -race ./...

# Check test count
TEST_COUNT=$(go test -v ./... | grep -c "^=== RUN")
FUNC_COUNT=$(grep -r "^func [A-Z]" --include="*.go" . | wc -l)
```

**Failure Actions**:
- Immediate pipeline failure
- Detailed test failure report
- Notify responsible developer
- Block all deployments

### 3. Static Code Analysis Gate
**Purpose**: Identify code quality issues, bugs, and security vulnerabilities

**Criteria**:
- **Complexity**: Maximum cyclomatic complexity of 15
- **Maintainability**: Minimum maintainability index of 70
- **Duplication**: Maximum 3% code duplication
- **Security**: No high/critical security issues

**Tools and Configuration**:
```yaml
# SonarQube configuration
sonar.projectKey=apm-stack
sonar.projectName=APM Stack
sonar.sources=.
sonar.exclusions=vendor/**,**/*_test.go,**/testdata/**
sonar.tests=.
sonar.test.inclusions=**/*_test.go
sonar.go.coverage.reportPaths=coverage.out

# Quality gate conditions
sonar.qualitygate.wait=true
sonar.qualitygate.timeout=300
```

**Measurement**:
```bash
# Run SonarQube analysis
sonar-scanner -Dsonar.projectKey=apm-stack

# Check quality gate status
curl -u admin:admin "http://sonarqube:9000/api/qualitygates/project_status?projectKey=apm-stack"
```

**Failure Actions**:
- Generate detailed code quality report
- Highlight specific issues
- Suggest fixes
- Block deployment until resolved

### 4. Security Scan Gate
**Purpose**: Identify security vulnerabilities in code and dependencies

**Criteria**:
- **Critical Vulnerabilities**: 0 allowed
- **High Vulnerabilities**: 0 allowed
- **Medium Vulnerabilities**: Maximum 5
- **Secret Detection**: No secrets in code

**Tools**:
- **Trivy**: Container and filesystem scanning
- **Gosec**: Go security analysis
- **git-secrets**: Secret detection
- **Dependency-Check**: Vulnerability scanning

**Measurement**:
```bash
# Run Gosec security scan
gosec -fmt json -out gosec-report.json ./...

# Run Trivy filesystem scan
trivy fs --format json --output trivy-report.json .

# Check for secrets
git-secrets --scan

# Dependency vulnerability check
go list -json -m all | nancy sleuth
```

**Failure Actions**:
- Immediate pipeline failure for critical/high vulnerabilities
- Detailed security report
- Notify security team
- Block all deployments

### 5. Performance Gate
**Purpose**: Ensure application performance meets requirements

**Criteria**:
- **Memory Usage**: < 100MB for basic operations
- **CPU Usage**: < 50% under normal load
- **Response Time**: < 100ms for health checks
- **Throughput**: > 1000 requests/second

**Measurement**:
```bash
# Performance benchmark
go test -bench=. -benchmem ./...

# Memory profiling
go test -memprofile=mem.prof ./...
go tool pprof mem.prof

# Load testing
curl -X POST http://localhost:8080/load-test \
  -H "Content-Type: application/json" \
  -d '{"concurrent_users": 100, "duration": "60s"}'
```

**Failure Actions**:
- Generate performance report
- Highlight performance regressions
- Suggest optimizations
- Allow deployment with monitoring

### 6. Linting Gate
**Purpose**: Enforce code style and consistency

**Criteria**:
- **Go Linting**: golangci-lint with strict configuration
- **YAML Linting**: yamllint for configuration files
- **Markdown Linting**: markdownlint for documentation
- **Dockerfile Linting**: hadolint for Docker files

**Configuration**:
```yaml
# .golangci.yml
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - typecheck
    - unused
    - gocyclo
    - gofmt
    - gosec

linters-settings:
  gocyclo:
    min-complexity: 15
  gosec:
    severity: medium
```

**Measurement**:
```bash
# Run golangci-lint
golangci-lint run --config .golangci.yml

# YAML linting
yamllint configs/

# Markdown linting
markdownlint docs/

# Dockerfile linting
hadolint Dockerfile
```

**Failure Actions**:
- Detailed linting report
- Suggest auto-fixes where possible
- Block deployment until resolved

## Threshold Explanations

### Coverage Thresholds
- **80% Overall**: Balances development velocity with quality
- **85% New Code**: Ensures improvements don't decrease coverage
- **90% Critical Files**: Core functionality needs higher coverage

### Performance Thresholds
- **100ms Response Time**: Industry standard for health checks
- **100MB Memory**: Reasonable for microservice architecture
- **1000 RPS**: Baseline performance requirement

### Security Thresholds
- **Zero Critical/High**: No compromise on security
- **5 Medium**: Allows for reasonable technical debt
- **Secret Detection**: Prevents credential leaks

### Complexity Thresholds
- **Cyclomatic Complexity 15**: Maintains code readability
- **Maintainability Index 70**: Ensures long-term maintainability
- **3% Duplication**: Minimal code duplication

## Override Procedures

### Emergency Override
**When to Use**: Critical production issues requiring immediate deployment

**Process**:
1. **Approval Required**: Engineering manager + security team
2. **Documentation**: Detailed justification and risk assessment
3. **Time Limit**: Maximum 24 hours
4. **Follow-up**: Issues must be resolved within 48 hours

```bash
# Emergency override command
export EMERGENCY_OVERRIDE=true
export OVERRIDE_REASON="Critical production fix for issue #1234"
export OVERRIDE_APPROVER="engineering-manager@company.com"
export OVERRIDE_EXPIRY="2024-01-15T10:00:00Z"

# Deploy with override
./scripts/deploy.sh --emergency-override
```

### Temporary Exception
**When to Use**: Short-term quality gate failures with planned resolution

**Process**:
1. **Approval Required**: Tech lead
2. **Documentation**: Tracking issue with timeline
3. **Time Limit**: Maximum 7 days
4. **Monitoring**: Daily progress check

```bash
# Temporary exception
export TEMP_EXCEPTION=true
export EXCEPTION_GATE="coverage"
export EXCEPTION_REASON="Refactoring in progress - Issue #5678"
export EXCEPTION_EXPIRY="2024-01-22T00:00:00Z"

# Deploy with exception
./scripts/deploy.sh --temp-exception
```

### Staged Rollout Override
**When to Use**: Gradual deployment with monitoring

**Process**:
1. **Approval Required**: Product owner
2. **Monitoring**: Enhanced observability
3. **Rollback Plan**: Automatic triggers defined
4. **Validation**: Incremental traffic increase

```bash
# Staged rollout
export STAGED_ROLLOUT=true
export ROLLOUT_PERCENTAGE=10
export ROLLOUT_MONITOR_DURATION=60m

# Deploy with staged rollout
./scripts/deploy.sh --staged-rollout
```

## Metrics Tracked

### Quality Metrics
```yaml
# Code quality metrics
code_coverage_percentage: 85.2
test_pass_rate: 100.0
complexity_average: 8.5
duplication_percentage: 1.2
security_vulnerabilities: 0
linting_issues: 0

# Performance metrics  
response_time_p95: 45ms
memory_usage_average: 78MB
cpu_usage_average: 35%
throughput_rps: 1250

# Process metrics
pipeline_success_rate: 98.5
average_pipeline_duration: 8.5m
quality_gate_failure_rate: 2.1
override_usage_rate: 0.5
```

### Alerting Configuration
```yaml
# Quality gate alerts
alerts:
  - alert: QualityGateFailure
    expr: quality_gate_status != 0
    for: 0m
    labels:
      severity: warning
    annotations:
      summary: "Quality gate failed"
      description: "Quality gate {{ $labels.gate }} failed with threshold {{ $labels.threshold }}"

  - alert: CoverageDropped
    expr: code_coverage_percentage < 80
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Code coverage dropped below threshold"
      description: "Coverage is {{ $value }}%, below 80% threshold"

  - alert: SecurityVulnerabilityDetected
    expr: security_vulnerabilities > 0
    for: 0m
    labels:
      severity: critical
    annotations:
      summary: "Security vulnerability detected"
      description: "{{ $value }} security vulnerabilities found"
```

### Reporting Dashboard
```yaml
# Grafana dashboard panels
panels:
  - title: "Quality Gate Status"
    type: "stat"
    targets:
      - expr: quality_gate_status
        legendFormat: "{{ gate }}"
    
  - title: "Code Coverage Trend"
    type: "graph"
    targets:
      - expr: code_coverage_percentage
        legendFormat: "Coverage %"
    
  - title: "Pipeline Success Rate"
    type: "stat"
    targets:
      - expr: pipeline_success_rate
        legendFormat: "Success Rate"
    
  - title: "Override Usage"
    type: "table"
    targets:
      - expr: override_usage_by_reason
        legendFormat: "{{ reason }}"
```

## Best Practices

### Setting Up Quality Gates
1. **Start with relaxed thresholds** and tighten over time
2. **Include team in threshold setting** to ensure buy-in
3. **Monitor metrics regularly** to adjust thresholds
4. **Document all exceptions** for audit trail

### Maintaining Quality Gates
1. **Regular threshold review** (monthly)
2. **Tool updates** to latest versions
3. **Team training** on quality standards
4. **Automated reporting** for transparency

### Handling Failures
1. **Fast feedback** to developers
2. **Actionable reports** with specific fixes
3. **Coaching** rather than punishment
4. **Continuous improvement** of processes

## Integration Examples

### GitHub Actions
```yaml
# .github/workflows/quality-gates.yml
name: Quality Gates
on: [push, pull_request]

jobs:
  quality-gates:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v3
        with:
          go-version: 1.21
      
      - name: Run Tests
        run: |
          go test -v -coverprofile=coverage.out ./...
          go tool cover -func=coverage.out
      
      - name: Quality Gates
        run: |
          ./scripts/quality-gates.sh
      
      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

### Jenkins Pipeline
```groovy
pipeline {
    agent any
    
    stages {
        stage('Quality Gates') {
            parallel {
                stage('Tests') {
                    steps {
                        sh 'go test -v -coverprofile=coverage.out ./...'
                        sh 'go tool cover -func=coverage.out'
                    }
                }
                
                stage('Linting') {
                    steps {
                        sh 'golangci-lint run'
                    }
                }
                
                stage('Security') {
                    steps {
                        sh 'gosec ./...'
                        sh 'trivy fs .'
                    }
                }
            }
        }
    }
    
    post {
        always {
            publishHTML([
                allowMissing: false,
                alwaysLinkToLastBuild: true,
                keepAll: true,
                reportDir: 'reports',
                reportFiles: 'coverage.html',
                reportName: 'Coverage Report'
            ])
        }
    }
}
```

## Troubleshooting

### Common Issues

#### 1. Coverage Calculation Errors
```bash
# Check if coverage file exists
ls -la coverage.out

# Verify coverage format
head -n 5 coverage.out

# Recalculate coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total
```

#### 2. SonarQube Integration Issues
```bash
# Check SonarQube connection
curl -u admin:admin http://sonarqube:9000/api/system/status

# Verify project exists
curl -u admin:admin http://sonarqube:9000/api/projects/search?projects=apm-stack

# Manual analysis
sonar-scanner -Dsonar.projectKey=apm-stack -Dsonar.verbose=true
```

#### 3. Performance Test Failures
```bash
# Check system resources
free -h
top -p $(pgrep -f "your-app")

# Run isolated performance test
go test -bench=BenchmarkCriticalPath -count=5

# Profile memory usage
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

## Related Documentation
- [CI/CD Pipeline](ci-cd-pipeline.md)
- [CI Configuration](../ci/README.md)
- [Best Practices](best-practices.md)
- [Monitoring Setup](../deployments/kubernetes/prometheus/README.md)