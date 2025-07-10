---
layout: default
title: CLI Reference - APM
description: Complete reference for APM CLI commands
---

# APM CLI Reference

Complete reference guide for all APM CLI commands and options.

## Global Options

These options are available for all commands:

```bash
--config, -c    Path to config file (default: ./apm.yaml)
--verbose, -v   Enable verbose output
--json          Output in JSON format
--no-color      Disable colored output
--help, -h      Show help
--version       Show version information
```

## Commands

### `apm init`

Initialize APM configuration for your project.

```bash
apm init [options]
```

**Options:**
- `--force` - Overwrite existing configuration
- `--minimal` - Create minimal configuration
- `--preset <type>` - Use configuration preset (web, api, microservice)

**Interactive Prompts:**
1. Project name
2. Environment selection
3. APM tools to enable
4. Notification configuration
5. Application entry point

**Example:**
```bash
# Interactive setup
apm init

# Force overwrite with API preset
apm init --force --preset api
```

### `apm run`

Run your application with APM instrumentation and hot reload.

```bash
apm run [command] [options]
```

**Options:**
- `--no-reload` - Disable hot reload
- `--port <port>` - Override application port
- `--env <file>` - Load environment from file
- `--build` - Build before running

**Examples:**
```bash
# Run with default command from apm.yaml
apm run

# Run specific command
apm run "go run cmd/server/main.go"

# Run without hot reload
apm run --no-reload

# Run with custom env file
apm run --env .env.production
```

### `apm test`

Validate configuration and test connectivity to APM tools.

```bash
apm test [options]
```

**Options:**
- `--fix` - Attempt to fix issues automatically
- `--component <name>` - Test specific component only

**Tests performed:**
- Configuration file syntax
- Required fields validation
- APM tool connectivity
- Port availability
- Webhook validation

**Example:**
```bash
# Test all components
apm test

# Test only Prometheus
apm test --component prometheus

# Test and fix issues
apm test --fix
```

### `apm dashboard`

Open interactive dashboard to access monitoring tools.

```bash
apm dashboard [options]
```

**Options:**
- `--list` - List available tools without opening
- `--tool <name>` - Open specific tool directly

**Example:**
```bash
# Interactive selection
apm dashboard

# Open Grafana directly
apm dashboard --tool grafana

# List all tools
apm dashboard --list
```

### `apm deploy`

Deploy your APM-instrumented application to various platforms.

```bash
apm deploy [target] [options]
```

**Targets:**
- `docker` - Build and push Docker image
- `kubernetes` - Deploy to Kubernetes cluster
- `cloud` - Deploy to cloud provider (AWS/Azure/GCP)

**Common Options:**
- `--dry-run` - Preview deployment without executing
- `--rollback` - Rollback to previous version
- `--wait` - Wait for deployment to complete
- `--timeout <duration>` - Deployment timeout

#### Docker Deployment

```bash
apm deploy docker [options]
```

**Options:**
- `--registry <url>` - Docker registry URL
- `--tag <tag>` - Image tag (default: latest)
- `--build-args <args>` - Docker build arguments
- `--push` - Push image after building

**Example:**
```bash
# Build and push to Docker Hub
apm deploy docker --registry docker.io/myorg --tag v1.0.0 --push

# Build only (no push)
apm deploy docker --tag dev
```

#### Kubernetes Deployment

```bash
apm deploy kubernetes [options]
```

**Options:**
- `--namespace <ns>` - Kubernetes namespace
- `--replicas <n>` - Number of replicas
- `--image <image>` - Docker image to deploy
- `--manifests <dir>` - Directory with K8s manifests

**Example:**
```bash
# Deploy to production namespace
apm deploy kubernetes --namespace production --replicas 3

# Deploy with custom manifests
apm deploy kubernetes --manifests ./k8s/production/
```

#### Cloud Deployment

```bash
apm deploy cloud [options]
```

**Options:**
- `--provider <name>` - Cloud provider (aws/azure/gcp)
- `--region <region>` - Deployment region
- `--cluster <name>` - Target cluster name
- `--account <id>` - Account/Project ID

**AWS-specific options:**
- `--role <arn>` - IAM role to assume
- `--mfa` - Enable MFA for deployment
- `--external-id <id>` - External ID for role assumption

