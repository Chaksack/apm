# APM CLI Tool Specification

## Overview

The APM CLI tool provides a unified interface for managing and interacting with the Application Performance Monitoring stack. It simplifies setup, development, testing, and monitoring access through four main commands.

## Command Structure

### Global Flags

```bash
apm [global-flags] <command> [command-flags]
```

**Global Flags:**
- `--config, -c` (string): Path to config file (default: "./apm.yaml")
- `--verbose, -v` (bool): Enable verbose output
- `--debug` (bool): Enable debug logging
- `--json` (bool): Output in JSON format
- `--no-color` (bool): Disable colored output
- `--help, -h` (bool): Show help information
- `--version` (bool): Show version information

## Commands

### 1. `apm init` - Interactive Setup Wizard

Initialize a new APM configuration for your project.

```bash
apm init [flags]
```

**Flags:**
- `--name, -n` (string): Project name (default: current directory name)
- `--type, -t` (string): Project type (gofiber|generic) (default: auto-detect)
- `--env, -e` (string): Environment (local|docker|kubernetes) (default: "local")
- `--force, -f` (bool): Overwrite existing configuration
- `--template` (string): Use configuration template (minimal|standard|full)
- `--skip-validation` (bool): Skip configuration validation

**Interactive Flow:**

1. **Project Detection**
   ```
   🔍 Detecting project type...
   ✓ GoFiber project detected (go.mod found)
   
   ? Project name: (my-app)
   ? Environment: 
     ▸ Local Development
       Docker Compose
       Kubernetes
   ```

2. **Component Selection**
   ```
   ? Select monitoring components to enable:
     ▸ ✓ Prometheus (Metrics)
       ✓ Grafana (Visualization)
       ✓ Loki (Logs)
       ✓ Jaeger (Tracing)
       ✓ AlertManager (Alerts)
       ○ SonarQube (Code Quality)
   ```

3. **Service Configuration**
   ```
   ? Configure Prometheus:
     Scrape interval (15s): 
     Retention period (15d): 
     
   ? Configure Grafana:
     Admin password: ****
     Enable anonymous access? (Y/n)
   ```

4. **Instrumentation Setup**
   ```
   ? Enable automatic instrumentation? (Y/n)
   ? Select instrumentation features:
     ▸ ✓ HTTP Metrics
       ✓ Distributed Tracing
       ✓ Structured Logging
       ○ Custom Metrics
   ```

5. **Output Generation**
   ```
   📝 Generating configuration files...
   ✓ Created apm.yaml
   ✓ Created docker-compose.apm.yml
   ✓ Created configs/prometheus.yml
   ✓ Created configs/grafana/datasources.yml
   
   🎉 APM initialization complete!
   
   Next steps:
   1. Run 'apm run' to start the monitoring stack
   2. Run 'apm test' to validate the setup
   3. Run 'apm dashboard' to access monitoring UIs
   ```

### 2. `apm run` - Run Application with Hot Reload

Start the application and monitoring stack with development features.

```bash
apm run [flags] [-- application-args]
```

**Flags:**
- `--stack-only` (bool): Run only the monitoring stack
- `--app-only` (bool): Run only the application
- `--hot-reload, -r` (bool): Enable hot reload (default: true)
- `--port, -p` (int): Application port (default: from config)
- `--env-file` (string): Environment file to load
- `--detach, -d` (bool): Run in background
- `--follow, -f` (bool): Follow log output
- `--tail` (int): Number of lines to tail (default: 100)

**Output Format:**

```
🚀 Starting APM stack...

[APM] Starting monitoring components...
  ✓ Prometheus    http://localhost:9090    [RUNNING]
  ✓ Grafana       http://localhost:3000    [RUNNING]
  ✓ Jaeger        http://localhost:16686   [RUNNING]
  ✓ Loki          http://localhost:3100    [RUNNING]
  ✓ AlertManager  http://localhost:9093    [RUNNING]

[APM] Starting application...
  ✓ my-app        http://localhost:8080    [RUNNING]

[APM] Hot reload enabled. Watching for changes...

2024-01-10 10:00:00 [INFO]  Server started on :8080
2024-01-10 10:00:01 [INFO]  Connected to Prometheus
2024-01-10 10:00:01 [INFO]  Tracing enabled with Jaeger
```

**Hot Reload Behavior:**
- Monitors Go files for changes
- Automatically rebuilds and restarts on changes
- Preserves monitoring stack state
- Shows rebuild progress and errors

### 3. `apm test` - Validate Configuration and Health

Test and validate the APM setup and component health.

```bash
apm test [flags] [component...]
```

