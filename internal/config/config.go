package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the APM solution
type Config struct {
	// Server configuration
	Server ServerConfig `mapstructure:"server"`

	// Monitoring endpoints
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
	Grafana    GrafanaConfig    `mapstructure:"grafana"`
	Loki       LokiConfig       `mapstructure:"loki"`
	Jaeger     JaegerConfig     `mapstructure:"jaeger"`

	// Alert configurations
	AlertManager AlertManagerConfig `mapstructure:"alertmanager"`

	// Notification configurations
	Notifications NotificationConfig `mapstructure:"notifications"`

	// Kubernetes configurations
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`

	// Service discovery configurations
	ServiceDiscovery ServiceDiscoveryConfig `mapstructure:"service_discovery"`
}

// ServerConfig holds GoFiber server configuration
type ServerConfig struct {
	Port         string `mapstructure:"port"`
	ReadTimeout  string `mapstructure:"read_timeout"`
	WriteTimeout string `mapstructure:"write_timeout"`
	Prefork      bool   `mapstructure:"prefork"`
}

// PrometheusConfig holds Prometheus-specific configuration
type PrometheusConfig struct {
	Endpoint           string `mapstructure:"endpoint"`
	ScrapeInterval     string `mapstructure:"scrape_interval"`
	EvaluationInterval string `mapstructure:"evaluation_interval"`
}

// GrafanaConfig holds Grafana-specific configuration
type GrafanaConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	APIKey   string `mapstructure:"api_key"`
	OrgID    int    `mapstructure:"org_id"`
}

// LokiConfig holds Loki-specific configuration
type LokiConfig struct {
	Endpoint     string `mapstructure:"endpoint"`
	PushEndpoint string `mapstructure:"push_endpoint"`
	QueryTimeout string `mapstructure:"query_timeout"`
}

// JaegerConfig holds Jaeger-specific configuration
type JaegerConfig struct {
	Endpoint          string `mapstructure:"endpoint"`
	CollectorEndpoint string `mapstructure:"collector_endpoint"`
	AgentHost         string `mapstructure:"agent_host"`
	AgentPort         int    `mapstructure:"agent_port"`
}

// AlertManagerConfig holds AlertManager-specific configuration
type AlertManagerConfig struct {
	Endpoint       string `mapstructure:"endpoint"`
	WebhookURL     string `mapstructure:"webhook_url"`
	ResolveTimeout string `mapstructure:"resolve_timeout"`
}

// NotificationConfig holds notification settings
type NotificationConfig struct {
	Email EmailConfig `mapstructure:"email"`
	Slack SlackConfig `mapstructure:"slack"`
}

// EmailConfig holds SMTP settings for email notifications
type EmailConfig struct {
	SMTPHost       string `mapstructure:"smtp_host"`
	SMTPPort       int    `mapstructure:"smtp_port"`
	SMTPUsername   string `mapstructure:"smtp_username"`
	SMTPPassword   string `mapstructure:"smtp_password"`
	SMTPFrom       string `mapstructure:"smtp_from"`
	SMTPTLSEnabled bool   `mapstructure:"smtp_tls_enabled"`
}

// SlackConfig holds Slack notification settings
type SlackConfig struct {
	WebhookURL string `mapstructure:"webhook_url"`
	Channel    string `mapstructure:"channel"`
	Username   string `mapstructure:"username"`
}

// KubernetesConfig holds Kubernetes-specific configuration
type KubernetesConfig struct {
	Namespace      string   `mapstructure:"namespace"`
	InCluster      bool     `mapstructure:"in_cluster"`
	ConfigPath     string   `mapstructure:"config_path"`
	LabelSelectors []string `mapstructure:"label_selectors"`
}

// ServiceDiscoveryConfig holds service discovery settings
type ServiceDiscoveryConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	RefreshInterval  string   `mapstructure:"refresh_interval"`
	Namespaces       []string `mapstructure:"namespaces"`
	ServiceSelectors []string `mapstructure:"service_selectors"`
	PodSelectors     []string `mapstructure:"pod_selectors"`
}

// LoadConfig reads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// Enable environment variables
	v.SetEnvPrefix("APM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Explicitly bind environment variables for critical settings
	v.BindEnv("prometheus.endpoint", "APM_PROMETHEUS_ENDPOINT")
	v.BindEnv("grafana.api_key", "APM_GRAFANA_API_KEY")
	v.BindEnv("kubernetes.namespace", "APM_KUBERNETES_NAMESPACE")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		// Check if it's a file not found error
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// For explicitly provided config files, we should still allow falling back to defaults
			// if the file doesn't exist, but report other errors
			if configPath != "" && !strings.Contains(err.Error(), "no such file or directory") {
				return nil, fmt.Errorf("error reading config file: %w", err)
			}
		}
		// Config file not found; use defaults and environment variables
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", ":8080")
	v.SetDefault("server.read_timeout", "10s")
	v.SetDefault("server.write_timeout", "10s")
	v.SetDefault("server.prefork", false)

	// Prometheus defaults
	v.SetDefault("prometheus.endpoint", "http://localhost:9090")
	v.SetDefault("prometheus.scrape_interval", "15s")
	v.SetDefault("prometheus.evaluation_interval", "15s")

	// Grafana defaults
	v.SetDefault("grafana.endpoint", "http://localhost:3000")
	v.SetDefault("grafana.org_id", 1)

	// Loki defaults
	v.SetDefault("loki.endpoint", "http://localhost:3100")
	v.SetDefault("loki.push_endpoint", "http://localhost:3100/loki/api/v1/push")
	v.SetDefault("loki.query_timeout", "30s")

	// Jaeger defaults
	v.SetDefault("jaeger.endpoint", "http://localhost:16686")
	v.SetDefault("jaeger.collector_endpoint", "http://localhost:14268/api/traces")
	v.SetDefault("jaeger.agent_host", "localhost")
	v.SetDefault("jaeger.agent_port", 6831)

	// AlertManager defaults
	v.SetDefault("alertmanager.endpoint", "http://localhost:9093")
	v.SetDefault("alertmanager.resolve_timeout", "5m")

	// Email defaults
	v.SetDefault("notifications.email.smtp_port", 587)
	v.SetDefault("notifications.email.smtp_tls_enabled", true)

	// Kubernetes defaults
	v.SetDefault("kubernetes.namespace", "default")
	v.SetDefault("kubernetes.in_cluster", false)

	// Service discovery defaults
	v.SetDefault("service_discovery.enabled", true)
	v.SetDefault("service_discovery.refresh_interval", "30s")
}
