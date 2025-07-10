# APM Tool Integration Architecture

## Overview

This document outlines the integration approach for various APM tools including Prometheus, Grafana, Jaeger, Loki, AlertManager, and others. The design provides a flexible, extensible architecture that supports both containerized and native installations with automatic detection, validation, and health monitoring.

## 1. Tool Detection and Validation Framework

### 1.1 Detection Strategy

```go
// Tool represents an APM tool with its configuration
type Tool struct {
    Name            string
    Type            ToolType
    Version         string
    InstallType     InstallType // Docker, Native, Kubernetes
    Endpoint        string
    HealthEndpoint  string
    Port            int
    Status          ToolStatus
}

// ToolDetector interface for detecting tool installations
type ToolDetector interface {
    Detect() (*Tool, error)
    Validate() error
    GetVersion() (string, error)
}
```

### 1.2 Detection Methods

1. **Container Detection**
   - Docker API inspection
   - Container labels and environment variables
   - Network connectivity tests

2. **Native Installation Detection**
   - Process inspection (ps/pgrep)
   - Port scanning
   - Configuration file locations
   - Binary path detection

3. **Kubernetes Detection**
   - Service discovery via K8s API
   - Label selectors
   - Namespace scanning
   - Endpoint slices

### 1.3 Validation Rules

```yaml
validation:
  prometheus:
    required_endpoints:
      - /api/v1/query
      - /api/v1/targets
      - /-/healthy
    min_version: "2.30.0"
    required_features:
      - remote_write
      - service_discovery
    
  grafana:
    required_endpoints:
      - /api/health
      - /api/datasources
    min_version: "8.0.0"
    required_plugins:
      - prometheus
      - loki
      - jaeger
```

## 2. Configuration Templates

### 2.1 Prometheus Configuration Template

```yaml
# prometheus-template.yml
global:
  scrape_interval: {{ .ScrapeInterval | default "15s" }}
  evaluation_interval: {{ .EvaluationInterval | default "15s" }}
  external_labels:
    cluster: {{ .ClusterName }}
    environment: {{ .Environment }}

# Alertmanager configuration
alerting:
  alertmanagers:
    - static_configs:
        - targets:
          {{- range .AlertManagerTargets }}
          - {{ . }}
          {{- end }}

# Rule files
rule_files:
  {{- range .RuleFiles }}
  - {{ . }}
  {{- end }}

# Scrape configurations
scrape_configs:
  # Default job for self-monitoring
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  # Dynamic service discovery
  {{- if .ServiceDiscovery.Enabled }}
  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names: {{ .ServiceDiscovery.Namespaces | toJson }}
  {{- end }}
  
  # Custom scrape configs
  {{- range .CustomScrapeConfigs }}
  - {{ . | toYaml | nindent 4 }}
  {{- end }}
```

### 2.2 Grafana Configuration Template

```ini
# grafana-template.ini
[server]
protocol = {{ .Protocol | default "http" }}
http_port = {{ .Port | default 3000 }}
root_url = {{ .RootURL }}

[database]
type = {{ .Database.Type | default "sqlite3" }}
{{- if eq .Database.Type "postgres" }}
host = {{ .Database.Host }}
name = {{ .Database.Name }}
user = {{ .Database.User }}
password = {{ .Database.Password }}
{{- end }}

[security]
admin_user = {{ .AdminUser | default "admin" }}
admin_password = {{ .AdminPassword }}
disable_initial_admin_creation = {{ .DisableInitialAdmin | default false }}

[auth.anonymous]
enabled = {{ .AnonymousAuth | default false }}

[alerting]
enabled = {{ .Alerting.Enabled | default true }}

[unified_alerting]
enabled = {{ .UnifiedAlerting | default true }}
```

### 2.3 Jaeger Configuration Template

