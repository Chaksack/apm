# CI Configuration Guide

## Overview

This directory contains the CI/CD configuration files and scripts for the APM stack. The configuration supports GitHub Actions, Jenkins, and other CI/CD platforms with a focus on observability, testing, and deployment automation.

## Quick Start Guide

### Prerequisites

Before setting up the CI pipeline, ensure you have:

1. **Required Tools**:
   - Go 1.21 or higher
   - Docker and Docker Compose
   - kubectl (for Kubernetes deployments)
   - Helm 3.x (for Helm deployments)

2. **Access Requirements**:
   - Container registry access (Docker Hub, ACR, ECR, etc.)
   - Kubernetes cluster access
   - SonarQube server (for code quality)

3. **Environment Variables**:
   ```bash
   # Container registry
   export REGISTRY_URL="your-registry.com"
   export REGISTRY_USERNAME="your-username"
   export REGISTRY_PASSWORD="your-password"
   
   # Kubernetes
   export KUBE_CONFIG_DATA="base64-encoded-kubeconfig"
   export KUBE_NAMESPACE="apm-system"
   
   # SonarQube
   export SONAR_HOST_URL="http://sonarqube:9000"
   export SONAR_LOGIN="your-sonar-token"
   ```

### Initial Setup

1. **Clone and configure**:
   ```bash
   git clone <repository-url>
   cd apm
   cp ci/templates/github-actions.yml .github/workflows/ci.yml
   ```

2. **Configure secrets** (GitHub Actions):
   - `REGISTRY_USERNAME`: Container registry username
   - `REGISTRY_PASSWORD`: Container registry password
   - `KUBE_CONFIG_DATA`: Base64 encoded kubeconfig
   - `SONAR_TOKEN`: SonarQube authentication token

3. **Test local build**:
   ```bash
   make build-app
   make test
   make test-coverage
   ```

4. **Validate configuration**:
   ```bash
   ./ci/scripts/validate-config.sh
   ```

## Pipeline Customization

### GitHub Actions Configuration

#### Basic Workflow
```yaml
# .github/workflows/ci.yml
name: CI/CD Pipeline

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

env:
  GO_VERSION: "1.21"
  REGISTRY: "your-registry.com"
  IMAGE_NAME: "apm-stack"

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v3
        with:
          go-version: ${{ env.GO_VERSION }}
      
      - name: Cache Go modules
        uses: actions/cache@v3
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-
      
      - name: Run Tests
        run: |
          make test
          make test-coverage
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

#### Advanced Features
```yaml
  security-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Run Gosec Security Scanner
        uses: securecodewarrior/github-action-gosec@master
        with:
          args: '-fmt sarif -out gosec.sarif ./...'
      
      - name: Upload SARIF file
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: gosec.sarif
  
  build-and-push:
    needs: [test, security-scan]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build Docker image
        run: |
          docker build -t ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.sha }} .
          docker build -t ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:latest .
      
      - name: Push to registry
        run: |
          echo ${{ secrets.REGISTRY_PASSWORD }} | docker login ${{ env.REGISTRY }} -u ${{ secrets.REGISTRY_USERNAME }} --password-stdin
          docker push ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.sha }}
          docker push ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:latest
