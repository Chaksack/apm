# APM Solution Configuration Documentation

This document provides a comprehensive overview of all configuration options and settings available in the APM (Application Performance Monitoring) solution.

## Table of Contents
- [Configuration Overview](#configuration-overview)
- [Configuration Sources](#configuration-sources)
- [Core APM Configuration](#core-apm-configuration)
- [Instrumentation Configuration](#instrumentation-configuration)
- [Component-Specific Configuration](#component-specific-configuration)
- [Docker Compose Configuration](#docker-compose-configuration)
- [Kubernetes/Helm Configuration](#kubernetes-helm-configuration)
- [Code Quality Configuration](#code-quality-configuration)
- [Security Configuration](#security-configuration)

## Configuration Overview

The APM solution uses a hierarchical configuration system with the following precedence (highest to lowest):
1. Environment variables
2. Configuration files (YAML)
3. Default values

## Configuration Sources

### 1. Main Configuration File
- **Location**: `configs/config.yaml`
- **Format**: YAML
- **Purpose**: Primary configuration for APM components

### 2. Environment Variables
- **Prefix**: `APM_` for main config, various prefixes for instrumentation
- **Format**: Uppercase with underscores replacing dots
- **Example**: `APM_PROMETHEUS_ENDPOINT=http://prometheus:9090`

### 3. Default Values
- Built into the application code
- Provides sensible defaults for all settings

## Core APM Configuration

### Server Configuration
```yaml
server:
  port: ":8080"              # Server port (env: APM_SERVER_PORT)
  read_timeout: "10s"        # Read timeout (env: APM_SERVER_READ_TIMEOUT)
  write_timeout: "10s"       # Write timeout (env: APM_SERVER_WRITE_TIMEOUT)
  prefork: false             # Enable prefork mode (env: APM_SERVER_PREFORK)
```

### Prometheus Configuration
```yaml
prometheus:
  endpoint: "http://localhost:9090"    # Prometheus server URL (env: APM_PROMETHEUS_ENDPOINT)
  scrape_interval: "15s"               # Metrics scrape interval (env: APM_PROMETHEUS_SCRAPE_INTERVAL)
  evaluation_interval: "15s"           # Rule evaluation interval (env: APM_PROMETHEUS_EVALUATION_INTERVAL)
```

### Grafana Configuration
```yaml
grafana:
  endpoint: "http://localhost:3000"    # Grafana server URL (env: APM_GRAFANA_ENDPOINT)
  api_key: ""                          # API key for auth (env: APM_GRAFANA_API_KEY)
  org_id: 1                            # Organization ID (env: APM_GRAFANA_ORG_ID)
```

### Loki Configuration
```yaml
loki:
  endpoint: "http://localhost:3100"                      # Loki query endpoint (env: APM_LOKI_ENDPOINT)
  push_endpoint: "http://localhost:3100/loki/api/v1/push" # Push endpoint (env: APM_LOKI_PUSH_ENDPOINT)
  query_timeout: "30s"                                   # Query timeout (env: APM_LOKI_QUERY_TIMEOUT)
```

### Jaeger Configuration
```yaml
jaeger:
  endpoint: "http://localhost:16686"                # UI endpoint (env: APM_JAEGER_ENDPOINT)
  collector_endpoint: "http://localhost:14268/api/traces" # Collector endpoint (env: APM_JAEGER_COLLECTOR_ENDPOINT)
  agent_host: "localhost"                          # Agent host (env: APM_JAEGER_AGENT_HOST)
  agent_port: 6831                                 # Agent port (env: APM_JAEGER_AGENT_PORT)
```

### AlertManager Configuration
```yaml
alertmanager:
  endpoint: "http://localhost:9093"    # AlertManager endpoint (env: APM_ALERTMANAGER_ENDPOINT)
  webhook_url: ""                      # Custom webhook URL (env: APM_ALERTMANAGER_WEBHOOK_URL)
  resolve_timeout: "5m"                # Alert resolve timeout (env: APM_ALERTMANAGER_RESOLVE_TIMEOUT)
```

### Notification Configuration
```yaml
notifications:
  email:
    smtp_host: "smtp.gmail.com"        # SMTP server (env: APM_NOTIFICATIONS_EMAIL_SMTP_HOST)
    smtp_port: 587                     # SMTP port (env: APM_NOTIFICATIONS_EMAIL_SMTP_PORT)
    smtp_username: ""                  # SMTP username (env: APM_NOTIFICATIONS_EMAIL_SMTP_USERNAME)
    smtp_password: ""                  # SMTP password (env: APM_NOTIFICATIONS_EMAIL_SMTP_PASSWORD)
    smtp_from: "apm-alerts@example.com" # From address (env: APM_NOTIFICATIONS_EMAIL_SMTP_FROM)
    smtp_tls_enabled: true             # Enable TLS (env: APM_NOTIFICATIONS_EMAIL_SMTP_TLS_ENABLED)
  
  slack:
    webhook_url: ""                    # Slack webhook URL (env: APM_NOTIFICATIONS_SLACK_WEBHOOK_URL)
    channel: "#alerts"                 # Default channel (env: APM_NOTIFICATIONS_SLACK_CHANNEL)
    username: "APM Bot"                # Bot username (env: APM_NOTIFICATIONS_SLACK_USERNAME)
```

### Kubernetes Configuration
```yaml
kubernetes:
  namespace: "default"                 # Default namespace (env: APM_KUBERNETES_NAMESPACE)
  in_cluster: false                    # Running in cluster (env: APM_KUBERNETES_IN_CLUSTER)
  config_path: ""                      # Kubeconfig path (env: APM_KUBERNETES_CONFIG_PATH)
  label_selectors:                     # Label selectors for discovery
    - "app.kubernetes.io/managed-by=apm"
    - "monitoring=enabled"
```

### Service Discovery Configuration
```yaml
service_discovery:
  enabled: true                        # Enable discovery (env: APM_SERVICE_DISCOVERY_ENABLED)
  refresh_interval: "30s"              # Refresh interval (env: APM_SERVICE_DISCOVERY_REFRESH_INTERVAL)
  namespaces:                          # Namespaces to scan
    - "default"
    - "monitoring"
    - "production"
  service_selectors:                   # Service label selectors
    - "app.kubernetes.io/part-of=microservices"
    - "tier=backend"
  pod_selectors:                       # Pod label selectors
    - "app.kubernetes.io/component=api"
    - "monitoring.enabled=true"
```

## Instrumentation Configuration

### Service Configuration
- `SERVICE_NAME`: Application service name (default: "app")
- `ENVIRONMENT`: Environment name (default: "development")
- `VERSION`: Application version (default: "unknown")

### Metrics Configuration
- `METRICS_ENABLED`: Enable metrics collection (default: true)
- `METRICS_NAMESPACE`: Prometheus namespace for metrics
- `METRICS_SUBSYSTEM`: Prometheus subsystem for metrics
- `METRICS_PATH`: Metrics endpoint path (default: "/metrics")

### Logging Configuration
- `LOG_LEVEL`: Log level (debug, info, warn, error) (default: "info")
- `LOG_ENCODING`: Log encoding (json, console) (default: "json")
- `LOG_DEVELOPMENT`: Development mode (default: false)
- `LOG_OUTPUT_PATHS`: Comma-separated output paths (default: "stdout")
- `LOG_ERROR_OUTPUT_PATHS`: Error output paths (default: "stderr")
- `LOG_ENABLE_CALLER`: Include caller info (default: false)
- `LOG_ENABLE_STACKTRACE`: Include stack traces (default: false)

### OpenTelemetry Configuration
- `OTEL_SERVICE_NAME`: Service name for tracing
- `OTEL_EXPORTER_OTLP_ENDPOINT`: OTLP endpoint
- `OTEL_EXPORTER_OTLP_INSECURE`: Use insecure connection
- `OTEL_TRACES_EXPORTER`: Traces exporter type
- `OTEL_METRICS_EXPORTER`: Metrics exporter type

### Jaeger Configuration (Legacy)
- `JAEGER_AGENT_HOST`: Jaeger agent host
- `JAEGER_AGENT_PORT`: Jaeger agent port
- `JAEGER_SERVICE_NAME`: Service name
- `JAEGER_SAMPLER_TYPE`: Sampler type (const, probabilistic, etc.)
- `JAEGER_SAMPLER_PARAM`: Sampler parameter

## Component-Specific Configuration

### Prometheus Container
- `TZ`: Timezone (default: UTC)
- Storage path: `/prometheus`
- Config file: `/etc/prometheus/prometheus.yml`
- Web lifecycle API enabled

### Grafana Container
- `GF_SECURITY_ADMIN_USER`: Admin username (default: admin)
- `GF_SECURITY_ADMIN_PASSWORD`: Admin password
- `GF_INSTALL_PLUGINS`: Comma-separated plugin list
- `GF_SERVER_ROOT_URL`: Server root URL
- `GF_SMTP_ENABLED`: Enable SMTP

### Loki Container
- Config file: `/etc/loki/local-config.yaml`
- Storage path: `/loki`

### Jaeger Container
- `COLLECTOR_OTLP_ENABLED`: Enable OTLP collector
- `SPAN_STORAGE_TYPE`: Storage backend (badger, elasticsearch, cassandra)
- `BADGER_EPHEMERAL`: Ephemeral storage
- `BADGER_DIRECTORY_VALUE`: Data directory
- `BADGER_DIRECTORY_KEY`: Key directory

## Docker Compose Configuration

### Network Configuration
- Network name: `apm-network`
- Subnet: `172.28.0.0/16`

### Volume Configuration
Persistent volumes for:
- `prometheus_data`
- `grafana_data`
- `loki_data`
- `jaeger_data`
- `alertmanager_data`

### Port Mappings
- Prometheus: 9090
- Grafana: 3000
- Loki: 3100
- Jaeger UI: 16686
- Jaeger Agent: 6831/udp
- AlertManager: 9093
- Node Exporter: 9100
- cAdvisor: 8090 (mapped from 8080)
- Sample App: 8080, 9091 (metrics)

## Kubernetes/Helm Configuration

### Global Settings
```yaml
global:
  namespace: apm-system
  labels:
    app.kubernetes.io/part-of: apm-stack
    app.kubernetes.io/managed-by: helm
  storageClass: ""
```

### Component Toggles
```yaml
components:
  prometheus:
    enabled: true
  grafana:
    enabled: true
  loki:
    enabled: true
  jaeger:
    enabled: true
  alertmanager:
    enabled: true
```

### Resource Limits (per component)
Example for Prometheus:
```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "250m"
  limits:
    memory: "2Gi"
    cpu: "1000m"
```

### Storage Configuration
```yaml
storage:
  size: 10Gi
  retentionDays: 15
```

## Code Quality Configuration

### SonarQube Settings
- Project Key: `apm-solution`
- Language: Go
- Coverage reports: `coverage.out`
- Quality gate: Enabled with wait
- Security hotspots: Max 0, blocking

### Exclusions
- Test files: `**/*_test.go`
- Vendor: `**/vendor/**`
- Generated: `**/*.pb.go`, `**/generated/**`
- Documentation: `**/*.md`
- Configuration: `**/*.yml`, `**/*.yaml`

## Security Configuration

### Semgrep Analyzer Configuration
```go
type Config struct {
    SemgrepPath:       "semgrep"
    RuleSet:           "auto"
    OutputFormat:      "json"
    Timeout:           30 * time.Minute
    MaxMemory:         2048 // 2GB
    Jobs:              0     // auto-detect
    MetricsEnabled:    true
    MetricsPrefix:     "apm_semgrep"
    SeverityThreshold: SeverityInfo
    CacheDir:          "/tmp/semgrep-cache"
}
```

### Security Scanning Options
- Rule sets: auto, security, performance
- Output formats: json, sarif, text, junit-xml
- Custom rules support
- GitIgnore handling
- Severity filtering

## Performance Tuning Options

### Application Performance
- `server.prefork`: Enable fiber prefork mode for better performance
- `server.read_timeout`: Adjust based on expected request duration
- `server.write_timeout`: Adjust based on response size

### Monitoring Performance
- `prometheus.scrape_interval`: Balance between data granularity and load
- `prometheus.evaluation_interval`: Adjust based on alert requirements
- `service_discovery.refresh_interval`: Balance between discovery speed and API load

### Resource Limits
- Container memory and CPU limits in Kubernetes/Docker
- Semgrep analyzer memory limits
- Storage retention policies

## Feature Flags

### Service Discovery
- `service_discovery.enabled`: Toggle automatic service discovery

### Metrics Collection
- `METRICS_ENABLED`: Toggle metrics collection in instrumentation

### Logging Features
- `LOG_DEVELOPMENT`: Enable development mode with prettier output
- `LOG_ENABLE_CALLER`: Include caller information
- `LOG_ENABLE_STACKTRACE`: Include stack traces for errors

## Best Practices

1. **Environment-Specific Configuration**
   - Use environment variables for sensitive data
   - Keep base config in YAML files
   - Override per environment as needed

2. **Security**
   - Never commit secrets to version control
   - Use Kubernetes secrets for sensitive data
   - Rotate API keys and passwords regularly

3. **Performance**
   - Adjust scrape intervals based on needs
   - Set appropriate resource limits
   - Enable prefork for high-load scenarios

4. **Monitoring**
   - Enable all relevant exporters
   - Configure appropriate retention periods
   - Set up alerting thresholds

5. **Development vs Production**
   - Use development mode for local testing
   - Disable verbose logging in production
   - Adjust timeouts for production workloads