```yaml
# jaeger-template.yml
service:
  name: jaeger
  
extensions:
  health_check:
    endpoint: 0.0.0.0:13133

receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:{{ .OTLPGRPCPort | default 4317 }}
      http:
        endpoint: 0.0.0.0:{{ .OTLPHTTPPort | default 4318 }}
  
  jaeger:
    protocols:
      grpc:
        endpoint: 0.0.0.0:{{ .JaegerGRPCPort | default 14250 }}
      thrift_compact:
        endpoint: 0.0.0.0:{{ .JaegerThriftPort | default 6831 }}

processors:
  batch:
    timeout: {{ .BatchTimeout | default "1s" }}
    send_batch_size: {{ .BatchSize | default 1024 }}
  
  memory_limiter:
    check_interval: 1s
    limit_mib: {{ .MemoryLimit | default 512 }}

exporters:
  jaeger:
    endpoint: {{ .StorageEndpoint }}
    tls:
      insecure: {{ .TLSInsecure | default true }}

service:
  extensions: [health_check]
  pipelines:
    traces:
      receivers: [otlp, jaeger]
      processors: [memory_limiter, batch]
      exporters: [jaeger]
```

### 2.4 Loki Configuration Template

```yaml
# loki-template.yml
auth_enabled: {{ .AuthEnabled | default false }}

server:
  http_listen_port: {{ .HTTPPort | default 3100 }}
  grpc_listen_port: {{ .GRPCPort | default 9096 }}

ingester:
  lifecycler:
    address: 127.0.0.1
    ring:
      kvstore:
        store: {{ .KVStore | default "inmemory" }}
      replication_factor: {{ .ReplicationFactor | default 1 }}
  chunk_idle_period: {{ .ChunkIdlePeriod | default "5m" }}
  chunk_retain_period: {{ .ChunkRetainPeriod | default "30s" }}

schema_config:
  configs:
    - from: {{ .SchemaStartDate | default "2020-05-15" }}
      store: {{ .Store | default "boltdb-shipper" }}
      object_store: {{ .ObjectStore | default "filesystem" }}
      schema: v11
      index:
        prefix: loki_index_
        period: {{ .IndexPeriod | default "24h" }}

storage_config:
  {{- if eq .Store "boltdb-shipper" }}
  boltdb_shipper:
    active_index_directory: {{ .DataDir }}/loki/index
    cache_location: {{ .DataDir }}/loki/index_cache
    shared_store: {{ .ObjectStore | default "filesystem" }}
  {{- end }}
  
  {{- if eq .ObjectStore "filesystem" }}
  filesystem:
    directory: {{ .DataDir }}/loki/chunks
  {{- end }}

limits_config:
  enforce_metric_name: false
  reject_old_samples: true
  reject_old_samples_max_age: {{ .MaxSampleAge | default "168h" }}
  max_query_series: {{ .MaxQuerySeries | default 5000 }}
```

### 2.5 AlertManager Configuration Template

```yaml
# alertmanager-template.yml
global:
  resolve_timeout: {{ .ResolveTimeout | default "5m" }}
  {{- if .SMTPConfig }}
  smtp_smarthost: {{ .SMTPConfig.Host }}:{{ .SMTPConfig.Port }}
  smtp_from: {{ .SMTPConfig.From }}
  smtp_auth_username: {{ .SMTPConfig.Username }}
  smtp_auth_password: {{ .SMTPConfig.Password }}
  {{- end }}

route:
  receiver: {{ .DefaultReceiver | default "default" }}
  group_by: {{ .GroupBy | default "[alertname, cluster, service]" }}
  group_wait: {{ .GroupWait | default "10s" }}
  group_interval: {{ .GroupInterval | default "10s" }}
  repeat_interval: {{ .RepeatInterval | default "1h" }}
  
  routes:
  {{- range .Routes }}
  - match:
      severity: {{ .Severity }}
    receiver: {{ .Receiver }}
    {{- if .Continue }}
    continue: true
    {{- end }}
  {{- end }}

receivers:
  - name: default
    
  {{- range .Receivers }}
  - name: {{ .Name }}
    {{- if .EmailConfigs }}
    email_configs:
    {{- range .EmailConfigs }}
    - to: {{ .To }}
      headers:
        Subject: '{{ .Subject }}'
    {{- end }}
    {{- end }}
    
    {{- if .SlackConfigs }}
    slack_configs:
    {{- range .SlackConfigs }}
    - api_url: {{ .APIURL }}
      channel: {{ .Channel }}
      title: '{{ .Title }}'
      text: '{{ .Text }}'
    {{- end }}
    {{- end }}
  {{- end }}
```

