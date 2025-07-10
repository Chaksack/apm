package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chaksack/apm/pkg/cloud"
)

func main() {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Initialize cloud manager
	credentialPath := os.ExpandEnv("$HOME/.apm/credentials")
	manager, err := cloud.NewCloudManager(credentialPath)
	if err != nil {
		log.Fatalf("Failed to create cloud manager: %v", err)
	}

	// Step 1: Detect and validate cloud CLIs
	fmt.Println("=== Cloud CLI Detection ===")
	cliStatuses := cloud.DetectAllCLIs(ctx)
	for provider, status := range cliStatuses {
		fmt.Printf("\n%s CLI:\n", provider)
		if status.Installed {
			fmt.Printf("  ✓ Installed: v%s at %s\n", status.Version, status.Path)
			if !status.IsSupported {
				fmt.Printf("  ⚠ Warning: Version %s is below minimum %s\n", 
					status.Version, status.MinVersion)
			}
		} else {
			fmt.Printf("  ✗ Not installed\n")
			detector, _ := cloud.NewDetectorFactory().CreateDetector(provider)
			if detector != nil {
				fmt.Printf("  Installation instructions:\n%s\n", 
					detector.GetInstallInstructions())
			}
		}
	}

	// Step 2: Validate authentication for available providers
	fmt.Println("\n=== Authentication Validation ===")
	validationResults := manager.ValidateEnvironment(ctx)
	for provider, result := range validationResults {
		fmt.Printf("\n%s:\n", provider)
		if result.Valid {
			fmt.Printf("  ✓ Valid authentication\n")
			for key, value := range result.Details {
				fmt.Printf("  - %s: %s\n", key, value)
			}
		} else {
			fmt.Printf("  ✗ Invalid: %v\n", result.Errors)
			if len(result.Warnings) > 0 {
				fmt.Printf("  Warnings: %v\n", result.Warnings)
			}
		}
	}

	// Step 3: List available providers
	fmt.Println("\n=== Available Providers ===")
	availableProviders := manager.DetectAvailableProviders(ctx)
	if len(availableProviders) == 0 {
		fmt.Println("No cloud providers are properly configured")
		return
	}
	for _, provider := range availableProviders {
		fmt.Printf("- %s\n", provider)
	}

	// Step 4: List container registries
	fmt.Println("\n=== Container Registries ===")
	allRegistries, err := manager.ListAllRegistries(ctx)
	if err != nil {
		log.Printf("Error listing registries: %v", err)
	} else {
		for provider, registries := range allRegistries {
			fmt.Printf("\n%s Registries:\n", provider)
			if len(registries) == 0 {
				fmt.Println("  No registries found")
			}
			for _, registry := range registries {
				fmt.Printf("  - %s (%s) in %s\n", 
					registry.Name, registry.Type, registry.Region)
				fmt.Printf("    URL: %s\n", registry.URL)
			}
		}
	}

	// Step 5: List Kubernetes clusters
	fmt.Println("\n=== Kubernetes Clusters ===")
	allClusters, err := manager.ListAllClusters(ctx)
	if err != nil {
		log.Printf("Error listing clusters: %v", err)
	} else {
		for provider, clusters := range allClusters {
			fmt.Printf("\n%s Clusters:\n", provider)
			if len(clusters) == 0 {
				fmt.Println("  No clusters found")
			}
			for _, cluster := range clusters {
				fmt.Printf("  - %s (%s %s)\n", 
					cluster.Name, cluster.Type, cluster.Version)
				fmt.Printf("    Region: %s, Status: %s, Nodes: %d\n",
					cluster.Region, cluster.Status, cluster.NodeCount)
			}
		}
	}

	// Step 6: Demonstrate multi-cloud operations
	fmt.Println("\n=== Multi-Cloud Operations ===")
	multiOps := cloud.NewMultiCloudOperations(manager)

	// Example: Find a specific cluster (if any exist)
	if len(allClusters) > 0 {
		for _, clusters := range allClusters {
			if len(clusters) > 0 {
				clusterName := clusters[0].Name
				foundCluster, foundProvider, err := multiOps.FindCluster(ctx, clusterName)
				if err == nil {
					fmt.Printf("Found cluster '%s' in %s\n", 
						foundCluster.Name, foundProvider)
				}
				break
			}
		}
	}

	// Step 7: Demonstrate credential management (example only)
	fmt.Println("\n=== Credential Management Example ===")
	
	// Example of storing credentials (DO NOT use real credentials)
	exampleCreds := &cloud.Credentials{
		Provider:   cloud.ProviderAWS,
		AuthMethod: cloud.AuthMethodCLI,
		Profile:    "example",
		Region:     "us-east-1",
		Properties: map[string]string{
			"note": "This is an example credential",
		},
	}

	if err := manager.StoreCredentials(exampleCreds); err != nil {
		fmt.Printf("Failed to store example credentials: %v\n", err)
	} else {
		fmt.Println("Example credentials stored successfully")
		
		// Retrieve credentials
		retrieved, err := manager.GetCredentials(cloud.ProviderAWS, "example")
		if err == nil {
			fmt.Printf("Retrieved credentials for profile: %s\n", retrieved.Profile)
		}
	}

	// Step 8: Show platform compatibility
	fmt.Println("\n=== Platform Compatibility ===")
	for _, provider := range []cloud.Provider{
		cloud.ProviderAWS, cloud.ProviderAzure, cloud.ProviderGCP,
	} {
		compat := cloud.GetPlatformCompatibility(provider)
		if compat != nil {
			fmt.Printf("\n%s on %s/%s:\n", provider, compat.OS, compat.Arch)
			fmt.Printf("  CLI Command: %s\n", compat.CLICommand)
			fmt.Printf("  Config Locations: %v\n", compat.ConfigLocations)
			fmt.Printf("  Environment Variables: %v\n", compat.EnvVars)
		}
	}

	fmt.Println("\n=== Cloud Integration Demo Complete ===")
}

// Helper function to demonstrate registry authentication
func authenticateToRegistry(ctx context.Context, manager *cloud.CloudManager, 
	provider cloud.Provider, registryName string) error {
	
	p, err := manager.GetProvider(provider)
	if err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}

	registry, err := p.GetRegistry(ctx, registryName)
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}

	fmt.Printf("Authenticating to %s registry %s...\n", provider, registry.Name)
	if err := p.AuthenticateRegistry(ctx, registry); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Printf("Successfully authenticated to %s\n", registry.URL)
	return nil
}

// Helper function to get kubeconfig for a cluster
func getClusterKubeconfig(ctx context.Context, manager *cloud.CloudManager,
	provider cloud.Provider, clusterName string) error {
	
	p, err := manager.GetProvider(provider)
	if err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}

	cluster, err := p.GetCluster(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	fmt.Printf("Getting kubeconfig for %s cluster %s...\n", provider, cluster.Name)
	kubeconfig, err := p.GetKubeconfig(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Save to file
	filename := fmt.Sprintf("kubeconfig-%s-%s.yaml", provider, cluster.Name)
	if err := os.WriteFile(filename, kubeconfig, 0600); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	fmt.Printf("Kubeconfig saved to %s\n", filename)
	return nil
}