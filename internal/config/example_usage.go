package config

import (
	"fmt"
	"log"
)

// ExampleUsage demonstrates how to use the configuration
func ExampleUsage() {
	// Load configuration from default location
	cfg, err := LoadConfig("")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Access configuration values
	fmt.Printf("Prometheus endpoint: %s\n", cfg.Prometheus.Endpoint)
	fmt.Printf("Grafana endpoint: %s\n", cfg.Grafana.Endpoint)
	fmt.Printf("Kubernetes namespace: %s\n", cfg.Kubernetes.Namespace)

	// Check if service discovery is enabled
	if cfg.ServiceDiscovery.Enabled {
		fmt.Printf("Service discovery enabled with refresh interval: %s\n",
			cfg.ServiceDiscovery.RefreshInterval)
	}

	// Access notification settings
	if cfg.Notifications.Slack.WebhookURL != "" {
		fmt.Println("Slack notifications are configured")
	}

	// Load configuration from specific file
	customCfg, err := LoadConfig("/path/to/custom/config.yaml")
	if err != nil {
		log.Printf("Failed to load custom configuration: %v", err)
	} else {
		fmt.Printf("Loaded custom config successfully\n")
		_ = customCfg // Use customCfg as needed
	}
}