## 3. Health Check Endpoints and Methods

### 3.1 Health Check Interface

```go
// HealthChecker interface for tool health monitoring
type HealthChecker interface {
    Check(ctx context.Context) (*HealthStatus, error)
    GetMetrics() (*HealthMetrics, error)
}

// HealthStatus represents the health status of a tool
type HealthStatus struct {
    Status      string            `json:"status"` // healthy, degraded, unhealthy
    Version     string            `json:"version"`
    Uptime      time.Duration     `json:"uptime"`
    LastChecked time.Time         `json:"last_checked"`
    Details     map[string]string `json:"details"`
}

// HealthMetrics provides detailed metrics about tool health
type HealthMetrics struct {
    ResponseTime   time.Duration     `json:"response_time"`
    ErrorRate      float64           `json:"error_rate"`
    ResourceUsage  ResourceMetrics   `json:"resource_usage"`
    Availability   float64           `json:"availability"`
}
```

### 3.2 Tool-Specific Health Checks

```yaml
health_checks:
  prometheus:
    endpoints:
      - path: "/-/healthy"
        method: GET
        expected_status: 200
      - path: "/-/ready"
        method: GET
        expected_status: 200
    metrics:
      - path: "/api/v1/query"
        query: "up"
        expected_result: "1"
    
  grafana:
    endpoints:
      - path: "/api/health"
        method: GET
        expected_status: 200
        expected_body: '{"database": "ok"}'
    auth:
      type: "basic"
      credentials: "${GRAFANA_ADMIN_USER}:${GRAFANA_ADMIN_PASSWORD}"
    
  jaeger:
    endpoints:
      - path: "/"
        method: GET
        expected_status: 200
      - path: "/api/services"
        method: GET
        expected_status: 200
    
  loki:
    endpoints:
      - path: "/ready"
        method: GET
        expected_status: 200
      - path: "/metrics"
        method: GET
        expected_status: 200
    
  alertmanager:
    endpoints:
      - path: "/-/healthy"
        method: GET
        expected_status: 200
      - path: "/api/v2/status"
        method: GET
        expected_status: 200
```

## 4. Port Management and Conflict Resolution

### 4.1 Port Registry

```yaml
port_registry:
  prometheus:
    default: 9090
    alternatives: [9091, 9092, 9093]
    protocol: tcp
    
  grafana:
    default: 3000
    alternatives: [3001, 3002, 3003]
    protocol: tcp
    
  jaeger:
    ui:
      default: 16686
      alternatives: [16687, 16688]
    collector:
      grpc: 14250
      http: 14268
    agent:
      thrift_compact: 6831
      thrift_binary: 6832
    
  loki:
    http:
      default: 3100
      alternatives: [3101, 3102]
    grpc:
      default: 9096
      alternatives: [9097, 9098]
    
  alertmanager:
    default: 9093
    alternatives: [9094, 9095]
    cluster: 9094
```

### 4.2 Port Conflict Resolution