**Flags:**
- `--connectivity` (bool): Test network connectivity only
- `--config-only` (bool): Validate configuration without testing
- `--timeout` (duration): Test timeout (default: 30s)
- `--retry` (int): Number of retries (default: 3)
- `--fix` (bool): Attempt to fix common issues

**Test Categories:**

1. **Configuration Validation**
   ```
   🔍 Validating configuration...
   ✓ apm.yaml syntax valid
   ✓ All required fields present
   ✓ Port conflicts checked
   ✓ Resource limits reasonable
   ```

2. **Component Health Checks**
   ```
   🏥 Testing component health...
   
   Prometheus:
   ✓ API accessible at http://localhost:9090
   ✓ Targets discovered: 5
   ✓ No unhealthy targets
   ✓ Storage space available: 45GB
   
   Grafana:
   ✓ API accessible at http://localhost:3000
   ✓ Admin login successful
   ✓ Data sources connected: 3/3
   ✓ Dashboards loaded: 12
   
   Jaeger:
   ✓ Query service responding
   ✓ Collector accepting traces
   ✓ Storage backend connected
   ⚠ Low trace volume detected
   ```

3. **Integration Tests**
   ```
   🔗 Testing integrations...
   ✓ Prometheus → Grafana connection
   ✓ Application → Prometheus metrics
   ✓ Application → Jaeger traces
   ✓ Promtail → Loki logs
   ✓ Prometheus → AlertManager alerts
   ```

4. **Performance Tests**
   ```
   ⚡ Performance metrics...
   ✓ Prometheus query latency: 12ms (p95)
   ✓ Grafana dashboard load: 450ms
   ✓ Trace ingestion rate: 1000/s
   ⚠ High memory usage in Loki: 2.1GB
   ```

**Error Output Example:**
```
❌ Test failed: Jaeger connection error

Error: Failed to connect to Jaeger at http://localhost:16686
Cause: dial tcp 127.0.0.1:16686: connect: connection refused

Suggested fixes:
1. Check if Jaeger is running: 'docker ps | grep jaeger'
2. Verify port 16686 is not in use: 'lsof -i :16686'
3. Try restarting Jaeger: 'apm run --stack-only'

Run 'apm test --fix' to attempt automatic resolution.
```

### 4. `apm dashboard` - Access Monitoring Interfaces

Quick access to all monitoring dashboards and tools.

```bash
apm dashboard [component] [flags]
```

**Components:**
- `grafana` - Open Grafana dashboards
- `prometheus` - Open Prometheus UI
- `jaeger` - Open Jaeger UI
- `alertmanager` - Open AlertManager UI
- `logs` - Open Loki/Grafana logs view

**Flags:**
- `--browser, -b` (string): Browser to use (default: system default)
- `--no-browser` (bool): Show URLs without opening browser
- `--port-forward` (bool): Enable Kubernetes port forwarding
- `--namespace, -n` (string): Kubernetes namespace
- `--list, -l` (bool): List all available dashboards

**Interactive Mode (no component specified):**
```
🎯 APM Dashboards

Select a dashboard to open:
  1. Grafana         - Metrics & Visualizations  [http://localhost:3000]
  2. Prometheus      - Metrics Explorer          [http://localhost:9090]
  3. Jaeger          - Distributed Tracing       [http://localhost:16686]
  4. AlertManager    - Alert Management          [http://localhost:9093]
  5. Application     - GoFiber Metrics           [http://localhost:3000/d/app]
  6. Kubernetes      - Cluster Overview          [http://localhost:3000/d/k8s]
  7. Logs            - Centralized Logging       [http://localhost:3000/d/logs]

Enter selection (1-7) or 'q' to quit: 
```

**Kubernetes Port-Forward Mode:**
```
apm dashboard grafana --port-forward -n monitoring

📡 Setting up port forwarding...
✓ Found Grafana pod: grafana-7d9c5b7b5-x4j2m
✓ Port forward established: localhost:3000 → grafana:3000

🌐 Opening http://localhost:3000 in browser...
✓ Dashboard opened

Press Ctrl+C to stop port forwarding...
```

## Configuration File Format (apm.yaml)

