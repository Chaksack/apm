package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	configContent := `
prometheus:
  endpoint: "http://test-prometheus:9090"
  scrape_interval: "30s"

grafana:
  endpoint: "http://test-grafana:3000"
  api_key: "test-api-key"

kubernetes:
  namespace: "test-namespace"
  in_cluster: true

notifications:
  slack:
    webhook_url: "https://hooks.slack.com/test"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test loading configuration from file
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded values
	if cfg.Prometheus.Endpoint != "http://test-prometheus:9090" {
		t.Errorf("Expected Prometheus endpoint 'http://test-prometheus:9090', got '%s'", cfg.Prometheus.Endpoint)
	}

	if cfg.Prometheus.ScrapeInterval != "30s" {
		t.Errorf("Expected scrape interval '30s', got '%s'", cfg.Prometheus.ScrapeInterval)
	}

	if cfg.Grafana.APIKey != "test-api-key" {
		t.Errorf("Expected Grafana API key 'test-api-key', got '%s'", cfg.Grafana.APIKey)
	}

	if cfg.Kubernetes.Namespace != "test-namespace" {
		t.Errorf("Expected Kubernetes namespace 'test-namespace', got '%s'", cfg.Kubernetes.Namespace)
	}

	if !cfg.Kubernetes.InCluster {
		t.Error("Expected InCluster to be true")
	}

	if cfg.Notifications.Slack.WebhookURL != "https://hooks.slack.com/test" {
		t.Errorf("Expected Slack webhook URL 'https://hooks.slack.com/test', got '%s'", cfg.Notifications.Slack.WebhookURL)
	}
}

func TestLoadConfigWithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("APM_PROMETHEUS_ENDPOINT", "http://env-prometheus:9090")
	os.Setenv("APM_GRAFANA_API_KEY", "env-api-key")
	os.Setenv("APM_KUBERNETES_NAMESPACE", "env-namespace")
	defer func() {
		os.Unsetenv("APM_PROMETHEUS_ENDPOINT")
		os.Unsetenv("APM_GRAFANA_API_KEY")
		os.Unsetenv("APM_KUBERNETES_NAMESPACE")
	}()

	// Create a temporary config file for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	// Create a minimal config file
	configContent := `
prometheus:
  endpoint: "http://default-prometheus:9090"
grafana:
  endpoint: "http://default-grafana:3000"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Load config with environment variables
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify environment variables override defaults
	if cfg.Prometheus.Endpoint != "http://env-prometheus:9090" {
		t.Errorf("Expected Prometheus endpoint from env 'http://env-prometheus:9090', got '%s'", cfg.Prometheus.Endpoint)
	}

	if cfg.Grafana.APIKey != "env-api-key" {
		t.Errorf("Expected Grafana API key from env 'env-api-key', got '%s'", cfg.Grafana.APIKey)
	}

	if cfg.Kubernetes.Namespace != "env-namespace" {
		t.Errorf("Expected Kubernetes namespace from env 'env-namespace', got '%s'", cfg.Kubernetes.Namespace)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// Create a temporary config file for testing with minimal content
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	// Create a minimal config file with only a few settings
	configContent := `
server:
  port: ":9090"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Load config with minimal file (should use defaults for unspecified values)
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify default values for unspecified fields
	if cfg.Prometheus.Endpoint != "http://localhost:9090" {
		t.Errorf("Expected default Prometheus endpoint 'http://localhost:9090', got '%s'", cfg.Prometheus.Endpoint)
	}

	if cfg.Prometheus.ScrapeInterval != "15s" {
		t.Errorf("Expected default scrape interval '15s', got '%s'", cfg.Prometheus.ScrapeInterval)
	}

	if cfg.Jaeger.AgentPort != 6831 {
		t.Errorf("Expected default Jaeger agent port 6831, got %d", cfg.Jaeger.AgentPort)
	}

	if !cfg.Notifications.Email.SMTPTLSEnabled {
		t.Error("Expected SMTP TLS to be enabled by default")
	}

	if !cfg.ServiceDiscovery.Enabled {
		t.Error("Expected service discovery to be enabled by default")
	}

	// Verify the specified value was loaded correctly
	if cfg.Server.Port != ":9090" {
		t.Errorf("Expected server port ':9090', got '%s'", cfg.Server.Port)
	}
}