```go
// PortManager handles port allocation and conflict resolution
type PortManager struct {
    registry    map[string]PortConfig
    allocated   map[int]string
    mu          sync.RWMutex
}

// AllocatePort finds an available port for a tool
func (pm *PortManager) AllocatePort(toolName string) (int, error) {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    config, exists := pm.registry[toolName]
    if !exists {
        return 0, fmt.Errorf("unknown tool: %s", toolName)
    }
    
    // Try default port first
    if pm.isPortAvailable(config.Default) {
        pm.allocated[config.Default] = toolName
        return config.Default, nil
    }
    
    // Try alternative ports
    for _, port := range config.Alternatives {
        if pm.isPortAvailable(port) {
            pm.allocated[port] = toolName
            return port, nil
        }
    }
    
    // Find next available port in range
    return pm.findNextAvailablePort(config.Default)
}

// isPortAvailable checks if a port is available for use
func (pm *PortManager) isPortAvailable(port int) bool {
    // Check internal registry
    if _, allocated := pm.allocated[port]; allocated {
        return false
    }
    
    // Check if port is actually in use
    listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return false
    }
    listener.Close()
    return true
}
```

## 5. Docker/Container vs Native Installation Support

### 5.1 Installation Abstraction

```go
// Installer interface for different installation types
type Installer interface {
    Install(config ToolConfig) error
    Uninstall() error
    Upgrade(version string) error
    GetInstallationType() InstallType
}

// DockerInstaller handles Docker-based installations
type DockerInstaller struct {
    client      *docker.Client
    network     string
    volumeRoot  string
}

// NativeInstaller handles native binary installations
type NativeInstaller struct {
    binaryPath  string
    configPath  string
    dataPath    string
    systemd     bool
}

// KubernetesInstaller handles K8s deployments
type KubernetesInstaller struct {
    clientset   *kubernetes.Clientset
    namespace   string
    helmChart   string
}
```

### 5.2 Installation Configuration

```yaml
installation:
  docker:
    network: "apm-network"
    volume_root: "/var/lib/apm"
    restart_policy: "unless-stopped"
    resource_limits:
      memory: "512Mi"
      cpu: "0.5"
    
  native:
    install_root: "/opt/apm"
    config_root: "/etc/apm"
    data_root: "/var/lib/apm"
    user: "apm"
    group: "apm"
    systemd_enabled: true
    
  kubernetes:
    namespace: "apm-system"
    storage_class: "standard"
    ingress_enabled: true
    tls_enabled: true
```

### 5.3 Docker Compose Template

```yaml
version: '3.8'

networks:
  apm-network:
    driver: bridge

volumes:
  prometheus_data:
  grafana_data:
  loki_data:
  jaeger_data:

services:
  prometheus:
    image: prom/prometheus:{{ .PrometheusVersion }}
    container_name: apm-prometheus
    ports:
      - "{{ .PrometheusPort }}:9090"
    volumes:
      - ./configs/prometheus:/etc/prometheus
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/usr/share/prometheus/console_libraries'
      - '--web.console.templates=/usr/share/prometheus/consoles'
    networks:
      - apm-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:9090/-/healthy"]
      interval: 30s
      timeout: 10s
      retries: 3

  grafana:
    image: grafana/grafana:{{ .GrafanaVersion }}
    container_name: apm-grafana
    ports:
      - "{{ .GrafanaPort }}:3000"
    volumes:
      - ./configs/grafana/provisioning:/etc/grafana/provisioning
      - grafana_data:/var/lib/grafana
    environment:
      - GF_SECURITY_ADMIN_PASSWORD={{ .GrafanaAdminPassword }}
      - GF_USERS_ALLOW_SIGN_UP=false
    networks:
      - apm-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost:3000/api/health || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## 6. Tool Abstraction Layer

### 6.1 Tool Interface

```go
// APMTool interface for all monitoring tools
type APMTool interface {
    // Lifecycle management
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Restart(ctx context.Context) error
    
    // Configuration
    GetConfig() ToolConfig
    UpdateConfig(config ToolConfig) error
    ValidateConfig(config ToolConfig) error
    
    // Health and status
    HealthCheck(ctx context.Context) (*HealthStatus, error)
    GetMetrics() (*ToolMetrics, error)
    GetStatus() ToolStatus
    
    // Integration
    GetEndpoints() map[string]string
    GetAPIClient() interface{}
}

