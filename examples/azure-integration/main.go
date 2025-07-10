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
	ctx := context.Background()

	// Create Azure provider
	config := &cloud.ProviderConfig{
		Provider:      cloud.ProviderAzure,
		DefaultRegion: "eastus",
		EnableCache:   true,
		CacheDuration: 5 * time.Minute,
	}

	azureProvider, err := cloud.NewAzureProvider(config)
	if err != nil {
		log.Fatalf("Failed to create Azure provider: %v", err)
	}

	// Create credential manager
	credManager, err := cloud.NewAzureCredentialManager()
	if err != nil {
		log.Fatalf("Failed to create credential manager: %v", err)
	}

	fmt.Println("🚀 Azure CLI Integration Demo")
	fmt.Println("===============================")

	// 1. CLI Detection and Validation
	fmt.Println("\n📋 Step 1: Detecting Azure CLI...")
	cliStatus, err := azureProvider.DetectCLI()
	if err != nil {
		log.Printf("CLI detection failed: %v", err)
	} else {
		fmt.Printf("✅ Azure CLI detected: v%s at %s\n", cliStatus.Version, cliStatus.Path)
		fmt.Printf("   Supported: %t\n", cliStatus.IsSupported)
	}

	// Validate CLI
	if err := azureProvider.ValidateCLI(); err != nil {
		log.Printf("⚠️  CLI validation failed: %v", err)
		fmt.Println("💡 Please run 'az login' to authenticate")
		
		// Demonstrate different authentication methods
		fmt.Println("\n🔐 Authentication Options:")
		fmt.Println("1. Interactive: az login")
		fmt.Println("2. Device code: az login --use-device-code")
		fmt.Println("3. Service principal: az login --service-principal")
		
		// For demo purposes, skip if not authenticated
		return
	}

	// 2. Authentication Demo
	fmt.Println("\n🔐 Step 2: Authentication Validation...")
	if err := azureProvider.ValidateAuth(ctx); err != nil {
		log.Printf("Authentication validation failed: %v", err)
		return
	}
	fmt.Println("✅ Azure authentication validated")

	// Get current credentials
	creds, err := azureProvider.GetCredentials()
	if err != nil {
		log.Printf("Failed to get credentials: %v", err)
	} else {
		fmt.Printf("📋 Current credentials: %s (%s)\n", creds.Account, creds.AuthMethod)
	}

	// 3. Subscription Management
	fmt.Println("\n📑 Step 3: Subscription Management...")
	subscriptions, err := azureProvider.ListSubscriptions(ctx)
	if err != nil {
		log.Printf("Failed to list subscriptions: %v", err)
	} else {
		fmt.Printf("✅ Found %d subscriptions:\n", len(subscriptions))
		for i, sub := range subscriptions {
			if i < 3 { // Show first 3
				fmt.Printf("   - %s (%s) - %s\n", sub.Name, sub.ID[:8]+"...", sub.State)
			}
		}
		if len(subscriptions) > 3 {
			fmt.Printf("   ... and %d more\n", len(subscriptions)-3)
		}
	}

	// 4. Resource Group Management
	fmt.Println("\n📂 Step 4: Resource Group Management...")
	resourceGroups, err := azureProvider.ListResourceGroups(ctx)
	if err != nil {
		log.Printf("Failed to list resource groups: %v", err)
	} else {
		fmt.Printf("✅ Found %d resource groups:\n", len(resourceGroups))
		for i, rg := range resourceGroups {
			if i < 3 { // Show first 3
				fmt.Printf("   - %s (%s) - %s\n", rg.Name, rg.Location, rg.ProvisioningState)
			}
		}
		if len(resourceGroups) > 3 {
			fmt.Printf("   ... and %d more\n", len(resourceGroups)-3)
		}
	}

	// 5. Container Registry (ACR) Operations
	fmt.Println("\n📦 Step 5: Container Registry (ACR)...")
	registries, err := azureProvider.ListRegistries(ctx)
	if err != nil {
		log.Printf("Failed to list registries: %v", err)
	} else {
		fmt.Printf("✅ Found %d ACR registries:\n", len(registries))
		for _, registry := range registries {
			fmt.Printf("   - %s (%s) in %s\n", registry.Name, registry.URL, registry.Region)
		}
	}

	// 6. Kubernetes (AKS) Operations
	fmt.Println("\n☸️  Step 6: Kubernetes (AKS) Clusters...")
	clusters, err := azureProvider.ListClusters(ctx)
	if err != nil {
		log.Printf("Failed to list clusters: %v", err)
	} else {
		fmt.Printf("✅ Found %d AKS clusters:\n", len(clusters))
		for _, cluster := range clusters {
			fmt.Printf("   - %s (v%s) in %s - %s [%d nodes]\n", 
				cluster.Name, cluster.Version, cluster.Region, cluster.Status, cluster.NodeCount)
		}
	}

	// 7. Storage Account Management
	fmt.Println("\n💾 Step 7: Storage Account Management...")
	storageAccounts, err := azureProvider.ListStorageAccounts(ctx)
	if err != nil {
		log.Printf("Failed to list storage accounts: %v", err)
	} else {
		fmt.Printf("✅ Found %d storage accounts:\n", len(storageAccounts))
		for i, sa := range storageAccounts {
			if i < 3 { // Show first 3
				fmt.Printf("   - %s (%s) in %s - %s\n", sa.Name, sa.Kind, sa.Location, sa.ProvisioningState)
			}
		}
		if len(storageAccounts) > 3 {
			fmt.Printf("   ... and %d more\n", len(storageAccounts)-3)
		}
	}

	// 8. Key Vault Operations
	fmt.Println("\n🔑 Step 8: Key Vault Operations...")
	keyVaults, err := azureProvider.ListKeyVaults(ctx)
	if err != nil {
		log.Printf("Failed to list key vaults: %v", err)
	} else {
		fmt.Printf("✅ Found %d key vaults:\n", len(keyVaults))
		for i, kv := range keyVaults {
			if i < 3 { // Show first 3
				fmt.Printf("   - %s\n", kv)
			}
		}
		if len(keyVaults) > 3 {
			fmt.Printf("   ... and %d more\n", len(keyVaults)-3)
		}
	}

	// 9. Application Insights
	fmt.Println("\n📊 Step 9: Application Insights...")
	appInsights, err := azureProvider.ListApplicationInsights(ctx)
	if err != nil {
		log.Printf("Failed to list Application Insights: %v", err)
	} else {
		fmt.Printf("✅ Found %d Application Insights resources:\n", len(appInsights))
		for _, ai := range appInsights {
			fmt.Printf("   - %s in %s (%s)\n", ai.Name, ai.Location, ai.ApplicationType)
		}
	}

	// 10. Service Principal Management Demo
	fmt.Println("\n👤 Step 10: Service Principal Management...")
	servicePrincipals, err := azureProvider.ListServicePrincipals(ctx)
	if err != nil {
		log.Printf("Failed to list service principals: %v", err)
	} else {
		fmt.Printf("✅ Found %d service principals (showing first 3):\n", len(servicePrincipals))
		for i, sp := range servicePrincipals {
			if i < 3 {
				fmt.Printf("   - %s (%s)\n", sp.DisplayName, sp.AppID[:8]+"...")
			}
		}
		if len(servicePrincipals) > 3 {
			fmt.Printf("   ... and %d more\n", len(servicePrincipals)-3)
		}
	}

	// 11. Credential Management Demo
	fmt.Println("\n🗄️  Step 11: Credential Management...")
	
	// Store sample credentials
	sampleCreds := &cloud.Credentials{
		Provider:   cloud.ProviderAzure,
		AuthMethod: cloud.AuthMethodCLI,
		Profile:    "demo",
		Account:    "demo-subscription",
		Region:     "eastus",
		Properties: map[string]string{
			"tenant_id":       "demo-tenant",
			"subscription_id": "demo-subscription",
		},
	}

	if err := credManager.Store(sampleCreds); err != nil {
		log.Printf("Failed to store credentials: %v", err)
	} else {
		fmt.Println("✅ Sample credentials stored securely")
	}

	// List stored credentials
	storedCreds, err := credManager.List(cloud.ProviderAzure)
	if err != nil {
		log.Printf("Failed to list stored credentials: %v", err)
	} else {
		fmt.Printf("✅ Found %d stored credential profiles:\n", len(storedCreds))
		for _, creds := range storedCreds {
			fmt.Printf("   - %s (%s)\n", creds.Profile, creds.AuthMethod)
		}
	}

	// 12. Advanced Features Demo
	fmt.Println("\n🚀 Step 12: Advanced Features...")

	// Demonstrate ARM template validation (if we have a resource group)
	if len(resourceGroups) > 0 {
		fmt.Println("📝 ARM Template validation example:")
		sampleTemplate := &cloud.AzureARMTemplate{
			Name:          "demo-template",
			ResourceGroup: resourceGroups[0].Name,
			Template: map[string]interface{}{
				"$schema":        "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
				"contentVersion": "1.0.0.0",
				"resources":      []interface{}{},
			},
			Parameters: map[string]interface{}{},
			Mode:       "Incremental",
		}

		validationResult, err := azureProvider.ValidateARMTemplate(ctx, sampleTemplate)
		if err != nil {
			log.Printf("ARM template validation failed: %v", err)
		} else if validationResult.Valid {
			fmt.Println("✅ ARM template validation passed")
		} else {
			fmt.Printf("❌ ARM template validation failed: %v\n", validationResult.Errors)
		}
	}

	// Demonstrate Azure Monitor metrics (if we have resources)
	if len(resourceGroups) > 0 {
		fmt.Println("📈 Azure Monitor metrics example:")
		// This would need a specific resource ID in a real scenario
		fmt.Println("   (Requires specific resource ID for actual metrics retrieval)")
	}

	// 13. Region Operations
	fmt.Println("\n🌍 Step 13: Region Operations...")
	regions, err := azureProvider.ListRegions(ctx)
	if err != nil {
		log.Printf("Failed to list regions: %v", err)
	} else {
		fmt.Printf("✅ Found %d available regions (showing first 5):\n", len(regions))
		for i, region := range regions {
			if i < 5 {
				fmt.Printf("   - %s\n", region)
			}
		}
		if len(regions) > 5 {
			fmt.Printf("   ... and %d more\n", len(regions)-5)
		}
	}

	currentRegion := azureProvider.GetCurrentRegion()
	fmt.Printf("📍 Current default region: %s\n", currentRegion)

	// Clean up demo credentials
	fmt.Println("\n🧹 Cleanup: Removing demo credentials...")
	if err := credManager.Delete(cloud.ProviderAzure, "demo"); err != nil {
		log.Printf("Failed to delete demo credentials: %v", err)
	} else {
		fmt.Println("✅ Demo credentials cleaned up")
	}

	fmt.Println("\n🎉 Azure CLI Integration Demo Complete!")
	fmt.Println("=====================================")
	fmt.Println("\n💡 Key Features Demonstrated:")
	fmt.Println("   ✅ Azure CLI detection and validation")
	fmt.Println("   ✅ Multiple authentication methods")
	fmt.Println("   ✅ Subscription and resource group management")
	fmt.Println("   ✅ ACR registry operations")
	fmt.Println("   ✅ AKS cluster management")
	fmt.Println("   ✅ Storage account operations")
	fmt.Println("   ✅ Key Vault integration")
	fmt.Println("   ✅ Application Insights management")
	fmt.Println("   ✅ Service principal operations")
	fmt.Println("   ✅ Secure credential management")
	fmt.Println("   ✅ ARM template validation")
	fmt.Println("   ✅ Azure Monitor integration")
	fmt.Println("   ✅ Region management")
	
	fmt.Println("\n📖 For more information, see the documentation at:")
	fmt.Println("   https://github.com/ybke/apm/docs/azure-integration.md")
}