**Example:**
```bash
# Deploy to AWS EKS
apm deploy cloud --provider aws --region us-west-2 --cluster prod-cluster

# Deploy with cross-account role
apm deploy cloud --provider aws \
  --role arn:aws:iam::123456789012:role/DeployRole \
  --external-id unique-id
```

### `apm status`

Check deployment status and health.

```bash
apm status [options]
```

**Options:**
- `--deployment <id>` - Check specific deployment
- `--watch` - Continuously watch status
- `--interval <seconds>` - Watch interval

**Example:**
```bash
# Check current status
apm status

# Watch deployment progress
apm status --deployment dep-123 --watch
```

### `apm logs`

View application and APM component logs.

```bash
apm logs [component] [options]
```

**Options:**
- `--follow, -f` - Follow log output
- `--tail <n>` - Number of lines to show
- `--since <duration>` - Show logs since duration
- `--filter <pattern>` - Filter log entries

**Example:**
```bash
# View application logs
apm logs app --follow

# View Prometheus logs
apm logs prometheus --tail 100

# Filter error logs from last hour
apm logs app --since 1h --filter "error|ERROR"
```

### `apm config`

Manage APM configuration.

```bash
apm config [subcommand] [options]
```

**Subcommands:**
- `show` - Display current configuration
- `set` - Set configuration value
- `get` - Get configuration value
- `validate` - Validate configuration

**Example:**
```bash
# Show full configuration
apm config show

# Set Prometheus port
apm config set apm.prometheus.port 9091

# Get Grafana URL
apm config get apm.grafana.url

# Validate configuration
apm config validate
```

## Configuration File

The CLI uses `apm.yaml` configuration file:

```yaml
version: "1.0"
project:
  name: "my-app"
  environment: "production"

apm:
  prometheus:
    enabled: true
    port: 9090
  grafana:
    enabled: true
    port: 3000
  jaeger:
    enabled: true
    ui_port: 16686

application:
  entry_point: "./cmd/main.go"
  build_command: "go build"
  run_command: "./app"
  hot_reload:
    enabled: true
    paths: ["."]
    extensions: [".go"]

deployment:
  docker:
    registry: "ghcr.io/myorg"
  kubernetes:
    namespace: "default"
  cloud:
    provider: "aws"
    region: "us-west-2"
```

## Environment Variables

The CLI respects these environment variables:

```bash
# Override config file location
APM_CONFIG_FILE=/path/to/apm.yaml

# Set environment
APM_ENVIRONMENT=production

# Cloud provider credentials
AWS_PROFILE=production
AZURE_SUBSCRIPTION_ID=xxx
GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json

# Disable color output
NO_COLOR=1

# Enable debug logging
APM_DEBUG=true
```

## Exit Codes

The CLI uses standard exit codes:

- `0` - Success
- `1` - General error
- `2` - Configuration error
- `3` - Connection error
- `4` - Authentication error
- `5` - Validation error
- `126` - Command not executable
- `127` - Command not found

## Examples

### Complete Workflow

```bash
# 1. Initialize project
apm init

# 2. Test configuration
apm test

# 3. Run locally with hot reload
apm run

# 4. Build and test Docker image
apm deploy docker --tag test

# 5. Deploy to staging
apm deploy cloud --provider aws --account staging

# 6. Check deployment status
apm status --watch

# 7. View logs
apm logs app --follow

# 8. Open dashboards
apm dashboard
```

### CI/CD Integration

```bash
#!/bin/bash
# ci-deploy.sh

# Validate configuration
apm test || exit 1

# Build Docker image
apm deploy docker --tag ${CI_COMMIT_SHA} || exit 1

# Deploy to production
apm deploy cloud \
  --provider aws \
  --account production \
  --role ${DEPLOY_ROLE_ARN} \
  --dry-run

# If dry run succeeds, do actual deployment
if [ $? -eq 0 ]; then
  apm deploy cloud \
    --provider aws \
    --account production \
    --role ${DEPLOY_ROLE_ARN} \
    --wait \
    --timeout 10m
fi
```

---

[Back to Home](./index.md) | [Next: API Reference →](https://pkg.go.dev/github.com/chaksack/apm)