// ToolFactory creates tool instances
type ToolFactory struct {
    detectors   map[ToolType]ToolDetector
    installers  map[InstallType]Installer
    tools       map[string]APMTool
}

// CreateTool creates a new tool instance
func (tf *ToolFactory) CreateTool(toolType ToolType, config ToolConfig) (APMTool, error) {
    switch toolType {
    case ToolTypePrometheus:
        return NewPrometheusTool(config)
    case ToolTypeGrafana:
        return NewGrafanaTool(config)
    case ToolTypeJaeger:
        return NewJaegerTool(config)
    case ToolTypeLoki:
        return NewLokiTool(config)
    case ToolTypeAlertManager:
        return NewAlertManagerTool(config)
    default:
        return nil, fmt.Errorf("unsupported tool type: %s", toolType)
    }
}
```

### 6.2 Tool Registry

```go
// ToolRegistry manages all APM tools
type ToolRegistry struct {
    tools       map[string]APMTool
    mu          sync.RWMutex
    factory     *ToolFactory
    portManager *PortManager
}

// Register adds a new tool to the registry
func (tr *ToolRegistry) Register(name string, tool APMTool) error {
    tr.mu.Lock()
    defer tr.mu.Unlock()
    
    if _, exists := tr.tools[name]; exists {
        return fmt.Errorf("tool already registered: %s", name)
    }
    
    // Allocate ports
    config := tool.GetConfig()
    port, err := tr.portManager.AllocatePort(config.Type)
    if err != nil {
        return fmt.Errorf("failed to allocate port: %w", err)
    }
    config.Port = port
    
    // Update configuration
    if err := tool.UpdateConfig(config); err != nil {
        return fmt.Errorf("failed to update config: %w", err)
    }
    
    tr.tools[name] = tool
    return nil
}

// GetTool retrieves a tool by name
func (tr *ToolRegistry) GetTool(name string) (APMTool, bool) {
    tr.mu.RLock()
    defer tr.mu.RUnlock()
    
    tool, exists := tr.tools[name]
    return tool, exists
}
```

### 6.3 Plugin System for New Tools

```go
// ToolPlugin interface for extending with new tools
type ToolPlugin interface {
    // Metadata
    Name() string
    Version() string
    Description() string
    
    // Factory method
    CreateTool(config map[string]interface{}) (APMTool, error)
    
    // Configuration schema
    GetConfigSchema() json.RawMessage
    
    // Default configuration
    GetDefaultConfig() map[string]interface{}
}

// PluginManager manages tool plugins
type PluginManager struct {
    plugins map[string]ToolPlugin
    loader  PluginLoader
}

// LoadPlugin loads a new tool plugin
func (pm *PluginManager) LoadPlugin(path string) error {
    plugin, err := pm.loader.Load(path)
    if err != nil {
        return fmt.Errorf("failed to load plugin: %w", err)
    }
    
    pm.plugins[plugin.Name()] = plugin
    return nil
}
```

## 7. Integration Example

### 7.1 Complete Integration Flow

```go
// APMIntegrator orchestrates tool integration
type APMIntegrator struct {
    registry    *ToolRegistry
    factory     *ToolFactory
    portManager *PortManager
    config      *IntegrationConfig
}

// IntegrateTools sets up all APM tools
func (ai *APMIntegrator) IntegrateTools(ctx context.Context) error {
    // 1. Detect existing installations
    detected, err := ai.detectExistingTools()
    if err != nil {
        return fmt.Errorf("detection failed: %w", err)
    }
    
    // 2. Validate detected tools
    for _, tool := range detected {
        if err := tool.ValidateConfig(tool.GetConfig()); err != nil {
            log.Warnf("Validation failed for %s: %v", tool.GetConfig().Name, err)
            continue
        }
        
        // 3. Register validated tools
        if err := ai.registry.Register(tool.GetConfig().Name, tool); err != nil {
            return fmt.Errorf("registration failed: %w", err)
        }
    }
    
    // 4. Install missing tools
    missing := ai.identifyMissingTools(detected)
    for _, toolType := range missing {
        config := ai.config.GetToolConfig(toolType)
        
        // Allocate port
        port, err := ai.portManager.AllocatePort(string(toolType))
        if err != nil {
            return fmt.Errorf("port allocation failed: %w", err)
        }
        config.Port = port
        
        // Create and start tool
        tool, err := ai.factory.CreateTool(toolType, config)
        if err != nil {
            return fmt.Errorf("tool creation failed: %w", err)
        }
        
        if err := tool.Start(ctx); err != nil {
            return fmt.Errorf("tool start failed: %w", err)
        }
        
        // Register new tool
        if err := ai.registry.Register(config.Name, tool); err != nil {
            return fmt.Errorf("registration failed: %w", err)
        }
    }
    
    // 5. Configure integrations
    return ai.configureIntegrations()
}