```

### Jenkins Configuration

#### Pipeline Script
```groovy
pipeline {
    agent any
    
    environment {
        GO_VERSION = '1.21'
        REGISTRY = 'your-registry.com'
        IMAGE_NAME = 'apm-stack'
        DOCKER_REGISTRY = credentials('docker-registry')
        SONAR_TOKEN = credentials('sonar-token')
    }
    
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        
        stage('Setup') {
            steps {
                sh '''
                    # Install Go
                    wget https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz
                    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
                    export PATH=$PATH:/usr/local/go/bin
                    
                    # Verify installation
                    go version
                '''
            }
        }
        
        stage('Test') {
            parallel {
                stage('Unit Tests') {
                    steps {
                        sh '''
                            export PATH=$PATH:/usr/local/go/bin
                            make test
                            make test-coverage
                        '''
                    }
                    post {
                        always {
                            publishHTML([
                                allowMissing: false,
                                alwaysLinkToLastBuild: true,
                                keepAll: true,
                                reportDir: 'coverage',
                                reportFiles: 'index.html',
                                reportName: 'Coverage Report'
                            ])
                        }
                    }
                }
                
                stage('Security Scan') {
                    steps {
                        sh '''
                            export PATH=$PATH:/usr/local/go/bin
                            go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
                            gosec -fmt json -out gosec-report.json ./...
                        '''
                    }
                    post {
                        always {
                            archiveArtifacts artifacts: 'gosec-report.json', fingerprint: true
                        }
                    }
                }
                
                stage('Code Quality') {
                    steps {
                        sh '''
                            export PATH=$PATH:/usr/local/go/bin
                            sonar-scanner \
                                -Dsonar.projectKey=apm-stack \
                                -Dsonar.sources=. \
                                -Dsonar.host.url=${SONAR_HOST_URL} \
                                -Dsonar.login=${SONAR_TOKEN}
                        '''
                    }
                }
            }
        }
        
        stage('Build') {
            steps {
                sh '''
                    export PATH=$PATH:/usr/local/go/bin
                    make build-app
                    
                    # Build Docker image
                    docker build -t ${REGISTRY}/${IMAGE_NAME}:${BUILD_NUMBER} .
                    docker build -t ${REGISTRY}/${IMAGE_NAME}:latest .
                '''
            }
        }
        
        stage('Push') {
            steps {
                sh '''
                    echo ${DOCKER_REGISTRY_PSW} | docker login ${REGISTRY} -u ${DOCKER_REGISTRY_USR} --password-stdin
                    docker push ${REGISTRY}/${IMAGE_NAME}:${BUILD_NUMBER}
                    docker push ${REGISTRY}/${IMAGE_NAME}:latest
                '''
            }
        }
        
        stage('Deploy') {
            when {
                branch 'main'
            }
            steps {
                sh '''
                    # Deploy to Kubernetes
                    kubectl set image deployment/apm-app apm-app=${REGISTRY}/${IMAGE_NAME}:${BUILD_NUMBER} -n apm-system
                    kubectl rollout status deployment/apm-app -n apm-system
                '''
            }
        }
    }
    
    post {
        always {
            cleanWs()
        }
        failure {
            emailext (
                subject: "Build Failed: ${env.JOB_NAME} - ${env.BUILD_NUMBER}",
                body: "Build failed. Check console output at ${env.BUILD_URL}",
                to: "${env.CHANGE_AUTHOR_EMAIL}"
            )
        }
    }
}
```

### GitLab CI Configuration

#### .gitlab-ci.yml
```yaml
stages:
  - test
  - build
  - deploy

variables:
  GO_VERSION: "1.21"
  DOCKER_DRIVER: overlay2
  REGISTRY: "your-registry.com"
  IMAGE_NAME: "apm-stack"

before_script:
  - apt-get update -qq && apt-get install -y -qq git
  - wget https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz
  - tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
  - export PATH=$PATH:/usr/local/go/bin
  - go version

test:
  stage: test
  script:
    - make test
    - make test-coverage
  coverage: '/coverage: \d+\.\d+% of statements/'
  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: coverage.xml

security-scan:
  stage: test
  script:
    - go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    - gosec -fmt json -out gosec-report.json ./...
  artifacts:
    reports:
      sast: gosec-report.json

build:
  stage: build
  services:
    - docker:dind
  script:
    - make build-app
    - docker build -t ${REGISTRY}/${IMAGE_NAME}:${CI_COMMIT_SHA} .
    - docker build -t ${REGISTRY}/${IMAGE_NAME}:latest .
    - echo ${REGISTRY_PASSWORD} | docker login ${REGISTRY} -u ${REGISTRY_USERNAME} --password-stdin
    - docker push ${REGISTRY}/${IMAGE_NAME}:${CI_COMMIT_SHA}
    - docker push ${REGISTRY}/${IMAGE_NAME}:latest
  only:
    - main
    - develop

