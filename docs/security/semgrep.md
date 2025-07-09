# Semgrep Security Analysis

## Overview

Semgrep is a static analysis tool that helps identify security vulnerabilities, code quality issues, and performance anti-patterns in the APM codebase. It's integrated into our CI/CD pipelines to provide continuous security scanning and code quality checks.

## Features

Our Semgrep configuration includes rules for:

### Security Vulnerabilities
- **Hardcoded Credentials**: Detects passwords, API keys, and tokens in code
- **SQL Injection**: Identifies potential SQL injection vulnerabilities
- **Command Injection**: Flags unsafe command execution patterns
- **Weak Cryptography**: Warns about usage of weak cryptographic algorithms

### GoFiber-Specific Security
- **CSRF Protection**: Checks for missing CSRF middleware
- **CORS Configuration**: Identifies insecure CORS settings
- **Rate Limiting**: Alerts when rate limiting is not configured
- **Cookie Security**: Ensures cookies are marked as secure
- **Security Headers**: Checks for missing helmet middleware

### Code Quality
- **Error Handling**: Detects empty error handling blocks
- **Resource Management**: Identifies defer statements in loops and missing context cancellation
- **Type Safety**: Flags unchecked type assertions
- **Context Propagation**: Ensures proper context propagation to goroutines

### Performance Anti-patterns
- **String Concatenation**: Identifies inefficient string concatenation in loops
- **Memory Efficiency**: Detects unnecessary slice allocations
- **Regex Compilation**: Suggests moving regex compilation to package level
- **Memory Leaks**: Warns about potential time.After leaks in select loops

### Secret Detection
- **API Keys**: Generic API key patterns
- **AWS Credentials**: AWS access key detection
- **Private Keys**: RSA, EC, DSA, and OpenSSH private keys
- **JWT Tokens**: Hardcoded JWT tokens

### APM-Specific Patterns
- **Missing Instrumentation**: Checks for APM middleware configuration
- **Error Tracking**: Ensures errors are properly logged or tracked
- **Context Propagation**: Verifies context propagation for distributed tracing

## Configuration

The Semgrep rules are defined in `.semgrep.yml` at the root of the repository. Each rule includes:
- **ID**: Unique identifier for the rule
- **Pattern**: Code pattern to match
- **Message**: Description of the issue and suggested fix
- **Severity**: ERROR, WARNING, or INFO
- **Metadata**: Additional information like CWE ID and OWASP category

## Running Semgrep Locally

### Installation

```bash
# Using pip
pip install semgrep

# Using brew (macOS)
brew install semgrep

# Using Docker
docker pull returntocorp/semgrep
```

### Basic Usage

```bash
# Run all rules
semgrep --config=./.semgrep.yml .

# Run specific rule
semgrep --config=./.semgrep.yml --include='*.go' --pattern='$X := $Y.($TYPE)' .

# Output results in different formats
semgrep --config=./.semgrep.yml --json --output=results.json .
semgrep --config=./.semgrep.yml --sarif --output=results.sarif .
```

### Filtering Results

```bash
# Run only ERROR severity rules
semgrep --config=./.semgrep.yml --severity=ERROR .

# Exclude test files
semgrep --config=./.semgrep.yml --exclude='*_test.go' --exclude='tests/' .

# Run only security rules
semgrep --config=./.semgrep.yml --json . | jq '.results[] | select(.extra.metadata.category == "security")'
```

## CI/CD Integration

### GitHub Actions

Semgrep is integrated into the GitHub Actions workflow (`.github/workflows/ci.yml`):

```yaml
- name: Run Semgrep security scan
  uses: returntocorp/semgrep-action@v1
  with:
    config: ./.semgrep.yml
    generateSarif: true
    
- name: Upload Semgrep results
  uses: github/codeql-action/upload-sarif@v3
  if: always()
  with:
    sarif_file: semgrep.sarif
```

Results are uploaded to GitHub Security tab for easy review.

### GitLab CI

Semgrep is integrated into the GitLab CI pipeline (`.gitlab-ci.yml`):

```yaml
security:semgrep:
  stage: security
  image: returntocorp/semgrep:latest
  script:
    - semgrep --config=./.semgrep.yml --json --output=semgrep-report.json .
    - semgrep --config=./.semgrep.yml --verbose .
  artifacts:
    reports:
      sast: semgrep-report.json
```

Results are available in the GitLab Security Dashboard.

## Suppressing False Positives

### Inline Suppression

```go
// nosemgrep: go-hardcoded-credentials
apiKey := "test-key-for-testing"
```

### File-level Suppression

```go
// nosemgrep
package main
```

### Configuration Suppression

Add to `.semgrep.yml`:

```yaml
paths:
  exclude:
    - tests/
    - vendor/
    - '*.test.go'
```

## Writing Custom Rules

### Basic Pattern

```yaml
rules:
  - id: custom-rule
    pattern: |
      if $COND {
        panic($MSG)
      }
    message: "Avoid using panic in production code"
    languages: [go]
    severity: WARNING
```

### Pattern with Metavariables

```yaml
rules:
  - id: custom-log-check
    patterns:
      - pattern: $LOG.$METHOD($MSG)
      - metavariable-regex:
          metavariable: $METHOD
          regex: '(Debug|Info|Warn|Error)'
      - pattern-not: $LOG.$METHOD($MSG, ...)
    message: "Use structured logging with fields"
    languages: [go]
    severity: INFO
```

## Best Practices

1. **Regular Updates**: Keep Semgrep rules updated to catch new vulnerability patterns
2. **Custom Rules**: Create project-specific rules for your coding standards
3. **Progressive Enhancement**: Start with high-severity issues and gradually add more rules
4. **Team Review**: Review and discuss findings in code reviews
5. **Continuous Monitoring**: Monitor trends in security findings over time

## Integration with Other Tools

Semgrep complements other security tools in our pipeline:

- **Trivy**: Container and dependency scanning
- **Gosec**: Go-specific security analysis
- **SonarQube**: Code quality and security analysis
- **Govulncheck**: Go vulnerability database checks

## Troubleshooting

### Common Issues

1. **High False Positive Rate**
   - Review and tune rule patterns
   - Use more specific patterns
   - Add appropriate suppressions

2. **Performance Issues**
   - Exclude large generated files
   - Use `--jobs` flag for parallel processing
   - Consider running subset of rules in CI

3. **Missing Vulnerabilities**
   - Ensure rules cover your specific use cases
   - Add custom rules for project-specific patterns
   - Combine with other security tools

### Getting Help

- [Semgrep Documentation](https://semgrep.dev/docs/)
- [Rule Registry](https://semgrep.dev/r)
- [Community Slack](https://r2c.dev/slack)
- Internal security team: security@example.com

## Metrics and Reporting

Track security metrics over time:

```bash
# Generate metrics report
semgrep --config=./.semgrep.yml --json . | jq '{
  total: .results | length,
  by_severity: .results | group_by(.extra.severity) | map({severity: .[0].extra.severity, count: length}),
  by_category: .results | group_by(.extra.metadata.category) | map({category: .[0].extra.metadata.category, count: length})
}'
```

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CWE Database](https://cwe.mitre.org/)
- [Go Security Best Practices](https://golang.org/doc/security)
- [GoFiber Security Guide](https://docs.gofiber.io/guide/security)