// configureIntegrations sets up tool interconnections
func (ai *APMIntegrator) configureIntegrations() error {
    // Configure Prometheus to scrape all tools
    prometheus, exists := ai.registry.GetTool("prometheus")
    if exists {
        scrapeConfigs := ai.generateScrapeConfigs()
        config := prometheus.GetConfig()
        config.CustomScrapeConfigs = scrapeConfigs
        if err := prometheus.UpdateConfig(config); err != nil {
            return fmt.Errorf("failed to update Prometheus config: %w", err)
        }
    }
    
    // Configure Grafana datasources
    grafana, exists := ai.registry.GetTool("grafana")
    if exists {
        datasources := ai.generateDatasources()
        if err := ai.configureGrafanaDatasources(grafana, datasources); err != nil {
            return fmt.Errorf("failed to configure Grafana datasources: %w", err)
        }
    }
    
    return nil
}
```

## 8. Monitoring and Maintenance

### 8.1 Tool Monitoring

```yaml
monitoring:
  health_check_interval: 30s
  metric_collection_interval: 15s
  
  alerts:
    - name: ToolDown
      condition: up == 0
      duration: 5m
      severity: critical
      
    - name: ToolUnhealthy
      condition: health_status != "healthy"
      duration: 10m
      severity: warning
      
    - name: HighResourceUsage
      condition: resource_usage > 0.8
      duration: 15m
      severity: warning
```

### 8.2 Auto-Recovery

```go
// AutoRecovery handles automatic tool recovery
type AutoRecovery struct {
    registry    *ToolRegistry
    maxRetries  int
    retryDelay  time.Duration
}

// RecoverTool attempts to recover a failed tool
func (ar *AutoRecovery) RecoverTool(toolName string) error {
    tool, exists := ar.registry.GetTool(toolName)
    if !exists {
        return fmt.Errorf("tool not found: %s", toolName)
    }
    
    for i := 0; i < ar.maxRetries; i++ {
        // Attempt restart
        if err := tool.Restart(context.Background()); err != nil {
            log.Warnf("Restart attempt %d failed: %v", i+1, err)
            time.Sleep(ar.retryDelay)
            continue
        }
        
        // Verify health
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        
        health, err := tool.HealthCheck(ctx)
        if err == nil && health.Status == "healthy" {
            log.Infof("Tool %s recovered successfully", toolName)
            return nil
        }
    }
    
    return fmt.Errorf("failed to recover tool after %d attempts", ar.maxRetries)
}
```

## Summary

This architecture provides:

1. **Flexible Detection**: Automatic discovery of existing tool installations
2. **Template-Based Configuration**: Easy configuration management for all tools
3. **Comprehensive Health Checks**: Proactive monitoring of tool health
4. **Smart Port Management**: Automatic port allocation and conflict resolution
5. **Multi-Environment Support**: Docker, native, and Kubernetes installations
6. **Extensible Design**: Plugin system for adding new tools
7. **Auto-Recovery**: Automatic recovery from tool failures
8. **Unified Interface**: Consistent API for all tool interactions

The design ensures that the APM solution can adapt to various environments and requirements while maintaining simplicity and reliability.