// demoAuthenticationMethods demonstrates various Azure authentication methods
func demoAuthenticationMethods(ctx context.Context, provider *cloud.AzureProviderImpl) {
	fmt.Println("\n🔐 Authentication Methods Demo")
	fmt.Println("=============================")

	// Note: These methods would typically be used individually based on the environment

	// 1. Interactive Authentication
	fmt.Println("1. Interactive Authentication (Browser):")
	fmt.Println("   - Opens browser for user authentication")
	fmt.Println("   - Suitable for development environments")
	fmt.Println("   - Example: provider.AuthenticateInteractive(ctx)")

	// 2. Device Code Authentication
	fmt.Println("\n2. Device Code Authentication:")
	fmt.Println("   - Provides device code for authentication")
	fmt.Println("   - Suitable for headless environments")
	fmt.Println("   - Example: deviceAuth, _ := provider.AuthenticateDeviceCode(ctx)")

	// 3. Service Principal Authentication
	fmt.Println("\n3. Service Principal Authentication:")
	fmt.Println("   - Uses client ID, secret, and tenant ID")
	fmt.Println("   - Suitable for production environments")
	fmt.Println("   - Example: provider.AuthenticateServicePrincipal(ctx, clientID, secret, tenantID)")

	// 4. Managed Identity Authentication
	fmt.Println("\n4. Managed Identity Authentication:")
	fmt.Println("   - Uses Azure-assigned managed identity")
	fmt.Println("   - Suitable for Azure-hosted applications")
	fmt.Println("   - Example: provider.AuthenticateManagedIdentity(ctx)")

	fmt.Println("\n💡 Choose the authentication method that best fits your environment!")
}

