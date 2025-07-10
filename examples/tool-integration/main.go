package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/chaksack/apm/pkg/tools"
)

func main() {
	fmt.Println("APM Tool Integration Example")
	fmt.Println("============================")

	// 1. Detect all installed tools
	fmt.Println("\n1. Detecting installed APM tools...")
	ctx := context.Background()
	detectedTools, err := tools.DetectAllTools(ctx)
	if err != nil {
		log.Printf("Error detecting tools: %v", err)
	}

	fmt.Printf("Found %d tools:\n", len(detectedTools))
	for _, tool := range detectedTools {
		fmt.Printf("  - %s on port %d (status: %s)\n", tool.Name, tool.Port, tool.Status)
	}

	// 2. Check health of detected tools
	fmt.Println("\n2. Checking health of detected tools...")
	healthFactory := tools.NewHealthCheckerFactory()
	
	for _, tool := range detectedTools {
		checker, err := healthFactory.CreateHealthChecker(tool)
		if err != nil {
			fmt.Printf("  - %s: Failed to create health checker: %v\n", tool.Name, err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		health, err := checker.Check(ctx)
		cancel()

		if err != nil {
			fmt.Printf("  - %s: Health check failed: %v\n", tool.Name, err)
		} else {
			fmt.Printf("  - %s: %s (version: %s)\n", tool.Name, health.Status, health.Version)
		}
	}

	// 3. Port management demonstration
	fmt.Println("\n3. Port management demonstration...")
	portManager := tools.NewPortManager()

	// Allocate ports for tools
	toolTypes := []tools.ToolType{
		tools.ToolTypePrometheus,
		tools.ToolTypeGrafana,
		tools.ToolTypeJaeger,
	}

	for _, toolType := range toolTypes {
		port, err := portManager.AllocatePort(toolType)
		if err != nil {
			fmt.Printf("  - Failed to allocate port for %s: %v\n", toolType, err)
		} else {
			fmt.Printf("  - Allocated port %d for %s\n", port, toolType)
		}
	}

	// Show allocated ports
	fmt.Println("\n  All allocated ports:")
	for port, tool := range portManager.GetAllocatedPorts() {
		fmt.Printf("    - Port %d: %s\n", port, tool)
	}

	// 4. Configuration template rendering
	fmt.Println("\n4. Configuration template rendering...")
	renderer, err := tools.NewConfigTemplateRenderer()
	if err != nil {
		log.Fatal("Failed to create template renderer:", err)
	}

	// Render Prometheus configuration
	promConfig := map[string]interface{}{
		"ScrapeInterval":      "10s",
		"ClusterName":         "my-cluster",
		"Environment":         "production",
		"AlertManagerTargets": []string{"localhost:9093"},
	}

	config, err := renderer.Render(tools.ToolTypePrometheus, promConfig)
	if err != nil {
		fmt.Printf("  Failed to render Prometheus config: %v\n", err)
	} else {
		fmt.Println("  Prometheus configuration (first 200 chars):")
		if len(config) > 200 {
			fmt.Printf("    %s...\n", config[:200])
		} else {
			fmt.Printf("    %s\n", config)
		}
	}

	// 5. Test the API endpoints
	fmt.Println("\n5. Testing API endpoints (requires APM service running on port 8080)...")
	testEndpoints()
}

func testEndpoints() {
	client := &http.Client{Timeout: 5 * time.Second}
	baseURL := "http://localhost:8080"

	endpoints := []string{
		"/tools",
		"/tools/detect",
		"/tools/port-registry",
		"/tools/prometheus/health",
	}

	for _, endpoint := range endpoints {
		url := baseURL + endpoint
		resp, err := client.Get(url)
		if err != nil {
			fmt.Printf("  - GET %s: Failed - %v\n", endpoint, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
				fmt.Printf("  - GET %s: OK (received %d keys)\n", endpoint, len(result))
			} else {
				fmt.Printf("  - GET %s: OK (non-JSON response)\n", endpoint)
			}
		} else {
			fmt.Printf("  - GET %s: Status %d\n", endpoint, resp.StatusCode)
		}
	}
}