```yaml
# APM Configuration File
version: "1.0"

project:
  name: my-gofiber-app
  type: gofiber
  environment: local

# Application settings
application:
  port: 8080
  hot_reload: true
  build_command: "go build -o app cmd/main.go"
  run_command: "./app"
  watch_paths:
    - "**/*.go"
    - "configs/*.yaml"
  ignore_paths:
    - "vendor/"
    - "*.test.go"

# Monitoring components
components:
  prometheus:
    enabled: true
    port: 9090
    scrape_interval: 15s
    retention: 15d
    storage_size: 50GB
    
  grafana:
    enabled: true
    port: 3000
    admin_password: admin123
    anonymous_access: false
    provisioning:
      dashboards:
        - ./dashboards/*.json
      datasources:
        - name: Prometheus
          type: prometheus
          url: http://prometheus:9090
          
  loki:
    enabled: true
    port: 3100
    retention: 168h
    max_query_length: 5000
    
  jaeger:
    enabled: true
    port: 16686
    sampling_rate: 0.1
    storage: memory
    
  alertmanager:
    enabled: true
    port: 9093
    smtp:
      enabled: false
      host: smtp.gmail.com
      port: 587
    slack:
      enabled: true
      webhook_url: ${SLACK_WEBHOOK_URL}

# Instrumentation settings
instrumentation:
  metrics:
    enabled: true
    namespace: myapp
    subsystem: api
    
  tracing:
    enabled: true
    service_name: my-gofiber-app
    exporter: jaeger
    
  logging:
    enabled: true
    level: info
    format: json

# Deployment settings
deployment:
  type: docker-compose  # docker-compose | kubernetes
  compose_file: docker-compose.apm.yml
  
  kubernetes:
    namespace: monitoring
    helm_chart: ./charts/apm-stack
    values_file: ./values.yaml

# Dashboard shortcuts
dashboards:
  - name: "Application Overview"
    url: "/d/app-overview"
    component: grafana
  - name: "Trace Search"
    url: "/search"
    component: jaeger
  - name: "Active Alerts"
    url: "/#/alerts"
    component: alertmanager
```

## Error Handling

### Error Categories

1. **Configuration Errors**
   - Invalid YAML syntax
   - Missing required fields
   - Invalid values or types
   - File permission issues

2. **Runtime Errors**
   - Component start failures
   - Port conflicts
   - Resource exhaustion
   - Network connectivity

3. **Validation Errors**
   - Health check failures
   - Integration test failures
   - Performance degradation

### Error Message Format

```
Error: <brief description>
  
Details:
  Component: <component name>
  Operation: <what was being attempted>
  Cause: <root cause>
  
Context:
  Config file: ./apm.yaml
  Environment: local
  Time: 2024-01-10 10:00:00
  
Suggestions:
  1. <actionable suggestion>
  2. <alternative approach>
  
For more information, run 'apm doctor' or check logs at ~/.apm/logs/
```

### Exit Codes

- `0` - Success
- `1` - General error
- `2` - Configuration error
- `3` - Runtime error
- `4` - Validation error
- `5` - Network/connectivity error
- `126` - Permission denied
- `127` - Command not found

## Output Formatting

### Standard Output

- Use colors for status (green=success, yellow=warning, red=error)
- Use icons for visual clarity (✓, ✗, ⚠, 🚀, 📝, etc.)
- Indent nested information
- Show progress for long operations

### JSON Output

When `--json` flag is used:

```json
{
  "command": "test",
  "status": "success",
  "timestamp": "2024-01-10T10:00:00Z",
  "results": {
    "configuration": {
      "status": "valid",
      "checks": ["syntax", "required_fields", "ports", "resources"]
    },
    "components": {
      "prometheus": {
        "status": "healthy",
        "url": "http://localhost:9090",
        "metrics": {
          "targets": 5,
          "up": 5,
          "storage_available_gb": 45
        }
      }
    }
  }
}
```

### Log Levels

- **ERROR**: Component failures, invalid configuration
- **WARN**: Degraded performance, non-critical issues
- **INFO**: Status updates, normal operations
- **DEBUG**: Detailed execution flow, API calls

## Implementation Guidelines

### 1. Command Pattern

Use Cobra or similar CLI framework with command pattern:

```go
type Command struct {
    Name        string
    Description string
    Flags       []Flag
    Run         func(cmd *Command, args []string) error
}
```

### 2. Configuration Management

- Use Viper for configuration file handling
- Support environment variable overrides
- Validate configuration on load
- Provide sensible defaults

### 3. Interactive Prompts

- Use Survey or similar library for interactive prompts
- Provide keyboard navigation
- Support default values
- Allow non-interactive mode with flags

### 4. Error Handling

- Wrap errors with context
- Provide actionable error messages
- Log detailed errors to file
- Support error recovery where possible

### 5. Testing Support

- Unit tests for each command
- Integration tests for workflows
- Mock external dependencies
- Test error scenarios

## Future Enhancements

1. **Plugin System**
   - Custom commands
   - Additional monitoring tools
   - Third-party integrations

2. **Multi-cluster Support**
   - Context switching
   - Cross-cluster monitoring
   - Federated metrics

3. **Advanced Features**
   - Backup/restore configurations
   - Performance profiling
   - Cost analysis
   - Automated remediation

4. **CI/CD Integration**
   - GitHub Actions
   - GitLab CI
   - Jenkins plugins
   - ArgoCD hooks