// demoResourceManagement demonstrates Azure resource management
func demoResourceManagement(ctx context.Context, provider *cloud.AzureProviderImpl) {
	fmt.Println("\n📂 Resource Management Demo")
	fmt.Println("===========================")

	// Example of creating a resource group
	fmt.Println("Creating a demo resource group:")
	
	demoRGName := fmt.Sprintf("apm-demo-rg-%d", time.Now().Unix())
	tags := map[string]string{
		"Environment": "Demo",
		"Purpose":     "APM-Integration-Test",
		"CreatedBy":   "APM-CLI",
	}

	fmt.Printf("Resource Group: %s\n", demoRGName)
	fmt.Printf("Location: eastus\n")
	fmt.Printf("Tags: %v\n", tags)

	// In a real scenario, you would uncomment this:
	// rg, err := provider.CreateResourceGroup(ctx, demoRGName, "eastus", tags)
	// if err != nil {
	//     log.Printf("Failed to create resource group: %v", err)
	// } else {
	//     fmt.Printf("✅ Resource group created: %s\n", rg.Name)
	//     
	//     // Clean up
	//     fmt.Println("Cleaning up demo resource group...")
	//     if err := provider.DeleteResourceGroup(ctx, demoRGName); err != nil {
	//         log.Printf("Failed to delete resource group: %v", err)
	//     } else {
	//         fmt.Println("✅ Demo resource group deleted")
	//     }
	// }

	fmt.Println("(Resource group creation skipped in demo mode)")
}

