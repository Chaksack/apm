---
layout: default
title: APM - Application Performance Monitoring for GoFiber
description: Comprehensive APM solution for GoFiber applications with metrics, tracing, logging, and cloud deployment capabilities
---

# APM - Application Performance Monitoring for GoFiber

[![Go Reference](https://pkg.go.dev/badge/github.com/chaksack/apm.svg)](https://pkg.go.dev/github.com/chaksack/apm)
[![Go Report Card](https://goreportcard.com/badge/github.com/chaksack/apm)](https://goreportcard.com/report/github.com/chaksack/apm)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

## 🚀 Overview

APM is a comprehensive Application Performance Monitoring solution specifically designed for [GoFiber](https://gofiber.io/) applications. It provides enterprise-grade observability including metrics collection, distributed tracing, structured logging, and health checks, along with a powerful CLI tool for seamless cloud deployments.

### Key Features

- 📊 **Prometheus Metrics** - Pre-configured HTTP metrics and custom metric builders
- 🔍 **Distributed Tracing** - OpenTelemetry integration with Jaeger support
- 📝 **Structured Logging** - Zap-based logging with request correlation
- 💊 **Health Checks** - Kubernetes-ready health endpoints
- 🚢 **Multi-Cloud Deployment** - Deploy to AWS, Azure, or GCP with one command
- 🔐 **Enterprise Security** - Cross-account roles, MFA, and encrypted credentials
- 🎮 **Interactive CLI** - Wizard-based setup and deployment

## 📚 Table of Contents

- [Quick Start](#quick-start)
- [Installation](#installation)
- [CLI Tool](#cli-tool)
- [Cloud Deployments](#cloud-deployments)
- [AWS Cross-Account Support](#aws-cross-account-support)
- [Package Usage](#package-usage)
- [Configuration](#configuration)
- [Monitoring Stack](#monitoring-stack)
- [Examples](#examples)
- [Documentation](#documentation)

## 🏃 Quick Start

### Install the CLI Tool

```bash
# Install from source
go install github.com/chaksack/apm/cmd/apm@latest

# Or clone and build
git clone https://github.com/chaksack/apm.git
cd apm
go build -o apm ./cmd/apm/
sudo mv apm /usr/local/bin/
```

### Initialize Your Project

```bash
# Interactive setup wizard
apm init

# Run with hot reload
apm run

# Deploy to cloud
apm deploy
```

## 🛠️ Installation

### As a Go Package

```bash
go get github.com/chaksack/apm/pkg/instrumentation
```

### Basic Integration

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    apm "github.com/chaksack/apm/pkg/instrumentation"
)

func main() {
    // Initialize instrumentation
    instr, err := apm.New(apm.DefaultConfig())
    if err != nil {
        panic(err)
    }
    defer instr.Shutdown(context.Background())

    // Create Fiber app with APM
    app := fiber.New()
    app.Use(instr.FiberMiddleware())

    // Your routes
    app.Get("/", func(c *fiber.Ctx) error {
        return c.SendString("Hello, World!")
    })

    app.Listen(":3000")
}
```

## 🎮 CLI Tool

The APM CLI provides a comprehensive command-line interface for managing your APM stack.

### Commands Overview

| Command | Description |
|---------|-------------|
| `apm init` | Interactive configuration wizard |
| `apm run` | Run application with hot reload |
| `apm test` | Validate configuration |
| `apm dashboard` | Access monitoring UIs |
| `apm deploy` | Deploy to cloud platforms |

### Interactive Configuration

```bash
apm init
```

The wizard guides you through:
- 📋 Project configuration
- 🔧 APM tool selection (Prometheus, Grafana, Jaeger, Loki)
- 💬 Notification setup (Slack, Email)
- 💾 Configuration saved to `apm.yaml`

### Hot Reload Development

```bash
# Run with automatic restart on changes
apm run

# Run specific command
apm run "go run main.go"

# Disable hot reload
apm run --no-reload
```

### Monitoring Dashboard

```bash
apm dashboard
```

Access all your monitoring tools:
- Prometheus (http://localhost:9090)
- Grafana (http://localhost:3000)
- Jaeger (http://localhost:16686)
- Loki (http://localhost:3100)
- AlertManager (http://localhost:9093)

## ☁️ Cloud Deployments

### Multi-Cloud Support

The APM tool supports deployment to all major cloud providers with automatic APM instrumentation.

#### Supported Platforms

| Provider | Services | Features |
|----------|----------|----------|
| **AWS** | ECS, EKS, ECR | Cross-account roles, MFA, S3 storage |
| **Azure** | ACI, AKS, ACR | Service principals, managed identity |
| **GCP** | Cloud Run, GKE, GCR | Service accounts, Workload Identity |
| **Docker** | Local/Remote | Multi-stage builds, registry support |
| **Kubernetes** | Any cluster | Sidecar injection, Helm charts |

### Deployment Examples

```bash
# Interactive deployment wizard
apm deploy

# Deploy to specific platform
apm deploy --provider aws --region us-west-2
apm deploy --provider azure --resource-group mygroup
apm deploy --provider gcp --project myproject

# Deploy to Kubernetes
apm deploy kubernetes --namespace production

# Docker deployment
apm deploy docker --registry ghcr.io/myorg
```

### Cloud Provider Features

#### 🚀 AWS Integration
- ECR registry with automatic authentication
- EKS deployment with kubeconfig generation
- Cross-account role assumption
- S3 configuration storage
- CloudWatch monitoring integration

#### 🔷 Azure Integration
- ACR registry management
- AKS deployment with Azure Monitor
- Service principal authentication
- Key Vault integration
- Application Insights support

#### 🟡 Google Cloud Integration
- GCR/Artifact Registry support
- GKE deployment with Cloud Monitoring
- Service account management
- Cloud Storage for configurations
- Cloud Trace integration

## 🔐 AWS Cross-Account Support

### Enterprise-Grade Multi-Account Management

APM provides comprehensive support for complex AWS multi-account environments.

### Key Features

#### Security-First Design
- ✅ MFA (Multi-Factor Authentication) support
- ✅ External ID validation for partners
- ✅ Trust policy validation
- ✅ AES-256-GCM credential encryption

#### Advanced Role Management
- ✅ Role chaining (up to 5 hops)
- ✅ Automatic session refresh
- ✅ Cross-region role switching
- ✅ Concurrent session management

#### Configuration Management
- ✅ YAML-based multi-account setup
- ✅ Environment-specific settings
- ✅ Role hierarchy support
- ✅ Template validation

### Usage Examples

```bash
# Deploy with MFA
apm deploy --provider aws --account production --mfa

# Use role chain
apm deploy --provider aws --role-chain "dev-role,staging-role,prod-role"

# External ID for partners
apm deploy --provider aws --role arn:aws:iam::123456789012:role/PartnerRole \
  --external-id "unique-external-id"
```

### Configuration Example

```yaml
aws:
  accounts:
    dev:
      account_id: "111111111111"
      region: "us-east-1"
      roles:
        - name: "dev-deploy-role"
          arn: "arn:aws:iam::111111111111:role/DevDeployRole"
    
    production:
      account_id: "222222222222"
      region: "us-west-2"
      roles:
        - name: "prod-deploy-role"
          arn: "arn:aws:iam::222222222222:role/ProdDeployRole"
          mfa_required: true
          external_id: "prod-external-id"
  
  role_chains:
    - name: "dev-to-prod"
      steps:
        - role: "arn:aws:iam::111111111111:role/DevRole"
        - role: "arn:aws:iam::222222222222:role/ProdRole"
          external_id: "chain-external-id"
```

## 📦 Package Usage

### Core Components

#### Metrics Collection

```go
// Access metrics collector
metrics := instr.Metrics

// Create custom metrics
orderCounter := metrics.NewCounter(
    "orders_total",
    "Total number of orders",
    []string{"status", "payment_method"},
)

// Use in handlers
orderCounter.WithLabelValues("completed", "credit_card").Inc()
```

#### Distributed Tracing

```go
// Initialize tracer
tracerConfig := instrumentation.TracerConfig{
    ServiceName:    "my-service",
    ExporterType:   "otlp",
    Endpoint:       "localhost:4317",
    SampleRate:     0.1,
}

tracerProvider, cleanup, err := instrumentation.InitTracer(ctx, tracerConfig)
defer cleanup()

// Use in handlers
ctx, span := tracer.Start(ctx, "process-order")
defer span.End()

span.SetAttributes(
    attribute.String("order.id", orderID),
    attribute.Float64("order.total", orderTotal),
)
```

#### Structured Logging

```go
// Get request-scoped logger
logger := instrumentation.GetLogger(c)

logger.Info("Order processed",
    zap.String("order_id", orderID),
    zap.Float64("total", orderTotal),
    zap.String("request_id", c.Locals("requestid").(string)),
)
```

#### Health Checks

```go
app.Get("/health/ready", func(c *fiber.Ctx) error {
    checks := fiber.Map{
        "database": checkDatabase(),
        "cache": checkRedis(),
        "external_api": checkExternalAPI(),
    }
    
    return c.JSON(fiber.Map{
        "status": allHealthy,
        "checks": checks,
    })
})
```

## ⚙️ Configuration

### Environment Variables

```bash
# Service configuration
SERVICE_NAME=my-service
ENVIRONMENT=production
VERSION=1.0.0

# Metrics
METRICS_ENABLED=true
METRICS_NAMESPACE=myapp
METRICS_PATH=/metrics

# Logging
LOG_LEVEL=info
LOG_ENCODING=json

# Tracing
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
OTEL_TRACES_SAMPLER_ARG=0.1

# APM endpoints
APM_PROMETHEUS_ENDPOINT=http://localhost:9090
APM_GRAFANA_ENDPOINT=http://localhost:3000
APM_JAEGER_ENDPOINT=http://localhost:16686
```

### Configuration File (apm.yaml)

```yaml
version: "1.0"
project:
  name: "my-app"
  environment: "production"

apm:
  prometheus:
    enabled: true
    port: 9090
    scrape_interval: "15s"
  
  grafana:
    enabled: true
    port: 3000
    datasources:
      - name: "Prometheus"
        type: "prometheus"
        url: "http://localhost:9090"
  
  jaeger:
    enabled: true
    ui_port: 16686
  
  alertmanager:
    enabled: true
    port: 9093
    receivers:
      - name: "slack"
        webhook_url: "YOUR_WEBHOOK_URL"

notifications:
  slack:
    enabled: true
    webhook_url: "YOUR_WEBHOOK_URL"
    channel: "#alerts"

deployment:
  docker:
    registry: "ghcr.io/myorg"
  kubernetes:
    namespace: "production"
  cloud:
    provider: "aws"
    region: "us-west-2"
```

## 📊 Monitoring Stack

### Complete Observability

APM integrates with industry-standard monitoring tools to provide comprehensive observability.

#### Prometheus
- Automatic HTTP metrics collection
- Custom business metrics support
- Pre-configured alerting rules
- Service discovery integration

#### Grafana
- Pre-built dashboards for GoFiber apps
- Custom dashboard support
- Multi-datasource integration
- Alert visualization

#### Jaeger
- Distributed trace collection
- Service dependency mapping
- Performance bottleneck identification
- Error tracking across services

#### Loki
- Structured log aggregation
- LogQL query support
- Log-to-trace correlation
- Automatic log shipping with Promtail

#### AlertManager
- Intelligent alert routing
- Multiple notification channels
- Alert grouping and silencing
- Custom notification templates

### Key Metrics

| Metric Type | Examples |
|-------------|----------|
| **Application** | Request rate, error rate, response time (p50/p95/p99) |
| **Business** | Order count, payment success rate, cart size |
| **System** | CPU usage, memory consumption, disk I/O |
| **Container** | Resource limits, restart count, network traffic |

## 💡 Examples

### E-commerce Application

```go
// Custom business metrics
orderCounter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "orders_total",
        Help: "Total number of orders",
    },
    []string{"status", "payment_method"},
)

app.Post("/checkout", func(c *fiber.Ctx) error {
    ctx := instrumentation.FiberContextWithCorrelation(c)
    ctx, span := tracer.Start(ctx, "process-checkout")
    defer span.End()
    
    // Process order...
    
    orderCounter.WithLabelValues("completed", "credit_card").Inc()
    
    logger := instrumentation.GetLogger(c)
    logger.Info("Order completed",
        zap.String("order_id", orderID),
        zap.Float64("total", orderTotal),
    )
    
    return c.JSON(order)
})
```

### Microservice with Circuit Breaker

```go
type PaymentService struct {
    breaker *gobreaker.CircuitBreaker
    tracer  trace.Tracer
}

func (s *PaymentService) ProcessPayment(ctx context.Context, amount float64) error {
    _, span := s.tracer.Start(ctx, "payment.process")
    defer span.End()
    
    result, err := s.breaker.Execute(func() (interface{}, error) {
        return s.callPaymentAPI(ctx, amount)
    })
    
    if err != nil {
        span.RecordError(err)
        return err
    }
    
    span.SetAttributes(attribute.Float64("payment.amount", amount))
    return nil
}
```

## 📖 Documentation

### Guides

- [Quick Start Guide](./quickstart.md)
- [Configuration Reference](./configuration.md)
- [Deployment Guide](./deployment.md)
- [Monitoring Guide](./monitoring.md)
- [Troubleshooting](./troubleshooting.md)

### Cloud Provider Guides

- [AWS Integration Guide](./cloud-provider-integration.md#aws)
- [Cross-Account Role Assumption](./cross-account-role-assumption.md)
- [Azure Integration Guide](./azure-integration.md)
- [GCP Integration Guide](./gcp-integration-guide.md)

### API Reference

- [Go Package Documentation](https://pkg.go.dev/github.com/chaksack/apm)
- [CLI Reference](./cli-reference.md)
- [Configuration Schema](./configuration-schema.md)

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](https://github.com/chaksack/apm/blob/main/CONTRIBUTING.md) for details.

## 👥 Authors

- **Andrew Chakdahah** - [chakdahah@gmail.com](mailto:chakdahah@gmail.com)
- **Yaw Boateng Kessie** - [ybkess@gmail.com](mailto:ybkess@gmail.com)

## 📄 License

Licensed under the Apache License, Version 2.0. See [LICENSE](https://github.com/chaksack/apm/blob/main/LICENSE) for details.

## 💬 Support

- [GitHub Issues](https://github.com/chaksack/apm/issues)
- [Discussions](https://github.com/chaksack/apm/discussions)
- [Stack Overflow](https://stackoverflow.com/questions/tagged/apm-gofiber)

---

<div align="center">
  <p>Built with ❤️ for the GoFiber community</p>
  <p>
    <a href="https://github.com/chaksack/apm">GitHub</a> •
    <a href="https://pkg.go.dev/github.com/chaksack/apm">Go Docs</a> •
    <a href="https://github.com/chaksack/apm/releases">Releases</a>
  </p>
</div>