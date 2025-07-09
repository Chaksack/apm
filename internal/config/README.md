# APM Configuration Package

This package provides configuration management for the GoFiber-based APM solution using Viper.

## Usage

### Loading Configuration

```go
import "github.com/yourusername/apm/internal/config"

// Load from default location (./configs/config.yaml or ./config.yaml)
cfg, err := config.LoadConfig("")
if err != nil {
    log.Fatal(err)
}

// Load from specific file
cfg, err := config.LoadConfig("/path/to/config.yaml")
```

### Environment Variables

All configuration values can be overridden using environment variables with the `APM_` prefix:

- `APM_PROMETHEUS_ENDPOINT` - Override Prometheus endpoint
- `APM_GRAFANA_API_KEY` - Set Grafana API key
- `APM_NOTIFICATIONS_SLACK_WEBHOOK_URL` - Set Slack webhook URL
- `APM_KUBERNETES_NAMESPACE` - Override Kubernetes namespace

The environment variable names follow the pattern: `APM_<SECTION>_<KEY>` where dots in the configuration path are replaced with underscores.

### Configuration Structure

The configuration is organized into the following sections:

1. **GoFiber Application**
   - Server port and host configuration
   - Middleware settings (cors, rate limiting, etc.)
   - Performance tuning options

2. **Monitoring Endpoints**
   - Prometheus: Metrics collection
   - Grafana: Visualization
   - Loki: Log aggregation
   - Jaeger: Distributed tracing

3. **Alert Management**
   - AlertManager: Alert routing and management

4. **Notifications**
   - Email: SMTP configuration for email alerts
   - Slack: Webhook configuration for Slack notifications

5. **Kubernetes**
   - Namespace and cluster configuration
   - Label selectors for resource discovery

6. **Service Discovery**
   - Automatic discovery of services and pods
   - Configurable refresh intervals and selectors

### Example Configuration

See `configs/config.yaml` for a complete example configuration file.

### Security Notes

- Sensitive values like API keys, passwords, and webhook URLs should be set via environment variables
- Do not commit actual credentials to version control
- Use secret management solutions in production environments