deploy:
  stage: deploy
  image: bitnami/kubectl:latest
  script:
    - kubectl config use-context ${KUBE_CONTEXT}
    - kubectl set image deployment/apm-app apm-app=${REGISTRY}/${IMAGE_NAME}:${CI_COMMIT_SHA} -n apm-system
    - kubectl rollout status deployment/apm-app -n apm-system
  only:
    - main
```

## Configuration Templates

### Template Structure
```
ci/
├── templates/
│   ├── github-actions.yml      # GitHub Actions template
│   ├── jenkins-pipeline.groovy # Jenkins pipeline template
│   ├── gitlab-ci.yml          # GitLab CI template
│   └── azure-pipelines.yml    # Azure Pipelines template
├── scripts/
│   ├── setup-ci.sh           # CI setup script
│   ├── validate-config.sh    # Configuration validation
│   └── quality-gates.sh      # Quality gate checks
└── quality-gates/
    ├── sonar-project.properties
    ├── .golangci.yml
    └── trivy-config.yaml
```

### Quality Gates Configuration

#### SonarQube Project Properties
```properties
# sonar-project.properties
sonar.projectKey=apm-stack
sonar.projectName=APM Stack
sonar.projectVersion=1.0.0

# Source settings
sonar.sources=.
sonar.exclusions=vendor/**,**/*_test.go,**/testdata/**,**/*.pb.go

