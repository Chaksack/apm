---
layout: default
title: Quick Start Guide - APM
description: Get started with APM in 5 minutes
---

# Quick Start Guide

Get your GoFiber application monitored with APM in just 5 minutes!

## Prerequisites

- Go 1.21 or higher
- Docker (optional, for running the monitoring stack)
- Cloud CLI tools (optional, for cloud deployments)

## Step 1: Install APM CLI

```bash
# Install the APM CLI tool
go install github.com/chaksack/apm/cmd/apm@latest

# Verify installation
apm --version
```

## Step 2: Initialize Your Project

Navigate to your GoFiber project directory and run:

```bash
# Start the interactive setup wizard
apm init
```

The wizard will guide you through:

1. **Project Configuration**
   - Project name
   - Environment (development/staging/production)
   - Application entry point

2. **APM Tools Selection**
   - Prometheus (metrics)
   - Grafana (visualization)
   - Jaeger (distributed tracing)
   - Loki (log aggregation)
   - AlertManager (alerting)

3. **Notification Setup**
   - Slack webhook configuration
   - Email settings (coming soon)

This creates an `apm.yaml` configuration file in your project.

## Step 3: Add APM to Your Application

### Install the instrumentation package

```bash
go get github.com/chaksack/apm/pkg/instrumentation
```

### Update your main.go

```go
package main

import (
    "context"
    "log"
    
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/requestid"
    apm "github.com/chaksack/apm/pkg/instrumentation"
)

func main() {
    // Initialize APM instrumentation
    instr, err := apm.New(apm.LoadFromEnv())
    if err != nil {
        log.Fatal(err)
    }
    defer instr.Shutdown(context.Background())

    // Create Fiber app
    app := fiber.New()
    
    // Add middleware stack
    app.Use(requestid.New())
    app.Use(instr.FiberMiddleware())
    
    // Define routes
    app.Get("/", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{
            "message": "Hello, APM!",
            "request_id": c.Locals("requestid"),
        })
    })
    
    app.Get("/health", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{"status": "healthy"})
    })
    
    // Start server
    log.Fatal(app.Listen(":3000"))
}
```

## Step 4: Run with APM

### Development Mode (with hot reload)

```bash
# Run your application with APM instrumentation
apm run

# The APM tool will:
# - Set up environment variables
# - Inject APM agents
# - Enable hot reload
# - Display real-time logs
```

### Docker Compose (Full Stack)

```bash
# Start the complete monitoring stack
docker-compose up -d

# This starts:
# - Your application with APM
# - Prometheus
# - Grafana
# - Jaeger
# - Loki & Promtail
# - AlertManager
```

## Step 5: Access Monitoring Tools

```bash
# Open the interactive dashboard selector
apm dashboard
```

Or access directly:
- **Metrics**: http://localhost:9090 (Prometheus)
- **Dashboards**: http://localhost:3000 (Grafana, admin/admin)
- **Traces**: http://localhost:16686 (Jaeger)
- **Logs**: http://localhost:3000 (via Grafana)
- **Alerts**: http://localhost:9093 (AlertManager)

## Step 6: Deploy to Cloud (Optional)

### Deploy to AWS

```bash
# Interactive deployment wizard
apm deploy

# Select AWS and follow the prompts
# The tool will:
# - Build and push Docker image to ECR
# - Deploy to ECS or EKS
# - Configure CloudWatch integration
# - Set up APM instrumentation
```

### Deploy to Kubernetes

```bash
# Deploy to Kubernetes with sidecar injection
apm deploy kubernetes --namespace production

# This will:
# - Build your application image
# - Inject APM sidecars
# - Deploy to your cluster
# - Configure service discovery
```

## What's Next?

### Add Custom Metrics

```go
// Create custom business metrics
orderCounter := instr.Metrics.NewCounter(
    "orders_total",
    "Total orders processed",
    []string{"status"},
)

// Use in your handlers
orderCounter.WithLabelValues("completed").Inc()
```

### Add Structured Logging

```go
// Get request-scoped logger
logger := apm.GetLogger(c)

logger.Info("Processing order",
    zap.String("order_id", orderID),
    zap.Float64("amount", amount),
)
```

### Add Distributed Tracing

```go
// Start a span
ctx, span := tracer.Start(ctx, "process-payment")
defer span.End()

// Add attributes
span.SetAttributes(
    attribute.String("payment.method", "credit_card"),
    attribute.Float64("payment.amount", 99.99),
)
```

## Troubleshooting

### Common Issues

**Port conflicts**: If you see "address already in use", check the ports in `apm.yaml` and adjust as needed.

**Docker not running**: Ensure Docker Desktop is running before using `docker-compose`.

**Missing metrics**: Verify your application is using the APM middleware and the `/metrics` endpoint is accessible.

### Getting Help

- Check the [troubleshooting guide](./troubleshooting.md)
- Open an [issue on GitHub](https://github.com/chaksack/apm/issues)
- Join our [discussions](https://github.com/chaksack/apm/discussions)

## Example Applications

Explore complete examples:

- [Basic GoFiber App](https://github.com/chaksack/apm/tree/main/examples/gofiber-app)
- [Microservices Example](https://github.com/chaksack/apm/tree/main/examples/microservices)
- [E-commerce Demo](https://github.com/chaksack/apm/tree/main/examples/ecommerce)

---

[Back to Home](./index.md) | [Next: Configuration Guide →](./configuration.md)