// demoServicePrincipalManagement demonstrates service principal operations
func demoServicePrincipalManagement(ctx context.Context, provider *cloud.AzureProviderImpl) {
	fmt.Println("\n👤 Service Principal Management Demo")
	fmt.Println("===================================")

	demoSPName := fmt.Sprintf("apm-demo-sp-%d", time.Now().Unix())
	
	fmt.Printf("Creating service principal: %s\n", demoSPName)
	
	// In a real scenario, you would uncomment this:
	// sp, err := provider.CreateServicePrincipal(ctx, demoSPName)
	// if err != nil {
	//     log.Printf("Failed to create service principal: %v", err)
	// } else {
	//     fmt.Printf("✅ Service principal created: %s (App ID: %s)\n", sp.DisplayName, sp.AppID)
	//     
	//     // Demonstrate secret rotation
	//     fmt.Println("Rotating service principal secret...")
	//     rotatedSP, err := provider.RotateServicePrincipalSecret(ctx, sp.AppID)
	//     if err != nil {
	//         log.Printf("Failed to rotate secret: %v", err)
	//     } else {
	//         fmt.Println("✅ Service principal secret rotated")
	//     }
	//     
	//     // Clean up
	//     fmt.Println("Cleaning up demo service principal...")
	//     if err := provider.DeleteServicePrincipal(ctx, sp.AppID); err != nil {
	//         log.Printf("Failed to delete service principal: %v", err)
	//     } else {
	//         fmt.Println("✅ Demo service principal deleted")
	//     }
	// }

	fmt.Println("(Service principal creation skipped in demo mode)")
}