# Test settings
sonar.tests=.
sonar.test.inclusions=**/*_test.go
sonar.test.exclusions=vendor/**

# Coverage settings
sonar.go.coverage.reportPaths=coverage.out

# Quality gate settings
sonar.qualitygate.wait=true
sonar.qualitygate.timeout=300
```

#### Linting Configuration
```yaml
# .golangci.yml
linters:
  disable-all: true
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
    - goimports
    - gosec
    - misspell
    - unconvert
    - dupl
    - goconst
    - gocognit

linters-settings:
  gocyclo:
    min-complexity: 15
  gocognit:
    min-complexity: 15
  dupl:
    threshold: 100
  goconst:
    min-len: 2
    min-occurrences: 2
  misspell:
    locale: US
  gosec:
    excludes:
      - G304  # Allow file path from variable
      - G101  # Allow hardcoded credentials in test files
    exclude-generated: true

run:
  timeout: 5m
  issues-exit-code: 1
  tests: false
  skip-dirs:
    - vendor
    - testdata
  skip-files:
    - ".*\\.pb\\.go$"
    - ".*_generated\\.go$"

issues:
  exclude-use-default: false
  exclude-rules:
    - path: _test\.go
      linters:
        - gosec
        - dupl
    - path: cmd/
      linters:
        - gocyclo
        - gocognit
```

## Troubleshooting Guide

### Common Issues and Solutions

#### 1. Build Failures

**Issue**: Go build fails with missing dependencies
```bash
# Error: cannot find package "github.com/example/package"
go: github.com/example/package@v1.0.0: reading github.com/example/package/go.mod: 404 Not Found
```

**Solution**:
```bash
# Clean module cache
go clean -modcache

# Update dependencies
go mod tidy
go mod download

# Verify go.mod and go.sum
go mod verify
```

#### 2. Test Failures

**Issue**: Tests fail in CI but pass locally
```bash
# Common causes:
# - Different Go versions
# - Missing environment variables
# - Different timezone settings
# - Race conditions
```

**Solution**:
```bash
# Match CI environment locally
export GO_VERSION=1.21
export TZ=UTC
export CGO_ENABLED=0

# Run tests with same flags as CI
go test -v -race -timeout=5m ./...

# Run tests in Docker (closer to CI environment)
docker run --rm -v $(pwd):/app -w /app golang:1.21 go test -v ./...
```

#### 3. Docker Build Issues

**Issue**: Docker build fails with permission errors
```bash
# Error: permission denied while trying to connect to Docker daemon
```

**Solution**:
```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Or use sudo for CI
sudo docker build -t myapp .

# For rootless Docker
export DOCKER_HOST=unix:///run/user/$(id -u)/docker.sock
```

#### 4. Quality Gate Failures

**Issue**: SonarQube quality gate fails
```bash
# Check SonarQube logs
curl -u admin:admin "http://sonarqube:9000/api/qualitygates/project_status?projectKey=apm-stack"

# Common issues:
# - Coverage below threshold
# - Code duplication
# - Security vulnerabilities
# - Complexity issues
```

**Solution**:
```bash
# Generate detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Fix linting issues
golangci-lint run --fix

# Check security issues
gosec -fmt json -out gosec-report.json ./...
```

#### 5. Deployment Issues

**Issue**: Kubernetes deployment fails
```bash
# Check deployment status
kubectl get deployments -n apm-system
kubectl describe deployment apm-app -n apm-system

# Check pod logs
kubectl logs -f deployment/apm-app -n apm-system
```

**Solution**:
```bash
# Verify image exists
docker pull your-registry.com/apm-stack:latest

# Check resource limits
kubectl top pods -n apm-system

# Verify secrets and configmaps
kubectl get secrets -n apm-system
kubectl get configmaps -n apm-system

# Manual rollback if needed
kubectl rollout undo deployment/apm-app -n apm-system
```

### Debug Commands

#### CI Environment Debug
```bash
# Check environment variables
env | grep -E "(GO_|DOCKER_|KUBE_|SONAR_)"

# Verify tool versions
go version
docker version
kubectl version --client
helm version

# Check available resources
free -h
df -h
nproc
```

#### Build Debug
```bash
# Verbose build
go build -v -x ./cmd/apm

# Check build cache
go env GOCACHE
ls -la $(go env GOCACHE)

# Clean build
go clean -cache
go clean -modcache
go clean -testcache
```

#### Test Debug
```bash
# Run specific test
go test -v -run TestSpecificFunction ./pkg/service

# Debug test with delve
dlv test ./pkg/service -- -test.run TestSpecificFunction

# Check test coverage for specific package
go test -coverprofile=coverage.out ./pkg/service
go tool cover -func=coverage.out
```

### Performance Optimization

#### CI Pipeline Optimization
```yaml
# Cache dependencies
- name: Cache Go modules
  uses: actions/cache@v3
  with:
    path: ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}

# Parallel execution
jobs:
  test:
    strategy:
      matrix:
        go-version: [1.21]
        os: [ubuntu-latest]
    runs-on: ${{ matrix.os }}

# Use faster runners
runs-on: ubuntu-latest-4-cores
```

#### Build Optimization
```dockerfile
# Multi-stage build
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/apm

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
CMD ["./main"]
```

## Best Practices

### CI/CD Pipeline Best Practices

1. **Fast Feedback**:
   - Keep pipeline execution under 10 minutes
   - Run tests in parallel
   - Use caching effectively
   - Fail fast on critical errors

2. **Security**:
   - Never commit secrets to repository
   - Use encrypted secrets/variables
   - Scan for vulnerabilities regularly
   - Implement least-privilege access

3. **Reliability**:
   - Use deterministic builds
   - Implement proper error handling
   - Have rollback procedures
   - Monitor pipeline health

4. **Maintainability**:
   - Use infrastructure as code
   - Document all configurations
   - Regular dependency updates
   - Clear naming conventions

### Code Quality Best Practices

1. **Testing**:
   - Write tests before code (TDD)
   - Maintain high test coverage
   - Use meaningful test names
   - Test edge cases

2. **Code Style**:
   - Follow Go style guidelines
   - Use consistent formatting
   - Add meaningful comments
   - Implement proper error handling

3. **Performance**:
   - Profile critical paths
   - Optimize memory usage
   - Use appropriate data structures
   - Implement caching where needed

## Related Documentation

- [CI/CD Pipeline](../docs/ci-cd-pipeline.md)
- [Quality Gates](../docs/quality-gates.md)
- [Scripts Documentation](../scripts/README.md)
- [Deployment Guide](../deployments/README.md)

## Support

For issues and questions:
- Create an issue in the repository
- Contact the DevOps team
- Check the troubleshooting guide
- Review the monitoring dashboards