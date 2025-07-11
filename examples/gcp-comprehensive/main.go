package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/yourusername/apm/pkg/cloud"
)

func main() {
	ctx := context.Background()

	// Create GCP provider
	config := &cloud.ProviderConfig{
		Provider:      cloud.ProviderGCP,
		DefaultRegion: "us-central1",
		EnableCache:   true,
	}

	provider, err := cloud.NewGCPProvider(config)
	if err != nil {
		log.Fatalf("Failed to create GCP provider: %v", err)
	}

	// Validate gcloud CLI
	fmt.Println("=== Validating gcloud CLI ===")
	if err := provider.ValidateGCloudCLI(); err != nil {
		log.Printf("Warning: gcloud CLI validation failed: %v", err)
		// Continue anyway for demonstration
	}

	// Detect and validate CLI
	fmt.Println("\n=== Detecting CLI ===")
	cliStatus, err := provider.DetectCLI()
	if err != nil {
		log.Fatalf("Failed to detect CLI: %v", err)
	}
	fmt.Printf("CLI Status: Installed=%v, Version=%s, Path=%s\n", 
		cliStatus.Installed, cliStatus.Version, cliStatus.Path)

	// Validate authentication
	fmt.Println("\n=== Validating Authentication ===")
	if err := provider.ValidateAuth(ctx); err != nil {
		log.Printf("Authentication validation failed: %v", err)
		log.Println("Please run 'gcloud auth login' to authenticate")
		return
	}

	// Get credentials
	fmt.Println("\n=== Getting Credentials ===")
	credentials, err := provider.GetCredentials()
	if err != nil {
		log.Fatalf("Failed to get credentials: %v", err)
	}
	fmt.Printf("Provider: %s, Method: %s, Account: %s\n", 
		credentials.Provider, credentials.AuthMethod, credentials.Account)

	// Get advanced operations
	fmt.Println("\n=== Getting Advanced Operations ===")
	advOps := provider.GetAdvancedOperations()

	// Resource Manager operations
	fmt.Println("\n=== Resource Manager Operations ===")
	resourceManager := advOps.GetResourceManager()
	
	// List projects
	projects, err := resourceManager.ListProjects(ctx)
	if err != nil {
		log.Printf("Failed to list projects: %v", err)
	} else {
		fmt.Printf("Found %d projects:\n", len(projects))
		for i, project := range projects {
			if i < 3 { // Show only first 3 for brevity
				fmt.Printf("  - %s (%s)\n", project.Name, project.ProjectID)
			}
		}
		if len(projects) > 3 {
			fmt.Printf("  ... and %d more\n", len(projects)-3)
		}
	}

	// Get current project
	currentProject, err := resourceManager.GetCurrentProject(ctx)
	if err != nil {
		log.Printf("Failed to get current project: %v", err)
	} else {
		fmt.Printf("Current project: %s\n", currentProject)
	}

	// Service Account operations
	fmt.Println("\n=== Service Account Operations ===")
	serviceAccountManager := advOps.GetServiceAccountManager()
	
	// List service accounts
	serviceAccounts, err := serviceAccountManager.ListServiceAccounts(ctx)
	if err != nil {
		log.Printf("Failed to list service accounts: %v", err)
	} else {
		fmt.Printf("Found %d service accounts:\n", len(serviceAccounts))
		for i, sa := range serviceAccounts {
			if i < 3 { // Show only first 3 for brevity
				fmt.Printf("  - %s (%s)\n", sa.DisplayName, sa.Email)
			}
		}
		if len(serviceAccounts) > 3 {
			fmt.Printf("  ... and %d more\n", len(serviceAccounts)-3)
		}
	}

	// Authentication Manager operations
	fmt.Println("\n=== Authentication Manager Operations ===")
	authManager := advOps.GetAuthenticationManager()
	
	// List active accounts
	accounts, err := authManager.ListActiveAccounts(ctx)
	if err != nil {
		log.Printf("Failed to list active accounts: %v", err)
	} else {
		fmt.Printf("Active accounts: %v\n", accounts)
	}

	// Get access token (first few characters only)
	token, err := authManager.GetAccessToken(ctx)
	if err != nil {
		log.Printf("Failed to get access token: %v", err)
	} else {
		if len(token) > 20 {
			fmt.Printf("Access token: %s...\n", token[:20])
		} else {
			fmt.Printf("Access token: %s\n", token)
		}
	}

	// Registry operations
	fmt.Println("\n=== Registry Operations ===")
	registries, err := provider.ListRegistries(ctx)
	if err != nil {
		log.Printf("Failed to list registries: %v", err)
	} else {
		fmt.Printf("Found %d registries:\n", len(registries))
		for _, registry := range registries {
			fmt.Printf("  - %s (%s) in %s\n", registry.Name, registry.Type, registry.Region)
		}
	}

	// Cluster operations
	fmt.Println("\n=== Cluster Operations ===")
	clusters, err := provider.ListClusters(ctx)
	if err != nil {
		log.Printf("Failed to list clusters: %v", err)
	} else {
		fmt.Printf("Found %d GKE clusters:\n", len(clusters))
		for _, cluster := range clusters {
			fmt.Printf("  - %s (%s) in %s, Status: %s, Nodes: %d\n", 
				cluster.Name, cluster.Version, cluster.Region, cluster.Status, cluster.NodeCount)
		}
	}

	// Region operations
	fmt.Println("\n=== Region Operations ===")
	regions, err := provider.ListRegions(ctx)
	if err != nil {
		log.Printf("Failed to list regions: %v", err)
	} else {
		fmt.Printf("Found %d regions (showing first 5):\n", len(regions))
		for i, region := range regions {
			if i < 5 {
				fmt.Printf("  - %s\n", region)
			}
		}
		if len(regions) > 5 {
			fmt.Printf("  ... and %d more\n", len(regions)-5)
		}
	}

	currentRegion := provider.GetCurrentRegion()
	fmt.Printf("Current region: %s\n", currentRegion)

	// Storage operations
	fmt.Println("\n=== Storage Operations ===")
	storageManager := advOps.GetStorageManager()
	
	buckets, err := storageManager.ListStorageBuckets(ctx)
	if err != nil {
		log.Printf("Failed to list storage buckets: %v", err)
	} else {
		fmt.Printf("Found %d storage buckets:\n", len(buckets))
		for i, bucket := range buckets {
			if i < 3 { // Show only first 3 for brevity
				fmt.Printf("  - %s in %s (%s)\n", bucket.Name, bucket.Location, bucket.StorageClass)
			}
		}
		if len(buckets) > 3 {
			fmt.Printf("  ... and %d more\n", len(buckets)-3)
		}
	}

	// Monitoring operations
	fmt.Println("\n=== Monitoring Operations ===")
	monitoringManager := advOps.GetMonitoringManager()
	
	// Enable monitoring APIs (this might take some time)
	fmt.Println("Ensuring Cloud Monitoring API is enabled...")
	if err := monitoringManager.EnableMonitoringAPI(ctx); err != nil {
		log.Printf("Failed to enable monitoring API: %v", err)
	} else {
		fmt.Println("Cloud Monitoring API is enabled")
	}

	fmt.Println("Ensuring Cloud Trace API is enabled...")
	if err := monitoringManager.EnableCloudTraceAPI(ctx); err != nil {
		log.Printf("Failed to enable trace API: %v", err)
	} else {
		fmt.Println("Cloud Trace API is enabled")
	}

	// APM Integration example
	fmt.Println("\n=== APM Integration Setup ===")
	projectID := provider.GetProjectID()
	if projectID != "" {
		apmConfig := cloud.APMIntegrationConfig{
			ProjectID:            projectID,
			Region:               "us-central1",
			ServiceAccountID:     "apm-service-account",
			CreateServiceAccount: false, // Set to true to actually create
			SetupMonitoring:      true,
			CreateStorageBucket:  false, // Set to true to actually create
		}

		fmt.Printf("APM Integration configuration for project %s:\n", projectID)
		fmt.Printf("  - Region: %s\n", apmConfig.Region)
		fmt.Printf("  - Service Account ID: %s\n", apmConfig.ServiceAccountID)
		fmt.Printf("  - Create Service Account: %v\n", apmConfig.CreateServiceAccount)
		fmt.Printf("  - Setup Monitoring: %v\n", apmConfig.SetupMonitoring)
		fmt.Printf("  - Create Storage Bucket: %v\n", apmConfig.CreateStorageBucket)

		// Uncomment to actually set up APM integration
		// if err := advOps.SetupAPMIntegration(ctx, apmConfig); err != nil {
		//     log.Printf("Failed to setup APM integration: %v", err)
		// } else {
		//     fmt.Println("APM integration setup completed successfully")
		// }
	}

	fmt.Println("\n=== GCP Integration Demo Completed ===")
	fmt.Println("This example demonstrated comprehensive GCP CLI integration including:")
	fmt.Println("✓ CLI detection and validation")
	fmt.Println("✓ Authentication management")
	fmt.Println("✓ Project and resource management")
	fmt.Println("✓ Service account operations")
	fmt.Println("✓ Container registry support (GCR/Artifact Registry)")
	fmt.Println("✓ GKE cluster management")
	fmt.Println("✓ Cloud Storage integration")
	fmt.Println("✓ Cloud Monitoring and Trace APIs")
	fmt.Println("✓ Complete APM integration setup")
}

