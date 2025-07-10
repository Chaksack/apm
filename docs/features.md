---
layout: default
title: Features - APM
description: Comprehensive feature overview of APM for GoFiber
---

# APM Features

Comprehensive overview of all APM features and capabilities.

## 🎯 Core Features

### 📊 Metrics Collection

#### Automatic Metrics
- **HTTP Metrics**: Request count, duration, size automatically collected
- **System Metrics**: CPU, memory, disk I/O via Node Exporter
- **Container Metrics**: Resource usage and limits via cAdvisor
- **Runtime Metrics**: Goroutines, GC stats, memory allocations

#### Custom Metrics
```go
// Define business metrics
orderMetric := metrics.NewCounter(
    "orders_total",
    "Total orders processed",
    []string{"status", "region"},
)

// Track in your code
orderMetric.WithLabelValues("completed", "us-west").Inc()
```

#### Pre-built Dashboards
- Application Overview Dashboard
- Performance Metrics Dashboard
- Business KPIs Dashboard
- SLO/Error Budget Dashboard

### 🔍 Distributed Tracing

#### OpenTelemetry Integration
- W3C Trace Context propagation
- Automatic span creation for HTTP requests
- Support for custom spans and attributes
- Trace-to-logs correlation

#### Jaeger Features
- Service dependency mapping
- Critical path analysis
- Performance bottleneck identification
- Trace comparison and diff

#### Trace Sampling
```go
// Configurable sampling strategies
config := instrumentation.TracerConfig{
    SampleRate: 0.1,  // Sample 10% of traces
    // Or use adaptive sampling
    SamplerType: "adaptive",
}
```

### 📝 Structured Logging

#### Features
- JSON structured output
- Request correlation IDs
- Automatic request/response logging
- Error stack traces with context
- Log levels: Debug, Info, Warn, Error, Fatal

#### Integration
```go
logger := instrumentation.GetLogger(c)
logger.Info("Order processed",
    zap.String("order_id", orderID),
    zap.Duration("processing_time", duration),
    zap.Error(err),
)
```

#### Log Aggregation
- Loki integration with Promtail
- Automatic log shipping
- LogQL query support
- Grafana log visualization

### 💊 Health Checks

#### Kubernetes-Ready Endpoints
- `/health/live` - Liveness probe
- `/health/ready` - Readiness probe
- `/health/startup` - Startup probe

#### Dependency Checks
```go
checks := map[string]func() error{
    "database": checkPostgres,
    "redis": checkRedis,
    "external_api": checkAPI,
}
```

## 🚀 Deployment Features

### 🐳 Docker Support

#### Automatic APM Agent Injection
- Build-time agent embedding
- Runtime sidecar injection
- Multi-stage build optimization
- Language-specific agent support

#### Registry Support
- Docker Hub
- Amazon ECR
- Azure Container Registry
- Google Container Registry
- Private registries

### ☸️ Kubernetes Integration

#### Manifest Management
- Automatic sidecar injection
- ConfigMap generation
- Secret management
- Resource optimization

#### Deployment Strategies
- Rolling updates
- Blue-green deployments
- Canary deployments
- Rollback support

### ☁️ Multi-Cloud Support

#### AWS Features
- **ECS/Fargate**: Containerized deployments
- **EKS**: Kubernetes with IAM integration
- **ECR**: Container registry with scanning
- **CloudWatch**: Metrics and log integration
- **Cross-Account**: Role assumption and MFA
- **S3**: Configuration storage

#### Azure Features
- **Container Instances**: Simple container hosting
- **AKS**: Managed Kubernetes
- **ACR**: Container registry
- **Monitor**: Metrics and alerts
- **Key Vault**: Secret management
- **Application Insights**: APM integration

#### GCP Features
- **Cloud Run**: Serverless containers
- **GKE**: Kubernetes with Workload Identity
- **Container Registry**: Image storage
- **Cloud Monitoring**: Metrics and traces
- **Cloud Storage**: Configuration backup

## 🔐 Security Features

### Authentication & Authorization
- Cloud provider CLI integration
- Service account management
- IAM role support
- API key management

### Credential Management
- AES-256-GCM encryption
- Secure credential storage
- Automatic rotation support
- Multi-factor authentication

### Cross-Account Access (AWS)
- Role assumption chains
- External ID validation
- MFA enforcement
- Session management

## 🎮 CLI Features

### Interactive Wizards
- Project initialization
- Deployment configuration
- Tool selection
- Credential setup

### Development Tools
- Hot reload support
- Real-time log streaming
- Environment management
- Build optimization

### Deployment Automation
- One-command deployments
- Multi-environment support
- Rollback capabilities
- Status monitoring

## 📈 Monitoring Stack

### Prometheus
- Metric collection and storage
- Service discovery
- Alert rule evaluation
- PromQL query language

### Grafana
- Pre-built dashboards
- Custom visualization
- Multi-datasource support
- Alert visualization

### Jaeger
- Distributed trace collection
- Service topology
- Performance analysis
- Error tracking

### Loki
- Log aggregation
- Label-based indexing
- LogQL queries
- Grafana integration

### AlertManager
- Alert routing and grouping
- Multiple notification channels
- Silence management
- Alert templates

## 🔧 Configuration

### Environment-Based
```bash
SERVICE_NAME=my-app
ENVIRONMENT=production
METRICS_ENABLED=true
LOG_LEVEL=info
```

### File-Based (YAML)
```yaml
apm:
  prometheus:
    enabled: true
    retention: 30d
  grafana:
    enabled: true
    anonymous_access: false
```

### Dynamic Configuration
- Hot reload support
- Runtime updates
- A/B testing
- Feature flags

## 📊 Observability

### SLI/SLO Support
- Error budget tracking
- Multi-window burn rates
- SLO dashboards
- Automated alerts

### Business Metrics
- Custom KPI tracking
- Revenue metrics
- User engagement
- Performance indicators

### Compliance
- Audit logging
- Data retention policies
- GDPR compliance
- Security scanning

## 🛠️ Developer Experience

### Easy Integration
```go
// Just 3 lines to add APM
instr, _ := apm.New(apm.DefaultConfig())
defer instr.Shutdown(context.Background())
app.Use(instr.FiberMiddleware())
```

### Testing Support
- Mock collectors
- Test helpers
- Integration tests
- Benchmark tools

### Documentation
- Inline code docs
- API reference
- Usage examples
- Best practices

## 🌟 Advanced Features

### Circuit Breaker Integration
```go
breaker := gobreaker.NewCircuitBreaker(settings)
result, err := breaker.Execute(func() (interface{}, error) {
    return callExternalAPI()
})
```

### Service Mesh Support
- Istio integration
- Envoy sidecar metrics
- mTLS support
- Traffic management

### Cost Optimization
- Resource recommendations
- Unused resource detection
- Cost allocation tags
- Budget alerts

### Performance Optimization
- Automatic caching
- Connection pooling
- Batch processing
- Lazy loading

## 📱 Integrations

### Notification Channels
- Slack webhooks
- Email (SMTP)
- PagerDuty
- Custom webhooks

### CI/CD Platforms
- GitHub Actions
- GitLab CI
- Jenkins
- CircleCI

### Cloud Services
- AWS services
- Azure services
- GCP services
- Kubernetes operators

### Development Tools
- VS Code extension
- GoLand integration
- Terminal UI
- Web dashboard

---

[Back to Home](./index.md) | [Get Started →](./quickstart.md)