// Example usage scenarios:

// 1. Authenticate with service account:
func authenticateWithServiceAccount(ctx context.Context, provider *cloud.GCPProvider, keyFile string) error {
	authManager := provider.GetAdvancedOperations().GetAuthenticationManager()
	return authManager.AuthenticateWithServiceAccount(ctx, keyFile)
}

// 2. Setup Workload Identity for GKE:
func setupWorkloadIdentity(ctx context.Context, provider *cloud.GCPProvider) error {
	authManager := provider.GetAdvancedOperations().GetAuthenticationManager()
	return authManager.SetupWorkloadIdentity(ctx, 
		"my-project", 
		"my-cluster", 
		"us-central1-a", 
		"default", 
		"my-k8s-sa", 
		"my-gcp-sa@my-project.iam.gserviceaccount.com")
}

// 3. Create and configure storage bucket:
func createAPMStorageBucket(ctx context.Context, provider *cloud.GCPProvider, bucketName string) error {
	storageManager := provider.GetAdvancedOperations().GetStorageManager()
	_, err := storageManager.CreateStorageBucket(ctx, bucketName, "us-central1", "STANDARD")
	return err
}

// 4. Enable all required APIs for APM:
func enableAPMAPIs(ctx context.Context, provider *cloud.GCPProvider) error {
	return provider.EnableRequiredAPIs(ctx)
}

// 5. Full APM setup:
func fullAPMSetup(ctx context.Context, provider *cloud.GCPProvider, projectID string) error {
	config := cloud.APMIntegrationConfig{
		ProjectID:            projectID,
		Region:               "us-central1",
		ServiceAccountID:     "apm-monitoring",
		CreateServiceAccount: true,
		SetupMonitoring:      true,
		CreateStorageBucket:  true,
	}
	
	advOps := provider.GetAdvancedOperations()
	return advOps.SetupAPMIntegration(ctx, config)
}